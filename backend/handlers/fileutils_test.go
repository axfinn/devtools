package handlers

import (
	"testing"
)

func TestDetectFileType(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "JPEG",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46},
			expected: "image/jpeg",
		},
		{
			name:     "PNG",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expected: "image/png",
		},
		{
			name:     "GIF",
			data:     []byte{0x47, 0x49, 0x46, 0x38},
			expected: "image/gif",
		},
		{
			name:     "WebP",
			data:     []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
			expected: "image/webp",
		},
		{
			name:     "MP4",
			data:     []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, 0x69, 0x73, 0x6F, 0x6D},
			expected: "video/mp4",
		},
		{
			name:     "MOV",
			data:     []byte{0x00, 0x00, 0x00, 0x14, 0x66, 0x74, 0x79, 0x70, 0x71, 0x74, 0x20, 0x20},
			expected: "video/quicktime",
		},
		{
			name:     "WebM",
			data:     []byte{0x1A, 0x45, 0xDF, 0xA3},
			expected: "video/webm",
		},
		{
			name:     "AVI",
			data:     []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x41, 0x56, 0x49, 0x20},
			expected: "video/avi",
		},
		{
			name:     "MP3 with ID3",
			data:     []byte{0x49, 0x44, 0x33, 0x03, 0x00, 0x00, 0x00},
			expected: "audio/mpeg",
		},
		{
			name:     "MP3 without ID3",
			data:     []byte{0xFF, 0xFB, 0x90, 0x00},
			expected: "audio/mpeg",
		},
		{
			name:     "WAV",
			data:     []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x41, 0x56, 0x45},
			expected: "audio/wav",
		},
		{
			name:     "OGG",
			data:     []byte{0x4F, 0x67, 0x67, 0x53},
			expected: "audio/ogg",
		},
		{
			name:     "FLAC",
			data:     []byte{0x66, 0x4C, 0x61, 0x43},
			expected: "audio/flac",
		},
		{
			name:     "PDF",
			data:     []byte{0x25, 0x50, 0x44, 0x46},
			expected: "application/pdf",
		},
		{
			name:     "ZIP",
			data:     []byte{0x50, 0x4B, 0x03, 0x04},
			expected: "application/zip",
		},
		{
			name:     "RAR",
			data:     []byte{0x52, 0x61, 0x72, 0x21},
			expected: "application/x-rar-compressed",
		},
		{
			name:     "7z",
			data:     []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C},
			expected: "application/x-7z-compressed",
		},
		{
			name:     "Office 2007+ (DOCX/XLSX/PPTX)",
			data:     []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00},
			expected: "application/zip",
		},
		{
			name:     "Office 97-2003",
			data:     []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
			expected: "application/msoffice",
		},
		{
			name:     "unknown type",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: "",
		},
		{
			name:     "too short data",
			data:     []byte{0x00},
			expected: "",
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectFileType(tt.data)
			if result != tt.expected {
				t.Errorf("detectFileType(%v) = %q, want %q", tt.data, result, tt.expected)
			}
		})
	}
}

func TestGetExtFromMimeType(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"video/mp4", ".mp4"},
		{"video/quicktime", ".mov"},
		{"video/webm", ".webm"},
		{"video/avi", ".avi"},
		{"audio/mpeg", ".mp3"},
		{"audio/wav", ".wav"},
		{"audio/ogg", ".ogg"},
		{"audio/flac", ".flac"},
		{"audio/aac", ".aac"},
		{"application/pdf", ".pdf"},
		{"application/zip", ".zip"},
		{"application/x-rar-compressed", ".rar"},
		{"application/x-7z-compressed", ".7z"},
		{"application/vnd.openxmlformats", ".zip"},
		{"application/msoffice", ".doc"},
		{"unknown/type", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := getExtFromMimeType(tt.mimeType)
			if result != tt.expected {
				t.Errorf("getExtFromMimeType(%q) = %q, want %q", tt.mimeType, result, tt.expected)
			}
		})
	}
}

func TestGetFileCategory(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		{"image/jpeg", "image"},
		{"image/png", "image"},
		{"image/gif", "image"},
		{"image/svg+xml", "image"},
		{"image/bmp", "image"},
		{"image/tiff", "image"},
		{"video/mp4", "video"},
		{"video/quicktime", "video"},
		{"video/webm", "video"},
		{"video/x-matroska", "video"},
		{"video/x-flv", "video"},
		{"audio/mpeg", "audio"},
		{"audio/wav", "audio"},
		{"audio/ogg", "audio"},
		{"audio/flac", "audio"},
		{"audio/aac", "audio"},
		{"audio/midi", "audio"},
		{"application/pdf", "document"},
		{"application/zip", "archive"},
		{"application/x-rar-compressed", "archive"},
		{"application/x-7z-compressed", "archive"},
		{"application/x-tar", "archive"},
		{"application/gzip", "archive"},
		{"application/x-bzip2", "archive"},
		{"application/x-xz", "archive"},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "document"},
		{"application/msword", "document"},
		{"text/javascript", "code"},
		{"text/x-python", "code"},
		{"application/json", "code"},
		{"text/plain", "code"}, // 文本文件现在被识别为代码类型
		{"", "file"},
		{"unknown/type", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := getFileCategory(tt.mimeType)
			if result != tt.expected {
				t.Errorf("getFileCategory(%q) = %q, want %q", tt.mimeType, result, tt.expected)
			}
		})
	}
}

func TestDetectFileType_Extended(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "BMP",
			data:     []byte{0x42, 0x4D, 0x00, 0x00},
			expected: "image/bmp",
		},
		{
			name:     "ICO",
			data:     []byte{0x00, 0x00, 0x01, 0x00},
			expected: "image/x-icon",
		},
		{
			name:     "TIFF (little endian)",
			data:     []byte{0x49, 0x49, 0x2A, 0x00},
			expected: "image/tiff",
		},
		{
			name:     "TIFF (big endian)",
			data:     []byte{0x4D, 0x4D, 0x00, 0x2A},
			expected: "image/tiff",
		},
		{
			name:     "MIDI",
			data:     []byte{0x4D, 0x54, 0x68, 0x64},
			expected: "audio/midi",
		},
		{
			name:     "GZIP",
			data:     []byte{0x1F, 0x8B, 0x08, 0x00},
			expected: "application/gzip",
		},
		{
			name:     "BZ2",
			data:     []byte{0x42, 0x5A, 0x68, 0x00},
			expected: "application/x-bzip2",
		},
		{
			name:     "XZ",
			data:     []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00},
			expected: "application/x-xz",
		},
		{
			name:     "JSON",
			data:     []byte{0x7B, 0x22, 0x6B, 0x65, 0x79, 0x22, 0x3A, 0x22, 0x76, 0x61, 0x6C, 0x75, 0x65, 0x22, 0x7D},
			expected: "application/json",
		},
		{
			name:     "Plain text",
			data:     []byte("Hello, World!"),
			expected: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectFileType(tt.data)
			if result != tt.expected {
				t.Errorf("detectFileType(%q) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}
