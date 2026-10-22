package middleware

// FINN-108 / N6 回归测试：压缩中间件不得覆盖 handler 已声明的 Vary 维度。
//
// decide() 原本用 header.Set("Vary", "Accept-Encoding")，会把 handler 声明的
// Vary（例：Vary: Origin）整个覆盖掉 —— 上游已声明的缓存维度被抹掉。
// 修复后改为 appendVary：已存在则跳过，不存在才追加。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// varyValue 把响应的 Vary 头拼成单个逗号分隔字符串。
//
// Vary 是列表型头，handler 声明的值与中间件追加的值可能分成多行下发，
// 而 Header.Get 只返回第一行 —— 所以这里用 Values 取回全部再拼接。
func varyValue(w *httptest.ResponseRecorder) string {
	return strings.Join(w.Header().Values("Vary"), ", ")
}

// registerVaryJSON 注册一个先声明 Vary、再返回 JSON 的 GET 路由。
func registerVaryJSON(r *gin.Engine, path, vary string) {
	r.GET(path, func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("Vary", vary)
		_, _ = c.Writer.WriteString(`{"status":"ok"}`)
	})
}

// TestCompress_VaryKeepsHandlerDimension：handler 已声明 Vary: Origin 时，
// 压缩后 Vary 必须同时含 Origin 与 Accept-Encoding。
func TestCompress_VaryKeepsHandlerDimension(t *testing.T) {
	r := newTestRouter(5)
	registerVaryJSON(r, "/api/vary-origin", "Origin")

	req := httptest.NewRequest(http.MethodGet, "/api/vary-origin", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip", got)
	}
	got := varyValue(w)
	if !strings.Contains(got, "Origin") {
		t.Fatalf("Vary: got %q, want to keep handler's Origin", got)
	}
	if !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("Vary: got %q, want to contain Accept-Encoding", got)
	}
}

// TestCompress_VaryNoDuplicate：handler 已声明 Vary: Accept-Encoding 时，
// 压缩后 Vary 不得出现重复值（Accept-Encoding 只能出现一次）。
func TestCompress_VaryNoDuplicate(t *testing.T) {
	r := newTestRouter(5)
	registerVaryJSON(r, "/api/vary-accept-encoding", "Accept-Encoding")

	req := httptest.NewRequest(http.MethodGet, "/api/vary-accept-encoding", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	got := varyValue(w)
	if n := strings.Count(got, "Accept-Encoding"); n != 1 {
		t.Fatalf("Vary: got %q, want exactly one Accept-Encoding, got %d", got, n)
	}
}

// TestCompress_VaryTokenExactMatch：Vary 判据必须是逗号分隔的 token 精确比较，
// 不能用子串包含 —— X-Accept-Encoding 不是 Accept-Encoding，两者都要保留。
func TestCompress_VaryTokenExactMatch(t *testing.T) {
	r := newTestRouter(5)
	registerVaryJSON(r, "/api/vary-prefix", "X-Accept-Encoding")

	req := httptest.NewRequest(http.MethodGet, "/api/vary-prefix", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	got := varyValue(w)
	if !strings.Contains(got, "X-Accept-Encoding") {
		t.Fatalf("Vary: got %q, want to keep handler's X-Accept-Encoding", got)
	}
	if !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("Vary: got %q, want to contain Accept-Encoding (prefix must not be mistaken for it)", got)
	}
}

// TestCompress_VaryCommaSeparated：handler 声明的是逗号分隔的列表时，
// 也要按 token 判重 —— `Origin, Accept-Encoding` 里已含 Accept-Encoding。
func TestCompress_VaryCommaSeparated(t *testing.T) {
	r := newTestRouter(5)
	registerVaryJSON(r, "/api/vary-list", "Origin, Accept-Encoding")

	req := httptest.NewRequest(http.MethodGet, "/api/vary-list", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	got := varyValue(w)
	if n := strings.Count(got, "Accept-Encoding"); n != 1 {
		t.Fatalf("Vary: got %q, want exactly one Accept-Encoding, got %d", got, n)
	}
	if !strings.Contains(got, "Origin") {
		t.Fatalf("Vary: got %q, want to keep handler's Origin", got)
	}
}
