package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"time"

	"devtools/config"
)

// 本文件集中承载"出网请求在弱网下如何不轻易失败"的策略：
//  1. 哪些错误值得重试（连接被拒/重置/中断、DNS 抖动、握手超时、上游提前断流）；
//  2. 重试几次、退避多久（指数退避 + 抖动，且有总预算上限，避免 5 分钟长请求被放大成 20 分钟）；
//  3. 流式响应中途静默（弱网断流最典型的症状）如何检测并明确告知调用方；
//  4. 连接池/keepalive 参数，避免复用已经"半死"的连接导致每次卡满超时。

const (
	defaultOutboundAttempts    = 4
	defaultOutboundBaseBackoff = 250 * time.Millisecond
	defaultOutboundMaxBackoff  = 4 * time.Second
	defaultOutboundBudget      = 120 * time.Second
	defaultStreamIdleTimeout   = 120 * time.Second
	defaultOutboundDialTimeout = 10 * time.Second
)

// outboundRetryPolicy 全局出网重试策略；启动时由 config.ai_gateway.retry 覆盖。
type outboundRetryPolicy struct {
	attempts    int           // 含首次请求的最大尝试次数
	baseBackoff time.Duration // 首次退避基数
	maxBackoff  time.Duration // 单次退避上限
	budget      time.Duration // 所有重试的总耗时预算（从首次尝试开始计）
	streamIdle  time.Duration // 流式上游"静默"判定阈值
	dialTimeout time.Duration // TCP 拨号超时
}

var outboundPolicy = outboundRetryPolicy{
	attempts:    defaultOutboundAttempts,
	baseBackoff: defaultOutboundBaseBackoff,
	maxBackoff:  defaultOutboundMaxBackoff,
	budget:      defaultOutboundBudget,
	streamIdle:  defaultStreamIdleTimeout,
	dialTimeout: defaultOutboundDialTimeout,
}

// applyOutboundRetryConfig 把配置里的弱网参数合并进全局策略，越界值会被夹紧到安全范围。
func applyOutboundRetryConfig(cfg config.AIGatewayRetryConfig) {
	p := outboundPolicy

	if cfg.MaxAttempts > 0 {
		p.attempts = clampInt(cfg.MaxAttempts, 1, 8)
	}
	if cfg.BaseBackoffMS > 0 {
		p.baseBackoff = time.Duration(clampInt(cfg.BaseBackoffMS, 20, 5000)) * time.Millisecond
	}
	if cfg.MaxBackoffMS > 0 {
		p.maxBackoff = time.Duration(clampInt(cfg.MaxBackoffMS, 100, 30000)) * time.Millisecond
	}
	if p.maxBackoff < p.baseBackoff {
		p.maxBackoff = p.baseBackoff
	}
	if cfg.RetryBudgetSec > 0 {
		p.budget = time.Duration(clampInt(cfg.RetryBudgetSec, 5, 900)) * time.Second
	}
	if cfg.StreamIdleSec > 0 {
		p.streamIdle = time.Duration(clampInt(cfg.StreamIdleSec, 10, 1800)) * time.Second
	}
	if cfg.DialTimeoutSec > 0 {
		p.dialTimeout = time.Duration(clampInt(cfg.DialTimeoutSec, 2, 60)) * time.Second
	}
	outboundPolicy = p
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// backoffFor 返回第 attempt 次重试前的等待时间（attempt 从 0 开始）。
// 指数增长 + "equal jitter"：既避免所有请求同步重试打爆上游，也不会退避到 0。
func backoffFor(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	d := outboundPolicy.baseBackoff
	for i := 0; i < attempt && d < outboundPolicy.maxBackoff; i++ {
		d *= 2
	}
	if d > outboundPolicy.maxBackoff {
		d = outboundPolicy.maxBackoff
	}
	half := d / 2
	if half <= 0 {
		return d
	}
	return half + time.Duration(rand.Int63n(int64(half)+1))
}

// isTransientConnErr 判断错误是否属于"可安全重试"的瞬时网络故障。
// 客户端主动取消（context.Canceled）与自己设的 deadline 到期不重试，避免无意义放大。
func isTransientConnErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// 连接建立/复用/读取阶段的典型断链
	for _, target := range []error{
		syscall.ECONNREFUSED,
		syscall.ECONNRESET,
		syscall.ECONNABORTED,
		syscall.EPIPE,
		syscall.ETIMEDOUT,
		syscall.ENETUNREACH,
		syscall.EHOSTUNREACH,
		io.ErrUnexpectedEOF,
		io.EOF,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	// DNS 抖动：弱网下 "lookup api.xxx: i/o timeout" 非常常见，换一次解析往往就好了
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && (dnsErr.IsTimeout || dnsErr.IsTemporary) {
		return true
	}
	// 拨号 / TLS 握手 / 读超时（net.Error.Timeout 覆盖 *url.Error 与 *net.OpError）
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// 兜底：不同 Go 版本/底层库对"上游提前关闭"的措辞不统一
	msg := err.Error()
	return strings.Contains(msg, "unexpected EOF") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "broken pipe")
}

// readResponseBody 读取上游响应体，并把读取失败（弱网断流最常见的表现）显式暴露出来。
// 之前直接 `respBody, _ := io.ReadAll(...)` 会把截断的 body 当成功，
// 后续 JSON 解析报 "unexpected end of JSON input"，既看不出是网络问题也不会触发重试。
func readResponseBody(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return body, fmt.Errorf("读取上游响应中断(可能是弱网断流): %w", err)
	}
	return body, nil
}

// stallGuard 检测流式读取"长时间没有任何字节"的情况。
// http.Client.Timeout 为 0（SSE 长流不能设总超时），ResponseHeaderTimeout 只管响应头，
// 因此一旦连接在半途静默，读操作会永久阻塞——加上客户端心跳后前端只会看到"一直在想"。
// guard 在静默超过阈值时触发 onStall（通常是 cancel 上游请求），让 Read 立即返回。
type stallGuard struct {
	mu      sync.Mutex
	timer   *time.Timer
	idle    time.Duration
	stalled bool
	stopped bool
	onStall func()
}

func newStallGuard(idle time.Duration, onStall func()) *stallGuard {
	if idle <= 0 {
		idle = defaultStreamIdleTimeout
	}
	g := &stallGuard{idle: idle, onStall: onStall}
	g.timer = time.AfterFunc(idle, g.markStalled)
	return g
}

func (g *stallGuard) markStalled() {
	g.mu.Lock()
	if g.stopped {
		g.mu.Unlock()
		return
	}
	g.stalled = true
	cb := g.onStall
	g.mu.Unlock()
	if cb != nil {
		cb()
	}
}

// touch 每读到数据就重置静默计时。
func (g *stallGuard) touch() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stopped || g.stalled {
		return
	}
	g.timer.Reset(g.idle)
}

// stop 结束计时并返回是否曾判定为静默。可重复调用。
func (g *stallGuard) stop() bool {
	if g == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.stopped = true
	g.timer.Stop()
	return g.stalled
}

// stallMessage 给调用方看的中断原因，便于前端提示"重试"而不是空白。
func stallMessage(idle time.Duration) string {
	return fmt.Sprintf("上游 %s 无任何数据返回，已判定为网络断流（弱网）。请重试。", idle.Round(time.Second))
}
