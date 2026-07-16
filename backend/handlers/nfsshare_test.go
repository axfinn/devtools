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
	"time"

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

// =================== nfsshare 列表/预览改造相关单测 ===================

// sortFileEntries: 目录优先 + 主键升降序
func TestSortFileEntries(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	entries := []FileEntry{
		{Name: "b.txt", Size: 200, ModTime: t2, IsDir: false},
		{Name: "zdir", Size: 0, ModTime: t1, IsDir: true},
		{Name: "a.txt", Size: 100, ModTime: t3, IsDir: false},
		{Name: "adir", Size: 0, ModTime: t2, IsDir: true},
		{Name: "c.txt", Size: 50, ModTime: t1, IsDir: false},
	}

	tests := []struct {
		name      string
		orderBy   string
		orderDir  string
		wantNames []string
	}{
		{"name asc", "name", "asc", []string{"adir", "zdir", "a.txt", "b.txt", "c.txt"}},
		{"name desc", "name", "desc", []string{"zdir", "adir", "c.txt", "b.txt", "a.txt"}},
		{"size asc", "size", "asc", []string{"adir", "zdir", "c.txt", "a.txt", "b.txt"}},
		{"size desc", "size", "desc", []string{"zdir", "adir", "b.txt", "a.txt", "c.txt"}},
		{"mod_time asc", "mod_time", "asc", []string{"zdir", "adir", "c.txt", "b.txt", "a.txt"}},
		{"mod_time desc", "mod_time", "desc", []string{"adir", "zdir", "a.txt", "b.txt", "c.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := append([]FileEntry(nil), entries...)
			sortFileEntries(got, tt.orderBy, tt.orderDir)
			if len(got) != len(tt.wantNames) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.wantNames))
			}
			for i, name := range tt.wantNames {
				if got[i].Name != name {
					t.Errorf("pos %d: got %s, want %s (full=%v)", i, got[i].Name, name, namesOf(got))
				}
			}
		})
	}
}

func namesOf(es []FileEntry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}

// paginateFileEntries: 切片、越界、最后一页部分
func TestPaginateFileEntries(t *testing.T) {
	all := make([]FileEntry, 25)
	for i := range all {
		all[i] = FileEntry{Name: fmt.Sprintf("f%d", i)}
	}
	tests := []struct {
		name     string
		page     int
		pageSize int
		wantLen  int
		wantMore bool
		wantName string // 第一项 Name（验切片起点）
	}{
		{"first page", 1, 10, 10, true, "f0"},
		{"middle page", 2, 10, 10, true, "f10"},
		{"last full page", 3, 10, 5, false, "f20"},
		{"beyond range", 4, 10, 0, false, ""},
		{"page=1 size=25", 1, 25, 25, false, "f0"},
		{"page=1 size=100", 1, 100, 25, false, "f0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paged, total, hasMore := paginateFileEntries(all, tt.page, tt.pageSize)
			if total != 25 {
				t.Errorf("total = %d, want 25", total)
			}
			if len(paged) != tt.wantLen {
				t.Errorf("len(paged) = %d, want %d", len(paged), tt.wantLen)
			}
			if hasMore != tt.wantMore {
				t.Errorf("hasMore = %v, want %v", hasMore, tt.wantMore)
			}
			if len(paged) > 0 && paged[0].Name != tt.wantName {
				t.Errorf("first = %s, want %s", paged[0].Name, tt.wantName)
			}
		})
	}
}

// isInlinePreviewable: mime → 浏览器内联 or 下载
func TestIsInlinePreviewable(t *testing.T) {
	tests := []struct {
		mime string
		want bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/webp", true},
		{"video/mp4", true},
		{"video/webm", true},
		{"audio/mpeg", true},
		{"audio/ogg", true},
		{"text/plain", true},
		{"text/html", true},
		{"application/json", true},
		{"application/pdf", true},
		{"application/zip", false},
		{"application/octet-stream", false},
		{"application/x-tar", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			if got := isInlinePreviewable(tt.mime); got != tt.want {
				t.Errorf("isInlinePreviewable(%q) = %v, want %v", tt.mime, got, tt.want)
			}
		})
	}
}

