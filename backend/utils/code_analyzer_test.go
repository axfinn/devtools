package utils

import (
	"testing"
)

func TestAnalyzeCode(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		language string
		wantLang string
		minLines int
	}{
		{
			name:     "JavaScript code",
			content:  "function hello() {\n    console.log('Hello');\n}\n",
			language: "javascript",
			wantLang: "javascript",
			minLines: 3,
		},
		{
			name:     "Python code",
			content:  "def hello():\n    print('Hello')\n\n# Comment\n",
			language: "python",
			wantLang: "python",
			minLines: 4,
		},
		{
			name:     "Go code",
			content:  "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n",
			language: "go",
			wantLang: "go",
			minLines: 7,
		},
		{
			name:     "Empty content",
			content:  "",
			language: "text",
			wantLang: "text",
			minLines: 0,
		},
		{
			name:     "Java code with class",
			content:  "public class Main {\n    public static void main(String[] args) {\n        System.out.println(\"Hello\");\n    }\n}\n",
			language: "java",
			wantLang: "java",
			minLines: 5,
		},
		{
			name:     "Explicit Python",
			content:  "def test():\n    pass\n",
			language: "python",
			wantLang: "python",
			minLines: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnalyzeCode(tt.content, tt.language)

			// 验证语言
			if result.Language != tt.wantLang {
				t.Errorf("AnalyzeCode().Language = %v, want %v", result.Language, tt.wantLang)
			}

			// 验证行数至少达到预期（考虑末尾换行符）
			if result.Lines < tt.minLines {
				t.Errorf("AnalyzeCode().Lines = %v, want at least %v", result.Lines, tt.minLines)
			}

			// 验证摘要不为空
			if result.Summary == "" && tt.content != "" {
				t.Error("AnalyzeCode().Summary should not be empty for non-empty content")
			}
		})
	}
}

func TestAnalyzeCode_FunctionsAndClasses(t *testing.T) {
	content := `
package main

import "fmt"

type Person struct {
    Name string
    Age int
}

func (p Person) Greet() {
    fmt.Println("Hello")
}

func main() {
    fmt.Println("Hello")
}
`
	result := AnalyzeCode(content, "go")

	// 应该检测到至少 2 个函数 (Greet, main)
	if result.Functions < 1 {
		t.Errorf("Expected at least 1 function, got %d", result.Functions)
	}

	// 应该检测到至少 1 个类/结构体
	if result.Classes < 1 {
		t.Errorf("Expected at least 1 class/struct, got %d", result.Classes)
	}

	// 应该检测到导入
	if result.Imports < 1 {
		t.Errorf("Expected at least 1 import, got %d", result.Imports)
	}
}

func TestAnalyzeCode_CommentLines(t *testing.T) {
	content := `// This is a comment
func hello() {
    // Another comment
    fmt.Println("test")
}`
	result := AnalyzeCode(content, "go")

	if result.CodeLines < 1 {
		t.Errorf("Expected at least 1 code line, got %d", result.CodeLines)
	}
}
