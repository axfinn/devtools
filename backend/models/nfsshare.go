package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"time"
)

func init() {
	RegisterInit("NFS分享(nfs_shares)", (*DB).InitNFSShare)
}

// NFSShare NFS 文件分享记录
type NFSShare struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	FilePath      string     `json:"file_path"`
	FileSize      int64      `json:"file_size"`
	MimeType      string     `json:"mime_type"`
	MaxViews      int        `json:"max_views"`
	Views         int        `json:"views"`
	Password      string     `json:"-"` // bcrypt hash，不对外暴露
	WatchEnabled  bool       `json:"watch_enabled"`
	RecordEnabled bool       `json:"record_enabled"`
	ShowRecordIndicator bool       `json:"show_record_indicator"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	CreatorIP     string     `json:"creator_ip"`
}

// NFSShareLog NFS 分享访问日志
type NFSShareLog struct {
	ID         int64     `json:"id"`
	ShareID    string    `json:"share_id"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	Status     string    `json:"status"`     // success / denied_views / denied_expired / file_missing / error
	BytesSent  int64     `json:"bytes_sent"`
	AudioURL   string    `json:"audio_url"`  // 录音文件路径，非空则永久保留
	AccessedAt time.Time `json:"accessed_at"`
}

// InitNFSShare 初始化 NFS 分享数据库表
func (db *DB) InitNFSShare() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS nfs_shares (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			file_path  TEXT NOT NULL,
			file_size  INTEGER NOT NULL DEFAULT 0,
			mime_type  TEXT NOT NULL DEFAULT 'application/octet-stream',
			max_views  INTEGER NOT NULL DEFAULT 1,
			views      INTEGER NOT NULL DEFAULT 0,
			password   TEXT NOT NULL DEFAULT '',
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			creator_ip TEXT,
			show_record_indicator INTEGER NOT NULL DEFAULT 1
		);
		CREATE INDEX IF NOT EXISTS idx_nfs_shares_expires_at ON nfs_shares(expires_at);
		CREATE TABLE IF NOT EXISTS nfs_share_logs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			share_id    TEXT NOT NULL,
			client_ip   TEXT,
			user_agent  TEXT,
			status      TEXT NOT NULL,
			bytes_sent  INTEGER NOT NULL DEFAULT 0,
			accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_nfs_share_logs_share_id   ON nfs_share_logs(share_id);
		CREATE INDEX IF NOT EXISTS idx_nfs_share_logs_accessed_at ON nfs_share_logs(accessed_at);
	`)
	if err != nil {
		return err
	}
	// 迁移：为旧数据库添加 password 列
	db.conn.Exec(`ALTER TABLE nfs_shares ADD COLUMN password TEXT NOT NULL DEFAULT ''`)
	// 迁移：为旧数据库添加 watch_enabled 列
	db.conn.Exec(`ALTER TABLE nfs_shares ADD COLUMN watch_enabled INTEGER NOT NULL DEFAULT 0`)
	// 迁移：为旧数据库添加 record_enabled 列
	db.conn.Exec(`ALTER TABLE nfs_shares ADD COLUMN record_enabled INTEGER NOT NULL DEFAULT 0`)
	// 迁移：为旧数据库添加 show_record_indicator 列
	db.conn.Exec(`ALTER TABLE nfs_shares ADD COLUMN show_record_indicator INTEGER NOT NULL DEFAULT 1`)
	// 迁移：为旧数据库添加 audio_url 列
	db.conn.Exec(`ALTER TABLE nfs_share_logs ADD COLUMN audio_url TEXT NOT NULL DEFAULT ''`)
	return nil
}

// CreateNFSShare 创建 NFS 分享
func (db *DB) CreateNFSShare(name, filePath, mimeType, password string, fileSize int64, maxViews int, expiresAt *time.Time, creatorIP string, recordEnabled, showRecordIndicator bool) (*NFSShare, error) {
	b := make([]byte, 4)
	rand.Read(b)
	id := hex.EncodeToString(b)

	var expiresAtVal interface{}
	if expiresAt != nil {
		expiresAtVal = *expiresAt
	}

	_, err := db.conn.Exec(
		`INSERT INTO nfs_shares (id, name, file_path, file_size, mime_type, max_views, views, password, record_enabled, show_record_indicator, expires_at, creator_ip)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?)`,
		id, name, filePath, fileSize, mimeType, maxViews, password, recordEnabled, showRecordIndicator, expiresAtVal, creatorIP,
	)
	if err != nil {
		return nil, err
	}
	return db.GetNFSShare(id)
}

// GetNFSShare 根据 ID 获取分享记录
func (db *DB) GetNFSShare(id string) (*NFSShare, error) {
	s := &NFSShare{}
	var expiresAt sql.NullTime
	err := db.conn.QueryRow(
		`SELECT id, name, file_path, file_size, mime_type, max_views, views, password, watch_enabled, record_enabled, show_record_indicator, expires_at, created_at, creator_ip
		 FROM nfs_shares WHERE id = ?`, id,
	).Scan(&s.ID, &s.Name, &s.FilePath, &s.FileSize, &s.MimeType, &s.MaxViews, &s.Views, &s.Password, &s.WatchEnabled, &s.RecordEnabled, &s.ShowRecordIndicator, &expiresAt, &s.CreatedAt, &s.CreatorIP)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		s.ExpiresAt = &expiresAt.Time
	}
	return s, nil
}

// IncrementNFSShareViews 原子自增访问次数
func (db *DB) IncrementNFSShareViews(id string) error {
	_, err := db.conn.Exec(`UPDATE nfs_shares SET views = views + 1 WHERE id = ?`, id)
	return err
}

// AddNFSShareLog 记录一条访问日志
func (db *DB) AddNFSShareLog(shareID, clientIP, userAgent, status string, bytesSent int64) error {
	_, err := db.conn.Exec(
		`INSERT INTO nfs_share_logs (share_id, client_ip, user_agent, status, bytes_sent) VALUES (?, ?, ?, ?, ?)`,
		shareID, clientIP, userAgent, status, bytesSent,
	)
	return err
}

// AppendNFSShareLogAudio 追加录音文件路径到日志（JSON 数组）
func (db *DB) AppendNFSShareLogAudio(logID int64, audioURL string) error {
	var existing string
	db.conn.QueryRow(`SELECT audio_url FROM nfs_share_logs WHERE id = ?`, logID).Scan(&existing)
	var urls []string
	if existing != "" {
		// try JSON array first
		if err := json.Unmarshal([]byte(existing), &urls); err != nil {
			// legacy single URL
			urls = []string{existing}
		}
	}
	urls = append(urls, audioURL)
	b, _ := json.Marshal(urls)
	_, err := db.conn.Exec(`UPDATE nfs_share_logs SET audio_url = ? WHERE id = ?`, string(b), logID)
	return err
}

// LastNFSShareLogID 返回最近插入的日志 ID（用于关联录音）
func (db *DB) LastNFSShareLogID(shareID, clientIP string) int64 {
	var id int64
	db.conn.QueryRow(
		`SELECT id FROM nfs_share_logs WHERE share_id = ? AND client_ip = ? ORDER BY accessed_at DESC LIMIT 1`,
		shareID, clientIP,
	).Scan(&id)
	return id
}

// GetAllNFSShares 分页获取所有分享（管理员用）;支持按状态/关键字筛选
//   status: "" / "all" 不过滤; "active" 未过期且未用完; "expired" 已过期;
//           "exhausted" 次数耗尽
//   q:      模糊匹配 name / file_path / id
func (db *DB) GetAllNFSShares(page, pageSize int, status, q string) ([]NFSShare, int, error) {
	offset := (page - 1) * pageSize

	var (
		where []string
		args  []interface{}
	)
	switch status {
	case "active":
		// 未过期(not expires_at < now) 且 未耗尽(views < max_views)
		where = append(where, "(expires_at IS NULL OR expires_at > datetime('now')) AND views < max_views")
	case "expired":
		where = append(where, "expires_at IS NOT NULL AND expires_at <= datetime('now')")
	case "exhausted":
		where = append(where, "views >= max_views")
	}
	if q != "" {
		where = append(where, "(name LIKE ? OR file_path LIKE ? OR id LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	listSQL := `SELECT id, name, file_path, file_size, mime_type, max_views, views, watch_enabled, record_enabled, show_record_indicator, expires_at, created_at, creator_ip
		FROM nfs_shares ` + whereSQL + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	listArgs := append(append([]interface{}{}, args...), pageSize, offset)
	rows, err := db.conn.Query(listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shares []NFSShare
	for rows.Next() {
		var s NFSShare
		var expiresAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.Name, &s.FilePath, &s.FileSize, &s.MimeType, &s.MaxViews, &s.Views, &s.WatchEnabled, &s.RecordEnabled, &s.ShowRecordIndicator, &expiresAt, &s.CreatedAt, &s.CreatorIP); err != nil {
			continue
		}
		if expiresAt.Valid {
			s.ExpiresAt = &expiresAt.Time
		}
		shares = append(shares, s)
	}

	countSQL := `SELECT COUNT(*) FROM nfs_shares ` + whereSQL
	var total int
	if err := db.conn.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return shares, total, nil
}

// GetNFSShareLogs 分页获取分享访问日志
func (db *DB) GetNFSShareLogs(shareID string, page, pageSize int) ([]NFSShareLog, int, error) {
	offset := (page - 1) * pageSize
	rows, err := db.conn.Query(
		`SELECT id, share_id, client_ip, user_agent, status, bytes_sent, audio_url, accessed_at
		 FROM nfs_share_logs WHERE share_id = ? ORDER BY accessed_at DESC LIMIT ? OFFSET ?`,
		shareID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []NFSShareLog
	for rows.Next() {
		var l NFSShareLog
		if err := rows.Scan(&l.ID, &l.ShareID, &l.ClientIP, &l.UserAgent, &l.Status, &l.BytesSent, &l.AudioURL, &l.AccessedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}

	var total int
	db.conn.QueryRow(`SELECT COUNT(*) FROM nfs_share_logs WHERE share_id = ?`, shareID).Scan(&total)
	return logs, total, nil
}

// DeleteNFSShare 删除分享及其无录音的访问日志;带录音的日志永久保留
// (与 CleanExpiredNFSShares 行为一致,符合"录音永久保留"语义)
func (db *DB) DeleteNFSShare(id string) error {
	if _, err := db.conn.Exec(`DELETE FROM nfs_share_logs WHERE share_id = ? AND (audio_url IS NULL OR audio_url = '')`, id); err != nil {
		return err
	}
	_, err := db.conn.Exec(`DELETE FROM nfs_shares WHERE id = ?`, id)
	return err
}

// NFSShareSummary 删除/调整分享前的预览,告诉管理员"会涉及多少东西"
type NFSShareSummary struct {
	Share         NFSShare `json:"share"`
	LogsCount     int      `json:"logs_count"`     // 总访问日志条数
	LogsWithAudio int      `json:"logs_with_audio"` // 带录音的日志条数(不会被删除)
	AudioBytes    int64    `json:"audio_bytes"`     // 关联录音文件总字节数
}

// GetNFSShareSummary 汇总分享的访问日志与录音占用,用于删除前的二次确认
func (db *DB) GetNFSShareSummary(id string) (*NFSShareSummary, error) {
	share, err := db.GetNFSShare(id)
	if err != nil {
		return nil, err
	}
	var total, withAudio int
	var audioBytes int64
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM nfs_share_logs WHERE share_id = ?`, id).Scan(&total); err != nil {
		return nil, err
	}
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM nfs_share_logs WHERE share_id = ? AND audio_url IS NOT NULL AND audio_url != ''`, id).Scan(&withAudio); err != nil {
		return nil, err
	}
	// 汇总带录音日志的 audio_url,解析 filename,累加文件大小
	rows, err := db.conn.Query(`SELECT audio_url FROM nfs_share_logs WHERE share_id = ? AND audio_url IS NOT NULL AND audio_url != ''`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var raw string
			if err := rows.Scan(&raw); err != nil {
				continue
			}
			// audio_url 可能是 JSON 数组(多录音)或单字符串
			var urls []string
			if jerr := json.Unmarshal([]byte(raw), &urls); jerr != nil {
				urls = []string{raw}
			}
			for _, u := range urls {
				// URL 形如 /api/nfsshare/<id>/record/<filename>
				idx := strings.LastIndex(u, "/")
				if idx < 0 || idx+1 >= len(u) {
					continue
				}
				path := "./data/records/" + id + "/" + u[idx+1:]
				if fi, err := os.Stat(path); err == nil {
					audioBytes += fi.Size()
				}
			}
		}
	}
	return &NFSShareSummary{
		Share:         *share,
		LogsCount:     total,
		LogsWithAudio: withAudio,
		AudioBytes:    audioBytes,
	}, nil
}

// UpdateNFSShare 更新分享的访问次数上限、过期时间与开关
func (db *DB) UpdateNFSShare(id string, maxViews int, expiresAt *time.Time, watchEnabled, recordEnabled bool) error {
	var expiresAtVal interface{}
	if expiresAt != nil {
		expiresAtVal = *expiresAt
	}
	_, err := db.conn.Exec(
		`UPDATE nfs_shares SET max_views = ?, expires_at = ?, watch_enabled = ?, record_enabled = ? WHERE id = ?`,
		maxViews, expiresAtVal, watchEnabled, recordEnabled, id,
	)
	return err
}

// ActiveUploadFilenames 返回所有 NFS 分享中引用的 __uploads__/ 文件名集合（包括已过期的）
// 用于防止清理任务误删上传文件——NFS 上传文件永不自动清理，只通过管理员手动删除分享来释放
func (db *DB) ActiveUploadFilenames() (map[string]struct{}, error) {
	rows, err := db.conn.Query(`
		SELECT file_path FROM nfs_shares
		WHERE file_path LIKE '__uploads__/%'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]struct{})
	for rows.Next() {
		var fp string
		if err := rows.Scan(&fp); err != nil {
			continue
		}
		// fp 格式为 "__uploads__/filename.ext"，提取文件名
		if idx := strings.LastIndex(fp, "/"); idx >= 0 {
			result[fp[idx+1:]] = struct{}{}
		}
	}
	return result, rows.Err()
}

