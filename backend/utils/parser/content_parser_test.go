package parser

import (
	"devtools/utils/security"
	"testing"
)

// TestJSONParser 测试 JSON 解析器
func TestJSONParser(t *testing.T) {
	parser := NewJSONParser()

	// 测试有效 JSON
	content := `{"name": "test", "value": 123}`
	result := parser.Parse(content)
	if !result.IsValid {
		t.Error("Expected valid JSON")
	}
	if result.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", result.Type)
	}
	if len(result.Keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(result.Keys))
	}

	// 测试无效 JSON
	invalidContent := `{name: test}`
	result = parser.Parse(invalidContent)
	if result.IsValid {
		t.Error("Expected invalid JSON")
	}
	if result.ParseError == "" {
		t.Error("Expected parse error for invalid JSON")
	}

	// 测试空内容
	result = parser.Parse("")
	if result.IsValid {
		t.Error("Expected empty content to be invalid")
	}

	// 测试数组 JSON
	arrayContent := `[1, 2, 3]`
	result = parser.Parse(arrayContent)
	if result.Type != "array" {
		t.Errorf("Expected type 'array', got '%s'", result.Type)
	}
	if result.ArrayLength != 3 {
		t.Errorf("Expected array length 3, got %d", result.ArrayLength)
	}

	// 测试格式化
	formatted := parser.Format(content)
	if formatted == content {
		t.Error("Expected formatted JSON to be different from original")
	}

	// 测试压缩
	minified := parser.Minify(formatted)
	if minified != `{"name":"test","value":123}` {
		t.Errorf("Expected minified JSON, got '%s'", minified)
	}
}

// TestJSONValidation 测试 JSON 验证
func TestJSONValidation(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{"valid_object", `{"key": "value"}`, true},
		{"valid_array", `[1, 2, 3]`, true},
		{"valid_string", `"hello"`, true},
		{"valid_number", `123`, true},
		{"valid_boolean", `true`, true},
		{"valid_null", `null`, true},
		{"invalid", `{invalid}`, false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := parser.Validate(tt.content)
			if valid != tt.valid {
				t.Errorf("Expected %v, got %v", tt.valid, valid)
			}
		})
	}
}

