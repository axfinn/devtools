package handlers

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
	smb2 "github.com/hirochachacha/go-smb2"
)

// -------- SMB 后端 --------

// smbBackend 持有一个已连接的 SMB Share，支持懒连接和重连
type smbBackend struct {
	mu        sync.Mutex
	cfg       config.MountConfig
	session   *smb2.Session
	share     *smb2.Share
	connected bool
}

func newSMBBackend(cfg config.MountConfig) *smbBackend {
	return &smbBackend{cfg: cfg}
}

func (b *smbBackend) connect() error {
	b.disconnect()
	conn, err := net.DialTimeout("tcp", b.cfg.Host+":445", 10*time.Second)
	if err != nil {
		return fmt.Errorf("TCP 连接失败: %v", err)
	}
	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     b.cfg.Username,
			Password: b.cfg.Password,
			Domain:   b.cfg.Domain,
		},
	}
	session, err := d.Dial(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMB 认证失败: %v", err)
	}
	share, err := session.Mount(b.cfg.Share)
	if err != nil {
		session.Logoff()
		return fmt.Errorf("挂载共享 %q 失败: %v", b.cfg.Share, err)
	}
	b.session = session
	b.share = share
	b.connected = true
	return nil
}

func (b *smbBackend) disconnect() {
	if b.share != nil {
		b.share.Umount()
		b.share = nil
	}
	if b.session != nil {
		b.session.Logoff()
		b.session = nil
	}
	b.connected = false
}

// ensureConnected 保证连接可用（断线自动重连）
func (b *smbBackend) ensureConnected() error {
	if b.connected && b.share != nil {
		return nil
	}
	return b.connect()
}

func (b *smbBackend) ReadDir(path string) ([]os.FileInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureConnected(); err != nil {
		return nil, err
	}
	entries, err := b.share.ReadDir(path)
	if err != nil {
		b.connected = false
		return nil, err
	}
	return entries, nil
}

func (b *smbBackend) Stat(path string) (os.FileInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureConnected(); err != nil {
		return nil, err
	}
	info, err := b.share.Stat(path)
	if err != nil {
		b.connected = false
		return nil, err
	}
	return info, nil
}

// Open 打开文件，返回 ReadSeekCloser 和文件大小
func (b *smbBackend) Open(path string) (io.ReadSeekCloser, int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureConnected(); err != nil {
		return nil, 0, err
	}
	f, err := b.share.Open(path)
	if err != nil {
		b.connected = false
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}
	return f, info.Size(), nil
}

// -------- 挂载状态 --------

// MountStatus 单个挂载点的运行时状态
type MountStatus struct {
	Config     config.MountConfig
	LocalPath  string // NFS/local 使用；SMB 不使用
	smb        *smbBackend
	Mounted    bool
	ErrMessage string
	MountedAt  *time.Time
}

// MountInfo 挂载点对外展示信息（不含密码）
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
	mounts map[string]*MountStatus
	mu     sync.RWMutex
}

// NewNFSShareHandler 创建 Handler 并初始化所有挂载点
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

func (h *NFSShareHandler) initMounts() {
	os.MkdirAll("./data/mounts", 0755)
	for _, mc := range h.cfg.Mounts {
		ms := h.buildMountStatus(mc)
		h.mounts[mc.Name] = ms
		if err := h.doMount(ms); err != nil {
			log.Printf("[NFSShare] 挂载 %s (%s) 失败: %v", mc.Name, mc.Type, err)
		} else {
			log.Printf("[NFSShare] 挂载 %s (%s) 成功", mc.Name, mc.Type)
		}
	}
}

func (h *NFSShareHandler) buildMountStatus(mc config.MountConfig) *MountStatus {
	ms := &MountStatus{Config: mc}
	if strings.ToLower(mc.Type) == "smb" {
		ms.smb = newSMBBackend(mc)
	} else {
		local := mc.MountPoint
		if local == "" {
			local = filepath.Join("./data/mounts", mc.Name)
		}
		ms.LocalPath = local
	}
	return ms
}

