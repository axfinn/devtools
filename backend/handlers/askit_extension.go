package handlers

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// AskitExtensionDownload 提供 AskIt Chrome 扩展的 zip 下载
// GET /api/askit/extension
func (h *AIGatewayHandler) AskitExtensionDownload(c *gin.Context) {
	zipPath := filepath.Join(".", "data", "askit", "askit-extension.zip")
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "扩展包暂未上传，请联系管理员"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=askit-extension.zip")
	c.File(zipPath)
}
