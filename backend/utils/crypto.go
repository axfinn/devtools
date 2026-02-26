package utils

import (
	"crypto/rand"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 10 // 默认 bcrypt 计算成本，可以在配置中调整

// HashPassword 使用 bcrypt 对密码进行哈希
// bcrypt 会自动添加盐值，比 SHA256 更安全
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword 验证密码是否匹配
// 返回 true 表示密码正确，false 表示密码错误
func VerifyPassword(password, hashedPassword string) bool {
	if password == "" && hashedPassword == "" {
		return true
	}
	if password == "" || hashedPassword == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateHexKey 生成指定长度的随机十六进制密钥（用于创建者密钥、访问密钥等）
// length 参数指定字节数，返回的十六进制字符串长度为 length*2
func GenerateHexKey(length int) string {
	if length <= 0 {
		length = 16 // 默认 16 字节（32 字符）
	}
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
