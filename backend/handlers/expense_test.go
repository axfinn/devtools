package handlers

import (
	"testing"
)

// TestParseChineseNumber 测试中文数字解析
func TestParseChineseNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		// 基础数字
		{"零", 0},
		{"一", 1},
		{"二", 2},
		{"五", 5},
		// 十的倍数
		{"十", 10},
		{"二十", 20},
		{"三十", 30},
		// 带个位数
		{"十五", 15},
		{"二十五", 25},
		{"三十五", 35},
		// 百以上（简化处理）
		{"一百", 100},
		{"一百二十五", 125},
		{"一百三十五", 135},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseChineseNumber(tt.input)
			if result != tt.expected {
				t.Errorf("parseChineseNumber(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseChineseAmount 测试中文金额解析（实际使用场景）
func TestParseChineseAmount(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		// 纯数字
		{"35", 35},
		{"100", 100},
		{"12.5", 12.5},
		// 数字 + 单位
		{"花了30", 30},
		{"花了35.5", 35.5},
		{"付了50", 50},
		{"花35块", 35},
		// 实际语音输入场景
		{"吃饭花了35块", 35},
		{"打车花 了20", 20},
		{"买奶茶用了15元", 15},
		{"加油300", 300},
		{"花了50元", 50},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseChineseAmount(tt.input)
			if result != tt.expected {
				t.Errorf("parseChineseAmount(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestBasicVoiceParse 测试语音解析完整流程（实际使用场景）
func TestBasicVoiceParse(t *testing.T) {
	tests := []struct {
		input         string
		wantAmount    float64
		wantCategory  string
		wantType      string
	}{
		// 餐饮场景
		{"吃饭花了35块", 35, "餐饮", "expense"},
		{"中午吃外卖花了28元", 28, "餐饮", "expense"},
		{"喝奶茶花了15块", 15, "餐饮", "expense"},
		{"去超市买菜用了80", 80, "餐饮", "expense"},
		{"叫外卖40元", 40, "餐饮", "expense"},

		// 交通场景
		{"打车花了20块", 20, "交通", "expense"},
		{"坐地铁用了5元", 5, "交通", "expense"},
		{"加油花了300", 300, "交通", "expense"},
		{"停车费15元", 15, "交通", "expense"},

		// 购物场景
		{"淘宝买衣服花了200", 200, "购物", "expense"},
		{"京东花了500块", 500, "购物", "expense"},
		{"快递费10元", 10, "购物", "expense"},

		// 收入场景
		{"发工资了5000", 5000, "工资", "income"},
		{"发奖金3000元", 3000, "奖金", "income"},
		{"收到稿费1000", 1000, "其他收入", "income"},

		// 纯数字（兜底）
		{"花了50", 50, "", "expense"},
		{"收入10000", 10000, "其他收入", "income"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := basicVoiceParse(tt.input, nil)

			if result.Amount != tt.wantAmount {
				t.Errorf("Amount = %v, want %v", result.Amount, tt.wantAmount)
			}
			if tt.wantCategory != "" && result.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", result.Category, tt.wantCategory)
			}
			if result.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", result.Type, tt.wantType)
			}
		})
	}
}
