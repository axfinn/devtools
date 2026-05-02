package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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

	respBody, _, err := h.doMediaRequestWithResp(upstreamURL, apiKey, "POST", upstreamBytes, c.Request.Header)
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
