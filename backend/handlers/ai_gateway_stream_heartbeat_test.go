package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"devtools/config"

	"github.com/gin-gonic/gin"
)

// 本文件覆盖 SSE 心跳 goroutine 的 join 语义(commit c3119bd 引入,自带测试 = 0)。
//
// proxyAnthropicStream 起两个心跳 goroutine:
//   1) 拨号期高频心跳(1s):上游还没回响应头时先撑住客户端 read timeout
//   2) 生成期心跳(5s):thinking 阶段无字节输出时防中间代理空闲断流
//
// 两者都往同一个 sseWriter 写,必须满足:
//   - handler 返回前把 goroutine 收干净(stopPreHeartbeat / heartbeatWG.Wait())
//   - handler 返回后不再有任何写入(否则会写进已完成的 ResponseWriter,
//     表现为流内容错乱 / "superfluous response.WriteHeader")
//   - 并发写由 sseWriter 的锁串行化,不能出现半帧交错

// countWriter 记录写入次数与内容的 ResponseWriter(实现 http.Flusher,
// gin 的 responseWriter.Flush() 会对底层做 http.Flusher 断言)。
type countWriter struct {
	mu     sync.Mutex
	header http.Header
	buf    bytes.Buffer
	writes int
}

func newCountWriter() *countWriter {
	return &countWriter{header: make(http.Header)}
}

func (w *countWriter) Header() http.Header { return w.header }

func (w *countWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writes++
	return w.buf.Write(p)
}

func (w *countWriter) WriteHeader(int) {}

func (w *countWriter) Flush() {}

func (w *countWriter) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writes
}

func (w *countWriter) body() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// newHeartbeatTestHandler 造一个只用于流式测试的 handler。
func newHeartbeatTestHandler(t *testing.T) *AIGatewayHandler {
	t.Helper()
	db := setupAIGatewayEncryptDB(t)
	t.Cleanup(func() { _ = db.Close() })
	return NewAIGatewayHandler(db, config.DefaultConfig(), nil, newEncryptEnc(t))
}

// newStreamTestContext 造一个带 ResponseWriter 计数器的 gin 上下文。
func newStreamTestContext(cw *countWriter) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(cw)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/anthropic/v1/messages", nil)
	return c
}

