package handlers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// setupOneClickMusicTest 起一个 httptest 模拟 MiniMax 上游，把 AIGatewayHandler 指向它，配置超管密码鉴权。
// 返回的 upstream hitCount 用于校验代理真的打到了 mock 而不是被短路。
func setupOneClickMusicTest(t *testing.T) (*gin.Engine, *httptest.Server, *atomic.Int32) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/lyrics_generation":
			body, _ := io.ReadAll(r.Body)
			_, _ = w.Write(lyricsResponseFor(body))
		default:
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusBadRequest)
		}
	}))

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	// :memory: + sql.DB 连接池会让并发请求落到不同 in-memory DB，
	// 出现"刚插入的 task 读不到"的诡异 404。把池压到 1 强制串行。
	db.SetMaxOpenConns(1)
	if err := db.InitMiniMaxResultShares(); err != nil {
		t.Fatalf("init result shares: %v", err)
	}
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init media tasks: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-minimax-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.noProxyClient.Timeout = 5 * time.Second
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)
	return router, upstream, &hits
}

// pollLyricsUntilDone 异步歌词任务:拿到 task_id 后轮询 /api/minimax/music/v1/lyrics_tasks/:id,
// 直到 status ∈ {succeeded, failed} 或超时。返回最后一次轮询的响应。
func pollLyricsUntilDone(t *testing.T, router *gin.Engine, taskID, adminPwd string, timeout time.Duration) *httptest.ResponseRecorder {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastWR *httptest.ResponseRecorder
	for time.Now().Before(deadline) {
		req := httptest.NewRequest(http.MethodGet, "/api/minimax/music/v1/lyrics_tasks/"+taskID, nil)
		req.Header.Set("X-Super-Admin-Password", adminPwd)
		wr := httptest.NewRecorder()
		router.ServeHTTP(wr, req)
		lastWR = wr
		if wr.Code != http.StatusOK {
			return wr
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(wr.Body.Bytes(), &resp)
		switch resp["status"] {
		case "succeeded", "failed":
			return wr
		}
		time.Sleep(50 * time.Millisecond)
	}
	return lastWR
}

// submitLyricsAsync 提交异步歌词生成,返回 (status code, task_id, body)。
func submitLyricsAsync(t *testing.T, router *gin.Engine, body []byte, adminPwd string) (int, string, []byte) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader(body))
	req.Header.Set("X-Super-Admin-Password", adminPwd)
	req.Header.Set("Content-Type", "application/json")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	taskID, _ := resp["task_id"].(string)
	return wr.Code, taskID, wr.Body.Bytes()
}

