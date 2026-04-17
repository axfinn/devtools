package handlers

import (
	crypto_rand "crypto/rand"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
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
	"devtools/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

// Create 在 SMB 共享上创建/覆盖文件，返回 WriteCloser
func (b *smbBackend) Create(path string) (io.WriteCloser, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureConnected(); err != nil {
		return nil, err
	}
	f, err := b.share.Create(path)
	if err != nil {
		b.connected = false
		return nil, err
	}
	return f, nil
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

// hlsJob 表示一个 HLS 转码任务
type hlsJob struct {
	done        chan struct{} // 关闭表示完成
	err         error
	viewCounted bool // 是否已为本 job 计入过 view（防止重复计数）
}

// NFSShareHandler NFS/SMB 文件分享处理器
type NFSShareHandler struct {
	db           *models.DB
	cfg          config.NFSShareConfig
	mounts       map[string]*MountStatus
	mu           sync.RWMutex
	hlsJobs      sync.Map // key: shareID, value: *hlsJob
	watchRooms   sync.Map // key: shareID, value: *watchRoom
	recordSessions sync.Map // key: shareID+"/"+sessionID, value: *recordSession
}

type recordSession struct {
	shareID  string
	sessionID string
	clientIP string
	timer    *time.Timer
	mu       sync.Mutex
}

const recordIdleTimeout = 10 * time.Second

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
	// 内置上传目录挂载点
	uploadDir := "./data/uploads"
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(filepath.Join(uploadDir, ".tmp"), 0755)
	h.mounts["__uploads__"] = &MountStatus{
		Config:    config.MountConfig{Name: "__uploads__", Type: "local", Export: uploadDir},
		LocalPath: uploadDir,
		Mounted:   true,
	}
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

// checkSharePassword 校验分享访问密码，通过返回 true；未通过时自动写 403 响应
func checkSharePassword(c *gin.Context, share *models.NFSShare, password string) bool {
	if share.Password == "" {
		return true
	}
	if password == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "该分享需要密码", "need_password": true})
		return false
	}
	if !utils.VerifyPassword(password, share.Password) {
		c.JSON(http.StatusForbidden, gin.H{"error": "密码错误"})
		return false
	}
	return true
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
				mt = detectMimeType(info.Name())
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
				mt = detectMimeType(e.Name())
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

// -------- MIME 类型检测 --------

// videoExtensions 常见视频文件扩展名到 MIME 类型的映射
var videoExtensions = map[string]string{
	".mp4":  "video/mp4",
	".m4v":  "video/mp4",
	".mkv":  "video/x-matroska",
	".avi":  "video/x-msvideo",
	".mov":  "video/quicktime",
	".qt":   "video/quicktime",
	".webm": "video/webm",
	".flv":  "video/x-flv",
	".wmv":  "video/x-ms-wmv",
	".ts":   "video/mp2t",
	".ogv":  "video/ogg",
	".3gp":  "video/3gpp",
	".3g2":  "video/3gpp2",
}

// detectMimeType 根据文件扩展名检测 MIME 类型，对常见视频格式有可靠的回退
func detectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if mt := mime.TypeByExtension(ext); mt != "" && mt != "application/octet-stream" {
		return mt
	}
	if mt, ok := videoExtensions[ext]; ok {
		return mt
	}
	mt := mime.TypeByExtension(ext)
	if mt == "" {
		return "application/octet-stream"
	}
	return mt
}

// -------- 创建分享 --------

// CreateNFSShareRequest 创建分享请求
type CreateNFSShareRequest struct {
	AdminPassword string `json:"admin_password" binding:"required"`
	Name          string `json:"name" binding:"required"`
	FilePath      string `json:"file_path" binding:"required"`
	MaxViews      int    `json:"max_views" binding:"required,min=1"`
	ExpiresDays   int    `json:"expires_days"`
	Password      string `json:"password"`       // 可选，访问密码
	RecordEnabled bool   `json:"record_enabled"` // 是否开启访客录音
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

	mimeType := detectMimeType(req.FilePath)

	var expiresAt *time.Time
	if req.ExpiresDays > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresDays) * 24 * time.Hour)
		expiresAt = &t
	}

	hashedPwd := ""
	if req.Password != "" {
		hashedPwd, err = utils.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败"})
			return
		}
	}

	share, err := h.db.CreateNFSShare(req.Name, req.FilePath, mimeType, hashedPwd, fileSize, req.MaxViews, expiresAt, c.ClientIP(), req.RecordEnabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":           share.ID,
		"name":         share.Name,
		"has_password": share.Password != "",
	})
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
	if !checkSharePassword(c, share, c.Query("password")) {
		return
	}

	// 禁止下载
	if h.cfg.DisableDownload {
		c.JSON(http.StatusForbidden, gin.H{"error": "文件下载已禁用"})
		return
	}
	// 禁止视频直接下载（可通过配置关闭）
	if h.cfg.DisableVideoDownload && strings.HasPrefix(share.MimeType, "video/") {
		c.JSON(http.StatusForbidden, gin.H{"error": "视频文件不允许直接下载，请通过播放页面访问"})
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

// isVideoFile 通过 MIME 类型或扩展名判断是否为视频文件
func isVideoFile(mimeType, filePath string) bool {
	if strings.HasPrefix(mimeType, "video/") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp4", ".m4v", ".mkv", ".avi", ".mov", ".webm", ".flv", ".wmv", ".ts", ".m2ts", ".ogv", ".3gp":
		return true
	}
	return false
}

func isAudioFile(mimeType, filePath string) bool {
	if strings.HasPrefix(mimeType, "audio/") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3", ".flac", ".wav", ".aac", ".ogg", ".opus", ".m4a", ".wma", ".ape":
		return true
	}
	return false
}

func isShareImageFile(mimeType, filePath string) bool {
	if strings.HasPrefix(mimeType, "image/") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".ico", ".avif", ".tiff":
		return true
	}
	return false
}

