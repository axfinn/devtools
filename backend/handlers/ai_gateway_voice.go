package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

func (h *AIGatewayHandler) UploadVoiceClone(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	// 获取 API Key 的 ID（超级管理员为空）
	apiKeyID := ""
	if key != nil {
		apiKeyID = key.ID
	}

	// 解析 multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB limit
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析表单失败", "code": 400})
		return
	}

	// 获取音色名称
	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "音色名称不能为空", "code": 400})
		return
	}

	// 获取音频文件
	file, header, err := c.Request.FormFile("audio_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传音频文件", "code": 400})
		return
	}
	defer file.Close()

	// 检查文件大小（限制 10MB）
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "音频文件大小不能超过 10MB", "code": 400})
		return
	}

	// 检查文件类型
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "audio/") && !strings.Contains(contentType, "octet-stream") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传音频文件", "code": 400})
		return
	}

	// 读取文件内容
	audioData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取音频文件失败", "code": 500})
		return
	}

	// 获取 API Key
	apiKey := h.cfg.MiniMaxVoiceCloning.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey
	}
	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax API Key", "code": 502})
		return
	}

	baseURL := h.cfg.MiniMaxVoiceCloning.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	// 调用 MiniMax Voice Cloning API
	// POST /v1/voice_cloning/upload_clone_audio
	upstreamURL := strings.TrimRight(baseURL, "/") + "/v1/voice_cloning/upload_clone_audio"

	// 构建 multipart form 请求
	body := &bytes.Buffer{}
	writer := &multipart.Writer{}
	writer = multipart.NewWriter(body)

	// 添加 audio_file 字段
	part, err := writer.CreateFormFile("audio_file", header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建表单文件失败", "code": 500})
		return
	}
	if _, err := part.Write(audioData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入音频数据失败", "code": 500})
		return
	}

	if err := writer.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "关闭表单写入器失败", "code": 500})
		return
	}

	req, err := http.NewRequest("POST", upstreamURL, bytes.NewReader(body.Bytes()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建请求失败", "code": 500})
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	start := time.Now()
	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/upload", "media", http.StatusBadGateway, false, err.Error(), fmt.Sprintf("name=%s, size=%d", name, len(audioData)), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("上传音色失败: %s", err.Error()), "code": 502})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/upload", "media", resp.StatusCode, resp.StatusCode < 400, string(respBody), fmt.Sprintf("name=%s, size=%d", name, len(audioData)), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("解析响应失败: %s", string(respBody)), "code": 502})
		return
	}

	// 检查 API 错误
	if baseResp, ok := result["base_resp"].(map[string]interface{}); ok {
		if code, ok := baseResp["status_code"].(float64); ok && int(code) != 0 {
			msg, _ := baseResp["status_msg"].(string)
			h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/upload", "media", resp.StatusCode, false, msg, fmt.Sprintf("name=%s, size=%d", name, len(audioData)), string(respBody), c.ClientIP(), time.Since(start), usageSummary{})
			c.JSON(http.StatusBadGateway, gin.H{"error": msg, "code": 502})
			return
		}
	}

	// 提取 voice_id
	voiceID := ""
	if data, ok := result["data"].(map[string]interface{}); ok {
		if vid, ok := data["voice_id"].(string); ok {
			voiceID = vid
		}
	}
	if voiceID == "" {
		h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/upload", "media", resp.StatusCode, false, "未获取到 voice_id", fmt.Sprintf("name=%s, size=%d", name, len(audioData)), string(respBody), c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": "未获取到 voice_id", "code": 502})
		return
	}

	// 保存到数据库
	clone := &models.VoiceClone{
		APIKeyID: apiKeyID,
		VoiceID:  voiceID,
		Name:     name,
		Status:   "active",
	}
	if err := h.db.CreateVoiceClone(clone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存音色记录失败", "code": 500})
		return
	}

	h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/upload", "media", http.StatusOK, true, "", fmt.Sprintf("name=%s, size=%d, voice_id=%s", name, len(audioData), voiceID), string(respBody), c.ClientIP(), time.Since(start), usageSummary{})

	c.JSON(http.StatusOK, gin.H{
		"voice_id": voiceID,
		"name":     name,
		"status":   "active",
		"message":  "音色创建成功",
	})
}

// ListVoiceClones 获取音色列表
// GET /api/minimax/voice-cloning/voices

