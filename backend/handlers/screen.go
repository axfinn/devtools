package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ScreenHandler 屏幕共享会话 + WebRTC 信令转发。
// 内存态房间(sessions[id]→room)仅做信令中转,不持久化 viewer 列表;
// 房间空了就自动删除,无数据库轮询开销。
type ScreenHandler struct {
	db         *models.DB
	cfg        config.Config
	rooms      sync.Map // id(string) → *screenRoom
	relayRooms sync.Map // id(string) → *relayRoom (binary frame fallback)
	upgrader   websocket.Upgrader
}

// screenWSMsg 客户端 ↔ 服务器 信令消息(双向共用同一组字段)。
//   - 客户端→服务器:填 SDP / Candidate / Kind / Data / To
//   - 服务器→客户端:填 From / Viewer / Session / ViewerList(按需)
type screenWSMsg struct {
	Type       string                `json:"type"`
	SDP        string                `json:"sdp,omitempty"`
	Candidate  json.RawMessage       `json:"candidate,omitempty"`
	Kind       string                `json:"kind,omitempty"`
	Data       json.RawMessage       `json:"data,omitempty"`
	To         string                `json:"to,omitempty"`
	From       string                `json:"from,omitempty"`
	Viewer     *screenViewer         `json:"viewer,omitempty"`
	Session    *models.ScreenSession `json:"session,omitempty"`
	ViewerList []screenViewer        `json:"viewer_list,omitempty"`
	Error      string                `json:"error,omitempty"`
}

type screenViewer struct {
	PeerID   string `json:"peer_id"`
	Nickname string `json:"nickname"`
	IsAnon   bool   `json:"is_anon"`
}

type screenClient struct {
	conn     *websocket.Conn
	send     chan []byte
	peerID   string
	isHost   bool
	nickname string // viewer 的昵称(空=匿名)
}

type screenRoom struct {
	mu       sync.RWMutex
	host     *screenClient
	viewers  map[string]*screenClient
	lastUsed time.Time
}

const screenRoomIdleTTL = 5 * time.Minute

func newScreenRoom() *screenRoom {
	return &screenRoom{
		viewers:  make(map[string]*screenClient),
		lastUsed: time.Now(),
	}
}

func (r *screenRoom) addViewer(c *screenClient) {
	r.mu.Lock()
	r.viewers[c.peerID] = c
	r.lastUsed = time.Now()
	r.mu.Unlock()
}

func (r *screenRoom) removeViewer(peerID string) {
	r.mu.Lock()
	delete(r.viewers, peerID)
	r.lastUsed = time.Now()
	r.mu.Unlock()
}

func (r *screenRoom) viewerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.viewers)
}

func (r *screenRoom) viewerList() []screenViewer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]screenViewer, 0, len(r.viewers))
	for _, c := range r.viewers {
		out = append(out, screenViewer{
			PeerID:   c.peerID,
			Nickname: c.nickname,
			IsAnon:   c.nickname == "",
		})
	}
	return out
}

func (r *screenRoom) sendJSON(c *screenClient, msg screenWSMsg) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}

func (r *screenRoom) broadcastToViewers(msg screenWSMsg) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, _ := json.Marshal(msg)
	for _, c := range r.viewers {
		select {
		case c.send <- data:
		default:
		}
	}
}

func (r *screenRoom) sendToHost(msg screenWSMsg) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.host == nil {
		return
	}
	data, _ := json.Marshal(msg)
	select {
	case r.host.send <- data:
	default:
	}
}

func (r *screenRoom) sendToViewer(peerID string, msg screenWSMsg) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.viewers[peerID]
	if !ok {
		return
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

func (r *screenRoom) touch() {
	r.mu.Lock()
	r.lastUsed = time.Now()
	r.mu.Unlock()
}

func NewScreenHandler(db *models.DB, cfg config.Config) *ScreenHandler {
	return &ScreenHandler{
		db:  db,
		cfg: cfg,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  8192,
			WriteBufferSize: 8192,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

// -------- HTTP handlers --------

type screenCreateReq struct {
	Title              string `json:"title"`
	Password           string `json:"password"`
	AllowRemoteControl bool   `json:"allow_remote_control"`
	DurationMinutes    int    `json:"duration_minutes"`
}

// ScreenCreate POST /api/screen/sessions — 创建会话,要求 askit Bearer token。
func (h *ScreenHandler) ScreenCreate(c *gin.Context) {
	userID, ok := askitBearerUser(c, h.db)
	if !ok {
		return
	}
	var req screenCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if req.Password != "" && len(req.Password) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password_too_long"})
		return
	}
	if req.DurationMinutes < 0 {
		req.DurationMinutes = 0
	}
	if req.DurationMinutes > 24*60 {
		req.DurationMinutes = 24 * 60
	}
	var expiresAt *time.Time
	if req.DurationMinutes > 0 {
		t := time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute)
		expiresAt = &t
	}
	pwHash := ""
	if req.Password != "" {
		sum := sha256.Sum256([]byte(req.Password))
		pwHash = hex.EncodeToString(sum[:])
	}
	session, err := h.db.CreateScreenSession(userID, strings.TrimSpace(req.Title), pwHash, req.AllowRemoteControl, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"session":    publicScreenSession(session),
		"viewer_url": buildScreenViewerURL(c, session.ID),
	})
}