// lyricsResponseFor 造一个同时含 root + nested 两种字段的歌词响应，
// 这样不论前端 read path 走 `lyricsData.lyrics` 还是 `lyricsData.data?.lyrics` 都能拿到。
// 入参 body 必须是 JSON，含 prompt+mode 字段；advancedParams.language 决定返回内容。
func lyricsResponseFor(body []byte) []byte {
	var req map[string]interface{}
	_ = json.Unmarshal(body, &req)
	mode, _ := req["mode"].(string)
	prompt, _ := req["prompt"].(string)
	language := "zh"
	if ap, ok := req["advancedParams"].(map[string]interface{}); ok {
		if l, ok := ap["language"].(string); ok && l != "" {
			language = l
		}
	}
	lyrics := "[Verse]\nGenerated for: " + truncate(prompt, 24) + " (" + mode + ") [" + language + "]"
	songTitle := truncate(prompt, 12) + " (" + language + ")"
	out := map[string]interface{}{
		"lyrics":     lyrics,
		"song_title": songTitle,
		"data": map[string]interface{}{
			"lyrics":     lyrics + " (from data)",
			"song_title": songTitle,
			"title":      songTitle,
		},
		"base_resp": map[string]interface{}{"status_code": 0, "status_msg": "success"},
	}
	b, _ := json.Marshal(out)
	return b
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// TestLyricsProxyPassthroughRootAndNestedShapes 锁定 lyrics 异步任务透传:
// POST 立即返回 202 + task_id;轮询 task 后 result 字段里含 root + nested 两种字段。
// 前端 runOneClickMusic / extractLyricsText 从 task.result 里拿 lyrics。
func TestLyricsProxyPassthroughRootAndNestedShapes(t *testing.T) {
	withZeroRetryBackoff(t)
	router, _, hits := setupOneClickMusicTest(t)

	body, _ := json.Marshal(map[string]interface{}{
		"mode":   "write_full_song",
		"prompt": "我在北疆自驾了一周",
		"advancedParams": map[string]interface{}{
			"language": "zh",
		},
	})

	status, taskID, submitBody := submitLyricsAsync(t, router, body, "test-admin-pw")
	if status != http.StatusAccepted {
		t.Fatalf("submit status = %d, want 202, body = %s", status, submitBody)
	}
	if taskID == "" {
		t.Fatalf("submit missing task_id, body = %s", submitBody)
	}

	wr := pollLyricsUntilDone(t, router, taskID, "test-admin-pw", 5*time.Second)
	if wr.Code != http.StatusOK {
		t.Fatalf("poll status = %d, body = %s", wr.Code, wr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(wr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, _ := resp["status"].(string); got != "succeeded" {
		t.Fatalf("task status = %q, want succeeded, body = %s", got, wr.Body.String())
	}

	result, _ := resp["result"].(map[string]interface{})
	if result == nil {
		t.Fatalf("missing nested result, body = %s", wr.Body.String())
	}
	if got, _ := result["lyrics"].(string); got == "" {
		t.Fatalf("missing root-level lyrics, body = %s", wr.Body.String())
	}
	if got, _ := result["song_title"].(string); got == "" {
		t.Fatalf("missing root-level song_title, body = %s", wr.Body.String())
	}
	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		t.Fatalf("missing nested data, body = %s", wr.Body.String())
	}
	if got, _ := data["lyrics"].(string); got == "" {
		t.Fatalf("missing nested data.lyrics, body = %s", wr.Body.String())
	}
	if got, _ := data["title"].(string); got == "" {
		t.Fatalf("missing nested data.title, body = %s", wr.Body.String())
	}

	if hits.Load() != 1 {
		t.Fatalf("upstream hit count = %d, want 1", hits.Load())
	}
}

// TestLyricsProxyRejectsWithoutAuth 保证鉴权失败时不会打到 upstream。
func TestLyricsProxyRejectsWithoutAuth(t *testing.T) {
	router, _, hits := setupOneClickMusicTest(t)

	body, _ := json.Marshal(map[string]interface{}{"mode": "write_full_song", "prompt": "test"})
	req := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusAccepted || w.Code == http.StatusOK {
		t.Fatalf("expected non-2xx without admin pwd, got %d body=%s", w.Code, w.Body.String())
	}
	if hits.Load() != 0 {
		t.Fatalf("upstream hit count = %d, want 0", hits.Load())
	}
}

// TestOneClickMusicEndToEndYieldsAudioURL 端到端验证一键音乐链路：
//   1. /v1/lyrics_generation 返回歌词
//   2. /v1/music_generation 提交任务（异步） 返回 external task_id
//   3. devtools 后端后台轮询 /v1/tasks/<ext-id>，upstream 返回 success + audio URL
//   4. 前端轮询 GET /api/minimax/token-plan/tasks/:local-id，最终拿到 status=succeeded + result_urls 含音频 URL
func TestOneClickMusicEndToEndYieldsAudioURL(t *testing.T) {
	withZeroRetryBackoff(t)
	gin.SetMode(gin.TestMode)

	const (
		wantAudioURL  = "https://minimax.chat/music/end-to-end-track.mp3"
		wantExtTaskID = "ext_abc123"
	)

	var (
		lyricsHits    atomic.Int32
		submitHits    atomic.Int32
		pollHits      atomic.Int32
		pollResponses atomic.Int32 // 记录已被调用的次数
	)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/lyrics_generation" && r.Method == http.MethodPost:
			lyricsHits.Add(1)
			body, _ := io.ReadAll(r.Body)
			_, _ = w.Write(lyricsResponseFor(body))
		case r.URL.Path == "/v1/music_generation" && r.Method == http.MethodPost:
			submitHits.Add(1)
			// 异步提交：仅返回 external task_id，立即完成
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"task_id": wantExtTaskID,
				"base_resp": map[string]interface{}{
					"status_code": 0,
					"status_msg":  "success",
				},
			})
		case strings.HasPrefix(r.URL.Path, "/v1/tasks/") && r.Method == http.MethodGet:
			n := pollResponses.Add(1)
			pollHits.Add(1)
			// 第 1 次返回 processing，让 backend 后台轮询真的有"轮询"动作
			// 第 2 次起返回 success + audio url
			if n == 1 {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"status":     "processing",
					"base_resp":  map[string]interface{}{"status_code": 0, "status_msg": "processing"},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"audio":     wantAudioURL,
					"status":    2,
					"extra_info": map[string]interface{}{
						"music_duration": 30000,
						"bitrate":        256000,
					},
				},
				"base_resp": map[string]interface{}{"status_code": 0, "status_msg": "success"},
			})
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
		}
	}))
	defer upstream.Close()

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	// 强制只用一条 in-memory 连接，避免 Submit goroutine 和测试主线程拿到不同 DB 视图
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init media tasks: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-minimax-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.noProxyClient.Timeout = 5 * time.Second
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)
	router.POST("/api/minimax/token-plan/v1/generations", h.ProxyMinimaxTokenPlan)
	router.GET("/api/minimax/token-plan/tasks/:id", h.GetMinimaxTokenPlanTask)

	// 模拟前端一键生成第 1 步：异步提交歌词任务 + 轮询拿到结果
	lyricsBody, _ := json.Marshal(map[string]interface{}{
		"mode":   "write_full_song",
		"prompt": "我在北疆自驾了一周",
		"advancedParams": map[string]interface{}{"language": "zh"},
	})
	lyricsStatus, lyricsTaskID, lyricsBody2 := submitLyricsAsync(t, router, lyricsBody, "test-admin-pw")
	if lyricsStatus != http.StatusAccepted {
		t.Fatalf("lyrics submit status = %d, want 202, body = %s", lyricsStatus, lyricsBody2)
	}
	lyricsWR := pollLyricsUntilDone(t, router, lyricsTaskID, "test-admin-pw", 5*time.Second)
	var lyricsRespOuter map[string]interface{}
	_ = json.Unmarshal(lyricsWR.Body.Bytes(), &lyricsRespOuter)
	lyricsResp, _ := lyricsRespOuter["result"].(map[string]interface{})
	if lyricsResp == nil {
		t.Fatalf("lyrics task result empty, body = %s", lyricsWR.Body.String())
	}
	if got, _ := lyricsResp["lyrics"].(string); got == "" {
		t.Fatalf("lyrics empty, body = %s", lyricsWR.Body.String())
	}
	if lyricsHits.Load() != 1 {
		t.Fatalf("lyrics upstream hits = %d, want 1", lyricsHits.Load())
	}

	// 模拟前端一键生成第 2 步：提交音乐任务
	musicBody, _ := json.Marshal(map[string]interface{}{
		"model":         "music-3.0",
		"prompt":        "我在北疆自驾了一周",
		"lyrics":        lyricsResp["lyrics"],
		"output_format": "url",
		"audio_setting": map[string]interface{}{"sample_rate": 44100, "bitrate": 256000, "format": "mp3"},
	})
	musicReq := httptest.NewRequest(http.MethodPost, "/api/minimax/token-plan/v1/generations", bytes.NewReader(musicBody))
	musicReq.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	musicReq.Header.Set("Content-Type", "application/json")
	musicW := httptest.NewRecorder()
	router.ServeHTTP(musicW, musicReq)
	if musicW.Code != http.StatusOK {
		t.Fatalf("submit proxy status = %d, body = %s", musicW.Code, musicW.Body.String())
	}
	var submitResp map[string]interface{}
	_ = json.Unmarshal(musicW.Body.Bytes(), &submitResp)
	localTaskID, _ := submitResp["task_id"].(string)
	if localTaskID == "" {
		t.Fatalf("submit response missing task_id: %s", musicW.Body.String())
	}

	// 模拟前端轮询 GET /api/minimax/token-plan/tasks/<local-id>
	deadline := time.Now().Add(15 * time.Second)
	var final map[string]interface{}
	for time.Now().Before(deadline) {
		pollReq := httptest.NewRequest(http.MethodGet, "/api/minimax/token-plan/tasks/"+localTaskID, nil)
		pollReq.Header.Set("X-Super-Admin-Password", "test-admin-pw")
		pollW := httptest.NewRecorder()
		router.ServeHTTP(pollW, pollReq)
		if pollW.Code != http.StatusOK {
			t.Fatalf("poll status = %d, body = %s", pollW.Code, pollW.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(pollW.Body.Bytes(), &resp)
		if status, _ := resp["status"].(string); status == "succeeded" {
			final = resp
			break
		}
		if status, _ := resp["status"].(string); status == "failed" {
			t.Fatalf("task failed: %s", pollW.Body.String())
		}
		time.Sleep(300 * time.Millisecond)
	}
	if final == nil {
		t.Fatalf("task %s never reached succeeded within 15s; pollHits=%d", localTaskID, pollHits.Load())
	}

	// 验证：result_urls 必须含音频 URL（前端 extractTaskUrls 就靠这个字段）
	resultURLs, _ := final["result_urls"].([]interface{})
	var gotURL string
	for _, u := range resultURLs {
		if s, ok := u.(string); ok {
			gotURL = s
			break
		}
	}
	if gotURL != wantAudioURL {
		t.Fatalf("result_urls[0] = %q, want %q; full task = %v", gotURL, wantAudioURL, final)
	}

	if pollHits.Load() < 2 {
		t.Fatalf("upstream was polled %d times, want >=2 (1 processing + 1 success)", pollHits.Load())
	}
}

// TestOneClickMusicPollingBackoffReturnsFinalURLAfterFailure 验证 backend 后台轮询的 backoff 循环：
//  1. upstream 提交返回 task_id（不同步返回媒体）
//  2. upstream /v1/tasks/<id> 前几次返回 running（还没有音频）
//  3. 最后一次返回 success + audio URL
// 前端最终拿到的是 success 那一帧的 URL（不是更早的 running/空数据）。
func TestOneClickMusicPollingBackoffReturnsFinalURLAfterFailure(t *testing.T) {
	withZeroRetryBackoff(t)
	gin.SetMode(gin.TestMode)

	const wantAudioURL = "https://minimax.chat/music/recovery.mp3"
	var pollCount atomic.Int32

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/lyrics_generation":
			body, _ := io.ReadAll(r.Body)
			_, _ = w.Write(lyricsResponseFor(body))
		case r.URL.Path == "/v1/music_generation":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"task_id":   "ext_recovery",
				"base_resp": map[string]interface{}{"status_code": 0},
			})
		case strings.HasPrefix(r.URL.Path, "/v1/tasks/"):
			n := pollCount.Add(1)
			// 前 3 次都还在 running/尚未就绪，模拟长任务生成过程
			// 第 3 次之后才给出真正的 success + audio URL
			if n < 4 {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"status":    "running",
					"base_resp": map[string]interface{}{"status_code": 0},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"audio": wantAudioURL,
				},
				"base_resp": map[string]interface{}{"status_code": 0},
			})
		default:
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer upstream.Close()

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.noProxyClient.Timeout = 5 * time.Second
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)
	router.POST("/api/minimax/token-plan/v1/generations", h.ProxyMinimaxTokenPlan)
	router.GET("/api/minimax/token-plan/tasks/:id", h.GetMinimaxTokenPlanTask)

	// lyrics 步:异步提交 + 轮询
	lyricsBody, _ := json.Marshal(map[string]interface{}{"mode": "write_full_song", "prompt": "test"})
	lyricsStatus, lyricsTaskID, lyricsBody2 := submitLyricsAsync(t, router, lyricsBody, "test-admin-pw")
	if lyricsStatus != http.StatusAccepted {
		t.Fatalf("lyrics submit: status=%d body=%s", lyricsStatus, lyricsBody2)
	}
	w1 := pollLyricsUntilDone(t, router, lyricsTaskID, "test-admin-pw", 5*time.Second)
	var lyricsRespOuter map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &lyricsRespOuter)
	lyricsResp, _ := lyricsRespOuter["result"].(map[string]interface{})
	if lyricsResp == nil {
		t.Fatalf("lyrics task result empty, body = %s", w1.Body.String())
	}

	// submit
	musicBody, _ := json.Marshal(map[string]interface{}{
		"model": "music-3.0", "prompt": "test",
		"lyrics": lyricsResp["lyrics"], "output_format": "url",
		"audio_setting": map[string]interface{}{"sample_rate": 44100, "bitrate": 256000, "format": "mp3"},
	})
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPost, "/api/minimax/token-plan/v1/generations", bytes.NewReader(musicBody))
	r2.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	r2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("submit: status=%d body=%s", w2.Code, w2.Body.String())
	}
	var sub map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &sub)
	localID, _ := sub["task_id"].(string)
	if localID == "" {
		t.Fatalf("submit empty task_id, body=%s", w2.Body.String())
	}

	// 轮询直到 succeeded 且 result_urls 含目标 audio URL
	var final map[string]interface{}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		w3 := httptest.NewRecorder()
		r3 := httptest.NewRequest(http.MethodGet, "/api/minimax/token-plan/tasks/"+localID, nil)
		r3.Header.Set("X-Super-Admin-Password", "test-admin-pw")
		router.ServeHTTP(w3, r3)
		if w3.Code != http.StatusOK {
			t.Fatalf("poll status %d body=%s", w3.Code, w3.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(w3.Body.Bytes(), &resp)
		if status, _ := resp["status"].(string); status == "succeeded" {
			urls, _ := resp["result_urls"].([]interface{})
			if len(urls) > 0 && urls[0] == wantAudioURL {
				final = resp
				break
			}
			// 第一次 success 没 audio URL，继续等
			if len(urls) == 0 {
				time.Sleep(300 * time.Millisecond)
				continue
			}
		}
		if status, _ := resp["status"].(string); status == "failed" {
			t.Fatalf("task failed: %s", w3.Body.String())
		}
		time.Sleep(300 * time.Millisecond)
	}
	if final == nil {
		t.Fatalf("task %s never returned audio url %s", localID, wantAudioURL)
	}
}

