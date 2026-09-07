package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

// modelFieldRe 匹配 JSON body 中 "model": "..." 字段
// 用于保留原始 JSON 格式替换 model 名，避免第三方代理检测到 body 被篡改
var modelFieldRe = regexp.MustCompile(`"model"\s*:\s*"[^"]*"`)

// ssePingFrame 是 Anthropic 协议官方 ping 心跳帧。
//
// 为什么不用 ": heartbeat\n\n"(SSE 注释行)?
// SSE 规范规定以 ":" 开头的行是注释,EventSource 不会派发 message 事件,
// 客户端 SDK 也不会 reset read timeout。Claude Code 用的是 Anthropic SDK,
// SDK 内部的 read deadline 只在收到 message 事件时才会 reset,因此纯注释行
// 无法阻止 ~15s 的 read_timeout 触发。
//
// "event: ping\ndata: {"type":"ping"}\n\n" 是 Anthropic /v1/messages SSE
// 协议的合法 ping 事件,SDK 必识别并 reset read deadline。同样的格式对
// curl/浏览器 EventSource 也都合法。
const ssePingFrame = "event: ping\ndata: {\"type\":\"ping\"}\n\n"

// builtinAnthropicProviders 返回默认内置的 Anthropic 提供商列表
// 当 config 中未配置 anthropic_providers 时使用
func (h *AIGatewayHandler) builtinAnthropicProviders() []config.AnthropicProviderConfig {
	return []config.AnthropicProviderConfig{
		{
			Name:   "MiniMax",
			APIURL: "https://api.minimaxi.com/anthropic",
			APIKey: h.cfg.MiniMax.APIKey,
			Models: []string{"MiniMax-M3", "MiniMax-M2.5", "MiniMax-M2.1", "MiniMax-M2", "MiniMax-M2.7"},
		},
		{
			Name:   "DashScope",
			APIURL: "https://coding.dashscope.aliyuncs.com/apps/anthropic",
			APIKey: h.cfg.DashScope.APIKey,
			Models: []string{"qwen3.5-plus", "qwen3-max-2026-01-23", "qwen3-coder-next", "qwen3-coder-plus", "glm-5", "glm-4.7", "kimi-k2.5", "MiniMax-M3", "MiniMax-M2.5"},
		},
		{
			Name:   "DeepSeek",
			APIURL: "https://api.deepseek.com/anthropic",
			APIKey: h.cfg.DeepSeek.APIKey,
			Models: []string{"deepseek-chat", "deepseek-reasoner", "deepseek-v4-flash", "deepseek-v4-pro"},
		},
		{
			Name:   "OpenClaudeCode",
			APIURL: "https://www.openclaudecode.cn",
			APIKey: os.Getenv("ANTHROPIC_OPENCLOUDECODE_API_KEY"),
			Models: []string{"claude-opus-4-7", "claude-sonnet-4-6", "claude-haiku-4-5-20251001", "claude-sonnet-4-5"},
		},
		{
			Name:   "PackyAPI",
			APIURL: "https://www.packyapi.com",
			APIKey: os.Getenv("ANTHROPIC_PACKYAPI_API_KEY"),
			Models: []string{"claude-opus-4-7", "claude-sonnet-4-6", "claude-haiku-4-5-20251001", "claude-sonnet-4-5"},
		},
	}
}

