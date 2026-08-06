package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

// ============================================================
// Skills 模块: 对外工具入口,让 codex/Claude Code 等外部 AI 编程助手
// 通过 HTTP + JSON-RPC 调用 DevTools 的非鉴权基础能力。
//
// 协议:
//   - GET  /api/skills/manifest   -> OpenAI function-calling 风格清单
//   - POST /api/skills/invoke     -> 内部简洁 JSON-RPC: {name, arguments}
//   - GET  /api/skills/mcp        -> MCP server info / SSE stream
//   - POST /api/skills/mcp        -> MCP Streamable HTTP(JSON-RPC)
//
// 设计原则:
//   - 0 鉴权、依赖上层 SkillsGuard(enabled + IP + 可选 Origin)
//   - 0 侵入 paste/shorturl/dns handler(走最简路径,自带上限)
//   - 写库类带更严专项限流(risk=="write")
//   - SSRF / Regex DoS / 大输入 全部在 skill 层自己防
// ============================================================

// Skill 单个对外工具的元数据 + 行为
type Skill struct {
	Name        string                  // tool name,如 "base64_encode"
	Description string                  // 一句话描述
	InputSchema map[string]interface{}  // OpenAI tools 风格:{"type":"object","properties":...,"required":[...], "additionalProperties":false}
	Risk        string                  // "compute" 或 "write",决定是否走更严限流
	Invoke      func(args map[string]any, ctx *SkillContext) (any, error)
}

// SkillContext 运行期上下文(客户端 IP / host / DB / Store 等)
type SkillContext struct {
	DB           *models.DB
	IP           string
	Host         string              // c.Request.Host(可能含端口)
	Scheme       string              // "http" / "https",短链/paste 分享链接拼接用
	Guard        skillsGuardLocal    // SkillsHandler 内部传一个简化 guard,只暴露写库专项限流
	RequestCtx   context.Context     // 请求级 ctx,转交给 handler 复用 helper
	PasteHandler *PasteHandler       // 注入后 paste_create 可复用 createPasteCore(全功能)
}

type skillsGuardLocal interface {
	CheckWriteLimit(skillName, ip string) error
}

// SkillsHandler Skills 模块 HTTP 处理器
type SkillsHandler struct {
	db           *models.DB
	guard        skillsGuardLocal
	pasteHandler *PasteHandler
	tools        []Skill
}

// NewSkillsHandler 返回 SkillsHandler;tools 已从 SkillRegistry 静态组装
func NewSkillsHandler(db *models.DB) *SkillsHandler {
	return &SkillsHandler{
		db:    db,
		guard: nil,
		tools: defaultSkillRegistry(),
	}
}

// AttachGuard 给 SkillsHandler 注入 guard(由调用方 app_handlers 在初始化阶段装配)
// Skills handler 自己不直接 import middleware,以避免循环依赖
func (h *SkillsHandler) AttachGuard(g skillsGuardLocal) {
	h.guard = g
}

// AttachPasteHandler 把 PasteHandler 注入 skills,使 paste_create 能复用完整逻辑
//(支持 title / language / password / expires_in / max_views / admin_password)。
// 与 AttachGuard 一样,留空时 skill 走"瘦壳"实现(自己限参数)。
func (h *SkillsHandler) AttachPasteHandler(p *PasteHandler) {
	h.pasteHandler = p
}

// SetRegistry 替换完整 tool 注册表(测试用)
func (h *SkillsHandler) SetRegistry(tools []Skill) { h.tools = tools }

// SetRegistrySlice 增量追加 tools
func (h *SkillsHandler) AddSkill(s Skill) { h.tools = append(h.tools, s) }

// ToolByName 查找单个 skill 用于分发
func (h *SkillsHandler) toolByName(name string) (Skill, bool) {
	for _, s := range h.tools {
		if s.Name == name {
			return s, true
		}
	}
	return Skill{}, false
}

// GetManifest 返回 OpenAI function-calling tools 清单
// 兼容 Claude(读 input_schema)/ OpenAI(读 parameters)/ Cursor(两者皆读)
func (h *SkillsHandler) GetManifest(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server": gin.H{
			"name":     "devtools-skills",
			"version":  "1.0.0",
			"protocol": "openai-tools + mcp-streamable-http",
			"endpoint": "/api/skills",
			"install":  "/api/skills/install",
		},
		"tools": h.openAIToolsMetadata(),
	})
}

// GetInstall 返回各客户端的一键安装命令。
// 默认 text/plain(curl 一目了然),?format=json 拿结构化配置(JSON for tooling),
// ?client=X 拿单一客户端的完整片段。
func (h *SkillsHandler) GetInstall(c *gin.Context) {
	base := buildExternalBase(c)
	all := buildInstallSnippets(base)
	client := strings.ToLower(strings.TrimSpace(c.Query("client")))
	format := strings.ToLower(strings.TrimSpace(c.Query("format")))

	if client != "" && client != "all" {
		snippet, ok := all[client]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":     "unknown client",
				"supported": sortedKeys(all),
			})
			return
		}
		if format == "json" {
			c.JSON(http.StatusOK, gin.H{"client": client, "snippet": snippet})
		} else {
			c.Header("Content-Type", "text/plain; charset=utf-8")
			c.String(http.StatusOK, renderSnippetText(client, snippet))
		}
		return
	}

	if format == "json" {
		c.JSON(http.StatusOK, gin.H{
			"server": gin.H{
				"name":     "devtools-skills",
				"version":  "1.0.0",
				"endpoint": base + "/api/skills/mcp",
			},
			"clients": all,
		})
		return
	}

	// 默认:纯文本目录,curl 出来像 README,人 / AI 都可读
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, renderDirectoryText(base, all))
}