// TestOneClickMusicShareDownloadsAudioAndExposesAsset 验证一键音乐分享链路：
//  1. 拿到了一键音乐生成结果里的 sourceUrl（上游 mp3 地址）
//  2. 前端把 assets=[{url: sourceUrl, kind: "audio"}] 提交到 /api/minimax/result-shares
//  3. 后端真的把 mp3 拉下来存到 data/minimax_result_shares/<id>/
//  4. 返回的 share payload 含 lyrics/theme/model 等上下文
//  5. 文件大小 > 1KB，且 magic bytes 是合法音频（mp3 ID3 头）
func TestOneClickMusicShareDownloadsAudioAndExposesAsset(t *testing.T) {
	withZeroRetryBackoff(t)
	gin.SetMode(gin.TestMode)

	// 切换到临时目录,避免污染 repo 下的 data/minimax_result_shares
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldCwd) })

	// wantAudioURL 用 mock upstream 的实际地址,让 /v1/tasks/... 响应里的 audio URL
	// 跟后端拉资产的 GET 都落到同一个 httptest server(否则会去解析真实域名)
	var wantAudioURL string

	var servedAudio atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/lyrics_generation" && r.Method == http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			_, _ = w.Write(lyricsResponseFor(body))
		case r.URL.Path == "/v1/music_generation" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"task_id":   "ext_share_test",
				"base_resp": map[string]interface{}{"status_code": 0},
			})
		case r.URL.Path == "/v1/tasks/ext_share_test" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"audio": wantAudioURL,
				},
				"base_resp": map[string]interface{}{"status_code": 0},
			})
		case strings.HasSuffix(r.URL.Path, "/music/share-test.mp3") && r.Method == http.MethodGet:
			servedAudio.Store(true)
			// 构造一个最小合法 mp3：ID3 头 + 一些伪 payload 字节
			// 真实 mp3 大小写很多,这里只关心后端能成功 GET 并落盘
			w.Header().Set("Content-Type", "audio/mpeg")
			header := []byte("ID3")
			header = append(header, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // ID3v2.3 size=0
			payload := make([]byte, 4096)
			for i := range payload {
				payload[i] = 0xAB
			}
			_, _ = w.Write(append(header, payload...))
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
		}
	}))
	defer upstream.Close()
	wantAudioURL = upstream.URL + "/music/share-test.mp3"

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init media tasks: %v", err)
	}
	if err := db.InitMiniMaxResultShares(); err != nil {
		t.Fatalf("init result shares: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.noProxyClient.Timeout = 5 * time.Second
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)
	router.POST("/api/minimax/token-plan/v1/generations", h.ProxyMinimaxTokenPlan)
	router.GET("/api/minimax/token-plan/tasks/:id", h.GetMinimaxTokenPlanTask)
	router.POST("/api/minimax/result-shares", h.CreateMiniMaxResultShare)
	router.GET("/api/minimax/result-shares/:id", h.GetMiniMaxResultShare)

	// Step 1: 异步提交歌词任务 + 轮询拿到结果
	lyricsBody, _ := json.Marshal(map[string]interface{}{
		"mode":   "write_full_song",
		"prompt": "我在北疆自驾了一周",
		"advancedParams": map[string]interface{}{"language": "zh"},
	})
	lyricsStatus, lyricsTaskID, lyricsBody2 := submitLyricsAsync(t, router, lyricsBody, "test-admin-pw")
	if lyricsStatus != http.StatusAccepted {
		t.Fatalf("lyrics submit status=%d body=%s", lyricsStatus, lyricsBody2)
	}
	w1 := pollLyricsUntilDone(t, router, lyricsTaskID, "test-admin-pw", 5*time.Second)
	var lyricsRespOuter map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &lyricsRespOuter)
	lyricsResp, _ := lyricsRespOuter["result"].(map[string]interface{})
	if lyricsResp == nil {
		t.Fatalf("lyrics task result empty, body = %s", w1.Body.String())
	}

	// Step 2: 提交并轮询到 succeeded,模拟前端拿到了 sourceUrl
	musicBody, _ := json.Marshal(map[string]interface{}{
		"model":         "music-3.0",
		"prompt":        "我在北疆自驾了一周",
		"lyrics":        lyricsResp["lyrics"],
		"output_format": "url",
		"audio_setting": map[string]interface{}{"sample_rate": 44100, "bitrate": 256000, "format": "mp3"},
	})
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPost, "/api/minimax/token-plan/v1/generations", bytes.NewReader(musicBody))
	r2.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	r2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("submit status=%d body=%s", w2.Code, w2.Body.String())
	}
	var sub map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &sub)
	localID, _ := sub["task_id"].(string)
	if localID == "" {
		t.Fatalf("submit empty task_id body=%s", w2.Body.String())
	}

	// 轮询 devtools 任务,直到 status=succeeded 且 result_urls[0] 是音频 URL
	var sourceAudioURL string
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		w3 := httptest.NewRecorder()
		r3 := httptest.NewRequest(http.MethodGet, "/api/minimax/token-plan/tasks/"+localID, nil)
		r3.Header.Set("X-Super-Admin-Password", "test-admin-pw")
		router.ServeHTTP(w3, r3)
		if w3.Code != http.StatusOK {
			t.Fatalf("poll status=%d body=%s", w3.Code, w3.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(w3.Body.Bytes(), &resp)
		status, _ := resp["status"].(string)
		if status == "succeeded" {
			urls, _ := resp["result_urls"].([]interface{})
			if len(urls) > 0 {
				sourceAudioURL, _ = urls[0].(string)
				break
			}
		}
		if status == "failed" {
			t.Fatalf("task failed: %s", w3.Body.String())
		}
		time.Sleep(300 * time.Millisecond)
	}
	if sourceAudioURL != wantAudioURL {
		t.Fatalf("result_urls[0] = %q, want %q", sourceAudioURL, wantAudioURL)
	}

	// Step 3: 前端组装 share 请求体（和 buildShareDraft case 'oneclick' 对齐）
	shareBody, _ := json.Marshal(map[string]interface{}{
		"title":       "一键音乐 - 我在北疆自驾了一周",
		"summary":     "主题：我在北疆自驾了一周 · 模型：music-3.0",
		"result_type": "audio",
		"model":       "music-3.0",
		"payload": map[string]interface{}{
			"theme":           "我在北疆自驾了一周",
			"language":        "zh",
			"model":           "music-3.0",
			"title":           "我在北疆自驾了一周",
			"lyrics":          lyricsResp["lyrics"],
			"elapsed_sec":     42,
			"source_audio_url": sourceAudioURL,
		},
		"assets": []map[string]interface{}{
			{
				"url":      sourceAudioURL,
				"filename": "我在北疆自驾了一周.mp3",
				"kind":     "audio",
			},
		},
	})
	w4 := httptest.NewRecorder()
	r4 := httptest.NewRequest(http.MethodPost, "/api/minimax/result-shares", bytes.NewReader(shareBody))
	r4.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	r4.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w4, r4)
	if w4.Code != http.StatusCreated {
		t.Fatalf("create share status=%d body=%s", w4.Code, w4.Body.String())
	}
	if !servedAudio.Load() {
		t.Fatal("upstream was never hit for the audio asset")
	}

	var shareResp map[string]interface{}
	if err := json.Unmarshal(w4.Body.Bytes(), &shareResp); err != nil {
		t.Fatalf("decode share resp: %v body=%s", err, w4.Body.String())
	}
	shareID, _ := shareResp["id"].(string)
	if shareID == "" {
		t.Fatalf("share response missing id: %s", w4.Body.String())
	}
	shareURL, _ := shareResp["share_url"].(string)
	if shareURL == "" {
		t.Fatalf("share response missing share_url: %s", w4.Body.String())
	}

	assets, _ := shareResp["assets"].([]interface{})
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d in %s", len(assets), w4.Body.String())
	}
	asset0, _ := assets[0].(map[string]interface{})
	if asset0 == nil {
		t.Fatalf("asset[0] not object: %v", assets)
	}
	assetID, _ := asset0["id"].(string)
	if assetID == "" {
		t.Fatalf("asset[0] missing id: %v", asset0)
	}

	// Step 4: GET share 应能在本地磁盘上找到音频
	shareDir := filepath.Join(tmpDir, "data", "minimax_result_shares", shareID)
	entries, err := os.ReadDir(shareDir)
	if err != nil {
		t.Fatalf("read share dir %s: %v", shareDir, err)
	}
	if len(entries) != 1 {
		t.Fatalf("share dir should have exactly 1 file, got %d (%v)", len(entries), entries)
	}
	filePath := filepath.Join(shareDir, entries[0].Name())
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat %s: %v", filePath, err)
	}
	if info.Size() < 1024 {
		t.Fatalf("audio file too small: %d bytes", info.Size())
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("readfile %s: %v", filePath, err)
	}
	if len(data) >= 3 && string(data[:3]) != "ID3" {
		t.Fatalf("audio file doesn't start with ID3 magic, got %x", data[:16])
	}

	// Step 5: 公开 GET share 应该返回 payload+assets+share_url
	w5 := httptest.NewRecorder()
	r5 := httptest.NewRequest(http.MethodGet, "/api/minimax/result-shares/"+shareID, nil)
	router.ServeHTTP(w5, r5)
	if w5.Code != http.StatusOK {
		t.Fatalf("public get share status=%d body=%s", w5.Code, w5.Body.String())
	}
	var publicShare map[string]interface{}
	_ = json.Unmarshal(w5.Body.Bytes(), &publicShare)
	if publicShare["share_url"] != shareURL {
		t.Fatalf("public share share_url=%v, want %s", publicShare["share_url"], shareURL)
	}
	publicAssets, _ := publicShare["assets"].([]interface{})
	if len(publicAssets) != 1 {
		t.Fatalf("public share expected 1 asset, got %d", len(publicAssets))
	}
	publicAsset0, _ := publicAssets[0].(map[string]interface{})
	if id, _ := publicAsset0["id"].(string); id != assetID {
		t.Fatalf("public asset id=%v, want %s", publicAsset0["id"], assetID)
	}
}

