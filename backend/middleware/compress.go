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
// 本中间件只暴露 Handler（gin 中间件），便于单测；调用方只需 router.Use(...)。

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// ResponseCompression 范围收窄后的响应压缩中间件。
//
// level: gzip 压缩级别（0-9，0=不压缩，9=最大压缩；建议 5）。
//
// 已排除路径（按前缀匹配）：
//   - /api/health                                  健康探活
//   - /s/                                          短链 302 重定向
//   - /sub/proxy, /mock/                           公共非 API 下载与 Mock
//   - /api/internal/chat/stream                    SSE AI 网关聊天
//   - /api/aigw/v1/image/understanding/sse         SSE 图片理解
//   - /api/aigw/v1/image/understanding/stream/     SSE 图片理解进度
//   - /api/image/sse/stream/                       SSE 图片理解
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
	return gzip.Gzip(level,
		gzip.WithExcludedExtensions([]string{
			// 图片
			".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".ico", ".svg",
			// 音视频
			".mp4", ".webm", ".mov", ".mkv", ".avi", ".flv", ".ts", ".m4s", ".m3u8",
			".mp3", ".wav", ".ogg", ".flac",
			// 字体
			".woff", ".woff2", ".ttf", ".otf", ".eot",
			// 压缩归档
			".zip", ".gz", ".tgz", ".br", ".7z", ".rar", ".tar", ".bz2", ".xz",
			// 文档（PDF 已压缩）
			".pdf",
		}),
		gzip.WithExcludedPaths([]string{
			"/api/health",
			"/s/",
			"/sub/proxy",
			"/mock/",
			"/api/internal/chat/stream",
			"/api/aigw/v1/image/understanding/sse",
			"/api/aigw/v1/image/understanding/stream/",
			"/api/image/sse/stream/",
			"/api/autodev/",
			"/api/terminal/",
			"/api/screen/",
			"/api/nfsshare/",
			"/api/proxy/ws-tunnel",
			"/api/game/arcade/ws/",
			"/api/chat/room/",
		}),
	)
}
