package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

// ── 音乐库(Music Library) ──
// 面向个人的 AI 音乐收藏:上传本地音频 → 流式播放(Range)→ 元数据/歌词/封面/收藏/播放次数管理。
// 音频与封面落盘 ./data/music(容器内 /app/data 命名卷,重建不丢),元数据落 SQLite music_tracks。
// 播放/列表为公开只读;写入接口与全站个人工具保持一致(内网/个人使用,不加密码门)。

const (
	musicFileDir   = "./data/music"
	musicCoverDir  = "./data/music/covers"
	musicMaxAudio  = 200 << 20 // 单曲上限 200MB
	musicMaxCover  = 15 << 20  // 封面上限 15MB
	musicMaxLyrics = 128 << 10 // 歌词文本上限 128KB
)

// 扩展名 → Content-Type 白名单,流式响应按此返回。
var musicAudioTypes = map[string]string{
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".flac": "audio/flac",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".opus": "audio/opus",
	".webm": "audio/webm",
	".wma":  "audio/x-ms-wma",
	".aiff": "audio/aiff",
	".aif":  "audio/aiff",
	".ape":  "audio/x-ape",
}

var musicImageTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
	".avif": "image/avif",
}

type MusicHandler struct {
	db *models.DB
}

func NewMusicHandler(db *models.DB) *MusicHandler {
	return &MusicHandler{db: db}
}

func randHexName(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("f_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func sanitizeExt(name string) string {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(name)))
	if len(ext) > 12 || strings.ContainsAny(ext, "/\\") {
		return ""
	}
	return ext
}

// ── 上传 ──

// UploadTrack POST /api/music/tracks  multipart: file(必填) + title/artist/album/lyrics/note/duration
func (h *MusicHandler) UploadTrack(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少音频文件字段 file"})
		return
	}
	if fileHeader.Size > musicMaxAudio {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": fmt.Sprintf("文件超过 %dMB 上限", musicMaxAudio>>20)})
		return
	}
	ext := sanitizeExt(fileHeader.Filename)
	contentType, ok := musicAudioTypes[ext]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的音频格式,允许:mp3/wav/flac/m4a/aac/ogg/opus/webm/wma/aiff/ape"})
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		base := filepath.Base(fileHeader.Filename)
		title = strings.TrimSuffix(base, filepath.Ext(base))
	}
	lyrics := c.PostForm("lyrics")
	if len(lyrics) > musicMaxLyrics {
		c.JSON(http.StatusBadRequest, gin.H{"error": "歌词过长(上限 128KB)"})
		return
	}

	track := &models.MusicTrack{
		Title:        title,
		Artist:       strings.TrimSpace(c.PostForm("artist")),
		Album:        strings.TrimSpace(c.PostForm("album")),
		Lyrics:       lyrics,
		Note:         strings.TrimSpace(c.PostForm("note")),
		OriginalName: fileHeader.Filename,
		ContentType:  contentType,
		SizeBytes:    fileHeader.Size,
		DurationSec:  parsePositiveFloat(c.PostForm("duration")),
	}
	if err := h.db.CreateMusicTrack(track); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入数据库失败: " + err.Error()})
		return
	}

	// 文件名 = 随机 hex + 扩展名,避免路径穿越与重名。
	filename := track.ID + "_" + randHexName(8) + ext
	if err := saveMultipartFile(fileHeader, filepath.Join(musicFileDir, filename)); err != nil {
		_ = h.db.DeleteMusicTrack(track.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存音频文件失败: " + err.Error()})
		return
	}
	track.Filename = filename
	c.JSON(http.StatusCreated, gin.H{"item": track})
}

// UploadCover POST /api/music/tracks/:id/cover  multipart: cover
func (h *MusicHandler) UploadCover(c *gin.Context) {
	track, ok := h.loadTrack(c)
	if !ok {
		return
	}
	fileHeader, err := c.FormFile("cover")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少封面字段 cover"})
		return
	}
	if fileHeader.Size > musicMaxCover {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "封面超过 15MB 上限"})
		return
	}
	ext := sanitizeExt(fileHeader.Filename)
	if _, ok := musicImageTypes[ext]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的图片格式,允许:jpg/png/webp/gif/avif"})
		return
	}
	// 删旧封面
	if track.CoverFilename != "" {
		_ = os.Remove(filepath.Join(musicCoverDir, track.CoverFilename))
	}
	filename := track.ID + "_" + randHexName(8) + ext
	if err := saveMultipartFile(fileHeader, filepath.Join(musicCoverDir, filename)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存封面失败: " + err.Error()})
		return
	}
	if err := h.db.UpdateMusicTrackCover(track.ID, filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新数据库失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cover_filename": filename})
}

// ── 查询 ──

