package handlers

import (
	"net/http"
	"time"

	"devtools/config"
	"devtools/models"
)

type AIGatewayHandler struct {
	db            *models.DB
	cfg           *config.Config
	bailian       *BailianHandler
	imageHandler  *ImageUnderstandingHandler
	client        *http.Client // 带代理，用于 OpenAI 兼容接口
	noProxyClient *http.Client // 不走代理，用于 MiniMax 等外部 API
}

type usageSummary struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Cost         float64
	Currency     string
}

type CreateAIAPIKeyRequest struct {
	SuperAdminPassword string   `json:"super_admin_password"`
	Name               string   `json:"name" binding:"required"`
	AllowedModels      []string `json:"allowed_models"`
	AllowedScopes      []string `json:"allowed_scopes"`
	ExpiresDays        int      `json:"expires_days"`
	RateLimitPerHour   int      `json:"rate_limit_per_hour"`
	BudgetLimit        float64  `json:"budget_limit"`
	AlertThreshold     float64  `json:"alert_threshold"`
	Notes              string   `json:"notes"`
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

type MediaGenerationRequest struct {
	Model           string                 `json:"model" binding:"required"`
	Prompt          string                 `json:"prompt" binding:"required"`
	NegativePrompt  string                 `json:"negative_prompt"`
	Image           string                 `json:"image"`
	Images          []string               `json:"images"`
	Size            string                 `json:"size"`
	Count           int                    `json:"count"`
	Seed            *int                   `json:"seed"`
	Watermark       *bool                  `json:"watermark"`
	Duration        int                    `json:"duration"`
	Resolution      string                 `json:"resolution"`
	FPS             int                    `json:"fps"`
	AutoPoll        bool                   `json:"auto_poll"`
	WaitSeconds     int                    `json:"wait_seconds"`
	ClientName      string                 `json:"client_name"`
	ClientRequestID string                 `json:"client_request_id"`
	Parameters      map[string]interface{} `json:"parameters"`
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
	// Music（官方模型名: music-2.5, music-2.6, music-cover）
	"music-2.5",
	"music-2.6",
	"music-cover",
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
	"music-2.5",
	"music-2.6",
	"music-cover",
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

func NewAIGatewayHandler(db *models.DB, cfg *config.Config, bailian *BailianHandler, imageHandler *ImageUnderstandingHandler) *AIGatewayHandler {
	return &AIGatewayHandler{
		db:           db,
		cfg:          cfg,
		bailian:      bailian,
		imageHandler: imageHandler,
		client:       &http.Client{Timeout: 90 * time.Second},
		noProxyClient: &http.Client{
			Timeout: 90 * time.Second,
			Transport: &http.Transport{
				Proxy: nil, // 不使用任何代理
			},
		},
	}
}