func isShareTextFile(mimeType, filePath string) bool {
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	if mimeType == "application/json" || mimeType == "application/xml" ||
		mimeType == "application/javascript" || mimeType == "application/x-sh" {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".txt", ".md", ".json", ".xml", ".yaml", ".yml", ".toml", ".ini", ".cfg",
		".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".vue", ".html", ".css", ".scss",
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd",
		".c", ".cpp", ".h", ".hpp", ".java", ".kt", ".rs", ".rb", ".php", ".swift",
		".sql", ".graphql", ".proto", ".tf", ".hcl", ".dockerfile", ".env", ".log":
		return true
	}
	return false
}

func isPDFFile(mimeType, filePath string) bool {
	return mimeType == "application/pdf" || strings.ToLower(filepath.Ext(filePath)) == ".pdf"
}

// nativeVideoMime 浏览器可直接播放的 MIME 类型（无需 HLS 转码）
var nativeVideoMime = map[string]bool{
	"video/mp4":       true,
	"video/webm":      true,
	"video/ogg":       true,
	"video/quicktime": true,
}

// isNativeVideo 判断是否为浏览器原生支持的视频格式
func isNativeVideo(mimeType, filePath string) bool {
	if nativeVideoMime[mimeType] {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp4", ".m4v", ".webm", ".ogg", ".ogv", ".mov":
		return true
	}
	return false
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
	isVideo := isVideoFile(share.MimeType, share.FilePath)
	c.JSON(http.StatusOK, gin.H{
		"id":                     share.ID,
		"name":                   share.Name,
		"file_size":              share.FileSize,
		"mime_type":              share.MimeType,
		"max_views":              share.MaxViews,
		"views":                  share.Views,
		"remaining_views":        remaining,
		"expires_at":             share.ExpiresAt,
		"created_at":             share.CreatedAt,
		"expired":                share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt),
		"exhausted":              share.MaxViews > 0 && share.Views >= share.MaxViews,
		"is_video":               isVideo,
		"is_native_video":        isNativeVideo(share.MimeType, share.FilePath),
		"is_audio":               isAudioFile(share.MimeType, share.FilePath),
		"is_image":               isShareImageFile(share.MimeType, share.FilePath),
		"is_text":                isShareTextFile(share.MimeType, share.FilePath),
		"is_pdf":                 isPDFFile(share.MimeType, share.FilePath),
		"disable_video_download": h.cfg.DisableVideoDownload && isVideo,
		"disable_download":       h.cfg.DisableDownload,
		"has_password":           share.Password != "",
		"watch_enabled":          share.WatchEnabled,
		"record_enabled":         share.RecordEnabled,
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

// UploadRecord POST /api/nfsshare/:id/record?password=xxx&session=xxx&seq=N
// 访客上传录音分片，10 秒无新分片自动触发服务端拼接
func (h *NFSShareHandler) UploadRecord(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	id := c.Param("id")
	share, err := h.db.GetNFSShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
		return
	}
	if !share.RecordEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "该分享未开启录音"})
		return
	}
	if share.Password != "" {
		if !utils.VerifyPassword(c.Query("password"), share.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
			return
		}
	}

	sessionID := c.Query("session")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 session"})
		return
	}
	sessionID = filepath.Base(sessionID) // 防路径穿越

	seqStr := c.Query("seq")
	seq, _ := strconv.Atoi(seqStr)
	clientIP := c.ClientIP()

	file, header, err := c.Request.FormFile("audio")
	if err != nil || header.Size < 1024 {
		// 静音或无 audio：只重置定时器
		h.touchRecordSession(id, sessionID, clientIP)
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": true})
		return
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "分片超过 10MB"})
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".webm"
	}
	chunkDir := filepath.Join("./data/records", id, "chunks", sessionID)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "目录创建失败"})
		return
	}
	chunkFile := filepath.Join(chunkDir, fmt.Sprintf("%06d%s", seq, ext))
	out, err := os.Create(chunkFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件创建失败"})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入失败"})
		return
	}

	// 每次收到有效 chunk，重置 10s 空闲定时器
	h.touchRecordSession(id, sessionID, clientIP)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// touchRecordSession 重置（或创建）session 的空闲定时器
func (h *NFSShareHandler) touchRecordSession(shareID, sessionID, clientIP string) {
	key := shareID + "/" + sessionID
	val, loaded := h.recordSessions.Load(key)
	if loaded {
		sess := val.(*recordSession)
		sess.mu.Lock()
		sess.timer.Reset(recordIdleTimeout)
		sess.mu.Unlock()
		return
	}
	sess := &recordSession{
		shareID:   shareID,
		sessionID: sessionID,
		clientIP:  clientIP,
	}
	sess.timer = time.AfterFunc(recordIdleTimeout, func() {
		h.recordSessions.Delete(key)
		h.finalizeRecording(shareID, sessionID, clientIP)
	})
	// 防止并发重复创建
	if _, existed := h.recordSessions.LoadOrStore(key, sess); existed {
		sess.timer.Stop()
		// 已有 session，重置它
		if v, ok := h.recordSessions.Load(key); ok {
			s := v.(*recordSession)
			s.mu.Lock()
			s.timer.Reset(recordIdleTimeout)
			s.mu.Unlock()
		}
	}
}

