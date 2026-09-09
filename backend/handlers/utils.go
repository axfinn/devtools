package handlers

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// truncateString 安全地按 rune 数截断字符串，避免把多字节 UTF-8 字符截到中间。
// 截断后会追加 "...(truncated)" 标记。
func truncateString(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "...(truncated)"
}

// sanitizeJSON 把任意值序列化成 JSON 字符串，所有 string 节点都会被截断以防写入超长日志。
func sanitizeJSON(payload interface{}) string {
	data, err := json.Marshal(sanitizeValue(payload))
	if err != nil {
		return "{}"
	}
	return string(data)
}

// sanitizeValue 递归处理 gin.H / map / slice / string，做长度截断与 nil 安全。
func sanitizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return truncateString(v, 2000)
	case []string:
		items := make([]interface{}, 0, len(v))
		for _, item := range v {
			items = append(items, truncateString(item, 500))
		}
		return items
	case []interface{}:
		items := make([]interface{}, 0, len(v))
		for _, item := range v {
			items = append(items, sanitizeValue(item))
		}
		return items
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, item := range v {
			result[key] = sanitizeValue(item)
		}
		return result
	case gin.H:
		result := make(map[string]interface{}, len(v))
		for key, item := range v {
			result[key] = sanitizeValue(item)
		}
		return result
	default:
		return value
	}
}

// extractString 按路径从嵌套 map 中提取字符串值，路径不存在时返回空串。
func extractString(payload map[string]interface{}, path ...string) string {
	var current interface{} = payload
	for _, key := range path {
		node, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = node[key]
		if !ok {
			return ""
		}
	}
	switch value := current.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	default:
		return ""
	}
}

// parseInt 解析整数字符串，解析失败或为空时返回 fallback。
func parseInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}
