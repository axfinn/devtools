package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

const defaultMiniMaxSpeechBaseURL = "https://api.minimaxi.com"

func (h *AIGatewayHandler) GetMiniMaxSpeechDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "MiniMax Speech Gateway 文档",
		"summary": "复用 DevTools AI Gateway API Key 鉴权，接入 MiniMax 官方语音 HTTP 能力：同步 TTS、异步长文本 TTS、音色管理、音色设计、音色复刻、文件管理。",
		"auth": gin.H{
			"api_key":      "Authorization: Bearer dtk_ai_xxx",
			"admin_header": "X-Super-Admin-Password",
			"scope":        "media",
		},
		"base_url": "/api/minimax/speech",
		"upstream": defaultMiniMaxSpeechBaseURL,
		"models":   TTSAllowedModels,
		"routes": []gin.H{
			{"method": "GET", "path": "/api/minimax/speech/docs", "description": "获取语音网关文档"},
			{"method": "POST", "path": "/api/minimax/speech/v1/t2a_v2", "description": "同步文本转语音（官方 HTTP）"},
			{"method": "POST", "path": "/api/minimax/speech/v1/t2a_async_v2", "description": "异步长文本转语音"},
			{"method": "GET", "path": "/api/minimax/speech/v1/query/t2a_async_query_v2", "description": "查询异步长文本语音任务"},
			{"method": "POST", "path": "/api/minimax/speech/v1/get_voice", "description": "查询音色列表或单个音色"},
			{"method": "POST", "path": "/api/minimax/speech/v1/voice_design", "description": "音色设计"},
			{"method": "POST", "path": "/api/minimax/speech/v1/voice_clone", "description": "音色复刻"},
			{"method": "POST", "path": "/api/minimax/speech/v1/delete_voice", "description": "删除自定义音色"},
			{"method": "POST", "path": "/api/minimax/speech/v1/files/upload", "description": "上传语音相关文件"},
			{"method": "GET", "path": "/api/minimax/speech/v1/files/list", "description": "列出当前 API Key 通过网关创建的文件"},
			{"method": "GET", "path": "/api/minimax/speech/v1/files/retrieve", "description": "获取文件元信息"},
			{"method": "GET", "path": "/api/minimax/speech/v1/files/retrieve_content", "description": "下载文件内容"},
			{"method": "POST", "path": "/api/minimax/speech/v1/files/delete", "description": "删除文件"},
			{"method": "GET", "path": "/api/minimax/speech/tasks", "description": "DevTools 扩展：列出当前 API Key 的异步语音任务"},
			{"method": "GET", "path": "/api/minimax/speech/tasks/:id", "description": "DevTools 扩展：查看单个异步语音任务"},
		},
		"examples": gin.H{
			"sync_tts": gin.H{
				"model": "speech-2.8-hd",
				"text":  "你好，这是通过 DevTools AI Gateway 发起的 MiniMax 语音合成测试。",
				"voice_setting": gin.H{
					"voice_id": "male-qn-qingse",
					"speed":    1.0,
				},
				"audio_setting": gin.H{
					"sample_rate": 32000,
					"format":      "mp3",
				},
			},
			"async_tts": gin.H{
				"model":        "speech-2.8-hd",
				"text_file_id": "file_xxx",
				"voice_id":     "male-qn-qingse",
			},
			"voice_design": gin.H{
				"voice_id":     "custom-designer-001",
				"prompt":       "一位温柔、专业、清晰的中文女声，适合知识讲解和客服播报。",
				"preview_text": "你好，欢迎使用 DevTools 语音网关。",
			},
			"voice_clone": gin.H{
				"voice_id":             "custom-clone-001",
				"file_id":              "file_xxx",
				"need_noise_reduction": true,
			},
			"file_upload": gin.H{
				"purpose": "voice_clone / prompt_audio / t2a_async_input",
				"field":   "file",
			},
		},
	})
}

