package models

import (
	"encoding/json"
	"log"
	"strings"
)

// ReferencedUploadFilenames 返回所有被"长期事件/档案"内容引用的 ./data/uploads 文件名集合。
// 用于防止 hourly cleanup 误删 7 天前的文件——但只覆盖"事件相关"的资源:
//   - nfs_shares.file_path              （NFS 长期分享）
//   - planner_task_comments.image_urls  （事项评论贴图,这是用户最关心的"事件"记忆）
//   - voice_memos.audio_url             （语音备忘 / 事项录音,长期记忆的一部分）
//   - photowall_items.filename          （照片墙,本身就是长期归档）
//
// 不覆盖的范围(故意不保护,让它们走各自的过期/清理策略):
//   - pastes.files                      （粘贴板有自己的过期逻辑,不需要保）
//   - chat_messages.content             （聊天室 7 天后连房带消息一起清,不要变僵尸）
//   - markdown_shares.content           （md 分享有自己的过期 / 浏览次数限制）
//   - excalidraw_shares.content         （画图有自己的过期）
//
// 失败容忍:任一来源报错仅记日志,不影响其他来源收集。
func (db *DB) ReferencedUploadFilenames() (map[string]struct{}, error) {
	result := make(map[string]struct{})

	// 1) NFS shares
	if names, err := db.activeNFSUploadFilenames(); err != nil {
		log.Printf("ReferencedUploadFilenames: nfs shares 读取失败: %v", err)
	} else {
		for n := range names {
			result[n] = struct{}{}
		}
	}

	// 2) planner_task_comments.image_urls (JSON 字符串数组,元素是 URL 形如 /api/paste/files/xxx)
	if rows, err := db.conn.Query(`SELECT image_urls FROM planner_task_comments WHERE image_urls IS NOT NULL AND image_urls != '' AND image_urls != '[]'`); err != nil {
		log.Printf("ReferencedUploadFilenames: planner_task_comments.image_urls 读取失败: %v", err)
	} else {
		func() {
			defer rows.Close()
			for rows.Next() {
				var raw string
				if err := rows.Scan(&raw); err != nil {
					continue
				}
				var list []string
				if err := json.Unmarshal([]byte(raw), &list); err != nil {
					continue
				}
				for _, u := range list {
					if name := extractUploadFilename(u); name != "" {
						result[name] = struct{}{}
					}
				}
			}
		}()
	}

	// 3) voice_memos.audio_url
	if rows, err := db.conn.Query(`SELECT audio_url FROM voice_memos WHERE audio_url IS NOT NULL AND audio_url != ''`); err != nil {
		log.Printf("ReferencedUploadFilenames: voice_memos.audio_url 读取失败: %v", err)
	} else {
		func() {
			defer rows.Close()
			for rows.Next() {
				var url string
				if err := rows.Scan(&url); err != nil {
					continue
				}
				if name := extractUploadFilename(url); name != "" {
					result[name] = struct{}{}
				}
			}
		}()
	}

	// 4) photowall_items.filename
	if rows, err := db.conn.Query(`SELECT filename FROM photowall_items WHERE filename IS NOT NULL AND filename != ''`); err != nil {
		log.Printf("ReferencedUploadFilenames: photowall_items.filename 读取失败: %v", err)
	} else {
		func() {
			defer rows.Close()
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					continue
				}
				if name != "" {
					result[name] = struct{}{}
				}
			}
		}()
	}

	return result, nil
}

// activeNFSUploadFilenames 包装原 ActiveUploadFilenames,失败也安全。
func (db *DB) activeNFSUploadFilenames() (map[string]struct{}, error) {
	return db.ActiveUploadFilenames()
}
// 识别以下 URL 形态:
//   - /api/chat/uploads/<filename>
//   - /api/paste/files/<filename>
//   - __uploads__/<filename>   (NFS 虚拟路径)
// 否则尝试取 path.Base。
// 为减少假阳性,要求文件名至少含一个 "." 且扩展名 ≥ 2 字符。
func extractUploadFilename(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	// 取掉 query string 和 fragment
	if idx := strings.IndexAny(u, "?#"); idx >= 0 {
		u = u[:idx]
	}
	// 任何 ".." 路径穿越直接拒绝
	if strings.Contains(u, "..") {
		return ""
	}
	// 找到最后一个 "/" 之后的内容
	if idx := strings.LastIndex(u, "/"); idx >= 0 && idx+1 < len(u) {
		candidate := u[idx+1:]
		if looksLikeRealFilename(candidate) {
			return candidate
		}
	}
	return ""
}

// looksLikeRealFilename 简单判定文件名是否像真实文件(防止误把单词误判为文件)。
// 规则:长度 ≥ 4、含 "."、扩展名(末段)≥ 2 字符且全字母数字(允许 mp4 / webm 等)。
func looksLikeRealFilename(name string) bool {
	if len(name) < 4 {
		return false
	}
	dot := strings.LastIndex(name, ".")
	if dot <= 0 || dot == len(name)-1 {
		return false
	}
	ext := name[dot+1:]
	if len(ext) < 2 {
		return false
	}
	for _, c := range ext {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}
