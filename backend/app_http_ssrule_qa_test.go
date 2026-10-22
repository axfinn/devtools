package main

// =====================================================================
//  FINN-94 Stage 5 · ssrules HTTP 级补测(package main)
//
//  补 Stage 3 交付的 app_http_ssrule_test.go 没覆盖的断言:
//
//    A6(f)/Q3 真实分发端把 120 次配额打满之后,**别的模块**的限流计数
//             不能被连带消耗(分发端限流桶必须与全站共享桶隔离)
//    裁决 2   NoRoute 里的 /ss/ 特判只拦 /ss/ 前缀 —— 其它路径的
//             SPA fallback 行为不能被误伤(反向对照)
//    裁决 2   裸 /ss/ 前缀(未匹配到路由)必须回同款 404,不能回落 SPA
//
//  这些必须在**真实接线**上验(package main 复用 registerSSRulePublicRoutes
//  + registerStaticRoutes),手抄路由等于没测到接线。
// =====================================================================

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"devtools/handlers"
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// ssRuleQASharedLimiterLimit 是本用例里「别的模块」的共享限流器阈值。
//
// 取 10(与全站 createRateLimiter 的默认阈值同量级)即可:
// 本用例要证明的是**两个桶互不干扰**,不是阈值本身。
const ssRuleQASharedLimiterLimit = 10

// TestSSRulePublicRouteRateLimitDoesNotConsumeSharedBucket A6(f)/Q3:
// 真实分发端把 SSRuleDistributeRateLimitPerMinute 次/分钟 的配额打满
// (第 N+1 次 429)之后,挂在**共享限流器**上的其它模块端点仍然放行 ——
// 说明分发端的限流流量没有记到全站共享桶上。
//
// 反事实:把分发端改挂共享限流器 → 本用例必须红。
func TestSSRulePublicRouteRateLimitDoesNotConsumeSharedBucket(t *testing.T) {
	r, token := newSSRulePublicTestRouter(t)

	shared := middleware.NewRateLimiter(ssRuleQASharedLimiterLimit, time.Minute, nil)
	r.POST("/api/other-module", shared.Middleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	url := "/ss/" + token + "/rules.acl"
	for i := 1; i <= handlers.SSRuleDistributeRateLimitPerMinute; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("第 %d 次不该被限流,得到 429", i)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("第 %d 次想要 429,得到 %d",
			handlers.SSRuleDistributeRateLimitPerMinute+1, w.Code)
	}

	// 共享桶必须还是空的:第 1 次请求就该放行。
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/other-module", strings.NewReader("{}")))
	if w.Code != http.StatusOK {
		t.Fatalf("分发端打满 %d 次后,共享桶的第 1 次请求想要 200,得到 %d"+
			"(分发端的流量污染了全站共享限流桶)", handlers.SSRuleDistributeRateLimitPerMinute, w.Code)
	}
}

// TestSSRulePublicRouteNoRouteOnlyGuardsSSPrefix 裁决 2 的反向对照:
// NoRoute 里的 /ss/ 特判只该拦 /ss/ 前缀 —— 其它路径不能被误伤。
//
// 这里只断言「不会回那句规则文案的 404」:test 环境里没有 dist/index.html,
// SPA fallback 本来也拿不到 200,但**绝不该**变成「规则文件不存在」。
//
// 反事实:把特判改成拦所有路径 → 本用例必须红。
func TestSSRulePublicRouteNoRouteOnlyGuardsSSPrefix(t *testing.T) {
	r, _ := newSSRulePublicTestRouter(t)

	for _, p := range []string{"/some/spa/path", "/another/page", "/assets/x.js"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if strings.Contains(w.Body.String(), handlers.SSRuleNotFoundBody) {
			t.Errorf("%s 不该被 /ss/ 特判拦下(正文: %q)", p, w.Body.String())
		}
	}
}

// TestSSRulePublicRouteTrailingSlashReturnsNotFound 裁决 2 的尾斜杠子项:
// 尾斜杠形态必须回同款 404,而不是 200 + 规则正文。
//
// 方案 §6.3 给上游 NoRedirect 那条写的反事实是「把 /ss/:token/:filename/
// 指向 Download → 尾斜杠那条必须红」—— 实测**不成立**:
// NoRedirect 只断言「不是 3xx」,而指向 Download 时尾斜杠回的是 **200**
// (不是 3xx),于是那条依然是绿的。真正红的是 TestSSRulePublicRouteShapes。
// 本用例把「尾斜杠必须 404 + 正文逐字节等于「规则文件不存在」」
// 单独钉死,让上面那个注入真的能红。
//
// 反事实:把 /ss/:token/:filename/ 指向 h.Download → 本用例必须红。
func TestSSRulePublicRouteTrailingSlashReturnsNotFound(t *testing.T) {
	r, token := newSSRulePublicTestRouter(t)

	for _, p := range []string{
		"/ss/" + token + "/rules.acl/",
		"/ss/" + token + "/rules.txt/",
		"/ss/" + token + "/",
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s 想要 404,得到 %d", p, w.Code)
		}
		if got := w.Body.String(); got != handlers.SSRuleNotFoundBody {
			t.Errorf("%s 的正文想要 %q,得到 %q", p, handlers.SSRuleNotFoundBody, got)
		}
		if strings.Contains(w.Body.String(), "[proxy_list]") {
			t.Errorf("%s 泄露了规则正文:\n%s", p, w.Body.String())
		}
	}
}

// TestSSRulePublicRouteBarePrefixReturnsNotFound 裁决 2:
// 未匹配到路由的 /ss/ 前缀路径必须回同款 404「规则文件不存在」,
// 不能回落 SPA 兜成 index.html。
//
// 反事实:删掉 NoRoute 里的 /ss/ 特判 → 本用例必须红。
func TestSSRulePublicRouteBarePrefixReturnsNotFound(t *testing.T) {
	r, _ := newSSRulePublicTestRouter(t)

	for _, p := range []string{"/ss/", "/ss/unknown-token", "/ss/unknown-token/"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s 想要 404,得到 %d", p, w.Code)
		}
		if got := w.Body.String(); got != handlers.SSRuleNotFoundBody {
			t.Errorf("%s 的正文想要 %q,得到 %q", p, handlers.SSRuleNotFoundBody, got)
		}
	}
}
