package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// JSONParser JSON 解析器
type JSONParser struct{}

// NewJSONParser 创建 JSON 解析器
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// JSONParseResult JSON 解析结果
type JSONParseResult struct {
	Original     string      `json:"original"`      // 原始内容
	Formatted    string      `json:"formatted"`    // 格式化后的内容
	Minified     string      `json:"minified"`     // 压缩后的内容
	IsValid      bool        `json:"is_valid"`     // 是否为有效 JSON
	ParseError   string      `json:"parse_error"`  // 解析错误信息
	Type         string      `json:"type"`         // JSON 类型 (object, array, string, number, boolean, null)
	Keys         []string    `json:"keys"`         // 顶层键名列表
	ArrayLength  int         `json:"array_length"` // 数组长度（如果是数组）
	Depth        int         `json:"depth"`        // 嵌套深度
	ByteSize     int         `json:"byte_size"`    // 字节大小
	LineCount    int         `json:"line_count"`   // 行数
}

// Parse 解析 JSON 内容
func (p *JSONParser) Parse(content string) *JSONParseResult {
	result := &JSONParseResult{
		Original:  content,
		IsValid:   false,
		LineCount: strings.Count(content, "\n") + 1,
		ByteSize:  len(content),
	}

	if content == "" {
		result.ParseError = "内容为空"
		return result
	}

	// 尝试解析
	var data interface{}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		result.ParseError = err.Error()
		return result
	}

	result.IsValid = true

	// 分析 JSON 结构
	result.analyzeStructure(data)

	// 格式化
	result.Formatted = p.Format(content)
	result.Minified = p.Minify(content)

	return result
}

// analyzeStructure 分析 JSON 结构
func (r *JSONParseResult) analyzeStructure(data interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		r.Type = "object"
		for k := range v {
			r.Keys = append(r.Keys, k)
		}
		r.Depth = r.calculateDepth(v, 1)
	case []interface{}:
		r.Type = "array"
		r.ArrayLength = len(v)
		r.Depth = r.calculateDepth(v, 1)
	case string:
		r.Type = "string"
	case json.Number:
		r.Type = "number"
	case bool:
		r.Type = "boolean"
	case nil:
		r.Type = "null"
	}
}

// calculateDepth 计算嵌套深度
func (r *JSONParseResult) calculateDepth(data interface{}, depth int) int {
	maxDepth := depth

	switch v := data.(type) {
	case map[string]interface{}:
		for _, val := range v {
			d := r.calculateDepth(val, depth+1)
			if d > maxDepth {
				maxDepth = d
			}
		}
	case []interface{}:
		for _, item := range v {
			d := r.calculateDepth(item, depth+1)
			if d > maxDepth {
				maxDepth = d
			}
		}
	}

	return maxDepth
}

// Format 格式化 JSON
func (p *JSONParser) Format(content string) string {
	var data interface{}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		return content
	}

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return content
	}

	return string(output)
}

// Minify 压缩 JSON
func (p *JSONParser) Minify(content string) string {
	var data interface{}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		return content
	}

	output, err := json.Marshal(data)
	if err != nil {
		return content
	}

	return string(output)
}

// Validate 验证 JSON 是否有效
func (p *JSONParser) Validate(content string) (bool, string) {
	if content == "" {
		return false, "内容为空"
	}

	var data interface{}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		return false, err.Error()
	}

	return true, ""
}

// GetValue 获取 JSON 中的指定路径的值
func (p *JSONParser) GetValue(content string, path string) (string, error) {
	var data interface{}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		return "", err
	}

	keys := strings.Split(path, ".")
	current := data

	for _, key := range keys {
		switch c := current.(type) {
		case map[string]interface{}:
			val, ok := c[key]
			if !ok {
				return "", fmt.Errorf("路径不存在: %s", path)
			}
			current = val
		case []interface{}:
			idx := 0
			fmt.Sscanf(key, "%d", &idx)
			if idx < 0 || idx >= len(c) {
				return "", fmt.Errorf("数组索引越界: %d", idx)
			}
			current = c[idx]
		default:
			return "", fmt.Errorf("无效的路径: %s", path)
		}
	}

	result, err := json.Marshal(current)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// Diff 比较两个 JSON 对象的差异
func (p *JSONParser) Diff(json1, json2 string) (string, error) {
	var data1, data2 interface{}
	decoder := json.NewDecoder(strings.NewReader(json1))
	decoder.UseNumber()
	if err := decoder.Decode(&data1); err != nil {
		return "", fmt.Errorf("JSON1 解析失败: %v", err)
	}

	decoder = json.NewDecoder(strings.NewReader(json2))
	decoder.UseNumber()
	if err := decoder.Decode(&data2); err != nil {
		return "", fmt.Errorf("JSON2 解析失败: %v", err)
	}

	diff := p.findDiff("", data1, data2)

	if len(diff) == 0 {
		return "两个 JSON 对象相同", nil
	}

	var buf bytes.Buffer
	buf.WriteString("差异:\n")
	for _, d := range diff {
		buf.WriteString(d + "\n")
	}

	return buf.String(), nil
}

