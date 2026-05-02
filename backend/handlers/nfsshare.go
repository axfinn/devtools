package handlers

import (
	crypto_rand "crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
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
	case ".txt", ".md", ".json", ".xml", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".csv", ".tsv",
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
	Type      string  `json:"type"`                // join | chat | danmaku | sync | voice_*
	Nickname  string  `json:"nickname"`            // join 时设置
	Text      string  `json:"text"`                // chat / danmaku 内容
	Action    string  `json:"action"`              // sync: play | pause | seek
	Time      float64 `json:"time"`                // sync: 当前播放时间（秒）
	To        string  `json:"to,omitempty"`        // WebRTC: 目标 peerID
	SDP       string  `json:"sdp,omitempty"`       // WebRTC: offer/answer SDP
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
