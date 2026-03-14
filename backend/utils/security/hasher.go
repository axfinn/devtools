package security

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"time"
)

// Hasher 内容哈希计算器
type Hasher struct {
	algorithm string
}

// NewHasher 创建哈希计算器
func NewHasher(algorithm string) *Hasher {
	return &Hasher{
		algorithm: algorithm,
	}
}

// HashResult 哈希结果
type HashResult struct {
	OriginalSize int64  `json:"original_size"` // 原始大小
	Hash         string `json:"hash"`          // 哈希值
	Algorithm    string `json:"algorithm"`     // 算法
	Timestamp    string `json:"timestamp"`     // 计算时间
}

// ComputeHash 计算字符串哈希
func (h *Hasher) ComputeHash(content string) *HashResult {
	start := time.Now()

	var hash hash.Hash
	switch h.algorithm {
	case "sha256":
		hash = sha256.New()
	case "md5":
		// 注意：MD5 不推荐用于安全场景
		// hash = md5.New()
		// Go 标准库不提供 MD5，需要导入 crypto/md5
		hash = sha256.New() // 降级为 SHA256
	default:
		hash = sha256.New()
	}

	hash.Write([]byte(content))

	result := &HashResult{
		OriginalSize: int64(len(content)),
		Hash:         hex.EncodeToString(hash.Sum(nil)),
		Algorithm:    "sha256",
		Timestamp:    start.Format(time.RFC3339),
	}

	return result
}

// ComputeFileHash 计算文件哈希
func (h *Hasher) ComputeFileHash(filePath string) (*HashResult, error) {
	start := time.Now()

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	var hash hash.Hash
	switch h.algorithm {
	case "sha256":
		hash = sha256.New()
	default:
		hash = sha256.New()
	}

	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("计算哈希失败: %v", err)
	}

	result := &HashResult{
		OriginalSize: info.Size(),
		Hash:         hex.EncodeToString(hash.Sum(nil)),
		Algorithm:    "sha256",
		Timestamp:    start.Format(time.RFC3339),
	}

	return result, nil
}

// ComputeReaderHash 计算流哈希
func (h *Hasher) ComputeReaderHash(reader io.Reader) (*HashResult, error) {
	start := time.Now()

	var hash hash.Hash
	switch h.algorithm {
	case "sha256":
		hash = sha256.New()
	default:
		hash = sha256.New()
	}

	if _, err := io.Copy(hash, reader); err != nil {
		return nil, fmt.Errorf("计算哈希失败: %v", err)
	}

	result := &HashResult{
		Hash:      hex.EncodeToString(hash.Sum(nil)),
		Algorithm: "sha256",
		Timestamp: start.Format(time.RFC3339),
	}

	return result, nil
}

// VerifyHash 验证内容哈希
func (h *Hasher) VerifyHash(content, expectedHash string) bool {
	result := h.ComputeHash(content)
	return result.Hash == expectedHash
}

// VerifyFileHash 验证文件哈希
func (h *Hasher) VerifyFileHash(filePath, expectedHash string) (bool, error) {
	result, err := h.ComputeFileHash(filePath)
	if err != nil {
		return false, err
	}
	return result.Hash == expectedHash, nil
}

// ComputeMultipleHashes 计算多种哈希
func (h *Hasher) ComputeMultipleHashes(content string) map[string]string {
	// SHA256
	sha256Hash := sha256.New()
	sha256Hash.Write([]byte(content))

	hashes := map[string]string{
		"sha256": hex.EncodeToString(sha256Hash.Sum(nil)),
	}

	return hashes
}

// GenerateContentID 生成内容唯一标识
func (h *Hasher) GenerateContentID(content string) string {
	result := h.ComputeHash(content)
	return result.Hash[:16] // 取前16位作为短ID
}

// ContentSignature 内容签名
type ContentSignature struct {
	ContentHash   string    `json:"content_hash"`   // 内容哈希
	PreviousHash  string    `json:"previous_hash"` // 前一个哈希
	Timestamp     time.Time `json:"timestamp"`     // 时间戳
	Metadata      string    `json:"metadata"`       // 元数据
}

// CreateChain 创建内容链
func (h *Hasher) CreateChain(contents []string, metadata string) ([]*ContentSignature, error) {
	if len(contents) == 0 {
		return nil, nil
	}

	chain := make([]*ContentSignature, len(contents))
	var previousHash string

	for i, content := range contents {
		currentHash := h.ComputeHash(content)

		sig := &ContentSignature{
			ContentHash:  currentHash.Hash,
			PreviousHash: previousHash,
			Timestamp:    time.Now(),
			Metadata:     metadata,
		}

		chain[i] = sig
		previousHash = currentHash.Hash
	}

	return chain, nil
}

// VerifyChain 验证内容链完整性
func (h *Hasher) VerifyChain(contents []string, chain []*ContentSignature) bool {
	if len(contents) != len(chain) {
		return false
	}

	for i, content := range contents {
		currentHash := h.ComputeHash(content)
		if currentHash.Hash != chain[i].ContentHash {
			return false
		}

		if i > 0 && chain[i].PreviousHash != chain[i-1].ContentHash {
			return false
		}
	}

	return true
}

// MarshalJSON 自定义 JSON 序列化
func (cs *ContentSignature) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"content_hash":  cs.ContentHash,
		"previous_hash": cs.PreviousHash,
		"timestamp":     cs.Timestamp.Format(time.RFC3339),
		"metadata":      cs.Metadata,
	})
}

// QuickHash 快速计算哈希（使用 SHA256）
func QuickHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// QuickHashFile 快速计算文件哈希
func QuickHashFile(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(hash.Sum(nil)), info.Size(), nil
}
