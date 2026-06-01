package handlers

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

// ============================================================
// Shadowrocket 订阅解析单元测试
// ============================================================

func TestParseSSURL(t *testing.T) {
	// ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTp0ZXN0cGFzc3dvcmQ=@1.2.3.4:8388#TestNode
	methodPw := base64.RawStdEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:testpassword"))
	uri := "ss://" + methodPw + "@1.2.3.4:8388#Test%20Node"
	node, err := parseSSURL(uri)
	if err != nil {
		t.Fatalf("parseSSURL failed: %v", err)
	}
	if node.Type != "ss" {
		t.Errorf("type = %s, want ss", node.Type)
	}
	if node.Server != "1.2.3.4" {
		t.Errorf("server = %s, want 1.2.3.4", node.Server)
	}
	if node.Port != 8388 {
		t.Errorf("port = %d, want 8388", node.Port)
	}
	if node.Name != "Test Node" {
		t.Errorf("name = %s, want Test Node", node.Name)
	}
	if node.Extra["cipher"] != "chacha20-ietf-poly1305" {
		t.Errorf("cipher = %v", node.Extra["cipher"])
	}
	if node.Extra["password"] != "testpassword" {
		t.Errorf("password = %v", node.Extra["password"])
	}
}

func TestParseSSURLWithoutName(t *testing.T) {
	methodPw := base64.RawStdEncoding.EncodeToString([]byte("aes-256-gcm:mypassword"))
	uri := "ss://" + methodPw + "@10.0.0.1:443"
	node, err := parseSSURL(uri)
	if err != nil {
		t.Fatalf("parseSSURL failed: %v", err)
	}
	if !strings.HasPrefix(node.Name, "SS ") {
		t.Errorf("auto-generated name should start with 'SS ', got %q", node.Name)
	}
}

func TestParseTrojanURL(t *testing.T) {
	uri := "trojan://mypassword@trojan.example.com:443?security=tls&type=tcp#My%20Trojan"
	node, err := parseTrojanURL(uri)
	if err != nil {
		t.Fatalf("parseTrojanURL failed: %v", err)
	}
	if node.Type != "trojan" {
		t.Errorf("type = %s, want trojan", node.Type)
	}
	if node.Server != "trojan.example.com" {
		t.Errorf("server = %s", node.Server)
	}
	if node.Port != 443 {
		t.Errorf("port = %d, want 443", node.Port)
	}
	if node.Extra["password"] != "mypassword" {
		t.Errorf("password = %v", node.Extra["password"])
	}
	if node.Extra["security"] != "tls" {
		t.Errorf("security = %v", node.Extra["security"])
	}
	if node.Name != "My Trojan" {
		t.Errorf("name = %s, want My Trojan", node.Name)
	}
}

func TestParseTrojanURLDefaultPort(t *testing.T) {
	uri := "trojan://pw@server.com#Node"
	node, err := parseTrojanURL(uri)
	if err != nil {
		t.Fatalf("parseTrojanURL failed: %v", err)
	}
	if node.Port != 443 {
		t.Errorf("default port = %d, want 443", node.Port)
	}
	if node.Name != "Node" {
		t.Errorf("name = %s, want Node", node.Name)
	}
}

func TestParseVlessURL(t *testing.T) {
	uri := "vless://my-uuid-1234@vless.example.com:8443?security=reality&encryption=none&type=ws&path=%2Fws#VLESS%20Node"
	node, err := parseVlessURL(uri)
	if err != nil {
		t.Fatalf("parseVlessURL failed: %v", err)
	}
	if node.Type != "vless" {
		t.Errorf("type = %s, want vless", node.Type)
	}
	if node.Server != "vless.example.com" {
		t.Errorf("server = %s", node.Server)
	}
	if node.Port != 8443 {
		t.Errorf("port = %d, want 8443", node.Port)
	}
	if node.Extra["uuid"] != "my-uuid-1234" {
		t.Errorf("uuid = %v", node.Extra["uuid"])
	}
	if node.Extra["security"] != "reality" {
		t.Errorf("security = %v", node.Extra["security"])
	}
	if node.Extra["type"] != "ws" {
		t.Errorf("type = %v", node.Extra["type"])
	}
	if node.Name != "VLESS Node" {
		t.Errorf("name = %s, want VLESS Node", node.Name)
	}
}

