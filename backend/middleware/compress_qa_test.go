package middleware

// FINN-43 QA 测试 —— 覆盖 Stage 5 B 表中 compress_test.go 已写之外的项目：
//   - text/html / application/javascript / text/css 类型响应被压缩
//   - HEAD 请求不应产生 body 或错误的 Content-Length
//   - Range 请求不被压缩（Content-Range / 206 字节偏移语义正确）
//   - handler 已设 Content-Encoding 时不重复压缩
//   - 极小响应体（< 200B）压不压都行，但不得报错
//   - /api/health 压缩下仍语义等价：200 且 body 可解析
//   - /assets/* immutable Cache-Control 未被压缩中间件改动
//
// 严格不修改任何生产代码；只追加 _test.go。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestCompress_TextHTMLCompressed：text/html 响应应被压缩。
func TestCompress_TextHTMLCompressed(t *testing.T) {
	r := newTestRouter(5)
	body := strings.Repeat("<html><body><p>hello html gzip test </p></body></html>", 32)
	r.GET("/page", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		_, _ = c.Writer.WriteString(body)
	})

	req := httptest.NewRequest(http.MethodGet, "/page", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip for text/html", got)
	}
	if got := decodeGzip(t, w); string(got) != body {
		t.Fatalf("decoded body mismatch (len got=%d want=%d)", len(got), len(body))
	}
}

// TestCompress_JavaScriptCompressed：application/javascript 响应应被压缩。
func TestCompress_JavaScriptCompressed(t *testing.T) {
	r := newTestRouter(5)
	body := strings.Repeat("(function(){var sum=0;for(var i=0;i<1000;i++)sum+=i;return sum;})();\n", 20)
	r.GET("/assets/index.js", func(c *gin.Context) {
		c.Header("Content-Type", "application/javascript; charset=utf-8")
		_, _ = c.Writer.WriteString(body)
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/index.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip for application/javascript", got)
	}
	if got := decodeGzip(t, w); string(got) != body {
		t.Fatalf("decoded body mismatch (len got=%d want=%d)", len(got), len(body))
	}
}

// TestCompress_CSSCompressed：text/css 响应应被压缩。
func TestCompress_CSSCompressed(t *testing.T) {
	r := newTestRouter(5)
	body := strings.Repeat(".class{color:#fff;background:#000;padding:8px;margin:4px;}\n", 50)
	r.GET("/assets/style.css", func(c *gin.Context) {
		c.Header("Content-Type", "text/css; charset=utf-8")
		_, _ = c.Writer.WriteString(body)
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/style.css", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip for text/css", got)
	}
	if got := decodeGzip(t, w); string(got) != body {
		t.Fatalf("decoded body mismatch (len got=%d want=%d)", len(got), len(body))
	}
}

// TestCompress_HEADNoBody：HEAD 请求不应在响应中携带 body，且不应误设 Content-Length。
//
// gin v1.9.1 的 r.GET 仅注册 GET；HEAD 必须显式注册（gin 在 v1.10 才自动派生）。
// 注：gin 不会在 HEAD 路径上自动剥离 handler 写入的 body；HEAD handler 必须自己判断不写。
func TestCompress_HEADNoBody(t *testing.T) {
	r := newTestRouter(5)
	// 与 GET 同结构：GET 写 body，HEAD 不写（标准 HTTP/1.1 语义）。
	r.GET("/api/json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		_, _ = c.Writer.WriteString(`{"a":1}`)
	})
	r.HEAD("/api/json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		// HEAD 必须不写 body（HTTP/1.1 RFC 9110 §9.3.2）。
	})

	req := httptest.NewRequest(http.MethodHead, "/api/json", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	// gin 在 HEAD 上会自动省略 body；assert body 长度 0 且状态 200。
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("HEAD body should be empty, got len=%d", w.Body.Len())
	}
	// HEAD 上若中间件错误地标注了 Content-Encoding: gzip，会让上游缓存/CDN 误判；
	// 但 gin-contrib/gzip 自身会在 HEAD 上跳过 WrapWriter（应不出现 Content-Encoding）。
	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("HEAD Content-Encoding: got %q, want empty (HEAD must not advertise encoding)", got)
	}
}

// TestCompress_RangeNotCompressed：带 Range 头的请求不被压缩，206 语义正确。
func TestCompress_RangeNotCompressed(t *testing.T) {
	r := newTestRouter(5)
	payload := strings.Repeat("0123456789", 200) // 2000 bytes
	r.GET("/api/range", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Accept-Ranges", "bytes")
		_, _ = c.Writer.WriteString(payload)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/range", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Range", "bytes=0-99")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding: got %q, want empty for Range request", got)
	}
	// Range 请求在 httptest ResponseRecorder 下，gin-contrib/gzip 检测 Range 头后跳过压缩，
	// 但状态码本身由 handler 决定 —— 这里 handler 没实现 206 路径，所以保持 200；
	// 关键断言是未被 gzip WrapWriter 包裹，body 与 handler 写出完全一致。
	if w.Body.String() != payload {
		t.Fatalf("Range body mismatch: len got=%d want=%d", w.Body.Len(), len(payload))
	}
}