// ProxyAnthropicModels 返回可用模型列表
// GET /api/anthropic/v1/models
func (h *AIGatewayHandler) ProxyAnthropicModels(c *gin.Context) {
	providers := h.allAnthropicProviders()
	seen := map[string]bool{"gateway": true}
	type modelInfo struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	models := []modelInfo{{ID: "gateway", Type: "model"}}
	for _, p := range providers {
		for _, m := range p.Models {
			if !seen[m] {
				seen[m] = true
				models = append(models, modelInfo{ID: m, Type: "model"})
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": models})
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
// 解密 APIKeyEncrypted;若为空但 legacy APIKey 非空(升级前老数据),懒迁移一次:加密后写回 DB 并清明文。
func (h *AIGatewayHandler) dbAnthropicProvidersToConfig(dbProviders []*models.AnthropicProvider) []config.AnthropicProviderConfig {
	result := make([]config.AnthropicProviderConfig, 0, len(dbProviders))
	for _, p := range dbProviders {
		cfg := config.AnthropicProviderConfig{
			Name:         p.Name,
			APIURL:       p.APIURL,
			DefaultModel: p.DefaultModel,
			IsDefault:    p.IsDefault,
		}
		// 优先解密密文;失败/为空时回退 legacy 明文(并触发懒迁移)
		if p.APIKeyEncrypted != "" {
			plain, err := h.enc.Decrypt(p.APIKeyEncrypted)
			if err == nil {
				cfg.APIKey = plain
			} else {
				// 解密失败极少见(主密钥变更/数据损坏),回退 legacy 明文继续工作
				cfg.APIKey = p.APIKey
			}
		} else if p.APIKey != "" {
			// 旧明文:懒迁移一次
			cfg.APIKey = p.APIKey
			enc, err := h.enc.Encrypt(p.APIKey)
			if err == nil {
				p.APIKeyEncrypted = enc
				p.APIKey = ""
				// 异步写回,不阻塞当前请求(失败下次重启再迁移)
				go func(pid int64, cipher string) {
					_ = h.db.UpdateAnthropicProvider(&models.AnthropicProvider{
						ID:              pid,
						APIKeyEncrypted: cipher,
						APIKey:          "",
					})
				}(p.ID, enc)
			}
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
	// OpenClaudeCode 优先级高于 PackyAPI：当两者都匹配同一模型时，首选 OpenClaudeCode
	for i := 0; i < len(result); i++ {
		if result[i].Name == "PackyAPI" {
			for j := i + 1; j < len(result); j++ {
				if result[j].Name == "OpenClaudeCode" {
					result[i], result[j] = result[j], result[i]
					break
				}
			}
			break
		}
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
	p := h.resolveProviderByNameOrModel("MiniMax", "MiniMax-M3")
	h.proxyAnthropic(c, p, "/api/minimax/anthropic/v1/messages", []string{"MiniMax-M3", "MiniMax-M2.5", "MiniMax-M2.1", "MiniMax-M2", "MiniMax-M2.7"})
}

// ProxyDashScopeAnthropic 转发 Anthropic 协议格式的请求到 DashScope Anthropic 兼容端点
// POST /api/dashscope/anthropic/v1/messages
func (h *AIGatewayHandler) ProxyDashScopeAnthropic(c *gin.Context) {
	p := h.resolveProviderByNameOrModel("DashScope", "qwen3.5-plus")
	h.proxyAnthropic(c, p, "/api/dashscope/anthropic/v1/messages", []string{"qwen3.5-plus", "qwen3-max-2026-01-23", "qwen3-coder-next", "qwen3-coder-plus", "glm-5", "glm-4.7", "kimi-k2.5", "MiniMax-M3", "MiniMax-M2.5"})
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
// 如果 API Key 配置了 anthropic_provider_id，则强制走指定提供商
func (h *AIGatewayHandler) ProxyAnthropicGeneric(c *gin.Context) {
	// 1. 先认证，获取 API Key（可能为 nil = admin）
	authKey, authOK := h.authenticateAdminOrAPIKey(c, "chat")
	if !authOK {
		return
	}

	// 2. 读取请求体
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

	// 3. 确定下游提供商：Key 级别配置优先
	var provider *config.AnthropicProviderConfig
	if authKey != nil && authKey.AnthropicProviderID > 0 {
		// Key 指定了提供商 → 直接使用
		dbProvider, err := h.db.GetAnthropicProviderByID(int64(authKey.AnthropicProviderID))
		if err != nil || dbProvider == nil || !dbProvider.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "API Key 配置的下游提供商不可用，请联系管理员"})
			return
		}
		configs := h.dbAnthropicProvidersToConfig([]*models.AnthropicProvider{dbProvider})
		if len(configs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "API Key 配置的下游提供商不可用"})
			return
		}
		provider = &configs[0]
	} else {
		// 无 Key 级别配置 → 按模型名路由
		var found bool
		provider, found = h.resolveAnthropicProvider(model)
		if !found {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("未找到支持模型 %s 的提供商，请查看 /api/ai-gateway/docs/anthropic 获取可用模型列表", model)})
			return
		}
	}

	// 4. 模型名重写
	upstreamModel := resolveModelUpstream(provider, model)
	if upstreamModel == model && !isModelAllowed(model, provider.Models) {
		if provider.DefaultModel != "" {
			upstreamModel = provider.DefaultModel
		}
	}

	// 5. 重写 body 中的 model（正则替换，保留原始 JSON 格式）
	// json.Marshal 会改变字段序和数字精度，导致 OpenClaudeCode/PackyAPI
	// 等校验 body 完整性的上游拒绝请求。
	if upstreamModel != model {
		bodyBytes = rewriteModelField(bodyBytes, upstreamModel)
		bodyMap["model"] = upstreamModel // 保持 bodyMap 同步，后续 checkModel 用
	}

	allModels := h.allModelsAcrossProviders()

	// 6. 转发（传 preAuthKey 跳过内部认证）
	// userModel 用原始模型名，这样 key 的 allowed_models 可以用 "gateway" 等占位名
	//
	// 流式 / 非流式 分流:
	//   - 流式(stream:true)→ 现有路径,ssePingFrame 持续 reset read timeout(P0 修复)
	//   - 非流式 → 改走异步 task 模式,POST 立即 202 + Location 头,避免 CF origin
	//     read timeout 15s 把 Claude Code 混合调用里的非流式子任务(haiku 后台探测、
	//     compact summarization 等)cut 成 524。
	isStream, _ := bodyMap["stream"].(bool)
	if isStream {
		h.proxyAnthropicWithBody(c, provider, "/api/anthropic/v1/messages", allModels, model, upstreamModel, bodyBytes, bodyMap, authKey)
		return
	}

	// 非流式路径:转异步前先校验 key 的 allowed_models(同 proxyAnthropicWithBody 行 451 行为)。
	// 避免 model 不在白名单时还创建 task。
	if authKey != nil && !h.ensureModelAllowed(c, authKey, model) {
		return
	}

	// 非流式路径:转异步。
	// 客户端 SDK 看到 202 + Location 头通常会按 HTTP 语义去 poll /v1/messages/tasks/:id;
	// 即使 SDK 不识别 202 + task polling(标准 Anthropic SDK 不支持),
	// 至少不再触发"524 → 重试 10 次 → 全 524"的风暴。
	start := time.Now()
	endpoint := "/api/anthropic/v1/messages/tasks"
	taskID := "oant_" + utils.GenerateHexKey(12)

	// 防御: 上游 URL/Key 缺失,提前返清晰错误(同 proxyAnthropicWithBody 行 461-467)
	if provider.APIURL == "" {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":    fmt.Sprintf("Provider %s 未配置 API URL", provider.Name),
			"provider": provider.Name,
			"code":     502,
		})
		return
	}
	if provider.APIKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置上游 API Key", "code": 502})
		return
	}

	task := &models.AnthropicTask{
		ID:          taskID,
		APIKeyID:    firstAPIKeyID(authKey),
		Model:       model,
		Provider:    provider.Name,
		Status:      "pending",
		RequestBody: truncateString(string(bodyBytes), 50000),
		ClientIP:    c.ClientIP(),
	}
	if err := h.db.CreateAnthropicTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败", "code": 500})
		return
	}

	h.logAPIRequest(authKey, model, "anthropic", "/api/anthropic/v1/messages", "chat", http.StatusAccepted, true, "", string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})

	c.Header("Location", endpoint+"/"+taskID)
	c.JSON(http.StatusAccepted, gin.H{
		"task_id":    taskID,
		"model":      model,
		"provider":   provider.Name,
		"status":     "pending",
		"created_at": task.CreatedAt.Format(time.RFC3339),
		"poll_url":   endpoint + "/" + taskID,
		"message":    "Anthropic 异步任务已提交(由 /v1/messages 非流式自动转异步),请通过 GET " + endpoint + "/" + taskID + " 轮询结果",
	})

	// 后台跑上游调用。复用 runAsyncAnthropicTask 已有逻辑:doRawRequestLong 5min timeout + stripThinkingBlocks + 失败回写。
	upstreamURL := strings.TrimRight(provider.APIURL, "/") + "/v1/messages"
	go h.runAsyncAnthropicTask(taskID, provider, upstreamURL, bodyBytes, firstAPIKeyID(authKey))
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
	h.proxyAnthropicWithBody(c, provider, logPath, allowedModels, userModel, upstreamModel, nil, nil, nil)
}

