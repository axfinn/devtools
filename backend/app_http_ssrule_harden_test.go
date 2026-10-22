package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devtools/handlers"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// ssRuleHardenTestToken 是单测里用的假分发 token。
//
// 必须是**足够独特**的字符串:「日志里没有它」这条断言只有在 token
// 不会与别的字符串撞上时才有意义 —— 用 token / test 之类通用词,
// 断言可能因为别处出现同名字符串而恒真。
const ssRuleHardenTestToken = "0123456789abcdef0123456789abcdef"

// ssRuleHardenTestContent 是存进库里的规则正文,用来证明分发链路真的通。
const ssRuleHardenTestContent = "# Shadowsocks 分流规则\n\n[bypass_all]\n\n[proxy_list]\n(^|\\.)google\\.com$\n"

// newSSRuleHardenedTestRouter 搭一个带**真实**遮罩 + **真实** gin Logger
// + **真实** /ss/ 分发路由的引擎。
//
// gin.Logger() 在 LoggerWithConfig 调用时就把输出目标(gin.DefaultWriter)
// 固定下来 ⇒ 必须在调用 gin.Logger() **之前**把 DefaultWriter 换成缓冲区,
// 事后再换就晚了(日志会打到 stdout,断言永远读到空串)。
func newSSRuleHardenedTestRouter(t *testing.T, buf *bytes.Buffer) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	prevWriter := gin.DefaultWriter
	gin.DefaultWriter = buf
	t.Cleanup(func() { gin.DefaultWriter = prevWriter })

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf(":memory: DB 创建失败: %v", err)
	}
	// R5：:memory: 必须单连接,否则会出现「刚插入就查不到」的假 404。
	db.SetMaxOpenConns(1)
	if err := db.InitSSRuleFiles(); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	file, err := db.SaveSSRuleFile("google.com", ssRuleHardenTestContent)
	if err != nil {
		t.Fatalf("保存规则失败: %v", err)
	}

	r := gin.New()
	r.Use(redactSSRuleTokenPath())
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	registerSSRulePublicRoutes(r, handlers.NewSSRuleHandler(db, "test-admin-pwd"))
	registerStaticRoutes(r, t.TempDir())
	return r, file.Token
}

// TestSSRuleAccessLogMasksToken /ss/ 分发请求的访问日志里不能出现 token。
//
// token 等同于凭据:访问日志一旦被聚合/落盘,留着 token 就是留着一份
// 可用的凭据副本。
func TestSSRuleAccessLogMasksToken(t *testing.T) {
	var buf bytes.Buffer
	r, token := newSSRuleHardenedTestRouter(t, &buf)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ss/"+token+"/rules.acl", nil)
	r.ServeHTTP(w, req)

	logged := buf.String()
	if strings.Contains(logged, token) {
		t.Fatalf("访问日志泄露了分发 token:\n%s", logged)
	}
	// 遮罩不能把可观测性一起抹掉:路径形状与状态码还要看得见。
	if !strings.Contains(logged, "/ss/"+ssRuleTokenMask+"/rules.acl") {
		t.Fatalf("访问日志里没有遮罩后的路径(可观测性被误伤):\n%s", logged)
	}
	// 遮罩的是**日志**,不是分发本身:正文必须照常发出去。
	if w.Code != http.StatusOK {
		t.Fatalf("分发状态码 = %d, 期望 200", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "proxy_list") {
		t.Fatalf("分发正文不对: %q", body)
	}
}

// TestSSRuleAccessLogKeepsOtherPaths 遮罩只作用于 /ss/ 前缀。
//
// 这是「不影响其它模块日志行为」的回归闸门:非 /ss/ 路径必须原样进日志。
func TestSSRuleAccessLogKeepsOtherPaths(t *testing.T) {
	var buf bytes.Buffer
	r, _ := newSSRuleHardenedTestRouter(t, &buf)
	r.GET("/api/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/ping", nil))

	logged := buf.String()
	if !strings.Contains(logged, "/api/ping") {
		t.Fatalf("非 /ss/ 路径的访问日志被误伤:\n%s", logged)
	}
}

// TestSSRuleAccessLogMasksTokenOn404Forms /ss/ 下的畸形形态同样不能泄 token。
//
// 客户端会带各种奇怪的路径来试;这些形态虽然一律 404,但**日志里照样会
// 出现路径** —— 遮罩必须覆盖全部形态,不能只覆盖标准形态。
func TestSSRuleAccessLogMasksTokenOn404Forms(t *testing.T) {
	var buf bytes.Buffer
	r, token := newSSRuleHardenedTestRouter(t, &buf)

	paths := []string{
		"/ss/" + token,
		"/ss/" + token + "/",
		"/ss/" + token + "/rules.acl/",
		"/ss/" + token + "/../../etc/passwd",
		"/ss/" + token + "/rules.pac",
	}
	for _, p := range paths {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s 状态码 = %d, 期望 404", p, w.Code)
		}
	}

	logged := buf.String()
	if strings.Contains(logged, token) {
		t.Fatalf("畸形路径的访问日志泄露了分发 token:\n%s", logged)
	}
}

// TestSSRulePublicRoutesNilHandler handler 缺失时不得 panic。
//
// 生产路径上 handler 恒非 nil,这条是「将来若加条件构造」的护栏:
// 注册阶段不许 nil 解引用。
func TestSSRulePublicRoutesNilHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	distDir := t.TempDir()
	// 放一个可识别的 index.html:这样「掉进 SPA fallback」(200 + HTML)
	// 与「被 /ss/ 的 404 兜住」就能被区分开,断言才是可证伪的。
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<!doctype html><title>spa</title>"), 0644); err != nil {
		t.Fatalf("写 index.html 失败: %v", err)
	}

	r := gin.New()
	// 传 nil:不许 panic(panic 会让本用例直接失败)。
	registerSSRulePublicRoutes(r, nil)
	registerStaticRoutes(r, distDir)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ss/"+ssRuleHardenTestToken+"/rules.acl", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("nil handler 下 /ss/ 状态码 = %d, 期望 404", w.Code)
	}
	if body := w.Body.String(); strings.Contains(body, "<!doctype html>") {
		t.Fatalf("nil handler 下 /ss/ 掉进了 SPA fallback: %q", body)
	}
}
