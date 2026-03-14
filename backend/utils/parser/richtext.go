package parser

import (
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// RichTextParser 富文本解析器
type RichTextParser struct{}

// NewRichTextParser 创建富文本解析器
func NewRichTextParser() *RichTextParser {
	return &RichTextParser{}
}

// RichTextParseResult 富文本解析结果
type RichTextParseResult struct {
	Original      string   `json:"original"`       // 原始内容
	Sanitized     string   `json:"sanitized"`      // 消毒后的 HTML
	PlainText     string   `json:"plain_text"`     // 纯文本内容
	IsValid       bool     `json:"is_valid"`       // 是否为有效 HTML
	ParseError    string   `json:"parse_error"`    // 解析错误信息
	HasFormatting bool     `json:"has_formatting"` // 是否有格式
	HasLinks      bool     `json:"has_links"`      // 是否有链接
	HasImages     bool     `json:"has_images"`     // 是否有图片
	HasTables     bool     `json:"has_tables"`     // 是否有表格
	HasLists      bool     `json:"has_lists"`      // 是否有列表
	Links         []string `json:"links"`          // 链接列表
	Images        []string `json:"images"`         // 图片列表
	WordCount     int      `json:"word_count"`     // 字数统计
	ByteSize      int      `json:"byte_size"`      // 字节大小
	LineCount     int      `json:"line_count"`     // 行数
}

// Parse 解析富文本内容
func (p *RichTextParser) Parse(content string) *RichTextParseResult {
	result := &RichTextParseResult{
		Original:   content,
		IsValid:    false,
		LineCount: strings.Count(content, "\n") + 1,
		ByteSize:  len(content),
	}

	if content == "" {
		result.ParseError = "内容为空"
		return result
	}

	// 检查基本 HTML 格式
	result.HasFormatting = hasFormatting(content)
	result.HasLinks = hasLinks(content)
	result.HasImages = hasImages(content)
	result.HasTables = hasTables(content)
	result.HasLists = hasLists(content)

	// 提取链接
	result.Links = extractLinks(content)

	// 提取图片
	result.Images = extractImages(content)

	// 消毒 HTML
	result.Sanitized = p.Sanitize(content)

	// 转换为纯文本
	result.PlainText = toPlainText(content)

	// 统计字数
	result.WordCount = len(strings.Fields(result.PlainText))

	result.IsValid = true

	return result
}

// Sanitize 消毒 HTML 内容
func (p *RichTextParser) Sanitize(content string) string {
	// 使用 bluemonday 进行安全过滤
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("p", "br", "b", "i", "u", "em", "strong", "a", "img",
		"ul", "ol", "li", "h1", "h2", "h3", "h4", "h5", "h6",
		"blockquote", "pre", "code", "span", "div", "table", "thead", "tbody",
		"tr", "th", "td", "hr", "sub", "sup")

	// 允许部分属性
	policy.AllowAttrs("href", "src", "alt", "title", "class").OnElements("a", "img", "span", "div", "table")
	policy.AllowAttrs("colspan", "rowspan").OnElements("td", "th")

	// 允许外部链接
	policy.AllowURLSchemes("http", "https", "mailto")

	return policy.Sanitize(content)
}

// SanitizeStrict 严格消毒（仅保留基本格式）
func (p *RichTextParser) SanitizeStrict(content string) string {
	policy := bluemonday.StrictPolicy()
	policy.AllowElements("p", "br", "b", "i", "u", "em", "strong", "a", "ul", "ol", "li")
	policy.AllowAttrs("href").OnElements("a")

	return policy.Sanitize(content)
}

// FormatHTML 格式化 HTML
func (p *RichTextParser) FormatHTML(content string) string {
	// 简单的格式化（实际可以用 goquery 等库）
	// 规范化标签
	re := regexp.MustCompile(`>\s+<`)
	content = re.ReplaceAllString(content, "><")

	// 添加缩进
	var result strings.Builder
	indent := 0
	parts := regexp.MustCompile(`(?s)<[^>]+>`).Split(content, -1)

	tags := regexp.MustCompile(`(?s)<[^>]+>`).FindAllString(content, -1)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 处理开始标签
		for _, tag := range tags {
			if strings.Contains(part, tag) {
				break
			}
		}

		// 简单的缩进
		result.WriteString(strings.Repeat("  ", indent))
		result.WriteString(part)
		result.WriteString("\n")
	}

	return result.String()
}

// ToPlainText 转换为纯文本
func (p *RichTextParser) ToPlainText(content string) string {
	return toPlainText(content)
}

// ExtractText 提取文本内容（带格式信息）
func (p *RichTextParser) ExtractText(content string) string {
	return toPlainText(content)
}

