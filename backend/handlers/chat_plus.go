package handlers

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 本文件：新版 AI 聊天页（chat-plus，Open WebUI 风格）的后端。
//
// 设计要点：
//   - 页面密码存服务端 config.yaml 的 chat_plus.password，前端只做校验；
//     密码为空 = 模块整体禁用（fail-closed），与其他模块惯例一致。
//   - 双引擎：
//       1) 局域网 Ollama（chat_plus.ollama_base_url，OpenAI 兼容 /v1/chat/completions），
//          模型列表运行时从 /api/tags 动态拉取，用户自选；
//       2) MiniMax（复用 AI Gateway 的 cfg.MiniMax.APIKey，key 全程不出服务端），
//          走已有的 streamMiniMaxChat（Anthropic 兼容端点 → OpenAI SSE 转换）。
//   - 流式转发直接复用 streamOpenAICompatible / streamMiniMaxChat（含心跳与 sseWriter）。

// chatPlusMinimaxModels 聊天页暴露的 MiniMax 模型（与 resolveChatProvider 保持一致）。
var chatPlusMinimaxModels = []gin.H{
	{"id": "MiniMax-M3", "label": "MiniMax-M3", "description": "旗舰推理模型"},
	{"id": "MiniMax-M2.7", "label": "MiniMax-M2.7", "description": "新一代通用模型"},
	{"id": "MiniMax-M2.5", "label": "MiniMax-M2.5", "description": "均衡型模型"},
}

// checkChatPlusPassword 校验聊天页访问密码。
// models/stream 走 header X-Chat-Password；verify 走 body（由 handler 自行处理）。
func (h *AIGatewayHandler) checkChatPlusPassword(c *gin.Context) bool {
	pwd := strings.TrimSpace(h.cfg.ChatPlus.Password)
	if pwd == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "chat_plus 未配置访问密码，功能已禁用"})
		return false
	}
	given := c.GetHeader("X-Chat-Password")
	if given == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少访问密码"})
		return false
	}
	if subtle.ConstantTimeCompare([]byte(given), []byte(pwd)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return false
	}
	return true
}

// ChatPlusVerify POST /api/chat-plus/verify  —— 聊天页密码门校验
func (h *AIGatewayHandler) ChatPlusVerify(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	pwd := strings.TrimSpace(h.cfg.ChatPlus.Password)
	if pwd == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "chat_plus 未配置访问密码，功能已禁用"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(pwd)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ollamaTagModel /api/tags 单条模型（只取需要的字段）。
type ollamaTagModel struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Details struct {
		ParameterSize     string `json:"parameter_size"`
		QuantizationLevel string `json:"quantization_level"`
		Family            string `json:"family"`
	} `json:"details"`
}

// ollamaTagsResponse /api/tags 响应结构。
type ollamaTagsResponse struct {
	Models []ollamaTagModel `json:"models"`
}

// ollamaProbeTransport / 两个探活客户端：局域网（Wi-Fi）同样会抖动，
// 每次新建 http.Client 意味着每次都要重新 TCP 握手，弱网下更容易超时。
// 这里共用带连接池的 transport，并把原先 5s/4s 的硬超时放宽到能容忍一次丢包重传。
var ollamaProbeTransport = &http.Transport{
	Proxy:               nil,
	DialContext:         (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 15 * time.Second}).DialContext,
	ForceAttemptHTTP2:   false,
	MaxIdleConns:        32,
	MaxIdleConnsPerHost: 16,
	IdleConnTimeout:     45 * time.Second,
	DisableCompression:  true,
}

var (
	ollamaListClient   = &http.Client{Timeout: 15 * time.Second, Transport: ollamaProbeTransport}
	ollamaVisionClient = &http.Client{Timeout: 10 * time.Second, Transport: ollamaProbeTransport}
)

