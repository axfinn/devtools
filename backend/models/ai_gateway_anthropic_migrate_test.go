package models

import (
	"path/filepath"
	"strings"
	"testing"
)

// anthropicProviderTableInfo 读 PRAGMA table_info(anthropic_providers),
// 返回 列名 -> 默认值 的映射。用于直接断言"迁移后某列存在"。
func anthropicProviderTableInfo(t *testing.T, db *DB) map[string]string {
	t.Helper()
	rows, err := db.conn.Query(`PRAGMA table_info(anthropic_providers)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(anthropic_providers): %v", err)
	}
	defer rows.Close()
	cols := map[string]string{}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue, pk interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan table_info row: %v", err)
		}
		dflt := ""
		if dfltValue != nil {
			if s, ok := dfltValue.(string); ok {
				dflt = s
			}
		}
		cols[name] = dflt
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info: %v", err)
	}
	return cols
}

// newAnthropicMigrateTestDB 建一个落在本地临时目录的 DB(不落 SMB 挂载盘)。
func newAnthropicMigrateTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := NewDB(filepath.Join(t.TempDir(), "anthropic_migrate.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestAnthropicProviderInit 表结构幂等 + 关键列存在。
//
// 背景:anthropic_providers 后续新增过 default_model / api_key_encrypted /
// header_timeout_sec 三列。SQLite 的 ALTER TABLE ADD COLUMN 不支持 IF NOT EXISTS,
// 迁移只能靠 "duplicate column" 错误串来判重 —— 因此"连启两次不报错"就是幂等性的
// 唯一可观测判据(第二次执行时三列都已存在,必须走 duplicate column 分支吞掉错误)。
func TestAnthropicProviderInit(t *testing.T) {
	db := newAnthropicMigrateTestDB(t)

	// 第一次:建表
	if err := db.InitAnthropicProviders(); err != nil {
		t.Fatalf("InitAnthropicProviders (第 1 次): %v", err)
	}
	cols := anthropicProviderTableInfo(t, db)
	for _, want := range []string{"header_timeout_sec", "default_model", "api_key_encrypted"} {
		if _, ok := cols[want]; !ok {
			t.Fatalf("首次建表后缺少列 %q; 实际列: %v", want, cols)
		}
	}
	if got := cols["header_timeout_sec"]; got != "0" {
		t.Errorf("header_timeout_sec 默认值 = %q, want \"0\"", got)
	}

	// 第二次:幂等(重复执行必须无错)
	if err := db.InitAnthropicProviders(); err != nil {
		t.Fatalf("InitAnthropicProviders (第 2 次) 应幂等, 实际报错: %v", err)
	}
	// 第三次:再确认一次幂等
	if err := db.InitAnthropicProviders(); err != nil {
		t.Fatalf("InitAnthropicProviders (第 3 次) 应幂等, 实际报错: %v", err)
	}

	cols2 := anthropicProviderTableInfo(t, db)
	if _, ok := cols2["header_timeout_sec"]; !ok {
		t.Fatalf("重复初始化后缺少列 header_timeout_sec; 实际列: %v", cols2)
	}
}

// TestAnthropicProviderInitMigratesLegacyTable 老库(缺 header_timeout_sec 等列)
// 升级到当前 schema 时,必须补齐新列且不丢老数据。
//
// 判据:
//   - 缺列的老表 + InitAnthropicProviders → 新列全部补齐
//   - 老行的 api_url / models / enabled 原样保留
//   - 新列在老行上取默认值(header_timeout_sec=0 / default_model 为空串)
//   - 再执行一次仍幂等
func TestAnthropicProviderInitMigratesLegacyTable(t *testing.T) {
	db := newAnthropicMigrateTestDB(t)

	// 造一张"升级前"的旧表:没有 default_model / api_key_encrypted / header_timeout_sec
	legacyDDL := `
	CREATE TABLE anthropic_providers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		api_url TEXT NOT NULL,
		api_key TEXT NOT NULL DEFAULT '',
		models TEXT NOT NULL DEFAULT '[]',
		aliases TEXT NOT NULL DEFAULT '[]',
		enabled INTEGER NOT NULL DEFAULT 1,
		is_default INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`
	if _, err := db.conn.Exec(legacyDDL); err != nil {
		t.Fatalf("建旧表失败: %v", err)
	}
	if _, err := db.conn.Exec(
		`INSERT INTO anthropic_providers (name, api_url, api_key, models, aliases, enabled, is_default) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"LegacyProv", "https://legacy.example.com/anthropic", "legacy-plain-key",
		`["legacy-m1","legacy-m2"]`, `[]`, 1, 1,
	); err != nil {
		t.Fatalf("插旧行失败: %v", err)
	}

	// 升级
	if err := db.InitAnthropicProviders(); err != nil {
		t.Fatalf("InitAnthropicProviders 迁移旧表失败: %v", err)
	}

	cols := anthropicProviderTableInfo(t, db)
	for _, want := range []string{"header_timeout_sec", "default_model", "api_key_encrypted"} {
		if _, ok := cols[want]; !ok {
			t.Fatalf("迁移旧表后缺少列 %q; 实际列: %v", want, cols)
		}
	}

	// 老数据必须原样保留
	var apiURL, apiKey, modelJSON string
	var enabled, isDefault, headerTimeout int
	var defaultModel, apiKeyEnc string
	err := db.conn.QueryRow(
		`SELECT api_url, api_key, models, enabled, is_default, default_model, api_key_encrypted, header_timeout_sec FROM anthropic_providers WHERE name = ?`,
		"LegacyProv",
	).Scan(&apiURL, &apiKey, &modelJSON, &enabled, &isDefault, &defaultModel, &apiKeyEnc, &headerTimeout)
	if err != nil {
		t.Fatalf("读迁移后的旧行失败: %v", err)
	}
	if apiURL != "https://legacy.example.com/anthropic" {
		t.Errorf("api_url 被改动 = %q", apiURL)
	}
	if apiKey != "legacy-plain-key" {
		t.Errorf("api_key 被改动 = %q", apiKey)
	}
	if modelJSON != `["legacy-m1","legacy-m2"]` {
		t.Errorf("models 被改动 = %q", modelJSON)
	}
	if enabled != 1 {
		t.Errorf("enabled 被改动 = %d", enabled)
	}
	if isDefault != 1 {
		t.Errorf("is_default 被改动 = %d", isDefault)
	}
	// 新列在老行上必须取默认值(不能是随机值/空行爆炸)
	if headerTimeout != 0 {
		t.Errorf("老行 header_timeout_sec 应为默认 0, 实际 %d", headerTimeout)
	}
	if defaultModel != "" {
		t.Errorf("老行 default_model 应为默认 '', 实际 %q", defaultModel)
	}
	if apiKeyEnc != "" {
		t.Errorf("老行 api_key_encrypted 应为默认 '', 实际 %q", apiKeyEnc)
	}

	// 再执行一次仍幂等
	if err := db.InitAnthropicProviders(); err != nil {
		t.Fatalf("InitAnthropicProviders 重复执行应幂等, 实际报错: %v", err)
	}
}