// finalizeRecording 拼接同一 session 的所有分片为一个文件，关联到访问日志
func (h *NFSShareHandler) finalizeRecording(shareID, sessionID, clientIP string) {
	chunkDir := filepath.Join("./data/records", shareID, "chunks", sessionID)
	entries, err := os.ReadDir(chunkDir)
	if err != nil || len(entries) == 0 {
		return
	}

	// 按文件名排序（已用 %06d 序号命名）
	var chunks []string
	for _, e := range entries {
		if !e.IsDir() && e.Name() != "list.txt" {
			chunks = append(chunks, filepath.Join(chunkDir, e.Name()))
		}
	}
	if len(chunks) == 0 {
		return
	}

	// 根据第一个 chunk 的扩展名决定输出格式
	firstExt := strings.ToLower(filepath.Ext(chunks[0]))
	outExt := ".webm"
	if firstExt == ".mp4" {
		outExt = ".mp4"
	} else if firstExt == ".ogg" {
		outExt = ".ogg"
	}

	outDir := filepath.Join("./data/records", shareID)
	os.MkdirAll(outDir, 0755)
	b := make([]byte, 8)
	crypto_rand.Read(b)
	outFile := filepath.Join(outDir, hex.EncodeToString(b)+outExt)

	// 用 ffmpeg concat 拼接
	listFile := filepath.Join(chunkDir, "list.txt")
	var listContent string
	for _, c := range chunks {
		abs, _ := filepath.Abs(c)
		listContent += fmt.Sprintf("file '%s'\n", abs)
	}
	if err := os.WriteFile(listFile, []byte(listContent), 0644); err != nil {
		return
	}

	absOut, _ := filepath.Abs(outFile)
	var cmd *exec.Cmd
	if outExt == ".mp4" {
		// mp4 分片需要重新封装，不能直接 concat copy
		cmd = exec.Command("ffmpeg", "-y", "-f", "concat", "-safe", "0",
			"-i", listFile, "-c", "copy", "-movflags", "+faststart", absOut)
	} else {
		cmd = exec.Command("ffmpeg", "-y", "-f", "concat", "-safe", "0",
			"-i", listFile, "-c", "copy", absOut)
	}
	if err := cmd.Run(); err != nil {
		// ffmpeg 不可用时，直接拼接原始字节（webm/ogg 分片可直接拼接，mp4 不行但聊胜于无）
		outF, err2 := os.Create(outFile)
		if err2 != nil {
			return
		}
		for _, c := range chunks {
			data, err3 := os.ReadFile(c)
			if err3 == nil {
				outF.Write(data)
			}
		}
		outF.Close()
	}

	// 清理分片目录
	os.RemoveAll(chunkDir)

	// 关联到访问日志
	audioURL := "/api/nfsshare/" + shareID + "/record/" + filepath.Base(outFile)
	logID := h.db.LastNFSShareLogID(shareID, clientIP)
	if logID > 0 {
		h.db.AppendNFSShareLogAudio(logID, audioURL)
	}
}

// ServeRecord GET /api/nfsshare/:id/record/:filename?admin_password=xxx
// 超管播放录音文件
func (h *NFSShareHandler) ServeRecord(c *gin.Context) {
	if !h.verifyAdmin(c.Query("admin_password")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}
	id := c.Param("id")
	filename := c.Param("filename")
	// 防路径穿越
	filename = filepath.Base(filename)
	path := filepath.Join("./data/records", id, filename)
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "录音不存在"})
		return
	}
	// 根据扩展名设置正确的 Content-Type，确保移动端能播放
	ext := strings.ToLower(filepath.Ext(filename))
	ct := "audio/webm"
	switch ext {
	case ".mp4":
		ct = "audio/mp4"
	case ".ogg":
		ct = "audio/ogg"
	case ".mp3":
		ct = "audio/mpeg"
	}
	c.Header("Content-Type", ct)
	c.File(path)
}


type UpdateNFSShareRequest struct {
	AdminPassword string     `json:"admin_password" binding:"required"`
	MaxViews      int        `json:"max_views"`
	AddViews      int        `json:"add_views"`
	ExpiresAt     *time.Time `json:"expires_at"`
	AddDays       int        `json:"add_days"`
	WatchEnabled  *bool      `json:"watch_enabled"`
	RecordEnabled *bool      `json:"record_enabled"`
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
	newWatchEnabled := share.WatchEnabled
	if req.WatchEnabled != nil {
		newWatchEnabled = *req.WatchEnabled
	}
	newRecordEnabled := share.RecordEnabled
	if req.RecordEnabled != nil {
		newRecordEnabled = *req.RecordEnabled
	}
	if err := h.db.UpdateNFSShare(id, newMaxViews, newExpiresAt, newWatchEnabled, newRecordEnabled); err != nil {
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
	CleanHLSCache(id)
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// Status 返回功能是否启用（公开）
func (h *NFSShareHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled":                h.cfg.Enabled,
		"disable_video_download": h.cfg.DisableVideoDownload,
	})
}

// Stream 原生视频流式播放（公开，消耗1次view）
// 用于 MP4/WebM 等浏览器原生支持的格式，不走 HLS 转码，支持 Range 断点续传
func (h *NFSShareHandler) Stream(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "分享已过期"})
		return
	}
	if share.MaxViews > 0 && share.Views >= share.MaxViews {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("分享次数已用完（最大 %d 次）", share.MaxViews)})
		return
	}
	if !checkSharePassword(c, share, c.Query("password")) {
		return
	}
	if !isVideoFile(share.MimeType, share.FilePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该分享不是视频文件"})
		return
	}

	pp, err := h.parsePath(share.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "挂载点不可用: " + err.Error()})
		return
	}

	h.db.IncrementNFSShareViews(id)
	mimeType := share.MimeType
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = detectMimeType(share.FilePath)
	}

	if strings.ToLower(pp.ms.Config.Type) == "smb" {
		f, size, err := pp.ms.smb.Open(pp.relPath)
		if err != nil {
			h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "file_missing", 0)
			c.JSON(http.StatusNotFound, gin.H{"error": "源文件不存在"})
			return
		}
		defer f.Close()
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "success", size)
		c.Header("Content-Type", mimeType)
		c.Header("Content-Length", fmt.Sprintf("%d", size))
		io.Copy(c.Writer, f)
	} else {
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "success", share.FileSize)
		c.Header("Content-Type", mimeType)
		// 用 http.ServeContent 支持 Range 请求（浏览器 seek）
		f, err := os.Open(pp.absPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "源文件不存在"})
			return
		}
		defer f.Close()
		stat, err := f.Stat()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
			return
		}
		http.ServeContent(c.Writer, c.Request, filepath.Base(share.FilePath), stat.ModTime(), f)
	}
}

// -------- HLS 转码播放 --------

const transcodDir = "./data/transcode"

// QualityPreset 清晰度预设
type QualityPreset struct {
	Name    string // "1080p" / "720p" / "480p" / "360p"
	Height  int    // 目标高度，-1 表示保持原始分辨率
	CRF     int    // 质量因子
	AudioBR string // 音频码率
}

