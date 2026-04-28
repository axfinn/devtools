package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// modelFieldRe 匹配 JSON body 中 "model": "..." 字段
// 用于保留原始 JSON 格式替换 model 名，避免第三方代理检测到 body 被篡改
var modelFieldRe = regexp.MustCompile(`"model"\s*:\s*"[^"]*"`)


// builtinAnthropicProviders 返回默认内置的 Anthropic 提供商列表
// 当 config 中未配置 anthropic_providers 时使用
func (h *AIGatewayHandler) builtinAnthropicProviders() []config.AnthropicProviderConfig {
	return []config.AnthropicProviderConfig{
		{
			Name:   "MiniMax",
			APIURL: "https://api.minimaxi.com/anthropic",
			APIKey: h.cfg.MiniMax.APIKey,
			Models: []string{"MiniMax-M2.5", "MiniMax-M2.1", "MiniMax-M2", "MiniMax-M2.7"},
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
func (h *AIGatewayHandler) dbAnthropicProvidersToConfig(dbProviders []*models.AnthropicProvider) []config.AnthropicProviderConfig {
	result := make([]config.AnthropicProviderConfig, 0, len(dbProviders))
	for _, p := range dbProviders {
		cfg := config.AnthropicProviderConfig{
			Name:         p.Name,
			APIURL:       p.APIURL,
			APIKey:       p.APIKey,
			DefaultModel: p.DefaultModel,
			IsDefault:    p.IsDefault,
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
	p := h.resolveProviderByNameOrModel("MiniMax", "MiniMax-M2.5")
	h.proxyAnthropic(c, p, "/api/minimax/anthropic/v1/messages", []string{"MiniMax-M2.5", "MiniMax-M2.1", "MiniMax-M2", "MiniMax-M2.7"})
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
	h.proxyAnthropicWithBody(c, provider, "/api/anthropic/v1/messages", allModels, model, upstreamModel, bodyBytes, bodyMap, authKey)
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
		h.logAPIRequest(key, model, "anthropic", logPath, "chat", statusCode, streamErr == nil, safeError(streamErr), string(bodyBytes), "[stream]", c.ClientIP(), time.Since(start), usageSummary{})
		return
	}

	raw, err := h.doRawRequest(upstreamURL, provider.APIKey, "POST", bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "anthropic", logPath, "chat", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	h.logAPIRequest(key, model, "anthropic", logPath, "chat", http.StatusOK, true, "", string(bodyBytes), string(raw), c.ClientIP(), time.Since(start), usageSummary{})

	c.Data(http.StatusOK, "application/json", raw)
}

// proxyAnthropicStream 流式代理 Anthropic 请求（SSE 透传）
// 对于 DeepSeek，过滤掉 type="thinking" 的内容块事件
func (h *AIGatewayHandler) proxyAnthropicStream(c *gin.Context, provider *config.AnthropicProviderConfig, upstreamURL string, bodyBytes []byte) (int, error) {
	// 使用 c.Request.Context()：客户端断开时自动取消 upstream 请求
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", upstreamURL, bytes.NewReader(bodyBytes))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return http.StatusBadGateway, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	// 透传客户端的 Anthropic 相关头
	for key, values := range c.Request.Header {
		ck := http.CanonicalHeaderKey(key)
		switch ck {
		case "Content-Type", "Authorization", "Content-Length", "Host",
			"Connection", "Transfer-Encoding", "Te", "Trailer",
			"Keep-Alive", "Proxy-Connection", "Upgrade":
			continue
		}
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}

	// 使用 streamClient（无超时），避免 http.Client.Timeout 在长流时强制断开
	resp, err := h.streamClient.Do(req)
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

	// 透传上游响应头
	for key, values := range resp.Header {
		for _, v := range values {
			c.Writer.Header().Add(key, v)
		}
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("streaming not supported")
	}

	buf := make([]byte, 1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
				return http.StatusOK, writeErr
			}
			flusher.Flush()
		}
		if readErr != nil {
			if readErr != io.EOF {
				return http.StatusOK, readErr
			}
			break
		}
	}
	return http.StatusOK, nil
}

// sseThinkingFilter 过滤 DeepSeek SSE 流中的 type="thinking" 内容块
// 以 SSE event 为单位（空行分隔），若事件属于 thinking 块则整体丢弃
type sseThinkingFilter struct {
	src         io.Reader
	scanner     *bufio.Scanner
	skipIndices map[int]bool  // 被标记为 thinking 的 content_block index
	pending     []byte         // 已过滤、待输出的数据
}

func newSSEThinkingFilter(src io.Reader) io.Reader {
	f := &sseThinkingFilter{
		src:         src,
		scanner:     bufio.NewScanner(src),
		skipIndices: make(map[int]bool),
	}
	return f
}

func (f *sseThinkingFilter) Read(p []byte) (int, error) {
	// 如果还有待输出数据，直接返回
	if len(f.pending) > 0 {
		n := copy(p, f.pending)
		f.pending = f.pending[n:]
		return n, nil
	}

	if !f.scanner.Scan() {
		return 0, io.EOF
	}

	// 累积一个完整 SSE event（遇空行为止）
	var lines []string
	line := f.scanner.Text()
	lines = append(lines, line)

	// 收集当前 event 的所有行
	for f.scanner.Scan() {
		next := f.scanner.Text()
		if next == "" {
			// 空行 = event 结束
			lines = append(lines, "")
			break
		}
		lines = append(lines, next)
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

func isModelAllowed(model string, allowed []string) bool {
	for _, m := range allowed {
		if m == model {
			return true
		}
	}
	return false
}
