package parser

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ContentType 内容类型
type ContentType string

const (
	ContentTypeUnknown  ContentType = "unknown"
	ContentTypeText     ContentType = "text"
	ContentTypeMarkdown ContentType = "markdown"
	ContentTypeJSON     ContentType = "json"
	ContentTypeXML      ContentType = "xml"
	ContentTypeCSV      ContentType = "csv"
	ContentTypeHTML     ContentType = "html"
	ContentTypeCode     ContentType = "code"
)

// ContentParser 统一内容解析器
type ContentParser struct {
	markdownParser *MarkdownParser
	jsonParser     *JSONParser
	xmlParser      *XMLParser
	csvParser      *CSVParser
	richTextParser *RichTextParser
}

// NewContentParser 创建统一内容解析器
func NewContentParser() *ContentParser {
	return &ContentParser{
		markdownParser: NewMarkdownParser(),
		jsonParser:     NewJSONParser(),
		xmlParser:      NewXMLParser(),
		csvParser:      NewCSVParser(),
		richTextParser: NewRichTextParser(),
	}
}

// UnifiedParseResult 统一解析结果
type UnifiedParseResult struct {
	ContentType ContentType   `json:"content_type"` // 检测到的内容类型
	IsValid     bool          `json:"is_valid"`     // 内容是否有效
	CanParse    bool          `json:"can_parse"`    // 是否可以被解析
	ParsedData  interface{}   `json:"parsed_data"`  // 解析后的数据
	Error       string        `json:"error"`        // 错误信息
	Metadata    ContentMeta   `json:"metadata"`     // 元数据
}

// ContentMeta 内容元数据
type ContentMeta struct {
	OriginalFormat string `json:"original_format"` // 原始格式
	Language      string `json:"language"`        // 编程语言
	LineCount     int    `json:"line_count"`      // 行数
	ByteSize      int    `json:"byte_size"`       // 字节大小
	HasLineNumbers bool  `json:"has_line_numbers"` // 是否有行号
	IsEncrypted   bool   `json:"is_encrypted"`    // 是否加密
	DetectedFrom  string `json:"detected_from"`   // 检测来源
}

// Parse 自动检测并解析内容
func (cp *ContentParser) Parse(content string) *UnifiedParseResult {
	result := &UnifiedParseResult{
		ContentType: ContentTypeUnknown,
		IsValid:     false,
		CanParse:    false,
		Metadata: ContentMeta{
			OriginalFormat: "unknown",
			LineCount:       strings.Count(content, "\n") + 1,
			ByteSize:        len(content),
		},
	}

	if content == "" {
		result.Error = "内容为空"
		return result
	}

	// 自动检测内容类型
	contentType := cp.DetectContentType(content)
	result.ContentType = contentType

	// 根据类型解析
	switch contentType {
	case ContentTypeJSON:
		jsonResult := cp.jsonParser.Parse(content)
		result.IsValid = jsonResult.IsValid
		result.CanParse = jsonResult.IsValid
		result.ParsedData = jsonResult
		result.Metadata.Language = "json"
		result.Metadata.OriginalFormat = "json"
		if !jsonResult.IsValid {
			result.Error = jsonResult.ParseError
		}

	case ContentTypeXML:
		xmlResult := cp.xmlParser.Parse(content)
		result.IsValid = xmlResult.IsValid
		result.CanParse = xmlResult.IsValid
		result.ParsedData = xmlResult
		result.Metadata.Language = "xml"
		result.Metadata.OriginalFormat = "xml"
		if !xmlResult.IsValid {
			result.Error = xmlResult.ParseError
		}

	case ContentTypeCSV:
		csvResult := cp.csvParser.Parse(content)
		result.IsValid = csvResult.IsValid
		result.CanParse = csvResult.IsValid
		result.ParsedData = csvResult
		result.Metadata.Language = "csv"
		result.Metadata.OriginalFormat = "csv"
		if !csvResult.IsValid {
			result.Error = csvResult.ParseError
		}

	case ContentTypeMarkdown:
		parseResult := cp.markdownParser.Parse(content)
		result.IsValid = parseResult.IsValid
		result.CanParse = true
		result.ParsedData = parseResult
		result.Metadata.Language = "markdown"
		result.Metadata.OriginalFormat = "markdown"
		result.Metadata.HasLineNumbers = false

	case ContentTypeHTML:
		parseResult := cp.richTextParser.Parse(content)
		result.IsValid = parseResult.IsValid
		result.CanParse = parseResult.IsValid
		result.ParsedData = parseResult
		result.Metadata.Language = "html"
		result.Metadata.OriginalFormat = "html"

	case ContentTypeCode:
		result.IsValid = true
		result.CanParse = true
		result.Metadata.Language = cp.detectLanguage(content)
		result.Metadata.OriginalFormat = "code"
		result.Metadata.HasLineNumbers = hasLineNumbers(content)

	case ContentTypeText:
		result.IsValid = true
		result.CanParse = true
		result.Metadata.OriginalFormat = "text"
		result.ParsedData = map[string]interface{}{
			"content": content,
			"length":  len(content),
		}
	}

	return result
}