// proxyAnthropicWithBody 与 proxyAnthropicWithRewrite 相同，但接受预读的 body 数据
// preReadBody/preReadMap 非 nil 时跳过 c.Request.Body 的读取
// preAuthKey 非 nil 时跳过认证，直接使用该 key（nil key = admin，非 nil key = 受限 key）
func (h *AIGatewayHandler) proxyAnthropicWithBody(c *gin.Context, provider *config.AnthropicProviderConfig, logPath string, allowedModels []string, userModel, upstreamModel string, preReadBody []byte, preReadMap map[string]interface{}, preAuthKey *models.AIAPIKey) {
	var key *models.AIAPIKey
	var ok bool
	if preAuthKey != nil {
		// 已认证的 API Key
		key = preAuthKey
		ok = true
	} else if preReadBody != nil {
		// preReadBody 非 nil 但 preAuthKey 为 nil → 由调用方提前认证
		// 此时不做认证，直接放行（调用方在外部已处理 auth）
		ok = true
	} else {
		key, ok = h.authenticateAdminOrAPIKey(c, "chat")
	}
	if !ok {
		return
	}

	var bodyBytes []byte
	var bodyMap map[string]interface{}

	if preReadBody != nil {
		bodyBytes = preReadBody
		bodyMap = preReadMap
	} else {
		var err error
		bodyBytes, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
			return
		}
		if len(bodyBytes) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空"})
			return
		}
		bodyMap = make(map[string]interface{})
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求体 JSON 格式错误"})
			return
		}
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

	// Key 的 allowed_models 用原始 userModel 校验，
	// 这样 key 写 "gateway" 也能通过，无需关心上游真实模型名
	checkModel := model
	if userModel != "" {
		checkModel = userModel
	}
	if key != nil && !h.ensureModelAllowed(c, key, checkModel) {
		return
	}

	if provider == nil || provider.APIKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置上游 API Key"})
		return
	}
	// 防御: URL 为空时直接给清晰错误,避免拼出 "/v1/messages"(无 host)导致请求"裸奔"。
	// 触发场景: admin 把一个空 provider 设为默认(default_provider),或新建 provider 没填 URL 就保存。
	if provider.APIURL == "" {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":    fmt.Sprintf("Provider %s 未配置 API URL,请在 AI Gateway 管理后台 → Anthropic 下游管理 → 编辑补全 URL", provider.Name),
			"provider": provider.Name,
		})
		return
	}

	// 模型名重写：用户侧别名 → 上游真实模型名
	rewriteModel := upstreamModel
	if rewriteModel == "" {
		rewriteModel = resolveModelUpstream(provider, model)
	}
	if rewriteModel != "" && rewriteModel != model {
		bodyBytes = rewriteModelField(bodyBytes, rewriteModel)
		bodyMap["model"] = rewriteModel
	}

	start := time.Now()
	upstreamURL := strings.TrimRight(provider.APIURL, "/") + "/v1/messages"

	// 流式请求：直接透传 SSE，不解包
	if isStreaming, _ := bodyMap["stream"].(bool); isStreaming {
		statusCode, streamErr := h.proxyAnthropicStream(c, provider, upstreamURL, bodyBytes)
		// context.Canceled = 客户端主动断开，属于正常中断，不计为失败
		streamSuccess := streamErr == nil || streamErr == context.Canceled
		streamErrMsg := safeError(streamErr)
		if streamErr == context.Canceled {
			streamErrMsg = ""
		}
		h.logAPIRequest(key, model, "anthropic", logPath, "chat", statusCode, streamSuccess, streamErrMsg, string(bodyBytes), "[stream]", c.ClientIP(), time.Since(start), usageSummary{})
		return
	}

	// 非流式转发用长超时客户端：DeepSeek reasoner / MiniMax M3 长推理响应常超 90s。
	raw, err := h.doRawRequestLong(upstreamURL, provider.APIKey, "POST", bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "anthropic", logPath, "chat", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// DeepSeek / Ollama(qwen3 系列)等上游会返回 type="thinking" 内容块,
	// 标准 Anthropic 客户端下一轮请求把 thinking 原样回传时会 400；网关层统一剥离。
	if providerEmitsThinkingBlocks(provider) {
		raw = stripThinkingBlocks(raw)
	}

	h.logAPIRequest(key, model, "anthropic", logPath, "chat", http.StatusOK, true, "", string(bodyBytes), string(raw), c.ClientIP(), time.Since(start), usageSummary{})

	c.Data(http.StatusOK, "application/json", raw)
}

