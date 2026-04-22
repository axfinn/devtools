package handlers

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"testing"
)

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

func TestExtractLatestSiteURL(t *testing.T) {
	html := `
<html>
<script>
window.location = "https://gdmn2.top";
</script>
<a href="https://t.me/muacloud_admina">tg</a>
</html>`

	got := extractLatestSiteURL("https://muacloud.github.io/", html)
	if got != "https://gdmn2.top" {
		t.Fatalf("extractLatestSiteURL() = %q", got)
	}
}

func TestSelectPreferredSubscriptionURL(t *testing.T) {
	urls := []string{
		"http://gdmn4.top/link/abc?sub=1",
		"http://gdmn4.top/link/abc?clash=2",
		"http://gdmn4.top/link/abc?clash=1",
	}

	got := selectPreferredSubscriptionURL(urls, "clash")
	if got != "http://gdmn4.top/link/abc?clash=1" {
		t.Fatalf("selectPreferredSubscriptionURL() = %q", got)
	}
}

func TestExtractManagedSubscribeURLs(t *testing.T) {
	html := `
<button data-clipboard-text="http://gdmn4.top/link/W1vPgE31CVSG8qt5?sub=1"></button>
<button data-clipboard-text="http://gdmn4.top/link/W1vPgE31CVSG8qt5?clash=1"></button>
<button data-clipboard-text="http://gdmn4.top/link/W1vPgE31CVSG8qt5?clash=1"></button>`

	got := extractManagedSubscribeURLs(html)
	if len(got) != 2 {
		t.Fatalf("len(extractManagedSubscribeURLs) = %d, want 2", len(got))
	}
	if got[1] != "http://gdmn4.top/link/W1vPgE31CVSG8qt5?clash=1" {
		t.Fatalf("extractManagedSubscribeURLs[1] = %q", got[1])
	}
}

func TestCurrentProxyURLUsesActiveListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	globalSession.mu.Lock()
	backupListener := globalSession.listener
	globalSession.listener = ln
	globalSession.mu.Unlock()

	backupHandler := globalProxyHandler
	globalProxyHandler = &ProxyHandler{adminPassword: "secret"}

	t.Cleanup(func() {
		globalSession.mu.Lock()
		globalSession.listener = backupListener
		globalSession.mu.Unlock()
		globalProxyHandler = backupHandler
	})

	got := currentProxyURL()
	if !strings.HasPrefix(got, "http://proxy:secret@127.0.0.1:") {
		t.Fatalf("currentProxyURL() = %q", got)
	}
}

func TestSplitAlertRecipients(t *testing.T) {
	got := splitAlertRecipients(" alpha@example.test;beta@example.test,\nalpha@example.test \t gamma@example.test ")
	want := []string{
		"alpha@example.test",
		"beta@example.test",
		"gamma@example.test",
	}

	if len(got) != len(want) {
		t.Fatalf("len(splitAlertRecipients) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitAlertRecipients[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSendAlertSMTPFlow(t *testing.T) {
	server := newFakeSMTPServer(t)
	defer server.Close()

	h := &ProxyHandler{
		alertEmail: "alpha@example.test; beta@example.test",
		smtpHost:   "localhost",
		smtpPort:   server.Port(),
		smtpUser:   "sender@example.test",
		smtpPass:   "test-password",
	}

	h.sendAlert("线路切换成功", "当前线路: 美国-chatGPT-02")

	session := server.WaitForSession(t)
	if session.mailFrom != "sender@example.test" {
		t.Fatalf("MAIL FROM = %q, want %q", session.mailFrom, "sender@example.test")
	}

	wantRecipients := []string{"alpha@example.test", "beta@example.test"}
	if len(session.recipients) != len(wantRecipients) {
		t.Fatalf("len(recipients) = %d, want %d", len(session.recipients), len(wantRecipients))
	}
	for i := range wantRecipients {
		if session.recipients[i] != wantRecipients[i] {
			t.Fatalf("recipients[%d] = %q, want %q", i, session.recipients[i], wantRecipients[i])
		}
	}

	if !strings.Contains(session.data, "Subject: =?UTF-8?") {
		t.Fatalf("message subject is not MIME-encoded: %q", session.data)
	}
	if !strings.Contains(session.data, "[DevTools]") {
		t.Fatalf("message missing subject prefix: %q", session.data)
	}
	if !strings.Contains(session.data, "To: alpha@example.test, beta@example.test") {
		t.Fatalf("message missing To header: %q", session.data)
	}
	if !strings.Contains(session.data, "当前线路: 美国-chatGPT-02") {
		t.Fatalf("message missing body content: %q", session.data)
	}
}

func TestManagedSubscriptionAttachmentFilename(t *testing.T) {
	filename := managedSubscriptionAttachmentFilename("https://sub.example/link/abc?clash=1", "clash")
	if !strings.HasPrefix(filename, "devtools-managed-subscription-") {
		t.Fatalf("unexpected filename prefix: %q", filename)
	}
	if !strings.HasSuffix(filename, ".yaml") {
		t.Fatalf("unexpected filename suffix: %q", filename)
	}
}

func TestSendAlertSMTPFlowWithAttachment(t *testing.T) {
	server := newFakeSMTPServer(t)
	defer server.Close()

	h := &ProxyHandler{
		alertEmail: "alpha@example.test",
		smtpHost:   "localhost",
		smtpPort:   server.Port(),
		smtpUser:   "sender@example.test",
		smtpPass:   "test-password",
	}

	attachment := managedSubscriptionAttachment(
		"https://sub.example/link/abc?clash=1",
		"clash",
		[]byte("proxies:\n  - name: test\n"),
	)
	h.sendAlertWithAttachment("代理订阅刷新成功", "正文内容", attachment)

	session := server.WaitForSession(t)
	msg, err := mail.ReadMessage(strings.NewReader(session.data))
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType failed: %v", err)
	}
	if mediaType != "multipart/mixed" {
		t.Fatalf("Content-Type = %q, want multipart/mixed", mediaType)
	}

	mr := multipart.NewReader(msg.Body, params["boundary"])
	textPart, err := mr.NextPart()
	if err != nil {
		t.Fatalf("NextPart(text) failed: %v", err)
	}
	textBody, err := io.ReadAll(textPart)
	if err != nil {
		t.Fatalf("ReadAll(text) failed: %v", err)
	}
	if strings.TrimSpace(string(textBody)) != "正文内容" {
		t.Fatalf("text body = %q", string(textBody))
	}

	filePart, err := mr.NextPart()
	if err != nil {
		t.Fatalf("NextPart(file) failed: %v", err)
	}
	if got := filePart.FileName(); got == "" || !strings.HasSuffix(got, ".yaml") {
		t.Fatalf("attachment filename = %q", got)
	}
	if got := filePart.Header.Get("Content-Transfer-Encoding"); got != "base64" {
		t.Fatalf("attachment encoding = %q, want base64", got)
	}
	encodedBody, err := io.ReadAll(filePart)
	if err != nil {
		t.Fatalf("ReadAll(file) failed: %v", err)
	}
	decodedBody, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encodedBody)))
	if err != nil {
		t.Fatalf("DecodeString failed: %v", err)
	}
	if string(decodedBody) != "proxies:\n  - name: test\n" {
		t.Fatalf("attachment body = %q", string(decodedBody))
	}
}

