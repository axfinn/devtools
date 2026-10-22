package models

import (
	"database/sql"
	"time"
)

func init() {
	RegisterInit("Shadowsocks规则文件(ss_rule_files)", (*DB).InitSSRuleFiles)
}

// ssRuleFileID 全局单行的固定主键。
// 本模块是「编辑一个规则文件」而非规则引擎,只有一套内容,不需要多行。
const ssRuleFileID = "default"

// SSRuleFile 是全站唯一的一份 Shadowsocks 规则文件。
//
// Source 与 Content 分开存:
//   - Source 是用户原始输入的域名/URL 列表,回显到编辑框,让用户下次还能改。
//   - Content 是按 ACL 模板生成好的正文,公开链接直接吐这个。
//
// 若只存 Content,用户再次打开页面就得反向解析自己生成的 ACL,体验很差。
//
// UpdatedAt 一律写 UTC:DDL 里的 CURRENT_TIMESTAMP 本来就是 UTC,代码里显式写时间
// 就必须跟它对齐,否则库里会出现带本地偏移的字符串,前端解析出来的时间会偏。
type SSRuleFile struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	Source    string    `json:"source"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (db *DB) InitSSRuleFiles() error {
	_, err := db.conn.Exec(`
	CREATE TABLE IF NOT EXISTS ss_rule_files (
		id TEXT PRIMARY KEY,
		token TEXT NOT NULL,
		source TEXT DEFAULT '',
		content TEXT DEFAULT '',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_ss_rule_files_token ON ss_rule_files(token);
	`)
	return err
}

// GetSSRuleFile 读取唯一一行。
// 首次调用时懒创建:生成 token 并写入空行,调用方无需关心初始化时机。
func (db *DB) GetSSRuleFile() (*SSRuleFile, error) {
	file, err := db.scanSSRuleFile()
	if err == nil {
		return file, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// 懒创建。并发首次请求可能同时插入,用 INSERT OR IGNORE 吞掉竞争,
	// 失败方重新读一次即可拿到对方写入的行。
	_, err = db.conn.Exec(`
		INSERT OR IGNORE INTO ss_rule_files (id, token, source, content, updated_at)
		VALUES (?, ?, '', '', ?)
	`, ssRuleFileID, generateID(32), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return db.scanSSRuleFile()
}

func (db *DB) scanSSRuleFile() (*SSRuleFile, error) {
	file := &SSRuleFile{}
	err := db.conn.QueryRow(`
		SELECT id, token, COALESCE(source, ''), COALESCE(content, ''), updated_at
		FROM ss_rule_files WHERE id = ?
	`, ssRuleFileID).Scan(&file.ID, &file.Token, &file.Source, &file.Content, &file.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// GetSSRuleFileByToken 按 token 查规则文件,供公开下载链接使用。
// token 不存在返回 sql.ErrNoRows。
func (db *DB) GetSSRuleFileByToken(token string) (*SSRuleFile, error) {
	if token == "" {
		return nil, sql.ErrNoRows
	}
	file := &SSRuleFile{}
	err := db.conn.QueryRow(`
		SELECT id, token, COALESCE(source, ''), COALESCE(content, ''), updated_at
		FROM ss_rule_files WHERE token = ?
	`, token).Scan(&file.ID, &file.Token, &file.Source, &file.Content, &file.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// SaveSSRuleFile 覆盖保存原始输入与生成正文。
func (db *DB) SaveSSRuleFile(source, content string) (*SSRuleFile, error) {
	if _, err := db.GetSSRuleFile(); err != nil {
		return nil, err
	}
	_, err := db.conn.Exec(`
		UPDATE ss_rule_files SET source = ?, content = ?, updated_at = ? WHERE id = ?
	`, source, content, time.Now().UTC(), ssRuleFileID)
	if err != nil {
		return nil, err
	}
	return db.scanSSRuleFile()
}

// RegenerateSSRuleToken 换发 token,旧链接立即失效。
// 用于公开链接外泄时止损 —— 否则用户唯一的办法是清库。
func (db *DB) RegenerateSSRuleToken() (*SSRuleFile, error) {
	if _, err := db.GetSSRuleFile(); err != nil {
		return nil, err
	}
	_, err := db.conn.Exec(`
		UPDATE ss_rule_files SET token = ?, updated_at = ? WHERE id = ?
	`, generateID(32), time.Now().UTC(), ssRuleFileID)
	if err != nil {
		return nil, err
	}
	return db.scanSSRuleFile()
}