// proxyAnthropicStream 流式代理 Anthropic 请求（SSE 透传）
// 对于 DeepSeek/Ollama，过滤掉 type="thinking" 的内容块事件
func (h *AIGatewayHandler) proxyAnthropicStream(c *gin.Context, provider *config.AnthropicProviderConfig, upstreamURL string, bodyBytes []byte) (int, error) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// 先校验 Flusher（响应头一旦写入就不能改状态码了，必须在 WriteHeader 前检查）
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("streaming not supported")
	}

	// 关键修复: 立即发送 SSE 响应头 + 第一个 ping heartbeat,**然后再拨号上游**。
	// Ollama 本地大模型(27B+)处理 Claude Code 的超长 system prompt(15K+ tokens)
	// 需要 30+ 秒才能返回第一个字节;如果等上游响应头到了再写 header+heartbeat,
	// Claude Code 等不到任何字节会在 ~15s 断开(read timeout,context canceled),表现为 "API error" 重试。
	//
	// 心跳格式必须用 `event: ping\ndata: {"type":"ping"}\n\n`(Anthropic 官方 ping 事件),
	// 而不是 SSE 注释行 `: heartbeat\n\n`。注释行不触发 message 事件,EventSource 不会
	// reset read timeout;ping event 是 Anthropic SDK 显式识别的合法心跳,必触发 message
	// handler → SDK 内部 reset read deadline。
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	if _, err := io.WriteString(c.Writer, ssePingFrame); err != nil {
		return http.StatusOK, context.Canceled
	}
	flusher.Flush()

	// 上游拨号期间的高频 ping(每 1s):Claude Code read_timeout 约 15s,即使下游要 30s+
	// 才回首字节,持续 1s 一次的 ping event 也能让 SDK 内部反复 reset read deadline。
	// 频率不能太高:每次 Write+Flush 都是 syscall,1s 平衡"reset 间隔"与"goroutine 开销"。
	preUpstreamHeartbeatDone := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in pre-upstream heartbeat goroutine: %v", r)
			}
		}()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := io.WriteString(c.Writer, ssePingFrame); err != nil {
					return
				}
				flusher.Flush()
			case <-preUpstreamHeartbeatDone:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	req, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(bodyBytes))
	if err != nil {
		close(preUpstreamHeartbeatDone)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return http.StatusBadGateway, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	// 透传客户端的 Anthropic 相关头（先于默认值设置，保留客户端自定义）
	for key, values := range c.Request.Header {
		ck := http.CanonicalHeaderKey(key)
		switch ck {
		case "Content-Type", "Authorization", "Content-Length", "Host",
			"Connection", "Transfer-Encoding", "Te", "Trailer",
			"Keep-Alive", "Proxy-Connection", "Upgrade", "Accept-Encoding":
			continue
		}
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}

	// 与直连路径对齐：补齐 anthropic-version + x-api-key，
	// 客户端没传时也能保证上游识别为 Anthropic 协议（MiniMax/DeepSeek 必需）。
	if req.Header.Get("x-api-key") == "" {
		req.Header.Set("x-api-key", provider.APIKey)
	}
	if req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	// 使用 streamClient（body 无超时），避免 http.Client.Timeout 在长流时强制断开；
	// 连接/握手/响应头超时由 streamTransport 兜底，瞬时连接错误在拿响应前做一次重试。
	resp, err := doStreamRequest(h.streamClient, req)
	// 上游开始响应后停掉拨号期高频心跳,改用主循环的 20s 心跳(防止 thinking 阶段断流)
	close(preUpstreamHeartbeatDone)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", errBody)
		return resp.StatusCode, fmt.Errorf("upstream error %d", resp.StatusCode)
	}

	// 心跳与主循环并发写 c.Writer，用 sseWriter 串行化 Write+Flush 避免数据竞争。
	writer := newSSEWriter(c.Writer, flusher)

	// 持续心跳 goroutine：thinking 阶段长(本地 27B 模型常 30s+)无字节输出,
	// 中间代理(nginx/CDN)或客户端会因空闲超时断开。改 5s 一次 ping event,
	// 与 Anthropic SDK 的 ping 频率对齐,既保证 read timeout 持续 reset 又不过度刷流量。
	heartbeatDone := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in background goroutine: %v", r)
			}
		}()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := writer.write([]byte(ssePingFrame)); err != nil {
					return
				}
			case <-ctx.Done():
				return
			case <-heartbeatDone:
				return
			}
		}
	}()
	defer close(heartbeatDone)

	buf := make([]byte, 1024)
	// DeepSeek / Ollama(qwen3 系列)等上游会返回 type="thinking" 的内容块，标准 Anthropic 客户端
	// 无法识别，多轮对话会 400；这里在网关层过滤掉 thinking 事件再透传。
	var bodyReader io.Reader = resp.Body
	if providerEmitsThinkingBlocks(provider) {
		bodyReader = newSSEThinkingFilter(resp.Body)
	}
	for {
		n, readErr := bodyReader.Read(buf)
		if n > 0 {
			if err := writer.write(buf[:n]); err != nil {
				cancel() // 客户端断开，取消上游请求
				return http.StatusOK, context.Canceled
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				// 上游读取错误：若是 context 取消（客户端断开），统一返回 context.Canceled
				if c.Request.Context().Err() != nil {
					return http.StatusOK, context.Canceled
				}
				return http.StatusOK, readErr
			}
			break
		}
	}
	return http.StatusOK, nil
}

