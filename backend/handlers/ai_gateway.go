package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

type AIGatewayHandler struct {
	db            *models.DB
	cfg           *config.Config
	bailian       *BailianHandler
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

func NewAIGatewayHandler(db *models.DB, cfg *config.Config, bailian *BailianHandler) *AIGatewayHandler {
	return &AIGatewayHandler{
		db:      db,
		cfg:     cfg,
		bailian: bailian,
		client:  &http.Client{Timeout: 90 * time.Second},
		noProxyClient: &http.Client{
			Timeout: 90 * time.Second,
			Transport: &http.Transport{
				Proxy: nil, // 不使用任何代理
			},
		},
	}
}

func (h *AIGatewayHandler) GetDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "AI Gateway API 文档",
		"summary": "统一对外开放 DeepSeek、MiniMax、Bailian 模型能力，使用超级管理员签发的 API Key 访问。DeepSeek 已兼容 deepseek-chat、deepseek-reasoner、deepseek-v4-flash、deepseek-v4-pro。",
		"auth": gin.H{
			"admin_header": "X-Super-Admin-Password",
			"api_key":      "Authorization: Bearer dtk_ai_xxx",
		},
		"billing": gin.H{
			"fields":  []string{"input_tokens", "output_tokens", "total_tokens", "estimated_cost", "currency"},
			"rule":    "文本模型优先读取上游 usage，缺失时后端本地估算；图片/视频按配置 request_cost 计费。",
			"pricing": h.cfg.AIGateway.Pricing,
		},
		"routes": []gin.H{
			{"method": "GET", "path": "/api/ai-gateway/docs", "description": "获取 API 文档"},
			{"method": "GET", "path": "/api/ai-gateway/catalog", "description": "获取可用模型目录"},
			{"method": "POST", "path": "/api/ai-gateway/admin/keys", "description": "超级管理员创建 API Key"},
			{"method": "GET", "path": "/api/ai-gateway/admin/keys", "description": "超级管理员查看 Key 列表"},
			{"method": "GET", "path": "/api/ai-gateway/admin/keys/:id", "description": "超级管理员查看 Key 详情和最近明细"},
			{"method": "POST", "path": "/api/ai-gateway/admin/keys/:id/revoke", "description": "超级管理员吊销 Key"},
			{"method": "GET", "path": "/api/ai-gateway/admin/logs", "description": "超级管理员查看请求明细"},
			{"method": "GET", "path": "/api/ai-gateway/admin/reports", "description": "超级管理员查看按天/月聚合报表"},
			{"method": "GET", "path": "/api/ai-gateway/admin/alerts", "description": "超级管理员查看预算阈值告警"},
			{"method": "POST", "path": "/api/ai-gateway/v1/chat/completions", "description": "统一文本模型接口"},
			{"method": "POST", "path": "/api/ai-gateway/v1/media/generations", "description": "统一图片/视频模型接口"},
			{"method": "GET", "path": "/api/ai-gateway/v1/media/tasks", "description": "当前 API Key 维度的媒体任务列表"},
			{"method": "GET", "path": "/api/ai-gateway/v1/media/tasks/:id", "description": "当前 API Key 维度的媒体任务详情"},
		},
		"examples": gin.H{
			"chat": gin.H{
				"model": "deepseek-v4-pro",
				"messages": []gin.H{
					{"role": "system", "content": "你是一个专业助手"},
					{"role": "user", "content": "请写一个 Go HTTP 服务的示例"},
				},
				"temperature":      0.7,
				"max_tokens":       1024,
				"reasoning_effort": "medium",
			},
			"media": gin.H{
				"model":        "qwen-image-2.0-pro",
				"prompt":       "一只橘猫穿宇航服，电影海报风格",
				"images":       []string{"data:image/png;base64,..."},
				"size":         "1328x1328",
				"auto_poll":    true,
				"wait_seconds": 30,
			},
		},
	})
}

// GetAnthropicDocs 返回 Anthropic 协议接入文档
func (h *AIGatewayHandler) GetAnthropicDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "Anthropic 协议接入文档",
		"summary": "通过 AI Gateway 使用 Anthropic 兼容协议调用 MiniMax 或 DashScope 上游，支持标准 Anthropic SDK。",
		"auth": gin.H{
			"api_key": "Authorization: Bearer dtk_ai_xxx",
		},
		"providers": []gin.H{
			{
				"name":        "MiniMax",
				"base_url":    "/api/minimax/anthropic",
				"upstream":    "https://api.minimaxi.com/anthropic",
				"models":      []string{"MiniMax-M2.7", "MiniMax-M2.5"},
				"description": "MiniMax Anthropic 兼容端点",
			},
			{
				"name":        "DashScope",
				"base_url":    "/api/dashscope/anthropic",
				"upstream":    "https://coding.dashscope.aliyuncs.com/apps/anthropic",
				"models":      []string{"qwen3.5-plus", "qwen3-max-2026-01-23", "qwen3-coder-next", "qwen3-coder-plus", "glm-5", "glm-4.7", "kimi-k2.5", "MiniMax-M2.5"},
				"description": "DashScope Anthropic 兼容端点（阿里云百炼）",
			},
		},
		"routes": []gin.H{
			{"method": "GET", "path": "/api/ai-gateway/docs/anthropic", "description": "获取本文档"},
			{"method": "POST", "path": "/api/minimax/anthropic/v1/messages", "description": "MiniMax Anthropic 接口"},
			{"method": "POST", "path": "/api/dashscope/anthropic/v1/messages", "description": "DashScope Anthropic 接口"},
		},
		"examples": gin.H{
			"minimax": gin.H{
				"description": "使用 MiniMax 上游",
				"request": gin.H{
					"model": "MiniMax-M2.5",
					"messages": []gin.H{
						{"role": "user", "content": "你好"},
					},
					"max_tokens": 1024,
				},
				"claude_code_config": gin.H{
					"language":    "Claude Code",
					"description": "Claude Code MiniMax 配置示例",
					"code": `{
  "skills": {
    "paths": [
      "~/.claude/skills"
    ]
  },
  "env": {
    "ANTHROPIC_BASE_URL": "https://your-devtools:8080/api/minimax/anthropic",
    "ANTHROPIC_AUTH_TOKEN": "dtk_ai_xxx",
    "API_TIMEOUT_MS": "300000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": 1,
    "ANTHROPIC_MODEL": "MiniMax-M2.7",
    "ANTHROPIC_SMALL_FAST_MODEL": "MiniMax-M2.7",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "MiniMax-M2.7",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "MiniMax-M2.7",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "MiniMax-M2.7"
  }
}`,
				},
			},
			"dashscope": gin.H{
				"description": "使用 DashScope 上游",
				"request": gin.H{
					"model": "qwen3.5-plus",
					"messages": []gin.H{
						{"role": "user", "content": "你好"},
					},
					"max_tokens": 1024,
				},
				"claude_code_config": gin.H{
					"language":    "Claude Code",
					"description": "Claude Code DashScope 配置示例",
					"code": `{
  "skills": {
    "paths": [
      "~/.claude/skills"
    ]
  },
  "env": {
    "ANTHROPIC_BASE_URL": "https://your-devtools:8080/api/dashscope/anthropic",
    "ANTHROPIC_AUTH_TOKEN": "dtk_ai_xxx",
    "API_TIMEOUT_MS": "300000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": 1,
    "ANTHROPIC_MODEL": "qwen3.5-plus",
    "ANTHROPIC_SMALL_FAST_MODEL": "qwen3-coder-next",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "qwen3.5-plus",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "qwen3-max-2026-01-23",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "qwen3-coder-plus"
  }
}`,
				},
			},
			"python_sdk": gin.H{
				"language": "Python",
				"code": `from anthropic import Anthropic

client = Anthropic(
    base_url="http://your-devtools:8080/api/minimax/anthropic/v1",
    api_key="dtk_ai_xxx"  # 你的 AI Gateway API Key
)

response = client.messages.create(
    model="MiniMax-M2.5",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}]
)
print(response.content[0].text)`,
			},
			"javascript_sdk": gin.H{
				"language": "JavaScript/TypeScript",
				"code": `import { Anthropic } from '@anthropic-ai/sdk';

const client = new Anthropic({
  baseURL: 'http://your-devtools:8080/api/minimax/anthropic/v1',
  apiKey: 'dtk_ai_xxx', // 你的 AI Gateway API Key
});

async function main() {
  const message = await client.messages.create({
    model: 'MiniMax-M2.5',
    max_tokens: 1024,
    messages: [{ role: 'user', content: 'Hello' }],
  });
  console.log(message.content[0].text);
}
main();`,
			},
			"curl": gin.H{
				"language": "cURL",
				"code": `curl -X POST http://your-devtools:8080/api/minimax/anthropic/v1/messages \\
  -H "Authorization: Bearer dtk_ai_xxx" \\
  -H "Content-Type: application/json" \\
  -H "Anthropic-Version: 2023-06-01" \\
  -d '{
    "model": "MiniMax-M2.5",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 1024
  }'`,
			},
		},
	})
}

func (h *AIGatewayHandler) GetCatalog(c *gin.Context) {
	catalog := make([]gin.H, 0)
	if h.cfg.DeepSeek.APIKey != "" {
		deepseekModels := []struct {
			model       string
			description string
		}{
			{fallbackString(h.cfg.DeepSeek.Model, "deepseek-chat"), "DeepSeek 默认文本模型（来自配置）"},
			{"deepseek-chat", "DeepSeek 通用聊天模型"},
			{"deepseek-reasoner", "DeepSeek 推理模型，返回 reasoning_content"},
			{"deepseek-v4-flash", "DeepSeek V4 Flash，偏向速度与成本"},
			{"deepseek-v4-pro", "DeepSeek V4 Pro，偏向质量与复杂任务"},
		}
		seen := map[string]bool{}
		for _, model := range deepseekModels {
			if seen[model.model] || strings.TrimSpace(model.model) == "" {
				continue
			}
			seen[model.model] = true
			catalog = append(catalog, gin.H{
				"model":       model.model,
				"provider":    "deepseek",
				"type":        "chat",
				"endpoint":    "/api/ai-gateway/v1/chat/completions",
				"description": model.description,
			})
		}
	}
	if h.cfg.MiniMax.APIKey != "" {
		catalog = append(catalog, gin.H{
			"model":       fallbackString(h.cfg.MiniMax.Model, "abab6.5s-chat"),
			"provider":    "minimax",
			"type":        "chat",
			"endpoint":    "/api/ai-gateway/v1/chat/completions",
			"description": "MiniMax 文本模型",
		})
	}
	if h.cfg.DashScope.APIKey != "" {
		dashscopeModels := []struct{ model, brand, caps string }{
			{"qwen3.5-plus", "千问", "文本生成、深度思考、视觉理解"},
			{"qwen3-max-2026-01-23", "千问", "文本生成、深度思考"},
			{"qwen3-coder-next", "千问", "文本生成"},
			{"qwen3-coder-plus", "千问", "文本生成"},
			{"glm-5", "智谱", "文本生成、深度思考"},
			{"glm-4.7", "智谱", "文本生成、深度思考"},
			{"kimi-k2.5", "Kimi", "文本生成、深度思考、视觉理解"},
		}
		for _, m := range dashscopeModels {
			catalog = append(catalog, gin.H{
				"model":       m.model,
				"provider":    "dashscope",
				"brand":       m.brand,
				"type":        "chat",
				"endpoint":    "/api/ai-gateway/v1/chat/completions",
				"description": m.brand + " · " + m.caps,
			})
		}
	}
	if h.hasProxyConfig() {
		proxyModels := h.getProxyModels()
		for _, model := range proxyModels {
			catalog = append(catalog, gin.H{
				"model":       model.Model,
				"provider":    "proxy",
				"type":        "chat",
				"endpoint":    "/api/ai-gateway/v1/chat/completions",
				"description": fallbackString(model.Description, "OpenAI 兼容代理文本模型"),
			})
		}
	}
	for _, model := range h.cfg.Bailian.Models {
		catalog = append(catalog, gin.H{
			"model":       model.Name,
			"provider":    "bailian",
			"type":        model.Type,
			"endpoint":    "/api/ai-gateway/v1/media/generations",
			"enabled":     model.Enabled,
			"description": model.Description,
			"expires_at":  model.ExpiresAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"models": catalog})
}

func (h *AIGatewayHandler) AdminCreateKey(c *gin.Context) {
	var req CreateAIAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整", "code": 400})
		return
	}
	if !h.requireSuperAdmin(c, req.SuperAdminPassword) {
		return
	}

	plainKey := "dtk_ai_" + utils.GenerateHexKey(20)
	keyHash, err := utils.HashPassword(plainKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 API Key 失败", "code": 500})
		return
	}
	prefix := plainKey[:18]
	expiresDays := req.ExpiresDays
	if expiresDays <= 0 {
		expiresDays = h.cfg.AIGateway.DefaultKeyExpiresDays
	}
	rateLimit := req.RateLimitPerHour
	if rateLimit <= 0 {
		rateLimit = h.cfg.AIGateway.DefaultRateLimitPerHour
	}
	allowedModels := req.AllowedModels
	if len(allowedModels) == 0 {
		allowedModels = []string{"*"}
	}
	allowedScopes := req.AllowedScopes
	if len(allowedScopes) == 0 {
		allowedScopes = []string{"chat", "media"}
	}

	expiresAt := time.Now().Add(time.Duration(expiresDays) * 24 * time.Hour)
	alertThreshold := req.AlertThreshold
	if alertThreshold <= 0 || alertThreshold > 1 {
		alertThreshold = 0.8
	}
	key := &models.AIAPIKey{
		Name:             req.Name,
		KeyPrefix:        prefix,
		KeyHash:          keyHash,
		Status:           "active",
		AllowedModels:    models.MustJSONString(allowedModels),
		AllowedScopes:    models.MustJSONString(allowedScopes),
		RateLimitPerHour: rateLimit,
		BudgetLimit:      req.BudgetLimit,
		AlertThreshold:   alertThreshold,
		ExpiresAt:        &expiresAt,
		CreatorIP:        c.ClientIP(),
		Notes:            req.Notes,
	}
	if err := h.db.CreateAIAPIKey(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存 API Key 失败", "code": 500})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"key":       key,
		"plain_key": plainKey,
		"warning":   "明文 API Key 只会返回这一次，请立即保存。",
	})
}

