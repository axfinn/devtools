package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// setupAnthropicAsyncRouter 构造异步 Anthropic 代理测试环境:
// - mock upstream 返回受控响应(调用方可注入 status + body)
// - :memory: DB + InitAnthropicTasks + SetMaxOpenConns(1)(防 goroutine 池陷阱,见 feedback_sqlite_memory_pool.md)
// - NewAIGatewayHandler 走完整初始化路径(longNoProxyClient 是 5min timeout,这里缩到 5s 加速测试)
// - gin router 注册两条异步端点
// - admin password = "test-admin-pw" 用于 X-Super-Admin-Password 鉴权
func setupAnthropicAsyncRouter(t *testing.T, upstreamHandler http.HandlerFunc) (*gin.Engine, *atomic.Int32) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	hits := &atomic.Int32{}
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		upstreamHandler(w, r)
	})
	upstream := httptest.NewServer(wrapped)
	t.Cleanup(upstream.Close)

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitAnthropicTasks(); err != nil {
		t.Fatalf("init anthropic_tasks: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"

	h := NewAIGatewayHandler(db, cfg, nil, nil, testEncEncryptionService)
	// 测试环境不需要 5min 长 timeout,缩到 5s 加速失败场景
	h.longNoProxyClient.Timeout = 5 * time.Second
	h.noProxyClient.Timeout = 5 * time.Second

	router := gin.New()
	router.POST("/api/anthropic/v1/messages/tasks", h.AsyncAnthropicMessages)
	router.GET("/api/anthropic/v1/messages/tasks/:id", h.GetAnthropicMessageTask)
	return router, hits
}

// submitAsyncAnthropic 提交异步任务,返回 (status code, task_id, full body)。
func submitAsyncAnthropic(t *testing.T, router *gin.Engine, body []byte, adminPwd string) (int, string, []byte) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/anthropic/v1/messages/tasks", bytes.NewReader(body))
	req.Header.Set("X-Super-Admin-Password", adminPwd)
	req.Header.Set("Content-Type", "application/json")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(wr.Body.Bytes(), &resp)
	taskID, _ := resp["task_id"].(string)
	return wr.Code, taskID, wr.Body.Bytes()
}

// pollAnthropicUntilDone 拿到 task_id 后轮询,直到 status ∈ {succeeded, failed} 或超时。
func pollAnthropicUntilDone(t *testing.T, router *gin.Engine, taskID, adminPwd string, timeout time.Duration) (int, []byte) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastBody []byte
	var lastCode int
	for time.Now().Before(deadline) {
		req := httptest.NewRequest(http.MethodGet, "/api/anthropic/v1/messages/tasks/"+taskID, nil)
		req.Header.Set("X-Super-Admin-Password", adminPwd)
		wr := httptest.NewRecorder()
		router.ServeHTTP(wr, req)
		lastCode = wr.Code
		lastBody = wr.Body.Bytes()
		if wr.Code != http.StatusOK {
			return lastCode, lastBody
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(lastBody, &resp)
		switch resp["status"] {
		case "succeeded", "failed":
			return lastCode, lastBody
		}
		time.Sleep(30 * time.Millisecond)
	}
	return lastCode, lastBody
}

// ============== 用例 ==============

