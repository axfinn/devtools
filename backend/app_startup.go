package main

import (
	"log"
	"time"

	"devtools/handlers"
)

func startBackgroundServices(rt *appRuntime, handlerSet *routeHandlers) {
	go preloadBackgroundImages()

	handlers.InitGFWList("./data/proxy.db")
	handlerSet.proxyHandler.AutoSelectOnStartup()
	handlerSet.proxyHandler.StartAutoMaintenance()
	handlerSet.plannerHandler.ProcessDueReminders()
	handlerSet.terminalHandler.StartCleanupRoutine()
	// 恢复上次崩溃/重启遗留的孤儿录音分片(30s 阈值,确保不和在跑 session 撞车)
	go handlerSet.nfsShareHandler.RecoverOrphanRecordings(30 * time.Second)
	startCleanupRoutine(rt.db, handlerSet.plannerHandler, rt.cfg)
}

func preloadBackgroundImages() {
	defer func() { if r := recover(); r != nil { log.Printf("PANIC in preloadBackgroundImages: %v", r) } }()
	time.Sleep(3 * time.Second)
	handlers.InitBackgroundImages()
	log.Println("背景图预加载完成")
}