// TestLyricsProxyRetriesOn524ThenSucceeds 验证歌词接口偶发 524 (Cloudflare origin timeout)
// 会被后端自动重试:前 3 次返回 5xx,第 4 次返回 200,异步任务最终 status=succeeded。
func TestLyricsProxyRetriesOn524ThenSucceeds(t *testing.T) {
	withZeroRetryBackoff(t)

	gin.SetMode(gin.TestMode)

	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 4 {
			http.Error(w, "upstream busy", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(lyricsResponseFor(body))
	}))
	defer upstream.Close()

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init media tasks: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	status, taskID, submitBody := submitLyricsAsync(t, router, []byte(`{"mode":"write_full_song","prompt":"test"}`), "test-admin-pw")
	if status != http.StatusAccepted {
		t.Fatalf("submit status = %d, want 202, body = %s", status, submitBody)
	}

	wr := pollLyricsUntilDone(t, router, taskID, "test-admin-pw", 5*time.Second)
	if wr.Code != http.StatusOK {
		t.Fatalf("poll status = %d, body = %s", wr.Code, wr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(wr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, wr.Body.String())
	}
	if got, _ := resp["status"].(string); got != "succeeded" {
		t.Fatalf("task status = %q, want succeeded, body=%s", got, wr.Body.String())
	}
	result, _ := resp["result"].(map[string]interface{})
	if result == nil {
		t.Fatalf("missing result, body=%s", wr.Body.String())
	}
	if got, _ := result["lyrics"].(string); got == "" {
		t.Fatalf("missing lyrics, body=%s", wr.Body.String())
	}
	if got := hits.Load(); got != 4 {
		t.Fatalf("upstream hits = %d, want 4 (3 retries + 1 success)", got)
	}
}