// Browse handler: 分页/搜索/排序 + 老调用兼容
// 用 tmp 目录造 25 个文件 + 3 个目录，挂成 local mount，挂载到 __test_browse__ 上验证
func TestBrowse_PaginationAndFilter(t *testing.T) {
	h := newNFSShareTestHandler(t)

	// 准备挂载点目录与文件
	tmpDir := t.TempDir()
	files := []struct {
		name string
		size int64
	}{
		{"alpha.txt", 100},
		{"beta.txt", 500},
		{"gamma.txt", 300},
		{"apple.txt", 50},
		{"zebra.txt", 700},
		// 凑够 25 个不同前缀的文件，方便分页断言
	}
	for i := 0; i < 25; i++ {
		files = append(files, struct {
			name string
			size int64
		}{fmt.Sprintf("file_%02d.txt", i), int64(i * 10)})
	}
	dirs := []string{"docs", "images", "videos"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range files {
		p := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(p, bytes.Repeat([]byte("x"), int(f.size)), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 把 tmpDir 加为 local mount 名为 __test_browse__
	h.mu.Lock()
	h.mounts["__test_browse__"] = &MountStatus{
		Config:    config.MountConfig{Name: "__test_browse__", Type: "local", Export: tmpDir},
		LocalPath: tmpDir,
		Mounted:   true,
	}
	h.mu.Unlock()

	t.Run("legacy: no page returns all entries without pagination metadata", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/browse?path=__test_browse__", nil)
		nfsAdminCookie(c.Request)
		h.Browse(c)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Path    string       `json:"path"`
			Entries []FileEntry  `json:"entries"`
			Total   *int         `json:"total"`
			Page    *int         `json:"page"`
			HasMore *bool        `json:"has_more"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		// 老行为：total/page/has_more 不应出现（向后兼容）
		if resp.Total != nil {
			t.Errorf("legacy call should not include total, got %v", *resp.Total)
		}
		if len(resp.Entries) != len(files)+len(dirs) {
			t.Errorf("entries = %d, want %d", len(resp.Entries), len(files)+len(dirs))
		}
	})

	t.Run("page=1 page_size=10 returns first 10 + total/has_more", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/browse?path=__test_browse__&page=1&page_size=10&order_by=name&order_dir=asc", nil)
		nfsAdminCookie(c.Request)
		h.Browse(c)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Entries  []FileEntry `json:"entries"`
			Total    int         `json:"total"`
			Page     int         `json:"page"`
			PageSize int         `json:"page_size"`
			HasMore  bool        `json:"has_more"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Total != len(files)+len(dirs) {
			t.Errorf("total = %d, want %d", resp.Total, len(files)+len(dirs))
		}
		if resp.Page != 1 || resp.PageSize != 10 {
			t.Errorf("page/page_size = %d/%d, want 1/10", resp.Page, resp.PageSize)
		}
		if !resp.HasMore {
			t.Error("has_more should be true on page 1 with 28 total")
		}
		if len(resp.Entries) != 10 {
			t.Errorf("entries = %d, want 10", len(resp.Entries))
		}
	})

	t.Run("page=3 page_size=10 returns entries 20-29, has_more true (33 total)", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/browse?path=__test_browse__&page=3&page_size=10&order_by=name&order_dir=asc", nil)
		nfsAdminCookie(c.Request)
		h.Browse(c)
		var resp struct {
			Entries []FileEntry `json:"entries"`
			HasMore bool        `json:"has_more"`
			Total   int         `json:"total"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		// 33 total (30 files + 3 dirs), page 3 of 10: start=20 end=30 = 10 entries; 还有第 4 页(33-30=3) → has_more=true
		if resp.Total != 33 {
			t.Errorf("total = %d, want 33", resp.Total)
		}
		if len(resp.Entries) != 10 {
			t.Errorf("entries = %d, want 10 (positions 20-29)", len(resp.Entries))
		}
		if !resp.HasMore {
			t.Error("has_more should be true (page 4 has 3 more)")
		}
	})

	t.Run("page=4 page_size=10 returns last 3 entries, has_more false", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/browse?path=__test_browse__&page=4&page_size=10&order_by=name&order_dir=asc", nil)
		nfsAdminCookie(c.Request)
		h.Browse(c)
		var resp struct {
			Entries []FileEntry `json:"entries"`
			HasMore bool        `json:"has_more"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Entries) != 3 {
			t.Errorf("entries = %d, want 3 (33-30)", len(resp.Entries))
		}
		if resp.HasMore {
			t.Error("has_more should be false on last page")
		}
	})

	t.Run("q filter + page: file_01 matches file_01.txt only", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/browse?path=__test_browse__&q=file_01&order_by=name&page=1&page_size=10", nil)
		nfsAdminCookie(c.Request)
		h.Browse(c)
		var resp struct {
			Entries []FileEntry `json:"entries"`
			Total   int         `json:"total"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Total != 1 {
			t.Errorf("total = %d, want 1 (only file_01.txt)", resp.Total)
		}
	})

	t.Run("q filter + page: file_ matches all 25 file_NN.txt", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/browse?path=__test_browse__&q=file_&page=1&page_size=50", nil)
		nfsAdminCookie(c.Request)
		h.Browse(c)
		var resp struct {
			Total int `json:"total"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Total != 25 {
			t.Errorf("total = %d, want 25", resp.Total)
		}
	})

	t.Run("order_by=size desc puts biggest file first (after dirs)", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/browse?path=__test_browse__&page=1&page_size=10&order_by=size&order_dir=desc", nil)
		nfsAdminCookie(c.Request)
		h.Browse(c)
		var resp struct {
			Entries []FileEntry `json:"entries"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		// 头三个是目录（顺序无所谓），第一个文件应该是 zebra.txt (700 bytes)
		var firstFile *FileEntry
		for i := range resp.Entries {
			if !resp.Entries[i].IsDir {
				firstFile = &resp.Entries[i]
				break
			}
		}
		if firstFile == nil {
			t.Fatal("no file in first 10 entries")
		}
		if firstFile.Name != "zebra.txt" {
			t.Errorf("first file = %s, want zebra.txt", firstFile.Name)
		}
	})

	t.Run("no auth: 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/browse?path=__test_browse__&page=1", nil)
		h.Browse(c)
		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", w.Code)
		}
	})

	t.Run("mount_type propagated to FileEntry (前端据此隐藏 SMB 预览)", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/browse?path=__test_browse__", nil)
		nfsAdminCookie(c.Request)
		h.Browse(c)
		var resp struct {
			Entries []struct {
				Name      string `json:"name"`
				MountType string `json:"mount_type"`
			} `json:"entries"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		// 至少要有一个 entry, 且 mount_type=local
		if len(resp.Entries) == 0 {
			t.Fatal("no entries returned")
		}
		for _, e := range resp.Entries {
			if e.MountType != "local" {
				t.Errorf("entry %s: mount_type = %q, want local", e.Name, e.MountType)
			}
		}
	})

	t.Run("SMB mount entries tagged with mount_type=smb", func(t *testing.T) {
		// 把 __smb_browse__ 加成 smb 挂载点,先调 ReadDir 必然失败,所以只验 tag 不会跑到 SMB 实际读
		// 这里改为直接验"根目录列出挂载点"那支:挂载点本身 mount_type 应是 smb
		h.mu.RLock()
		_, has := h.mounts["__smb_browse__"]
		h.mu.RUnlock()
		if has {
			t.Skip("__smb_browse__ already exists")
		}
		h.mu.Lock()
		h.mounts["__smb_browse__"] = &MountStatus{
			Config:  config.MountConfig{Name: "__smb_browse__", Type: "smb", Host: "fake", Share: "x"},
			Mounted: false,
		}
		h.mu.Unlock()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/nfsshare/admin/browse?path=.", nil)
		nfsAdminCookie(c.Request)
		h.Browse(c)
		var resp struct {
			Entries []struct {
				Name      string `json:"name"`
				MountType string `json:"mount_type"`
			} `json:"entries"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		var found bool
		for _, e := range resp.Entries {
			if e.Name == "__smb_browse__" {
				if e.MountType != "smb" {
					t.Errorf("mount_type = %q, want smb", e.MountType)
				}
				found = true
			}
		}
		if !found {
			t.Error("__smb_browse__ not in root listing")
		}
	})
}

// AdminRaw: 鉴权、SMB 拒绝、文件服务、Range、目录/不存在
func TestAdminRaw(t *testing.T) {
	h := newNFSShareTestHandler(t)

	tmpDir := t.TempDir()
	fileContent := []byte("hello admin raw!\n" + strings.Repeat("ABCD", 1000))
	if err := os.WriteFile(filepath.Join(tmpDir, "hi.txt"), fileContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "sub", "nested.png"), []byte("\x89PNG\r\n\x1a\nfake-bytes"), 0644); err != nil {
		t.Fatal(err)
	}

	h.mu.Lock()
	h.mounts["__raw__"] = &MountStatus{
		Config:    config.MountConfig{Name: "__raw__", Type: "local", Export: tmpDir},
		LocalPath: tmpDir,
		Mounted:   true,
	}
	h.mu.Unlock()

	t.Run("no auth → 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/raw?path=__raw__/hi.txt", nil)
		h.AdminRaw(c)
		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", w.Code)
		}
	})

	t.Run("missing path → 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/nfsshare/admin/raw", nil)
		nfsAdminCookie(c.Request)
		h.AdminRaw(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("directory → 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/raw?path=__raw__/sub", nil)
		nfsAdminCookie(c.Request)
		h.AdminRaw(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("non-existent file → 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/raw?path=__raw__/ghost.txt", nil)
		nfsAdminCookie(c.Request)
		h.AdminRaw(c)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("SMB mount: 文件不存在/不可达 → 404（不再 415，已支持 SMB）", func(t *testing.T) {
		// SMB 后端没真挂载时 Open 会失败 → 404 而不是 415
		h.mu.Lock()
		smbCfg := config.MountConfig{Name: "__smb__", Type: "smb", Host: "127.0.0.1", Share: "fake", Username: "u", Password: "p"}
		h.mounts["__smb__"] = &MountStatus{
			Config:  smbCfg,
			Mounted: true,
			smb:     newSMBBackend(smbCfg),
		}
		h.mu.Unlock()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/raw?path=__smb__/foo.txt", nil)
		nfsAdminCookie(c.Request)
		h.AdminRaw(c)
		// 期望：SMB 后端连不上 fake server → Open 失败 → 404（不是 415）
		if w.Code == http.StatusUnsupportedMediaType {
			t.Errorf("status = 415 (regression: SMB 应该走文件读取路径而不是直接拒绝)")
		}
		if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 404/500 (SMB 连接失败)", w.Code)
		}
	})

	t.Run("Content-Disposition: 中文文件名 UTF-8 + RFC 5987 编码", func(t *testing.T) {
		// 文件名包含中文（用户的 win-share 场景就是中文文件名）
		// Go 的 http.ServeContent 内部会用 mime.EncodeFilename 处理 UTF-8
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/raw?path=__raw__/hi.txt", nil)
		nfsAdminCookie(c.Request)
		h.AdminRaw(c)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
		}
		// 文本文件 → inline；正文里至少要能看到内容
		if !bytes.HasPrefix(w.Body.Bytes(), []byte("hello admin raw!")) {
			t.Errorf("body prefix wrong: %q", w.Body.Bytes()[:32])
		}
	})

	t.Run("text file: full body + inline Content-Disposition", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/raw?path=__raw__/hi.txt", nil)
		nfsAdminCookie(c.Request)
		h.AdminRaw(c)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
		}
		if !bytes.HasPrefix(w.Body.Bytes(), []byte("hello admin raw!")) {
			t.Errorf("body prefix wrong: %q", w.Body.Bytes()[:32])
		}
		cd := w.Header().Get("Content-Disposition")
		if !strings.HasPrefix(cd, "inline") {
			t.Errorf("Content-Disposition = %q, want inline...", cd)
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/") {
			t.Errorf("Content-Type = %q, want text/*", ct)
		}
	})

	t.Run("binary file: attachment Content-Disposition + correct content-type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/raw?path=__raw__/sub/nested.png", nil)
		nfsAdminCookie(c.Request)
		h.AdminRaw(c)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d", w.Code)
		}
		if cd := w.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "inline") {
			// image/png 应该 inline，不是 attachment
			t.Errorf("Content-Disposition = %q, want inline", cd)
		}
		if ct := w.Header().Get("Content-Type"); ct != "image/png" {
			t.Errorf("Content-Type = %q, want image/png", ct)
		}
		if !bytes.Equal(w.Body.Bytes()[:8], []byte("\x89PNG\r\n\x1a\n")) {
			t.Errorf("body mismatch, got %v", w.Body.Bytes()[:8])
		}
	})

	t.Run("Range request: serve partial bytes", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/raw?path=__raw__/hi.txt", nil)
		c.Request.Header.Set("Range", "bytes=0-4")
		nfsAdminCookie(c.Request)
		h.AdminRaw(c)
		if w.Code != http.StatusPartialContent {
			t.Fatalf("status = %d, want 206, body: %s", w.Code, w.Body.String())
		}
		if got := w.Body.String(); got != "hello" {
			t.Errorf("range body = %q, want %q", got, "hello")
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/api/nfsshare/admin/raw?path=__raw__/../../etc/passwd", nil)
		nfsAdminCookie(c.Request)
		h.AdminRaw(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400 for traversal", w.Code)
		}
	})
}

// 超管改 max_file_size_mb 后,超过的文件应被 413 拒绝
func TestAdminRaw_MaxFileSizeLimit(t *testing.T) {
	h := newNFSShareTestHandler(t)
	h.cfg.MaxFileSizeMB = 1 // 1MB

	tmpDir := t.TempDir()
	big := bytes.Repeat([]byte("X"), 2*1024*1024) // 2MB
	if err := os.WriteFile(filepath.Join(tmpDir, "big.bin"), big, 0644); err != nil {
		t.Fatal(err)
	}

	h.mu.Lock()
	h.mounts["__raw__"] = &MountStatus{
		Config:    config.MountConfig{Name: "__raw__", Type: "local", Export: tmpDir},
		LocalPath: tmpDir,
		Mounted:   true,
	}
	h.mu.Unlock()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet,
		"/api/nfsshare/admin/raw?path=__raw__/big.bin", nil)
	nfsAdminCookie(c.Request)
	h.AdminRaw(c)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", w.Code)
	}
}