func (h *AIGatewayHandler) AdminListKeys(c *gin.Context) {
	if !h.requireSuperAdmin(c, "") {
		return
	}
	limit := boundedInt(c.Query("limit"), 20, 1, 100)
	offset := boundedInt(c.Query("offset"), 0, 0, 100000)
	keys, err := h.db.ListAIAPIKeys(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Key 列表失败", "code": 500})
		return
	}
	total, _ := h.db.CountAIAPIKeys()
	c.JSON(http.StatusOK, gin.H{"keys": keys, "total": total, "limit": limit, "offset": offset})
}

func (h *AIGatewayHandler) AdminGetKey(c *gin.Context) {
	if !h.requireSuperAdmin(c, "") {
		return
	}
	key, err := h.db.GetAIAPIKeyByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API Key 不存在", "code": 404})
		return
	}
	logs, _ := h.db.ListAIAPIRequestLogs(key.ID, 50, 0)
	c.JSON(http.StatusOK, gin.H{"key": key, "logs": logs})
}

func (h *AIGatewayHandler) AdminRevokeKey(c *gin.Context) {
	var req struct {
		SuperAdminPassword string `json:"super_admin_password"`
	}
	_ = c.ShouldBindJSON(&req)
	if !h.requireSuperAdmin(c, req.SuperAdminPassword) {
		return
	}
	key, err := h.db.GetAIAPIKeyByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API Key 不存在", "code": 404})
		return
	}
	key.Status = "revoked"
	if err := h.db.UpdateAIAPIKey(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "吊销失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "key": key})
}

func (h *AIGatewayHandler) AdminListLogs(c *gin.Context) {
	if !h.requireSuperAdmin(c, "") {
		return
	}
	logs, err := h.db.ListAIAPIRequestLogs(c.Query("api_key_id"), boundedInt(c.Query("limit"), 50, 1, 200), 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取请求日志失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (h *AIGatewayHandler) AdminReports(c *gin.Context) {
	if !h.requireSuperAdmin(c, "") {
		return
	}
	groupBy := c.DefaultQuery("group_by", "day")
	if groupBy != "day" && groupBy != "month" {
		groupBy = "day"
	}
	days := boundedInt(c.Query("days"), 30, 1, 3650)
	rows, err := h.db.GetAIUsageReport(groupBy, c.Query("api_key_id"), time.Now().Add(-time.Duration(days)*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取报表失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rows": rows, "group_by": groupBy, "days": days})
}

// AdminTestModel 使用服务端配置直连上游，测试指定模型可用性（需要超级管理员密码）
// POST /api/ai-gateway/admin/test-model
func (h *AIGatewayHandler) AdminTestModel(c *gin.Context) {
	var req struct {
		SuperAdminPassword string `json:"super_admin_password"`
		Model              string `json:"model" binding:"required"`
		Prompt             string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model 字段"})
		return
	}
	if !h.requireSuperAdmin(c, req.SuperAdminPassword) {
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = "你好，请用一句话介绍你自己（包含你的模型名）。"
	}
	chatReq := ChatCompletionRequest{
		Model:    req.Model,
		Messages: []map[string]interface{}{{"role": "user", "content": prompt}},
	}
	start := time.Now()
	result, _, err := h.executeChatRequest(chatReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"model":   req.Model,
			"status":  "error",
			"error":   err.Error(),
			"latency": latency,
		})
		return
	}
	content, _ := result["content"].(string)
	usage, _ := result["usage_summary"].(gin.H)
	tokens := 0
	if usage != nil {
		if t, ok := usage["total_tokens"].(int); ok {
			tokens = t
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"model":   req.Model,
		"status":  "ok",
		"reply":   content,
		"latency": latency,
		"tokens":  tokens,
	})
}

func (h *AIGatewayHandler) AdminAlerts(c *gin.Context) {
	if !h.requireSuperAdmin(c, "") {
		return
	}
	keys, err := h.db.ListAIAPIKeys(500, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取告警失败", "code": 500})
		return
	}
	alerts := make([]gin.H, 0)
	for _, key := range keys {
		if key.BudgetLimit <= 0 {
			continue
		}
		ratio := key.TotalCost / key.BudgetLimit
		if ratio >= key.AlertThreshold {
			level := "warning"
			if ratio >= 1 {
				level = "critical"
			}
			alerts = append(alerts, gin.H{
				"api_key_id":      key.ID,
				"name":            key.Name,
				"total_cost":      key.TotalCost,
				"budget_limit":    key.BudgetLimit,
				"alert_threshold": key.AlertThreshold,
				"usage_ratio":     ratio,
				"currency":        key.BillingCurrency,
				"level":           level,
				"total_requests":  key.TotalRequests,
				"total_tokens":    key.TotalTokens,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (h *AIGatewayHandler) ChatCompletions(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "chat")
	if !ok {
		return
	}

	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整", "code": 400})
		return
	}
	if !h.ensureModelAllowed(c, key, req.Model) {
		return
	}

	start := time.Now()
	statusCode := http.StatusOK
	rawRequest := sanitizeJSON(req)
	var responseBody string
	var success bool
	var errMessage string
	provider := h.resolveChatProvider(req.Model)

	result, rawMap, err := h.executeChatRequest(req)
	usage := h.buildChatUsage(req, rawMap, provider)
	if err != nil {
		statusCode = http.StatusBadGateway
		errMessage = err.Error()
		c.JSON(statusCode, gin.H{"error": errMessage, "code": statusCode})
	} else {
		success = true
		responseBody = sanitizeJSON(rawMap)
		result["usage_summary"] = gin.H{
			"input_tokens":   usage.InputTokens,
			"output_tokens":  usage.OutputTokens,
			"total_tokens":   usage.TotalTokens,
			"estimated_cost": usage.Cost,
			"currency":       usage.Currency,
		}
		c.JSON(http.StatusOK, result)
	}

	h.logAPIRequest(key, req.Model, provider, "/api/ai-gateway/v1/chat/completions", "chat", statusCode, success, errMessage, rawRequest, responseBody, c.ClientIP(), time.Since(start), usage)
}

func (h *AIGatewayHandler) MediaGenerations(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "media")
	if !ok {
		return
	}

	var req MediaGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整", "code": 400})
		return
	}
	if !h.ensureModelAllowed(c, key, req.Model) {
		return
	}

	start := time.Now()
	statusCode := http.StatusOK
	responseBody := ""
	success := false
	errMessage := ""

	result, sc, err := h.bailian.ExecuteTask(CreateBailianTaskRequest{
		Model:           req.Model,
		Prompt:          req.Prompt,
		NegativePrompt:  req.NegativePrompt,
		Image:           req.Image,
		Images:          req.Images,
		Size:            req.Size,
		Count:           req.Count,
		Seed:            req.Seed,
		Watermark:       req.Watermark,
		Duration:        req.Duration,
		Resolution:      req.Resolution,
		FPS:             req.FPS,
		AutoPoll:        req.AutoPoll,
		WaitSeconds:     req.WaitSeconds,
		ClientName:      "api-key:" + key.ID + ":" + req.ClientName,
		ClientRequestID: req.ClientRequestID,
		Parameters:      req.Parameters,
	}, "ai-gateway", c.ClientIP())
	statusCode = sc
	usage := h.buildMediaUsage(req.Model)
	if result != nil {
		responseBody = sanitizeJSON(result)
	}
	if err != nil {
		errMessage = err.Error()
		c.JSON(statusCode, gin.H{"error": errMessage, "code": statusCode, "result": result})
	} else {
		success = true
		c.JSON(http.StatusOK, result)
	}

	h.logAPIRequest(key, req.Model, "bailian", "/api/ai-gateway/v1/media/generations", "media", statusCode, success, errMessage, sanitizeJSON(req), responseBody, c.ClientIP(), time.Since(start), usage)
}

func (h *AIGatewayHandler) ListMediaTasks(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	tasks, err := h.db.ListBailianTasksByClientPrefix(
		boundedInt(c.Query("limit"), 20, 1, 100),
		boundedInt(c.Query("offset"), 0, 0, 100000),
		"api-key:"+key.ID+":",
		c.Query("model"),
		c.Query("status"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (h *AIGatewayHandler) GetMediaTask(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	task, err := h.db.GetBailianTask(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}
	if !strings.HasPrefix(task.ClientName, "api-key:"+key.ID+":") {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该任务", "code": 403})
		return
	}
	events, _ := h.db.ListBailianTaskEvents(task.ID)
	c.JSON(http.StatusOK, gin.H{"task": task, "events": events})
}

// AsyncChatCompletions 异步聊天接口：立即返回 task_id，后台调用 LLM，避免 Cloudflare 524
// POST /api/ai-gateway/v1/chat/tasks
func (h *AIGatewayHandler) AsyncChatCompletions(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "chat")
	if !ok {
		return
	}
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整", "code": 400})
		return
	}
	if !h.ensureModelAllowed(c, key, req.Model) {
		return
	}

	provider := h.resolveChatProvider(req.Model)
	taskID := "ltask_" + utils.GenerateHexKey(12)
	reqJSON, _ := json.Marshal(req)

	task := &models.LLMTask{
		ID:          taskID,
		APIKeyID:    key.ID,
		Model:       req.Model,
		Provider:    provider,
		Status:      "pending",
		RequestBody: truncateString(string(reqJSON), 50000),
		ClientIP:    c.ClientIP(),
	}
	if err := h.db.CreateLLMTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败", "code": 500})
		return
	}

	go h.runAsyncChatTask(taskID, req)

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"status":  "pending",
		"model":   req.Model,
	})
}

// GetChatTask 查询异步聊天任务状态
// GET /api/ai-gateway/v1/chat/tasks/:id
func (h *AIGatewayHandler) GetChatTask(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "chat")
	if !ok {
		return
	}
	task, err := h.db.GetLLMTask(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}
	if task.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该任务", "code": 403})
		return
	}

	resp := gin.H{
		"task_id":    task.ID,
		"status":     task.Status,
		"model":      task.Model,
		"provider":   task.Provider,
		"created_at": task.CreatedAt,
	}
	if task.CompletedAt != nil {
		resp["completed_at"] = task.CompletedAt
	}
	if task.Status == "succeeded" && task.ResultJSON != "" {
		var result interface{}
		if json.Unmarshal([]byte(task.ResultJSON), &result) == nil {
			resp["result"] = result
		}
	}
	if task.Status == "failed" {
		resp["error"] = task.ErrorMessage
	}
	c.JSON(http.StatusOK, resp)
}

// runAsyncChatTask 后台执行 LLM 调用，更新任务状态
func (h *AIGatewayHandler) runAsyncChatTask(taskID string, req ChatCompletionRequest) {
	task, err := h.db.GetLLMTask(taskID)
	if err != nil {
		return
	}
	task.Status = "running"
	_ = h.db.UpdateLLMTask(task)

	result, _, err := h.executeChatRequest(req)
	now := time.Now()
	task.CompletedAt = &now
	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
	} else {
		task.Status = "succeeded"
		if data, jsonErr := json.Marshal(result); jsonErr == nil {
			task.ResultJSON = string(data)
		}
	}
	_ = h.db.UpdateLLMTask(task)
}

const internalQwenVisionKeyID = "internal:qwen-vision"
const internalChatKeyID = "internal:chat"

// InternalChat 内部免 API Key 聊天接口，仅限同域浏览器调用
// POST /api/internal/chat
func (h *AIGatewayHandler) InternalChat(c *gin.Context) {
	if !requireSameOrigin(c) {
		return
	}
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数解析失败"})
		return
	}
	if req.Model == "" {
		req.Model = fallbackString(h.cfg.MiniMax.Model, "MiniMax-M2.5")
	}

	start := time.Now()
	result, _, err := h.executeChatRequest(req)
	if err != nil {
		_ = h.db.CreateAIAPIRequestLog(&models.AIAPIRequestLog{
			APIKeyID: internalChatKeyID, Model: req.Model, Provider: "minimax",
			Endpoint: "/api/internal/chat", RequestType: "chat",
			StatusCode: http.StatusBadGateway, Success: false, ErrorMessage: err.Error(),
			ClientIP: c.ClientIP(), LatencyMS: time.Since(start).Milliseconds(),
		})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	_ = h.db.CreateAIAPIRequestLog(&models.AIAPIRequestLog{
		APIKeyID: internalChatKeyID, Model: req.Model, Provider: "minimax",
		Endpoint: "/api/internal/chat", RequestType: "chat",
		StatusCode: http.StatusOK, Success: true,
		ClientIP: c.ClientIP(), LatencyMS: time.Since(start).Milliseconds(),
	})
	c.JSON(http.StatusOK, result)
}

// requireSameOrigin 校验请求来源是否来自当前服务器（同域浏览器请求）
// 非浏览器直接调用（无 Origin/Referer）会被拒绝，防止外部直接访问内部免密钥接口
func requireSameOrigin(c *gin.Context) bool {
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	referer := strings.TrimSpace(c.GetHeader("Referer"))
	source := origin
	if source == "" {
		source = referer
	}
	if source == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "此接口仅限项目内部调用，不允许外部直接请求", "code": 403})
		return false
	}
	parsed, err := url.Parse(source)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "无效的请求来源", "code": 403})
		return false
	}
	sourceHost := parsed.Hostname()
	// 允许 localhost / 127.0.0.1 / ::1（开发模式）
	if sourceHost == "localhost" || sourceHost == "127.0.0.1" || sourceHost == "::1" {
		return true
	}
	// 允许与当前请求 host 相同的来源
	reqHost := c.Request.Host
	if idx := strings.LastIndex(reqHost, ":"); idx >= 0 {
		reqHost = reqHost[:idx]
	}
	if sourceHost == reqHost {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "此接口仅限项目内部调用，不允许外部直接请求", "code": 403})
	return false
}

