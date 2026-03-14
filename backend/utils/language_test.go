package utils

import (
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "JavaScript",
			content:  `function hello() { console.log("Hello"); }`,
			expected: "javascript",
		},
		{
			name:     "TypeScript",
			content:  `interface User { name: string; age: number; }`,
			expected: "typescript",
		},
		{
			name:     "Python",
			content:  `def hello(): print("Hello")`,
			expected: "python",
		},
		{
			name:     "Go",
			content:  `package main; func main() { fmt.Println("Hello") }`,
			expected: "go",
		},
		{
			name:     "Java",
			content:  `public static void main(String[] args) { System.out.println("Hello"); }`,
			expected: "java",
		},
		{
			name:     "C",
			content:  `#include <stdio.h> int main() { printf("Hello"); }`,
			expected: "c",
		},
		{
			name:     "C++",
			content:  `#include <iostream> int main() { std::cout << "Hello"; }`,
			expected: "cpp",
		},
		{
			name:     "C#",
			content:  `using System; class Program { static void Main() { Console.WriteLine("Hello"); } }`,
			expected: "csharp",
		},
		{
			name:     "Swift",
			content:  `func hello() { print("Hello") }`,
			expected: "swift",
		},
		{
			name:     "Kotlin",
			content:  `fun main() { println("Hello") }`,
			expected: "kotlin",
		},
		{
			name:     "CSS",
			content:  `.class { color: #fff; background: red; }`,
			expected: "css",
		},
		{
			name:     "JSON",
			content:  `{"name": "test", "value": 123}`,
			expected: "json",
		},
		{
			name:     "Empty content",
			content:  "",
			expected: "text",
		},
		{
			name:     "Plain text",
			content:  "Hello, this is just plain text without any code.",
			expected: "text",
		},
		{
			name:     "Go with package and import",
			content:  `package main
import "fmt"
func main() { fmt.Println("Hello") }`,
			expected: "go",
		},
		{
			name:     "Python with class",
			content:  `class Hello:
    def __init__(self):
        pass`,
			expected: "python",
		},
		{
			name:     "JavaScript arrow function",
			content:  `const hello = () => { console.log("Hello"); };`,
			expected: "javascript",
		},
		{
			name:     "TypeScript interface",
			content:  `interface Props { name: string; }`,
			expected: "typescript",
		},
		{
			name:     "Bash script",
			content:  `#!/bin/bash
echo "Hello"`,
			expected: "bash",
		},
		{
			name:     "Ruby def",
			content:  `def hello
  puts "Hello"
end`,
			expected: "ruby",
		},
		{
			name:     "PHP tag",
			content:  `<?php
echo "Hello";`,
			expected: "php",
		},
	}

	passed := 0
	failed := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectLanguage(tt.content)
			if result != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q; want %q", tt.content, result, tt.expected)
				failed++
			} else {
				passed++
			}
		})
	}
	t.Logf("Passed: %d, Failed: %d", passed, failed)
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		language  string
		expected  string
	}{
		{
			name:      "Markdown language",
			content:   "# Hello",
			language:  "markdown",
			expected:  "markdown",
		},
		{
			name:      "JavaScript code",
			content:   "function hello() { console.log('Hello'); }",
			language:  "javascript",
			expected:  "code",
		},
		{
			name:      "Plain text",
			content:   "Hello, this is plain text.",
			language:  "text",
			expected:  "text",
		},
		{
			name:      "Empty content",
			content:   "",
			language:  "text",
			expected:  "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectContentType(tt.content, tt.language)
			if result != tt.expected {
				t.Errorf("DetectContentType(%q, %q) = %q; want %q", tt.content, tt.language, result, tt.expected)
			}
		})
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	langs := GetSupportedLanguages()

	// 检查是否包含关键语言
	keyLangs := []string{"javascript", "python", "go", "markdown", "text"}

	for _, lang := range keyLangs {
		found := false
		for _, l := range langs {
			if l == lang {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected language %q not found in supported languages", lang)
		}
	}

	// 检查数量 - 至少支持30种语言
	if len(langs) < 30 {
		t.Errorf("Expected at least 30 supported languages, got %d", len(langs))
	}
}