// GetInstallShell 直接给一个 ready-to-pipe 的 shell 脚本。
// 任何 AI 编程助手 / 用户 curl | bash 即可完成所有客户端的安装骨架
// (实际安装命令按本机工具链 patch;各客户端 if 块注释掉了,按需取消)。
func (h *SkillsHandler) GetInstallShell(c *gin.Context) {
	base := buildExternalBase(c)
	all := buildInstallSnippets(base)
	c.Header("Content-Type", "text/x-shellscript; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="devtools-skills-install.sh"`)
	c.String(http.StatusOK, renderInstallScript(base, all))
}

// GetDirectory 是 .well-known/skills 风格的 AI agent 发现端点:
// 返回 JSON 目录,列出可用的 skill 和安装命令入口(供智能体自动发现)。
// 路径用 .well-known 是因为这是 IETF RFC 8615 + Anthropic 推荐的
// AI agent 自描述约定。
func (h *SkillsHandler) GetDirectory(c *gin.Context) {
	base := buildExternalBase(c)
	all := buildInstallSnippets(base)
	c.JSON(http.StatusOK, gin.H{
		"server": gin.H{
			"name":        "devtools-skills",
			"version":     "1.0.0",
			"description": "DevTools 对外工具入口:14 个非鉴权 skill(纯计算 12 + 写库 2),MCP Streamable HTTP + OpenAI tools 双协议",
			"homepage":    base + "/skills",
		},
		"endpoints": gin.H{
			"manifest":       base + "/api/skills/manifest",
			"mcp":            base + "/api/skills/mcp",
			"invoke":         base + "/api/skills/invoke",
			"install_text":   base + "/api/skills/install",
			"install_json":   base + "/api/skills/install?format=json",
			"install_shell":  base + "/api/skills/install.sh",
			"install_client": base + "/api/skills/install?client=<name>",
		},
		"clients": sortedKeys(all),
		"skills":  h.openAIToolsMetadata(),
	})
}

func buildInstallSnippets(base string) map[string]gin.H {
	mcpURL := base + "/api/skills/mcp"
	manifestURL := base + "/api/skills/manifest"
	invokeURL := base + "/api/skills/invoke"
	return map[string]gin.H{
		"claude_code": {
			"label": "Claude Code (Anthropic CLI)",
			"one_liner":   "claude mcp add --transport http devtools " + mcpURL,
			"interactive": "claude mcp add devtools",
			"config_json": `{
  "mcpServers": {
    "devtools": {
      "type": "http",
      "url": "` + mcpURL + `"
    }
  }
}`,
		},
		"codex": {
			"label":     "OpenAI Codex CLI",
			"one_liner": "codex mcp add devtools --url " + mcpURL,
			"config_toml": "[mcp_servers.devtools]\n" +
				"url = \"" + mcpURL + "\"\n" +
				"transport = \"http\"\n",
		},
		"cursor": {
			"label":     "Cursor (~/.cursor/mcp.json)",
			"one_liner": "open -a Cursor ~/.cursor/mcp.json",
			"config_json": `{
  "mcpServers": {
    "devtools": {
      "url": "` + mcpURL + `"
    }
  }
}`,
		},
		"vscode": {
			"label":     "VS Code (Copilot Chat)",
			"one_liner": `code --add-mcp '{"name":"devtools","url":"` + mcpURL + `"}'`,
			"config_json": `{
  "servers": {
    "devtools": {
      "type": "http",
      "url": "` + mcpURL + `"
    }
  }
}`,
		},
		"continue": {
			"label": "Continue (VS Code / JetBrains)",
			"config_yaml": "name: DevTools\n" +
				"version: 1.0.0\n" +
				"schema: v1\n" +
				"mcpServers:\n" +
				"  - name: devtools\n" +
				"    type: http\n" +
				"    url: " + mcpURL + "\n",
		},
		"curl": {
			"label": "纯 cURL(任意 HTTP 客户端)",
			"snippets": gin.H{
				"manifest": "curl -s " + manifestURL,
				"mcp_initialize": "curl -sX POST " + mcpURL + " \\\n" +
					"  -H 'Content-Type: application/json' \\\n" +
					"  -H 'Accept: application/json, text/event-stream' \\\n" +
					"  -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2025-06-18\"}}'",
				"invoke_uuid": "curl -sX POST " + invokeURL + " \\\n" +
					"  -H 'Content-Type: application/json' \\\n" +
					"  -d '{\"name\":\"uuid_v4\",\"arguments\":{}}'",
			},
		},
	}
}

// renderDirectoryText 把所有客户端的安装命令拼成 README 风格的纯文本。
// 任何 AI 编程助手 / 人 curl 一次直接看明白。
func renderDirectoryText(base string, all map[string]gin.H) string {
	var b strings.Builder
	b.WriteString("DevTools Skills — 一键安装\n")
	b.WriteString("========================\n\n")
	b.WriteString("MCP endpoint:  " + base + "/api/skills/mcp\n")
	b.WriteString("Manifest:      " + base + "/api/skills/manifest\n")
	b.WriteString("Invoke:        " + base + "/api/skills/invoke\n")
	b.WriteString("Install (text): " + base + "/api/skills/install\n")
	b.WriteString("Install (json): " + base + "/api/skills/install?format=json\n\n")

	// 排序后输出,稳定
	keys := sortedKeys(all)
	for _, k := range keys {
		snip := all[k]
		b.WriteString("## " + k + "\n")
		if label, ok := snip["label"].(string); ok {
			b.WriteString(label + "\n")
		}
		if one, ok := snip["one_liner"].(string); ok {
			b.WriteString("  $ " + one + "\n")
		}
		if cj, ok := snip["config_json"].(string); ok {
			b.WriteString("  config_json:\n")
			for _, line := range strings.Split(cj, "\n") {
				b.WriteString("    " + line + "\n")
			}
		}
		if ct, ok := snip["config_toml"].(string); ok {
			b.WriteString("  config_toml:\n")
			for _, line := range strings.Split(ct, "\n") {
				b.WriteString("    " + line + "\n")
			}
		}
		if cy, ok := snip["config_yaml"].(string); ok {
			b.WriteString("  config_yaml:\n")
			for _, line := range strings.Split(cy, "\n") {
				b.WriteString("    " + line + "\n")
			}
		}
		if subs, ok := snip["snippets"].(gin.H); ok {
			b.WriteString("  snippets:\n")
			// 按已知 key 顺序
			for _, name := range []string{"manifest", "mcp_initialize", "invoke_uuid"} {
				if v, ok := subs[name].(string); ok {
					b.WriteString("    " + name + ":\n")
					for _, line := range strings.Split(v, "\n") {
						b.WriteString("      " + line + "\n")
					}
				}
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderSnippetText 单个客户端的纯文本渲染
func renderSnippetText(name string, snip gin.H) string {
	var b strings.Builder
	b.WriteString("# " + name)
	if label, ok := snip["label"].(string); ok {
		b.WriteString(" — " + label)
	}
	b.WriteString("\n\n")
	if one, ok := snip["one_liner"].(string); ok {
		b.WriteString("$ " + one + "\n\n")
	}
	for _, key := range []string{"config_json", "config_toml", "config_yaml"} {
		if v, ok := snip[key].(string); ok {
			b.WriteString(key + ":\n")
			for _, line := range strings.Split(v, "\n") {
				b.WriteString("  " + line + "\n")
			}
			b.WriteString("\n")
		}
	}
	if subs, ok := snip["snippets"].(gin.H); ok {
		for _, name := range []string{"manifest", "mcp_initialize", "invoke_uuid"} {
			if v, ok := subs[name].(string); ok {
				b.WriteString(name + ":\n")
				for _, line := range strings.Split(v, "\n") {
					b.WriteString("  " + line + "\n")
				}
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

// renderInstallScript 把所有客户端的安装命令拼成一个 ready-to-pipe 的 shell 脚本。
// 每个客户端的安装块是独立的 if 分支,默认全部注释掉(防误装),
// 用户按需取消对应行的注释即可。
func renderInstallScript(base string, all map[string]gin.H) string {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("# DevTools Skills 一键安装脚本\n")
	b.WriteString("# 由 " + base + "/api/skills/install.sh 自动生成\n")
	b.WriteString("# 默认所有安装块被注释,按需取消;每个 block 独立可跑。\n\n")
	b.WriteString("set -euo pipefail\n\n")
	b.WriteString("DEVTOOLS_MCP_URL=\"" + base + "/api/skills/mcp\"\n\n")
	for _, k := range sortedKeys(all) {
		snip := all[k]
		b.WriteString("# ===== " + k)
		if label, ok := snip["label"].(string); ok {
			b.WriteString(" — " + label)
		}
		b.WriteString(" =====\n")
		if one, ok := snip["one_liner"].(string); ok {
			// 加 # 注释掉,保留原命令供查阅
			b.WriteString("# " + one + "\n")
		}
		// 对 claude_code 这种带 one_liner 的,生成可执行的命令块
		if k == "claude_code" {
			b.WriteString("# claude mcp add --transport http devtools \"$DEVTOOLS_MCP_URL\"\n")
		}
		if k == "codex" {
			b.WriteString("# codex mcp add devtools --url \"$DEVTOOLS_MCP_URL\"\n")
		}
		if cj, ok := snip["config_json"].(string); ok {
			b.WriteString("# config_json:\n")
			for _, line := range strings.Split(cj, "\n") {
				b.WriteString("# " + line + "\n")
			}
			b.WriteString("#\n")
			b.WriteString("# cat > ~/.cursor/mcp.json <<'EOF'\n")
			for _, line := range strings.Split(cj, "\n") {
				b.WriteString("# " + line + "\n")
			}
			b.WriteString("# EOF\n")
		}
		if ct, ok := snip["config_toml"].(string); ok {
			b.WriteString("# config_toml:\n")
			for _, line := range strings.Split(ct, "\n") {
				b.WriteString("# " + line + "\n")
			}
		}
		if cy, ok := snip["config_yaml"].(string); ok {
			b.WriteString("# config_yaml:\n")
			for _, line := range strings.Split(cy, "\n") {
				b.WriteString("# " + line + "\n")
			}
		}
		if subs, ok := snip["snippets"].(gin.H); ok {
			b.WriteString("# smoke test:\n")
			if v, ok := subs["mcp_initialize"].(string); ok {
				for _, line := range strings.Split(v, "\n") {
					b.WriteString("# " + line + "\n")
				}
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func sortedKeys(m map[string]gin.H) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// buildExternalBase 从 Host / X-Forwarded-Proto / scheme helper 拼出公网基址。
// SkillsGuard 已读 X-Forwarded-For,这里也读 X-Forwarded-Proto
// (反代到 HTTPS 容器内仍是 HTTP,客户端展示需真实 scheme)。
// Host 为空时(测试 / 裸调用)回退到生产域名,免得文本里出现 "http:///" 这种丑字串。
func buildExternalBase(c *gin.Context) string {
	scheme := schemeFrom(c)
	host := c.Request.Host
	if xfh := c.GetHeader("X-Forwarded-Host"); xfh != "" {
		host = xfh
	}
	if host == "" {
		host = "t.jaxiu.cn"
	}
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + host
}

// toolMetadata 输出每个 skill 的元数据,严格遵守 MCP 2025-06-18 规范:
//   - name         (string, 必填)
//   - description  (string, 可选但建议)
//   - inputSchema  (object, 必填, JSON Schema 格式,additionalProperties: false)
// 注意:不输出 `risk`(服务端内部限流用,非协议字段)、
// 也不输出 `parameters`(那是 OpenAI 工具格式,见 manifest 端点)。
// 历史坑:之前误用 snake_case `input_schema`,MCP 客户端按规范查 `inputSchema` 会读到 undefined。
func (h *SkillsHandler) toolMetadata() []map[string]any {
	out := make([]map[string]any, 0, len(h.tools))
	for _, s := range h.tools {
		out = append(out, map[string]any{
			"name":        s.Name,
			"description": s.Description,
			"inputSchema": s.InputSchema,
		})
	}
	return out
}

// openAIToolsMetadata 输出 OpenAI function-calling 风格的元数据(用于
// GET /api/skills/manifest,供 OpenAI 工具调用 / Cursor 之类使用)。
// 包含 `parameters`(OpenAI 标准字段),同时保留 `inputSchema` 供 Claude 读。
func (h *SkillsHandler) openAIToolsMetadata() []map[string]any {
	out := make([]map[string]any, 0, len(h.tools))
	for _, s := range h.tools {
		out = append(out, map[string]any{
			"name":        s.Name,
			"description": s.Description,
			"parameters":  s.InputSchema, // OpenAI 标准
			"inputSchema": s.InputSchema, // Claude / 跨平台兼容
		})
	}
	return out
}

// Invoke POST /api/skills/invoke 内部简洁 JSON-RPC
// 请求: {"name":"base64_encode","arguments":{"text":"hello"}}
// 响应(成功): {"ok":true,"data":...}
// 响应(失败): {"ok":false,"error":"...","code":400|429|500}
func (h *SkillsHandler) Invoke(c *gin.Context) {
	var req struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "无效请求体", "code": 400})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "缺少 name", "code": 400})
		return
	}

	skill, ok := h.toolByName(req.Name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "未找到 skill: " + req.Name, "code": 404})
		return
	}

	ip := clientIP(c)
	sctx := &SkillContext{
		DB:           h.db,
		IP:           ip,
		Host:         c.Request.Host,
		Scheme:       schemeFrom(c),
		Guard:        h.guard,
		RequestCtx:   c.Request.Context(),
		PasteHandler: h.pasteHandler,
	}

	// 写库类先过专项限流
	if skill.Risk == "write" && h.guard != nil {
		if err := h.guard.CheckWriteLimit(skill.Name, ip); err != nil {
			c.JSON(http.StatusTooManyRequests, gin.H{"ok": false, "error": err.Error(), "code": 429})
			return
		}
	}

	result, err := skill.Invoke(req.Arguments, sctx)
	if err != nil {
		code := 400
		if strings.HasPrefix(err.Error(), "rate") {
			code = 429
		} else if strings.Contains(err.Error(), "storage") || strings.Contains(err.Error(), "失败") {
			code = 500
		}
		c.JSON(code, gin.H{"ok": false, "error": err.Error(), "code": code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": result})
}

// ============================================================
// MCP Streamable HTTP(JSON-RPC over HTTP)
// 覆盖 2024-11 / 2025-03 / 2025-06 spec 的核心 method:
//   - initialize             -> 返回 serverInfo + capabilities(版本协商)
//   - notifications/initialized -> 202 Accepted
//   - tools/list             -> 返回 manifest 中的 tools
//   - tools/call             -> 分发到注册 skill
//   - ping                   -> 心跳
// ============================================================

// supportedMCPProtocolVersions 按"发布时间晚→早"排列,协商时优先回最新版。
// 真实 MCP 协议发布日期:
//   - 2024-11-05  初始
//   - 2025-03-26  引入 audio content / structured tool output
//   - 2025-06-18  最新
// 绝不要塞不存在的日期(比如 2025-05-06),客户端会拒握手。
var supportedMCPProtocolVersions = []string{
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}

// negotiateMCPVersion 按 MCP spec 实现:如果 client 发的版本在白名单内,echo 回去;
// 否则返回服务端最新支持的版本。空字符串(client 没发)也走"返回最新"路径。
func negotiateMCPVersion(clientVersion string) string {
	for _, v := range supportedMCPProtocolVersions {
		if v == clientVersion {
			return v
		}
	}
	return supportedMCPProtocolVersions[0]
}

// MCPGetHandler GET 形式返回 server info(供 stdio/SSE client 做 init probe)
func (h *SkillsHandler) MCPGetHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"jsonrpc": "2.0",
		"id":      nil,
		"result": gin.H{
			"serverInfo": gin.H{
				"name":    "devtools-skills",
				"version": "1.0.0",
			},
			"capabilities": gin.H{
				"tools": gin.H{"listChanged": false},
			},
			"protocolVersion": supportedMCPProtocolVersions[0],
		},
	})
}

// MCPPostHandler POST JSON-RPC endpoint,处理 initialize / tools/list / tools/call
func (h *SkillsHandler) MCPPostHandler(c *gin.Context) {
	var req struct {
		JSONRPC string         `json:"jsonrpc"`
		ID      any            `json:"id"`
		Method  string         `json:"method"`
		Params  map[string]any `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, mcpError(nil, -32700, "Parse error: "+err.Error()))
		return
	}
	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}

	// MCP Streamable HTTP 允许响应走 SSE 模式:当 client 在 Accept 里带上
	// text/event-stream 时,把 JSON-RPC 响应包成 event-stream 事件流
	// (event: message / data: <json> / 空行)。所有 14 个 skill 都同步,
	// SSE 在这里主要是兼容 codex / Claude Code 客户端的默认行为。
	sseMode := acceptsEventStream(c.GetHeader("Accept"))

	switch req.Method {
	case "initialize":
		clientVer, _ := req.Params["protocolVersion"].(string)
		resp := mcpResult(req.ID, gin.H{
			"serverInfo": gin.H{"name": "devtools-skills", "version": "1.0.0"},
			"capabilities": gin.H{
				"tools": gin.H{"listChanged": false},
			},
			"protocolVersion": negotiateMCPVersion(clientVer),
		})
		writeMCPResponse(c, sseMode, resp)

	case "notifications/initialized":
		// 客户端不需要回应,返回 202 Accepted(notification 无响应体)
		c.JSON(http.StatusAccepted, gin.H{})

	case "tools/list":
		writeMCPResponse(c, sseMode, mcpResult(req.ID, gin.H{
			"tools": h.toolMetadata(),
		}))

	case "tools/call":
		name, _ := req.Params["name"].(string)
		args, _ := req.Params["arguments"].(map[string]any)
		if args == nil {
			args = map[string]any{}
		}
		if name == "" {
			writeMCPResponse(c, sseMode, mcpError(req.ID, -32602, "missing tool name"))
			return
		}
		skill, ok := h.toolByName(name)
		if !ok {
			writeMCPResponse(c, sseMode, mcpError(req.ID, -32602, "unknown tool: "+name))
			return
		}
		ip := clientIP(c)
		sctx := &SkillContext{
			DB:           h.db,
			IP:           ip,
			Host:         c.Request.Host,
			Scheme:       schemeFrom(c),
			Guard:        h.guard,
			RequestCtx:   c.Request.Context(),
			PasteHandler: h.pasteHandler,
		}
		if skill.Risk == "write" && h.guard != nil {
			if err := h.guard.CheckWriteLimit(name, ip); err != nil {
				writeMCPResponse(c, sseMode, mcpError(req.ID, -32000, err.Error()))
				return
			}
		}
		result, err := skill.Invoke(args, sctx)
		if err != nil {
			writeMCPResponse(c, sseMode, mcpResult(req.ID, gin.H{
				"content": []gin.H{
					{"type": "text", "text": "Error: " + err.Error()},
				},
				"isError": true,
			}))
			return
		}
		// content 数组里塞 JSON 化结果
		text, _ := json.Marshal(result)
		writeMCPResponse(c, sseMode, mcpResult(req.ID, gin.H{
			"content": []gin.H{
				{"type": "text", "text": string(text)},
			},
			"isError": false,
		}))

	case "ping":
		writeMCPResponse(c, sseMode, mcpResult(req.ID, gin.H{"pong": true}))

	default:
		writeMCPResponse(c, sseMode, mcpError(req.ID, -32601, "method not found: "+req.Method))
	}
}

// acceptsEventStream 检查 Accept 头是否声明了 SSE 模式。
// MCP Streamable HTTP 允许 client 用 `Accept: application/json, text/event-stream`
// 协商走 SSE 响应;我们一律支持。
func acceptsEventStream(accept string) bool {
	if accept == "" {
		return false
	}
	for _, part := range strings.Split(accept, ",") {
		if strings.Contains(strings.TrimSpace(strings.ToLower(part)), "text/event-stream") {
			return true
		}
	}
	return false
}

// writeMCPResponse 按 sseMode 走两条路径之一:
//   - JSON 模式: Content-Type: application/json,单次 JSON 响应(默认)
//   - SSE 模式 : Content-Type: text/event-stream,响应包成单个 message 事件
// 所有 14 个 skill 都是同步响应,所以 SSE 模式下也只发一个事件再关流。
func writeMCPResponse(c *gin.Context, sseMode bool, payload any) {
	if !sseMode {
		c.JSON(http.StatusOK, payload)
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, mcpError(nil, -32603, "marshal failed: "+err.Error()))
		return
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	// SSE 帧格式:event: <name>\ndata: <json>\n\n
	fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", data)
	c.Writer.Flush()
}

func mcpResult(id any, result any) gin.H {
	return gin.H{"jsonrpc": "2.0", "id": id, "result": result}
}

func mcpError(id any, code int, msg string) gin.H {
	return gin.H{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   gin.H{"code": code, "message": msg},
	}
}

// ============================================================
// 工具函数
// ============================================================

func clientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		for _, p := range strings.Split(xff, ",") {
			if ip := strings.TrimSpace(p); ip != "" {
				return ip
			}
		}
	}
	if xri := strings.TrimSpace(c.GetHeader("X-Real-IP")); xri != "" {
		return xri
	}
	return c.ClientIP()
}

// hostBlacklisted 用于 dns_lookup,挡掉内网主机做 SSRF 防护。
// 黑名单覆盖:localhost / *.local / *.internal / RFC1918 / loopback / link-local / CGNAT
func hostBlacklisted(host string) bool {
	if host == "" {
		return true
	}
	low := strings.ToLower(host)
	if low == "localhost" || strings.HasSuffix(low, ".localhost") {
		return true
	}
	if strings.HasSuffix(low, ".local") || strings.HasSuffix(low, ".internal") {
		return true
	}
	if ip := net.ParseIP(low); ip != nil {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
			return true
		}
		// 100.64.0.0/10 CGNAT
		if v4 := ip.To4(); v4 != nil {
			if v4[0] == 100 && (v4[1] >= 64 && v4[1] <= 127) {
				return true
			}
		}
	}
	return false
}

// ============================================================
// Skill Registry — 静态注册 14 个 skill
// ============================================================

func defaultSkillRegistry() []Skill {
	// 只保留"只有这个服务能做"的能力:短链/粘贴(文本+附件)/服务端视角的 IP+DNS。
	// 其它(base64/url/json/timestamp/uuid/hash/regex)本地就能算,
	// 走网络无价值,反而拖慢响应、占配额。
	// paste 的附件上传走三个独立 skill(init/chunk/merge),与 /api/paste 共享存储路径。
	return []Skill{
		dnsLookupSkill(),
		ipLookupSkill(),
		shorturlCreateSkill(),
		pasteUploadInitSkill(),
		pasteUploadChunkSkill(),
		pasteUploadMergeSkill(),
		pasteCreateSkill(),
	}
}


// -------- dns_lookup --------
func dnsLookupSkill() Skill {
	return Skill{
		Name: "dns_lookup",
		Description: "DNS 解析。type 可空,空则返回所有常见记录。支持 A/AAAA/CNAME/MX/NS/TXT。",
		Risk:  "compute",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"domain": map[string]any{"type": "string", "description": "域名,如 example.com"},
				"type":   map[string]any{"type": "string", "description": "记录类型 A/AAAA/CNAME/MX/NS/TXT,默认全部"},
			},
			"required": []string{"domain"},
			"additionalProperties": false,
		},
		Invoke: func(args map[string]any, _ *SkillContext) (any, error) {
			domain := strings.TrimSpace(stringArg(args, "domain"))
			domain = strings.TrimPrefix(domain, "http://")
			domain = strings.TrimPrefix(domain, "https://")
			domain = strings.TrimSuffix(domain, "/")
			if idx := strings.Index(domain, "/"); idx >= 0 {
				domain = domain[:idx]
			}
			if domain == "" {
				return nil, errors.New("domain 不能为空")
			}
			if hostBlacklisted(domain) {
				return nil, errors.New("拒绝解析内网/保留域名: " + domain)
			}
			recordType := strings.ToUpper(strings.TrimSpace(stringArg(args, "type")))
			result := &DNSResult{}
			if recordType == "" {
				recordType = "ALL"
			}
			switch recordType {
			case "A":
				result.A = lookupA(domain)
			case "AAAA":
				result.AAAA = lookupAAAA(domain)
			case "CNAME":
				result.CNAME = lookupCNAME(domain)
			case "MX":
				result.MX = lookupMX(domain)
			case "NS":
				result.NS = lookupNS(domain)
			case "TXT":
				result.TXT = lookupTXT(domain)
			case "ALL":
				result.A = lookupA(domain)
				result.AAAA = lookupAAAA(domain)
				result.CNAME = lookupCNAME(domain)
				result.MX = lookupMX(domain)
				result.NS = lookupNS(domain)
				result.TXT = lookupTXT(domain)
			default:
				return nil, errors.New("不支持的 type: " + recordType)
			}
			return result, nil
		},
	}
}

// -------- ip_lookup --------
func ipLookupSkill() Skill {
	return Skill{
		Name: "ip_lookup",
		Description: "返回服务器看到的客户端 IP(从 X-Forwarded-For 第 1 个读,无则用连接 IP)。",
		Risk:  "compute",
		InputSchema: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
		// Invoke 由 SkillsHandler 直接覆盖;这里 placeholder 不会触发
		Invoke: func(_ map[string]any, ctx *SkillContext) (any, error) {
			return gin.H{"ip": ctx.IP}, nil
		},
	}
}

// -------- shorturl_create --------
func shorturlCreateSkill() Skill {
	return Skill{
		Name: "shorturl_create",
		Description: "为长 URL 生成短链。无密码模式,默认 48h 过期、200 次点击上限、5/min/IP;URL ≤ 512B、必须 http/https。",
		Risk:  "write",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{"type": "string", "description": "目标 URL,必须 http(s)://"},
			},
			"required": []string{"url"},
			"additionalProperties": false,
		},
		Invoke: func(args map[string]any, ctx *SkillContext) (any, error) {
			rawURL := strings.TrimSpace(stringArg(args, "url"))
			if rawURL == "" {
				return nil, errors.New("url 不能为空")
			}
			if len(rawURL) > 512 {
				return nil, errors.New("url 长度超过 512B 限制")
			}
			parsed, err := url.Parse(rawURL)
			if err != nil {
				return nil, fmt.Errorf("url 解析失败: %v", err)
			}
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				return nil, errors.New("仅允许 http/https")
			}
			if parsed.Host == "" {
				return nil, errors.New("url 必须包含 host")
			}
			if ctx.DB == nil {
				return nil, errors.New("DB 不可用")
			}
			shortURL, err := ctx.DB.CreateShortURL(rawURL, 48, 200, ctx.IP)
			if err != nil {
				return nil, fmt.Errorf("创建短链失败: %v", err)
			}
			return gin.H{
				"id":         shortURL.ID,
				"short_url":  buildExternalURL(ctx, "/s/"+shortURL.ID),
				"expires_at": shortURL.ExpiresAt,
				"max_clicks": 200,
			}, nil
		},
	}
}

// -------- paste_upload_init --------
// paste_upload_init: 初始化一个分片上传会话,返回 file_id。
// 客户端后续:paste_upload_chunk (file_id, chunk_index, data_b64) × N,最后 paste_upload_merge (file_id)。
// 与 /api/paste 的 chunk upload 共享存储;content 大小受 config.paste.max_file_size 限制(默认 50MB)。
func pasteUploadInitSkill() Skill {
	return Skill{
		Name:        "paste_upload_init",
		Description: "初始化一个 paste 文件分片上传会话,返回 file_id。后续通过 paste_upload_chunk 上传分片,最后 paste_upload_merge 合并得到最终文件(file_id 可在 paste_create 的 file_ids 中引用)。共享 /api/paste 的存储路径与 max_file_size 限制(默认 50MB)。",
		Risk:        "write",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_name":    map[string]any{"type": "string", "description": "原始文件名,如 hello.png;决定最终扩展名"},
				"file_size":    map[string]any{"type": "integer", "description": "文件总字节数,服务端会校验不超过 config.paste.max_file_size"},
				"chunk_size":   map[string]any{"type": "integer", "description": "每个分片字节数,客户端按此切分"},
				"total_chunks": map[string]any{"type": "integer", "description": "总分片数"},
			},
			"required":     []string{"file_name", "file_size", "chunk_size", "total_chunks"},
			"additionalProperties": false,
		},
		Invoke: func(args map[string]any, ctx *SkillContext) (any, error) {
			if ctx.PasteHandler == nil {
				return nil, errors.New("PasteHandler 未注入,paste_upload_init 不可用")
			}
			fileName := stringArg(args, "file_name")
			if fileName == "" {
				return nil, errors.New("file_name 不能为空")
			}
			fileSize := int64(numArgOrZero(args, "file_size"))
			chunkSize := int64(numArgOrZero(args, "chunk_size"))
			totalChunks := intArg(args, "total_chunks")
			if totalChunks == 0 {
				// JSON-RPC 数字默认走 float64,intArg 已能处理;此处再保一道
				if v, ok := args["total_chunks"].(float64); ok {
					totalChunks = int(v)
				}
			}
			fileID, err := ctx.PasteHandler.initChunkUploadCore(fileName, fileSize, chunkSize, totalChunks)
			if err != nil {
				return nil, err
			}
			return gin.H{
				"file_id":     fileID,
				"chunk_size":  chunkSize,
				"total_chunks": totalChunks,
				"message":     "分片上传初始化成功,后续按 chunk_index 0..N-1 上传",
			}, nil
		},
	}
}

// numArgOrZero 读 float64;缺省返 0,类型不对返 0(用于 init 的可数字参数)
func numArgOrZero(args map[string]any, key string) float64 {
	v, ok := args[key]
	if !ok {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return f
		}
	}
	return 0
}

// -------- paste_upload_chunk --------
func pasteUploadChunkSkill() Skill {
	return Skill{
		Name:        "paste_upload_chunk",
		Description: "上传 paste 分片。data 接受 base64 编码的二进制分片内容(标准 base64 即可,不要 base64url)。返回已上传分片数 / 总分片数。",
		Risk:        "write",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_id":     map[string]any{"type": "string", "description": "paste_upload_init 返回的 file_id"},
				"chunk_index": map[string]any{"type": "integer", "description": "分片序号,从 0 开始"},
				"data_b64":    map[string]any{"type": "string", "description": "base64 编码的分片字节;空串或缺字段视为空分片"},
			},
			"required":     []string{"file_id", "chunk_index", "data_b64"},
			"additionalProperties": false,
		},
		Invoke: func(args map[string]any, ctx *SkillContext) (any, error) {
			if ctx.PasteHandler == nil {
				return nil, errors.New("PasteHandler 未注入,paste_upload_chunk 不可用")
			}
			fileID := stringArg(args, "file_id")
			if fileID == "" {
				return nil, errors.New("file_id 不能为空")
			}
			chunkIndex := intArg(args, "chunk_index")
			dataB64 := stringArg(args, "data_b64")
			if dataB64 == "" {
				return nil, errors.New("data_b64 不能为空")
			}
			data, err := base64.StdEncoding.DecodeString(dataB64)
			if err != nil {
				// 试一下 base64url(MIME 格式常见)
				if d2, err2 := base64.URLEncoding.DecodeString(dataB64); err2 == nil {
					data = d2
				} else {
					return nil, fmt.Errorf("data_b64 不是合法 base64: %v", err)
				}
			}
			uploaded, total, err := ctx.PasteHandler.uploadChunkCore(fileID, chunkIndex, data)
			if err != nil {
				return nil, err
			}
			return gin.H{
				"uploaded_chunks": uploaded,
				"total_chunks":    total,
				"message":         "分片上传成功",
			}, nil
		},
	}
}

// -------- paste_upload_merge --------
func pasteUploadMergeSkill() Skill {
	return Skill{
		Name:        "paste_upload_merge",
		Description: "合并 paste 分片为最终文件,返回文件元信息(filename / url / type / size)。支持图片/视频/音频自动识别,其它类型透明打成 zip。",
		Risk:        "write",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_id": map[string]any{"type": "string", "description": "paste_upload_init 返回的 file_id"},
			},
			"required":     []string{"file_id"},
			"additionalProperties": false,
		},
		Invoke: func(args map[string]any, ctx *SkillContext) (any, error) {
			if ctx.PasteHandler == nil {
				return nil, errors.New("PasteHandler 未注入,paste_upload_merge 不可用")
			}
			fileID := stringArg(args, "file_id")
			if fileID == "" {
				return nil, errors.New("file_id 不能为空")
			}
			fm, err := ctx.PasteHandler.mergeChunksCore(fileID)
			if err != nil {
				return nil, err
			}
			resp := gin.H{
				"id":            fm.Filename,
				"filename":      fm.Filename,
				"original_name": fm.OriginalName,
				"type":          fm.Type,
				"size":          fm.Size,
				"url":           fm.URL,
			}
			if fm.Type == "archive" {
				resp["zipped"] = true
			}
			return resp, nil
		},
	}
}

// -------- paste_create --------
func pasteCreateSkill() Skill {
	return Skill{
		Name:        "paste_create",
		Description: "创建一段文本/代码 paste 并返回 ID 与分享 URL。完整支持 title / language / password / expires_in(小时,1-168,默认 24) / max_views(1-1000,管理员可 999999) / admin_password(后台设置后允许更大配额) / file_ids(预先通过 paste_upload_init/chunk/merge 上传得到的 file_id 列表,可关联图片/视频/文件)。当 /api/paste handler 已注入(默认部署形态)时行为与之完全对齐。",
		Risk:        "write",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content":        map[string]any{"type": "string", "description": "粘贴内容,≤ 100KB(管理员密码可放宽)"},
				"title":          map[string]any{"type": "string", "description": "可选标题,≤ 200 字符"},
				"language":       map[string]any{"type": "string", "description": "可选语言标识,如 javascript/json/text/markdown;留空则自动检测"},
				"password":       map[string]any{"type": "string", "description": "可选查看密码;设置后查看 /api/paste/:id 时需要 ?password=xxx"},
				"expires_in":     map[string]any{"type": "integer", "description": "过期小时数,1-168(7 天),默认 24"},
				"max_views":      map[string]any{"type": "integer", "description": "最大查看次数,1-1000(非管理员),默认 100;管理员密码模式可设 999999"},
				"admin_password": map[string]any{"type": "string", "description": "可选管理员密码(后端 config.paste.admin_password 设置后生效);通过后可设更高 max_views"},
				"file_ids":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "可选附件 file_id 列表,需先用 paste_upload_init/chunk/merge 三步把文件上传完,合并后即可在 paste 中引用(图片/视频/音频/任意文件)"},
			},
			"required": []string{"content"},
			"additionalProperties": false,
		},
		Invoke: func(args map[string]any, ctx *SkillContext) (any, error) {
			content := stringArg(args, "content")
			if content == "" {
				return nil, errors.New("content 不能为空")
			}

			// 优先走 PasteHandler.createPasteCore:跟 /api/paste 行为 100% 对齐
			//(XSS / 安全扫描 / 语言自动检测 / 密码 hash / admin / 默认值 / 附件等全套)
			if ctx.PasteHandler != nil && ctx.RequestCtx != nil {
				req := CreatePasteRequest{
					Content:       content,
					Title:         stringArg(args, "title"),
					Language:      strings.TrimSpace(stringArg(args, "language")),
					Password:      stringArg(args, "password"),
					ExpiresIn:     intArg(args, "expires_in"),
					MaxViews:      intArg(args, "max_views"),
					AdminPassword: stringArg(args, "admin_password"),
					FileIDs:       stringSliceArg(args, "file_ids"),
				}
				paste, err := ctx.PasteHandler.createPasteCore(ctx.RequestCtx, req, ctx.IP, createPasteOptions{
					SkipFiles:     false, // 允许附加 file_ids 引用的上传文件
					SkipRateLimit: true, // skill 层自己走 SkillsGuard 的 5/min/IP 专项限流,handler 内层不再叠加
				})
				if err != nil {
					return nil, err
				}
				fileCount := 0
				if paste.Files != "" {
					var fs []*FileMetadata
					if err := json.Unmarshal([]byte(paste.Files), &fs); err == nil {
						fileCount = len(fs)
					}
				}
				return gin.H{
					"id":           paste.ID,
					"url":          buildExternalURL(ctx, "/paste/"+paste.ID),
					"expires_at":   paste.ExpiresAt,
					"max_views":    paste.MaxViews,
					"has_password": paste.Password != "",
					"file_count":   fileCount,
				}, nil
			}

			// 兜底:PasteHandler 未注入时走瘦壳实现(老单元测试 / 异常部署形态)
			if ctx.DB == nil {
				return nil, errors.New("DB 不可用")
			}
			if len(content) > 8*1024 {
				return nil, errors.New("content 长度超过 8KB 限制")
			}
			language := strings.TrimSpace(stringArg(args, "language"))
			if language == "" {
				language = "text"
			}
			paste := &models.Paste{
				Content:   content,
				Language:  language,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
				MaxViews:  10,
				CreatorIP: ctx.IP,
			}
			if err := ctx.DB.CreatePaste(paste); err != nil {
				return nil, fmt.Errorf("创建 paste 失败: %v", err)
			}
			return gin.H{
				"id":         paste.ID,
				"url":        buildExternalURL(ctx, "/paste/"+paste.ID),
				"expires_at": paste.ExpiresAt,
				"max_views":  10,
			}, nil
		},
	}
}


// ============================================================
// Argument helpers
// ============================================================

func stringArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// intArg 读取 int 参数;缺省或类型不匹配返回 0(skill 业务里 0 通常代表"未指定/走默认值")
func intArg(args map[string]any, key string) int {
	v, ok := args[key]
	if !ok {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case json.Number:
		if f, err := x.Int64(); err == nil {
			return int(f)
		}
	}
	return 0
}

// stringSliceArg 读取字符串数组参数(JSON-RPC 透传 []any,容错 nil/单值)
func stringSliceArg(args map[string]any, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		// 兼容逗号分隔:"abc,def"
		if x == "" {
			return nil
		}
		parts := strings.Split(x, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return nil
}

func numArg(args map[string]any, key string) (float64, error) {
	v, ok := args[key]
	if !ok {
		return 0, errors.New("缺少参数: " + key)
	}
	switch x := v.(type) {
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case json.Number:
		f, err := x.Float64()
		return f, err
	}
	return 0, errors.New("参数类型错误,期望 number: " + key)
}

// schemeFrom 探测请求的 scheme:优先 X-Forwarded-Proto(反代场景),其次看 c.Request.TLS
func schemeFrom(c *gin.Context) string {
	if c.GetHeader("X-Forwarded-Proto") == "https" {
		return "https"
	}
	if c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

// buildExternalURL 用 ctx 的 host + scheme 拼出对外 URL(用于 shorturl / paste 返回)
func buildExternalURL(ctx *SkillContext, path string) string {
	scheme := ctx.Scheme
	if scheme == "" {
		scheme = "http"
	}
	return scheme + "://" + ctx.Host + path
}