// InternalQwenVision 内部免 API Key 图像理解接口（qwen3.5-plus 视觉能力）
// POST /api/image-understanding/qwen-vision
func (h *AIGatewayHandler) InternalQwenVision(c *gin.Context) {
	if !requireSameOrigin(c) {
		return
	}

	var req struct {
		Images []string `json:"images"` // 多图，每项可为 base64 data URL 或 HTTP URL
		Image  string   `json:"image"`  // 单图兼容，合并入 images
		Prompt string   `json:"prompt"`
		Model  string   `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数解析失败"})
		return
	}
	// 兼容单图字段
	if strings.TrimSpace(req.Image) != "" {
		req.Images = append([]string{req.Image}, req.Images...)
	}
	// 去空
	imgs := make([]string, 0, len(req.Images))
	for _, img := range req.Images {
		if strings.TrimSpace(img) != "" {
			imgs = append(imgs, strings.TrimSpace(img))
		}
	}
	if len(imgs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请至少提供一张图片（image 或 images）"})
		return
	}
	if len(imgs) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片数量不能超过 10 张"})
		return
	}
	if strings.TrimSpace(h.cfg.DashScope.APIKey) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "未配置 DashScope API Key，请联系管理员"})
		return
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = "qwen3.5-plus"
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}

	// 处理每张图：HTTP URL 下载转 base64，data URL 直接使用
	logSources := make([]string, 0, len(imgs))
	contentParts := make([]map[string]interface{}, 0, len(imgs)+1)
	totalSizeKB := 0
	for i, img := range imgs {
		if strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://") {
			logSources = append(logSources, fmt.Sprintf("[%d]url:%s", i+1, truncateString(img, 80)))
			downloaded, mimeType, dlErr := fetchImageAsBase64(h.client, img)
			if dlErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("图片 %d 下载失败: %s", i+1, dlErr.Error())})
				return
			}
			img = "data:" + mimeType + ";base64," + downloaded
		} else {
			mime, sizeKB := extractImageMeta(img)
			totalSizeKB += sizeKB
			logSources = append(logSources, fmt.Sprintf("[%d]%s~%dKB", i+1, mime, sizeKB))
		}
		contentParts = append(contentParts, map[string]interface{}{
			"type":      "image_url",
			"image_url": map[string]interface{}{"url": img},
		})
	}
	contentParts = append(contentParts, map[string]interface{}{"type": "text", "text": prompt})

	chatReq := ChatCompletionRequest{
		Model: model,
		Messages: []map[string]interface{}{
			{"role": "user", "content": contentParts},
		},
	}

	start := time.Now()
	result, _, err := h.callDashScope(chatReq)
	latency := time.Since(start)

	statusCode := http.StatusOK
	success := err == nil
	errMsg := ""
	content := ""
	if err != nil {
		statusCode = http.StatusBadGateway
		errMsg = err.Error()
	} else {
		content, _ = result["content"].(string)
	}

	// 记录请求流水
	reqInfo, _ := json.Marshal(map[string]interface{}{
		"prompt":    prompt,
		"model":     model,
		"images":    logSources,
		"total_kb":  totalSizeKB,
		"img_count": len(imgs),
	})
	_ = h.db.CreateAIAPIRequestLog(&models.AIAPIRequestLog{
		APIKeyID:     internalQwenVisionKeyID,
		Model:        model,
		Provider:     "dashscope",
		Endpoint:     "/api/image-understanding/qwen-vision",
		RequestType:  "vision",
		StatusCode:   statusCode,
		Success:      success,
		ErrorMessage: errMsg,
		RequestBody:  string(reqInfo),
		ResponseBody: truncateString(content, 5000),
		ClientIP:     c.ClientIP(),
		LatencyMS:    latency.Milliseconds(),
	})

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"text":   content,
		"model":  model,
		"result": result,
	})
}

// AdminListQwenVisionLogs 管理员查看 Qwen 视觉理解请求流水
// GET /api/image-understanding/qwen-vision/logs
// 认证: X-Image-Admin-Password header 或 admin_password query（独立于 super_admin_password）
func (h *AIGatewayHandler) AdminListQwenVisionLogs(c *gin.Context) {
	if !h.requireImageUnderstandingAdmin(c) {
		return
	}
	limit := boundedInt(c.DefaultQuery("limit", "50"), 50, 1, 200)
	offset := boundedInt(c.DefaultQuery("offset", "0"), 0, 0, 1000000)
	logs, err := h.db.ListInternalRequestLogs(internalQwenVisionKeyID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取流水失败"})
		return
	}
	total, _ := h.db.CountInternalRequestLogs(internalQwenVisionKeyID)
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total, "limit": limit, "offset": offset})
}

func (h *AIGatewayHandler) executeChatRequest(req ChatCompletionRequest) (gin.H, map[string]interface{}, error) {
	provider := h.resolveChatProvider(req.Model)
	switch provider {
	case "dashscope":
		return h.callDashScope(req)
	case "deepseek":
		return h.callDeepSeek(req)
	case "minimax":
		return h.callMiniMax(req)
	case "proxy":
		return h.callProxyChat(req)
	default:
		return nil, nil, fmt.Errorf("不支持的文本模型: %s", req.Model)
	}
}

func (h *AIGatewayHandler) callDashScope(req ChatCompletionRequest) (gin.H, map[string]interface{}, error) {
	if strings.TrimSpace(h.cfg.DashScope.APIKey) == "" {
		return nil, nil, fmt.Errorf("未配置 dashscope.api_key 或 DASHSCOPE_API_KEY")
	}
	bodyMap := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		bodyMap["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}

	baseURL := fallbackString(h.cfg.DashScope.BaseURL, "https://coding.dashscope.aliyuncs.com/v1")
	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	raw, err := h.doJSONRequest(endpoint, h.cfg.DashScope.APIKey, bodyMap)
	if err != nil {
		return nil, raw, err
	}
	content := extractString(raw, "choices", "0", "message", "content")
	if content == "" {
		content = extractMessageContentFromChoices(raw["choices"])
	}
	result := gin.H{
		"id":           fallbackString(extractString(raw, "id"), "chatcmpl-"+utils.GenerateHexKey(4)),
		"object":       "chat.completion",
		"created":      time.Now().Unix(),
		"model":        req.Model,
		"provider":     "dashscope",
		"choices":      raw["choices"],
		"usage":        raw["usage"],
		"content":      content,
		"raw_response": raw,
	}
	return result, raw, nil
}

func (h *AIGatewayHandler) callDeepSeek(req ChatCompletionRequest) (gin.H, map[string]interface{}, error) {
	bodyMap := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		bodyMap["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}
	if req.Stop != nil {
		bodyMap["stop"] = req.Stop
	}
	if len(req.ResponseFormat) > 0 {
		bodyMap["response_format"] = req.ResponseFormat
	}
	if len(req.Tools) > 0 {
		bodyMap["tools"] = req.Tools
	}
	if req.ToolChoice != nil {
		bodyMap["tool_choice"] = req.ToolChoice
	}
	if strings.TrimSpace(req.ReasoningEffort) != "" {
		bodyMap["reasoning_effort"] = strings.TrimSpace(req.ReasoningEffort)
	}
	for key, value := range req.ExtraBody {
		bodyMap[key] = value
	}

	raw, err := h.doJSONRequest(deepseekChatCompletionURL(req.Messages), h.cfg.DeepSeek.APIKey, bodyMap)
	if err != nil {
		return nil, raw, err
	}
	content := extractString(raw, "choices", "0", "message", "content")
	if content == "" {
		content = extractMessageContentFromChoices(raw["choices"])
	}
	reasoningContent := extractReasoningContentFromChoices(raw["choices"])
	result := gin.H{
		"id":           fallbackString(extractString(raw, "id"), "chatcmpl-"+utils.GenerateHexKey(4)),
		"object":       "chat.completion",
		"created":      time.Now().Unix(),
		"model":        req.Model,
		"provider":     "deepseek",
		"choices":      raw["choices"],
		"usage":        raw["usage"],
		"content":      content,
		"raw_response": raw,
	}
	if reasoningContent != "" {
		result["reasoning_content"] = reasoningContent
	}
	return result, raw, nil
}

func (h *AIGatewayHandler) callMiniMax(req ChatCompletionRequest) (gin.H, map[string]interface{}, error) {
	bodyMap := map[string]interface{}{
		"model":      req.Model,
		"messages":   req.Messages,
		"max_tokens": valueOrDefaultInt(req.MaxTokens, 1024),
	}
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}

	raw, err := h.doJSONRequest("https://api.minimaxi.com/anthropic/v1/messages", h.cfg.MiniMax.APIKey, bodyMap)
	if err != nil {
		return nil, raw, err
	}
	content := extractMiniMaxText(raw)
	result := gin.H{
		"id":       fallbackString(extractString(raw, "id"), "chatcmpl-"+utils.GenerateHexKey(4)),
		"object":   "chat.completion",
		"created":  time.Now().Unix(),
		"model":    req.Model,
		"provider": "minimax",
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"content":      content,
		"raw_response": raw,
	}
	return result, raw, nil
}

// ProxyMinimaxAnthropic 转发 Anthropic 协议格式的请求到 MiniMax Anthropic 兼容端点
// POST /api/minimax/anthropic/v1/messages
func (h *AIGatewayHandler) ProxyMinimaxAnthropic(c *gin.Context) {
	h.proxyAnthropic(c, "https://api.minimaxi.com/anthropic", h.cfg.MiniMax.APIKey, "/api/minimax/anthropic/v1/messages", []string{"MiniMax-M2.7", "MiniMax-M2.5"})
}

// ProxyDashScopeAnthropic 转发 Anthropic 协议格式的请求到 DashScope Anthropic 兼容端点
// POST /api/dashscope/anthropic/v1/messages
func (h *AIGatewayHandler) ProxyDashScopeAnthropic(c *gin.Context) {
	h.proxyAnthropic(c, "https://coding.dashscope.aliyuncs.com/apps/anthropic", h.cfg.DashScope.APIKey, "/api/dashscope/anthropic/v1/messages", []string{"qwen3.5-plus", "qwen3-max-2026-01-23", "qwen3-coder-next", "qwen3-coder-plus", "glm-5", "glm-4.7", "kimi-k2.5", "MiniMax-M2.5"})
}

// ProxyMinimaxTTS 转发 TTS 请求到 MiniMax TTS 端点
// POST /api/minimax/tts/v1/generations
func (h *AIGatewayHandler) ProxyMinimaxTTS(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	// 直接透传请求体到上游
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	if len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空"})
		return
	}

	// 解析 model 字段用于校验和日志
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体 JSON 格式错误"})
		return
	}

	model, _ := bodyMap["model"].(string)
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model 字段"})
		return
	}

	// MiniMax 音乐模型优先返回可直接播放/下载的 URL，避免默认十六进制音频结果无法在页面里直接使用。
	if strings.HasPrefix(model, "music-") {
		if _, exists := bodyMap["output_format"]; !exists {
			bodyMap["output_format"] = "url"
			bodyBytes, _ = json.Marshal(bodyMap)
		}
	}

	// 校验模型是否在允许列表中
	if !isModelAllowed(model, TTSAllowedModels) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("该端点不支持模型 %s，允许的模型: %v", model, TTSAllowedModels)})
		return
	}

	if key != nil && !h.ensureModelAllowed(c, key, model) {
		return
	}

	apiKey := h.cfg.MiniMaxTTS.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey // fallback to MiniMax APIKey
	}

	baseURL := h.cfg.MiniMaxTTS.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax TTS API Key"})
		return
	}

	// 转换请求格式：MiniMax TTS API 要求 voice_setting.voice_id 格式
	// 客户端可能发送 voice: "xxx"，需要转换为 voice_setting: {voice_id: "xxx"}
	upstreamReq := make(map[string]interface{})
	upstreamReq["model"] = model
	if text, ok := bodyMap["text"].(string); ok {
		upstreamReq["text"] = text
	}
	if voice, ok := bodyMap["voice"].(string); ok && voice != "" {
		if upstreamReq["voice_setting"] == nil {
			upstreamReq["voice_setting"] = map[string]interface{}{}
		}
		if vs, ok := upstreamReq["voice_setting"].(map[string]interface{}); ok {
			vs["voice_id"] = voice
		}
	}
	if speed, ok := bodyMap["speed"]; ok {
		if upstreamReq["voice_setting"] == nil {
			upstreamReq["voice_setting"] = map[string]interface{}{}
		}
		if vs, ok := upstreamReq["voice_setting"].(map[string]interface{}); ok {
			vs["speed"] = speed
		}
	}
	// 透传 audio_format 和 sample_rate 到 audio_setting
	if af, ok := bodyMap["audio_format"].(string); ok && af != "" {
		upstreamReq["audio_setting"] = map[string]interface{}{"audio_format": af}
		if sr, ok := bodyMap["sample_rate"].(float64); ok {
			if as, ok := upstreamReq["audio_setting"].(map[string]interface{}); ok {
				as["sample_rate"] = sr
			}
		}
	}
	// 其他字段直接透传
	for k, v := range bodyMap {
		if k != "voice" && k != "speed" && k != "audio_format" && k != "sample_rate" {
			upstreamReq[k] = v
		}
	}

	upstreamBytes, _ := json.Marshal(upstreamReq)

	start := time.Now()
	upstreamURL := strings.TrimRight(baseURL, "/") + "/v1/t2a_v2"

	respBody, _, err := h.doRawRequestWithResp(upstreamURL, apiKey, "POST", upstreamBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/tts/v1/generations", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// 计算使用量（按字符数计费）
	textLen := len(interfaceToString(bodyMap["text"]))
	usage := usageSummary{Cost: float64(textLen) * 0.001, Currency: "CNY"} // 估算成本

	// MiniMax TTS 返回格式：{"data":{"audio":"base64..."}} 或 {"base_resp":{"status_code":xxx,"status_msg":"..."}}
	var respData map[string]interface{}
	if err := json.Unmarshal(respBody, &respData); err != nil {
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/tts/v1/generations", "media", http.StatusBadGateway, false, "上游响应解析失败: "+err.Error(), string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
		c.JSON(http.StatusBadGateway, gin.H{"error": "上游响应解析失败"})
		return
	}

	// 检查是否有业务错误
	if baseResp, ok := respData["base_resp"].(map[string]interface{}); ok {
		if code, ok := baseResp["status_code"].(float64); ok && int(code) != 0 {
			msg, _ := baseResp["status_msg"].(string)
			h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/tts/v1/generations", "media", http.StatusBadGateway, false, msg, string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
			c.JSON(http.StatusBadGateway, gin.H{"error": msg})
			return
		}
	}

	// 提取 base64 音频数据
	var audioData string
	if data, ok := respData["data"].(map[string]interface{}); ok {
		if audio, ok := data["audio"].(string); ok {
			audioData = audio
		}
	}

	if audioData == "" {
		// 没有音频数据，返回错误
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/tts/v1/generations", "media", http.StatusBadGateway, false, "未获取到音频数据", string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
		c.JSON(http.StatusBadGateway, gin.H{"error": "未获取到音频数据"})
		return
	}

	// 解码 base64 音频
	audioBytes, err := base64.StdEncoding.DecodeString(audioData)
	if err != nil {
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/tts/v1/generations", "media", http.StatusBadGateway, false, "音频base64解码失败: "+err.Error(), string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
		c.JSON(http.StatusBadGateway, gin.H{"error": "音频数据解码失败"})
		return
	}

	h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/tts/v1/generations", "media", http.StatusOK, true, "", string(bodyBytes), "[base64 audio]", c.ClientIP(), time.Since(start), usage)

	// 返回音频二进制
	c.DataFromReader(http.StatusOK, int64(len(audioBytes)), "audio/mpeg", bytes.NewReader(audioBytes), nil)
}

// GetTTSDocs 返回 TTS 端点的 API 文档
// GET /api/minimax/tts/docs
func (h *AIGatewayHandler) GetTTSDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "MiniMax TTS 接口文档",
		"summary": "通过 AI Gateway 调用 MiniMax Text to Speech HD 模型，支持语音合成。",
		"auth": gin.H{
			"api_key": "Authorization: Bearer dtk_ai_xxx",
			"scope":   "media",
		},
		"base_url": "/api/minimax/tts",
		"upstream": "https://api.minimaxi.com/v1/t2a_v2",
		"models":   TTSAllowedModels,
		"routes": []gin.H{
			{"method": "GET", "path": "/api/minimax/tts/docs", "description": "获取本文档"},
			{"method": "POST", "path": "/api/minimax/tts/v1/generations", "description": "MiniMax TTS 接口"},
		},
		"examples": gin.H{
			"request": gin.H{
				"model":        "speech-2.8-hd",
				"text":         "你好，这是语音合成测试",
				"voice":        "shanghai",
				"speed":        1.0,
				"audio_format": "mp3",
			},
			"curl": gin.H{
				"language": "cURL",
				"code": `curl -X POST https://your-devtools:8080/api/minimax/tts/v1/generations \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "text": "你好，这是语音合成测试",
    "voice": "shanghai",
    "speed": 1.0,
    "audio_format": "mp3"
  }' \
  --output audio.mp3`,
			},
		},
		"voice_ids": gin.H{
			"description":  "有效的 voice_id 列表（根据实际测试）",
			"valid_voices": []string{"shanghai", "woman", "man", "cantonese", "cantonese_male"},
		},
		"model_descriptions": gin.H{
			"speech-01-hd":     "高清语音合成（speech-01 系列）",
			"speech-01-turbo":  "标准语音合成（speech-01 系列 turbo 版）",
			"speech-02-hd":     "高清语音合成（speech-02 系列）",
			"speech-02-turbo":  "标准语音合成（speech-02 系列 turbo 版）",
			"speech-2.6-hd":    "高清语音合成（speech-2.6 系列）",
			"speech-2.6-turbo": "标准语音合成（speech-2.6 系列 turbo 版）",
			"speech-2.8-hd":    "高清语音合成（speech-2.8 系列，推荐）",
			"speech-2.8-turbo": "标准语音合成（speech-2.8 系列 turbo 版）",
		},
	})
}

