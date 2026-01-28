package utils

import (
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