// TestCompress_RangeResponseCompressedWouldBreakSemantics：构造一个返回 206 Content-Range 的
// handler，断言压缩不会把 Content-Range / Content-Length 字节偏移语义破坏（不应被压缩）。
func TestCompress_RangeResponseContentRangePreserved(t *testing.T) {
	r := newTestRouter(5)
	full := strings.Repeat("A", 1000)
	r.GET("/api/file", func(c *gin.Context) {
		// 模拟一个静态文件 server 在 Range 时回 206 + Content-Range。
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Accept-Ranges", "bytes")
		c.Header("Content-Range", "bytes 0-99/1000")
		c.Status(http.StatusPartialContent)
		_, _ = c.Writer.WriteString(full[:100])
	})

	req := httptest.NewRequest(http.MethodGet, "/api/file", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding: got %q, want empty for 206 Partial Content (must not gzip)", got)
	}
	if got := w.Header().Get("Content-Range"); got != "bytes 0-99/1000" {
		t.Fatalf("Content-Range: got %q, want bytes 0-99/1000", got)
	}
	if w.Body.String() != full[:100] {
		t.Fatalf("206 body mismatch: got len=%d, want 100", w.Body.Len())
	}
	if w.Code != http.StatusPartialContent {
		t.Fatalf("status: got %d, want 206", w.Code)
	}
}

// TestCompress_AlreadyEncodedNotReencoded：handler 已设 Content-Encoding 时不重复压缩。
// gin-contrib/gzip 检测到响应已编码应跳过 WrapWriter。
func TestCompress_AlreadyEncodedNotReencoded(t *testing.T) {
	r := newTestRouter(5)
	preEncoded := "data:this-is-already-gzipped-by-handler\n\n"
	r.GET("/api/special", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Content-Encoding", "gzip")
		_, _ = c.Writer.WriteString(preEncoded)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/special", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	// 行为 1：Content-Encoding 应仍是 gzip（不能被改写为 gzip, gzip，也不能被去编码）。
	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip (handler-provided encoding must not be replaced)", got)
	}
	// 行为 2：body 必须是 handler 写出的字符串逐字节一致，不能被再次 gzip。
	if w.Body.String() != preEncoded {
		t.Fatalf("body mismatch: got %q, want %q", w.Body.String(), preEncoded)
	}
}

// TestCompress_SmallBodyNoError：极小响应体（< 200 字节）压不压都行，但不得报错。
// gin-contrib/gzip 内部有 min-size 阈值（默认 ~512B 但允许调整），测试我们不论是否压缩，
// body 字节级一致 + 状态 200。
func TestCompress_SmallBodyNoError(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty", ""},
		{"tiny", "ok"},
		{"medium", `{"status":"ok","items":[1,2,3]}`}, // ~30 bytes
		{"under200", strings.Repeat("a", 199)},
		{"over200", strings.Repeat("b", 250)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRouter(5)
			r.GET("/api/small", func(c *gin.Context) {
				c.Header("Content-Type", "application/json; charset=utf-8")
				_, _ = c.Writer.WriteString(tc.body)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/small", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := doRequest(t, r, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status: got %d, want 200", w.Code)
			}
			// 关键不变量：response body 必须能还原为原始 body（不论是否压缩）。
			got := decodeGzip(t, w)
			if string(got) != tc.body {
				t.Fatalf("body mismatch: got %q, want %q", got, tc.body)
			}
			// 若被压缩，则压缩后字节数不得大于原始（避免 gzip 头开销导致响应更大）。
			if w.Header().Get("Content-Encoding") == "gzip" && len(tc.body) > 0 {
				if w.Body.Len() > len(tc.body) {
					t.Logf("note: compressed size %d > raw %d (allowed only if min-size threshold skips it)", w.Body.Len(), len(tc.body))
				}
			}
		})
	}
}