// findDiff 递归查找差异
func (p *JSONParser) findDiff(path string, v1, v2 interface{}) []string {
	var diffs []string

	switch val1 := v1.(type) {
	case map[string]interface{}:
		val2, ok := v2.(map[string]interface{})
		if !ok {
			diffs = append(diffs, fmt.Sprintf("%s: 类型不匹配", path))
			return diffs
		}

		// 检查键的差异
		for k, v := range val1 {
			currentPath := k
			if path != "" {
				currentPath = path + "." + k
			}

			if v2, exists := val2[k]; !exists {
				diffs = append(diffs, fmt.Sprintf("%s: 在第一个对象中存在，第二个对象中不存在", currentPath))
			} else {
				diffs = append(diffs, p.findDiff(currentPath, v, v2)...)
			}
		}

		for k := range val2 {
			if _, exists := val1[k]; !exists {
				currentPath := k
				if path != "" {
					currentPath = path + "." + k
				}
				diffs = append(diffs, fmt.Sprintf("%s: 在第二个对象中存在，第一个对象中不存在", currentPath))
			}
		}

	case []interface{}:
		val2, ok := v2.([]interface{})
		if !ok {
			diffs = append(diffs, fmt.Sprintf("%s: 类型不匹配", path))
			return diffs
		}

		if len(val1) != len(val2) {
			diffs = append(diffs, fmt.Sprintf("%s: 数组长度不同 (%d vs %d)", path, len(val1), len(val2)))
		}

		maxLen := len(val1)
		if len(val2) > maxLen {
			maxLen = len(val2)
		}

		for i := 0; i < maxLen; i++ {
			currentPath := fmt.Sprintf("%s[%d]", path, i)
			if i >= len(val1) {
				diffs = append(diffs, fmt.Sprintf("%s: 在第二个对象中存在，第一个对象中不存在", currentPath))
			} else if i >= len(val2) {
				diffs = append(diffs, fmt.Sprintf("%s: 在第一个对象中存在，第二个对象中不存在", currentPath))
			} else {
				diffs = append(diffs, p.findDiff(currentPath, val1[i], val2[i])...)
			}
		}

	default:
		if v1 != v2 {
			diffs = append(diffs, fmt.Sprintf("%s: %v vs %v", path, v1, v2))
		}
	}

	return diffs
}

// Query JSONPath 查询
func (p *JSONParser) Query(content string, query string) (string, error) {
	// 简化的 JSONPath 查询实现
	// 支持 $ 根对象，. 键访问，[] 数组访问

	var data interface{}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		return "", err
	}

	result := p.queryPath(data, query)
	if result == nil {
		return "", fmt.Errorf("查询结果为空")
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// queryPath 查询路径
func (p *JSONParser) queryPath(data interface{}, query string) interface{} {
	// 移除开头的 $
	query = regexp.MustCompile(`^\$\.?`).ReplaceAllString(query, "")

	if query == "" {
		return data
	}

	parts := strings.Split(query, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch c := current.(type) {
		case map[string]interface{}:
			current = c[part]
		case []interface{}:
			// 处理数组索引
			if idx, err := strconv.Atoi(part); err == nil && idx >= 0 && idx < len(c) {
				current = c[idx]
			} else {
				return nil
			}
		default:
			return nil
		}

		if current == nil {
			return nil
		}
	}

	return current
}

// ConvertToJSONLines 将 JSON 数组转换为 JSONL 格式
func (p *JSONParser) ConvertToJSONLines(content string) (string, error) {
	var data interface{}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		return "", err
	}

	arr, ok := data.([]interface{})
	if !ok {
		return "", fmt.Errorf("内容不是 JSON 数组")
	}

	var buf bytes.Buffer
	for _, item := range arr {
		line, err := json.Marshal(item)
		if err != nil {
			continue
		}
		buf.Write(line)
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

// ConvertToJSON 将 JSONL 转换为 JSON 数组
func (p *JSONParser) ConvertToJSON(content string) (string, error) {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	var result []interface{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var item interface{}
		decoder := json.NewDecoder(strings.NewReader(line))
		decoder.UseNumber()

		if err := decoder.Decode(&item); err != nil {
			return "", fmt.Errorf("解析行失败: %v", err)
		}

		result = append(result, item)
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(output), nil
}