// TestLyricsProxyDoesNotRetryOn4xx 验证 4xx (客户端错误,如 401/400) 不会被重试:
// 鉴权或参数错误重试也没用,白白浪费请求。异步任务最终 status=failed,error 字段带上游错误。
func TestLyricsProxyDoesNotRetryOn4xx(t *testing.T) {
	withZeroRetryBackoff(t)

	gin.SetMode(gin.TestMode)

	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
	}))
	defer upstream.Close()

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second
	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	status, taskID, _ := submitLyricsAsync(t, router, []byte(`{"mode":"write_full_song","prompt":"test"}`), "test-admin-pw")
	if status != http.StatusAccepted {
		t.Fatalf("submit status = %d, want 202", status)
	}

	wr := pollLyricsUntilDone(t, router, taskID, "test-admin-pw", 5*time.Second)
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	if got, _ := resp["status"].(string); got != "failed" {
		t.Fatalf("task status = %q, want failed (4xx 不重试), body=%s", got, wr.Body.String())
	}
	errStr, _ := resp["error"].(string)
	if !strings.Contains(errStr, "401") {
		t.Fatalf("error 字段应该含 401 状态码, got %q", errStr)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("upstream hits = %d, want 1 (no retry on 4xx)", got)
	}
}

// TestLyricsProxyGivesUpAfterMaxRetries 验证 4 次尝试全部失败时,
// 异步任务最终 status=failed,error 字段含上游最后一次的错误信息(让前端知道是上游超时)。
func TestLyricsProxyGivesUpAfterMaxRetries(t *testing.T) {
	withZeroRetryBackoff(t)

	gin.SetMode(gin.TestMode)

	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "persistent 524", 524)
	}))
	defer upstream.Close()

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second
	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	status, taskID, _ := submitLyricsAsync(t, router, []byte(`{"mode":"write_full_song","prompt":"test"}`), "test-admin-pw")
	if status != http.StatusAccepted {
		t.Fatalf("submit status = %d, want 202", status)
	}

	wr := pollLyricsUntilDone(t, router, taskID, "test-admin-pw", 5*time.Second)
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	if got, _ := resp["status"].(string); got != "failed" {
		t.Fatalf("task status = %q, want failed, body=%s", got, wr.Body.String())
	}
	if got := hits.Load(); got != 4 {
		t.Fatalf("upstream hits = %d, want 4 (initial + 3 retries)", got)
	}
}

