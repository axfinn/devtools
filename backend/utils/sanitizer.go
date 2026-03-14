package utils

import (
	"html"
	"html/template"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// ContentSanitizer 内容消毒器
// 用于防止XSS攻击和其他安全问题

// 定义危险模式
var (
	scriptPattern     = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	stylePattern      = regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	eventHandlerRegex = regexp.MustCompile(`(?i)\s+on\w+\s*=`)

	// 可能被滥用的HTML标签
	dangerousTags = regexp.MustCompile(`(?i)<(iframe|object|embed|form|input|button|meta|link|base|applet)\b`)
)

// SanitizeContent 对文本内容进行XSS防护
func SanitizeContent(content string) string {
	if content == "" {
		return ""
	}

	// 1. 先转义HTML实体
	content = html.EscapeString(content)

	// 2. 移除script标签
	content = scriptPattern.ReplaceAllString(content, "")

	// 3. 移除style标签
	content = stylePattern.ReplaceAllString(content, "")

	// 4. 移除事件处理器属性
	content = eventHandlerRegex.ReplaceAllString(content, "")

	// 5. 移除危险标签
	content = dangerousTags.ReplaceAllString(content, "")

	// 6. 限制内容长度
	maxLength := 100 * 1024 // 100KB
	if len(content) > maxLength {
		content = content[:maxLength]
	}

	return content
}

// SanitizeHTML 使用bluemonday进行更安全的HTML清理
func SanitizeHTML(content string) string {
	if content == "" {
		return ""
	}

	// 使用严格的策略
	p := bluemonday.StrictPolicy()

	// 允许特定的标签和属性
	p.AllowElements("p", "br", "b", "i", "u", "em", "strong", "code", "pre", "blockquote", "ul", "ol", "li", "h1", "h2", "h3", "h4", "h5", "h6", "a", "img", "table", "thead", "tbody", "tr", "th", "td")

	// 允许特定的属性
	p.AllowAttrs("href", "alt", "title", "class").OnElements("a", "img")
	p.AllowAttrs("class").OnElements("code", "pre", "p", "div", "span")

	return p.Sanitize(content)
}

// SanitizeForAttribute 转义用于HTML属性值的内容
func SanitizeForAttribute(content string) string {
	return html.EscapeString(content)
}

// ValidateContentLength 验证内容长度
func ValidateContentLength(content string, maxLength int) bool {
	return len(content) <= maxLength
}

// DetectPotentialXSS 检测潜在的XSS尝试
func DetectPotentialXSS(content string) bool {
	// 检查常见的XSS尝试模式
	xssPatterns := []string{
		`(?i)javascript:`,
		`(?i)onerror\s*=`,
		`(?i)onload\s*=`,
		`(?i)onclick\s*=`,
		`(?i)onmouseover\s*=`,
		`<script`,
		`</script`,
		`(?i)eval\s*\(`,
		`(?i)expression\s*\(`,
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range xssPatterns {
		if matched, _ := regexp.MatchString(pattern, contentLower); matched {
			return true
		}
	}

	return false
}

// SanitizeFilename 清理文件名
func SanitizeFilename(filename string) string {
	// 移除非法的文件系统字符
	// Windows: < > : " / \ | ? *
	// Unix: /
	illegalChars := regexp.MustCompile(`[<>:"/\\|?*]`)

	filename = illegalChars.ReplaceAllString(filename, "_")

	// 移除控制字符
	controlChars := regexp.MustCompile(`[\x00-\x1f]`)
	filename = controlChars.ReplaceAllString(filename, "")

	// 限制长度
	if len(filename) > 255 {
		ext := ""
		if idx := strings.LastIndex(filename, "."); idx > 0 {
			ext = filename[idx:]
			filename = filename[:255-len(ext)]
			filename = filename + ext
		} else {
			filename = filename[:255]
		}
	}

	return filename
}

// IsAllowedFileType 检查文件类型是否允许上传
func IsAllowedFileType(mimeType string, allowedTypes []string) bool {
	if len(allowedTypes) == 0 {
		// 默认允许的类型
		allowedTypes = []string{
			"image/jpeg",
			"image/png",
			"image/gif",
			"image/webp",
			"video/mp4",
			"video/quicktime",
			"video/webm",
			"video/avi",
			"audio/mpeg",
			"audio/wav",
			"audio/ogg",
			"audio/flac",
			"audio/aac",
			"application/pdf",
			"application/zip",
			"application/x-rar-compressed",
			"application/x-7z-compressed",
		}
	}

	for _, allowed := range allowedTypes {
		if allowed == mimeType {
			return true
		}
	}

	return false
}

// SafeHTML 创建一个安全的HTML模板
func SafeHTML(html string) template.HTML {
	return template.HTML(SanitizeHTML(html))
}
