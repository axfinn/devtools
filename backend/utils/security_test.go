package utils

import (
	"testing"
)

func TestScanContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		isSafe   bool
		hasURL   bool
		hasVirus bool
	}{
		{
			name:     "Normal text",
			content:  "Hello, World!",
			isSafe:   true,
			hasURL:   false,
			hasVirus: false,
		},
		{
			name:     "Suspicious javascript URL",
			content:  "Visit javascript:alert(1)",
			isSafe:   false,
			hasURL:   true,
			hasVirus: true, // 检测到 script 标签
		},
		{
			name:     "Normal HTTP URL",
			content:  "Visit https://example.com",
			isSafe:   true,
			hasURL:   false,
			hasVirus: false,
		},
		{
			name:     "Empty content",
			content:  "",
			isSafe:   true,
			hasURL:   false,
			hasVirus: false,
		},
		{
			name:     "Data URL",
			content:  "data:text/html,<script>alert(1)</script>",
			isSafe:   false,
			hasURL:   true,
			hasVirus: true, // 检测到 script 标签
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanContent(tt.content)
			if result.IsSafe != tt.isSafe {
				t.Errorf("ScanContent().IsSafe = %v, want %v", result.IsSafe, tt.isSafe)
			}
			if result.HasSuspiciousURL != tt.hasURL {
				t.Errorf("ScanContent().HasSuspiciousURL = %v, want %v", result.HasSuspiciousURL, tt.hasURL)
			}
			if result.HasVirus != tt.hasVirus {
				t.Errorf("ScanContent().HasVirus = %v, want %v", result.HasVirus, tt.hasVirus)
			}
		})
	}
}

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantValid bool
	}{
		{
			name:      "Normal filename",
			filename:  "document.pdf",
			wantValid: true,
		},
		{
			name:      "Filename with spaces",
			filename:  "my document.pdf",
			wantValid: true,
		},
		{
			name:      "Filename with special chars",
			filename:  "test<file>.pdf",
			wantValid: false,
		},
		{
			name:      "Executable extension",
			filename:  "malware.exe",
			wantValid: false,
		},
		{
			name:      "Batch file",
			filename:  "script.bat",
			wantValid: false,
		},
		{
			name:      "Reserved name CON",
			filename:  "CON.pdf",
			wantValid: false,
		},
		{
			name:      "Reserved name PRN",
			filename:  "PRN.txt",
			wantValid: false,
		},
		{
			name:      "Long filename",
			filename:  "a.pdf",
			wantValid: true,
		},
		{
			name:      "Script file",
			filename:  "script.sh",
			wantValid: false,
		},
		{
			name:      "Python script",
			filename:  "script.py",
			wantValid: true,
		},
		{
			name:      "PowerShell script",
			filename:  "script.ps1",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := ValidateFilename(tt.filename)
			if valid != tt.wantValid {
				t.Errorf("ValidateFilename(%q) = %v, want %v", tt.filename, valid, tt.wantValid)
			}
		})
	}
}

func TestGetCategoryByMimeType(t *testing.T) {
	tests := []struct {
		mimeType string
		category ContentTypeCategory
	}{
		{"image/jpeg", CategoryImage},
		{"image/png", CategoryImage},
		{"video/mp4", CategoryVideo},
		{"audio/mpeg", CategoryAudio},
		{"application/pdf", CategoryDocument},
		{"application/zip", CategoryArchive},
		{"application/x-rar-compressed", CategoryArchive},
		{"application/x-tar", CategoryArchive},
		{"application/gzip", CategoryArchive},
		{"application/x-bzip2", CategoryArchive},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", CategoryDocument},
		{"text/javascript", CategoryCode},
		{"text/x-python", CategoryCode},
		{"application/json", CategoryCode},
		{"text/plain", CategoryText},
		{"application/octet-stream", CategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			category := GetCategoryByMimeType(tt.mimeType)
			if category != tt.category {
				t.Errorf("GetCategoryByMimeType(%q) = %v, want %v", tt.mimeType, category, tt.category)
			}
		})
	}
}