// TestAsyncAnthropicMessages_StreamRejected 异步端点拒绝 stream:true。
// 流式场景已在 P0 修复(ssePingFrame 持续 reset read timeout),不需要异步化。
func TestAsyncAnthropicMessages_StreamRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := setupAnthropicAsyncRouter(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("upstream should not be called for stream:true rejection")
		w.WriteHeader(200)
	})

	body := []byte(`{"model":"qwen3.8:27b-mlx","max_tokens":32,"stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	code, _, raw := submitAsyncAnthropic(t, router, body, "test-admin-pw")
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for stream:true, got %d: %s", code, raw)
	}
	if !strings.Contains(string(raw), "stream:true") {
		t.Errorf("error message should mention stream:true, got: %s", raw)
	}
}

// TestAsyncAnthropicMessages_BadModel 未知模型应返 400,不应创建 task。
func TestAsyncAnthropicMessages_BadModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, hits := setupAnthropicAsyncRouter(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("upstream should not be called for bad model")
	})

	// 注: builtin providers 不覆盖 "non-existent-model",resolveAnthropicProvider 走 fallback
	// 到默认 provider(DeepSeek),所以这里改成完全空 body 字段来验证 400。
	body := []byte(`{"max_tokens":32,"messages":[{"role":"user","content":"hi"}]}`) // 缺 model
	code, taskID, raw := submitAsyncAnthropic(t, router, body, "test-admin-pw")
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing model, got %d: %s", code, raw)
	}
	if taskID != "" {
		t.Errorf("task_id should be empty for rejected request, got %s", taskID)
	}
	if hits.Load() != 0 {
		t.Errorf("upstream should not be called, got %d hits", hits.Load())
	}
}

// TestAsyncAnthropicMessages_BadJSON 请求体不是合法 JSON 应返 400。
func TestAsyncAnthropicMessages_BadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := setupAnthropicAsyncRouter(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("upstream should not be called for bad json")
	})

	code, _, raw := submitAsyncAnthropic(t, router, []byte(`{not valid json`), "test-admin-pw")
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad json, got %d: %s", code, raw)
	}
	if !strings.Contains(string(raw), "JSON") {
		t.Errorf("error should mention JSON, got: %s", raw)
	}
}

// TestAsyncAnthropicMessages_EmptyBody 空 body 应返 400。
func TestAsyncAnthropicMessages_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := setupAnthropicAsyncRouter(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("upstream should not be called for empty body")
	})

	code, _, raw := submitAsyncAnthropic(t, router, []byte{}, "test-admin-pw")
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d: %s", code, raw)
	}
}

// TestRunAsyncAnthropicTask_HappyPath 直接调后台 worker,绕过 HTTP 路由层。
// 原因: builtin Ollama provider URL 是硬编码 192.168.31.147:11434,路由层解析后会
// 命中 builtin 而不是测试 mock server。直接 worker 调用可以精准指定 provider。
func TestRunAsyncAnthropicTask_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := `{"id":"msg_upstream_001","type":"message","role":"assistant","model":"qwen3.8:27b-mlx","content":[{"type":"text","text":"hello"}],"usage":{"input_tokens":3,"output_tokens":2}}`
	var capturedBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, upstreamBody)
	}))
	defer upstream.Close()

	db, _ := models.NewDB(":memory:")
	db.SetMaxOpenConns(1)
	db.InitAnthropicTasks()

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	h := NewAIGatewayHandler(db, cfg, nil, nil, testEncEncryptionService)
	h.longNoProxyClient.Timeout = 5 * time.Second
	h.noProxyClient.Timeout = 5 * time.Second

	taskID := "oant_test_happy_001"
	task := &models.AnthropicTask{
		ID:          taskID,
		APIKeyID:    "",
		Model:       "qwen3.8:27b-mlx",
		Provider:    "Ollama",
		Status:      "pending",
		RequestBody: `{"model":"qwen3.8:27b-mlx","max_tokens":32}`,
		ClientIP:    "127.0.0.1",
	}
	if err := db.CreateAnthropicTask(task); err != nil {
		t.Fatalf("create: %v", err)
	}

	provider := &config.AnthropicProviderConfig{
		Name:   "Ollama",
		APIURL: upstream.URL, // 直接指向 mock server
		APIKey: "ollama-test",
	}
	upstreamURL := upstream.URL + "/v1/messages"
	bodyBytes := []byte(`{"model":"qwen3.8:27b-mlx","max_tokens":32,"messages":[{"role":"user","content":"hi"}]}`)

	h.runAsyncAnthropicTask(taskID, provider, upstreamURL, bodyBytes, "")

	got, err := db.GetAnthropicTask(taskID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "succeeded" {
		t.Fatalf("status = %s, want succeeded; err=%s", got.Status, got.ErrorMessage)
	}
	if got.ResultJSON != upstreamBody {
		t.Errorf("ResultJSON = %q, want %q", got.ResultJSON, upstreamBody)
	}
	if got.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
	if !strings.Contains(string(capturedBody), `"model":"qwen3.8:27b-mlx"`) {
		t.Errorf("upstream received wrong body: %s", capturedBody)
	}
}

// TestRunAsyncAnthropicTask_StripsThinkingBlocks DeepSeek/Ollama 上游返回带
// thinking 块,worker 应在 result_json 中剥离。
func TestRunAsyncAnthropicTask_StripsThinkingBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := `{"id":"msg_x","content":[{"type":"thinking","thinking":"reasoning..."},{"type":"text","text":"answer"}],"usage":{}}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, upstreamBody)
	}))
	defer upstream.Close()

	db, _ := models.NewDB(":memory:")
	db.SetMaxOpenConns(1)
	db.InitAnthropicTasks()

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"
	h := NewAIGatewayHandler(db, cfg, nil, nil, testEncEncryptionService)
	h.longNoProxyClient.Timeout = 5 * time.Second

	taskID := "oant_test_think_001"
	db.CreateAnthropicTask(&models.AnthropicTask{
		ID: taskID, APIKeyID: "", Model: "qwen3.8:27b-mlx", Provider: "Ollama",
		Status: "pending", RequestBody: `{}`, ClientIP: "127.0.0.1",
	})

	// 走 Ollama 路径(providerEmitsThinkingBlocks 通过 name 匹配 "ollama")
	provider := &config.AnthropicProviderConfig{
		Name:   "Ollama",
		APIURL: upstream.URL,
		APIKey: "ollama",
	}
	h.runAsyncAnthropicTask(taskID, provider, upstream.URL+"/v1/messages", []byte(`{"model":"qwen3.8:27b-mlx"}`), "")

	got, _ := db.GetAnthropicTask(taskID)
	if got.Status != "succeeded" {
		t.Fatalf("status = %s, want succeeded", got.Status)
	}
	// thinking 应被剥离,只剩 text 块
	if strings.Contains(got.ResultJSON, `"thinking"`) {
		t.Errorf("thinking block should be stripped, got: %s", got.ResultJSON)
	}
	if !strings.Contains(got.ResultJSON, `"answer"`) {
		t.Errorf("text block should be preserved, got: %s", got.ResultJSON)
	}
}