func (h *AIGatewayHandler) MiniMaxSpeechSyncT2A(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	bodyBytes, bodyMap, err := readJSONBodyMap(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}

	model := strings.TrimSpace(mapString(bodyMap, "model"))
	if !h.ensureSpeechModelAllowed(c, key, model) {
		return
	}

	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/v1/t2a_v2"
	start := time.Now()
	if mapBool(bodyMap, "stream") {
		statusCode, err := h.proxyMiniMaxStream(c, apiKey, http.MethodPost, endpoint, bodyBytes, c.Request.Header)
		if err != nil {
			h.logAPIRequest(key, model, "minimax", "/api/minimax/speech/v1/t2a_v2", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "[stream]", c.ClientIP(), time.Since(start), usageSummary{})
			if !c.Writer.Written() {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
			}
			return
		}
		h.logAPIRequest(key, model, "minimax", "/api/minimax/speech/v1/t2a_v2", "media", statusCode, statusCode < 400, "", string(bodyBytes), "[stream]", c.ClientIP(), time.Since(start), usageSummary{})
		return
	}

	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodPost, endpoint, bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "minimax", "/api/minimax/speech/v1/t2a_v2", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}

	usage, baseErr := h.buildSpeechUsage(model, respBody)
	success := statusCode < 400 && baseErr == ""
	logStatus := statusCode
	if baseErr != "" && logStatus < 400 {
		logStatus = http.StatusBadGateway
	}
	h.logAPIRequest(key, model, "minimax", "/api/minimax/speech/v1/t2a_v2", "media", logStatus, success, baseErr, string(bodyBytes), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usage)
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) MiniMaxSpeechAsyncCreate(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	apiKeyID := speechAPIKeyID(key)

	bodyBytes, bodyMap, err := readJSONBodyMap(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	model := strings.TrimSpace(mapString(bodyMap, "model"))
	if !h.ensureSpeechModelAllowed(c, key, model) {
		return
	}
	for _, candidate := range []string{mapString(bodyMap, "text_file_id"), mapString(bodyMap, "prompt_audio")} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && !h.ensureSpeechFileAllowed(c, key, candidate) {
			return
		}
	}

	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/v1/t2a_async_v2"
	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodPost, endpoint, bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "minimax", "/api/minimax/speech/v1/t2a_async_v2", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}

	usage, raw, baseErr := h.buildSpeechUsageAndRaw(model, respBody)
	taskID := minimaxSpeechTaskID(raw)
	outputFileID := minimaxSpeechFileID(raw)
	status := minimaxSpeechStatus(raw)
	if status == "" {
		status = "submitted"
	}
	if taskID != "" {
		_ = h.db.UpsertMiniMaxSpeechTask(&models.MiniMaxSpeechTask{
			TaskID:       taskID,
			APIKeyID:     apiKeyID,
			Model:        model,
			Status:       status,
			OutputFileID: outputFileID,
			RequestBody:  truncateString(string(bodyBytes), 10000),
			ResultBody:   truncateString(string(respBody), 10000),
			ErrorMessage: baseErr,
		})
	}
	if outputFileID != "" {
		_ = h.db.UpsertMiniMaxSpeechFile(&models.MiniMaxSpeechFile{
			APIKeyID: apiKeyID,
			FileID:   outputFileID,
			Purpose:  "t2a_async_output",
			Status:   status,
		})
	}

	success := statusCode < 400 && baseErr == ""
	logStatus := statusCode
	if baseErr != "" && logStatus < 400 {
		logStatus = http.StatusBadGateway
	}
	h.logAPIRequest(key, model, "minimax", "/api/minimax/speech/v1/t2a_async_v2", "media", logStatus, success, baseErr, string(bodyBytes), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usage)
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) MiniMaxSpeechAsyncQuery(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	taskID := strings.TrimSpace(c.Query("task_id"))
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 task_id", "code": 400})
		return
	}
	if !h.ensureSpeechTaskAllowed(c, key, taskID) {
		return
	}

	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/v1/query/t2a_async_query_v2?" + url.Values{"task_id": []string{taskID}}.Encode()
	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodGet, endpoint, nil, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, "speech-async", "minimax", "/api/minimax/speech/v1/query/t2a_async_query_v2", "media", http.StatusBadGateway, false, err.Error(), taskID, "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}

	usage, raw, baseErr := h.buildSpeechUsageAndRaw("", respBody)
	status := minimaxSpeechStatus(raw)
	outputFileID := minimaxSpeechFileID(raw)
	if task, err := h.db.GetMiniMaxSpeechTask(taskID); err == nil {
		model := task.Model
		if model != "" {
			usage, _, _ = h.buildSpeechUsageAndRaw(model, respBody)
		}
		_ = h.db.UpsertMiniMaxSpeechTask(&models.MiniMaxSpeechTask{
			TaskID:       taskID,
			APIKeyID:     task.APIKeyID,
			Model:        task.Model,
			Status:       fallbackString(status, task.Status),
			OutputFileID: fallbackString(outputFileID, task.OutputFileID),
			RequestBody:  task.RequestBody,
			ResultBody:   truncateString(string(respBody), 10000),
			ErrorMessage: baseErr,
		})
		if outputFileID != "" {
			_ = h.db.UpsertMiniMaxSpeechFile(&models.MiniMaxSpeechFile{
				APIKeyID: task.APIKeyID,
				FileID:   outputFileID,
				Purpose:  "t2a_async_output",
				Status:   fallbackString(status, "available"),
			})
		}
	}

	success := statusCode < 400 && baseErr == ""
	logStatus := statusCode
	if baseErr != "" && logStatus < 400 {
		logStatus = http.StatusBadGateway
	}
	h.logAPIRequest(key, "speech-async", "minimax", "/api/minimax/speech/v1/query/t2a_async_query_v2", "media", logStatus, success, baseErr, taskID, truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usage)
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) MiniMaxSpeechListTasks(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	limit := boundedInt(c.Query("limit"), 20, 1, 100)
	offset := boundedInt(c.Query("offset"), 0, 0, 100000)
	tasks, err := h.db.ListMiniMaxSpeechTasks(speechAPIKeyID(key), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取语音任务列表失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks, "limit": limit, "offset": offset})
}

