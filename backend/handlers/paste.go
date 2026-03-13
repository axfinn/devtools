package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

const (
	pasteUploadDir = "./data/paste_files"
	chunkDir       = "./data/paste_chunks" // 分片临时目录
)

// ChunkUploadInfo 分片上传信息
type ChunkUploadInfo struct {
	FileID         string    `json:"file_id"`
	FileName       string    `json:"file_name"`
	TotalChunks    int       `json:"total_chunks"`
	ChunkSize      int64     `json:"chunk_size"`
	FileSize       int64     `json:"file_size"`
	UploadedChunks []int     `json:"uploaded_chunks"`
	CreatedAt      time.Time `json:"created_at"`
	mu             sync.Mutex
}

var (
	// 全局分片上传管理器
	chunkUploads   = make(map[string]*ChunkUploadInfo)
	chunkUploadsMu sync.RWMutex
)

// FileMetadata 文件元数据
type FileMetadata struct {
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	Type         string `json:"type"` // image, video
	Size         int64  `json:"size"`
	URL          string `json:"url"`
}

// PasteHandler 粘贴板处理器
type PasteHandler struct {
	db         *models.DB
	maxTotal   int
	maxPerIP   int
	ipWindow   time.Duration
}

// NewPasteHandler 创建粘贴板处理器
func NewPasteHandler(db *models.DB) *PasteHandler {
	return &PasteHandler{
		db:       db,
		maxTotal: 10000,        // 最多存储 10000 条
		maxPerIP: 10,           // 每 IP 每分钟最多 10 条（与中间件限流一致）
		ipWindow: time.Minute,
	}
}

// SupportedContentTypes 支持的内容类型
var SupportedContentTypes = []string{
	"text", "code", "markdown", "json", "html", "xml", "sql", "log",
}

// SupportedLanguages 支持的编程语言列表（增强版）
var SupportedLanguages = []string{
	"javascript", "typescript", "python", "go", "rust", "java", "c", "cpp",
	"csharp", "php", "ruby", "swift", "kotlin", "scala", "html", "css",
	"scss", "json", "yaml", "xml", "sql", "bash", "shell", "powershell",
	"dockerfile", "markdown", "r", "matlab", "julia", "haskell", "elixir",
	"erlang", "clojure", "fsharp", "ocaml", "dart", "lua", "perl", "coffeescript",
	"vue", "react", "makefile", "cmake", "nginx", "apache", "gradle", "toml",
	"ini", "protobuf", "graphql", "terraform", "assembly", "vim", "latex",
	"sass", "less", "objectivec", "text", "plaintext", "log",
	// 额外支持的语言
	"pascal", "delphi", "fortran", "cobol", "lisp", "scheme", "prolog",
	"actionscript", "apex", "sol", "move", "cairo", "sway", "pil",
}

// FileCategoryInfo 文件分类信息
type FileCategoryInfo struct {
	Category string `json:"category"`
	Icon     string `json:"icon"`
	Color    string `json:"color"`
}

// GetFileCategoryInfo 获取文件分类详细信息
func GetFileCategoryInfo(mimeType string) FileCategoryInfo {
	category := getFileCategory(mimeType)
	info := FileCategoryInfo{Category: category}

	switch category {
	case "image":
		info.Icon = "🖼️"
		info.Color = "#4CAF50"
	case "video":
		info.Icon = "🎬"
		info.Color = "#2196F3"
	case "audio":
		info.Icon = "🎵"
		info.Color = "#9C27B0"
	case "document":
		info.Icon = "📄"
		info.Color = "#FF9800"
	case "archive":
		info.Icon = "📦"
		info.Color = "#795548"
	case "code":
		info.Icon = "💻"
		info.Color = "#607D8B"
	default:
		info.Icon = "📁"
		info.Color = "#9E9E9E"
	}

	return info
}

type CreatePasteRequest struct {
	Content       string   `json:"content"`
	Title         string   `json:"title"`
	Language      string   `json:"language"`
	Password      string   `json:"password"`
	ExpiresIn     int      `json:"expires_in"` // 过期时间（小时）
	MaxViews      int      `json:"max_views"`
	FileIDs       []string `json:"file_ids"`       // 上传文件的ID列表
	AdminPassword string   `json:"admin_password"` // 管理员密码（设置更多访问次数或永久）
}

