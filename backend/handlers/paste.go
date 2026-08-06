package handlers

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/state"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

const (
	pasteUploadDir = "./data/paste_files"
	chunkDir       = "./data/paste_chunks" // 分片临时目录
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
	db             *models.DB
	store          state.TransientStore
	chunkUploadTTL time.Duration
	maxTotal       int
	maxPerIP       int
	ipWindow       time.Duration
}

// NewPasteHandler 创建粘贴板处理器
func NewPasteHandler(db *models.DB, cfg *config.Config, store state.TransientStore) *PasteHandler {
	chunkTTL := time.Duration(cfg.Redis.ChunkUploadTTLHours) * time.Hour
	if chunkTTL <= 0 {
		chunkTTL = 24 * time.Hour
	}

	return &PasteHandler{
		db:             db,
		store:          store,
		chunkUploadTTL: chunkTTL,
		maxTotal:       10000, // 最多存储 10000 条
		maxPerIP:       10,    // 每 IP 每分钟最多 10 条（与中间件限流一致）
		ipWindow:       time.Minute,
	}
}

func (h *PasteHandler) saveChunkUpload(upload *state.ChunkUploadInfo) error {
	return h.store.SaveChunkUpload(context.Background(), upload, h.chunkUploadTTL)
}

func (h *PasteHandler) getChunkUpload(fileID string) (*state.ChunkUploadInfo, bool) {
	upload, err := h.store.GetChunkUpload(context.Background(), fileID)
	if err != nil {
		return nil, false
	}
	return upload, true
}

func (h *PasteHandler) deleteChunkUpload(fileID string) error {
	return h.store.DeleteChunkUpload(context.Background(), fileID)
}

func (h *PasteHandler) markChunkUploaded(fileID string, chunkIndex int) (int, error) {
	return h.store.MarkChunkUploaded(context.Background(), fileID, chunkIndex, h.chunkUploadTTL)
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
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Language    string          `json:"language"`
	ContentType string          `json:"content_type"`
	Content     string          `json:"content,omitempty"`
	ExpiresAt   time.Time       `json:"expires_at"`
	MaxViews    int             `json:"max_views"`
	Views       int             `json:"views"`
	CreatedAt   time.Time       `json:"created_at"`
	HasPassword bool            `json:"has_password"`
	Files       []*FileMetadata `json:"files,omitempty"`
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

	// 生成随机文件 ID
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "code": 500})
		return
	}
	fileID := hex.EncodeToString(randomBytes)

	// 不支持的文件类型：透明打包成 zip 后存储，而不是直接失败
	if detectedType == "" {
		file.Seek(0, 0)
		finalFilename, zipName, zipSize, err := zipUnsupportedFile(file, header.Filename, fileID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败", "code": 500})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"id":            finalFilename,
			"url":           "/api/paste/files/" + finalFilename,
			"filename":      finalFilename,
			"original_name": zipName,
			"type":          "archive",
			"size":          zipSize,
			"zipped":        true,
		})
		return
	}

	// 确定文件类别
	fileCategory := getFileCategory(detectedType)

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = getExtFromMimeType(detectedType)
	}
	filename := fmt.Sprintf("%s%s", fileID, ext)

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

	ip := c.ClientIP()
	paste, err := h.createPasteCore(c.Request.Context(), req, ip, createPasteOptions{})
	if err != nil {
		status := http.StatusInternalServerError
		if pe, ok := err.(*pasteError); ok {
			status = pe.status
		}
		c.JSON(status, gin.H{"error": err.Error(), "code": status})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         paste.ID,
		"expires_at": paste.ExpiresAt,
		"max_views":  paste.MaxViews,
	})
}

