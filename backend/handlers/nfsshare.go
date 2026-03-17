package handlers

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// NFSShareHandler NFS 文件分享处理器
type NFSShareHandler struct {
	db  *models.DB
	cfg config.NFSShareConfig
}

// NewNFSShareHandler 创建 NFS 分享处理器
func NewNFSShareHandler(db *models.DB, cfg config.NFSShareConfig) *NFSShareHandler {
	return &NFSShareHandler{db: db, cfg: cfg}
}

// verifyAdmin 校验超管密码（直接比较，密码存于配置文件）
func (h *NFSShareHandler) verifyAdmin(password string) bool {
	return h.cfg.AdminPassword != "" && password == h.cfg.AdminPassword
}

// validatePath 校验相对路径，防止路径穿越，返回绝对路径
func (h *NFSShareHandler) validatePath(relPath string) (string, error) {
	cleaned := filepath.Clean(relPath)
	full := filepath.Join(h.cfg.MountPath, cleaned)
	rel, err := filepath.Rel(h.cfg.MountPath, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("路径越界")
	}
	return full, nil
}

// checkEnabled 检查 NFS 功能是否已启用
func (h *NFSShareHandler) checkEnabled(c *gin.Context) bool {
	if !h.cfg.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "NFS 分享功能未启用，请在 config.yaml 中配置 nfs_share.enabled: true"})
		return false
	}
	return true
}

// -------- 目录浏览 --------