func (h *AIGatewayHandler) ListVoiceClones(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	apiKeyID := ""
	if key != nil {
		apiKeyID = key.ID
	}
	limit := boundedInt(c.Query("limit"), 20, 1, 100)
	offset := boundedInt(c.Query("offset"), 0, 0, 100000)

	clones, err := h.db.ListVoiceClones(apiKeyID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取音色列表失败", "code": 500})
		return
	}

	// 调用 MiniMax API 获取最新音色状态
	apiKey := h.cfg.MiniMaxVoiceCloning.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey
	}
	baseURL := h.cfg.MiniMaxVoiceCloning.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	// 尝试调用 MiniMax 获取音色列表
	upstreamURL := strings.TrimRight(baseURL, "/") + "/v1/voice_cloning/voice_list"
	req, err := http.NewRequest("GET", upstreamURL, nil)
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		resp, err := h.noProxyClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var upstreamResult map[string]interface{}
				if body, err := io.ReadAll(resp.Body); err == nil {
					json.Unmarshal(body, &upstreamResult)
					// 合并上游音色状态
					if data, ok := upstreamResult["data"].(map[string]interface{}); ok {
						if voices, ok := data["voice_list"].([]interface{}); ok {
							// 创建 voice_id -> status 映射
							statusMap := make(map[string]string)
							for _, v := range voices {
								if voice, ok := v.(map[string]interface{}); ok {
									if vid, ok := voice["voice_id"].(string); ok {
										statusMap[vid] = "active"
									}
								}
							}
							// 更新本地音色状态
							for _, clone := range clones {
								if s, exists := statusMap[clone.VoiceID]; exists {
									clone.Status = s
								}
							}
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"voices": clones,
		"total":  len(clones),
	})
}

// DeleteVoiceClone 删除音色
// DELETE /api/minimax/voice-cloning/voices/:id

func (h *AIGatewayHandler) DeleteVoiceClone(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	apiKeyID := ""
	if key != nil {
		apiKeyID = key.ID
	}

	// 解析 ID
	idStr := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的音色 ID", "code": 400})
		return
	}

	// 获取音色记录
	clone, err := h.db.GetVoiceClone(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "音色不存在", "code": 404})
		return
	}

	// 检查权限：超级管理员可以删除任何音色，普通用户只能删除自己创建的音色
	if key != nil && clone.APIKeyID != apiKeyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除该音色", "code": 403})
		return
	}

	// 调用 MiniMax API 删除音色
	apiKey := h.cfg.MiniMaxVoiceCloning.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey
	}
	baseURL := h.cfg.MiniMaxVoiceCloning.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	upstreamURL := strings.TrimRight(baseURL, "/") + "/v1/voice_cloning/delete_voice"
	delReq, _ := http.NewRequest("DELETE", upstreamURL, nil)
	delReq.Header.Set("Authorization", "Bearer "+apiKey)
	delReq.Header.Set("Content-Type", "application/json")

	body, _ := json.Marshal(map[string]string{"voice_id": clone.VoiceID})
	delReq.Body = io.NopCloser(bytes.NewReader(body))

	start := time.Now()
	resp, err := h.noProxyClient.Do(delReq)
	if err != nil {
		// 即使上游调用失败，也删除本地记录
		h.db.DeleteVoiceClone(id, apiKeyID)
		c.JSON(http.StatusOK, gin.H{"message": "音色已删除（本地上游同步失败）"})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 删除本地记录
	h.db.DeleteVoiceClone(id, apiKeyID)

	h.logAPIRequest(key, "voice-cloning", "minimax", "/api/minimax/voice-cloning/voices/"+idStr, "media", http.StatusOK, true, "", fmt.Sprintf("voice_id=%s", clone.VoiceID), string(respBody), c.ClientIP(), time.Since(start), usageSummary{})

	c.JSON(http.StatusOK, gin.H{"message": "音色已删除"})
}

// TTSWithVoiceClone 使用自定义音色进行 TTS
// POST /api/minimax/voice-cloning/tts

