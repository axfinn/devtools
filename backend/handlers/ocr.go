package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const ocrMaxImageSize = 30 * 1024 * 1024 // 30MB

type OCRHandler struct {
	serviceURL string
	client     *http.Client
}

func NewOCRHandler() *OCRHandler {
	serviceURL := os.Getenv("OCR_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://ocr-service:8000"
	}
	// 禁用代理，确保能访问 Docker 内部网络服务
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	return &OCRHandler{serviceURL: serviceURL, client: client}
}

type ocrRequest struct {
	Image string `json:"image" binding:"required"`
}

// Extract handles POST /api/ocr — proxies to the RapidOCR Python service
func (h *OCRHandler) Extract(c *gin.Context) {
	var req ocrRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 image 字段"})
		return
	}

	// 检查图片大小
	imageData := req.Image
	// 移除 data:image/xxx;base64, 前缀
	if strings.Contains(imageData, ",") {
		imageData = strings.Split(imageData, ",")[1]
	}
	// 计算解码后的大小
	decodedLen := base64.StdEncoding.DecodedLen(len(imageData))
	if decodedLen > ocrMaxImageSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片大小不能超过 30MB"})
		return
	}

	body, _ := json.Marshal(req)
	resp, err := h.client.Post(h.serviceURL+"/ocr", "application/json", bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OCR 服务不可用，请稍后重试"})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json; charset=utf-8", respBody)
}
