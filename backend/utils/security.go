package utils

import (
	"path/filepath"
	"regexp"
	"strings"
)

// MagicBytesSignatures 文件魔数（Magic Bytes）签名表
// 用于通过文件头部字节识别真实文件类型
// 注意：顺序很重要，需要先匹配更长的签名
var MagicBytesSignatures = []struct {
	fileType  string
	magic     []byte
	extraCheck func([]byte) bool // 可选的额外检查函数
}{
	// 图片格式 - 需要更长的签名以避免冲突
	{"avif", []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, 0x61, 0x76, 0x69, 0x66}, nil}, // ftypavif (12 bytes)
	{"webp", []byte{0x52, 0x49, 0x46, 0x46}, func(b []byte) bool { return len(b) >= 12 && string(b[8:12]) == "WEBP" }}, // RIFF....WEBP
	{"png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, nil}, // 8 bytes PNG
	{"tiff", []byte{0x49, 0x49, 0x2A, 0x00}, nil}, // Little endian TIFF
	{"tiff", []byte{0x4D, 0x4D, 0x00, 0x2A}, nil}, // Big endian TIFF
	{"gif", []byte{0x47, 0x49, 0x46, 0x38}, nil}, // GIF
	{"bmp", []byte{0x42, 0x4D}, nil}, // BM
	{"ico", []byte{0x00, 0x00, 0x01, 0x00}, nil},
	{"jpg", []byte{0xFF, 0xD8, 0xFF}, nil}, // JPEG
	{"jpeg", []byte{0xFF, 0xD8, 0xFF}, nil},

	// 视频格式
	{"webm", []byte{0x1A, 0x45, 0xDF, 0xA3}, nil}, // WebM
	{"mkv", []byte{0x1A, 0x45, 0xDF, 0xA3}, nil}, // Matroska (same as webm)

	// 音频格式
	{"ogg", []byte{0x4F, 0x67, 0x67, 0x53}, nil}, // OggS
	{"flac", []byte{0x66, 0x4C, 0x61, 0x43}, nil}, // fLaC
	{"wav", []byte{0x52, 0x49, 0x46, 0x46}, func(b []byte) bool { return len(b) >= 12 && string(b[8:12]) == "WAVE" }}, // RIFF....WAVE
	{"avi", []byte{0x52, 0x49, 0x46, 0x46}, func(b []byte) bool { return len(b) >= 12 && string(b[8:12]) == "AVI " }}, // RIFF....AVI
	{"mp3", []byte{0xFF, 0xFB}, nil}, // MP3
	{"mp3", []byte{0xFF, 0xF3}, nil}, // MP3
	{"mp3", []byte{0xFF, 0xF1}, nil}, // AAC
	{"aac", []byte{0xFF, 0xF1}, nil},

	// 文档格式
	{"pdf", []byte{0x25, 0x50, 0x44, 0x46}, nil}, // %PDF
	{"doc", []byte{0xD0, 0xCF, 0x11, 0xE0}, nil}, // OLE Compound Document
	{"xls", []byte{0xD0, 0xCF, 0x11, 0xE0}, nil},
	{"ppt", []byte{0xD0, 0xCF, 0x11, 0xE0}, nil},

	// Office Open XML (OOXML) - 实际是 ZIP 格式，需要更长的签名来区分
	{"docx", []byte{0x50, 0x4B, 0x03, 0x04}, func(b []byte) bool { return len(b) >= 30 }}, // ZIP with docx content
	{"xlsx", []byte{0x50, 0x4B, 0x03, 0x04}, func(b []byte) bool { return len(b) >= 30 }},
	{"pptx", []byte{0x50, 0x4B, 0x03, 0x04}, func(b []byte) bool { return len(b) >= 30 }},

	// 压缩包格式
	{"zip", []byte{0x50, 0x4B, 0x03, 0x04}, nil}, // ZIP
	{"rar", []byte{0x52, 0x61, 0x72, 0x21}, nil}, // Rar!
	{"7z", []byte{0x37, 0x7A, 0xBC, 0xAF}, nil}, // 7zbc
	{"tar", []byte{0x75, 0x73, 0x74, 0x61}, nil}, // ustar
	{"gz", []byte{0x1F, 0x8B}, nil}, // gzip
	{"bz2", []byte{0x42, 0x5A, 0x68}, nil}, // BZh
	{"xz", []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}, nil},

	// MP4 - 需要特殊处理，因为它可能以各种大小开头
	{"mp4", []byte{0x00, 0x00, 0x00}, nil},

	// 可执行文件 (危险) - 放在最后
	{"exe", []byte{0x4D, 0x5A}, nil}, // MZ
}

