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
	if strings.TrimSpace(h.cfg.DashScope.APIKey) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "未配置 DashScope API Key，请联系管理员"})
		return
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = "qwen3.5-plus"
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}

	// 处理每张图：HTTP URL 下载转 base64，data URL 直接使用
	logSources := make([]string, 0, len(imgs))
	contentParts := make([]map[string]interface{}, 0, len(imgs)+1)
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
		contentParts = append(contentParts, map[string]interface{}{
			"type":      "image_url",
			"image_url": map[string]interface{}{"url": img},
		})
	}
	contentParts = append(contentParts, map[string]interface{}{"type": "text", "text": prompt})

	chatReq := ChatCompletionRequest{
		Model: model,
		Messages: []map[string]interface{}{
			{"role": "user", "content": contentParts},
		},
	}

	start := time.Now()
	result, _, err := h.callDashScope(chatReq)
	latency := time.Since(start)

	statusCode := http.StatusOK
	success := err == nil
	errMsg := ""
	content := ""
	if err != nil {
		statusCode = http.StatusBadGateway
		errMsg = err.Error()
	} else {
		content, _ = result["content"].(string)
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
		Provider:     "dashscope",
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

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"text":   content,
		"model":  model,
		"result": result,
	})
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