// ScreenInfo GET /api/screen/sessions/:id/info — 公开元数据。
func (h *ScreenHandler) ScreenInfo(c *gin.Context) {
	id := c.Param("id")
	s, err := h.db.GetScreenSessionPublic(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
		return
	}
	if !screenSessionJoinable(s) {
		c.JSON(http.StatusGone, gin.H{"error": "session_unavailable", "status": s.Status})
		return
	}
	c.JSON(http.StatusOK, gin.H{"session": publicScreenSession(s)})
}

type screenCheckPasswordReq struct {
	Password string `json:"password"`
}

// ScreenCheckPassword POST /api/screen/sessions/:id/check-password — 校验 viewer 密码。
func (h *ScreenHandler) ScreenCheckPassword(c *gin.Context) {
	id := c.Param("id")
	s, err := h.db.GetScreenSession(id) // 用内部版本,需保留 password 哈希做比对
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
		return
	}
	if !screenSessionJoinable(s) {
		c.JSON(http.StatusGone, gin.H{"error": "session_unavailable"})
		return
	}
	if s.Password == "" {
		c.JSON(http.StatusOK, gin.H{"ok": true, "need_password": false})
		return
	}
	var req screenCheckPasswordReq
	_ = c.ShouldBindJSON(&req)
	sum := sha256.Sum256([]byte(req.Password))
	if hex.EncodeToString(sum[:]) != s.Password {
		c.JSON(http.StatusOK, gin.H{"ok": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "need_password": true})
}

