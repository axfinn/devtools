package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashPassword 使用 SHA256 对密码进行哈希
// 注意: 对于高安全要求场景，建议使用 bcrypt
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// VerifyPassword 验证密码是否匹配
func VerifyPassword(password, hashedPassword string) bool {
	return HashPassword(password) == hashedPassword
}