// TestLyricsProxyRetriesOnNetworkError 验证底层网络错误(timeout/connection reset)
// 也会触发重试,而不是直接把"EOF/timeout"吐给前端。异步任务最终 succeeded。
func TestLyricsProxyRetriesOnNetworkError(t *testing.T) {
	withZeroRetryBackoff(t)

	gin.SetMode(gin.TestMode)

	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 2 {
			// 模拟上游连接挂掉:直接关连接不发响应,触发 connection reset / EOF
			hj, _ := w.(http.Hijacker)
			if hj != nil {
				conn, _, err := hj.Hijack()
				if err == nil {
					_ = conn.Close()
				}
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(lyricsResponseFor(body))
	}))
	defer upstream.Close()

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second
	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	status, taskID, _ := submitLyricsAsync(t, router, []byte(`{"mode":"write_full_song","prompt":"test"}`), "test-admin-pw")
	if status != http.StatusAccepted {
		t.Fatalf("submit status = %d, want 202", status)
	}

	wr := pollLyricsUntilDone(t, router, taskID, "test-admin-pw", 5*time.Second)
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	if got, _ := resp["status"].(string); got != "succeeded" {
		t.Fatalf("task status = %q, want succeeded, body=%s", got, wr.Body.String())
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("upstream hits = %d, want 2 (1 network error + 1 success)", got)
	}
}

// withZeroRetryBackoff 把 musicProxyRetryBackoff 临时设成 0,加速重试测试。
// 默认是 1s/2s,3 次失败要等 3s,对单测不友好。
func withZeroRetryBackoff(t *testing.T) {
	t.Helper()
	orig := musicProxyRetryBackoff
	musicProxyRetryBackoff = func(attempt int) time.Duration { return 0 }
	t.Cleanup(func() { musicProxyRetryBackoff = orig })
}

// TestLyricsAsyncAcceptsEmptyFromUpstream 后端不画蛇添足判定歌词空:上游 base_resp.status_code=0
// 就当 succeeded,把空 lyrics 留给前端 extractLyricsText 报"歌词为空,请重试"。
// 这是 2026-08-04 线上炸了的修法:之前后端强制判定空就 retry 3 次再 failed,反而把整个歌曲生成流程打断。
func TestLyricsAsyncAcceptsEmptyFromUpstream(t *testing.T) {
	withZeroRetryBackoff(t)
	gin.SetMode(gin.TestMode)

	var totalHits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		// 上游 200 + 空歌词:后端要直传,不能误判 failed
		_, _ = w.Write([]byte(`{"lyrics":"","song_title":"","base_resp":{"status_code":0,"status_msg":"success"}}`))
	}))
	t.Cleanup(upstream.Close)

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init media tasks: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-minimax-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.musicSubmitClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	body, _ := json.Marshal(map[string]interface{}{
		"mode":   "write_full_song",
		"prompt": "上游返回空歌词的回归测试",
	})
	code, taskID, _ := submitLyricsAsync(t, router, body, "test-admin-pw")
	if code != http.StatusAccepted {
		t.Fatalf("submit failed: status=%d", code)
	}
	wr := pollLyricsUntilDone(t, router, taskID, "test-admin-pw", 5*time.Second)
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	if got, _ := resp["status"].(string); got != "succeeded" {
		t.Fatalf("status=%q want succeeded (后端不该自己判空 failed), error=%v", got, resp["error"])
	}
	// 必须只打上游一次:不再做"空内容再重试 N 次"的怪动作
	if got := totalHits.Load(); got != 1 {
		t.Fatalf("upstream hits = %d, want 1 (不该再 retry)", got)
	}
	// result 透传,lyrics 字段空字符串照常返回给前端
	result, _ := resp["result"].(map[string]interface{})
	if result == nil {
		t.Fatalf("result 字段缺失: %+v", resp)
	}
	if _, ok := result["base_resp"]; !ok {
		t.Fatalf("result 缺 base_resp,没透传上游响应: %+v", result)
	}
}

