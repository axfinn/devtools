package handlers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"
)

type AIGatewayHandler struct {
	db                *models.DB
	cfg               *config.Config
	imageHandler      *ImageUnderstandingHandler
	enc               *utils.EncryptionService // 用于 AnthropicProvider.APIKey 加解密
	client            *http.Client             // 带代理，用于 OpenAI 兼容接口
	noProxyClient     *http.Client             // 不走代理，禁用压缩（用于 Chat/Anthropic 避免 Brotli）
	longNoProxyClient *http.Client             // 不走代理，禁用压缩，超时放宽到 5 分钟（非流式长推理聊天：DeepSeek reasoner / MiniMax M3）
	mediaClient       *http.Client             // 不走代理，启用压缩（用于 MiniMax 媒体 API：图片/视频/TTS）
	musicSubmitClient *http.Client             // 音乐生成专用：MiniMax /v1/music_generation 是同步接口，会阻塞到音频生成完成才返回（5 分钟音频常见 1-3 分钟），不能复用 mediaClient 的 90s 超时
	streamClient      *http.Client             // 不走代理，用于 SSE 流式请求（body 无超时，依赖 context 取消；但连接/握手/响应头有显式超时兜底）
	// streamClientCache 按 (providerKey, headerTimeoutSec) 缓存派生出的 *http.Client。
	// providerKey 通常为 provider.Name,headerTimeoutSec 决定 ResponseHeaderTimeout。
	// 老的全局 streamClient 字段保留作兼容(某些 handler 还直接用它),不强制所有路径迁移。
	streamClientCache sync.Map // map[string]*http.Client, key 见 streamClientCacheKey
}

// streamClientCacheKey 生成缓存 key。headerTimeoutSec=0 表示默认 60s。
func streamClientCacheKey(providerName string, headerTimeoutSec int) string {
	return providerName + "|" + time.Duration(headerTimeoutSec).String()
}

type usageSummary struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Cost         float64
	Currency     string
}

type CreateAIAPIKeyRequest struct {
	SuperAdminPassword  string   `json:"super_admin_password"`
	Name                string   `json:"name" binding:"required"`
	AllowedModels       []string `json:"allowed_models"`
	AllowedScopes       []string `json:"allowed_scopes"`
	ExpiresDays         int      `json:"expires_days"`
	RateLimitPerHour    int      `json:"rate_limit_per_hour"`
	BudgetLimit         float64  `json:"budget_limit"`
	AlertThreshold      float64  `json:"alert_threshold"`
	Notes               string   `json:"notes"`
	AnthropicProviderID int      `json:"anthropic_provider_id"` // 0=默认，非0=指定下游
}

type UpdateAIAPIKeyRequest struct {
	SuperAdminPassword  string   `json:"super_admin_password"`
	Name                string   `json:"name"`
	AllowedModels       []string `json:"allowed_models"`
	AllowedScopes       []string `json:"allowed_scopes"`
	ExpiresDays         int      `json:"expires_days"`     // >0 表示从当前时间延长
	ClearExpiration     bool     `json:"clear_expiration"` // true 时把 expires_at 置 NULL（永不过期），优先级高于 ExpiresDays
	RateLimitPerHour    int      `json:"rate_limit_per_hour"`
	BudgetLimit         float64  `json:"budget_limit"`
	AlertThreshold      float64  `json:"alert_threshold"`
	Notes               string   `json:"notes"`
	AnthropicProviderID int      `json:"anthropic_provider_id"`
}

type ChatCompletionRequest struct {
	Model           string                   `json:"model" binding:"required"`
	Messages        []map[string]interface{} `json:"messages" binding:"required"`
	Temperature     *float64                 `json:"temperature"`
	MaxTokens       *int                     `json:"max_tokens"`
	TopP            *float64                 `json:"top_p"`
	Stop            interface{}              `json:"stop"`
	ResponseFormat  map[string]interface{}   `json:"response_format"`
	Tools           []map[string]interface{} `json:"tools"`
	ToolChoice      interface{}              `json:"tool_choice"`
	ReasoningEffort string                   `json:"reasoning_effort"`
	ExtraBody       map[string]interface{}   `json:"extra_body"`
	Stream          bool                     `json:"stream"`
	Metadata        map[string]interface{}   `json:"metadata"`
}

