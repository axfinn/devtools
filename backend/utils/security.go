package utils

import (
	"path/filepath"
	"regexp"
	"strings"
)

// SecurityScanner 安全扫描器
// 用于检测恶意内容和可疑链接

// 可疑 URL 模式
var suspiciousURLPatterns = []string{
	`javascript:`,
	`data:text/html`,
	`vbscript:`,
	`file://`,
	`\\`,
	`\x00`, // 空字节
}

// 恶意文件特征（简化检测，不依赖外部库）
var maliciousPatterns = []string{
	`\x00\x00\x00`,               // 空字节头部
	`PK\x03\x04.*\.exe`,         // 伪装成zip的可执行文件
	`MZ.*\.pif`,                  // PIF文件
	`\.scr\s`,                    // 屏保文件
	`\.bat\s`,                    // 批处理文件
	`\.cmd\s`,                    // CMD文件
	`<script[^>]*>`,              // 脚本标签
	`javascript:`,                // JS协议
	`onerror=`,                   // 事件处理器
	`onclick=`,                   // 事件处理器
	`eval\(`,                     // eval执行
	`document\.cookie`,           // Cookie访问
	`window\.location`,           // 位置操作
}

// 可疑文件扩展名白名单
var allowedExtensions = map[string]bool{
	// 图片
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".svg": true, ".bmp": true, ".ico": true, ".tiff": true, ".avif": true,
	// 视频
	".mp4": true, ".mov": true, ".webm": true, ".avi": true, ".mkv": true, ".flv": true,
	// 音频
	".mp3": true, ".wav": true, ".ogg": true, ".flac": true, ".aac": true, ".midi": true,
	// 文档
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".txt": true, ".md": true,
	// 代码/配置
	".js": true, ".ts": true, ".jsx": true, ".tsx": true, ".json": true,
	".html": true, ".css": true, ".xml": true, ".yaml": true, ".yml": true,
	".go": true, ".py": true, ".java": true, ".c": true, ".cpp": true,
	".h": true, ".hpp": true, ".cs": true, ".rb": true, ".php": true,
	".swift": true, ".kt": true, ".scala": true, ".rs": true,
	".sh": true, ".bash": true, ".sql": true,
	// 压缩包
	".zip": true, ".rar": true, ".7z": true, ".tar": true, ".gz": true,
	".bz2": true, ".xz": true,
	// 其他
	".log": true, ".env": true, ".gitignore": true,
}

// MaliciousPattern 恶意特征
type MaliciousPattern struct {
	Name    string
	Pattern *regexp.Regexp
}

// CompileMaliciousPatterns 编译恶意特征
func CompileMaliciousPatterns() []MaliciousPattern {
	patterns := make([]MaliciousPattern, 0, len(maliciousPatterns))
	for _, p := range maliciousPatterns {
		re := regexp.MustCompile(p)
		patterns = append(patterns, MaliciousPattern{
			Name:    p,
			Pattern: re,
		})
	}
	return patterns
}

// SecurityScanResult 扫描结果
type SecurityScanResult struct {
	IsSafe       bool     `json:"is_safe"`
	HasVirus     bool     `json:"has_virus"`
	HasSuspiciousURL bool `json:"has_suspicious_url"`
	Warnings     []string `json:"warnings"`
}

// ScanContent 扫描文本内容
func ScanContent(content string) *SecurityScanResult {
	if content == "" {
		return &SecurityScanResult{IsSafe: true}
	}

	result := &SecurityScanResult{
		IsSafe:   true,
		Warnings: []string{},
	}

	// 1. 检测可疑 URL
	for _, pattern := range suspiciousURLPatterns {
		if strings.Contains(strings.ToLower(content), pattern) {
			result.HasSuspiciousURL = true
			result.IsSafe = false
			result.Warnings = append(result.Warnings, "检测到可疑 URL 模式: "+pattern)
		}
	}

	// 2. 检测潜在的恶意代码模式
	compiledPatterns := CompileMaliciousPatterns()
	for _, mp := range compiledPatterns {
		if mp.Pattern.MatchString(content) {
			result.HasVirus = true
			result.IsSafe = false
			result.Warnings = append(result.Warnings, "检测到可疑文件模式: "+mp.Name)
		}
	}

	return result
}

// IsExecutableExtension 检查文件扩展名是否为可执行文件
func IsExecutableExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	execExtensions := []string{
		".exe", ".bat", ".cmd", ".com", ".pif", ".scr",
		".vbs", ".vbe", ".js", ".jse", ".wsf", ".wsh",
		".ps1", ".sh", ".bash", ".bin", ".app",
		".dmg", ".pkg", ".deb", ".rpm",
	}
	for _, e := range execExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

