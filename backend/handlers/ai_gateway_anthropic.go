package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *AIGatewayHandler) ProxyMinimaxAnthropic(c *gin.Context) {
	h.proxyAnthropic(c, "https://api.minimaxi.com/anthropic", h.cfg.MiniMax.APIKey, "/api/minimax/anthropic/v1/messages", []string{"MiniMax-M2.5", "MiniMax-M2.5-highspeed", "MiniMax-M2.1", "MiniMax-M2.1-highspeed", "MiniMax-M2", "MiniMax-M2.7"})
}

// ProxyDashScopeAnthropic 转发 Anthropic 协议格式的请求到 DashScope Anthropic 兼容端点
// POST /api/dashscope/anthropic/v1/messages

func (h *AIGatewayHandler) ProxyDashScopeAnthropic(c *gin.Context) {
	h.proxyAnthropic(c, "https://coding.dashscope.aliyuncs.com/apps/anthropic", h.cfg.DashScope.APIKey, "/api/dashscope/anthropic/v1/messages", []string{"qwen3.5-plus", "qwen3-max-2026-01-23", "qwen3-coder-next", "qwen3-coder-plus", "glm-5", "glm-4.7", "kimi-k2.5", "MiniMax-M2.5"})
}

// ProxyDeepSeekAnthropic 转发 Anthropic 协议格式的请求到 DeepSeek Anthropic 兼容端点
// POST /api/deepseek/anthropic/v1/messages
func (h *AIGatewayHandler) ProxyDeepSeekAnthropic(c *gin.Context) {
	h.proxyAnthropic(c, "https://api.deepseek.com/anthropic", h.cfg.DeepSeek.APIKey, "/api/deepseek/anthropic/v1/messages", []string{"deepseek-chat", "deepseek-reasoner", "deepseek-v4-flash", "deepseek-v4-pro"})
}

// ProxyMinimaxTTS 转发 TTS 请求到 MiniMax TTS 端点
// POST /api/minimax/tts/v1/generations

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