func (h *NFSShareHandler) doMount(ms *MountStatus) error {
	mc := ms.Config
	switch strings.ToLower(mc.Type) {
	case "smb":
		if err := ms.smb.connect(); err != nil {
			ms.Mounted = false
			ms.ErrMessage = err.Error()
			return err
		}
		ms.Mounted = true
		now := time.Now()
		ms.MountedAt = &now
		ms.ErrMessage = ""
		return nil

	case "local":
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

	case "nfs":
		// NFS 挂载需要系统 root 权限（Docker 需 --cap-add SYS_ADMIN），暂不支持
		ms.Mounted = false
		ms.ErrMessage = "NFS 暂不支持（需要 root/SYS_ADMIN 权限），请改用 smb 或 local 类型"
		return fmt.Errorf(ms.ErrMessage)

	default:
		return fmt.Errorf("不支持的挂载类型: %s（支持 nfs / smb / local）", mc.Type)
	}
}

func (h *NFSShareHandler) doUmount(ms *MountStatus) error {
	switch strings.ToLower(ms.Config.Type) {
	case "smb":
		ms.smb.disconnect()
		ms.Mounted = false
		ms.MountedAt = nil
		return nil
	case "local":
		ms.Mounted = false
		return nil
	default:
		// nfs 等暂不支持，直接标记为未挂载
		ms.Mounted = false
		ms.MountedAt = nil
		return nil
	}
}

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

// -------- 鉴权 / 路径解析 --------

func (h *NFSShareHandler) verifyAdmin(password string) bool {
	return h.cfg.AdminPassword != "" && password == h.cfg.AdminPassword
}

func (h *NFSShareHandler) checkEnabled(c *gin.Context) bool {
	if !h.cfg.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "NFS/SMB 分享功能未启用，请在 config.yaml 中配置 nfs_share.enabled: true"})
		return false
	}
	return true
}

// parsePath 解析 "mount_name/relative/path"，返回 mountStatus 和相对路径（SMB 用）或绝对路径（NFS/local 用）
type parsedPath struct {
	ms      *MountStatus
	relPath string // SMB 使用（相对于 share 根）
	absPath string // NFS/local 使用
}

func (h *NFSShareHandler) parsePath(path string) (*parsedPath, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

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
		return nil, fmt.Errorf("挂载点 %q 不存在", mountName)
	}
	if !ms.Mounted {
		return nil, fmt.Errorf("挂载点 %q 未连接/挂载", mountName)
	}

	pp := &parsedPath{ms: ms, relPath: relPath}

	if strings.ToLower(ms.Config.Type) != "smb" {
		// NFS / local：转换为绝对路径并防路径穿越
		cleaned := filepath.Clean(relPath)
		full := filepath.Join(ms.LocalPath, cleaned)
		rel, err := filepath.Rel(ms.LocalPath, full)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("路径越界")
		}
		pp.absPath = full
	} else {
		// SMB：确保路径不含 .. 穿越
		cleaned := filepath.ToSlash(filepath.Clean(relPath))
		if strings.HasPrefix(cleaned, "..") {
			return nil, fmt.Errorf("路径越界")
		}
		pp.relPath = cleaned
	}

	return pp, nil
}

// -------- 挂载管理 API --------

// MountsList 列出挂载点及状态（超管）
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

// MountsRemount 重新连接/挂载（超管）
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
	if ms.Mounted {
		h.doUmount(ms)
	}
	if err := h.doMount(ms); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "mount": toMountInfo(ms)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "连接成功", "mount": toMountInfo(ms)})
}

// MountsUmount 断开连接/卸载（超管）
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
	c.JSON(http.StatusOK, gin.H{"message": "已断开", "mount": toMountInfo(ms)})
}

// -------- 目录浏览 --------