// ContentType 内容类型
type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeCode     ContentType = "code"
	ContentTypeMarkdown ContentType = "markdown"
	ContentTypeJSON     ContentType = "json"
	ContentTypeHTML     ContentType = "html"
	ContentTypeXML      ContentType = "xml"
	ContentTypeSQL      ContentType = "sql"
	ContentTypeLog      ContentType = "log"
)

type PasteResponse struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Language     string          `json:"language"`
	ContentType  string          `json:"content_type"`
	Content      string          `json:"content,omitempty"`
	ExpiresAt    time.Time       `json:"expires_at"`
	MaxViews     int             `json:"max_views"`
	Views        int             `json:"views"`
	CreatedAt    time.Time       `json:"created_at"`
	HasPassword bool            `json:"has_password"`
	Files        []*FileMetadata `json:"files,omitempty"`
}

// UploadFile 上传文件（图片或视频）
func (h *PasteHandler) UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择文件", "code": 400})
		return
	}
	defer file.Close()

	cfg := config.Get()

	// 检查文件大小
	if header.Size > cfg.Paste.MaxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("文件大小超过限制 (最大 %dMB)", cfg.Paste.MaxFileSize/1024/1024),
			"code":  400,
		})
		return
	}

	// 读取文件头检测类型
	magicBytes := make([]byte, 16)
	n, _ := file.Read(magicBytes)
	magicBytes = magicBytes[:n]
	file.Seek(0, 0)

	detectedType := detectFileType(magicBytes)
	if detectedType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型", "code": 400})
		return
	}

	// 确定文件类别
	fileCategory := getFileCategory(detectedType)

	// 生成随机文件名
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "code": 500})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = getExtFromMimeType(detectedType)
	}
	filename := fmt.Sprintf("%s%s", hex.EncodeToString(randomBytes), ext)

	// 确保上传目录存在
	if err := os.MkdirAll(pasteUploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "code": 500})
		return
	}

	// 保存文件
	filePath := filepath.Join(pasteUploadDir, filename)
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败", "code": 500})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败", "code": 500})
		return
	}

	// 返回文件信息，对原始文件名进行安全处理
	fileURL := "/api/paste/files/" + filename
	safeOriginalName := utils.SanitizeFilename(header.Filename)
	c.JSON(http.StatusOK, gin.H{
		"id":            filename,
		"url":           fileURL,
		"filename":      filename,
		"original_name": safeOriginalName,
		"type":          fileCategory,
		"size":          header.Size,
	})
}

