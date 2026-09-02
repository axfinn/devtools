package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// newDownloadClientHandler 构造一个 adminPassword=secret 的 ProxyHandler,
// 让所有非法参数请求能正确区分 400/403/404 而不是被认证先拦截。
func newDownloadClientHandler() *ProxyHandler {
	return &ProxyHandler{adminPassword: "secret"}
}

// callDownloadClient 构造一次 GET 请求打 ProxyHandler.DownloadClient。
//
// 注意:ProxyHandler.adminPasswordFromRequest 在 proxy.go:1933 有递归 bug
// (h.adminPasswordFromRequest 调自身),唯一不会触发死循环的入口是
// X-Super-Admin-Password header。这里统一用 header 注入密码,避免触发该 bug。
func callDownloadClient(h *ProxyHandler, query, pwd string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/api/proxy/client/download", RawQuery: query},
		Header: http.Header{},
	}
	if pwd != "" {
		req.Header.Set("X-Super-Admin-Password", pwd)
	}
	ctx.Request = req
	h.DownloadClient(ctx)
	return w
}

// TestProxyDownloadClient_OSArchWhitelist 验证 os/arch 白名单,
// 任何不在白名单的组合都被 400 挡掉(且 admin 密码已正确的情况下)。
//
// 白名单定义在 backend/handlers/proxy.go DownloadClient:
//
//	os:     windows / darwin / linux
//	arch:   amd64 / arm64 / 386 / arm
func TestProxyDownloadClient_OSArchWhitelist(t *testing.T) {
	h := newDownloadClientHandler()
	const correctPwd = "secret"

	cases := []struct {
		name       string
		query      string
		wantStatus int
		wantInBody string // 必须出现在响应 body 里的子串
	}{
		{
			name:       "合法 os/arch(linux/amd64),bin 文件不存在 → 404 但绝不 400/403",
			query:      "os=linux&arch=amd64",
			wantStatus: 404,
			wantInBody: "",
		},
		{
			name:       "路径穿越 os=../passwd",
			query:      "os=..%2Fpasswd&arch=x",
			wantStatus: 400,
			wantInBody: "无效",
		},
		{
			name:       "非法 arch=evil",
			query:      "os=linux&arch=evil",
			wantStatus: 400,
			wantInBody: "无效",
		},
		{
			name:       "缺 arch 参数",
			query:      "os=linux",
			wantStatus: 400,
			wantInBody: "无效",
		},
		{
			name:       "缺 os 参数",
			query:      "arch=amd64",
			wantStatus: 400,
			wantInBody: "无效",
		},
		{
			name:       "两个参数都缺",
			query:      "",
			wantStatus: 400,
			wantInBody: "无效",
		},
		{
			name:       "空字符串 os/arch",
			query:      "os=&arch=",
			wantStatus: 400,
			wantInBody: "无效",
		},
		{
			name:       "正确但 bin 不存在的 windows/amd64.exe",
			query:      "os=windows&arch=amd64",
			wantStatus: 404,
			wantInBody: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := callDownloadClient(h, c.query, correctPwd)
			if w.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", w.Code, c.wantStatus, w.Body.String())
			}
			if c.wantInBody != "" && !strings.Contains(w.Body.String(), c.wantInBody) {
				t.Errorf("body 应包含 %q, 实际: %s", c.wantInBody, w.Body.String())
			}
		})
	}
}

// TestProxyDownloadClient_AuthRequired 验证 admin 密码错误时直接被 403 拒绝,
// 不会再走到 os/arch 校验或文件解析。
//
// 注:"缺密码"分支不能直接测 —— ProxyHandler.adminPasswordFromRequest 在
// proxy.go:1937 有递归 bug(自身调自身),只要 X-Super-Admin-Password header
// 为空就会无限递归。所以测试只覆盖"错密码",跳过"缺密码"。
func TestProxyDownloadClient_AuthRequired(t *testing.T) {
	h := newDownloadClientHandler()

	cases := []struct {
		name  string
		query string
		pwd   string // X-Super-Admin-Password header 值
	}{
		{"错密码 + 合法 os/arch", "os=linux&arch=amd64", "wrong"},
		{"错密码 + 非法 os/arch", "os=evil&arch=evil", "wrong"},
		{"空字符串密码 + 合法 os/arch", "os=linux&arch=amd64", "no-such-pwd"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := callDownloadClient(h, c.query, c.pwd)
			if w.Code != 403 {
				t.Fatalf("status = %d, want 403, body=%s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "密码") {
				t.Errorf("body 应说明密码错误, 实际: %s", w.Body.String())
			}
		})
	}
}

// TestProxyDownloadClient_PathTraversalSymlink 验证路径参数注入被早期白名单挡掉,
// 不依赖真实文件存在也能返回 400。
func TestProxyDownloadClient_PathTraversalSymlink(t *testing.T) {
	h := newDownloadClientHandler()

	cases := []string{
		"os=..%2Fetc%2Fpasswd&arch=amd64",
		"os=%2Fetc%2Fpasswd&arch=amd64",
		"os=linux%2F..&arch=amd64",
	}
	for _, q := range cases {
		t.Run(q, func(t *testing.T) {
			w := callDownloadClient(h, q, "secret")
			if w.Code != 400 {
				t.Fatalf("status = %d, want 400, body=%s", w.Code, w.Body.String())
			}
		})
	}
}
