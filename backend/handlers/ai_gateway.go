package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	db      *models.DB
	cfg     *config.Config
	bailian *BailianHandler
	client  *http.Client
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
	Model       string                   `json:"model" binding:"required"`
	Messages    []map[string]interface{} `json:"messages" binding:"required"`
	Temperature *float64                 `json:"temperature"`
	MaxTokens   *int                     `json:"max_tokens"`
	TopP        *float64                 `json:"top_p"`
	Stream      bool                     `json:"stream"`
	Metadata    map[string]interface{}   `json:"metadata"`
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

func NewAIGatewayHandler(db *models.DB, cfg *config.Config, bailian *BailianHandler) *AIGatewayHandler {
	return &AIGatewayHandler{
		db:      db,
		cfg:     cfg,
		bailian: bailian,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (h *AIGatewayHandler) GetDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "AI Gateway API 文档",
		"summary": "统一对外开放 DeepSeek、MiniMax、Bailian 模型能力，使用超级管理员签发的 API Key 访问。",
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
				"model": "deepseek-chat",
				"messages": []gin.H{
					{"role": "system", "content": "你是一个专业助手"},
					{"role": "user", "content": "请写一个 Go HTTP 服务的示例"},
				},
				"temperature": 0.7,
				"max_tokens":  1024,
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
				"models":      []string{"MiniMax-M2.5", "MiniMax-M2.7"},
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
					"language": "Claude Code",
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
					"language": "Claude Code",
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
		catalog = append(catalog, gin.H{
			"model":       fallbackString(h.cfg.DeepSeek.Model, "deepseek-chat"),
			"provider":    "deepseek",
			"type":        "chat",
			"endpoint":    "/api/ai-gateway/v1/chat/completions",
			"description": "DeepSeek 文本模型",
		})
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

	raw, err := h.doJSONRequest("https://api.deepseek.com/chat/completions", h.cfg.DeepSeek.APIKey, bodyMap)
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
		"provider":     "deepseek",
		"choices":      raw["choices"],
		"usage":        raw["usage"],
		"content":      content,
		"raw_response": raw,
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
	h.proxyAnthropic(c, "https://api.minimaxi.com/anthropic", h.cfg.MiniMax.APIKey, "/api/minimax/anthropic/v1/messages", []string{"MiniMax-M2.5", "MiniMax-M2.7"})
}

// ProxyDashScopeAnthropic 转发 Anthropic 协议格式的请求到 DashScope Anthropic 兼容端点
// POST /api/dashscope/anthropic/v1/messages
func (h *AIGatewayHandler) ProxyDashScopeAnthropic(c *gin.Context) {
	h.proxyAnthropic(c, "https://coding.dashscope.aliyuncs.com/apps/anthropic", h.cfg.DashScope.APIKey, "/api/dashscope/anthropic/v1/messages", []string{"qwen3.5-plus", "qwen3-max-2026-01-23", "qwen3-coder-next", "qwen3-coder-plus", "glm-5", "glm-4.7", "kimi-k2.5", "MiniMax-M2.5"})
}

// proxyAnthropic 转发 Anthropic 协议请求到指定上游
func (h *AIGatewayHandler) proxyAnthropic(c *gin.Context, upstreamBase, apiKey, logPath string, allowedModels []string) {
	key, ok := h.authenticateAPIKey(c, "chat")
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

	if !h.ensureModelAllowed(c, key, model) {
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
	resp, err := h.client.Do(req)
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
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	// 透传必要的 headers（过滤掉代理相关和可能导致 HTTP/2 问题的 headers）
	skipHeaders := map[string]bool{
		"Content-Type":     true,
		"Authorization":    true,
		"Accept":           true,
		"Connection":       true,
		"Proxy-Connection": true,
		"Upgrade":          true,
		"Keep-Alive":       true,
	}
	for key, values := range headers {
		if skipHeaders[key] {
			continue
		}
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("上游返回错误(%d): %s", resp.StatusCode, truncateString(string(respBody), 400))
	}
	return respBody, nil
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
	"glm-5":               true,
	"glm-4.7":             true,
	"kimi-k2.5":           true,
}

func (h *AIGatewayHandler) resolveChatProvider(model string) string {
	// DashScope 优先（若配置了 API Key）
	if h.cfg.DashScope.APIKey != "" && dashscopeModels[model] {
		return "dashscope"
	}
	switch model {
	case fallbackString(h.cfg.DeepSeek.Model, "deepseek-chat"), "deepseek-chat":
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
