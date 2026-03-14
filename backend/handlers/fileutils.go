package handlers

import (
	"strings"
)

// detectFileType 检测文件真实类型
func detectFileType(data []byte) string {
	if len(data) < 4 {
		return ""
	}

	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}
	// GIF
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return "image/gif"
	}
	// WebP (RIFF....WEBP)
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp"
	}
	// AVIF
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x41 && data[9] == 0x56 && data[10] == 0x49 && data[11] == 0x46 {
		return "image/avif"
	}
	// SVG (文本格式)
	if len(data) >= 5 && data[0] == 0x3C && data[1] == 0x73 && data[2] == 0x76 && data[3] == 0x67 {
		return "image/svg+xml"
	}
	// BMP
	if data[0] == 0x42 && data[1] == 0x4D {
		return "image/bmp"
	}
	// ICO
	if data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && data[3] == 0x00 {
		return "image/x-icon"
	}
	// TIFF
	if (data[0] == 0x49 && data[1] == 0x49 && data[2] == 0x2A && data[3] == 0x00) ||
		(data[0] == 0x4D && data[1] == 0x4D && data[2] == 0x00 && data[3] == 0x2A) {
		return "image/tiff"
	}
	// MP4/MOV (ftyp)
	if len(data) >= 12 && data[4] == 0x66 && data[5] == 0x74 && data[6] == 0x79 && data[7] == 0x70 {
		// 检查brand type来区分MP4和MOV
		// MOV常见brand: qt__, M4V, M4A
		// MP4常见brand: isom, mp41, mp42, avc1
		if len(data) >= 12 {
			brand := string(data[8:12])
			if strings.HasPrefix(brand, "qt") || brand == "M4V " || brand == "M4A " {
				return "video/quicktime"
			}
		}
		return "video/mp4"
	}
	// WebM/MKV
	if data[0] == 0x1A && data[1] == 0x45 && data[2] == 0xDF && data[3] == 0xA3 {
		return "video/webm"
	}
	// AVI
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x41 && data[9] == 0x56 && data[10] == 0x49 && data[11] == 0x20 {
		return "video/avi"
	}
	// MKV (Matroska)
	if len(data) >= 4 && data[0] == 0x1A && data[1] == 0x45 && data[2] == 0xDF && data[3] == 0xA3 {
		// 进一步区分 WebM 和 MKV
		if len(data) >= 12 {
			docType := string(data[4:8])
			if docType == "\x00\x00\x00\x00" || docType == "webm" {
				return "video/webm"
			}
		}
		return "video/x-matroska"
	}
	// FLV
	if data[0] == 0x46 && data[1] == 0x4C && data[2] == 0x56 && data[3] == 0x01 {
		return "video/x-flv"
	}
	// MP3 (ID3 or sync)
	if (data[0] == 0x49 && data[1] == 0x44 && data[2] == 0x33) || // ID3
		(data[0] == 0xFF && (data[1]&0xE0) == 0xE0) { // sync
		return "audio/mpeg"
	}
	// WAV (RIFF....WAVE)
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x41 && data[10] == 0x56 && data[11] == 0x45 {
		return "audio/wav"
	}
	// OGG
	if data[0] == 0x4F && data[1] == 0x67 && data[2] == 0x67 && data[3] == 0x53 {
		return "audio/ogg"
	}
	// FLAC
	if data[0] == 0x66 && data[1] == 0x4C && data[2] == 0x61 && data[3] == 0x43 {
		return "audio/flac"
	}
	// AAC (ADTS header)
	if len(data) >= 2 && data[0] == 0xFF && (data[1]&0xF0) == 0xF0 {
		return "audio/aac"
	}
	// MIDI
	if data[0] == 0x4D && data[1] == 0x54 && data[2] == 0x68 && data[3] == 0x64 {
		return "audio/midi"
	}
	// WebM audio (WEBM is based on Matroska)
	if len(data) >= 4 && data[0] == 0x1A && data[1] == 0x45 && data[2] == 0xDF && data[3] == 0xA3 {
		// Check for WebM specific EBML ID
		if len(data) >= 12 && string(data[4:8]) == "\x00\x00\x00\x00" {
			// This is WebM format, could be video or audio
		}
	}
	// PDF
	if data[0] == 0x25 && data[1] == 0x50 && data[2] == 0x44 && data[3] == 0x46 {
		return "application/pdf"
	}
	// ZIP
	if data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04 {
		return "application/zip"
	}
	// ZIP (empty archive)
	if data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x05 && data[3] == 0x06 {
		return "application/zip"
	}
	// ZIP (spanned)
	if data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x07 && data[3] == 0x08 {
		return "application/zip"
	}
	// RAR
	if data[0] == 0x52 && data[1] == 0x61 && data[2] == 0x72 && data[3] == 0x21 {
		return "application/x-rar-compressed"
	}
	// RAR5
	if data[0] == 0x52 && data[1] == 0x61 && data[2] == 0x72 && data[3] == 0x1A && len(data) > 5 && data[5] == 0x07 {
		return "application/x-rar-compressed"
	}
	// 7z
	if len(data) >= 6 && data[0] == 0x37 && data[1] == 0x7A && data[2] == 0xBC && data[3] == 0xAF && data[4] == 0x27 && data[5] == 0x1C {
		return "application/x-7z-compressed"
	}
	// TAR (POSIX ustar)
	if len(data) >= 512 {
		if (data[257] == 0x75 && data[258] == 0x73 && data[259] == 0x74 && data[260] == 0x61 && data[261] == 0x72) ||
			(data[257] == 0x00 && data[258] == 0x00 && data[259] == 0x00) {
			return "application/x-tar"
		}
	}
	// GZIP
	if data[0] == 0x1F && data[1] == 0x8B {
		return "application/gzip"
	}
	// BZ2
	if data[0] == 0x42 && data[1] == 0x5A && data[2] == 0x68 {
		return "application/x-bzip2"
	}
	// XZ
	if data[0] == 0xFD && data[1] == 0x37 && data[2] == 0x7A && data[3] == 0x58 && data[4] == 0x5A && data[5] == 0x00 {
		return "application/x-xz"
	}
	// Microsoft Office 2007+ (DOCX, XLSX, PPTX - 都是ZIP格式)
	if data[0] == 0x50 && data[1] == 0x4B && len(data) >= 30 {
		// 检查是否包含 [Content_Types].xml 标记
		return "application/vnd.openxmlformats"
	}
	// Microsoft Office 97-2003 (DOC, XLS, PPT)
	if len(data) >= 8 && data[0] == 0xD0 && data[1] == 0xCF && data[2] == 0x11 && data[3] == 0xE0 &&
		data[4] == 0xA1 && data[5] == 0xB1 && data[6] == 0x1A && data[7] == 0xE1 {
		return "application/msoffice"
	}

	// 检查文本/代码类型（通过文件内容检测）
	// 检查是否为文本文件
	if isTextFile(data) {
		// 可能是代码或文本文件
		return detectTextFileType(data)
	}

	return ""
}