// sseThinkingFilter 过滤 DeepSeek / Ollama(qwen3 系列)SSE 流中的 type="thinking" 内容块
// 以 SSE event 为单位(空行分隔),若事件属于 thinking 块则整体丢弃。
//
// 实现细节: 用 bufio.Reader.ReadString('\n') 而不是 bufio.Scanner,因为 Scanner 的
// MaxScanTokenSize 默认 64KB,Claude Code 的首个 message_start event(JSON 包含
// 超长 system prompt + tools 定义)经常超过 64KB,Scanner 会抛 bufio.ErrTooLong,
// 整个流后续被截断。ReadString 没有 token size 限制,只受初始 buffer 影响。
type sseThinkingFilter struct {
	src         *bufio.Reader
	skipIndices map[int]bool // 被标记为 thinking 的 content_block index
	pending     []byte       // 已过滤、待输出的数据
}

func newSSEThinkingFilter(src io.Reader) io.Reader {
	return &sseThinkingFilter{
		// 1MB 初始 buffer:Claude Code 长 system prompt + tools 的 message_start
		// event JSON 经常 100-500KB,1MB 起步覆盖 99% 场景;ReadString 会按需自动 grow。
		src:         bufio.NewReaderSize(src, 1<<20),
		skipIndices: make(map[int]bool),
	}
}

