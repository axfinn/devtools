package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

const testChatPwd = "test-chat-pw"

// newChatTaskTestEnv 起一个假 Ollama（OpenAI 兼容 SSE）+ 内存库 + 任务路由。
func newChatTaskTestEnv(t *testing.T, upstream http.HandlerFunc) (*gin.Engine, *AIGatewayHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	srv := httptest.NewServer(upstream)
	t.Cleanup(srv.Close)

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitChatPlusTasks(); err != nil {
		t.Fatalf("init chat_plus_tasks: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.ChatPlus.Password = testChatPwd
	cfg.ChatPlus.OllamaBaseURL = srv.URL

	h := NewAIGatewayHandler(db, cfg, nil, testEncEncryptionService)

	router := gin.New()
	router.POST("/api/chat-plus/tasks", h.ChatPlusTaskCreate)
	router.GET("/api/chat-plus/tasks", h.ChatPlusTaskList)
	router.GET("/api/chat-plus/tasks/:id", h.ChatPlusTaskGet)
	router.GET("/api/chat-plus/tasks/:id/stream", h.ChatPlusTaskStream)
	router.POST("/api/chat-plus/tasks/:id/stop", h.ChatPlusTaskStop)
	return router, h
}

func chatTaskRequest(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Chat-Password", testChatPwd)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func createOllamaTask(t *testing.T, router *gin.Engine) string {
	t.Helper()
	rec := chatTaskRequest(t, router, http.MethodPost, "/api/chat-plus/tasks",
		`{"model":"ollama:qwen3:14b","messages":[{"role":"user","content":"hi"}],"conversation_id":"conv-1"}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("创建任务状态码 = %d, body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析创建响应: %v", err)
	}
	if out.TaskID == "" {
		t.Fatal("task_id 为空")
	}
	if out.Status != "running" {
		t.Errorf("初始状态 = %s, want running", out.Status)
	}
	return out.TaskID
}

// waitChatTask 轮询任务直到终态。
func waitChatTask(t *testing.T, router *gin.Engine, taskID string) map[string]interface{} {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rec := chatTaskRequest(t, router, http.MethodGet, "/api/chat-plus/tasks/"+taskID, "")
		var out map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("解析任务快照: %v (%s)", err, rec.Body.String())
		}
		if done, _ := out["done"].(bool); done {
			return out
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("任务 %s 5s 内未结束", taskID)
	return nil
}

func ollamaSSEUpstream(chunks ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, c := range chunks {
			io.WriteString(w, c)
			w.(http.Flusher).Flush()
		}
	}
}

// 端到端：创建任务 → 后台跑完 → 快照/列表都能拿到完整内容（刷新页面靠这条路径）。
func TestChatPlusTaskRunsInBackgroundAndPersists(t *testing.T) {
	router, _ := newChatTaskTestEnv(t, ollamaSSEUpstream(
		`data: {"choices":[{"delta":{"content":"你好"}}]}`+"\n\n",
		`data: {"choices":[{"delta":{"content":"，世界"}}]}`+"\n\n",
		"data: [DONE]\n\n",
	))

	taskID := createOllamaTask(t, router)
	snap := waitChatTask(t, router, taskID)

	if got := snap["status"]; got != "succeeded" {
		t.Fatalf("status = %v, want succeeded (error=%v)", got, snap["error_message"])
	}
	if got := snap["content"]; got != "你好，世界" {
		t.Errorf("content = %q, want 你好，世界", got)
	}

	// 列表：刷新后按 conversation_id 找回结果
	rec := chatTaskRequest(t, router, http.MethodGet, "/api/chat-plus/tasks?conversation_id=conv-1", "")
	var list struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("解析列表: %v", err)
	}
	if len(list.Tasks) != 1 {
		t.Fatalf("列表应返回 1 条任务, got %d", len(list.Tasks))
	}
	if list.Tasks[0]["content"] != "你好，世界" {
		t.Errorf("列表里的 content = %v", list.Tasks[0]["content"])
	}

	// SSE 回放（from=0）：能补全整段内容
	streamRec := chatTaskRequest(t, router, http.MethodGet, "/api/chat-plus/tasks/"+taskID+"/stream?from=0", "")
	body := streamRec.Body.String()
	if !strings.Contains(body, "你好") || !strings.Contains(body, "世界") {
		t.Errorf("回放内容不完整: %q", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Errorf("回放缺少 [DONE]: %q", body)
	}
	if !strings.Contains(body, "id: 1") {
		t.Errorf("回放缺少 seq 标识（续传依赖它）: %q", body)
	}
}

// 上游失败：任务必须失败，且错误原因可读（"失败也会知道"）。
func TestChatPlusTaskFailureIsVisible(t *testing.T) {
	router, _ := newChatTaskTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error":{"message":"model overloaded"}}`)
	})

	taskID := createOllamaTask(t, router)
	snap := waitChatTask(t, router, taskID)

	if snap["status"] != "failed" {
		t.Fatalf("status = %v, want failed", snap["status"])
	}
	errMsg, _ := snap["error_message"].(string)
	if !strings.Contains(errMsg, "overloaded") {
		t.Errorf("error_message 应包含上游原因, got %q", errMsg)
	}

	// 流式订阅也要给出明确错误帧，而不是静默结束
	rec := chatTaskRequest(t, router, http.MethodGet, "/api/chat-plus/tasks/"+taskID+"/stream?from=0", "")
	body := rec.Body.String()
	if !strings.Contains(body, "task_failed") {
		t.Errorf("订阅流缺少 task_failed 帧: %q", body)
	}
}

