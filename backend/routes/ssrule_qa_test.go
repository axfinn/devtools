package routes

// =====================================================================
//  FINN-94 Stage 5 · ssrules 路由层单测补测
//
//  补 Stage 3 交付的 routes/ssrule_test.go 没覆盖的断言:
//
//    A16   413 与 400 必须**报文形态**可区分(上游那 4 条只断言了状态码)
//    M1–M4 管理端鉴权失败的报文 {"error":"密码错误"}
//    Q3/A6(f) 本模块的限流桶与**全站共享限流器**互不干扰
//    Q6   管理端只有 4 个端点(不做「多套规则」)
//
//  另:上游那 4 条只断言状态码,报文形态全靠前端约定 —— 这里补齐。
// =====================================================================

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"devtools/handlers"
	"devtools/middleware"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// ssRuleQASharedLimiterLimit 是本用例里「别的模块」的共享限流器阈值。
//
// 取 10(与全站 createRateLimiter 的默认阈值同量级)即可:
// 本用例要证明的是**两个桶互不干扰**,不是阈值本身。
const ssRuleQASharedLimiterLimit = 10

// newSSRuleQARoutesEngine 搭一个带「共享限流器」的引擎。
//
// 第三个参数是 RegisterSSRuleRoutes 声明了但**刻意不用**的共享限流器。
// 本用例要用它来证明:ssrules 的限流流量不会消耗共享限流器的计数,
// 反过来共享限流器打满也不会连累 ssrules(Q3 / A6(f))。
func newSSRuleQARoutesEngine(t *testing.T, shared *middleware.RateLimiter) *gin.Engine {
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
	}, shared)
	return r
}

// TestSSRuleRoutes_ErrorBodiesMatchContract A16:413 与 400 必须能被前端分开处理。
//
// 报文形态:
//
//	413 → {"error":"内容大小超过限制","code":413}
//	400 → {"error":"请求格式错误"}
//
// 上游那 4 条只断言了状态码 —— 报文写错了前端照样分不开,
// 所以这里按 JSON 字段逐个比对(不依赖 map 键序)。
//
// 反事实:把 ContentSizeLimiter 的报文改成 400 那句 → 413 子项必须红;
//
//	把 ShouldBindJSON 的分支删掉(直接 500)→ 400 子项必须红。
func TestSSRuleRoutes_ErrorBodiesMatchContract(t *testing.T) {
	r := newSSRuleRoutesTestEngine(t)
	auth := map[string]string{"X-Super-Admin-Password": ssRuleRoutesTestPassword}

	t.Run("413 超限请求体", func(t *testing.T) {
		oversize := `{"source":"` + strings.Repeat("a", ssRuleMaxBodySize) + `"}`
		w := ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules", auth, oversize)
		if w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("想要 413,得到 %d", w.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("413 报文不是合法 JSON: %v\n%s", err, w.Body.String())
		}
		if body["error"] != "内容大小超过限制" {
			t.Errorf("413 的 error 想要 %q,得到 %v", "内容大小超过限制", body["error"])
		}
		if body["code"] != float64(http.StatusRequestEntityTooLarge) {
			t.Errorf("413 的 code 想要 %d,得到 %v", http.StatusRequestEntityTooLarge, body["code"])
		}
	})

	t.Run("400 非法 JSON", func(t *testing.T) {
		w := ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules", auth, "not-json")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("想要 400,得到 %d", w.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("400 报文不是合法 JSON: %v\n%s", err, w.Body.String())
		}
		if body["error"] != "请求格式错误" {
			t.Errorf("400 的 error 想要 %q,得到 %v", "请求格式错误", body["error"])
		}
		if _, hasCode := body["code"]; hasCode {
			t.Errorf("400 不该带 code 字段(前端按 413/400 分支处理): %s", w.Body.String())
		}
	})
}

// TestSSRuleRoutes_UnauthorizedBody M1–M4:鉴权失败的报文统一是
// {"error":"密码错误"} —— 前端据此回到密码门。
func TestSSRuleRoutes_UnauthorizedBody(t *testing.T) {
	r := newSSRuleRoutesTestEngine(t)

	cases := []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/ssrules/verify", "{}"},
		{http.MethodGet, "/api/ssrules", ""},
		{http.MethodPost, "/api/ssrules", `{"source":"a.com"}`},
		{http.MethodPost, "/api/ssrules/regenerate-token", ""},
	}
	for _, c := range cases {
		w := ssRuleRoutesDo(r, c.method, c.path, nil, c.body)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s 想要 401,得到 %d", c.method, c.path, w.Code)
			continue
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s %s 的 401 报文不是合法 JSON: %v\n%s", c.method, c.path, err, w.Body.String())
		}
		if body["error"] != "密码错误" {
			t.Errorf("%s %s 的 401 报文想要 error=%q,得到 %v",
				c.method, c.path, "密码错误", body["error"])
		}
	}
}

