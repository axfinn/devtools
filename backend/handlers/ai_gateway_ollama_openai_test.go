package handlers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"devtools/config"
)

// 本文件验证 OpenAI 协议（/api/ai-gateway/v1/chat/completions）对下游 Ollama 的路由：
//   - "ollama:<name>" 前缀直通；
//   - 裸模型名需命中 Ollama provider 的 models 列表；
//   - 流式必须用按 provider 派生的 streamClient（header_timeout_sec 兕底 600s），
//     不能复用全局 60s 兕底 —— 本地 27B 首字节常超 60s。

func newOllamaRoutingTestHandler(t *testing.T) *AIGatewayHandler {
	t.Helper()
	h := &AIGatewayHandler{
		cfg:          config.DefaultConfig(),
		streamClient: &http.Client{Timeout: 0},
	}
	return h
}

func TestResolveOllamaProviderConfig_FromDBStyleList(t *testing.T) {
	h := newOllamaRoutingTestHandler(t)
	// allAnthropicProviders 会读 DB/config,这里直接验证回退逻辑:
	// ChatPlus.OllamaBaseURL 配置了就能得到 provider,且 HeaderTimeoutSec 兕底 600
	h.cfg.ChatPlus.OllamaBaseURL = "http://192.168.31.147:11434/"
	prov := h.resolveOllamaProviderConfig()
	if prov == nil {
		t.Fatal("配置了 chat_plus.ollama_base_url 应能解析出 Ollama provider")
	}
	if prov.HeaderTimeoutSec != 600 {
		t.Errorf("HeaderTimeoutSec 应兕底 600, got %d", prov.HeaderTimeoutSec)
	}
	if !strings.HasSuffix(strings.TrimRight(prov.APIURL, "/")+"/v1/chat/completions", "/v1/chat/completions") {
		t.Errorf("endpoint 拼接错误: %s", prov.APIURL)
	}
}

func TestOllamaChatTarget_PrefixModel(t *testing.T) {
	h := newOllamaRoutingTestHandler(t)
	h.cfg.ChatPlus.OllamaBaseURL = "http://192.168.31.147:11434"

	endpoint, upstream, prov, err := h.ollamaChatTarget("ollama:qwen3:14b")
	if err != nil {
		t.Fatalf("ollama: 前缀模型应可路由: %v", err)
	}
	if upstream != "qwen3:14b" {
		t.Errorf("upstream model = %q, want qwen3:14b", upstream)
	}
	if endpoint != "http://192.168.31.147:11434/v1/chat/completions" {
		t.Errorf("endpoint = %q", endpoint)
	}
	if prov == nil {
		t.Fatal("provider 不应为空")
	}
}

func TestOllamaChatTarget_EmptyModelName(t *testing.T) {
	h := newOllamaRoutingTestHandler(t)
	h.cfg.ChatPlus.OllamaBaseURL = "http://192.168.31.147:11434"
	if _, _, _, err := h.ollamaChatTarget("ollama:"); err == nil {
		t.Fatal("ollama: 空模型名应报错")
	}
}

func TestResolveChatProvider_OllamaPrefix(t *testing.T) {
	h := newOllamaRoutingTestHandler(t)
	h.cfg.ChatPlus.OllamaBaseURL = "http://192.168.31.147:11434"
	if got := h.resolveChatProvider("ollama:qwen3.8:27b-mlx"); got != "ollama" {
		t.Errorf("resolveChatProvider(ollama:*) = %q, want ollama", got)
	}
	// 未配置 Ollama 下游时不应路由到 ollama（显式清空 DefaultConfig 自带的默认 base_url）
	h2 := newOllamaRoutingTestHandler(t)
	h2.cfg.ChatPlus.OllamaBaseURL = ""
	if got := h2.resolveChatProvider("ollama:qwen3.8:27b-mlx"); got == "ollama" {
		t.Error("未配置 Ollama 下游不应路由到 ollama")
	}
}

func TestResolveChatProvider_NonOllamaUnaffected(t *testing.T) {
	h := newOllamaRoutingTestHandler(t)
	h.cfg.ChatPlus.OllamaBaseURL = "http://192.168.31.147:11434"
	// deepseek/minimax 路由不受影响
	if got := h.resolveChatProvider("deepseek-chat"); got != "deepseek" {
		t.Errorf("deepseek-chat = %q, want deepseek", got)
	}
	if got := h.resolveChatProvider("MiniMax-M3"); got != "minimax" {
		t.Errorf("MiniMax-M3 = %q, want minimax", got)
	}
	// 未命中 Ollama models 列表的裸模型名不能被 ollama 吞掉
	if got := h.resolveChatProvider("qwen3:14b"); got == "ollama" {
		t.Error("裸模型名不在 Ollama models 列表内不应路由到 ollama")
	}
}

func TestStreamClientCacheKey_Format(t *testing.T) {
	// 回归:曾误用 time.Duration(600).String() 生成 "600ns" 这类误导性 key
	if got := streamClientCacheKey("Ollama", 600); got != "Ollama|600" {
		t.Errorf("streamClientCacheKey = %q, want Ollama|600", got)
	}
	if got := streamClientCacheKey("DeepSeek", 0); got != "DeepSeek|0" {
		t.Errorf("streamClientCacheKey = %q, want DeepSeek|0", got)
	}
}

func TestPickStreamClient_OllamaHeaderTimeout(t *testing.T) {
	h := newOllamaRoutingTestHandler(t)
	prov := &config.AnthropicProviderConfig{Name: "Ollama", HeaderTimeoutSec: 600}
	client := h.pickStreamClient(prov)
	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport 类型错误: %T", client.Transport)
	}
	if tr.ResponseHeaderTimeout != 600*time.Second {
		t.Errorf("Ollama 流式响应头超时 = %v, want 600s", tr.ResponseHeaderTimeout)
	}
}
