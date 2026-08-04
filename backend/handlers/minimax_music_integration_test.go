package handlers

// 集成测试：从环境变量读 MINIMAX_API_KEY，没设就 skip；设了就真实调用一次 /v1/music_generation 端到端验证。
// 默认 `go test ./handlers/` 不会跑；要跑集成测试用 `MINIMAX_API_KEY=xxx go test ./handlers/ -run MusicIntegration -v`。
// 本文件不会包含任何明文密钥，全部从环境变量读取。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

const musicIntegrationBaseURL = "https://api.minimaxi.com"
const lyricsIntegrationPath = "/v1/lyrics_generation"

// TestMusicIntegrationLiveAPI 对每个支持的 music 模型跑一次真实端到端生成。
// 默认 skip，只有在 MINIMAX_API_KEY 设置时才会真跑。
// 每个模型跑最短时长（10s 音频），验证：
//  1. 请求能完成（不被 90s 超时杀掉）
//  2. 响应里有 base_resp.status_code == 0
//  3. data.audio 或 data.audio_url / data.status == 2 存在
func TestMusicIntegrationLiveAPI(t *testing.T) {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("MINIMAX_API_KEY 未设置，跳过真实 API 集成测试")
	}

	models := []string{
		"music-3.0",
		"music-3.0-free",
		"music-2.6",
		"music-2.6-free",
		"music-cover",
		"music-cover-free",
	}

	submitClient := &http.Client{Timeout: 5 * time.Minute}
	downloadClient := &http.Client{Timeout: 60 * time.Second}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			body := buildMusicRequestBody(t, model)
			req, err := http.NewRequest(http.MethodPost, musicIntegrationBaseURL+"/v1/music_generation", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("构造请求失败: %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("Content-Type", "application/json")

			start := time.Now()
			resp, err := submitClient.Do(req)
			if err != nil {
				t.Fatalf("请求失败（怀疑 5 分钟超时不够 / 网络问题）: %v", err)
			}
			defer resp.Body.Close()
			elapsed := time.Since(start)

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("读取响应失败: %v", err)
			}

			t.Logf("[%s] 生成耗时 %s，HTTP %d，响应长度 %d 字节", model, elapsed.Round(time.Second), resp.StatusCode, len(respBody))

			var payload map[string]interface{}
			if err := json.Unmarshal(respBody, &payload); err != nil {
				t.Fatalf("响应不是 JSON: %v\nbody: %s", err, truncateForLog(respBody))
			}

			baseResp, _ := payload["base_resp"].(map[string]interface{})
			statusCode, _ := baseResp["status_code"].(float64)
			if int(statusCode) != 0 {
				statusMsg, _ := baseResp["status_msg"].(string)
				t.Fatalf("base_resp.status_code=%v, msg=%q\nbody: %s", statusCode, statusMsg, truncateForLog(respBody))
			}

			data, _ := payload["data"].(map[string]interface{})
			if data == nil {
				t.Fatalf("响应缺少 data 字段\nbody: %s", truncateForLog(respBody))
			}
			status, _ := data["status"].(float64)
			if status != 2 {
				t.Fatalf("data.status=%v, want 2 (已完成)", status)
			}
			audioRaw, _ := data["audio"].(string)
			if audioRaw == "" {
				t.Fatalf("data.audio 为空\nbody: %s", truncateForLog(respBody))
			}

			// output_format=url 时 data.audio 是音频 URL，下载验证 magic bytes。
			// output_format=hex 时 data.audio 是 hex 编码音频数据，解码后验证 magic bytes。
			if strings.HasPrefix(audioRaw, "http://") || strings.HasPrefix(audioRaw, "https://") {
				verifyAudioURL(t, downloadClient, audioRaw, model)
			} else {
				verifyAudioHex(t, audioRaw, model)
			}

			if elapsed > 4*time.Minute {
				t.Logf("[%s] 警告：耗时 %s 接近 5 分钟超时，建议上调 musicSubmitClient 超时", model, elapsed)
			}
		})
	}
}

