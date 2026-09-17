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
//   1. HEAD 请求不压缩（HTTP/1.1 RFC 9110 §9.3.2 HEAD 不应有 body，
//      而 gzip.WrapWriter.Close() 会无条件写出 ~23 字节 gzip frame 头）。
//   2. Range 请求不压缩（请求头 Range 与压缩互斥：Content-Range / Content-Length
//      是字节偏移语义，gzip 后偏移失效）。
//   3. 响应 Content-Range 头已设时不压缩（206 Partial Content 字节偏移）。
//   4. 响应 Content-Encoding 已设时不二次压缩（handler 已自压）。
//
// 实现策略：原 gin-contrib/gzip v1.0.1 在 c.Next() 前无条件注入 Content-Encoding
// 并替换 c.Writer 为 gzipWriter。这不支持 #3/#4 的"handler 已设 Content-Encoding
// 时跳过"判定（handler 设 Content-Encoding 时，c.Writer 已被替换为 gzipWriter，
// 字节已被 gzip.Writer 吞下，事后无法撤销）。
//
// 本实现改用 captureWriter 拦截 handler 的 headers 与 body 写入，待 handler
// 返回后再依据捕获的响应头决定是否真正 gzip。路径/扩展名/请求头协商仍走请求侧
// 前置判定（与 FINN-41/FINN-49 行为等价）。
//
// 本中间件只暴露 Handler（gin 中间件），便于单测；调用方只需 router.Use(...)。

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// captureWriter 拦截 handler 的 headers 与 body 写入；外层中间件 flush 时
// 再依据捕获的响应头（Content-Range / Content-Encoding）决定是否走 gzip。
type captureWriter struct {
	gin.ResponseWriter
	bufHead   http.Header
	body      *bytes.Buffer
	status    int
	wroteHead bool
}

func newCaptureWriter(w gin.ResponseWriter) *captureWriter {
	return &captureWriter{
		ResponseWriter: w,
		bufHead:        make(http.Header),
		body:           &bytes.Buffer{},
		status:         http.StatusOK,
	}
}

// Header 让 handler 写入的 header 落到 bufHead，不污染原始 writer。
func (c *captureWriter) Header() http.Header { return c.bufHead }

// WriteHeader / WriteHeaderNow / Flush 仅记录状态，由外层 flush 时再写原始 writer。
// （拦截延迟：原始 writer 的 WriteHeaderNow 会立刻把 headers 推到网络/recorder，
// 与"延迟到 flush"的策略冲突；handler 的 Flush() 多用于 SSE 流式，SSE 路径已被
// 前置排除，不会走到本 captureWriter。）
func (c *captureWriter) WriteHeader(code int) {
	if !c.wroteHead {
		c.status = code
		c.wroteHead = true
	}
}
func (c *captureWriter) WriteHeaderNow() {}
func (c *captureWriter) Flush()          {}

// Write / WriteString 把 handler body 落到 buffer，不真正走网络/recorder。
func (c *captureWriter) Write(b []byte) (int, error) {
	if !c.wroteHead {
		c.WriteHeader(http.StatusOK)
	}
	return c.body.Write(b)
}
func (c *captureWriter) WriteString(s string) (int, error) {
	if !c.wroteHead {
		c.WriteHeader(http.StatusOK)
	}
	return c.body.WriteString(s)
}

// Status / Size / Written 让外层 flush 时能读出 capture 状态。
func (c *captureWriter) Status() int              { return c.status }
func (c *captureWriter) Size() int                { return c.body.Len() }
func (c *captureWriter) Written() bool            { return c.wroteHead }
func (c *captureWriter) WriteHeaderWritten() bool { return c.wroteHead }

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

// excludedPathPrefixes 镜像 FINN-41/FINN-49 的 WithExcludedPaths 列表
// （prefix 匹配；trailing slash 决定是否匹配同级根路径）。
var excludedPathPrefixes = []string{
	"/api/health",
	"/s/",
	"/sub/proxy",
	"/mock/",
	// SSE 排除路径必须与 backend/routes/{ai_gateway,image_understanding}.go
	// 的 Group() 注册字符串完全一致；不要凭记忆或缩写。详见 FINN-49。
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
// level: gzip 压缩级别（0-9，0=不压缩，9=最大压缩；建议 5）。
//
// 已排除路径（按前缀匹配）：
//   - /api/health                                  健康探活
//   - /s/                                          短链 302 重定向
//   - /sub/proxy, /mock/                           公共非 API 下载与 Mock
//   - /api/internal/chat/stream                    SSE AI 网关聊天
//   - /api/ai-gateway/v1/image/understanding/sse   SSE 图片理解
//   - /api/ai-gateway/v1/image/understanding/stream/  SSE 图片理解进度
//   - /api/image-understanding/sse/stream/          SSE 图片理解
//   - /api/api-gateway/cpa/                         SSE CPA 反代（兜底）
//   - /api/autodev/                                SSE 项目初始化与 CLI 更新流
//   - /api/terminal/                               WebSocket SSH
//   - /api/screen/                                 WebSocket 屏幕共享信令
//   - /api/nfsshare/                               WebSocket 监视 + HLS 分片 + Range 视频
//   - /api/proxy/ws-tunnel                         WebSocket 隧道
//   - /api/game/arcade/ws/                         WebSocket 游戏
//   - /api/chat/room/                              WebSocket 聊天室
//
// 已排除扩展名：图片、音视频、字体、压缩归档、PDF。
func ResponseCompression(level int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 缺陷 #4：HEAD 不应被压缩（gzip frame 头/校验和 ≈23 字节会被注入）。
		if c.Request.Method == http.MethodHead {
			c.Next()
			return
		}
		// 缺陷 #1：Range 请求字节偏移与压缩互斥。
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

		// 用 captureWriter 拦截 handler 的 headers 与 body，待 handler 返回后
		// 再依据捕获的响应头决定是否走 gzip（缺陷 #2 / #3 的关键）。
		original := c.Writer
		cap := newCaptureWriter(original)
		c.Writer = cap
		c.Next()

		// 缺陷 #2 / #3：响应头已表明不应压缩（handler 显式声明 Content-Range
		// 或 Content-Encoding）。直接把捕获的 headers + body 写回原始 writer。
		if cap.bufHead.Get("Content-Range") != "" ||
			cap.bufHead.Get("Content-Encoding") != "" {
			for k, vv := range cap.bufHead {
				for _, v := range vv {
					original.Header().Add(k, v)
				}
			}
			original.WriteHeader(cap.status)
			_, _ = original.Write(cap.body.Bytes())
			return
		}

		// 压缩路径：headers 合并 + Content-Encoding/Vary + body 走 gzip 流。
		for k, vv := range cap.bufHead {
			for _, v := range vv {
				original.Header().Add(k, v)
			}
		}
		original.Header().Del("Content-Length")
		original.Header().Set("Content-Encoding", "gzip")
		original.Header().Set("Vary", "Accept-Encoding")
		original.WriteHeader(cap.status)

		var gz *gzip.Writer
		if level == 5 {
			gz = gzipWriterPool.Get().(*gzip.Writer)
			gz.Reset(original)
		} else {
			gz, _ = gzip.NewWriterLevel(original, level)
		}
		_, _ = gz.Write(cap.body.Bytes())
		_ = gz.Close()
		original.Header().Set("Content-Length", fmt.Sprint(original.Size()))
		if level == 5 {
			gz.Reset(io.Discard)
			gzipWriterPool.Put(gz)
		}
	}
}