// DetectContentType 检测内容类型
func (cp *ContentParser) DetectContentType(content string) ContentType {
	content = strings.TrimSpace(content)

	// 空内容
	if content == "" {
		return ContentTypeUnknown
	}

	// JSON 检测
	if isJSON(content) {
		return ContentTypeJSON
	}

	// XML 检测
	if isXML(content) {
		return ContentTypeXML
	}

	// CSV 检测
	if isCSV(content) {
		return ContentTypeCSV
	}

	// Markdown 检测
	if isMarkdown(content) {
		return ContentTypeMarkdown
	}

	// HTML 检测
	if isHTML(content) {
		return ContentTypeHTML
	}

	// 代码检测
	if isCode(content) {
		return ContentTypeCode
	}

	return ContentTypeText
}

// isJSON 检测是否为 JSON
func isJSON(content string) bool {
	content = strings.TrimSpace(content)
	// 必须以 { 或 [ 开头
	if !(strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[")) {
		return false
	}

	// 尝试解析
	var v interface{}
	if err := json.Unmarshal([]byte(content), &v); err != nil {
		return false
	}

	return true
}

// isXML 检测是否为 XML
func isXML(content string) bool {
	content = strings.TrimSpace(content)
	// 必须以 <?xml 或 < 开头
	if !strings.HasPrefix(content, "<?xml") && !strings.HasPrefix(content, "<") {
		return false
	}

	// 简单检查：是否有匹配的标签
	openCount := strings.Count(content, "<")
	closeCount := strings.Count(content, ">")

	if openCount < 2 || closeCount < 2 {
		return false
	}

	// 检查是否有自闭合标签或正常标签
	re := regexp.MustCompile(`<[^>]+>`)
	matches := re.FindAllString(content, -1)
	if len(matches) < 2 {
		return false
	}

	return true
}

// isCSV 检测是否为 CSV
func isCSV(content string) bool {
	content = strings.TrimSpace(content)
	lines := strings.Split(content, "\n")

	if len(lines) < 2 {
		return false
	}

	// 检测分隔符
	delimiter := detectCSVDelimiter(lines[0])

	// 检查每行的字段数是否一致
	firstLineCount := strings.Count(lines[0], string(delimiter)) + 1
	if firstLineCount < 2 {
		return false
	}

	// 检查前几行
	for i := 1; i < min(3, len(lines)); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		count := strings.Count(line, string(delimiter)) + 1
		if count != firstLineCount {
			return false
		}
	}

	return true
}

// detectCSVDelimiter 检测 CSV 分隔符
func detectCSVDelimiter(line string) rune {
	delimiters := []rune{',', ';', '\t', '|'}
	bestDelimiter := ','
	maxCount := 0

	for _, d := range delimiters {
		count := strings.Count(line, string(d))
		if count > maxCount {
			maxCount = count
			bestDelimiter = d
		}
	}

	return bestDelimiter
}

