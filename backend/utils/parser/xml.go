package parser

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// XMLParser XML 解析器
type XMLParser struct{}

// NewXMLParser 创建 XML 解析器
func NewXMLParser() *XMLParser {
	return &XMLParser{}
}

// XMLParseResult XML 解析结果
type XMLParseResult struct {
	Original    string   `json:"original"`    // 原始内容
	Formatted   string   `json:"formatted"`  // 格式化后的内容
	IsValid     bool     `json:"is_valid"`   // 是否为有效 XML
	ParseError  string   `json:"parse_error"`// 解析错误信息
	RootElement string   `json:"root_element"`// 根元素名称
	Elements    []string `json:"elements"`   // 所有元素名称
	Attributes  []string `json:"attributes"`  // 所有属性名称
	Depth       int      `json:"depth"`       // 嵌套深度
	ByteSize    int      `json:"byte_size"`   // 字节大小
	LineCount   int      `json:"line_count"`  // 行数
	HasCDATA    bool     `json:"has_cdata"`   // 是否包含 CDATA
	HasComment  bool     `json:"has_comment"` // 是否包含注释
}

// Parse 解析 XML 内容
func (p *XMLParser) Parse(content string) *XMLParseResult {
	result := &XMLParseResult{
		Original:  content,
		IsValid:   false,
		LineCount: strings.Count(content, "\n") + 1,
		ByteSize:  len(content),
	}

	if content == "" {
		result.ParseError = "内容为空"
		return result
	}

	// 检查特性
	result.HasCDATA = strings.Contains(content, "<![CDATA[")
	result.HasComment = strings.Contains(content, "<!--")

	// 解析 XML
	decoder := xml.NewDecoder(strings.NewReader(content))

	var elements []string
	var attrs []string
	depth := 0
	maxDepth := 0
	var rootElement string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.ParseError = err.Error()
			return result
		}

		switch t := token.(type) {
		case xml.StartElement:
			if rootElement == "" {
				rootElement = t.Name.Local
			}
			elements = append(elements, t.Name.Local)
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
			for _, attr := range t.Attr {
				attrs = append(attrs, attr.Name.Local)
			}
		case xml.EndElement:
			depth--
		}
	}

	result.IsValid = true
	result.RootElement = rootElement
	result.Elements = removeDuplicates(elements)
	result.Attributes = removeDuplicates(attrs)
	result.Depth = maxDepth

	// 格式化
	result.Formatted = p.Format(content)

	return result
}

// Format 格式化 XML
func (p *XMLParser) Format(content string) string {
	var buf bytes.Buffer
	decoder := xml.NewDecoder(strings.NewReader(content))
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return content
		}

		if err := encoder.EncodeToken(token); err != nil {
			return content
		}
	}

	if err := encoder.Flush(); err != nil {
		return content
	}

	return buf.String()
}

// Minify 压缩 XML（移除多余空白）
func (p *XMLParser) Minify(content string) string {
	// 移除注释
	re := regexp.MustCompile(`<!--[\s\S]*?-->`)
	content = re.ReplaceAllString(content, "")

	// 移除处理指令
	re = regexp.MustCompile(`<\?[\s\S]*?\?>`)
	content = re.ReplaceAllString(content, "")

	// 规范化空白
	re = regexp.MustCompile(`>\s+<`)
	content = re.ReplaceAllString(content, "><")

	// 移除多余空行
	re = regexp.MustCompile(`[\r\n]+`)
	content = re.ReplaceAllString(content, "")

	return strings.TrimSpace(content)
}

// Validate 验证 XML 是否有效
func (p *XMLParser) Validate(content string) (bool, string) {
	if content == "" {
		return false, "内容为空"
	}

	decoder := xml.NewDecoder(strings.NewReader(content))

	for {
		_, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, err.Error()
		}
	}

	return true, ""
}

// ConvertToJSON 将 XML 转换为 JSON
func (p *XMLParser) ConvertToJSON(content string) (string, error) {
	var result map[string]interface{}

	if err := xml.Unmarshal([]byte(content), &result); err != nil {
		// 尝试通用转换
		var data interface{}
		if err := xml.Unmarshal([]byte(content), &data); err != nil {
			return "", err
		}
		output, err := json.MarshalIndent(data, "", "  ")
		return string(output), err
	}

	output, err := json.MarshalIndent(result, "", "  ")
	return string(output), err
}

// ConvertJSONToXML 将 JSON 转换为 XML
func (p *XMLParser) ConvertJSONToXML(content string, rootName string) (string, error) {
	if rootName == "" {
		rootName = "root"
	}

	var data interface{}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		return "", err
	}

	output, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	buf.WriteString("<")
	buf.WriteString(rootName)
	buf.WriteString(">\n")
	buf.Write(output)
	buf.WriteString("\n</")
	buf.WriteString(rootName)
	buf.WriteString(">")

	return buf.String(), nil
}

// XPathQuery 简单的 XPath 查询
func (p *XMLParser) XPathQuery(content string, xpath string) (string, error) {
	var result map[string]interface{}
	if err := xml.Unmarshal([]byte(content), &result); err != nil {
		return "", err
	}

	// 简化的 XPath 实现
	parts := strings.Split(strings.TrimPrefix(xpath, "/"), "/")
	current := interface{}(result)

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch c := current.(type) {
		case map[string]interface{}:
			current = c[part]
		default:
			return "", fmt.Errorf("无效的 XPath: %s", xpath)
		}

		if current == nil {
			return "", fmt.Errorf("路径不存在: %s", xpath)
		}
	}

	output, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// GetElements 获取所有元素（带层级）
func (p *XMLParser) GetElements(content string) ([]map[string]string, error) {
	type ElementInfo struct {
		Name     string `xml:"name"`
		Level    int    `xml:"level"`
		Parent   string `xml:"parent"`
		Children int    `xml:"children"`
	}

	var elements []map[string]string

	decoder := xml.NewDecoder(strings.NewReader(content))

	var parents []string
	depth := 0

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			parent := ""
			if len(parents) > 0 {
				parent = parents[len(parents)-1]
			}
			elements = append(elements, map[string]string{
				"name":   t.Name.Local,
				"level":  strconv.Itoa(depth),
				"parent": parent,
			})
			parents = append(parents, t.Name.Local)
			depth++
		case xml.EndElement:
			if len(parents) > 0 {
				parents = parents[:len(parents)-1]
			}
			depth--
		}
	}

	return elements, nil
}

// SchemaValidate 使用内置规则验证 XML
func (p *XMLParser) SchemaValidate(content string) (bool, []string) {
	var errors []string

	// 检查根元素
	decoder := xml.NewDecoder(strings.NewReader(content))

	var rootElement string
	firstElement := true

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, err.Error())
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			if firstElement {
				rootElement = t.Name.Local
				firstElement = false
			}
		case xml.EndElement:
			if t.Name.Local != rootElement && rootElement != "" {
				errors = append(errors, fmt.Sprintf("根元素不匹配: 期望 %s, 实际 %s", rootElement, t.Name.Local))
			}
		}
	}

	// 检查未闭合的标签
	if strings.Count(content, "<") != strings.Count(content, ">") {
		errors = append(errors, "标签未正确闭合")
	}

	// 检查属性引号
	re := regexp.MustCompile(`\s+\w+=[^"']\S*`)
	matches := re.FindAllString(content, -1)
	for _, match := range matches {
		errors = append(errors, fmt.Sprintf("属性值未使用引号: %s", match))
	}

	return len(errors) == 0, errors
}

func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
