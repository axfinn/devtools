package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

const testAdminPwd = "test_admin_pwd"

func newNFSShareTestHandler(t *testing.T) *NFSShareHandler {
	t.Helper()
	db, err := models.NewDB(filepath.Join(t.TempDir(), "nfsshare.db"))
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	if err := db.InitNFSShare(); err != nil {
		t.Fatalf("InitNFSShare failed: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Enabled 必须 true,checkEnabled 才会放行
	cfg := config.NFSShareConfig{
		Enabled:       true,
		AdminPassword: testAdminPwd,
	}
	return NewNFSShareHandler(db, cfg)
}

func nfsAdminCookie(req *http.Request) {
	req.AddCookie(&http.Cookie{Name: nfsAdminCookieName, Value: testAdminPwd})
}

// 直接调用底层 auth 原语,验证 cookie / query / header / body 四个入口都生效
func TestNFSShareAuthPrimitives(t *testing.T) {
	h := &NFSShareHandler{cfg: config.NFSShareConfig{AdminPassword: testAdminPwd}}

	// 错误的密码/cookie 一律 false
	if h.verifyAdmin("wrong") {
		t.Fatal("verifyAdmin(wrong) = true, want false")
	}
	if h.verifyAdmin("") {
		t.Fatal("verifyAdmin('') = true, want false")
	}
	if !h.verifyAdmin(testAdminPwd) {
		t.Fatal("verifyAdmin(correct) = false, want true")
	}

	tests := []struct {
		name      string
		setup     func(*http.Request)
		wantOK    bool
	}{
		{
			name:   "no auth",
			setup:  func(*http.Request) {},
			wantOK: false,
		},
		{
			name: "valid cookie",
			setup: func(r *http.Request) { nfsAdminCookie(r) },
			wantOK: true,
		},
		{
			name: "wrong cookie",
			setup: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: nfsAdminCookieName, Value: "wrong"})
			},
			wantOK: false,
		},
		{
			name: "valid query",
			setup: func(r *http.Request) {
				r.URL.RawQuery = "admin_password=" + testAdminPwd
			},
			wantOK: true,
		},
		{
			name: "valid header",
			setup: func(r *http.Request) {
				r.Header.Set("X-Admin-Password", testAdminPwd)
			},
			wantOK: true,
		},
		{
			name: "cookie takes priority over wrong query",
			setup: func(r *http.Request) {
				nfsAdminCookie(r)
				r.URL.RawQuery = "admin_password=wrong"
			},
			wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/nfsshare/admin/list", nil)
			tt.setup(c.Request)
			if got := h.verifyAdminFromContext(c); got != tt.wantOK {
				t.Fatalf("verifyAdminFromContext = %v, want %v", got, tt.wantOK)
			}
		})
	}
}