func (h *AIGatewayHandler) MiniMaxSpeechGetTask(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	taskID := c.Param("id")
	if !h.ensureSpeechTaskAllowed(c, key, taskID) {
		return
	}
	task, err := h.db.GetMiniMaxSpeechTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}
	payload := gin.H{"task": task}
	if task.ResultBody != "" {
		var raw map[string]interface{}
		if json.Unmarshal([]byte(task.ResultBody), &raw) == nil {
			payload["raw_response"] = raw
		}
	}
	c.JSON(http.StatusOK, payload)
}

func (h *AIGatewayHandler) MiniMaxSpeechGetVoice(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	bodyBytes, err := readOptionalJSONBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	bodyMap := map[string]interface{}{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体 JSON 格式错误", "code": 400})
		return
	}
	voiceID := strings.TrimSpace(firstNonEmpty(mapString(bodyMap, "voice_id"), c.Query("voice_id")))
	if key != nil && voiceID != "" {
		if clone, err := h.db.GetVoiceCloneByVoiceID(voiceID); err == nil && clone.APIKeyID != key.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权查看该自定义音色", "code": 403})
			return
		}
	}
	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/get_voice"
	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodPost, endpoint, bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, "speech-voice", "minimax", "/api/minimax/speech/v1/get_voice", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	_, raw, baseErr := h.buildSpeechUsageAndRaw("", respBody)
	if key != nil && voiceID == "" {
		filterSpeechVoices(raw, h.ownedSpeechVoiceIDs(key.ID))
		if filteredBody, err := json.Marshal(raw); err == nil {
			respBody = filteredBody
		}
	}
	h.logAPIRequest(key, "speech-voice", "minimax", "/api/minimax/speech/v1/get_voice", "media", statusCode, statusCode < 400 && baseErr == "", baseErr, string(bodyBytes), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usageSummary{})
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) MiniMaxSpeechVoiceDesign(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	apiKeyID := speechAPIKeyID(key)
	bodyBytes, bodyMap, err := readJSONBodyMap(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	voiceID := strings.TrimSpace(mapString(bodyMap, "voice_id"))
	if voiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 voice_id", "code": 400})
		return
	}
	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/voice_design"
	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodPost, endpoint, bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, "voice-design", "minimax", "/api/minimax/speech/v1/voice_design", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	_, _, baseErr := h.buildSpeechUsageAndRaw("", respBody)
	if baseErr == "" {
		h.persistSpeechVoice(apiKeyID, voiceID, firstNonEmpty(mapString(bodyMap, "voice_name"), mapString(bodyMap, "prompt"), voiceID), "active")
	}
	h.logAPIRequest(key, "voice-design", "minimax", "/api/minimax/speech/v1/voice_design", "media", statusCode, statusCode < 400 && baseErr == "", baseErr, string(bodyBytes), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usageSummary{})
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) MiniMaxSpeechVoiceClone(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	apiKeyID := speechAPIKeyID(key)
	bodyBytes, bodyMap, err := readJSONBodyMap(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	fileID := strings.TrimSpace(firstNonEmpty(mapString(bodyMap, "file_id"), mapString(bodyMap, "source_file_id")))
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 file_id", "code": 400})
		return
	}
	if !h.ensureSpeechFileAllowed(c, key, fileID) {
		return
	}
	promptAudio := strings.TrimSpace(mapString(bodyMap, "prompt_audio"))
	if promptAudio != "" && !h.ensureSpeechFileAllowed(c, key, promptAudio) {
		return
	}
	voiceID := strings.TrimSpace(firstNonEmpty(mapString(bodyMap, "voice_id"), mapString(bodyMap, "custom_voice_id")))
	if voiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 voice_id", "code": 400})
		return
	}
	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/voice_clone"
	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodPost, endpoint, bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, "voice-clone", "minimax", "/api/minimax/speech/v1/voice_clone", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	_, _, baseErr := h.buildSpeechUsageAndRaw("", respBody)
	if baseErr == "" {
		h.persistSpeechVoice(apiKeyID, voiceID, firstNonEmpty(mapString(bodyMap, "voice_name"), voiceID), "active")
	}
	h.logAPIRequest(key, "voice-clone", "minimax", "/api/minimax/speech/v1/voice_clone", "media", statusCode, statusCode < 400 && baseErr == "", baseErr, string(bodyBytes), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usageSummary{})
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) MiniMaxSpeechDeleteVoice(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	bodyBytes, bodyMap, err := readJSONBodyMap(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	voiceID := strings.TrimSpace(firstNonEmpty(mapString(bodyMap, "voice_id"), c.Query("voice_id")))
	if voiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 voice_id", "code": 400})
		return
	}
	var localVoice *models.VoiceClone
	if key != nil {
		localVoice, err = h.db.GetVoiceCloneByVoiceID(voiceID)
		if err != nil || localVoice.APIKeyID != key.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权删除该音色", "code": 403})
			return
		}
	}
	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/delete_voice"
	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodPost, endpoint, bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, "delete-voice", "minimax", "/api/minimax/speech/v1/delete_voice", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	_, _, baseErr := h.buildSpeechUsageAndRaw("", respBody)
	if baseErr == "" {
		if localVoice != nil {
			_ = h.db.DeleteVoiceClone(localVoice.ID, localVoice.APIKeyID)
		} else if adminVoice, err := h.db.GetVoiceCloneByVoiceID(voiceID); err == nil {
			_ = h.db.DeleteVoiceCloneAny(adminVoice.ID)
		}
	}
	h.logAPIRequest(key, "delete-voice", "minimax", "/api/minimax/speech/v1/delete_voice", "media", statusCode, statusCode < 400 && baseErr == "", baseErr, string(bodyBytes), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usageSummary{})
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) MiniMaxSpeechUploadFile(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	apiKeyID := speechAPIKeyID(key)
	purpose := strings.TrimSpace(c.PostForm("purpose"))
	if purpose == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 purpose", "code": 400})
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 file", "code": 400})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败", "code": 500})
		return
	}

	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("purpose", purpose)
	part, err := writer.CreateFormFile("file", header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "构造上传请求失败", "code": 500})
		return
	}
	if _, err := part.Write(fileBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入上传请求失败", "code": 500})
		return
	}
	_ = writer.Close()

	forwardHeaders := cloneHeaders(c.Request.Header)
	forwardHeaders.Set("Content-Type", writer.FormDataContentType())
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/files/upload"
	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodPost, endpoint, body.Bytes(), forwardHeaders)
	if err != nil {
		h.logAPIRequest(key, "speech-file", "minimax", "/api/minimax/speech/v1/files/upload", "media", http.StatusBadGateway, false, err.Error(), fmt.Sprintf("purpose=%s,name=%s,size=%d", purpose, header.Filename, len(fileBytes)), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	_, raw, baseErr := h.buildSpeechUsageAndRaw("", respBody)
	fileID := minimaxSpeechFileID(raw)
	if baseErr == "" && fileID != "" {
		_ = h.db.UpsertMiniMaxSpeechFile(&models.MiniMaxSpeechFile{
			APIKeyID: apiKeyID,
			FileID:   fileID,
			Purpose:  purpose,
			Filename: header.Filename,
			Bytes:    int64(len(fileBytes)),
			Status:   "available",
		})
	}
	h.logAPIRequest(key, "speech-file", "minimax", "/api/minimax/speech/v1/files/upload", "media", statusCode, statusCode < 400 && baseErr == "", baseErr, fmt.Sprintf("purpose=%s,name=%s,size=%d", purpose, header.Filename, len(fileBytes)), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usageSummary{})
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) MiniMaxSpeechListFiles(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/files/list"
	if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
		endpoint += "?" + rawQuery
	}
	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodGet, endpoint, nil, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, "speech-file", "minimax", "/api/minimax/speech/v1/files/list", "media", http.StatusBadGateway, false, err.Error(), c.Request.URL.RawQuery, "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	_, raw, baseErr := h.buildSpeechUsageAndRaw("", respBody)
	if key != nil {
		filterSpeechFiles(raw, h.ownedSpeechFileIDs(key.ID))
		respBody, _ = json.Marshal(raw)
	}
	h.logAPIRequest(key, "speech-file", "minimax", "/api/minimax/speech/v1/files/list", "media", statusCode, statusCode < 400 && baseErr == "", baseErr, c.Request.URL.RawQuery, truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usageSummary{})
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) MiniMaxSpeechRetrieveFile(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	fileID := strings.TrimSpace(c.Query("file_id"))
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 file_id", "code": 400})
		return
	}
	if !h.ensureSpeechFileAllowed(c, key, fileID) {
		return
	}
	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/files/retrieve?" + url.Values{"file_id": []string{fileID}}.Encode()
	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodGet, endpoint, nil, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, "speech-file", "minimax", "/api/minimax/speech/v1/files/retrieve", "media", http.StatusBadGateway, false, err.Error(), fileID, "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	_, _, baseErr := h.buildSpeechUsageAndRaw("", respBody)
	h.logAPIRequest(key, "speech-file", "minimax", "/api/minimax/speech/v1/files/retrieve", "media", statusCode, statusCode < 400 && baseErr == "", baseErr, fileID, truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usageSummary{})
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) MiniMaxSpeechRetrieveFileContent(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	fileID := strings.TrimSpace(c.Query("file_id"))
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 file_id", "code": 400})
		return
	}
	if !h.ensureSpeechFileAllowed(c, key, fileID) {
		return
	}
	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/files/retrieve_content?" + url.Values{"file_id": []string{fileID}}.Encode()
	start := time.Now()
	statusCode, err := h.proxyMiniMaxBinary(c, apiKey, http.MethodGet, endpoint, nil, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, "speech-file", "minimax", "/api/minimax/speech/v1/files/retrieve_content", "media", http.StatusBadGateway, false, err.Error(), fileID, "", c.ClientIP(), time.Since(start), usageSummary{})
		if !c.Writer.Written() {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		}
		return
	}
	h.logAPIRequest(key, "speech-file", "minimax", "/api/minimax/speech/v1/files/retrieve_content", "media", statusCode, statusCode < 400, "", fileID, "[binary]", c.ClientIP(), time.Since(start), usageSummary{})
}