// FileEntry 目录条目
type FileEntry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	IsDir    bool      `json:"is_dir"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	MimeType string    `json:"mime_type"`
}

// Browse 浏览目录（超管）
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
			status := "已连接"
			if !ms.Mounted {
				status = "未连接: " + ms.ErrMessage
			}
			entries = append(entries, FileEntry{
				Name:     ms.Config.Name,
				Path:     ms.Config.Name,
				IsDir:    true,
				MimeType: fmt.Sprintf("[%s] %s", strings.ToUpper(ms.Config.Type), status),
			})
		}
		c.JSON(http.StatusOK, gin.H{"path": ".", "entries": entries})
		return
	}

	pp, err := h.parsePath(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mountName := pp.ms.Config.Name
	isSMB := strings.ToLower(pp.ms.Config.Type) == "smb"

	// 读取目录内容，统一转为 FileEntry
	var result []FileEntry
	if isSMB {
		smbPath := pp.relPath // 保持 "."，go-smb2 用 "." 读根目录，不能传空字符串
		infos, err2 := pp.ms.smb.ReadDir(smbPath)
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取目录: " + err2.Error()})
			return
		}
		relBase := smbPath
		if relBase == "." {
			relBase = ""
		}
		result = make([]FileEntry, 0, len(infos))
		for _, info := range infos {
			var entryPath string
			if relBase == "" {
				entryPath = mountName + "/" + info.Name()
			} else {
				entryPath = mountName + "/" + relBase + "/" + info.Name()
			}
			mt := ""
			if !info.IsDir() {
				mt = mime.TypeByExtension(filepath.Ext(info.Name()))
				if mt == "" {
					mt = "application/octet-stream"
				}
			}
			result = append(result, FileEntry{
				Name:     info.Name(),
				Path:     entryPath,
				IsDir:    info.IsDir(),
				Size:     info.Size(),
				ModTime:  info.ModTime(),
				MimeType: mt,
			})
		}
	} else {
		dirEntries, err2 := os.ReadDir(pp.absPath)
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取目录: " + err2.Error()})
			return
		}
		relBase, _ := filepath.Rel(pp.ms.LocalPath, pp.absPath)
		if relBase == "." {
			relBase = ""
		}
		result = make([]FileEntry, 0, len(dirEntries))
		for _, e := range dirEntries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			var entryPath string
			if relBase == "" {
				entryPath = mountName + "/" + e.Name()
			} else {
				entryPath = mountName + "/" + relBase + "/" + e.Name()
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
	}
	c.JSON(http.StatusOK, gin.H{"path": path, "entries": result})
}

// -------- 创建分享 --------

// CreateNFSShareRequest 创建分享请求
type CreateNFSShareRequest struct {
	AdminPassword string `json:"admin_password" binding:"required"`
	Name          string `json:"name" binding:"required"`
	FilePath      string `json:"file_path" binding:"required"`
	MaxViews      int    `json:"max_views" binding:"required,min=1"`
	ExpiresDays   int    `json:"expires_days"`
}

// Create 创建分享（超管）
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

	pp, err := h.parsePath(req.FilePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var fileSize int64
	if strings.ToLower(pp.ms.Config.Type) == "smb" {
		info, err := pp.ms.smb.Stat(pp.relPath)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件不存在: " + err.Error()})
			return
		}
		if info.IsDir() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能分享目录，请选择具体文件"})
			return
		}
		fileSize = info.Size()
	} else {
		info, err := os.Stat(pp.absPath)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件不存在: " + err.Error()})
			return
		}
		if info.IsDir() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能分享目录，请选择具体文件"})
			return
		}
		fileSize = info.Size()
	}

	if h.cfg.MaxFileSizeMB > 0 && fileSize > h.cfg.MaxFileSizeMB*1024*1024 {
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

	share, err := h.db.CreateNFSShare(req.Name, req.FilePath, mimeType, fileSize, req.MaxViews, expiresAt, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, share)
}

// -------- 公开访问 --------

// Access 流式传输共享文件（公开，消耗访问次数）
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

	pp, err := h.parsePath(share.FilePath)
	if err != nil {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "error", 0)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "挂载点不可用: " + err.Error()})
		return
	}

	h.db.IncrementNFSShareViews(id)
	filename := filepath.Base(share.FilePath)

	if strings.ToLower(pp.ms.Config.Type) == "smb" {
		// SMB：通过 go-smb2 流式传输
		f, size, err := pp.ms.smb.Open(pp.relPath)
		if err != nil {
			h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "file_missing", 0)
			c.JSON(http.StatusNotFound, gin.H{"error": "源文件不存在"})
			return
		}
		defer f.Close()
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "success", size)
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.Header("Content-Type", share.MimeType)
		c.Header("Content-Length", fmt.Sprintf("%d", size))
		io.Copy(c.Writer, f)
	} else {
		// NFS / local：直接 sendfile
		info, err := os.Stat(pp.absPath)
		if err != nil {
			h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "file_missing", 0)
			c.JSON(http.StatusNotFound, gin.H{"error": "源文件不存在"})
			return
		}
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "success", info.Size())
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.Header("Content-Type", share.MimeType)
		c.File(pp.absPath)
	}
}

// Info 返回分享公开信息（不消耗次数）
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
	c.JSON(http.StatusOK, gin.H{"shares": shares, "total": total, "page": page, "page_size": pageSize})
}

// AdminGetLogs 获取分享访问日志（超管）
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
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total, "page": page, "page_size": pageSize})
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
func (h *NFSShareHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"enabled": h.cfg.Enabled})
}