// isMarkdown 检测是否为 Markdown
func isMarkdown(content string) bool {
	content = strings.TrimSpace(content)

	// 检查 Markdown 特征
	patterns := []string{
		`^#{1,6}\s+`,           // 标题
		`^\*\s+`,               // 无序列表
		`^-\s+`,                // 无序列表
		`^\d+\.\s+`,            // 有序列表
		`\[.+?\]\(.+?\)`,       // 链接
		`!\[.*?\]\(.+?\)`,      // 图片
		`\*\*.*?\*\*`,          // 加粗
		`\*.*?\*`,              // 斜体
		`\|.+\|`,               // 表格
	}

	count := 0
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(content) {
			count++
		}
	}

	// 至少匹配 2 个特征
	return count >= 2
}

// isHTML 检测是否为 HTML
func isHTML(content string) bool {
	content = strings.TrimSpace(content)

	// 必须以 < 开头
	if !strings.HasPrefix(content, "<") {
		return false
	}

	// 检查常见 HTML 标签
	htmlTags := []string{"<html", "<div", "<span", "<p>", "<table", "<ul", "<ol", "<li", "<a ", "<img", "<br", "<head", "<body"}
	for _, tag := range htmlTags {
		if strings.Contains(strings.ToLower(content), tag) {
			return true
		}
	}

	// 检查标签配对
	re := regexp.MustCompile(`<([a-zA-Z]+)[^>]*>`)
	matches := re.FindAllString(content, -1)

	return len(matches) >= 2
}

// isCode 检测是否为代码
func isCode(content string) bool {
	content = strings.TrimSpace(content)

	// 检查代码特征
	codePatterns := []string{
		`^package\s+\w+`,                          // Go
		`^import\s+`,                              // 导入
		`^func\s+`,                                // 函数
		`^class\s+\w+`,                           // 类
		`^def\s+`,                                // Python
		`^function\s+`,                           // JS
		`^const\s+\w+\s*=`,                       // JS 常量
		`^let\s+\w+\s*=`,                         // JS 变量
		`^var\s+\w+\s*=`,                         // JS 变量
		`^public\s+`,                             // Java/C#
		`^private\s+`,                            // Java/C#
		`^import\s+\w+`,                          // Java/Python
		`^from\s+\w+\s+import`,                   // Python
		`^#include\s*<`,                          // C/C++
		`^using\s+namespace`,                     // C++
		`^\$\w+\s*=\s*`,                          // PHP/Shell
		`^\s*SELECT\s+`,                          // SQL
		`^\s*INSERT\s+`,                          // SQL
		`^\s*CREATE\s+TABLE`,                     // SQL
	}

	count := 0
	for _, pattern := range codePatterns {
		re := regexp.MustCompile(`(?im)` + pattern)
		if re.MatchString(content) {
			count++
		}
	}

	// 代码块特征
	hasCodeBlock := regexp.MustCompile("```").MatchString(content)
	if hasCodeBlock {
		count += 2
	}

	// 括号匹配
	openBraces := strings.Count(content, "{") + strings.Count(content, "(") + strings.Count(content, "[")
	closeBraces := strings.Count(content, "}") + strings.Count(content, ")") + strings.Count(content, "]")
	balanced := openBraces > 0 && abs(openBraces-closeBraces) < 3

	if balanced && (count >= 1 || hasCodeBlock) {
		return true
	}

	return false
}

