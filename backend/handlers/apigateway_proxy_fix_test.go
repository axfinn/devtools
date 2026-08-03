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

	"github.com/gin-gonic/gin"
)

// 修复：callProxyChat 之前漏 stop / response_format / tools / tool_choice /
// reasoning_effort / extra_body，对 DeepSeek-reasoner 等关键参数（reasoning_effort）
// 会导致模型行为完全不同。本测试确认所有字段都被透传。

func TestCallProxyChat_ForwardsAllReasoningAndToolFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var captured map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Errorf("upstream received non-JSON body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"x","choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{}}`))
	}))
	defer upstream.Close()

	cfg := config.DefaultConfig()
	cfg.AIGateway.Proxy.APIURL = upstream.URL
	cfg.AIGateway.Proxy.APIKey = "test-key"
	cfg.AIGateway.Proxy.UpstreamModel = ""

	h := &AIGatewayHandler{cfg: cfg, noProxyClient: &http.Client{Timeout: 5 * time.Second}}

	temp := 0.3
	maxTok := 256
	topP := 0.9
	stop := []string{"\n\nUser:"}
	reasoning := "high"
	req := ChatCompletionRequest{
		Model: "proxy-chat",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "hi"},
		},
		Temperature:     &temp,
		MaxTokens:       &maxTok,
		TopP:            &topP,
		Stop:            stop,
		ReasoningEffort: reasoning,
		ResponseFormat:  map[string]interface{}{"type": "json_object"},
		Tools: []map[string]interface{}{
			{"type": "function", "function": map[string]interface{}{"name": "get_time"}},
		},
		ToolChoice: "auto",
		ExtraBody:  map[string]interface{}{"custom_flag": true},
	}

	if _, _, err := h.callProxyChat(req); err != nil {
		t.Fatalf("callProxyChat failed: %v", err)
	}

	mustField := func(key string) {
		if _, ok := captured[key]; !ok {
			t.Errorf("expected field %q in upstream body, got %v", key, captured)
		}
	}
	mustField("stop")
	mustField("response_format")
	mustField("tools")
	mustField("tool_choice")
	mustField("reasoning_effort")
	mustField("custom_flag")

	if captured["reasoning_effort"] != "high" {
		t.Errorf("reasoning_effort = %v, want high", captured["reasoning_effort"])
	}
	if captured["stop"] == nil {
		t.Error("stop field missing or nil")
	}
}

// 修复：doRequestWithClient 在 anthropicMode=true 时补齐 anthropic-version 与 x-api-key。
// 即便客户端没传，上游（MiniMax/DeepSeek Anthropic 端点）也应能识别为 Anthropic 协议。

func TestDoRequestWithClient_AnthropicModeSetsDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var seenAuth, seenXAPIKey, seenVersion string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		seenXAPIKey = r.Header.Get("x-api-key")
		seenVersion = r.Header.Get("anthropic-version")
		w.WriteHeader(200)
		w.Write([]byte(`ok`))
	}))
	defer upstream.Close()

	h := &AIGatewayHandler{
		noProxyClient: &http.Client{Timeout: 5 * time.Second},
	}

	_, _, err := h.doRequestWithClient(h.noProxyClient, upstream.URL+"/v1/messages", "sk-test", "POST", []byte(`{}`), nil, true)
	if err != nil {
		t.Fatalf("doRequestWithClient: %v", err)
	}

	if !strings.HasPrefix(seenAuth, "Bearer ") {
		t.Errorf("Authorization = %q, want Bearer prefix", seenAuth)
	}
	if seenXAPIKey != "sk-test" {
		t.Errorf("x-api-key = %q, want sk-test", seenXAPIKey)
	}
	if seenVersion != "2023-06-01" {
		t.Errorf("anthropic-version = %q, want 2023-06-01", seenVersion)
	}
}

// 修复：客户端已传 anthropic-version/x-api-key 时，代理不能覆盖（透传优先）。

func TestDoRequestWithClient_AnthropicModeRespectsClientHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var seenXAPIKey, seenVersion string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenXAPIKey = r.Header.Get("x-api-key")
		seenVersion = r.Header.Get("anthropic-version")
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	h := &AIGatewayHandler{noProxyClient: &http.Client{Timeout: 5 * time.Second}}

	clientHeaders := http.Header{}
	clientHeaders.Set("x-api-key", "client-supplied-key")
	clientHeaders.Set("anthropic-version", "2024-01-01")

	_, _, err := h.doRequestWithClient(h.noProxyClient, upstream.URL+"/v1/messages", "sk-server", "POST", []byte(`{}`), clientHeaders, true)
	if err != nil {
		t.Fatalf("doRequestWithClient: %v", err)
	}

	if seenXAPIKey != "client-supplied-key" {
		t.Errorf("x-api-key = %q, want client-supplied-key (must not be overwritten)", seenXAPIKey)
	}
	if seenVersion != "2024-01-01" {
		t.Errorf("anthropic-version = %q, want 2024-01-01 (must not be overwritten)", seenVersion)
	}
}

// 非 Anthropic 路径（MiniMax 媒体 API）不应被强行加 anthropic 头。

func TestDoRequestWithClient_NonAnthropicModeOmitsHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var seenXAPIKey, seenVersion string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenXAPIKey = r.Header.Get("x-api-key")
		seenVersion = r.Header.Get("anthropic-version")
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	h := &AIGatewayHandler{noProxyClient: &http.Client{Timeout: 5 * time.Second}}

	_, _, err := h.doRequestWithClient(h.noProxyClient, upstream.URL+"/v1/t2a_v2", "sk-test", "POST", []byte(`{}`), nil, false)
	if err != nil {
		t.Fatalf("doRequestWithClient: %v", err)
	}

	if seenXAPIKey != "" {
		t.Errorf("x-api-key should not be set for non-anthropic endpoints, got %q", seenXAPIKey)
	}
	if seenVersion != "" {
		t.Errorf("anthropic-version should not be set for non-anthropic endpoints, got %q", seenVersion)
	}
}

// 修复：isDeepSeekProvider 应该按 Name 和 URL 两种方式都识别。

func TestIsDeepSeekProvider(t *testing.T) {
	cases := []struct {
		name string
		p    *config.AnthropicProviderConfig
		want bool
	}{
		{"nil provider", nil, false},
		{"name DeepSeek", &config.AnthropicProviderConfig{Name: "DeepSeek", APIURL: "https://other.example.com/v1"}, true},
		{"name lowercase", &config.AnthropicProviderConfig{Name: "deepseek", APIURL: "https://other.example.com/v1"}, true},
		{"url deepseek.com", &config.AnthropicProviderConfig{Name: "MiniMax", APIURL: "https://api.deepseek.com/anthropic"}, true},
		{"unrelated", &config.AnthropicProviderConfig{Name: "MiniMax", APIURL: "https://api.minimaxi.com/anthropic"}, false},
	}
	for _, tc := range cases {
		if got := isDeepSeekProvider(tc.p); got != tc.want {
			t.Errorf("%s: isDeepSeekProvider = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// 修复：proxyAnthropicStream 对 DeepSeek 上游应自动套上 sseThinkingFilter。
// 这里用 httptest 模拟上游发送 thinking+text 事件，验证客户端只收到 text。

func TestProxyAnthropicStream_FiltersThinkingForDeepSeek(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("anthropic-version") == "" {
			t.Error("upstream did not receive anthropic-version header")
		}
		if r.Header.Get("x-api-key") == "" {
			t.Error("upstream did not receive x-api-key header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)

		events := []string{
			`event: message_start` + "\n" + `data: {"type":"message_start","message":{"id":"m1"}}` + "\n\n",
			`event: content_block_start` + "\n" + `data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"..."}}` + "\n\n",
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"step"}}` + "\n\n",
			`event: content_block_stop` + "\n" + `data: {"type":"content_block_stop","index":0}` + "\n\n",
			`event: content_block_start` + "\n" + `data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}` + "\n\n",
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"final-answer"}}` + "\n\n",
			`event: content_block_stop` + "\n" + `data: {"type":"content_block_stop","index":1}` + "\n\n",
			`event: message_stop` + "\n" + `data: {"type":"message_stop"}` + "\n\n",
		}
		for _, e := range events {
			io.WriteString(w, e)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer upstream.Close()

	cfg := config.DefaultConfig()
	cfg.AIGateway.Proxy.APIURL = ""
	cfg.MiniMax.APIKey = ""
	h := &AIGatewayHandler{
		cfg:          cfg,
		streamClient: &http.Client{Timeout: 5 * time.Second},
	}
	provider := &config.AnthropicProviderConfig{
		Name:   "DeepSeek",
		APIURL: upstream.URL,
		APIKey: "sk-ds",
	}

	body := []byte(`{"model":"deepseek-reasoner","max_tokens":16,"stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/deepseek/anthropic/v1/messages", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	_, err := h.proxyAnthropicStream(c, provider, upstream.URL+"/v1/messages", body)
	if err != nil {
		t.Fatalf("proxyAnthropicStream: %v", err)
	}

	out := rec.Body.String()
	if strings.Contains(out, "thinking_delta") || strings.Contains(out, `"thinking":`) {
		t.Errorf("thinking blocks should be filtered, got: %s", out)
	}
	if !strings.Contains(out, "final-answer") {
		t.Errorf("text content should be preserved, got: %s", out)
	}
}