// resolveTokenPlanModelEndpoint 根据模型返回对应的上游 API 路径
func resolveTokenPlanModelEndpoint(model string) string {
	switch model {
	// TTS - 正确端点: /v1/t2a_v2
	case "speech-01-hd", "speech-01-turbo",
		"speech-02-hd", "speech-02-turbo",
		"speech-2.6-hd", "speech-2.6-turbo",
		"speech-2.8-hd", "speech-2.8-turbo":
		return "/v1/t2a_v2"
	// Hailuo 视频 - 正确端点: /v1/video_generation
	// 官方模型名: MiniMax-Hailuo-2.3-Fast, MiniMax-Hailuo-2.3, MiniMax-Hailuo-02, T2V-01-Director, T2V-01
	case "MiniMax-Hailuo-2.3-Fast", "MiniMax-Hailuo-2.3", "MiniMax-Hailuo-02",
		"T2V-01-Director", "T2V-01":
		return "/v1/video_generation"
	// Music - 正确端点: /v1/music_generation
	case "music-2.5", "music-2.6", "music-cover":
		return "/v1/music_generation"
	// Image - 正确端点: /v1/image_generation
	case "image-01", "image-01-live":
		return "/v1/image_generation"
	default:
		return ""
	}
}

// ProxyMinimaxTokenPlan 转发 Token Plan 请求到 MiniMax Token Plan 端点
// POST /api/minimax/token-plan/v1/generations
func (h *AIGatewayHandler) ProxyMinimaxTokenPlan(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	// 直接透传请求体到上游
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	if len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空"})
		return
	}

	// 解析 model 字段用于校验和日志
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体 JSON 格式错误"})
		return
	}

	model, _ := bodyMap["model"].(string)
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model 字段"})
		return
	}

	if model == "image-01" {
		if _, exists := bodyMap["style"]; exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image-01 不支持 style 参数，请改用 response_format / subject_reference 等官方字段", "code": 400})
			return
		}
		if _, exists := bodyMap["response_format"]; !exists {
			bodyMap["response_format"] = "url"
		}
		if count, exists := bodyMap["count"]; exists {
			if _, hasN := bodyMap["n"]; !hasN {
				bodyMap["n"] = count
			}
			delete(bodyMap, "count")
		}
		if size, ok := bodyMap["size"].(string); ok && size != "" {
			if _, hasAspect := bodyMap["aspect_ratio"]; !hasAspect {
				aspectRatio, supported := minimaxImageAspectRatioFromSize(size)
				if !supported {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("image-01 不支持尺寸 %s，请使用官方支持的尺寸", size), "code": 400})
					return
				}
				bodyMap["aspect_ratio"] = aspectRatio
			}
			delete(bodyMap, "size")
		}
		bodyBytes, _ = json.Marshal(bodyMap)
	}

	// MiniMax-Hailuo-2.3-Fast 是图生视频模型，兼容前端已有的 image 字段。
	if model == "MiniMax-Hailuo-2.3-Fast" {
		if firstFrame := interfaceToString(bodyMap["first_frame_image"]); strings.TrimSpace(firstFrame) == "" {
			if image := strings.TrimSpace(interfaceToString(bodyMap["image"])); image != "" {
				bodyMap["first_frame_image"] = image
				delete(bodyMap, "image")
				bodyBytes, _ = json.Marshal(bodyMap)
			}
		}
	}

	// 音乐模型默认要求 URL 输出，避免返回不可直接播放/分享的十六进制音频内容。
	if strings.HasPrefix(model, "music-") {
		if _, exists := bodyMap["output_format"]; !exists {
			bodyMap["output_format"] = "url"
			bodyBytes, _ = json.Marshal(bodyMap)
		}
	}

	// 校验模型是否在允许列表中
	if !isModelAllowed(model, TokenPlanAllowedModels) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("该端点不支持模型 %s，允许的模型: %v", model, TokenPlanAllowedModels)})
		return
	}

	if key != nil && !h.ensureModelAllowed(c, key, model) {
		return
	}

	apiKey := h.cfg.MiniMaxTokenPlan.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey // fallback to MiniMax APIKey
	}

	baseURL := h.cfg.MiniMaxTokenPlan.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax Token Plan API Key"})
		return
	}

	// 根据模型确定上游路径
	upstreamPath := resolveTokenPlanModelEndpoint(model)
	if upstreamPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("模型 %s 未知的 API 路径", model)})
		return
	}

	start := time.Now()
	upstreamURL := strings.TrimRight(baseURL, "/") + upstreamPath

	// 判断是否为异步模型（视频、音乐、图片生成需要轮询）
	if isTokenPlanAsyncModel(model) {
		h.handleAsyncTokenPlanRequest(c, key, model, apiKey, baseURL, upstreamURL, bodyBytes, bodyMap, start)
		return
	}

	// 同步模型（TTS）：直接透传请求
	h.handleSyncTokenPlanRequest(c, key, model, apiKey, upstreamURL, bodyBytes, bodyMap, start)
}

// handleSyncTokenPlanRequest 处理同步 Token Plan 请求（TTS）
func (h *AIGatewayHandler) handleSyncTokenPlanRequest(c *gin.Context, key *models.AIAPIKey, model, apiKey, upstreamURL string, bodyBytes []byte, bodyMap map[string]interface{}, start time.Time) {
	respBody, respContentType, err := h.doRawRequestWithResp(upstreamURL, apiKey, "POST", bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	usage := h.buildMediaUsage(model)
	logRespBody := ""
	if !strings.HasPrefix(respContentType, "audio/") {
		logRespBody = string(respBody)
	}
	h.logAPIRequest(key, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusOK, true, "", string(bodyBytes), logRespBody, c.ClientIP(), time.Since(start), usage)

	if strings.HasPrefix(respContentType, "audio/") {
		c.DataFromReader(http.StatusOK, int64(len(respBody)), respContentType, bytes.NewReader(respBody), nil)
	} else {
		c.Data(http.StatusOK, respContentType, respBody)
	}
}

// handleAsyncTokenPlanRequest 处理异步 Token Plan 请求（视频/音乐/图片生成）
func (h *AIGatewayHandler) handleAsyncTokenPlanRequest(c *gin.Context, key *models.AIAPIKey, model, apiKey, baseURL, upstreamURL string, bodyBytes []byte, bodyMap map[string]interface{}, start time.Time) {
	// 创建本地任务记录
	taskID := "mmt_" + utils.GenerateHexKey(12)
	task := &models.MiniMaxMediaTask{
		ID:          taskID,
		APIKeyID:    firstAPIKeyID(key),
		Model:       model,
		Provider:    "minimax",
		Status:      "pending",
		RequestBody: truncateString(string(bodyBytes), 50000),
		ClientIP:    c.ClientIP(),
	}
	if err := h.db.CreateMiniMaxMediaTask(task); err != nil {
		h.logAPIRequest(key, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusInternalServerError, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败", "code": 500})
		return
	}

	// 立即返回任务ID，让客户端可以轮询
	c.JSON(http.StatusOK, gin.H{
		"task_id":    taskID,
		"model":      model,
		"status":     "pending",
		"created_at": task.CreatedAt.Format(time.RFC3339),
		"message":    "任务已提交，请通过 GET /api/minimax/token-plan/tasks/" + taskID + " 查询状态",
	})

	// 异步提交到 MiniMax API 并轮询结果
	go h.runAsyncMinimaxMediaTask(taskID, apiKey, baseURL, upstreamURL, bodyBytes, model, firstAPIKeyID(key))
}

// runAsyncMinimaxMediaTask 后台异步执行 MiniMax 媒体任务并轮询结果
func (h *AIGatewayHandler) runAsyncMinimaxMediaTask(taskID, apiKey, baseURL, upstreamURL string, bodyBytes []byte, model, apiKeyID string) {
	task, err := h.db.GetMiniMaxMediaTask(taskID)
	if err != nil {
		return
	}

	// 1. 提交任务到 MiniMax
	task.Status = "running"
	_ = h.db.UpdateMiniMaxMediaTask(task)

	start := time.Now()
	respBody, respContentType, err := h.doRawRequestWithResp(upstreamURL, apiKey, "POST", bodyBytes, nil)

	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateMiniMaxMediaTask(task)
		h.logAPIRequestByID(apiKeyID, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", task.ClientIP, time.Since(start), usageSummary{})
		return
	}

	// 解析响应获取 task_id 和检查是否有同步结果
	var respMap map[string]interface{}
	hasSyncResult := false
	baseErr := ""
	if err := json.Unmarshal(respBody, &respMap); err == nil {
		// 尝试从响应中提取 MiniMax 的任务 ID
		externalTaskID := extractMinimaxTaskID(respMap)
		if externalTaskID != "" {
			task.ExternalTaskID = externalTaskID
			_ = h.db.UpdateMiniMaxMediaTask(task)
		}

		baseErr = minimaxBaseRespError(respMap)
		hasSyncResult = minimaxHasInlineMediaResult(respMap, model, respContentType)
	}

	// 同步返回了媒体内容
	if hasSyncResult {
		task.Status = "succeeded"
		task.ResultJSON = string(respBody)
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateMiniMaxMediaTask(task)
		h.logAPIRequestByID(apiKeyID, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusOK, true, "", string(bodyBytes), string(respBody), task.ClientIP, time.Since(start), usageSummary{})
		return
	}

	// 异步模式：需要轮询 external_task_id
	if task.ExternalTaskID == "" {
		task.Status = "failed"
		task.ResultJSON = string(respBody)
		if baseErr != "" {
			task.ErrorMessage = baseErr
		} else {
			task.ErrorMessage = "无法获取 MiniMax 任务 ID"
		}
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateMiniMaxMediaTask(task)
		return
	}

	// 轮询 MiniMax 任务状态
	deadline := start.Add(minimaxAsyncTimeout(model))
	for {
		if time.Now().After(deadline) {
			task.Status = "failed"
			task.ErrorMessage = "任务超时未完成"
			now := time.Now()
			task.CompletedAt = &now
			_ = h.db.UpdateMiniMaxMediaTask(task)
			return
		}

		time.Sleep(3 * time.Second)

		pollResp, err := h.pollMinimaxTaskStatus(task.ExternalTaskID, apiKey, baseURL, model)
		if err != nil {
			continue
		}

		task.Status = pollResp.Status
		if pollResp.ResultJSON != "" {
			task.ResultJSON = pollResp.ResultJSON
		}
		if pollResp.ErrorMessage != "" {
			task.ErrorMessage = pollResp.ErrorMessage
		}

		if pollResp.Status == "succeeded" || pollResp.Status == "failed" {
			now := time.Now()
			task.CompletedAt = &now
			_ = h.db.UpdateMiniMaxMediaTask(task)
			h.logAPIRequestByID(apiKeyID, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusOK, true, "", string(bodyBytes), task.ResultJSON, task.ClientIP, time.Since(start), usageSummary{})
			return
		}
	}
}

// minimaxPollResponse MiniMax 轮询响应
type minimaxPollResponse struct {
	Status       string
	ResultJSON   string
	ErrorMessage string
}

// pollMinimaxTaskStatus 轮询 MiniMax 任务状态
func (h *AIGatewayHandler) pollMinimaxTaskStatus(externalTaskID, apiKey, baseURL, model string) (*minimaxPollResponse, error) {
	pollURL := resolveMinimaxPollURL(baseURL, externalTaskID, model)

	req, err := http.NewRequest(http.MethodGet, pollURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		msg := extractString(payload, "message")
		if msg == "" {
			msg = fmt.Sprintf("轮询失败，状态码 %d", resp.StatusCode)
		}
		return &minimaxPollResponse{Status: "failed", ErrorMessage: msg}, nil
	}

	// 解析状态
	status := extractMinimaxTaskStatus(payload)
	resultPayload := payload
	errorMessage := firstNonEmpty(minimaxBaseRespError(payload), extractMinimaxTaskError(payload))

	if mapMinimaxAsyncStatus(status) == "succeeded" {
		if fileID := extractString(payload, "file_id"); fileID != "" {
			if filePayload, err := h.retrieveMinimaxFile(apiKey, baseURL, fileID); err == nil {
				resultPayload = mergeMaps(payload, filePayload)
				if fileObj, ok := filePayload["file"].(map[string]interface{}); ok {
					if downloadURL := extractString(fileObj, "download_url"); downloadURL != "" {
						resultPayload["download_url"] = downloadURL
						if isMiniMaxVideoModel(model) {
							resultPayload["video_url"] = downloadURL
						}
					}
				}
			}
		}
	}
	resultJSON, _ := json.Marshal(resultPayload)

	return &minimaxPollResponse{
		Status:       mapMinimaxAsyncStatus(status),
		ResultJSON:   string(resultJSON),
		ErrorMessage: errorMessage,
	}, nil
}

func minimaxAsyncTimeout(model string) time.Duration {
	switch {
	case isMiniMaxVideoModel(model):
		return 15 * time.Minute
	case strings.HasPrefix(model, "music-"):
		return 8 * time.Minute
	default:
		return 5 * time.Minute
	}
}

func minimaxImageAspectRatioFromSize(size string) (string, bool) {
	switch strings.TrimSpace(size) {
	case "1024x1024":
		return "1:1", true
	case "1280x720":
		return "16:9", true
	case "720x1280":
		return "9:16", true
	case "1152x864":
		return "4:3", true
	case "864x1152":
		return "3:4", true
	case "1248x832":
		return "3:2", true
	case "832x1248":
		return "2:3", true
	case "1344x576":
		return "21:9", true
	case "576x1344":
		return "9:21", true
	default:
		return "", false
	}
}

func resolveMinimaxPollURL(baseURL, externalTaskID, model string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if isMiniMaxVideoModel(model) {
		return baseURL + "/v1/query/video_generation?" + url.Values{"task_id": []string{externalTaskID}}.Encode()
	}
	return baseURL + "/v1/tasks/" + externalTaskID
}

func isMiniMaxVideoModel(model string) bool {
	return strings.HasPrefix(model, "MiniMax-Hailuo-") || strings.HasPrefix(model, "T2V-")
}

func (h *AIGatewayHandler) retrieveMinimaxFile(apiKey, baseURL, fileID string) (map[string]interface{}, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/files/retrieve?" + url.Values{"file_id": []string{fileID}}.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf(firstNonEmpty(minimaxBaseRespError(payload), fmt.Sprintf("文件查询失败，状态码 %d", resp.StatusCode)))
	}
	return payload, nil
}

func mergeMaps(primary, secondary map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(primary)+len(secondary))
	for key, value := range primary {
		merged[key] = value
	}
	for key, value := range secondary {
		if _, exists := merged[key]; exists && key == "file" {
			continue
		}
		merged[key] = value
	}
	return merged
}

