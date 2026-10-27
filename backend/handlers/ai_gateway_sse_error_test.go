package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归：proxyAnthropicStream 在 WriteHeader(200) 之后，上游 4xx/5xx / 拨号失败
// 曾用 c.JSON/c.Data 尝试改状态码（superfluous WriteHeader 被 Go 静默忽略），
// 客户端实际收到 200 + 与 ping 帧混杂的裸 JSON，标准 Anthropic SDK 解析失败。
// 修复后错误必须以 Anthropic 规范的 `event: error` 帧写给客户端。

func newSSETestWriter() (*sseWriter, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	return newSSEWriter(rec, rec), rec
}

func TestWriteSSEErrorFrame_Format(t *testing.T) {
	w, rec := newSSETestWriter()
	writeSSEErrorFrame(w, 502, "dial tcp: connection refused")

	body := rec.Body.String()
	if !strings.Contains(body, "event: error\n") {
		t.Errorf("缺少 event: error 帧: %q", body)
	}
	if !strings.Contains(body, `"type":"error"`) || !strings.Contains(body, `"type":"api_error"`) {
		t.Errorf("error 帧结构不符合 Anthropic 规范: %q", body)
	}
	if !strings.Contains(body, "dial tcp: connection refused") {
		t.Errorf("错误信息丢失: %q", body)
	}
}

func TestWriteSSEErrorFrame_ErrorTypeMapping(t *testing.T) {
	cases := []struct {
		status  int
		errType string
	}{
		{400, "invalid_request_error"},
		{401, "authentication_error"},
		{403, "authentication_error"},
		{404, "not_found_error"},
		{429, "rate_limit_error"},
		{529, "overloaded_error"},
		{500, "api_error"},
		{502, "api_error"},
	}
	for _, tc := range cases {
		w, rec := newSSETestWriter()
		writeSSEErrorFrame(w, tc.status, "x")
		if !strings.Contains(rec.Body.String(), `"`+tc.errType+`"`) {
			t.Errorf("status %d 应映射为 %s, got %q", tc.status, tc.errType, rec.Body.String())
		}
	}
}

func TestForwardUpstreamSSEError_AnthropicStylePassthrough(t *testing.T) {
	w, rec := newSSETestWriter()
	upstream := []byte(`{"type":"error","error":{"type":"not_found_error","message":"model not found"}}`)
	forwardUpstreamSSEError(w, 404, upstream)

	body := rec.Body.String()
	if !strings.Contains(body, "event: error\n") || !strings.Contains(body, "model not found") {
		t.Errorf("Anthropic 风格错误体应原样转发: %q", body)
	}
}

func TestForwardUpstreamSSEError_NonAnthropicWrapped(t *testing.T) {
	w, rec := newSSETestWriter()
	forwardUpstreamSSEError(w, 500, []byte("  plain upstream crash  "))

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"api_error"`) || !strings.Contains(body, "plain upstream crash") {
		t.Errorf("非 Anthropic 错误体应被包装成 api_error: %q", body)
	}
	// 空错误体
	w2, rec2 := newSSETestWriter()
	forwardUpstreamSSEError(w2, 503, nil)
	if !strings.Contains(rec2.Body.String(), "upstream error 503") {
		t.Errorf("空错误体应生成默认消息: %q", rec2.Body.String())
	}
}
