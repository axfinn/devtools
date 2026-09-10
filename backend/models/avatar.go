package models

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"
)

func init() {
	RegisterInit("虚拟形象(avatar_models/avatar_clips/avatar_me_assets)", (*DB).InitAvatar)
}

// 媒体类型常量 —— 现阶段只支持 glTF/glb;FBX v1 不做(前端 FBXLoader 走纯前端路径)。
const (
	AvatarFormatGLB  = "glb"
	AvatarFormatGLTF = "gltf"
	AvatarFormatPose = "pose"
)

var ErrAvatarModelNotFound = errors.New("avatar model not found")
var ErrAvatarClipNotFound = errors.New("avatar clip not found")
var ErrAvatarMeAssetNotFound = errors.New("avatar me asset not found")

// AvatarModel 模型库条目 —— 全局共享 + 用户上传;owner_id=NULL 表示系统/社区共享。
// P1 起携带 password_hash(bcrypt)对齐 excalidraw 的双因子鉴权 —— creator_key + password 都要对才能删。
type AvatarModel struct {
	ID           string     `json:"id"`
	OwnerID      string     `json:"owner_id,omitempty"` // creator_key(轻量 owner_id);空表示共享模型
	Title        string     `json:"title"`
	Filename     string     `json:"filename"`     // data/avatar/models/<id>.glb
	Original     string     `json:"original"`     // 用户上传时的原始文件名
	Format       string     `json:"format"`       // glb / gltf
	Size         int64      `json:"size"`
	BoneCount    int        `json:"bone_count"`   // 骨骼数(前端解析时填,后端只存)
	PolyCount    int        `json:"poly_count"`   // 三角面数
	HasSkeleton  bool       `json:"has_skeleton"` // 是否含 SkinnedMesh + Skeleton
	PasswordHash string     `json:"-"`            // bcrypt;json 永不返回(R5 鉴权对齐 excalidraw)
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

// AvatarClip 姿态片段 —— 动捕数据 / 单帧姿态。
type AvatarClip struct {
	ID           string     `json:"id"`
	ModelID      string     `json:"model_id"` // 可空(脱离模型独立)
	OwnerID      string     `json:"owner_id,omitempty"`
	Title        string     `json:"title"`
	FPS          int        `json:"fps"`
	FrameCount   int        `json:"frame_count"`
	BoneNames    string     `json:"bone_names"` // JSON 数组
	FilePath     string     `json:"file_path"`  // data/avatar/clips/<id>.json
	Format       string     `json:"format"`     // pose
	PasswordHash string     `json:"-"`          // bcrypt;json 永不返回
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

// AvatarShare 分享短链 —— 把一个 clip / model 打包成只读 code;不泄露原 owner_id;
// 通过 GET /api/avatar/share/:code 取到 target_type + target_id,前端再决定要不要拿内容。
// code 是 8 字节 hex(16 chars) —— 64-bit 空间,实际可注册数量远小于短链池,够用。
type AvatarShare struct {
	Code        string     `json:"code"`
	TargetType  string     `json:"target_type"`  // "model" / "clip"
	TargetID    string     `json:"target_id"`
	OwnerID     string     `json:"-"`             // 创建者;json 不返回
	PasswordHash string    `json:"-"`             // 访问密码(可选);空表示无密码
	HitCount    int        `json:"hit_count"`     // 已访问次数
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// AvatarMeAsset 我的资产 —— 用户在调试时标记"我用过的模型",跨设备复用。
type AvatarMeAsset struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	ModelID   string    `json:"model_id"`
	Title     string    `json:"title"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// InitAvatar 同时建 3 张表 + P1 加的 avatar_shares 表。
// CREATE 里直接带 password_hash 列(P1 起);旧 DB 用 ALTER 加列幂等迁移。
func (db *DB) InitAvatar() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS avatar_models (
			id TEXT PRIMARY KEY,
			owner_id TEXT,
			title TEXT NOT NULL DEFAULT '',
			filename TEXT NOT NULL,
			original TEXT NOT NULL DEFAULT '',
			format TEXT NOT NULL DEFAULT 'glb',
			size INTEGER NOT NULL DEFAULT 0,
			bone_count INTEGER NOT NULL DEFAULT 0,
			poly_count INTEGER NOT NULL DEFAULT 0,
			has_skeleton INTEGER NOT NULL DEFAULT 0,
			password_hash TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_avatar_models_owner ON avatar_models(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_avatar_models_expires ON avatar_models(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_avatar_models_created ON avatar_models(created_at)`,

		`CREATE TABLE IF NOT EXISTS avatar_clips (
			id TEXT PRIMARY KEY,
			model_id TEXT,
			owner_id TEXT,
			title TEXT NOT NULL DEFAULT '',
			fps INTEGER NOT NULL DEFAULT 30,
			frame_count INTEGER NOT NULL DEFAULT 0,
			bone_names TEXT NOT NULL DEFAULT '[]',
			file_path TEXT NOT NULL,
			format TEXT NOT NULL DEFAULT 'pose',
			password_hash TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_avatar_clips_model ON avatar_clips(model_id)`,
		`CREATE INDEX IF NOT EXISTS idx_avatar_clips_owner ON avatar_clips(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_avatar_clips_expires ON avatar_clips(expires_at)`,

		`CREATE TABLE IF NOT EXISTS avatar_me_assets (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			model_id TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			note TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_avatar_me_owner ON avatar_me_assets(owner_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_avatar_me_unique ON avatar_me_assets(owner_id, model_id)`,

		`CREATE TABLE IF NOT EXISTS avatar_shares (
			code TEXT PRIMARY KEY,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			owner_id TEXT,
			password_hash TEXT NOT NULL DEFAULT '',
			hit_count INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_avatar_shares_target ON avatar_shares(target_type, target_id)`,
		`CREATE INDEX IF NOT EXISTS idx_avatar_shares_expires ON avatar_shares(expires_at)`,
	}
	for _, stmt := range statements {
		if _, err := db.conn.Exec(stmt); err != nil {
			return err
		}
	}
	// 幂等迁移:为旧版 DB(P0 没建 password_hash 列)补列。SQLite 没有
	// ADD COLUMN IF NOT EXISTS,所以用 PRAGMA 探后再 ALTER;列已存在则跳过。
	if err := db.addAvatarColumnIfMissing("avatar_models", "password_hash", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := db.addAvatarColumnIfMissing("avatar_clips", "password_hash", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return nil
}

// addAvatarColumnIfMissing —— 用 PRAGMA table_info 看列是否存在;不存在才 ALTER ADD COLUMN。
func (db *DB) addAvatarColumnIfMissing(table, column, decl string) error {
	rows, err := db.conn.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil // 已存在,跳过
		}
	}
	_, err = db.conn.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + decl)
	return err
}

