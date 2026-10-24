package handlers

import (
	"context"
	"fmt"
)

// Complete 内部免 API Key 文本补全,供其它 handler(如新闻生成)复用统一的模型路由。
//
// model 为空时回退到配置默认模型(MiniMax.Model),再回退到 MiniMax-M2.5。
// 返回模型输出的纯文本;若未配置任何可用 provider / key 则返回 error,调用方应做好降级。
func (h *AIGatewayHandler) Complete(ctx context.Context, model, system, user string) (string, error) {
	if model == "" {
		model = h.cfg.MiniMax.Model
	}
	if model == "" {
		model = "MiniMax-M2.5"
	}

	messages := make([]map[string]interface{}, 0, 2)
	if system != "" {
		messages = append(messages, map[string]interface{}{"role": "system", "content": system})
	}
	messages = append(messages, map[string]interface{}{"role": "user", "content": user})

	req := ChatCompletionRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}

	result, _, err := h.executeChatRequest(req)
	if err != nil {
		return "", err
	}

	// executeChatRequest 已按 provider 归一化出 content:
	// MiniMax 走 Anthropic 格式(经 extractMiniMaxText 提取),其余 provider 走 OpenAI choices 提取。
	content, _ := result["content"].(string)
	if content == "" {
		// 兜底:极少数 provider 可能只在 choices 中携带文本
		if choices, ok := result["choices"].([]interface{}); ok {
			content = extractMessageContentFromChoices(choices)
		}
	}
	if content == "" {
		return "", fmt.Errorf("AI 返回内容为空")
	}
	return content, nil
}