func (f *sseThinkingFilter) Read(p []byte) (int, error) {
	// 如果还有待输出数据，直接返回
	if len(f.pending) > 0 {
		n := copy(p, f.pending)
		f.pending = f.pending[n:]
		return n, nil
	}

	// 累积一个完整 SSE event(遇空行为止)。
	// ReadString 不会因为单行过长而失败,只受 reader buffer 限制(且自动 grow)。
	var lines []string
	for {
		line, err := f.src.ReadString('\n')
		// ReadString 在遇到错误时仍返回已读取的数据;若完全没读到任何字节才返回 err。
		// 这种情况(stream 末尾或网络中断)直接退出,让外层处理 EOF/错误。
		if len(line) == 0 {
			if err != nil {
				return 0, err
			}
			continue
		}
		// 去掉行尾的 \r\n,后续统一用 "\n" 重新拼接,保持输出格式一致。
		line = strings.TrimRight(line, "\r\n")
		lines = append(lines, line)
		if line == "" {
			// 空行 = event 结束
			break
		}
		if err != nil {
			// 流中途错误(不是 EOF):已读到的部分也作为 event 提交,交给 updateSkipState/shouldSkip 处理,
			// 之后再把 err 透传给外层。但通常事件以空行结束,这种情况极少见,先简单处理:返回 EOF。
			if err == io.EOF {
				break
			}
			return 0, err
		}
	}

	// 判断该 event 是否属于 thinking 块
	f.updateSkipState(lines)

	if f.shouldSkip(lines) {
		// 跳过这个 event，继续读下一个
		return f.Read(p)
	}

	// 输出这个 event
	var buf bytes.Buffer
	for _, l := range lines {
		buf.WriteString(l + "\n")
	}
	f.pending = buf.Bytes()
	if len(f.pending) == 0 {
		return f.Read(p)
	}
	n := copy(p, f.pending)
	f.pending = f.pending[n:]
	return n, nil
}

func (f *sseThinkingFilter) updateSkipState(lines []string) {
	for _, line := range lines {
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		jsonStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var event map[string]interface{}
		if json.Unmarshal([]byte(jsonStr), &event) != nil {
			continue
		}
		typ, _ := event["type"].(string)
		idx := eventIndex(event)

		switch typ {
		case "content_block_start":
			if cb, _ := event["content_block"].(map[string]interface{}); cb != nil {
				if blockType, _ := cb["type"].(string); blockType == "thinking" {
					f.skipIndices[idx] = true
				}
			}
		case "content_block_stop":
			delete(f.skipIndices, idx)
		}
	}
}

func (f *sseThinkingFilter) shouldSkip(lines []string) bool {
	for _, line := range lines {
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		jsonStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var event map[string]interface{}
		if json.Unmarshal([]byte(jsonStr), &event) != nil {
			continue
		}
		if f.skipIndices[eventIndex(event)] {
			return true
		}
	}
	return false
}

func eventIndex(event map[string]interface{}) int {
	if idx, ok := event["index"].(float64); ok {
		return int(idx)
	}
	return -1
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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

// allModelsAcrossProviders 返回所有提供商的所有模型（含别名），用于通用端点校验
func (h *AIGatewayHandler) allModelsAcrossProviders() []string {
	providers := h.allAnthropicProviders()
	models := make([]string, 0)
	for i := range providers {
		for _, a := range providers[i].Aliases {
			models = append(models, a.Model)
		}
		for _, m := range providers[i].Models {
			models = append(models, m)
		}
	}
	return models
}

// rewriteModelField 用正则替换 JSON body 中的 "model": "..." 字段值
// 保留原始 JSON 格式（空白、字段序、数字精度），避免 json.Unmarshal+Marshal
// 导致的 body 变化被 OpenClaudeCode/PackyAPI 等上游检测为篡改。
func rewriteModelField(body []byte, newModel string) []byte {
	return modelFieldRe.ReplaceAll(body, []byte(`"model":"`+newModel+`"`))
}

// stripThinkingBlocks 从 Anthropic 响应 JSON 中移除 type="thinking" 的内容块
// 某些下游（DeepSeek）返回 thinking 块要求客户端原样回传，但标准 Anthropic 客户端
// 不识此格式，导致后续多轮对话 400 错误。直接在网关层剥离。
func stripThinkingBlocks(raw []byte) []byte {
	var resp map[string]interface{}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return raw // 非 JSON 或格式异常，原样返回
	}
	content, ok := resp["content"].([]interface{})
	if !ok || len(content) == 0 {
		return raw
	}
	filtered := make([]interface{}, 0, len(content))
	hasThinking := false
	for _, block := range content {
		b, ok := block.(map[string]interface{})
		if ok && b["type"] == "thinking" {
			hasThinking = true
			continue
		}
		filtered = append(filtered, block)
	}
	if !hasThinking {
		return raw
	}
	resp["content"] = filtered
	out, err := json.Marshal(resp)
	if err != nil {
		return raw
	}
	return out
}

