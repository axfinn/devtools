package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"devtools/config"
	"devtools/state"

	"github.com/gin-gonic/gin"
)

// ============================================================
// /api/health 单测 — 对应设计文档 docs/plans/health-dependency-probe.md 的 T1–T22
//
// 规则：
//   - plain testing + httptest + gin.TestMode，NO testify
//   - 复用同包 newMemDB(t)（已遵守 R5 的 SetMaxOpenConns(1)）
//   - 全程不起 dev server、不 go run、不 docker compose
// ============================================================

const healthTestPassword = "test-admin-pw"

// healthAdminHeader 是详细形态的单测固定装置。
func healthAdminHeader() map[string]string {
	return map[string]string{"X-Super-Admin-Password": healthTestPassword}
}

// newHealthTestConfig 造一个只开 monitoring.admin_password 的配置，
// 另两个密码字段置空，保证鉴权来源唯一可预测。
func newHealthTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Monitoring.AdminPassword = healthTestPassword
	cfg.AIGateway.SuperAdminPassword = ""
	cfg.Console.AdminPassword = ""
	cfg.Redis.Enabled = false
	cfg.Redis.Addr = "127.0.0.1:6379"
	cfg.Chat.TTSServiceURL = "http://127.0.0.1:1"
	return cfg
}

func newHealthEngine(h *HealthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/health", h.Handle)
	return r
}

func doHealthRequest(r *gin.Engine, target string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// decodeHealthPayload 解析详细形态响应体。
func decodeHealthPayload(t *testing.T, body string) HealthPayload {
	t.Helper()
	var p HealthPayload
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("详细形态响应体不是合法 JSON: %v\nbody=%s", err, body)
	}
	return p
}

func depByName(t *testing.T, p HealthPayload, name string) HealthDependency {
	t.Helper()
	for _, d := range p.Dependencies {
		if d.Name == name {
			return d
		}
	}
	t.Fatalf("dependencies 里找不到 %q: %+v", name, p.Dependencies)
	return HealthDependency{}
}

// healthMessageWhitelist 是设计 1.2.1 的闭集白名单。
// 「unexpected status <code>」是模板，用正则单独判定。
var healthMessageWhitelist = map[string]bool{
	"":                   true,
	"connection refused": true,
	"timeout":            true,
	"DNS 解析失败（地址可能仅适用于容器内）": true,
	"warming up":  true,
	"未启用（使用内存存储）": true,
	"已启用但未配置地址":   true,
	"已降级到内存存储":    true,
	"探测失败":        true,
}

var healthUnexpectedStatusRe = regexp.MustCompile(`^unexpected status \d+$`)

func assertMessageInWhitelist(t *testing.T, d HealthDependency) {
	t.Helper()
	if healthMessageWhitelist[d.Message] || healthUnexpectedStatusRe.MatchString(d.Message) {
		return
	}
	t.Errorf("%s 的 message 不在白名单内: %q", d.Name, d.Message)
}

// ============================================================
// T1 默认形态向后兼容（回归红线）
// ============================================================

func TestHealth_DefaultForm_BackwardCompatible(t *testing.T) {
	// 注入一个会 sleep 3s 的「慢依赖」，默认形态必须完全不碰它。
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slow.Close()
	t.Setenv("ASR_SERVICE_URL", slow.URL)

	cfg := newHealthTestConfig()
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())
	r := newHealthEngine(h)

	start := time.Now()
	w := doHealthRequest(r, "/api/health", nil)
	elapsed := time.Since(start)

	if w.Code != http.StatusOK {
		t.Fatalf("默认形态状态码 = %d, 期望 200", w.Code)
	}
	if got := w.Body.String(); got != `{"status":"ok"}` {
		t.Fatalf("默认形态响应体 = %q, 期望严格等于 {\"status\":\"ok\"}", got)
	}
	if elapsed >= 50*time.Millisecond {
		t.Fatalf("默认形态耗时 %v, 期望 < 50ms（不应执行任何探测）", elapsed)
	}
}

