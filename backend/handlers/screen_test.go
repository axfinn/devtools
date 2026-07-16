package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestScreenSession_CreateAndGet(t *testing.T) {
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()

	// 直接构造 host_user_id(避开 askit 注册流程)
	const hostID = "host-aaaa"

	exp := time.Now().Add(1 * time.Hour)
	s, err := db.CreateScreenSession(hostID, "demo", "hashed-pw", true, &exp)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.ID == "" || s.HostUserID != hostID {
		t.Fatalf("unexpected session: %+v", s)
	}
	if !s.AllowRemoteControl || s.Status != "active" {
		t.Fatalf("flags wrong: %+v", s)
	}
	if s.ExpiresAt == nil || s.ExpiresAt.Before(time.Now()) {
		t.Fatalf("ExpiresAt not set or in past: %v", s.ExpiresAt)
	}

	got, err := db.GetScreenSession(s.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Password != "hashed-pw" {
		t.Fatalf("Password hash lost: %q", got.Password)
	}

	pub, err := db.GetScreenSessionPublic(s.ID)
	if err != nil {
		t.Fatalf("GetPublic: %v", err)
	}
	if pub.Password != "" {
		t.Fatalf("Password hash leaked in public view: %q", pub.Password)
	}
}

func TestScreenSession_StopOnlyByHost(t *testing.T) {
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()

	s, err := db.CreateScreenSession("host-aaaa", "", "", false, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// 错误 host
	if err := db.StopScreenSession(s.ID, "host-bbbb"); err != models.ErrScreenSessionNotStoppable {
		t.Fatalf("expected ErrScreenSessionNotStoppable, got %v", err)
	}
	// 正确 host
	if err := db.StopScreenSession(s.ID, "host-aaaa"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	// 重复 stop
	if err := db.StopScreenSession(s.ID, "host-aaaa"); err != models.ErrScreenSessionNotStoppable {
		t.Fatalf("expected ErrScreenSessionNotStoppable on re-stop, got %v", err)
	}
}

func TestScreenSession_CleanExpired(t *testing.T) {
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()

	// 已过期
	past := time.Now().Add(-1 * time.Minute)
	if _, err := db.CreateScreenSession("h", "", "", false, &past); err != nil {
		t.Fatalf("Create past: %v", err)
	}
	// 永不过期
	if _, err := db.CreateScreenSession("h", "", "", false, nil); err != nil {
		t.Fatalf("Create nil-exp: %v", err)
	}
	// 未来过期
	future := time.Now().Add(10 * time.Minute)
	if _, err := db.CreateScreenSession("h", "", "", false, &future); err != nil {
		t.Fatalf("Create future: %v", err)
	}

	n, err := db.CleanExpiredScreenSessions()
	if err != nil {
		t.Fatalf("CleanExpired: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 expired, got %d", n)
	}

	// ListActiveScreenSessionsByHost 应该只返回未来过期 + 无限期的 2 条
	list, err := db.ListActiveScreenSessionsByHost("h")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 active, got %d", len(list))
	}
}

// ── HTTP handler 集成测试 ─────────────────────────────────────

func newScreenTestRouter(h *ScreenHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	screen := r.Group("/api/screen")
	screen.GET("/sessions/:id/info", h.ScreenInfo)
	screen.POST("/sessions/:id/check-password", h.ScreenCheckPassword)
	return r
}

func TestScreen_InfoRequiresActive(t *testing.T) {
	db, _ := models.NewDB(":memory:")
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()
	h := NewScreenHandler(db, config.Config{})
	r := newScreenTestRouter(h)

	// 不存在
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/screen/sessions/deadbeef/info", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}

	// active
	s, _ := db.CreateScreenSession("host", "title", "", false, nil)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/screen/sessions/"+s.ID+"/info", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Session struct {
			ID            string `json:"id"`
			HostUserID    string `json:"host_user_id"`
			Title         string `json:"title"`
			Status        string `json:"status"`
			AllowRemote   bool   `json:"allow_remote_control"`
		} `json:"session"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Session.ID != s.ID || resp.Session.HostUserID != "host" || resp.Session.Title != "title" {
		t.Fatalf("unexpected public view: %+v", resp)
	}

	// stopped
	_ = db.StopScreenSession(s.ID, "host")
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/screen/sessions/"+s.ID+"/info", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusGone {
		t.Fatalf("expected 410 for stopped, got %d", w.Code)
	}
}

// TestScreen_InfoHasPasswordVisibility 验证公开 /info 返回 has_password 字段,
// 前端用这个字段决定是否走密码框。原 bug:password 字段被置空,前端无法判断。
func TestScreen_InfoHasPasswordVisibility(t *testing.T) {
	db, _ := models.NewDB(":memory:")
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()
	h := NewScreenHandler(db, config.Config{})
	r := newScreenTestRouter(h)

	// 无密码 session
	noPw, _ := db.CreateScreenSession("host", "no-pw", "", false, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/screen/sessions/"+noPw.ID+"/info", nil)
	r.ServeHTTP(w, req)
	var resp struct {
		Session struct {
			HasPassword bool `json:"has_password"`
			Password    string `json:"password"` // 公开视图必须为空
		} `json:"session"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Session.HasPassword {
		t.Fatalf("no-password session should have has_password=false")
	}
	if resp.Session.Password != "" {
		t.Fatalf("password hash leaked: %q", resp.Session.Password)
	}

	// 有密码 session
	pwHash := "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8" // "password" 的 sha256
	withPw, _ := db.CreateScreenSession("host", "with-pw", pwHash, false, nil)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/screen/sessions/"+withPw.ID+"/info", nil)
	r.ServeHTTP(w, req)
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Session.HasPassword {
		t.Fatalf("password-protected session should have has_password=true")
	}
	if resp.Session.Password != "" {
		t.Fatalf("password hash leaked: %q", resp.Session.Password)
	}
}

func TestScreen_CheckPassword(t *testing.T) {
	db, _ := models.NewDB(":memory:")
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()
	h := NewScreenHandler(db, config.Config{})
	r := newScreenTestRouter(h)

	s, _ := db.CreateScreenSession("host", "", "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8", false, nil)

	// 错密码
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/screen/sessions/"+s.ID+"/check-password",
		bytes.NewReader([]byte(`{"password":"wrong"}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"ok":false`)) {
		t.Fatalf("expected ok:false, got %s", w.Body.String())
	}

	// 对密码("password" 的 SHA-256 hex)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/screen/sessions/"+s.ID+"/check-password",
		bytes.NewReader([]byte(`{"password":"password"}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if !bytes.Contains(w.Body.Bytes(), []byte(`"ok":true`)) {
		t.Fatalf("expected ok:true, got %s", w.Body.String())
	}
}

func TestScreen_TurnCredentials_NoConfig(t *testing.T) {
	db, _ := models.NewDB(":memory:")
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()
	h := NewScreenHandler(db, config.Config{}) // TURN 未配置
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/screen/turn-credentials", h.ScreenTurnCredentials)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/screen/turn-credentials", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestScreen_TurnCredentials_Configured(t *testing.T) {
	db, _ := models.NewDB(":memory:")
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()
	h := NewScreenHandler(db, config.Config{
		TURN: config.TURNConfig{
			Host:   "turn.example.com",
			Port:   3478,
			Secret: "shhh",
			TTL:    600,
		},
	})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/screen/turn-credentials", h.ScreenTurnCredentials)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/screen/turn-credentials", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["turn"] != "turn:turn.example.com:3478" {
		t.Fatalf("turn url wrong: %v", resp["turn"])
	}
	if _, ok := resp["credential"]; !ok {
		t.Fatalf("credential missing: %v", resp)
	}
}

// TestScreenSignalingRelay 端到端验证 WS 中转:
//   - host 创建会话、连 WS、收到 joined{viewer_list=[]}
//   - viewer 连 WS、host 收到 viewer_joined
//   - viewer 发 offer → host 收到(原样转发)
//   - host 发 answer{to=viewer} → viewer 收到
//   - host 发 ice{to=viewer} → viewer 收到
//   - viewer 发 ice → host 收到
//
// 这是回归测试:之前 host 还没建 peer 时 viewer 的 offer 会被前端丢,
// 现在前端会缓冲并回放,这里验证服务端从未丢过任何 offer/answer/ice。
func TestScreenSignalingRelay(t *testing.T) {
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()

	const hostID = "host-relay"
	exp := time.Now().Add(1 * time.Hour)
	s, err := db.CreateScreenSession(hostID, "relay", "", false, &exp)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// 直接发 askit token,跳过注册流程
	const hostToken = "tok-host-relay"
	if err := db.CreateAskitToken(hostToken, hostID, "access", time.Now().Add(time.Hour).UnixMilli(), time.Now().UnixMilli()); err != nil {
		t.Fatalf("CreateAskitToken: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewScreenHandler(db, config.Config{})
	r.GET("/api/screen/sessions/:id/ws", h.ScreenSignalingWS)

	// httptest.NewServer 拿到真实的 ws:// URL(Upgrade 需要 http.Server)
	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1) + "/api/screen/sessions/" + s.ID + "/ws"

	// ── HOST 连 WS ──
	hostConn, _, err := websocket.DefaultDialer.Dial(wsURL+"?role=host&token="+hostToken, nil)
	if err != nil {
		t.Fatalf("host dial: %v", err)
	}
	defer hostConn.Close()

	// host 应立即收到 joined
	var hostJoined screenWSMsg
	if err := hostConn.ReadJSON(&hostJoined); err != nil {
		t.Fatalf("host read joined: %v", err)
	}
	if hostJoined.Type != "joined" {
		t.Fatalf("host expected joined, got %q", hostJoined.Type)
	}
	if hostJoined.From == "" {
		t.Fatalf("host joined missing from")
	}

	// ── VIEWER 连 WS ──
	viewerConn, _, err := websocket.DefaultDialer.Dial(wsURL+"?role=viewer&nickname=alice", nil)
	if err != nil {
		t.Fatalf("viewer dial: %v", err)
	}
	defer viewerConn.Close()

	// viewer 收到自己的 joined
	var viewerJoined screenWSMsg
	if err := viewerConn.ReadJSON(&viewerJoined); err != nil {
		t.Fatalf("viewer read joined: %v", err)
	}
	if viewerJoined.Type != "joined" {
		t.Fatalf("viewer expected joined, got %q", viewerJoined.Type)
	}
	viewerPeerID := viewerJoined.From
	if viewerPeerID == "" {
		t.Fatalf("viewer joined missing from")
	}

	// host 收到 viewer_joined
	var hostVJ screenWSMsg
	if err := hostConn.ReadJSON(&hostVJ); err != nil {
		t.Fatalf("host read viewer_joined: %v", err)
	}
	if hostVJ.Type != "viewer_joined" || hostVJ.Viewer == nil {
		t.Fatalf("host expected viewer_joined, got %+v", hostVJ)
	}
	if hostVJ.Viewer.PeerID != viewerPeerID {
		t.Fatalf("viewer peer id mismatch: %s vs %s", hostVJ.Viewer.PeerID, viewerPeerID)
	}

	// ── viewer 发 offer ──
	viewerConn.WriteJSON(screenWSMsg{Type: "offer", SDP: "v=0\r\no=- 1 1\r\n"})
	var hostOffer screenWSMsg
	if err := hostConn.ReadJSON(&hostOffer); err != nil {
		t.Fatalf("host read offer: %v", err)
	}
	if hostOffer.Type != "offer" || hostOffer.SDP != "v=0\r\no=- 1 1\r\n" || hostOffer.From != viewerPeerID {
		t.Fatalf("host offer wrong: %+v", hostOffer)
	}

	// ── host 发 answer (with to=viewerPeerID) ──
	hostConn.WriteJSON(screenWSMsg{Type: "answer", To: viewerPeerID, SDP: "v=0\r\no=- 2 2\r\n"})
	var viewerAnswer screenWSMsg
	if err := viewerConn.ReadJSON(&viewerAnswer); err != nil {
		t.Fatalf("viewer read answer: %v", err)
	}
	if viewerAnswer.Type != "answer" || viewerAnswer.From == "" {
		t.Fatalf("viewer answer wrong: %+v", viewerAnswer)
	}

	// ── 双向 ICE ──
	hostConn.WriteJSON(screenWSMsg{Type: "ice", To: viewerPeerID, Candidate: json.RawMessage(`{"candidate":"host-cand"}`)})
	var viewerICE screenWSMsg
	if err := viewerConn.ReadJSON(&viewerICE); err != nil {
		t.Fatalf("viewer read ice: %v", err)
	}
	if viewerICE.Type != "ice" || string(viewerICE.Candidate) != `{"candidate":"host-cand"}` {
		t.Fatalf("viewer ice wrong: %+v", viewerICE)
	}

	viewerConn.WriteJSON(screenWSMsg{Type: "ice", Candidate: json.RawMessage(`{"candidate":"viewer-cand"}`)})
	var hostICE screenWSMsg
	if err := hostConn.ReadJSON(&hostICE); err != nil {
		t.Fatalf("host read ice: %v", err)
	}
	if hostICE.Type != "ice" || string(hostICE.Candidate) != `{"candidate":"viewer-cand"}` || hostICE.From != viewerPeerID {
		t.Fatalf("host ice wrong: %+v", hostICE)
	}

	// ── viewer 关闭,host 应收到 viewer_left ──
	viewerConn.Close()
	var hostLeft screenWSMsg
	if err := hostConn.ReadJSON(&hostLeft); err != nil {
		t.Fatalf("host read viewer_left: %v", err)
	}
	if hostLeft.Type != "viewer_left" || hostLeft.From != viewerPeerID {
		t.Fatalf("host viewer_left wrong: %+v", hostLeft)
	}
}

// TestScreenRelayFanout 验证 binary WS relay 的核心契约:
//   - host 上传 → 所有 viewer 收到原样 bytes
//   - late joiner:连上后立即收到 host 的 latest 帧,不用等下一帧
//   - 多 viewer fanout 互不干扰
//   - 鉴权:错误 host token 被拒;有密码 session 的 viewer 必须带正确密码
func TestScreenRelayFanout(t *testing.T) {
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()

	const hostID = "host-relay-fanout"
	exp := time.Now().Add(1 * time.Hour)
	s, err := db.CreateScreenSession(hostID, "relay-fanout", "", false, &exp)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	const hostToken = "tok-relay-fanout"
	if err := db.CreateAskitToken(hostToken, hostID, "access", time.Now().Add(time.Hour).UnixMilli(), time.Now().UnixMilli()); err != nil {
		t.Fatalf("CreateAskitToken: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewScreenHandler(db, config.Config{})
	r.GET("/api/screen/sessions/:id/relay-host", h.ScreenRelayHostWS)
	r.GET("/api/screen/sessions/:id/relay-viewer", h.ScreenRelayViewerWS)

	srv := httptest.NewServer(r)
	defer srv.Close()

	baseWS := strings.Replace(srv.URL, "http://", "ws://", 1) + "/api/screen/sessions/" + s.ID

	// ── HOST 上传端 ──
	hostConn, _, err := websocket.DefaultDialer.Dial(baseWS+"/relay-host?token="+hostToken, nil)
	if err != nil {
		t.Fatalf("host dial: %v", err)
	}
	defer hostConn.Close()

	// ── VIEWER 1(无密码 session,直接连) ──
	v1Conn, _, err := websocket.DefaultDialer.Dial(baseWS+"/relay-viewer", nil)
	if err != nil {
		t.Fatalf("v1 dial: %v", err)
	}
	defer v1Conn.Close()

	// ── VIEWER 2(也连上,测试 fanout 到多个 viewer) ──
	v2Conn, _, err := websocket.DefaultDialer.Dial(baseWS+"/relay-viewer", nil)
	if err != nil {
		t.Fatalf("v2 dial: %v", err)
	}
	defer v2Conn.Close()

	// ── HOST 发一帧:0x01 type + JPEG-like payload ──
	frame1 := []byte{0x01, 0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	if err := hostConn.WriteMessage(websocket.BinaryMessage, frame1); err != nil {
		t.Fatalf("host write frame1: %v", err)
	}

	// 两个 viewer 都应收到原样 frame1
	for name, conn := range map[string]*websocket.Conn{"v1": v1Conn, "v2": v2Conn} {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		mt, got, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("%s read frame1: %v", name, err)
		}
		if mt != websocket.BinaryMessage {
			t.Fatalf("%s frame1 mt=%d want BinaryMessage", name, mt)
		}
		if !bytes.Equal(got, frame1) {
			t.Fatalf("%s frame1 mismatch: got %v want %v", name, got, frame1)
		}
	}

	// ── 鉴权:错误 host token 应被拒(invalid_token → 401,not_host → 403) ──
	// 1. 完全无效的 token → 401 invalid_token
	_, resp, err := websocket.DefaultDialer.Dial(baseWS+"/relay-host?token=wrong-token", nil)
	if err == nil {
		t.Fatalf("invalid host token should be rejected, but dial succeeded")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid host token expected 401, got %v", resp)
	}

	// 2. 有效 token 但属于别的用户 → 403 not_host
	const intruderID = "host-relay-fanout-intruder"
	const intruderToken = "tok-relay-intruder"
	if err := db.CreateAskitToken(intruderToken, intruderID, "access", time.Now().Add(time.Hour).UnixMilli(), time.Now().UnixMilli()); err != nil {
		t.Fatalf("CreateAskitToken intruder: %v", err)
	}
	_, resp, err = websocket.DefaultDialer.Dial(baseWS+"/relay-host?token="+intruderToken, nil)
	if err == nil {
		t.Fatalf("intruder host token should be rejected, but dial succeeded")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("intruder host token expected 403, got %v", resp)
	}

	// ── LATEST FRAME:host 发第二帧,然后让 viewer3 后加入 ──
	frame2 := []byte{0x01, 0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x20, 'f', 'r', 'a', 'm', 'e', '2'}
	if err := hostConn.WriteMessage(websocket.BinaryMessage, frame2); err != nil {
		t.Fatalf("host write frame2: %v", err)
	}
	// 让 v1/v2 先把 frame2 收掉,避免污染下面"只发 1 帧"的检查
	for _, conn := range []*websocket.Conn{v1Conn, v2Conn} {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Fatalf("drain frame2: %v", err)
		}
	}

	v3Conn, _, err := websocket.DefaultDialer.Dial(baseWS+"/relay-viewer", nil)
	if err != nil {
		t.Fatalf("v3 (late joiner) dial: %v", err)
	}
	defer v3Conn.Close()

	// late joiner 连上立即收到 host 的 latest(frame2),不用等下一帧
	_ = v3Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	mt, got, err := v3Conn.ReadMessage()
	if err != nil {
		t.Fatalf("v3 read latest: %v", err)
	}
	if mt != websocket.BinaryMessage {
		t.Fatalf("v3 latest mt=%d want BinaryMessage", mt)
	}
	if !bytes.Equal(got, frame2) {
		t.Fatalf("v3 latest mismatch: got %v want %v", got, frame2)
	}

	// ── HOST 关闭:host 端 close 后,v3 之后的 ReadMessage 应该返回 EOF ──
	hostConn.Close()
	_ = v3Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := v3Conn.ReadMessage(); err == nil {
		t.Fatalf("v3 should have observed host disconnect, but read succeeded")
	}
}

// TestScreenRelayViewerAuth 验证 viewer 端的密码鉴权(无密码 session 任意连接,
// 有密码 session 必须带正确密码,错密码 401)。
func TestScreenRelayViewerAuth(t *testing.T) {
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()

	const hostID = "host-relay-auth"
	// "secret" 的 sha256 hex
	const pwHash = "2bb80d537b1da3e38bd30361aa855686bde0eacd7162fef6a25fe97bf527a25b"
	exp := time.Now().Add(1 * time.Hour)
	s, err := db.CreateScreenSession(hostID, "relay-auth", pwHash, false, &exp)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewScreenHandler(db, config.Config{})
	r.GET("/api/screen/sessions/:id/relay-viewer", h.ScreenRelayViewerWS)

	srv := httptest.NewServer(r)
	defer srv.Close()

	baseWS := strings.Replace(srv.URL, "http://", "ws://", 1) + "/api/screen/sessions/" + s.ID + "/relay-viewer"

	// 错密码 → 401
	_, resp, err := websocket.DefaultDialer.Dial(baseWS+"?password=wrong", nil)
	if err == nil {
		t.Fatalf("wrong password should be rejected")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password expected 401, got %v", resp)
	}

	// 对密码 → upgrade 成功
	conn, resp, err := websocket.DefaultDialer.Dial(baseWS+"?password=secret", nil)
	if err != nil {
		t.Fatalf("correct password dial: %v (resp=%v)", err, resp)
	}
	defer conn.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("correct password expected 101, got %d", resp.StatusCode)
	}
}

// TestScreenRelayViewerKeepalive 验证 viewer WS 在 60s+ idle 之后仍能持续
// 收帧,不会被 server 60s read deadline 误判死亡而 close 1006。
//
// 背景:server 每 30s 发一个 PingMessage,browser 协议层自动回 Pong。但
// gorilla/websocket 把 Pong 帧在 ReadMessage 路径之前消费,SetReadDeadline
// 不会自动重置 → 没 SetPongHandler 的话,viewer 这条不主动推业务数据
// 的 WS 在 60s 后会读到 timeout,browser 看到 1006 ("自己断开了")。
//
// 测试逻辑:连接后 viewer 不发任何业务数据,等过 75s(>60s deadline,>1 个
// 30s ping 周期),host 写一帧,viewer 应能读到。负向断言:把
// SetPongHandler 改空函数后这个测试在 75s 之后应失败。
func TestScreenRelayViewerKeepalive(t *testing.T) {
	if testing.Short() {
		t.Skip("keepalive test takes ~75s, skipped in -short mode")
	}
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	defer db.Close()

	const hostID = "host-relay-keepalive"
	exp := time.Now().Add(1 * time.Hour)
	s, err := db.CreateScreenSession(hostID, "relay-keepalive", "", false, &exp)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	const hostToken = "tok-relay-keepalive"
	if err := db.CreateAskitToken(hostToken, hostID, "access", time.Now().Add(time.Hour).UnixMilli(), time.Now().UnixMilli()); err != nil {
		t.Fatalf("CreateAskitToken: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewScreenHandler(db, config.Config{})
	r.GET("/api/screen/sessions/:id/relay-host", h.ScreenRelayHostWS)
	r.GET("/api/screen/sessions/:id/relay-viewer", h.ScreenRelayViewerWS)

	srv := httptest.NewServer(r)
	defer srv.Close()

	baseWS := strings.Replace(srv.URL, "http://", "ws://", 1) + "/api/screen/sessions/" + s.ID

	hostConn, _, err := websocket.DefaultDialer.Dial(baseWS+"/relay-host?token="+hostToken, nil)
	if err != nil {
		t.Fatalf("host dial: %v", err)
	}
	defer hostConn.Close()
	// host conn 也需要 drain:server 60s read deadline 必须靠 client 回 Pong 才能重置,
	// 否则测试里 host 这条 conn 自己会在 60s 被 server 踢掉,无关本测试要验证的 viewer。
	go func() {
		for {
			if _, _, err := hostConn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	viewerConn, _, err := websocket.DefaultDialer.Dial(baseWS+"/relay-viewer", nil)
	if err != nil {
		t.Fatalf("viewer dial: %v", err)
	}
	defer viewerConn.Close()

	// 重要:gorilla/websocket client 必须主动读连接,server 发来的 Ping 才会被
	// 协议层处理并自动回 Pong("The application must read the connection to process
	// ping messages")。浏览器天然有读循环所以没这个问题,但 Go 测试 client 必须
	// 自己起一个 goroutine 跑 ReadMessage 让控制帧被处理。
	//
	// 收消息的 chan:测试主流程用 <-recvCh 拿,主流程和 drain goroutine 不会同时
	// 调 ReadMessage(NextReader 限制一个 reader),drain 收到所有数据帧(忽略控制
	// 帧自动回 Pong)。
	recvCh := make(chan []byte, 16)
	go func() {
		for {
			mt, data, err := viewerConn.ReadMessage()
			if err != nil {
				close(recvCh)
				return
			}
			if mt == websocket.BinaryMessage || mt == websocket.TextMessage {
				recvCh <- data
			}
		}
	}()

	// 等 75s:期间 viewer 端不发任何业务数据。
	// 没有 SetPongHandler 重置 deadline,server 60s 时 ReadMessage 返回
	// timeout,browser viewer 端 WS 收到 close → 之后 host 写一帧,viewer
	// 读不到(error)。有 SetPongHandler,server 30s/60s 两次 ping 都会触发
	// handler 重置 deadline,viewer 一直活。
	t.Logf("waiting 75s for viewer to survive 60s read deadline via pong handler...")
	time.Sleep(75 * time.Second)

	// 75s 后 host 仍能写一帧 → viewer 应能读到 → 证明 server 没把 viewer 踢了
	frame := []byte{0x01, 0xAA, 0xBB, 0xCC}
	if err := hostConn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		t.Fatalf("host write after 75s: %v", err)
	}
	select {
	case got, ok := <-recvCh:
		if !ok {
			t.Fatalf("viewer conn closed before receiving frame (server probably closed at 60s deadline — keepalive broken)")
		}
		if !bytes.Equal(got, frame) {
			t.Fatalf("viewer got unexpected payload: %v want %v", got, frame)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("viewer did not receive frame within 3s — server connection broken")
	}
}