// ListTracks GET /api/music/tracks?keyword=&fav=&sort=&limit=&offset=
func (h *MusicHandler) ListTracks(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	favOnly := c.Query("fav") == "1" || c.Query("fav") == "true"
	sort := c.Query("sort")
	limit := parseIntDefault(c.Query("limit"), 100)
	offset := parseIntDefault(c.Query("offset"), 0)
	if limit > 500 {
		limit = 500
	}
	items, err := h.db.ListMusicTracks(keyword, favOnly, sort, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败: " + err.Error()})
		return
	}
	total, err := h.db.CountMusicTracks(keyword, favOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "统计失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// GetTrack GET /api/music/tracks/:id
func (h *MusicHandler) GetTrack(c *gin.Context) {
	track, ok := h.loadTrack(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": track})
}

// GetStats GET /api/music/stats
func (h *MusicHandler) GetStats(c *gin.Context) {
	stats, err := h.db.MusicTrackStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "统计失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// ── 修改 ──

// UpdateTrack PATCH /api/music/tracks/:id  JSON: title/artist/album/lyrics/note/duration_sec
func (h *MusicHandler) UpdateTrack(c *gin.Context) {
	track, ok := h.loadTrack(c)
	if !ok {
		return
	}
	var req struct {
		Title       *string  `json:"title"`
		Artist      *string  `json:"artist"`
		Album       *string  `json:"album"`
		Lyrics      *string  `json:"lyrics"`
		Note        *string  `json:"note"`
		DurationSec *float64 `json:"duration_sec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能为空"})
			return
		}
		track.Title = title
	}
	if req.Artist != nil {
		track.Artist = strings.TrimSpace(*req.Artist)
	}
	if req.Album != nil {
		track.Album = strings.TrimSpace(*req.Album)
	}
	if req.Lyrics != nil {
		if len(*req.Lyrics) > musicMaxLyrics {
			c.JSON(http.StatusBadRequest, gin.H{"error": "歌词过长(上限 128KB)"})
			return
		}
		track.Lyrics = *req.Lyrics
	}
	if req.Note != nil {
		track.Note = strings.TrimSpace(*req.Note)
	}
	if req.DurationSec != nil && *req.DurationSec > 0 {
		track.DurationSec = *req.DurationSec
	}
	if err := h.db.UpdateMusicTrackMeta(track); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": track})
}

// ToggleFavorite POST /api/music/tracks/:id/favorite  JSON: {favorite: bool} 缺省取反
func (h *MusicHandler) ToggleFavorite(c *gin.Context) {
	track, ok := h.loadTrack(c)
	if !ok {
		return
	}
	var req struct {
		Favorite *bool `json:"favorite"`
	}
	_ = c.ShouldBindJSON(&req) // body 可省略
	target := !track.Favorite
	if req.Favorite != nil {
		target = *req.Favorite
	}
	if err := h.db.SetMusicTrackFavorite(track.ID, target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorite": target})
}

// MarkPlayed POST /api/music/tracks/:id/played
func (h *MusicHandler) MarkPlayed(c *gin.Context) {
	track, ok := h.loadTrack(c)
	if !ok {
		return
	}
	if err := h.db.IncrementMusicTrackPlayCount(track.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DeleteTrack DELETE /api/music/tracks/:id — 删元数据并清理音频/封面文件
func (h *MusicHandler) DeleteTrack(c *gin.Context) {
	track, ok := h.loadTrack(c)
	if !ok {
		return
	}
	if err := h.db.DeleteMusicTrack(track.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败: " + err.Error()})
		return
	}
	if track.Filename != "" {
		_ = os.Remove(filepath.Join(musicFileDir, track.Filename))
	}
	if track.CoverFilename != "" {
		_ = os.Remove(filepath.Join(musicCoverDir, track.CoverFilename))
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── 文件服务 ──

// StreamTrack GET /api/music/stream/:id — http.ServeContent 原生支持 Range(拖动进度条/边下边播)
func (h *MusicHandler) StreamTrack(c *gin.Context) {
	track, ok := h.loadTrack(c)
	if !ok {
		return
	}
	if track.Filename == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频文件缺失"})
		return
	}
	path := filepath.Join(musicFileDir, track.Filename)
	file, err := os.Open(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频文件不存在"})
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	contentType := track.ContentType
	if contentType == "" {
		contentType = "audio/mpeg"
	}
	c.Header("Content-Type", contentType)
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "public, max-age=3600")
	http.ServeContent(c.Writer, c.Request, track.Filename, info.ModTime(), file)
}

// ServeCover GET /api/music/cover/:id
func (h *MusicHandler) ServeCover(c *gin.Context) {
	track, ok := h.loadTrack(c)
	if !ok {
		return
	}
	if track.CoverFilename == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "无封面"})
		return
	}
	path := filepath.Join(musicCoverDir, track.CoverFilename)
	file, err := os.Open(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "封面不存在"})
		return
	}
	defer file.Close()
	ext := sanitizeExt(track.CoverFilename)
	contentType := musicImageTypes[ext]
	if contentType == "" {
		contentType = "image/jpeg"
	}
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=86400")
	http.ServeContent(c.Writer, c.Request, track.CoverFilename, time.Time{}, file)
}

// ── 内部工具 ──

func (h *MusicHandler) loadTrack(c *gin.Context) (*models.MusicTrack, bool) {
	track, err := h.db.GetMusicTrack(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "曲目不存在"})
		return nil, false
	}
	return track, true
}

// saveMultipartFile 把上传文件落盘(自动建目录)。
func saveMultipartFile(fh *multipart.FileHeader, dest string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	dst, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil || v < 0 {
		return def
	}
	return v
}

func parsePositiveFloat(s string) float64 {
	if s == "" {
		return 0
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%g", &v); err != nil || v < 0 {
		return 0
	}
	return v
}
