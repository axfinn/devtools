package middleware

// 响应压缩（gzip）。
//
// 设计要点（见 docs/plans/finn-29-first-paint-perf.md §C 与 FINN-41 评审结论）：
//   - 仅压缩可压缩内容（text/*、application/json、application/javascript、image/svg+xml 等）；
//     已压缩 / 媒体扩展名一律排除，避免再压无收益烧 CPU。
//   - SSE / WebSocket / Range / HLS / 代理隧道 / 健康探活 排除，原因：
//       * gzip.Writer 自带缓冲，会吞 Flush()，导致 SSE 客户端看到「一次挤一大坨」的延迟
//       * HTTP/1.1 Upgrade 握手与 WebSocket 帧协议不应被中间件介入
//       * Range 请求的 Content-Range / Content-Length 是字节偏移语义，gzip 后失效
//   - gin-contrib/gzip v1.0.1 shouldCompress 内置已处理：
//       * Accept-Encoding 必须包含 gzip
//       * Connection: Upgrade
//       * Accept: text/event-stream
//   - 压缩级别 5（不是默认的 6）：本机同跑 OCR / ASR / TTS sidecar，
//     level 5 与 level 6 压缩比差异 < 1%，CPU 多 ~10%，权衡后取 5。
//
// FINN-51 Stage3 修复（与 gin-contrib/gzip v1.0.1 shouldCompress 行为差异）：
//   1. HEAD 请求不压缩（HTTP/1.1 RFC 9110 §9.3.2 HEAD 不应有 body）。
//   2. Range 请求不压缩（请求头 Range 与压缩互斥：Content-Range / Content-Length
//      是字节偏移语义，gzip 后偏移失效）。
//   3. 响应 Content-Range 头已设时不压缩（206 Partial Content 字节偏移）。
//   4. 响应 Content-Encoding 已设时不二次压缩（handler 已自压）。
//
// FINN-52 重构（去掉全量缓冲，改响应侧判定）：
//
//   FINN-41/FINN-49 的实现用 captureWriter 把 handler 的全部响应字节先攒进
//   bytes.Buffer，等 c.Next() 返回后再决定是否 gzip。两个后果：
//
//   a) captureWriter.Flush() 是空实现 —— SSE handler 的 `c.Writer.(http.Flusher)`
//      断言通过（captureWriter 确有 Flush 方法），于是**静默**吞掉每一次 Flush。
//      Anthropic 代理精心设计的 ping 心跳（拨号期 1s 一次、生成期 5s 一次，
//      目的就是压住 Cloudflare/nginx 的空闲超时）一个字节都到不了客户端，
//      首字节延迟 = 整段回答生成时间，且 CF 仍按空闲超时 524。
//   b) 压不压缩只靠请求侧白名单（excludedPathPrefixes）。白名单是手工维护的，
//      FINN-49 已经因为「白名单路径和 routes/*.go 的 Group() 对不上」翻过一次车，
//      而 /api/anthropic/v1/messages、/api/ai-gateway/v1/chat/completions 这些
//      主力 LLM 流式端点至今不在名单里。
//
//   现在改为：**在响应头真正下发的那一刻，按响应 Content-Type 判定**。
//   Content-Type 是比「请求 URL 前缀」可靠得多的信号 —— 流式响应必然声明
//   text/event-stream，且 handler 一定在写第一个字节前就设好它。
//
//   gin 的写入时序是这里的关键，必须对齐：
//       c.Status(code)            → ResponseWriter.WriteHeader(code)，**只记录状态码**
//       writeContentType(w, ...)  → 设置响应头 map
//       w.Write(body)             → ResponseWriter.WriteHeaderNow() 才真正下发响应头
//   所以判定必须放在 WriteHeaderNow/首次 Write 里，而不是 WriteHeader 里，
//   否则读到的是「还没设 Content-Type」的空头。
//
//   透传路径（不压缩）直接写原始 ResponseWriter，Flush 真实转发，SSE 恢复逐帧送达。
//   请求侧白名单保留为快路径（省掉包装开销），但不再是正确性的唯一依赖。
//
// 本中间件只暴露 Handler（gin 中间件），便于单测；调用方只需 router.Use(...)。

