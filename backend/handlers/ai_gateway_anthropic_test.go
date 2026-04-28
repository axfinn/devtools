package handlers

import (
	"encoding/json"
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
