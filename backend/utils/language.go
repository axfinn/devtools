package utils

import (
	"regexp"
	"strings"
)

// LanguageDetector 语言检测器
// 基于内容特征自动检测编程语言

// commonPatterns 常用模式
var (
	goFuncPattern = regexp.MustCompile(`(?i)func\s+(\w+|\()`)
	goPackagePattern = regexp.MustCompile(`(?i)package\s+\w+`)
	goImportPattern = regexp.MustCompile(`(?i)import\s+(\(|")`)
	goFmtPattern = regexp.MustCompile(`(?i)fmt\.`)

	jsFuncPattern = regexp.MustCompile(`(?i)(function\s+\w+|const\s+\w+\s*=|let\s+\w+\s*=|var\s+\w+\s*=|=>\s*\{)`)
	jsConsolePattern = regexp.MustCompile(`(?i)console\.(log|error|warn)`)
	jsImportPattern = regexp.MustCompile(`(?i)(require\s*\(|module\.exports|import\s+.*from)`)

	pyDefPattern = regexp.MustCompile(`(?i)def\s+\w+\s*\(`)
	pyClassPattern = regexp.MustCompile(`(?i)class\s+\w+\s*(:|\()`)
	pyImportPattern = regexp.MustCompile(`(?i)(import\s+\w+|from\s+\w+\s+import)`)

	tsTypePattern = regexp.MustCompile(`:\s*(string|number|boolean|any|void|never|unknown|object)\b`)
	tsInterfacePattern = regexp.MustCompile(`(?i)interface\s+\w+`)

	javaPublicPattern = regexp.MustCompile(`(?i)public\s+(static\s+)?(void|class|interface|enum)`)
	javaSystemPattern = regexp.MustCompile(`(?i)System\.out\.print`)

	cIncludePattern = regexp.MustCompile(`(?i)#include\s+[<"]`)
	cMainPattern = regexp.MustCompile(`(?i)int\s+main\s*\(`)
	cPrintfPattern = regexp.MustCompile(`(?i)printf\s*\(`)

	cppStdPattern = regexp.MustCompile(`(?i)std::`)
	cppIostreamPattern = regexp.MustCompile(`(?i)#include\s+<iostream>`)

	csUsingPattern = regexp.MustCompile(`(?i)using\s+System`)
	csNamespacePattern = regexp.MustCompile(`(?i)namespace\s+\w+`)

	phpTagPattern = regexp.MustCompile(`<\?php`)
	phpVarPattern = regexp.MustCompile(`\$\w+\s*=`)

	rubyDefPattern = regexp.MustCompile(`(?i)def\s+\w+`)
	rubyClassPattern = regexp.MustCompile(`(?i)class\s+\w+`)

	swiftFuncPattern = regexp.MustCompile(`(?i)func\s+\w+`)
	swiftVarPattern = regexp.MustCompile(`(?i)(var\s+\w+\s*:|let\s+\w+\s*:)`)

	ktFunPattern = regexp.MustCompile(`(?i)fun\s+\w+`)
	ktValPattern = regexp.MustCompile(`(?i)(val\s+\w+\s*:|var\s+\w+\s*:)`)

	htmlTagPattern = regexp.MustCompile(`(?i)<(html|head|body|div|span|p|a|img|script|style|link)`)
	htmlDoctypePattern = regexp.MustCompile(`(?i)<!DOCTYPE\s+html`)

	cssSelectorPattern = regexp.MustCompile(`(\.|#)[\w-]+\s*\{`)
	cssPropPattern = regexp.MustCompile(`[\w-]+\s*:\s*[^;]+;`)

	jsonPattern = regexp.MustCompile(`^\s*\{.*".*":\s*`)

	yamlPattern = regexp.MustCompile(`^[\w-]+:\s*$`)

	sqlSelectPattern = regexp.MustCompile(`(?i)(SELECT\s+.*FROM|INSERT\s+INTO|UPDATE\s+.*SET|DELETE\s+FROM)`)

	bashShebangPattern = regexp.MustCompile(`^#!.*/(bash|sh)`)
	bashEchoPattern = regexp.MustCompile(`(?i)(echo\s+|printf\s+|source\s+)`)

	markdownHeaderPattern = regexp.MustCompile("(?mi)^#{1,6}\\s+")
	markdownListPattern = regexp.MustCompile("(?mi)^\\s*[-*+]\\s+")

	dockerfilePattern = regexp.MustCompile(`(?i)(FROM\s+\w+|RUN\s+|CMD\s+|ENTRYPOINT\s+|ENV\s+|COPY\s+)`)

	markdownIndicators = []string{"```", "**", "__", "*", "_", "[]("}
)

