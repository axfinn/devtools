package models

import (
	"database/sql"
	"encoding/json"
	"errors"
)

// ── AskIt 云同步数据模型 ─────────────────────────────────────────────
// 同步本质是「带身份的远端 export/import」：每条记录包成统一信封
// (id, updatedAt, deleted, data)，按 id 合并、按 updatedAt 取新（LWW），
// 删除用墓碑传播。每用户每集合维护单调递增 version 用于增量拉取。

type AskitUser struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	Verified     bool   `json:"verified"`
	CreatedAt    int64  `json:"createdAt"` // ms
}

// AskitSyncRecord 同步记录信封，data 为业务负载，后端不解析。
type AskitSyncRecord struct {
	ID        string          `json:"id"`
	UpdatedAt int64           `json:"updatedAt"`
	Deleted   bool            `json:"deleted"`
	Data      json.RawMessage `json:"data"`
}

// ErrAskitEmailTaken 邮箱已注册。
var ErrAskitEmailTaken = errors.New("email_taken")

// ErrAskitBlobTooLarge push 累计负载超过 BlobMaxBytes。
var ErrAskitBlobTooLarge = errors.New("blob_too_large")

func init() {
	RegisterInit("AskIt 云同步(askit_*)", func(db *DB) error { return db.initAskitSync() })
}

func (db *DB) initAskitSync() error {
	_, err := db.conn.Exec(`
	CREATE TABLE IF NOT EXISTS askit_users (
		id            TEXT PRIMARY KEY,
		email         TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		verified      INTEGER NOT NULL DEFAULT 0,
		created_at    INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS askit_email_codes (
		email      TEXT NOT NULL,
		code       TEXT NOT NULL,
		purpose    TEXT NOT NULL,            -- 'verify' | 'reset'
		expires_at INTEGER NOT NULL,
		PRIMARY KEY (email, purpose)
	);
	CREATE TABLE IF NOT EXISTS askit_invite_codes (
		code       TEXT PRIMARY KEY,
		created_at INTEGER NOT NULL,
		expires_at INTEGER,
		used_by    TEXT,
		used_at    INTEGER
	);
	CREATE TABLE IF NOT EXISTS askit_tokens (
		token      TEXT PRIMARY KEY,
		user_id    TEXT NOT NULL,
		kind       TEXT NOT NULL,            -- 'access' | 'refresh'
		expires_at INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_askit_tokens_user ON askit_tokens(user_id);
	CREATE TABLE IF NOT EXISTS askit_sync_records (
		user_id    TEXT NOT NULL,
		collection TEXT NOT NULL,
		record_id  TEXT NOT NULL,
		updated_at INTEGER NOT NULL,
		version    INTEGER NOT NULL,
		deleted    INTEGER NOT NULL DEFAULT 0,
		data       BLOB,
		PRIMARY KEY (user_id, collection, record_id)
	);
	CREATE INDEX IF NOT EXISTS idx_askit_sync_user_version ON askit_sync_records(user_id, version);
	CREATE TABLE IF NOT EXISTS askit_sync_versions (
		user_id     TEXT PRIMARY KEY,
		cur_version INTEGER NOT NULL DEFAULT 0
	);
	`)
	return err
}

// ── 用户 ─────────────────────────────────────────────