func (h *AIGatewayHandler) MiniMaxSpeechDeleteFile(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}
	bodyBytes, bodyMap, err := readJSONBodyMap(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	fileID := strings.TrimSpace(firstNonEmpty(mapString(bodyMap, "file_id"), c.Query("file_id")))
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 file_id", "code": 400})
		return
	}
	if !h.ensureSpeechFileAllowed(c, key, fileID) {
		return
	}
	apiKey, baseURL, err := h.minimaxSpeechConfig()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/files/delete"
	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxRequest(apiKey, http.MethodPost, endpoint, bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, "speech-file", "minimax", "/api/minimax/speech/v1/files/delete", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	_, _, baseErr := h.buildSpeechUsageAndRaw("", respBody)
	if baseErr == "" {
		_ = h.db.DeleteMiniMaxSpeechFile(fileID, speechAPIKeyID(key))
	}
	h.logAPIRequest(key, "speech-file", "minimax", "/api/minimax/speech/v1/files/delete", "media", statusCode, statusCode < 400 && baseErr == "", baseErr, string(bodyBytes), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), usageSummary{})
	if baseErr != "" && statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) minimaxSpeechConfig() (string, string, error) {
	apiKey := firstNonEmpty(
		h.cfg.MiniMaxTTS.APIKey,
		h.cfg.MiniMaxVoiceCloning.APIKey,
		h.cfg.MiniMaxTokenPlan.APIKey,
		h.cfg.MiniMax.APIKey,
	)
	baseURL := firstNonEmpty(
		h.cfg.MiniMaxTTS.BaseURL,
		h.cfg.MiniMaxVoiceCloning.BaseURL,
		h.cfg.MiniMaxTokenPlan.BaseURL,
		defaultMiniMaxSpeechBaseURL,
	)
	if apiKey == "" {
		return "", "", fmt.Errorf("未配置 MiniMax Speech API Key")
	}
	return apiKey, baseURL, nil
}

