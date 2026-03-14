package utils

import (
	"regexp"
	"strings"
)

// CodeAnalyzer 代码分析器
// 用于分析代码内容并提供统计信息

// CodeAnalysisResult 代码分析结果
type CodeAnalysisResult struct {
	Language      string   `json:"language"`
	Lines         int      `json:"lines"`
	CodeLines     int      `json:"code_lines"`
	CommentLines  int      `json:"comment_lines"`
	BlankLines    int      `json:"blank_lines"`
	Functions     int      `json:"functions"`
	Classes       int      `json:"classes"`
	Imports       int      `json:"imports"`
	HasSyntaxError bool    `json:"has_syntax_error"`
	Summary       string   `json:"summary"`
}

// AnalyzeCode 分析代码内容
func AnalyzeCode(content string, language string) *CodeAnalysisResult {
	if content == "" {
		return &CodeAnalysisResult{
			Language:    "text",
			Lines:       0,
			CodeLines:   0,
			Summary:    "空内容",
		}
	}

	result := &CodeAnalysisResult{
		Language: language,
	}

	// 统计行数
	lines := strings.Split(content, "\n")
	result.Lines = len(lines)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result.BlankLines++
			continue
		}

		// 检测注释行
		if isCommentLine(trimmed, language) {
			result.CommentLines++
		} else {
			result.CodeLines++
		}
	}

	// 统计函数/方法数量
	result.Functions = countFunctions(content, language)

	// 统计类/结构体数量
	result.Classes = countClasses(content, language)

	// 统计导入语句数量
	result.Imports = countImports(content, language)

	// 生成摘要
	result.Summary = generateSummary(result)

	return result
}

// isCommentLine 检查是否为注释行
func isCommentLine(line string, language string) bool {
	trimmed := strings.TrimSpace(line)

	// 单行注释
	singleLineComments := map[string][]string{
		"default": {"//", "#"},
		"python":  {"#"},
		"ruby":    {"#"},
		"shell":   {"#"},
		"bash":    {"#"},
		"yaml":    {"#"},
		"sql":     {"--"},
		"lua":     {"--"},
		"haskell": {"--"},
		"elixir":  {"#"},
		"erlang":  {"%"},
		"makefile": {"#"},
	}

	comments, ok := singleLineComments[language]
	if !ok {
		comments = singleLineComments["default"]
	}

	for _, c := range comments {
		if strings.HasPrefix(trimmed, c) {
			return true
		}
	}

	// 多行注释开始/结束
	if strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*/") ||
		strings.HasPrefix(trimmed, "<!--") || strings.HasPrefix(trimmed, "<!--") {
		return true
	}

	return false
}

// countFunctions 统计函数/方法数量
func countFunctions(content string, language string) int {
	patterns := map[string]*regexp.Regexp{
		"default":   regexp.MustCompile(`(?i)(func\s+\w+|function\s+\w+|def\s+\w+|sub\s+\w+)`),
		"python":    regexp.MustCompile(`(?i)def\s+\w+`),
		"javascript": regexp.MustCompile(`(?i)(function\s+\w+|const\s+\w+\s*=\s*(async\s*)?\(|let\s+\w+\s*=\s*(async\s*)?\(|=>\s*\{)`),
		"typescript": regexp.MustCompile(`(?i)(function\s+\w+|const\s+\w+\s*=\s*(async\s*)?\(|=>\s*\{|:\s*(async\s*)?function)`),
		"go":        regexp.MustCompile(`(?i)func\s+(\w+|\()`),
		"rust":      regexp.MustCompile(`(?i)fn\s+\w+`),
		"java":      regexp.MustCompile(`(?i)(public|private|protected)\s+(static\s+)?void\s+\w+`),
		"c":         regexp.MustCompile(`(?i)^\s*\w+\s+\w+\s*\([^)]*\)\s*\{`),
		"cpp":       regexp.MustCompile(`(?i)^\s*\w+::\w+\s*\([^)]*\)\s*\{`),
		"csharp":    regexp.MustCompile(`(?i)(public|private|protected|internal)\s+(static\s+)?(void|async|Task)\s+\w+`),
		"php":       regexp.MustCompile(`(?i)function\s+\w+`),
		"ruby":      regexp.MustCompile(`(?i)def\s+\w+`),
		"swift":     regexp.MustCompile(`(?i)func\s+\w+`),
		"kotlin":    regexp.MustCompile(`(?i)fun\s+\w+`),
		"scala":     regexp.MustCompile(`(?i)def\s+\w+`),
	}

	re, ok := patterns[language]
	if !ok {
		re = patterns["default"]
	}

	matches := re.FindAllString(content, -1)
	return len(matches)
}