// CreateAskitUser 创建用户。邮箱已存在返回 ErrAskitEmailTaken。
func (db *DB) CreateAskitUser(id, email, passwordHash string, verified bool, createdAt int64) error {
	var exists int
	err := db.conn.QueryRow(`SELECT COUNT(1) FROM askit_users WHERE email = ?`, email).Scan(&exists)
	if err != nil {
		return err
	}
	if exists > 0 {
		return ErrAskitEmailTaken
	}
	v := 0
	if verified {
		v = 1
	}
	_, err = db.conn.Exec(
		`INSERT INTO askit_users (id, email, password_hash, verified, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, email, passwordHash, v, createdAt,
	)
	return err
}

func scanAskitUser(row interface{ Scan(...interface{}) error }) (*AskitUser, error) {
	u := &AskitUser{}
	var verified int
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &verified, &u.CreatedAt); err != nil {
		return nil, err
	}
	u.Verified = verified == 1
	return u, nil
}

// GetAskitUserByEmail 按邮箱查用户，不存在返回 (nil, nil)。
func (db *DB) GetAskitUserByEmail(email string) (*AskitUser, error) {
	row := db.conn.QueryRow(`SELECT id, email, password_hash, verified, created_at FROM askit_users WHERE email = ?`, email)
	u, err := scanAskitUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

// GetAskitUserByID 按 id 查用户，不存在返回 (nil, nil)。
func (db *DB) GetAskitUserByID(id string) (*AskitUser, error) {
	row := db.conn.QueryRow(`SELECT id, email, password_hash, verified, created_at FROM askit_users WHERE id = ?`, id)
	u, err := scanAskitUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (db *DB) SetAskitUserVerified(email string) error {
	_, err := db.conn.Exec(`UPDATE askit_users SET verified = 1 WHERE email = ?`, email)
	return err
}

func (db *DB) SetAskitUserPassword(email, passwordHash string) error {
	_, err := db.conn.Exec(`UPDATE askit_users SET password_hash = ? WHERE email = ?`, passwordHash, email)
	return err
}

// ── 邮箱验证码 / 重置码 ─────────────────────────────────────────────

func (db *DB) UpsertAskitEmailCode(email, code, purpose string, expiresAt int64) error {
	_, err := db.conn.Exec(
		`INSERT INTO askit_email_codes (email, code, purpose, expires_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(email, purpose) DO UPDATE SET code = excluded.code, expires_at = excluded.expires_at`,
		email, code, purpose, expiresAt,
	)
	return err
}

// ConsumeAskitEmailCode 校验并消费验证码：匹配且未过期返回 true 并删除该码。
func (db *DB) ConsumeAskitEmailCode(email, code, purpose string, now int64) (bool, error) {
	var stored string
	var expiresAt int64
	err := db.conn.QueryRow(
		`SELECT code, expires_at FROM askit_email_codes WHERE email = ? AND purpose = ?`,
		email, purpose,
	).Scan(&stored, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if stored != code || now > expiresAt {
		return false, nil
	}
	_, _ = db.conn.Exec(`DELETE FROM askit_email_codes WHERE email = ? AND purpose = ?`, email, purpose)
	return true, nil
}

// ── 邀请码 ─────────────────────────────────────────────

func (db *DB) CreateAskitInviteCode(code string, createdAt int64, expiresAt *int64) error {
	_, err := db.conn.Exec(
		`INSERT INTO askit_invite_codes (code, created_at, expires_at) VALUES (?, ?, ?)`,
		code, createdAt, expiresAt,
	)
	return err
}

// CheckAskitInviteCode 返回邀请码是否可用（存在、未使用、未过期）。
func (db *DB) CheckAskitInviteCode(code string, now int64) (bool, error) {
	var usedBy sql.NullString
	var expiresAt sql.NullInt64
	err := db.conn.QueryRow(
		`SELECT used_by, expires_at FROM askit_invite_codes WHERE code = ?`, code,
	).Scan(&usedBy, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if usedBy.Valid && usedBy.String != "" {
		return false, nil
	}
	if expiresAt.Valid && now > expiresAt.Int64 {
		return false, nil
	}
	return true, nil
}

func (db *DB) MarkAskitInviteUsed(code, userID string, usedAt int64) error {
	_, err := db.conn.Exec(
		`UPDATE askit_invite_codes SET used_by = ?, used_at = ? WHERE code = ?`,
		userID, usedAt, code,
	)
	return err
}

// ── token（不透明随机令牌，服务端可吊销）─────────────────────────────

func (db *DB) CreateAskitToken(token, userID, kind string, expiresAt, createdAt int64) error {
	_, err := db.conn.Exec(
		`INSERT INTO askit_tokens (token, user_id, kind, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		token, userID, kind, expiresAt, createdAt,
	)
	return err
}

// GetAskitTokenUser 校验 token 类型与有效期，返回 user_id；无效返回 ""。
func (db *DB) GetAskitTokenUser(token, kind string, now int64) (string, error) {
	var userID string
	var expiresAt int64
	err := db.conn.QueryRow(
		`SELECT user_id, expires_at FROM askit_tokens WHERE token = ? AND kind = ?`,
		token, kind,
	).Scan(&userID, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if now > expiresAt {
		_, _ = db.conn.Exec(`DELETE FROM askit_tokens WHERE token = ?`, token)
		return "", nil
	}
	return userID, nil
}

func (db *DB) DeleteAskitToken(token string) error {
	_, err := db.conn.Exec(`DELETE FROM askit_tokens WHERE token = ?`, token)
	return err
}

// DeleteAskitUserTokens 吊销某用户的全部 token（登出所有设备）。
func (db *DB) DeleteAskitUserTokens(userID string) error {
	_, err := db.conn.Exec(`DELETE FROM askit_tokens WHERE user_id = ?`, userID)
	return err
}

// CleanupAskitExpiredTokens 清理过期 token。
func (db *DB) CleanupAskitExpiredTokens(now int64) error {
	_, err := db.conn.Exec(`DELETE FROM askit_tokens WHERE expires_at < ?`, now)
	return err
}

// ── 同步记录(LWW + 墓碑 + 单调 version)─────────────────────────────

// AskitCollectionRecords 按 collection 分组的记录信封。
type AskitCollectionRecords map[string][]AskitSyncRecord

// GetAskitServerVersion 返回用户当前的服务端版本号（无记录返回 0）。
func (db *DB) GetAskitServerVersion(userID string) (int64, error) {
	var v int64
	err := db.conn.QueryRow(
		`SELECT cur_version FROM askit_sync_versions WHERE user_id = ?`, userID,
	).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return v, err
}

// PullAskitChanges 拉取 since 之后(version > since)的全部变更，按 collection 分组。
// 返回记录集合与本次最新 serverVersion。
func (db *DB) PullAskitChanges(userID string, since int64) (AskitCollectionRecords, int64, error) {
	serverVersion, err := db.GetAskitServerVersion(userID)
	if err != nil {
		return nil, 0, err
	}
	rows, err := db.conn.Query(
		`SELECT collection, record_id, updated_at, deleted, data
		   FROM askit_sync_records
		  WHERE user_id = ? AND version > ?
		  ORDER BY version`,
		userID, since,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := AskitCollectionRecords{}
	for rows.Next() {
		var collection string
		var rec AskitSyncRecord
		var deleted int
		var data []byte
		if err := rows.Scan(&collection, &rec.ID, &rec.UpdatedAt, &deleted, &data); err != nil {
			return nil, 0, err
		}
		rec.Deleted = deleted == 1
		if len(data) > 0 {
			rec.Data = json.RawMessage(data)
		}
		out[collection] = append(out[collection], rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, serverVersion, nil
}

// AskitPushResult push 的结果：最新版本、写入条数、需客户端回写的冲突记录。
type AskitPushResult struct {
	ServerVersion int64
	Applied       int
	Conflicts     AskitCollectionRecords
}

// PushAskitChanges 记录级 LWW 合并：
//   - 传入 updatedAt ≥ 已存 → 写入(含墓碑)，version++；
//   - 传入 updatedAt < 已存 → 丢弃，并在 conflicts 回传服务端较新记录。
//
// blobMax > 0 时校验本次写入累计 data 字节数,超限返回 (nil, ErrAskitBlobTooLarge)。
// 整个用户的写入在单事务内串行执行,保证 version 单调。
func (db *DB) PushAskitChanges(userID string, incoming AskitCollectionRecords, blobMax int64) (*AskitPushResult, error) {
	if blobMax > 0 {
		var total int64
		for _, recs := range incoming {
			for _, r := range recs {
				total += int64(len(r.Data))
			}
		}
		if total > blobMax {
			return nil, ErrAskitBlobTooLarge
		}
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var curVersion int64
	err = tx.QueryRow(`SELECT cur_version FROM askit_sync_versions WHERE user_id = ?`, userID).Scan(&curVersion)
	if errors.Is(err, sql.ErrNoRows) {
		curVersion = 0
	} else if err != nil {
		return nil, err
	}

	res := &AskitPushResult{Conflicts: AskitCollectionRecords{}}
	for collection, recs := range incoming {
		for _, rec := range recs {
			var existingUpdated int64
			var existingDeleted int
			var existingData []byte
			err := tx.QueryRow(
				`SELECT updated_at, deleted, data FROM askit_sync_records
				  WHERE user_id = ? AND collection = ? AND record_id = ?`,
				userID, collection, rec.ID,
			).Scan(&existingUpdated, &existingDeleted, &existingData)
			hasExisting := !errors.Is(err, sql.ErrNoRows)
			if err != nil && hasExisting {
				return nil, err
			}

			// 服务端较新 → 丢弃传入，回传冲突供客户端吸收。
			if hasExisting && rec.UpdatedAt < existingUpdated {
				conflict := AskitSyncRecord{
					ID:        rec.ID,
					UpdatedAt: existingUpdated,
					Deleted:   existingDeleted == 1,
				}
				if len(existingData) > 0 {
					conflict.Data = json.RawMessage(existingData)
				}
				res.Conflicts[collection] = append(res.Conflicts[collection], conflict)
				continue
			}

			curVersion++
			deleted := 0
			var data []byte
			if rec.Deleted {
				deleted = 1 // 墓碑 data 置空
			} else if len(rec.Data) > 0 {
				data = []byte(rec.Data)
			}
			_, err = tx.Exec(
				`INSERT INTO askit_sync_records (user_id, collection, record_id, updated_at, version, deleted, data)
				 VALUES (?, ?, ?, ?, ?, ?, ?)
				 ON CONFLICT(user_id, collection, record_id)
				 DO UPDATE SET updated_at = excluded.updated_at, version = excluded.version,
				               deleted = excluded.deleted, data = excluded.data`,
				userID, collection, rec.ID, rec.UpdatedAt, curVersion, deleted, data,
			)
			if err != nil {
				return nil, err
			}
			res.Applied++
		}
	}

	if res.Applied > 0 {
		_, err = tx.Exec(
			`INSERT INTO askit_sync_versions (user_id, cur_version) VALUES (?, ?)
			 ON CONFLICT(user_id) DO UPDATE SET cur_version = excluded.cur_version`,
			userID, curVersion,
		)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	res.ServerVersion = curVersion
	return res, nil
}

// GetAskitSnapshot 返回用户全部非墓碑记录,按 collection 分组(调试/迁移)。
func (db *DB) GetAskitSnapshot(userID string) (AskitCollectionRecords, int64, error) {
	serverVersion, err := db.GetAskitServerVersion(userID)
	if err != nil {
		return nil, 0, err
	}
	rows, err := db.conn.Query(
		`SELECT collection, record_id, updated_at, data
		   FROM askit_sync_records
		  WHERE user_id = ? AND deleted = 0
		  ORDER BY collection, record_id`,
		userID,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := AskitCollectionRecords{}
	for rows.Next() {
		var collection string
		var rec AskitSyncRecord
		var data []byte
		if err := rows.Scan(&collection, &rec.ID, &rec.UpdatedAt, &data); err != nil {
			return nil, 0, err
		}
		if len(data) > 0 {
			rec.Data = json.RawMessage(data)
		}
		out[collection] = append(out[collection], rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, serverVersion, nil
}

// CleanupAskitTombstones 清理早于 before 的墓碑(保留期满后物理删除)。
func (db *DB) CleanupAskitTombstones(before int64) error {
	_, err := db.conn.Exec(
		`DELETE FROM askit_sync_records WHERE deleted = 1 AND updated_at < ?`, before,
	)
	return err
}

// AskitUserOverview 一名用户的同步「元数据」视图——只看有没有备份、何时更新,
// 绝不读取 data 内容(密钥是端到端密文,服务器本就无法解密)。供管理页排障。
type AskitUserOverview struct {
	Email          string `json:"email"`
	CreatedAt      int64  `json:"createdAt"`      // ms,注册时间
	ServerVersion  int64  `json:"serverVersion"`  // 当前同步版本游标
	RecordCount    int64  `json:"recordCount"`    // 非墓碑记录条数(不含密钥)
	LastBackupAt   int64  `json:"lastBackupAt"`   // 最近一次任意记录更新时间(0 表示无备份)
	HasSecrets     bool   `json:"hasSecrets"`     // 是否存在密钥备份(secrets/__secrets__)
	SecretsUpdated int64  `json:"secretsUpdated"` // 密钥备份最近更新时间(0 表示无)
}

// GetAskitUsersOverview 汇总所有用户的备份元数据(不含任何 data 内容)。
// 密钥备份单独识别 collection='secrets' 且 record_id='__secrets__' 的非墓碑记录。
func (db *DB) GetAskitUsersOverview() ([]AskitUserOverview, error) {
	rows, err := db.conn.Query(`
		SELECT
			u.email,
			u.created_at,
			COALESCE(v.cur_version, 0) AS server_version,
			COALESCE(r.record_count, 0) AS record_count,
			COALESCE(r.last_backup_at, 0) AS last_backup_at,
			COALESCE(s.secrets_updated, 0) AS secrets_updated
		FROM askit_users u
		LEFT JOIN askit_sync_versions v ON v.user_id = u.id
		LEFT JOIN (
			SELECT user_id,
			       COUNT(1) AS record_count,
			       MAX(updated_at) AS last_backup_at
			FROM askit_sync_records
			WHERE deleted = 0
			  AND NOT (collection = 'secrets' AND record_id = '__secrets__')
			GROUP BY user_id
		) r ON r.user_id = u.id
		LEFT JOIN (
			SELECT user_id, updated_at AS secrets_updated
			FROM askit_sync_records
			WHERE deleted = 0 AND collection = 'secrets' AND record_id = '__secrets__'
		) s ON s.user_id = u.id
		ORDER BY u.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AskitUserOverview{}
	for rows.Next() {
		var o AskitUserOverview
		if err := rows.Scan(&o.Email, &o.CreatedAt, &o.ServerVersion, &o.RecordCount, &o.LastBackupAt, &o.SecretsUpdated); err != nil {
			return nil, err
		}
		o.HasSecrets = o.SecretsUpdated > 0
		out = append(out, o)
	}
	return out, rows.Err()
}