// providerEmitsThinkingBlocks 判断上游是否会在 Anthropic 响应中夹带 type="thinking" 内容块。
// DeepSeek(Official) 与 Ollama(qwen3 系列默认开启 extended thinking)都属于这一类,
// 标准 Anthropic SDK 不识别 thinking 块,多轮对话回传会 400,需在网关层剥离。
func providerEmitsThinkingBlocks(provider *config.AnthropicProviderConfig) bool {
	if provider == nil {
		return false
	}
	name := strings.ToLower(provider.Name)
	url := strings.ToLower(provider.APIURL)
	// 名称匹配
	if name == "deepseek" || name == "ollama" {
		return true
	}
	// URL 启发式:DeepSeek 官方域 / Ollama 默认 11434 端口
	if strings.Contains(url, "deepseek.com") {
		return true
	}
	if strings.HasSuffix(url, ":11434") || strings.Contains(url, ":11434/") {
		return true
	}
	return false
}

// isModelAllowed 判断 model 是否在白名单中
func isModelAllowed(model string, allowed []string) bool {
	for _, m := range allowed {
		if m == model {
			return true
		}
	}
	return false
}

// ===================== 异步 Anthropic 代理(task 模式,防 CF 524) =====================
//
// 背景: 流式 Anthropic 请求已经被 ssePingFrame 修复(P0 改动),但非流式请求走
// doRawRequestLong(5min) 等 Ollama 返回,CF(t.jaxiu.cn) origin read timeout 15s
// 会直接把客户端掐了返 524。
//
// 方案: 新增独立端点 POST /api/anthropic/v1/messages/tasks,立即返 202 + task_id,
// 后台 goroutine 调 Ollama,客户端 GET /api/anthropic/v1/messages/tasks/:id 轮询。
// CF 只看到 <500ms 的 POST→202,看不到长上游调用。
//
// 为什么不动原 POST /api/anthropic/v1/messages? Claude Code 默认走流式,同步
// 路径仍可能有客户端(admin 后台、其他工具)需要,改异步会破坏现有调用方。

// AsyncAnthropicMessages POST /api/anthropic/v1/messages/tasks
// 提交一个异步 Anthropic /v1/messages 调用,立即返回 202 + task_id。
//
// body 必须是合法 Anthropic messages 协议 JSON,但 stream 字段必须为 false 或缺省
// (流式请走原 POST /api/anthropic/v1/messages)。
func (h *AIGatewayHandler) AsyncAnthropicMessages(c *gin.Context) {
	start := time.Now()
	endpoint := "/api/anthropic/v1/messages/tasks"

	key, ok := h.authenticateAdminOrAPIKey(c, "chat")
	if !ok {
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败", "code": 400})
		return
	}
	if len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空", "code": 400})
		return
	}

	var probe map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &probe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体 JSON 格式错误", "code": 400})
		return
	}

	// 异步端点只处理非流式。流式场景流式 SSE 已经在第一个 heartbeat 阶段告诉客户端连接已建立,
	// read_timeout 持续 reset,根本不存在 524 问题,不需要异步化。
	if isStream, _ := probe["stream"].(bool); isStream {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "异步端点不支持 stream:true,请用 POST /api/anthropic/v1/messages(流式)",
			"code":  400,
		})
		return
	}

	model, _ := probe["model"].(string)
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model 字段", "code": 400})
		return
	}

	// 解析下游 provider
	provider, found := h.resolveAnthropicProvider(model)
	if !found {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("未找到支持模型 %s 的提供商,请查看 /api/ai-gateway/docs/anthropic 获取可用模型列表", model),
			"code":  400,
		})
		return
	}

	// Key 级别 allowed_models 校验(同 ProxyAnthropicGeneric 行为)
	if key != nil && !h.ensureModelAllowed(c, key, model) {
		return
	}

	// 模型名重写:用户侧别名 → 上游真实 model。
	// 用 rewriteModelField 保留原始 JSON 格式(防 OpenClaudeCode/PackyAPI 篡改检测)。
	upstreamModel := resolveModelUpstream(provider, model)
	if upstreamModel != "" && upstreamModel != model {
		bodyBytes = rewriteModelField(bodyBytes, upstreamModel)
		probe["model"] = upstreamModel // 同步给后续后台 worker 看的字段
	}

	// 上游 URL 防御(同 proxyAnthropicWithBody 行 461-467)
	if provider.APIURL == "" {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":    fmt.Sprintf("Provider %s 未配置 API URL", provider.Name),
			"provider": provider.Name,
			"code":     502,
		})
		return
	}
	if provider.APIKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置上游 API Key", "code": 502})
		return
	}
	upstreamURL := strings.TrimRight(provider.APIURL, "/") + "/v1/messages"

	// 创建 task 记录
	taskID := "oant_" + utils.GenerateHexKey(12)
	task := &models.AnthropicTask{
		ID:          taskID,
		APIKeyID:    firstAPIKeyID(key),
		Model:       model,
		Provider:    provider.Name,
		Status:      "pending",
		RequestBody: truncateString(string(bodyBytes), 50000),
		ClientIP:    c.ClientIP(),
	}
	if err := h.db.CreateAnthropicTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败", "code": 500})
		return
	}

	// logAPIRequest 记 "accepted" 一行,便于 admin 日志视图排查客户端发了啥
	h.logAPIRequest(key, model, "anthropic", endpoint, "chat", http.StatusAccepted, true, "", string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})

	c.JSON(http.StatusAccepted, gin.H{
		"task_id":    taskID,
		"model":      model,
		"provider":   provider.Name,
		"status":     "pending",
		"created_at": task.CreatedAt.Format(time.RFC3339),
		"poll_url":   endpoint + "/" + taskID,
		"message":    "Anthropic 异步任务已提交,请通过 GET " + endpoint + "/" + taskID + " 轮询结果",
	})

	// 后台跑上游调用。doRawRequestLong 是 5min timeout,足够 Ollama 27B 长推理。
	// bodyBytes 已是重写 model 后的最终上游 body(若不需要重写就是原 body)。
	go h.runAsyncAnthropicTask(taskID, provider, upstreamURL, bodyBytes, firstAPIKeyID(key))
}