// detectLanguage 检测编程语言
func (cp *ContentParser) detectLanguage(content string) string {
	content = strings.ToLower(content)

	// 简单语言检测
	patterns := map[string][]string{
		"go":       {"package ", "func ", "import (", "go func", ":="},
		"python":   {"def ", "import ", "from ", "class ", "print("},
		"javascript": {"function ", "const ", "let ", "var ", "=>", "require("},
		"typescript": {"interface ", ": string", ": number", ": boolean", "<T>"},
		"java":     {"public class", "private ", "public static", "system.out"},
		"c":        {"#include", "int main", "printf", "scanf", "void "},
		"cpp":      {"#include <iostream>", "std::", "cout", "cin", "namespace "},
		"csharp":   {"using system", "namespace ", "public class", "console.writeline"},
		"php":      {"<?php", "echo ", "$", "function ", "class "},
		"ruby":     {"def ", "end", "require ", "class ", "puts "},
		"rust":     {"fn ", "let mut", "impl ", "pub fn", "use "},
		"sql":      {"select ", "insert ", "update ", "delete ", "from ", "where "},
		"html":     {"<html", "<div", "<span", "<!doctype"},
		"css":      {"{", "}", "margin", "padding", "color:", "background:"},
		"shell":    {"#!/bin/bash", "echo ", "if [", "fi", "done"},
		"yaml":     {"- ", ": ", "---"},
		"json":     {"{", "}", ":"},
	}

	for lang, patterns := range patterns {
		count := 0
		for _, pattern := range patterns {
			if strings.Contains(content, pattern) {
				count++
			}
		}
		if count >= 2 {
			return lang
		}
	}

	return "text"
}

// hasLineNumbers 检查是否有行号
func hasLineNumbers(content string) bool {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return false
	}

	// 检查前几行是否以数字开头
	count := 0
	for i := 0; i < min(5, len(lines)); i++ {
		line := strings.TrimSpace(lines[i])
		if match, _ := regexp.MatchString(`^\d+\s+`, line); match {
			count++
		}
	}

	return count >= 3
}

// GetPreview 获取内容预览
func (cp *ContentParser) GetPreview(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "..."
}

// FormatContent 格式化内容
func (cp *ContentParser) FormatContent(content string, contentType ContentType) string {
	switch contentType {
	case ContentTypeJSON:
		return cp.jsonParser.Format(content)
	case ContentTypeXML:
		return cp.xmlParser.Format(content)
	case ContentTypeCSV:
		return cp.csvParser.Format(content)
	case ContentTypeMarkdown:
		return cp.markdownParser.FormatMarkdown(content)
	case ContentTypeHTML:
		return cp.richTextParser.FormatHTML(content)
	default:
		return content
	}
}

// MinifyContent 压缩内容
func (cp *ContentParser) MinifyContent(content string, contentType ContentType) string {
	switch contentType {
	case ContentTypeJSON:
		return cp.jsonParser.Minify(content)
	case ContentTypeXML:
		return cp.xmlParser.Minify(content)
	case ContentTypeHTML:
		return cp.richTextParser.MinifyHTML(content)
	default:
		return content
	}
}

// ConvertContent 转换内容格式
func (cp *ContentParser) ConvertContent(content string, fromType, toType ContentType) (string, error) {
	switch {
	case fromType == ContentTypeCSV && toType == ContentTypeJSON:
		return cp.csvParser.ToJSON(content)
	case fromType == ContentTypeCSV && toType == ContentTypeMarkdown:
		return cp.csvParser.ToMarkdown(content)
	case fromType == ContentTypeCSV && toType == ContentTypeHTML:
		return cp.csvParser.ToHTML(content)
	case fromType == ContentTypeJSON && toType == ContentTypeXML:
		return cp.xmlParser.ConvertJSONToXML(content, "root")
	case fromType == ContentTypeXML && toType == ContentTypeJSON:
		return cp.xmlParser.ConvertToJSON(content)
	case fromType == ContentTypeHTML && toType == ContentTypeMarkdown:
		return cp.richTextParser.ConvertToMarkdown(content), nil
	case fromType == ContentTypeHTML && toType == ContentTypeText:
		return cp.richTextParser.ToPlainText(content), nil
	default:
		return "", ErrConversionNotSupported
	}
}

// ErrConversionNotSupported 转换不支持错误
var ErrConversionNotSupported = &ParseError{Message: "不支持的格式转换"}

// ParseError 解析错误
type ParseError struct {
	Message string
}

func (e *ParseError) Error() string {
	return e.Message
}

// abs 计算绝对值
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// min 取最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