// ============================================================
// T2 详细形态正常路径（已授权）
// ============================================================

func TestHealth_DetailForm_Authorized_OK(t *testing.T) {
	db := newMemDB(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	t.Setenv("OCR_SERVICE_URL", upstream.URL)
	t.Setenv("ASR_SERVICE_URL", upstream.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = upstream.URL
	h := NewHealthHandler(db, cfg, state.NewMemoryStore())
	r := newHealthEngine(h)

	w := doHealthRequest(r, "/api/health?detail=1", healthAdminHeader())
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200; body=%s", w.Code, w.Body.String())
	}
	p := decodeHealthPayload(t, w.Body.String())

	if len(p.Dependencies) != 5 {
		t.Fatalf("dependencies 长度 = %d, 期望 5", len(p.Dependencies))
	}
	wantOrder := []string{"sqlite", "redis", "ocr", "asr", "tts"}
	for i, name := range wantOrder {
		if p.Dependencies[i].Name != name {
			t.Errorf("dependencies[%d].name = %q, 期望 %q", i, p.Dependencies[i].Name, name)
		}
	}
	for _, d := range p.Dependencies {
		if d.Status == "" || d.Source == "" {
			t.Errorf("%s 的 status/source 为空: %+v", d.Name, d)
		}
	}
	if p.Version == "" {
		t.Error("version 为空")
	}
	if p.Commit == "" || p.BuildTime == "" {
		t.Errorf("commit/build_time 为空: %q / %q", p.Commit, p.BuildTime)
	}
	if p.UptimeSeconds < 0 {
		t.Errorf("uptime_seconds = %d, 期望 >= 0", p.UptimeSeconds)
	}
	if _, err := time.Parse(time.RFC3339, p.ProbedAt); err != nil {
		t.Errorf("probed_at 不可解析: %v (%q)", err, p.ProbedAt)
	}
	if _, err := time.Parse(time.RFC3339, p.ServerTime); err != nil {
		t.Errorf("server_time 不可解析: %v (%q)", err, p.ServerTime)
	}
	if _, err := time.Parse(time.RFC3339, p.StartedAt); err != nil {
		t.Errorf("started_at 不可解析: %v (%q)", err, p.StartedAt)
	}
	if p.Status != healthOverallOK {
		t.Errorf("整体 status = %q, 期望 ok; deps=%+v", p.Status, p.Dependencies)
	}
	if p.Cached {
		t.Error("首次请求不应命中缓存")
	}
}

// ============================================================
// T3 detail 取值容错
// ============================================================

func TestHealth_DetailFlagParsing(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	t.Setenv("OCR_SERVICE_URL", upstream.URL)
	t.Setenv("ASR_SERVICE_URL", upstream.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = upstream.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())
	r := newHealthEngine(h)

	// 合法值：走详细形态
	for _, v := range []string{"1", "true", "TRUE", " yes ", "on"} {
		w := doHealthRequest(r, "/api/health?detail="+url.QueryEscape(v), healthAdminHeader())
		if w.Code != http.StatusOK {
			t.Errorf("detail=%q 状态码 = %d, 期望 200", v, w.Code)
		}
		if !strings.Contains(w.Body.String(), "dependencies") {
			t.Errorf("detail=%q 未走详细形态: %s", v, w.Body.String())
		}
	}

	// 非法值：即使带了正确 header 也按默认形态走，不返回 400
	for _, v := range []string{"0", "", "abc", "2"} {
		w := doHealthRequest(r, "/api/health?detail="+url.QueryEscape(v), healthAdminHeader())
		if w.Code != http.StatusOK {
			t.Errorf("detail=%q 状态码 = %d, 期望 200", v, w.Code)
		}
		if got := w.Body.String(); got != `{"status":"ok"}` {
			t.Errorf("detail=%q 响应体 = %q, 期望严格等于 {\"status\":\"ok\"}", v, got)
		}
	}
}

// ============================================================
// T4 Redis 未启用
// ============================================================

func TestHealth_RedisDisabled(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()
	t.Setenv("OCR_SERVICE_URL", stub.URL)
	t.Setenv("ASR_SERVICE_URL", stub.URL)

	cfg := newHealthTestConfig()
	cfg.Redis.Enabled = false
	cfg.Chat.TTSServiceURL = stub.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader())
	p := decodeHealthPayload(t, w.Body.String())
	redis := depByName(t, p, depRedis)

	if redis.Status != healthStatusDisabled {
		t.Errorf("redis.status = %q, 期望 disabled", redis.Status)
	}
	if redis.Backend != string(state.BackendMemory) {
		t.Errorf("redis.backend = %q, 期望 memory", redis.Backend)
	}
	if p.Status != healthOverallOK {
		t.Errorf("整体 status = %q, 期望 ok（disabled 不拉低）", p.Status)
	}
}

