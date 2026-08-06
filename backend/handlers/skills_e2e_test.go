package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtools/config"
	"devtools/middleware"
	"devtools/models"
	"devtools/state"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 端到端 Skills 路由测试 — 装 SkillsGuard + SkillsHandler,模拟真实生产 path
// ============================================================

func newEndToEndRouter(t *testing.T, perMin, writePerMin int) (*gin.Engine, *models.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf(":memory: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll %v", err)
	}

	store := state.NewMemoryStore()
	skillsHandler := NewSkillsHandler(db)
	skillsGuard := middleware.NewSkillsGuard(config.SkillsConfig{
		Enabled:                 true,
		RateLimitPerMinute:      perMin,
		WriteRateLimitPerMinute: writePerMin,
		AllowedOrigins:          []string{},
	}, store)
	skillsHandler.AttachGuard(skillsGuard)

	r := gin.New()
	skills := r.Group("/api/skills", skillsGuard.Middleware())
	{
		skills.GET("/manifest", skillsHandler.GetManifest)
		skills.GET("/mcp", skillsHandler.MCPGetHandler)
		skills.POST("/mcp", skillsHandler.MCPPostHandler)
		skills.POST("/invoke", skillsHandler.Invoke)
	}
	return r, db
}

func TestSkills_E2E_Manifest(t *testing.T) {
	r, _ := newEndToEndRouter(t, 100, 5)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/skills/manifest", nil)
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("manifest 应 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"tools":`) {
		t.Errorf("缺 tools 字段: %s", w.Body.String())
	}
}

func TestSkills_E2E_MCPInitialize(t *testing.T) {
	r, _ := newEndToEndRouter(t, 100, 5)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/skills/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("MCP init 应 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"serverInfo"`) {
		t.Errorf("MCP init 缺 serverInfo: %s", w.Body.String())
	}
}

func TestSkills_E2E_MCPToolsCall_IpLookup(t *testing.T) {
	r, _ := newEndToEndRouter(t, 100, 5)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/skills/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ip_lookup","arguments":{}}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("tools/call ip_lookup 应 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"content":`) {
		t.Errorf("缺 content 字段, body=%s", w.Body.String())
	}
}

func TestSkills_E2E_PasteCreate_ThroughGuard(t *testing.T) {
	r, _ := newEndToEndRouter(t, 100, 5)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/skills/invoke",
		strings.NewReader(`{"name":"paste_create","arguments":{"content":"hello"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "2.2.2.2")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("paste 通过 guard 应 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"ok":true`) {
		t.Errorf("body 无 ok=true: %s", w.Body.String())
	}
}

func TestSkills_E2E_PasteExhaustsWriteBucket(t *testing.T) {
	r, _ := newEndToEndRouter(t, 100, 2) // 写库专项 2/min/IP
	hdr := func() (string, string) { return "X-Forwarded-For", "3.3.3.3" }

	// 第 1、2 次 paste_create OK
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/skills/invoke",
			strings.NewReader(`{"name":"paste_create","arguments":{"content":"hi"}}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(hdr())
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("paste 第 %d 次 应 200, got %d body=%s", i+1, w.Code, w.Body.String())
		}
	}
	// 第 3 次写库超限
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/skills/invoke",
		strings.NewReader(`{"name":"paste_create","arguments":{"content":"hi"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hdr())
	r.ServeHTTP(w, req)
	if w.Code != 429 {
		t.Fatalf("paste 第 3 次应 429, got %d body=%s", w.Code, w.Body.String())
	}

	// 但同 IP 的 compute 类不受影响(ip_lookup 应仍能调)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/skills/invoke",
		strings.NewReader(`{"name":"ip_lookup","arguments":{}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hdr())
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("写库打满不应影响 compute 类, got %d", w.Code)
	}
}

func TestSkills_E2E_Disabled_404s(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _ := models.NewDB(":memory:"); db.SetMaxOpenConns(1); db.InitAll()

	store := state.NewMemoryStore()
	skillsHandler := NewSkillsHandler(db)
	skillsGuard := middleware.NewSkillsGuard(config.SkillsConfig{
		Enabled: false,
	}, store)
	skillsHandler.AttachGuard(skillsGuard)

	r := gin.New()
	skills := r.Group("/api/skills", skillsGuard.Middleware())
	{
		skills.GET("/manifest", skillsHandler.GetManifest)
		skills.POST("/invoke", skillsHandler.Invoke)
	}

	for _, path := range []string{"/api/skills/manifest", "/api/skills/invoke"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		r.ServeHTTP(w, req)
		if w.Code != 404 && w.Code != 405 { // GET /invoke 会 405,但 group guard 在 route 匹配前先跑
			t.Fatalf("disabled 下 %s 应被 guard 404, got %d", path, w.Code)
		}
	}
}