// TestAnthropicProviderInitSeedsOllamaHeaderTimeout 内置 Ollama 线路必须带
// header_timeout_sec=600(本地 27B 首字节窗口)。这条值是"懒迁移不能清零"的锚点:
// 旧的懒迁移写回 bug 会把这一列打成 0。
func TestAnthropicProviderInitSeedsOllamaHeaderTimeout(t *testing.T) {
	db := newAnthropicMigrateTestDB(t)
	if err := db.InitAnthropicProviders(); err != nil {
		t.Fatalf("InitAnthropicProviders: %v", err)
	}

	var got int
	err := db.conn.QueryRow(`SELECT header_timeout_sec FROM anthropic_providers WHERE name = 'Ollama'`).Scan(&got)
	if err != nil {
		t.Fatalf("读 Ollama 行的 header_timeout_sec 失败: %v", err)
	}
	if got != 600 {
		t.Errorf("Ollama header_timeout_sec = %d, want 600", got)
	}
}

// TestAnthropicProviderSelectIncludesHeaderTimeout 读路径的 SELECT 列清单必须包含
// header_timeout_sec —— 否则所有 provider 读出来的 HeaderTimeoutSec 都是 0,
// 懒迁移写回时会把管理员配置的 600 秒超时清零(这正是本次修复要防的 bug)。
//
// 这是"静态断言必须可证伪"的锚点:该常量一旦漏列,下面的断言立刻失败。
func TestAnthropicProviderSelectIncludesHeaderTimeout(t *testing.T) {
	if !strings.Contains(anthropicProviderColumns, "header_timeout_sec") {
		t.Fatalf("SELECT 列清单缺少 header_timeout_sec: %q", anthropicProviderColumns)
	}
}