// fetchOllamaModels 拉取局域网 Ollama 的模型列表（不缓存——局域网服务可能随时增删模型）。
// 弱网加固：15s 超时 + 瞬时错误重试（复用 outboundPolicy 的退避策略）。
func (h *AIGatewayHandler) fetchOllamaModels(baseURL string) ([]gin.H, error) {
	url := strings.TrimRight(baseURL, "/") + "/api/tags"
	var tags ollamaTagsResponse
	err := doWithRetry(func() error {
		resp, err := ollamaListClient.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			return fmt.Errorf("ollama 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		tags = ollamaTagsResponse{}
		return json.NewDecoder(resp.Body).Decode(&tags)
	})
	if err != nil {
		return nil, err
	}
	models := make([]gin.H, len(tags.Models))
	var wg sync.WaitGroup
	for i, m := range tags.Models {
		wg.Add(1)
		go func(i int, m ollamaTagModel) {
			defer wg.Done()
			models[i] = gin.H{
				"id":             "ollama:" + m.Name,
				"name":           m.Name,
				"parameter_size": m.Details.ParameterSize,
				"family":         m.Details.Family,
				"size_gb":        float64(m.Size) / (1024 * 1024 * 1024),
				"vision":         h.ollamaModelVision(baseURL, m.Name),
			}
		}(i, m)
	}
	wg.Wait()
	return models, nil
}

// ollamaNameVisionKeywords 已知视觉/多模态模型的名称关键词（启发式兜底，避免 /api/show 探测失败）。
var ollamaNameVisionKeywords = []string{
	"llava", "bakllava", "moondream", "minicpm-v", "minicpm-o", "nanollava",
	"qwen-vl", "qwen2-vl", "qwen2.5-vl", "qwen3-vl", "qwen3.5-vl", "qwen-image",
	"llama3.1-vision", "llama3.2-vision", "llama4", "pixtral", "gemma3", "gemma-3",
	"glm-4v", "internvl", "vision", "joyvasam", "xgen-mm", "deepseek-vl", "janus",
	"phi-3-vision", "phi-4-multimodal", "phi-4-v", "mistral-small3.1", "granite-vision",
	"smolvlm", "molmo", "eagle", "minicpm", "cogvlm", "cogagent", "yi-vl", "omnilmm",
	"ovis", "mmproj", "awq-vision", "fuyu", "kosmos",
}

// ollamaNameLooksVision 仅按模型名启发式判断是否支持图片（兜底用）。
func ollamaNameLooksVision(name string) bool {
	lower := strings.ToLower(name)
	for _, k := range ollamaNameVisionKeywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

// ollamaModelVision 探测模型是否支持图片输入。
// 优先用 Ollama /api/show 的 model_info.vision 字段（最可靠），探测失败则回退到名称启发式。
func (h *AIGatewayHandler) ollamaModelVision(baseURL, name string) bool {
	url := strings.TrimRight(baseURL, "/") + "/api/show"
	body, _ := json.Marshal(map[string]string{"name": name})
	resp, err := ollamaVisionClient.Post(url, "application/json", bytes.NewReader(body))
	if err == nil {
		defer resp.Body.Close()
		var out struct {
			Capabilities  []string               `json:"capabilities"`
			ModelInfo     map[string]interface{} `json:"model_info"`
			ProjectorInfo map[string]interface{} `json:"projector_info"`
		}
		if json.NewDecoder(resp.Body).Decode(&out) == nil {
			// 1) 首选（最权威）：Ollama 0.6+ 的 capabilities 数组，含 "vision" 即支持图片。
			//    注意：capabilities 非空时不含 vision = 明确不支持，不要再被名称启发式误判。
			if len(out.Capabilities) > 0 {
				for _, cap := range out.Capabilities {
					if strings.EqualFold(strings.TrimSpace(cap), "vision") {
						return true
					}
				}
				return false
			}
			// 2) 老版本 Ollama 无 capabilities：有 projector_info（多模态投影器）即支持。
			if len(out.ProjectorInfo) > 0 {
				return true
			}
			// 3) 兜底：model_info 里若有显式 vision 标记。
			if v, ok := out.ModelInfo["vision"]; ok {
				if b, ok2 := v.(bool); ok2 {
					return b
				}
			}
		}
	}
	return ollamaNameLooksVision(name)
}

// ChatPlusModels GET /api/chat-plus/models —— 聚合双引擎可用模型列表
func (h *AIGatewayHandler) ChatPlusModels(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}

	// MiniMax 引擎可用性取决于服务端是否配置了 api_key（key 不下发前端）
	minimaxAvailable := strings.TrimSpace(h.cfg.MiniMax.APIKey) != ""

	// Ollama 引擎：动态拉取，失败不阻塞接口（局域网主机可能离线）
	ollamaAvailable := false
	ollamaModels := []gin.H{}
	ollamaError := ""
	if strings.TrimSpace(h.cfg.ChatPlus.OllamaBaseURL) != "" {
		models, err := h.fetchOllamaModels(h.cfg.ChatPlus.OllamaBaseURL)
		if err != nil {
			ollamaError = err.Error()
			log.Printf("[chat-plus] 拉取 Ollama 模型列表失败: %v", err)
		} else {
			ollamaAvailable = true
			ollamaModels = models
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"minimax": gin.H{
			"available": minimaxAvailable,
			"models":    chatPlusMinimaxModels,
		},
		"ollama": gin.H{
			"available": ollamaAvailable,
			"base_url":  h.cfg.ChatPlus.OllamaBaseURL,
			"models":    ollamaModels,
			"error":     ollamaError,
		},
	})
}

// ChatPlusStream POST /api/chat-plus/chat/stream —— 双引擎流式聊天（SSE，OpenAI chunk 格式）
func (h *AIGatewayHandler) ChatPlusStream(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	req.Model = strings.TrimSpace(req.Model)
	if err := h.validateChatPlusModel(req.Model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.runChatPlusStream(c, req)
}

// validateChatPlusModel 校验模型属于当前可用引擎。同步流与异步任务创建共用，
// 保证"建任务时就报错"，而不是任务建好了才在后台失败。
func (h *AIGatewayHandler) validateChatPlusModel(model string) error {
	if model == "" {
		return errors.New("缺少 model")
	}
	if after, ok := strings.CutPrefix(model, "ollama:"); ok {
		if strings.TrimSpace(after) == "" {
			return errors.New("缺少 Ollama 模型名")
		}
		if h.resolveOllamaProviderConfig() == nil {
			return errors.New("未配置 Ollama 下游（chat_plus.ollama_base_url）")
		}
		return nil
	}
	switch model {
	case fallbackString(h.cfg.MiniMax.Model, "abab6.5s-chat"), "MiniMax-M3", "MiniMax-M2.7", "MiniMax-M2.5":
		return nil
	}
	return fmt.Errorf("不支持的模型: %s", model)
}

// runChatPlusStream 按模型路由到上游并把 SSE 写进 c。
// 同步接口直接写客户端连接；异步任务则把 c 换成"写进任务缓冲"的 sink，
// 这样上游产出的每个片段既进了内存缓冲（供续传）也会被累积成最终正文（落库）。
func (h *AIGatewayHandler) runChatPlusStream(c *gin.Context, req ChatCompletionRequest) {
	model := strings.TrimSpace(req.Model)

	// Ollama 本地引擎：model 形如 "ollama:qwen3:14b"，
	// 转发到局域网 Ollama 的 OpenAI 兼容端点，SSE 透传（复用心跳/重试逻辑）。
	// 用按 provider 派生的流式客户端（header_timeout_sec 兜底 600s）：
	// 本地 27B 模型首字节常超 60s，全局 streamClient 会把请求掐死。
	if after, ok := strings.CutPrefix(model, "ollama:"); ok {
		if after == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 Ollama 模型名"})
			return
		}
		prov := h.resolveOllamaProviderConfig()
		if prov == nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 Ollama 下游（chat_plus.ollama_base_url）"})
			return
		}
		req.Model = after
		h.streamOpenAICompatibleWithClient(c, req, strings.TrimRight(prov.APIURL, "/")+"/v1/chat/completions", "ollama", h.pickStreamClient(prov))
		return
	}

	// MiniMax 云端引擎：key 在服务端，走已有 Anthropic 兼容流式通道
	switch model {
	case fallbackString(h.cfg.MiniMax.Model, "abab6.5s-chat"), "MiniMax-M3", "MiniMax-M2.7", "MiniMax-M2.5":
		h.streamMiniMaxChat(c, req)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的模型: " + model})
	}
}