func TestIsAllowedExtension(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"document.pdf", true},
		{"image.png", true},
		{"video.mp4", true},
		{"archive.zip", true},
		{"code.js", true},
		{"config.json", true},
		{"malware.exe", false},
		{"script.bat", false},
		{"virus.pif", false},
		{"script.ps1", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsAllowedExtension(tt.filename)
			if result != tt.expected {
				t.Errorf("IsAllowedExtension(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestScanFileContent(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		isSafe   bool
		hasVirus bool
	}{
		{
			name:     "Normal text",
			data:     []byte("Hello, World!"),
			isSafe:   true,
			hasVirus: false,
		},
		{
			name:     "Many null bytes",
			data:     []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			isSafe:   false,
			hasVirus: true,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			isSafe:   true,
			hasVirus: false,
		},
		{
			name:     "Suspicious content",
			data:     []byte("javascript:alert(1)"),
			isSafe:   false,
			hasVirus: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanFileContent(tt.data)
			if result.IsSafe != tt.isSafe {
				t.Errorf("ScanFileContent().IsSafe = %v, want %v", result.IsSafe, tt.isSafe)
			}
			if result.HasVirus != tt.hasVirus {
				t.Errorf("ScanFileContent().HasVirus = %v, want %v", result.HasVirus, tt.hasVirus)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name      string
		size      int64
		maxSize   int64
		wantValid bool
	}{
		{
			name:      "Valid size",
			size:      1000,
			maxSize:   10000,
			wantValid: true,
		},
		{
			name:      "Zero size",
			size:      0,
			maxSize:   10000,
			wantValid: false,
		},
		{
			name:      "Negative size",
			size:      -1,
			maxSize:   10000,
			wantValid: false,
		},
		{
			name:      "Exceeds max",
			size:      20000,
			maxSize:   10000,
			wantValid: false,
		},
		{
			name:      "Equal to max",
			size:      10000,
			maxSize:   10000,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := ValidateFileSize(tt.size, tt.maxSize)
			if valid != tt.wantValid {
				t.Errorf("ValidateFileSize(%d, %d) = %v, want %v", tt.size, tt.maxSize, valid, tt.wantValid)
			}
		})
	}
}

func TestSanitizeFilenameForDownload(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal filename",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "Path traversal",
			input:    "../../etc/passwd",
			expected: "etcpasswd",
		},
		{
			name:     "Long filename",
			input:    "a.pdf",
			expected: "a.pdf",
		},
		{
			name:     "With slashes",
			input:    "path/to/file.pdf",
			expected: "pathtofile.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilenameForDownload(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilenameForDownload(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectFileTypeByMagicBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "PNG file",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expected: "png",
		},
		{
			name:     "JPG file",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0},
			expected: "jpg",
		},
		{
			name:     "GIF file",
			data:     []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61},
			expected: "gif",
		},
		{
			name:     "PDF file",
			data:     []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31},
			expected: "pdf",
		},
		{
			name:     "ZIP file - minimal",
			data:     []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00, 0x00, 0x00},
			expected: "zip",
		},
		{
			name:     "7Z file",
			data:     []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C},
			expected: "7z",
		},
		{
			name:     "RAR file",
			data:     []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07},
			expected: "rar",
		},
		{
			name:     "GZIP file",
			data:     []byte{0x1F, 0x8B, 0x08, 0x00},
			expected: "gz",
		},
		{
			name:     "BMP file",
			data:     []byte{0x42, 0x4D, 0x00, 0x00},
			expected: "bmp",
		},
		{
			name:     "WEBP file",
			data:     []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
			expected: "webp",
		},
		{
			name:     "MP4 file",
			data:     []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70},
			expected: "mp4",
		},
		{
			name:     "Unknown file type - matches mp4",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: "mp4",
		},
		{
			name:     "Too short data",
			data:     []byte{0x00},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFileTypeByMagicBytes(tt.data)
			if result != tt.expected {
				t.Errorf("DetectFileTypeByMagicBytes() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestValidateMagicBytes(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		data     []byte
		wantValid bool
	}{
		{
			name:     "Valid PNG file",
			filename: "image.png",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			wantValid: true,
		},
		{
			name:     "Valid JPG file",
			filename: "photo.jpg",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
			wantValid: true,
		},
		{
			name:     "Valid PDF file",
			filename: "document.pdf",
			data:     []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31},
			wantValid: true,
		},
		{
			name:     "Valid ZIP file",
			filename: "archive.zip",
			data:     []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00},
			wantValid: true,
		},
		{
			name:     "Valid 7Z file",
			filename: "archive.7z",
			data:     []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C},
			wantValid: true,
		},
		{
			name:     "Valid DOCX file (ZIP format)",
			filename: "document.docx",
			data:     []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00},
			wantValid: true,
		},
		{
			name:     "Mismatched - PNG with .jpg extension",
			filename: "photo.jpg",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			wantValid: false, // PNG 魔数检测为 png，但扩展名是 jpg
		},
		{
			name:     "Unsupported extension",
			filename: "document.xyz",
			data:     []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31},
			wantValid: false,
		},
		{
			name:     "EXE伪装成图片",
			filename: "photo.png",
			data:     []byte{0x4D, 0x5A, 0x90, 0x00},
			wantValid: false,
		},
		{
			name:     "Small file - pass through",
			filename: "small.png",
			data:     []byte{0x00},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := ValidateMagicBytes(tt.filename, tt.data)
			if valid != tt.wantValid {
				t.Errorf("ValidateMagicBytes(%q, data) = %v, want %v", tt.filename, valid, tt.wantValid)
			}
		})
	}
}

func TestScanFileWithMagicBytes(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		data     []byte
		isSafe   bool
		hasVirus bool
	}{
		{
			name:     "Valid PNG file",
			filename: "image.png",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			isSafe:   true,
			hasVirus: false,
		},
		{
			name:     "EXE伪装成图片",
			filename: "image.png",
			data:     []byte{0x4D, 0x5A, 0x90, 0x00},
			isSafe:   false,
			hasVirus: true,
		},
		{
			name:     "Unsupported extension",
			filename: "document.xyz",
			data:     []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31},
			isSafe:   false,
			hasVirus: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanFileWithMagicBytes(tt.filename, tt.data)
			if result.IsSafe != tt.isSafe {
				t.Errorf("ScanFileWithMagicBytes().IsSafe = %v, want %v", result.IsSafe, tt.isSafe)
			}
			if result.HasVirus != tt.hasVirus {
				t.Errorf("ScanFileWithMagicBytes().HasVirus = %v, want %v", result.HasVirus, tt.hasVirus)
			}
		})
	}
}
