package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devtools/config"

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
	}
}

// resolveAnthropicProvider 根据模型名查找匹配的提供商
// 优先从配置文件查找，未配置则使用内置默认列表
func (h *AIGatewayHandler) resolveAnthropicProvider(model string) (*config.AnthropicProviderConfig, bool) {
	providers := h.cfg.AIGateway.AnthropicProviders
	if len(providers) == 0 {
		providers = h.builtinAnthropicProviders()
	} else {
		providers = h.fillProviderAPIKeys(providers)
	}
	for i := range providers {
		for _, m := range providers[i].Models {
			if m == model {
				return &providers[i], true
			}
		}
	}
	return nil, false
}

// allAnthropicProviders 返回所有可用的提供商（配置优先，否则内置）
// 自动填充空的 APIKey 字段（从对应 provider 的配置 fallback）
func (h *AIGatewayHandler) allAnthropicProviders() []config.AnthropicProviderConfig {
	if len(h.cfg.AIGateway.AnthropicProviders) > 0 {
		return h.fillProviderAPIKeys(h.cfg.AIGateway.AnthropicProviders)
	}
	return h.builtinAnthropicProviders()
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
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("未找到支持模型 %s 的提供商", model)})
		return
	}

	h.proxyAnthropic(c, provider, "/api/anthropic/v1/messages", provider.Models)
}

// resolveProviderByNameOrModel 根据名称或模型查找提供商，优先匹配配置，否则回退内置
func (h *AIGatewayHandler) resolveProviderByNameOrModel(name, fallbackModel string) *config.AnthropicProviderConfig {
	providers := h.allAnthropicProviders()
	for i := range providers {
		if providers[i].Name == name {
			return &providers[i]
		}
	}
	// fallback to builtin list
	for _, p := range h.builtinAnthropicProviders() {
		if p.Name == name {
			return &p
		}
	}
	// last resort: try model lookup
	p, _ := h.resolveAnthropicProvider(fallbackModel)
	return p
}

func (h *AIGatewayHandler) proxyAnthropic(c *gin.Context, provider *config.AnthropicProviderConfig, logPath string, allowedModels []string) {
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

func isModelAllowed(model string, allowed []string) bool {
	for _, m := range allowed {
		if m == model {
			return true
		}
	}
	return false
}
