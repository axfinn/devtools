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

	"devtools/models"

	"github.com/gin-gonic/gin"
)

func (h *AIGatewayHandler) InternalQwenVision(c *gin.Context) {
	if !requireSameOrigin(c) {
		return
	}

	var req struct {
		Images []string `json:"images"` // 多图，每项可为 base64 data URL 或 HTTP URL
		Image  string   `json:"image"`  // 单图兼容，合并入 images
		Prompt string   `json:"prompt"`
		Model  string   `json:"model"`
		Stream bool     `json:"stream"` // true 时以 SSE 流式返回，避免长输出超时
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数解析失败"})
		return
	}
	// 兼容单图字段
	if strings.TrimSpace(req.Image) != "" {
		req.Images = append([]string{req.Image}, req.Images...)
	}
	// 去空
	imgs := make([]string, 0, len(req.Images))
	for _, img := range req.Images {
		if strings.TrimSpace(img) != "" {
			imgs = append(imgs, strings.TrimSpace(img))
		}
	}
	if len(imgs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请至少提供一张图片（image 或 images）"})
		return
	}
	if len(imgs) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片数量不能超过 10 张"})
		return
	}
	model := strings.TrimSpace(req.Model)
	// 图片理解统一走 MiniMax-M3 原生多模态（不再使用 qwen/DashScope，避免会员失效导致请求挂起超时）
	if !strings.HasPrefix(model, "MiniMax") {
		model = "MiniMax-M3"
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}

	// 归一化每张图：HTTP URL 下载转 base64 data URL，data URL 原样保留
	dataURLs := make([]string, 0, len(imgs))
	logSources := make([]string, 0, len(imgs))
	totalSizeKB := 0
	for i, img := range imgs {
		if strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://") {
			logSources = append(logSources, fmt.Sprintf("[%d]url:%s", i+1, truncateString(img, 80)))
			downloaded, mimeType, dlErr := fetchImageAsBase64(h.client, img)
			if dlErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("图片 %d 下载失败: %s", i+1, dlErr.Error())})
				return
			}
			img = "data:" + mimeType + ";base64," + downloaded
		} else {
			mime, sizeKB := extractImageMeta(img)
			totalSizeKB += sizeKB
			logSources = append(logSources, fmt.Sprintf("[%d]%s~%dKB", i+1, mime, sizeKB))
		}
		dataURLs = append(dataURLs, img)
	}

	logFields := map[string]interface{}{
		"prompt":    prompt,
		"model":     model,
		"images":    logSources,
		"total_kb":  totalSizeKB,
		"img_count": len(imgs),
	}

	if req.Stream {
		h.streamMiniMaxVision(c, model, prompt, dataURLs, logFields)
		return
	}

	var (
		content    string
		rawPayload map[string]interface{}
		callErr    error
	)
	const provider = "minimax"
	start := time.Now()
	content, rawPayload, callErr = h.callMiniMaxVision(model, prompt, dataURLs)
	latency := time.Since(start)

	statusCode := http.StatusOK
	success := callErr == nil
	errMsg := ""
	if callErr != nil {
		statusCode = http.StatusBadGateway
		errMsg = callErr.Error()
	}

	// 记录请求流水
	reqInfo, _ := json.Marshal(logFields)
	_ = h.db.CreateAIAPIRequestLog(&models.AIAPIRequestLog{
		APIKeyID:     internalQwenVisionKeyID,
		Model:        model,
		Provider:     provider,
		Endpoint:     "/api/image-understanding/qwen-vision",
		RequestType:  "vision",
		StatusCode:   statusCode,
		Success:      success,
		ErrorMessage: errMsg,
		RequestBody:  string(reqInfo),
		ResponseBody: truncateString(content, 5000),
		ClientIP:     c.ClientIP(),
		LatencyMS:    latency.Milliseconds(),
	})

	if callErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"text":   content,
		"model":  model,
		"result": rawPayload,
	})
}

// visionMaxTokens 视觉理解回复的最大 token 数
const visionMaxTokens = 2048

// minimaxAnthropicMessagesURL MiniMax Anthropic 兼容端点
const minimaxAnthropicMessagesURL = "https://api.minimaxi.com/anthropic/v1/messages"