// TestSSRuleRoutes_SharedRateLimiterIsIsolated Q3 / A6(f):
// 本模块自建独立限流桶,**绝不**回落到全站共享限流器。
//
// 两个方向都要咬合:
//
//	① ssrules 打满自己的桶之后,共享限流器的第 1 次请求仍然放行
//	   (证明 ssrules 的流量没被记到共享桶上)
//	② 共享限流器打满之后,ssrules 仍然放行
//	   (证明 ssrules 用的不是共享桶)
//
// 反事实:把 NewSSRuleLimiter 换成共享的 createRateLimiter → 两条都必须红。
func TestSSRuleRoutes_SharedRateLimiterIsIsolated(t *testing.T) {
	shared := middleware.NewRateLimiter(ssRuleQASharedLimiterLimit, time.Minute, nil)
	r := newSSRuleQARoutesEngine(t, shared)

	// 「别的模块」的端点:挂共享限流器,代表全站其它模块的处境。
	r.POST("/api/other-module", shared.Middleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// ① 先把 ssrules 自己的 verify 桶打满(阈值 handlers.SSRuleVerifyRateLimitPerMinute)。
	for i := 1; i <= handlers.SSRuleVerifyRateLimitPerMinute; i++ {
		w := ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules/verify", nil, "{}")
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("verify 第 %d 次不该被限流,得到 429", i)
		}
	}
	if w := ssRuleRoutesDo(r, http.MethodPost, "/api/ssrules/verify", nil, "{}"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("verify 第 %d 次想要 429,得到 %d",
			handlers.SSRuleVerifyRateLimitPerMinute+1, w.Code)
	}

	// 共享桶的第 1 次请求必须仍然放行 —— ssrules 的 10 次没进共享桶。
	if w := ssRuleRoutesDo(r, http.MethodPost, "/api/other-module", nil, "{}"); w.Code != http.StatusOK {
		t.Fatalf("共享桶的第 1 次请求想要 200,得到 %d(ssrules 的流量污染了共享桶)", w.Code)
	}

	// ② 反向:把共享桶打满,ssrules 仍然放行 —— 它用的是自己的桶。
	for i := 2; i <= ssRuleQASharedLimiterLimit; i++ {
		w := ssRuleRoutesDo(r, http.MethodPost, "/api/other-module", nil, "{}")
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("共享桶第 %d 次不该被限流,得到 429", i)
		}
	}
	if w := ssRuleRoutesDo(r, http.MethodPost, "/api/other-module", nil, "{}"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("共享桶第 %d 次想要 429,得到 %d", ssRuleQASharedLimiterLimit+1, w.Code)
	}
	if w := ssRuleRoutesDo(r, http.MethodGet, "/api/ssrules", nil, ""); w.Code == http.StatusTooManyRequests {
		t.Error("共享桶打满不该连累 ssrules —— 两者没共用计数桶")
	}
}

// TestSSRuleRoutes_NoMultiRuleEndpoints Q6:管理端只有 4 个端点,
// 不做「多套规则」(没有 /ssrules/:id 之类的多套开关)。
//
// 反事实:加一条 /api/ssrules/:name 之类的路由 → 本用例必须红。
func TestSSRuleRoutes_NoMultiRuleEndpoints(t *testing.T) {
	r := newSSRuleRoutesTestEngine(t)

	got := map[string]bool{}
	for _, ri := range r.Routes() {
		got[ri.Method+" "+ri.Path] = true
	}
	want := []string{
		"POST /api/ssrules/verify",
		"GET /api/ssrules",
		"POST /api/ssrules",
		"POST /api/ssrules/regenerate-token",
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("缺少端点 %s", w)
		}
	}
	for path := range got {
		if !strings.HasPrefix(path, "POST /api/ssrules") && !strings.HasPrefix(path, "GET /api/ssrules") {
			t.Errorf("出现了本模块之外的端点 %s", path)
		}
	}
	if len(got) != len(want) {
		t.Errorf("管理端应当恰好 %d 个端点,实际 %d 个: %v", len(want), len(got), got)
	}
}
