package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// =====================================================================
// Proxy Models — SQLite-backed storage (Phase 1 of storage unification)
// Previously: data/proxy_config.json, data/subscription_cache.json
// =====================================================================

// ProxyNodeRow mirrors the ProxyNode JSON structure
type ProxyNodeRow struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Server    string    `json:"server"`
	Port      int       `json:"port"`
	Extra     string    `json:"extra,omitempty"` // JSON map
	Latency   int64     `json:"latency"`         // ms, -1=failed
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProxyConfigRow mirrors the proxyPersist JSON structure
type ProxyConfigRow struct {
	ID               int64     `json:"id"`
	SourceURL        string    `json:"source_url,omitempty"`
	SourceURLs       string    `json:"source_urls,omitempty"` // JSON array
	YAMLContent      string    `json:"yaml_content,omitempty"`
	RoutingMode      string    `json:"routing_mode,omitempty"`
	DefaultNodeName  string    `json:"default_node_name,omitempty"`
	DefaultNodeRegex string    `json:"default_node_regex,omitempty"`
	AINodeName       string    `json:"ai_node_name,omitempty"`
	AINodeRegex      string    `json:"ai_node_regex,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// InitProxySchema registers the proxy tables
func InitProxySchema(db *DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS proxy_nodes (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT NOT NULL,
		type       TEXT NOT NULL,
		server     TEXT NOT NULL,
		port       INTEGER NOT NULL,
		extra      TEXT DEFAULT '{}',
		latency    INTEGER DEFAULT -1,
		status     TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS proxy_config (
		id                   INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
		source_url          TEXT DEFAULT '',
		source_urls         TEXT DEFAULT '[]',
		yaml_content        TEXT DEFAULT '',
		routing_mode        TEXT DEFAULT 'smart',
		default_node_name   TEXT DEFAULT '',
		default_node_regex  TEXT DEFAULT '',
		ai_node_name        TEXT DEFAULT '',
		ai_node_regex       TEXT DEFAULT '',
		updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS proxy_subscription_cache (
		source_url TEXT PRIMARY KEY,
		content    TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS proxy_subscription_backup (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		content    TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_proxy_nodes_latency ON proxy_nodes(latency);
	CREATE INDEX IF NOT EXISTS idx_proxy_nodes_status ON proxy_nodes(status);
	CREATE INDEX IF NOT EXISTS idx_proxy_subscription_cache_updated ON proxy_subscription_cache(updated_at);
	`
	_, err := db.conn.Exec(schema)
	return err
}

// =====================================================================
// Proxy Nodes
// =====================================================================