// TTSAllowedModels MiniMax TTS 允许的模型列表（官方模型名）
var TTSAllowedModels = []string{
	"speech-01-hd",
	"speech-01-turbo",
	"speech-02-hd",
	"speech-02-turbo",
	"speech-2.6-hd",
	"speech-2.6-turbo",
	"speech-2.8-hd",
	"speech-2.8-turbo",
}

// TokenPlanAllowedModels MiniMax Token Plan 允许的模型列表（官方模型名）
var TokenPlanAllowedModels = []string{
	// TTS HD / Turbo
	"speech-01-hd",
	"speech-01-turbo",
	"speech-02-hd",
	"speech-02-turbo",
	"speech-2.6-hd",
	"speech-2.6-turbo",
	"speech-2.8-hd",
	"speech-2.8-turbo",
	// Hailuo 视频（官方模型名: MiniMax-Hailuo-2.3-Fast, MiniMax-Hailuo-2.3, MiniMax-Hailuo-02, T2V-01-Director, T2V-01）
	"MiniMax-Hailuo-2.3-Fast",
	"MiniMax-Hailuo-2.3",
	"MiniMax-Hailuo-02",
	"T2V-01-Director",
	"T2V-01",
	// Music（官方模型名: music-3.0 / music-2.6 / music-cover + 限免版 -free）
	"music-2.6",
	"music-2.6-free",
	"music-cover",
	"music-cover-free",
	"music-3.0",
	"music-3.0-free",
	// Image（官方模型名: image-01, image-01-live）
	"image-01",
	"image-01-live",
}

// TokenPlanAsyncModels 需要异步轮询的模型（视频、音乐、图片生成）
var TokenPlanAsyncModels = []string{
	"MiniMax-Hailuo-2.3-Fast",
	"MiniMax-Hailuo-2.3",
	"MiniMax-Hailuo-02",
	"T2V-01-Director",
	"T2V-01",
	"music-2.6",
	"music-2.6-free",
	"music-cover",
	"music-cover-free",
	"music-3.0",
	"music-3.0-free",
	"image-01",
	"image-01-live",
}

// isTokenPlanAsyncModel 判断模型是否需要异步轮询
func isTokenPlanAsyncModel(model string) bool {
	for _, m := range TokenPlanAsyncModels {
		if m == model {
			return true
		}
	}
	return false
}

// TokenPlanRequest MiniMax Token Plan 通用请求
type TokenPlanRequest struct {
	Model      string                 `json:"model" binding:"required"`
	Prompt     string                 `json:"prompt"`
	Text       string                 `json:"text"`
	Image      string                 `json:"image"`
	Images     []string               `json:"images"`
	Size       string                 `json:"size"`
	Duration   int                    `json:"duration"`
	Count      int                    `json:"count"`
	Parameters map[string]interface{} `json:"parameters"`
}

// TTSRequest MiniMax TTS 请求
type TTSRequest struct {
	Model       string  `json:"model" binding:"required"`
	Text        string  `json:"text" binding:"required"`
	Voice       string  `json:"voice"`
	Speed       float64 `json:"speed"`
	Volume      float64 `json:"volume"`
	Pitch       float64 `json:"pitch"`
	AudioFormat string  `json:"audio_format"` // mp3/wav/pcm
}

