package handlers

import (
	"crypto/hmac"
	crypto_rand "crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"devtools/config"
	"devtools/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

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
		"stun":       "stun:" + host,
		"turn":       "turn:" + host,
		"username":   username,
		"credential": password,
		"ttl":        ttl,
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