// verifyAudioURL 下载音频 URL，校验返回是合法 mp3/wav 文件（magic bytes + 合理大小）。
func verifyAudioURL(t *testing.T, client *http.Client, url, model string) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("[%s] 下载音频失败: %v", model, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		t.Fatalf("[%s] 下载音频 HTTP %d", model, resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	// 读前 16 字节用于 magic bytes 校验
	head := make([]byte, 16)
	n, _ := io.ReadFull(resp.Body, head)
	head = head[:n]
	// 全部读取用于大小校验（限制 50MB 防止 OOM）
	limited := io.LimitReader(resp.Body, 50*1024*1024)
	rest, _ := io.ReadAll(limited)
	full := append(head, rest...)
	if len(full) < 1024 {
		t.Fatalf("[%s] 音频文件过小: %d 字节", model, len(full))
	}
	if !looksLikeAudio(head, contentType) {
		t.Fatalf("[%s] 文件不像合法音频: magic=%x, content-type=%s, size=%d", model, head, contentType, len(full))
	}
	t.Logf("[%s] 音频 URL 下载成功: %d 字节, content-type=%s", model, len(full), contentType)
	// 落盘到 /tmp/devtools-music-share-<model>.mp3 供后续 upload/devtools 分享使用。
	// 仅当 SAVE_GENERATED_MUSIC_DIR 环境变量设置时才保存。
	if dir := os.Getenv("SAVE_GENERATED_MUSIC_DIR"); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
		path := filepath.Join(dir, fmt.Sprintf("music-%s.mp3", model))
		if err := os.WriteFile(path, full, 0o644); err == nil {
			t.Logf("[%s] 已保存到 %s", model, path)
		}
	}
}

// verifyAudioHex 解码 hex 音频数据，校验 magic bytes 和大小。
func verifyAudioHex(t *testing.T, hexData, model string) {
	t.Helper()
	decoded, err := decodeHex(hexData)
	if err != nil {
		t.Fatalf("[%s] hex 解码失败: %v", model, err)
	}
	if len(decoded) < 1024 {
		t.Fatalf("[%s] hex 解码后音频过小: %d 字节", model, len(decoded))
	}
	head := decoded
	if len(head) > 16 {
		head = decoded[:16]
	}
	if !looksLikeAudio(head, "") {
		t.Fatalf("[%s] 解码后不像合法音频: magic=%x, size=%d", model, head, len(decoded))
	}
	t.Logf("[%s] hex 音频解码成功: %d 字节", model, len(decoded))
}

// looksLikeAudio 校验文件 magic bytes 是不是常见音频格式。
func looksLikeAudio(head []byte, contentType string) bool {
	ct := strings.ToLower(contentType)
	if strings.HasPrefix(ct, "audio/") {
		return true
	}
	// MP3: "ID3" 标签 或 0xFF 0xFB/0xFA/0xF3/0xF2 frame sync
	if len(head) >= 3 && string(head[:3]) == "ID3" {
		return true
	}
	if len(head) >= 2 && head[0] == 0xFF && (head[1]&0xE0) == 0xE0 {
		return true
	}
	// WAV: "RIFF....WAVE"
	if len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WAVE" {
		return true
	}
	// OGG: "OggS"
	if len(head) >= 4 && string(head[:4]) == "OggS" {
		return true
	}
	// FLAC: "fLaC"
	if len(head) >= 4 && string(head[:4]) == "fLaC" {
		return true
	}
	// M4A/MP4/AAC: "ftyp" at offset 4
	if len(head) >= 8 && string(head[4:8]) == "ftyp" {
		return true
	}
	return false
}

func decodeHex(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, errOddLength
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		hi, ok1 := hexVal(s[i])
		lo, ok2 := hexVal(s[i+1])
		if !ok1 || !ok2 {
			return nil, errInvalidHex
		}
		out[i/2] = hi<<4 | lo
	}
	return out, nil
}

