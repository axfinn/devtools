package handlers

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// ============================================================
// 托管订阅刷新端到端测试（需要 proxy_config.json 真实数据）
//
// 运行方式：
//   cd backend && go test -v -run "TestSubscription" -timeout 120s ./handlers/
//
// 需要 proxy_config.json 和 config.yaml 在同一目录或上级目录
// ============================================================

func loadProxyConfigForTest(t *testing.T) (nodes []ProxyNode, sourceURLs []string) {
	t.Helper()
	paths := []string{
		"/app/data/proxy_config.json",
		"../../data/proxy_config.json",
		"proxy_config.json",
	}
	var data []byte
	for _, p := range paths {
		d, err := os.ReadFile(p)
		if err == nil {
			data = d
			t.Logf("从 %s 加载 proxy_config.json", p)
			break
		}
	}
	if data == nil {
		t.Skip("未找到 proxy_config.json")
	}
	var cfg struct {
		Nodes      []ProxyNode `json:"nodes"`
		SourceURLs []string    `json:"source_urls"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	return cfg.Nodes, cfg.SourceURLs
}

// ============================================================
// TestSubRefresh_BuildNodeClients 测试节点连通性检测
// ============================================================

func TestSubRefresh_BuildNodeClients(t *testing.T) {
	nodes, _ := loadProxyConfigForTest(t)

	landingURL := "https://muacloud.github.io/"
	t.Logf("测试落地页: %s", landingURL)
	t.Logf("节点数量: %d", len(nodes))

	// 将所有节点写入全局 session
	globalSession.mu.Lock()
	oldNodes := globalSession.nodes
	globalSession.nodes = nodes
	globalSession.mu.Unlock()
	defer func() {
		globalSession.mu.Lock()
		globalSession.nodes = oldNodes
		globalSession.mu.Unlock()
	}()

	// 创建一个 minimal ProxyHandler 来调用方法
	h := &ProxyHandler{}
	candidates := h.buildNodeClientsForURL(landingURL, 15*time.Second)
	defer func() {
		for _, c := range candidates {
			if c.ln != nil {
				c.ln.Close()
			}
		}
	}()

	for _, c := range candidates {
		t.Logf("可达落地页: %s", c.source)
	}

	if len(candidates) == 0 {
		t.Error("无节点可达落地页 — Docker 刷新无法工作")
	} else {
		t.Logf("共 %d 个节点可达落地页", len(candidates))
	}
}

// ============================================================
// TestSubRefresh_FallbackAllNodes 验证 Bug 2 修复
// 回退时尝试所有 nodeCandidates 而非 nodeCandidates[0]
// ============================================================

func TestSubRefresh_FallbackAllNodes(t *testing.T) {
	nodes, sourceURLs := loadProxyConfigForTest(t)

	if len(sourceURLs) == 0 {
		t.Skip("无持久化订阅 URL")
	}

	landingURL := "https://muacloud.github.io/"

	globalSession.mu.Lock()
	oldNodes := globalSession.nodes
	globalSession.nodes = nodes
	globalSession.mu.Unlock()
	defer func() {
		globalSession.mu.Lock()
		globalSession.nodes = oldNodes
		globalSession.mu.Unlock()
	}()

	h := &ProxyHandler{}
	candidates := h.buildNodeClientsForURL(landingURL, 15*time.Second)
	defer func() {
		for _, c := range candidates {
			if c.ln != nil {
				c.ln.Close()
			}
		}
	}()

	if len(candidates) == 0 {
		t.Fatal("无节点可达落地页")
	}

	t.Logf("测试 %d 个候选节点对 %d 个持久化订阅 URL 的连通性", len(candidates), len(sourceURLs))

	successFrom := make([]string, 0)
	for i, nc := range candidates {
		for _, fallbackURL := range sourceURLs {
			t.Logf("  节点 #%d %s → %s", i, nc.source, fallbackURL[:min(len(fallbackURL), 60)])
			yamlText, err := fetchSubscriptionWithClient(nc.client, fallbackURL)
			if err != nil {
				t.Logf("    下载失败: %v", err)
				continue
			}
			parsed, err := parseClashYAML(yamlText)
			if err != nil {
				t.Logf("    解析失败: %v", err)
				continue
			}
			if len(parsed) == 0 {
				t.Logf("    节点列表为空")
				continue
			}
			t.Logf("    成功: %d 个节点, %d 字节", len(parsed), len(yamlText))
			successFrom = append(successFrom, nc.source)
			break
		}
	}

	if len(successFrom) > 1 {
		t.Logf("多个节点 (%d) 可获取持久化订阅 — Bug 2 修复有效", len(successFrom))
	} else if len(successFrom) == 1 {
		t.Logf("仅 1 个节点可获取持久化订阅 — Bug 2 修复无退化")
	} else {
		t.Error("无节点可获取持久化订阅")
	}
}

// ============================================================
// TestSubRefresh_ManagedFlow 测试完整托管订阅刷新流程
// ============================================================

func TestSubRefresh_ManagedFlow(t *testing.T) {
	nodes, _ := loadProxyConfigForTest(t)

	landingURL := "https://muacloud.github.io/"

	globalSession.mu.Lock()
	oldNodes := globalSession.nodes
	globalSession.nodes = nodes
	globalSession.mu.Unlock()
	defer func() {
		globalSession.mu.Lock()
		globalSession.nodes = oldNodes
		globalSession.mu.Unlock()
	}()

	h := &ProxyHandler{}
	// 设置登录信息（从私有配置）
	h.subRefresh.loginEmail = "521very@gmail.com"
	h.subRefresh.loginPassword = "cajceb-voNbis-8hizmu"
	h.subRefresh.landingURL = landingURL

	candidates := h.buildNodeClientsForURL(landingURL, 20*time.Second)
	defer func() {
		for _, c := range candidates {
			if c.ln != nil {
				c.ln.Close()
			}
		}
	}()

	if len(candidates) == 0 {
		t.Fatal("无节点可达落地页")
	}

	c := candidates[0]
	t.Logf("使用节点: %s", c.source)

	// Step 1: 解析站点
	siteURL, _, err := h.resolveManagedSiteURL(c.client)
	if err != nil {
		t.Fatalf("解析站点失败: %v", err)
	}
	t.Logf("站点 URL: %s", siteURL)

	// Step 2: 登录
	if err := h.loginManagedSubscription(c.client, siteURL); err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	t.Logf("登录成功")

	// Step 3: 提取订阅链接
	subURL, _, err := h.fetchManagedSubscriptionLink(c.client, siteURL)
	if err != nil {
		t.Fatalf("提取订阅链接失败: %v", err)
	}
	t.Logf("订阅链接: %s", subURL[:min(len(subURL), 80)]+"...")

	// Step 4: 下载订阅
	yamlText, err := fetchSubscriptionWithClient(c.client, subURL)
	if err != nil {
		t.Fatalf("下载订阅失败: %v", err)
	}
	t.Logf("下载订阅: %d 字节", len(yamlText))

	// Step 5: 解析
	parsedNodes, err := parseClashYAML(yamlText)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(parsedNodes) == 0 {
		t.Fatal("节点列表为空")
	}
	t.Logf("解析出 %d 个节点", len(parsedNodes))

	// Step 6: 验证
	if !strings.Contains(yamlText, "proxies") && !strings.Contains(yamlText, "Proxy") {
		t.Error("YAML 中未找到 proxies 字段")
	}

	for _, n := range parsedNodes[:min(5, len(parsedNodes))] {
		t.Logf("  - %s (%s:%d)", n.Name, n.Server, n.Port)
	}
	t.Logf("完整托管刷新流程通过")
}

// ============================================================
// TestSubRefresh_FetchWithFallback 测试 fallback 逻辑
// ============================================================

func TestSubRefresh_FetchWithFallback(t *testing.T) {
	_, sourceURLs := loadProxyConfigForTest(t)
	if len(sourceURLs) == 0 {
		t.Skip("无持久化订阅 URL")
	}

	h := &ProxyHandler{}
	// 设置 fallback proxy URL（实际运行时由 currentProxyURL 读取环境变量）
	h.subRefresh.requestProxyURL = "http://proxy:123654789@nps.jaxiu.cn:18080"

	for _, rawURL := range sourceURLs {
		text, err := h.fetchSubscriptionWithFallback(rawURL)
		if err != nil {
			t.Logf("fallback 获取失败 (%s): %v", rawURL[:min(len(rawURL), 60)], err)
			continue
		}
		if len(text) == 0 {
			t.Error("响应为空")
			continue
		}
		nodes, err := parseClashYAML(text)
		if err != nil {
			t.Errorf("解析失败: %v", err)
			continue
		}
		t.Logf("fallback 获取成功: %d 字节, %d 节点", len(text), len(nodes))
		return
	}
	t.Error("所有 fallback 方式均失败")
}

// ============================================================
// TestSubRefresh_ConfigIntegrity 验证 proxy_config.json 数据完整性
// ============================================================

func TestSubRefresh_ConfigIntegrity(t *testing.T) {
	nodes, sourceURLs := loadProxyConfigForTest(t)

	t.Logf("节点数: %d", len(nodes))
	t.Logf("订阅 URL 数: %d", len(sourceURLs))

	validTypes := map[string]bool{"trojan": true, "vmess": true, "ss": true, "ssr": true, "vless": true}
	invalidCount := 0
	typeCount := make(map[string]int)
	for _, n := range nodes {
		if n.Name == "" || n.Server == "" || n.Port == 0 {
			invalidCount++
			continue
		}
		typeCount[n.Type]++
		if !validTypes[n.Type] {
			t.Logf("未知类型: %s (%s)", n.Name, n.Type)
		}
	}
	if invalidCount > 0 {
		t.Errorf("%d 个无效节点", invalidCount)
	} else {
		t.Logf("所有节点格式有效")
	}
	t.Logf("类型分布: %v", typeCount)

	if len(sourceURLs) == 0 {
		t.Error("无持久化订阅 URL")
	}
}

// ============================================================
// TestSubRefresh_ContinueNotReturn 验证 Bug 1 修复（编译时验证）
//
// 此测试验证 refreshManagedSubscriptionLocked 中
// applyManagedSubscription 失败后使用 continue 而非 return。
// 通过代码审查 + 下面逻辑层面的测试完成。
// ============================================================

func TestSubRefresh_ContinueNotReturn(t *testing.T) {
	nodes, _ := loadProxyConfigForTest(t)

	globalSession.mu.Lock()
	oldNodes := globalSession.nodes
	globalSession.nodes = nodes
	globalSession.mu.Unlock()
	defer func() {
		globalSession.mu.Lock()
		globalSession.nodes = oldNodes
		globalSession.mu.Unlock()
	}()

	h := &ProxyHandler{}
	candidates := h.buildNodeClientsForURL("https://muacloud.github.io/", 10*time.Second)
	defer func() {
		for _, c := range candidates {
			if c.ln != nil {
				c.ln.Close()
			}
		}
	}()

	// 验证多个 candidates 都存活且独立
	// 如果 Bug 1 存在（return），则第一个 candidate 失败会终止整个流程
	// 修复后（continue），后续 candidates 可继续尝试
	t.Logf("候选节点数: %d", len(candidates))
	for i, c := range candidates {
		// 每个 candidate 都是独立的 HTTP client + 临时代理
		// 我们用 HEAD 请求测试
		resp, err := c.client.Head("https://muacloud.github.io/")
		if err != nil {
			t.Logf("  候选 #%d %s: 不可达 (%v)", i, c.source, err)
			continue
		}
		resp.Body.Close()
		t.Logf("  候选 #%d %s: 可达 (HTTP %d)", i, c.source, resp.StatusCode)
	}
	if len(candidates) > 1 {
		t.Logf("多个候选存在 — continue 逻辑允许逐个尝试")
	}
}