// extractMinimaxTaskID 从响应中提取 MiniMax 任务 ID
func extractMinimaxTaskID(payload map[string]interface{}) string {
	for _, keyPath := range [][]string{
		{"task_id"},
		{"taskId"},
		{"id"},
		{"data", "task_id"},
		{"data", "taskId"},
		{"data", "id"},
		{"output", "task_id"},
		{"output", "taskId"},
		{"output", "id"},
		{"data", "task", "task_id"},
		{"data", "task", "taskId"},
		{"task", "task_id"},
		{"task", "taskId"},
	} {
		if id := extractString(payload, keyPath...); id != "" {
			return id
		}
	}
	return ""
}

func minimaxHasInlineMediaResult(payload map[string]interface{}, model, respContentType string) bool {
	if len(extractMediaURLs(payload)) > 0 {
		return true
	}
	if strings.HasPrefix(respContentType, "audio/") || strings.HasPrefix(respContentType, "image/") || strings.HasPrefix(respContentType, "video/") {
		return true
	}
	for _, candidate := range []map[string]interface{}{
		payload,
		mapAt(payload, "data"),
		mapAt(payload, "output"),
		mapAt(mapAt(payload, "data"), "task"),
	} {
		if candidate == nil {
			continue
		}
		for _, key := range []string{
			"audio", "audio_url", "audio_urls",
			"video", "video_url", "video_urls",
			"image", "image_url", "image_urls",
			"file_url", "file_urls", "url", "urls",
		} {
			if hasNonEmptyValue(candidate[key]) {
				return true
			}
		}
	}
	// 音乐 / TTS 同步接口即便没有 URL，也可能直接内嵌音频内容。
	if strings.HasPrefix(model, "music-") || strings.HasPrefix(model, "speech-") {
		if hasNonEmptyValue(extractAny(payload, "data", "audio")) || hasNonEmptyValue(extractAny(payload, "audio")) {
			return true
		}
	}
	return false
}

func mapAt(payload map[string]interface{}, key string) map[string]interface{} {
	if payload == nil {
		return nil
	}
	value, ok := payload[key]
	if !ok {
		return nil
	}
	result, _ := value.(map[string]interface{})
	return result
}

func extractAny(payload map[string]interface{}, path ...string) interface{} {
	var current interface{} = payload
	for _, key := range path {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = obj[key]
	}
	return current
}

func hasNonEmptyValue(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(v) != ""
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

// extractMinimaxTaskStatus 从响应中提取任务状态
func extractMinimaxTaskStatus(payload map[string]interface{}) string {
	// MiniMax 可能的状态：pending, processing, succeeded, failed
	if status := extractString(payload, "status"); status != "" {
		return status
	}
	if output, ok := payload["output"].(map[string]interface{}); ok {
		if status := extractString(output, "status"); status != "" {
			return status
		}
	}
	return ""
}

// extractMinimaxTaskError 从响应中提取错误信息
func extractMinimaxTaskError(payload map[string]interface{}) string {
	if msg := extractString(payload, "message"); msg != "" {
		return msg
	}
	if msg := extractString(payload, "error", "message"); msg != "" {
		return msg
	}
	if output, ok := payload["output"].(map[string]interface{}); ok {
		if msg := extractString(output, "error_message"); msg != "" {
			return msg
		}
	}
	return ""
}

// mapMinimaxAsyncStatus 将 MiniMax 状态映射为内部状态
func mapMinimaxAsyncStatus(status string) string {
	switch strings.ToLower(status) {
	case "pending", "queued":
		return "pending"
	case "processing", "running":
		return "running"
	case "succeeded", "success":
		return "succeeded"
	case "failed", "fail":
		return "failed"
	default:
		return "pending"
	}
}

// ListMinimaxTokenPlanTasks 获取当前 API Key 的 MiniMax 媒体任务列表
// GET /api/minimax/token-plan/tasks
func (h *AIGatewayHandler) ListMinimaxTokenPlanTasks(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	limit := 20
	offset := 0
	if l := parseInt(c.Query("limit"), 0); l > 0 && l <= 100 {
		limit = l
	}
	if o := parseInt(c.Query("offset"), 0); o >= 0 {
		offset = o
	}

	apiKeyID := firstAPIKeyID(key)
	tasks, err := h.db.ListMiniMaxMediaTasks(apiKeyID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败", "code": 500})
		return
	}

	total, _ := h.db.CountMiniMaxMediaTasks(apiKeyID)

	// 转换输出格式
	taskList := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		item := gin.H{
			"task_id":    t.ID,
			"model":      t.Model,
			"provider":   t.Provider,
			"status":     t.Status,
			"error":      t.ErrorMessage,
			"created_at": t.CreatedAt.Format(time.RFC3339),
		}
		if t.CompletedAt != nil {
			item["completed_at"] = t.CompletedAt.Format(time.RFC3339)
		}
		if t.ExternalTaskID != "" {
			item["external_task_id"] = t.ExternalTaskID
		}
		// 如果有结果，解析并提取 URL
		if t.ResultJSON != "" && t.Status == "succeeded" {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(t.ResultJSON), &result); err == nil {
				if output, ok := result["output"].(map[string]interface{}); ok {
					urls := extractMediaURLs(output)
					if len(urls) > 0 {
						item["result_urls"] = urls
					}
				}
			}
		}
		taskList = append(taskList, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":  taskList,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetMinimaxTokenPlanTask 获取 MiniMax 媒体任务详情
// GET /api/minimax/token-plan/tasks/:id
func (h *AIGatewayHandler) GetMinimaxTokenPlanTask(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	taskID := c.Param("id")
	task, err := h.db.GetMiniMaxMediaTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}

	// 验证任务属于当前 API Key
	if key != nil && task.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此任务", "code": 403})
		return
	}

	result := gin.H{
		"task_id":    task.ID,
		"model":      task.Model,
		"provider":   task.Provider,
		"status":     task.Status,
		"error":      task.ErrorMessage,
		"request":    task.RequestBody,
		"created_at": task.CreatedAt.Format(time.RFC3339),
	}

	if task.CompletedAt != nil {
		result["completed_at"] = task.CompletedAt.Format(time.RFC3339)
	}
	if task.ExternalTaskID != "" {
		result["external_task_id"] = task.ExternalTaskID
	}
	if task.ResultJSON != "" {
		var resultMap map[string]interface{}
		if err := json.Unmarshal([]byte(task.ResultJSON), &resultMap); err == nil {
			result["result"] = resultMap
			// 提取媒体 URL 列表
			if urls := extractMediaURLs(resultMap); len(urls) > 0 {
				result["result_urls"] = urls
			}
			// 提取内容类型
			if contentType := extractContentType(resultMap); contentType != "" {
				result["content_type"] = contentType
			}
		} else {
			result["result"] = task.ResultJSON
		}
	}

	c.JSON(http.StatusOK, result)
}

// extractMediaURLs 从结果中提取媒体 URL
func extractMediaURLs(output map[string]interface{}) []string {
	urls := make([]string, 0)
	seen := make(map[string]struct{})
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch val := v.(type) {
		case string:
			if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
				if _, exists := seen[val]; !exists {
					seen[val] = struct{}{}
					urls = append(urls, val)
				}
			}
		case map[string]interface{}:
			for _, v := range val {
				walk(v)
			}
		case []interface{}:
			for _, v := range val {
				walk(v)
			}
		}
	}
	walk(output)
	return urls
}

// extractContentType 从结果中提取内容类型
func extractContentType(output map[string]interface{}) string {
	// 优先从顶层 content_type 字段获取
	if ct := extractString(output, "content_type"); ct != "" {
		return ct
	}
	// 尝试从 output 对象中获取
	if outputObj, ok := output["output"].(map[string]interface{}); ok {
		if ct := extractString(outputObj, "content_type"); ct != "" {
			return ct
		}
		// 如果 output 是 URL 字符串，根据扩展名推断
		if urlStr := extractString(outputObj, "url"); urlStr != "" {
			return inferContentType(urlStr)
		}
		if urlStr := extractString(outputObj, "audio_url"); urlStr != "" {
			return inferContentType(urlStr)
		}
		if urlStr := extractString(outputObj, "video_url"); urlStr != "" {
			return inferContentType(urlStr)
		}
		if urlStr := extractString(outputObj, "image_url"); urlStr != "" {
			return inferContentType(urlStr)
		}
	}
	// 根据模型推断
	if model := extractString(output, "model"); model != "" {
		return inferContentTypeByModel(model)
	}
	return ""
}

