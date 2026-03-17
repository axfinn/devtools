package handlers

import (
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// -------- 挂载状态 --------

// MountStatus 单个挂载点的运行时状态
type MountStatus struct {
	Config     config.MountConfig
	LocalPath  string // 实际本地挂载目录
	Mounted    bool
	ErrMessage string
	MountedAt  *time.Time
}

// MountInfo 挂载点信息（对外输出，不含密码）
type MountInfo struct {
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Host       string     `json:"host"`
	Export     string     `json:"export"`
	Share      string     `json:"share"`
	Username   string     `json:"username"`
	LocalPath  string     `json:"local_path"`
	Mounted    bool       `json:"mounted"`
	ErrMessage string     `json:"error,omitempty"`
	MountedAt  *time.Time `json:"mounted_at,omitempty"`
}

// -------- Handler --------

// NFSShareHandler NFS/SMB 文件分享处理器
type NFSShareHandler struct {
	db     *models.DB
	cfg    config.NFSShareConfig
	mounts map[string]*MountStatus // name -> status
	mu     sync.RWMutex
}

// NewNFSShareHandler 创建 Handler 并自动挂载所有配置的挂载点
func NewNFSShareHandler(db *models.DB, cfg config.NFSShareConfig) *NFSShareHandler {
	h := &NFSShareHandler{
		db:     db,
		cfg:    cfg,
		mounts: make(map[string]*MountStatus),
	}
	if cfg.Enabled {
		h.initMounts()
	}
	return h
}

// initMounts 初始化所有挂载点（启动时调用）
func (h *NFSShareHandler) initMounts() {
	os.MkdirAll("./data/mounts", 0755)
	for _, mc := range h.cfg.Mounts {
		ms := h.buildMountStatus(mc)
		h.mounts[mc.Name] = ms
		if err := h.doMount(ms); err != nil {
			log.Printf("[NFSShare] 挂载 %s 失败: %v", mc.Name, err)
		} else {
			log.Printf("[NFSShare] 挂载 %s 成功: %s", mc.Name, ms.LocalPath)
		}
	}
}

// buildMountStatus 根据配置构建 MountStatus
func (h *NFSShareHandler) buildMountStatus(mc config.MountConfig) *MountStatus {
	local := mc.MountPoint
	if local == "" {
		local = filepath.Join("./data/mounts", mc.Name)
	}
	return &MountStatus{Config: mc, LocalPath: local}
}

// doMount 执行挂载操作
func (h *NFSShareHandler) doMount(ms *MountStatus) error {
	os.MkdirAll(ms.LocalPath, 0755)
	mc := ms.Config

	var cmd *exec.Cmd
	switch strings.ToLower(mc.Type) {
	case "nfs":
		args := []string{"-t", "nfs"}
		opts := "soft,timeo=30"
		if mc.Options != "" {
			opts = mc.Options
		}
		args = append(args, "-o", opts,
			fmt.Sprintf("%s:%s", mc.Host, mc.Export), ms.LocalPath)
		cmd = exec.Command("mount", args...)

	case "smb", "cifs":
		src := fmt.Sprintf("//%s/%s", mc.Host, mc.Share)
		opts := fmt.Sprintf("username=%s,password=%s", mc.Username, mc.Password)
		if mc.Domain != "" {
			opts += ",domain=" + mc.Domain
		}
		if mc.Options != "" {
			opts += "," + mc.Options
		}
		cmd = exec.Command("mount", "-t", "cifs", src, ms.LocalPath, "-o", opts)

	case "local":
		// 本地目录，直接检查是否存在
		if _, err := os.Stat(mc.Export); err != nil {
			ms.Mounted = false
			ms.ErrMessage = "本地目录不存在: " + mc.Export
			return fmt.Errorf(ms.ErrMessage)
		}
		ms.LocalPath = mc.Export
		ms.Mounted = true
		now := time.Now()
		ms.MountedAt = &now
		ms.ErrMessage = ""
		return nil

	default:
		return fmt.Errorf("不支持的挂载类型: %s", mc.Type)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		ms.Mounted = false
		ms.ErrMessage = strings.TrimSpace(string(out))
		if ms.ErrMessage == "" {
			ms.ErrMessage = err.Error()
		}
		return fmt.Errorf("%s", ms.ErrMessage)
	}
	ms.Mounted = true
	now := time.Now()
	ms.MountedAt = &now
	ms.ErrMessage = ""
	return nil
}

// doUmount 执行卸载操作
func (h *NFSShareHandler) doUmount(ms *MountStatus) error {
	if strings.ToLower(ms.Config.Type) == "local" {
		ms.Mounted = false
		return nil
	}
	cmd := exec.Command("umount", ms.LocalPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	ms.Mounted = false
	ms.MountedAt = nil
	return nil
}

// getMountStatus 取得可对外展示的挂载信息（不含密码）
func toMountInfo(ms *MountStatus) MountInfo {
	return MountInfo{
		Name:       ms.Config.Name,
		Type:       ms.Config.Type,
		Host:       ms.Config.Host,
		Export:     ms.Config.Export,
		Share:      ms.Config.Share,
		Username:   ms.Config.Username,
		LocalPath:  ms.LocalPath,
		Mounted:    ms.Mounted,
		ErrMessage: ms.ErrMessage,
		MountedAt:  ms.MountedAt,
	}
}

// -------- 鉴权/工具 --------

func (h *NFSShareHandler) verifyAdmin(password string) bool {
	return h.cfg.AdminPassword != "" && password == h.cfg.AdminPassword
}

// parseMountPath 解析 "mount_name/relative/path" 格式，返回 (mountStatus, absPath, error)
func (h *NFSShareHandler) parseMountPath(path string) (*MountStatus, string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 找第一个 / 分割挂载名和相对路径
	idx := strings.Index(path, "/")
	var mountName, relPath string
	if idx < 0 {
		mountName = path
		relPath = "."
	} else {
		mountName = path[:idx]
		relPath = path[idx+1:]
		if relPath == "" {
			relPath = "."
		}
	}

	ms, ok := h.mounts[mountName]
	if !ok {
		return nil, "", fmt.Errorf("挂载点 %q 不存在", mountName)
	}
	if !ms.Mounted {
		return nil, "", fmt.Errorf("挂载点 %q 未挂载", mountName)
	}

	cleaned := filepath.Clean(relPath)
	full := filepath.Join(ms.LocalPath, cleaned)
	// 防路径穿越
	rel, err := filepath.Rel(ms.LocalPath, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, "", fmt.Errorf("路径越界")
	}
	return ms, full, nil
}

func (h *NFSShareHandler) checkEnabled(c *gin.Context) bool {
	if !h.cfg.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "NFS/SMB 分享功能未启用，请在 config.yaml 中配置 nfs_share.enabled: true"})
		return false
	}
	return true
}

