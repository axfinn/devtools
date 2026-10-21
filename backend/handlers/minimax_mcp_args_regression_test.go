package handlers

// =====================================================================
//  MiniMax MCP 默认 args 回归护栏 (FINN-56 Stage 5)
//
//  背景:
//    MiniMax MCP 的官方调用形式是 `uvx minimax-coding-plan-mcp -y`,
//    `-y` 必须紧跟包名(否则 MCP 进程会卡在交互确认)。
//    仓库里有 4 处写死了这套 args,其中 3 处历史上缺 `-y`:
//      1) backend/config/config.go        (DefaultConfig)
//      2) backend/config.example.yaml     (部署示例)
//      3) docs/image-understanding-api.md (接口文档示例)
//    第 4 处 entrypoint.sh 本来就是对的(它是容器内 Claude Code CLI 的配置,
//    与 Go 后端读 config.yaml 是两条独立链路)。
//
//  本测试的三条护栏:
//    护栏 1:DefaultConfig().MiniMaxMCP.Args 必须是 ["minimax-coding-plan-mcp","-y"]
//    护栏 2:没有 config.yaml 的全新部署,Load() 出来仍是同一组 args
//    护栏 3:config.example.yaml / docs 示例 / entrypoint.sh 四处逐字一致
//            (防止「三处再次漂移」—— 这正是本 issue 的根因)
//
//  跑通 = 任何人把某处的 `-y` 删掉,CI 立刻红。
// =====================================================================

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"devtools/config"
)

// wantMiniMaxMCPArgs 是四处(MiniMax MCP 官方调用形式)必须逐字一致的 args。
// 这里刻意用字面量而不是实现里的常量 —— 目的是锁住真实取值,
// 防止有人把常量本身改错时测试跟着一起「通过」。
var wantMiniMaxMCPArgs = []string{"minimax-coding-plan-mcp", "-y"}

// minimaxArgsFromYAML 读一个 yaml 文件,取出 minimax_mcp.args。
func minimaxArgsFromYAML(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", path, err)
	}
	var doc struct {
		MiniMaxMCP struct {
			Args []string `yaml:"args"`
		} `yaml:"minimax_mcp"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("解析 %s 失败: %v", path, err)
	}
	if doc.MiniMaxMCP.Args == nil {
		t.Fatalf("%s 里没有 minimax_mcp.args(示例配置被删/改键名了?)", path)
	}
	return doc.MiniMaxMCP.Args
}

// extractYAMLBlocks 从 markdown 中抽出所有 ```yaml 围栏代码块。
func extractYAMLBlocks(content string) []string {
	var blocks []string
	rest := content
	for {
		start := strings.Index(rest, "```yaml")
		if start < 0 {
			break
		}
		rest = rest[start+len("```yaml"):]
		end := strings.Index(rest, "```")
		if end < 0 {
			break
		}
		blocks = append(blocks, rest[:end])
		rest = rest[end+len("```"):]
	}
	return blocks
}

// minimaxArgsFromMarkdownDoc 从 markdown 文档里找出含 minimax_mcp 的 yaml 块并取 args。
func minimaxArgsFromMarkdownDoc(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", path, err)
	}
	for i, block := range extractYAMLBlocks(string(data)) {
		if !strings.Contains(block, "minimax_mcp:") {
			continue
		}
		var doc struct {
			MiniMaxMCP struct {
				Args []string `yaml:"args"`
			} `yaml:"minimax_mcp"`
		}
		if err := yaml.Unmarshal([]byte(block), &doc); err != nil {
			t.Fatalf("解析 %s 第 %d 个 yaml 代码块失败: %v", path, i, err)
		}
		if doc.MiniMaxMCP.Args == nil {
			t.Fatalf("%s 第 %d 个 yaml 代码块里没有 minimax_mcp.args", path, i)
		}
		return doc.MiniMaxMCP.Args
	}
	t.Fatalf("%s 里找不到含 minimax_mcp 的 yaml 代码块", path)
	return nil
}

// minimaxArgsFromEntrypoint 从 entrypoint.sh 里抓出 mcpServers.MiniMax 的 args 数组。
func minimaxArgsFromEntrypoint(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", path, err)
	}
	re := regexp.MustCompile(`"args"\s*:\s*(\[[^\]]*\])`)
	match := re.FindStringSubmatch(string(data))
	if match == nil {
		t.Fatalf("%s 里找不到 \"args\": [...] 形式的参数数组", path)
	}
	var args []string
	if err := json.Unmarshal([]byte(match[1]), &args); err != nil {
		t.Fatalf("解析 %s 里的 args 数组失败: %v", path, err)
	}
	return args
}

