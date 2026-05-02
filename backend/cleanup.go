package main

import (
	"log"
	"os"
	"time"

	"devtools/config"
	"devtools/handlers"
	"devtools/models"
	"devtools/utils"
)

// startCleanupRoutine 启动定时清理协程，每小时清理过期数据
func startCleanupRoutine(db *models.DB, plannerHandler *handlers.PlannerHandler, cfg *config.Config) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		for range ticker.C {
			// 清理过期粘贴板
			count, err := db.CleanExpired()
			if err == nil && count > 0 {
				log.Printf("已清理 %d 条过期粘贴板", count)
			}
			// 清理过期聊天室（7天不活跃）
			roomCount, err := db.CleanExpiredRooms(7)
			if err == nil && roomCount > 0 {
				log.Printf("已清理 %d 个过期聊天室", roomCount)
			}
			// 清理过期消息（7天）
			msgCount, err := db.CleanExpiredMessages(7)
			if err == nil && msgCount > 0 {
				log.Printf("已清理 %d 条过期消息", msgCount)
			}
			// 清理过期短链
			err = db.CleanExpiredShortURLs()
			if err == nil {
				log.Printf("已清理过期短链")
			}
			// 清理过期 Mock APIs
			err = db.CleanExpiredMockAPIs()
			if err == nil {
				log.Printf("已清理过期 Mock APIs")
			}
			// 清理过期 Markdown 分享
			mdCount, err := db.CleanExpiredMDShares()
			if err == nil && mdCount > 0 {
				log.Printf("已清理 %d 个过期 Markdown 分享", mdCount)
			}
			// 清理过期 Excalidraw 画图
			excalidrawCount, err := db.CleanExpiredExcalidrawShares()
			if err == nil && excalidrawCount > 0 {
				log.Printf("已清理 %d 个过期 Excalidraw 画图", excalidrawCount)
			}
			// 清理过期孕期档案
			pregnancyCount, err := db.CleanExpiredPregnancyProfiles()
			if err == nil && pregnancyCount > 0 {
				log.Printf("已清理 %d 个过期孕期档案", pregnancyCount)
			}
			// 清理过期生活记账档案
			expenseCount, err := db.CleanExpiredExpenseProfiles()
			if err == nil && expenseCount > 0 {
				log.Printf("已清理 %d 个过期生活记账档案", expenseCount)
			}
			// 清理过期血糖档案
			glucoseCount, err := db.CleanExpiredGlucoseProfiles()
			if err == nil && glucoseCount > 0 {
				log.Printf("已清理 %d 个过期血糖监测档案", glucoseCount)
			}
			// 清理过期事项档案
			plannerCount, err := db.CleanExpiredPlannerProfiles()
			if err == nil && plannerCount > 0 {
				log.Printf("已清理 %d 个过期事项档案", plannerCount)
			}
			// 清理过期照片墙档案
			photoWallCount, err := db.CleanExpiredPhotoWallProfiles()
			if err == nil && photoWallCount > 0 {
				log.Printf("已清理 %d 个过期照片墙档案", photoWallCount)
			}
			// 清理过期菜谱
			recipeCount, err := db.CleanExpiredRecipes()
			if err == nil && recipeCount > 0 {
				log.Printf("已清理 %d 个过期菜谱", recipeCount)
			}
			// 清理过期 SSH 会话
			sshExpiredCount, err := db.CleanExpiredSSHSessions()
			if err == nil && sshExpiredCount > 0 {
				log.Printf("已清理 %d 个过期 SSH 会话", sshExpiredCount)
			}
			// 清理不活跃的 SSH 会话
			sshInactiveCount, err := db.CleanInactiveSSHSessions(cfg.SSH.SessionMaxAgeDays)
			if err == nil && sshInactiveCount > 0 {
				log.Printf("已清理 %d 个不活跃 SSH 会话（超过%d天）", sshInactiveCount, cfg.SSH.SessionMaxAgeDays)
			}
			// 清理旧的 SSH 历史记录
			historyCount, err := db.CleanOldSSHHistory(cfg.SSH.HistoryMaxAgeDays)
			if err == nil && historyCount > 0 {
				log.Printf("已清理 %d 条旧 SSH 历史记录（超过%d天）", historyCount, cfg.SSH.HistoryMaxAgeDays)
			}
			// 清理过期上传文件（7天），跳过 NFS 分享上传的文件
			protected, _ := db.ActiveUploadFilenames()
			uploadCount, err := utils.CleanExpiredUploads("./data/uploads", 7, protected)
			if err == nil && uploadCount > 0 {
				log.Printf("已清理 %d 个过期上传文件", uploadCount)
			}
			// 清理旧的百炼任务
			bailianCount, err := db.CleanOldBailianTasks(cfg.Bailian.TaskRetentionDays)
			if err == nil && bailianCount > 0 {
				log.Printf("已清理 %d 条旧百炼任务", bailianCount)
			}
			// 清理旧的 AI Gateway 请求明细
			aiLogCount, err := db.CleanOldAIAPIRequestLogs(cfg.AIGateway.RequestRetentionDays)
			if err == nil && aiLogCount > 0 {
				log.Printf("已清理 %d 条旧 AI Gateway 请求明细", aiLogCount)
			}
			// 扫描事项提醒
			plannerHandler.ProcessDueReminders()
			// 清理过期/耗尽的 NFS 分享

	// 清理过期语音备忘录（草稿14天未处理 + 已删除7天）
	vmCount, err := db.CleanExpiredVoiceMemos()
	if err == nil && vmCount > 0 {
		log.Printf("已清理 %d 条过期语音备忘", vmCount)
	}

			nfsCount, err := db.CleanExpiredNFSShares()
			if err == nil && nfsCount > 0 {
				log.Printf("已清理 %d 个过期 NFS 分享", nfsCount)
			}
			// 清理孤立的 HLS 转码缓存（分享已删除但目录残留）
			if entries, err := os.ReadDir("./data/transcode"); err == nil {
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					if _, err := db.GetNFSShare(e.Name()); err != nil {
						handlers.CleanHLSCache(e.Name())
					}
				}
			}
		}
	}()
}
