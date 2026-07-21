package handlers

import (
	"bytes"
	"encoding/json"
	"io"
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

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-minimax-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.noProxyClient.Timeout = 5 * time.Second
	h.mediaClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	return router, upstream, &hits
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

// TestLyricsProxyPassthroughRootAndNestedShapes 锁定 /api/minimax/music/v1/lyrics_generation 代理透传：
// upstream 同时返回 root 和 nested 字段时，响应里都还在，让前端 extractLyricsText / runOneClickMusic 都能解析。
func TestLyricsProxyPassthroughRootAndNestedShapes(t *testing.T) {
	router, _, hits := setupOneClickMusicTest(t)

	body, _ := json.Marshal(map[string]interface{}{
		"mode":   "write_full_song",
		"prompt": "我在北疆自驾了一周",
		"advancedParams": map[string]interface{}{
			"language": "zh",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader(body))
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got, _ := resp["lyrics"].(string); got == "" {
		t.Fatalf("missing root-level lyrics, body = %s", w.Body.String())
	}
	if got, _ := resp["song_title"].(string); got == "" {
		t.Fatalf("missing root-level song_title, body = %s", w.Body.String())
	}
	data, _ := resp["data"].(map[string]interface{})
	if data == nil {
		t.Fatalf("missing nested data, body = %s", w.Body.String())
	}
	if got, _ := data["lyrics"].(string); got == "" {
		t.Fatalf("missing nested data.lyrics, body = %s", w.Body.String())
	}
	if got, _ := data["title"].(string); got == "" {
		t.Fatalf("missing nested data.title, body = %s", w.Body.String())
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

	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200 without admin pwd, got %d body=%s", w.Code, w.Body.String())
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
	router.POST("/api/minimax/token-plan/v1/generations", h.ProxyMinimaxTokenPlan)
	router.GET("/api/minimax/token-plan/tasks/:id", h.GetMinimaxTokenPlanTask)

	// 模拟前端一键生成第 1 步：拿歌词
	lyricsBody, _ := json.Marshal(map[string]interface{}{
		"mode":   "write_full_song",
		"prompt": "我在北疆自驾了一周",
		"advancedParams": map[string]interface{}{"language": "zh"},
	})
	lyricsReq := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader(lyricsBody))
	lyricsReq.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	lyricsReq.Header.Set("Content-Type", "application/json")
	lyricsW := httptest.NewRecorder()
	router.ServeHTTP(lyricsW, lyricsReq)
	if lyricsW.Code != http.StatusOK {
		t.Fatalf("lyrics proxy status = %d, body = %s", lyricsW.Code, lyricsW.Body.String())
	}
	var lyricsResp map[string]interface{}
	_ = json.Unmarshal(lyricsW.Body.Bytes(), &lyricsResp)
	if got, _ := lyricsResp["lyrics"].(string); got == "" {
		t.Fatalf("lyrics empty, body = %s", lyricsW.Body.String())
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
	router.POST("/api/minimax/token-plan/v1/generations", h.ProxyMinimaxTokenPlan)
	router.GET("/api/minimax/token-plan/tasks/:id", h.GetMinimaxTokenPlanTask)

	// lyrics 步
	lyricsBody, _ := json.Marshal(map[string]interface{}{"mode": "write_full_song", "prompt": "test"})
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader(lyricsBody))
	r1.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	r1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("lyrics: %s", w1.Body.String())
	}
	var lyricsResp map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &lyricsResp)

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
	router.POST("/api/minimax/token-plan/v1/generations", h.ProxyMinimaxTokenPlan)
	router.GET("/api/minimax/token-plan/tasks/:id", h.GetMinimaxTokenPlanTask)
	router.POST("/api/minimax/result-shares", h.CreateMiniMaxResultShare)
	router.GET("/api/minimax/result-shares/:id", h.GetMiniMaxResultShare)

	// Step 1: 拿歌词
	lyricsBody, _ := json.Marshal(map[string]interface{}{
		"mode":   "write_full_song",
		"prompt": "我在北疆自驾了一周",
		"advancedParams": map[string]interface{}{"language": "zh"},
	})
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader(lyricsBody))
	r1.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	r1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("lyrics proxy status=%d body=%s", w1.Code, w1.Body.String())
	}
	var lyricsResp map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &lyricsResp)

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
// 会被后端自动重试:前两次返回 524,第三次返回 200,前端只看到一次成功响应。
func TestLyricsProxyRetriesOn524ThenSucceeds(t *testing.T) {
	withZeroRetryBackoff(t)

	gin.SetMode(gin.TestMode)

	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
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
	t.Cleanup(func() { db.Close() })

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)

	req := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader([]byte(`{"mode":"write_full_song","prompt":"test"}`)))
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after retries, got %d body=%s", w.Code, w.Body.String())
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("upstream hits = %d, want 3 (2 retries + 1 success)", got)
	}
	if rc := w.Header().Get("X-Retry-Count"); rc != "2" {
		t.Fatalf("X-Retry-Count = %q, want \"2\"", rc)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if got, _ := resp["lyrics"].(string); got == "" {
		t.Fatalf("missing lyrics, body=%s", w.Body.String())
	}
}

// TestLyricsProxyDoesNotRetryOn4xx 验证 4xx (客户端错误,如 401/400) 不会被重试:
// 鉴权或参数错误重试也没用,白白浪费请求。
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
	t.Cleanup(func() { db.Close() })

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)

	req := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader([]byte(`{"mode":"write_full_song","prompt":"test"}`)))
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 passthrough, got %d body=%s", w.Code, w.Body.String())
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("upstream hits = %d, want 1 (no retry on 4xx)", got)
	}
	if rc := w.Header().Get("X-Retry-Count"); rc != "" {
		t.Fatalf("X-Retry-Count should be empty on 4xx, got %q", rc)
	}
}

// TestLyricsProxyGivesUpAfterMaxRetries 验证 3 次尝试全部失败时,
// 返回最后一次的 5xx 状态(让前端知道是上游超时,而不是显示成"未知错误")。
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
	t.Cleanup(func() { db.Close() })

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)

	req := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader([]byte(`{"mode":"write_full_song","prompt":"test"}`)))
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 524 {
		t.Fatalf("expected 524 after exhausting retries, got %d body=%s", w.Code, w.Body.String())
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("upstream hits = %d, want 3 (initial + 2 retries)", got)
	}
	if rc := w.Header().Get("X-Retry-Count"); rc != "2" {
		t.Fatalf("X-Retry-Count = %q, want \"2\"", rc)
	}
}

// TestLyricsProxyRetriesOnNetworkError 验证底层网络错误(timeout/connection reset)
// 也会触发重试,而不是直接把"EOF/timeout"吐给前端。
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
	t.Cleanup(func() { db.Close() })

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = "test-key"
	cfg.MiniMaxTokenPlan.BaseURL = upstream.URL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 5 * time.Second
	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)

	req := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader([]byte(`{"mode":"write_full_song","prompt":"test"}`)))
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after network-error retry, got %d body=%s", w.Code, w.Body.String())
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