// TestAnthropicStreamPreUpstreamHeartbeatJoin 拨号期心跳(1s)必须在 handler 返回前
// 被 join 掉:返回之后再等 1.5s(>1 个心跳周期)不应出现任何新写入。
//
// 上游故意延迟 2.5s 才回响应头 → 拨号期心跳至少触发 2 次,
// 保证"返回后无写入"这条断言不是空断言。
func TestAnthropicStreamPreUpstreamHeartbeatJoin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"upstream boom"}`))
	}))
	defer srv.Close()

	h := newHeartbeatTestHandler(t)
	provider := &config.AnthropicProviderConfig{
		Name:             "HeartbeatPreTest",
		APIURL:           srv.URL,
		APIKey:           "test-key",
		HeaderTimeoutSec: 30,
	}

	cw := newCountWriter()
	c := newStreamTestContext(cw)

	status, err := h.proxyAnthropicStream(c, provider, srv.URL+"/v1/messages", []byte(`{}`))
	if status != http.StatusInternalServerError {
		t.Fatalf("proxyAnthropicStream 状态码 = %d, want 500(上游返回 500); err=%v", status, err)
	}
	if err == nil {
		t.Errorf("上游 500 时应同时返回 error")
	}

	// 拨号期心跳必须真的触发过,否则下面的"无新写入"是空断言
	if pings := strings.Count(cw.body(), "event: ping"); pings < 2 {
		t.Fatalf("拨号期心跳未触发(event: ping 只出现 %d 次, 期望 >= 2), 断言会失去意义; body=%q", pings, cw.body())
	}

	before := cw.count()
	time.Sleep(1500 * time.Millisecond) // > 1 个拨号期心跳周期
	if after := cw.count(); after != before {
		t.Errorf("handler 返回后仍有 %d 次写入(拨号期心跳 goroutine 未 join)", after-before)
	}
}

// TestAnthropicStreamGenerationHeartbeatJoin 生成期心跳(5s)必须在 handler 返回前
// 被 join 掉:返回之后再等 5.2s(>1 个心跳周期)不应出现任何新写入。
//
// 上游先回 SSE 响应头,再把 body 挂住 5.5s —— 生成期心跳在第 5s 触发一次,
// 保证"返回后无写入"这条断言不是空断言。
func TestAnthropicStreamGenerationHeartbeatJoin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\"}\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(5500 * time.Millisecond) // 挂住,让生成期心跳(5s)触发
	}))
	defer srv.Close()

	h := newHeartbeatTestHandler(t)
	provider := &config.AnthropicProviderConfig{
		Name:             "HeartbeatGenTest",
		APIURL:           srv.URL,
		APIKey:           "test-key",
		HeaderTimeoutSec: 30,
	}

	cw := newCountWriter()
	c := newStreamTestContext(cw)

	status, err := h.proxyAnthropicStream(c, provider, srv.URL+"/v1/messages", []byte(`{}`))
	if status != http.StatusOK {
		t.Fatalf("proxyAnthropicStream 状态码 = %d, want 200; err=%v", status, err)
	}
	if err != nil {
		t.Errorf("上游正常结束不应返回 error, 实际: %v", err)
	}

	// 上游内容必须透传出去
	if !strings.Contains(cw.body(), "message_start") {
		t.Errorf("上游 SSE 内容未透传; body=%q", cw.body())
	}
	// 生成期心跳必须真的触发过(初始 ping + 第 5s 的心跳 ping => >= 2)
	if pings := strings.Count(cw.body(), "event: ping"); pings < 2 {
		t.Fatalf("生成期心跳未触发(event: ping 只出现 %d 次, 期望 >= 2), 断言会失去意义; body=%q", pings, cw.body())
	}

	before := cw.count()
	time.Sleep(5200 * time.Millisecond) // > 1 个生成期心跳周期
	if after := cw.count(); after != before {
		t.Errorf("handler 返回后仍有 %d 次写入(生成期心跳 goroutine 未 join)", after-before)
	}
}

// TestSSEWriterConcurrentWriteIsSerialized sseWriter 的并发写必须被串行化:
// 心跳 goroutine 与主复制循环会并发 Write+Flush,不加锁时 net/http 的
// Flush 与 Write 在底层 bufio 上存在数据竞争,表现为半帧交错。
//
// 本用例并发写 200 次,断言 (a) 写入次数不丢 (b) 输出里 200 个帧都完整。
// 配合 `go test -race` 可直接暴露数据竞争。
func TestSSEWriterConcurrentWriteIsSerialized(t *testing.T) {
	cw := newCountWriter()
	w := newSSEWriter(cw, cw)

	const goroutines = 8
	const perGoroutine = 25
	frame := "event: ping\ndata: {\"type\":\"ping\"}\n\n"

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				if err := w.write([]byte(frame)); err != nil {
					t.Errorf("sseWriter.write: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	want := goroutines * perGoroutine
	if got := cw.count(); got != want {
		t.Errorf("写入次数 = %d, want %d", got, want)
	}
	if got := strings.Count(cw.body(), "event: ping"); got != want {
		t.Errorf("完整帧数 = %d, want %d(并发写被交错撕裂了)", got, want)
	}
	if got := len(cw.body()); got != want*len(frame) {
		t.Errorf("输出字节数 = %d, want %d(存在半帧交错)", got, want*len(frame))
	}
}

// TestAnthropicStreamHeartbeatLeakWouldBeDetected 反事实校验(可证伪性保障):
// 证明上面两个 join 用例依赖的"写入计数"确实能抓到 handler 返回后的写入 ——
// 这里用一个模拟"泄漏心跳 goroutine"的循环写,断言计数会增长。
//
// 若这个用例失败,说明计数探针不可信,上面两个 join 用例的"无新写入"断言就是恒真闸门。
func TestAnthropicStreamHeartbeatLeakWouldBeDetected(t *testing.T) {
	cw := newCountWriter()
	w := newSSEWriter(cw, cw)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = w.write([]byte(ssePingFrame))
			case <-stop:
				return
			}
		}
	}()

	before := cw.count()
	time.Sleep(700 * time.Millisecond)
	close(stop)
	wg.Wait()

	if after := cw.count(); after <= before {
		t.Errorf("模拟泄漏的心跳没有产生写入(前后计数 %d -> %d), 计数探针不可信", before, after)
	}
}