// ScreenStop DELETE /api/screen/sessions/:id — 仅 host 可结束。
func (h *ScreenHandler) ScreenStop(c *gin.Context) {
	id := c.Param("id")
	userID, ok := askitBearerUser(c, h.db)
	if !ok {
		return
	}
	if err := h.db.StopScreenSession(id, userID); err != nil {
		if err == models.ErrScreenSessionNotStoppable {
			c.JSON(http.StatusConflict, gin.H{"error": "not_stoppable"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stop_failed"})
		return
	}
	if rv, ok := h.rooms.Load(id); ok {
		room := rv.(*screenRoom)
		room.broadcastToViewers(screenWSMsg{Type: "host_disconnected"})
		room.mu.Lock()
		room.host = nil
		room.mu.Unlock()
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ScreenMySessions GET /api/screen/sessions/mine — 当前 askit 用户的活跃会话。
func (h *ScreenHandler) ScreenMySessions(c *gin.Context) {
	userID, ok := askitBearerUser(c, h.db)
	if !ok {
		return
	}
	list, err := h.db.ListActiveScreenSessionsByHost(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_failed"})
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, s := range list {
		out = append(out, gin.H{
			"id":           s.ID,
			"title":        s.Title,
			"status":       s.Status,
			"has_password": s.Password != "",
			"created_at":   s.CreatedAt,
			"expires_at":   s.ExpiresAt,
			"viewer_url":   buildScreenViewerURL(c, s.ID),
		})
	}
	c.JSON(http.StatusOK, gin.H{"sessions": out})
}

// ScreenTurnCredentials GET /api/screen/turn-credentials — 公共 TURN 临时凭证。
// 与 nfsshare 同源算法(coturn use-auth-secret),共用全局 TURN 配置。
func (h *ScreenHandler) ScreenTurnCredentials(c *gin.Context) {
	turnCfg := h.cfg.TURN
	if turnCfg.Secret == "" || turnCfg.Host == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "turn_not_configured"})
		return
	}
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

// -------- WebSocket 信令 --------

// ScreenSignalingWS GET /api/screen/sessions/:id/ws?role=host|viewer&token=...&password=...&nickname=...
//
//	host:   token 必须是该 session host 的 askit access token
//	viewer: 若 session 有密码,password 必须匹配;否则任何人都可加入
func (h *ScreenHandler) ScreenSignalingWS(c *gin.Context) {
	id := c.Param("id")
	role := strings.ToLower(c.Query("role"))
	if role != "host" && role != "viewer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_role"})
		return
	}
	session, err := h.db.GetScreenSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
		return
	}
	if !screenSessionJoinable(session) {
		c.JSON(http.StatusGone, gin.H{"error": "session_unavailable"})
		return
	}

	if role == "host" {
		userID, ok := askitBearerQueryUser(c, h.db, "token")
		if !ok {
			return
		}
		if userID != session.HostUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "not_host"})
			return
		}
	} else if session.Password != "" {
		provided := sha256.Sum256([]byte(c.Query("password")))
		if hex.EncodeToString(provided[:]) != session.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "bad_password"})
			return
		}
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	peerID := randomPeerID()

	roomVal, _ := h.rooms.LoadOrStore(id, newScreenRoom())
	room := roomVal.(*screenRoom)

	client := &screenClient{
		conn:   conn,
		send:   make(chan []byte, 64),
		peerID: peerID,
		isHost: role == "host",
	}

	if role == "host" {
		room.mu.Lock()
		if room.host != nil {
			room.sendJSON(room.host, screenWSMsg{Type: "error", Error: "host_replaced"})
			close(room.host.send)
			_ = room.host.conn.Close()
		}
		room.host = client
		room.lastUsed = time.Now()
		room.mu.Unlock()
		// host 看到当前 viewer 列表(便于断线重连时主动 offer)
		room.sendJSON(client, screenWSMsg{
			Type:       "joined",
			From:       peerID,
			Session:    publicScreenSession(session),
			ViewerList: room.viewerList(),
		})
	} else {
		client.nickname = strings.TrimSpace(c.Query("nickname"))
		if len(client.nickname) > 32 {
			client.nickname = client.nickname[:32]
		}
		room.addViewer(client)
		room.sendJSON(client, screenWSMsg{
			Type:    "joined",
			From:    peerID,
			Session: publicScreenSession(session),
		})
		room.sendToHost(screenWSMsg{
			Type:   "viewer_joined",
			From:   peerID,
			Viewer: &screenViewer{PeerID: peerID, Nickname: client.nickname, IsAnon: client.nickname == ""},
		})
	}

	// 写协程
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC screen ws write: %v", r)
			}
		}()
		defer conn.Close()
		for data := range client.send {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}()

	// ping 心跳:每 30s 写 ping,60s 无消息即断开
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	pingTicker := time.NewTicker(30 * time.Second)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC screen ws ping: %v", r)
			}
		}()
		defer pingTicker.Stop()
		for range pingTicker.C {
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}()

	defer func() {
		pingTicker.Stop()
		_ = conn.Close()
		close(client.send)
		if client.isHost {
			room.mu.Lock()
			if room.host == client {
				room.host = nil
			}
			room.mu.Unlock()
			room.broadcastToViewers(screenWSMsg{Type: "host_disconnected"})
		} else {
			room.removeViewer(client.peerID)
			room.sendToHost(screenWSMsg{Type: "viewer_left", From: client.peerID})
		}
		if room.host == nil && room.viewerCount() == 0 {
			go func() {
				time.Sleep(screenRoomIdleTTL)
				if room.host == nil && room.viewerCount() == 0 {
					h.rooms.Delete(id)
				}
			}()
		}
	}()

	conn.SetReadLimit(1 << 20) // 1MB
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		room.touch()

		var msg screenWSMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "offer":
			if client.isHost {
				continue
			}
			room.sendToHost(screenWSMsg{Type: "offer", From: client.peerID, SDP: msg.SDP})
		case "answer":
			if !client.isHost {
				continue
			}
			room.sendToViewer(msg.To, screenWSMsg{Type: "answer", From: client.peerID, SDP: msg.SDP})
		case "ice":
			if client.isHost {
				room.sendToViewer(msg.To, screenWSMsg{Type: "ice", From: client.peerID, Candidate: msg.Candidate})
			} else {
				room.sendToHost(screenWSMsg{Type: "ice", From: client.peerID, Candidate: msg.Candidate})
			}
		case "input":
			if client.isHost {
				continue
			}
			if !session.AllowRemoteControl {
				continue
			}
			room.sendToHost(screenWSMsg{Type: "input", From: client.peerID, Kind: msg.Kind, Data: msg.Data})
		case "ping":
			// 仅刷新 deadline
		}
	}
}