// 允许的文件扩展名对应的魔数类型
var extensionToMagicTypes = map[string][]string{
	".jpg":  {"jpg", "jpeg"},
	".jpeg": {"jpg", "jpeg"},
	".png":  {"png"},
	".gif":  {"gif"},
	".webp": {"webp"},
	".bmp":  {"bmp"},
	".ico":  {"ico"},
	".tiff": {"tiff"},
	".avif": {"avif"},
	".webm": {"webm"},
	".avi":  {"avi", "webm"},
	".mkv":  {"mkv", "webm"},
	".mp3":  {"mp3", "aac"},
	".wav":  {"wav"},
	".ogg":  {"ogg"},
	".flac": {"flac"},
	".aac":  {"aac", "mp3"},
	".pdf":  {"pdf"},
	".doc":  {"doc"},
	".xls":  {"xls"},
	".ppt":  {"ppt"},
	".docx": {"docx", "zip"},
	".xlsx": {"xlsx", "zip"},
	".pptx": {"pptx", "zip"},
	".zip":  {"zip", "docx", "xlsx", "pptx"},
	".rar":  {"rar"},
	".7z":   {"7z"},
	".tar":  {"tar"},
	".gz":   {"gz"},
	".bz2":  {"bz2"},
	".xz":   {"xz"},
	".exe":  {"exe"},
	".mp4":  {"mp4"},
	".m4a":  {"m4a", "mp4"},
}

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

// DetectFileTypeByMagicBytes 通过魔数检测文件真实类型
func DetectFileTypeByMagicBytes(data []byte) string {
	if len(data) < 4 {
		return "unknown"
	}

	// 检查每种文件类型的魔数
	for _, sig := range MagicBytesSignatures {
		if len(data) >= len(sig.magic) {
			match := true
			for i, b := range sig.magic {
				if data[i] != b {
					match = false
					break
				}
			}
			if match {
				// 如果有额外检查函数，执行它
				if sig.extraCheck != nil {
					if !sig.extraCheck(data) {
						continue
					}
				}
				return sig.fileType
			}
		}
	}

	// 特殊处理 MP4 (需要更多字节)
	if len(data) >= 8 {
		// MP4: ftyp box
		if data[4] == 0x66 && data[5] == 0x74 && data[6] == 0x79 && data[7] == 0x70 {
			return "mp4"
		}
		// M4A: ftypM4A
		if data[4] == 0x66 && data[5] == 0x74 && data[6] == 0x79 && data[7] == 0x4D {
			return "m4a"
		}
	}

	return "unknown"
}

// ValidateMagicBytes 验证文件的魔数是否与扩展名匹配
func ValidateMagicBytes(filename string, data []byte) (bool, string) {
	ext := strings.ToLower(filepath.Ext(filename))

	// 获取该扩展名允许的魔数类型
	allowedTypes, exists := extensionToMagicTypes[ext]
	if !exists {
		// 不在白名单中的扩展名，拒绝上传（安全考虑）
		return false, "不支持的文件扩展名: " + ext
	}

	// 如果文件太小，无法验证
	if len(data) < 4 {
		// 对于小文件，只检查扩展名白名单
		return true, ""
	}

	// 检测文件真实类型
	detectedType := DetectFileTypeByMagicBytes(data)

	// 检查检测到的类型是否在允许列表中
	for _, allowed := range allowedTypes {
		if detectedType == allowed {
			return true, ""
		}
	}

	// 特殊处理：ZIP 格式可以包含多种 OOXML 文件
	if detectedType == "zip" {
		for _, allowed := range allowedTypes {
			if allowed == "zip" || allowed == "docx" || allowed == "xlsx" || allowed == "pptx" {
				return true, ""
			}
		}
	}

	// 危险检测：可执行文件伪装成其他文件
	if detectedType == "exe" {
		return false, "检测到可执行文件伪装"
	}

	return false, "文件类型不匹配：扩展名为 " + ext + "，但文件魔数检测为 " + detectedType
}

// ScanFileWithMagicBytes 使用魔数增强的文件扫描
func ScanFileWithMagicBytes(filename string, data []byte) *SecurityScanResult {
	result := ScanFileContent(data)

	// 验证魔数
	isValid, msg := ValidateMagicBytes(filename, data)
	if !isValid {
		result.HasVirus = true
		result.IsSafe = false
		result.Warnings = append(result.Warnings, "魔数检测失败: "+msg)
	}

	return result
}
