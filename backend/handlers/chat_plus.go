package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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

// ollamaTagsResponse /api/tags 响应结构（只取需要的字段）。
type ollamaTagsResponse struct {
	Models []struct {
		Name       string `json:"name"`
		Size       int64  `json:"size"`
		Details    struct {
			ParameterSize  string `json:"parameter_size"`
			QuantizationLevel string `json:"quantization_level"`
			Family         string `json:"family"`
		} `json:"details"`
	} `json:"models"`
}

// fetchOllamaModels 拉取局域网 Ollama 的模型列表（短超时，不缓存——局域网服务可能随时增删模型）。
func (h *AIGatewayHandler) fetchOllamaModels(baseURL string) ([]gin.H, error) {
	url := strings.TrimRight(baseURL, "/") + "/api/tags"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("ollama 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	models := make([]gin.H, 0, len(tags.Models))
	for _, m := range tags.Models {
		models = append(models, gin.H{
			"id":            "ollama:" + m.Name,
			"name":          m.Name,
			"parameter_size": m.Details.ParameterSize,
			"family":        m.Details.Family,
			"size_gb":       float64(m.Size) / (1024 * 1024 * 1024),
		})
	}
	return models, nil
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
	model := strings.TrimSpace(req.Model)
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model"})
		return
	}

	// Ollama 本地引擎：model 形如 "ollama:qwen3:14b"，
	// 转发到局域网 Ollama 的 OpenAI 兼容端点，SSE 透传（复用心跳/重试逻辑）。
	if after, ok := strings.CutPrefix(model, "ollama:"); ok {
		if after == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 Ollama 模型名"})
			return
		}
		base := fallbackString(h.cfg.ChatPlus.OllamaBaseURL, "http://192.168.31.147:11434")
		req.Model = after
		h.streamOpenAICompatible(c, req, strings.TrimRight(base, "/")+"/v1/chat/completions", "ollama")
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