// GetAllProxyNodes returns all proxy nodes ordered by id
func (db *DB) GetAllProxyNodes() ([]ProxyNodeRow, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, type, server, port, extra, latency, status, created_at, updated_at
		FROM proxy_nodes ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []ProxyNodeRow
	for rows.Next() {
		var n ProxyNodeRow
		if err := rows.Scan(&n.ID, &n.Name, &n.Type, &n.Server, &n.Port,
			&n.Extra, &n.Latency, &n.Status, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// UpsertProxyNode inserts or replaces a proxy node
func (db *DB) UpsertProxyNode(n *ProxyNodeRow) error {
	extra := n.Extra
	if extra == "" {
		extra = "{}"
	}
	_, err := db.conn.Exec(`
		INSERT INTO proxy_nodes (name, type, server, port, extra, latency, status, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, type=excluded.type, server=excluded.server,
			port=excluded.port, extra=excluded.extra, latency=excluded.latency,
			status=excluded.status, updated_at=CURRENT_TIMESTAMP`,
		n.Name, n.Type, n.Server, n.Port, extra, n.Latency, n.Status)
	return err
}

// ReplaceAllProxyNodes replaces all nodes (used for subscription import)
func (db *DB) ReplaceAllProxyNodes(nodes []ProxyNodeRow) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM proxy_nodes"); err != nil {
		return err
	}
	for _, n := range nodes {
		extra := n.Extra
		if extra == "" {
			extra = "{}"
		}
		if _, err := tx.Exec(`
			INSERT INTO proxy_nodes (name, type, server, port, extra, latency, status)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			n.Name, n.Type, n.Server, n.Port, extra, n.Latency, n.Status); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpdateProxyNodeLatency updates latency for a node by server:port
func (db *DB) UpdateProxyNodeLatency(server string, port int, latency int64) error {
	_, err := db.conn.Exec(`
		UPDATE proxy_nodes SET latency=?, updated_at=CURRENT_TIMESTAMP
		WHERE server=? AND port=?`, latency, server, port)
	return err
}

// DeleteProxyNode deletes a node by id
func (db *DB) DeleteProxyNode(id int64) error {
	_, err := db.conn.Exec("DELETE FROM proxy_nodes WHERE id=?", id)
	return err
}

// =====================================================================
// Proxy Config (singleton row id=1)
// =====================================================================

// GetProxyConfig returns the singleton config row
func (db *DB) GetProxyConfig() (*ProxyConfigRow, error) {
	var c ProxyConfigRow
	err := db.conn.QueryRow(`
		SELECT id, source_url, source_urls, yaml_content, routing_mode,
		       default_node_name, default_node_regex, ai_node_name, ai_node_regex, updated_at
		FROM proxy_config WHERE id=1`).Scan(
		&c.ID, &c.SourceURL, &c.SourceURLs, &c.YAMLContent,
		&c.RoutingMode, &c.DefaultNodeName, &c.DefaultNodeRegex,
		&c.AINodeName, &c.AINodeRegex, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		// Insert default row
		_, err = db.conn.Exec(`INSERT INTO proxy_config (id) VALUES (1)`)
		if err != nil {
			return nil, err
		}
		return &ProxyConfigRow{ID: 1, RoutingMode: "smart", SourceURLs: "[]"}, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// SaveProxyConfig saves the singleton config row
func (db *DB) SaveProxyConfig(c *ProxyConfigRow) error {
	_, err := db.conn.Exec(`
		INSERT INTO proxy_config (id, source_url, source_urls, yaml_content, routing_mode,
			default_node_name, default_node_regex, ai_node_name, ai_node_regex, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			source_url=excluded.source_url, source_urls=excluded.source_urls,
			yaml_content=excluded.yaml_content, routing_mode=excluded.routing_mode,
			default_node_name=excluded.default_node_name, default_node_regex=excluded.default_node_regex,
			ai_node_name=excluded.ai_node_name, ai_node_regex=excluded.ai_node_regex,
			updated_at=CURRENT_TIMESTAMP`,
		c.SourceURL, c.SourceURLs, c.YAMLContent, c.RoutingMode,
		c.DefaultNodeName, c.DefaultNodeRegex, c.AINodeName, c.AINodeRegex)
	return err
}

// ProxyConfigToJSON converts ProxyConfigRow to JSON bytes (for compatibility)
func ProxyConfigToJSON(c *ProxyConfigRow, nodes []ProxyNodeRow) ([]byte, error) {
	type jsonNode struct {
		Name    string                 `json:"name"`
		Type    string                 `json:"type"`
		Server  string                 `json:"server"`
		Port    int                    `json:"port"`
		Extra   map[string]interface{} `json:"extra,omitempty"`
		Latency int64                  `json:"latency"`
		Status  string                 `json:"status,omitempty"`
	}
	var jsonNodes []jsonNode
	for _, n := range nodes {
		var extra map[string]interface{}
		if n.Extra != "" && n.Extra != "{}" {
			json.Unmarshal([]byte(n.Extra), &extra)
		}
		jsonNodes = append(jsonNodes, jsonNode{
			Name:    n.Name,
			Type:    n.Type,
			Server:  n.Server,
			Port:    n.Port,
			Extra:   extra,
			Latency: n.Latency,
			Status:  n.Status,
		})
	}

	type proxyPersist struct {
		SourceURL        string     `json:"source_url,omitempty"`
		SourceURLs       []string   `json:"source_urls,omitempty"`
		YAMLContent      string     `json:"yaml_content,omitempty"`
		RoutingMode      string     `json:"routing_mode,omitempty"`
		DefaultNodeName  string     `json:"default_node_name,omitempty"`
		DefaultNodeRegex string     `json:"default_node_regex,omitempty"`
		AINodeName       string     `json:"ai_node_name,omitempty"`
		AINodeRegex      string     `json:"ai_node_regex,omitempty"`
		Nodes            []jsonNode `json:"nodes"`
	}

	var sourceURLs []string
	if c.SourceURLs != "" {
		json.Unmarshal([]byte(c.SourceURLs), &sourceURLs)
	}

	p := proxyPersist{
		SourceURL:        c.SourceURL,
		SourceURLs:       sourceURLs,
		YAMLContent:      c.YAMLContent,
		RoutingMode:      c.RoutingMode,
		DefaultNodeName:  c.DefaultNodeName,
		DefaultNodeRegex: c.DefaultNodeRegex,
		AINodeName:       c.AINodeName,
		AINodeRegex:      c.AINodeRegex,
		Nodes:            jsonNodes,
	}
	return json.Marshal(p)
}

// =====================================================================
// Subscription Cache
// =====================================================================

// GetProxySubscriptionCache returns all cached subscription entries
func (db *DB) GetProxySubscriptionCache() (map[string]string, error) {
	rows, err := db.conn.Query("SELECT source_url, content FROM proxy_subscription_cache")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cache := make(map[string]string)
	for rows.Next() {
		var url, content string
		if err := rows.Scan(&url, &content); err != nil {
			return nil, err
		}
		cache[url] = content
	}
	return cache, rows.Err()
}

// SetProxySubscriptionCacheEntry upserts a cache entry
func (db *DB) SetProxySubscriptionCacheEntry(sourceURL, content string) error {
	_, err := db.conn.Exec(`
		INSERT INTO proxy_subscription_cache (source_url, content, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(source_url) DO UPDATE SET content=excluded.content, updated_at=CURRENT_TIMESTAMP`,
		sourceURL, content)
	return err
}

// GetProxySubscriptionCacheEntry returns a single cache entry
func (db *DB) GetProxySubscriptionCacheEntry(sourceURL string) (string, bool, error) {
	var content string
	err := db.conn.QueryRow("SELECT content FROM proxy_subscription_cache WHERE source_url=?", sourceURL).Scan(&content)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return content, true, nil
}

// =====================================================================
// Subscription Backup
// =====================================================================

// AddProxySubscriptionBackup adds a new backup entry
func (db *DB) AddProxySubscriptionBackup(content string) error {
	_, err := db.conn.Exec("INSERT INTO proxy_subscription_backup (content) VALUES (?)", content)
	return err
}

// GetLatestProxySubscriptionBackup returns the most recent backup
func (db *DB) GetLatestProxySubscriptionBackup() (string, error) {
	var content string
	err := db.conn.QueryRow(`
		SELECT content FROM proxy_subscription_backup ORDER BY created_at DESC LIMIT 1`).Scan(&content)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return content, err
}

func init() {
	RegisterInit("proxy_config", InitProxySchema)
}