// ============================================================
// T5 Redis 不可用降级路径（启动时已降级 → store 是 Memory）
// ============================================================

func TestHealth_RedisDegraded(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()
	t.Setenv("OCR_SERVICE_URL", stub.URL)
	t.Setenv("ASR_SERVICE_URL", stub.URL)

	cfg := newHealthTestConfig()
	cfg.Redis.Enabled = true
	cfg.Redis.Addr = "127.0.0.1:1"
	cfg.Chat.TTSServiceURL = stub.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader())
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200（不是 500）", w.Code)
	}
	p := decodeHealthPayload(t, w.Body.String())
	redis := depByName(t, p, depRedis)

	if redis.Status != healthStatusDown {
		t.Errorf("redis.status = %q, 期望 down", redis.Status)
	}
	if redis.Backend != string(state.BackendMemory) {
		t.Errorf("redis.backend = %q, 期望 memory", redis.Backend)
	}
	assertMessageInWhitelist(t, redis)
	if redis.Message == "" {
		t.Error("redis.message 不应为空")
	}
	if p.Status != healthOverallDegraded {
		t.Errorf("整体 status = %q, 期望 degraded", p.Status)
	}
}

// ============================================================
// T6 依赖探测超时路径
// ============================================================

func TestHealth_ProbeTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slow.Close()
	t.Setenv("ASR_SERVICE_URL", slow.URL)

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	t.Setenv("OCR_SERVICE_URL", ok.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = ok.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())
	r := newHealthEngine(h)

	start := time.Now()
	w := doHealthRequest(r, "/api/health?detail=1", healthAdminHeader())
	elapsed := time.Since(start)

	p := decodeHealthPayload(t, w.Body.String())
	asr := depByName(t, p, depASR)
	if asr.Status != healthStatusDown {
		t.Errorf("asr.status = %q, 期望 down", asr.Status)
	}
	if asr.Message != healthMsgTimeout {
		t.Errorf("asr.message = %q, 期望 %q", asr.Message, healthMsgTimeout)
	}
	if elapsed >= 2500*time.Millisecond {
		t.Errorf("探测耗时 %v, 期望 < 2500ms", elapsed)
	}
}

// ============================================================
// T7 预热态（503）
// ============================================================