// allQualityPresets 按高到低排列
var allQualityPresets = []QualityPreset{
	{Name: "1080p", Height: 1080, CRF: 22, AudioBR: "192k"},
	{Name: "720p", Height: 720, CRF: 23, AudioBR: "128k"},
	{Name: "480p", Height: 480, CRF: 24, AudioBR: "96k"},
	{Name: "360p", Height: 360, CRF: 25, AudioBR: "64k"},
}

// findPreset 按名称查找预设，找不到返回 false
func findPreset(name string) (QualityPreset, bool) {
	for _, p := range allQualityPresets {
		if p.Name == name {
			return p, true
		}
	}
	return QualityPreset{}, false
}

// availableQualities 根据源视频高度过滤可用清晰度列表
// srcHeight <= 0 表示无法探测，返回全部预设
func availableQualities(srcHeight int) []QualityPreset {
	if srcHeight <= 0 {
		return allQualityPresets
	}
	var list []QualityPreset
	for _, p := range allQualityPresets {
		if p.Height <= srcHeight {
			list = append(list, p)
		}
	}
	if len(list) == 0 {
		// 源分辨率比360p还低，只给360p
		return allQualityPresets[len(allQualityPresets)-1:]
	}
	return list
}

// probeVideoHeight 用 ffprobe 获取视频源高度，失败返回 0
func probeVideoHeight(filePath string) int {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=height",
		"-of", "csv=p=0",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	h, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return h
}

// hlsJobKey 转码任务唯一键（分享ID + 清晰度）
func hlsJobKey(id, quality string) string {
	return id + "/" + quality
}

// HLSQualities 返回该分享可用的清晰度列表（公开）
func (h *NFSShareHandler) HLSQualities(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	id := c.Param("id")
	share, err := h.db.GetNFSShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
		return
	}
	if !checkSharePassword(c, share, c.Query("password")) {
		return
	}
	if !isVideoFile(share.MimeType, share.FilePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非视频文件"})
		return
	}

	srcHeight := 0
	// 仅对本地/NFS 文件运行 ffprobe；SMB 需要先下载，暂跳过探测
	pp, err := h.parsePath(share.FilePath)
	if err == nil && strings.ToLower(pp.ms.Config.Type) != "smb" {
		srcHeight = probeVideoHeight(pp.absPath)
	}

	presets := availableQualities(srcHeight)
	type qualityInfo struct {
		Name   string `json:"name"`
		Height int    `json:"height"`
		Ready  bool   `json:"ready"` // 已转码完成
	}
	list := make([]qualityInfo, 0, len(presets))
	for _, p := range presets {
		outDir := filepath.Join(transcodDir, id, p.Name)
		m3u8 := filepath.Join(outDir, "index.m3u8")
		ready := false
		if data, err := os.ReadFile(m3u8); err == nil && strings.Contains(string(data), "#EXT-X-ENDLIST") {
			ready = true
		}
		list = append(list, qualityInfo{Name: p.Name, Height: p.Height, Ready: ready})
	}
	c.JSON(http.StatusOK, gin.H{"qualities": list, "source_height": srcHeight})
}

// waitForPlayable 等待转码输出目录中至少有2个分片可供播放，或等待转码完成
// 最多等待 timeout 时间；一旦可播放或完成立即返回
func waitForPlayable(outDir string, job *hlsJob, timeout time.Duration) error {
	seg1 := filepath.Join(outDir, "001.ts") // 第2个分片存在 → 已有约20秒内容
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(timeout)
	for {
		select {
		case <-job.done:
			return job.err
		case <-ticker.C:
			if _, err := os.Stat(seg1); err == nil {
				return nil
			}
		case <-deadline:
			return fmt.Errorf("等待转码超时，请重试")
		}
	}
}

// HLSPlaylist 触发指定清晰度的 HLS 转码并返回 m3u8（公开，消耗1次view）
// 有前2个分片即可返回，hls.js 作为 event 流继续拉取后续分片
func (h *NFSShareHandler) HLSPlaylist(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	id := c.Param("id")
	qualityName := c.Param("quality")
	preset, ok := findPreset(qualityName)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的清晰度: " + qualityName})
		return
	}

	share, err := h.db.GetNFSShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
		return
	}
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "分享已过期"})
		return
	}
	if share.MaxViews > 0 && share.Views >= share.MaxViews {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("分享次数已用完（最大 %d 次）", share.MaxViews)})
		return
	}
	if !checkSharePassword(c, share, c.Query("password")) {
		return
	}
	if !isVideoFile(share.MimeType, share.FilePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该分享不是视频文件"})
		return
	}

	outDir := filepath.Join(transcodDir, id, qualityName)
	m3u8Path := filepath.Join(outDir, "index.m3u8")

	// 已转码完成（m3u8 存在且含 ENDLIST 标记）
	if data, err := os.ReadFile(m3u8Path); err == nil && strings.Contains(string(data), "#EXT-X-ENDLIST") {
		// 如果同一个 job 已计过 view（hls.js 轮询到 ENDLIST），不重复计数
		key := hlsJobKey(id, qualityName)
		alreadyCounted := false
		if jobVal, ok := h.hlsJobs.Load(key); ok {
			alreadyCounted = jobVal.(*hlsJob).viewCounted
		}
		if !alreadyCounted {
			h.db.IncrementNFSShareViews(id)
			h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "success", 0)
		}
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
		c.File(m3u8Path)
		return
	}

	// 获取或创建该清晰度的转码 job
	key := hlsJobKey(id, qualityName)
	jobVal, loaded := h.hlsJobs.LoadOrStore(key, &hlsJob{done: make(chan struct{})})
	job := jobVal.(*hlsJob)

	if !loaded {
		// 本次请求触发新转码，计入一次 view
		job.viewCounted = true
		h.db.IncrementNFSShareViews(id)
		h.db.AddNFSShareLog(id, c.ClientIP(), c.GetHeader("User-Agent"), "success", 0)
		go func() {
			defer h.hlsJobs.Delete(key)
			job.err = h.doTranscode(id, share, preset, outDir, m3u8Path)
			close(job.done)
		}()
	}

	// 等待可播放状态（有2个分片即可，最多等2分钟）
	if err := waitForPlayable(outDir, job, 2*time.Minute); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "转码失败: " + err.Error()})
		return
	}

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.File(m3u8Path)
}