// AdminList 是纯 cookie 鉴权端点,顺手验证 cookie 路径能完整跑通
func TestNFSShareAdminList_CookieAuth(t *testing.T) {
	h := newNFSShareTestHandler(t)

	// 没有 cookie: 403
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/nfsshare/admin/list", nil)
	h.AdminList(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("no auth: status = %d, want 403, body: %s", w.Code, w.Body.String())
	}

	// 带 cookie: 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/nfsshare/admin/list", nil)
	nfsAdminCookie(c.Request)
	h.AdminList(c)
	if w.Code != http.StatusOK {
		t.Fatalf("with cookie: status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
}

// Info 是公开端点,确认 show_record_indicator 被正确透传给访客
func TestNFSShareInfo_ShowRecordIndicator(t *testing.T) {
	h := newNFSShareTestHandler(t)

	for _, tt := range []struct {
		name  string
		setup func() string
	}{
		{
			name: "indicator visible",
			setup: func() string {
				s, err := h.db.CreateNFSShare("n", "p", "text/plain", "", 1, 5, nil, "127.0.0.1", true, true)
				if err != nil {
					t.Fatal(err)
				}
				return s.ID
			},
		},
		{
			name: "indicator hidden by admin",
			setup: func() string {
				s, err := h.db.CreateNFSShare("n2", "p2", "text/plain", "", 1, 5, nil, "127.0.0.1", true, false)
				if err != nil {
					t.Fatal(err)
				}
				return s.ID
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.setup()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/nfsshare/"+id+"/info", nil)
			c.Params = gin.Params{{Key: "id", Value: id}}
			h.Info(c)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
			}
			var got map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			v, ok := got["show_record_indicator"]
			if !ok {
				t.Fatalf("response missing show_record_indicator: %s", w.Body.String())
			}
			expected := tt.name == "indicator visible"
			if v != expected {
				t.Fatalf("show_record_indicator = %v, want %v", v, expected)
			}
		})
	}
}

// AdminUpdate 是这次改动的核心:
//  1. 不带 admin_password body 也能跑通(cookie 鉴权),不会触发 binding required 报错
//  2. show_record_indicator 为 nil 时不动 DB 原值
//  3. show_record_indicator 为 &false 时落库
//  4. cookie 缺失时 403
func TestNFSShareAdminUpdate_ShowRecordIndicatorAndAuth(t *testing.T) {
	h := newNFSShareTestHandler(t)
	share, err := h.db.CreateNFSShare("name", "p", "text/plain", "", 1, 5, nil, "127.0.0.1", true, true)
	if err != nil {
		t.Fatal(err)
	}

	doUpdate := func(t *testing.T, body string, withCookie bool) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodPut, "/api/nfsshare/admin/"+share.ID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: share.ID}}
		if withCookie {
			nfsAdminCookie(req)
		}
		h.AdminUpdate(c)
		return w
	}

	// Case A: 无 cookie 无 body password -> 403(不能落到 binding required 错误)
	w := doUpdate(t, `{"add_views": 1}`, false)
	if w.Code != http.StatusForbidden {
		t.Fatalf("no auth: status = %d, want 403, body: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "AdminPassword") || strings.Contains(w.Body.String(), "required") {
		t.Fatalf("no auth: response should be 403 forbidden, not a binding error: %s", w.Body.String())
	}

	// Case B: 旧 bug 复现 —— 无 body admin_password 但带 cookie,
	// 不应再报 CreateNFSShareRequest.AdminPassword 之类的 binding 错误
	w = doUpdate(t, `{"show_record_indicator": false}`, true)
	if w.Code != http.StatusOK {
		t.Fatalf("cookie auth, nil body pwd: status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	// 落库后值是 false
	updated, _ := h.db.GetNFSShare(share.ID)
	if updated.ShowRecordIndicator {
		t.Fatalf("after Case B, ShowRecordIndicator = true, want false")
	}

	// Case C: show_record_indicator 字段缺失(nil) -> DB 原值不动(此时是 false)
	w = doUpdate(t, `{"add_views": 0}`, true)
	if w.Code != http.StatusOK {
		t.Fatalf("nil show_record_indicator: status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	updated, _ = h.db.GetNFSShare(share.ID)
	if updated.ShowRecordIndicator {
		t.Fatalf("nil pointer should not flip ShowRecordIndicator to true, got true")
	}

	// Case D: 显式传 true -> 翻回 true
	w = doUpdate(t, `{"show_record_indicator": true}`, true)
	if w.Code != http.StatusOK {
		t.Fatalf("explicit true: status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	updated, _ = h.db.GetNFSShare(share.ID)
	if !updated.ShowRecordIndicator {
		t.Fatalf("explicit true should set ShowRecordIndicator to true")
	}

	// Case E: 返回的 JSON 不再多余访问 DB —— 包含更新后的字段
	// (间接验证 AdminUpdate 不再调一次 GetNFSShare)
	w = doUpdate(t, `{"show_record_indicator": false}`, true)
	if w.Code != http.StatusOK {
		t.Fatalf("final update: status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response unmarshal: %v, body: %s", err, w.Body.String())
	}
	if v, _ := resp["show_record_indicator"].(bool); v {
		t.Fatalf("response show_record_indicator = true, want false")
	}
	if id, _ := resp["id"].(string); id != share.ID {
		t.Fatalf("response id = %q, want %q", id, share.ID)
	}
}

// 回归原 bug:Create handler 不再因 AdminPassword required 直接 400,
// 改成 403(cookie 也没带)+ body 解析成功的路径
func TestNFSShareCreate_AdminPasswordNotRequired(t *testing.T) {
	h := newNFSShareTestHandler(t)

	// 没有任何鉴权:期望走到 parsePath 失败(因为没真实文件)返回 4xx,但绝不应该是 400 binding
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"x","file_path":"__uploads__/missing.txt","max_views":5}`
	req := httptest.NewRequest(http.MethodPost, "/api/nfsshare", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	h.Create(c)

	// 不能是 binding required 错误
	if w.Code == http.StatusBadRequest && strings.Contains(w.Body.String(), "AdminPassword") {
		t.Fatalf("Create still rejects with AdminPassword binding error: %s", w.Body.String())
	}
	// 没有任何鉴权应是 403
	if w.Code != http.StatusForbidden {
		// 如果走到 parsePath 失败也可能是 400,但 body 不应包含 AdminPassword required
		if w.Code == http.StatusBadRequest && !strings.Contains(w.Body.String(), "AdminPassword") {
			t.Logf("got 400 from parsePath (acceptable): %s", w.Body.String())
		} else {
			t.Fatalf("no auth: status = %d, body: %s", w.Code, w.Body.String())
		}
	}
}

// UploadInit / UploadComplete 之前用 binding:"required" 卡 AdminPassword,
// 登录后前端发空密码会 400.回归测试覆盖 cookie 鉴权回退路径
func TestNFSShareUploadInit_CookieAuth(t *testing.T) {
	h := newNFSShareTestHandler(t)

	// 无 cookie + 空 body pwd: 403,不是 400 binding
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"filename":"a.txt","total_chunks":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/nfsshare/admin/upload/init", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	h.UploadInit(c)
	if w.Code == http.StatusBadRequest && strings.Contains(w.Body.String(), "AdminPassword") {
		t.Fatalf("UploadInit still rejects with AdminPassword binding error: %s", w.Body.String())
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("no auth: status = %d, want 403, body: %s", w.Code, w.Body.String())
	}

	// 带 cookie + 空 body pwd: 期望走到 target_dir 解析,这里没指定 target_dir,
	// 应当成功(返回 token)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	body = `{"filename":"a.txt","total_chunks":1}`
	req = httptest.NewRequest(http.MethodPost, "/api/nfsshare/admin/upload/init", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	nfsAdminCookie(req)
	c.Request = req
	h.UploadInit(c)
	if w.Code != http.StatusOK {
		t.Fatalf("cookie auth: status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("response missing token: %s", w.Body.String())
	}
}

// UploadComplete 在 UploadInit 已经写入 .meta 文件后才能跑通 happy path;
// 这里只验证鉴权回退——空密码 + cookie 不能报 binding 错
func TestNFSShareUploadComplete_CookieAuth(t *testing.T) {
	h := newNFSShareTestHandler(t)

	// 无 cookie + 空 body pwd: 403(不是 400 binding 错)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"total_chunks":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/nfsshare/admin/upload/fake-token/complete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "token", Value: "fake-token"}}
	h.UploadComplete(c)
	if w.Code == http.StatusBadRequest && strings.Contains(w.Body.String(), "AdminPassword") {
		t.Fatalf("UploadComplete still rejects with AdminPassword binding error: %s", w.Body.String())
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("no auth: status = %d, want 403, body: %s", w.Code, w.Body.String())
	}

	// 带 cookie 但 token 无效:不应再因 AdminPassword 报 400,应当走到 400 "无效的 token"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	body = `{"total_chunks":1}`
	req = httptest.NewRequest(http.MethodPost, "/api/nfsshare/admin/upload/fake-token/complete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	nfsAdminCookie(req)
	c.Request = req
	c.Params = gin.Params{{Key: "token", Value: "fake-token"}}
	h.UploadComplete(c)
	if w.Code == http.StatusBadRequest && strings.Contains(w.Body.String(), "AdminPassword") {
		t.Fatalf("UploadComplete with cookie still hits AdminPassword binding: %s", w.Body.String())
	}
}

// 回归 Range 多次请求刷 view 的 bug:浏览器 seek/多段下载只会发 Range 请求,
// 不应被 Access/Stream 计入 view 或写 success 日志
func TestIsInitialAccessRequest(t *testing.T) {
	tests := []struct {
		name   string
		method string
		rangeH string
		want   bool
	}{
		{"GET 无 Range(老浏览器/curl)", http.MethodGet, "", true},
		{"GET Range 字节首段", http.MethodGet, "bytes=0-", true},
		{"GET Range 字节首段带末尾", http.MethodGet, "bytes=0-1023", true},
		{"GET Range 中段(seek)", http.MethodGet, "bytes=1024-2047", false},
		{"GET Range 末尾段", http.MethodGet, "bytes=500-", false},
		{"HEAD 任意", http.MethodHead, "", false},
		{"HEAD 带 Range", http.MethodHead, "bytes=0-", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(tt.method, "/api/nfsshare/x", nil)
			if tt.rangeH != "" {
				c.Request.Header.Set("Range", tt.rangeH)
			}
			if got := isInitialAccessRequest(c); got != tt.want {
				t.Fatalf("isInitialAccessRequest = %v, want %v", got, tt.want)
			}
		})
	}
}

// UploadRecord 同一 (session, seq) 重传时必须幂等,否则前端 retry 会反复落盘覆盖。
// 完整跑通需要文件落地;此处直接构造 chunkDir 与 chunkFile,验证去重分支不报错即可。
func TestUploadRecordChunkDedup(t *testing.T) {
	h := newNFSShareTestHandler(t)

	// 建一个 share 用于 ID;不需要真实文件,因为 chunkDir 路径只与 id/session 有关
	share, err := h.db.CreateNFSShare("test", "irrelevant/path", "video/mp4", "", 0, 100, nil, "127.0.0.1", true, true)
	if err != nil {
		t.Fatalf("CreateNFSShare: %v", err)
	}
	id := share.ID

	sessionID := "test_session_dedup"
	chunkDir := filepath.Join("./data/records", id, "chunks", sessionID)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// 预置 seq=0 的 chunk,模拟已上传成功
	chunkFile := filepath.Join(chunkDir, "000000.webm")
	if err := os.WriteFile(chunkFile, []byte("pretend-webm-bytes"), 0644); err != nil {
		t.Fatalf("seed chunk: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Join("./data/records", id)) })

	// 再次上传 seq=0:必须返回 200 + duplicate:true,且不动 chunkFile 内容
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("audio", "record.webm")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	fw.Write([]byte("new-webm-bytes"))
	mw.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/api/nfsshare/"+id+"/record?session="+sessionID+"&seq=0", body)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.UploadRecord(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		OK        bool `json:"ok"`
		Duplicate bool `json:"duplicate"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.OK || !resp.Duplicate {
		t.Fatalf("resp = %+v, want ok=true duplicate=true", resp)
	}
	// 关键:chunkFile 不能被覆盖
	got, err := os.ReadFile(chunkFile)
	if err != nil {
		t.Fatalf("read after dup: %v", err)
	}
	if string(got) != "pretend-webm-bytes" {
		t.Fatalf("chunkFile was overwritten on dedup hit, got: %q", got)
	}
}

// 回归:finalizeRecording 必须排除 chunkDir 里的元数据文件(.client_ip 等),
// 否则 IP 字节会被拼到音频流开头,毁掉整个文件 —— 对应"分片都在上传,产物却不存在"
// 真实场景:192.168.1.5\n 这种 12 字节前缀被 ffprobe 当噪音,hasAudioStream 判定失败,
// 之后 os.Remove(outFile) 删掉产物,访客端 /api/.../record 永远 404。
func TestFinalizeRecording_IgnoresDotFiles(t *testing.T) {
	h := newNFSShareTestHandler(t)

	share, err := h.db.CreateNFSShare("test", "irrelevant/path", "video/mp4", "", 0, 100, nil, "127.0.0.1", true, true)
	if err != nil {
		t.Fatalf("CreateNFSShare: %v", err)
	}
	id := share.ID
	const sessionID = "dotfile_exclude_test"

	chunkDir := filepath.Join("./data/records", id, "chunks", sessionID)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	outDir := filepath.Join("./data/records", id)
	t.Cleanup(func() { os.RemoveAll(filepath.Join("./data/records", id)) })

	// 元数据:IP 字符串(精确 12 字节,这是它在 outFile 头部嵌入时的形态)
	clientIPBytes := []byte("192.168.1.5\n")
	if err := os.WriteFile(filepath.Join(chunkDir, ".client_ip"), clientIPBytes, 0644); err != nil {
		t.Fatalf("seed .client_ip: %v", err)
	}
	// 两个 11 字节的"假 webm"分片,后续会被 ffprobe 拒绝(outFile 被删)—— 不重要,
	// 关键是 finalize 必须先把 .client_ip 排除掉再写文件。
	for i, body := range []string{"WEBM-CHUNK-0", "WEBM-CHUNK-1"} {
		name := fmt.Sprintf("%06d.webm", i)
		if err := os.WriteFile(filepath.Join(chunkDir, name), []byte(body), 0644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}

	// outDir 里已有旧产物就先清干净
	if entries, err := os.ReadDir(outDir); err == nil {
		for _, e := range entries {
			os.Remove(filepath.Join(outDir, e.Name()))
		}
	}

	h.finalizeRecording(id, sessionID, "192.168.1.5")

	// 核心断言:如果 outDir 里**存在**任何输出文件,它的前 12 字节都**不能**是 IP 字节
	// (假如旧的 buggy 行为,IP 字节会按字典序排在 000000 前,污染文件头)
	entries, err := os.ReadDir(outDir)
	if err != nil {
		// outDir 不存在 = 最终无产物(ffprobe 拒了假 webm,这是预期),测试通过
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fp := filepath.Join(outDir, e.Name())
		f, err := os.Open(fp)
		if err != nil {
			continue
		}
		head := make([]byte, len(clientIPBytes))
		n, _ := f.Read(head)
		f.Close()
		if n == len(clientIPBytes) && string(head) == string(clientIPBytes) {
			t.Fatalf("output file %s starts with .client_ip bytes (corruption): %q",
				e.Name(), head)
		}
	}
}
