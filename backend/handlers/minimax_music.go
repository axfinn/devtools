package handlers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

const defaultMiniMaxMusicBaseURL = "https://api.minimaxi.com"

func (h *AIGatewayHandler) GetMiniMaxMusicDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "MiniMax Music Gateway 文档",
		"summary": "MiniMax 音乐相关能力聚合入口，支持歌词生成与 music-cover 翻唱前处理；音乐生成本身继续复用 /api/minimax/token-plan/v1/generations。",
		"auth": gin.H{
			"api_key":      "Authorization: Bearer dtk_ai_xxx",
			"admin_header": "X-Super-Admin-Password",
			"scope":        "media",
		},
		"base_url": "/api/minimax/music",
		"upstream": defaultMiniMaxMusicBaseURL,
		"routes": []gin.H{
			{"method": "GET", "path": "/api/minimax/music/docs", "description": "获取音乐工作流文档"},
			{"method": "POST", "path": "/api/minimax/music/v1/lyrics_generation", "description": "异步歌词生成(立即返回 task_id,后台跑最多 5 分钟;前端 GET /api/minimax/music/v1/lyrics_tasks/:id 轮询)"},
			{"method": "GET", "path": "/api/minimax/music/v1/lyrics_tasks/:id", "description": "轮询歌词生成任务状态;status=succeeded 时 result 字段是上游 lyrics 响应"},
			{"method": "POST", "path": "/api/minimax/music/v1/cover_preprocess", "description": "翻唱前处理，获取 cover_feature_id"},
			{"method": "POST", "path": "/api/minimax/token-plan/v1/generations", "description": "音乐生成（支持 music-3.0 / music-3.0-free / music-2.6 / music-2.6-free / music-cover / music-cover-free）"},
		},
		"examples": gin.H{
			"lyrics_generation": gin.H{
				"mode":   "write_full_song",
				"prompt": "一首关于夏日海边的轻快情歌",
			},
			"cover_preprocess": gin.H{
				"model":     "music-cover",
				"audio_url": "https://example.com/song.mp3",
			},
			"music_generation": gin.H{
				"model":            "music-cover",
				"prompt":           "Mandopop, warm male vocal, emotional chorus",
				"cover_feature_id": "your-cover-feature-id",
				"lyrics":           "[Verse]\\n这一段歌词...",
				"formatted_lyrics": "[Verse]\\n这一段歌词...",
				"structure_result": "{\"num_segments\":2,\"segments\":[...]}",
				"audio_duration":   19.2,
			},
		},
	})
}