// doTranscode 执行 FFmpeg 转码，将源文件转为指定清晰度的 HLS
func (h *NFSShareHandler) doTranscode(id string, share *models.NFSShare, preset QualityPreset, outDir, m3u8Path string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	pp, err := h.parsePath(share.FilePath)
	if err != nil {
		return fmt.Errorf("挂载点不可用: %v", err)
	}

	var sourceFile string
	isSMB := strings.ToLower(pp.ms.Config.Type) == "smb"

	if isSMB {
		// SMB：先把文件复制到本地临时文件
		ext := filepath.Ext(share.FilePath)
		sourceFile = filepath.Join(outDir, "source"+ext)
		if err := h.copyFromSMB(pp, sourceFile); err != nil {
			return fmt.Errorf("复制 SMB 文件失败: %v", err)
		}
		defer os.Remove(sourceFile)
	} else {
		sourceFile = pp.absPath
	}

	segPattern := filepath.Join(outDir, "%03d.ts")
	args := []string{"-y", "-i", sourceFile,
		"-c:v", "libx264", "-preset", "fast", "-crf", strconv.Itoa(preset.CRF),
		"-c:a", "aac", "-b:a", preset.AudioBR,
	}
	if preset.Height > 0 {
		// 按高度缩放，宽度自适应（-2 保证被2整除）
		args = append(args, "-vf", fmt.Sprintf("scale=-2:%d", preset.Height))
	}
	args = append(args,
		"-f", "hls",
		"-hls_time", "10",
		"-hls_playlist_type", "event", // event 类型：分片增量写入，完成后追加 EXT-X-ENDLIST
		"-hls_segment_filename", segPattern,
		m3u8Path,
	)
	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[NFSShare] HLS 转码失败 %s: %v\n%s", id, err, string(out))
		// 清理不完整的输出
		os.RemoveAll(outDir)
		return fmt.Errorf("ffmpeg 退出错误: %v", err)
	}
	log.Printf("[NFSShare] HLS 转码完成 %s", id)
	return nil
}

// copyFromSMB 把 SMB 文件流式复制到本地路径
func (h *NFSShareHandler) copyFromSMB(pp *parsedPath, dst string) error {
	f, _, err := pp.ms.smb.Open(pp.relPath)
	if err != nil {
		return err
	}
	defer f.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, f)
	return err
}

// HLSSegment 返回 HLS 分片文件（公开，不消耗 view）
func (h *NFSShareHandler) HLSSegment(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	id := c.Param("id")
	quality := c.Param("quality")
	segment := c.Param("segment")
	// 防路径穿越
	if strings.Contains(segment, "..") || strings.Contains(segment, "/") ||
		strings.Contains(quality, "..") || strings.Contains(quality, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法路径"})
		return
	}
	segPath := filepath.Join(transcodDir, id, quality, segment)
	if _, err := os.Stat(segPath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分片不存在"})
		return
	}
	c.Header("Content-Type", "video/mp2t")
	c.File(segPath)
}

// CleanHLSCache 删除指定分享的 HLS 转码缓存（删除分享时调用）
func CleanHLSCache(id string) {
	os.RemoveAll(filepath.Join(transcodDir, id))
}

// -------- 一起看 Watch Party --------

var watchUpgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
}

// watchMsg 客户端 → 服务器 消息
type watchMsg struct {
	Type      string  `json:"type"`              // join | chat | danmaku | sync | voice_*
	Nickname  string  `json:"nickname"`           // join 时设置
	Text      string  `json:"text"`               // chat / danmaku 内容
	Action    string  `json:"action"`             // sync: play | pause | seek
	Time      float64 `json:"time"`               // sync: 当前播放时间（秒）
	To        string  `json:"to,omitempty"`       // WebRTC: 目标 peerID
	SDP       string  `json:"sdp,omitempty"`      // WebRTC: offer/answer SDP
	Candidate string  `json:"candidate,omitempty"` // WebRTC: ICE candidate JSON
}

// voicePeerInfo 语音参与者信息
type voicePeerInfo struct {
	PeerID   string `json:"peer_id"`
	Nickname string `json:"nickname"`
}

// watchBroadcast 服务器 → 所有客户端 消息
type watchBroadcast struct {
	Type         string          `json:"type"`               // joined | left | chat | danmaku | sync | voice_*
	Nickname     string          `json:"nickname,omitempty"` // 消息来源
	Text         string          `json:"text,omitempty"`
	Action       string          `json:"action,omitempty"`
	Time         float64         `json:"time,omitempty"`
	Count        int             `json:"count,omitempty"`         // viewers 类型：当前人数
	IsHost       bool            `json:"is_host,omitempty"`       // 是否房主
	PeerID       string          `json:"peer_id,omitempty"`       // WebRTC: 发起者 peerID
	From         string          `json:"from,omitempty"`          // WebRTC: 来源 peerID
	SDP          string          `json:"sdp,omitempty"`           // WebRTC: offer/answer SDP
	Candidate    string          `json:"candidate,omitempty"`     // WebRTC: ICE candidate JSON
	Peers        []voicePeerInfo `json:"peers,omitempty"`         // voice_peers: 已在语音的成员
	VoiceEnabled bool            `json:"voice_enabled,omitempty"` // voice_state: 语音频道是否开启
	HostActive   bool            `json:"host_active,omitempty"`   // force_watch: 房主是否在线
}

// watchClient 单个 WebSocket 连接
type watchClient struct {
	conn        *websocket.Conn
	nickname    string
	isHost      bool
	send        chan []byte
	peerID      string // WebRTC 信令唯一标识
	voiceActive bool   // 是否已加入语音
}