func (h *PasteHandler) Create(c *gin.Context) {
	var req CreatePasteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// 检查是否有内容或文件
	if req.Content == "" && len(req.FileIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入内容或上传文件", "code": 400})
		return
	}

	// 文本内容限制（100KB）
	if len(req.Content) > 100*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文本内容超过 100KB 限制", "code": 400})
		return
	}

	// XSS安全检查：检测潜在的XSS攻击
	if utils.DetectPotentialXSS(req.Content) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容包含不安全字符", "code": 400})
		return
	}

	// 内容安全扫描
	securityResult := utils.ScanContent(req.Content)
	if !securityResult.IsSafe {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "内容包含不安全元素",
			"warnings": securityResult.Warnings,
			"code":     400,
		})
		return
	}

	// 对内容进行XSS防护消毒
	req.Content = utils.SanitizeContent(req.Content)
	req.Title = utils.SanitizeContent(req.Title)

	ip := c.ClientIP()

	// 检查 IP 限流
	count, err := h.db.CountByIP(ip, time.Now().Add(-h.ipWindow))
	if err == nil && count >= h.maxPerIP {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

	hourlyCount, err := h.db.CountByIP(ip, time.Now().Add(-time.Hour))
	if err == nil && hourlyCount >= 100 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

	// 检查总量限制
	total, err := h.db.TotalCount()
	if err == nil && total >= h.maxTotal {
		h.db.CleanExpired()
		total, _ = h.db.TotalCount()
		if total >= h.maxTotal {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "存储已满，请稍后再试", "code": 503})
			return
		}
	}

	// 设置默认值
	if req.Language == "" {
		// 自动检测语言
		req.Language = utils.DetectLanguage(req.Content)
		// 如果检测到的是 markdown，设置内容类型
		contentType := utils.DetectContentType(req.Content, req.Language)
		if contentType == "markdown" {
			req.Language = "markdown"
		}
	}
	if req.ExpiresIn <= 0 {
		req.ExpiresIn = 24
	}
	if req.ExpiresIn > 168 {
		req.ExpiresIn = 168
	}

	cfg := config.Get()

	// 检查文件并生成元数据
	var files []*FileMetadata
	hasVideo := false
	for _, fileID := range req.FileIDs {
		filePath := filepath.Join(pasteUploadDir, fileID)
		info, err := os.Stat(filePath)
		if err != nil {
			continue // 跳过不存在的文件
		}

		// 检测文件类型
		f, _ := os.Open(filePath)
		magicBytes := make([]byte, 16)
		f.Read(magicBytes)
		f.Close()

		detectedType := detectFileType(magicBytes)
		fileType := getFileCategory(detectedType)
		if fileType == "video" {
			hasVideo = true
		}

		files = append(files, &FileMetadata{
			Filename:     fileID,
			OriginalName: fileID,
			Type:         fileType,
			Size:         info.Size(),
			URL:          "/api/paste/files/" + fileID,
		})
	}

	// 视频访问次数限制
	if req.MaxViews <= 0 {
		if hasVideo {
			req.MaxViews = cfg.Paste.DefaultVideoMaxViews
		} else {
			req.MaxViews = 100
		}
	}

	// 管理员可以设置更多次数或永久
	if req.AdminPassword != "" {
		// 用户输入了管理员密码
		if cfg.Paste.AdminPassword == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "系统未设置管理员密码，请联系管理员在config.yaml中配置paste.admin_password", "code": 403})
			return
		}
		if req.AdminPassword != cfg.Paste.AdminPassword {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误", "code": 401})
			return
		}
		// 管理员模式，使用用户指定的值
		if req.MaxViews == 0 {
			req.MaxViews = 999999 // 近似永久
		}
	} else {
		// 非管理员模式，限制最大访问次数
		if hasVideo && req.MaxViews > cfg.Paste.DefaultVideoMaxViews {
			req.MaxViews = cfg.Paste.DefaultVideoMaxViews
		} else if !hasVideo && req.MaxViews > 1000 {
			req.MaxViews = 1000
		}
	}

	// 将文件元数据转为 JSON
	filesJSON := ""
	if len(files) > 0 {
		jsonBytes, _ := json.Marshal(files)
		filesJSON = string(jsonBytes)
	}

	paste := &models.Paste{
		Content:   req.Content,
		Title:     req.Title,
		Language:  req.Language,
		ExpiresAt: time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour),
		MaxViews:  req.MaxViews,
		CreatorIP: ip,
		Files:     filesJSON,
	}

	// 密码加密存储
	if req.Password != "" {
		hashedPassword, err := utils.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败", "code": 500})
			return
		}
		paste.Password = hashedPassword
	}

	if err := h.db.CreatePaste(paste); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         paste.ID,
		"expires_at": paste.ExpiresAt,
		"max_views":  paste.MaxViews,
	})
}