func TestHealth_WarmingUp(t *testing.T) {
	warm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"detail":"warming up"}`))
	}))
	defer warm.Close()
	t.Setenv("OCR_SERVICE_URL", warm.URL)

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	t.Setenv("ASR_SERVICE_URL", ok.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = ok.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader())
	p := decodeHealthPayload(t, w.Body.String())
	ocr := depByName(t, p, depOCR)

	if ocr.Status != healthStatusWarming {
		t.Errorf("ocr.status = %q, 期望 warming", ocr.Status)
	}
	assertMessageInWhitelist(t, ocr)
	if p.Status != healthOverallDegraded {
		t.Errorf("整体 status = %q, 期望 degraded", p.Status)
	}
}

// ============================================================
// T8 非 200/503 状态
// ============================================================

func TestHealth_UnexpectedStatus(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	t.Setenv("OCR_SERVICE_URL", bad.URL)

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	t.Setenv("ASR_SERVICE_URL", ok.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = ok.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader())
	p := decodeHealthPayload(t, w.Body.String())
	ocr := depByName(t, p, depOCR)

	if ocr.Status != healthStatusDown {
		t.Errorf("ocr.status = %q, 期望 down", ocr.Status)
	}
	if ocr.Message != "unexpected status 500" {
		t.Errorf("ocr.message = %q, 期望 unexpected status 500", ocr.Message)
	}
}

// ============================================================
// T9 SQLite 故障
// ============================================================

func TestHealth_SQLiteDown(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()
	t.Setenv("OCR_SERVICE_URL", stub.URL)
	t.Setenv("ASR_SERVICE_URL", stub.URL)

	db := newMemDB(t)
	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = stub.URL
	h := NewHealthHandler(db, cfg, state.NewMemoryStore())

	if err := db.Close(); err != nil {
		t.Fatalf("关闭 DB 失败: %v", err)
	}

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader())
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200", w.Code)
	}
	p := decodeHealthPayload(t, w.Body.String())
	sqlite := depByName(t, p, depSQLite)

	if sqlite.Status != healthStatusDown {
		t.Errorf("sqlite.status = %q, 期望 down", sqlite.Status)
	}
	if p.Status != healthOverallError {
		t.Errorf("整体 status = %q, 期望 error", p.Status)
	}
}

// ============================================================
// T10 缓存
// ============================================================

func TestHealth_Cache(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()
	t.Setenv("OCR_SERVICE_URL", stub.URL)
	t.Setenv("ASR_SERVICE_URL", stub.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = stub.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())
	r := newHealthEngine(h)

	first := decodeHealthPayload(t, doHealthRequest(r, "/api/health?detail=1", healthAdminHeader()).Body.String())
	if first.Cached {
		t.Fatal("首次请求不应命中缓存")
	}

	second := decodeHealthPayload(t, doHealthRequest(r, "/api/health?detail=1", healthAdminHeader()).Body.String())
	if !second.Cached {
		t.Error("第二次请求应命中缓存")
	}
	if second.ProbeMS != 0 {
		t.Errorf("缓存命中时 probe_ms = %d, 期望 0", second.ProbeMS)
	}
	if second.ProbedAt != first.ProbedAt {
		t.Errorf("缓存命中时 probed_at 应保持不变: %q → %q", first.ProbedAt, second.ProbedAt)
	}

	time.Sleep(2100 * time.Millisecond)

	third := decodeHealthPayload(t, doHealthRequest(r, "/api/health?detail=1", healthAdminHeader()).Body.String())
	if third.Cached {
		t.Error("缓存过期后不应命中缓存")
	}
	if third.ProbedAt == first.ProbedAt {
		t.Errorf("缓存过期后 probed_at 应前进: 仍是 %q", third.ProbedAt)
	}
}

// ============================================================
// T11 探测 goroutine 不倒灌 panic
// ============================================================

func TestHealth_ProbePanicIsContained(t *testing.T) {
	boom := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	}))
	defer boom.Close()
	t.Setenv("OCR_SERVICE_URL", boom.URL)

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	t.Setenv("ASR_SERVICE_URL", ok.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = ok.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader())
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200（探测 panic 不得冒泡）", w.Code)
	}
	p := decodeHealthPayload(t, w.Body.String())
	ocr := depByName(t, p, depOCR)
	if ocr.Status != healthStatusDown {
		t.Errorf("ocr.status = %q, 期望 down", ocr.Status)
	}
	assertMessageInWhitelist(t, ocr)
}

// ============================================================
// T12 版本 fallback 三级
// ============================================================

func TestHealth_VersionFallback(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()
	t.Setenv("OCR_SERVICE_URL", stub.URL)
	t.Setenv("ASR_SERVICE_URL", stub.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = stub.URL
	db := newMemDB(t)

	t.Setenv("DEVTOOLS_VERSION", "")
	h := NewHealthHandler(db, cfg, state.NewMemoryStore())
	p := decodeHealthPayload(t, doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader()).Body.String())
	if p.Version != "dev" {
		t.Errorf("无 ldflags / 无 env 时 version = %q, 期望 dev", p.Version)
	}

	t.Setenv("DEVTOOLS_VERSION", "v9.9.9")
	h2 := NewHealthHandler(db, cfg, state.NewMemoryStore())
	p2 := decodeHealthPayload(t, doHealthRequest(newHealthEngine(h2), "/api/health?detail=1", healthAdminHeader()).Body.String())
	if p2.Version != "v9.9.9" {
		t.Errorf("DEVTOOLS_VERSION 生效时 version = %q, 期望 v9.9.9", p2.Version)
	}
}

// ============================================================
// T13 state 层新方法
// ============================================================

func TestState_BackendAndPing(t *testing.T) {
	mem := state.NewMemoryStore()
	if got := mem.Backend(); got != state.BackendMemory {
		t.Errorf("MemoryStore.Backend() = %q, 期望 memory", got)
	}
	if err := mem.Ping(context.Background()); err != nil {
		t.Errorf("MemoryStore.Ping() = %v, 期望 nil", err)
	}

	cfg := config.DefaultConfig()
	cfg.Redis.Enabled = false
	store, err := state.New(cfg.Redis)
	if err != nil {
		t.Fatalf("state.New 失败: %v", err)
	}
	if store.Backend() != state.BackendMemory {
		t.Errorf("Redis 未启用时 store.Backend() = %q, 期望 memory", store.Backend())
	}
}

// ============================================================
// T14 未授权不泄露
// ============================================================

func TestHealth_DetailForm_Unauthorized_NoLeak(t *testing.T) {
	cfg := newHealthTestConfig()
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("状态码 = %d, 期望 401", w.Code)
	}
	body := w.Body.String()
	for _, forbidden := range []string{"dependencies", "version", "uptime", `"ocr"`, `"asr"`, `"tts"`, `"redis"`, `"sqlite"`} {
		if strings.Contains(body, forbidden) {
			t.Errorf("401 响应体泄露 %q: %s", forbidden, body)
		}
	}
}

// ============================================================
// T15 已授权正常（覆盖 T2 的最小版本）
// ============================================================

func TestHealth_DetailForm_Authorized_Parses(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()
	t.Setenv("OCR_SERVICE_URL", stub.URL)
	t.Setenv("ASR_SERVICE_URL", stub.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = stub.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader())
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200", w.Code)
	}
	p := decodeHealthPayload(t, w.Body.String())
	if len(p.Dependencies) != 5 {
		t.Fatalf("dependencies 长度 = %d, 期望 5", len(p.Dependencies))
	}
}

// ============================================================
// T16 密码错误
// ============================================================

func TestHealth_DetailForm_WrongPassword(t *testing.T) {
	cfg := newHealthTestConfig()
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1",
		map[string]string{"X-Super-Admin-Password": "wrong"})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("状态码 = %d, 期望 401", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "error") {
		t.Errorf("401 响应体应只有 error 字段: %s", body)
	}
	for _, forbidden := range []string{"dependencies", "version", "sqlite", "redis"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("401 响应体泄露 %q: %s", forbidden, body)
		}
	}
}

// ============================================================
// T17 未配置任何管理员密码
// ============================================================

func TestHealth_DetailForm_NoPasswordConfigured(t *testing.T) {
	cfg := newHealthTestConfig()
	cfg.Monitoring.AdminPassword = ""
	cfg.AIGateway.SuperAdminPassword = ""
	cfg.Console.AdminPassword = ""
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())
	r := newHealthEngine(h)

	w := doHealthRequest(r, "/api/health?detail=1", healthAdminHeader())
	if w.Code != http.StatusForbidden {
		t.Fatalf("状态码 = %d, 期望 403", w.Code)
	}

	// 默认形态不做鉴权判定，仍是 200
	d := doHealthRequest(r, "/api/health", nil)
	if d.Code != http.StatusOK {
		t.Fatalf("默认形态状态码 = %d, 期望 200", d.Code)
	}
	if got := d.Body.String(); got != `{"status":"ok"}` {
		t.Fatalf("默认形态响应体 = %q, 期望严格等于 {\"status\":\"ok\"}", got)
	}
}

// ============================================================
// T18 响应体泄露断言（核心验收项）
// ============================================================

func TestHealth_ResponseBody_NoInternalLeak(t *testing.T) {
	t.Setenv("OCR_SERVICE_URL", "http://127.0.0.1:1")
	t.Setenv("ASR_SERVICE_URL", "http://127.0.0.1:1")

	db := newMemDB(t)
	cfg := newHealthTestConfig()
	cfg.Redis.Enabled = true
	cfg.Redis.Addr = "127.0.0.1:1"
	cfg.Chat.TTSServiceURL = "http://127.0.0.1:1"
	h := NewHealthHandler(db, cfg, state.NewMemoryStore())
	if err := db.Close(); err != nil {
		t.Fatalf("关闭 DB 失败: %v", err)
	}

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader())
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200", w.Code)
	}
	body := w.Body.String()

	for _, forbidden := range []string{
		"ocr-service", "asr-service", "redis:6379", "127.0.0.1",
		"http://", "https://", "/app/", ".db",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("响应体泄露 %q:\n%s", forbidden, body)
		}
	}
	// 内网 IP 正则：设计文档 T18 给的是 `\b(10\.|172\.(1[6-9]|2\d|3[01])\.|192\.168\.)`，
	// 该写法过宽 —— 任何带毫秒的 RFC3339 时间戳（如 ...:52:10.228839Z）都会被
	// `\b10\.` 命中，导致这条断言恒失败。这里保留原意（10./172.16-31./192.168. 三段内网段），
	// 但要求完整的点分四段，避免把时间戳误判成内网地址。
	internalIP := regexp.MustCompile(`\b(10|172\.(1[6-9]|2\d|3[01])|192\.168)\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	if internalIP.MatchString(body) {
		t.Errorf("响应体含内网 IP:\n%s", body)
	}
}

