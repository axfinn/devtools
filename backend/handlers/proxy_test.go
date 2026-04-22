package handlers

import "testing"

func setProxySessionForTest(t *testing.T, nodes []ProxyNode, active *ProxyNode, routeMode, defaultNodeName, defaultNodeRegex, aiNodeName, aiNodeRegex string) {
	t.Helper()

	globalSession.mu.Lock()
	backupNodes := append([]ProxyNode(nil), globalSession.nodes...)
	backupSourceURL := globalSession.sourceURL
	backupSourceURLs := append([]string(nil), globalSession.sourceURLs...)
	backupYAMLContent := globalSession.yamlContent
	backupRoutingMode := globalSession.routingMode
	backupDefaultNodeName := globalSession.defaultNodeName
	backupDefaultNodeRegex := globalSession.defaultNodeRegex
	backupAINodeName := globalSession.aiNodeName
	backupAINodeRegex := globalSession.aiNodeRegex
	backupActive := cloneNode(globalSession.active)
	backupListener := globalSession.listener
	globalSession.nodes = append([]ProxyNode(nil), nodes...)
	globalSession.active = cloneNode(active)
	globalSession.routingMode = routeMode
	globalSession.defaultNodeName = defaultNodeName
	globalSession.defaultNodeRegex = defaultNodeRegex
	globalSession.aiNodeName = aiNodeName
	globalSession.aiNodeRegex = aiNodeRegex
	globalSession.mu.Unlock()

	t.Cleanup(func() {
		globalSession.mu.Lock()
		globalSession.nodes = backupNodes
		globalSession.sourceURL = backupSourceURL
		globalSession.sourceURLs = backupSourceURLs
		globalSession.yamlContent = backupYAMLContent
		globalSession.routingMode = backupRoutingMode
		globalSession.defaultNodeName = backupDefaultNodeName
		globalSession.defaultNodeRegex = backupDefaultNodeRegex
		globalSession.aiNodeName = backupAINodeName
		globalSession.aiNodeRegex = backupAINodeRegex
		globalSession.active = backupActive
		globalSession.listener = backupListener
		globalSession.mu.Unlock()
	})
}

