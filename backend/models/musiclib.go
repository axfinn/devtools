package models

import (
	"errors"
	"time"
)

func init() {
	RegisterInit("音乐库(music_tracks)", (*DB).InitMusicTracks)
}

// MusicTrack 音乐库曲目。音频与封面文件落盘 ./data/music(容器内挂 /app/data 持久卷),
// 数据库存元数据;lyrics 支持纯文本或 LRC 格式,由前端播放器解析同步。
type MusicTrack struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Artist        string    `json:"artist"`
	Album         string    `json:"album"`
	Lyrics        string    `json:"lyrics"`
	Note          string    `json:"note"`
	Filename      string    `json:"filename"`
	OriginalName  string    `json:"original_name"`
	ContentType   string    `json:"content_type"`
	CoverFilename string    `json:"cover_filename"`
	SizeBytes     int64     `json:"size_bytes"`
	DurationSec   float64   `json:"duration_sec"`
	Favorite      bool      `json:"favorite"`
	PlayCount     int       `json:"play_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (db *DB) InitMusicTracks() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS music_tracks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			artist TEXT DEFAULT '',
			album TEXT DEFAULT '',
			lyrics TEXT DEFAULT '',
			note TEXT DEFAULT '',
			filename TEXT NOT NULL,
			original_name TEXT DEFAULT '',
			content_type TEXT DEFAULT '',
			cover_filename TEXT DEFAULT '',
			size_bytes INTEGER DEFAULT 0,
			duration_sec REAL DEFAULT 0,
			favorite INTEGER DEFAULT 0,
			play_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_music_tracks_created_at ON music_tracks(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_music_tracks_favorite ON music_tracks(favorite, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_music_tracks_play_count ON music_tracks(play_count DESC);
	`)
	return err
}

const musicTrackColumns = `id, title, artist, album, lyrics, note, filename, original_name, content_type, cover_filename, size_bytes, duration_sec, favorite, play_count, created_at, updated_at`

func scanMusicTrack(row interface{ Scan(...any) error }) (*MusicTrack, error) {
	t := &MusicTrack{}
	var favorite int
	err := row.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.Lyrics, &t.Note, &t.Filename,
		&t.OriginalName, &t.ContentType, &t.CoverFilename, &t.SizeBytes, &t.DurationSec,
		&favorite, &t.PlayCount, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Favorite = favorite != 0
	return t, nil
}

func (db *DB) CreateMusicTrack(t *MusicTrack) error {
	if t == nil {
		return errors.New("track is nil")
	}
	if t.ID == "" {
		t.ID = "mus_" + generateID(12)
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Title == "" {
		t.Title = "未命名"
	}
	_, err := db.conn.Exec(`
		INSERT INTO music_tracks (
			id, title, artist, album, lyrics, note, filename, original_name, content_type,
			cover_filename, size_bytes, duration_sec, favorite, play_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.Title, t.Artist, t.Album, t.Lyrics, t.Note, t.Filename, t.OriginalName,
		t.ContentType, t.CoverFilename, t.SizeBytes, t.DurationSec, boolToInt(t.Favorite),
		t.PlayCount, t.CreatedAt, t.UpdatedAt)
	return err
}

func (db *DB) GetMusicTrack(id string) (*MusicTrack, error) {
	row := db.conn.QueryRow(`SELECT `+musicTrackColumns+` FROM music_tracks WHERE id = ?`, id)
	t, err := scanMusicTrack(row)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ListMusicTracks 排序只接受白名单值:created(最新)|title(标题)|plays(最多播放)。
func (db *DB) ListMusicTracks(keyword string, favOnly bool, sort string, limit, offset int) ([]*MusicTrack, error) {
	query := `SELECT ` + musicTrackColumns + ` FROM music_tracks WHERE 1 = 1`
	args := make([]any, 0, 4)
	if keyword != "" {
		query += ` AND (title LIKE ? OR artist LIKE ? OR album LIKE ? OR note LIKE ?)`
		like := "%" + keyword + "%"
		args = append(args, like, like, like, like)
	}
	if favOnly {
		query += ` AND favorite = 1`
	}
	switch sort {
	case "title":
		query += ` ORDER BY title COLLATE NOCASE ASC`
	case "plays":
		query += ` ORDER BY play_count DESC, created_at DESC`
	default:
		query += ` ORDER BY created_at DESC`
	}
	if limit <= 0 {
		limit = 50
	}
	query += ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*MusicTrack, 0)
	for rows.Next() {
		t, err := scanMusicTrack(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (db *DB) CountMusicTracks(keyword string, favOnly bool) (int, error) {
	query := `SELECT COUNT(*) FROM music_tracks WHERE 1 = 1`
	args := make([]any, 0, 2)
	if keyword != "" {
		query += ` AND (title LIKE ? OR artist LIKE ? OR album LIKE ? OR note LIKE ?)`
		like := "%" + keyword + "%"
		args = append(args, like, like, like, like)
	}
	if favOnly {
		query += ` AND favorite = 1`
	}
	var total int
	err := db.conn.QueryRow(query, args...).Scan(&total)
	return total, err
}

type MusicStats struct {
	TotalCount    int     `json:"total_count"`
	FavoriteCount int     `json:"favorite_count"`
	TotalDuration float64 `json:"total_duration"`
	TotalBytes    int64   `json:"total_bytes"`
	TotalPlays    int     `json:"total_plays"`
}

func (db *DB) MusicTrackStats() (*MusicStats, error) {
	s := &MusicStats{}
	err := db.conn.QueryRow(`
		SELECT COUNT(*),
			COALESCE(SUM(favorite), 0),
			COALESCE(SUM(duration_sec), 0),
			COALESCE(SUM(size_bytes), 0),
			COALESCE(SUM(play_count), 0)
		FROM music_tracks
	`).Scan(&s.TotalCount, &s.FavoriteCount, &s.TotalDuration, &s.TotalBytes, &s.TotalPlays)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// UpdateMusicTrackMeta 更新可编辑字段(标题/歌手/专辑/歌词/备注/时长)。
func (db *DB) UpdateMusicTrackMeta(t *MusicTrack) error {
	if t == nil || t.ID == "" {
		return errors.New("track is nil")
	}
	t.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE music_tracks
		SET title = ?, artist = ?, album = ?, lyrics = ?, note = ?, duration_sec = ?, updated_at = ?
		WHERE id = ?
	`, t.Title, t.Artist, t.Album, t.Lyrics, t.Note, t.DurationSec, t.UpdatedAt, t.ID)
	return err
}

func (db *DB) UpdateMusicTrackCover(id, coverFilename string) error {
	_, err := db.conn.Exec(`
		UPDATE music_tracks SET cover_filename = ?, updated_at = ? WHERE id = ?
	`, coverFilename, time.Now(), id)
	return err
}

func (db *DB) SetMusicTrackFavorite(id string, fav bool) error {
	_, err := db.conn.Exec(`
		UPDATE music_tracks SET favorite = ?, updated_at = ? WHERE id = ?
	`, boolToInt(fav), time.Now(), id)
	return err
}

func (db *DB) IncrementMusicTrackPlayCount(id string) error {
	_, err := db.conn.Exec(`
		UPDATE music_tracks SET play_count = play_count + 1, updated_at = updated_at WHERE id = ?
	`, id)
	return err
}

func (db *DB) DeleteMusicTrack(id string) error {
	_, err := db.conn.Exec(`DELETE FROM music_tracks WHERE id = ?`, id)
	return err
}