func speechAPIKeyID(key *models.AIAPIKey) string {
	if key == nil {
		return ""
	}
	return key.ID
}

func readJSONBodyMap(c *gin.Context) ([]byte, map[string]interface{}, error) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("读取请求体失败")
	}
	if len(bytes.TrimSpace(bodyBytes)) == 0 {
		return nil, nil, fmt.Errorf("请求体为空")
	}
	bodyMap := map[string]interface{}{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		return nil, nil, fmt.Errorf("请求体 JSON 格式错误")
	}
	return bodyBytes, bodyMap, nil
}

func readOptionalJSONBody(c *gin.Context) ([]byte, error) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("读取请求体失败")
	}
	trimmed := bytes.TrimSpace(bodyBytes)
	if len(trimmed) == 0 {
		return []byte(`{}`), nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return nil, fmt.Errorf("请求体 JSON 格式错误")
	}
	return trimmed, nil
}

func mapString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if value, ok := m[key]; ok {
		return interfaceToString(value)
	}
	return ""
}

func mapBool(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	value, ok := m[key]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (h *AIGatewayHandler) ensureSpeechModelAllowed(c *gin.Context, key *models.AIAPIKey, model string) bool {
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model 字段", "code": 400})
		return false
	}
	if !isModelAllowed(model, TTSAllowedModels) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("当前语音网关不支持模型 %s", model), "code": 400})
		return false
	}
	if key != nil {
		return h.ensureModelAllowed(c, key, model)
	}
	return true
}