func (h *PasteHandler) Get(c *gin.Context) {
	id := c.Param("id")
	password := c.Query("password")

	paste, err := h.db.GetPaste(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// 检查是否过期
	if time.Now().After(paste.ExpiresAt) {
		h.cleanupPasteFiles(paste.Files)
		h.db.DeletePaste(id)
		c.JSON(http.StatusGone, gin.H{"error": "该分享已过期", "code": 410})
		return
	}

	// 检查访问次数
	if paste.Views >= paste.MaxViews {
		h.cleanupPasteFiles(paste.Files)
		h.db.DeletePaste(id)
		c.JSON(http.StatusGone, gin.H{"error": "该分享已达到最大访问次数", "code": 410})
		return
	}

	// 检查密码
	if paste.Password != "" {
		if password == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":        "需要密码",
				"code":         401,
				"has_password": true,
			})
			return
		}
		if !utils.VerifyPassword(password, paste.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
			return
		}
	}

	// 增加访问次数
	h.db.IncrementViews(id)
	paste.Views++

	// 解析文件 JSON
	var files []*FileMetadata
	if paste.Files != "" {
		json.Unmarshal([]byte(paste.Files), &files)
	}

	// 对输出进行安全处理
	safeTitle := utils.SanitizeForAttribute(paste.Title)

	// 检测内容类型
	contentType := utils.DetectContentType(paste.Content, paste.Language)
	if contentType == "" {
		contentType = string(ContentTypeText)
	}

	c.JSON(http.StatusOK, PasteResponse{
		ID:          paste.ID,
		Title:       safeTitle,
		Language:    paste.Language,
		ContentType: contentType,
		Content:     paste.Content,
		ExpiresAt:   paste.ExpiresAt,
		MaxViews:    paste.MaxViews,
		Views:       paste.Views,
		CreatedAt:   paste.CreatedAt,
		HasPassword: paste.Password != "",
		Files:       files,
	})
}

func (h *PasteHandler) GetInfo(c *gin.Context) {
	id := c.Param("id")

	paste, err := h.db.GetPaste(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// 检查是否过期
	if time.Now().After(paste.ExpiresAt) {
		h.db.DeletePaste(id)
		c.JSON(http.StatusGone, gin.H{"error": "该分享已过期", "code": 410})
		return
	}

	// 检测内容类型
	contentType := utils.DetectContentType(paste.Content, paste.Language)
	if contentType == "" {
		contentType = string(ContentTypeText)
	}

	c.JSON(http.StatusOK, PasteResponse{
		ID:          paste.ID,
		Title:       paste.Title,
		Language:    paste.Language,
		ContentType: contentType,
		ExpiresAt:   paste.ExpiresAt,
		MaxViews:    paste.MaxViews,
		Views:       paste.Views,
		CreatedAt:   paste.CreatedAt,
		HasPassword: paste.Password != "",
	})
}

// AdminListPastes 管理员获取所有粘贴板列表
func (h *PasteHandler) AdminListPastes(c *gin.Context) {
	cfg := config.Get()
	adminPassword := c.Query("admin_password")

	// 验证管理员密码
	if cfg.Paste.AdminPassword == "" || adminPassword != cfg.Paste.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误", "code": 401})
		return
	}

	// 分页参数
	limit := 50
	offset := 0

	pastes, err := h.db.GetAllPastes(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取列表失败", "code": 500})
		return
	}

	// 构建响应
	var responses []gin.H
	for _, paste := range pastes {
		var files []*FileMetadata
		if paste.Files != "" {
			json.Unmarshal([]byte(paste.Files), &files)
		}

		responses = append(responses, gin.H{
			"id":           paste.ID,
			"title":        paste.Title,
			"language":     paste.Language,
			"expires_at":   paste.ExpiresAt,
			"max_views":    paste.MaxViews,
			"views":        paste.Views,
			"created_at":   paste.CreatedAt,
			"has_password": paste.Password != "",
			"has_content":  paste.Content != "",
			"file_count":   len(files),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"pastes": responses,
		"total":  len(responses),
	})
}

// AdminGetPaste 管理员获取指定粘贴板详情
func (h *PasteHandler) AdminGetPaste(c *gin.Context) {
	cfg := config.Get()
	adminPassword := c.Query("admin_password")

	// 验证管理员密码
	if cfg.Paste.AdminPassword == "" || adminPassword != cfg.Paste.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误", "code": 401})
		return
	}

	id := c.Param("id")
	paste, err := h.db.GetPaste(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// 解析文件
	var files []*FileMetadata
	if paste.Files != "" {
		json.Unmarshal([]byte(paste.Files), &files)
	}

	// 检测内容类型
	contentType := utils.DetectContentType(paste.Content, paste.Language)
	if contentType == "" {
		contentType = string(ContentTypeText)
	}

	c.JSON(http.StatusOK, PasteResponse{
		ID:          paste.ID,
		Title:       paste.Title,
		Language:    paste.Language,
		ContentType: contentType,
		Content:     paste.Content,
		ExpiresAt:   paste.ExpiresAt,
		MaxViews:    paste.MaxViews,
		Views:       paste.Views,
		CreatedAt:   paste.CreatedAt,
		HasPassword: paste.Password != "",
		Files:       files,
	})
}

// AdminUpdatePaste 管理员更新粘贴板
func (h *PasteHandler) AdminUpdatePaste(c *gin.Context) {
	cfg := config.Get()

	var req struct {
		AdminPassword string `json:"admin_password" binding:"required"`
		ExpiresIn     int    `json:"expires_in"` // 延长小时数
		MaxViews      int    `json:"max_views"`  // 新的最大访问次数
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// 验证管理员密码
	if cfg.Paste.AdminPassword == "" || req.AdminPassword != cfg.Paste.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误", "code": 401})
		return
	}

	id := c.Param("id")
	paste, err := h.db.GetPaste(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// 更新过期时间
	newExpiresAt := paste.ExpiresAt
	if req.ExpiresIn > 0 {
		newExpiresAt = time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
	}

	// 更新最大访问次数
	newMaxViews := paste.MaxViews
	if req.MaxViews > 0 {
		newMaxViews = req.MaxViews
	}

	if err := h.db.UpdatePaste(id, newExpiresAt, newMaxViews); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "更新成功",
		"expires_at": newExpiresAt,
		"max_views":  newMaxViews,
	})
}