// visionHeartbeatInterval 流式无输出时的心跳间隔，防止中间网关按空闲断流
const visionHeartbeatInterval = 20 * time.Second

// visionSSEBufferSize 流式扫描器单行缓冲上限
const visionSSEBufferSize = 64 * 1024

// callMiniMaxVision 使用 MiniMax-M3 原生多模态（Anthropic image blocks）理解图片
func (h *AIGatewayHandler) callMiniMaxVision(model, prompt string, dataURLs []string) (string, map[string]interface{}, error) {
	if strings.TrimSpace(h.cfg.MiniMax.APIKey) == "" {
		return "", nil, fmt.Errorf("未配置 minimax.api_key 或 MINIMAX_API_KEY")
	}
	bodyMap, err := buildMiniMaxVisionBody(model, prompt, dataURLs)
	if err != nil {
		return "", nil, err
	}
	raw, err := h.doMiniMaxAnthropicRequest(minimaxAnthropicMessagesURL, h.cfg.MiniMax.APIKey, bodyMap)
	if err != nil {
		return "", raw, err
	}
	return extractMiniMaxText(raw), raw, nil
}

// buildMiniMaxVisionBody 构造 MiniMax Anthropic 视觉请求体（图片 image blocks + 文本 prompt）
func buildMiniMaxVisionBody(model, prompt string, dataURLs []string) (map[string]interface{}, error) {
	contentBlocks := make([]map[string]interface{}, 0, len(dataURLs)+1)
	for i, du := range dataURLs {
		mime, b64 := splitDataURL(du)
		if b64 == "" {
			return nil, fmt.Errorf("图片 %d 不是合法的 data URL", i+1)
		}
		contentBlocks = append(contentBlocks, map[string]interface{}{
			"type": "image",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": mime,
				"data":       b64,
			},
		})
	}
	contentBlocks = append(contentBlocks, map[string]interface{}{"type": "text", "text": prompt})

	return map[string]interface{}{
		"model":      model,
		"max_tokens": visionMaxTokens,
		"messages": []map[string]interface{}{
			{"role": "user", "content": contentBlocks},
		},
	}, nil
}

// streamMiniMaxVision 以 SSE 流式返回 M3 视觉理解结果，避免长输出触发网关超时（524）。
// 下游使用 Anthropic SSE，转换为前端友好的 {delta}/{done}/{error} 事件，并在结束后记录流水。
func (h *AIGatewayHandler) streamMiniMaxVision(c *gin.Context, model, prompt string, dataURLs []string, logFields map[string]interface{}) {
	reqInfo, _ := json.Marshal(logFields)
	writeLog := func(statusCode int, success bool, errMsg, content string, latency time.Duration) {
		_ = h.db.CreateAIAPIRequestLog(&models.AIAPIRequestLog{
			APIKeyID:     internalQwenVisionKeyID,
			Model:        model,
			Provider:     "minimax",
			Endpoint:     "/api/image-understanding/qwen-vision",
			RequestType:  "vision",
			StatusCode:   statusCode,
			Success:      success,
			ErrorMessage: errMsg,
			RequestBody:  string(reqInfo),
			ResponseBody: truncateString(content, 5000),
			ClientIP:     c.ClientIP(),
			LatencyMS:    latency.Milliseconds(),
		})
	}
	start := time.Now()

	if strings.TrimSpace(h.cfg.MiniMax.APIKey) == "" {
		writeLog(http.StatusServiceUnavailable, false, "未配置 minimax.api_key", "", time.Since(start))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "未配置 minimax.api_key 或 MINIMAX_API_KEY"})
		return
	}
	bodyMap, err := buildMiniMaxVisionBody(model, prompt, dataURLs)
	if err != nil {
		writeLog(http.StatusBadRequest, false, err.Error(), "", time.Since(start))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bodyMap["stream"] = true

	body, _ := json.Marshal(bodyMap)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, minimaxAnthropicMessagesURL, bytes.NewReader(body))
	if err != nil {
		writeLog(http.StatusBadGateway, false, err.Error(), "", time.Since(start))
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
		writeLog(http.StatusBadGateway, false, err.Error(), "", time.Since(start))
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(resp.Body)
		writeLog(resp.StatusCode, false, truncateString(string(errBody), 400), "", time.Since(start))
		c.Data(resp.StatusCode, "application/json", errBody)
		return
	}

	h.relayVisionSSE(c, ctx, cancel, resp.Body, model, writeLog, start)
}