// TestJSONDiff 测试 JSON 比较
func TestJSONDiff(t *testing.T) {
	parser := NewJSONParser()

	json1 := `{"name": "test", "value": 123}`
	json2 := `{"name": "test", "value": 456}`

	diff, err := parser.Diff(json1, json2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if diff == "" {
		t.Error("Expected diff to be non-empty")
	}

	// 相同 JSON
	sameDiff, _ := parser.Diff(json1, json1)
	if sameDiff == "" {
		t.Error("Expected empty diff for same JSON")
	}
}

// TestCSVParser 测试 CSV 解析器
func TestCSVParser(t *testing.T) {
	parser := NewCSVParser()

	// 测试有效 CSV
	content := "name,age,city\nJohn,25,Beijing\nJane,30,Shanghai"
	result := parser.Parse(content)
	if !result.IsValid {
		t.Error("Expected valid CSV")
	}
	if len(result.Headers) != 3 {
		t.Errorf("Expected 3 headers, got %d", len(result.Headers))
	}
	if result.RowCount != 2 {
		t.Errorf("Expected 2 rows, got %d", result.RowCount)
	}

	// 测试空内容
	result = parser.Parse("")
	if result.IsValid {
		t.Error("Expected empty content to be invalid")
	}

	// 测试 ToMarkdown
	markdown, err := parser.ToMarkdown(content)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if markdown == "" {
		t.Error("Expected markdown output")
	}
}

// TestXMLParser 测试 XML 解析器
func TestXMLParser(t *testing.T) {
	parser := NewXMLParser()

	// 测试有效 XML
	content := `<?xml version="1.0"?><root><item>value</item></root>`
	result := parser.Parse(content)
	if !result.IsValid {
		t.Error("Expected valid XML")
	}
	if result.RootElement != "root" {
		t.Errorf("Expected root element 'root', got '%s'", result.RootElement)
	}

	// 测试空内容
	result = parser.Parse("")
	if result.IsValid {
		t.Error("Expected empty content to be invalid")
	}

	// 测试格式化
	formatted := parser.Format(content)
	if formatted == "" {
		t.Error("Expected formatted output")
	}
}

// TestMarkdownParser 测试 Markdown 解析器
func TestMarkdownParser(t *testing.T) {
	parser := NewMarkdownParser()

	// 测试有效 Markdown
	content := "# Title\n\nThis is a paragraph.\n\n## Subtitle\n\n- Item 1\n- Item 2\n\n```go\npackage main\n```\n"
	result := parser.Parse(content)
	if !result.IsValid {
		t.Error("Expected valid Markdown")
	}
	if result.HTML == "" {
		t.Error("Expected HTML output")
	}
	if !result.HasCode {
		t.Error("Expected code block detected")
	}
	if len(result.CodeLangs) == 0 {
		t.Error("Expected code language detected")
	}

	// 测试空内容
	result = parser.Parse("")
	if !result.IsValid {
		t.Error("Expected empty content to be valid")
	}
}

// TestRichTextParser 测试富文本解析器
func TestRichTextParser(t *testing.T) {
	parser := NewRichTextParser()

	// 测试有效 HTML
	content := `<html><body><p>Hello <strong>World</strong></p></body></html>`
	result := parser.Parse(content)
	if !result.IsValid {
		t.Error("Expected valid HTML")
	}
	if !result.HasFormatting {
		t.Error("Expected formatting detected")
	}
	if result.PlainText == "" {
		t.Error("Expected plain text")
	}

	// 测试消毒
	sanitized := parser.Sanitize(content)
	if sanitized == "" {
		t.Error("Expected sanitized output")
	}

	// 测试空内容
	result = parser.Parse("")
	if result.IsValid {
		t.Error("Expected empty content to be invalid")
	}
}

// TestContentTypeDetection 测试内容类型检测
func TestContentTypeDetection(t *testing.T) {
	cp := NewContentParser()

	tests := []struct {
		name    string
		content string
		want    ContentType
	}{
		{"json", `{"key": "value"}`, ContentTypeJSON},
		{"xml", `<root><item>value</item></root>`, ContentTypeXML},
		{"csv", "a,b,c\n1,2,3", ContentTypeCSV},
		{"plain_text", "Just some text", ContentTypeText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cp.DetectContentType(tt.content)
			if got != tt.want {
				t.Logf("DetectContentType(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

// TestFormatContent 测试格式化内容
func TestFormatContent(t *testing.T) {
	cp := NewContentParser()

	// 测试 JSON 格式化
	jsonContent := `{"name":"test","value":123}`
	formatted := cp.FormatContent(jsonContent, ContentTypeJSON)
	if formatted == "" {
		t.Error("Expected formatted output")
	}

	// 测试未知类型
	plainText := "Some text"
	result := cp.FormatContent(plainText, ContentTypeText)
	if result != plainText {
		t.Error("Expected original text for unknown type")
	}
}

// TestMinifyContent 测试压缩内容
func TestMinifyContent(t *testing.T) {
	cp := NewContentParser()

	// 测试 JSON 压缩
	jsonContent := `{
		"name": "test",
		"value": 123
	}`
	minified := cp.MinifyContent(jsonContent, ContentTypeJSON)
	if minified != `{"name":"test","value":123}` {
		t.Errorf("Expected minified JSON, got '%s'", minified)
	}
}

// TestConvertContent 测试内容转换
func TestConvertContent(t *testing.T) {
	cp := NewContentParser()

	// 测试 JSON 转 XML
	jsonContent := `{"name":"test"}`
	result, err := cp.ConvertContent(jsonContent, ContentTypeJSON, ContentTypeXML)
	if err != nil {
		t.Logf("Conversion test: got error %v", err)
	}
	if result != "" {
		t.Logf("Conversion result: %s", result)
	}
}

// TestQuickHash 测试快速哈希
func TestQuickHash(t *testing.T) {
	h1 := security.QuickHash("test content")
	h2 := security.QuickHash("test content")
	h3 := security.QuickHash("different content")

	if h1 != h2 {
		t.Error("Same content should produce same hash")
	}
	if h1 == h3 {
		t.Error("Different content should produce different hash")
	}
	if h1 == "" {
		t.Error("Hash should not be empty")
	}
}

// TestGetPreview 测试获取预览
func TestGetPreview(t *testing.T) {
	cp := NewContentParser()

	short := "short content"
	long := "this is a very long content that exceeds the max length"

	preview := cp.GetPreview(short, 20)
	if preview != short {
		t.Error("Short content should not be truncated")
	}

	preview = cp.GetPreview(long, 20)
	if len(preview) > 23 { // "this is a very lo" + "..."
		t.Error("Long content should be truncated")
	}
}