// createPasteCore 是 Create handler 的核心逻辑,无 c.JSON / 无 IP/storage 限流,
// 返回 *models.Paste + error。供 skills/内部调用复用,与 HTTP handler 行为对齐。
//
// opts 控制可选行为:
//   - MaxContentBytes:content 长度上限(0 = 不限,默认 100KB)
//   - SkipFiles:true 时忽略 req.FileIDs(技能调用没有 multipart 上传链路)
//   - SkipRateLimit:true 时跳过 IP/storage 上限(由调用方自行保证)
func (h *PasteHandler) createPasteCore(ctx context.Context, req CreatePasteRequest, ip string, opts createPasteOptions) (*models.Paste, error) {
	// 必须有内容或文件(skill 路径通常没有文件)
	if req.Content == "" && len(req.FileIDs) == 0 && !opts.SkipFiles {
		return nil, errPaste("请输入内容或上传文件", 400)
	}

	// 文本内容限制
	maxContent := opts.MaxContentBytes
	if maxContent == 0 {
		maxContent = 100 * 1024
	}
	if int64(len(req.Content)) > maxContent {
		return nil, errPaste(fmt.Sprintf("文本内容超过 %d 限制", maxContent), 400)
	}

	// XSS 安全检查
	if utils.DetectPotentialXSS(req.Content) {
		return nil, errPaste("内容包含不安全字符", 400)
	}

	// 内容安全扫描
	securityResult := utils.ScanContent(req.Content)
	if !securityResult.IsSafe {
		return nil, errPaste("内容包含不安全元素: "+strings.Join(securityResult.Warnings, "; "), 400)
	}

	cfg := config.Get()

	if !opts.SkipRateLimit {
		// IP 限流
		count, err := h.db.CountByIP(ip, time.Now().Add(-h.ipWindow))
		if err == nil && count >= h.maxPerIP {
			return nil, errPaste("创建过于频繁，请稍后再试", 429)
		}
		hourlyCount, err := h.db.CountByIP(ip, time.Now().Add(-time.Hour))
		if err == nil && hourlyCount >= 100 {
			return nil, errPaste("创建过于频繁，请稍后再试", 429)
		}

		// 总量限制
		total, err := h.db.TotalCount()
		if err == nil && total >= h.maxTotal {
			h.db.CleanExpired()
			total, _ = h.db.TotalCount()
			if total >= h.maxTotal {
				return nil, errPaste("存储已满，请稍后再试", 503)
			}
		}
	}

	// 默认语言
	if req.Language == "" {
		req.Language = utils.DetectLanguage(req.Content)
		contentType := utils.DetectContentType(req.Content, req.Language)
		if contentType == "markdown" {
			req.Language = "markdown"
		}
	}

	// 内容消毒(HTML 保留原始标记供 iframe 渲染)
	if req.Language != "html" {
		req.Content = utils.SanitizeContent(req.Content)
	}
	req.Title = utils.SanitizeContent(req.Title)

	// 过期时间:默认 24h,硬上限 168h(7d)
	if req.ExpiresIn <= 0 {
		req.ExpiresIn = 24
	}
	if req.ExpiresIn > 168 {
		req.ExpiresIn = 168
	}

	// 处理文件(skill 路径可跳过)
	var files []*FileMetadata
	hasVideo := false
	if !opts.SkipFiles {
		for _, fileID := range req.FileIDs {
			filePath := filepath.Join(pasteUploadDir, fileID)
			info, err := os.Stat(filePath)
			if err != nil {
				continue
			}
			f, err := os.Open(filePath)
			if err != nil {
				continue
			}
			magicBytes := make([]byte, 16)
			_, _ = f.Read(magicBytes)
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
	}

	// 默认 max_views
	if req.MaxViews <= 0 {
		if hasVideo {
			req.MaxViews = cfg.Paste.DefaultVideoMaxViews
			if req.MaxViews <= 0 {
				req.MaxViews = 10
			}
		} else {
			req.MaxViews = 100
		}
	}

	// 管理员模式(允许更高的 max_views / 永久)
	isAdmin := false
	if req.AdminPassword != "" {
		if cfg.Paste.AdminPassword == "" {
			return nil, errPaste("系统未设置管理员密码", 403)
		}
		if req.AdminPassword != cfg.Paste.AdminPassword {
			return nil, errPaste("管理员密码错误", 401)
		}
		isAdmin = true
		if req.MaxViews == 0 {
			req.MaxViews = 999999 // 近似永久
		}
	}

	// 非管理员 max_views 上限
	if !isAdmin {
		if hasVideo && req.MaxViews > cfg.Paste.DefaultVideoMaxViews {
			req.MaxViews = cfg.Paste.DefaultVideoMaxViews
		} else if !hasVideo && req.MaxViews > 1000 {
			req.MaxViews = 1000
		}
	}

	// 文件元数据 JSON
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

	// 密码加密
	if req.Password != "" {
		hashedPassword, err := utils.HashPassword(req.Password)
		if err != nil {
			return nil, errPaste("密码处理失败: "+err.Error(), 500)
		}
		paste.Password = hashedPassword
	}

	if err := h.db.CreatePaste(paste); err != nil {
		return nil, errPaste("创建失败: "+err.Error(), 500)
	}
	return paste, nil
}

// createPasteOptions createPasteCore 的可选行为
type createPasteOptions struct {
	MaxContentBytes int64 // 0 = 默认 100KB
	SkipFiles       bool  // true 忽略 req.FileIDs
	SkipRateLimit   bool  // true 跳过 IP/storage 限流
}

// errPaste 把 HTTP 状态码附在 error 上,handler 据此选 status
type pasteError struct {
	msg    string
	status int
}

func (e *pasteError) Error() string { return e.msg }
func errPaste(msg string, status int) error {
	return &pasteError{msg: msg, status: status}
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

	// 增加访问次数（原子,超过 max_views 直接拒绝）
	if err := h.db.IncrementViews(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.cleanupPasteFiles(paste.Files)
			h.db.DeletePaste(id)
			c.JSON(http.StatusGone, gin.H{"error": "该分享已达到最大访问次数", "code": 410})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新访问次数失败", "code": 500})
		return
	}
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
	filename := filepath.Base(c.Param("filename"))
	// 显式拒绝路径分隔符与特殊名(防御 URL 编码绕过 ../ 等变体)
	if filename == "" || filename == "." || filename == "/" || filename == ".." ||
		strings.ContainsAny(filename, `/\`) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法文件名", "code": 400})
		return
	}
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

	fileID, err := h.initChunkUploadCore(req.FileName, req.FileSize, req.ChunkSize, req.TotalChunks)
	if err != nil {
		status := http.StatusInternalServerError
		if pe, ok := err.(*pasteError); ok {
			status = pe.status
		}
		c.JSON(status, gin.H{"error": err.Error(), "code": status})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_id": fileID,
		"message": "分片上传初始化成功",
	})
}

// UploadChunk 上传分片
func (h *PasteHandler) UploadChunk(c *gin.Context) {
	fileID := c.Param("file_id")
	chunkIndexStr := c.PostForm("chunk_index")

	if fileID == "" || chunkIndexStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数", "code": 400})
		return
	}

	var chunkIdx int
	if _, err := fmt.Sscanf(chunkIndexStr, "%d", &chunkIdx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chunk_index 格式错误", "code": 400})
		return
	}

	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取分片失败", "code": 400})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取分片失败", "code": 500})
		return
	}

	uploadedCount, totalChunks, err := h.uploadChunkCore(fileID, chunkIdx, data)
	if err != nil {
		status := http.StatusInternalServerError
		if pe, ok := err.(*pasteError); ok {
			status = pe.status
		}
		c.JSON(status, gin.H{"error": err.Error(), "code": status})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "分片上传成功",
		"uploaded_chunks": uploadedCount,
		"total_chunks":    totalChunks,
	})
}

// MergeChunks 合并分片
func (h *PasteHandler) MergeChunks(c *gin.Context) {
	fileID := c.Param("file_id")

	fm, err := h.mergeChunksCore(fileID)
	if err != nil {
		status := http.StatusInternalServerError
		if pe, ok := err.(*pasteError); ok {
			status = pe.status
		}
		c.JSON(status, gin.H{"error": err.Error(), "code": status})
		return
	}

	resp := gin.H{
		"id":            fm.Filename,
		"url":           fm.URL,
		"filename":      fm.Filename,
		"original_name": fm.OriginalName,
		"type":          fm.Type,
		"size":          fm.Size,
		"message":       "文件合并成功",
	}
	if fm.Type == "archive" {
		resp["zipped"] = true
	}
	c.JSON(http.StatusOK, resp)
}

// CheckChunkStatus 检查分片上传状态
func (h *PasteHandler) CheckChunkStatus(c *gin.Context) {
	fileID := c.Param("file_id")

	uploadInfo, exists := h.getChunkUpload(fileID)

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
	_ = h.deleteChunkUpload(fileID)

	// 删除临时分片目录
	chunkPath := filepath.Join(chunkDir, fileID)
	os.RemoveAll(chunkPath)
}

// ============================================================
// 分片上传 core helpers(供 HTTP handler 与 skills 复用)
// 接受 []byte,不依赖 multipart.FileHeader,便于 JSON-RPC / base64 场景
// ============================================================

// initChunkUploadCore 初始化一个分片上传会话,返回 fileID
func (h *PasteHandler) initChunkUploadCore(fileName string, fileSize, chunkSize int64, totalChunks int) (string, error) {
	cfg := config.Get()

	if fileSize > cfg.Paste.MaxFileSize {
		return "", errPaste(fmt.Sprintf("文件大小超过限制 (最大 %dMB)", cfg.Paste.MaxFileSize/1024/1024), 400)
	}
	if totalChunks <= 0 {
		return "", errPaste("total_chunks 必须 > 0", 400)
	}
	if chunkSize <= 0 {
		return "", errPaste("chunk_size 必须 > 0", 400)
	}

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", errPaste("服务器错误", 500)
	}
	fileID := hex.EncodeToString(randomBytes)

	chunkPath := filepath.Join(chunkDir, fileID)
	if err := os.MkdirAll(chunkPath, 0755); err != nil {
		return "", errPaste("服务器错误", 500)
	}

	uploadInfo := &state.ChunkUploadInfo{
		FileID:         fileID,
		FileName:       fileName,
		TotalChunks:    totalChunks,
		ChunkSize:      chunkSize,
		FileSize:       fileSize,
		UploadedChunks: []int{},
		CreatedAt:      time.Now(),
	}
	if err := h.saveChunkUpload(uploadInfo); err != nil {
		return "", errPaste("保存上传会话失败", 500)
	}
	return fileID, nil
}

// uploadChunkCore 保存单个分片;返回 (已上传分片数, 总分片数)
// 注意:chunkSize 上限在这里强制 — 每个分片不允许超过 cfg.Paste.MaxFileSize
func (h *PasteHandler) uploadChunkCore(fileID string, chunkIndex int, data []byte) (int, int, error) {
	if fileID == "" {
		return 0, 0, errPaste("file_id 必填", 400)
	}
	if chunkIndex < 0 {
		return 0, 0, errPaste("chunk_index 必须 >= 0", 400)
	}
	cfg := config.Get()
	if int64(len(data)) > cfg.Paste.MaxFileSize {
		return 0, 0, errPaste("分片过大", 400)
	}

	uploadInfo, exists := h.getChunkUpload(fileID)
	if !exists {
		return 0, 0, errPaste("上传会话不存在", 404)
	}
	if time.Since(uploadInfo.CreatedAt) > h.chunkUploadTTL {
		h.CleanupChunkUpload(fileID)
		return 0, 0, errPaste("上传会话已过期", 410)
	}

	chunkPath := filepath.Join(chunkDir, fileID, fmt.Sprintf("%d", chunkIndex))
	if err := os.MkdirAll(filepath.Dir(chunkPath), 0755); err != nil {
		return 0, 0, errPaste("服务器错误", 500)
	}
	if err := os.WriteFile(chunkPath, data, 0644); err != nil {
		return 0, 0, errPaste("保存分片失败", 500)
	}

	uploadedCount, err := h.markChunkUploaded(fileID, chunkIndex)
	if err != nil {
		return 0, 0, errPaste("更新上传状态失败", 500)
	}
	return uploadedCount, uploadInfo.TotalChunks, nil
}

// mergeChunksCore 合并分片为最终文件,返回 *FileMetadata(包含 filename / url / type / size)
// 支持的格式直接落 pasteUploadDir;不支持的格式透明打成 zip 后保存
func (h *PasteHandler) mergeChunksCore(fileID string) (*FileMetadata, error) {
	uploadInfo, exists := h.getChunkUpload(fileID)
	if !exists {
		return nil, errPaste("上传会话不存在", 404)
	}
	if len(uploadInfo.UploadedChunks) != uploadInfo.TotalChunks {
		return nil, errPaste(
			fmt.Sprintf("分片未全部上传 (已上传 %d / 共 %d)", len(uploadInfo.UploadedChunks), uploadInfo.TotalChunks),
			400)
	}

	if err := os.MkdirAll(pasteUploadDir, 0755); err != nil {
		return nil, errPaste("服务器错误", 500)
	}

	// 读取第一个分片检测文件类型
	firstChunkPath := filepath.Join(chunkDir, fileID, "0")
	firstChunk, err := os.ReadFile(firstChunkPath)
	if err != nil {
		return nil, errPaste("读取分片失败", 500)
	}
	magicBytes := firstChunk
	if len(firstChunk) > 16 {
		magicBytes = firstChunk[:16]
	}
	detectedType := detectFileType(magicBytes)

	// 不支持的文件类型:打成 zip
	if detectedType == "" {
		var readers []io.Reader
		var closers []io.Closer
		for i := 0; i < uploadInfo.TotalChunks; i++ {
			p := filepath.Join(chunkDir, fileID, fmt.Sprintf("%d", i))
			f, err := os.Open(p)
			if err != nil {
				for _, cl := range closers {
					cl.Close()
				}
				h.CleanupChunkUpload(fileID)
				return nil, errPaste("读取分片失败", 500)
			}
			readers = append(readers, f)
			closers = append(closers, f)
		}
		finalFilename, zipName, zipSize, err := zipUnsupportedFile(io.MultiReader(readers...), uploadInfo.FileName, fileID)
		for _, cl := range closers {
			cl.Close()
		}
		h.CleanupChunkUpload(fileID)
		if err != nil {
			return nil, errPaste("保存文件失败", 500)
		}
		return &FileMetadata{
			Filename:     finalFilename,
			OriginalName: zipName,
			Type:         "archive",
			Size:         zipSize,
			URL:          "/api/paste/files/" + finalFilename,
		}, nil
	}

	fileCategory := getFileCategory(detectedType)
	ext := strings.ToLower(filepath.Ext(uploadInfo.FileName))
	if ext == "" {
		ext = getExtFromMimeType(detectedType)
	}
	finalFilename := fmt.Sprintf("%s%s", fileID, ext)
	finalPath := filepath.Join(pasteUploadDir, finalFilename)

	finalFile, err := os.Create(finalPath)
	if err != nil {
		return nil, errPaste("创建文件失败", 500)
	}
	defer finalFile.Close()

	for i := 0; i < uploadInfo.TotalChunks; i++ {
		chunkPath := filepath.Join(chunkDir, fileID, fmt.Sprintf("%d", i))
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			h.CleanupChunkUpload(fileID)
			os.Remove(finalPath)
			return nil, errPaste("读取分片失败", 500)
		}
		if _, err := finalFile.Write(chunkData); err != nil {
			h.CleanupChunkUpload(fileID)
			os.Remove(finalPath)
			return nil, errPaste("合并分片失败", 500)
		}
	}

	h.CleanupChunkUpload(fileID)

	fileInfo, _ := os.Stat(finalPath)
	var fileSize int64
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	return &FileMetadata{
		Filename:     finalFilename,
		OriginalName: uploadInfo.FileName,
		Type:         fileCategory,
		Size:         fileSize,
		URL:          "/api/paste/files/" + finalFilename,
	}, nil
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
		safeName := filepath.Base(file.Filename)
		if safeName == "" || safeName == "." || safeName == "/" {
			continue
		}
		filePath := filepath.Join(pasteUploadDir, safeName)
		os.Remove(filePath)
	}
}

// zipUnsupportedFile 将不被识别的文件流式打包为 zip 后存储。
// 返回最终文件名、原始文件名(带 .zip 后缀)和文件大小。
func zipUnsupportedFile(src io.Reader, originalName, fileID string) (string, string, int64, error) {
	if err := os.MkdirAll(pasteUploadDir, 0755); err != nil {
		return "", "", 0, err
	}

	// zip 内部条目名：清理原始文件名，缺省回退到 fileID
	entryName := utils.SanitizeFilename(originalName)
	if entryName == "" {
		entryName = fileID
	}

	finalFilename := fileID + ".zip"
	finalPath := filepath.Join(pasteUploadDir, finalFilename)

	out, err := os.Create(finalPath)
	if err != nil {
		return "", "", 0, err
	}

	zw := zip.NewWriter(out)
	w, err := zw.Create(entryName)
	if err != nil {
		zw.Close()
		out.Close()
		os.Remove(finalPath)
		return "", "", 0, err
	}
	if _, err := io.Copy(w, src); err != nil {
		zw.Close()
		out.Close()
		os.Remove(finalPath)
		return "", "", 0, err
	}
	if err := zw.Close(); err != nil {
		out.Close()
		os.Remove(finalPath)
		return "", "", 0, err
	}
	if err := out.Close(); err != nil {
		os.Remove(finalPath)
		return "", "", 0, err
	}

	info, err := os.Stat(finalPath)
	if err != nil {
		os.Remove(finalPath)
		return "", "", 0, err
	}

	zipName := entryName + ".zip"
	return finalFilename, zipName, info.Size(), nil
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
		"file_type":     detectedType,
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
		"total_pastes":     total,
		"today_creates":    todayCount,
		"max_file_size":    config.Get().Paste.MaxFileSize,
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
			"id":         p.ID,
			"title":      p.Title,
			"language":   p.Language,
			"created_at": p.CreatedAt,
			"expires_at": p.ExpiresAt,
			"views":      p.Views,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"pastes": responses,
		"total":  len(results),
		"limit":  limit,
		"offset": offset,
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
		"is_safe":        result.IsSafe,
		"has_virus":      result.HasVirus,
		"has_suspicious": result.HasSuspiciousURL,
		"warnings":       result.Warnings,
		"language":       language,
		"content_type":   contentType,
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
