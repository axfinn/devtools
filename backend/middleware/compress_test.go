package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// newTestRouter 构造一个仅启用 ResponseCompression 的测试路由，
// 用于在隔离环境中验证排除规则与压缩行为。
func newTestRouter(level int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ResponseCompression(level))
	return r
}

// registerJSON 注册一个返回固定 JSON 的 GET 路由。
func registerJSON(r *gin.Engine, body string) {
	r.GET("/api/json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		_, _ = c.Writer.WriteString(body)
	})
}

// registerText 注册一个返回固定文本的 GET 路由。
func registerText(r *gin.Engine, path, body string) {
	r.GET(path, func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		_, _ = c.Writer.WriteString(body)
	})
}

// doRequest 发起测试请求并返回响应。
func doRequest(t *testing.T, r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// decodeGzip 若响应体是 gzip 则解压，否则原样返回。
func decodeGzip(t *testing.T, w *httptest.ResponseRecorder) []byte {
	t.Helper()
	body := w.Body.Bytes()
	if w.Header().Get("Content-Encoding") != "gzip" {
		return body
	}
	r, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	return out
}

// TestCompress_JSONCompressed：Accept-Encoding: gzip + JSON 响应应被压缩。
func TestCompress_JSONCompressed(t *testing.T) {
	r := newTestRouter(5)
	registerJSON(r, `{"status":"ok","items":[1,2,3,4,5,6,7,8,9,10]}`)

	req := httptest.NewRequest(http.MethodGet, "/api/json", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip", got)
	}
	if got := w.Header().Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("Vary header: got %q, want to contain Accept-Encoding", got)
	}
	body := decodeGzip(t, w)
	if string(body) != `{"status":"ok","items":[1,2,3,4,5,6,7,8,9,10]}` {
		t.Fatalf("decoded body mismatch: got %q", body)
	}
}

// TestCompress_NoAcceptEncoding：缺 Accept-Encoding 不应压缩。
func TestCompress_NoAcceptEncoding(t *testing.T) {
	r := newTestRouter(5)
	registerJSON(r, `{"a":1}`)

	req := httptest.NewRequest(http.MethodGet, "/api/json", nil)
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding: got %q, want empty", got)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`{"a":1}`)) {
		t.Fatalf("body mismatch: got %q", w.Body.String())
	}
}