// ValidateFilename 验证文件名安全性
func ValidateFilename(filename string) (bool, string) {
	// 检查长度
	if len(filename) > 255 {
		return false, "文件名过长"
	}

	// 检查非法字符
	illegalChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	if illegalChars.MatchString(filename) {
		return false, "文件名包含非法字符"
	}

	// 检查危险扩展名
	if IsExecutableExtension(filename) {
		return false, "不允许上传可执行文件"
	}

	// 检查保留名称
	reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "LPT1"}
	baseName := strings.Split(filename, ".")[0]
	for _, reserved := range reservedNames {
		if strings.ToUpper(baseName) == reserved {
			return false, "文件名不能使用系统保留名称"
		}
	}

	return true, ""
}

// ContentTypeCategory 内容类型分类
type ContentTypeCategory string

const (
	CategoryImage    ContentTypeCategory = "image"
	CategoryVideo    ContentTypeCategory = "video"
	CategoryAudio    ContentTypeCategory = "audio"
	CategoryDocument ContentTypeCategory = "document"
	CategoryArchive  ContentTypeCategory = "archive"
	CategoryCode     ContentTypeCategory = "code"
	CategoryText     ContentTypeCategory = "text"
	CategoryUnknown  ContentTypeCategory = "unknown"
)

// GetCategoryByMimeType 根据 MIME 类型获取分类
func GetCategoryByMimeType(mimeType string) ContentTypeCategory {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return CategoryImage
	case strings.HasPrefix(mimeType, "video/"):
		return CategoryVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return CategoryAudio
	case mimeType == "application/pdf":
		return CategoryDocument
	case strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "rar") ||
		strings.Contains(mimeType, "7z") || strings.Contains(mimeType, "tar") ||
		strings.Contains(mimeType, "gz") || strings.Contains(mimeType, "bz2"):
		return CategoryArchive
	case strings.Contains(mimeType, "office") || strings.Contains(mimeType, "openxmlformats") ||
		strings.Contains(mimeType, "msword") || strings.Contains(mimeType, "ms-excel") ||
		strings.Contains(mimeType, "ms-powerpoint"):
		return CategoryDocument
	case isCodeMimeType(mimeType):
		return CategoryCode
	case strings.HasPrefix(mimeType, "text/"):
		return CategoryText
	default:
		return CategoryUnknown
	}
}

// isCodeMimeType 检查是否为代码 MIME 类型
func isCodeMimeType(mimeType string) bool {
	codeTypes := []string{
		"text/x-go", "text/x-python", "text/x-java", "text/x-c",
		"text/x-c++", "text/x-csharp", "text/x-ruby", "text/x-php",
		"text/x-swift", "text/x-kotlin", "text/x-scala", "text/x-rust",
		"text/javascript", "text/typescript", "text/html", "text/css",
		"text/x-sql", "text/x-shellscript", "text/x-yaml", "text/xml",
		"application/json", "application/javascript",
	}
	for _, t := range codeTypes {
		if mimeType == t || strings.Contains(mimeType, t) {
			return true
		}
	}
	return false
}

// IsAllowedExtension 检查文件扩展名是否允许上传
func IsAllowedExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return allowedExtensions[ext]
}

// ScanFileContent 扫描文件内容（用于已上传的文件）
func ScanFileContent(data []byte) *SecurityScanResult {
	result := &SecurityScanResult{
		IsSafe:   true,
		Warnings: []string{},
	}

	if len(data) == 0 {
		return result
	}

	// 检查空字节
	nullCount := 0
	for i := 0; i < min(len(data), 1000); i++ {
		if data[i] == 0 {
			nullCount++
		}
	}

	if nullCount > 10 {
		result.HasVirus = true
		result.IsSafe = false
		result.Warnings = append(result.Warnings, "检测到异常空字节，可能为二进制文件")
	}

	// 检查是否包含恶意内容特征
	content := string(data[:min(len(data), 10000)])
	for _, pattern := range suspiciousURLPatterns {
		if strings.Contains(content, pattern) {
			result.HasSuspiciousURL = true
			result.IsSafe = false
			result.Warnings = append(result.Warnings, "检测到可疑模式: "+pattern)
		}
	}

	return result
}

// ValidateFileSize 验证文件大小是否在允许范围内
func ValidateFileSize(size int64, maxSize int64) (bool, string) {
	if size <= 0 {
		return false, "文件大小无效"
	}
	if size > maxSize {
		return false, "文件大小超过限制"
	}
	return true, ""
}

// SanitizeFilenameForDownload 安全的下载文件名
func SanitizeFilenameForDownload(filename string) string {
	// 移除路径遍历字符
	filename = strings.ReplaceAll(filename, "..", "")
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")

	// 限制长度
	if len(filename) > 200 {
		filename = filename[:200]
	}

	return filename
}