import (
	"compress/gzip"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// compressWriter 包装 gin.ResponseWriter，在响应头真正下发时决定是否 gzip。
//
// 不压缩时它是**透明**的：Write/WriteString/Flush 全部直接转发给原始 writer，
// 不引入任何缓冲，SSE 逐帧送达语义与不用中间件时完全一致。
type compressWriter struct {
	gin.ResponseWriter

	level    int
	status   int
	headSent bool // 响应头是否已下发（下发后状态码/响应头都不可再改）
	compress bool // 本次响应是否走 gzip
	gz       *gzip.Writer
	pooled   bool // gz 是否来自 gzipWriterPool（决定归还还是丢弃）
}

// newCompressWriter 构造包装器。
// level 越界时回退到 5：gzip.NewWriterLevel 对非法 level 返回 error，
// 而那时 Content-Encoding: gzip 已经写进响应头了，无法撤销 —— 与其在 Write
// 里踩 nil gz，不如构造期就纠正。
func newCompressWriter(w gin.ResponseWriter, level int) *compressWriter {
	if level < gzip.BestSpeed || level > gzip.BestCompression {
		level = 5
	}
	return &compressWriter{
		ResponseWriter: w,
		level:          level,
		status:         http.StatusOK,
	}
}

// WriteHeader 只记录状态码，与 gin 原生 responseWriter 语义一致：
// 真正把响应头推给 net/http 的是 WriteHeaderNow()（由 Write/Flush 触发）。
// 此处若直接下发，就再也读不到 render 稍后才设置的 Content-Type 了。
func (w *compressWriter) WriteHeader(code int) {
	if code <= 0 {
		return
	}
	if w.headSent {
		// 响应头已下发：交给底层（gin 会按 RFC 语义忽略并记 warning，不改状态码）。
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.status = code
}

// WriteHeaderNow 强制下发响应头（gin 在 Write 内部也会调，幂等）。
func (w *compressWriter) WriteHeaderNow() { w.writeHeadNow() }

// writeHeadNow 下发响应头：此刻 handler 设置的响应头已全部就位，是做判定的唯一正确时机。
func (w *compressWriter) writeHeadNow() {
	if w.headSent {
		return
	}
	w.headSent = true
	w.decide()
	w.ResponseWriter.WriteHeader(w.status)
	w.ResponseWriter.WriteHeaderNow()
	if w.compress {
		w.startGzip()
	}
}

// decide 依据当前响应头判定是否压缩。只在下发前调用一次。
func (w *compressWriter) decide() {
	header := w.ResponseWriter.Header()
	// 204/304 无 body，挂 Content-Encoding 属于无意义甚至有害的语义污染。
	if w.status == http.StatusNoContent || w.status == http.StatusNotModified {
		return
	}
	// 流式响应：gzip.Writer 的缓冲会吞掉 Flush，一旦压缩，SSE 就退化成
	// 「整段生成完才吐第一个字节」。这是 FINN-52 的核心修复点。
	if isStreamingContentType(header.Get("Content-Type")) {
		return
	}
	// handler 已声明字节偏移语义（206）或已自行压缩，不介入。
	if header.Get("Content-Encoding") != "" || header.Get("Content-Range") != "" {
		return
	}
	w.compress = true
	header.Del("Content-Length") // 压缩后长度未知，删掉让 net/http 走 chunked
	header.Set("Content-Encoding", "gzip")
	header.Set("Vary", "Accept-Encoding")
}

// isStreamingContentType 识别不应压缩的流式响应。
// 用 Contains 而非相等：Content-Type 可能带 charset 等参数。
func isStreamingContentType(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}

// startGzip 惰性创建 gzip.Writer。gzip.NewWriter 在首次 Write 前不写任何字节，
// 因此放在响应头下发之后创建是安全的。
func (w *compressWriter) startGzip() {
	if w.gz != nil || !w.compress {
		return
	}
	if w.level == 5 {
		w.gz = gzipWriterPool.Get().(*gzip.Writer)
		w.gz.Reset(w.ResponseWriter)
		w.pooled = true
		return
	}
	gz, err := gzip.NewWriterLevel(w.ResponseWriter, w.level)
	if err != nil {
		// 理论上不可达（构造期已把 level 夹到 [1,9]，gzip 只对 <-2 或 >9 报错）。
		// 兜底仍用 level 5：绝不能留下 gz==nil 而 compress==true 的状态 ——
		// 那会变成「声明了 Content-Encoding: gzip 却写明文」的静默损坏。
		gz, _ = gzip.NewWriterLevel(w.ResponseWriter, 5)
	}
	w.gz = gz
}

// finish 由中间件在 c.Next() 返回后调用：兜底下发响应头并收尾 gzip 流。
func (w *compressWriter) finish() {
	if !w.headSent {
		// handler 什么都没写（如 OPTIONS 预检直接 return）—— 与原生行为一致地下发响应头。
		w.writeHeadNow()
	}
	if w.gz == nil {
		return
	}
	_ = w.gz.Close()
	if w.pooled {
		w.gz.Reset(io.Discard)
		gzipWriterPool.Put(w.gz)
	}
	w.gz = nil
}

func (w *compressWriter) Write(b []byte) (int, error) {
	w.writeHeadNow()
	if w.compress {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *compressWriter) WriteString(s string) (int, error) {
	w.writeHeadNow()
	if w.compress {
		return w.gz.Write([]byte(s))
	}
	return w.ResponseWriter.WriteString(s)
}

// Flush 必须在两条路径上都是「真 Flush」：
//   - 透传路径直接转发，SSE 逐帧送达；
//   - 压缩路径先 gz.Flush()（把 gzip 缓冲吐出来）再转发。
//
// FINN-41/FINN-49 的 captureWriter 在这里是空实现，是本轮 SSE 被吞的根因。
func (w *compressWriter) Flush() {
	w.writeHeadNow()
	if w.gz != nil {
		_ = w.gz.Flush()
	}
	w.ResponseWriter.Flush()
}

// gzipWriterPool 复用 gzip.Writer 实例，避免每请求 alloc。
// 项目实际调用固定 level=5（backend/app_http.go），其他 level 走非池化路径。
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		gz, _ := gzip.NewWriterLevel(io.Discard, 5)
		return gz
	},
}

// excludedExt 镜像 FINN-41/FINN-49 的 WithExcludedExtensions 列表。
var excludedExt = map[string]bool{
	// 图片
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".bmp": true, ".ico": true, ".svg": true,
	// 音视频
	".mp4": true, ".webm": true, ".mov": true, ".mkv": true,
	".avi": true, ".flv": true, ".ts": true, ".m4s": true, ".m3u8": true,
	".mp3": true, ".wav": true, ".ogg": true, ".flac": true,
	// 字体
	".woff": true, ".woff2": true, ".ttf": true, ".otf": true, ".eot": true,
	// 压缩归档
	".zip": true, ".gz": true, ".tgz": true, ".br": true,
	".7z": true, ".rar": true, ".tar": true, ".bz2": true, ".xz": true,
	// 文档（PDF 已压缩）
	".pdf": true,
}

// excludedPathPrefixes 是**快路径**白名单：命中即完全不包装 writer（省掉包装开销）。
//
// 注意：正确性不再依赖本列表 —— 响应 Content-Type 为 text/event-stream 时
// compressWriter 会自动走透传（见 decide）。所以这里**不要**为了「保险」把
// LLM JSON 端点加进来：/api/anthropic/v1/messages 这类端点同时承载流式(SSE)
// 与非流式(大 JSON)两种响应，按路径排除会连带丢掉非流式大响应的压缩收益。
//
// （prefix 匹配；trailing slash 决定是否匹配同级根路径。）
var excludedPathPrefixes = []string{
	"/api/health",
	"/s/",
	"/sub/proxy",
	"/ss/",
	"/mock/",
	// 以下是确定只有 SSE/WS 语义、永不需要压缩的路径。
	"/api/internal/chat/stream",
	"/api/ai-gateway/v1/image/understanding/sse",
	"/api/ai-gateway/v1/image/understanding/stream/",
	"/api/image-understanding/sse/stream/",
	"/api/api-gateway/cpa/",
	"/api/autodev/",
	"/api/terminal/",
	"/api/screen/",
	"/api/nfsshare/",
	"/api/proxy/ws-tunnel",
	"/api/game/arcade/ws/",
	"/api/chat/room/",
}

// ResponseCompression 范围收窄后的响应压缩中间件。
//
// level: gzip 压缩级别（0-9；建议 5）。
//
// 请求侧快路径排除（命中即不包装 writer）：
//   - HEAD 请求                                  RFC 9110 §9.3.2 不应有 body
//   - 带 Range 的请求                            字节偏移语义与压缩互斥
//   - Accept-Encoding 不含 gzip / Connection: Upgrade / Accept: text/event-stream
//   - 已压缩 / 媒体扩展名（excludedExt）
//   - SSE / WS / HLS 路径前缀（excludedPathPrefixes）
//
// 响应侧兜底判定（compressWriter.decide，见文件头 FINN-52 说明）：
//   - 204 / 304                                          无 body
//   - Content-Type 含 text/event-stream                  流式，压缩会吞 Flush
//   - Content-Encoding / Content-Range 已设               handler 已处理或字节偏移语义
func ResponseCompression(level int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodHead {
			c.Next()
			return
		}
		if c.Request.Header.Get("Range") != "" {
			c.Next()
			return
		}
		// 请求头协商（与 gin-contrib/gzip v1.0.1 shouldCompress 一致）。
		if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") ||
			strings.Contains(c.Request.Header.Get("Connection"), "Upgrade") ||
			strings.Contains(c.Request.Header.Get("Accept"), "text/event-stream") {
			c.Next()
			return
		}
		// 扩展名排除。
		if excludedExt[filepath.Ext(c.Request.URL.Path)] {
			c.Next()
			return
		}
		// 路径前缀排除。
		for _, p := range excludedPathPrefixes {
			if strings.HasPrefix(c.Request.URL.Path, p) {
				c.Next()
				return
			}
		}

		w := newCompressWriter(c.Writer, level)
		c.Writer = w
		c.Next()
		w.finish()
	}
}
