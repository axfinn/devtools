package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// MarkdownParser Markdown 解析器
type MarkdownParser struct{}

// NewMarkdownParser 创建 Markdown 解析器
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

// MarkdownParseResult Markdown 解析结果
type MarkdownParseResult struct {
	HTML        string   `json:"html"`         // 转换后的 HTML
	TOC         string   `json:"toc"`          // 目录结构
	WordCount   int      `json:"word_count"`  // 字数统计
	LineCount   int      `json:"line_count"`  // 行数统计
	HasCode     bool     `json:"has_code"`    // 是否包含代码块
	HasImage    bool     `json:"has_image"`   // 是否包含图片
	HasTable    bool     `json:"has_table"`   // 是否包含表格
	CodeLangs   []string `json:"code_langs"`  // 代码语言列表
	IsValid     bool     `json:"is_valid"`    // 是否为有效的 Markdown
	ParseError  string   `json:"parse_error"` // 解析错误信息
}

// Parse 解析 Markdown 内容
func (p *MarkdownParser) Parse(content string) *MarkdownParseResult {
	if content == "" {
		return &MarkdownParseResult{
			IsValid:    true,
			WordCount:  0,
			LineCount:  0,
		}
	}

	result := &MarkdownParseResult{
		IsValid:   true,
		LineCount: strings.Count(content, "\n") + 1,
	}

	// 统计字数
	contentWithoutCode := p.removeCodeBlocks(content)
	result.WordCount = len(strings.Fields(contentWithoutCode))

	// 检查特性
	result.HasCode = containsCodeBlock(content)
	result.HasImage = containsImage(content)
	result.HasTable = containsTable(content)

	if result.HasCode {
		result.CodeLangs = extractCodeLanguages(content)
	}

	// 配置 Goldmark 解析器
	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,           // GitHub Flavored Markdown
			extension.Table,         // 表格支持
			extension.Strikethrough, // 删除线支持
			extension.TaskList,      // 任务列表支持
			extension.Linkify,       // 自动链接
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // 自动生成标题 ID
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // 允许生成一些 HTML
		),
	)

	// 转换为 HTML
	var buf strings.Builder
	if err := markdown.Convert([]byte(content), &buf); err != nil {
		result.IsValid = false
		result.ParseError = err.Error()
		return result
	}
	result.HTML = buf.String()

	// 生成 TOC
	result.TOC = p.generateTOC(content)

	return result
}

// removeCodeBlocks 移除代码块，只保留文本内容用于字数统计
func (p *MarkdownParser) removeCodeBlocks(content string) string {
	re := regexp.MustCompile("```[\\s\\S]*?```")
	content = re.ReplaceAllString(content, "")
	re = regexp.MustCompile("`[^`]+`")
	content = re.ReplaceAllString(content, "")
	return content
}

// containsCodeBlock 检查是否包含代码块
func containsCodeBlock(content string) bool {
	re := regexp.MustCompile("```")
	return re.MatchString(content)
}

// containsImage 检查是否包含图片
func containsImage(content string) bool {
	re := regexp.MustCompile(`!\[.*?\]\(.*?\)`)
	return re.MatchString(content)
}

// containsTable 检查是否包含表格
func containsTable(content string) bool {
	re := regexp.MustCompile(`\|.*\|`)
	lines := strings.Split(content, "\n")
	count := 0
	for _, line := range lines {
		if re.MatchString(strings.TrimSpace(line)) {
			count++
		}
	}
	return count >= 2
}

// extractCodeLanguages 提取代码语言
func extractCodeLanguages(content string) []string {
	re := regexp.MustCompile("```(\\w+)")
	matches := re.FindAllStringSubmatch(content, -1)
	langs := make([]string, 0)
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			lang := match[1]
			if !seen[lang] {
				seen[lang] = true
				langs = append(langs, lang)
			}
		}
	}
	return langs
}

// generateTOC 生成目录结构
func (p *MarkdownParser) generateTOC(content string) string {
	re := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	matches := re.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return ""
	}

	var toc strings.Builder
	toc.WriteString("<ul class=\"toc\">")
	for _, match := range matches {
		if len(match) > 2 {
			level := len(match[1])
			title := strings.TrimSpace(match[2])
			// 生成锚点 ID
			id := strings.ToLower(title)
			id = regexp.MustCompile(`[^\w]+`).ReplaceAllString(id, "-")
			id = strings.Trim(id, "-")
			toc.WriteString(`<li class="toc-level-`)
			toc.WriteString(fmt.Sprintf("%d", level))
			toc.WriteString(`"><a href="#`)
			toc.WriteString(id)
			toc.WriteString(`">`)
			toc.WriteString(title)
			toc.WriteString("</a></li>")
		}
	}
	toc.WriteString("</ul>")
	return toc.String()
}

// FormatMarkdown 格式化 Markdown（美化排版）
func (p *MarkdownParser) FormatMarkdown(content string) string {
	lines := strings.Split(content, "\n")
	var formatted []string

	for i, line := range lines {
		// 移除行尾空格
		line = strings.TrimRight(line, " ")

		// 代码块前后空行
		if strings.HasPrefix(line, "```") {
			if i > 0 && strings.TrimSpace(lines[i-1]) != "" {
				formatted = append(formatted, "")
			}
		}

		formatted = append(formatted, line)

		if strings.HasPrefix(line, "```") && i < len(lines)-1 && strings.TrimSpace(lines[i+1]) != "" {
			formatted = append(formatted, "")
		}
	}

	// 移除多余空行
	var result []string
	prevEmpty := false
	for _, line := range formatted {
		isEmpty := strings.TrimSpace(line) == ""
		if isEmpty && prevEmpty {
			continue
		}
		result = append(result, line)
		prevEmpty = isEmpty
	}

	return strings.Join(result, "\n")
}
