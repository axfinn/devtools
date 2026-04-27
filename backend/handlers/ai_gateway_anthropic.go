package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// builtinAnthropicProviders 返回默认内置的 Anthropic 提供商列表
// 当 config 中未配置 anthropic_providers 时使用
func (h *AIGatewayHandler) builtinAnthropicProviders() []config.AnthropicProviderConfig {
	return []config.AnthropicProviderConfig{
		{
			Name:   "MiniMax",
			APIURL: "https://api.minimaxi.com/anthropic",
			APIKey: h.cfg.MiniMax.APIKey,
			Models: []string{"MiniMax-M2.5", "MiniMax-M2.5-highspeed", "MiniMax-M2.1", "MiniMax-M2.1-highspeed", "MiniMax-M2", "MiniMax-M2.7"},
		},
		{
			Name:   "DashScope",
			APIURL: "https://coding.dashscope.aliyuncs.com/apps/anthropic",
			APIKey: h.cfg.DashScope.APIKey,
			Models: []string{"qwen3.5-plus", "qwen3-max-2026-01-23", "qwen3-coder-next", "qwen3-coder-plus", "glm-5", "glm-4.7", "kimi-k2.5", "MiniMax-M2.5"},
		},
		{
			Name:   "DeepSeek",
			APIURL: "https://api.deepseek.com/anthropic",
			APIKey: h.cfg.DeepSeek.APIKey,
			Models: []string{"deepseek-chat", "deepseek-reasoner", "deepseek-v4-flash", "deepseek-v4-pro"},
		},
		{
			Name:   "PackyAPI",
			APIURL: "https://www.packyapi.com",
			APIKey: os.Getenv("ANTHROPIC_PACKYAPI_API_KEY"),
			Models: []string{"claude-opus-4-7", "claude-sonnet-4-6", "claude-haiku-4-5-20251001", "claude-sonnet-4-5"},
		},
		{
			Name:   "OpenClaudeCode",
			APIURL: "https://www.openclaudecode.cn",
			APIKey: os.Getenv("ANTHROPIC_OPENCLOUDECODE_API_KEY"),
			Models: []string{"claude-opus-4-7", "claude-sonnet-4-6", "claude-haiku-4-5-20251001", "claude-sonnet-4-5"},
		},
	}
}

// resolveModelUpstream 解析模型名 → 上游真实模型名
// 如果匹配到别名，返回 upstreamModel；否则返回原始 model（直通）
func resolveModelUpstream(provider *config.AnthropicProviderConfig, model string) string {
	for _, a := range provider.Aliases {
		if a.Model == model {
			return a.UpstreamModel
		}
	}
	return model // 直通
}

// providerUserModels 返回提供商对用户暴露的所有模型名（含别名）
func providerUserModels(provider *config.AnthropicProviderConfig) []string {
	models := make([]string, 0, len(provider.Models)+len(provider.Aliases))
	for _, a := range provider.Aliases {
		models = append(models, a.Model)
	}
	models = append(models, provider.Models...)
	return models
}

// providerAllModels 返回提供商所有模型（含别名），用于 allowedModels
func (h *AIGatewayHandler) providerAllModels(provider *config.AnthropicProviderConfig) []string {
	return providerUserModels(provider)
}

// resolveAnthropicProvider 根据模型名查找匹配的提供商（DB 优先，未匹配走默认线路）
func (h *AIGatewayHandler) resolveAnthropicProvider(model string) (*config.AnthropicProviderConfig, bool) {
	providers := h.allAnthropicProviders()
	for i := range providers {
		for _, a := range providers[i].Aliases {
			if a.Model == model {
				return &providers[i], true
			}
		}
		for _, m := range providers[i].Models {
			if m == model {
				return &providers[i], true
			}
		}
	}
	// Fallback: 默认线路
	if def := h.getDefaultAnthropicProviderConfig(); def != nil {
		return def, true
	}
	return nil, false
}