// ValidateHTML 验证 HTML 语法
func (p *RichTextParser) ValidateHTML(content string) (bool, []string) {
	var errors []string

	// 检查未闭合的标签
	openTags := regexp.MustCompile(`<([a-zA-Z]+)[^>]*(?<!/)>`)
	closeTags := regexp.MustCompile(`</([a-zA-Z]+)>`)

	opens := openTags.FindAllStringSubmatch(content, -1)
	closes := closeTags.FindAllStringSubmatch(content, -1)

	// 简单检查：开始和结束标签数量
	tagStack := []string{}
	openMap := make(map[string]int)
	closeMap := make(map[string]int)

	for _, match := range opens {
		if len(match) > 1 {
			tag := strings.ToLower(match[1])
			// 跳过自闭合标签
			if tag == "br" || tag == "hr" || tag == "img" || tag == "input" || tag == "meta" || tag == "link" {
				continue
			}
			tagStack = append(tagStack, tag)
			openMap[tag]++
		}
	}

	for _, match := range closes {
		if len(match) > 1 {
			tag := strings.ToLower(match[1])
			closeMap[tag]++
		}
	}

	// 检查配对
	for tag, count := range openMap {
		if closeMap[tag] != count {
			errors = append(errors, "标签 <"+tag+"> 未正确闭合")
		}
	}

	// 检查危险标签
	dangerousTags := []string{"script", "iframe", "object", "embed", "form"}
	for _, tag := range dangerousTags {
		re := regexp.MustCompile(`(?i)<` + tag)
		if re.MatchString(content) {
			errors = append(errors, "检测到危险标签: "+tag)
		}
	}

	return len(errors) == 0, errors
}

// ExtractMetadata 提取元数据
func (p *RichTextParser) ExtractMetadata(content string) map[string]string {
	metadata := make(map[string]string)

	// 提取 title
	titleRe := regexp.MustCompile(`(?i)<title>([^<]+)</title>`)
	if match := titleRe.FindStringSubmatch(content); len(match) > 1 {
		metadata["title"] = match[1]
	}

	// 提取 meta 标签
	metaRe := regexp.MustCompile(`(?i)<meta[^>]+>`)
	matches := metaRe.FindAllString(content, -1)
	for _, meta := range matches {
		nameRe := regexp.MustCompile(`(?i)name=["']([^"']+)["']`)
		contentRe := regexp.MustCompile(`(?i)content=["']([^"']+)["']`)

		nameMatch := nameRe.FindStringSubmatch(meta)
		contentMatch := contentRe.FindStringSubmatch(meta)

		if len(nameMatch) > 1 && len(contentMatch) > 1 {
			metadata[nameMatch[1]] = contentMatch[1]
		}
	}

	return metadata
}

// ExtractHeadings 提取所有标题
func (p *RichTextParser) ExtractHeadings(content string) []map[string]string {
	var headings []map[string]string

	re := regexp.MustCompile(`(?i)<h([1-6])[^>]*>([^<]+)</h\1>`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			headings = append(headings, map[string]string{
				"level": match[1],
				"text":  strings.TrimSpace(match[2]),
			})
		}
	}

	return headings
}

// CountElements 统计元素数量
func (p *RichTextParser) CountElements(content string) map[string]int {
	counts := make(map[string]int)

	// 统计常见标签
	tags := []string{"p", "div", "span", "a", "img", "ul", "ol", "li",
		"table", "tr", "td", "th", "h1", "h2", "h3", "h4", "h5", "h6",
		"pre", "code", "blockquote", "br", "hr"}

	for _, tag := range tags {
		re := regexp.MustCompile(`(?i)<` + tag + `[^>]*>`)
		counts[tag] = len(re.FindAllString(content, -1))
	}

	return counts
}

// MinifyHTML 压缩 HTML
func (p *RichTextParser) MinifyHTML(content string) string {
	// 移除注释
	re := regexp.MustCompile(`<!--[\s\S]*?-->`)
	content = re.ReplaceAllString(content, "")

	// 移除多余空白
	re = regexp.MustCompile(`\s+`)
	content = re.ReplaceAllString(content, " ")

	// 移除标签间多余空白
	re = regexp.MustCompile(`>\s+<`)
	content = re.ReplaceAllString(content, "><")

	return strings.TrimSpace(content)
}

// hasFormatting 检查是否有格式标签
func hasFormatting(content string) bool {
	formatTags := []string{"b", "i", "u", "em", "strong", "span", "font", "small", "big"}
	for _, tag := range formatTags {
		re := regexp.MustCompile(`(?i)<` + tag)
		if re.MatchString(content) {
			return true
		}
	}
	return false
}

// hasLinks 检查是否有链接
func hasLinks(content string) bool {
	re := regexp.MustCompile(`(?i)<a\s`)
	return re.MatchString(content)
}

// hasImages 检查是否有图片
func hasImages(content string) bool {
	re := regexp.MustCompile(`(?i)<img\s`)
	return re.MatchString(content)
}

// hasTables 检查是否有表格
func hasTables(content string) bool {
	re := regexp.MustCompile(`(?i)<table`)
	return re.MatchString(content)
}

// hasLists 检查是否有列表
func hasLists(content string) bool {
	re := regexp.MustCompile(`(?i)<(ul|ol)\s`)
	return re.MatchString(content)
}