// TestMiniMaxMCPArgsDefaultRegression 三条护栏合一:
//
//	护栏 1 —— DefaultConfig() 的默认 args 必须带 -y
//	护栏 2 —— 没有 config.yaml 的全新部署,Load() 出来仍是同一组 args
//	护栏 3 —— config.example.yaml / docs 示例 / entrypoint.sh 与 DefaultConfig 逐字一致
func TestMiniMaxMCPArgsDefaultRegression(t *testing.T) {
	// —— 护栏 1:默认配置 ——
	defaultArgs := config.DefaultConfig().MiniMaxMCP.Args
	if !equalArgs(defaultArgs, wantMiniMaxMCPArgs) {
		t.Fatalf("护栏 1 失败:DefaultConfig().MiniMaxMCP.Args\n  期望 = %#v\n  实际 = %#v\n"+
			"官方调用形式是 `uvx minimax-coding-plan-mcp -y`,`-y` 必须紧跟包名。",
			wantMiniMaxMCPArgs, defaultArgs)
	}

	// —— 护栏 2:全新部署(无 config.yaml)回落仍是同一组 args ——
	nonexistent := filepath.Join(t.TempDir(), "config-does-not-exist.yaml")
	t.Setenv("CONFIG_PATH", nonexistent)
	loaded, err := config.Load(nonexistent)
	if err != nil {
		t.Fatalf("config.Load(%s) 失败: %v", nonexistent, err)
	}
	if !equalArgs(loaded.MiniMaxMCP.Args, wantMiniMaxMCPArgs) {
		t.Fatalf("护栏 2 失败:无 config.yaml 时 Load 出来的 args\n  期望 = %#v\n  实际 = %#v",
			wantMiniMaxMCPArgs, loaded.MiniMaxMCP.Args)
	}

	// —— 护栏 3:四处逐字一致(防止三处再次漂移) ——
	// 本测试文件在 backend/handlers/ 下,仓库根在 ../../。
	repoRoot := filepath.Join("..", "..")

	exampleYAML := filepath.Join(repoRoot, "backend", "config.example.yaml")
	exampleArgs := minimaxArgsFromYAML(t, exampleYAML)
	if !equalArgs(exampleArgs, wantMiniMaxMCPArgs) {
		t.Fatalf("护栏 3 失败:%s 的 minimax_mcp.args\n  期望 = %#v\n  实际 = %#v",
			exampleYAML, wantMiniMaxMCPArgs, exampleArgs)
	}

	docPath := filepath.Join(repoRoot, "docs", "image-understanding-api.md")
	docArgs := minimaxArgsFromMarkdownDoc(t, docPath)
	if !equalArgs(docArgs, wantMiniMaxMCPArgs) {
		t.Fatalf("护栏 3 失败:%s 的示例 args\n  期望 = %#v\n  实际 = %#v",
			docPath, wantMiniMaxMCPArgs, docArgs)
	}

	entrypointPath := filepath.Join(repoRoot, "entrypoint.sh")
	entrypointArgs := minimaxArgsFromEntrypoint(t, entrypointPath)
	if !equalArgs(entrypointArgs, wantMiniMaxMCPArgs) {
		t.Fatalf("护栏 3 失败:%s 的 args\n  期望 = %#v\n  实际 = %#v",
			entrypointPath, wantMiniMaxMCPArgs, entrypointArgs)
	}

	// 护栏 3 的反面:示例配置与默认配置之间不能漂移(逐字相等,不只是都带 -y)
	if !equalArgs(exampleArgs, defaultArgs) {
		t.Fatalf("护栏 3 失败:%s 的 args 与 DefaultConfig() 不一致\n  DefaultConfig = %#v\n  示例配置    = %#v",
			exampleYAML, defaultArgs, exampleArgs)
	}
	if !equalArgs(docArgs, defaultArgs) {
		t.Fatalf("护栏 3 失败:%s 的示例 args 与 DefaultConfig() 不一致\n  DefaultConfig = %#v\n  文档示例    = %#v",
			docPath, defaultArgs, docArgs)
	}
}
