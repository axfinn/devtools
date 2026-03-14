package utils

import (
	"testing"
)

func TestSanitizeContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "script tag removal",
			input:    "Hello <script>alert('xss')</script> World",
			expected: "Hello &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt; World",
		},
		{
			name:     "style tag removal",
			input:    "Hello <style>body{color:red}</style> World",
			expected: "Hello &lt;style&gt;body{color:red}&lt;/style&gt; World",
		},
		{
			name:     "event handler removal",
			input:    "Hello onclick=alert(1) World",
			expected: "Helloalert(1) World",
		},
		{
			name:     "HTML entity encoding",
			input:    "Hello <script>&lt;script&gt;</script>",
			expected: "Hello &lt;script&gt;&amp;lt;script&amp;gt;&lt;/script&gt;",
		},
		{
			name:     "iframe removal",
			input:    "Hello <iframe src='evil.com'></iframe> World",
			expected: "Hello &lt;iframe src=&#39;evil.com&#39;&gt;&lt;/iframe&gt; World",
		},
		{
			name:     "content length limit",
			input:    string(make([]byte, 150*1024)),
			expected: string(make([]byte, 100*1024)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeContent(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetectPotentialXSS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "plain text",
			input:    "Hello, World!",
			expected: false,
		},
		{
			name:     "script tag",
			input:    "<script>alert('xss')</script>",
			expected: true,
		},
		{
			name:     "javascript protocol",
			input:    "javascript:alert(1)",
			expected: true,
		},
		{
			name:     "onerror handler",
			input:    "<img src=x onerror=alert(1)>",
			expected: true,
		},
		{
			name:     "onload handler",
			input:    "<img src=x onload=alert(1)>",
			expected: true,
		},
		{
			name:     "onclick handler",
			input:    "<button onclick=alert(1)>Click</button>",
			expected: true,
		},
		{
			name:     "onmouseover handler",
			input:    "<div onmouseover=alert(1)>Hover</div>",
			expected: true,
		},
		{
			name:     "eval usage",
			input:    "eval(document.cookie)",
			expected: true,
		},
		{
			name:     "expression usage",
			input:    "expression(alert(1))",
			expected: true,
		},
		{
			name:     "case insensitive script",
			input:    "<SCRIPT>alert(1)</SCRIPT>",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectPotentialXSS(tt.input)
			if result != tt.expected {
				t.Errorf("DetectPotentialXSS(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "filename with spaces",
			input:    "my document.pdf",
			expected: "my document.pdf",
		},
		{
			name:     "filename with illegal chars",
			input:    "file<name>.pdf",
			expected: "file_name_.pdf",
		},
		{
			name:     "filename with path separators",
			input:    "../../../etc/passwd",
			expected: ".._.._.._etc_passwd",
		},
		{
			name:     "filename with colon",
			input:    "file:name.pdf",
			expected: "file_name.pdf",
		},
		{
			name:     "filename with pipe",
			input:    "file|name.pdf",
			expected: "file_name.pdf",
		},
		{
			name:     "filename with question mark",
			input:    "file?.pdf",
			expected: "file_.pdf",
		},
		{
			name:     "filename with asterisk",
			input:    "file*.pdf",
			expected: "file_.pdf",
		},
		{
			name:     "control characters",
			input:    "file\x00\x01.pdf",
			expected: "file.pdf",
		},
		{
			name:     "long filename truncation",
			input:    string(make([]byte, 300)) + ".pdf",
			expected: ".pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateContentLength(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		maxLength int
		expected  bool
	}{
		{
			name:      "valid length",
			content:   "Hello",
			maxLength: 10,
			expected:  true,
		},
		{
			name:      "exact length",
			content:   "Hello",
			maxLength: 5,
			expected:  true,
		},
		{
			name:      "exceeds length",
			content:   "Hello World",
			maxLength: 5,
			expected:  false,
		},
		{
			name:      "empty content",
			content:   "",
			maxLength: 5,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateContentLength(tt.content, tt.maxLength)
			if result != tt.expected {
				t.Errorf("ValidateContentLength(%q, %d) = %v, want %v", tt.content, tt.maxLength, result, tt.expected)
			}
		})
	}
}

func TestSanitizeForAttribute(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello",
			expected: "Hello",
		},
		{
			name:     "HTML entities",
			input:    "<script>alert(1)</script>",
			expected: "&lt;script&gt;alert(1)&lt;/script&gt;",
		},
		{
			name:     "quote characters",
			input:    `Hello "World"`,
			expected: `Hello &#34;World&#34;`,
		},
		{
			name:     "ampersand",
			input:    "A & B",
			expected: "A &amp; B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForAttribute(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeForAttribute(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsAllowedFileType(t *testing.T) {
	// Test with default allowed types
	tests := []struct {
		name        string
		mimeType    string
		allowedType []string
		expected    bool
	}{
		{
			name:     "allowed image type",
			mimeType: "image/jpeg",
			expected: true,
		},
		{
			name:     "allowed video type",
			mimeType: "video/mp4",
			expected: true,
		},
		{
			name:     "allowed audio type",
			mimeType: "audio/mpeg",
			expected: true,
		},
		{
			name:     "allowed pdf",
			mimeType: "application/pdf",
			expected: true,
		},
		{
			name:     "disallowed exe",
			mimeType: "application/x-executable",
			expected: false,
		},
		{
			name:     "disallowed html",
			mimeType: "text/html",
			expected: false,
		},
		{
			name:     "new flac type allowed",
			mimeType: "audio/flac",
			expected: true,
		},
		{
			name:     "new aac type allowed",
			mimeType: "audio/aac",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAllowedFileType(tt.mimeType, tt.allowedType)
			if result != tt.expected {
				t.Errorf("IsAllowedFileType(%q) = %v, want %v", tt.mimeType, result, tt.expected)
			}
		})
	}

	// Test with custom allowed types
	t.Run("custom allowed types", func(t *testing.T) {
		customTypes := []string{"image/png", "image/jpeg"}
		if !IsAllowedFileType("image/png", customTypes) {
			t.Error("image/png should be allowed")
		}
		if IsAllowedFileType("image/gif", customTypes) {
			t.Error("image/gif should not be allowed")
		}
	})
}