// getDefaultAnthropicProviderConfig 获取默认线路（config 格式）
func (h *AIGatewayHandler) getDefaultAnthropicProviderConfig() *config.AnthropicProviderConfig {
	// DB 优先
	if dbDef, err := h.db.GetDefaultAnthropicProvider(); err == nil && dbDef != nil {
		configs := h.dbAnthropicProvidersToConfig([]*models.AnthropicProvider{dbDef})
		if len(configs) > 0 {
			return &configs[0]
		}
	}
	// Builtin fallback: DeepSeek 作为默认
	builtins := h.builtinAnthropicProviders()
	for i := range builtins {
		if builtins[i].Name == "DeepSeek" {
			return &builtins[i]
		}
	}
	return nil
}

// allAnthropicProviders 返回所有可用的提供商（DB 优先，否则 config，否则内置）
func (h *AIGatewayHandler) allAnthropicProviders() []config.AnthropicProviderConfig {
	// 1. DB 存储优先
	dbProviders, err := h.db.ListEnabledAnthropicProviders()
	if err == nil && len(dbProviders) > 0 {
		return h.dbAnthropicProvidersToConfig(dbProviders)
	}
	// 2. config.yaml 配置
	if len(h.cfg.AIGateway.AnthropicProviders) > 0 {
		return h.fillProviderAPIKeys(h.cfg.AIGateway.AnthropicProviders)
	}
	// 3. 内置默认
	return h.builtinAnthropicProviders()
}

// dbAnthropicProvidersToConfig 将 DB 模型转换为 config 结构
func (h *AIGatewayHandler) dbAnthropicProvidersToConfig(dbProviders []*models.AnthropicProvider) []config.AnthropicProviderConfig {
	result := make([]config.AnthropicProviderConfig, 0, len(dbProviders))
	for _, p := range dbProviders {
		cfg := config.AnthropicProviderConfig{
			Name:         p.Name,
			APIURL:       p.APIURL,
			APIKey:       p.APIKey,
			DefaultModel: p.DefaultModel,
		}
		json.Unmarshal([]byte(p.Models), &cfg.Models)
		var aliases []config.AnthropicModelAlias
		if err := json.Unmarshal([]byte(p.Aliases), &aliases); err == nil {
			cfg.Aliases = aliases
		}
		if cfg.APIKey == "" {
			cfg.APIKey = h.fallbackAPIKeyForProvider(cfg.Name)
		}
		result = append(result, cfg)
	}
	return result
}

// fillProviderAPIKeys 为 api_key 为空的 provider 填充 fallback API Key
func (h *AIGatewayHandler) fillProviderAPIKeys(providers []config.AnthropicProviderConfig) []config.AnthropicProviderConfig {
	result := make([]config.AnthropicProviderConfig, len(providers))
	copy(result, providers)
	for i := range result {
		if result[i].APIKey == "" {
			result[i].APIKey = h.fallbackAPIKeyForProvider(result[i].Name)
		}
	}
	return result
}

// fallbackAPIKeyForProvider 根据 provider 名称返回对应的 fallback API Key
func (h *AIGatewayHandler) fallbackAPIKeyForProvider(name string) string {
	switch strings.ToLower(name) {
	case "minimax":
		return h.cfg.MiniMax.APIKey
	case "dashscope":
		return h.cfg.DashScope.APIKey
	case "deepseek":
		return h.cfg.DeepSeek.APIKey
	case "packyapi":
		return os.Getenv("ANTHROPIC_PACKYAPI_API_KEY")
	case "openclaudecode":
		return os.Getenv("ANTHROPIC_OPENCLOUDECODE_API_KEY")
	default:
		return ""
	}
}

