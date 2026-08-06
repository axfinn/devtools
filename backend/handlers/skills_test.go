package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 纯单元测试 —— 不依赖 DB / Network,对每个 skill 的核心路径跑 happy path + 失败用例
// 设计原则:不依赖 models.NewDB(":memory:") 初始化(那会触发 InitAll + 路径差异),
// 改用直接调 Skill.Invoke 跳过 HTTP 入口与 DB
// ============================================================

func TestSkillsManifest(t *testing.T) {
	h := NewSkillsHandler(nil)
	if len(h.tools) != 4 {
		t.Fatalf("expected exactly 4 skills (短链/粘贴/IP/DNS), got %d", len(h.tools))
	}

	// 所有 skill 必须有 OpenAI/Claude 双格式 schema,且 additionalProperties=false
	for _, s := range h.tools {
		if s.Name == "" {
			t.Errorf("skill missing name")
		}
		if s.Description == "" {
			t.Errorf("skill %s missing description", s.Name)
		}
		if s.InputSchema == nil {
			t.Errorf("skill %s missing InputSchema", s.Name)
			continue
		}
		if s.InputSchema["type"] != "object" {
			t.Errorf("skill %s schema type != object", s.Name)
		}
		if s.InputSchema["additionalProperties"] != false {
			t.Errorf("skill %s schema must have additionalProperties:false", s.Name)
		}
		if s.Risk != "compute" && s.Risk != "write" {
			t.Errorf("skill %s risk must be compute|write, got %q", s.Name, s.Risk)
		}
		if s.Invoke == nil {
			t.Errorf("skill %s missing Invoke", s.Name)
		}
	}

	// 必须包含这 4 个"只有服务端能做"的 skill
	names := map[string]bool{}
	for _, s := range h.tools {
		names[s.Name] = true
	}
	for _, required := range []string{"shorturl_create", "paste_create", "ip_lookup", "dns_lookup"} {
		if !names[required] {
			t.Errorf("missing required skill %q", required)
		}
	}

	// 测试 toolMetadata 输出形态 —— 必须严格遵守 MCP 2025-06-18 规范
	meta := h.toolMetadata()
	if len(meta) != len(h.tools) {
		t.Fatalf("toolMetadata count mismatch")
	}
	for _, m := range meta {
		// MCP 规范字段:camelCase inputSchema
		if _, ok := m["inputSchema"]; !ok {
			t.Errorf("missing inputSchema in toolMetadata (MCP 规范字段,camelCase)")
		}
		// 旧 snake_case 字段应不再出现
		if _, ok := m["input_schema"]; ok {
			t.Errorf("toolMetadata 不应再含 input_schema(snake_case),客户端按规范读 inputSchema 会读到 undefined")
		}
		// 非标准的 risk 字段不应出现在对外 wire format
		if _, ok := m["risk"]; ok {
			t.Errorf("toolMetadata 不应含 risk 字段(非 MCP 规范)")
		}
		// parameters 也不应在 MCP tools/list 响应里(那是 OpenAI 工具格式)
		if _, ok := m["parameters"]; ok {
			t.Errorf("toolMetadata 不应含 parameters(MCP 规范无此字段,见 openAIToolsMetadata)")
		}
		// 校验能 JSON 序列化
		b, err := json.Marshal(m)
		if err != nil {
			t.Errorf("manifest item marshal: %v", err)
		}
		if len(b) == 0 {
			t.Errorf("manifest item empty marshal")
		}
	}
}

// 测试 openAIToolsMetadata 输出形态 —— OpenAI tools 风格 + 跨平台兼容
func TestSkillsOpenAIToolsMetadata(t *testing.T) {
	h := NewSkillsHandler(nil)
	meta := h.openAIToolsMetadata()
	if len(meta) == 0 {
		t.Fatal("openAIToolsMetadata 空")
	}
	for _, m := range meta {
		if _, ok := m["parameters"]; !ok {
			t.Errorf("openAIToolsMetadata 应含 parameters(OpenAI 标准)")
		}
		if _, ok := m["inputSchema"]; !ok {
			t.Errorf("openAIToolsMetadata 也应含 inputSchema(Claude 跨平台)")
		}
	}
}

