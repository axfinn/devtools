package main

// =====================================================================
//  FINN-114 · ssrules 分发 token 遮罩中间件与访问日志中间件的**顺序**闸门
//
//  背景(FINN-111 交回的未覆盖项):
//
//  backend/app_http.go 的 newHTTPRouter 里:
//
//      router := gin.New()
//      router.Use(redactSSRuleTokenPath())   ← 必须排在 gin.Logger() 之前
//      router.Use(gin.Logger())
//      router.Use(gin.Recovery())
//
//  这个顺序是**承重**的: gin 的 Logger 在**进入时**就把 c.Request.URL.Path
//  抓进局部变量, c.Next() 返回之后才拿它拼日志 ⇒
//    - 遮罩排在 Logger **之前** ⇒ Logger 抓到的是遮罩后的路径,日志里没有 token;
//    - 遮罩排在 Logger **之后** ⇒ Logger 抓到的仍是原始路径,token 重新泄进 stdout。
//
//  FINN-110 / FINN-111 的用例都没有**直接**断言这个顺序:
//    - FINN-110 自己拼 router,整条链钉在 helper 上,与生产接线无关;
//    - FINN-111 走生产接线,但断言的是「日志里没有 token」这个**结果**,
//      而不是「谁排在谁前面」这个**结构** —— 顺序一旦被调换,结果同样会坏,
//      但闸门钉的是结果而不是原因。
//
//  本文件补的就是这条**结构**闸门: 直接读生产 router 的中间件链
//  (gin.Engine.Routers 里的根 RouterGroup.Handlers), 找出
//    - 「会把 /ss/<token>/... 里的 token 换成占位符」的那个中间件;
//    - 「会把访问日志写到 gin.DefaultWriter」的那个中间件;
//  然后断言**遮罩那条排在前**。
//
//  识别方式全部走**行为**(跑一遍看副作用),不靠函数名/函数指针 ——
//  中间件被改名或换实现时,闸门依然成立; 真出问题时闸门会**响**。
//
//  可证伪要求(PM 验收点):
//    - 把 newHTTPRouter 里 router.Use(redactSSRuleTokenPath()) 与
//      router.Use(gin.Logger()) 两行对调(两行都还在,只换顺序)
//      ⇒ 本文件用例必须 FAIL;
//    - 恢复原顺序 ⇒ 本文件用例必须 PASS。
// =====================================================================

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ssRuleOrderProbeToken 是探针用的假 token: 它不对应任何真实规则文件,
// 只是让探针路径「长得像」一条带分发 token 的路径。
//
// 必须是**足够独特**的字符串:「日志里没有它」这条断言只有在它不会与
// 别的字符串撞上时才有意义。
const ssRuleOrderProbeToken = "ssrule-order-probe-token-0123456789abcdef"

// ssRuleOrderProbePath 是探针请求的完整路径。
const ssRuleOrderProbePath = "/ss/" + ssRuleOrderProbeToken + "/rules.acl"

// ssRuleOrderMaskedProbePath 是探针路径被遮罩中间件改写之后的形态。
const ssRuleOrderMaskedProbePath = "/ss/" + ssRuleTokenMask + "/rules.acl"

// probeHandlerInIsolation 在隔离环境里单独跑一个中间件, 返回跑完之后
// 请求路径的形态与访问日志缓冲区的内容。
//
// 为什么可以单独跑: gin 的 Context.Next() 在 c.handlers 为空时直接返回,
// 中间件「进入时」的那段逻辑照样会执行 —— 而我们要观察的恰恰就是
// 「进入时」的那段(遮罩中间件改写路径, Logger 抓路径)。
func probeHandlerInIsolation(h gin.HandlerFunc, logBuf *bytes.Buffer) (string, string) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, ssRuleOrderProbePath, nil)

	logBuf.Reset()
	h(c)

	return c.Request.URL.Path, logBuf.String()
}

// findMaskingHandlerIndex 找出中间件链里「会把 /ss/<token>/... 的 token
// 换成占位符」的那个中间件的下标; 找不到返回 -1。
//
// 链上有两个这样的中间件 ⇒ 直接判失败: 顺序闸门没法判定谁是谁。
func findMaskingHandlerIndex(t *testing.T, chain gin.HandlersChain) int {
	t.Helper()

	var logBuf bytes.Buffer
	found := -1
	for i, h := range chain {
		path, _ := probeHandlerInIsolation(h, &logBuf)
		if path != ssRuleOrderMaskedProbePath {
			continue
		}
		if found >= 0 {
			t.Fatalf("中间件链上有两个中间件都会遮罩 /ss/ token(下标 %d 与 %d),顺序闸门无从判定", found, i)
		}
		found = i
	}
	return found
}

