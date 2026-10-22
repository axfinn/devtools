package main

// =====================================================================
//  FINN-111 · ssrules 分发 token 遮罩的**生产接线**回归闸门
//
//  背景(FINN-110 的遗留项):
//
//  FINN-110 交付的回归用例 backend/app_http_ssrule_harden_test.go 里,
//  测试**自己拼了一个 router**(gin.New() + r.Use(redactSSRuleTokenPath())
//  + registerSSRulePublicRoutes(...) + registerStaticRoutes(...)),
//  走的是**测试侧装配**,不是生产接线 newHTTPRouter(rt, handlers)。
//
//  ⇒ 后果:把 backend/app_http.go 里 newHTTPRouter 中的
//    `router.Use(redactSSRuleTokenPath())` 注掉,那 4 条用例**照样全绿**,
//    而生产环境里 token 已经重新泄进 stdout。
//
//  本文件补的就是这条闸门:**必须走 newHTTPRouter**。
//
//  可证伪要求(PM 验收点):
//    - 注掉生产代码 newHTTPRouter 里的 router.Use(redactSSRuleTokenPath())
//      ⇒ 本文件用例必须 FAIL(日志里会出现真实 token)
//    - 恢复该行 ⇒ 本文件用例必须 PASS
// =====================================================================

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtools/config"
	"devtools/models"
	"devtools/state"

	"github.com/gin-gonic/gin"
)

// ssRuleWiringTestContent 是存进库里的规则正文,用来证明分发链路真的通。
const ssRuleWiringTestContent = "# Shadowsocks 分流规则 —— 生产接线回归用\n\n[bypass_all]\n\n[proxy_list]\n(^|\\.)google\\.com$\n"

// ssRuleWiringTestAdminPwd 是测试用的 ssrules 管理密码。
//
// 非空 ⇒ 模块处于「开启」状态:分发端在 admin_password 为空时会回同款 404
// (handlers.SSRuleHandler.Download),开着才能验到真实的 200 分发路径。
const ssRuleWiringTestAdminPwd = "ssrule-wiring-test-admin-pwd"

// ssRuleWiringTestMasterKey 是测试用的 AI Gateway 主密钥。
//
// buildRouteHandlers 缺 DEPLOY_MASTER_KEY 会直接 log.Fatal,整个测试进程
// 会被干掉 —— 这里给一个固定的假值即可(不落库、不出网)。
const ssRuleWiringTestMasterKey = "ssrule-wiring-test-master-key"

// newSSRuleProductionRouter 构造**生产接线**的 router:newHTTPRouter(rt, hs)。
//
// 为什么必须真的调 newHTTPRouter:本用例要钉住的是「遮罩中间件有没有被
// 挂到生产 router 上」,而不是「遮罩中间件本身对不对」。自己拼一个
// gin.New() 的 router,断言就永远只钉在 helper 上 —— 生产接线漂移时
// 整条链不会红(这正是 FINN-110 遗留的那个缺口)。
//
// 为什么 gin.DefaultWriter 必须在 newHTTPRouter **之前**换掉:
// gin.Logger() 在被调用的那一刻就把输出目标固定下来,事后再换就晚了
// (日志会打到 stdout,断言永远读到空串)。
func newSSRuleProductionRouter(t *testing.T, buf *bytes.Buffer) (*gin.Engine, string) {
	t.Helper()

	prevWriter := gin.DefaultWriter
	gin.DefaultWriter = buf
	t.Cleanup(func() { gin.DefaultWriter = prevWriter })

	// buildRouteHandlers 缺 DEPLOY_MASTER_KEY 会直接 log.Fatal,
	// 整个测试进程会被干掉 —— 这里给一个固定的假值即可。
	t.Setenv("DEPLOY_MASTER_KEY", ssRuleWiringTestMasterKey)

	// DefaultConfig() 每次返回新实例,不会污染 config 的全局单例。
	cfg := config.DefaultConfig()
	cfg.Monitoring.Enabled = false
	// ssrules 模块「未配管理密码 = 模块关闭」:分发端也会回同款 404。
	// 这里给一个假密码把模块打开,否则断言只能落在 404 分支上。
	cfg.SSRules.AdminPassword = ssRuleWiringTestAdminPwd

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf(":memory: DB 创建失败: %v", err)
	}
	// R5：:memory: 必须单连接,否则会出现「刚插入就查不到」的假 404。
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := db.InitAll(); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	// 用**真实**生成的 token:断言「日志里没有它」才有意义。
	file, err := db.SaveSSRuleFile("google.com", ssRuleWiringTestContent)
	if err != nil {
		t.Fatalf("保存规则失败: %v", err)
	}
	if file.Token == "" {
		t.Fatal("保存规则后 token 为空,后续断言无意义")
	}

	// 生产形态的 runtime:内存 DB + 内存版 transient store(Redis 软依赖降级)。
	rt := &appRuntime{
		cfg:            cfg,
		port:           "0",
		db:             db,
		transientStore: state.NewMemoryStore(),
	}

	hs, err := buildRouteHandlers(rt)
	if err != nil {
		t.Fatalf("构造 routeHandlers 失败: %v", err)
	}

	// ★ 生产接线 ★
	router := newHTTPRouter(rt, hs)
	return router, file.Token
}