func TestNormalizeSourceURLs(t *testing.T) {
	got := normalizeSourceURLs([]string{
		" https://a.example/sub \nhttps://b.example/sub ",
		"https://a.example/sub",
	}, "https://c.example/sub")

	want := []string{
		"https://a.example/sub",
		"https://b.example/sub",
		"https://c.example/sub",
	}

	if len(got) != len(want) {
		t.Fatalf("len(normalizeSourceURLs) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalizeSourceURLs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMergeProxyNodesDeduplicates(t *testing.T) {
	nodes := mergeProxyNodes(
		[]ProxyNode{
			{Name: "美国-chatGPT-02", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 35147},
			{Name: "日本|隧道 ➤03", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 23655},
		},
		[]ProxyNode{
			{Name: "美国-chatGPT-02", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 35147},
		},
	)

	if len(nodes) != 2 {
		t.Fatalf("len(mergeProxyNodes) = %d, want 2", len(nodes))
	}
}

func TestMatchNodeByRegex(t *testing.T) {
	nodes := []ProxyNode{
		{Name: "台湾|隧道 ➤03 流媒体 Netfilx ChatGPT|限速50M"},
		{Name: "韩国|隧道 ➤02 IEPL*3"},
		{Name: "美国-chatGPT-02 限速20M 不适合大流量"},
		{Name: "日本|隧道 ➤03"},
	}

	got := matchNodeByRegex(nodes, "(?i)(chatgpt|gpt)", nil)
	if got == nil || got.Name != "台湾|隧道 ➤03 流媒体 Netfilx ChatGPT|限速50M" {
		t.Fatalf("matchNodeByRegex(chatgpt) = %#v", got)
	}

	got = matchNodeByRegex(nodes, "日本\\|隧道\\s*➤03", nil)
	if got == nil || got.Name != "日本|隧道 ➤03" {
		t.Fatalf("matchNodeByRegex(japan) = %#v", got)
	}
}

func TestAIDedicatedHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{host: "chat.openai.com:443", want: true},
		{host: "api.anthropic.com:443", want: true},
		{host: "example.com:443", want: false},
	}

	for _, tc := range cases {
		if got := isAIDedicatedHost(tc.host); got != tc.want {
			t.Fatalf("isAIDedicatedHost(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestSelectBestMatchingNodePrefersGPTDedicatedLine(t *testing.T) {
	nodes := []ProxyNode{
		{Name: "台湾|隧道 ➤03 流媒体 Netfilx ChatGPT|限速50M", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 42147, Latency: 88},
		{Name: "美国-chatGPT-02 限速20M 不适合大流量", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 35147, Latency: 95},
		{Name: "日本|隧道 ➤03", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 23655, Latency: 99},
	}

	got := selectBestMatchingNode(nodes, "(?i)(chatgpt|gpt)", "chat.openai.com:443", nil)
	if got == nil {
		t.Fatal("selectBestMatchingNode returned nil")
	}
	if got.Name != "美国-chatGPT-02 限速20M 不适合大流量" {
		t.Fatalf("selectBestMatchingNode picked %q", got.Name)
	}
}

func TestSelectBestMatchingNodePrefersCurrentLine(t *testing.T) {
	nodes := []ProxyNode{
		{Name: "美国-chatGPT-02 限速20M 不适合大流量", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 35147, Latency: 95},
		{Name: "台湾|隧道 ➤03 流媒体 Netfilx ChatGPT|限速50M", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 42147, Latency: 88},
	}
	preferred := &ProxyNode{Name: "台湾|隧道 ➤03 流媒体 Netfilx ChatGPT|限速50M", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 42147, Latency: 88}

	got := selectBestMatchingNode(nodes, "(?i)(chatgpt|gpt)", "chat.openai.com:443", preferred)
	if got == nil {
		t.Fatal("selectBestMatchingNode returned nil")
	}
	if got.Name != preferred.Name {
		t.Fatalf("selectBestMatchingNode should keep preferred line, got %q", got.Name)
	}
}

func TestResolveConfiguredNodesPrefersManualDefaultNode(t *testing.T) {
	nodes := []ProxyNode{
		{Name: "韩国|隧道 ➤02 IEPL*3", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 42894, Latency: 88},
		{Name: "美国-chatGPT-02 限速20M 不适合大流量", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 35147, Latency: 95},
		{Name: "日本|隧道 ➤03", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 23655, Latency: 99},
	}
	active := &ProxyNode{Name: "日本|隧道 ➤03", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 23655, Latency: 99}

	setProxySessionForTest(
		t,
		nodes,
		active,
		proxyRouteModeAIPriority,
		"韩国|隧道 ➤02 IEPL*3",
		"(?i)日本",
		"",
		"(?i)(chatgpt|gpt)",
	)

	defaultNode, aiNode := resolveConfiguredNodes()
	if defaultNode == nil || defaultNode.Name != "韩国|隧道 ➤02 IEPL*3" {
		t.Fatalf("default node = %#v, want manual default node", defaultNode)
	}
	if aiNode == nil || aiNode.Name != "美国-chatGPT-02 限速20M 不适合大流量" {
		t.Fatalf("ai node = %#v, want GPT dedicated node", aiNode)
	}
}

func TestResolveRouteNodeUsesManualDefaultAndManualAINode(t *testing.T) {
	nodes := []ProxyNode{
		{Name: "韩国|隧道 ➤02 IEPL*3", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 42894, Latency: 88},
		{Name: "美国-chatGPT-02 限速20M 不适合大流量", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 35147, Latency: 95},
		{Name: "日本|隧道 ➤03", Type: "trojan", Server: "a.twodnwnpoe.xyz", Port: 23655, Latency: 99},
	}

	setProxySessionForTest(
		t,
		nodes,
		nil,
		proxyRouteModeAIPriority,
		"韩国|隧道 ➤02 IEPL*3",
		"",
		"美国-chatGPT-02 限速20M 不适合大流量",
		"(?i)(chatgpt|gpt)",
	)

	selectedNode, shouldProxy, fallbackNode := resolveRouteNode(nil, "chat.openai.com:443")
	if !shouldProxy {
		t.Fatal("resolveRouteNode should proxy AI host")
	}
	if selectedNode == nil || selectedNode.Name != "美国-chatGPT-02 限速20M 不适合大流量" {
		t.Fatalf("selected node = %#v, want manual AI node", selectedNode)
	}
	if fallbackNode == nil || fallbackNode.Name != "韩国|隧道 ➤02 IEPL*3" {
		t.Fatalf("fallback node = %#v, want manual default node", fallbackNode)
	}
}