type fakeSMTPSession struct {
	mailFrom   string
	recipients []string
	data       string
}

type fakeSMTPServer struct {
	t        *testing.T
	listener net.Listener
	done     chan fakeSMTPSession
	errc     chan error
	once     sync.Once
}

func newFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("fake smtp listen failed: %v", err)
	}

	s := &fakeSMTPServer{
		t:        t,
		listener: ln,
		done:     make(chan fakeSMTPSession, 1),
		errc:     make(chan error, 1),
	}
	go s.serve()
	return s
}

func (s *fakeSMTPServer) Port() int {
	s.t.Helper()
	_, portStr, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		s.t.Fatalf("split host port failed: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		s.t.Fatalf("parse port failed: %v", err)
	}
	return port
}

func (s *fakeSMTPServer) Close() {
	s.once.Do(func() {
		_ = s.listener.Close()
	})
}

func (s *fakeSMTPServer) WaitForSession(t *testing.T) fakeSMTPSession {
	t.Helper()

	select {
	case err := <-s.errc:
		t.Fatalf("fake smtp server failed: %v", err)
	case session := <-s.done:
		return session
	}
	return fakeSMTPSession{}
}

func (s *fakeSMTPServer) serve() {
	conn, err := s.listener.Accept()
	if err != nil {
		s.errc <- err
		return
	}
	defer conn.Close()

	session, err := handleFakeSMTPConn(conn)
	if err != nil {
		s.errc <- err
		return
	}
	s.done <- session
}

func handleFakeSMTPConn(conn net.Conn) (fakeSMTPSession, error) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	session := fakeSMTPSession{}

	writeLine := func(line string) error {
		if _, err := writer.WriteString(line + "\r\n"); err != nil {
			return err
		}
		return writer.Flush()
	}

	if err := writeLine("220 localhost ESMTP test"); err != nil {
		return session, err
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return session, nil
			}
			return session, err
		}
		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(upper, "EHLO "), strings.HasPrefix(upper, "HELO "):
			if _, err := writer.WriteString("250-localhost\r\n250 AUTH PLAIN\r\n"); err != nil {
				return session, err
			}
			if err := writer.Flush(); err != nil {
				return session, err
			}
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			if err := writeLine("235 2.7.0 Authentication successful"); err != nil {
				return session, err
			}
		case strings.HasPrefix(upper, "MAIL FROM:"):
			session.mailFrom = normalizeSMTPPath(line[len("MAIL FROM:"):])
			if err := writeLine("250 2.1.0 Ok"); err != nil {
				return session, err
			}
		case strings.HasPrefix(upper, "RCPT TO:"):
			session.recipients = append(session.recipients, normalizeSMTPPath(line[len("RCPT TO:"):]))
			if err := writeLine("250 2.1.5 Ok"); err != nil {
				return session, err
			}
		case upper == "DATA":
			if err := writeLine("354 End data with <CR><LF>.<CR><LF>"); err != nil {
				return session, err
			}
			var data strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return session, err
				}
				if dataLine == ".\r\n" {
					break
				}
				data.WriteString(dataLine)
			}
			session.data = data.String()
			if err := writeLine("250 2.0.0 queued"); err != nil {
				return session, err
			}
		case upper == "QUIT":
			if err := writeLine("221 2.0.0 Bye"); err != nil {
				return session, err
			}
			return session, nil
		default:
			return session, fmt.Errorf("unexpected smtp command: %s", line)
		}
	}
}

func normalizeSMTPPath(raw string) string {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "<") && strings.Contains(value, ">") {
		value = value[1:strings.Index(value, ">")]
	}
	if idx := strings.Index(value, " "); idx >= 0 {
		value = value[:idx]
	}
	return value
}
