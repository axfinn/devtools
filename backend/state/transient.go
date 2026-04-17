package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"devtools/config"

	"github.com/redis/go-redis/v9"
)

type ImageTask struct {
	ID        string          `json:"id"`
	Status    string          `json:"status"`
	Tool      string          `json:"tool,omitempty"`
	Text      string          `json:"text,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Args      json.RawMessage `json:"args,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type ChunkUploadInfo struct {
	FileID         string    `json:"file_id"`
	FileName       string    `json:"file_name"`
	TotalChunks    int       `json:"total_chunks"`
	ChunkSize      int64     `json:"chunk_size"`
	FileSize       int64     `json:"file_size"`
	UploadedChunks []int     `json:"uploaded_chunks"`
	CreatedAt      time.Time `json:"created_at"`
}

type TransientStore interface {
	AllowRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
	SaveImageTask(ctx context.Context, task *ImageTask, ttl time.Duration) error
	GetImageTask(ctx context.Context, id string) (*ImageTask, error)
	FailProcessingImageTasks(ctx context.Context, reason string, ttl time.Duration) error
	SaveChunkUpload(ctx context.Context, upload *ChunkUploadInfo, ttl time.Duration) error
	MarkChunkUploaded(ctx context.Context, fileID string, chunkIndex int, ttl time.Duration) (int, error)
	GetChunkUpload(ctx context.Context, fileID string) (*ChunkUploadInfo, error)
	DeleteChunkUpload(ctx context.Context, fileID string) error
}

type MemoryStore struct {
	rateMu       sync.Mutex
	rateRequests map[string][]time.Time

	imageMu    sync.RWMutex
	imageTasks map[string]*ImageTask

	chunkMu      sync.RWMutex
	chunkUploads map[string]*ChunkUploadInfo
}

type RedisStore struct {
	client    *redis.Client
	keyPrefix string
}

const (
	imageProcessingSetSuffix = "image_task:processing"
)

func New(cfg config.RedisConfig) (TransientStore, error) {
	if !cfg.Enabled {
		return NewMemoryStore(), nil
	}

	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		return NewMemoryStore(), errors.New("redis 已启用但未配置地址")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	prefix := strings.TrimSpace(cfg.KeyPrefix)
	if prefix == "" {
		prefix = "devtools:"
	}

	return &RedisStore{
		client:    client,
		keyPrefix: prefix,
	}, nil
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rateRequests: make(map[string][]time.Time),
		imageTasks:   make(map[string]*ImageTask),
		chunkUploads: make(map[string]*ChunkUploadInfo),
	}
}

func (s *MemoryStore) AllowRateLimit(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	s.rateMu.Lock()
	defer s.rateMu.Unlock()

	now := time.Now()
	times := s.rateRequests[key]
	valid := make([]time.Time, 0, len(times)+1)
	for _, t := range times {
		if now.Sub(t) < window {
			valid = append(valid, t)
		}
	}

	if len(valid) >= limit {
		s.rateRequests[key] = valid
		return false, nil
	}

	valid = append(valid, now)
	s.rateRequests[key] = valid
	return true, nil
}

func (s *MemoryStore) SaveImageTask(_ context.Context, task *ImageTask, _ time.Duration) error {
	if task == nil {
		return errors.New("image task is nil")
	}

	copyTask := *task
	s.imageMu.Lock()
	s.imageTasks[task.ID] = &copyTask
	s.imageMu.Unlock()
	return nil
}

func (s *MemoryStore) GetImageTask(_ context.Context, id string) (*ImageTask, error) {
	s.imageMu.RLock()
	task := s.imageTasks[id]
	s.imageMu.RUnlock()
	if task == nil {
		return nil, redis.Nil
	}
	copyTask := *task
	return &copyTask, nil
}

func (s *MemoryStore) FailProcessingImageTasks(_ context.Context, reason string, _ time.Duration) error {
	s.imageMu.Lock()
	defer s.imageMu.Unlock()
	for _, task := range s.imageTasks {
		if task.Status == "processing" || task.Status == "pending" {
			task.Status = "failed"
			task.Error = reason
		}
	}
	return nil
}

func (s *MemoryStore) SaveChunkUpload(_ context.Context, upload *ChunkUploadInfo, _ time.Duration) error {
	if upload == nil {
		return errors.New("chunk upload is nil")
	}
	copyUpload := *upload
	copyUpload.UploadedChunks = append([]int(nil), upload.UploadedChunks...)
	s.chunkMu.Lock()
	s.chunkUploads[upload.FileID] = &copyUpload
	s.chunkMu.Unlock()
	return nil
}

func (s *MemoryStore) MarkChunkUploaded(_ context.Context, fileID string, chunkIndex int, _ time.Duration) (int, error) {
	s.chunkMu.Lock()
	defer s.chunkMu.Unlock()

	upload := s.chunkUploads[fileID]
	if upload == nil {
		return 0, redis.Nil
	}
	for _, existing := range upload.UploadedChunks {
		if existing == chunkIndex {
			return len(upload.UploadedChunks), nil
		}
	}
	upload.UploadedChunks = append(upload.UploadedChunks, chunkIndex)
	return len(upload.UploadedChunks), nil
}

func (s *MemoryStore) GetChunkUpload(_ context.Context, fileID string) (*ChunkUploadInfo, error) {
	s.chunkMu.RLock()
	upload := s.chunkUploads[fileID]
	s.chunkMu.RUnlock()
	if upload == nil {
		return nil, redis.Nil
	}
	copyUpload := *upload
	copyUpload.UploadedChunks = append([]int(nil), upload.UploadedChunks...)
	return &copyUpload, nil
}

