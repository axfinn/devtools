package main

import "log"

func main() {
	rt, err := newAppRuntime()
	if err != nil {
		log.Fatalf("运行时初始化失败: %v", err)
	}
	defer rt.Close()

	handlerSet, err := buildRouteHandlers(rt)
	if err != nil {
		log.Fatalf("处理器初始化失败: %v", err)
	}

	startBackgroundServices(rt, handlerSet)

	router := newHTTPRouter(rt, handlerSet)
	startTunnelProxyServer(rt.cfg, handlerSet.proxyHandler)

	log.Printf("服务器启动在端口 %s", rt.port)
	server := newMainHTTPServer(rt.port, router, handlerSet.proxyHandler)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