// ProxyMinimaxAnthropic 转发 Anthropic 协议格式的请求到 MiniMax Anthropic 兼容端点
// POST /api/minimax/anthropic/v1/messages
func (h *AIGatewayHandler) ProxyMinimaxAnthropic(c *gin.Context) {
	p := h.resolveProviderByNameOrModel("MiniMax", "MiniMax-M2.5")
	h.proxyAnthropic(c, p, "/api/minimax/anthropic/v1/messages", []string{"MiniMax-M2.5", "MiniMax-M2.5-highspeed", "MiniMax-M2.1", "MiniMax-M2.1-highspeed", "MiniMax-M2", "MiniMax-M2.7"})
}

// ProxyDashScopeAnthropic 转发 Anthropic 协议格式的请求到 DashScope Anthropic 兼容端点
// POST /api/dashscope/anthropic/v1/messages
func (h *AIGatewayHandler) ProxyDashScopeAnthropic(c *gin.Context) {
	p := h.resolveProviderByNameOrModel("DashScope", "qwen3.5-plus")
	h.proxyAnthropic(c, p, "/api/dashscope/anthropic/v1/messages", []string{"qwen3.5-plus", "qwen3-max-2026-01-23", "qwen3-coder-next", "qwen3-coder-plus", "glm-5", "glm-4.7", "kimi-k2.5", "MiniMax-M2.5"})
}

// ProxyDeepSeekAnthropic 转发 Anthropic 协议格式的请求到 DeepSeek Anthropic 兼容端点
// POST /api/deepseek/anthropic/v1/messages
func (h *AIGatewayHandler) ProxyDeepSeekAnthropic(c *gin.Context) {
	p := h.resolveProviderByNameOrModel("DeepSeek", "deepseek-chat")
	h.proxyAnthropic(c, p, "/api/deepseek/anthropic/v1/messages", []string{"deepseek-chat", "deepseek-reasoner", "deepseek-v4-flash", "deepseek-v4-pro"})
}

// ProxyAnthropicGeneric 通用 Anthropic 协议代理端点
// POST /api/anthropic/v1/messages
// 根据请求中的 model 字段自动路由到匹配的下游提供商
// 支持模型别名：用户写别名，网关自动替换为上游真实模型名
func (h *AIGatewayHandler) ProxyAnthropicGeneric(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}
	if len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空"})
		return
	}

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

	provider, found := h.resolveAnthropicProvider(model)
	if !found {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("未找到支持模型 %s 的提供商，请查看 /api/ai-gateway/docs/anthropic 获取可用模型列表", model)})
		return
	}

	// 模型名重写：别名优先 → 未匹配走默认线路的 default_model
	upstreamModel := resolveModelUpstream(provider, model)
	if upstreamModel == model && !isModelAllowed(model, provider.Models) {
		// model 不在此 provider 的模型列表中，说明走了默认线路 — 用 provider 的默认模型
		if provider.DefaultModel != "" {
			upstreamModel = provider.DefaultModel
		}
	}
	allModels := h.providerAllModels(provider)

	h.proxyAnthropicWithRewrite(c, provider, "/api/anthropic/v1/messages", allModels, model, upstreamModel)
}

// resolveProviderByNameOrModel 根据名称或模型查找提供商，优先匹配配置，否则回退内置
func (h *AIGatewayHandler) resolveProviderByNameOrModel(name, fallbackModel string) *config.AnthropicProviderConfig {
	providers := h.allAnthropicProviders()
	for i := range providers {
		if providers[i].Name == name {
			return &providers[i]
		}
	}
	for _, p := range h.builtinAnthropicProviders() {
		if p.Name == name {
			return &p
		}
	}
	p, _ := h.resolveAnthropicProvider(fallbackModel)
	return p
}

func (h *AIGatewayHandler) proxyAnthropic(c *gin.Context, provider *config.AnthropicProviderConfig, logPath string, allowedModels []string) {
	h.proxyAnthropicWithRewrite(c, provider, logPath, allowedModels, "", "")
}