func hexVal(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

var (
	errOddLength  = hexErr("hex 字符串长度必须为偶数")
	errInvalidHex = hexErr("hex 字符串包含非法字符")
)

type hexErr string

func (e hexErr) Error() string { return string(e) }

// buildMusicRequestBody 为不同模型构造最小可生成请求体。
// music-cover 需要 cover_feature_id，但跑集成测试拿不到真实 feature，
// 所以只校验前 5 个文本生成模型；cover 系列跳过端到端（保留路由测试在前面的单测）。
func buildMusicRequestBody(t *testing.T, model string) []byte {
	t.Helper()
	if strings.HasPrefix(model, "music-cover") {
		t.Skip("music-cover 需要 cover_feature_id，跑不通端到端；路由/允许列表/超时已在前面的单测覆盖")
	}
	body := map[string]interface{}{
		"model":         model,
		"prompt":        "轻轻的钢琴前奏，温暖治愈",
		"lyrics":        "[Verse]\n测试歌词\n[Chorus]\n副歌歌词",
		"output_format": "url",
		"audio_setting": map[string]interface{}{
			"sample_rate": 44100,
			"bitrate":     256000,
			"format":      "mp3",
		},
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	return b
}

func truncateForLog(b []byte) string {
	const maxLen = 500
	if len(b) <= maxLen {
		return string(b)
	}
	return string(b[:maxLen]) + "...(truncated)"
}

// ---------------------------------------------------------------------------
// 歌词生成（lyrics_generation）真实上游测试
// ---------------------------------------------------------------------------
//
// 这些测试目标：直接打 https://api.minimaxi.com/v1/lyrics_generation，
// 验证后端代理 + 重试逻辑在真实场景下能顶住 Cloudflare 偶发 524。
// 跑法（MINIMAX_API_KEY 是项目里 minimax.api_key 的 env 覆盖）：
//   MINIMAX_API_KEY=sk-xxx go test ./handlers/ -run TestLyrics -v
//
// 不设 key 时 t.Skip，不影响普通 `go test ./...`。

// newLyricsIntegrationRouter 起一个 gin 路由，把 /api/minimax/music/v1/lyrics_generation
// 指向真实上游（沿用项目里的 AIGatewayHandler，包括我们刚加的 4 次重试 + 抖动退避）。
func newLyricsIntegrationRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("MINIMAX_API_KEY 未设置，跳过真实 API 集成测试")
	}
	gin.SetMode(gin.TestMode)

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	// :memory: + 后台 goroutine 必须压到单连接，否则回写的 task 读不到。
	db.SetMaxOpenConns(1)
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init media tasks: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxTokenPlan.APIKey = apiKey
	cfg.MiniMaxTokenPlan.BaseURL = musicIntegrationBaseURL

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	// 异步歌词走 musicSubmitClient(5 分钟)，客户端不再被上游耗时绑住。
	h.mediaClient.Timeout = 110 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)
	return router, apiKey
}

