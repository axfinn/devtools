package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"devtools/config"
	"devtools/state"

	"github.com/gin-gonic/gin"
)

// ============================================================
// SkillsGuard 中间件单元测试
// 覆盖:
//   1. enabled=false 时 404(无论 method)
//   2. enabled=true 时 GET(manifest 路径)不限流,通过
//   3. enabled=true 时 POST 走总配额限流,触发后 429
//   4. allowed_origins 非空时,不在白名单的 Origin → 403
// ============================================================

func newSkillsGuard(t *testing.T, enabled bool, perMin int, origins []string) (*SkillsGuard, state.TransientStore) {
	t.Helper()
	store := state.NewMemoryStore()
	guard := NewSkillsGuard(config.SkillsConfig{
		Enabled:                enabled,
		RateLimitPerMinute:     perMin,
		WriteRateLimitPerMinute: 5,
		AllowedOrigins:         origins,
	}, store)
	return guard, store
}

func TestSkillsGuard_DisabledBlocksAll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g, _ := newSkillsGuard(t, false, 60, nil)
	r := gin.New()
	r.GET("/api/skills/manifest", g.Middleware(), func(c *gin.Context) { c.String(200, "ok") })
	r.POST("/api/skills/invoke", g.Middleware(), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/skills/manifest", nil)
	r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("disabled 应 404 manifest, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/skills/invoke", nil)
	r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("disabled 应 404 invoke, got %d", w.Code)
	}
}

func TestSkillsGuard_GETAlwaysPasses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g, _ := newSkillsGuard(t, true, 60, nil)
	r := gin.New()
	r.GET("/api/skills/manifest", g.Middleware(), func(c *gin.Context) { c.String(200, "ok") })

	// 连发 100 次 GET 都应通过
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/skills/manifest", nil)
		req.Header.Set("X-Forwarded-For", "1.1.1.1")
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("GET 第 %d 次 应 200, got %d", i+1, w.Code)
		}
	}
}

func TestSkillsGuard_RateLimitIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g, _ := newSkillsGuard(t, true, 3, nil) // 3/min 配额,易于触发
	r := gin.New()
	r.POST("/api/skills/invoke", g.Middleware(), func(c *gin.Context) { c.String(200, "ok") })

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/skills/invoke", nil)
		req.Header.Set("X-Forwarded-For", "9.9.9.9")
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("POST 第 %d 次 应 200, got %d", i+1, w.Code)
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/skills/invoke", nil)
	req.Header.Set("X-Forwarded-For", "9.9.9.9")
	r.ServeHTTP(w, req)
	if w.Code != 429 {
		t.Fatalf("第 4 次 POST 应 429, got %d", w.Code)
	}
}

func TestSkillsGuard_DifferentIPsIndependent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g, _ := newSkillsGuard(t, true, 2, nil) // 2/min/IP
	r := gin.New()
	r.POST("/api/skills/invoke", g.Middleware(), func(c *gin.Context) { c.String(200, "ok") })

	// IP-A 用满 2 次
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/skills/invoke", nil)
		req.Header.Set("X-Forwarded-For", "1.1.1.1")
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("A 第 %d 次 应 200, got %d", i+1, w.Code)
		}
	}
	// IP-A 第三次应 429
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/skills/invoke", nil)
		req.Header.Set("X-Forwarded-For", "1.1.1.1")
		r.ServeHTTP(w, req)
		if w.Code != 429 {
			t.Fatalf("A 第三次应 429, got %d", w.Code)
		}
	}
	// IP-B 仍应可调
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/skills/invoke", nil)
		req.Header.Set("X-Forwarded-For", "2.2.2.2")
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("B 应 200(独立 IP), got %d", w.Code)
		}
	}
}

func TestSkillsGuard_OriginWhitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g, _ := newSkillsGuard(t, true, 60, []string{"https://allowed.example.com"})
	r := gin.New()
	r.POST("/api/skills/invoke", g.Middleware(), func(c *gin.Context) { c.String(200, "ok") })

	// 不在白名单 → 403
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/skills/invoke", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		r.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatalf("非白名单 Origin 应 403, got %d", w.Code)
		}
	}
	// 在白名单 → 200
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/skills/invoke", nil)
		req.Header.Set("Origin", "https://allowed.example.com")
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("白名单 Origin 应 200, got %d", w.Code)
		}
	}
	// 空 Origin(cURL/CLI) — 白名单非空时 仍 403(更严)
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/skills/invoke", nil)
		r.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatalf("空 Origin 白名单非空时 应 403, got %d", w.Code)
		}
	}
}

func TestSkillsGuard_OriginEmptyWhitelistAllowsAll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g, _ := newSkillsGuard(t, true, 60, nil)
	r := gin.New()
	r.POST("/api/skills/invoke", g.Middleware(), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/skills/invoke", nil)
	req.Header.Set("Origin", "https://whatever.com")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("白名单空时应全放行, got %d", w.Code)
	}
}

func TestSkillsGuard_WriteLimitSeparateBucket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g, store := newSkillsGuard(t, true, 100, nil)

	// 5/min 写库专项
	for i := 0; i < 5; i++ {
		if err := g.CheckWriteLimit("shorturl_create", "5.5.5.5"); err != nil {
			t.Fatalf("写库第 %d 次 应通过, got %v", i+1, err)
		}
	}
	if err := g.CheckWriteLimit("shorturl_create", "5.5.5.5"); err == nil {
		t.Fatalf("写库第 6 次 应拒绝")
	}
	// 不同 IP 独立
	if err := g.CheckWriteLimit("shorturl_create", "6.6.6.6"); err != nil {
		t.Fatalf("不同 IP 写库应仍可, got %v", err)
	}
	// 不同 skill 独立
	if err := g.CheckWriteLimit("paste_create", "5.5.5.5"); err != nil {
		t.Fatalf("不同 skill 写库应独立, got %v", err)
	}
	_ = store
}

func TestGetClientIP_XForwardedForFirst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Forwarded-For", "1.1.1.1, 10.0.0.1, 192.168.1.1")
	if got := GetClientIP(c); got != "1.1.1.1" {
		t.Errorf("GetClientIP XFF 第一个, got %q", got)
	}
}

func TestGetClientIP_XRealIPFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Real-IP", "8.8.8.8")
	if got := GetClientIP(c); got != "8.8.8.8" {
		t.Errorf("GetClientIP XRI fallback, got %q", got)
	}
}

func TestGetClientIP_NoHeader_UsesRemoteAddr(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "203.0.113.7:55555"
	if got := GetClientIP(c); got != "203.0.113.7" {
		t.Errorf("无头时用 RemoteAddr, got %q", got)
	}
	_ = time.Now() // keep time import touched
}
