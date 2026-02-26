package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

// EncryptionService 提供 AES-256-GCM 加密/解密服务
type EncryptionService struct {
	masterKey []byte // 32字节主密钥
}

// NewEncryptionService 创建加密服务实例
// masterKey 应该是32字节的密钥，如果提供的密钥长度不足，会使用 SHA256 派生
func NewEncryptionService(masterKey string) (*EncryptionService, error) {
	if masterKey == "" {
		return nil, errors.New("master key cannot be empty")
	}

	// 使用 SHA256 确保密钥长度正好是32字节
	hash := sha256.Sum256([]byte(masterKey))

	return &EncryptionService{
		masterKey: hash[:],
	}, nil
}

// NewEncryptionServiceFromEnv 从环境变量创建加密服务
// 如果环境变量未设置，会生成随机密钥并警告（重启后无法解密）
func NewEncryptionServiceFromEnv() (*EncryptionService, error) {
	masterKey := os.Getenv("TERMINAL_ENCRYPTION_KEY")

	if masterKey == "" {
		// 生成随机密钥并警告
		randomKey := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, randomKey); err != nil {
			return nil, fmt.Errorf("failed to generate random key: %w", err)
		}

		fmt.Println("WARNING: TERMINAL_ENCRYPTION_KEY not set, using temporary random key")
		fmt.Println("WARNING: Encrypted data will be lost after server restart")
		fmt.Printf("WARNING: To persist encryption, set: export TERMINAL_ENCRYPTION_KEY=%s\n",
			base64.StdEncoding.EncodeToString(randomKey))

		return &EncryptionService{
			masterKey: randomKey,
		}, nil
	}

	return NewEncryptionService(masterKey)
}

// Encrypt 加密字符串，返回 base64 编码的密文
func (e *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(e.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建 GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密（nonce + ciphertext + tag）
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密 base64 编码的密文
func (e *EncryptionService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(e.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建 GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 检查数据长度
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// 分离 nonce 和密文
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptIfNotEmpty 如果明文不为空则加密，否则返回空字符串
func (e *EncryptionService) EncryptIfNotEmpty(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	return e.Encrypt(plaintext)
}

// DecryptIfNotEmpty 如果密文不为空则解密，否则返回空字符串
func (e *EncryptionService) DecryptIfNotEmpty(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	return e.Decrypt(ciphertext)
}

// GenerateRandomKey 生成一个随机的32字节密钥（用于初始化）
func GenerateRandomKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