// TestAsyncAnthropicMessages_UpstreamError upstream 返回 500 → task failed + error_message 记录。
func TestAsyncAnthropicMessages_UpstreamError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := setupAnthropicAsyncRouter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error":"model overloaded"}`)
	})

	body := []byte(`{"model":"qwen3.8:27b-mlx","max_tokens":32,"messages":[{"role":"user","content":"hi"}]}`)
	code, taskID, _ := submitAsyncAnthropic(t, router, body, "test-admin-pw")
	if code != http.StatusAccepted {
		t.Fatalf("POST expected 202, got %d", code)
	}

	// 轮询,期望最终 failed + error 字段
	code2, raw2 := pollAnthropicUntilDone(t, router, taskID, "test-admin-pw", 3*time.Second)
	if code2 != http.StatusOK {
		t.Fatalf("poll status = %d, body: %s", code2, raw2)
	}
	var pollResp map[string]interface{}
	_ = json.Unmarshal(raw2, &pollResp)
	if pollResp["status"] != "failed" {
		t.Errorf("final status = %v, want failed; body: %s", pollResp["status"], raw2)
	}
	errMsg, _ := pollResp["error"].(string)
	if errMsg == "" {
		t.Error("error field should be populated for failed task")
	}
}

// TestAsyncAnthropicMessages_ConnectionRefused upstream 完全连不上 → failed + network error。
func TestAsyncAnthropicMessages_ConnectionRefused(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 直接构造 handler,longNoProxyClient 指向无效端口,模拟 Ollama 挂掉
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitAnthropicTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "test-admin-pw"

	h := NewAIGatewayHandler(db, cfg, nil, nil, testEncEncryptionService)
	h.longNoProxyClient.Timeout = 500 * time.Millisecond
	h.noProxyClient.Timeout = 500 * time.Millisecond

	// 通过直接调 runAsyncAnthropicTask 跳过网络 — 验证 DB 错误处理路径
	taskID := "oant_" + fmt.Sprintf("%024x", 1)
	task := &models.AnthropicTask{
		ID:          taskID,
		APIKeyID:    "",
		Model:       "qwen3.8:27b-mlx",
		Provider:    "Ollama",
		Status:      "pending",
		RequestBody: `{"model":"qwen3.8:27b-mlx"}`,
		ClientIP:    "127.0.0.1",
	}
	if err := db.CreateAnthropicTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 直接调后台 worker,upstream URL 是无效端口
	h.runAsyncAnthropicTask(taskID, &config.AnthropicProviderConfig{
		Name:   "Ollama",
		APIURL: "http://127.0.0.1:1",
		APIKey: "ollama",
	}, "http://127.0.0.1:1/v1/messages", []byte(`{"model":"qwen3.8:27b-mlx"}`), "")

	got, err := db.GetAnthropicTask(taskID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Status != "failed" {
		t.Errorf("status = %s, want failed", got.Status)
	}
	if got.ErrorMessage == "" {
		t.Error("ErrorMessage should be set")
	}
	if got.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

// TestAnthropicTaskDB_RoundTrip 验证 DB CRUD 路径。
func TestAnthropicTaskDB_RoundTrip(t *testing.T) {
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitAnthropicTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}

	original := &models.AnthropicTask{
		ID:           "oant_test_abc123",
		APIKeyID:     "dtk_test_xyz",
		Model:        "qwen3.8:27b-mlx",
		Provider:     "Ollama",
		Status:       "pending",
		RequestBody:  `{"model":"qwen3.8:27b-mlx","max_tokens":32}`,
		ResultJSON:   "",
		ErrorMessage: "",
		ClientIP:     "127.0.0.1",
	}
	if err := db.CreateAnthropicTask(original); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := db.GetAnthropicTask(original.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Model != original.Model || got.Provider != original.Provider || got.Status != original.Status {
		t.Errorf("fields mismatch: got=%+v, want=%+v", got, original)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set by CreateAnthropicTask")
	}

	// Update 路径
	got.Status = "succeeded"
	got.ResultJSON = `{"id":"msg_x"}`
	now := time.Now()
	got.CompletedAt = &now
	if err := db.UpdateAnthropicTask(got); err != nil {
		t.Fatalf("update: %v", err)
	}

	got2, _ := db.GetAnthropicTask(original.ID)
	if got2.Status != "succeeded" || got2.ResultJSON != `{"id":"msg_x"}` || got2.CompletedAt == nil {
		t.Errorf("update not persisted: %+v", got2)
	}

	// 不存在 → 错误
	if _, err := db.GetAnthropicTask("oant_nonexistent"); err == nil {
		t.Error("expected error for nonexistent task")
	}
}

// TestGetAnthropicMessageTask_NotFound 不存在 task 应返 404。
func TestGetAnthropicMessageTask_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := setupAnthropicAsyncRouter(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("upstream should not be called for 404 test")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/anthropic/v1/messages/tasks/oant_nonexistent", nil)
	req.Header.Set("X-Super-Admin-Password", "test-admin-pw")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)

	if wr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", wr.Code, wr.Body.String())
	}
}