func TestParseVmessURL(t *testing.T) {
	// vmess://base64(json)
	cfg := `{"ps":"My VMess","add":"vmess.example.com","port":443,"id":"my-uuid","aid":0,"net":"ws","type":"none","host":"example.com","path":"/ws","tls":"tls","sni":"example.com"}`
	encoded := base64.RawStdEncoding.EncodeToString([]byte(cfg))
	uri := "vmess://" + encoded
	node, err := parseVmessURL(uri)
	if err != nil {
		t.Fatalf("parseVmessURL failed: %v", err)
	}
	if node.Type != "vmess" {
		t.Errorf("type = %s, want vmess", node.Type)
	}
	if node.Server != "vmess.example.com" {
		t.Errorf("server = %s", node.Server)
	}
	if node.Port != 443 {
		t.Errorf("port = %d, want 443", node.Port)
	}
	if node.Extra["uuid"] != "my-uuid" {
		t.Errorf("uuid = %v", node.Extra["uuid"])
	}
	if node.Extra["network"] != "ws" {
		t.Errorf("network = %v", node.Extra["network"])
	}
	if node.Name != "My VMess" {
		t.Errorf("name = %s, want My VMess", node.Name)
	}
}

func TestParseSSRURL(t *testing.T) {
	// ssr://server:port:protocol:method:obfs:password_base64/?remarks=base64remarks
	plain := "ssr.example.com:443:auth_aes128_md5:aes-128-ctr:tls1.2_ticket_auth:" +
		base64.RawStdEncoding.EncodeToString([]byte("mypassword")) +
		"/?remarks=" + base64.RawStdEncoding.EncodeToString([]byte("My SSR Node"))
	encoded := base64.RawStdEncoding.EncodeToString([]byte(plain))
	uri := "ssr://" + encoded
	node, err := parseSSRURL(uri)
	if err != nil {
		t.Fatalf("parseSSRURL failed: %v", err)
	}
	if node.Type != "ssr" {
		t.Errorf("type = %s, want ssr", node.Type)
	}
	if node.Server != "ssr.example.com" {
		t.Errorf("server = %s", node.Server)
	}
	if node.Port != 443 {
		t.Errorf("port = %d, want 443", node.Port)
	}
	if node.Extra["protocol"] != "auth_aes128_md5" {
		t.Errorf("protocol = %v", node.Extra["protocol"])
	}
	if node.Extra["cipher"] != "aes-128-ctr" {
		t.Errorf("cipher = %v", node.Extra["cipher"])
	}
	if node.Extra["obfs"] != "tls1.2_ticket_auth" {
		t.Errorf("obfs = %v", node.Extra["obfs"])
	}
	if node.Name != "My SSR Node" {
		t.Errorf("name = %s, want My SSR Node", node.Name)
	}
}

