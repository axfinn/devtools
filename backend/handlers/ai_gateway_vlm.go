package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

const internalMinimaxVisionKeyID = "internal:minimax-vlm"

// InternalMinimaxVision 内部免 API Key 图像理解接口（直连 MiniMax VLM，无 MCP 子进程）
// POST /api/image-understanding/minimax-vision
// 对齐 Askit：POST {host}/v1/coding_plan/vlm  body {prompt, image_url}  读 data.content
func (h *AIGatewayHandler) InternalMinimaxVision(c *gin.Context) {
	if !requireSameOrigin(c) {
		return
	}

	var req struct {
		Image  string `json:"image"`     // base64 data URL 或 HTTP URL
		Images []string `json:"images"`  // 兼容多图字段，取第一张
		Prompt string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数解析失败"})
		return
	}

	imageURL := strings.TrimSpace(req.Image)
	if imageURL == "" {
		for _, img := range req.Images {
			if strings.TrimSpace(img) != "" {
				imageURL = strings.TrimSpace(img)
				break
			}
		}
	}
	if imageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供图片（image 或 images）"})
		return
	}

	apiKey := strings.TrimSpace(h.cfg.MiniMaxMCP.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(h.cfg.MiniMax.APIKey)
	}
	if apiKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "未配置 MiniMax API Key，请联系管理员"})
		return
	}

	host := strings.TrimRight(strings.TrimSpace(h.cfg.MiniMaxMCP.APIHost), "/")
	if host == "" {
		host = "https://api.minimaxi.com"
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}

	body, _ := json.Marshal(map[string]interface{}{
		"prompt":    prompt,
		"image_url": imageURL,
	})

	start := time.Now()
	respBody, statusCode, err := h.performMinimaxVisionRequest(apiKey, host+"/v1/coding_plan/vlm", body)
	latency := time.Since(start)

	content := ""
	errMsg := ""
	success := err == nil && statusCode < 400
	if err != nil {
		errMsg = err.Error()
	} else {
		var payload map[string]interface{}
		if jErr := json.Unmarshal(respBody, &payload); jErr == nil {
			if baseErr := minimaxBaseRespError(payload); baseErr != "" {
				success = false
				errMsg = baseErr
			}
			if v, ok := payload["content"].(string); ok {
				content = v
			}
		}
		if !success && errMsg == "" {
			errMsg = fmt.Sprintf("上游返回错误(%d): %s", statusCode, truncateString(string(respBody), 500))
		}
	}

	reqInfo, _ := json.Marshal(map[string]interface{}{
		"prompt":    prompt,
		"image_src": truncateString(imageURL, 80),
	})
	_ = h.db.CreateAIAPIRequestLog(&models.AIAPIRequestLog{
		APIKeyID:     internalMinimaxVisionKeyID,
		Model:        "minimax-vlm",
		Provider:     "minimax",
		Endpoint:     "/api/image-understanding/minimax-vision",
		RequestType:  "vision",
		StatusCode:   statusCode,
		Success:      success,
		ErrorMessage: errMsg,
		RequestBody:  string(reqInfo),
		ResponseBody: truncateString(content, 5000),
		ClientIP:     c.ClientIP(),
		LatencyMS:    latency.Milliseconds(),
	})

	if !success {
		c.JSON(http.StatusBadGateway, gin.H{"error": errMsg})
		return
	}
	if content == "" {
		content = "无法识别图片内容"
	}
	c.JSON(http.StatusOK, gin.H{
		"text":  content,
		"model": "minimax-vlm",
	})
}

func (h *AIGatewayHandler) performMinimaxVisionRequest(apiKey, url string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.mediaClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBody, resp.StatusCode, nil
}