func (h *AIGatewayHandler) TTSWithVoiceClone(c *gin.Context) {
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

	// 校验模型是否在允许列表中
	if !isModelAllowed(model, TTSAllowedModels) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("该端点不支持模型 %s，允许的模型: %v", model, TTSAllowedModels)})
		return
	}

	// 超级管理员跳过 API Key 模型权限检查
	if key != nil {
		if !h.ensureModelAllowed(c, key, model) {
			return
		}
	}

	// 获取 voice_id
	voiceID, _ := bodyMap["voice_id"].(string)
	if voiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 voice_id 字段"})
		return
	}

	// 验证 voice_id 是否属于当前用户（超级管理员可以使用任何音色）
	clone, err := h.db.GetVoiceCloneByVoiceID(voiceID)
	if err != nil || (key != nil && clone.APIKeyID != key.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权使用该音色", "code": 403})
		return
	}

	apiKey := h.cfg.MiniMaxVoiceCloning.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey
	}
	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax API Key"})
		return
	}

	baseURL := h.cfg.MiniMaxVoiceCloning.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	// 转换请求格式：MiniMax TTS API 要求 voice_setting.voice_id 格式
	upstreamReq := make(map[string]interface{})
	upstreamReq["model"] = model
	if text, ok := bodyMap["text"].(string); ok {
		upstreamReq["text"] = text
	}
	// 使用自定义 voice_id
	if upstreamReq["voice_setting"] == nil {
		upstreamReq["voice_setting"] = map[string]interface{}{}
	}
	if vs, ok := upstreamReq["voice_setting"].(map[string]interface{}); ok {
		vs["voice_id"] = voiceID
	}
	if speed, ok := bodyMap["speed"]; ok {
		if vs, ok := upstreamReq["voice_setting"].(map[string]interface{}); ok {
			vs["speed"] = speed
		}
	}
	// 透传 audio_format
	if af, ok := bodyMap["audio_format"].(string); ok && af != "" {
		upstreamReq["audio_setting"] = map[string]interface{}{"audio_format": af}
	}
	// 其他字段直接透传
	for k, v := range bodyMap {
		if k != "voice_id" && k != "speed" && k != "audio_format" {
			upstreamReq[k] = v
		}
	}

	upstreamBytes, _ := json.Marshal(upstreamReq)

	start := time.Now()
	upstreamURL := strings.TrimRight(baseURL, "/") + "/v1/t2a_v2"

	respBody, _, err := h.doMediaRequestWithResp(upstreamURL, apiKey, "POST", upstreamBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// 计算使用量（按字符数计费）
	textLen := len(interfaceToString(bodyMap["text"]))
	usage := usageSummary{Cost: float64(textLen) * 0.001, Currency: "CNY"}

	// 解析响应
	var respData map[string]interface{}
	if err := json.Unmarshal(respBody, &respData); err != nil {
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusBadGateway, false, "上游响应解析失败: "+err.Error(), string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
		c.JSON(http.StatusBadGateway, gin.H{"error": "上游响应解析失败"})
		return
	}

	// 检查业务错误
	if baseResp, ok := respData["base_resp"].(map[string]interface{}); ok {
		if code, ok := baseResp["status_code"].(float64); ok && int(code) != 0 {
			msg, _ := baseResp["status_msg"].(string)
			h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusBadGateway, false, msg, string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
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
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusBadGateway, false, "未获取到音频数据", string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
		c.JSON(http.StatusBadGateway, gin.H{"error": "未获取到音频数据"})
		return
	}

	// 解码 base64 音频
	audioBytes, err := base64.StdEncoding.DecodeString(audioData)
	if err != nil {
		h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusBadGateway, false, "音频base64解码失败: "+err.Error(), string(bodyBytes), string(respBody), c.ClientIP(), time.Since(start), usage)
		c.JSON(http.StatusBadGateway, gin.H{"error": "音频数据解码失败"})
		return
	}

	h.logAPIRequest(key, model, "minimax-tts", "/api/minimax/voice-cloning/tts", "media", http.StatusOK, true, "", string(bodyBytes), "[base64 audio]", c.ClientIP(), time.Since(start), usage)

	// 返回音频二进制
	c.DataFromReader(http.StatusOK, int64(len(audioBytes)), "audio/mpeg", bytes.NewReader(audioBytes), nil)
}

// GetVoiceCloningDocs 返回 Voice Cloning API 文档
// GET /api/minimax/voice-cloning/docs

func (h *AIGatewayHandler) GetVoiceCloningDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "MiniMax Voice Cloning 接口文档",
		"summary": "通过 AI Gateway 调用 MiniMax Voice Cloning 音色克隆与 TTS 合成功能。",
		"auth": gin.H{
			"api_key": "Authorization: Bearer dtk_ai_xxx",
			"scope":   "media",
		},
		"base_url": "/api/minimax/voice-cloning",
		"upstream": "https://api.minimaxi.com/v1/voice_cloning",
		"routes": []gin.H{
			{"method": "GET", "path": "/api/minimax/voice-cloning/docs", "description": "获取本文档"},
			{"method": "POST", "path": "/api/minimax/voice-cloning/upload", "description": "上传音频复刻音色"},
			{"method": "GET", "path": "/api/minimax/voice-cloning/voices", "description": "获取音色列表"},
			{"method": "DELETE", "path": "/api/minimax/voice-cloning/voices/:id", "description": "删除音色"},
			{"method": "POST", "path": "/api/minimax/voice-cloning/tts", "description": "使用自定义音色 TTS"},
		},
		"examples": gin.H{
			"upload_request": gin.H{
				"description": "上传音频复刻音色 (multipart/form-data)",
				"fields": gin.H{
					"name":       "音色名称，如'我的音色'",
					"audio_file": "音频文件（支持 wav/mp3/m4a，最大 10MB）",
				},
			},
			"tts_request": gin.H{
				"model":        "speech-2.8-hd",
				"text":         "你好，这是使用自定义音色的语音合成",
				"voice_id":     "clone_xxx",
				"speed":        1.0,
				"audio_format": "mp3",
			},
			"curl_upload": gin.H{
				"language": "cURL",
				"code": `curl -X POST https://your-devtools:8080/api/minimax/voice-cloning/upload \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -F "name=我的音色" \
  -F "audio_file=@voice.wav"`,
			},
			"curl_tts": gin.H{
				"language": "cURL",
				"code": `curl -X POST https://your-devtools:8080/api/minimax/voice-cloning/tts \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "text": "你好，这是使用自定义音色的语音合成",
    "voice_id": "clone_xxx",
    "speed": 1.0
  }' \
  --output output.mp3`,
			},
		},
		"voice_ids_note": "自定义音色通过 upload 接口创建，voice_id 以 clone_ 开头",
	})
}