func (h *AIGatewayHandler) ensureSpeechFileAllowed(c *gin.Context, key *models.AIAPIKey, fileID string) bool {
	if key == nil {
		return true
	}
	file, err := h.db.GetMiniMaxSpeechFileByFileID(fileID)
	if err != nil || file.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该文件，请先通过当前 API Key 上传或生成此文件", "code": 403})
		return false
	}
	return true
}

func (h *AIGatewayHandler) ensureSpeechTaskAllowed(c *gin.Context, key *models.AIAPIKey, taskID string) bool {
	if key == nil {
		return true
	}
	task, err := h.db.GetMiniMaxSpeechTask(taskID)
	if err != nil || task.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该语音任务", "code": 403})
		return false
	}
	return true
}

func cloneHeaders(src http.Header) http.Header {
	result := make(http.Header)
	for key, values := range src {
		switch http.CanonicalHeaderKey(key) {
		case "Authorization", "Content-Length", "Host", "Connection", "Proxy-Connection", "Upgrade", "Keep-Alive", "Te", "Trailer", "Transfer-Encoding":
			continue
		}
		for _, value := range values {
			result.Add(key, value)
		}
	}
	return result
}

func (h *AIGatewayHandler) performMiniMaxRequest(apiKey, method, endpoint string, body []byte, incoming http.Header) ([]byte, int, http.Header, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header = cloneHeaders(incoming)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, nil, err
	}
	return respBody, resp.StatusCode, resp.Header.Clone(), nil
}

func (h *AIGatewayHandler) proxyMiniMaxBinary(c *gin.Context, apiKey, method, endpoint string, body []byte, incoming http.Header) (int, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return 0, err
	}
	req.Header = cloneHeaders(incoming)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	copyHeaders(c.Writer.Header(), resp.Header)
	c.Status(resp.StatusCode)
	_, err = io.Copy(c.Writer, resp.Body)
	return resp.StatusCode, err
}

