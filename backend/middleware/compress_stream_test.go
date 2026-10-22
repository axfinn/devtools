package middleware

// FINN-52 回归测试：压缩中间件不得缓冲/吞掉流式响应。
//
// 背景：FINN-41/FINN-49 的 captureWriter 实现把 handler 的全部响应字节先攒进
// bytes.Buffer，且 Flush() 是空实现。后果是主力 LLM 流式端点（Claude Code 走的
// /api/anthropic/v1/messages 等）首字节延迟 = 整段回答生成时间，且为压住
// Cloudflare 空闲超时而发的 ping 心跳一个字节都到不了客户端。
//
// 本文件断言的不变量（与既有 compress_test.go 的「压不压」正交，这里管「缓不缓」）：
//   I1. Content-Type 为 text/event-stream 的响应不压缩 —— 无论请求侧 Accept 头怎么写。
//   I2. handler 仍在执行时，已 Flush 的字节必须已在 writer 上（证明中间件无缓冲）。
//   I3. 同一条路径上的非流式 JSON 响应仍被压缩（不得为了 SSE 把整条路径排除）。
//
// 严格只追加 _test.go；生产代码改动见 compress.go 的 FINN-52 段落。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const ssePingFrameForTest = "event: ping\ndata: {\"type\":\"ping\"}\n\n"

const sseStreamTailForTest = "data: {\"delta\":\"hi\"}\n\n"

// sseProbe 记录 handler 执行期间 writer 上的可见状态，供「未缓冲」断言使用。
type sseProbe struct {
	bodyAfterFirstFlush string
	flusherMissing      bool
}

// newSSERoute 注册一个「先发 ping 事件并 Flush，再发正文事件」的 SSE handler。
//
// 两个事件之间对 recorder 取一次快照 —— 这正是区分「流式」与「全量缓冲」的观测点：
// 若中间件缓冲了整个响应，快照必为空串。
//
// recorder 由调用方构造并同时传进来，handler 才能在执行途中读到尚未返回的响应体。
func newSSERoute(t *testing.T, r *gin.Engine, method, path string, probe *sseProbe, recorder *httptest.ResponseRecorder) {
	t.Helper()
	handler := func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(http.StatusOK)

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			probe.flusherMissing = true
			return
		}

		// 网关用来压住 CF/nginx 空闲超时的 ping 帧。
		_, _ = c.Writer.Write([]byte(ssePingFrameForTest))
		flusher.Flush()
		probe.bodyAfterFirstFlush = recorder.Body.String()

		_, _ = c.Writer.Write([]byte(sseStreamTailForTest))
		flusher.Flush()
	}

	if method == http.MethodPost {
		r.POST(path, handler)
		return
	}
	r.GET(path, handler)
}

// serveWithRecorder 用显式构造的 recorder 跑请求。
func serveWithRecorder(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// assertSSENotBuffered 校验 I1 + I2 两条不变量。
func assertSSENotBuffered(t *testing.T, path string, w *httptest.ResponseRecorder, probe *sseProbe) {
	t.Helper()

	if probe.flusherMissing {
		t.Fatal("c.Writer 不满足 http.Flusher：SSE 无法流式送达")
	}
	// I1：SSE 响应不得被 gzip —— gzip.Writer 的缓冲会吞掉 Flush。
	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding: got %q, want empty for %s (SSE must not be gzip-wrapped)", got, path)
	}
	// I2：handler 尚未返回时，首个事件必须已经落到 writer 上。
	if !strings.Contains(probe.bodyAfterFirstFlush, "event: ping") {
		t.Fatalf("首字节被缓冲（%s）：handler 内 Flush 后 recorder 仍为 %q。\n"+
			"后果：首字节延迟退化成整段生成时间，且 CF/nginx 仍按空闲超时 524。",
			path, probe.bodyAfterFirstFlush)
	}
	// 观测点自检：第二个事件此刻不应已经可见，否则说明两次写入之间没有被区分开。
	if strings.Contains(probe.bodyAfterFirstFlush, `"delta":"hi"`) {
		t.Fatalf("观测点失效：第二个事件在快照前就已可见 (%q)", probe.bodyAfterFirstFlush)
	}
	// 字节级一致：中间件不得改写流内容。
	if want := ssePingFrameForTest + sseStreamTailForTest; w.Body.String() != want {
		t.Fatalf("body mismatch:\n got %q\nwant %q", w.Body.String(), want)
	}
}

// TestCompress_SSEStreamNotBuffered_RealAnthropicRoute 是 FINN-52 的核心回归点。
//
// 用真实生产路径 /api/anthropic/v1/messages（Claude Code / Anthropic SDK 的入口），
// 且请求侧**刻意只带 Accept: */* + Accept-Encoding: gzip** —— 这是 fetch / httpx /
// openai SDK 的默认形态，请求侧白名单里的 "Accept: text/event-stream" 协商在这种
// 客户端上根本不成立。断言响应侧判定能独立兜住。
func TestCompress_SSEStreamNotBuffered_RealAnthropicRoute(t *testing.T) {
	const path = "/api/anthropic/v1/messages"
	probe := &sseProbe{}

	r := newTestRouter(5)
	w := httptest.NewRecorder()
	newSSERoute(t, r, http.MethodPost, path, probe, w)

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"claude-sonnet-4-6","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept", "*/*") // 刻意不声明 text/event-stream
	r.ServeHTTP(w, req)

	assertSSENotBuffered(t, path, w, probe)
}

