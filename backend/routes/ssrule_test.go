package routes

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtools/handlers"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

const ssRuleRoutesTestPassword = "test-admin-pwd"

// newSSRuleRoutesTestEngine 搭一个只挂 ssrules 路由组的引擎。
//
// 每个用例都要**新的引擎**:限流器是注册时创建的,共用引擎会让前一个用例的
// 计数泄漏到下一个用例里,断言就成了恒真闸门。
func newSSRuleRoutesTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf(":memory: DB 创建失败: %v", err)
	}
	// R5：:memory: 必须单连接,否则会出现「刚插入就查不到」的假 404。
	db.SetMaxOpenConns(1)
	if err := db.InitSSRuleFiles(); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	r := gin.New()
	api := r.Group("/api")
	RegisterSSRuleRoutes(api, &RouteHandlers{
		SSRuleHandler: handlers.NewSSRuleHandler(db, ssRuleRoutesTestPassword),
	}, nil)
	return r
}

func ssRuleRoutesDo(r *gin.Engine, method, path string, headers map[string]string, body string) *httptest.ResponseRecorder {
	var reader io.Reader = strings.NewReader(body)
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestSSRuleRoutes_EndpointRateLimits 每个端点各有一个桶,阈值各不相同。
//
// 反事实:去掉 GET 上的限流 → GET 那条必须红;把 4 个管理端 limiter 合成 1 个
// → regenerate/GET/POST 三条的阈值必须红。
func TestSSRuleRoutes_EndpointRateLimits(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		body   string
		limit  int
	}{
		{"verify", http.MethodPost, "/api/ssrules/verify", "{}", 10},
		{"regenerate-token", http.MethodPost, "/api/ssrules/regenerate-token", "", 10},
		{"GET", http.MethodGet, "/api/ssrules", "", 30},
		{"POST", http.MethodPost, "/api/ssrules", `{"source":"a.com"}`, 20},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newSSRuleRoutesTestEngine(t)

			for i := 1; i <= c.limit; i++ {
				w := ssRuleRoutesDo(r, c.method, c.path, nil, c.body)
				if w.Code == http.StatusTooManyRequests {
					t.Fatalf("第 %d 次不该被限流,得到 429", i)
				}
			}
			w := ssRuleRoutesDo(r, c.method, c.path, nil, c.body)
			if w.Code != http.StatusTooManyRequests {
				t.Fatalf("第 %d 次想要 429,得到 %d", c.limit+1, w.Code)
			}
		})
	}
}

// TestSSRuleRoutes_RateLimitBucketsAreIndependent 桶之间互不干扰。
//
// 反事实:把 4 个管理端 limiter 合成 1 个 → 本用例必须红。
func TestSSRuleRoutes_RateLimitBucketsAreIndependent(t *testing.T) {
	r := newSSRuleRoutesTestEngine(t)

	// 先把 verify 的桶打满(10 次)。
	for i := 0; i < 10; i++ {
		ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules/verify", nil, "{}")
	}
	if w := ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules/verify", nil, "{}"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("verify 第 11 次想要 429,得到 %d", w.Code)
	}

	// 其它端点的桶是独立的,不该被连带限流。
	for _, c := range []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/ssrules", ""},
		{http.MethodPost, "/api/ssrules", `{"source":"a.com"}`},
		{http.MethodPost, "/api/ssrules/regenerate-token", ""},
	} {
		w := ssRuleRoutesDo(r, c.method, c.path, nil, c.body)
		if w.Code == http.StatusTooManyRequests {
			t.Errorf("%s %s 被 verify 的桶连带限流了 —— 桶没有隔离", c.method, c.path)
		}
	}
}

// TestSSRuleRoutes_RateLimitKeyNotSpoofable 限流键不可被客户端伪造。
//
// 这是 B2 的核心:gin 默认 trustedProxies=0.0.0.0/0 ⇒ c.ClientIP() 取的是
// XFF **最左**那一项 ⇒ 客户端自带 X-Forwarded-For 就能轮换限流桶。
//
// 反事实:把限流键改回 c.ClientIP() → 前两个子用例必须红。
func TestSSRuleRoutes_RateLimitKeyNotSpoofable(t *testing.T) {
	t.Run("CF-Connecting-IP 优先于 XFF", func(t *testing.T) {
		// 对外拓扑只有 Cloudflare,CF 会覆写 CF-Connecting-IP。
		// 客户端再怎么改 XFF 也不该换掉限流桶。
		r := newSSRuleRoutesTestEngine(t)
		for i := 0; i < 10; i++ {
			ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules/verify", map[string]string{
				"CF-Connecting-IP": "1.2.3.4",
				"X-Forwarded-For":  fmt.Sprintf("9.9.9.%d", i),
			}, "{}")
		}
		w := ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules/verify", map[string]string{
			"CF-Connecting-IP": "1.2.3.4",
			"X-Forwarded-For":  "8.8.8.8",
		}, "{}")
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("换掉 XFF 后仍应被限流,得到 %d", w.Code)
		}
	})

	t.Run("无 CF 时取 XFF 最右一项", func(t *testing.T) {
		// 代理追加的那一项在**最右**;最左那项是客户端自己写的,不能信。
		r := newSSRuleRoutesTestEngine(t)
		for i := 0; i < 10; i++ {
			ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules/verify", map[string]string{
				"X-Forwarded-For": "9.9.9.1, 4.4.4.4",
			}, "{}")
		}
		w := ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules/verify", map[string]string{
			"X-Forwarded-For": "9.9.9.2, 4.4.4.4",
		}, "{}")
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("XFF 最右一项相同就该仍被限流,得到 %d", w.Code)
		}
	})

	t.Run("无 CF 无 XFF 时退到 RemoteAddr", func(t *testing.T) {
		r := newSSRuleRoutesTestEngine(t)
		for i := 0; i < 10; i++ {
			ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules/verify", nil, "{}")
		}
		w := ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules/verify", nil, "{}")
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("想要 429,得到 %d", w.Code)
		}
	})
}

// TestSSRuleRoutes_ContentSizeLimit 请求体上限与 400/413 的区分。
//
// 反事实:去掉组上的 ContentSizeLimiter(1<<20) → 413 那条必须红。
func TestSSRuleRoutes_ContentSizeLimit(t *testing.T) {
	r := newSSRuleRoutesTestEngine(t)
	auth := map[string]string{"X-Super-Admin-Password": ssRuleRoutesTestPassword}

	oversize := `{"source":"` + strings.Repeat("a", 1<<20) + `"}`
	w := ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules", auth, oversize)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("超限请求体想要 413,得到 %d", w.Code)
	}

	w = ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules", auth, "not-json")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 JSON 想要 400,得到 %d", w.Code)
	}
}
