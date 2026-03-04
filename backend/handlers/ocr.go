package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type OCRHandler struct {
	serviceURL string
}

func NewOCRHandler() *OCRHandler {
	serviceURL := os.Getenv("OCR_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://ocr-service:8000"
	}
	return &OCRHandler{serviceURL: serviceURL}
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

	body, _ := json.Marshal(req)
	resp, err := http.Post(h.serviceURL+"/ocr", "application/json", bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OCR 服务不可用，请稍后重试"})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json; charset=utf-8", respBody)
}