// runAsyncAnthropicTask 后台 goroutine:跑上游 Anthropic 调用 + 回写 task 状态。
// 模式同 runAsyncLyricsGeneration(minimax_music.go:137),失败也回写 failed + error_message。
//
// 为什么不在前端异步 poll 时也复用 stripThinkingBlocks?非流式响应是单个 JSON,
// 上游 DeepSeek/Ollama 会在 content 数组里夹 type="thinking" 块。网关层统一剥离,
// 否则客户端下一轮回传 thinking 块会被标准 Anthropic SDK 判 400。
func (h *AIGatewayHandler) runAsyncAnthropicTask(taskID string, provider *config.AnthropicProviderConfig, upstreamURL string, bodyBytes []byte, apiKeyID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in runAsyncAnthropicTask: %v", r)
		}
	}()

	task, err := h.db.GetAnthropicTask(taskID)
	if err != nil {
		log.Printf("runAsyncAnthropicTask: task %s not found: %v", taskID, err)
		return
	}

	task.Status = "running"
	_ = h.db.UpdateAnthropicTask(task)

	start := time.Now()
	endpoint := "/api/anthropic/v1/messages/tasks"
	raw, err := h.doRawRequestLong(upstreamURL, provider.APIKey, "POST", bodyBytes, nil)

	now := time.Now()
	task.CompletedAt = &now

	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		_ = h.db.UpdateAnthropicTask(task)
		h.logAPIRequestByID(apiKeyID, task.Model, "anthropic", endpoint, "chat", http.StatusBadGateway, false, err.Error(), truncateString(string(bodyBytes), 10000), "", task.ClientIP, time.Since(start), usageSummary{})
		return
	}

	// DeepSeek / Ollama 等上游会夹带 type="thinking" 块,网关层剥离
	if providerEmitsThinkingBlocks(provider) {
		raw = stripThinkingBlocks(raw)
	}

	task.Status = "succeeded"
	task.ResultJSON = string(raw)
	_ = h.db.UpdateAnthropicTask(task)
	h.logAPIRequestByID(apiKeyID, task.Model, "anthropic", endpoint, "chat", http.StatusOK, true, "", truncateString(string(bodyBytes), 10000), truncateString(string(raw), 10000), task.ClientIP, time.Since(start), usageSummary{})
}

// GetAnthropicMessageTask GET /api/anthropic/v1/messages/tasks/:id
// 轮询异步任务状态。succeeded 时 result 字段是上游完整 Anthropic messages 响应对象。
func (h *AIGatewayHandler) GetAnthropicMessageTask(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "chat")
	if !ok {
		return
	}

	taskID := c.Param("id")
	task, err := h.db.GetAnthropicTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}

	// admin 提交时 APIKeyID 为空,可看所有 task;普通 key 只能看自己的
	if key != nil && task.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此任务", "code": 403})
		return
	}

	resp := gin.H{
		"task_id":    task.ID,
		"model":      task.Model,
		"provider":   task.Provider,
		"status":     task.Status,
		"created_at": task.CreatedAt.Format(time.RFC3339),
	}
	if task.CompletedAt != nil {
		resp["completed_at"] = task.CompletedAt.Format(time.RFC3339)
	}
	if task.Status == "succeeded" && task.ResultJSON != "" {
		// 上游 Anthropic messages 协议 JSON — 直接作为对象返回,客户端无需再 parse
		var r map[string]interface{}
		if err := json.Unmarshal([]byte(task.ResultJSON), &r); err == nil {
			resp["result"] = r
		} else {
			resp["result_raw"] = task.ResultJSON
		}
	}
	if task.Status == "failed" {
		resp["error"] = task.ErrorMessage
	}
	c.JSON(http.StatusOK, resp)
}
