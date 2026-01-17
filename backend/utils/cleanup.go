package utils

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

// CleanExpiredUploads 清理过期的上传文件
// 删除超过指定天数的文件
func CleanExpiredUploads(uploadDir string, days int) (int64, error) {
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		return 0, nil // 目录不存在，无需清理
	}

	expireTime := time.Now().AddDate(0, 0, -days)
	var count int64

	err := filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过无法访问的文件
		}
		if info.IsDir() {
			return nil // 跳过目录
		}

		// 检查文件修改时间
		if info.ModTime().Before(expireTime) {
			if err := os.Remove(path); err != nil {
				log.Printf("删除过期文件失败 %s: %v", path, err)
			} else {
				count++
			}
		}
		return nil
	})

	return count, err
}