// inferContentType 根据 URL 扩展名推断内容类型
func inferContentType(url string) string {
	if strings.HasSuffix(url, ".mp3") || strings.HasSuffix(url, ".mpeg") {
		return "audio/mpeg"
	}
	if strings.HasSuffix(url, ".wav") {
		return "audio/wav"
	}
	if strings.HasSuffix(url, ".mp4") {
		return "video/mp4"
	}
	if strings.HasSuffix(url, ".png") {
		return "image/png"
	}
	if strings.HasSuffix(url, ".jpg") || strings.HasSuffix(url, ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(url, ".gif") {
		return "image/gif"
	}
	if strings.HasSuffix(url, ".webp") {
		return "image/webp"
	}
	return "application/octet-stream"
}

// inferContentTypeByModel 根据模型名称推断内容类型
func inferContentTypeByModel(model string) string {
	switch {
	case strings.HasPrefix(model, "speech-"):
		return "audio/mpeg"
	case strings.HasPrefix(model, "MiniMax-Hailuo-") || strings.HasPrefix(model, "T2V-"):
		return "video/mp4"
	case strings.HasPrefix(model, "music-"):
		return "audio/mpeg"
	case model == "image-01" || model == "image-01-live":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}

// GetTokenPlanDocs 返回 Token Plan 端点的 API 文档
// GET /api/minimax/token-plan/docs
func (h *AIGatewayHandler) GetTokenPlanDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "MiniMax Token Plan 接口文档",
		"summary": "通过 AI Gateway 调用 MiniMax Token Plan 媒体生成模型，支持 TTS HD、Hailuo 视频、Music 音乐、Image 生成。",
		"auth": gin.H{
			"api_key": "Authorization: Bearer dtk_ai_xxx",
			"scope":   "media",
		},
		"base_url": "/api/minimax/token-plan",
		"upstream": "https://api.minimaxi.com",
		"models":   TokenPlanAllowedModels,
		"routes": []gin.H{
			{"method": "GET", "path": "/api/minimax/token-plan/docs", "description": "获取本文档"},
			{"method": "POST", "path": "/api/minimax/token-plan/v1/generations", "description": "MiniMax Token Plan 媒体生成接口"},
			{"method": "GET", "path": "/api/minimax/token-plan/tasks", "description": "获取当前 API Key 的任务列表"},
			{"method": "GET", "path": "/api/minimax/token-plan/tasks/:id", "description": "获取任务详情（含 result_urls）"},
			{"method": "GET", "path": "/api/minimax/token-plan/tasks/:id/download", "description": "下载任务产物（代理 MiniMax 媒体文件）"},
		},
		"examples": gin.H{
			"tts_hd_request": gin.H{
				"model": "speech-2.8-hd",
				"text":  "你好，这是语音合成测试",
				"voice_setting": gin.H{
					"voice_id": "male-qn",
					"speed":    1.0,
				},
				"audio_setting": gin.H{
					"audio_format": "mp3",
					"sample_rate":  32000,
				},
			},
			"hailuo_request": gin.H{
				"model":      "MiniMax-Hailuo-2.3-Fast",
				"prompt":     "一只猫在草地上玩耍",
				"duration":   6,
				"resolution": "768P",
			},
			"music_request": gin.H{
				"model":    "music-2.6",
				"prompt":   "轻松的爵士音乐，适合咖啡厅背景",
				"duration": 300,
			},
			"image_request": gin.H{
				"model":  "image-01",
				"prompt": "一个穿着汉服的少女在樱花树下",
				"size":   "1024x1024",
			},
			"curl": gin.H{
				"language": "cURL",
				"code": `curl -X POST https://your-devtools:8080/api/minimax/token-plan/v1/generations \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "text": "你好，这是语音合成测试"
  }'`,
			},
		},
		"model_descriptions": gin.H{
			// TTS HD / Turbo 系列
			"speech-01-hd":     "高清语音合成（speech-01 系列）",
			"speech-01-turbo":  "标准语音合成（speech-01 系列 turbo 版）",
			"speech-02-hd":     "高清语音合成（speech-02 系列）",
			"speech-02-turbo":  "标准语音合成（speech-02 系列 turbo 版）",
			"speech-2.6-hd":    "高清语音合成（speech-2.6 系列）",
			"speech-2.6-turbo": "标准语音合成（speech-2.6 系列 turbo 版）",
			"speech-2.8-hd":    "高清语音合成（speech-2.8 系列，推荐）",
			"speech-2.8-turbo": "标准语音合成（speech-2.8 系列 turbo 版）",
			// Hailuo 视频（官方模型名）
			"MiniMax-Hailuo-2.3-Fast": "Hailuo 视频生成（Fast 6s / 768P）",
			"MiniMax-Hailuo-2.3":      "Hailuo 视频生成（MiniMax-Hailuo-2.3）",
			"MiniMax-Hailuo-02":       "Hailuo 视频生成（MiniMax-Hailuo-02）",
			"T2V-01-Director":         "视频生成（T2V-01-Director）",
			"T2V-01":                  "视频生成（T2V-01）",
			// Music
			"music-2.5":   "音乐生成（2.5）",
			"music-2.6":   "音乐生成（2.6）",
			"music-cover": "翻唱生成（需先拿到 cover_feature_id）",
			// Image
			"image-01":      "图像生成",
			"image-01-live": "图像生成（live 版）",
		},
	})
}

// proxyAnthropic 转发 Anthropic 协议请求到指定上游
func (h *AIGatewayHandler) proxyAnthropic(c *gin.Context, upstreamBase, apiKey, logPath string, allowedModels []string) {
	key, ok := h.authenticateAdminOrAPIKey(c, "chat")
	if !ok {
		return
	}

	// 直接透传请求体到上游
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	if len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空"})
		return
	}

	// 解析 model 字段用于校验和日志
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体 JSON 格式错误"})
		return
	}

	model, _ := bodyMap["model"].(string)
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model 字段"})
		return
	}

	// 校验模型是否在允许列表中
	if !isModelAllowed(model, allowedModels) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("该端点不支持模型 %s，允许的模型: %v", model, allowedModels)})
		return
	}

	if key != nil && !h.ensureModelAllowed(c, key, model) {
		return
	}

	start := time.Now()
	upstreamURL := strings.TrimRight(upstreamBase, "/") + "/v1/messages"

	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置上游 API Key"})
		return
	}

	// 转发到上游 Anthropic 端点
	raw, err := h.doRawRequest(upstreamURL, apiKey, "POST", bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "anthropic", logPath, "chat", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	h.logAPIRequest(key, model, "anthropic", logPath, "chat", http.StatusOK, true, "", string(bodyBytes), string(raw), c.ClientIP(), time.Since(start), usageSummary{})

	// 返回原始响应
	c.Data(http.StatusOK, "application/json", raw)
}

// isModelAllowed 检查模型是否在允许列表中
func isModelAllowed(model string, allowed []string) bool {
	for _, m := range allowed {
		if m == model {
			return true
		}
	}
	return false
}

func (h *AIGatewayHandler) callProxyChat(req ChatCompletionRequest) (gin.H, map[string]interface{}, error) {
	if strings.TrimSpace(h.cfg.AIGateway.Proxy.APIURL) == "" || strings.TrimSpace(h.cfg.AIGateway.Proxy.APIKey) == "" {
		return nil, nil, fmt.Errorf("未配置 ai_gateway.proxy.api_url 或 ai_gateway.proxy.api_key")
	}

	bodyMap := map[string]interface{}{
		"model":    h.proxyUpstreamModel(req.Model),
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		bodyMap["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}

	endpoint := strings.TrimRight(h.cfg.AIGateway.Proxy.APIURL, "/") + "/chat/completions"
	raw, err := h.doJSONRequest(endpoint, h.cfg.AIGateway.Proxy.APIKey, bodyMap)
	if err != nil {
		return nil, raw, err
	}
	content := extractString(raw, "choices", "0", "message", "content")
	if content == "" {
		content = extractMessageContentFromChoices(raw["choices"])
	}
	result := gin.H{
		"id":           fallbackString(extractString(raw, "id"), "chatcmpl-"+utils.GenerateHexKey(4)),
		"object":       "chat.completion",
		"created":      time.Now().Unix(),
		"model":        req.Model,
		"provider":     "proxy",
		"choices":      raw["choices"],
		"usage":        raw["usage"],
		"content":      content,
		"raw_response": raw,
	}
	return result, raw, nil
}

func (h *AIGatewayHandler) doJSONRequest(url, apiKey string, bodyMap map[string]interface{}) (map[string]interface{}, error) {
	body, _ := json.Marshal(bodyMap)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var payload map[string]interface{}
	_ = json.Unmarshal(respBody, &payload)
	if resp.StatusCode >= 400 {
		return payload, fmt.Errorf("上游返回错误(%d): %s", resp.StatusCode, truncateString(string(respBody), 400))
	}
	return payload, nil
}

// doRawRequest 转发原始请求到上游
func (h *AIGatewayHandler) doRawRequest(url, apiKey, method string, body []byte, headers http.Header) ([]byte, error) {
	respBody, _, err := h.doRawRequestWithResp(url, apiKey, method, body, headers)
	return respBody, err
}

// doRawRequestWithResp 转发原始请求到上游，返回响应体、Content-Type 和错误
func (h *AIGatewayHandler) doRawRequestWithResp(url, apiKey, method string, body []byte, headers http.Header) ([]byte, string, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	// 透传必要的 headers（过滤掉代理相关和可能导致 HTTP/2 问题的 headers）
	skipHeaders := map[string]bool{
		"Content-Type":      true,
		"Authorization":     true,
		"Accept":            true,
		"Connection":        true,
		"Proxy-Connection":  true,
		"Upgrade":           true,
		"Keep-Alive":        true,
		"TE":                true,
		"Trailer":           true,
		"Transfer-Encoding": true,
		"Host":              true,
		"Content-Length":    true,
	}
	for key, values := range headers {
		if skipHeaders[http.CanonicalHeaderKey(key)] {
			continue
		}
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}
	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	contentType := resp.Header.Get("Content-Type")
	if resp.StatusCode >= 400 {
		return respBody, contentType, fmt.Errorf("上游返回错误(%d): %s", resp.StatusCode, truncateString(string(respBody), 400))
	}
	return respBody, contentType, nil
}

func (h *AIGatewayHandler) authenticateAPIKey(c *gin.Context, scope string) (*models.AIAPIKey, bool) {
	token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("X-API-Key"))
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少 API Key", "code": 401})
		return nil, false
	}
	if len(token) < 18 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API Key 格式错误", "code": 401})
		return nil, false
	}

	keys, err := h.db.GetAIAPIKeysByPrefix(token[:18])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "校验 API Key 失败", "code": 500})
		return nil, false
	}
	var matched *models.AIAPIKey
	for _, item := range keys {
		if utils.VerifyPassword(token, item.KeyHash) {
			matched = item
			break
		}
	}
	if matched == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API Key 无效", "code": 401})
		return nil, false
	}
	if matched.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "API Key 已停用", "code": 403})
		return nil, false
	}
	if matched.ExpiresAt != nil && time.Now().After(*matched.ExpiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "API Key 已过期", "code": 403})
		return nil, false
	}
	if !jsonStringContains(matched.AllowedScopes, scope) && !jsonStringContains(matched.AllowedScopes, "*") {
		c.JSON(http.StatusForbidden, gin.H{"error": "当前 API Key 没有该接口权限", "code": 403})
		return nil, false
	}
	if matched.RateLimitPerHour > 0 {
		count, _ := h.db.CountAIAPIRequestsSince(matched.ID, time.Now().Add(-time.Hour))
		if count >= matched.RateLimitPerHour {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "API Key 已达到每小时调用上限", "code": 429})
			return nil, false
		}
	}
	return matched, true
}

// authenticateAdminOrAPIKey 双重认证：先尝试超级管理员认证，失败后尝试 API Key 认证
// 如果是超级管理员认证成功，返回 (nil, true)，表示不限制 API Key
// 如果是 API Key 认证成功，返回 (apiKey, true)
// 如果都失败，返回 (nil, false)
func (h *AIGatewayHandler) authenticateAdminOrAPIKey(c *gin.Context, scope string) (*models.AIAPIKey, bool) {
	// 1. 检查是否有超级管理员密码
	adminPassword := c.GetHeader("X-Super-Admin-Password")
	if adminPassword == "" {
		adminPassword = c.Query("super_admin_password")
	}

	// 2. 如果有超级管理员密码，验证它
	if adminPassword != "" {
		if strings.TrimSpace(h.cfg.AIGateway.SuperAdminPassword) != "" &&
			adminPassword == h.cfg.AIGateway.SuperAdminPassword {
			return nil, true // 超级管理员认证成功
		}
	}

	// 3. 如果没有超级管理员密码或验证失败，尝试 API Key 认证
	return h.authenticateAPIKey(c, scope)
}

func firstAPIKeyID(key *models.AIAPIKey) string {
	if key == nil {
		return ""
	}
	return key.ID
}

func (h *AIGatewayHandler) ensureModelAllowed(c *gin.Context, key *models.AIAPIKey, model string) bool {
	if jsonStringContains(key.AllowedModels, "*") || jsonStringContains(key.AllowedModels, model) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "当前 API Key 无权访问该模型", "code": 403})
	return false
}

func (h *AIGatewayHandler) logAPIRequest(key *models.AIAPIKey, model, provider, endpoint, requestType string, statusCode int, success bool, errMessage, requestBody, responseBody, clientIP string, latency time.Duration, usage usageSummary) {
	if key == nil {
		return
	}
	_ = h.db.CreateAIAPIRequestLog(&models.AIAPIRequestLog{
		APIKeyID:      key.ID,
		Model:         model,
		Provider:      provider,
		Endpoint:      endpoint,
		RequestType:   requestType,
		StatusCode:    statusCode,
		Success:       success,
		ErrorMessage:  errMessage,
		RequestBody:   truncateString(requestBody, 10000),
		ResponseBody:  truncateString(responseBody, 10000),
		ClientIP:      clientIP,
		LatencyMS:     latency.Milliseconds(),
		InputTokens:   usage.InputTokens,
		OutputTokens:  usage.OutputTokens,
		TotalTokens:   usage.TotalTokens,
		EstimatedCost: usage.Cost,
		Currency:      usage.Currency,
	})
	_ = h.db.TouchAIAPIKeyUsage(key.ID, time.Now(), usage.InputTokens, usage.OutputTokens, usage.TotalTokens, usage.Cost, usage.Currency)
}

func (h *AIGatewayHandler) logAPIRequestByID(apiKeyID, model, provider, endpoint, requestType string, statusCode int, success bool, errMessage, requestBody, responseBody, clientIP string, latency time.Duration, usage usageSummary) {
	_ = h.db.CreateAIAPIRequestLog(&models.AIAPIRequestLog{
		APIKeyID:      apiKeyID,
		Model:         model,
		Provider:      provider,
		Endpoint:      endpoint,
		RequestType:   requestType,
		StatusCode:    statusCode,
		Success:       success,
		ErrorMessage:  errMessage,
		RequestBody:   truncateString(requestBody, 10000),
		ResponseBody:  truncateString(responseBody, 10000),
		ClientIP:      clientIP,
		LatencyMS:     latency.Milliseconds(),
		InputTokens:   usage.InputTokens,
		OutputTokens:  usage.OutputTokens,
		TotalTokens:   usage.TotalTokens,
		EstimatedCost: usage.Cost,
		Currency:      usage.Currency,
	})
	_ = h.db.TouchAIAPIKeyUsage(apiKeyID, time.Now(), usage.InputTokens, usage.OutputTokens, usage.TotalTokens, usage.Cost, usage.Currency)
}