// ============================================================
// T19 message 白名单（覆盖 T5/T6/T7/T8/T11 的响应）
// ============================================================

func TestHealth_MessageWhitelist(t *testing.T) {
	// 全依赖 down：sqlite 关闭、redis 降级、ocr/asr/tts 指向必然 refused 的地址。
	t.Setenv("OCR_SERVICE_URL", "http://127.0.0.1:1")
	t.Setenv("ASR_SERVICE_URL", "http://127.0.0.1:1")

	db := newMemDB(t)
	cfg := newHealthTestConfig()
	cfg.Redis.Enabled = true
	cfg.Redis.Addr = "127.0.0.1:1"
	cfg.Chat.TTSServiceURL = "http://127.0.0.1:1"
	h := NewHealthHandler(db, cfg, state.NewMemoryStore())
	if err := db.Close(); err != nil {
		t.Fatalf("关闭 DB 失败: %v", err)
	}

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader())
	p := decodeHealthPayload(t, w.Body.String())
	for _, d := range p.Dependencies {
		assertMessageInWhitelist(t, d)
		if d.Status == healthStatusUp && d.Message != "" {
			t.Errorf("%s 为 up 时 message 应为空, 实际 %q", d.Name, d.Message)
		}
	}
}

// ============================================================
// T20 source 字段
// ============================================================

