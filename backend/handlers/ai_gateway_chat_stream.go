package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// sseWriter 对 SSE 写操作加锁：心跳 goroutine 与主复制循环会并发 Write+Flush，
// 不加锁时 net/http 的 Flush 与 Write 在底层 bufio 上存在数据竞争。
type sseWriter struct {
	mu      sync.Mutex
	w       http.ResponseWriter
	flusher http.Flusher
}

func newSSEWriter(w http.ResponseWriter, flusher http.Flusher) *sseWriter {
	return &sseWriter{w: w, flusher: flusher}
}

// write 原子地写入并 flush；失败返回错误，供调用方结束流。
func (s *sseWriter) write(p []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.w.Write(p); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}

// doStreamRequest 执行流式上游请求，对瞬时连接级错误做一次安全重试。
// 重试只发生在"尚未拿到任何响应"（client.Do 失败）之前，已开始读到响应体后不重试。
// 客户端已断开（context 取消）时返回原始错误，不重试。
func doStreamRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err == nil {
		return resp, nil
	}
	if !isTransientConnErr(err) || req.GetBody == nil {
		return nil, err
	}
	body, bodyErr := req.GetBody()
	if bodyErr != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	clone.Body = body
	time.Sleep(300 * time.Millisecond)
	return client.Do(clone)
}

// StreamChatCompletions 处理 OpenAI 兼容格式的流式聊天请求
// 当 ChatCompletions 收到 stream: true 时调用此方法
func (h *AIGatewayHandler) StreamChatCompletions(c *gin.Context, req ChatCompletionRequest) {
	provider := h.resolveChatProvider(req.Model)

	switch provider {
	case "deepseek":
		h.streamOpenAICompatible(c, req, "https://api.deepseek.com/chat/completions", h.cfg.DeepSeek.APIKey)
	case "dashscope":
		baseURL := fallbackString(h.cfg.DashScope.BaseURL, "https://coding.dashscope.aliyuncs.com/v1")
		endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
		h.streamOpenAICompatible(c, req, endpoint, h.cfg.DashScope.APIKey)
	case "minimax":
		h.streamMiniMaxChat(c, req)
	case "proxy":
		endpoint := strings.TrimRight(h.cfg.AIGateway.Proxy.APIURL, "/") + "/chat/completions"
		req.Model = h.proxyUpstreamModel(req.Model)
		h.streamOpenAICompatible(c, req, endpoint, h.cfg.AIGateway.Proxy.APIKey)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("不支持流式的模型: %s", req.Model)})
	}
}

// streamOpenAICompatible 透传 OpenAI 兼容格式的 SSE 流
// 带 20s 心跳（DeepSeek reasoner 长思考期可能长时间无 data 事件，
// 中间代理/CDN 会因空闲超时断开连接）。
func (h *AIGatewayHandler) streamOpenAICompatible(c *gin.Context, req ChatCompletionRequest, endpoint, apiKey string) {
	bodyMap := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		bodyMap["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}
	if req.Stop != nil {
		bodyMap["stop"] = req.Stop
	}
	if len(req.ResponseFormat) > 0 {
		bodyMap["response_format"] = req.ResponseFormat
	}
	if len(req.Tools) > 0 {
		bodyMap["tools"] = req.Tools
	}
	if req.ToolChoice != nil {
		bodyMap["tool_choice"] = req.ToolChoice
	}
	if strings.TrimSpace(req.ReasoningEffort) != "" {
		bodyMap["reasoning_effort"] = strings.TrimSpace(req.ReasoningEffort)
	}
	for key, value := range req.ExtraBody {
		bodyMap[key] = value
	}

	body, _ := json.Marshal(bodyMap)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := doStreamRequest(h.streamClient, httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", errBody)
		return
	}

	// 先校验 Flusher 再 WriteHeader，避免非流式场景留下半截响应。
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	writer := newSSEWriter(c.Writer, flusher)

	// 先发一个心跳，让客户端立即感知连接已建立
	if err := writer.write([]byte(": heartbeat\n\n")); err != nil {
		cancel()
		return
	}

	// 持续心跳：每 20s 发一条 SSE 注释，防止 thinking 阶段空闲被中间层断开
	heartbeatDone := make(chan struct{})
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("PANIC in background goroutine: %v", r) } }()
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := writer.write([]byte(": heartbeat\n\n")); err != nil {
					return
				}
			case <-ctx.Done():
				return
			case <-heartbeatDone:
				return
			}
		}
	}()
	defer close(heartbeatDone)

	buf := make([]byte, 1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if err := writer.write(buf[:n]); err != nil {
				cancel() // 客户端断开，取消上游请求
				return
			}
		}
		if readErr != nil {
			return
		}
	}
}

// streamMiniMaxChat 通过 MiniMax Anthropic 兼容端点实现流式输出，
// 将 Anthropic SSE 格式转换为 OpenAI SSE 格式
func (h *AIGatewayHandler) streamMiniMaxChat(c *gin.Context, req ChatCompletionRequest) {
	if strings.TrimSpace(h.cfg.MiniMax.APIKey) == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 minimax.api_key"})
		return
	}

	bodyMap := buildMiniMaxAnthropicBody(req)
	bodyMap["stream"] = true
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}

	body, _ := json.Marshal(bodyMap)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.minimaxi.com/anthropic/v1/messages", bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+h.cfg.MiniMax.APIKey)
	httpReq.Header.Set("x-api-key", h.cfg.MiniMax.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := doStreamRequest(h.streamClient, httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", errBody)
		return
	}

	// 先校验 Flusher 再 WriteHeader，避免非流式场景留下半截响应。
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	writer := newSSEWriter(c.Writer, flusher)

	// 先发一个心跳，让客户端立即感知连接已建立
	if err := writer.write([]byte(": heartbeat\n\n")); err != nil {
		cancel()
		return
	}

	// 心跳 goroutine（写操作与主循环通过 sseWriter 串行化）
	heartbeatDone := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in heartbeat goroutine: %v", r)
			}
		}()
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := writer.write([]byte(": heartbeat\n\n")); err != nil {
					return
				}
			case <-ctx.Done():
				return
			case <-heartbeatDone:
				return
			}
		}
	}()
	defer close(heartbeatDone)

	// 将 Anthropic SSE 转换为 OpenAI SSE 格式。
	// 行缓冲放宽到 1MB：单个大分片（长代码/长文本）超过 64KB 时
	// bufio.Scanner 会静默截断流，导致客户端收到不完整的回复。
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			_ = writer.write([]byte("data: [DONE]\n\n"))
			return
		}

		var event map[string]interface{}
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case "content_block_delta":
			delta, _ := event["delta"].(map[string]interface{})
			text, _ := delta["text"].(string)
			if text == "" {
				continue
			}
			chunk := map[string]interface{}{
				"id":      "chatcmpl-stream",
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   req.Model,
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": text,
						},
					},
				},
			}
			chunkJSON, _ := json.Marshal(chunk)
			if err := writer.write([]byte("data: " + string(chunkJSON) + "\n\n")); err != nil {
				cancel()
				return
			}

		case "message_stop":
			chunk := map[string]interface{}{
				"id":      "chatcmpl-stream",
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   req.Model,
				"choices": []map[string]interface{}{
					{
						"index":         0,
						"delta":         map[string]interface{}{},
						"finish_reason": "stop",
					},
				},
			}
			chunkJSON, _ := json.Marshal(chunk)
			_ = writer.write([]byte("data: " + string(chunkJSON) + "\n\n"))
			_ = writer.write([]byte("data: [DONE]\n\n"))
			return
		}
	}
}