// DetectLanguage 检测编程语言
func DetectLanguage(content string) string {
	if content == "" {
		return "text"
	}

	// 简单检测纯文本
	lines := strings.Split(content, "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	if nonEmptyLines <= 1 {
		trimmed := strings.TrimSpace(content)
		if len(trimmed) < 50 && !strings.Contains(trimmed, "{") && !strings.Contains(trimmed, "(") && !strings.Contains(trimmed, "[") {
			return "text"
		}
	}

	// 计分
	scores := map[string]int{}

	// Go (需要优先检测，因为 package 声明是 Go 特有)
	if goPackagePattern.MatchString(content) {
		// 有 package 声明，一定是 Go
		scores["go"] += 5
	}
	if goFuncPattern.MatchString(content) || goImportPattern.MatchString(content) || goFmtPattern.MatchString(content) {
		scores["go"] += 2
	}

	// Swift (需要 package 声明不存在才检测)
	if swiftFuncPattern.MatchString(content) && !goPackagePattern.MatchString(content) {
		scores["swift"] += 3
	}
	if swiftVarPattern.MatchString(content) && !goPackagePattern.MatchString(content) {
		scores["swift"] += 2
	}

	// JavaScript/TypeScript
	if jsFuncPattern.MatchString(content) {
		scores["javascript"] += 2
	}
	if jsConsolePattern.MatchString(content) || jsImportPattern.MatchString(content) {
		scores["javascript"] += 1
	}

	// TypeScript
	if tsTypePattern.MatchString(content) || tsInterfacePattern.MatchString(content) {
		scores["typescript"] += 2
	}

	// Python
	if pyDefPattern.MatchString(content) || pyClassPattern.MatchString(content) {
		scores["python"] += 2
	}
	if pyImportPattern.MatchString(content) {
		scores["python"] += 1
	}

	// Java
	if javaPublicPattern.MatchString(content) || javaSystemPattern.MatchString(content) {
		scores["java"] += 2
	}

	// C
	if cIncludePattern.MatchString(content) && cMainPattern.MatchString(content) {
		scores["c"] += 2
	}
	if cPrintfPattern.MatchString(content) && !cppStdPattern.MatchString(content) { // C++ 也会用 printf，但 std:: 是 C++ 特有
		scores["c"] += 1
	}

	// C++ (需要比 C 更严格)
	if cppStdPattern.MatchString(content) || cppIostreamPattern.MatchString(content) {
		scores["cpp"] += 3 // 提高 C++ 优先级
	}

	// C#
	if csUsingPattern.MatchString(content) || csNamespacePattern.MatchString(content) {
		scores["csharp"] += 2
	}

	// PHP - 标签匹配优先级更高，因为 <?php 是 PHP 独有的标识
	if phpTagPattern.MatchString(content) {
		scores["php"] += 5 // 提高优先级
	}
	if phpVarPattern.MatchString(content) {
		scores["php"] += 2
	}

	// Ruby
	if rubyDefPattern.MatchString(content) || rubyClassPattern.MatchString(content) {
		scores["ruby"] += 2
	}

	// Kotlin
	if ktFunPattern.MatchString(content) {
		scores["kotlin"] += 2
	}
	if ktValPattern.MatchString(content) {
		scores["kotlin"] += 1
	}

	// HTML
	if htmlTagPattern.MatchString(content) || htmlDoctypePattern.MatchString(content) {
		scores["html"] += 2
	}

	// CSS
	if cssSelectorPattern.MatchString(content) && cssPropPattern.MatchString(content) {
		scores["css"] += 2
	}

	// JSON
	if jsonPattern.MatchString(content) {
		scores["json"] += 2
	}

	// YAML
	if yamlPattern.MatchString(content) {
		scores["yaml"] += 2
	}

	// SQL
	if sqlSelectPattern.MatchString(content) {
		scores["sql"] += 2
	}

	// Bash/Shell
	if bashShebangPattern.MatchString(content) || bashEchoPattern.MatchString(content) {
		scores["bash"] += 2
	}

	// Markdown
	if markdownHeaderPattern.MatchString(content) {
		scores["markdown"] += 2
	}
	if markdownListPattern.MatchString(content) {
		scores["markdown"] += 1
	}

	// Dockerfile
	if dockerfilePattern.MatchString(content) {
		scores["dockerfile"] += 2
	}

	// 找出最高分
	var maxLang string
	var maxScore int
	for lang, score := range scores {
		if score > maxScore {
			maxScore = score
			maxLang = lang
		}
	}

	if maxScore >= 1 {
		return maxLang
	}

	return "text"
}

// DetectContentType 检测内容类型
func DetectContentType(content string, language string) string {
	if content == "" {
		return "text"
	}

	if language == "markdown" {
		return "markdown"
	}

	markdownScore := 0
	for _, indicator := range markdownIndicators {
		if strings.Contains(content, indicator) {
			markdownScore++
		}
	}

	if markdownScore >= 2 {
		return "markdown"
	}

	if language != "text" && language != "" {
		return "code"
	}

	return "text"
}

// GetSupportedLanguages 返回支持的语言列表
func GetSupportedLanguages() []string {
	return []string{
		"javascript", "typescript", "python", "go", "rust", "java", "c", "cpp",
		"csharp", "php", "ruby", "swift", "kotlin", "scala", "html", "css",
		"scss", "json", "yaml", "xml", "sql", "bash", "shell", "powershell",
		"dockerfile", "markdown", "r", "matlab", "julia", "haskell", "elixir",
		"erlang", "clojure", "fsharp", "ocaml", "dart", "lua", "perl", "coffeescript",
		"vue", "react", "makefile", "cmake", "nginx", "apache", "gradle", "toml",
		"ini", "protobuf", "graphql", "terraform", "assembly", "vim", "latex",
		"sass", "less", "objectivec", "text",
	}
}