func TestHealth_SourceField(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			hits.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	t.Setenv("ASR_SERVICE_URL", ok.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = ok.URL
	db := newMemDB(t)

	// 不设 OCR_SERVICE_URL → default
	t.Setenv("OCR_SERVICE_URL", "")
	h := NewHealthHandler(db, cfg, state.NewMemoryStore())
	p := decodeHealthPayload(t, doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader()).Body.String())
	if ocr := depByName(t, p, depOCR); ocr.Source != healthSourceDefault {
		t.Errorf("未设 OCR_SERVICE_URL 时 source = %q, 期望 default", ocr.Source)
	}

	// 设为 httptest 地址 → env，且探测确实打到该 server
	t.Setenv("OCR_SERVICE_URL", srv.URL)
	h2 := NewHealthHandler(db, cfg, state.NewMemoryStore())
	p2 := decodeHealthPayload(t, doHealthRequest(newHealthEngine(h2), "/api/health?detail=1", healthAdminHeader()).Body.String())
	ocr := depByName(t, p2, depOCR)
	if ocr.Source != healthSourceEnv {
		t.Errorf("设了 OCR_SERVICE_URL 时 source = %q, 期望 env", ocr.Source)
	}
	if ocr.Status != healthStatusUp {
		t.Errorf("ocr.status = %q, 期望 up", ocr.Status)
	}
	if hits.Load() == 0 {
		t.Error("OCR 探测没有打到 stub server")
	}
}