// countClasses 统计类/结构体数量
func countClasses(content string, language string) int {
	patterns := map[string]*regexp.Regexp{
		"default":    regexp.MustCompile(`(?i)(class\s+\w+|struct\s+\w+)`),
		"python":     regexp.MustCompile(`(?i)class\s+\w+`),
		"javascript": regexp.MustCompile(`(?i)class\s+\w+`),
		"typescript": regexp.MustCompile(`(?i)(class|interface)\s+\w+`),
		"go":         regexp.MustCompile(`(?i)type\s+\w+\s+(struct|interface)`),
		"rust":       regexp.MustCompile(`(?i)(struct|enum)\s+\w+`),
		"java":       regexp.MustCompile(`(?i)(public\s+)?class\s+\w+`),
		"csharp":     regexp.MustCompile(`(?i)(public\s+)?class\s+\w+`),
		"php":        regexp.MustCompile(`(?i)class\s+\w+`),
		"ruby":       regexp.MustCompile(`(?i)class\s+\w+`),
		"swift":      regexp.MustCompile(`(?i)class\s+\w+`),
		"kotlin":     regexp.MustCompile(`(?i)(class|data class)\s+\w+`),
	}

	re, ok := patterns[language]
	if !ok {
		re = patterns["default"]
	}

	matches := re.FindAllString(content, -1)
	return len(matches)
}

// countImports 统计导入语句数量
func countImports(content string, language string) int {
	patterns := map[string]*regexp.Regexp{
		"default":   regexp.MustCompile(`(?i)(import\s+|require\s*\()`),
		"go":        regexp.MustCompile(`(?i)import\s+`),
		"python":    regexp.MustCompile(`(?i)(import\s+|from\s+\w+\s+import)`),
		"javascript": regexp.MustCompile(`(?i)(require\s*\(|import\s+.*from)`),
		"typescript": regexp.MustCompile(`(?i)(require\s*\(|import\s+.*from)`),
		"java":      regexp.MustCompile(`(?i)import\s+`),
		"csharp":    regexp.MustCompile(`(?i)using\s+`),
		"php":       regexp.MustCompile(`(?i)(require|include)`),
		"ruby":      regexp.MustCompile(`(?i)require\s+`),
	}

	re, ok := patterns[language]
	if !ok {
		re = patterns["default"]
	}

	matches := re.FindAllString(content, -1)
	return len(matches)
}

// generateSummary 生成摘要
func generateSummary(result *CodeAnalysisResult) string {
	lang := strings.ToUpper(result.Language)

	switch {
	case result.Language == "text" || result.Language == "":
		return "纯文本内容"
	case result.CodeLines == 0:
		return "空代码文件"
	default:
		return lang + " 代码，" + formatNumber(result.CodeLines) + " 行代码，" +
			formatNumber(result.Functions) + " 个函数，" + formatNumber(result.Classes) + " 个类"
	}
}

// formatNumber 格式化数字
func formatNumber(n int) string {
	if n >= 1000 {
		return string(rune('0' + n/1000%10)) + "," + string(rune('0'+(n/100)%10)) + string(rune('0'+(n/10)%10)) + string(rune('0'+n%10))
	}
	return string(rune('0' + n%10))
}