// TestLyricsProxyDecompressesGzipBodyWithoutHeader 兜底测试:
// 上游漏写 Content-Encoding 头但 body 是真 gzip(0x1f 0x8b magic),后端必须按 magic 解压,
// 否则前端拿到二进制就报"歌词生成成功但内容为空"。
func TestLyricsProxyDecompressesGzipBodyWithoutHeader(t *testing.T) {
	withZeroRetryBackoff(t)
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		plain := lyricsResponseFor(body)
		// 注意:故意不写 Content-Encoding,模拟"CF 漏标头"的真实故障
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, _ = gz.Write(plain)
		_ = gz.Close()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(buf.Bytes())
	}))
	t.Cleanup(upstream.Close)

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init media tasks: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-minimax-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL
	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.musicSubmitClient.Timeout = 5 * time.Second
	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	body, _ := json.Marshal(map[string]interface{}{"mode": "write_full_song", "prompt": "测试 gzip magic 兜底解压"})
	code, taskID, _ := submitLyricsAsync(t, router, body, "test-admin-pw")
	if code != http.StatusAccepted {
		t.Fatalf("submit status=%d", code)
	}
	wr := pollLyricsUntilDone(t, router, taskID, "test-admin-pw", 5*time.Second)
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	if got, _ := resp["status"].(string); got != "succeeded" {
		t.Fatalf("status=%q (上游返 gzip 但没 Content-Encoding 头,后端没解压), error=%v", got, resp["error"])
	}
	// 关键:result.lyrics 必须有内容,不是空字符串(那意味着还是二进制)
	result, _ := resp["result"].(map[string]interface{})
	lyrics, _ := result["lyrics"].(string)
	if strings.TrimSpace(lyrics) == "" {
		t.Fatalf("result.lyrics 为空,后端没解压 gzip,前端会报'歌词生成成功但内容为空', body=%s", wr.Body.String())
	}
}

// TestMaybeGunzip 单元测一下兜底解压函数
func TestMaybeGunzip(t *testing.T) {
	plain := []byte(`{"lyrics":"测试歌词","base_resp":{"status_code":0}}`)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write(plain)
	_ = gz.Close()
	gzipped := buf.Bytes()
	if got := maybeGunzip(gzipped); string(got) != string(plain) {
		t.Fatalf("maybeGunzip(gzip) = %q, want %q", got, plain)
	}
	// 不是 gzip magic 的原样返回
	notGz := []byte(`{"lyrics":"明文 JSON"}`)
	if got := maybeGunzip(notGz); string(got) != string(notGz) {
		t.Fatalf("maybeGunzip(plain) = %q, want %q", got, notGz)
	}
	// 空 body 也安全
	if got := maybeGunzip(nil); got != nil {
		t.Fatalf("maybeGunzip(nil) = %q, want nil", got)
	}
}

// TestLyricsProxySucceedsOnLastAttemptAfter524s 压测边界:连续 3 次 524 后第 4 次才成功。
// 这是 Cloudflare 偶发抖动最严重的场景,验证重试次数够用。异步任务最终 succeeded。
func TestLyricsProxySucceedsOnLastAttemptAfter524s(t *testing.T) {
	withZeroRetryBackoff(t)

	gin.SetMode(gin.TestMode)

	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 4 {
			http.Error(w, "cloudflare origin timeout", 524)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(lyricsResponseFor(body))
	}))
	defer upstream.Close()

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	status, taskID, _ := submitLyricsAsync(t, router, []byte(`{"mode":"write_full_song","prompt":"jitter-test"}`), "test-admin-pw")
	if status != http.StatusAccepted {
		t.Fatalf("submit status = %d, want 202", status)
	}

	wr := pollLyricsUntilDone(t, router, taskID, "test-admin-pw", 5*time.Second)
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	if got, _ := resp["status"].(string); got != "succeeded" {
		t.Fatalf("task status = %q, want succeeded, body=%s", got, wr.Body.String())
	}
	if got := hits.Load(); got != 4 {
		t.Fatalf("upstream hits = %d, want 4 (3×524 + 1×200)", got)
	}
}