// --- avatar_models CRUD ---

func (db *DB) CreateAvatarModel(m *AvatarModel) error {
	if m.ID == "" {
		m.ID = generateID(10)
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	_, err := db.conn.Exec(`
		INSERT INTO avatar_models (id, owner_id, title, filename, original, format, size, bone_count, poly_count, has_skeleton, password_hash, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.OwnerID, m.Title, m.Filename, m.Original, m.Format, m.Size, m.BoneCount, m.PolyCount, boolToInt(m.HasSkeleton), m.PasswordHash, m.CreatedAt, m.ExpiresAt)
	return err
}

func (db *DB) GetAvatarModel(id string) (*AvatarModel, error) {
	m := &AvatarModel{}
	var ownerID, original sql.NullString
	var expiresAt sql.NullTime
	err := db.conn.QueryRow(`
		SELECT id, owner_id, title, filename, original, format, size, bone_count, poly_count, has_skeleton, password_hash, created_at, expires_at
		FROM avatar_models WHERE id = ?
	`, id).Scan(&m.ID, &ownerID, &m.Title, &m.Filename, &original, &m.Format, &m.Size, &m.BoneCount, &m.PolyCount, &m.HasSkeleton, &m.PasswordHash, &m.CreatedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, ErrAvatarModelNotFound
	}
	if err != nil {
		return nil, err
	}
	m.OwnerID = ownerID.String
	m.Original = original.String
	if expiresAt.Valid {
		t := expiresAt.Time
		m.ExpiresAt = &t
	}
	return m, nil
}

// ListAvatarModels 分页列出 —— 三种模式:
//   - ownerID == "": 全库(共享 + 所有用户的),系统共享放前面
//   - ownerID == "alice": 共享 + alice 自己的
//   - 共享 = owner_id IS NULL OR owner_id = ''
// 系统共享(owner_id=NULL 或 owner_id='')放前面;同 owner 内按时间倒序。
func (db *DB) ListAvatarModels(ownerID string, limit, offset int) ([]*AvatarModel, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if ownerID == "" {
		rows, err = db.conn.Query(`
			SELECT id, owner_id, title, filename, original, format, size, bone_count, poly_count, has_skeleton, password_hash, created_at, expires_at
			FROM avatar_models
			WHERE (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
			ORDER BY (CASE WHEN owner_id IS NULL OR owner_id = '' THEN 0 ELSE 1 END), created_at DESC
			LIMIT ? OFFSET ?
		`, limit, offset)
	} else {
		rows, err = db.conn.Query(`
			SELECT id, owner_id, title, filename, original, format, size, bone_count, poly_count, has_skeleton, password_hash, created_at, expires_at
			FROM avatar_models
			WHERE (owner_id IS NULL OR owner_id = '' OR owner_id = ?)
			  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
			ORDER BY (CASE WHEN owner_id IS NULL OR owner_id = '' THEN 0 ELSE 1 END), created_at DESC
			LIMIT ? OFFSET ?
		`, ownerID, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*AvatarModel, 0, limit)
	for rows.Next() {
		m := &AvatarModel{}
		var ownerID, original sql.NullString
		var expiresAt sql.NullTime
		if err := rows.Scan(&m.ID, &ownerID, &m.Title, &m.Filename, &original, &m.Format, &m.Size, &m.BoneCount, &m.PolyCount, &m.HasSkeleton, &m.PasswordHash, &m.CreatedAt, &expiresAt); err != nil {
			return nil, err
		}
		m.OwnerID = ownerID.String
		m.Original = original.String
		if expiresAt.Valid {
			t := expiresAt.Time
			m.ExpiresAt = &t
		}
		out = append(out, m)
	}
	return out, nil
}

func (db *DB) DeleteAvatarModel(id string) error {
	res, err := db.conn.Exec(`DELETE FROM avatar_models WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAvatarModelNotFound
	}
	return nil
}

// ListExpiredAvatarModels 列出所有 expires_at 已过期的模型,返回 (id, filename) 对,
// 供 cleanup.go 串行删除磁盘文件后批量 DELETE 行。
func (db *DB) ListExpiredAvatarModels() ([]struct {
	ID       string
	Filename string
}, error) {
	rows, err := db.conn.Query(`
		SELECT id, filename FROM avatar_models
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		ID       string
		Filename string
	}
	for rows.Next() {
		var pair struct {
			ID       string
			Filename string
		}
		if err := rows.Scan(&pair.ID, &pair.Filename); err != nil {
			return nil, err
		}
		out = append(out, pair)
	}
	return out, nil
}

// CleanExpiredAvatarModels 删除过期 avatar_models 记录 + 对应磁盘文件,
// 避免 TTL 名存实亡。
func (db *DB) CleanExpiredAvatarModels() (int, error) {
	pairs, err := db.ListExpiredAvatarModels()
	if err != nil {
		return 0, err
	}
	if len(pairs) == 0 {
		return 0, nil
	}
	ids := make([]string, 0, len(pairs))
	for _, p := range pairs {
		ids = append(ids, p.ID)
		if p.Filename != "" {
			if err := os.Remove(p.Filename); err != nil && !os.IsNotExist(err) {
				// 文件缺失/权限错误不阻断 DB 清理;DB 行必须清,否则下次还会命中。
				fmt.Fprintf(os.Stderr, "[avatar] 删除过期模型文件失败 %s: %v\n", p.Filename, err)
			}
		}
	}
	// 批量删除(DB 行),用 IN (...) 一次性提交。
	placeholders := make([]byte, 0, len(ids)*2)
	args := make([]interface{}, 0, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders = append(placeholders, ',')
		}
		placeholders = append(placeholders, '?')
		args = append(args, id)
	}
	res, err := db.conn.Exec(
		"DELETE FROM avatar_models WHERE id IN ("+string(placeholders)+") AND expires_at IS NOT NULL AND expires_at < ?",
		append(args, time.Now())...,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// --- avatar_clips CRUD ---

func (db *DB) CreateAvatarClip(c *AvatarClip) error {
	if c.ID == "" {
		c.ID = generateID(10)
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	if c.BoneNames == "" {
		c.BoneNames = "[]"
	}
	_, err := db.conn.Exec(`
		INSERT INTO avatar_clips (id, model_id, owner_id, title, fps, frame_count, bone_names, file_path, format, password_hash, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.ModelID, c.OwnerID, c.Title, c.FPS, c.FrameCount, c.BoneNames, c.FilePath, c.Format, c.PasswordHash, c.CreatedAt, c.ExpiresAt)
	return err
}

func (db *DB) GetAvatarClip(id string) (*AvatarClip, error) {
	c := &AvatarClip{}
	var modelID, ownerID sql.NullString
	var expiresAt sql.NullTime
	err := db.conn.QueryRow(`
		SELECT id, model_id, owner_id, title, fps, frame_count, bone_names, file_path, format, password_hash, created_at, expires_at
		FROM avatar_clips WHERE id = ?
	`, id).Scan(&c.ID, &modelID, &ownerID, &c.Title, &c.FPS, &c.FrameCount, &c.BoneNames, &c.FilePath, &c.Format, &c.PasswordHash, &c.CreatedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, ErrAvatarClipNotFound
	}
	if err != nil {
		return nil, err
	}
	c.ModelID = modelID.String
	c.OwnerID = ownerID.String
	if expiresAt.Valid {
		t := expiresAt.Time
		c.ExpiresAt = &t
	}
	return c, nil
}

func (db *DB) DeleteAvatarClip(id string) error {
	res, err := db.conn.Exec(`DELETE FROM avatar_clips WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAvatarClipNotFound
	}
	return nil
}

// ListExpiredAvatarClips 列出所有 expires_at 已过期的 clip。
func (db *DB) ListExpiredAvatarClips() ([]struct {
	ID       string
	FilePath string
}, error) {
	rows, err := db.conn.Query(`
		SELECT id, file_path FROM avatar_clips
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		ID       string
		FilePath string
	}
	for rows.Next() {
		var pair struct {
			ID       string
			FilePath string
		}
		if err := rows.Scan(&pair.ID, &pair.FilePath); err != nil {
			return nil, err
		}
		out = append(out, pair)
	}
	return out, nil
}

// CleanExpiredAvatarClips 删除过期 avatar_clips 记录 + 磁盘 .json 文件。
func (db *DB) CleanExpiredAvatarClips() (int, error) {
	pairs, err := db.ListExpiredAvatarClips()
	if err != nil {
		return 0, err
	}
	if len(pairs) == 0 {
		return 0, nil
	}
	ids := make([]string, 0, len(pairs))
	for _, p := range pairs {
		ids = append(ids, p.ID)
		if p.FilePath != "" {
			if err := os.Remove(p.FilePath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "[avatar] 删除过期 clip 文件失败 %s: %v\n", p.FilePath, err)
			}
		}
	}
	placeholders := make([]byte, 0, len(ids)*2)
	args := make([]interface{}, 0, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders = append(placeholders, ',')
		}
		placeholders = append(placeholders, '?')
		args = append(args, id)
	}
	res, err := db.conn.Exec(
		"DELETE FROM avatar_clips WHERE id IN ("+string(placeholders)+") AND expires_at IS NOT NULL AND expires_at < ?",
		append(args, time.Now())...,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// --- avatar_me_assets CRUD ---

func (db *DB) UpsertAvatarMeAsset(a *AvatarMeAsset) error {
	if a.ID == "" {
		a.ID = generateID(10)
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	_, err := db.conn.Exec(`
		INSERT INTO avatar_me_assets (id, owner_id, model_id, title, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(owner_id, model_id) DO UPDATE SET
			title = excluded.title,
			note = excluded.note
	`, a.ID, a.OwnerID, a.ModelID, a.Title, a.Note, a.CreatedAt)
	return err
}

func (db *DB) ListAvatarMeAssets(ownerID string) ([]*AvatarMeAsset, error) {
	rows, err := db.conn.Query(`
		SELECT id, owner_id, model_id, title, note, created_at
		FROM avatar_me_assets WHERE owner_id = ?
		ORDER BY created_at DESC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*AvatarMeAsset, 0)
	for rows.Next() {
		a := &AvatarMeAsset{}
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.ModelID, &a.Title, &a.Note, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (db *DB) DeleteAvatarMeAsset(id, ownerID string) error {
	res, err := db.conn.Exec(`DELETE FROM avatar_me_assets WHERE id = ? AND owner_id = ?`, id, ownerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAvatarMeAssetNotFound
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- avatar_shares CRUD (P1 起) ---

var ErrAvatarShareNotFound = errors.New("avatar share not found")

// CreateAvatarShare —— code 必须已生成;INSERT 即可。
func (db *DB) CreateAvatarShare(s *AvatarShare) error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	_, err := db.conn.Exec(`
		INSERT INTO avatar_shares (code, target_type, target_id, owner_id, password_hash, hit_count, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, s.Code, s.TargetType, s.TargetID, s.OwnerID, s.PasswordHash, s.HitCount, s.CreatedAt, s.ExpiresAt)
	return err
}

// GetAvatarShare 按 code 查;过期的不直接过滤(让 handler 决定 410 / 自动清理)。
func (db *DB) GetAvatarShare(code string) (*AvatarShare, error) {
	s := &AvatarShare{}
	var ownerID sql.NullString
	var expiresAt sql.NullTime
	err := db.conn.QueryRow(`
		SELECT code, target_type, target_id, owner_id, password_hash, hit_count, created_at, expires_at
		FROM avatar_shares WHERE code = ?
	`, code).Scan(&s.Code, &s.TargetType, &s.TargetID, &ownerID, &s.PasswordHash, &s.HitCount, &s.CreatedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, ErrAvatarShareNotFound
	}
	if err != nil {
		return nil, err
	}
	s.OwnerID = ownerID.String
	if expiresAt.Valid {
		t := expiresAt.Time
		s.ExpiresAt = &t
	}
	return s, nil
}

// IncrementAvatarShareHit 访问计数 +1 —— 异步调用也可,失败不影响主流程。
func (db *DB) IncrementAvatarShareHit(code string) {
	_, _ = db.conn.Exec(`UPDATE avatar_shares SET hit_count = hit_count + 1 WHERE code = ?`, code)
}

// DeleteAvatarShare —— 只有创建者能删;用 owner_id + code 双重 key 防越权。
func (db *DB) DeleteAvatarShare(code, ownerID string) error {
	res, err := db.conn.Exec(`DELETE FROM avatar_shares WHERE code = ? AND (owner_id = ? OR owner_id IS NULL OR owner_id = '')`, code, ownerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAvatarShareNotFound
	}
	return nil
}

// ListExpiredAvatarShares 列出过期 share code,供 cleanup goroutine 清表。
// 注意:share 本身不占磁盘,过期直接删 DB 行即可。
func (db *DB) ListExpiredAvatarShares() ([]string, error) {
	rows, err := db.conn.Query(`
		SELECT code FROM avatar_shares
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, nil
}

// CleanExpiredAvatarShares —— 删行,不碰磁盘(share 不落盘)。
func (db *DB) CleanExpiredAvatarShares() (int, error) {
	codes, err := db.ListExpiredAvatarShares()
	if err != nil {
		return 0, err
	}
	if len(codes) == 0 {
		return 0, nil
	}
	placeholders := make([]byte, 0, len(codes)*2)
	args := make([]interface{}, 0, len(codes))
	for i, code := range codes {
		if i > 0 {
			placeholders = append(placeholders, ',')
		}
		placeholders = append(placeholders, '?')
		args = append(args, code)
	}
	res, err := db.conn.Exec(
		"DELETE FROM avatar_shares WHERE code IN ("+string(placeholders)+") AND expires_at IS NOT NULL AND expires_at < ?",
		append(args, time.Now())...,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}