func TestSkillsComputeHappy(t *testing.T) {
	h := NewSkillsHandler(nil)
	ctx := &SkillContext{IP: "1.2.3.4", Host: "example.com", Scheme: "https"}

	tests := []struct {
		skillName string
		args      map[string]any
		check     func(t *testing.T, got any)
	}{
		{
			skillName: "ip_lookup",
			args:      map[string]any{},
			check: func(t *testing.T, got any) {
				m := got.(gin.H)
				if m["ip"] != "1.2.3.4" {
					t.Errorf("ip_lookup 应回 ctx.IP, got %v", m)
				}
			},
		},
		{
			// dns_lookup 是 compute 类,允许在沙箱里 mock net.Resolver 或依赖真实 DNS。
			// 这里只验证 "非黑名单域名能进 invoke 链不 panic",不验证具体结果。
			skillName: "dns_lookup",
			args:      map[string]any{"domain": "example.com", "type": "A"},
			check: func(t *testing.T, got any) {
				// 真实 DNS 解析可能成功可能失败(网络隔离),只校验返回结构
				_ = got
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.skillName, func(t *testing.T) {
			skill, ok := h.toolByName(tc.skillName)
			if !ok {
				t.Fatalf("skill %s 未注册", tc.skillName)
			}
			got, err := skill.Invoke(tc.args, ctx)
			if err != nil {
				// dns_lookup 在测试环境无网络时允许失败
				if tc.skillName == "dns_lookup" {
					t.Skipf("dns_lookup 在无网络环境跳过: %v", err)
				}
				t.Fatalf("Invoke 错: %v", err)
			}
			tc.check(t, got)
		})
	}
}

func TestSkillsComputeFailures(t *testing.T) {
	h := NewSkillsHandler(nil)
	ctx := &SkillContext{IP: "1.2.3.4", Host: "example.com", Scheme: "https"}

	tests := []struct {
		skillName string
		args      map[string]any
		wantSub   string
	}{
		// dns_lookup 必须拒内网(host 黑名单)
		{"dns_lookup", map[string]any{"domain": "localhost"}, "内网"},
		{"dns_lookup", map[string]any{"domain": "127.0.0.1"}, "内网"},
		{"dns_lookup", map[string]any{"domain": "10.0.0.1"}, "内网"},
		{"dns_lookup", map[string]any{"domain": "192.168.1.1"}, "内网"},
		// ip_lookup 不需要参数,空 args 也应能跑
		{"ip_lookup", map[string]any{}, ""},
	}

	for i, tc := range tests {
		t.Run(tc.skillName+"-"+string(rune('0'+i)), func(t *testing.T) {
			skill, _ := h.toolByName(tc.skillName)
			_, err := skill.Invoke(tc.args, ctx)
			if tc.wantSub == "" {
				// ip_lookup 这种不需要参数的应该成功
				if err != nil {
					t.Fatalf("expect success, got err: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expect error containing %q, got nil", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("err = %v, want sub %q", err, tc.wantSub)
			}
		})
	}
}

func TestHostBlacklisted(t *testing.T) {
	cases := []struct {
		host      string
		blacklist bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"foo.localhost", true},
		{"foo.local", true},
		{"foo.internal", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"172.16.0.1", true},
		{"169.254.1.1", true},
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		{"8.8.8.8", false},
		{"example.com", false},
		{"1.1.1.1", false},
		{"", true},
	}
	for _, c := range cases {
		got := hostBlacklisted(c.host)
		if got != c.blacklist {
			t.Errorf("hostBlacklisted(%q) = %v, want %v", c.host, got, c.blacklist)
		}
	}
}

func TestDNSLookupRejectsInternal(t *testing.T) {
	h := NewSkillsHandler(nil)
	skill, _ := h.toolByName("dns_lookup")
	ctx := &SkillContext{IP: "1.2.3.4"}
	_, err := skill.Invoke(map[string]any{"domain": "localhost"}, ctx)
	if err == nil || !strings.Contains(err.Error(), "拒绝解析") {
		t.Fatalf("dns localhost 应该拒绝, got %v", err)
	}

	_, err = skill.Invoke(map[string]any{"domain": "127.0.0.1"}, ctx)
	if err == nil || !strings.Contains(err.Error(), "拒绝解析") {
		t.Fatalf("dns 127.0.0.1 应该拒绝, got %v", err)
	}
}

func TestSkillsInvoke_ParseError(t *testing.T) {
	// 用 gin.Engine + Recorder 直接打 routes,验 JSON 错误流
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.POST("/api/skills/invoke", h.Invoke)

	// 1. 错误 JSON
	w := doRequest(r, "POST", "/api/skills/invoke", "{not json")
	if w.Code != 400 {
		t.Fatalf("expect 400 on bad json, got %d", w.Code)
	}

	// 2. 缺 name
	w = doRequest(r, "POST", "/api/skills/invoke", `{"arguments":{}}`)
	if w.Code != 400 {
		t.Fatalf("expect 400 missing name, got %d", w.Code)
	}

	// 3. 不存在的 skill
	w = doRequest(r, "POST", "/api/skills/invoke", `{"name":"doesnt_exist","arguments":{}}`)
	if w.Code != 404 {
		t.Fatalf("expect 404 unknown, got %d", w.Code)
	}

	// 4. OK 路径(ip_lookup 不需参数)
	w = doRequest(r, "POST", "/api/skills/invoke", `{"name":"ip_lookup","arguments":{}}`)
	if w.Code != 200 {
		t.Fatalf("expect 200 ip_lookup, got %d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Fatalf("uuid_v4 should ok:true, body=%s", w.Body.String())
	}
}

func TestSkillsMCP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.POST("/api/skills/mcp", h.MCPPostHandler)

	// initialize
	w := doRequest(r, "POST", "/api/skills/mcp", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if w.Code != 200 {
		t.Fatalf("initialize HTTP %d", w.Code)
	}
	var initResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &initResp)
	if initResp["jsonrpc"] != "2.0" {
		t.Errorf("mcp 缺 jsonrpc 字段,body=%s", w.Body.String())
	}
	if result, ok := initResp["result"].(map[string]any); ok {
		if _, has := result["serverInfo"]; !has {
			t.Errorf("initialize 缺 serverInfo")
		}
		if _, has := result["capabilities"]; !has {
			t.Errorf("initialize 缺 capabilities")
		}
	}

	// tools/list —— 必须严格遵守 MCP 2025-06-18 规范
	w = doRequest(r, "POST", "/api/skills/mcp", `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if w.Code != 200 {
		t.Fatalf("tools/list HTTP %d", w.Code)
	}
	var listResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	result, ok := listResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/list 缺 result, body=%s", w.Body.String())
	}
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("tools/list 缺 tools 数组, body=%s", w.Body.String())
	}
	// 校验每个 tool 都符合 MCP 规范字段
	for i, toolAny := range tools {
		tool, _ := toolAny.(map[string]any)
		if _, has := tool["name"]; !has {
			t.Errorf("tools[%d] 缺 name", i)
		}
		if _, has := tool["inputSchema"]; !has {
			t.Errorf("tools[%d] 缺 inputSchema(MCP 规范要求 camelCase)", i)
		}
		// 这些字段不应出现在 MCP tools/list 响应里
		if _, has := tool["input_schema"]; has {
			t.Errorf("tools[%d] 不应含 input_schema(snake_case),client 读 inputSchema 会失败", i)
		}
		if _, has := tool["risk"]; has {
			t.Errorf("tools[%d] 不应含 risk 字段(非规范)", i)
		}
		if _, has := tool["parameters"]; has {
			t.Errorf("tools[%d] 不应含 parameters(MCP 规范无此字段)", i)
		}
	}

	// tools/call ip_lookup(不需参数)
	w = doRequest(r, "POST", "/api/skills/mcp",
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ip_lookup","arguments":{}}}`)
	if w.Code != 200 {
		t.Fatalf("tools/call HTTP %d", w.Code)
	}
	var callResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &callResp)
	callResult, _ := callResp["result"].(map[string]any)
	if content, _ := callResult["content"].([]any); len(content) == 0 {
		t.Errorf("tools/call 内容缺失")
	}

	// 未知 method
	w = doRequest(r, "POST", "/api/skills/mcp", `{"jsonrpc":"2.0","id":4,"method":"unknown/method"}`)
	if w.Code != 200 {
		t.Fatalf("unknown method 应仍 200(JSON-RPC 错误 in body), got %d", w.Code)
	}
	var errResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if err, ok := errResp["error"].(map[string]any); !ok {
		t.Errorf("unknown method 应返回 error 字段, got %s", w.Body.String())
	} else if err["code"].(float64) != -32601 {
		t.Errorf("error code 应为 -32601, got %v", err)
	}
}

func TestNegotiateMCPVersion(t *testing.T) {
	// client 发白名单内的版本 -> echo 回去
	for _, v := range []string{"2024-11-05", "2025-03-26", "2025-06-18"} {
		if got := negotiateMCPVersion(v); got != v {
			t.Errorf("client 发 %q 应 echo 回去, got %q", v, got)
		}
	}
	// 未知 / 拼错 / 不存在 的日期 -> 回最新
	for _, bad := range []string{"2025-05-06", "2025-99-99", "1.0", ""} {
		got := negotiateMCPVersion(bad)
		if got != "2025-06-18" {
			t.Errorf("client 发 %q 应回 2025-06-18, got %q", bad, got)
		}
	}
}

func TestSkillsMCP_InitializeVersionNegotiation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.POST("/api/skills/mcp", h.MCPPostHandler)

	// client 声明 2025-03-26 -> server echo 回去(不能擅自升到 2025-06-18)
	w := doRequest(r, "POST", "/api/skills/mcp",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"claude-code"}}}`)
	if w.Code != 200 {
		t.Fatalf("HTTP %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if result, ok := resp["result"].(map[string]any); !ok {
		t.Fatalf("缺 result, body=%s", w.Body.String())
	} else if got := result["protocolVersion"]; got != "2025-03-26" {
		t.Errorf("client 发 2025-03-26 应 echo, got %v", got)
	}

	// client 发 clientCodex 不认的 2025-05-06 -> server 回最新 2025-06-18
	w = doRequest(r, "POST", "/api/skills/mcp",
		`{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2025-05-06"}}`)
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if got := resp["result"].(map[string]any)["protocolVersion"]; got != "2025-06-18" {
		t.Errorf("client 发 2025-05-06 应回 2025-06-18, got %v", got)
	}

	// client 没发 protocolVersion(params 空) -> server 回最新
	w = doRequest(r, "POST", "/api/skills/mcp",
		`{"jsonrpc":"2.0","id":3,"method":"initialize","params":{}}`)
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if got := resp["result"].(map[string]any)["protocolVersion"]; got != "2025-06-18" {
		t.Errorf("client 未发 version 应回 2025-06-18, got %v", got)
	}

	// 真实场景:Claude Code 当前发 2025-06-18 -> 应该 echo 回去(不退化)
	w = doRequest(r, "POST", "/api/skills/mcp",
		`{"jsonrpc":"2.0","id":4,"method":"initialize","params":{"protocolVersion":"2025-06-18","clientInfo":{"name":"claude-code","version":"2.0.0"}}}`)
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if got := resp["result"].(map[string]any)["protocolVersion"]; got != "2025-06-18" {
		t.Errorf("Claude Code 2025-06-18 应 echo, got %v", got)
	}
}

