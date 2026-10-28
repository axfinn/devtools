package handlers

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"devtools/config"

	"github.com/gin-gonic/gin"
)

// bodyErrReadCloser 模拟"响应头 OK 但 body 传到一半断掉"的弱网场景。
type bodyErrReadCloser struct {
	err  error
	done bool
}

func (b *bodyErrReadCloser) Read(p []byte) (int, error) {
	if b.done {
		return 0, b.err
	}
	b.done = true
	// 先给半截数据再报错，等价于 io.ReadAll 返回 (partial, ErrUnexpectedEOF)
	return copy(p, `{"id":"half`), nil
}

func (b *bodyErrReadCloser) Close() error { return nil }

// truncatingRoundTripper 第一次返回截断的 body，之后正常。
type truncatingRoundTripper struct {
	mu       sync.Mutex
	attempts int
}

func (t *truncatingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.attempts++
	attempt := t.attempts
	t.mu.Unlock()

	if attempt == 1 {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       &bodyErrReadCloser{err: io.ErrUnexpectedEOF},
			Request:    req,
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"complete","choices":[]}`)),
		Request:    req,
	}, nil
}

func (t *truncatingRoundTripper) count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.attempts
}

// withStreamIdle 临时把全局流式静默阈值改小，测试结束恢复。
func withStreamIdle(t *testing.T, d time.Duration) {
	t.Helper()
	old := outboundPolicy
	outboundPolicy.streamIdle = d
	t.Cleanup(func() { outboundPolicy = old })
}

// 弱网下最常见的错误都应判定为可重试。
func TestIsTransientConnErrWeakNetwork(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"client canceled", context.Canceled, false},
		{"caller deadline", context.DeadlineExceeded, false},
		{"business error", errors.New("上游返回错误(400): bad request"), false},
		{"connection refused", &net.OpError{Err: syscall.ECONNREFUSED}, true},
		{"connection reset", &net.OpError{Err: syscall.ECONNRESET}, true},
		{"network unreachable", &net.OpError{Err: syscall.ENETUNREACH}, true},
		{"host unreachable", &net.OpError{Err: syscall.EHOSTUNREACH}, true},
		{"conn aborted", &net.OpError{Err: syscall.ECONNABORTED}, true},
		{"unexpected eof", io.ErrUnexpectedEOF, true},
		{"bare eof", io.EOF, true},
		{"tls handshake timeout", &net.OpError{Err: timeoutErr{}}, true},
		{"dns timeout", &net.DNSError{Err: "i/o timeout", Name: "api.deepseek.com", IsTimeout: true}, true},
		{"dns temporary", &net.DNSError{Err: "server misbehaving", Name: "api.deepseek.com", IsTemporary: true}, true},
		{"dns not found", &net.DNSError{Err: "no such host", Name: "api.deepseek.com"}, false},
	}
	for _, tc := range cases {
		if got := isTransientConnErr(tc.err); got != tc.want {
			t.Errorf("%s: isTransientConnErr = %v, want %v", tc.name, got, tc.want)
		}
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

// 指数退避应有界、随重试增长，且带抖动（不会所有请求同一时刻重试）。
func TestBackoffForIsBoundedAndGrows(t *testing.T) {
	old := outboundPolicy
	outboundPolicy.baseBackoff = 100 * time.Millisecond
	outboundPolicy.maxBackoff = 800 * time.Millisecond
	defer func() { outboundPolicy = old }()

	base, capD := 100*time.Millisecond, 800*time.Millisecond
	for attempt := 0; attempt < 6; attempt++ {
		expect := base << attempt
		if expect > capD || expect <= 0 {
			expect = capD
		}
		floor := expect / 2
		lo, hi := time.Hour, time.Duration(0)
		for i := 0; i < 60; i++ {
			d := backoffFor(attempt)
			if d < lo {
				lo = d
			}
			if d > hi {
				hi = d
			}
		}
		if hi > capD {
			t.Fatalf("attempt %d 退避 %s 超过上限 %s", attempt, hi, capD)
		}
		if lo < floor {
			t.Fatalf("attempt %d 退避 %s 低于下限 %s", attempt, lo, floor)
		}
		if lo == hi {
			t.Fatalf("attempt %d 退避没有抖动，全部为 %s", attempt, lo)
		}
	}
	// 到达上限后必须封顶
	if got := backoffFor(20); got > outboundPolicy.maxBackoff {
		t.Fatalf("高次数退避未封顶: %s", got)
	}
}

// 配置里的弱网参数应生效，越界值要被夹紧。
func TestApplyOutboundRetryConfigClamps(t *testing.T) {
	old := outboundPolicy
	defer func() { outboundPolicy = old }()

	applyOutboundRetryConfig(config.AIGatewayRetryConfig{
		MaxAttempts:    99,
		BaseBackoffMS:  1,
		MaxBackoffMS:   10,
		RetryBudgetSec: 1,
		StreamIdleSec:  1,
		DialTimeoutSec: 999,
	})
	if outboundPolicy.attempts != 8 {
		t.Errorf("attempts = %d, want 8 (clamped)", outboundPolicy.attempts)
	}
	if outboundPolicy.baseBackoff != 20*time.Millisecond {
		t.Errorf("baseBackoff = %s, want 20ms (clamped)", outboundPolicy.baseBackoff)
	}
	if outboundPolicy.maxBackoff < outboundPolicy.baseBackoff {
		t.Errorf("maxBackoff(%s) 必须 >= baseBackoff(%s)", outboundPolicy.maxBackoff, outboundPolicy.baseBackoff)
	}
	if outboundPolicy.streamIdle != 10*time.Second {
		t.Errorf("streamIdle = %s, want 10s (clamped)", outboundPolicy.streamIdle)
	}
	if outboundPolicy.dialTimeout != 60*time.Second {
		t.Errorf("dialTimeout = %s, want 60s (clamped)", outboundPolicy.dialTimeout)
	}

	// 全 0 = 保留默认值
	outboundPolicy = old
	applyOutboundRetryConfig(config.AIGatewayRetryConfig{})
	if outboundPolicy != old {
		t.Errorf("空配置不应改变策略: got %+v, want %+v", outboundPolicy, old)
	}
}

// 响应体被截断时应重试（旧实现把 ReadAll 的错误吞掉了，只会报 JSON 解析失败）。
func TestDoJSONRequestRetriesOnTruncatedBody(t *testing.T) {
	rt := &truncatingRoundTripper{}
	h := &AIGatewayHandler{longNoProxyClient: &http.Client{Transport: rt}}

	payload, err := h.doJSONRequest("http://upstream/chat/completions", "sk-test", map[string]interface{}{"model": "deepseek-chat"})
	if err != nil {
		t.Fatalf("截断响应应重试并成功，实际错误: %v", err)
	}
	if payload["id"] != "complete" {
		t.Errorf("payload id = %v, want complete", payload["id"])
	}
	if got := rt.count(); got != 2 {
		t.Errorf("expected 2 attempts (truncated + retry), got %d", got)
	}
}

// 上游返回 200 但 body 完全无法解析时，应给出可读错误而不是 nil。
func TestDoJSONRequestEmptyBodyReportsError(t *testing.T) {
	h := &AIGatewayHandler{longNoProxyClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})}}

	payload, err := h.doJSONRequest("http://upstream/chat/completions", "sk-test", map[string]interface{}{})
	if err == nil {
		t.Fatal("空响应应返回错误")
	}
	if payload != nil {
		t.Errorf("payload 应为 nil, got %v", payload)
	}
	if !strings.Contains(err.Error(), "弱网") {
		t.Errorf("错误信息应提示弱网/重试, got: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// stallGuard 到点应触发回调，且 touch 能续命。
func TestStallGuardFiresAndExtends(t *testing.T) {
	fired := make(chan struct{}, 1)
	g := newStallGuard(150*time.Millisecond, func() { fired <- struct{}{} })

	for i := 0; i < 3; i++ {
		time.Sleep(80 * time.Millisecond)
		g.touch()
	}
	select {
	case <-fired:
		t.Fatal("持续有数据时不应判定静默")
	default:
	}

	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		t.Fatal("静默超时后应触发回调")
	}
	if !g.stop() {
		t.Error("stop 应报告曾判定静默")
	}
}

func TestStallGuardStopBeforeFire(t *testing.T) {
	g := newStallGuard(50*time.Millisecond, func() { t.Error("已停止的 guard 不应触发") })
	if g.stop() {
		t.Error("未静默时 stop 应为 false")
	}
	time.Sleep(120 * time.Millisecond)
}

// 流式上游中途静默时，应向前端发 upstream_stalled 错误帧并及时收尾，
// 而不是靠着心跳无限等待（旧实现会一直挂着）。
func TestStreamStallEmitsErrorFrame(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withStreamIdle(t, 200*time.Millisecond)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "data: {\"id\":\"1\",\"choices\":[{\"delta\":{\"content\":\"半句\"}}]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done() // 静默，直到被看门狗取消
	}))
	defer upstream.Close()

	h := &AIGatewayHandler{streamClient: &http.Client{}}
	req := ChatCompletionRequest{
		Model:    "deepseek-chat",
		Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/ai-gateway/v1/chat/completions", strings.NewReader(`{}`))

	done := make(chan struct{})
	go func() {
		h.streamOpenAICompatible(c, req, upstream.URL, "sk-test")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("静默断流后 handler 未退出（看门狗失效）")
	}

	out := rec.Body.String()
	if !strings.Contains(out, "upstream_stalled") {
		t.Errorf("缺少断流错误帧, got: %q", out)
	}
	if !strings.Contains(out, "半句") {
		t.Errorf("已收到的内容应保留, got: %q", out)
	}
}

// 连接池参数：弱网下要更短的空闲回收，避免复用半死连接。
func TestStreamTransportTuning(t *testing.T) {
	tr := buildStreamTransport(60)
	if tr.IdleConnTimeout != 45*time.Second {
		t.Errorf("IdleConnTimeout = %s, want 45s", tr.IdleConnTimeout)
	}
	if tr.MaxConnsPerHost != 32 {
		t.Errorf("MaxConnsPerHost = %d, want 32", tr.MaxConnsPerHost)
	}
	if tr.TLSHandshakeTimeout == 0 {
		t.Error("TLSHandshakeTimeout 不能为 0（会无限等待）")
	}
}