// ============================================================
// T21 响应头
// ============================================================

func TestHealth_ResponseHeaders(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()
	t.Setenv("OCR_SERVICE_URL", stub.URL)
	t.Setenv("ASR_SERVICE_URL", stub.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = stub.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())
	r := newHealthEngine(h)

	assertDetailHeaders := func(label string, w *httptest.ResponseRecorder) {
		t.Helper()
		cc := w.Header().Get("Cache-Control")
		if !strings.Contains(cc, "no-store") || !strings.Contains(cc, "private") {
			t.Errorf("%s: Cache-Control = %q, 期望含 no-store 与 private", label, cc)
		}
		if v := w.Header().Get("Vary"); v != "X-Super-Admin-Password" {
			t.Errorf("%s: Vary = %q, 期望 X-Super-Admin-Password", label, v)
		}
		if v := w.Header().Get("X-Content-Type-Options"); v != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q, 期望 nosniff", label, v)
		}
	}

	assertDetailHeaders("200", doHealthRequest(r, "/api/health?detail=1", healthAdminHeader()))
	assertDetailHeaders("401", doHealthRequest(r, "/api/health?detail=1", map[string]string{"X-Super-Admin-Password": "nope"}))

	noPW := newHealthTestConfig()
	noPW.Monitoring.AdminPassword = ""
	noPW.AIGateway.SuperAdminPassword = ""
	noPW.Console.AdminPassword = ""
	h403 := NewHealthHandler(newMemDB(t), noPW, state.NewMemoryStore())
	assertDetailHeaders("403", doHealthRequest(newHealthEngine(h403), "/api/health?detail=1", healthAdminHeader()))

	// 默认形态：一个响应头都不加
	w := doHealthRequest(r, "/api/health", nil)
	for _, name := range []string{"Cache-Control", "Vary", "X-Content-Type-Options"} {
		if v := w.Header().Get(name); v != "" {
			t.Errorf("默认形态不应设置 %s, 实际 %q", name, v)
		}
	}
}

// ============================================================
// T22 DNS 失败归类
// ============================================================

func TestHealth_DNSFailureMessage(t *testing.T) {
	// .invalid 是 RFC 2606 保留 TLD，保证解析不了。
	t.Setenv("ASR_SERVICE_URL", "http://asr-service.invalid:9000")

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	t.Setenv("OCR_SERVICE_URL", ok.URL)

	cfg := newHealthTestConfig()
	cfg.Chat.TTSServiceURL = ok.URL
	h := NewHealthHandler(newMemDB(t), cfg, state.NewMemoryStore())

	w := doHealthRequest(newHealthEngine(h), "/api/health?detail=1", healthAdminHeader())
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200", w.Code)
	}
	p := decodeHealthPayload(t, w.Body.String())
	asr := depByName(t, p, depASR)
	if asr.Status != healthStatusDown {
		t.Errorf("asr.status = %q, 期望 down", asr.Status)
	}
	if asr.Message != healthMsgDNSFailure {
		t.Errorf("asr.message = %q, 期望 %q", asr.Message, healthMsgDNSFailure)
	}
}
