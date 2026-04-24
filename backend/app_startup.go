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
	startCleanupRoutine(rt.db, handlerSet.plannerHandler, rt.cfg)
}

func preloadBackgroundImages() {
	time.Sleep(3 * time.Second)
	handlers.InitBackgroundImages()
	log.Println("背景图预加载完成")
}
