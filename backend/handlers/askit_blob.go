package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// askit 图片转存:把笔记图片字节存到本机,按 userID 隔离,返回稳定可访问的 URL。
// 解决「存的是外链 → 换设备/内网/防盗链/过期后看不到」的问题。
const askitBlobDir = "./data/askit_blobs"

// 单文件上限 10MB(图片转存够用,避免被塞大文件)。
const askitBlobMaxBytes = 10 * 1024 * 1024

// askitRandomName 生成不可枚举的随机文件名。
func askitRandomName(ext string) string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf) + ext
}

// askitSafeSeg 清洗路径段,杜绝目录穿越(只保留 base 名)。
func askitSafeSeg(s string) string {
	s = filepath.Base(s)
	if s == "." || s == "/" || s == ".." {
		return ""
	}
	return s
}

// BlobUpload POST /blob/upload —— 鉴权后上传一张图片,按 userID 存档,返回访问 URL。
// 仅接受图片;按文件头嗅探真实类型,拒绝非图片。
func (h *AskitSyncHandler) BlobUpload(c *gin.Context) {
	userID := c.GetString(askitUserIDKey)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no_file"})
		return
	}
	defer file.Close()

	if header.Size <= 0 || header.Size > askitBlobMaxBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "blob_too_large"})
		return
	}

	// 嗅探文件头判定真实类型,不信任客户端声明的 Content-Type。
	magic := make([]byte, 16)
	n, _ := file.Read(magic)
	magic = magic[:n]
	if _, err := file.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seek_failed"})
		return
	}
	detected := detectFileType(magic)
	if !strings.HasPrefix(detected, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not_image"})
		return
	}

	ext := getExtFromMimeType(detected)
	if ext == "" {
		ext = ".jpg"
	}

	dir := filepath.Join(askitBlobDir, askitSafeSeg(userID))
	if dir == askitBlobDir || askitSafeSeg(userID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_user"})
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mkdir_failed"})
		return
	}

	name := askitRandomName(ext)
	path := filepath.Join(dir, name)
	out, err := os.Create(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create_failed"})
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "write_failed"})
		return
	}
	out.Close()

	c.JSON(http.StatusOK, gin.H{"url": "/api/askit/v1/blob/files/" + askitSafeSeg(userID) + "/" + name})
}

// BlobServe GET /blob/files/:uid/:name —— 公开读取转存图片。文件名随机不可枚举,
// 路径段都经 base 清洗防穿越。
func (h *AskitSyncHandler) BlobServe(c *gin.Context) {
	uid := askitSafeSeg(c.Param("uid"))
	name := askitSafeSeg(c.Param("name"))
	if uid == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_path"})
		return
	}
	path := filepath.Join(askitBlobDir, uid, name)
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.File(path)
}
