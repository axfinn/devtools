//go:build stress

// 压力测试：并发 8 个 goroutine 各自连发 3 次 lyrics_generation，
// 试图触发 CF 524（origin timeout）。需要 MINIMAX_API_KEY。
//   MINIMAX_API_KEY=xxx go test -tags=stress -run TestLyrics -v -timeout 10m ./handlers/

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

func TestLyricsStressConcurrent(t *testing.T) {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("MINIMAX_API_KEY 未设置")
	}
	gin.SetMode(gin.TestMode)

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
	cfg.MiniMaxTokenPlan.APIKey = apiKey
	cfg.MiniMaxTokenPlan.BaseURL = "https://api.minimaxi.com"

	h := NewAIGatewayHandler(db, cfg, nil, nil)
	h.mediaClient.Timeout = 110 * time.Second

	router := gin.New()
	router.POST("/api/minimax/music/v1/lyrics_generation", h.MiniMaxLyricsGeneration)
	router.GET("/api/minimax/music/v1/lyrics_tasks/:id", h.GetMiniMaxLyricsTask)

	const (
		concurrency = 8
		perWorker   = 3
	)
	var (
		okCount       atomic.Int32
		retryAndOK    atomic.Int32
		fail524       atomic.Int32
		fail5xx       atomic.Int32
		fail4xx       atomic.Int32
		totalRetries  atomic.Int32
		maxRetry      atomic.Int32
		taskIDs       sync.Map // task_id (string) → struct{}
	)

	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				body, _ := json.Marshal(map[string]interface{}{
					"mode":   "write_full_song",
					"prompt": fmt.Sprintf("压测 worker=%d 第 %d 次:写一首关于秋天的歌", workerID, i+1),
					"advancedParams": map[string]interface{}{"language": "zh"},
				})
				req := httptest.NewRequest(http.MethodPost, "/api/minimax/music/v1/lyrics_generation", bytes.NewReader(body))
				req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
				req.Header.Set("Content-Type", "application/json")
				wr := httptest.NewRecorder()
				router.ServeHTTP(wr, req)

				rcStr := wr.Header().Get("X-Retry-Count")
				var rc int
				fmt.Sscanf(rcStr, "%d", &rc)
				totalRetries.Add(int32(rc))
				for {
					cur := maxRetry.Load()
					if int32(rc) <= cur || maxRetry.CompareAndSwap(cur, int32(rc)) {
						break
					}
				}

				switch {
				case wr.Code == http.StatusAccepted && rc > 0:
					retryAndOK.Add(1)
					okCount.Add(1)
				case wr.Code == http.StatusAccepted:
					okCount.Add(1)
				case wr.Code == http.StatusOK && rc > 0:
					retryAndOK.Add(1)
					okCount.Add(1)
				case wr.Code == http.StatusOK:
					okCount.Add(1)
				case wr.Code == 524:
					fail524.Add(1)
				case wr.Code >= 500:
					fail5xx.Add(1)
				default:
					fail4xx.Add(1)
				}

				// 收集 task_id，后面统一轮询验证后台 goroutine 也成功。
				var resp map[string]interface{}
				if json.Unmarshal(wr.Body.Bytes(), &resp) == nil {
					if tid, ok := resp["task_id"].(string); ok && tid != "" {
						taskIDs.Store(tid, struct{}{})
					}
				}
			}
		}(w)
	}
	wg.Wait()
	submitElapsed := time.Since(start)

	// 异步提交之后必须验证后台 goroutine 真的跑完了 — 这才是"端到端不 524"的最终证据。
	var (
		pendingOK   atomic.Int32
		pendingFail atomic.Int32
	)
	collected := make([]string, 0, concurrency*perWorker)
	taskIDs.Range(func(k, _ any) bool {
		collected = append(collected, k.(string))
		return true
	})
	t.Logf("=== polling %d async tasks ===", len(collected))
	pollStart := time.Now()
	var pollWG sync.WaitGroup
	for _, tid := range collected {
		pollWG.Add(1)
		go func(taskID string) {
			defer pollWG.Done()
			deadline := time.Now().Add(5 * time.Minute)
			for time.Now().Before(deadline) {
				req := httptest.NewRequest(http.MethodGet, "/api/minimax/music/v1/lyrics_tasks/"+taskID, nil)
				req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
				wr := httptest.NewRecorder()
				router.ServeHTTP(wr, req)
				if wr.Code != http.StatusOK {
					t.Logf("task %s: poll http=%d body=%s", taskID, wr.Code, truncateForLog(wr.Body.Bytes()))
					return
				}
				var resp map[string]interface{}
				if json.Unmarshal(wr.Body.Bytes(), &resp) != nil {
					return
				}
				switch resp["status"] {
				case "succeeded":
					pendingOK.Add(1)
					return
				case "failed":
					pendingFail.Add(1)
					t.Logf("task %s: failed error=%v", taskID, resp["error"])
					return
				}
				time.Sleep(1 * time.Second)
			}
			pendingFail.Add(1)
			t.Logf("task %s: poll timeout", taskID)
		}(tid)
	}
	pollWG.Wait()
	pollElapsed := time.Since(pollStart)

	t.Logf("=== stress result ===")
	t.Logf("submit: concurrency=%d per-worker=%d total=%d elapsed=%s",
		concurrency, perWorker, concurrency*perWorker, submitElapsed.Round(time.Millisecond))
	t.Logf("submit phase: ok=%d  retried-then-ok=%d  fail-524=%d  fail-5xx=%d  fail-4xx=%d",
		okCount.Load(), retryAndOK.Load(), fail524.Load(), fail5xx.Load(), fail4xx.Load())
	t.Logf("poll phase (background goroutine end-to-end): succeeded=%d failed=%d elapsed=%s",
		pendingOK.Load(), pendingFail.Load(), pollElapsed.Round(time.Second))
	if fail524.Load() > 0 {
		t.Logf("⚠️  提交阶段出现 %d 次 524 (CF origin timeout)", fail524.Load())
	}
	if pendingOK.Load() == 0 {
		t.Fatalf("0 个后台任务跑成功 — 检查 key/网络/超时配置")
	}
}