func (s *MemoryStore) DeleteChunkUpload(_ context.Context, fileID string) error {
	s.chunkMu.Lock()
	delete(s.chunkUploads, fileID)
	s.chunkMu.Unlock()
	return nil
}

func (s *RedisStore) AllowRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	fullKey := s.key("ratelimit:" + key)
	count, err := s.client.Incr(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err := s.client.Expire(ctx, fullKey, window).Err(); err != nil {
			return false, err
		}
	}
	return count <= int64(limit), nil
}

func (s *RedisStore) SaveImageTask(ctx context.Context, task *ImageTask, ttl time.Duration) error {
	if err := s.setJSON(ctx, s.imageTaskKey(task.ID), task, ttl); err != nil {
		return err
	}
	if s.isProcessingStatus(task.Status) {
		pipe := s.client.TxPipeline()
		pipe.SAdd(ctx, s.processingTasksKey(), task.ID)
		pipe.Expire(ctx, s.processingTasksKey(), ttl)
		_, err := pipe.Exec(ctx)
		return err
	}
	return s.client.SRem(ctx, s.processingTasksKey(), task.ID).Err()
}

func (s *RedisStore) GetImageTask(ctx context.Context, id string) (*ImageTask, error) {
	var task ImageTask
	if err := s.getJSON(ctx, s.imageTaskKey(id), &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *RedisStore) FailProcessingImageTasks(ctx context.Context, reason string, ttl time.Duration) error {
	taskIDs, err := s.client.SMembers(ctx, s.processingTasksKey()).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	for _, taskID := range taskIDs {
		task, err := s.GetImageTask(ctx, taskID)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return err
		}
		task.Status = "failed"
		task.Error = reason
		if err := s.SaveImageTask(ctx, task, ttl); err != nil {
			return err
		}
	}
	return s.client.Del(ctx, s.processingTasksKey()).Err()
}

func (s *RedisStore) SaveChunkUpload(ctx context.Context, upload *ChunkUploadInfo, ttl time.Duration) error {
	meta := *upload
	meta.UploadedChunks = nil
	if err := s.setJSON(ctx, s.chunkUploadMetaKey(upload.FileID), &meta, ttl); err != nil {
		return err
	}
	if len(upload.UploadedChunks) > 0 {
		members := make([]interface{}, 0, len(upload.UploadedChunks))
		for _, idx := range upload.UploadedChunks {
			members = append(members, idx)
		}
		pipe := s.client.TxPipeline()
		pipe.Del(ctx, s.chunkUploadSetKey(upload.FileID))
		pipe.SAdd(ctx, s.chunkUploadSetKey(upload.FileID), members...)
		pipe.Expire(ctx, s.chunkUploadSetKey(upload.FileID), ttl)
		_, err := pipe.Exec(ctx)
		return err
	}
	return nil
}

func (s *RedisStore) MarkChunkUploaded(ctx context.Context, fileID string, chunkIndex int, ttl time.Duration) (int, error) {
	pipe := s.client.TxPipeline()
	pipe.SAdd(ctx, s.chunkUploadSetKey(fileID), chunkIndex)
	cardCmd := pipe.SCard(ctx, s.chunkUploadSetKey(fileID))
	pipe.Expire(ctx, s.chunkUploadSetKey(fileID), ttl)
	pipe.Expire(ctx, s.chunkUploadMetaKey(fileID), ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return int(cardCmd.Val()), nil
}

func (s *RedisStore) GetChunkUpload(ctx context.Context, fileID string) (*ChunkUploadInfo, error) {
	var upload ChunkUploadInfo
	if err := s.getJSON(ctx, s.chunkUploadMetaKey(fileID), &upload); err != nil {
		return nil, err
	}
	members, err := s.client.SMembers(ctx, s.chunkUploadSetKey(fileID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	upload.UploadedChunks = make([]int, 0, len(members))
	for _, member := range members {
		var idx int
		if _, scanErr := fmt.Sscanf(member, "%d", &idx); scanErr == nil {
			upload.UploadedChunks = append(upload.UploadedChunks, idx)
		}
	}
	sort.Ints(upload.UploadedChunks)
	return &upload, nil
}

func (s *RedisStore) DeleteChunkUpload(ctx context.Context, fileID string) error {
	return s.client.Del(ctx, s.chunkUploadMetaKey(fileID), s.chunkUploadSetKey(fileID)).Err()
}

func (s *RedisStore) key(suffix string) string {
	return s.keyPrefix + suffix
}

func (s *RedisStore) imageTaskKey(id string) string {
	return s.key("image_task:" + id)
}

func (s *RedisStore) processingTasksKey() string {
	return s.key(imageProcessingSetSuffix)
}

func (s *RedisStore) chunkUploadMetaKey(fileID string) string {
	return s.key("chunk_upload:" + fileID + ":meta")
}

func (s *RedisStore) chunkUploadSetKey(fileID string) string {
	return s.key("chunk_upload:" + fileID + ":chunks")
}

func (s *RedisStore) isProcessingStatus(status string) bool {
	return status == "processing" || status == "pending"
}

func (s *RedisStore) setJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal redis value: %w", err)
	}
	return s.client.Set(ctx, key, data, ttl).Err()
}

func (s *RedisStore) getJSON(ctx context.Context, key string, dest any) error {
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal redis value: %w", err)
	}
	return nil
}