// requireImageUnderstandingAdmin 校验图像理解模块独立管理员密码
func (h *AIGatewayHandler) requireImageUnderstandingAdmin(c *gin.Context) bool {
	password := strings.TrimSpace(c.GetHeader("X-Image-Admin-Password"))
	if password == "" {
		password = strings.TrimSpace(c.Query("admin_password"))
	}
	configured := strings.TrimSpace(h.cfg.ImageUnderstanding.AdminPassword)
	if configured == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "未配置 image_understanding.admin_password", "code": 403})
		return false
	}
	if password != configured {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误", "code": 401})
		return false
	}
	return true
}

func (h *AIGatewayHandler) requireSuperAdmin(c *gin.Context, bodyPassword string) bool {
	password := bodyPassword
	if password == "" {
		password = c.GetHeader("X-Super-Admin-Password")
	}
	if password == "" {
		password = c.Query("super_admin_password")
	}
	if strings.TrimSpace(h.cfg.AIGateway.SuperAdminPassword) == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "未配置 ai_gateway.super_admin_password 或 AI_GATEWAY_SUPER_ADMIN_PASSWORD", "code": 403})
		return false
	}
	if password != h.cfg.AIGateway.SuperAdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "超级管理员密码错误", "code": 401})
		return false
	}
	return true
}

var dashscopeModels = map[string]bool{
	"qwen3.5-plus":         true,
	"qwen3-max-2026-01-23": true,
	"qwen3-coder-next":     true,
	"qwen3-coder-plus":     true,
	"glm-5":                true,
	"glm-4.7":              true,
	"kimi-k2.5":            true,
}

func (h *AIGatewayHandler) resolveChatProvider(model string) string {
	// DashScope 优先（若配置了 API Key）
	if h.cfg.DashScope.APIKey != "" && dashscopeModels[model] {
		return "dashscope"
	}
	switch model {
	case fallbackString(h.cfg.DeepSeek.Model, "deepseek-chat"), "deepseek-chat", "deepseek-reasoner", "deepseek-v4-pro", "deepseek-v4-flash", "deepseek-coder":
		return "deepseek"
	case fallbackString(h.cfg.MiniMax.Model, "abab6.5s-chat"), "abab6.5s-chat", "MiniMax-M2.5":
		return "minimax"
	default:
		if h.hasProxyConfig() && h.isProxyModel(model) {
			return "proxy"
		}
		return ""
	}
}

func (h *AIGatewayHandler) proxyUpstreamModel(requestModel string) string {
	for _, model := range h.cfg.AIGateway.Proxy.Models {
		if strings.TrimSpace(model.Model) == requestModel && strings.TrimSpace(model.UpstreamModel) != "" {
			return model.UpstreamModel
		}
	}
	if strings.TrimSpace(h.cfg.AIGateway.Proxy.UpstreamModel) != "" {
		return h.cfg.AIGateway.Proxy.UpstreamModel
	}
	return requestModel
}

func (h *AIGatewayHandler) hasProxyConfig() bool {
	return strings.TrimSpace(h.cfg.AIGateway.Proxy.APIURL) != "" && strings.TrimSpace(h.cfg.AIGateway.Proxy.APIKey) != ""
}

func (h *AIGatewayHandler) isProxyModel(model string) bool {
	for _, item := range h.getProxyModels() {
		if strings.TrimSpace(item.Model) == model {
			return true
		}
	}
	return false
}

func (h *AIGatewayHandler) getProxyModels() []config.AIGatewayProxyModelConfig {
	if len(h.cfg.AIGateway.Proxy.Models) > 0 {
		models := make([]config.AIGatewayProxyModelConfig, 0, len(h.cfg.AIGateway.Proxy.Models))
		for _, item := range h.cfg.AIGateway.Proxy.Models {
			if strings.TrimSpace(item.Model) == "" {
				continue
			}
			models = append(models, item)
		}
		if len(models) > 0 {
			return models
		}
	}
	return []config.AIGatewayProxyModelConfig{
		{
			Model:         fallbackString(h.cfg.AIGateway.Proxy.Model, "proxy-chat"),
			UpstreamModel: h.cfg.AIGateway.Proxy.UpstreamModel,
			Description:   "OpenAI 兼容代理文本模型",
		},
	}
}

func extractMiniMaxText(raw map[string]interface{}) string {
	if raw == nil {
		return ""
	}
	contentArr, ok := raw["content"].([]interface{})
	if !ok {
		return ""
	}
	for _, item := range contentArr {
		if m, ok := item.(map[string]interface{}); ok {
			if m["type"] == "text" {
				if text, ok := m["text"].(string); ok {
					return text
				}
			}
		}
	}
	return ""
}

func extractMessageContentFromChoices(choices interface{}) string {
	items, ok := choices.([]interface{})
	if !ok || len(items) == 0 {
		return ""
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		return ""
	}
	message, ok := first["message"].(map[string]interface{})
	if !ok {
		return ""
	}
	content, _ := message["content"].(string)
	return content
}

func extractReasoningContentFromChoices(choices interface{}) string {
	items, ok := choices.([]interface{})
	if !ok || len(items) == 0 {
		return ""
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		return ""
	}
	message, ok := first["message"].(map[string]interface{})
	if !ok {
		return ""
	}
	content, _ := message["reasoning_content"].(string)
	return content
}

func jsonStringContains(raw, target string) bool {
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return false
	}
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func deepseekChatCompletionURL(messages []map[string]interface{}) string {
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		if strings.EqualFold(fmt.Sprint(last["role"]), "assistant") {
			if prefix, ok := last["prefix"].(bool); ok && prefix {
				return "https://api.deepseek.com/beta/chat/completions"
			}
		}
	}
	return "https://api.deepseek.com/chat/completions"
}

func boundedInt(raw string, fallback, min, max int) int {
	value := parseInt(raw, fallback)
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func valueOrDefaultInt(value *int, fallback int) int {
	if value == nil || *value <= 0 {
		return fallback
	}
	return *value
}

func (h *AIGatewayHandler) buildChatUsage(req ChatCompletionRequest, raw map[string]interface{}, provider string) usageSummary {
	inputTokens, outputTokens, totalTokens := extractUsageFromRaw(raw)
	if inputTokens == 0 {
		inputTokens = estimateMessagesTokens(req.Messages)
	}
	if outputTokens == 0 {
		outputTokens = estimateTextTokens(extractMessageContentFromChoices(raw["choices"]) + extractMiniMaxText(raw))
	}
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}
	cost, currency := h.calculateCost(req.Model, provider, inputTokens, outputTokens)
	return usageSummary{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		Cost:         cost,
		Currency:     currency,
	}
}

func (h *AIGatewayHandler) buildMediaUsage(model string) usageSummary {
	cost, currency := h.calculateCost(model, "bailian", 0, 0)
	return usageSummary{Cost: cost, Currency: currency}
}

func extractUsageFromRaw(raw map[string]interface{}) (int, int, int) {
	if raw == nil {
		return 0, 0, 0
	}
	if usage, ok := raw["usage"].(map[string]interface{}); ok {
		input := interfaceToInt(usage["prompt_tokens"])
		if input == 0 {
			input = interfaceToInt(usage["input_tokens"])
		}
		output := interfaceToInt(usage["completion_tokens"])
		if output == 0 {
			output = interfaceToInt(usage["output_tokens"])
		}
		total := interfaceToInt(usage["total_tokens"])
		return input, output, total
	}
	return 0, 0, 0
}

func estimateMessagesTokens(messages []map[string]interface{}) int {
	total := 0
	for _, msg := range messages {
		for _, key := range []string{"content", "role"} {
			if value, ok := msg[key]; ok {
				total += estimateTextTokens(interfaceToString(value))
			}
		}
	}
	return total
}

func estimateTextTokens(text string) int {
	runes := len([]rune(strings.TrimSpace(text)))
	if runes == 0 {
		return 0
	}
	estimated := runes / 4
	if estimated < 1 {
		return 1
	}
	return estimated
}

func (h *AIGatewayHandler) calculateCost(model, provider string, inputTokens, outputTokens int) (float64, string) {
	currency := "CNY"
	for _, pricing := range h.cfg.AIGateway.Pricing {
		if pricing.Model == model && (pricing.Provider == "" || pricing.Provider == provider) {
			if pricing.Currency != "" {
				currency = pricing.Currency
			}
			cost := (float64(inputTokens)/1000.0)*pricing.InputPer1KTokens + (float64(outputTokens)/1000.0)*pricing.OutputPer1KTokens + pricing.RequestCost
			return cost, currency
		}
	}
	return 0, currency
}

func interfaceToInt(value interface{}) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func interfaceToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

// fetchImageAsBase64 下载远程图片，返回 base64 编码内容和 mime 类型（最大 10MB）
func fetchImageAsBase64(client *http.Client, imageURL string) (string, string, error) {
	const maxImageSize = 10 * 1024 * 1024 // 10MB
	resp, err := client.Get(imageURL)
	if err != nil {
		return "", "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSize+1))
	if err != nil {
		return "", "", fmt.Errorf("读取失败: %w", err)
	}
	if len(data) > maxImageSize {
		return "", "", fmt.Errorf("图片超过 10MB 限制")
	}
	// 从 Content-Type 获取 mime，默认 image/jpeg
	mimeType := resp.Header.Get("Content-Type")
	if idx := strings.Index(mimeType, ";"); idx >= 0 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	if mimeType == "" || !strings.HasPrefix(mimeType, "image/") {
		mimeType = "image/jpeg"
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, mimeType, nil
}

// DownloadMinimaxTokenPlanTask 下载 MiniMax 媒体任务产物
// GET /api/minimax/token-plan/tasks/:id/download
func (h *AIGatewayHandler) DownloadMinimaxTokenPlanTask(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	taskID := c.Param("id")
	task, err := h.db.GetMiniMaxMediaTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}

	// 验证任务属于当前 API Key
	if key != nil && task.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此任务", "code": 403})
		return
	}

	// 检查任务状态
	if task.Status != "succeeded" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("任务未完成，当前状态: %s", task.Status), "code": 400})
		return
	}

	if task.ResultJSON == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务结果为空", "code": 400})
		return
	}

	// 解析 ResultJSON 获取媒体 URL
	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(task.ResultJSON), &resultMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解析任务结果失败", "code": 500})
		return
	}

	// 提取媒体 URL
	urls := extractMediaURLs(resultMap)
	if len(urls) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到媒体文件 URL", "code": 400})
		return
	}

	mediaURL := urls[0] // 取第一个 URL

	// 提取 content type
	contentType := extractContentType(resultMap)
	if contentType == "" {
		contentType = inferContentType(mediaURL)
	}

	// 从 MiniMax URL 下载媒体内容
	req, err := http.NewRequest(http.MethodGet, mediaURL, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "创建请求失败", "code": 502})
		return
	}

	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("下载媒体失败: %s", err.Error()), "code": 502})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("媒体文件返回 HTTP %d", resp.StatusCode), "code": 502})
		return
	}

	// 从响应头获取真实 Content-Type
	if realCT := resp.Header.Get("Content-Type"); realCT != "" {
		contentType = realCT
	}

	// 设置文件名后缀
	filename := ""
	switch {
	case strings.HasPrefix(contentType, "audio/"):
		filename = "audio.mp3"
	case strings.HasPrefix(contentType, "video/"):
		filename = "video.mp4"
	case strings.HasPrefix(contentType, "image/"):
		ext := strings.TrimPrefix(contentType, "image/")
		filename = "image." + ext
	}
	if filename == "" {
		filename = "download"
	}

	// 设置下载 header
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", contentType)
	if resp.ContentLength > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	}

	// 透传媒体内容
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// extractImageMeta 从 data URL 中提取 mime 类型和估算大小（KB）
// data URL 格式: data:<mime>;base64,<data>
func extractImageMeta(dataURL string) (mime string, sizeKB int) {
	if !strings.HasPrefix(dataURL, "data:") {
		return "unknown", len(dataURL) / 1024
	}
	// 取 "data:" 和 ";" 之间的部分
	rest := dataURL[5:]
	semi := strings.Index(rest, ";")
	if semi < 0 {
		return "unknown", len(dataURL) / 1024
	}
	mime = rest[:semi]
	// base64 数据长度估算实际字节数
	comma := strings.Index(dataURL, ",")
	if comma >= 0 {
		b64Len := len(dataURL) - comma - 1
		sizeKB = b64Len * 3 / 4 / 1024
	}
	return mime, sizeKB
}

// ==================== Voice Cloning Handlers ====================