// relayVisionSSE 读取 Anthropic SSE，转换为前端事件流（delta/done/error），累计文本并在结束时记录流水。
func (h *AIGatewayHandler) relayVisionSSE(
	c *gin.Context,
	ctx context.Context,
	cancel context.CancelFunc,
	upstream io.Reader,
	model string,
	writeLog func(int, bool, string, string, time.Duration),
	start time.Time,
) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		writeLog(http.StatusInternalServerError, false, "writer 不支持 flush", "", time.Since(start))
		return
	}

	send := func(event map[string]interface{}) bool {
		payload, _ := json.Marshal(event)
		if _, err := c.Writer.Write([]byte("data: " + string(payload) + "\n\n")); err != nil {
			cancel()
			return false
		}
		flusher.Flush()
		return true
	}

	// 心跳：长时间无输出时维持连接，防止中间网关（Cloudflare）按空闲断流。
	heartbeatDone := make(chan struct{})
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("PANIC in heartbeat goroutine: %v", r) } }()
		ticker := time.NewTicker(visionHeartbeatInterval)
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

	var builder strings.Builder
	scanner := bufio.NewScanner(upstream)
	scanner.Buffer(make([]byte, visionSSEBufferSize), visionSSEBufferSize)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var event map[string]interface{}
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}
		switch fmt.Sprint(event["type"]) {
		case "content_block_delta":
			delta, _ := event["delta"].(map[string]interface{})
			text, _ := delta["text"].(string)
			if text == "" {
				continue
			}
			builder.WriteString(text)
			if !send(map[string]interface{}{"delta": text}) {
				writeLog(499, false, "客户端断开", builder.String(), time.Since(start))
				return
			}
		case "message_stop":
			send(map[string]interface{}{"done": true, "model": model})
			writeLog(http.StatusOK, true, "", builder.String(), time.Since(start))
			return
		}
	}
	if err := scanner.Err(); err != nil {
		send(map[string]interface{}{"error": err.Error()})
		writeLog(http.StatusBadGateway, false, err.Error(), builder.String(), time.Since(start))
		return
	}
	// 上游未发 message_stop 即结束：补一个 done，按已收到的文本记录成功。
	send(map[string]interface{}{"done": true, "model": model})
	writeLog(http.StatusOK, builder.Len() > 0, "", builder.String(), time.Since(start))
}

// splitDataURL 拆分 data:<mime>;base64,<data>，返回 mime 与裸 base64 数据
func splitDataURL(dataURL string) (mime, b64 string) {
	if !strings.HasPrefix(dataURL, "data:") {
		return "", ""
	}
	comma := strings.Index(dataURL, ",")
	if comma < 0 {
		return "", ""
	}
	header := dataURL[5:comma]
	b64 = dataURL[comma+1:]
	if semi := strings.Index(header, ";"); semi >= 0 {
		mime = header[:semi]
	} else {
		mime = header
	}
	if mime == "" {
		mime = "image/png"
	}
	return mime, b64
}

// AdminListQwenVisionLogs 管理员查看 Qwen 视觉理解请求流水
// GET /api/image-understanding/qwen-vision/logs
// 认证: X-Image-Admin-Password header 或 admin_password query（独立于 super_admin_password）
func (h *AIGatewayHandler) AdminListQwenVisionLogs(c *gin.Context) {
	if !h.requireImageUnderstandingAdmin(c) {
		return
	}
	limit := boundedInt(c.DefaultQuery("limit", "50"), 50, 1, 200)
	offset := boundedInt(c.DefaultQuery("offset", "0"), 0, 0, 1000000)
	logs, err := h.db.ListInternalRequestLogs(internalQwenVisionKeyID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取流水失败"})
		return
	}
	total, _ := h.db.CountInternalRequestLogs(internalQwenVisionKeyID)
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total, "limit": limit, "offset": offset})
}