func (h *AIGatewayHandler) proxyMiniMaxStream(c *gin.Context, apiKey, method, endpoint string, body []byte, incoming http.Header) (int, error) {
	return h.proxyMiniMaxBinary(c, apiKey, method, endpoint, body, incoming)
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		switch http.CanonicalHeaderKey(key) {
		case "Content-Length", "Transfer-Encoding", "Connection":
			continue
		}
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func writeMiniMaxResponse(c *gin.Context, statusCode int, headers http.Header, body []byte) {
	if headers != nil {
		copyHeaders(c.Writer.Header(), headers)
	}
	if c.Writer.Header().Get("Content-Type") == "" {
		c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	c.Data(statusCode, c.Writer.Header().Get("Content-Type"), body)
}

func minimaxBaseRespError(raw map[string]interface{}) string {
	for _, key := range []string{"base_resp", "base_response"} {
		baseResp, ok := raw[key].(map[string]interface{})
		if !ok {
			continue
		}
		statusCode := interfaceToInt(baseResp["status_code"])
		if statusCode == 0 {
			continue
		}
		return firstNonEmpty(interfaceToString(baseResp["status_msg"]), interfaceToString(baseResp["status_message"]), fmt.Sprintf("MiniMax 返回业务错误 %d", statusCode))
	}
	return ""
}

func (h *AIGatewayHandler) buildSpeechUsage(model string, respBody []byte) (usageSummary, string) {
	usage, _, baseErr := h.buildSpeechUsageAndRaw(model, respBody)
	return usage, baseErr
}

func (h *AIGatewayHandler) buildSpeechUsageAndRaw(model string, respBody []byte) (usageSummary, map[string]interface{}, string) {
	raw := map[string]interface{}{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return usageSummary{}, nil, ""
	}
	characters := 0
	if extraInfo, ok := raw["extra_info"].(map[string]interface{}); ok {
		characters = interfaceToInt(extraInfo["usage_characters"])
	}
	if characters == 0 {
		characters = interfaceToInt(raw["usage_characters"])
	}
	cost, currency := h.calculateCost(model, "minimax", characters, 0)
	usage := usageSummary{InputTokens: characters, TotalTokens: characters, Cost: cost, Currency: currency}
	return usage, raw, minimaxBaseRespError(raw)
}

func minimaxSpeechTaskID(raw map[string]interface{}) string {
	if raw == nil {
		return ""
	}
	if data, ok := raw["data"].(map[string]interface{}); ok {
		for _, key := range []string{"task_id", "taskId"} {
			if value := strings.TrimSpace(interfaceToString(data[key])); value != "" {
				return value
			}
		}
	}
	for _, key := range []string{"task_id", "taskId"} {
		if value := strings.TrimSpace(interfaceToString(raw[key])); value != "" {
			return value
		}
	}
	return ""
}

func minimaxSpeechFileID(raw map[string]interface{}) string {
	if raw == nil {
		return ""
	}
	for _, source := range []map[string]interface{}{speechResponseRoot(raw), raw} {
		if source == nil {
			continue
		}
		if fileMap, ok := source["file"].(map[string]interface{}); ok {
			for _, key := range []string{"file_id", "audio_file_id", "result_file_id"} {
				if value := strings.TrimSpace(interfaceToString(fileMap[key])); value != "" {
					return value
				}
			}
		}
		for _, key := range []string{"file_id", "audio_file_id", "result_file_id"} {
			if value := strings.TrimSpace(interfaceToString(source[key])); value != "" {
				return value
			}
		}
	}
	return ""
}

func minimaxSpeechStatus(raw map[string]interface{}) string {
	if raw == nil {
		return ""
	}
	if data, ok := raw["data"].(map[string]interface{}); ok {
		for _, key := range []string{"status", "task_status", "state"} {
			if value := strings.TrimSpace(interfaceToString(data[key])); value != "" {
				return value
			}
		}
	}
	for _, key := range []string{"status", "task_status", "state"} {
		if value := strings.TrimSpace(interfaceToString(raw[key])); value != "" {
			return value
		}
	}
	return ""
}

func speechResponseRoot(raw map[string]interface{}) map[string]interface{} {
	if raw == nil {
		return nil
	}
	if data, ok := raw["data"].(map[string]interface{}); ok {
		return data
	}
	return raw
}

func filterSpeechFiles(raw map[string]interface{}, allowed map[string]bool) {
	root := speechResponseRoot(raw)
	if root == nil {
		return
	}
	for _, key := range []string{"file_list", "files", "items"} {
		list, ok := root[key].([]interface{})
		if !ok {
			continue
		}
		filtered := make([]interface{}, 0, len(list))
		for _, item := range list {
			fileMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			fileID := strings.TrimSpace(firstNonEmpty(interfaceToString(fileMap["file_id"]), interfaceToString(fileMap["id"])))
			if allowed[fileID] {
				filtered = append(filtered, item)
			}
		}
		root[key] = filtered
	}
	if fileMap, ok := root["file"].(map[string]interface{}); ok {
		fileID := strings.TrimSpace(firstNonEmpty(interfaceToString(fileMap["file_id"]), interfaceToString(fileMap["id"])))
		if fileID != "" && !allowed[fileID] {
			delete(root, "file")
		}
	}
}

func filterSpeechVoices(raw map[string]interface{}, allowed map[string]bool) {
	root := speechResponseRoot(raw)
	if root == nil {
		return
	}
	for _, key := range []string{"voice_cloning", "voice_generation", "custom_voice_list", "voice_list", "voices", "items"} {
		list, ok := root[key].([]interface{})
		if !ok {
			continue
		}
		filtered := make([]interface{}, 0, len(list))
		for _, item := range list {
			voiceMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if speechVoiceVisible(voiceMap, allowed) {
				filtered = append(filtered, item)
			}
		}
		root[key] = filtered
	}
	if voiceMap, ok := root["voice"].(map[string]interface{}); ok && !speechVoiceVisible(voiceMap, allowed) {
		delete(root, "voice")
	}
}

func speechVoiceVisible(voice map[string]interface{}, allowed map[string]bool) bool {
	voiceID := strings.TrimSpace(firstNonEmpty(interfaceToString(voice["voice_id"]), interfaceToString(voice["id"])))
	if voiceID == "" {
		return false
	}
	if allowed[voiceID] {
		return true
	}
	voiceType := strings.ToLower(strings.TrimSpace(firstNonEmpty(
		interfaceToString(voice["voice_type"]),
		interfaceToString(voice["type"]),
		interfaceToString(voice["category"]),
	)))
	switch voiceType {
	case "system", "system_voice", "preset", "official", "public", "builtin":
		return true
	case "voice_cloning", "voice_generation", "custom", "user", "clone", "generated":
		return false
	}
	return strings.HasPrefix(voiceID, "male-") || strings.HasPrefix(voiceID, "female-")
}

func (h *AIGatewayHandler) ownedSpeechFileIDs(apiKeyID string) map[string]bool {
	files, err := h.db.ListMiniMaxSpeechFiles(apiKeyID, "", 1000, 0)
	if err != nil {
		return map[string]bool{}
	}
	result := make(map[string]bool, len(files))
	for _, file := range files {
		result[file.FileID] = true
	}
	return result
}

func (h *AIGatewayHandler) ownedSpeechVoiceIDs(apiKeyID string) map[string]bool {
	clones, err := h.db.ListVoiceClones(apiKeyID, 1000, 0)
	if err != nil {
		return map[string]bool{}
	}
	result := make(map[string]bool, len(clones))
	for _, clone := range clones {
		result[clone.VoiceID] = true
	}
	return result
}

func (h *AIGatewayHandler) persistSpeechVoice(apiKeyID, voiceID, name, status string) {
	if strings.TrimSpace(voiceID) == "" {
		return
	}
	if clone, err := h.db.GetVoiceCloneByVoiceID(voiceID); err == nil {
		if status != "" {
			_ = h.db.UpdateVoiceCloneStatus(clone.ID, status)
		}
		return
	}
	_ = h.db.CreateVoiceClone(&models.VoiceClone{
		APIKeyID: apiKeyID,
		VoiceID:  voiceID,
		Name:     truncateString(strings.TrimSpace(name), 120),
		Status:   firstNonEmpty(status, "active"),
	})
}

func decodeMiniMaxAudioHex(audioHex string) ([]byte, error) {
	audioHex = strings.TrimSpace(audioHex)
	audioHex = strings.TrimPrefix(audioHex, "0x")
	if audioHex == "" {
		return nil, fmt.Errorf("音频数据为空")
	}
	return hex.DecodeString(audioHex)
}
