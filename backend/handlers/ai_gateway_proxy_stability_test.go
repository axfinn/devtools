package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// flakyRoundTripper 前 failFirstAttempts 次返回瞬时连接错误（ECONNREFUSED），之后正常响应。
// 用于验证代理层对瞬时网络错误的有限重试。
type flakyRoundTripper struct {
	mu               sync.Mutex
	attempts         int
	failFirstAttempts int
	responder        func(*http.Request) (*http.Response, error)
}

func (f *flakyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.attempts++
	attempt := f.attempts
	f.mu.Unlock()

	if attempt <= f.failFirstAttempts {
		return nil, &url.Error{Op: "Post", URL: req.URL.String(), Err: syscall.ECONNREFUSED}
	}
	if f.responder != nil {
		return f.responder(req)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"x","choices":[]}`)),
		Request:    req,
	}, nil
}

func (f *flakyRoundTripper) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attempts
}

// isTransientConnErr 的纯函数表驱动测试：瞬时连接错误 → true，客户端取消/普通错误 → false。
func TestIsTransientConnErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"context canceled", context.Canceled, false},
		{"context deadline", context.DeadlineExceeded, false},
		{"plain error", &testError{"boom"}, false},
		{"connection refused (url.Error)", &url.Error{Op: "Post", URL: "x", Err: syscall.ECONNREFUSED}, true},
		{"connection reset", &url.Error{Op: "Post", URL: "x", Err: syscall.ECONNRESET}, true},
		{"dial timeout (opError)", &url.Error{Op: "Post", URL: "x", Err: &netOpTimeout{}}, true},
	}
	for _, tc := range cases {
		if got := isTransientConnErr(tc.err); got != tc.want {
			t.Errorf("%s: isTransientConnErr = %v, want %v", tc.name, got, tc.want)
		}
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

// netOpTimeout 实现 net.Error（Timeout()==true），模拟拨号/握手超时。
type netOpTimeout struct{}

func (e *netOpTimeout) Error() string   { return "i/o timeout" }
func (e *netOpTimeout) Timeout() bool   { return true }
func (e *netOpTimeout) Temporary() bool { return true }

// doJSONRequest 对瞬时连接错误应重试一次并成功。
func TestDoJSONRequestRetriesOnTransient(t *testing.T) {
	rt := &flakyRoundTripper{failFirstAttempts: 1}
	h := &AIGatewayHandler{longNoProxyClient: &http.Client{Transport: rt}}

	payload, err := h.doJSONRequest("http://upstream/chat/completions", "sk-test", map[string]interface{}{"model": "deepseek-chat"})
	if err != nil {
		t.Fatalf("doJSONRequest failed after retry: %v", err)
	}
	if payload["id"] != "x" {
		t.Errorf("payload id = %v, want x", payload["id"])
	}
	if got := rt.count(); got != 2 {
		t.Errorf("expected 2 attempts (1 fail + 1 retry), got %d", got)
	}
}

// doJSONRequest 对客户端取消（context.Canceled）不应重试。
func TestDoJSONRequestNoRetryOnCanceled(t *testing.T) {
	rt := &flakyRoundTripper{failFirstAttempts: 1}
	rt.responder = func(*http.Request) (*http.Response, error) {
		return nil, context.Canceled
	}
	h := &AIGatewayHandler{longNoProxyClient: &http.Client{Transport: rt}}

	_, err := h.doJSONRequest("http://upstream/chat/completions", "sk-test", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := rt.count(); got != 1 {
		t.Errorf("expected no retry on context.Canceled (1 attempt), got %d", got)
	}
}

// doStreamRequest 在拿到响应前遇到瞬时连接错误应重试一次。
func TestDoStreamRequestRetriesOnTransient(t *testing.T) {
	rt := &flakyRoundTripper{failFirstAttempts: 1}
	client := &http.Client{Transport: rt}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://upstream/v1/chat/completions", bytes.NewReader([]byte(`{"stream":true}`)))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := doStreamRequest(client, req)
	if err != nil {
		t.Fatalf("doStreamRequest failed after retry: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if got := rt.count(); got != 2 {
		t.Errorf("expected 2 attempts, got %d", got)
	}
}

// doRequestWithClient（Anthropic 非流式代理路径）对瞬时连接错误应重试一次。
func TestDoRequestWithClientRetriesOnTransient(t *testing.T) {
	rt := &flakyRoundTripper{failFirstAttempts: 1}
	client := &http.Client{Transport: rt}
	h := &AIGatewayHandler{}

	body, contentType, err := h.doRequestWithClient(client, "http://upstream/v1/messages", "sk-test", http.MethodPost, []byte(`{"model":"deepseek-reasoner"}`), nil, true)
	if err != nil {
		t.Fatalf("doRequestWithClient failed after retry: %v", err)
	}
	if !strings.Contains(string(body), `"id"`) {
		t.Errorf("unexpected body: %s", body)
	}
	if contentType == "" {
		t.Error("content-type should be application/json")
	}
	if got := rt.count(); got != 2 {
		t.Errorf("expected 2 attempts, got %d", got)
	}
}

// streamOpenAICompatible（DeepSeek/DashScope 流式路径）应写入首条心跳并透传上游 SSE 数据。
func TestStreamOpenAICompatibleWritesHeartbeatAndData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "data: {\"id\":\"1\"}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	h := &AIGatewayHandler{streamClient: &http.Client{Timeout: 5 * time.Second}}
	req := ChatCompletionRequest{
		Model:    "deepseek-chat",
		Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/ai-gateway/v1/chat/completions", strings.NewReader(`{}`))

	h.streamOpenAICompatible(c, req, upstream.URL, "sk-test")

	out := rec.Body.String()
	if !strings.Contains(out, ": heartbeat") {
		t.Errorf("missing initial heartbeat, got: %q", out)
	}
	if !strings.Contains(out, "data: {\"id\":\"1\"}") {
		t.Errorf("upstream SSE data not passed through, got: %q", out)
	}
	if !strings.Contains(out, "data: [DONE]") {
		t.Errorf("missing [DONE] marker, got: %q", out)
	}
}