// UploadVoiceClone 上传音频复刻音色
// POST /api/minimax/voice-cloning/upload
func (h *AIGatewayHandler) UploadVoiceClone(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	// 获取 API Key 的 ID（超级管理员为空）
	apiKeyID := ""
	if key != nil {
		apiKeyID = key.ID
	}

	// 解析 multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB limit
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析表单失败", "code": 400})
		return
	}

	// 获取音色名称
	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "音色名称不能为空", "code": 400})
		return
	}

	// 获取音频文件
	file, header, err := c.Request.FormFile("audio_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传音频文件", "code": 400})
		return
	}
	defer file.Close()

	// 检查文件大小（限制 10MB）
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "音频文件大小不能超过 10MB", "code": 400})
		return
	}

	// 检查文件类型
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "audio/") && !strings.Contains(contentType, "octet-stream") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传音频文件", "code": 400})
		return
	}

	// 读取文件内容
	audioData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取音频文件失败", "code": 500})
		return
	}

	// 获取 API Key
	apiKey := h.cfg.MiniMaxVoiceCloning.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey
	}
	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax API Key", "code": 502})
		return
	}

	baseURL := h.cfg.MiniMaxVoiceCloning.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	// 调用 MiniMax Voice Cloning API
	// POST /v1/voice_cloning/upload_clone_audio
	upstreamURL := strings.TrimRight(baseURL, "/") + "/v1/voice_cloning/upload_clone_audio"

	// 构建 multipart form 请求
	body := &bytes.Buffer{}
	writer := &multipart.Writer{}
	writer = multipart.NewWriter(body)

	// 添加 audio_file 字段
	part, err := writer.CreateFormFile("audio_file", header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建表单文件失败", "code": 500})
		return
	}
	if _, err := part.Write(audioData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入音频数据失败", "code": 500})
		return
	}

	if err := writer.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "关闭表单写入器失败", "code": 500})
		return
	}

	req, err := http.NewRequest("POST", upstreamURL, bytes.NewReader(body.Bytes()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建请求失败", "code": 500})
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	start := time.Now()
	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/upload", "media", http.StatusBadGateway, false, err.Error(), fmt.Sprintf("name=%s, size=%d", name, len(audioData)), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("上传音色失败: %s", err.Error()), "code": 502})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/upload", "media", resp.StatusCode, resp.StatusCode < 400, string(respBody), fmt.Sprintf("name=%s, size=%d", name, len(audioData)), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("解析响应失败: %s", string(respBody)), "code": 502})
		return
	}

	// 检查 API 错误
	if baseResp, ok := result["base_resp"].(map[string]interface{}); ok {
		if code, ok := baseResp["status_code"].(float64); ok && int(code) != 0 {
			msg, _ := baseResp["status_msg"].(string)
			h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/upload", "media", resp.StatusCode, false, msg, fmt.Sprintf("name=%s, size=%d", name, len(audioData)), string(respBody), c.ClientIP(), time.Since(start), usageSummary{})
			c.JSON(http.StatusBadGateway, gin.H{"error": msg, "code": 502})
			return
		}
	}

	// 提取 voice_id
	voiceID := ""
	if data, ok := result["data"].(map[string]interface{}); ok {
		if vid, ok := data["voice_id"].(string); ok {
			voiceID = vid
		}
	}
	if voiceID == "" {
		h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/upload", "media", resp.StatusCode, false, "未获取到 voice_id", fmt.Sprintf("name=%s, size=%d", name, len(audioData)), string(respBody), c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": "未获取到 voice_id", "code": 502})
		return
	}

	// 保存到数据库
	clone := &models.VoiceClone{
		APIKeyID: apiKeyID,
		VoiceID:  voiceID,
		Name:     name,
		Status:   "active",
	}
	if err := h.db.CreateVoiceClone(clone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存音色记录失败", "code": 500})
		return
	}

	h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/upload", "media", http.StatusOK, true, "", fmt.Sprintf("name=%s, size=%d, voice_id=%s", name, len(audioData), voiceID), string(respBody), c.ClientIP(), time.Since(start), usageSummary{})

	c.JSON(http.StatusOK, gin.H{
		"voice_id": voiceID,
		"name":     name,
		"status":   "active",
		"message":  "音色创建成功",
	})
}

// ListVoiceClones 获取音色列表
// GET /api/minimax/voice-cloning/voices
func (h *AIGatewayHandler) ListVoiceClones(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	apiKeyID := ""
	if key != nil {
		apiKeyID = key.ID
	}
	limit := boundedInt(c.Query("limit"), 20, 1, 100)
	offset := boundedInt(c.Query("offset"), 0, 0, 100000)

	clones, err := h.db.ListVoiceClones(apiKeyID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取音色列表失败", "code": 500})
		return
	}

	// 调用 MiniMax API 获取最新音色状态
	apiKey := h.cfg.MiniMaxVoiceCloning.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey
	}
	baseURL := h.cfg.MiniMaxVoiceCloning.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	// 尝试调用 MiniMax 获取音色列表
	upstreamURL := strings.TrimRight(baseURL, "/") + "/v1/voice_cloning/voice_list"
	req, err := http.NewRequest("GET", upstreamURL, nil)
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		resp, err := h.noProxyClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var upstreamResult map[string]interface{}
				if body, err := io.ReadAll(resp.Body); err == nil {
					json.Unmarshal(body, &upstreamResult)
					// 合并上游音色状态
					if data, ok := upstreamResult["data"].(map[string]interface{}); ok {
						if voices, ok := data["voice_list"].([]interface{}); ok {
							// 创建 voice_id -> status 映射
							statusMap := make(map[string]string)
							for _, v := range voices {
								if voice, ok := v.(map[string]interface{}); ok {
									if vid, ok := voice["voice_id"].(string); ok {
										statusMap[vid] = "active"
									}
								}
							}
							// 更新本地音色状态
							for _, clone := range clones {
								if s, exists := statusMap[clone.VoiceID]; exists {
									clone.Status = s
								}
							}
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"voices": clones,
		"total":  len(clones),
	})
}

// DeleteVoiceClone 删除音色
// DELETE /api/minimax/voice-cloning/voices/:id
func (h *AIGatewayHandler) DeleteVoiceClone(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	apiKeyID := ""
	if key != nil {
		apiKeyID = key.ID
	}

	// 解析 ID
	idStr := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的音色 ID", "code": 400})
		return
	}

	// 获取音色记录
	clone, err := h.db.GetVoiceClone(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "音色不存在", "code": 404})
		return
	}

	// 检查权限：超级管理员可以删除任何音色，普通用户只能删除自己创建的音色
	if key != nil && clone.APIKeyID != apiKeyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除该音色", "code": 403})
		return
	}

	// 调用 MiniMax API 删除音色
	apiKey := h.cfg.MiniMaxVoiceCloning.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey
	}
	baseURL := h.cfg.MiniMaxVoiceCloning.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	upstreamURL := strings.TrimRight(baseURL, "/") + "/v1/voice_cloning/delete_voice"
	delReq, _ := http.NewRequest("DELETE", upstreamURL, nil)
	delReq.Header.Set("Authorization", "Bearer "+apiKey)
	delReq.Header.Set("Content-Type", "application/json")

	body, _ := json.Marshal(map[string]string{"voice_id": clone.VoiceID})
	delReq.Body = io.NopCloser(bytes.NewReader(body))

	start := time.Now()
	resp, err := h.noProxyClient.Do(delReq)
	if err != nil {
		// 即使上游调用失败，也删除本地记录
		h.db.DeleteVoiceClone(id, apiKeyID)
		c.JSON(http.StatusOK, gin.H{"message": "音色已删除（本地上游同步失败）"})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 删除本地记录
	h.db.DeleteVoiceClone(id, apiKeyID)

	h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/voices/"+idStr, "media", http.StatusOK, true, "", fmt.Sprintf("voice_id=%s", clone.VoiceID), string(respBody), c.ClientIP(), time.Since(start), usageSummary{})

	c.JSON(http.StatusOK, gin.H{"message": "音色已删除"})
}

// TTSWithVoiceClone 使用自定义音色进行 TTS
// POST /api/minimax/voice-cloning/tts
func (h *AIGatewayHandler) TTSWithVoiceClone(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	// 直接透传请求体到上游
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	if len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空"})
		return
	}

	// 解析 model 字段用于校验和日志
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体 JSON 格式错误"})
		return
	}

	model, _ := bodyMap["model"].(string)
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model 字段"})
		return
	}

	// 校验模型是否在允许列表中
	if !isModelAllowed(model, TTSAllowedModels) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("该端点不支持模型 %s，允许的模型: %v", model, TTSAllowedModels)})
		return
	}

	// 超级管理员跳过 API Key 模型权限检查
	if key != nil {
		if !h.ensureModelAllowed(c, key, model) {
			return
		}
	}

	// 获取 voice_id
	voiceID, _ := bodyMap["voice_id"].(string)
	if voiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 voice_id 字段"})
		return
	}

	// 验证 voice_id 是否属于当前用户（超级管理员可以使用任何音色）
	clone, err := h.db.GetVoiceCloneByVoiceID(voiceID)
	if err != nil || (key != nil && clone.APIKeyID != key.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权使用该音色", "code": 403})
		return
	}

	apiKey := h.cfg.MiniMaxVoiceCloning.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey
	}
	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax API Key"})
		return
	}

	baseURL := h.cfg.MiniMaxVoiceCloning.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	// 转换请求格式：MiniMax TTS API 要求 voice_setting.voice_id 格式
	upstreamReq := make(map[string]interface{})
	upstreamReq["model"] = model
	if text, ok := bodyMap["text"].(string); ok {
		upstreamReq["text"] = text
	}
	// 使用自定义 voice_id
	if upstreamReq["voice_setting"] == nil {
		upstreamReq["voice_setting"] = map[string]interface{}{}
	}
	if vs, ok := upstreamReq["voice_setting"].(map[string]interface{}); ok {
		vs["voice_id"] = voiceID
	}
	if speed, ok := bodyMap["speed"]; ok {
		if vs, ok := upstreamReq["voice_setting"].(map[string]interface{}); ok {
			vs["speed"] = speed
		}
	}
	// 透传 audio_format
	if af, ok := bodyMap["audio_format"].(string); ok && af != "" {
		upstreamReq["audio_setting"] = map[string]interface{}{"audio_format": af}
	}
	// 其他字段直接透传
	for k, v := range bodyMap {
		if k != "voice_id" && k != "speed" && k != "audio_format" {
			upstreamReq[k] = v
		}
	}

	upstreamBytes, _ := json.Marshal(upstreamReq)

	start := time.Now()
	upstreamURL := strings.TrimRight(baseURL, "/") + "/v1/t2a_v2"

	respBody, _, err := h.doRawRequestWithResp(upstreamURL, apiKey, "POST", upstreamBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// 计算使用量（按字符数计费）
	textLen := len(interfaceToString(bodyMap["text"]))
	usage := usageSummary{Cost: float64(textLen) * 0.001, Currency: "CNY"}

	// 解析响应
	var respData map[string]interface{}
	if err := json.Unmarshal(respBody, &respData); err != nil {
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusBadGateway, false, "上游响应解析失败: "+err.Error(), string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
		c.JSON(http.StatusBadGateway, gin.H{"error": "上游响应解析失败"})
		return
	}

	// 检查业务错误
	if baseResp, ok := respData["base_resp"].(map[string]interface{}); ok {
		if code, ok := baseResp["status_code"].(float64); ok && int(code) != 0 {
			msg, _ := baseResp["status_msg"].(string)
			h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusBadGateway, false, msg, string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
			c.JSON(http.StatusBadGateway, gin.H{"error": msg})
			return
		}
	}

	// 提取 base64 音频数据
	var audioData string
	if data, ok := respData["data"].(map[string]interface{}); ok {
		if audio, ok := data["audio"].(string); ok {
			audioData = audio
		}
	}

	if audioData == "" {
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusBadGateway, false, "未获取到音频数据", string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
		c.JSON(http.StatusBadGateway, gin.H{"error": "未获取到音频数据"})
		return
	}

	// 解码 base64 音频
	audioBytes, err := base64.StdEncoding.DecodeString(audioData)
	if err != nil {
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusBadGateway, false, "音频base64解码失败: "+err.Error(), string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
		c.JSON(http.StatusBadGateway, gin.H{"error": "音频数据解码失败"})
		return
	}

	h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusOK, true, "", string(bodyBytes), "[base64 audio]", c.ClientIP(), time.Since(start), usage)

	// 返回音频二进制
	c.DataFromReader(http.StatusOK, int64(len(audioBytes)), "audio/mpeg", bytes.NewReader(audioBytes), nil)
}

// GetVoiceCloningDocs 返回 Voice Cloning API 文档
// GET /api/minimax/voice-cloning/docs
func (h *AIGatewayHandler) GetVoiceCloningDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "MiniMax Voice Cloning 接口文档",
		"summary": "通过 AI Gateway 调用 MiniMax Voice Cloning 音色克隆与 TTS 合成功能。",
		"auth": gin.H{
			"api_key": "Authorization: Bearer dtk_ai_xxx",
			"scope":   "media",
		},
		"base_url": "/api/minimax/voice-cloning",
		"upstream": "https://api.minimaxi.com/v1/voice_cloning",
		"routes": []gin.H{
			{"method": "GET", "path": "/api/minimax/voice-cloning/docs", "description": "获取本文档"},
			{"method": "POST", "path": "/api/minimax/voice-cloning/upload", "description": "上传音频复刻音色"},
			{"method": "GET", "path": "/api/minimax/voice-cloning/voices", "description": "获取音色列表"},
			{"method": "DELETE", "path": "/api/minimax/voice-cloning/voices/:id", "description": "删除音色"},
			{"method": "POST", "path": "/api/minimax/voice-cloning/tts", "description": "使用自定义音色 TTS"},
		},
		"examples": gin.H{
			"upload_request": gin.H{
				"description": "上传音频复刻音色 (multipart/form-data)",
				"fields": gin.H{
					"name":       "音色名称，如'我的音色'",
					"audio_file": "音频文件（支持 wav/mp3/m4a，最大 10MB）",
				},
			},
			"tts_request": gin.H{
				"model":        "speech-2.8-hd",
				"text":         "你好，这是使用自定义音色的语音合成",
				"voice_id":     "clone_xxx",
				"speed":        1.0,
				"audio_format": "mp3",
			},
			"curl_upload": gin.H{
				"language": "cURL",
				"code": `curl -X POST https://your-devtools:8080/api/minimax/voice-cloning/upload \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -F "name=我的音色" \
  -F "audio_file=@voice.wav"`,
			},
			"curl_tts": gin.H{
				"language": "cURL",
				"code": `curl -X POST https://your-devtools:8080/api/minimax/voice-cloning/tts \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "text": "你好，这是使用自定义音色的语音合成",
    "voice_id": "clone_xxx",
    "speed": 1.0
  }' \
  --output output.mp3`,
			},
		},
		"voice_ids_note": "自定义音色通过 upload 接口创建，voice_id 以 clone_ 开头",
	})
}
