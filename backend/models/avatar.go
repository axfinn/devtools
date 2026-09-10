package models

import (
	"database/sql"
	"errors"
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
type AvatarModel struct {
	ID         string     `json:"id"`
	OwnerID    string     `json:"owner_id,omitempty"` // creator_key(轻量 owner_id);空表示共享模型
	Title      string     `json:"title"`
	Filename   string     `json:"filename"`        // data/avatar/models/<id>.glb
	Original   string     `json:"original"`        // 用户上传时的原始文件名
	Format     string     `json:"format"`          // glb / gltf
	Size       int64      `json:"size"`
	BoneCount  int        `json:"bone_count"`      // 骨骼数(前端解析时填,后端只存)
	PolyCount  int        `json:"poly_count"`      // 三角面数
	HasSkeleton bool      `json:"has_skeleton"`    // 是否含 SkinnedMesh + Skeleton
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

// AvatarClip 姿态片段 —— 动捕数据 / 单帧姿态。
type AvatarClip struct {
	ID        string     `json:"id"`
	ModelID   string     `json:"model_id"` // 可空(脱离模型独立)
	OwnerID   string     `json:"owner_id,omitempty"`
	Title     string     `json:"title"`
	FPS       int        `json:"fps"`
	FrameCount int       `json:"frame_count"`
	BoneNames string     `json:"bone_names"`  // JSON 数组
	FilePath  string     `json:"file_path"`   // data/avatar/clips/<id>.json
	Format    string     `json:"format"`      // pose
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
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

// InitAvatar 同时建 3 张表。
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
	}
	for _, stmt := range statements {
		if _, err := db.conn.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
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
		INSERT INTO avatar_models (id, owner_id, title, filename, original, format, size, bone_count, poly_count, has_skeleton, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.OwnerID, m.Title, m.Filename, m.Original, m.Format, m.Size, m.BoneCount, m.PolyCount, boolToInt(m.HasSkeleton), m.CreatedAt, m.ExpiresAt)
	return err
}

func (db *DB) GetAvatarModel(id string) (*AvatarModel, error) {
	m := &AvatarModel{}
	var ownerID, original sql.NullString
	var expiresAt sql.NullTime
	err := db.conn.QueryRow(`
		SELECT id, owner_id, title, filename, original, format, size, bone_count, poly_count, has_skeleton, created_at, expires_at
		FROM avatar_models WHERE id = ?
	`, id).Scan(&m.ID, &ownerID, &m.Title, &m.Filename, &original, &m.Format, &m.Size, &m.BoneCount, &m.PolyCount, &m.HasSkeleton, &m.CreatedAt, &expiresAt)
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

// ListAvatarModels 分页列出 —— 系统共享(owner_id=NULL 或 owner_id='')放前面。
func (db *DB) ListAvatarModels(ownerID string, limit, offset int) ([]*AvatarModel, error) {
	rows, err := db.conn.Query(`
		SELECT id, owner_id, title, filename, original, format, size, bone_count, poly_count, has_skeleton, created_at, expires_at
		FROM avatar_models
		WHERE (owner_id IS NULL OR owner_id = '' OR owner_id = ?)
		  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY (CASE WHEN owner_id IS NULL OR owner_id = '' THEN 0 ELSE 1 END), created_at DESC
		LIMIT ? OFFSET ?
	`, ownerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*AvatarModel, 0, limit)
	for rows.Next() {
		m := &AvatarModel{}
		var ownerID, original sql.NullString
		var expiresAt sql.NullTime
		if err := rows.Scan(&m.ID, &ownerID, &m.Title, &m.Filename, &original, &m.Format, &m.Size, &m.BoneCount, &m.PolyCount, &m.HasSkeleton, &m.CreatedAt, &expiresAt); err != nil {
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
		INSERT INTO avatar_clips (id, model_id, owner_id, title, fps, frame_count, bone_names, file_path, format, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.ModelID, c.OwnerID, c.Title, c.FPS, c.FrameCount, c.BoneNames, c.FilePath, c.Format, c.CreatedAt, c.ExpiresAt)
	return err
}

func (db *DB) GetAvatarClip(id string) (*AvatarClip, error) {
	c := &AvatarClip{}
	var modelID, ownerID sql.NullString
	var expiresAt sql.NullTime
	err := db.conn.QueryRow(`
		SELECT id, model_id, owner_id, title, fps, frame_count, bone_names, file_path, format, created_at, expires_at
		FROM avatar_clips WHERE id = ?
	`, id).Scan(&c.ID, &modelID, &ownerID, &c.Title, &c.FPS, &c.FrameCount, &c.BoneNames, &c.FilePath, &c.Format, &c.CreatedAt, &expiresAt)
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