// AdminDeletePaste 管理员删除粘贴板
func (h *PasteHandler) AdminDeletePaste(c *gin.Context) {
	cfg := config.Get()
	adminPassword := c.Query("admin_password")

	// 验证管理员密码
	if cfg.Paste.AdminPassword == "" || adminPassword != cfg.Paste.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误", "code": 401})
		return
	}

	id := c.Param("id")
	paste, err := h.db.GetPaste(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// 删除关联文件
	h.cleanupPasteFiles(paste.Files)

	// 删除数据库记录
	if err := h.db.DeletePaste(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ServeFile 提供文件访问
func (h *PasteHandler) ServeFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(pasteUploadDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在", "code": 404})
		return
	}

	c.File(filePath)
}

// InitChunkUpload 初始化分片上传
func (h *PasteHandler) InitChunkUpload(c *gin.Context) {
	var req struct {
		FileName    string `json:"file_name" binding:"required"`
		FileSize    int64  `json:"file_size" binding:"required"`
		ChunkSize   int64  `json:"chunk_size" binding:"required"`
		TotalChunks int    `json:"total_chunks" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	cfg := config.Get()

	// 检查文件大小
	if req.FileSize > cfg.Paste.MaxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("文件大小超过限制 (最大 %dMB)", cfg.Paste.MaxFileSize/1024/1024),
			"code":  400,
		})
		return
	}

	// 生成文件ID
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "code": 500})
		return
	}
	fileID := hex.EncodeToString(randomBytes)

	// 确保分片目录存在
	chunkPath := filepath.Join(chunkDir, fileID)
	if err := os.MkdirAll(chunkPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "code": 500})
		return
	}

	// 创建分片上传信息
	uploadInfo := &ChunkUploadInfo{
		FileID:         fileID,
		FileName:       req.FileName,
		TotalChunks:    req.TotalChunks,
		ChunkSize:      req.ChunkSize,
		FileSize:       req.FileSize,
		UploadedChunks: []int{},
		CreatedAt:      time.Now(),
	}

	chunkUploadsMu.Lock()
	chunkUploads[fileID] = uploadInfo
	chunkUploadsMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"file_id": fileID,
		"message": "分片上传初始化成功",
	})
}

// UploadChunk 上传分片
func (h *PasteHandler) UploadChunk(c *gin.Context) {
	fileID := c.Param("file_id")
	chunkIndex := c.PostForm("chunk_index")

	if fileID == "" || chunkIndex == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数", "code": 400})
		return
	}

	// 获取上传信息
	chunkUploadsMu.RLock()
	uploadInfo, exists := chunkUploads[fileID]
	chunkUploadsMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "上传会话不存在", "code": 404})
		return
	}

	// 检查是否已超时(24小时)
	if time.Since(uploadInfo.CreatedAt) > 24*time.Hour {
		h.CleanupChunkUpload(fileID)
		c.JSON(http.StatusGone, gin.H{"error": "上传会话已过期", "code": 410})
		return
	}

	// 接收分片数据
	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取分片失败", "code": 400})
		return
	}
	defer file.Close()

	// 保存分片
	chunkPath := filepath.Join(chunkDir, fileID, chunkIndex)
	out, err := os.Create(chunkPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存分片失败", "code": 500})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存分片失败", "code": 500})
		return
	}

	// 更新已上传分片列表
	uploadInfo.mu.Lock()
	var chunkIdx int
	fmt.Sscanf(chunkIndex, "%d", &chunkIdx)
	uploadInfo.UploadedChunks = append(uploadInfo.UploadedChunks, chunkIdx)
	uploadInfo.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message":         "分片上传成功",
		"uploaded_chunks": len(uploadInfo.UploadedChunks),
		"total_chunks":    uploadInfo.TotalChunks,
	})
}

// MergeChunks 合并分片
func (h *PasteHandler) MergeChunks(c *gin.Context) {
	fileID := c.Param("file_id")

	// 获取上传信息
	chunkUploadsMu.RLock()
	uploadInfo, exists := chunkUploads[fileID]
	chunkUploadsMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "上传会话不存在", "code": 404})
		return
	}

	// 检查所有分片是否都已上传
	if len(uploadInfo.UploadedChunks) != uploadInfo.TotalChunks {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "分片未全部上传",
			"uploaded_chunks": len(uploadInfo.UploadedChunks),
			"total_chunks":    uploadInfo.TotalChunks,
			"code":            400,
		})
		return
	}

	// 确保上传目录存在
	if err := os.MkdirAll(pasteUploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "code": 500})
		return
	}

	// 读取第一个分片检测文件类型
	firstChunkPath := filepath.Join(chunkDir, fileID, "0")
	firstChunk, err := os.ReadFile(firstChunkPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取分片失败", "code": 500})
		return
	}

	magicBytes := firstChunk
	if len(firstChunk) > 16 {
		magicBytes = firstChunk[:16]
	}

	detectedType := detectFileType(magicBytes)
	if detectedType == "" {
		h.CleanupChunkUpload(fileID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型", "code": 400})
		return
	}

	fileCategory := getFileCategory(detectedType)

	// 确定最终文件扩展名
	ext := strings.ToLower(filepath.Ext(uploadInfo.FileName))
	if ext == "" {
		ext = getExtFromMimeType(detectedType)
	}

	finalFilename := fmt.Sprintf("%s%s", fileID, ext)
	finalPath := filepath.Join(pasteUploadDir, finalFilename)

	// 创建最终文件
	finalFile, err := os.Create(finalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件失败", "code": 500})
		return
	}
	defer finalFile.Close()

	// 按顺序合并分片
	for i := 0; i < uploadInfo.TotalChunks; i++ {
		chunkPath := filepath.Join(chunkDir, fileID, fmt.Sprintf("%d", i))
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			h.CleanupChunkUpload(fileID)
			os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取分片失败", "code": 500})
			return
		}

		if _, err := finalFile.Write(chunkData); err != nil {
			h.CleanupChunkUpload(fileID)
			os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "合并分片失败", "code": 500})
			return
		}
	}

	// 清理分片临时文件
	h.CleanupChunkUpload(fileID)

	// 获取文件大小
	fileInfo, _ := os.Stat(finalPath)
	fileSize := int64(0)
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	// 返回文件信息
	fileURL := "/api/paste/files/" + finalFilename
	c.JSON(http.StatusOK, gin.H{
		"id":            finalFilename,
		"url":           fileURL,
		"filename":      finalFilename,
		"original_name": uploadInfo.FileName,
		"type":          fileCategory,
		"size":          fileSize,
		"message":       "文件合并成功",
	})
}

// CheckChunkStatus 检查分片上传状态
func (h *PasteHandler) CheckChunkStatus(c *gin.Context) {
	fileID := c.Param("file_id")

	chunkUploadsMu.RLock()
	uploadInfo, exists := chunkUploads[fileID]
	chunkUploadsMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "上传会话不存在", "code": 404})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_id":         uploadInfo.FileID,
		"file_name":       uploadInfo.FileName,
		"total_chunks":    uploadInfo.TotalChunks,
		"uploaded_chunks": uploadInfo.UploadedChunks,
		"completed":       len(uploadInfo.UploadedChunks) == uploadInfo.TotalChunks,
	})
}

// CleanupChunkUpload 清理分片上传临时文件
func (h *PasteHandler) CleanupChunkUpload(fileID string) {
	chunkUploadsMu.Lock()
	delete(chunkUploads, fileID)
	chunkUploadsMu.Unlock()

	// 删除临时分片目录
	chunkPath := filepath.Join(chunkDir, fileID)
	os.RemoveAll(chunkPath)
}

// cleanupPasteFiles 清理 paste 关联的文件
func (h *PasteHandler) cleanupPasteFiles(filesJSON string) {
	if filesJSON == "" {
		return
	}

	var files []*FileMetadata
	if err := json.Unmarshal([]byte(filesJSON), &files); err != nil {
		return
	}

	for _, file := range files {
		filePath := filepath.Join(pasteUploadDir, file.Filename)
		os.Remove(filePath)
	}
}

// AnalyzeCode 分析代码内容（使用 analysis.go 中的 AnalyzeCodeRequest）
func (h *PasteHandler) AnalyzeCode(c *gin.Context) {
	var req AnalyzeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// 检查内容大小
	if len(req.Content) > 500*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容超过 500KB 限制", "code": 400})
		return
	}

	// 如果未指定语言，自动检测
	language := req.Language
	if language == "" {
		language = utils.DetectLanguage(req.Content)
	}

	// 分析代码
	result := utils.AnalyzeCode(req.Content, language)

	c.JSON(http.StatusOK, gin.H{
		"language":      result.Language,
		"lines":         result.Lines,
		"code_lines":    result.CodeLines,
		"comment_lines": result.CommentLines,
		"blank_lines":   result.BlankLines,
		"functions":     result.Functions,
		"classes":       result.Classes,
		"imports":       result.Imports,
		"summary":       result.Summary,
	})
}

// AnalyzeFile 分析上传的文件内容
func (h *PasteHandler) AnalyzeFile(c *gin.Context) {
	fileID := c.Param("file_id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件ID", "code": 400})
		return
	}

	filePath := filepath.Join(pasteUploadDir, fileID)

	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在", "code": 404})
		return
	}

	// 限制文件大小
	if info.Size() > 500*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大，无法分析", "code": 400})
		return
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败", "code": 500})
		return
	}

	// 检测文件类型
	magicBytes := content[:min(16, len(content))]
	detectedType := detectFileType(magicBytes)

	// 检测语言
	language := utils.DetectLanguage(string(content))

	// 分析代码
	result := utils.AnalyzeCode(string(content), language)

	c.JSON(http.StatusOK, gin.H{
		"filename":      fileID,
		"file_type":    detectedType,
		"language":      result.Language,
		"lines":         result.Lines,
		"code_lines":    result.CodeLines,
		"comment_lines": result.CommentLines,
		"blank_lines":   result.BlankLines,
		"functions":     result.Functions,
		"classes":       result.Classes,
		"imports":       result.Imports,
		"summary":       result.Summary,
		"size":          info.Size(),
	})
}

// GetSupportedLanguages 获取支持的语言列表
func (h *PasteHandler) GetSupportedLanguages(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"languages": SupportedLanguages,
		"count":     len(SupportedLanguages),
	})
}

// GetSupportedContentTypes 获取支持的内容类型列表
func (h *PasteHandler) GetSupportedContentTypes(c *gin.Context) {
	contentTypes := []gin.H{
		{"type": "text", "name": "纯文本", "icon": "📝"},
		{"type": "code", "name": "代码", "icon": "💻"},
		{"type": "markdown", "name": "Markdown", "icon": "📋"},
		{"type": "json", "name": "JSON", "icon": "🔧"},
		{"type": "html", "name": "HTML", "icon": "🌐"},
		{"type": "xml", "name": "XML", "icon": "📰"},
		{"type": "sql", "name": "SQL", "icon": "🗄️"},
		{"type": "log", "name": "日志", "icon": "📜"},
	}
	c.JSON(http.StatusOK, gin.H{
		"content_types": contentTypes,
		"count":         len(contentTypes),
	})
}

// GetStats 获取粘贴板统计信息
func (h *PasteHandler) GetStats(c *gin.Context) {
	total, err := h.db.TotalCount()
	if err != nil {
		total = 0
	}

	// 获取今日创建数
	today := time.Now().Truncate(24 * time.Hour)
	todayCount, _ := h.db.CountByIP("", today) // 这会返回全部，不准确，下面改进

	// 简单返回统计信息
	stats := gin.H{
		"total_pastes":    total,
		"today_creates":   todayCount,
		"max_file_size":   config.Get().Paste.MaxFileSize,
		"max_content_size": config.Get().Limits.PasteMaxContentSize,
	}

	c.JSON(http.StatusOK, stats)
}

// SearchPastes 搜索粘贴板
func (h *PasteHandler) SearchPastes(c *gin.Context) {
	keyword := c.Query("keyword")
	language := c.Query("language")
	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// 获取所有粘贴板进行筛选（简单实现）
	pastes, err := h.db.GetAllPastes(1000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取列表失败", "code": 500})
		return
	}

	var results []*models.Paste
	for _, p := range pastes {
		match := true
		if keyword != "" && !strings.Contains(strings.ToLower(p.Content), strings.ToLower(keyword)) &&
			!strings.Contains(strings.ToLower(p.Title), strings.ToLower(keyword)) {
			match = false
		}
		if language != "" && p.Language != language {
			match = false
		}
		if match {
			results = append(results, p)
		}
	}

	// 分页
	start := offset
	end := offset + limit
	if start > len(results) {
		results = []*models.Paste{}
	} else {
		if end > len(results) {
			end = len(results)
		}
		results = results[start:end]
	}

	var responses []gin.H
	for _, p := range results {
		responses = append(responses, gin.H{
			"id":          p.ID,
			"title":       p.Title,
			"language":    p.Language,
			"created_at":  p.CreatedAt,
			"expires_at":  p.ExpiresAt,
			"views":       p.Views,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"pastes":  responses,
		"total":   len(results),
		"limit":   limit,
		"offset":  offset,
	})
}

// ScanContent 扫描内容安全（公开API）
func (h *PasteHandler) ScanContent(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// 内容限制
	if len(req.Content) > 500*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容超过 500KB 限制", "code": 400})
		return
	}

	// 执行安全扫描
	result := utils.ScanContent(req.Content)

	// 检测语言
	language := utils.DetectLanguage(req.Content)

	// 检测内容类型
	contentType := utils.DetectContentType(req.Content, language)

	c.JSON(http.StatusOK, gin.H{
		"is_safe":       result.IsSafe,
		"has_virus":     result.HasVirus,
		"has_suspicious": result.HasSuspiciousURL,
		"warnings":      result.Warnings,
		"language":      language,
		"content_type":  contentType,
	})
}

// ValidateFile 验证文件安全性（公开API）
func (h *PasteHandler) ValidateFile(c *gin.Context) {
	fileID := c.Param("file_id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件ID", "code": 400})
		return
	}

	filePath := filepath.Join(pasteUploadDir, fileID)

	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在", "code": 404})
		return
	}

	// 限制文件大小
	if info.Size() > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大，无法验证", "code": 400})
		return
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败", "code": 500})
		return
	}

	// 验证文件
	isValid, reason := utils.ValidateFilename(fileID)
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": reason, "code": 400})
		return
	}

	// 扫描文件内容
	result := utils.ScanFileContent(data)

	// 获取文件分类信息
	categoryInfo := GetFileCategoryInfo(detectFileType(data[:min(16, len(data))]))

	c.JSON(http.StatusOK, gin.H{
		"filename":      fileID,
		"size":          info.Size(),
		"is_safe":       result.IsSafe,
		"has_virus":     result.HasVirus,
		"warnings":      result.Warnings,
		"category":      categoryInfo.Category,
		"category_icon": categoryInfo.Icon,
	})
}