// proxyAnthropicStream 对非 DeepSeek 上游不应过滤（透传原始事件）。

func TestProxyAnthropicStream_DoesNotFilterNonDeepSeek(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		io.WriteString(w, `event: message_start`+"\n"+`data: {"type":"message_start"}`+"\n\n")
		io.WriteString(w, `event: content_block_start`+"\n"+`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"hi"}}`+"\n\n")
		io.WriteString(w, `event: content_block_stop`+"\n"+`data: {"type":"content_block_stop","index":0}`+"\n\n")
		io.WriteString(w, `event: message_stop`+"\n"+`data: {"type":"message_stop"}`+"\n\n")
	}))
	defer upstream.Close()

	cfg := config.DefaultConfig()
	h := &AIGatewayHandler{
		cfg:          cfg,
		streamClient: &http.Client{Timeout: 5 * time.Second},
	}
	provider := &config.AnthropicProviderConfig{
		Name:   "MiniMax",
		APIURL: upstream.URL,
		APIKey: "sk-mn",
	}

	body := []byte(`{"model":"MiniMax-M3","stream":true,"messages":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/minimax/anthropic/v1/messages", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	_, err := h.proxyAnthropicStream(c, provider, upstream.URL+"/v1/messages", body)
	if err != nil {
		t.Fatalf("proxyAnthropicStream: %v", err)
	}

	out := rec.Body.String()
	if !strings.Contains(out, "thinking") {
		t.Errorf("non-DeepSeek should pass through thinking blocks unchanged, got: %s", out)
	}
}

// proxyAnthropicWithBody 对 DeepSeek 非流式响应也应剥掉 thinking 块。

func TestProxyAnthropicWithBody_StripsThinkingForDeepSeek(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("anthropic-version") == "" {
			t.Error("upstream did not receive anthropic-version header")
		}
		if r.Header.Get("x-api-key") == "" {
			t.Error("upstream did not receive x-api-key header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		io.WriteString(w, `{"id":"x","model":"deepseek-reasoner","content":[{"type":"thinking","thinking":"think"},{"type":"text","text":"42"}],"usage":{}}`)
	}))
	defer upstream.Close()

	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "admin-pw"
	h := &AIGatewayHandler{
		cfg:          cfg,
		noProxyClient: &http.Client{Timeout: 5 * time.Second},
	}
	provider := &config.AnthropicProviderConfig{
		Name:   "DeepSeek",
		APIURL: upstream.URL,
		APIKey: "sk-ds",
	}

	body := []byte(`{"model":"deepseek-reasoner","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/deepseek/anthropic/v1/messages", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Super-Admin-Password", "admin-pw")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	h.proxyAnthropic(c, provider, "/api/deepseek/anthropic/v1/messages", []string{"deepseek-reasoner"})

	out := rec.Body.String()
	if strings.Contains(out, `"thinking":`) {
		t.Errorf("thinking block should be stripped, got: %s", out)
	}
	if !strings.Contains(out, `"text":"42"`) {
		t.Errorf("text block should be preserved, got: %s", out)
	}
}