func TestSkillsMCP_SSEResponseMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.POST("/api/skills/mcp", h.MCPPostHandler)

	// Accept 头声明 SSE -> 响应走 text/event-stream,内容是 event: message + data: ...
	req, _ := http.NewRequest("POST", "/api/skills/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("HTTP %d body=%s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("应 text/event-stream, got %q", ct)
	}
	body := w.Body.String()
	if !strings.HasPrefix(body, "event: message\n") {
		t.Errorf("SSE 帧应以 event: message 开头, got %q", body)
	}
	if !strings.Contains(body, `"tools":`) {
		t.Errorf("SSE data 内缺 tools, body=%q", body)
	}
	if !strings.HasSuffix(body, "\n\n") {
		t.Errorf("SSE 帧应以 \\n\\n 结尾, got %q", body)
	}

	// 不带 Accept: text/event-stream -> 回到 JSON
	req2, _ := http.NewRequest("POST", "/api/skills/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"ping"}`))
	req2.Header.Set("Content-Type", "application/json")
	// 故意不设 Accept
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if ct := w2.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("无 SSE Accept 时应 application/json, got %q", ct)
	}
}

func TestAcceptsEventStream(t *testing.T) {
	cases := []struct {
		accept string
		want   bool
	}{
		{"", false},
		{"application/json", false},
		{"application/json, text/event-stream", true},
		{"text/event-stream", true},
		{"TEXT/EVENT-STREAM", true}, // case-insensitive
		{"application/json;q=0.9, text/event-stream;q=1.0", true},
	}
	for _, c := range cases {
		if got := acceptsEventStream(c.accept); got != c.want {
			t.Errorf("acceptsEventStream(%q) = %v, want %v", c.accept, got, c.want)
		}
	}
}

