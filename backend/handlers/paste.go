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

type PasteHandler struct {
	db         *models.DB
	maxTotal   int
	maxPerIP   int
	ipWindow   time.Duration
}

func NewPasteHandler(db *models.DB) *PasteHandler {
	return &PasteHandler{
		db:       db,
		maxTotal: 10000,        // 最多存储 10000 条
		maxPerIP: 10,           // 每 IP 每分钟最多 10 条（与中间件限流一致）
		ipWindow: time.Minute,
	}
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

type PasteResponse struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Language    string          `json:"language"`
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

	// 返回文件信息
	fileURL := "/api/paste/files/" + filename
	c.JSON(http.StatusOK, gin.H{
		"id":            filename,
		"url":           fileURL,
		"filename":      filename,
		"original_name": header.Filename,
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
		req.Language = "text"
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

	c.JSON(http.StatusOK, PasteResponse{
		ID:          paste.ID,
		Title:       paste.Title,
		Language:    paste.Language,
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

	c.JSON(http.StatusOK, PasteResponse{
		ID:          paste.ID,
		Title:       paste.Title,
		Language:    paste.Language,
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

	c.JSON(http.StatusOK, PasteResponse{
		ID:          paste.ID,
		Title:       paste.Title,
		Language:    paste.Language,
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