// -------- 挂载管理 API --------

// MountsList 列出所有挂载点及状态（超管）
// GET /api/nfsshare/admin/mounts?admin_password=xxx
func (h *NFSShareHandler) MountsList(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	if !h.verifyAdmin(c.Query("admin_password")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]MountInfo, 0, len(h.mounts))
	for _, ms := range h.mounts {
		list = append(list, toMountInfo(ms))
	}
	c.JSON(http.StatusOK, gin.H{"mounts": list})
}

// MountsRemount 重新挂载指定挂载点（超管）
// POST /api/nfsshare/admin/mounts/:name/remount?admin_password=xxx
func (h *NFSShareHandler) MountsRemount(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	if !h.verifyAdmin(c.Query("admin_password")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}

	name := c.Param("name")
	h.mu.Lock()
	ms, ok := h.mounts[name]
	h.mu.Unlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "挂载点不存在"})
		return
	}

	// 先尝试 umount
	if ms.Mounted {
		h.doUmount(ms)
	}
	err := h.doMount(ms)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "mount": toMountInfo(ms)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "挂载成功", "mount": toMountInfo(ms)})
}

// MountsUmount 卸载指定挂载点（超管）
// POST /api/nfsshare/admin/mounts/:name/umount?admin_password=xxx
func (h *NFSShareHandler) MountsUmount(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	if !h.verifyAdmin(c.Query("admin_password")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}

	name := c.Param("name")
	h.mu.Lock()
	ms, ok := h.mounts[name]
	h.mu.Unlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "挂载点不存在"})
		return
	}

	if err := h.doUmount(ms); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "卸载成功", "mount": toMountInfo(ms)})
}

// -------- 目录浏览 --------