func (h *AIGatewayHandler) MiniMaxLyricsGeneration(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
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

	apiKey := h.cfg.MiniMaxTokenPlan.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey
	}
	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax 音乐 API Key", "code": 502})
		return
	}

	baseURL := strings.TrimRight(h.cfg.MiniMaxTokenPlan.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultMiniMaxMusicBaseURL
	}
	upstreamURL := baseURL + "/v1/lyrics_generation"
	endpoint := "/api/minimax/music/v1/lyrics_generation"

	// 异步歌词生成：立即返回 task_id，把上游调用扔到 goroutine 里跑（最多 5 分钟 + 4 次重试）。
	// 同步等待 25-30s 会让 CF(t.jaxiu.cn) 的 100s origin read timeout 把客户端掐了返 524。
	start := time.Now()

	taskID := "mml_" + utils.GenerateHexKey(12)
	task := &models.MiniMaxMediaTask{
		ID:          taskID,
		APIKeyID:    firstAPIKeyID(key),
		Model:       "lyrics_generation",
		Provider:    "minimax-music",
		Status:      "pending",
		RequestBody: truncateString(string(bodyBytes), 50000),
		ClientIP:    c.ClientIP(),
	}
	if err := h.db.CreateMiniMaxMediaTask(task); err != nil {
		h.logAPIRequest(key, "lyrics_generation", "minimax-music", endpoint, "media", http.StatusInternalServerError, false, err.Error(), truncateString(string(bodyBytes), 10000), "", c.ClientIP(), time.Since(start), h.buildMediaUsage("lyrics_generation"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败", "code": 500})
		return
	}

	// 把"提交"本身也记一行日志，便于排查客户端发了啥。
	h.logAPIRequest(key, "lyrics_generation", "minimax-music", endpoint, "media", http.StatusAccepted, true, "", truncateString(string(bodyBytes), 10000), "", c.ClientIP(), time.Since(start), h.buildMediaUsage("lyrics_generation"))

	c.JSON(http.StatusAccepted, gin.H{
		"task_id":  taskID,
		"model":    "lyrics_generation",
		"status":   "pending",
		"created_at": task.CreatedAt.Format(time.RFC3339),
		"poll_url": "/api/minimax/music/v1/lyrics_tasks/" + taskID,
		"message":  "歌词生成任务已提交,请通过 GET " + "/api/minimax/music/v1/lyrics_tasks/" + taskID + " 轮询结果",
	})

	// 在 goroutine 里跑:用 musicSubmitClient(5 分钟)给慢场景留 buffer,
	// 复用 4 次重试 + 指数退避抖动的逻辑(performMiniMaxMusicRequestWithClient)。
	go h.runAsyncLyricsGeneration(taskID, apiKey, upstreamURL, bodyBytes, c.Request.Header, firstAPIKeyID(key))
}

// runAsyncLyricsGeneration 在后台 goroutine 里跑歌词生成,完成后回写 task 状态。
// 失败也回写 failed + error_message,前端轮询能看到原因。
//
// 上游判断"成功"只看 base_resp.status_code == 0,歌词字段空不空由前端 extractLyricsText 处理;
// 后端不要再画蛇添足去判定空内容然后重试,反而把正常响应误判成 failed 让整个歌曲生成流程断掉。
func (h *AIGatewayHandler) runAsyncLyricsGeneration(taskID, apiKey, upstreamURL string, bodyBytes []byte, incoming http.Header, apiKeyID string) {
	task, err := h.db.GetMiniMaxMediaTask(taskID)
	if err != nil {
		return
	}

	task.Status = "running"
	_ = h.db.UpdateMiniMaxMediaTask(task)

	start := time.Now()
	endpoint := "/api/minimax/music/v1/lyrics_generation"
	respBody, statusCode, _, err := h.performMiniMaxMusicRequestWithClient(h.musicSubmitClient, apiKey, upstreamURL, bodyBytes, incoming)

	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateMiniMaxMediaTask(task)
		h.logAPIRequestByID(apiKeyID, "lyrics_generation", "minimax-music", endpoint, "media", http.StatusBadGateway, false, err.Error(), truncateString(string(bodyBytes), 10000), "", task.ClientIP, time.Since(start), h.buildMediaUsage("lyrics_generation"))
		return
	}

	if statusCode >= 400 {
		task.Status = "failed"
		task.ErrorMessage = fmt.Sprintf("upstream HTTP %d: %s", statusCode, truncateString(string(respBody), 1000))
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateMiniMaxMediaTask(task)
		h.logAPIRequestByID(apiKeyID, "lyrics_generation", "minimax-music", endpoint, "media", statusCode, false, task.ErrorMessage, truncateString(string(bodyBytes), 10000), truncateString(string(respBody), 10000), task.ClientIP, time.Since(start), h.buildMediaUsage("lyrics_generation"))
		return
	}

	task.Status = "succeeded"
	task.ResultJSON = string(respBody)
	now := time.Now()
	task.CompletedAt = &now
	_ = h.db.UpdateMiniMaxMediaTask(task)
	h.logAPIRequestByID(apiKeyID, "lyrics_generation", "minimax-music", endpoint, "media", http.StatusOK, true, "", truncateString(string(bodyBytes), 10000), truncateString(string(respBody), 10000), task.ClientIP, time.Since(start), h.buildMediaUsage("lyrics_generation"))
}

