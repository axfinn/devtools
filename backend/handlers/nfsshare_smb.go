package handlers

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
	smb2 "github.com/hirochachacha/go-smb2"
)

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
	db             *models.DB
	cfg            config.NFSShareConfig
	mounts         map[string]*MountStatus
	mu             sync.RWMutex
	hlsJobs        sync.Map // key: shareID, value: *hlsJob
	watchRooms     sync.Map // key: shareID, value: *watchRoom
	recordSessions sync.Map // key: shareID+"/"+sessionID, value: *recordSession
}

type recordSession struct {
	shareID   string
	sessionID string
	clientIP  string
	timer     *time.Timer
	mu        sync.Mutex
}

const recordIdleTimeout = 10 * time.Second

// NewNFSShareHandler 创建 Handler 并初始化所有挂载点
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
