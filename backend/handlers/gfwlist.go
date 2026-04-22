package handlers

import (
	"bufio"
	"database/sql"
	"encoding/base64"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	gfwlistURL       = "https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt"
	gfwlistCacheFile = "/tmp/gfwlist_cache.txt"
	gfwlistTTL       = 24 * time.Hour
)

type gfwlistMatcher struct {
	mu        sync.RWMutex
	domains   map[string]bool // gfwlist 规则
	custom    map[string]bool // 用户自定义强制走代理的域名
	lastFetch time.Time
}

var globalGFW = &gfwlistMatcher{
	domains: map[string]bool{},
	custom:  map[string]bool{},
}

var customDomainDB *sql.DB

// InitGFWList 启动时加载，后台定期刷新
func InitGFWList(dbPath string) {
	initCustomDomainDB(dbPath)
	if err := globalGFW.load(); err != nil {
		log.Printf("gfwlist 初始加载失败: %v，使用空列表", err)
	}
	go func() {
		for range time.Tick(gfwlistTTL) {
			if err := globalGFW.load(); err != nil {
				log.Printf("gfwlist 刷新失败: %v", err)
			}
		}
	}()
}

func initCustomDomainDB(dbPath string) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("custom domain db 打开失败: %v", err)
		return
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS proxy_custom_domains (
		domain TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Printf("custom domain db 初始化失败: %v", err)
		return
	}
	customDomainDB = db
	// 加载到内存
	rows, err := db.Query("SELECT domain FROM proxy_custom_domains")
	if err != nil {
		return
	}
	defer rows.Close()
	globalGFW.mu.Lock()
	for rows.Next() {
		var d string
		rows.Scan(&d)
		globalGFW.custom[d] = true
	}
	globalGFW.mu.Unlock()
	log.Printf("自定义代理域名加载完成，共 %d 条", len(globalGFW.custom))
}

// ShouldProxy 判断 host（可含端口）是否需要走代理
func ShouldProxy(host string) bool {
	domain := host
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		if !strings.Contains(host[:idx], ":") {
			domain = host[:idx]
		}
	}
	if globalGFW.matchCustom(domain) {
		return true
	}
	return globalGFW.match(domain)
}

// AddCustomDomain 添加自定义代理域名
func AddCustomDomain(domain string) error {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if customDomainDB != nil {
		if _, err := customDomainDB.Exec("INSERT OR IGNORE INTO proxy_custom_domains(domain) VALUES(?)", domain); err != nil {
			return err
		}
	}
	globalGFW.mu.Lock()
	globalGFW.custom[domain] = true
	globalGFW.mu.Unlock()
	return nil
}

// RemoveCustomDomain 删除自定义代理域名
func RemoveCustomDomain(domain string) error {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if customDomainDB != nil {
		if _, err := customDomainDB.Exec("DELETE FROM proxy_custom_domains WHERE domain=?", domain); err != nil {
			return err
		}
	}
	globalGFW.mu.Lock()
	delete(globalGFW.custom, domain)
	globalGFW.mu.Unlock()
	return nil
}

// ListCustomDomains 返回自定义域名列表
func ListCustomDomains() []string {
	globalGFW.mu.RLock()
	defer globalGFW.mu.RUnlock()
	list := make([]string, 0, len(globalGFW.custom))
	for d := range globalGFW.custom {
		list = append(list, d)
	}
	return list
}

func (g *gfwlistMatcher) matchCustom(domain string) bool {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	g.mu.RLock()
	defer g.mu.RUnlock()
	for {
		if g.custom[domain] {
			return true
		}
		idx := strings.Index(domain, ".")
		if idx < 0 {
			return false
		}
		domain = domain[idx+1:]
	}
}

func (g *gfwlistMatcher) match(domain string) bool {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	g.mu.RLock()
	defer g.mu.RUnlock()
	// 逐级匹配：sub.example.com → example.com → com
	for {
		if g.domains[domain] {
			return true
		}
		idx := strings.Index(domain, ".")
		if idx < 0 {
			return false
		}
		domain = domain[idx+1:]
	}
}

func (g *gfwlistMatcher) load() error {
	raw, err := fetchGFWList()
	if err != nil {
		// 尝试读本地缓存
		raw, err = os.ReadFile(gfwlistCacheFile)
		if err != nil {
			return err
		}
		log.Printf("gfwlist 使用本地缓存")
	}

	domains := parseGFWList(raw)
	g.mu.Lock()
	g.domains = domains
	g.lastFetch = time.Now()
	g.mu.Unlock()
	log.Printf("gfwlist 加载完成，共 %d 条规则", len(domains))
	return nil
}

func fetchGFWList() ([]byte, error) {
	client := newDirectHTTPClient(30 * time.Second)
	resp, err := client.Get(gfwlistURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// gfwlist 是 base64 编码的
	dec := base64.NewDecoder(base64.StdEncoding, resp.Body)
	var buf []byte
	tmp := make([]byte, 4096)
	for {
		n, err := dec.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}

	// 写缓存
	os.WriteFile(gfwlistCacheFile, buf, 0644)
	return buf, nil
}

// parseGFWList 解析 ABP 格式规则，提取域名
func parseGFWList(data []byte) map[string]bool {
	domains := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") {
			continue
		}
		// 白名单规则跳过
		if strings.HasPrefix(line, "@@") {
			continue
		}

		domain := extractDomain(line)
		if domain != "" {
			domains[strings.ToLower(domain)] = true
		}
	}
	return domains
}

func extractDomain(rule string) string {
	// ||example.com^  →  example.com
	if strings.HasPrefix(rule, "||") {
		rule = strings.TrimPrefix(rule, "||")
		rule = strings.TrimSuffix(rule, "^")
		rule = strings.TrimSuffix(rule, "/")
		if isValidDomain(rule) {
			return rule
		}
		return ""
	}
	// |https://example.com  →  example.com
	if strings.HasPrefix(rule, "|") {
		rule = strings.TrimPrefix(rule, "|")
		rule = strings.TrimPrefix(rule, "https://")
		rule = strings.TrimPrefix(rule, "http://")
		rule = strings.SplitN(rule, "/", 2)[0]
		if isValidDomain(rule) {
			return rule
		}
		return ""
	}
	// .example.com 或 example.com（无通配符）
	rule = strings.TrimPrefix(rule, ".")
	if !strings.Contains(rule, "*") && !strings.Contains(rule, "/") && isValidDomain(rule) {
		return rule
	}
	return ""
}

func isValidDomain(s string) bool {
	if s == "" || strings.Contains(s, " ") || strings.Contains(s, "*") {
		return false
	}
	return strings.Contains(s, ".")
}
