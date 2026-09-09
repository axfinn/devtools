package models

import "encoding/json"

// MustJSONString 把任意值序列化成 JSON 字符串，失败时回退到 "{}"。
// 用于把切片 / map 存到 SQLite 文本列（AIAPIKey.AllowedModels / AllowedScopes 等）。
func MustJSONString(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