// watchRoom 一个视频分享对应的观看室
type watchRoom struct {
	mu           sync.RWMutex
	clients      map[*watchClient]bool
	byPeer       map[string]*watchClient // peerID → client
	lastAction   string                  // 最近一次 sync action: "play" | "pause"
	lastTime     float64                 // 最近一次 sync 时间（秒）
	lastSyncAt   time.Time               // 最近一次 sync 时刻（用于估算当前进度）
	voiceEnabled bool                    // 语音频道是否开启（由房主控制）
	hostActive   bool                    // 是否有房主在线（控制强制一起看模式）
}

func randomPeerID() string {
	b := make([]byte, 6)
	crypto_rand.Read(b)
	return hex.EncodeToString(b)
}

func newWatchRoom() *watchRoom {
	return &watchRoom{
		clients: make(map[*watchClient]bool),
		byPeer:  make(map[string]*watchClient),
	}
}

func (r *watchRoom) add(c *watchClient) {
	r.mu.Lock()
	r.clients[c] = true
	if c.peerID != "" {
		r.byPeer[c.peerID] = c
	}
	r.mu.Unlock()
}

func (r *watchRoom) remove(c *watchClient) {
	r.mu.Lock()
	delete(r.clients, c)
	if c.peerID != "" {
		delete(r.byPeer, c.peerID)
	}
	r.mu.Unlock()
}

// sendToPeer 向指定 peerID 发送定向消息（WebRTC 信令）
func (r *watchRoom) sendToPeer(peerID string, msg watchBroadcast) {
	data, _ := json.Marshal(msg)
	r.mu.RLock()
	c, ok := r.byPeer[peerID]
	if ok {
		select {
		case c.send <- data:
		default:
		}
	}
	r.mu.RUnlock()
}

// voicePeers 返回当前已加入语音的成员（排除 exclude）
func (r *watchRoom) voicePeers(exclude *watchClient) []voicePeerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var peers []voicePeerInfo
	for c := range r.clients {
		if c != exclude && c.voiceActive {
			peers = append(peers, voicePeerInfo{PeerID: c.peerID, Nickname: c.nickname})
		}
	}
	return peers
}

func (r *watchRoom) count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

func (r *watchRoom) broadcast(msg watchBroadcast, exclude *watchClient) {
	data, _ := json.Marshal(msg)
	r.mu.RLock()
	defer r.mu.RUnlock()
	for c := range r.clients {
		if c == exclude {
			continue
		}
		select {
		case c.send <- data:
		default:
		}
	}
}

func (r *watchRoom) broadcastAll(msg watchBroadcast) {
	r.broadcast(msg, nil)
}