// -------- 工具 --------

func publicScreenSession(s *models.ScreenSession) *models.ScreenSession {
	cp := *s
	cp.Password = ""
	// HasPassword 已在 GetScreenSessionPublic 算好;SignalingWS 路径直接复用 s.HasPassword
	if !s.HasPassword && s.Password != "" {
		cp.HasPassword = true
	}
	return &cp
}

func screenSessionJoinable(s *models.ScreenSession) bool {
	if s.Status != "active" {
		return false
	}
	if s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt) {
		return false
	}
	return true
}

func buildScreenViewerURL(c *gin.Context, id string) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	// 优先用 X-Forwarded-Host:用户通过 LAN IP / 隧道 / 反向代理访问时,
	// c.Request.Host 会被中间层改写(比如 vite proxy 的 changeOrigin=true),
	// 此时 viewer URL 写成改写后的 host,用户在浏览器里就访问不到。
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host + "/screen/view/" + id
}

// askitBearerUser 从 Authorization 头取 askit token 并解析 user_id。
func askitBearerUser(c *gin.Context, db *models.DB) (string, bool) {
	token := askitBearerToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing_token"})
		return "", false
	}
	uid, err := db.GetAskitTokenUser(token, "access", time.Now().UnixMilli())
	if err != nil || uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return "", false
	}
	return uid, true
}

// askitBearerQueryUser 从 query string 取 token(浏览器 WS 客户端不能加 header)。
func askitBearerQueryUser(c *gin.Context, db *models.DB, param string) (string, bool) {
	token := strings.TrimSpace(c.Query(param))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing_token"})
		return "", false
	}
	uid, err := db.GetAskitTokenUser(token, "access", time.Now().UnixMilli())
	if err != nil || uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return "", false
	}
	return uid, true
}

// ─────────────────────────────────────────────────────────────────
// Screen Relay (binary WebSocket fallback)
//
// 当 WebRTC P2P 在用户网络下不可达(TURN 不通 / 双端都在 NAT 后),
// host 通过 binary WS 把屏幕帧(JPEG)和音频块(Opus in WebM)推到后端,
// 后端 fanout 给所有 viewer。比 TURN 简单可靠,但 server 带宽 × viewer 数。
//
// 帧格式:每条 binary WS 消息第一个字节是 type,后面是 payload。
//   type=0x01 → JPEG 视频帧
//   type=0x02 → Opus 音频块(由 MediaRecorder 产出的 webm/opus 片段)
// ─────────────────────────────────────────────────────────────────

type relayRoom struct {
	mu          sync.RWMutex
	host        *screenClient
	viewers     map[string]*screenClient
	latestFrame []byte
	latestAt    time.Time
	lastUsed    time.Time
	// hostSendMu 串行化所有写入 host.conn 的操作(host.send chan 写协程是 binary 帧,
	// viewer→host 控制消息是 TextMessage 直接写 conn,两个写者要互斥)。
	hostSendMu sync.Mutex
}

func newRelayRoom() *relayRoom {
	return &relayRoom{
		viewers:  make(map[string]*screenClient),
		lastUsed: time.Now(),
	}
}

// writeToHost 把控制消息直接以 TextMessage 写到 host.conn。
// 与 host.send chan 的 binary 写协程并发写同一 conn,需持 hostSendMu 互斥。
func (r *relayRoom) writeToHost(textPayload []byte) bool {
	r.mu.RLock()
	host := r.host
	r.mu.RUnlock()
	if host == nil {
		return false
	}
	r.hostSendMu.Lock()
	defer r.hostSendMu.Unlock()
	if err := host.conn.WriteMessage(websocket.TextMessage, textPayload); err != nil {
		return false
	}
	return true
}

func (r *relayRoom) setHost(c *screenClient) {
	r.mu.Lock()
	r.host = c
	r.lastUsed = time.Now()
	r.mu.Unlock()
}

func (r *relayRoom) removeHost() {
	r.mu.Lock()
	r.host = nil
	r.lastUsed = time.Now()
	r.mu.Unlock()
}

