package handlers

import (
	"encoding/json"
	"fmt"
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
	if model == "" {
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

	var (
		content    string
		rawPayload map[string]interface{}
		callErr    error
		provider   string
	)
	start := time.Now()
	if strings.HasPrefix(model, "MiniMax") {
		provider = "minimax"
		content, rawPayload, callErr = h.callMiniMaxVision(model, prompt, dataURLs)
	} else {
		provider = "dashscope"
		content, rawPayload, callErr = h.callDashScopeVision(model, prompt, dataURLs)
	}
	latency := time.Since(start)

	statusCode := http.StatusOK
	success := callErr == nil
	errMsg := ""
	if callErr != nil {
		statusCode = http.StatusBadGateway
		errMsg = callErr.Error()
	}

	// 记录请求流水
	reqInfo, _ := json.Marshal(map[string]interface{}{
		"prompt":    prompt,
		"model":     model,
		"images":    logSources,
		"total_kb":  totalSizeKB,
		"img_count": len(imgs),
	})
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

// callMiniMaxVision 使用 MiniMax-M3 原生多模态（Anthropic image blocks）理解图片
func (h *AIGatewayHandler) callMiniMaxVision(model, prompt string, dataURLs []string) (string, map[string]interface{}, error) {
	if strings.TrimSpace(h.cfg.MiniMax.APIKey) == "" {
		return "", nil, fmt.Errorf("未配置 minimax.api_key 或 MINIMAX_API_KEY")
	}
	contentBlocks := make([]map[string]interface{}, 0, len(dataURLs)+1)
	for i, du := range dataURLs {
		mime, b64 := splitDataURL(du)
		if b64 == "" {
			return "", nil, fmt.Errorf("图片 %d 不是合法的 data URL", i+1)
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

	bodyMap := map[string]interface{}{
		"model":      model,
		"max_tokens": visionMaxTokens,
		"messages": []map[string]interface{}{
			{"role": "user", "content": contentBlocks},
		},
	}
	raw, err := h.doMiniMaxAnthropicRequest(minimaxAnthropicMessagesURL, h.cfg.MiniMax.APIKey, bodyMap)
	if err != nil {
		return "", raw, err
	}
	return extractMiniMaxText(raw), raw, nil
}

// callDashScopeVision 使用 DashScope OpenAI 兼容视觉模型（qwen/kimi）理解图片
func (h *AIGatewayHandler) callDashScopeVision(model, prompt string, dataURLs []string) (string, map[string]interface{}, error) {
	if strings.TrimSpace(h.cfg.DashScope.APIKey) == "" {
		return "", nil, fmt.Errorf("未配置 DashScope API Key，请联系管理员")
	}
	contentParts := make([]map[string]interface{}, 0, len(dataURLs)+1)
	for _, du := range dataURLs {
		contentParts = append(contentParts, map[string]interface{}{
			"type":      "image_url",
			"image_url": map[string]interface{}{"url": du},
		})
	}
	contentParts = append(contentParts, map[string]interface{}{"type": "text", "text": prompt})

	chatReq := ChatCompletionRequest{
		Model: model,
		Messages: []map[string]interface{}{
			{"role": "user", "content": contentParts},
		},
	}
	result, raw, err := h.callDashScope(chatReq)
	if err != nil {
		return "", raw, err
	}
	content, _ := result["content"].(string)
	return content, raw, nil
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