// WatchWS 处理一起看 WebSocket 连接
func (h *NFSShareHandler) WatchWS(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	id := c.Param("id")
	share, err := h.db.GetNFSShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
		return
	}
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "分享已过期"})
		return
	}
	if !checkSharePassword(c, share, c.Query("password")) {
		return
	}

	nickname := strings.TrimSpace(c.Query("nickname"))
	if nickname == "" {
		nickname = "匿名用户"
	}
	isHost := c.Query("admin_password") != "" && h.verifyAdmin(c.Query("admin_password"))

	conn, err := watchUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// 获取或创建 watchRoom
	roomVal, _ := h.watchRooms.LoadOrStore(id, newWatchRoom())
	room := roomVal.(*watchRoom)

	isPending := nickname == "__pending__"

	client := &watchClient{
		conn:     conn,
		nickname: nickname,
		isHost:   isHost,
		send:     make(chan []byte, 64),
		peerID:   randomPeerID(),
	}
	room.add(client)

	// 房主加入：激活强制一起看模式，广播给已在线的所有人
	if isHost {
		room.mu.Lock()
		room.hostActive = true
		room.mu.Unlock()
		room.broadcast(watchBroadcast{
			Type:       "force_watch",
			HostActive: true,
		}, client) // 不发给自己
	}

	// pending 连接不广播 joined，不影响人数
	if !isPending {
		room.broadcastAll(watchBroadcast{
			Type:     "joined",
			Nickname: nickname,
			IsHost:   isHost,
			Count:    room.count(),
			PeerID:   client.peerID,
		})
	}

	// 将当前播放状态发给新加入者（如果有）
	room.mu.RLock()
	lastAction := room.lastAction
	lastTime := room.lastTime
	lastSyncAt := room.lastSyncAt
	voiceEnabled := room.voiceEnabled
	hostActive := room.hostActive
	room.mu.RUnlock()
	if lastAction != "" {
		// 估算当前时间：如果是 play，加上已过去的秒数
		estimatedTime := lastTime
		if lastAction == "play" && !lastSyncAt.IsZero() {
			estimatedTime += time.Since(lastSyncAt).Seconds()
		}
		data, _ := json.Marshal(watchBroadcast{
			Type:   "sync",
			Action: lastAction,
			Time:   estimatedTime,
		})
		select {
		case client.send <- data:
		default:
		}
	}
	// 将当前语音频道状态发给新加入者
	{
		data, _ := json.Marshal(watchBroadcast{
			Type:         "voice_state",
			VoiceEnabled: voiceEnabled,
		})
		select {
		case client.send <- data:
		default:
		}
	}
	// 新加入的非房主：如果房主在线，通知其进入强制一起看模式
	if !isHost && hostActive {
		data, _ := json.Marshal(watchBroadcast{
			Type:       "force_watch",
			HostActive: true,
		})
		select {
		case client.send <- data:
		default:
		}
	}

	// 写 goroutine
	go func() {
		defer conn.Close()
		for data := range client.send {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				break
			}
		}
	}()

	// ping-pong 心跳：每 30s 发一次 ping，60s 内没有任何消息则断开
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	pingTicker := time.NewTicker(30 * time.Second)
	go func() {
		defer pingTicker.Stop()
		for range pingTicker.C {
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}()

	// 读循环（主 goroutine）
	defer func() {
		pingTicker.Stop()
		room.remove(client)
		close(client.send)
		// 房主离开：关闭强制一起看模式
		if isHost {
			room.mu.Lock()
			room.hostActive = false
			room.mu.Unlock()
			room.broadcastAll(watchBroadcast{
				Type:       "force_watch",
				HostActive: false,
			})
		}
		// 语音成员断开：通知其他人关闭 RTCPeerConnection
		if client.voiceActive {
			room.broadcastAll(watchBroadcast{
				Type:   "voice_leave",
				PeerID: client.peerID,
			})
		}
		// 广播：有人离开（pending 连接不广播）
		if !isPending {
			room.broadcastAll(watchBroadcast{
				Type:     "left",
				Nickname: nickname,
				Count:    room.count(),
			})
		}
		// 房间空了就清理
		if room.count() == 0 {
			h.watchRooms.Delete(id)
		}
	}()

	conn.SetReadLimit(65536) // 64KB，SDP/ICE 可能较大
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		var msg watchMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "chat":
			text := strings.TrimSpace(msg.Text)
			if text == "" {
				continue
			}
			room.broadcastAll(watchBroadcast{
				Type:     "chat",
				Nickname: nickname,
				Text:     text,
				IsHost:   isHost,
			})
		case "danmaku":
			text := strings.TrimSpace(msg.Text)
			if text == "" {
				continue
			}
			room.broadcastAll(watchBroadcast{
				Type:     "danmaku",
				Nickname: nickname,
				Text:     text,
			})
		case "sync":
			// 只有房主可以同步播放状态
			if !isHost {
				continue
			}
			// 记录最新播放状态，供新加入者使用
			room.mu.Lock()
			room.lastAction = msg.Action
			room.lastTime = msg.Time
			room.lastSyncAt = time.Now()
			room.mu.Unlock()
			room.broadcast(watchBroadcast{
				Type:   "sync",
				Action: msg.Action,
				Time:   msg.Time,
			}, client) // 不发给自己

		// ---- WebRTC 语音信令 ----
		case "voice_toggle":
			// 只有房主可以开关语音频道
			if !isHost {
				continue
			}
			room.mu.Lock()
			room.voiceEnabled = !room.voiceEnabled
			enabled := room.voiceEnabled
			room.mu.Unlock()
			// 如果关闭语音，强制所有语音成员离开
			if !enabled {
				room.mu.Lock()
				for c := range room.clients {
					if c.voiceActive {
						c.voiceActive = false
					}
				}
				room.mu.Unlock()
			}
			room.broadcastAll(watchBroadcast{
				Type:         "voice_state",
				VoiceEnabled: enabled,
			})

		case "voice_join":
			// 语音频道未开启时拒绝（房主除外）
			room.mu.RLock()
			voiceOpen := room.voiceEnabled
			room.mu.RUnlock()
			if !voiceOpen && !isHost {
				continue
			}
			// 标记自己已加入语音
			client.voiceActive = true
			// 将已在语音的成员列表发回给本人，让其主动发起 offer
			existingPeers := room.voicePeers(client)
			if len(existingPeers) > 0 {
				data, _ := json.Marshal(watchBroadcast{
					Type:  "voice_peers",
					Peers: existingPeers,
				})
				select {
				case client.send <- data:
				default:
				}
			}
			// 广播给其他人：新成员加入了语音
			room.broadcast(watchBroadcast{
				Type:     "voice_join",
				PeerID:   client.peerID,
				Nickname: nickname,
			}, client)

		case "voice_leave":
			client.voiceActive = false
			room.broadcast(watchBroadcast{
				Type:   "voice_leave",
				PeerID: client.peerID,
			}, client)

		case "voice_offer":
			// 转发 offer 给指定 peer
			room.sendToPeer(msg.To, watchBroadcast{
				Type: "voice_offer",
				From: client.peerID,
				SDP:  msg.SDP,
			})

		case "voice_answer":
			// 转发 answer 给指定 peer
			room.sendToPeer(msg.To, watchBroadcast{
				Type: "voice_answer",
				From: client.peerID,
				SDP:  msg.SDP,
			})

		case "voice_ice":
			// 转发 ICE candidate 给指定 peer
			room.sendToPeer(msg.To, watchBroadcast{
				Type:      "voice_ice",
				From:      client.peerID,
				Candidate: msg.Candidate,
			})
		}
	}
}

// -------- TURN 临时凭证 --------

// GetTurnCredentials 生成 coturn use-auth-secret 临时凭证
// 算法：username = "<expiry_unix>:user"，password = base64(HMAC-SHA1(secret, username))
func (h *NFSShareHandler) GetTurnCredentials(c *gin.Context) {
	cfg := h.cfg
	turnCfg := config.Get().TURN
	if turnCfg.Secret == "" || turnCfg.Host == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "TURN 未配置"})
		return
	}
	_ = cfg // suppress unused warning

	ttl := turnCfg.TTL
	if ttl <= 0 {
		ttl = 3600
	}
	expiry := time.Now().Add(time.Duration(ttl) * time.Second).Unix()
	username := fmt.Sprintf("%d:user", expiry)

	mac := hmac.New(sha1.New, []byte(turnCfg.Secret))
	mac.Write([]byte(username))
	password := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	port := turnCfg.Port
	if port == 0 {
		port = 3478
	}
	host := net.JoinHostPort(turnCfg.Host, strconv.Itoa(port))

	c.JSON(http.StatusOK, gin.H{
		"stun": "stun:" + host,
		"turn": "turn:" + host,
		"username": username,
		"credential": password,
		"ttl": ttl,
	})
}

// -------- 文件上传 --------

type uploadInitRequest struct {
	AdminPassword string `json:"admin_password" binding:"required"`
	Filename      string `json:"filename" binding:"required"`
	TotalSize     int64  `json:"total_size"`
	TotalChunks   int    `json:"total_chunks" binding:"required,min=1"`
	TargetDir     string `json:"target_dir"` // 可选，格式 "mountName/subdir"，空则存到 __uploads__
}