// 未配置密码 / 密码错误时不得建任务。
func TestChatPlusTaskRequiresPassword(t *testing.T) {
	router, _ := newChatTaskTestEnv(t, ollamaSSEUpstream("data: [DONE]\n\n"))

	req := httptest.NewRequest(http.MethodPost, "/api/chat-plus/tasks",
		strings.NewReader(`{"model":"ollama:qwen3:14b","messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("缺少密码时状态码 = %d, want 401", rec.Code)
	}
}

// 不支持的模型应在建任务时就拒绝，而不是建完再后台失败。
func TestChatPlusTaskRejectsUnknownModel(t *testing.T) {
	router, _ := newChatTaskTestEnv(t, ollamaSSEUpstream("data: [DONE]\n\n"))
	rec := chatTaskRequest(t, router, http.MethodPost, "/api/chat-plus/tasks",
		`{"model":"gpt-4o","messages":[]}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("未知模型状态码 = %d, want 400 (body=%s)", rec.Code, rec.Body.String())
	}
}

// 断点续传的核心：chunksSince 只回放 > from 的片段；跨 Write 的半行要能拼起来。
func TestChatTaskBufferResumeAndSplitLines(t *testing.T) {
	buf := newChatTaskBuffer("t1")
	buf.appendChunk(`data: {"choices":[{"delta":{"cont`)
	buf.appendChunk(`ent":"前半"}}]}` + "\n\n")
	buf.appendChunk(`data: {"choices":[{"delta":{"content":"后半"}}]}` + "\n\n")

	snap := buf.snapshot()
	if snap.Content != "前半后半" {
		t.Fatalf("跨片段正文拼接错误: %q", snap.Content)
	}
	if snap.Seq != 3 {
		t.Fatalf("seq = %d, want 3", snap.Seq)
	}

	got := buf.chunksSince(2)
	if len(got) != 1 || got[0].seq != 3 || !strings.Contains(got[0].data, "后半") {
		t.Fatalf("chunksSince(2) 应只回放第 3 段, got %+v", got)
	}
	if all := buf.chunksSince(0); len(all) != 3 {
		t.Fatalf("chunksSince(0) 应回放全部 3 段, got %d", len(all))
	}
	if none := buf.chunksSince(3); len(none) != 0 {
		t.Fatalf("chunksSince(3) 应为空, got %d", len(none))
	}
}

// 进度落库 + 进程重启后遗留 running 任务要被标失败（否则页面永远转圈）。
func TestChatPlusTaskStaleCleanup(t *testing.T) {
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitChatPlusTasks(); err != nil {
		t.Fatalf("init: %v", err)
	}
	task := &models.ChatPlusTask{ID: "ctask_stale", ConversationID: "c1", Model: "ollama:x", Status: "running"}
	if err := db.CreateChatPlusTask(task); err != nil {
		t.Fatalf("create: %v", err)
	}
	n, err := db.FailStaleChatPlusTasks(chatTaskStaleReason)
	if err != nil {
		t.Fatalf("FailStale: %v", err)
	}
	if n != 1 {
		t.Fatalf("清理数量 = %d, want 1", n)
	}
	got, err := db.GetChatPlusTask("ctask_stale")
	if err != nil || got == nil {
		t.Fatalf("读回任务失败: %v", err)
	}
	if got.Status != "failed" || got.ErrorMessage == "" {
		t.Errorf("遗留任务应标失败并带原因, got status=%s err=%q", got.Status, got.ErrorMessage)
	}
}

// 停止任务：取消后任务进入 canceled，已生成内容保留。
func TestChatPlusTaskStop(t *testing.T) {
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	router, _ := newChatTaskTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `data: {"choices":[{"delta":{"content":"开头"}}]}`+"\n\n")
		w.(http.Flusher).Flush()
		// 一直不结束，等测试取消（或客户端断开）
		select {
		case <-block:
		case <-r.Context().Done():
		}
	})

	taskID := createOllamaTask(t, router)
	time.Sleep(100 * time.Millisecond)

	rec := chatTaskRequest(t, router, http.MethodPost, "/api/chat-plus/tasks/"+taskID+"/stop", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("停止任务状态码 = %d, body=%s", rec.Code, rec.Body.String())
	}

	snap := waitChatTask(t, router, taskID)
	if snap["status"] != "canceled" {
		t.Errorf("status = %v, want canceled", snap["status"])
	}
	if content, _ := snap["content"].(string); !strings.Contains(content, "开头") {
		t.Errorf("取消后应保留已生成内容, got %q", content)
	}
}