// GetMiniMaxLyricsTask GET /api/minimax/music/v1/lyrics_tasks/:id
// 返回 task 状态,succeeded 时 result 字段里是上游 lyrics 响应(同 /v1/lyrics_generation 同步版的形状)。
func (h *AIGatewayHandler) GetMiniMaxLyricsTask(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	taskID := c.Param("id")
	task, err := h.db.GetMiniMaxMediaTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}
	if task.Model != "lyrics_generation" {
		c.JSON(http.StatusNotFound, gin.H{"error": "不是歌词生成任务", "code": 404})
		return
	}
	if key != nil && task.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此任务", "code": 403})
		return
	}

	result := gin.H{
		"task_id":    task.ID,
		"model":      task.Model,
		"provider":   task.Provider,
		"status":     task.Status,
		"error":      task.ErrorMessage,
		"created_at": task.CreatedAt.Format(time.RFC3339),
	}
	if task.CompletedAt != nil {
		result["completed_at"] = task.CompletedAt.Format(time.RFC3339)
	}
	if task.ResultJSON != "" {
		var r map[string]interface{}
		if err := json.Unmarshal([]byte(task.ResultJSON), &r); err == nil {
			result["result"] = r
		} else {
			result["result_raw"] = task.ResultJSON
		}
	}
	c.JSON(http.StatusOK, result)
}

func (h *AIGatewayHandler) MiniMaxMusicCoverPreprocess(c *gin.Context) {
	h.proxyMiniMaxMusicJSON(c, "/v1/music_cover_preprocess", "music-cover")
}

func (h *AIGatewayHandler) proxyMiniMaxMusicJSON(c *gin.Context, upstreamPath, model string) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
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

	apiKey := h.cfg.MiniMaxTokenPlan.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey
	}
	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax 音乐 API Key", "code": 502})
		return
	}

	baseURL := strings.TrimRight(h.cfg.MiniMaxTokenPlan.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultMiniMaxMusicBaseURL
	}
	upstreamURL := baseURL + upstreamPath
	endpoint := "/api/minimax/music" + upstreamPath

	start := time.Now()
	respBody, statusCode, respHeader, err := h.performMiniMaxMusicRequest(apiKey, upstreamURL, bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "minimax-music", endpoint, "media", http.StatusBadGateway, false, err.Error(), truncateString(string(bodyBytes), 10000), "", c.ClientIP(), time.Since(start), h.buildMediaUsage(model))
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}

	logStatus := statusCode
	success := statusCode < 400
	baseErr := ""
	var payload map[string]interface{}
	if err := json.Unmarshal(respBody, &payload); err == nil {
		baseErr = minimaxBaseRespError(payload)
		if baseErr != "" {
			success = false
			if logStatus < 400 {
				logStatus = http.StatusBadGateway
			}
		}
	}

	h.logAPIRequest(key, model, "minimax-music", endpoint, "media", logStatus, success, baseErr, truncateString(string(bodyBytes), 10000), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), h.buildMediaUsage(model))
	writeMiniMaxResponse(c, statusCode, respHeader, respBody)
}

func (h *AIGatewayHandler) performMiniMaxMusicRequest(apiKey, url string, body []byte, incoming http.Header) ([]byte, int, http.Header, error) {
	return h.performMiniMaxMusicRequestWithClient(h.mediaClient, apiKey, url, body, incoming)
}

// performMiniMaxMusicRequestWithClient 是异步 goroutine 用的版本:可以指定 http.Client,
// 默认走 mediaClient(90s);歌词异步场景传 musicSubmitClient(5 分钟)更稳。
func (h *AIGatewayHandler) performMiniMaxMusicRequestWithClient(client *http.Client, apiKey, url string, body []byte, incoming http.Header) ([]byte, int, http.Header, error) {
	const maxRetries = 3 // 总共 4 次尝试(初始 + 3 次重试),覆盖 Cloudflare 524 这类偶发抖动

	var (
		respBody   []byte
		statusCode int
		respHeader http.Header
		err        error
	)

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(musicProxyRetryBackoff(attempt))
		}

		respBody, statusCode, respHeader, err = h.performMiniMaxMusicRequestOnceWithClient(client, apiKey, url, body, incoming)
		if err == nil && !shouldRetryMusicProxyStatus(statusCode) {
			// 成功 / 4xx 客户端错误:重试无意义,直接返回
			if attempt > 0 && respHeader != nil {
				respHeader = respHeader.Clone()
				respHeader.Set("X-Retry-Count", strconv.Itoa(attempt))
			}
			return respBody, statusCode, respHeader, nil
		}
		// 5xx/524/网络错误 → 进入下一次重试
	}

	// 4 次都失败:最后一次的结果直接返回(让客户端看到原始错误码)
	if respHeader != nil {
		respHeader = respHeader.Clone()
		respHeader.Set("X-Retry-Count", strconv.Itoa(maxRetries))
	}
	return respBody, statusCode, respHeader, err
}