// UploadInit 初始化分片上传，返回 token
func (h *NFSShareHandler) UploadInit(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	var req uploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.verifyAdmin(req.AdminPassword) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}
	// 文件名安全处理
	filename := filepath.Base(req.Filename)
	if filename == "." || filename == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法文件名"})
		return
	}
	// 验证 target_dir（如果指定）
	targetDir := strings.TrimSpace(req.TargetDir)
	if targetDir != "" {
		pp, err := h.parsePath(targetDir + "/.keep")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "目标目录无效: " + err.Error()})
			return
		}
		// 对于 local/nfs，确保目录存在
		if strings.ToLower(pp.ms.Config.Type) != "smb" {
			dirPath := filepath.Dir(pp.absPath)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建目标目录"})
				return
			}
		}
	}
	// 生成 token
	b := make([]byte, 8)
	crypto_rand.Read(b)
	token := hex.EncodeToString(b)
	tmpDir := filepath.Join("./data/uploads/.tmp", token)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时目录失败"})
		return
	}
	// 写入元数据
	meta := filename + "\n" + targetDir
	if err := os.WriteFile(filepath.Join(tmpDir, ".meta"), []byte(meta), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入元数据失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "filename": filename})
}

// UploadChunk 上传单个分片
func (h *NFSShareHandler) UploadChunk(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 token"})
		return
	}
	tmpDir := filepath.Join("./data/uploads/.tmp", token)
	if _, err := os.Stat(tmpDir); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 token"})
		return
	}
	chunkIndex := c.PostForm("chunk_index")
	if chunkIndex == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 chunk_index"})
		return
	}
	idx, err := strconv.Atoi(chunkIndex)
	if err != nil || idx < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chunk_index 无效"})
		return
	}
	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 chunk 数据"})
		return
	}
	defer file.Close()
	chunkPath := filepath.Join(tmpDir, fmt.Sprintf("%05d", idx))
	dst, err := os.Create(chunkPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入分片失败"})
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入分片失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"received": idx})
}

type uploadCompleteRequest struct {
	AdminPassword string `json:"admin_password" binding:"required"`
	TotalChunks   int    `json:"total_chunks" binding:"required,min=1"`
}

// UploadComplete 合并分片，写入目标位置，返回 file_path
func (h *NFSShareHandler) UploadComplete(c *gin.Context) {
	if !h.checkEnabled(c) {
		return
	}
	token := c.Param("token")
	var req uploadCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.verifyAdmin(req.AdminPassword) {
		c.JSON(http.StatusForbidden, gin.H{"error": "超管密码错误"})
		return
	}
	tmpDir := filepath.Join("./data/uploads/.tmp", token)
	metaBytes, err := os.ReadFile(filepath.Join(tmpDir, ".meta"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 token"})
		return
	}
	parts := strings.SplitN(string(metaBytes), "\n", 2)
	filename := parts[0]
	targetDir := ""
	if len(parts) > 1 {
		targetDir = strings.TrimSpace(parts[1])
	}

	// 决定写入位置
	var filePath string // 最终 file_path（mountName/rel）
	var writeFunc func(r io.Reader) error

	if targetDir == "" {
		// 默认写到 __uploads__
		destPath := filepath.Join("./data/uploads", filename)
		if _, err := os.Stat(destPath); err == nil {
			ext := filepath.Ext(filename)
			base := strings.TrimSuffix(filename, ext)
			filename = fmt.Sprintf("%s_%d%s", base, time.Now().UnixMilli(), ext)
			destPath = filepath.Join("./data/uploads", filename)
		}
		filePath = "__uploads__/" + filename
		writeFunc = func(r io.Reader) error {
			f, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(f, r)
			return err
		}
	} else {
		// 写到指定挂载点目录
		pp, err := h.parsePath(targetDir + "/" + filename)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "目标路径无效: " + err.Error()})
			return
		}
		// 处理同名冲突
		isSMB := strings.ToLower(pp.ms.Config.Type) == "smb"
		if isSMB {
			if _, statErr := pp.ms.smb.Stat(pp.relPath); statErr == nil {
				ext := filepath.Ext(filename)
				base := strings.TrimSuffix(filename, ext)
				filename = fmt.Sprintf("%s_%d%s", base, time.Now().UnixMilli(), ext)
				pp, _ = h.parsePath(targetDir + "/" + filename)
			}
			filePath = targetDir + "/" + filename
			writeFunc = func(r io.Reader) error {
				w, err := pp.ms.smb.Create(pp.relPath)
				if err != nil {
					return err
				}
				defer w.Close()
				_, err = io.Copy(w, r)
				return err
			}
		} else {
			if _, statErr := os.Stat(pp.absPath); statErr == nil {
				ext := filepath.Ext(filename)
				base := strings.TrimSuffix(filename, ext)
				filename = fmt.Sprintf("%s_%d%s", base, time.Now().UnixMilli(), ext)
				pp, _ = h.parsePath(targetDir + "/" + filename)
			}
			os.MkdirAll(filepath.Dir(pp.absPath), 0755)
			filePath = targetDir + "/" + filename
			writeFunc = func(r io.Reader) error {
				f, err := os.Create(pp.absPath)
				if err != nil {
					return err
				}
				defer f.Close()
				_, err = io.Copy(f, r)
				return err
			}
		}
	}

	// 合并分片并写入目标
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		errCh <- writeFunc(pr)
	}()
	for i := 0; i < req.TotalChunks; i++ {
		chunkPath := filepath.Join(tmpDir, fmt.Sprintf("%05d", i))
		chunk, err := os.Open(chunkPath)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("分片 %d 缺失", i))
			<-errCh
			os.RemoveAll(tmpDir)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("分片 %d 缺失", i)})
			return
		}
		_, copyErr := io.Copy(pw, chunk)
		chunk.Close()
		if copyErr != nil {
			pw.CloseWithError(copyErr)
			<-errCh
			os.RemoveAll(tmpDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "合并分片失败"})
			return
		}
	}
	pw.Close()
	if writeErr := <-errCh; writeErr != nil {
		os.RemoveAll(tmpDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入目标文件失败: " + writeErr.Error()})
		return
	}
	os.RemoveAll(tmpDir)
	c.JSON(http.StatusOK, gin.H{
		"file_path": filePath,
		"filename":  filename,
	})
}
