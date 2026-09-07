package handlers

import (
	"net/http"
	"testing"
	"time"

	"devtools/config"
)

// 修复：流式上游的 streamTransport.ResponseHeaderTimeout 之前写死 60s。
// Ollama 本地 27B 在 15K+ tokens system prompt 下首字节常超 60s,导致 Claude Code
// 通过 devtools 网关调用本地 Ollama 时被掐死(Gin 日志显示 1m0s 截断)。
// 修复:按 provider 派生 streamClient,provider.HeaderTimeoutSec=0 时回落 60s,
// >0 时用该值(如 Ollama 600s)。

func TestBuildStreamTransport_DefaultWhenZero(t *testing.T) {
	tr := buildStreamTransport(0)
	if tr.ResponseHeaderTimeout != 60*time.Second {
		t.Errorf("header_timeout_sec=0 should fall back to 60s, got %v", tr.ResponseHeaderTimeout)
	}
}

func TestBuildStreamTransport_NegativeFallsBack(t *testing.T) {
	tr := buildStreamTransport(-1)
	if tr.ResponseHeaderTimeout != 60*time.Second {
		t.Errorf("negative header_timeout_sec should fall back to 60s, got %v", tr.ResponseHeaderTimeout)
	}
}

func TestBuildStreamTransport_CustomValue(t *testing.T) {
	tr := buildStreamTransport(600)
	if tr.ResponseHeaderTimeout != 600*time.Second {
		t.Errorf("header_timeout_sec=600 should produce 600s, got %v", tr.ResponseHeaderTimeout)
	}
}

func TestPickStreamClient_NilProviderUsesFallback(t *testing.T) {
	h := &AIGatewayHandler{
		streamClient: &http.Client{Timeout: 7 * time.Second},
	}
	got := h.pickStreamClient(nil)
	if got != h.streamClient {
		t.Errorf("nil provider should return h.streamClient fallback, got %p want %p", got, h.streamClient)
	}
}

func TestPickStreamClient_DerivesPerProvider(t *testing.T) {
	h := &AIGatewayHandler{
		cfg: config.DefaultConfig(),
		// 全局 streamClient 必须存在;pickStreamClient 在缓存命中失败时会派生新的
		streamClient: &http.Client{Timeout: 0},
	}

	fast := &config.AnthropicProviderConfig{Name: "CloudFast", HeaderTimeoutSec: 0}
	slow := &config.AnthropicProviderConfig{Name: "OllamaLocal", HeaderTimeoutSec: 600}

	fastClient := h.pickStreamClient(fast)
	slowClient := h.pickStreamClient(slow)

	if fastClient == slowClient {
		t.Fatal("fast and slow providers should produce different http.Client instances")
	}
	fastTr, ok := fastClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("fast client transport type = %T, want *http.Transport", fastClient.Transport)
	}
	if fastTr.ResponseHeaderTimeout != 60*time.Second {
		t.Errorf("fast (HeaderTimeoutSec=0) should produce 60s transport, got %v", fastTr.ResponseHeaderTimeout)
	}
	slowTr, ok := slowClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("slow client transport type = %T, want *http.Transport", slowClient.Transport)
	}
	if slowTr.ResponseHeaderTimeout != 600*time.Second {
		t.Errorf("Ollama (HeaderTimeoutSec=600) should produce 600s transport, got %v", slowTr.ResponseHeaderTimeout)
	}
}

func TestPickStreamClient_CachesSameKey(t *testing.T) {
	h := &AIGatewayHandler{
		cfg:          config.DefaultConfig(),
		streamClient: &http.Client{Timeout: 0},
	}
	p := &config.AnthropicProviderConfig{Name: "OllamaLocal", HeaderTimeoutSec: 600}

	first := h.pickStreamClient(p)
	second := h.pickStreamClient(p)
	if first != second {
		t.Errorf("same (name, timeout) should hit cache, got distinct clients %p vs %p", first, second)
	}
}

func TestPickStreamClient_DifferentTimeoutRebuilds(t *testing.T) {
	h := &AIGatewayHandler{
		cfg:          config.DefaultConfig(),
		streamClient: &http.Client{Timeout: 0},
	}
	p1 := &config.AnthropicProviderConfig{Name: "OllamaLocal", HeaderTimeoutSec: 600}
	p2 := &config.AnthropicProviderConfig{Name: "OllamaLocal", HeaderTimeoutSec: 120}

	c1 := h.pickStreamClient(p1)
	c2 := h.pickStreamClient(p2)
	if c1 == c2 {
		t.Errorf("different header_timeout_sec should rebuild transport, got same client %p", c1)
	}
	c2Tr := c2.Transport.(*http.Transport)
	if c2Tr.ResponseHeaderTimeout != 120*time.Second {
		t.Errorf("120s provider should produce 120s transport, got %v", c2Tr.ResponseHeaderTimeout)
	}
}