// musicProxyRetryBackoff 返回第 N 次重试前要等待多久。
// 指数 backoff:attempt=1→1s, 2→2s, 3→4s, 4→8s。
// 在基础上加 ±25% 随机抖动,防止多个客户端同时重试导致上游继续 524(雪崩)。
// 用包级 var 是为了测试时能覆盖成 0 加速。
var musicProxyRetryBackoff = func(attempt int) time.Duration {
	if attempt < 1 {
		return 0
	}
	base := time.Duration(1<<(attempt-1)) * time.Second // 1s, 2s, 4s, 8s
	if base <= 0 {
		return 0
	}
	span := int64(base) / 4 // ±25% 抖动区间
	if span == 0 {
		return base
	}
	jitter := rand.Int64N(2*span+1) - span // [-span, +span]
	d := base + time.Duration(jitter)
	if d < 0 {
		return 0
	}
	return d
}

// shouldRetryMusicProxyStatus 判断上游返回的状态码是否值得重试。
// 524 是 Cloudflare 的"origin timeout",会偶发;500/502/503/504 同理。
// 4xx 是请求本身有问题(鉴权/参数错误),重试也不会过。
func shouldRetryMusicProxyStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusInternalServerError,   // 500
		http.StatusBadGateway,            // 502
		http.StatusServiceUnavailable,    // 503
		http.StatusGatewayTimeout,        // 504
		524:                              // Cloudflare origin timeout
		return true
	}
	return false
}

func (h *AIGatewayHandler) performMiniMaxMusicRequestOnce(apiKey, url string, body []byte, incoming http.Header) ([]byte, int, http.Header, error) {
	return h.performMiniMaxMusicRequestOnceWithClient(h.mediaClient, apiKey, url, body, incoming)
}

func (h *AIGatewayHandler) performMiniMaxMusicRequestOnceWithClient(client *http.Client, apiKey, url string, body []byte, incoming http.Header) ([]byte, int, http.Header, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	for key, values := range incoming {
		switch http.CanonicalHeaderKey(key) {
		case "Authorization", "Content-Type", "Content-Length", "Host", "Connection", "Proxy-Connection", "Upgrade", "Keep-Alive", "Te", "Trailer", "Transfer-Encoding",
			"Accept-Encoding": // 不要把浏览器的 Accept-Encoding 透传给上游,否则上游可能返 gzip 而 Content-Encoding 头缺失,Go transport 不会自动解压,前端就拿到二进制乱码
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, nil, err
	}
	// 兜底:部分上游链路(经 CF / 中转代理)可能返回 gzip 体但漏写 Content-Encoding 头,
	// Go transport 看到没 Content-Encoding 就不解压,前端会拿到 0x1f 0x8b 开头的二进制当 JSON 解。
	// 检测 magic bytes 后手动解压,保证下游一定能拿到明文。
	respBody = maybeGunzip(respBody)
	return respBody, resp.StatusCode, resp.Header.Clone(), nil
}

// maybeGunzip 看到 gzip magic 0x1f 0x8b 就解压;否则原样返回。
func maybeGunzip(body []byte) []byte {
	if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
		if r, err := gzip.NewReader(bytes.NewReader(body)); err == nil {
			defer r.Close()
			if decoded, err := io.ReadAll(r); err == nil {
				return decoded
			}
		}
	}
	return body
}
