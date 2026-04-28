package handlers

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestStripThinkingBlocks_DeepSeekStyle(t *testing.T) {
	// 模拟 DeepSeek 返回：content 中包含 type="thinking" 块
	input := `{
		"id": "msg_001",
		"model": "deepseek-reasoner",
		"content": [
			{"type": "thinking", "thinking": "让我思考一下这个问题..."},
			{"type": "text", "text": "答案是 42"}
		],
		"usage": {"input_tokens": 100, "output_tokens": 50}
	}`

	output := stripThinkingBlocks([]byte(input))

	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}

	content, ok := resp["content"].([]interface{})
	if !ok {
		t.Fatal("content is not an array")
	}

	if len(content) != 1 {
		t.Fatalf("expected 1 content block (thinking stripped), got %d: %s", len(content), output)
	}

	block, _ := content[0].(map[string]interface{})
	if block["type"] != "text" {
		t.Errorf("expected text block, got %s", block["type"])
	}
}

func TestStripThinkingBlocks_PreservesAnthropicNative(t *testing.T) {
	// 真 Anthropic extended thinking：有 signature 字段，但 stripThinkingBlocks
	// 只看 type="thinking"，所以同样会剥离。
	// 但实际调用时已限制为仅 DeepSeek provider 才调用，所以真 Anthropic 不会触发。
	// 此测试验证 stripThinkingBlocks 函数本身的确定性。
	input := `{
		"id": "msg_002",
		"model": "claude-opus-4-7",
		"content": [
			{"type": "thinking", "thinking": "Let me analyze this...", "signature": "abc123"},
			{"type": "text", "text": "Here is the result"}
		],
		"usage": {"input_tokens": 200, "output_tokens": 100}
	}`

	output := stripThinkingBlocks([]byte(input))

	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}

	content, _ := resp["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("expected 1 content block (thinking stripped by this function), got %d", len(content))
	}
}

func TestStripThinkingBlocks_NoThinkingBlocks(t *testing.T) {
	// 无 thinking 块的响应：原样返回
	input := `{
		"id": "msg_003",
		"model": "qwen3.5-plus",
		"content": [
			{"type": "text", "text": "你好，有什么可以帮你的？"}
		],
		"usage": {"input_tokens": 50, "output_tokens": 30}
	}`

	output := stripThinkingBlocks([]byte(input))

	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}

	content, _ := resp["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(content))
	}
}

func TestStripThinkingBlocks_OnlyThinkingBlocks(t *testing.T) {
	// 仅有 thinking 块：全部剥离后 content 为空数组
	input := `{
		"id": "msg_004",
		"model": "deepseek-reasoner",
		"content": [
			{"type": "thinking", "thinking": "第一步..."},
			{"type": "thinking", "thinking": "第二步..."}
		],
		"usage": {}
	}`

	output := stripThinkingBlocks([]byte(input))

	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}

	content, _ := resp["content"].([]interface{})
	if len(content) != 0 {
		t.Fatalf("expected 0 content blocks, got %d", len(content))
	}
}

func TestStripThinkingBlocks_InvalidJSON(t *testing.T) {
	input := []byte("not json at all")
	output := stripThinkingBlocks(input)
	if string(output) != "not json at all" {
		t.Error("invalid JSON should be returned as-is")
	}
}

// ========== SSE 流式 thinking 过滤测试 ==========

func makeSSEEvent(event, data string) string {
	var parts []string
	if event != "" {
		parts = append(parts, "event: "+event)
	}
	if data != "" {
		parts = append(parts, "data: "+data)
	}
	return strings.Join(parts, "\n") + "\n\n"
}

func readAllSSE(r io.Reader) string {
	var buf strings.Builder
	io.Copy(&buf, r)
	return buf.String()
}

func TestSSEThinkingFilter_StripsThinkingBlocks(t *testing.T) {
	input := makeSSEEvent("message_start", `{"type":"message_start","message":{"id":"msg_1","model":"deepseek-reasoner"}}`) +
		makeSSEEvent("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"让我思考..."}}`) +
		makeSSEEvent("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"继续..."}}`) +
		makeSSEEvent("content_block_stop", `{"type":"content_block_stop","index":0}`) +
		makeSSEEvent("content_block_start", `{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`) +
		makeSSEEvent("content_block_delta", `{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"答案是42"}}`) +
		makeSSEEvent("content_block_stop", `{"type":"content_block_stop","index":1}`) +
		makeSSEEvent("message_stop", `{"type":"message_stop"}`)

	filter := newSSEThinkingFilter(strings.NewReader(input))
	output := readAllSSE(filter)

	// thinking 事件应被过滤
	if strings.Contains(output, "thinking") {
		t.Errorf("thinking event should be filtered, got: %s", output)
	}
	// text 事件应保留
	if !strings.Contains(output, "答案是42") {
		t.Errorf("text event should be preserved, got: %s", output)
	}
	// message_start 和 message_stop 应保留
	if !strings.Contains(output, "message_start") || !strings.Contains(output, "message_stop") {
		t.Errorf("message_start/stop should be preserved, got: %s", output)
	}
}

func TestSSEThinkingFilter_NoThinkingBlocks(t *testing.T) {
	input := makeSSEEvent("message_start", `{"type":"message_start","message":{"id":"msg_2"}}`) +
		makeSSEEvent("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`) +
		makeSSEEvent("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`) +
		makeSSEEvent("content_block_stop", `{"type":"content_block_stop","index":0}`) +
		makeSSEEvent("message_stop", `{"type":"message_stop"}`)

	filter := newSSEThinkingFilter(strings.NewReader(input))
	output := readAllSSE(filter)

	if !strings.Contains(output, "text_delta") || !strings.Contains(output, "hello") {
		t.Errorf("all non-thinking events should be preserved, output: %s", output)
	}
	if !strings.Contains(output, "message_stop") {
		t.Errorf("message_stop should be preserved")
	}
}

func TestSSEThinkingFilter_MultipleThinkingBlocks(t *testing.T) {
	// 两个 thinking 块，中间夹一个 text 块
	input := makeSSEEvent("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"思路1"}}`) +
		makeSSEEvent("content_block_stop", `{"type":"content_block_stop","index":0}`) +
		makeSSEEvent("content_block_start", `{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`) +
		makeSSEEvent("content_block_delta", `{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"结果1"}}`) +
		makeSSEEvent("content_block_stop", `{"type":"content_block_stop","index":1}`) +
		makeSSEEvent("content_block_start", `{"type":"content_block_start","index":2,"content_block":{"type":"thinking","thinking":"思路2"}}`) +
		makeSSEEvent("content_block_stop", `{"type":"content_block_stop","index":2}`) +
		makeSSEEvent("content_block_start", `{"type":"content_block_start","index":3,"content_block":{"type":"text","text":""}}`) +
		makeSSEEvent("content_block_delta", `{"type":"content_block_delta","index":3,"delta":{"type":"text_delta","text":"结果2"}}`) +
		makeSSEEvent("content_block_stop", `{"type":"content_block_stop","index":3}`)

	filter := newSSEThinkingFilter(strings.NewReader(input))
	output := readAllSSE(filter)

	if strings.Contains(output, "thinking") {
		t.Errorf("all thinking events should be filtered")
	}
	if !strings.Contains(output, "结果1") || !strings.Contains(output, "结果2") {
		t.Errorf("all text events should be preserved")
	}
}