// TestCompress_SSEStreamNotBuffered_RealQwenVisionRoute 覆盖另一条**不在请求侧白名单里**
// 的 SSE 生产路径（/api/image-understanding/qwen-vision）。
// 它证明修复不依赖手工维护的路径列表 —— 响应侧 Content-Type 判定独立生效。
func TestCompress_SSEStreamNotBuffered_RealQwenVisionRoute(t *testing.T) {
	const path = "/api/image-understanding/qwen-vision"
	probe := &sseProbe{}

	r := newTestRouter(5)
	w := httptest.NewRecorder()
	newSSERoute(t, r, http.MethodPost, path, probe, w)

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"image":"data:image/png;base64,AAAA"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w, req)

	assertSSENotBuffered(t, path, w, probe)
}

// TestCompress_SSEWithAcceptHeaderStillUnbuffered：请求侧已声明 text/event-stream 时
// 走快路径（完全不包装 writer），同样必须不缓冲。
func TestCompress_SSEWithAcceptHeaderStillUnbuffered(t *testing.T) {
	const path = "/api/internal/chat/stream"
	probe := &sseProbe{}

	r := newTestRouter(5)
	w := httptest.NewRecorder()
	newSSERoute(t, r, http.MethodPost, path, probe, w)

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept", "text/event-stream")
	r.ServeHTTP(w, req)

	assertSSENotBuffered(t, path, w, probe)
}

// TestCompress_NonStreamAnthropicJSONStillCompressed 是 I3：
// 同一条 /api/anthropic/v1/messages 上的非流式大 JSON 响应**必须仍被压缩**。
//
// 这条测试锁住一个设计决定：不要把 LLM 路径整体加进 excludedPathPrefixes ——
// 那些端点同时承载流式(SSE)与非流式(大 JSON)两种响应，按路径排除会连带丢掉
// 非流式长推理响应（Claude Code 的 subagent 上下文可达数百 KB）的压缩收益。
func TestCompress_NonStreamAnthropicJSONStillCompressed(t *testing.T) {
	const path = "/api/anthropic/v1/messages"
	body := `{"content":[{"type":"text","text":"` +
		strings.Repeat("code line with padding; ", 200) +
		`"}],"usage":{"input_tokens":10,"output_tokens":20}}`

	r := newTestRouter(5)
	r.POST(path, func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		_, _ = c.Writer.WriteString(body)
	})

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"stream":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	w := serveWithRecorder(r, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip (非流式 JSON 必须保留压缩收益)", got)
	}
	if got := string(decodeGzip(t, w)); got != body {
		t.Fatalf("decoded body mismatch: len got=%d want=%d", len(got), len(body))
	}
}

// TestCompress_FlushAdvancesCompressedStream：压缩路径上的 Flush 也必须是真操作。
// gzip.Writer 自带缓冲，若中间件的 Flush 被吞，压缩大响应同样会「憋住」。
func TestCompress_FlushAdvancesCompressedStream(t *testing.T) {
	r := newTestRouter(5)

	// 高度可压缩的文本，单块足以让 gzip.Flush() 吐出非空输出。
	chunk := strings.Repeat("var padding = 1;\n", 8192)
	recorder := httptest.NewRecorder()
	var bytesAfterFlush int

	r.GET("/assets/big.js", func(c *gin.Context) {
		c.Header("Content-Type", "application/javascript")
		_, _ = c.Writer.WriteString(chunk)
		c.Writer.Flush()
		// 此处 writer 上的字节 = gzip 已吐出的部分；Flush 被吞时为 0。
		bytesAfterFlush = recorder.Body.Len()
		_, _ = c.Writer.WriteString(chunk)
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/big.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip", got)
	}
	if bytesAfterFlush == 0 {
		t.Fatal("压缩路径的 Flush 没有产出任何字节：gzip.Writer 缓冲被吞，流式压缩响应会停滞")
	}
	// 完整流解码后必须与原始内容一致（Flush 不得破坏 gzip 流）。
	if got := string(decodeGzip(t, recorder)); got != chunk+chunk {
		t.Fatalf("decoded body mismatch: len got=%d want=%d", len(got), len(chunk)*2)
	}
}

// TestCompress_NoContentNotCompressed：204 无 body，不应挂 Content-Encoding。
func TestCompress_NoContentNotCompressed(t *testing.T) {
	r := newTestRouter(5)
	r.DELETE("/api/thing/1", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/thing/1", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := serveWithRecorder(r, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", w.Code)
	}
	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding: got %q, want empty for 204", got)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("204 must have empty body, got len=%d", w.Body.Len())
	}
}