// TestSSRuleProductionRouterMasksTokenInAccessLog 生产接线上,
// /ss/<token>/rules.acl 的访问日志里不能出现 token。
//
// token 等同于凭据:访问日志一旦被聚合/落盘,留着 token 就是留着一份
// 可用的凭据副本。
//
// 反事实:把 backend/app_http.go 里 newHTTPRouter 中的
// `router.Use(redactSSRuleTokenPath())` 注掉 ⇒ 本用例必须 FAIL。
func TestSSRuleProductionRouterMasksTokenInAccessLog(t *testing.T) {
	var buf bytes.Buffer
	router, token := newSSRuleProductionRouter(t, &buf)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ss/"+token+"/rules.acl", nil))

	logged := buf.String()

	// 先确认请求真的打到了分发端点 —— 否则「日志里没有 token」可能只是
	// 因为请求压根没进这条路由,断言就成了恒真闸门。
	if w.Code != http.StatusOK {
		t.Fatalf("生产接线上的分发状态码 = %d, 期望 200\n访问日志:\n%s", w.Code, logged)
	}
	if body := w.Body.String(); !strings.Contains(body, "proxy_list") {
		t.Fatalf("生产接线上的分发正文不对: %q", body)
	}

	if strings.Contains(logged, token) {
		t.Fatalf("生产接线的访问日志泄露了分发 token:\n%s", logged)
	}
	// 遮罩不能把可观测性一起抹掉:遮罩后的路径与状态码还要看得见。
	if !strings.Contains(logged, "/ss/"+ssRuleTokenMask+"/rules.acl") {
		t.Fatalf("生产接线的访问日志里没有遮罩后的路径(可观测性被误伤):\n%s", logged)
	}
	if !strings.Contains(logged, "200") {
		t.Fatalf("生产接线的访问日志里没有状态码:\n%s", logged)
	}
}

// TestSSRuleProductionRouterMasksTokenOn404Forms 生产接线上,
// /ss/ 下的畸形形态同样不能泄 token。
//
// 客户端会带各种奇怪的路径来试;这些形态虽然一律 404,但**日志里照样会
// 出现路径** —— 遮罩必须覆盖全部形态,不能只覆盖标准形态。
func TestSSRuleProductionRouterMasksTokenOn404Forms(t *testing.T) {
	var buf bytes.Buffer
	router, token := newSSRuleProductionRouter(t, &buf)

	paths := []string{
		"/ss/" + token,
		"/ss/" + token + "/",
		"/ss/" + token + "/rules.acl/",
		"/ss/" + token + "/../../etc/passwd",
		"/ss/" + token + "/rules.pac",
	}
	for _, p := range paths {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s 状态码 = %d, 期望 404", p, w.Code)
		}
	}

	logged := buf.String()
	if strings.Contains(logged, token) {
		t.Fatalf("生产接线：畸形路径的访问日志泄露了分发 token:\n%s", logged)
	}
}

// TestSSRuleProductionRouterKeepsNonSSPathInAccessLog 遮罩只作用于 /ss/ 前缀。
//
// 这是「不影响其它模块日志行为」的回归闸门:非 /ss/ 路径必须原样进日志。
//
// 反向对照:若遮罩中间件误伤全站路径(例如把前缀判断写错成 HasPrefix("/")),
// 本用例会红。
func TestSSRuleProductionRouterKeepsNonSSPathInAccessLog(t *testing.T) {
	var buf bytes.Buffer
	router, _ := newSSRuleProductionRouter(t, &buf)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	logged := buf.String()
	if !strings.Contains(logged, "/api/health") {
		t.Fatalf("生产接线：非 /ss/ 路径的访问日志被误伤:\n%s", logged)
	}
	if strings.Contains(logged, ssRuleTokenMask) {
		t.Fatalf("生产接线：非 /ss/ 路径被无端遮罩:\n%s", logged)
	}
}