func NewAIGatewayHandler(db *models.DB, cfg *config.Config, imageHandler *ImageUnderstandingHandler, enc *utils.EncryptionService) *AIGatewayHandler {
	// 上游 HTTP 传输统一显式配置拨号/TLS 超时与连接池。零值 Transport 下
	// TLSHandshakeTimeout / ResponseHeaderTimeout 均为 0（无限等待），上游"连上但
	// 不回响应头"会永久卡死请求 goroutine 并泄漏连接 —— 这是代理连接不稳定的主因之一。
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	buildTransport := func(disableCompression bool, responseHeaderTimeout time.Duration) *http.Transport {
		return &http.Transport{
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     false,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   16,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: responseHeaderTimeout,
			// DisableCompression=true：不发 Accept-Encoding，上游返回原始未压缩数据。
			// 避免上游返回 Brotli（br）时 Go 标准库无法解压导致 JSON 解析失败。
			// gzip 解压由各调用点按需处理，或直接透传给客户端。
			DisableCompression: disableCompression,
		}
	}
	// 非流式：不设 ResponseHeaderTimeout，长推理（DeepSeek reasoner 等）要等
	// generation 完成后才返回响应头，总时长由各 client 的 Timeout 兜底。
	noProxyTransport := buildTransport(true, 0)
	// 默认流式：响应头必须在 60s 内到达（防止上游挂起无限等待），body 无超时由 context 控制。
	// 注意:按 provider 派生的 streamClient(见 pickStreamClient)走 buildStreamTransport 单独构造,
	// Ollama 等慢本地大模型可用更长 ResponseHeaderTimeout(由 provider.HeaderTimeoutSec 决定)。
	streamTransport := buildTransport(true, 60*time.Second)
	// mediaTransport 启用压缩：MiniMax 图片/视频/音乐/TTS 端点要求 Accept-Encoding，
	// 否则会直接断开连接（EOF）。这些端点不返回 Brotli，安全启用。
	// TTS 生成可能耗时较长，不设响应头超时（由 mediaClient 90s Timeout 兜底）。
	mediaTransport := buildTransport(false, 0)

	return &AIGatewayHandler{
		db:           db,
		cfg:          cfg,
		imageHandler: imageHandler,
		enc:          enc,
		client:       &http.Client{Timeout: 600 * time.Second},
		noProxyClient: &http.Client{
			Timeout:   90 * time.Second,
			Transport: noProxyTransport,
		},
		// 非流式聊天调用专用：DeepSeek reasoner / MiniMax M3 长推理 90s 常被截断，
		// 放宽到 5 分钟（连接/握手仍由 transport 的 10s 超时兜底）。
		longNoProxyClient: &http.Client{
			Timeout:   5 * time.Minute,
			Transport: noProxyTransport,
		},
		mediaClient: &http.Client{
			Timeout:   90 * time.Second,
			Transport: mediaTransport,
		},
		musicSubmitClient: &http.Client{
			Timeout:   5 * time.Minute,
			Transport: mediaTransport,
		},
		// 流式请求不设 Timeout：http.Client.Timeout 包含读取 body 的时间，
		// 对 SSE 长流会在 90s 后强制断开。改用 context 取消（客户端断开时自动传播）。
		// 该字段为全局兜底(60s header timeout),新代码应优先用 pickStreamClient(provider)
		// 按 provider 派生(支持自定义 header_timeout_sec,Ollama 27B 用 600s)。
		streamClient: &http.Client{
			Timeout:   0,
			Transport: streamTransport,
		},
	}
}

// buildStreamTransport 按指定的 ResponseHeaderTimeout 构造流式 Transport。
// headerTimeoutSec <= 0 时 fallback 到默认 60s(老的全局 streamClient 行为)。
// 与 NewAIGatewayHandler 里的 buildTransport 同源,但独立暴露给 pickStreamClient 用。
func buildStreamTransport(headerTimeoutSec int) *http.Transport {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	timeout := time.Duration(headerTimeoutSec) * time.Second
	if headerTimeoutSec <= 0 {
		timeout = 60 * time.Second
	}
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
		// 流式上游不接收压缩:本地 Ollama 不支持 Brotli,云厂商 SSE 一般也不压缩
		DisableCompression: true,
	}
}

// pickStreamClient 根据 provider 派生 stream HTTP 客户端,带 sync.Map 缓存。
// key 包含 provider 名 + header_timeout_sec,改 provider 配置后下次请求自动重建。
// provider=nil 时返回全局 streamClient(老路径兼容,默认 60s)。
func (h *AIGatewayHandler) pickStreamClient(provider *config.AnthropicProviderConfig) *http.Client {
	if provider == nil {
		return h.streamClient
	}
	key := streamClientCacheKey(provider.Name, provider.HeaderTimeoutSec)
	if cached, ok := h.streamClientCache.Load(key); ok {
		return cached.(*http.Client)
	}
	client := &http.Client{
		Timeout:   0,
		Transport: buildStreamTransport(provider.HeaderTimeoutSec),
	}
	actual, _ := h.streamClientCache.LoadOrStore(key, client)
	return actual.(*http.Client)
}

// pickMediaSubmitClient 根据模型选择提交用的 HTTP 客户端。
// 音乐模型走 musicSubmitClient(5 分钟)，其他媒体模型走 mediaClient(90s)。

func (h *AIGatewayHandler) pickMediaSubmitClient(model string) *http.Client {
	if strings.HasPrefix(model, "music-") {
		return h.musicSubmitClient
	}
	return h.mediaClient
}