func (r *relayRoom) addViewer(c *screenClient) {
	r.mu.Lock()
	r.viewers[c.peerID] = c
	r.lastUsed = time.Now()
	r.mu.Unlock()
}

func (r *relayRoom) removeViewer(peerID string) {
	r.mu.Lock()
	delete(r.viewers, peerID)
	r.lastUsed = time.Now()
	r.mu.Unlock()
}

func (r *relayRoom) isEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.host == nil && len(r.viewers) == 0
}

// storeFrame 拷贝一份存为 latest:gorilla/websocket 的 ReadMessage 返回的 slice
// 每次都是新分配,理论上原地复用不会发生,但复制一次更稳。
func (r *relayRoom) storeFrame(p []byte) {
	r.mu.Lock()
	r.latestFrame = make([]byte, len(p))
	copy(r.latestFrame, p)
	r.latestAt = time.Now()
	r.lastUsed = time.Now()
	r.mu.Unlock()
}

func (r *relayRoom) snapshotLatest() ([]byte, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.latestFrame) == 0 {
		return nil, false
	}
	p := make([]byte, len(r.latestFrame))
	copy(p, r.latestFrame)
	return p, true
}

// broadcastBinary 不阻塞:每个 viewer 有独立 send chan(64 buffer),满了就丢。
// 屏幕帧是"丢旧保新"语义,丢一两帧不影响观看。
func (r *relayRoom) broadcastBinary(p []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.viewers {
		select {
		case c.send <- p:
		default:
		}
	}
}

// ScreenRelayHostWS GET /api/screen/sessions/:id/relay-host — host 端 binary 上传。
//
// 鉴权:?token=<askit_access> 必须属于该 session 的 host_user_id(同 ScreenSignalingWS 的 host 鉴权)。
// 协议:每条 binary 消息视为一帧(1B type + payload),广播给所有 viewer,同时保留 latest
// 供后到的 viewer 立即拿到最近一帧。
// 节流:每 conn 30 msg/s 滑动窗口,超过的帧直接丢(防恶意/异常 client 打爆 server 带宽)。
func (h *ScreenHandler) ScreenRelayHostWS(c *gin.Context) {
	id := c.Param("id")
	session, err := h.db.GetScreenSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
		return
	}
	if !screenSessionJoinable(session) {
		c.JSON(http.StatusGone, gin.H{"error": "session_unavailable"})
		return
	}
	userID, ok := askitBearerQueryUser(c, h.db, "token")
	if !ok {
		return
	}
	if userID != session.HostUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not_host"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	roomVal, _ := h.relayRooms.LoadOrStore(id, newRelayRoom())
	room := roomVal.(*relayRoom)

	client := &screenClient{
		conn:   conn,
		send:   make(chan []byte, 64),
		peerID: "host",
		isHost: true,
	}
	room.setHost(client)

	// 写协程:host 端不主动接收 server 数据,但用同一套 client.send 结构,
	// 方便清理时统一 close。host 的 send chan 永远空,这里就是 idle。
	// 与 writeToHost 共用 hostSendMu 互斥写 conn。
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC screen relay host ws write: %v", r)
			}
		}()
		defer conn.Close()
		for data := range client.send {
			room.hostSendMu.Lock()
			err := conn.WriteMessage(websocket.BinaryMessage, data)
			room.hostSendMu.Unlock()
			if err != nil {
				return
			}
		}
	}()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	// Pong 处理器:浏览器自动响应 server 发的 PingMessage,但 gorilla/websocket 把
	// Pong 帧在协议层吞掉,ReadMessage 看不到 → SetReadDeadline 不会自动重置。
	// 没有这个 handler,host 哪怕在持续推帧(reader 自然刷新),暂停推帧(例
	// 如屏幕静止时 host 端 rVFC 仍按 15FPS 推)、任何 60s 内没新帧的窗口都会让
	// server 误判对端死了,Close → viewer 看到 1006。
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	pingTicker := time.NewTicker(30 * time.Second)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC screen relay host ping: %v", r)
			}
		}()
		defer pingTicker.Stop()
		for range pingTicker.C {
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}()

	defer func() {
		pingTicker.Stop()
		_ = conn.Close()
		close(client.send)
		room.removeHost()
		if room.isEmpty() {
			go func() {
				time.Sleep(screenRoomIdleTTL)
				if room.isEmpty() {
					h.relayRooms.Delete(id)
				}
			}()
		}
	}()

	conn.SetReadLimit(2 << 20) // 2MB,允许单个 JPEG 帧(1280x720@0.7 ≈ 100-200KB)+ header
	// 滑动窗口节流:30 msg/s。host 实际 10 FPS 远低于此,这里主要是防失控。
	winStart := time.Now()
	winCount := 0
	const maxPerSecond = 30
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if len(raw) == 0 {
			continue
		}
		// host 上行只接 binary 帧。控制消息只走 viewer→host,host 不接收控制消息
		// (它自己产生 synthetic event)。
		now := time.Now()
		if now.Sub(winStart) >= time.Second {
			winStart = now
			winCount = 0
		}
		winCount++
		if winCount > maxPerSecond {
			continue // 丢帧,不打 warning(正常情况不会触发)
		}
		room.storeFrame(raw)
		room.broadcastBinary(raw)
	}
}

