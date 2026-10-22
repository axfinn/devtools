package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtools/handlers"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// ssRuleTestACLContent 是存进库里的规则正文。
//
// 手写而不是调 buildACLRules(那是 handlers 包的未导出函数):本文件在 package main,
// 跨包拿不到。内容只需带 ACL 的段标记,用来证明「正文真的能出去」。
const ssRuleTestACLContent = "# Shadowsocks 分流规则 —— 由 DevTools 生成\n\n[bypass_all]\n\n[proxy_list]\n(^|\\.)google\\.com$\n"

// ssRulePublicPayloads 是路径穿越载荷在 **URL 层** 的形态。
//
// 与 handlers 包的 ssRuleTraversalPayloads 一一对应(方案 §3.4.1(c) 的
// 10 行展开成 14 个具体载荷),区别只在于这里走真实的路由 + NoRoute 落点。
//
// 这里必须用**真实的** registerSSRulePublicRoutes + registerStaticRoutes:
// 手抄一遍路由就等于没有测到真实接线,接线漂移不会被发现。
func ssRulePublicPayloads(token string) []string {
	return []string{
		"/ss/" + token + "/../../etc/passwd",
		"/ss/" + token + "/..%2f..%2fetc%2fpasswd",
		"/ss/" + token + "/%2e%2e%2f%2e%2e%2fpasswd",
		"/ss/" + token + "/....//....//etc/passwd",
		"/ss/" + token + "/rules.acl%00.txt",
		"/ss/" + token + "/rules.acl%00",
		"/ss/" + token + "/rules.acl.txt",
		"/ss/" + token + "/rules.acl/",
		"/ss/" + token + `/rules.acl\..\x`,
		"/ss/" + token + "/rules.pac",
		"/ss/" + token + "/index.html",
		"/ss/" + token + "/rules.yaml",
		"/ss/" + token + "/rules",
		"/ss/" + token + "/",
	}
}

// newSSRulePublicTestRouter 搭一个带真实 /ss/ 路由与真实 NoRoute 的引擎。
//
// 返回真实 token:载荷 URL 必须用**真实存在的 token**,否则「白名单/路由被
// 指错」也会碰巧回 404 —— 那样断言就成了恒真闸门。
func newSSRulePublicTestRouter(t *testing.T) (*gin.Engine, string) {
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

	file, err := db.SaveSSRuleFile("google.com", ssRuleTestACLContent)
	if err != nil {
		t.Fatalf("保存规则失败: %v", err)
	}

	r := gin.New()
	registerSSRulePublicRoutes(r, handlers.NewSSRuleHandler(db, "test-admin-pwd"))
	registerStaticRoutes(r, t.TempDir())
	return r, file.Token
}

// TestSSRulePublicRouteShapes /ss/ 前缀下所有形态都必须回同款 404。
//
// 三条断言缺一不可:
//   - 状态码必须是 404(不是 3xx、不是 5xx)
//   - body 必须**逐字节**等于「规则文件不存在」(不能是 SPA 的 index.html)
//   - body 不能含规则正文特征(不能把 ACL 泄出去)
//
// 先跑一条**正向对照**:真实 token + 白名单文件名必须 200 且带规则正文 ——
// 没有这条,「全部 404」也能让本用例全绿,断言就成了恒真闸门。
//
// 反事实:
//   - 删掉 NoRoute 里的 /ss/ 特判 → 路径穿越那几条必须红(变成 200 + HTML)
//   - 把 /ss/:token/:filename/ 占位路由指向 Download → 尾斜杠那条必须红(返回 200 + 正文)
func TestSSRulePublicRouteShapes(t *testing.T) {
	r, token := newSSRulePublicTestRouter(t)

	t.Run("正向对照:真实 token 能取到规则", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ss/"+token+"/rules.acl", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("想要 200,得到 %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "[proxy_list]") {
			t.Fatalf("响应里没有规则正文:\n%s", w.Body.String())
		}
	})

	for _, p := range ssRulePublicPayloads(token) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))

		if w.Code != http.StatusNotFound {
			t.Errorf("%s 想要 404,得到 %d", p, w.Code)
		}
		if got := w.Body.String(); got != handlers.SSRuleNotFoundBody {
			t.Errorf("%s 的响应体想要 %q,得到 %q", p, handlers.SSRuleNotFoundBody, got)
		}
		if strings.Contains(w.Body.String(), "[proxy_list]") {
			t.Errorf("%s 泄露了规则正文:\n%s", p, w.Body.String())
		}
	}
}

// TestSSRulePublicRouteNoRedirect /ss/ 前缀不许出现 3xx。
//
// gin 默认 RedirectTrailingSlash=true,不加占位路由的话尾斜杠载荷会被
// 301 到无尾斜杠形态 —— 客户端拿到的不是规则文件也不是 404。
//
// 反事实:删掉 3 条占位路由 → 尾斜杠那条必须红(变 301)。
func TestSSRulePublicRouteNoRedirect(t *testing.T) {
	r, token := newSSRulePublicTestRouter(t)

	paths := ssRulePublicPayloads(token)
	paths = append(paths, "/ss/"+token, "/ss/")

	for _, p := range paths {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))

		if w.Code >= 300 && w.Code < 400 {
			t.Errorf("%s 不该是 3xx,得到 %d(Location=%q)", p, w.Code, w.Header().Get("Location"))
		}
		if loc := w.Header().Get("Location"); loc != "" {
			t.Errorf("%s 不该带 Location 头,得到 %q", p, loc)
		}
	}
}

// TestSSRulePublicRouteRateLimit 分发端有自己独立的限流桶:120 次/分钟/IP。
//
// 反事实:把分发端限流去掉 → 第 121 次必须红(得到 404 而不是 429)。
func TestSSRulePublicRouteRateLimit(t *testing.T) {
	r, token := newSSRulePublicTestRouter(t)

	url := fmt.Sprintf("/ss/%s/rules.acl", token)
	for i := 1; i <= 120; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("第 %d 次不该被限流,得到 429", i)
		}
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("第 121 次想要 429,得到 %d", w.Code)
	}
}