// TestLyricsProxyRetriesOn500ThenSucceeds 验证 500 Internal Server Error 也会触发重试:
// 上游服务过载时 500 跟 524 本质一样,只是 CF 没接住;重试常常能过。异步任务最终 succeeded。
func TestLyricsProxyRetriesOn500ThenSucceeds(t *testing.T) {
	withZeroRetryBackoff(t)

	gin.SetMode(gin.TestMode)

	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			http.Error(w, `{"error":"upstream overloaded"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(lyricsResponseFor(body))
	}))
	defer upstream.Close()

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	status, taskID, _ := submitLyricsAsync(t, router, []byte(`{"mode":"write_full_song","prompt":"500-retry"}`), "test-admin-pw")
	if status != http.StatusAccepted {
		t.Fatalf("submit status = %d, want 202", status)
	}

	wr := pollLyricsUntilDone(t, router, taskID, "test-admin-pw", 5*time.Second)
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	if got, _ := resp["status"].(string); got != "succeeded" {
		t.Fatalf("task status = %q, want succeeded, body=%s", got, wr.Body.String())
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("upstream hits = %d, want 3 (2×500 + 1×200)", got)
	}
}

// TestLyricsProxyRetriesOnMixedTransientFailures 压测混合失败:502→503→524→200。
// 模拟上游不同节点/不同原因轮流报错的真实场景,确认任何一种 5xx 都能触发重试。
// 异步任务最终 succeeded。
func TestLyricsProxyRetriesOnMixedTransientFailures(t *testing.T) {
	withZeroRetryBackoff(t)

	gin.SetMode(gin.TestMode)

	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		switch n {
		case 1:
			http.Error(w, "bad gateway", http.StatusBadGateway)
		case 2:
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		case 3:
			http.Error(w, "cf origin timeout", 524)
		default:
			w.Header().Set("Content-Type", "application/json")
			body, _ := io.ReadAll(r.Body)
			_, _ = w.Write(lyricsResponseFor(body))
		}
	}))
	defer upstream.Close()

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	status, taskID, _ := submitLyricsAsync(t, router, []byte(`{"mode":"write_full_song","prompt":"mixed"}`), "test-admin-pw")
	if status != http.StatusAccepted {
		t.Fatalf("submit status = %d, want 202", status)
	}

	wr := pollLyricsUntilDone(t, router, taskID, "test-admin-pw", 5*time.Second)
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	if got, _ := resp["status"].(string); got != "succeeded" {
		t.Fatalf("task status = %q, want succeeded, body=%s", got, wr.Body.String())
	}
	if got := hits.Load(); got != 4 {
		t.Fatalf("upstream hits = %d, want 4 (502+503+524+200)", got)
	}
}

// TestMusicProxyRetryBackoffHasJitter 验证 musicProxyRetryBackoff 在每档基础值上叠加抖动:
//  1. base 必须是指数(1s, 2s, 4s, 8s),确保重试间隔够长
//  2. 多次调用同一 attempt 得到的值不完全相等(抖动生效)
//  3. 抖动幅度不超过 ±25%
func TestMusicProxyRetryBackoffHasJitter(t *testing.T) {
	orig := musicProxyRetryBackoff
	t.Cleanup(func() { musicProxyRetryBackoff = orig })

	// 用 200ms 基数让抖动幅度计算稳定可读
	musicProxyRetryBackoff = func(attempt int) time.Duration {
		if attempt < 1 {
			return 0
		}
		base := time.Duration(200*(1<<(attempt-1))) * time.Millisecond
		if base <= 0 {
			return 0
		}
		span := int64(base) / 4
		if span == 0 {
			return base
		}
		jitter := rand.Int64N(2*span+1) - span
		d := base + time.Duration(jitter)
		if d < 0 {
			return 0
		}
		return d
	}

	// 1) 指数增长
	wantBase := []time.Duration{
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1600 * time.Millisecond,
	}
	for attempt, want := range wantBase {
		attempt++ // 1-based
		got := musicProxyRetryBackoff(attempt)
		// 抖动范围 ±25%,容差给到 ±30% 防止边界抖动把测试搞挂
		lo := time.Duration(float64(want) * 0.70)
		hi := time.Duration(float64(want) * 1.30)
		if got < lo || got > hi {
			t.Fatalf("attempt=%d base=%v got=%v, want within [%v, %v]", attempt, want, got, lo, hi)
		}
	}

	// 2) 同 attempt 多次调用应该不全相等(抖动真的生效了)
	seen := map[time.Duration]struct{}{}
	for i := 0; i < 50; i++ {
		seen[musicProxyRetryBackoff(2)] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("expected jitter to produce varying values, got %d unique values across 50 calls", len(seen))
	}
}

// TestLyricsProxyRealisticLoadExhaustsThenSucceeds 模拟"用户连点 5 次生成,前几次都 524"的真实场景:
// 第一个异步任务 4 次尝试全部 524 → failed;
// 第二个异步任务再 524 一次后第 7 次请求命中 200 → succeeded。
// 这覆盖了实际生产中观察到的"524 一波一波来"的模式。
func TestLyricsProxyRealisticLoadExhaustsThenSucceeds(t *testing.T) {
	withZeroRetryBackoff(t)

	gin.SetMode(gin.TestMode)

	var hitCount atomic.Int32

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits := hitCount.Add(1)
		// 第 1-4 次返回 524(第一次请求),第 5-6 次返回 524(第二次请求),
		// 第 7 次起返回 200
		if hits <= 6 {
			http.Error(w, "cf 524 burst", 524)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(lyricsResponseFor(body))
	}))
	defer upstream.Close()

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	h.musicSubmitClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	// 第一次:4 次尝试全部 524
	_, taskID1, _ := submitLyricsAsync(t, router, []byte(`{"mode":"write_full_song","prompt":"burst-1"}`), "test-admin-pw")
	wr1 := pollLyricsUntilDone(t, router, taskID1, "test-admin-pw", 5*time.Second)
	var r1 map[string]interface{}
	_ = json.Unmarshal(wr1.Body.Bytes(), &r1)
	if got, _ := r1["status"].(string); got != "failed" {
		t.Fatalf("first burst: expected failed after exhausting retries, got %q body=%s", got, wr1.Body.String())
	}

	// 第二次:再 524 一次后第 7 次请求命中 200
	_, taskID2, _ := submitLyricsAsync(t, router, []byte(`{"mode":"write_full_song","prompt":"burst-2"}`), "test-admin-pw")
	wr2 := pollLyricsUntilDone(t, router, taskID2, "test-admin-pw", 5*time.Second)
	var r2 map[string]interface{}
	_ = json.Unmarshal(wr2.Body.Bytes(), &r2)
	if got, _ := r2["status"].(string); got != "succeeded" {
		t.Fatalf("second burst: expected succeeded, got %q body=%s", got, wr2.Body.String())
	}

	total := hitCount.Load()
	if total != 7 {
		t.Fatalf("total upstream hits = %d, want 7 (4+1+1+1)", total)
	}
}

func TestExtractMediaURLsSupportsMusicPollShape(t *testing.T) {
	t.Run("data.audio as url", func(t *testing.T) {
		urls := extractMediaURLs(map[string]interface{}{
			"data": map[string]interface{}{
				"audio": "https://minimax.chat/music/abc123.mp3",
				"extra_info": map[string]interface{}{
					"music_duration": 30000,
					"bitrate":        256000,
				},
			},
			"base_resp": map[string]interface{}{"status_code": 0},
		})
		if len(urls) != 1 || urls[0] != "https://minimax.chat/music/abc123.mp3" {
			t.Fatalf("expected audio url, got %v", urls)
		}
	})
	t.Run("nested file.download_url", func(t *testing.T) {
		urls := extractMediaURLs(map[string]interface{}{
			"file_id": "file_123",
			"file": map[string]interface{}{
				"download_url": "https://cdn.example.com/track.mp3",
			},
		})
		if len(urls) != 1 || urls[0] != "https://cdn.example.com/track.mp3" {
			t.Fatalf("expected download_url, got %v", urls)
		}
	})
	t.Run("empty when only hex audio (no URL)", func(t *testing.T) {
		urls := extractMediaURLs(map[string]interface{}{
			"data": map[string]interface{}{
				"audio": "ffd8ffe000104a464946",
			},
		})
		if len(urls) != 0 {
			t.Fatalf("expected zero URLs for hex audio, got %v", urls)
		}
	})
}