func TestSkillsInstall_AllClients(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.GET("/api/skills/install", h.GetInstall)

	w := doRequest(r, "GET", "/api/skills/install", "")
	if w.Code != 200 {
		t.Fatalf("HTTP %d body=%s", w.Code, w.Body.String())
	}
	// 默认应 text/plain
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("默认应 text/plain, got %q", ct)
	}
	body := w.Body.String()
	// 关键内容:每个客户端的 one_liner 应出现在文本里
	for _, want := range []string{"claude mcp add", "codex mcp add", "curl -s"} {
		if !strings.Contains(body, want) {
			t.Errorf("install text 缺 %q", want)
		}
	}
	// 不应是裸 JSON 响应(应是 README 风格)
	if !strings.Contains(body, "DevTools Skills — 一键安装") {
		t.Errorf("缺 README 标题, got: %.100s...", body)
	}
	// 应有结构化分节
	if !strings.Contains(body, "## claude_code") {
		t.Errorf("缺 ## claude_code 分节, got: %.100s...", body)
	}
}

func TestSkillsInstall_FormatJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.GET("/api/skills/install", h.GetInstall)

	w := doRequest(r, "GET", "/api/skills/install?format=json", "")
	if w.Code != 200 {
		t.Fatalf("HTTP %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("?format=json 应 application/json, got %q", ct)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	clients, ok := resp["clients"].(map[string]any)
	if !ok {
		t.Fatalf("缺 clients, body=%s", w.Body.String())
	}
	for _, k := range []string{"claude_code", "codex", "cursor", "vscode", "continue", "curl"} {
		if _, ok := clients[k]; !ok {
			t.Errorf("缺客户端 %s", k)
		}
	}
}

func TestSkillsInstall_SingleClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.GET("/api/skills/install", h.GetInstall)

	// 单客户端 + 默认 text
	w := doRequest(r, "GET", "/api/skills/install?client=claude_code", "")
	if w.Code != 200 {
		t.Fatalf("HTTP %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("单客户端应 text/plain, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "claude mcp add") {
		t.Errorf("claude_code 应含 claude mcp add, got %s", w.Body.String())
	}

	// 单客户端 + format=json
	w = doRequest(r, "GET", "/api/skills/install?client=claude_code&format=json", "")
	if w.Code != 200 {
		t.Fatalf("HTTP %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["client"] != "claude_code" {
		t.Errorf("client 字段错, got %v", resp["client"])
	}
}

func TestSkillsInstall_UnknownClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.GET("/api/skills/install", h.GetInstall)

	w := doRequest(r, "GET", "/api/skills/install?client=emacs", "")
	if w.Code != 400 {
		t.Errorf("未知 client 应 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestSkillsInstall_ShellScript(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.GET("/api/skills/install.sh", h.GetInstallShell)

	w := doRequest(r, "GET", "/api/skills/install.sh", "")
	if w.Code != 200 {
		t.Fatalf("HTTP %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/x-shellscript") {
		t.Errorf("应 text/x-shellscript, got %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "devtools-skills-install.sh") {
		t.Errorf("应触发下载, got Content-Disposition=%q", cd)
	}
	body := w.Body.String()
	if !strings.HasPrefix(body, "#!/usr/bin/env bash") {
		t.Errorf("shell 脚本应有 shebang, got %q", strings.SplitN(body, "\n", 2)[0])
	}
	// 关键命令应作为注释被包进去
	for _, want := range []string{"claude mcp add", "codex mcp add", "curl -sX POST"} {
		if !strings.Contains(body, want) {
			t.Errorf("shell 脚本缺 %q", want)
		}
	}
}

func TestSkillsInstall_WellKnownDirectory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.GET("/.well-known/skills", h.GetDirectory)

	w := doRequest(r, "GET", "/.well-known/skills", "")
	if w.Code != 200 {
		t.Fatalf("HTTP %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["server"] == nil {
		t.Errorf("缺 server 字段, body=%s", w.Body.String())
	}
	endpoints, _ := resp["endpoints"].(map[string]any)
	for _, want := range []string{"manifest", "mcp", "invoke", "install_text", "install_shell"} {
		if endpoints[want] == nil {
			t.Errorf("endpoints 缺 %s", want)
		}
	}
	clients, _ := resp["clients"].([]any)
	if len(clients) == 0 {
		t.Errorf("clients 应非空, body=%s", w.Body.String())
	}
	// skills 数组应包含 4 个(只留"只有服务端能做"的)
	skills, _ := resp["skills"].([]any)
	if len(skills) != 4 {
		t.Errorf("skills 应 == 4, got %d", len(skills))
	}
}

func TestSkillsInstall_OnlyFourSkills(t *testing.T) {
	// 强约束:manifest/install 暴露的 skill 必须恰好是 4 个"只有服务端能做"的,
	// 否则会回退到之前 16 个里混了本地能算的(bug)。
	gin.SetMode(gin.TestMode)
	h := NewSkillsHandler(nil)
	r := gin.New()
	r.GET("/api/skills/manifest", h.GetManifest)

	w := doRequest(r, "GET", "/api/skills/manifest", "")
	if w.Code != 200 {
		t.Fatalf("HTTP %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	tools, _ := resp["tools"].([]any)
	if len(tools) != 4 {
		t.Fatalf("manifest 应有 4 个 skill, got %d", len(tools))
	}
	names := map[string]bool{}
	for _, t := range tools {
		if m, ok := t.(map[string]any); ok {
			names[m["name"].(string)] = true
		}
	}
	for _, required := range []string{"shorturl_create", "paste_create", "ip_lookup", "dns_lookup"} {
		if !names[required] {
			t.Errorf("manifest 缺 %s", required)
		}
	}
	// 强约束:不能包含之前误加的"本地能算"的
	for _, banned := range []string{"base64_encode", "base64_decode", "uuid_v4", "hash_sha256", "json_format", "regex_test", "install_info"} {
		if names[banned] {
			t.Errorf("manifest 不应含 %s(本地能算 / 已删除)", banned)
		}
	}
}