// FileEntry 目录条目信息
type FileEntry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"` // "mount_name/relative/path" 格式
	IsDir    bool      `json:"is_dir"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	MimeType string    `json:"mime_type"`
}

// Browse 浏览目录（超管）
// GET /api/nfsshare/admin/browse?path=mount_name/subdir&admin_password=xxx
// 当 path 为空时，列出所有已挂载的挂载点
func (h *NFSShareHandler) Browse(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	if !h.verifyAdmin(c.Query("admin_password")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}

	path := strings.TrimSpace(c.Query("path"))

	// 根路径：列出所有挂载点
	if path == "" || path == "." {
		h.mu.RLock()
		defer h.mu.RUnlock()
		entries := make([]FileEntry, 0, len(h.mounts))
		for _, ms := range h.mounts {
			entries = append(entries, FileEntry{
				Name:  ms.Config.Name,
				Path:  ms.Config.Name,
				IsDir: true,
				MimeType: fmt.Sprintf("[%s] %s", strings.ToUpper(ms.Config.Type),
					func() string {
						if ms.Mounted {
							return "已挂载"
						}
						return "未挂载: " + ms.ErrMessage
					}()),
			})
		}
		c.JSON(http.StatusOK, gin.H{"path": ".", "entries": entries})
		return
	}

	// 具体路径：浏览挂载点内的目录
	ms, fullPath, err := h.parseMountPath(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取目录: " + err.Error()})
		return
	}

	// 计算相对于挂载根的前缀
	relBase, _ := filepath.Rel(ms.LocalPath, fullPath)
	if relBase == "." {
		relBase = ""
	}

	result := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		var entryPath string
		if relBase == "" {
			entryPath = ms.Config.Name + "/" + e.Name()
		} else {
			entryPath = ms.Config.Name + "/" + relBase + "/" + e.Name()
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
			Path:     entryPath,
			IsDir:    e.IsDir(),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			MimeType: mt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"path": path, "entries": result})
}

// -------- 创建分享 --------

// CreateNFSShareRequest 创建分享请求
type CreateNFSShareRequest struct {
	AdminPassword string `json:"admin_password" binding:"required"`
	Name          string `json:"name" binding:"required"`
	FilePath      string `json:"file_path" binding:"required"` // "mount_name/relative/path"
	MaxViews      int    `json:"max_views" binding:"required,min=1"`
	ExpiresDays   int    `json:"expires_days"`
}

// Create 创建 NFS/SMB 文件分享（超管）
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

	_, fullPath, err := h.parseMountPath(req.FilePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// 存储 "mount_name:relative/path" 格式，解耦本地挂载路径
	share, err := h.db.CreateNFSShare(req.Name, req.FilePath, mimeType, info.Size(), req.MaxViews, expiresAt, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, share)
}

// -------- 公开访问 --------

// Access 流式传输共享文件（公开，消耗访问次数）
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

	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "denied_expired", 0)
		c.JSON(http.StatusForbidden, gin.H{"error": "分享已过期"})
		return
	}
	if share.MaxViews > 0 && share.Views >= share.MaxViews {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "denied_views", 0)
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("分享次数已用完（最大 %d 次）", share.MaxViews)})
		return
	}

	_, fullPath, err := h.parseMountPath(share.FilePath)
	if err != nil {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "error", 0)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "挂载点不可用: " + err.Error()})
		return
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "file_missing", 0)
		c.JSON(http.StatusNotFound, gin.H{"error": "源文件不存在"})
		return
	}

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

	remaining := share.MaxViews - share.Views
	if remaining < 0 {
		remaining = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"id":              share.ID,
		"name":            share.Name,
		"file_size":       share.FileSize,
		"mime_type":       share.MimeType,
		"max_views":       share.MaxViews,
		"views":           share.Views,
		"remaining_views": remaining,
		"expires_at":      share.ExpiresAt,
		"created_at":      share.CreatedAt,
		"expired":         share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt),
		"exhausted":       share.MaxViews > 0 && share.Views >= share.MaxViews,
	})
}

// -------- 管理接口 --------

// AdminList 列出所有分享（超管）
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

// AdminGetLogs 获取分享访问日志（超管）
// GET /api/nfsshare/admin/:id/logs?admin_password=xxx
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

// UpdateNFSShareRequest 更新分享配置
type UpdateNFSShareRequest struct {
	AdminPassword string     `json:"admin_password" binding:"required"`
	MaxViews      int        `json:"max_views"`
	AddViews      int        `json:"add_views"`
	ExpiresAt     *time.Time `json:"expires_at"`
	AddDays       int        `json:"add_days"`
}

// AdminUpdate 修改分享配置（超管）
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

	newMaxViews := share.MaxViews
	if req.MaxViews > 0 {
		newMaxViews = req.MaxViews
	} else if req.AddViews > 0 {
		newMaxViews = share.MaxViews + req.AddViews
	}

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

// AdminDelete 删除分享（超管）
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

// Status 返回功能是否启用（公开）
// GET /api/nfsshare/status
func (h *NFSShareHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"enabled": h.cfg.Enabled})
}