// extractLinks 提取所有链接
func extractLinks(content string) []string {
	var links []string

	re := regexp.MustCompile(`(?i)href=["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(content, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			link := match[1]
			if !seen[link] {
				seen[link] = true
				links = append(links, link)
			}
		}
	}

	return links
}

// extractImages 提取所有图片
func extractImages(content string) []string {
	var images []string

	re := regexp.MustCompile(`(?i)src=["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(content, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			src := match[1]
			if !seen[src] {
				seen[src] = true
				images = append(images, src)
			}
		}
	}

	return images
}

// toPlainText 转换为纯文本
func toPlainText(content string) string {
	// 移除 HTML 标签
	re := regexp.MustCompile(`(?s)<[^>]+>`)
	text := re.ReplaceAllString(content, "\n")

	// 规范化空白
	re = regexp.MustCompile(`[ \t]+`)
	text = re.ReplaceAllString(text, " ")

	// 移除多余换行
	re = regexp.MustCompile(`\n+`)
	text = re.ReplaceAllString(text, "\n")

	// 转义 HTML 实体
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return strings.TrimSpace(text)
}

// ConvertToMarkdown 将 HTML 转换为 Markdown
func (p *RichTextParser) ConvertToMarkdown(content string) string {
	// 移除 script 和 style 标签
	re := regexp.MustCompile(`(?si)<script[^>]*>.*?</script>`)
	content = re.ReplaceAllString(content, "")

	re = regexp.MustCompile(`(?si)<style[^>]*>.*?</style>`)
	content = re.ReplaceAllString(content, "")

	// 处理标题
	re = regexp.MustCompile(`(?i)<h1[^>]*>([^<]+)</h1>`)
	content = re.ReplaceAllString(content, "# $1\n")

	re = regexp.MustCompile(`(?i)<h2[^>]*>([^<]+)</h2>`)
	content = re.ReplaceAllString(content, "## $1\n")

	re = regexp.MustCompile(`(?i)<h3[^>]*>([^<]+)</h3>`)
	content = re.ReplaceAllString(content, "### $1\n")

	// 处理加粗和斜体
	re = regexp.MustCompile(`(?i)<strong>([^<]+)</strong>`)
	content = re.ReplaceAllString(content, "**$1**")

	re = regexp.MustCompile(`(?i)<b>([^<]+)</b>`)
	content = re.ReplaceAllString(content, "**$1**")

	re = regexp.MustCompile(`(?i)<em>([^<]+)</em>`)
	content = re.ReplaceAllString(content, "*$1*")

	re = regexp.MustCompile(`(?i)<i>([^<]+)</i>`)
	content = re.ReplaceAllString(content, "*$1*")

	// 处理链接
	re = regexp.MustCompile(`(?i)<a[^>]*href=["']([^"']+)["'][^>]*>([^<]+)</a>`)
	content = re.ReplaceAllString(content, "[$2]($1)")

	// 处理图片
	re = regexp.MustCompile(`(?i)<img[^>]*src=["']([^"']+)["'][^>]*alt=["']([^"']+)["'][^>]*>`)
	content = re.ReplaceAllString(content, "![$2]($1)")

	re = regexp.MustCompile(`(?i)<img[^>]*src=["']([^"']+)["'][^>]*>`)
	content = re.ReplaceAllString(content, "![]($1)")

	// 处理换行
	re = regexp.MustCompile(`(?i)<br\s*/?>`)
	content = re.ReplaceAllString(content, "\n")

	// 处理段落
	re = regexp.MustCompile(`(?i)</p>`)
	content = re.ReplaceAllString(content, "\n\n")

	// 移除剩余标签
	re = regexp.MustCompile(`(?s)<[^>]+>`)
	content = re.ReplaceAllString(content, "")

	// 清理空白
	re = regexp.MustCompile(`\n{3,}`)
	content = re.ReplaceAllString(content, "\n\n")

	return strings.TrimSpace(content)
}

// CreateRichText 创建富文本内容
func (p *RichTextParser) CreateRichText(text, format string) string {
	switch format {
	case "bold":
		return "<strong>" + text + "</strong>"
	case "italic":
		return "<em>" + text + "</em>"
	case "underline":
		return "<u>" + text + "</u>"
	case "code":
		return "<code>" + text + "</code>"
	case "link":
		return "<a href=\"" + text + "\">" + text + "</a>"
	default:
		return text
	}
}

// WrapInParagraph 包装为段落
func (p *RichTextParser) WrapInParagraph(text string) string {
	return "<p>" + text + "</p>"
}

// CreateList 创建列表
func (p *RichTextParser) CreateList(items []string, ordered bool) string {
	var result strings.Builder

	if ordered {
		result.WriteString("<ol>\n")
		for _, item := range items {
			result.WriteString("<li>")
			result.WriteString(item)
			result.WriteString("</li>\n")
		}
		result.WriteString("</ol>")
	} else {
		result.WriteString("<ul>\n")
		for _, item := range items {
			result.WriteString("<li>")
			result.WriteString(item)
			result.WriteString("</li>\n")
		}
		result.WriteString("</ul>")
	}

	return result.String()
}