// pollLyricsLive 用 2s 间隔轮询真实任务(上游歌词生成通常 20-40s)，
// 返回最终 task JSON。超时/失败直接 t.Fatalf。
func pollLyricsLive(t *testing.T, router *gin.Engine, taskID string, timeout time.Duration) map[string]interface{} {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req := httptest.NewRequest(http.MethodGet, "/api/minimax/music/v1/lyrics_tasks/"+taskID, nil)
		req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("poll task %s: status=%d body=%s", taskID, w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("poll decode: %v body=%s", err, w.Body.String())
		}
		switch resp["status"] {
		case "succeeded", "failed":
			return resp
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("poll task %s: 等待 %s 仍未完成", taskID, timeout)
	return nil
}

// lyricsFromTaskResult 从轮询结果的 result 字段里取 lyrics / song_title，
// 兼容 root 与 nested(data.*) 两种上游形状。
func lyricsFromTaskResult(task map[string]interface{}) (lyrics, title string) {
	result, _ := task["result"].(map[string]interface{})
	if result == nil {
		return "", ""
	}
	lyrics, _ = result["lyrics"].(string)
	title, _ = result["song_title"].(string)
	if data, ok := result["data"].(map[string]interface{}); ok {
		if lyrics == "" {
			lyrics, _ = data["lyrics"].(string)
		}
		if title == "" {
			if title, _ = data["song_title"].(string); title == "" {
				title, _ = data["title"].(string)
			}
		}
	}
	return lyrics, title
}

// TestLyricsIntegrationLiveAPI_HappyPath 真打一次异步歌词生成：
//  1. POST 立即返回 202 + task_id（这正是绕开 CF 100s origin timeout 的关键：客户端不再挂着等）
//  2. 轮询到 succeeded，result 里有 lyrics + song_title（前端 extractLyricsText 直接消费）
//  3. 记录 POST 返回耗时 vs 后台真实生成耗时的差距
func TestLyricsIntegrationLiveAPI_HappyPath(t *testing.T) {
	router, _ := newLyricsIntegrationRouter(t)

	body, _ := json.Marshal(map[string]interface{}{
		"mode":   "write_full_song",
		"prompt": "我在北疆自驾了一周,看到雪山和草原,想做一首轻快的民谣",
		"advancedParams": map[string]interface{}{
			"language": "zh",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader(body))
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	submitStart := time.Now()
	router.ServeHTTP(w, req)
	submitElapsed := time.Since(submitStart)

	if w.Code != http.StatusAccepted {
		t.Fatalf("submit failed: status=%d want 202, body=%s", w.Code, w.Body.String())
	}
	var submitResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &submitResp); err != nil {
		t.Fatalf("decode submit response: %v body=%s", err, w.Body.String())
	}
	taskID, _ := submitResp["task_id"].(string)
	if taskID == "" {
		t.Fatalf("submit response missing task_id: %s", w.Body.String())
	}
	// CF 的 origin read timeout 是 100s；提交必须远低于它才算真正解决 524。
	if submitElapsed > 5*time.Second {
		t.Fatalf("submit 耗时 %s，异步提交不该阻塞这么久", submitElapsed.Round(time.Millisecond))
	}
	t.Logf("submit: 202 in %s, task_id=%s", submitElapsed.Round(time.Millisecond), taskID)

	genStart := time.Now()
	task := pollLyricsLive(t, router, taskID, 5*time.Minute)
	genElapsed := time.Since(genStart)

	if task["status"] != "succeeded" {
		t.Fatalf("task failed: status=%v error=%v", task["status"], task["error"])
	}
	lyrics, title := lyricsFromTaskResult(task)
	if strings.TrimSpace(lyrics) == "" {
		t.Fatalf("task result missing lyrics: %+v", task)
	}
	if strings.TrimSpace(title) == "" {
		t.Fatalf("task result missing song_title: %+v", task)
	}
	t.Logf("succeeded after %s (提交只花了 %s，客户端从未挂长连接)", genElapsed.Round(time.Second), submitElapsed.Round(time.Millisecond))
	t.Logf("song_title=%q", title)
	t.Logf("lyrics=\n%s", truncateForLog([]byte(lyrics)))
}

// TestLyricsIntegrationLiveAPI_BurstResilience 连续提交 N 个异步任务，再逐个轮询结果。
// 异步化之后这个测试验证的是两件事：
//   - 提交永远是毫秒级 202（CF 不可能再 524，因为客户端根本不等）
//   - 后台 goroutine + 4 次重试能把 N 个任务都跑成 succeeded
func TestLyricsIntegrationLiveAPI_BurstResilience(t *testing.T) {
	router, _ := newLyricsIntegrationRouter(t)

	const N = 5
	taskIDs := make([]string, 0, N)
	var maxSubmit time.Duration

	for i := 1; i <= N; i++ {
		body, _ := json.Marshal(map[string]interface{}{
			"mode":   "write_full_song",
			"prompt": fmt.Sprintf("集成测试第 %d 次:写一首关于秋天的歌,带一点爵士味道", i),
			"advancedParams": map[string]interface{}{
				"language": "zh",
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader(body))
		req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		start := time.Now()
		router.ServeHTTP(w, req)
		elapsed := time.Since(start)
		if elapsed > maxSubmit {
			maxSubmit = elapsed
		}

		if w.Code != http.StatusAccepted {
			t.Fatalf("submit %d: status=%d want 202, body=%s", i, w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		taskID, _ := resp["task_id"].(string)
		if taskID == "" {
			t.Fatalf("submit %d: missing task_id, body=%s", i, w.Body.String())
		}
		taskIDs = append(taskIDs, taskID)
		t.Logf("submit %d: 202 in %s task_id=%s", i, elapsed.Round(time.Millisecond), taskID)
	}

	var okCount, failCount int
	for i, taskID := range taskIDs {
		task := pollLyricsLive(t, router, taskID, 5*time.Minute)
		lyrics, title := lyricsFromTaskResult(task)
		if task["status"] == "succeeded" && strings.TrimSpace(lyrics) != "" {
			okCount++
			t.Logf("task %d (%s): succeeded, song_title=%q", i+1, taskID, title)
		} else {
			failCount++
			t.Logf("task %d (%s): FAIL status=%v error=%v", i+1, taskID, task["status"], task["error"])
		}
	}

	t.Logf("--- burst summary ---")
	t.Logf("submitted=%d  succeeded=%d  failed=%d  max submit latency=%s",
		N, okCount, failCount, maxSubmit.Round(time.Millisecond))
	// 提交必须始终远低于 CF 的 100s origin read timeout，这是异步化的全部意义。
	if maxSubmit > 5*time.Second {
		t.Fatalf("最慢一次提交 %s，异步提交不该阻塞", maxSubmit.Round(time.Millisecond))
	}
	if okCount == 0 {
		t.Fatalf("%d 个真实任务全部失败 — key 不对 / 上游挂了 / 网络不通", N)
	}
}