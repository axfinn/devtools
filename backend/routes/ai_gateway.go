package routes

import (
	"time"

	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterAIGatewayRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	aigw := api.Group("/ai-gateway")
	{
		aigw.GET("/docs", h.AiGatewayHandler.GetDocs)
		aigw.GET("/docs/anthropic", h.AiGatewayHandler.GetAnthropicDocs)
		aigw.GET("/catalog", h.AiGatewayHandler.GetCatalog)
		aigw.POST("/admin/keys", h.AiGatewayHandler.AdminCreateKey)
		aigw.GET("/admin/keys", h.AiGatewayHandler.AdminListKeys)
		aigw.GET("/admin/keys/:id", h.AiGatewayHandler.AdminGetKey)
		aigw.PUT("/admin/keys/:id", h.AiGatewayHandler.AdminUpdateKey)
		aigw.POST("/admin/keys/:id/revoke", h.AiGatewayHandler.AdminRevokeKey)
		aigw.GET("/admin/logs", h.AiGatewayHandler.AdminListLogs)
		aigw.GET("/admin/reports", h.AiGatewayHandler.AdminReports)
		aigw.GET("/admin/alerts", h.AiGatewayHandler.AdminAlerts)
		aigw.POST("/admin/test-model", h.AiGatewayHandler.AdminTestModel)
		aigw.GET("/admin/anthropic/providers", h.AiGatewayHandler.AdminListAnthropicProviders)
		aigw.POST("/admin/anthropic/providers", h.AiGatewayHandler.AdminCreateAnthropicProvider)
		aigw.PUT("/admin/anthropic/providers/:id", h.AiGatewayHandler.AdminUpdateAnthropicProvider)
		aigw.DELETE("/admin/anthropic/providers/:id", h.AiGatewayHandler.AdminDeleteAnthropicProvider)
		aigw.POST("/v1/chat/completions", h.AiGatewayHandler.ChatCompletions)
		aigw.POST("/v1/chat/tasks", h.AiGatewayHandler.AsyncChatCompletions)
		aigw.GET("/v1/chat/tasks/:id", h.AiGatewayHandler.GetChatTask)
		aigw.POST("/v1/image/understanding", h.AiGatewayHandler.ImageUnderstanding)
		aigw.POST("/v1/image/understanding/file", h.AiGatewayHandler.ImageUnderstandingFile)
		aigw.POST("/v1/image/understanding/sse", h.AiGatewayHandler.ImageUnderstandingSSE)
		aigw.POST("/v1/image/understanding/sse/file", h.AiGatewayHandler.ImageUnderstandingSSEFile)
		aigw.GET("/v1/image/understanding/stream/:id", h.AiGatewayHandler.ImageUnderstandingStream)
		// OpenAI 兼容出图：本地 ComfyUI（Qwen-Image），同步返回，鉴权走 AI Gateway API Key
		aigw.POST("/v1/images/generations", createRateLimiter.Middleware(), h.AiGatewayHandler.OpenAIImagesGenerations)
	}

	api.POST("/internal/chat", createRateLimiter.Middleware(), h.AiGatewayHandler.InternalChat)
	api.POST("/internal/chat/stream", createRateLimiter.Middleware(), h.AiGatewayHandler.InternalChatStream)
	api.GET("/askit/extension", h.AiGatewayHandler.AskitExtensionDownload)
	api.POST("/askit/extension/refresh", h.AiGatewayHandler.AskitExtensionRefresh)
	api.POST("/ai-chat/verify", h.AiGatewayHandler.AIChatVerify)

	// 新版聊天页（chat-plus，Open WebUI 风格）：Ollama + MiniMax 双引擎
	api.POST("/chat-plus/verify", h.AiGatewayHandler.ChatPlusVerify)
	api.GET("/chat-plus/models", h.AiGatewayHandler.ChatPlusModels)
	api.POST("/chat-plus/chat/stream", createRateLimiter.Middleware(), h.AiGatewayHandler.ChatPlusStream)

	// 异步任务化聊天（弱网断线续传）：创建任务 → 订阅 SSE(from=seq) → 刷新后按会话对齐
	// 建任务用独立限流桶（60/分钟）：createRateLimiter 是 10/分钟，多端同用一个公网 IP 时
	// 正常聊天很快就撞 429，表现成"莫名其妙发送失败"。
	chatTaskLimiter := middleware.NewRateLimiter(60, time.Minute, nil)
	api.POST("/chat-plus/tasks", chatTaskLimiter.Middleware(), h.AiGatewayHandler.ChatPlusTaskCreate)
	api.GET("/chat-plus/tasks", h.AiGatewayHandler.ChatPlusTaskList)
	api.GET("/chat-plus/tasks/:id", h.AiGatewayHandler.ChatPlusTaskGet)
	api.GET("/chat-plus/tasks/:id/stream", h.AiGatewayHandler.ChatPlusTaskStream)
	api.POST("/chat-plus/tasks/:id/stop", h.AiGatewayHandler.ChatPlusTaskStop)

	// 会话/消息同步（多端一致）：服务端是唯一真相，localStorage 只是离线缓存
	api.GET("/chat-plus/conversations", h.AiGatewayHandler.ChatPlusConversations)
	api.POST("/chat-plus/conversations", h.AiGatewayHandler.ChatPlusConversations)
	api.DELETE("/chat-plus/conversations/:id", h.AiGatewayHandler.ChatPlusConversationDelete)
	api.GET("/chat-plus/conversations/:id/messages", h.AiGatewayHandler.ChatPlusMessages)
	api.POST("/chat-plus/conversations/:id/messages", h.AiGatewayHandler.ChatPlusMessages)
	// <img src> 带不了自定义 header，故不带密码门；文件名是 16 字节随机 hex
	api.GET("/chat-plus/images/:name", h.AiGatewayHandler.ChatPlusImage)
	// 本地出图（局域网 ComfyUI）：异步任务 + SSE 进度，密码门复用 chat_plus.password
	api.GET("/chat-plus/image/status", h.AiGatewayHandler.ChatPlusImageStatus)
	api.POST("/chat-plus/image/generate", createRateLimiter.Middleware(), h.AiGatewayHandler.ChatPlusImageGenerate)
	api.GET("/chat-plus/image/tasks/:id", h.AiGatewayHandler.ChatPlusImageTask)
	api.GET("/chat-plus/image/tasks/:id/stream", h.AiGatewayHandler.ChatPlusImageTaskStream)
	api.GET("/english-tutor/meta", h.AiGatewayHandler.EnglishTutorMeta)
	api.POST("/english-tutor", h.AiGatewayHandler.EnglishTutor)

	api.POST("/anthropic/v1/messages", h.AiGatewayHandler.ProxyAnthropicGeneric)
	api.POST("/anthropic/v1/messages/tasks", h.AiGatewayHandler.AsyncAnthropicMessages)
	api.GET("/anthropic/v1/messages/tasks/:id", h.AiGatewayHandler.GetAnthropicMessageTask)
	api.GET("/anthropic/v1/models", h.AiGatewayHandler.ProxyAnthropicModels)
	api.POST("/minimax/anthropic/v1/messages", h.AiGatewayHandler.ProxyMinimaxAnthropic)
	api.POST("/minimax/tts/v1/generations", h.AiGatewayHandler.ProxyMinimaxTTS)
	api.GET("/minimax/tts/docs", h.AiGatewayHandler.GetTTSDocs)
	api.POST("/minimax/token-plan/v1/generations", h.AiGatewayHandler.ProxyMinimaxTokenPlan)
	api.GET("/minimax/token-plan/docs", h.AiGatewayHandler.GetTokenPlanDocs)
	api.GET("/minimax/token-plan/tasks", h.AiGatewayHandler.ListMinimaxTokenPlanTasks)
	api.GET("/minimax/token-plan/tasks/:id", h.AiGatewayHandler.GetMinimaxTokenPlanTask)
	api.GET("/minimax/token-plan/tasks/:id/download", h.AiGatewayHandler.DownloadMinimaxTokenPlanTask)
	api.GET("/minimax/music/docs", h.AiGatewayHandler.GetMiniMaxMusicDocs)
	api.POST("/minimax/music/v1/lyrics_generation", h.AiGatewayHandler.MiniMaxLyricsGeneration)
	api.GET("/minimax/music/v1/lyrics_tasks/:id", h.AiGatewayHandler.GetMiniMaxLyricsTask)
	api.POST("/minimax/music/v1/cover_preprocess", h.AiGatewayHandler.MiniMaxMusicCoverPreprocess)
	api.GET("/minimax/h3/docs", h.AiGatewayHandler.GetMiniMaxH3VideoDocs)
	api.POST("/minimax/h3/v2/video_generation", h.AiGatewayHandler.MiniMaxH3VideoCreate)
	api.GET("/minimax/h3/v2/query/video_generation", h.AiGatewayHandler.MiniMaxH3VideoList)
	api.GET("/minimax/h3/v2/query/video_generation/:task_id", h.AiGatewayHandler.MiniMaxH3VideoQuery)
	api.DELETE("/minimax/h3/v2/video_generation/:task_id", h.AiGatewayHandler.MiniMaxH3VideoDelete)
	api.GET("/minimax/result-shares/docs", h.AiGatewayHandler.GetMiniMaxResultShareDocs)
	api.POST("/minimax/result-shares", h.AiGatewayHandler.CreateMiniMaxResultShare)
	api.GET("/minimax/result-shares/public/list", h.AiGatewayHandler.PublicListMiniMaxResultShares)
	api.GET("/minimax/result-shares/:id", h.AiGatewayHandler.GetMiniMaxResultShare)
	api.GET("/minimax/result-shares/:id/assets/:assetId", h.AiGatewayHandler.GetMiniMaxResultShareAsset)
	api.GET("/minimax/result-shares/admin/list", h.AiGatewayHandler.AdminListMiniMaxResultShares)
	api.GET("/minimax/result-shares/admin/:id", h.AiGatewayHandler.AdminGetMiniMaxResultShare)
	api.PUT("/minimax/result-shares/admin/:id", h.AiGatewayHandler.AdminUpdateMiniMaxResultShare)
	api.DELETE("/minimax/result-shares/admin/:id", h.AiGatewayHandler.AdminDeleteMiniMaxResultShare)
	api.POST("/minimax/voice-cloning/upload", h.AiGatewayHandler.UploadVoiceClone)
	api.GET("/minimax/voice-cloning/voices", h.AiGatewayHandler.ListVoiceClones)
	api.DELETE("/minimax/voice-cloning/voices/:id", h.AiGatewayHandler.DeleteVoiceClone)
	api.POST("/minimax/voice-cloning/tts", h.AiGatewayHandler.TTSWithVoiceClone)
	api.GET("/minimax/voice-cloning/docs", h.AiGatewayHandler.GetVoiceCloningDocs)
	api.GET("/minimax/speech/docs", h.AiGatewayHandler.GetMiniMaxSpeechDocs)
	api.POST("/minimax/speech/v1/t2a_v2", h.AiGatewayHandler.MiniMaxSpeechSyncT2A)
	api.POST("/minimax/speech/v1/t2a_async_v2", h.AiGatewayHandler.MiniMaxSpeechAsyncCreate)
	api.GET("/minimax/speech/v1/query/t2a_async_query_v2", h.AiGatewayHandler.MiniMaxSpeechAsyncQuery)
	api.POST("/minimax/speech/v1/get_voice", h.AiGatewayHandler.MiniMaxSpeechGetVoice)
	api.POST("/minimax/speech/v1/voice_design", h.AiGatewayHandler.MiniMaxSpeechVoiceDesign)
	api.POST("/minimax/speech/v1/voice_clone", h.AiGatewayHandler.MiniMaxSpeechVoiceClone)
	api.POST("/minimax/speech/v1/delete_voice", h.AiGatewayHandler.MiniMaxSpeechDeleteVoice)
	api.POST("/minimax/speech/v1/files/upload", h.AiGatewayHandler.MiniMaxSpeechUploadFile)
	api.GET("/minimax/speech/v1/files/list", h.AiGatewayHandler.MiniMaxSpeechListFiles)
	api.GET("/minimax/speech/v1/files/retrieve", h.AiGatewayHandler.MiniMaxSpeechRetrieveFile)
	api.GET("/minimax/speech/v1/files/retrieve_content", h.AiGatewayHandler.MiniMaxSpeechRetrieveFileContent)
	api.POST("/minimax/speech/v1/files/delete", h.AiGatewayHandler.MiniMaxSpeechDeleteFile)
	api.GET("/minimax/speech/tasks", h.AiGatewayHandler.MiniMaxSpeechListTasks)
	api.GET("/minimax/speech/tasks/:id", h.AiGatewayHandler.MiniMaxSpeechGetTask)
	api.POST("/dashscope/anthropic/v1/messages", h.AiGatewayHandler.ProxyDashScopeAnthropic)
	api.POST("/deepseek/anthropic/v1/messages", h.AiGatewayHandler.ProxyDeepSeekAnthropic)
}