// getExtFromMimeType 根据 MIME 类型获取扩展名
func getExtFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/quicktime":
		return ".mov"
	case "video/webm":
		return ".webm"
	case "video/avi":
		return ".avi"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav":
		return ".wav"
	case "audio/ogg":
		return ".ogg"
	case "audio/flac":
		return ".flac"
	case "audio/aac":
		return ".aac"
	case "audio/webm":
		return ".webm"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	case "application/x-rar-compressed":
		return ".rar"
	case "application/x-7z-compressed":
		return ".7z"
	case "application/vnd.openxmlformats":
		return ".zip"
	case "application/msoffice":
		return ".doc"
	default:
		return ""
	}
}

// getFileCategory 获取文件分类
func getFileCategory(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case mimeType == "application/pdf":
		return "document"
	case mimeType == "application/x-tar" || mimeType == "application/gzip" ||
		mimeType == "application/x-bzip2" || mimeType == "application/x-xz" ||
		mimeType == "application/x-rar-compressed" || mimeType == "application/x-7z-compressed" ||
		strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "rar") ||
		strings.Contains(mimeType, "7z") || strings.Contains(mimeType, "tar") ||
		strings.Contains(mimeType, "gz") || strings.Contains(mimeType, "bz2"):
		return "archive"
	case strings.Contains(mimeType, "office") || strings.Contains(mimeType, "openxmlformats") ||
		strings.Contains(mimeType, "msword") || strings.Contains(mimeType, "ms-excel") ||
		strings.Contains(mimeType, "ms-powerpoint"):
		return "document"
	case strings.Contains(mimeType, "text/x-") || strings.Contains(mimeType, "text/") ||
		mimeType == "application/json" || mimeType == "application/xml" ||
		mimeType == "application/javascript":
		return "code"
	default:
		return "file"
	}
}

// isTextFile 检查是否为文本文件
func isTextFile(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// 统计可打印字符和空白字符的比例
	printable := 0
	nullCount := 0

	for i := 0; i < min(len(data), 1024); i++ {
		if data[i] == 0 {
			nullCount++
			continue
		}
		// 可打印ASCII字符或Tab、换行、回车
		if (data[i] >= 32 && data[i] <= 126) || data[i] == 9 || data[i] == 10 || data[i] == 13 {
			printable++
		}
	}

	// 如果有超过2个null字节，很可能是二进制文件
	if nullCount > 2 {
		return false
	}

	// 如果可打印字符超过80%，认为是文本文件
	return float64(printable)/float64(min(len(data), 1024)) > 0.8
}

// detectTextFileType 检测文本文件类型（代码或配置）
func detectTextFileType(data []byte) string {
	content := string(data)

	// JSON
	if (len(data) > 2 && data[0] == 0x7B && data[len(data)-1] == 0x7D) ||
		(len(data) > 2 && data[0] == 0x5B && data[len(data)-1] == 0x5D) {
		return "application/json"
	}

	// XML
	if len(data) >= 5 && (string(data[:5]) == "<?xml" || string(data[:5]) == "<!DOC" || (data[0] == 0x3C && data[1] != 0x21)) {
		return "application/xml"
	}

	// HTML
	if len(data) >= 6 && (strings.Contains(content[:min(200, len(content))], "<!DOCTYPE") ||
		strings.Contains(content[:min(200, len(content))], "<html") ||
		strings.Contains(content[:min(200, len(content))], "<head")) {
		return "text/html"
	}

	// Markdown
	if len(data) >= 4 && strings.Contains(content[:min(100, len(content))], "#") {
		return "text/markdown"
	}

	// YAML
	if strings.Contains(content, "---") || strings.Contains(content[:min(100, len(content))], ": ") {
		return "text/x-yaml"
	}

	// Shell脚本
	if len(data) >= 2 && string(data[:2]) == "#!" {
		if strings.Contains(content, "/bash") || strings.Contains(content, "/sh") {
			return "text/x-shellscript"
		}
		if strings.Contains(content, "/python") {
			return "text/x-python"
		}
		if strings.Contains(content, "/node") {
			return "text/javascript"
		}
	}

	// 纯文本
	return "text/plain"
}