// FileEntry 目录条目信息
type FileEntry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	IsDir    bool      `json:"is_dir"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	MimeType string    `json:"mime_type"`
}

// Browse 浏览 NFS 目录（超管专用）
// GET /api/nfsshare/admin/browse?path=xxx&admin_password=xxx
func (h *NFSShareHandler) Browse(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	if !h.verifyAdmin(c.Query("admin_password")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}

	relPath := c.DefaultQuery("path", ".")
	fullPath, err := h.validatePath(relPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "路径非法"})
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取目录: " + err.Error()})
		return
	}

	result := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		var entryRel string
		if relPath == "." || relPath == "" {
			entryRel = e.Name()
		} else {
			entryRel = relPath + "/" + e.Name()
		}
		mt := ""
		if !e.IsDir() {
			mt = mime.TypeByExtension(filepath.Ext(e.Name()))
			if mt == "" {
				mt = "application/octet-stream"
			}
		}
		result = append(result, FileEntry{
			Name:     e.Name(),
			Path:     entryRel,
			IsDir:    e.IsDir(),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			MimeType: mt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"path":    relPath,
		"entries": result,
	})
}

// -------- 创建分享 --------

// CreateNFSShareRequest 创建分享请求体
type CreateNFSShareRequest struct {
	AdminPassword string `json:"admin_password" binding:"required"`
	Name          string `json:"name" binding:"required"`
	FilePath      string `json:"file_path" binding:"required"`
	MaxViews      int    `json:"max_views" binding:"required,min=1"`
	ExpiresDays   int    `json:"expires_days"` // 0 表示不过期
}

// Create 创建 NFS 文件分享（超管专用）
// POST /api/nfsshare
func (h *NFSShareHandler) Create(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}

	var req CreateNFSShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.verifyAdmin(req.AdminPassword) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}

	fullPath, err := h.validatePath(req.FilePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "路径非法"})
		return
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不存在: " + err.Error()})
		return
	}
	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能分享目录，请选择具体文件"})
		return
	}
	if h.cfg.MaxFileSizeMB > 0 && info.Size() > h.cfg.MaxFileSizeMB*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("文件超出限制，最大允许 %dMB", h.cfg.MaxFileSizeMB)})
		return
	}

	mimeType := mime.TypeByExtension(filepath.Ext(req.FilePath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var expiresAt *time.Time
	if req.ExpiresDays > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresDays) * 24 * time.Hour)
		expiresAt = &t
	}

	share, err := h.db.CreateNFSShare(req.Name, req.FilePath, mimeType, info.Size(), req.MaxViews, expiresAt, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, share)
}

// -------- 公开访问 --------

// Access 流式传输 NFS 共享文件，并检查次数/过期
// GET /api/nfsshare/:id
func (h *NFSShareHandler) Access(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}

	id := c.Param("id")
	share, err := h.db.GetNFSShare(id)
	if err != nil {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "not_found", 0)
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
		return
	}

	// 校验过期时间
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "denied_expired", 0)
		c.JSON(http.StatusForbidden, gin.H{"error": "分享已过期"})
		return
	}
	// 校验访问次数
	if share.MaxViews > 0 && share.Views >= share.MaxViews {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "denied_views", 0)
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("分享次数已用完（最大 %d 次）", share.MaxViews)})
		return
	}

	fullPath, err := h.validatePath(share.FilePath)
	if err != nil {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "error", 0)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "路径错误"})
		return
	}
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "file_missing", 0)
		c.JSON(http.StatusNotFound, gin.H{"error": "源文件不存在"})
		return
	}

	// 先自增次数，再记录日志，最后流式传输（即使客户端断开也已计数）
	h.db.IncrementNFSShareViews(id)
	h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "success", fileInfo.Size())

	filename := filepath.Base(share.FilePath)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", share.MimeType)
	c.File(fullPath)
}

// Info 返回分享公开信息（不消耗次数）
// GET /api/nfsshare/:id/info
func (h *NFSShareHandler) Info(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	id := c.Param("id")
	share, err := h.db.GetNFSShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
		return
	}

	type PublicInfo struct {
		ID           string     `json:"id"`
		Name         string     `json:"name"`
		FileSize     int64      `json:"file_size"`
		MimeType     string     `json:"mime_type"`
		MaxViews     int        `json:"max_views"`
		Views        int        `json:"views"`
		RemainingViews int      `json:"remaining_views"`
		ExpiresAt    *time.Time `json:"expires_at"`
		CreatedAt    time.Time  `json:"created_at"`
		Expired      bool       `json:"expired"`
		Exhausted    bool       `json:"exhausted"`
	}

	remaining := share.MaxViews - share.Views
	if remaining < 0 {
		remaining = 0
	}
	c.JSON(http.StatusOK, PublicInfo{
		ID:             share.ID,
		Name:           share.Name,
		FileSize:       share.FileSize,
		MimeType:       share.MimeType,
		MaxViews:       share.MaxViews,
		Views:          share.Views,
		RemainingViews: remaining,
		ExpiresAt:      share.ExpiresAt,
		CreatedAt:      share.CreatedAt,
		Expired:        share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt),
		Exhausted:      share.MaxViews > 0 && share.Views >= share.MaxViews,
	})
}

// -------- 管理接口 --------

// AdminList 列出所有分享（超管专用）
// GET /api/nfsshare/admin/list?admin_password=xxx&page=1&page_size=20
func (h *NFSShareHandler) AdminList(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	if !h.verifyAdmin(c.Query("admin_password")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	shares, total, err := h.db.GetAllNFSShares(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"shares":    shares,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminGetLogs 获取指定分享的访问日志（超管专用）
// GET /api/nfsshare/admin/:id/logs?admin_password=xxx&page=1&page_size=50
func (h *NFSShareHandler) AdminGetLogs(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	if !h.verifyAdmin(c.Query("admin_password")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}

	id := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	logs, total, err := h.db.GetNFSShareLogs(id, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdateNFSShareRequest 更新分享请求
type UpdateNFSShareRequest struct {
	AdminPassword string     `json:"admin_password" binding:"required"`
	MaxViews      int        `json:"max_views"`    // 直接设置新的最大次数（优先）
	AddViews      int        `json:"add_views"`    // 在现有基础上追加次数
	ExpiresAt     *time.Time `json:"expires_at"`   // 直接设置新的过期时间（优先）
	AddDays       int        `json:"add_days"`     // 在现有基础上延期天数
}

// AdminUpdate 修改分享配置（超管专用）
// PUT /api/nfsshare/admin/:id
func (h *NFSShareHandler) AdminUpdate(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}

	id := c.Param("id")
	var req UpdateNFSShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.verifyAdmin(req.AdminPassword) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}

	share, err := h.db.GetNFSShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
		return
	}

	// 计算新 maxViews
	newMaxViews := share.MaxViews
	if req.MaxViews > 0 {
		newMaxViews = req.MaxViews
	} else if req.AddViews > 0 {
		newMaxViews = share.MaxViews + req.AddViews
	}

	// 计算新过期时间
	var newExpiresAt *time.Time
	if req.ExpiresAt != nil {
		newExpiresAt = req.ExpiresAt
	} else if req.AddDays > 0 {
		base := time.Now()
		if share.ExpiresAt != nil && share.ExpiresAt.After(base) {
			base = *share.ExpiresAt
		}
		t := base.Add(time.Duration(req.AddDays) * 24 * time.Hour)
		newExpiresAt = &t
	} else {
		newExpiresAt = share.ExpiresAt
	}

	if err := h.db.UpdateNFSShare(id, newMaxViews, newExpiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated, _ := h.db.GetNFSShare(id)
	c.JSON(http.StatusOK, updated)
}

// AdminDelete 删除分享（超管专用）
// DELETE /api/nfsshare/admin/:id?admin_password=xxx
func (h *NFSShareHandler) AdminDelete(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	if !h.verifyAdmin(c.Query("admin_password")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}

	id := c.Param("id")
	if err := h.db.DeleteNFSShare(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// Status 返回 NFS 功能状态（供前端判断是否启用）
// GET /api/nfsshare/status
func (h *NFSShareHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"enabled": h.cfg.Enabled})
}