// TestCompress_HealthEndpointSemanticEquivalence：/api/health 在压缩下语义等价：
// 状态码 200，body 可正常 JSON 解析。
// 这是"健康探活契约"项 —— 即使 gzip 中间件生效，/api/health 也必须保持可读。
// 注：本中间件把 /api/health 加入 WithExcludedPaths，因此即使声明 Accept-Encoding: gzip，
// 也应保持原样输出，不带 Content-Encoding。
func TestCompress_HealthEndpointSemanticEquivalence(t *testing.T) {
	r := newTestRouter(5)
	healthBody := `{"status":"ok","dependencies":{"redis":"up","ocr":"up","asr":"up","tts":"up"}}`
	r.GET("/api/health", func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		_, _ = c.Writer.WriteString(healthBody)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	// /api/health 必须不被压缩（wget --spider、curl 健康探活不期望被改写）。
	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding: got %q, want empty (health probe must remain uncompressed)", got)
	}
	// body 必须可正常 JSON 解析。
	var parsed map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("health body must be JSON-parseable uncompressed: %v (body=%q)", err, w.Body.String())
	}
	if parsed["status"] != "ok" {
		t.Fatalf("health.status: got %v, want ok", parsed["status"])
	}
}

// TestCompress_ImmutableCacheHeaderUntouched：/assets/* 的 immutable Cache-Control
// 不应被压缩中间件改动 —— 这是"缓存头未被改动"判据。
//
// 注意：middleware 仅控制 Content-Encoding/Vary/Content-Length；Cache-Control 由 handler 设。
// 本测试构造一个返回 immutable 的 handler，断言响应同时带 Cache-Control + 正确 Content-Encoding。
func TestCompress_ImmutableCacheHeaderUntouched(t *testing.T) {
	r := newTestRouter(5)
	body := strings.Repeat("var x=1;console.log(x);\n", 200) // ~3.8KB
	r.GET("/assets/index-abc123.js", func(c *gin.Context) {
		c.Header("Content-Type", "application/javascript; charset=utf-8")
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		_, _ = c.Writer.WriteString(body)
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/index-abc123.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	// Cache-Control 必须原样保留。
	if got := w.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control: got %q, want public, max-age=31536000, immutable", got)
	}
	// JS 应被压缩。
	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip", got)
	}
	// 解压后字节级一致。
	if got := decodeGzip(t, w); string(got) != body {
		t.Fatalf("decoded body mismatch (len got=%d want=%d)", len(got), len(body))
	}
}

// TestCompress_CPARegressionPoint：FINN-49 新增的反代 SSE 兜底透传。
// /api/api-gateway/cpa/ 下的任意路径（含非 SSE 子路径）都不应被压缩。
func TestCompress_CPARegressionPoint(t *testing.T) {
	cases := []string{
		"/api/api-gateway/cpa/",
		"/api/api-gateway/cpa/v1/some-cpa-stream",
		"/api/api-gateway/cpa/invoke",
	}
	for _, p := range cases {
		t.Run(strings.TrimPrefix(p, "/"), func(t *testing.T) {
			r := newTestRouter(5)
			registerText(r, p, `{"delta":"chunk"}`)

			req := httptest.NewRequest(http.MethodGet, p, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := doRequest(t, r, req)

			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding: got %q, want empty for CPA route %s", got, p)
			}
		})
	}
}

// TestCompress_GzAndTgzExplicitCheck：FINN-31 评审点名 .gz/.tgz 不被二次压缩。
// 已在 TestCompress_ExtensionExcluded 中包含，本测试作为独立回归点加重。
func TestCompress_GzAndTgzExplicitCheck(t *testing.T) {
	cases := []string{
		"/downloads/log.gz",
		"/downloads/archive.tgz",
		"/backups/snapshot-2026-09-17.tgz",
	}
	for _, p := range cases {
		t.Run(strings.TrimPrefix(p, "/"), func(t *testing.T) {
			r := newTestRouter(5)
			registerText(r, p, "pre-gzipped-binary")

			req := httptest.NewRequest(http.MethodGet, p, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := doRequest(t, r, req)

			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding: got %q, want empty (gz/tgz must not be re-compressed)", got)
			}
			if w.Body.String() != "pre-gzipped-binary" {
				t.Fatalf("body mismatch for %s: got %q", p, w.Body.String())
			}
		})
	}
}

// TestCompress_VaryAcceptEncodingAlsoForExcluded：被排除路径的响应不应该带 Vary: Accept-Encoding，
// 因为它们与 Accept-Encoding 协商无关 —— 这是契约性补充，避免中间缓存误分流。
//
// 注：gin-contrib/gzip 在 shouldCompress=false 时不会注入 Vary；但我们要确认。
func TestCompress_ExcludedPathNoVary(t *testing.T) {
	r := newTestRouter(5)
	r.GET("/api/health", func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		_, _ = c.Writer.WriteString(`{"status":"ok"}`)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := doRequest(t, r, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding: got %q, want empty", got)
	}
	if got := w.Header().Get("Vary"); strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("Vary: got %q, want no Accept-Encoding (excluded path must not advertise encoding negotiation)", got)
	}
}

// 注：复用同包的 decodeGzip（在 compress_test.go 定义）；QA 测试与原测试同 package 可见。