// findAccessLoggerIndex 找出中间件链里「会把访问日志写到 gin.DefaultWriter」
// 的那个中间件的下标; 找不到返回 -1。
//
// logBuf 必须是**构造 router 时 gin.DefaultWriter 指向的那个缓冲区**:
// gin 的 Logger 在 gin.Logger() 被调用的那一刻就把输出目标固定下来了,
// 事后换 gin.DefaultWriter 已经晚了解 —— 探针要读到日志, 就得往同一个
// 缓冲区里看。
//
// 链上有两个这样的中间件 ⇒ 直接判失败: 顺序闸门没法判定谁是谁。
func findAccessLoggerIndex(t *testing.T, chain gin.HandlersChain, logBuf *bytes.Buffer) int {
	t.Helper()

	found := -1
	for i, h := range chain {
		_, logged := probeHandlerInIsolation(h, logBuf)
		if logged == "" {
			continue
		}
		if found >= 0 {
			t.Fatalf("中间件链上有两个中间件都会写访问日志(下标 %d 与 %d),顺序闸门无从判定", found, i)
		}
		found = i
	}
	return found
}

// TestSSRuleRedactMiddlewareMountedBeforeAccessLogger 生产接线的中间件链里,
// 遮罩 /ss/ 分发 token 的中间件必须排在写访问日志的中间件**之前**。
//
// 这条用例钉的是**结构**: 直接读 newHTTPRouter 装出来的 gin.Engine 的
// 中间件链, 找出两个角色各自的下标再比大小。
//
// 为什么结构比结果更该被钉住: 「日志里没有 token」是结构正确之后的**结果**,
// 结果会随着日志格式、路由改法、遮罩规则调整一起变; 而「遮罩排在日志之前」
// 才是那个不能动的**约束**本身。
//
// 反事实: 把 backend/app_http.go 里 newHTTPRouter 中的
// `router.Use(redactSSRuleTokenPath())` 与 `router.Use(gin.Logger())`
// 两行对调 ⇒ 本用例必须 FAIL。
func TestSSRuleRedactMiddlewareMountedBeforeAccessLogger(t *testing.T) {
	var buf bytes.Buffer
	// 复用 FINN-111 的生产接线构造器: 它保证 gin.DefaultWriter 在
	// gin.Logger() 被调用**之前**就已经换成缓冲区(gin 的 Logger 在构造时
	// 就把输出目标固定下来了), 所以探针能观测到访问日志中间件。
	router, _ := newSSRuleProductionRouter(t, &buf)

	chain := router.Handlers
	maskIdx := findMaskingHandlerIndex(t, chain)
	logIdx := findAccessLoggerIndex(t, chain, &buf)

	if maskIdx < 0 {
		t.Fatalf("生产接线的中间件链里找不到会遮罩 /ss/ token 的中间件(链长度 %d)", len(chain))
	}
	if logIdx < 0 {
		t.Fatalf("生产接线的中间件链里找不到会写访问日志的中间件(链长度 %d)", len(chain))
	}
	if maskIdx >= logIdx {
		t.Fatalf("遮罩中间件必须排在访问日志中间件之前: 遮罩下标=%d, 日志下标=%d, 链长度=%d",
			maskIdx, logIdx, len(chain))
	}
}

// TestSSRuleAccessLogShowsMaskedPathOnProductionRouter 上面那条顺序闸门的
// **功能后果**: 生产接线上 /ss/<token>/... 的访问日志里必须出现遮罩后的
// 路径, 且不能出现原始 token。
//
// 这条用例与 FINN-111 的 TestSSRuleProductionRouterMasksTokenInAccessLog
// 的分工: 那条验的是「真实 token 不泄」, 这条验的是「日志里显示的**就是**
// 遮罩后的路径」—— 后者才是顺序正确时日志该有的样子, 顺序一被调换,
// 遮罩中间件仍在链上(那两条用例照样抓得住), 但日志里显示的不再是
// 遮罩后的路径, 这条会红。
//
// 用的是不对应任何规则文件的假 token: 本用例只关心日志里的**路径形态**,
// 不关心分发结果, 因此不需要真实 token、也不依赖模块开关。
func TestSSRuleAccessLogShowsMaskedPathOnProductionRouter(t *testing.T) {
	var buf bytes.Buffer
	router, _ := newSSRuleProductionRouter(t, &buf)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ssRuleOrderProbePath, nil))

	logged := buf.String()
	if logged == "" {
		t.Fatal("生产接线上没有访问日志输出,断言无从谈起")
	}
	if !strings.Contains(logged, ssRuleOrderMaskedProbePath) {
		t.Fatalf("生产接线的访问日志里没有遮罩后的路径 %q:\n%s", ssRuleOrderMaskedProbePath, logged)
	}
	if strings.Contains(logged, ssRuleOrderProbeToken) {
		t.Fatalf("生产接线的访问日志泄露了分发 token:\n%s", logged)
	}
}
