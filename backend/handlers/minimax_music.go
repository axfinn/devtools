package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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
			{"method": "POST", "path": "/api/minimax/music/v1/lyrics_generation", "description": "歌词生成"},
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
	h.proxyMiniMaxMusicJSON(c, "/v1/lyrics_generation", "lyrics_generation")
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
	const maxRetries = 2 // 总共 3 次尝试(初始 + 2 次重试),覆盖 Cloudflare 524 这类偶发抖动

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

		respBody, statusCode, respHeader, err = h.performMiniMaxMusicRequestOnce(apiKey, url, body, incoming)
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

	// 3 次都失败:最后一次的结果直接返回(让客户端看到原始错误码)
	if respHeader != nil {
		respHeader = respHeader.Clone()
		respHeader.Set("X-Retry-Count", strconv.Itoa(maxRetries))
	}
	return respBody, statusCode, respHeader, err
}

// musicProxyRetryBackoff 返回第 N 次重试前要等待多久。
// attempt 从 1 开始:首次重试 1s,第二次 2s(指数 backoff)。
// 用包级 var 是为了测试时能覆盖成 0 加速。
var musicProxyRetryBackoff = func(attempt int) time.Duration {
	return time.Duration(attempt) * time.Second
}

// shouldRetryMusicProxyStatus 判断上游返回的状态码是否值得重试。
// 524 是 Cloudflare 的"origin timeout",会偶发;502/503/504 同理。
// 4xx 是请求本身有问题(鉴权/参数错误),重试也不会过。
func shouldRetryMusicProxyStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout,      // 504
		524:                            // Cloudflare origin timeout
		return true
	}
	return false
}

func (h *AIGatewayHandler) performMiniMaxMusicRequestOnce(apiKey, url string, body []byte, incoming http.Header) ([]byte, int, http.Header, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	for key, values := range incoming {
		switch http.CanonicalHeaderKey(key) {
		case "Authorization", "Content-Type", "Content-Length", "Host", "Connection", "Proxy-Connection", "Upgrade", "Keep-Alive", "Te", "Trailer", "Transfer-Encoding":
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := h.mediaClient.Do(req)
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
