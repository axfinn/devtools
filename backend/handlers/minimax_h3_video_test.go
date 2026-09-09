package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// setupH3VideoTest 起一个 httptest 模拟 MiniMax-H3 上游,挂上 H3 路由。
// 返回:router, upstream, hits 计数器。
func setupH3VideoTest(t *testing.T) (*gin.Engine, *httptest.Server, *atomic.Int32) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/video_generation":
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			_ = json.Unmarshal(body, &req)
			// 校验:每次请求必须含非空 text prompt
			hasText := false
			if content, ok := req["content"].([]interface{}); ok {
				for _, c := range content {
					if item, ok := c.(map[string]interface{}); ok {
						if item["type"] == "text" {
							if txt, _ := item["text"].(string); strings.TrimSpace(txt) != "" {
								hasText = true
							}
						}
					}
				}
			}
			if !hasText {
				http.Error(w, `{"error":{"message":"prompt required"}}`, http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"task_id":"424010985738629","base_resp":{"status_code":0}}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/query/video_generation/") && r.URL.Path != "/v2/query/video_generation":
			// GET 单任务 → 返回 succeeded + content.url
			_, _ = io.WriteString(w, `{"task":{"id":"424010985738629","model":"MiniMax-H3","status":"succeeded","created_at":1785125529,"updated_at":1785125946,"content":{"url":"https://example.com/h3-output.mp4"},"resolution":"2K","duration":5,"task_type":"generation"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/query/video_generation":
			_, _ = io.WriteString(w, `{"items":[{"id":"424010985738629","model":"MiniMax-H3","status":"succeeded","created_at":1785125529,"updated_at":1785125946,"content":{"url":"https://example.com/h3-output.mp4"},"task_type":"generation"}],"total":1}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v2/video_generation/"):
			_, _ = io.WriteString(w, `{"task_id":"424010985738629","action":"deleted","status":"deleted"}`)
		default:
			http.Error(w, `{"error":{"message":"unexpected path: `+r.Method+` `+r.URL.Path+`"}}`, http.StatusBadRequest)
		}
	}))

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitMiniMaxMediaTasks(); err != nil {
		t.Fatalf("init media tasks: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	cfg.MiniMaxH3Video.APIKey = "test-h3-key"
	cfg.MiniMaxH3Video.BaseURL = upstream.URL
	// 不要 MiniMax.APIKey,验证 fallback 不被触发也能正常工作
	cfg.MiniMax.APIKey = ""

	h := NewAIGatewayHandler(db, cfg, nil, testEncEncryptionService)
	h.mediaClient.Timeout = 5 * time.Second
	h.noProxyClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/minimax/h3/v2/video_generation", h.MiniMaxH3VideoCreate)
	router.GET("/api/minimax/h3/v2/query/video_generation", h.MiniMaxH3VideoList)
	router.GET("/api/minimax/h3/v2/query/video_generation/:task_id", h.MiniMaxH3VideoQuery)
	router.DELETE("/api/minimax/h3/v2/video_generation/:task_id", h.MiniMaxH3VideoDelete)
	return router, upstream, &hits
}

func TestH3VideoCreate_RejectsMissingPrompt(t *testing.T) {
	router, _, _ := setupH3VideoTest(t)
	body := strings.NewReader(`{"model":"MiniMax-H3","content":[{"type":"image_url","image_url":{"url":"https://example.com/a.png"},"role":"first_frame"}],"resolution":"2K","duration":5}`)
	req := httptest.NewRequest(http.MethodPost, "/api/minimax/h3/v2/video_generation", body)
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	req.Header.Set("Content-Type", "application/json")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	if wr.Code != http.StatusBadGateway && wr.Code != http.StatusBadRequest {
		t.Fatalf("expected 4xx upstream passthrough, got %d body=%s", wr.Code, wr.Body.String())
	}
}

func TestH3VideoCreate_ReturnsTaskIDAndLocalRecord(t *testing.T) {
	router, upstream, hits := setupH3VideoTest(t)
	body := strings.NewReader(`{"model":"MiniMax-H3","content":[{"type":"text","text":"a boy playing basketball on the beach"}],"resolution":"2K","duration":5,"ratio":"16:9"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/minimax/h3/v2/video_generation", body)
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	req.Header.Set("Content-Type", "application/json")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	if wr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", wr.Code, wr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(wr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.HasPrefix(resp["task_id"].(string), "mmh3_") {
		t.Fatalf("expected local mmh3_ task_id, got %v", resp["task_id"])
	}
	if resp["external_task_id"] != "424010985738629" {
		t.Fatalf("expected external task id from upstream, got %v", resp["external_task_id"])
	}
	if resp["task_type"] != "text_to_video" {
		t.Fatalf("expected task_type text_to_video, got %v", resp["task_type"])
	}
	if hits.Load() == 0 {
		t.Fatalf("upstream not called")
	}
	_ = upstream
}

func TestH3VideoCreate_DetectsI2V(t *testing.T) {
	router, _, _ := setupH3VideoTest(t)
	body := strings.NewReader(`{"model":"MiniMax-H3","content":[{"type":"text","text":"Pull focus"},{"type":"image_url","image_url":{"url":"https://example.com/a.png"},"role":"first_frame"}],"resolution":"2K","duration":5}`)
	req := httptest.NewRequest(http.MethodPost, "/api/minimax/h3/v2/video_generation", body)
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	req.Header.Set("Content-Type", "application/json")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	if wr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", wr.Code, wr.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	if resp["task_type"] != "image_to_video" {
		t.Fatalf("expected task_type image_to_video, got %v", resp["task_type"])
	}
}

func TestH3VideoQuery_LocalTaskID(t *testing.T) {
	router, _, _ := setupH3VideoTest(t)
	// 先提交一个
	body := strings.NewReader(`{"model":"MiniMax-H3","content":[{"type":"text","text":"x"}],"resolution":"2K","duration":5,"ratio":"16:9"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/minimax/h3/v2/video_generation", body)
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	var submitted map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &submitted)
	localID := submitted["task_id"].(string)

	// 再用 local id 查 — 后台 goroutine 已成功,result_json 已落库
	time.Sleep(200 * time.Millisecond)
	queryReq := httptest.NewRequest(http.MethodGet, "/api/minimax/h3/v2/query/video_generation/"+localID, nil)
	queryReq.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	queryWR := httptest.NewRecorder()
	router.ServeHTTP(queryWR, queryReq)
	if queryWR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", queryWR.Code, queryWR.Body.String())
	}
	var got map[string]interface{}
	_ = json.Unmarshal(queryWR.Body.Bytes(), &got)
	if got["task_id"] != localID {
		t.Fatalf("expected task_id %s, got %v", localID, got["task_id"])
	}
	if got["external_task_id"] != "424010985738629" {
		t.Fatalf("expected external_task_id, got %v", got["external_task_id"])
	}
}

func TestH3VideoQuery_ExternalTaskIDPassesThrough(t *testing.T) {
	router, _, _ := setupH3VideoTest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/minimax/h3/v2/query/video_generation/424010985738629", nil)
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	if wr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", wr.Code, wr.Body.String())
	}
	var got map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &got)
	if _, ok := got["task"]; !ok {
		t.Fatalf("expected upstream task passthrough, got %v", got)
	}
}

