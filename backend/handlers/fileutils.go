package handlers

import "strings"

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
	// PDF
	if data[0] == 0x25 && data[1] == 0x50 && data[2] == 0x44 && data[3] == 0x46 {
		return "application/pdf"
	}
	// ZIP
	if data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04 {
		return "application/zip"
	}
	// RAR
	if data[0] == 0x52 && data[1] == 0x61 && data[2] == 0x72 && data[3] == 0x21 {
		return "application/x-rar-compressed"
	}
	// 7z
	if len(data) >= 6 && data[0] == 0x37 && data[1] == 0x7A && data[2] == 0xBC && data[3] == 0xAF && data[4] == 0x27 && data[5] == 0x1C {
		return "application/x-7z-compressed"
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
	case strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "rar") || strings.Contains(mimeType, "7z"):
		return "archive"
	case strings.Contains(mimeType, "office") || strings.Contains(mimeType, "openxmlformats"):
		return "document"
	default:
		return "file"
	}
}