func TestParseShadowrocketSubscription(t *testing.T) {
	// 构建一个混合订阅 content（ss + trojan + vmess）
	methodPw := base64.RawStdEncoding.EncodeToString([]byte("aes-256-gcm:pass1"))
	ssLine := "ss://" + methodPw + "@1.2.3.4:8388#SS%20Node"

	trojanLine := "trojan://pw@trojan.example.com:443?security=tls#Trojan%20Node"

	vmessCfg := `{"ps":"VMess Node","add":"vm.example.com","port":443,"id":"uuid-1","aid":0,"net":"ws"}`
	vmessLine := "vmess://" + base64.RawStdEncoding.EncodeToString([]byte(vmessCfg))

	vlessLine := "vless://uuid-2@vl.example.com:443?security=reality#VLESS%20Node"

	rawContent := strings.Join([]string{ssLine, trojanLine, vmessLine, vlessLine}, "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(rawContent))

	nodes, err := parseShadowrocketSubscription(encoded)
	if err != nil {
		t.Fatalf("parseShadowrocketSubscription failed: %v", err)
	}
	if len(nodes) != 4 {
		t.Fatalf("got %d nodes, want 4", len(nodes))
	}

	wantTypes := map[string]bool{"ss": false, "trojan": false, "vmess": false, "vless": false}
	for _, n := range nodes {
		wantTypes[n.Type] = true
	}
	for typ, found := range wantTypes {
		if !found {
			t.Errorf("missing node type: %s", typ)
		}
	}
}

func TestParseShadowrocketSubscriptionEmpty(t *testing.T) {
	_, err := parseShadowrocketSubscription("")
	if err == nil {
		t.Error("should return error for empty input")
	}
}

func TestParseShadowrocketSubscriptionInvalidBase64(t *testing.T) {
	_, err := parseShadowrocketSubscription("!!!not-valid-base64!!!")
	if err == nil {
		t.Error("should return error for invalid base64")
	}
}

func TestParseShadowrocketSubscriptionSkipComments(t *testing.T) {
	methodPw := base64.RawStdEncoding.EncodeToString([]byte("aes-256-gcm:pass"))
	ssLine := "ss://" + methodPw + "@1.2.3.4:8388#Node"
	rawContent := "// This is a comment\n" + ssLine + "\n# Another comment"
	encoded := base64.StdEncoding.EncodeToString([]byte(rawContent))

	nodes, err := parseShadowrocketSubscription(encoded)
	if err != nil {
		t.Fatalf("parseShadowrocketSubscription failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1 (comments skipped)", len(nodes))
	}
}

func TestNodesToClashYAML(t *testing.T) {
	nodes := []ProxyNode{
		{Name: "Test Trojan", Type: "trojan", Server: "test.example.com", Port: 443,
			Extra: map[string]interface{}{"password": "pw", "security": "tls"}},
		{Name: "Test SS", Type: "ss", Server: "ss.example.com", Port: 8388,
			Extra: map[string]interface{}{"cipher": "aes-256-gcm", "password": "sspass"}},
	}
	yamlText, err := nodesToClashYAML(nodes)
	if err != nil {
		t.Fatalf("nodesToClashYAML failed: %v", err)
	}
	if !strings.Contains(yamlText, "proxies:") {
		t.Error("YAML should contain 'proxies:'")
	}
	if !strings.Contains(yamlText, "Test Trojan") {
		t.Error("YAML should contain 'Test Trojan'")
	}
	if !strings.Contains(yamlText, "Test SS") {
		t.Error("YAML should contain 'Test SS'")
	}
	if !strings.Contains(yamlText, "aes-256-gcm") {
		t.Error("YAML should contain cipher 'aes-256-gcm'")
	}

	// 验证 YAML 可被 parseClashYAML 解析回来
	parsedNodes, err := parseClashYAML(yamlText)
	if err != nil {
		t.Fatalf("parseClashYAML of converted output failed: %v", err)
	}
	if len(parsedNodes) != 2 {
		t.Fatalf("round-trip: got %d nodes, want 2", len(parsedNodes))
	}
	for i, pn := range parsedNodes {
		if pn.Name != nodes[i].Name {
			t.Errorf("round-trip node[%d].Name = %s, want %s", i, pn.Name, nodes[i].Name)
		}
		if pn.Type != nodes[i].Type {
			t.Errorf("round-trip node[%d].Type = %s, want %s", i, pn.Type, nodes[i].Type)
		}
	}
}

func TestNodesToClashYAMLEmpty(t *testing.T) {
	_, err := nodesToClashYAML(nil)
	if err == nil {
		t.Error("should return error for nil nodes")
	}
	_, err = nodesToClashYAML([]ProxyNode{})
	if err == nil {
		t.Error("should return error for empty nodes")
	}
}

func TestShadowrocketRoundTrip(t *testing.T) {
	// 构建一个完整的 Shadowrocket 订阅 → 解析 → YAML → 再解析
	methodPw := base64.RawStdEncoding.EncodeToString([]byte("aes-256-gcm:pass1"))
	ssLine := "ss://" + methodPw + "@1.2.3.4:8388#SS%20Node"
	trojanLine := "trojan://pw@trojan.example.com:443?security=tls#Trojan%20Node"
	rawContent := ssLine + "\n" + trojanLine
	encoded := base64.StdEncoding.EncodeToString([]byte(rawContent))

	nodes, err := parseShadowrocketSubscription(encoded)
	if err != nil {
		t.Fatalf("parseShadowrocketSubscription failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}

	yamlText, err := nodesToClashYAML(nodes)
	if err != nil {
		t.Fatalf("nodesToClashYAML failed: %v", err)
	}

	parsed, err := parseClashYAML(yamlText)
	if err != nil {
		t.Fatalf("parseClashYAML of round-trip failed: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("round-trip: got %d nodes, want 2", len(parsed))
	}

	// 验证节点信息完整保留
	typeSet := map[string]bool{}
	for _, n := range parsed {
		typeSet[n.Type] = true
		if n.Type == "ss" && n.Extra["cipher"] != "aes-256-gcm" {
			t.Errorf("ss cipher lost in round-trip: %v", n.Extra["cipher"])
		}
		if n.Type == "trojan" && n.Extra["password"] != "pw" {
			t.Errorf("trojan password lost in round-trip: %v", n.Extra["password"])
		}
	}
	if !typeSet["ss"] || !typeSet["trojan"] {
		t.Errorf("round-trip missing types: %v", typeSet)
	}
}

// TestRealSubscriptionFetch 测试从真实的 Shadowrocket 订阅 URL 获取并解析
// 此测试需要网络连接，如果订阅链接不可用会跳过
func TestParseManualProxyAddr(t *testing.T) {
	tests := []struct {
		addr       string
		wantServer string
		wantPort   int
		wantType   string
	}{
		{"127.0.0.1:7890", "127.0.0.1", 7890, "http"},
		{"192.168.1.1:1080", "192.168.1.1", 1080, "http"},
		{"10.0.0.1:8118", "10.0.0.1", 8118, "http"},
		{"localhost:7890", "localhost", 7890, "http"},
	}
	for _, tc := range tests {
		node, err := parseManualProxyAddr(tc.addr)
		if err != nil {
			t.Errorf("parseManualProxyAddr(%q) failed: %v", tc.addr, err)
			continue
		}
		if node.Server != tc.wantServer {
			t.Errorf("parseManualProxyAddr(%q).Server = %q, want %q", tc.addr, node.Server, tc.wantServer)
		}
		if node.Port != tc.wantPort {
			t.Errorf("parseManualProxyAddr(%q).Port = %d, want %d", tc.addr, node.Port, tc.wantPort)
		}
		if node.Type != tc.wantType {
			t.Errorf("parseManualProxyAddr(%q).Type = %q, want %q", tc.addr, node.Type, tc.wantType)
		}
		if !strings.HasPrefix(node.Name, "手动 ") {
			t.Errorf("parseManualProxyAddr(%q).Name = %q, want prefix '手动 '", tc.addr, node.Name)
		}
	}
}

func TestParseManualProxyAddrEmpty(t *testing.T) {
	_, err := parseManualProxyAddr("")
	if err == nil {
		t.Error("should return error for empty address")
	}
}

func TestRealSubscriptionFetch(t *testing.T) {
	rawURL := "https://port.shydx.com/link/fc078ba5a0cc6d3c2891cb74986e0106?flag=shadowrocket"

	// 获取 Shadowrocket 格式订阅
	srURL := subscriptionURLForType(rawURL, "shadowrocket")
	t.Logf("获取订阅: %s", srURL)

	text, err := fetchSubscriptionWithClientMode(newDirectHTTPClient(30*time.Second), srURL, false)
	if err != nil {
		t.Skipf("无法获取订阅（网络问题或链接失效）: %v", err)
	}
	if text == "" {
		t.Skip("订阅内容为空")
	}
	t.Logf("订阅原始大小: %d 字节", len(text))

	// 解析 Shadowrocket 订阅
	nodes, err := parseShadowrocketSubscription(text)
	if err != nil {
		t.Fatalf("解析 Shadowrocket 订阅失败: %v", err)
	}
	t.Logf("解析出 %d 个节点", len(nodes))

	if len(nodes) == 0 {
		t.Fatal("未解析到任何节点，订阅格式可能已变化")
	}

	// 验证节点完整性
	typeCount := map[string]int{}
	invalidCount := 0
	for _, n := range nodes {
		if n.Name == "" || n.Server == "" || n.Port == 0 {
			t.Logf("无效节点: name=%q server=%q port=%d type=%s", n.Name, n.Server, n.Port, n.Type)
			invalidCount++
		}
		typeCount[n.Type]++
	}
	t.Logf("类型分布: %v", typeCount)
	if invalidCount > 0 {
		t.Errorf("%d 个无效节点", invalidCount)
	}

	// 转换为 Clash YAML
	yamlText, err := nodesToClashYAML(nodes)
	if err != nil {
		t.Fatalf("转换为 Clash YAML 失败: %v", err)
	}
	t.Logf("Clash YAML 大小: %d 字节", len(yamlText))

	// 验证 YAML 可解析
	parsedNodes, err := parseClashYAML(yamlText)
	if err != nil {
		t.Fatalf("parseClashYAML of converted output failed: %v", err)
	}
	if len(parsedNodes) != len(nodes) {
		t.Errorf("Clash YAML 回解析节点数不匹配: got %d, want %d", len(parsedNodes), len(nodes))
	}
	t.Logf("Shadowrocket → Clash YAML 转换成功，%d 个节点", len(parsedNodes))
}