// proxyAnthropicWithRewrite 转发 Anthropic 请求到上游
// 如果 upstreamModel 与 userModel 不同，会重写请求体中的 model 字段
func (h *AIGatewayHandler) proxyAnthropicWithRewrite(c *gin.Context, provider *config.AnthropicProviderConfig, logPath string, allowedModels []string, userModel, upstreamModel string) {
	key, ok := h.authenticateAdminOrAPIKey(c, "chat")
	if !ok {
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	if len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空"})
		return
	}

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

	if !isModelAllowed(model, allowedModels) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("该端点不支持模型 %s，允许的模型: %v", model, allowedModels)})
		return
	}

	if key != nil && !h.ensureModelAllowed(c, key, model) {
		return
	}

	if provider == nil || provider.APIKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置上游 API Key"})
		return
	}

	// 模型名重写：用户侧别名 → 上游真实模型名
	rewriteModel := upstreamModel
	if rewriteModel == "" {
		rewriteModel = resolveModelUpstream(provider, model)
	}
	if rewriteModel != "" && rewriteModel != model {
		bodyMap["model"] = rewriteModel
		rewrittenBytes, err := json.Marshal(bodyMap)
		if err == nil {
			bodyBytes = rewrittenBytes
		}
	}

	start := time.Now()
	upstreamURL := strings.TrimRight(provider.APIURL, "/") + "/v1/messages"

	raw, err := h.doRawRequest(upstreamURL, provider.APIKey, "POST", bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "anthropic", logPath, "chat", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	h.logAPIRequest(key, model, "anthropic", logPath, "chat", http.StatusOK, true, "", string(bodyBytes), string(raw), c.ClientIP(), time.Since(start), usageSummary{})

	c.Data(http.StatusOK, "application/json", raw)
}

// testAnthropicModel 直连 Anthropic 上游测试模型可用性（AdminTestModel 调用）
func (h *AIGatewayHandler) testAnthropicModel(c *gin.Context, model, prompt string, provider *config.AnthropicProviderConfig) {
	upstreamModel := resolveModelUpstream(provider, model)
	body := map[string]interface{}{
		"model":      upstreamModel,
		"max_tokens": 256,
		"messages":   []map[string]interface{}{{"role": "user", "content": prompt}},
	}
	bodyBytes, _ := json.Marshal(body)
	upstreamURL := strings.TrimRight(provider.APIURL, "/") + "/v1/messages"

	start := time.Now()
	raw, err := h.doRawRequest(upstreamURL, provider.APIKey, "POST", bodyBytes, nil)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"model":          model,
			"upstream_model": upstreamModel,
			"provider":       provider.Name,
			"status":         "error",
			"error":          err.Error(),
			"latency":        latency,
		})
		return
	}

	var respMap map[string]interface{}
	if json.Unmarshal(raw, &respMap) == nil {
		// 提取 content
		if contentBlocks, ok := respMap["content"].([]interface{}); ok && len(contentBlocks) > 0 {
			if block, ok := contentBlocks[0].(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					usage, _ := respMap["usage"].(map[string]interface{})
					var tokens int
					if usage != nil {
						if it, ok := usage["input_tokens"].(float64); ok {
							tokens += int(it)
						}
						if ot, ok := usage["output_tokens"].(float64); ok {
							tokens += int(ot)
						}
					}
					c.JSON(http.StatusOK, gin.H{
						"model":          model,
						"upstream_model": upstreamModel,
						"provider":       provider.Name,
						"status":         "ok",
						"reply":          text,
						"latency":        latency,
						"tokens":         tokens,
					})
					return
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"model":          model,
		"upstream_model": upstreamModel,
		"provider":       provider.Name,
		"status":         "ok",
		"reply":          string(raw),
		"latency":        latency,
	})
}

func isModelAllowed(model string, allowed []string) bool {
	for _, m := range allowed {
		if m == model {
			return true
		}
	}
	return false
}