func TestH3VideoList_PassesThroughFilterQuery(t *testing.T) {
	router, _, _ := setupH3VideoTest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/minimax/h3/v2/query/video_generation?page_num=1&page_size=10&filter.status=succeeded", nil)
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	if wr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", wr.Code, wr.Body.String())
	}
	var got map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &got)
	if _, ok := got["items"]; !ok {
		t.Fatalf("expected items array passthrough, got %v", got)
	}
}

func TestH3VideoDelete_RequiresAuth(t *testing.T) {
	router, _, _ := setupH3VideoTest(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/minimax/h3/v2/video_generation/424010985738629", nil)
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	if wr.Code == http.StatusOK {
		t.Fatalf("expected auth failure, got %d", wr.Code)
	}
}

func TestH3VideoDelete_PassesThrough(t *testing.T) {
	router, _, _ := setupH3VideoTest(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/minimax/h3/v2/video_generation/424010985738629", nil)
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	if wr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", wr.Code, wr.Body.String())
	}
	var got map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &got)
	if got["status"] != "deleted" {
		t.Fatalf("expected status deleted, got %v", got)
	}
}

func TestResolveH3VideoConfig_Fallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.MiniMaxH3Video.APIKey = ""
	cfg.MiniMax.APIKey = "fallback-key"
	cfg.MiniMaxH3Video.BaseURL = ""
	h := NewAIGatewayHandler(db, cfg, nil, testEncEncryptionService)
	apiKey, baseURL := h.resolveH3VideoConfig()
	if apiKey != "fallback-key" {
		t.Fatalf("expected fallback to MiniMax.APIKey, got %q", apiKey)
	}
	if baseURL != defaultMiniMaxH3BaseURL {
		t.Fatalf("expected default baseURL, got %q", baseURL)
	}
}

func TestMapH3Status(t *testing.T) {
	cases := map[string]string{
		"queued":    "pending",
		"running":   "running",
		"succeeded": "succeeded",
		"failed":    "failed",
		"cancelled": "cancelled",
		"deleted":   "deleted",
		"unknown":   "pending",
	}
	for in, want := range cases {
		if got := mapH3Status(in); got != want {
			t.Errorf("mapH3Status(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInferH3TaskType(t *testing.T) {
	cases := []struct {
		name string
		body map[string]interface{}
		want string
	}{
		{"empty", map[string]interface{}{}, "text_to_video"},
		{"t2v", map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": "x"}}}, "text_to_video"},
		{"i2v", map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": "x"}, map[string]interface{}{"type": "image_url", "role": "first_frame"}}}, "image_to_video"},
		{"r2va", map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": "x"}, map[string]interface{}{"type": "image_url", "role": "reference_image"}, map[string]interface{}{"type": "video_url", "role": "reference_video"}, map[string]interface{}{"type": "audio_url", "role": "reference_audio"}}}, "h3_context_ir"},
		{"regen", map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": "x"}, map[string]interface{}{"type": "video_url", "role": "base_video"}}}, "regeneration"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferH3TaskType(tc.body); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
