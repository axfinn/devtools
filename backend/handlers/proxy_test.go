package handlers

import "testing"

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
