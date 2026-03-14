package handlers

import (
	"net/http"
	"path/filepath"
	"strings"

	"devtools/utils"

	"github.com/gin-gonic/gin"
)

// AnalysisHandler 代码分析处理器
type AnalysisHandler struct{}

// NewAnalysisHandler 创建分析处理器
func NewAnalysisHandler() *AnalysisHandler {
	return &AnalysisHandler{}
}

// AnalyzeCodeRequest 代码分析请求
type AnalyzeCodeRequest struct {
	Content  string `json:"content" binding:"required"`
	Language string `json:"language"`
}

// AnalyzeCode 分析代码
func (h *AnalysisHandler) AnalyzeCode(c *gin.Context) {
	var req AnalyzeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// 内容大小限制 (1MB)
	if len(req.Content) > 1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容超过 1MB 限制", "code": 400})
		return
	}

	// 自动检测语言
	language := req.Language
	if language == "" {
		language = utils.DetectLanguage(req.Content)
	}

	// 执行代码分析
	result := utils.AnalyzeCode(req.Content, language)

	c.JSON(http.StatusOK, result)
}

// ScanContentRequest 内容扫描请求
type ScanContentRequest struct {
	Content string `json:"content" binding:"required"`
}

// ScanContent 扫描内容安全性
func (h *AnalysisHandler) ScanContent(c *gin.Context) {
	var req ScanContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// 内容大小限制 (100KB)
	if len(req.Content) > 100*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容超过 100KB 限制", "code": 400})
		return
	}

	// 执行安全扫描
	result := utils.ScanContent(req.Content)

	c.JSON(http.StatusOK, result)
}

// ValidateFileRequest 文件验证请求
type ValidateFileRequest struct {
	Filename string `json:"filename" binding:"required"`
}

// ValidateFile 验证文件安全性
func (h *AnalysisHandler) ValidateFile(c *gin.Context) {
	var req ValidateFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// 验证文件名
	valid, message := utils.ValidateFilename(req.Filename)

	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":   false,
			"message": message,
			"code":    400,
		})
		return
	}

	// 获取文件分类
	ext := strings.ToLower(filepath.Ext(req.Filename))
	mimeType := getMimeTypeFromExt(ext)
	category := utils.GetCategoryByMimeType(mimeType)

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "文件名安全",
		"category": category,
		"mime_type": mimeType,
	})
}

// getMimeTypeFromExt 根据扩展名获取 MIME 类型
func getMimeTypeFromExt(ext string) string {
	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".html": "text/html",
		".htm":  "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "text/xml",
		".md":   "text/markdown",
		".py":   "text/x-python",
		".go":   "text/x-go",
		".java": "text/x-java",
		".c":    "text/x-c",
		".cpp":  "text/x-c++",
		".h":    "text/x-c",
		".hpp":  "text/x-c++",
		".cs":   "text/x-csharp",
		".rb":   "text/x-ruby",
		".php":  "text/x-php",
		".swift": "text/x-swift",
		".kt":   "text/x-kotlin",
		".scala": "text/x-scala",
		".rs":   "text/x-rust",
		".sql":  "text/x-sql",
		".sh":   "text/x-shellscript",
		".bash": "text/x-shellscript",
		".zsh":  "text/x-shellscript",
		".ps1":  "text/x-powershell",
		".yaml": "text/x-yaml",
		".yml":  "text/x-yaml",
		".toml": "text/x-toml",
		".ini":  "text/x-ini",
		".conf": "text/plain",
		".cfg":  "text/plain",
		".log":  "text/plain",
		".csv":  "text/csv",
		".svg":  "image/svg+xml",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".bz2":  "application/x-bzip2",
		".xz":   "application/x-xz",
		".tgz":  "application/gzip",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".bmp":  "image/bmp",
		".ico":  "image/x-icon",
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".avi":  "video/avi",
		".webm": "video/webm",
		".mkv":  "video/x-matroska",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".wasm": "application/wasm",
	}

	return mimeTypes[ext]
}

// GetSupportedLanguages 获取支持的语言列表
func (h *AnalysisHandler) GetSupportedLanguages(c *gin.Context) {
	languages := utils.GetSupportedLanguages()

	c.JSON(http.StatusOK, gin.H{
		"languages": languages,
		"count":    len(languages),
	})
}