// CleanExpiredNFSShares 清理已过期超过 7 天的分享（次数耗尽的不自动删除，由管理员手动清理）
// 有录音的日志永久保留，不随分享删除
func (db *DB) CleanExpiredNFSShares() (int, error) {
	rows, err := db.conn.Query(`
		SELECT id FROM nfs_shares
		WHERE expires_at IS NOT NULL AND expires_at < datetime('now', '-7 days')
	`)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()

	for _, id := range ids {
		// 只删除没有录音的日志，有录音的永久保留
		db.conn.Exec(`DELETE FROM nfs_share_logs WHERE share_id = ? AND (audio_url IS NULL OR audio_url = '')`, id)
		db.conn.Exec(`DELETE FROM nfs_shares WHERE id = ?`, id)
	}
	return len(ids), nil
}

// RecordingFilter 跨 share 查录音时的筛选条件
type RecordingFilter struct {
	ShareID string // 空字符串表示全部
	ClientIP string
	Since   string // ISO datetime,空字符串表示不限
	Until   string // ISO datetime
	Page    int
	PageSize int
}

// RecordingEntry 录音库条目，附带 share 信息（已被删的 share 也保留 NULL）
type RecordingEntry struct {
	LogID      int64     `json:"log_id"`
	ShareID    string    `json:"share_id"`
	ShareName  *string   `json:"share_name"` // share 已删时为 null
	ShareFilePath *string `json:"share_file_path"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	Status     string    `json:"status"`
	AudioURL   string    `json:"audio_url"`
	AccessedAt time.Time `json:"accessed_at"`
	ShareDeleted bool    `json:"share_deleted"`
}

// GetAllRecordings 跨 share 列出所有带 audio_url 的日志（LEFT JOIN nfs_shares，
// 所以已删除 share 的孤儿日志也能查到）。share 没了的录音就靠这个入口找回。
func (db *DB) GetAllRecordings(f RecordingFilter) ([]RecordingEntry, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 50
	}
	offset := (f.Page - 1) * f.PageSize

	var (
		where  []string
		args   []interface{}
	)
	where = append(where, "l.audio_url IS NOT NULL AND l.audio_url != ''")
	if f.ShareID != "" {
		where = append(where, "l.share_id = ?")
		args = append(args, f.ShareID)
	}
	if f.ClientIP != "" {
		where = append(where, "l.client_ip LIKE ?")
		args = append(args, "%"+f.ClientIP+"%")
	}
	if f.Since != "" {
		where = append(where, "l.accessed_at >= ?")
		args = append(args, f.Since)
	}
	if f.Until != "" {
		where = append(where, "l.accessed_at <= ?")
		args = append(args, f.Until)
	}
	whereSQL := strings.Join(where, " AND ")

	// 总数
	var total int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM nfs_share_logs l WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := `
		SELECT l.id, l.share_id, s.name, s.file_path, l.client_ip, l.user_agent,
		       l.status, l.audio_url, l.accessed_at,
		       CASE WHEN s.id IS NULL THEN 1 ELSE 0 END AS share_deleted
		FROM nfs_share_logs l
		LEFT JOIN nfs_shares s ON l.share_id = s.id
		WHERE ` + whereSQL + `
		ORDER BY l.accessed_at DESC
		LIMIT ? OFFSET ?`
	queryArgs := append(args, f.PageSize, offset)
	rows, err := db.conn.Query(q, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]RecordingEntry, 0)
	for rows.Next() {
		var e RecordingEntry
		var shareName, sharePath sql.NullString
		var shareDeleted int
		if err := rows.Scan(&e.LogID, &e.ShareID, &shareName, &sharePath, &e.ClientIP, &e.UserAgent,
			&e.Status, &e.AudioURL, &e.AccessedAt, &shareDeleted); err != nil {
			continue
		}
		if shareName.Valid {
			v := shareName.String
			e.ShareName = &v
		}
		if sharePath.Valid {
			v := sharePath.String
			e.ShareFilePath = &v
		}
		e.ShareDeleted = shareDeleted == 1
		out = append(out, e)
	}
	return out, total, nil
}
