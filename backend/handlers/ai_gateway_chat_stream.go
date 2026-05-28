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
	"time"

	"github.com/gin-gonic/gin"
)

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

	resp, err := h.streamClient.Do(httpReq)
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

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	buf := make([]byte, 1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
				cancel()
				return
			}
			flusher.Flush()
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

	resp, err := h.streamClient.Do(httpReq)
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

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	// 心跳 goroutine
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
				if _, err := c.Writer.Write([]byte(": heartbeat\n\n")); err != nil {
					return
				}
				flusher.Flush()
			case <-ctx.Done():
				return
			case <-heartbeatDone:
				return
			}
		}
	}()
	defer close(heartbeatDone)

	// 将 Anthropic SSE 转换为 OpenAI SSE 格式
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			c.Writer.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
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
			c.Writer.Write([]byte("data: " + string(chunkJSON) + "\n\n"))
			flusher.Flush()

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
			c.Writer.Write([]byte("data: " + string(chunkJSON) + "\n\n"))
			c.Writer.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
			return
		}
	}
}
