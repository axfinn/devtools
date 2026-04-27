package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

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
	providers := h.allAnthropicProviders()

	// 动态构建 provider 文档列表
	providerDocs := make([]gin.H, 0, len(providers))
	providerNameMap := map[string]string{
		"MiniMax":       "/api/minimax/anthropic",
		"DashScope":     "/api/dashscope/anthropic",
		"DeepSeek":      "/api/deepseek/anthropic",
		"PackyAPI":      "/api/anthropic",
		"OpenClaudeCode": "/api/anthropic",
	}
	for _, p := range providers {
		baseURL, ok := providerNameMap[p.Name]
		if !ok {
			baseURL = "/api/anthropic" // 通用端点
		}
		desc := p.Name + " Anthropic 兼容端点"
		// 合并别名和直通模型为用户可见模型列表
		userModels := make([]string, 0, len(p.Aliases)+len(p.Models))
		for _, a := range p.Aliases {
			userModels = append(userModels, a.Model)
		}
		userModels = append(userModels, p.Models...)

		// 构建别名文档
		aliasDocs := make([]gin.H, 0, len(p.Aliases))
		for _, a := range p.Aliases {
			aliasDocs = append(aliasDocs, gin.H{
				"model":          a.Model,
				"upstream_model": a.UpstreamModel,
			})
		}

		providerDocs = append(providerDocs, gin.H{
			"name":        p.Name,
			"base_url":    baseURL,
			"generic_url": "/api/anthropic",
			"upstream":    p.APIURL,
			"models":      p.Models,
			"aliases":     aliasDocs,
			"user_models": userModels,
			"description": desc,
		})
	}

	// 构建路由文档
	routes := []gin.H{
		{"method": "GET", "path": "/api/ai-gateway/docs/anthropic", "description": "获取本文档"},
		{"method": "POST", "path": "/api/anthropic/v1/messages", "description": "通用 Anthropic 端点（自动路由到匹配的下游）"},
	}
	// 保留历史端点信息
	for _, p := range providers {
		if baseURL, ok := providerNameMap[p.Name]; ok {
			routes = append(routes, gin.H{
				"method":      "POST",
				"path":        baseURL + "/v1/messages",
				"description": p.Name + " Anthropic 接口（兼容保留）",
			})
		}
	}

	// 收集所有模型用于 Claude Code 配置示例
	allModels := make([]string, 0)
	providerModels := make([]gin.H, 0)
	for _, p := range providers {
		for _, a := range p.Aliases {
			allModels = append(allModels, a.Model)
		}
		for _, m := range p.Models {
			allModels = append(allModels, m)
		}
		userModels := make([]string, 0, len(p.Aliases)+len(p.Models))
		for _, a := range p.Aliases {
			userModels = append(userModels, a.Model)
		}
		userModels = append(userModels, p.Models...)
		aliasList := make([]gin.H, 0, len(p.Aliases))
		for _, a := range p.Aliases {
			aliasList = append(aliasList, gin.H{
				"model":          a.Model,
				"upstream_model": a.UpstreamModel,
			})
		}
		providerModels = append(providerModels, gin.H{
			"name":        p.Name,
			"models":      p.Models,
			"aliases":     aliasList,
			"user_models": userModels,
		})
	}

	firstModel := "MiniMax-M2.5"
	if len(allModels) > 0 {
		firstModel = allModels[0]
	}
	// 尽量选取不同提供商的模型作为 Claude Code 各种角色的示例
	fastModel := firstModel
	sonnetModel := firstModel
	opusModel := firstModel
	haikuModel := firstModel
	if len(allModels) > 1 {
		fastModel = allModels[1%len(allModels)]
	}
	if len(allModels) > 2 {
		sonnetModel = allModels[2%len(allModels)]
	}
	if len(allModels) > 3 {
		haikuModel = allModels[3%len(allModels)]
	}

	c.JSON(http.StatusOK, gin.H{
		"title":   "Anthropic 协议接入文档",
		"summary": "通过 AI Gateway 的通用 Anthropic 端点 `/api/anthropic/v1/messages`，根据 model 自动路由到下游提供商。可通过管理后台选择下游，config.yaml 配置固定下游连接信息。",
		"auth": gin.H{
			"api_key": "Authorization: Bearer dtk_ai_xxx",
		},
		"providers": providerDocs,
		"routes":    routes,
		"examples": gin.H{
			"generic": gin.H{
				"description": "使用通用端点（推荐）",
				"request": gin.H{
					"model":      firstModel,
					"messages":   []gin.H{{"role": "user", "content": "你好"}},
					"max_tokens": 1024,
				},
				"claude_code_config": gin.H{
					"language":    "Claude Code",
					"description": "不同角色可用不同下游模型，BASE_URL 统一指向通用端点",
					"code": `{
  "env": {
    "ANTHROPIC_BASE_URL": "https://your-devtools:8080/api/anthropic",
    "ANTHROPIC_AUTH_TOKEN": "dtk_ai_xxx",
    "API_TIMEOUT_MS": "300000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": 1,
    "ANTHROPIC_MODEL": "` + firstModel + `",
    "ANTHROPIC_SMALL_FAST_MODEL": "` + fastModel + `",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "` + sonnetModel + `",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "` + opusModel + `",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "` + haikuModel + `"
  }
}`,
					},
					"provider_models": providerModels,
			},
			"python_sdk": gin.H{
				"language": "Python",
				"code": `from anthropic import Anthropic

client = Anthropic(
    base_url="http://your-devtools:8080/api/anthropic/v1",
    api_key="dtk_ai_xxx"  # 你的 AI Gateway API Key
)

response = client.messages.create(
    model="` + firstModel + `",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}]
)
print(response.content[0].text)`,
			},
			"javascript_sdk": gin.H{
				"language": "JavaScript/TypeScript",
				"code": `import { Anthropic } from '@anthropic-ai/sdk';

const client = new Anthropic({
  baseURL: 'http://your-devtools:8080/api/anthropic/v1',
  apiKey: 'dtk_ai_xxx', // 你的 AI Gateway API Key
});

async function main() {
  const message = await client.messages.create({
    model: '` + firstModel + `',
    max_tokens: 1024,
    messages: [{ role: 'user', content: 'Hello' }],
  });
  console.log(message.content[0].text);
}
main();`,
			},
			"curl": gin.H{
				"language": "cURL",
				"code": `curl -X POST http://your-devtools:8080/api/anthropic/v1/messages \\
  -H "Authorization: Bearer dtk_ai_xxx" \\
  -H "Content-Type: application/json" \\
  -H "Anthropic-Version: 2023-06-01" \\
  -d '{
    "model": "` + firstModel + `",
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
	// Anthropic 协议模型（通用端点，支持别名）
	anthropicProviders := h.allAnthropicProviders()
	for _, p := range anthropicProviders {
		// 别名模型
		for _, a := range p.Aliases {
			catalog = append(catalog, gin.H{
				"model":          a.Model,
				"provider":       strings.ToLower(p.Name),
				"type":           "anthropic",
				"endpoint":       "/api/anthropic/v1/messages",
				"description":    p.Name + " Anthropic 别名模型 → " + a.UpstreamModel,
				"upstream_model": a.UpstreamModel,
			})
		}
		// 直通模型
		for _, m := range p.Models {
			catalog = append(catalog, gin.H{
				"model":       m,
				"provider":    strings.ToLower(p.Name),
				"type":        "anthropic",
				"endpoint":    "/api/anthropic/v1/messages",
				"description": p.Name + " Anthropic 直通模型",
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

	// 检查是否为 Anthropic 模型，走 Anthropic 上游测试
	if provider, found := h.resolveAnthropicProvider(req.Model); found {
		h.testAnthropicModel(c, req.Model, prompt, provider)
		return
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

// AdminListAnthropicProviders 列出所有 Anthropic 下游提供商
// GET /api/ai-gateway/admin/anthropic/providers
func (h *AIGatewayHandler) AdminListAnthropicProviders(c *gin.Context) {
	if !h.requireSuperAdmin(c, "") {
		return
	}
	providers, err := h.db.ListAnthropicProviders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取提供商列表失败"})
		return
	}
	if providers == nil {
		providers = []*models.AnthropicProvider{}
	}
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// AdminCreateAnthropicProvider 新增 Anthropic 下游提供商
// POST /api/ai-gateway/admin/anthropic/providers
func (h *AIGatewayHandler) AdminCreateAnthropicProvider(c *gin.Context) {
	var req struct {
		SuperAdminPassword string `json:"super_admin_password"`
		Name               string `json:"name" binding:"required"`
		APIURL             string `json:"api_url" binding:"required"`
		APIKey             string `json:"api_key"`
		Models             string `json:"models"`
		Aliases            string `json:"aliases"`
		IsDefault          bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必填字段 name / api_url"})
		return
	}
	if !h.requireSuperAdmin(c, req.SuperAdminPassword) {
		return
	}
	if req.Models == "" {
		req.Models = "[]"
	}
	if req.Aliases == "" {
		req.Aliases = "[]"
	}
	p := &models.AnthropicProvider{
		Name:      req.Name,
		APIURL:    req.APIURL,
		APIKey:    req.APIKey,
		Models:    req.Models,
		Aliases:   req.Aliases,
		Enabled:   true,
		IsDefault: req.IsDefault,
	}
	if err := h.db.CreateAnthropicProvider(p); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "创建失败: " + err.Error()})
		return
	}
	// 如果设为默认，更新其他 provider 为非默认
	if p.IsDefault {
		h.db.SetDefaultAnthropicProvider(p.ID)
	}
	c.JSON(http.StatusCreated, gin.H{"provider": p})
}

// AdminUpdateAnthropicProvider 更新 Anthropic 下游提供商
// PUT /api/ai-gateway/admin/anthropic/providers/:id
func (h *AIGatewayHandler) AdminUpdateAnthropicProvider(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		SuperAdminPassword string `json:"super_admin_password"`
		Name               string `json:"name"`
		APIURL             string `json:"api_url"`
		APIKey             string `json:"api_key"`
		Models             string `json:"models"`
		Aliases            string `json:"aliases"`
		Enabled            *bool  `json:"enabled"`
		IsDefault          *bool  `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if !h.requireSuperAdmin(c, req.SuperAdminPassword) {
		return
	}
	pid := int64(0)
	if _, err := fmt.Sscanf(id, "%d", &pid); err != nil || pid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	existing, err := h.db.GetAnthropicProviderByID(pid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "提供商不存在"})
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.APIURL != "" {
		existing.APIURL = req.APIURL
	}
	if req.APIKey != "" {
		existing.APIKey = req.APIKey
	}
	if req.Models != "" {
		existing.Models = req.Models
	}
	if req.Aliases != "" {
		existing.Aliases = req.Aliases
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.IsDefault != nil {
		existing.IsDefault = *req.IsDefault
	}
	if err := h.db.UpdateAnthropicProvider(existing); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "更新失败: " + err.Error()})
		return
	}
	// 如果设为默认，更新其他 provider 为非默认
	if existing.IsDefault {
		h.db.SetDefaultAnthropicProvider(existing.ID)
	}
	c.JSON(http.StatusOK, gin.H{"provider": existing})
}

// AdminDeleteAnthropicProvider 删除 Anthropic 下游提供商
// DELETE /api/ai-gateway/admin/anthropic/providers/:id
func (h *AIGatewayHandler) AdminDeleteAnthropicProvider(c *gin.Context) {
	id := c.Param("id")
	if !h.requireSuperAdmin(c, c.Query("super_admin_password")) {
		return
	}
	pid := int64(0)
	if _, err := fmt.Sscanf(id, "%d", &pid); err != nil || pid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.db.DeleteAnthropicProvider(pid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
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