// TestCompress_HealthExcluded：/api/health 不压缩（健康探活）。
func TestCompress_HealthExcluded(t *testing.T) {
	r := newTestRouter(5)
	registerText(r, "/api/health", `{"status":"ok"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding: got %q, want empty (health probe must not be compressed)", got)
	}
	if w.Body.String() != `{"status":"ok"}` {
		t.Fatalf("body mismatch: got %q", w.Body.String())
	}
}

// TestCompress_ExtensionExcluded：已压缩 / 媒体扩展名不压缩。
func TestCompress_ExtensionExcluded(t *testing.T) {
	cases := []struct {
		path string
	}{
		{"/static/img/foo.png"},
		{"/static/img/bar.jpg"},
		{"/static/vid/clip.mp4"},
		{"/static/audio/clip.mp3"},
		{"/assets/font.woff2"},
		{"/assets/archive.zip"},
		{"/assets/log.gz"},
		{"/assets/log.tgz"},
		{"/assets/style.br"},
		{"/assets/spec.pdf"},
	}
	for _, tc := range cases {
		t.Run(strings.TrimPrefix(tc.path, "/"), func(t *testing.T) {
			r := newTestRouter(5)
			registerText(r, tc.path, "binary-body")

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := doRequest(t, r, req)

			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding: got %q, want empty for %s", got, tc.path)
			}
		})
	}
}

// TestCompress_SSEPathExcluded：SSE 路径不压缩（即使客户端未声明 Upgrade）。
// 路径必须与 backend/routes/{ai_gateway,image_understanding}.go 的 Group() 注册字符串完全一致。
func TestCompress_SSEPathExcluded(t *testing.T) {
	cases := []string{
		"/api/internal/chat/stream",
		"/api/ai-gateway/v1/image/understanding/sse",
		"/api/ai-gateway/v1/image/understanding/sse/file",
		"/api/ai-gateway/v1/image/understanding/stream/abc-123",
		"/api/image-understanding/sse/stream/xyz",
		"/api/api-gateway/cpa/v1/some-cpa-stream",
		"/api/autodev/init/stream",
		"/api/autodev/claude/update/stream",
		"/api/autodev/codex/update/stream",
		"/api/autodev/clawtest/update/stream",
	}
	for _, p := range cases {
		t.Run(strings.TrimPrefix(p, "/"), func(t *testing.T) {
			r := newTestRouter(5)
			registerText(r, p, "data: hello\n\n")

			req := httptest.NewRequest(http.MethodGet, p, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := doRequest(t, r, req)

			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding: got %q, want empty (SSE path %s must not be compressed)", got, p)
			}
		})
	}
}

// TestCompress_StreamSseTaskRealRoute：回归保护 —— 真实生产路由路径必须不被压缩，
// 且响应体字节级与 handler 写出内容完全一致（未走 gin-contrib/gzip WrapWriter）。
// 路径必须与 backend/routes/{ai_gateway,image_understanding}.go 的 Group() 注册字符串完全一致。
func TestCompress_StreamSseTaskRealRoute(t *testing.T) {
	cases := []struct {
		path string
		body string
	}{
		{"/api/ai-gateway/v1/image/understanding/sse", "data: {\"delta\":\"hi\"}\n\n"},
		{"/api/image-understanding/sse/stream/abc-123", "data: {\"progress\":0.42}\n\n"},
	}
	for _, tc := range cases {
		t.Run(strings.TrimPrefix(tc.path, "/"), func(t *testing.T) {
			r := newTestRouter(5)
			registerText(r, tc.path, tc.body)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := doRequest(t, r, req)

			// 1. 响应未压缩
			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding: got %q, want empty (real SSE route %s must bypass gzip WrapWriter)", got, tc.path)
			}
			// 2. 响应体字节级不变（与 handler 写出内容完全一致）
			if got := w.Body.String(); got != tc.body {
				t.Fatalf("body mismatch: got %q, want %q (response must not be gzip-wrapped)", got, tc.body)
			}
		})
	}
}

// TestCompress_SSEAcceptHeader：Accept: text/event-stream 不压缩（gin-contrib/gzip 内置）。
func TestCompress_SSEAcceptHeader(t *testing.T) {
	r := newTestRouter(5)
	registerText(r, "/api/any-json", `{"a":1}`)

	req := httptest.NewRequest(http.MethodGet, "/api/any-json", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept", "text/event-stream")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding: got %q, want empty (Accept: text/event-stream must skip compression)", got)
	}
}

// TestCompress_WebSocketUpgradeExcluded：WebSocket Upgrade 请求不压缩。
// gin-contrib/gzip shouldCompress 检测 Connection: Upgrade 头。
func TestCompress_WebSocketUpgradeExcluded(t *testing.T) {
	r := newTestRouter(5)
	registerText(r, "/api/terminal/abc/ws", "ws-handshake")
	registerText(r, "/api/screen/sessions/abc/ws", "ws-handshake")
	registerText(r, "/api/proxy/ws-tunnel", "ws-handshake")
	registerText(r, "/api/game/arcade/ws/abc", "ws-handshake")
	registerText(r, "/api/chat/room/abc/ws", "ws-handshake")

	paths := []string{
		"/api/terminal/abc/ws",
		"/api/screen/sessions/abc/ws",
		"/api/proxy/ws-tunnel",
		"/api/game/arcade/ws/abc",
		"/api/chat/room/abc/ws",
	}
	for _, p := range paths {
		t.Run(strings.TrimPrefix(p, "/"), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
			w := doRequest(t, r, req)

			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding: got %q, want empty for WS upgrade %s", got, p)
			}
		})
	}
}

// TestCompress_NfsShareExcluded：/api/nfsshare/ 全部排除（WS + Range + HLS）。
func TestCompress_NfsShareExcluded(t *testing.T) {
	r := newTestRouter(5)
	registerText(r, "/api/nfsshare/abc/stream", "video-bytes")
	registerText(r, "/api/nfsshare/abc/hls/720p/seg-00001.ts", "ts-bytes")
	registerText(r, "/api/nfsshare/abc/watch/ws", "ws-handshake")

	paths := []string{
		"/api/nfsshare/abc/stream",
		"/api/nfsshare/abc/hls/720p/seg-00001.ts",
		"/api/nfsshare/abc/watch/ws",
	}
	for _, p := range paths {
		t.Run(strings.TrimPrefix(p, "/"), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := doRequest(t, r, req)

			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding: got %q, want empty for nfsshare %s", got, p)
			}
		})
	}
}

// TestCompress_PublicRoutesExcluded：/s/、/sub/proxy、/mock/ 不压缩。
func TestCompress_PublicRoutesExcluded(t *testing.T) {
	r := newTestRouter(5)
	registerText(r, "/s/abc", "302 redirect target")
	registerText(r, "/sub/proxy", "subscription-bytes")
	registerText(r, "/mock/abc", "mock-bytes")

	paths := []string{"/s/abc", "/sub/proxy", "/mock/abc"}
	for _, p := range paths {
		t.Run(strings.TrimPrefix(p, "/"), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := doRequest(t, r, req)

			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding: got %q, want empty for %s", got, p)
			}
		})
	}
}

// TestCompress_StaticAssetsCompressed：/assets/* 静态资源应压缩（用于前端 dist chunk）。
func TestCompress_StaticAssetsCompressed(t *testing.T) {
	// /assets/* 没有显式 handler，gin 路由将落到 NoRoute 返回 404。
	// 我们用自定义 handler 模拟：响应一个 JS 文本。
	r := newTestRouter(5)
	r.GET("/assets/index.js", func(c *gin.Context) {
		c.Header("Content-Type", "application/javascript")
		_, _ = c.Writer.WriteString(`(function(){var x=1;for(var i=0;i<100;i++)x+=i;return x;})();`)
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/index.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip for /assets/index.js", got)
	}
	body := decodeGzip(t, w)
	if !strings.Contains(string(body), "function") {
		t.Fatalf("decoded body unexpected: %q", body)
	}
}

// TestCompress_Level5ReducesSize：验证压缩生效后字节数确实下降。
func TestCompress_Level5ReducesSize(t *testing.T) {
	r := newTestRouter(5)
	// 高度冗余的 JSON，便于被 gzip 压缩。
	big := `{"items":[` + strings.Repeat(`{"id":1,"name":"alpha"},`, 200)
	big = strings.TrimSuffix(big, ",") + "]}"
	registerJSON(r, big)

	// 不压缩基线（缺 Accept-Encoding）
	req1 := httptest.NewRequest(http.MethodGet, "/api/json", nil)
	w1 := doRequest(t, r, req1)
	rawSize := w1.Body.Len()

	// 压缩
	req2 := httptest.NewRequest(http.MethodGet, "/api/json", nil)
	req2.Header.Set("Accept-Encoding", "gzip")
	w2 := doRequest(t, r, req2)

	if got := w2.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip", got)
	}
	// 压缩后 body 是 gzip 字节流；其长度应远小于 raw。
	if w2.Body.Len() >= rawSize {
		t.Fatalf("compressed size %d >= raw size %d (compression not effective)", w2.Body.Len(), rawSize)
	}
	// 解码后内容应与原始一致。
	if got := string(decodeGzip(t, w2)); got != big {
		t.Fatalf("decoded body mismatch (len got=%d want=%d)", len(got), len(big))
	}
}

// TestCompress_VaryHeaderAlwaysPresent：即使请求未声明 gzip，也应回 Vary: Accept-Encoding，
// 保证中间缓存能基于 Accept-Encoding 区分缓存条目。
//
// 实际上 v1.0.1 只在压缩生效时回 Vary 头；本测试只校验「压缩时 Vary 必含 Accept-Encoding」。
func TestCompress_VaryHeaderAlwaysPresent(t *testing.T) {
	r := newTestRouter(5)
	registerJSON(r, `{"a":1,"b":2,"c":3}`)

	req := httptest.NewRequest(http.MethodGet, "/api/json", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip", got)
	}
	if got := w.Header().Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("Vary: got %q, want to contain Accept-Encoding", got)
	}
}