// ScreenRelayViewerWS GET /api/screen/sessions/:id/relay-viewer — viewer 端 binary 订阅。
//
// 鉴权:无密码 session 任意连接;有密码 session 必须传 ?password=<plaintext>(同 ScreenSignalingWS)。
// 协议:连上后立即收到 host 的 latest 帧(如有),之后持续收到 host 的新帧直到任一端断开。
// 客户端不应发送 binary 数据(setReadLimit=256,只够 pong 之类的小响应)。
func (h *ScreenHandler) ScreenRelayViewerWS(c *gin.Context) {
	id := c.Param("id")
	session, err := h.db.GetScreenSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
		return
	}
	if !screenSessionJoinable(session) {
		c.JSON(http.StatusGone, gin.H{"error": "session_unavailable"})
		return
	}
	if session.Password != "" {
		provided := sha256.Sum256([]byte(c.Query("password")))
		if hex.EncodeToString(provided[:]) != session.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "bad_password"})
			return
		}
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	roomVal, _ := h.relayRooms.LoadOrStore(id, newRelayRoom())
	room := roomVal.(*relayRoom)

	peerID := randomPeerID()
	client := &screenClient{
		conn:   conn,
		send:   make(chan []byte, 64),
		peerID: peerID,
		isHost: false,
	}
	room.addViewer(client)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC screen relay viewer ws write: %v", r)
			}
		}()
		defer conn.Close()
		frames := 0
		for data := range client.send {
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				log.Printf("relay viewer write err peer=%s frames=%d: %v", peerID, frames, err)
				return
			}
			frames++
			if frames == 1 || frames%100 == 0 {
				log.Printf("relay viewer write peer=%s frame=%d size=%d", peerID, frames, len(data))
			}
		}
		log.Printf("relay viewer write done peer=%s frames=%d", peerID, frames)
	}()

	// late joiner:立即发 latest 帧,viewer 不用等下一个 host 帧
	if latest, ok := room.snapshotLatest(); ok {
		select {
		case client.send <- latest:
		default:
		}
	}

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	// viewer 不发业务数据(viewer→host 的 control 走 TextMessage,但也只是偶尔一两条),
	// 主要靠 server 发的 Ping → browser 自动 Pong 来保活。gorilla/websocket 把 Pong
	// 在协议层消费,ReadMessage 看不到,SetReadDeadline 不会自动重置 → 没有
	// SetPongHandler 的话 60s 到期 server 主动 close,browser 看到 1006 abnormal closure
	// ("自己断开了")。
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	pingTicker := time.NewTicker(30 * time.Second)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC screen relay viewer ping: %v", r)
			}
		}()
		defer pingTicker.Stop()
		for range pingTicker.C {
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}()

	defer func() {
		pingTicker.Stop()
		_ = conn.Close()
		close(client.send)
		room.removeViewer(peerID)
		if room.isEmpty() {
			go func() {
				time.Sleep(screenRoomIdleTTL)
				if room.isEmpty() {
					h.relayRooms.Delete(id)
				}
			}()
		}
	}()

	// viewer 不发送业务数据,只接收(以及 ping/pong 控制帧)。256 字节够任何控制帧。
	conn.SetReadLimit(2048) // 控制消息 JSON 单条 < 1KB,2048 足够;binary 仍只接收 host 上行
	for {
		mt, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		// 只接 TextMessage 做 control fanout(viewer→host)。
		// binary 在 viewer 端不被使用,直接丢弃防意外。
		if mt != websocket.TextMessage {
			continue
		}
		if len(raw) == 0 {
			continue
		}
		room.writeToHost(raw)
	}
}
