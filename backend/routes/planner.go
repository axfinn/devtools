package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPlannerRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	planner := api.Group("/planner")
	{
		planner.POST("/profile", createRateLimiter.Middleware(), h.PlannerHandler.CreateProfile)
		planner.POST("/profile/login", h.PlannerHandler.LoginProfile)
		planner.GET("/profile/:id", h.PlannerHandler.GetProfile)
		planner.PUT("/profile/:id", h.PlannerHandler.UpdateProfile)
		planner.DELETE("/profile/:id", h.PlannerHandler.DeleteProfile)
		planner.GET("/profile/:id/timeline", h.PlannerHandler.ListTimeline)
		planner.GET("/profile/:id/review", h.PlannerHandler.Review)
		planner.GET("/profile/:id/search", h.PlannerHandler.Search)
		planner.POST("/profile/:id/tasks", h.PlannerHandler.CreateTask)
		planner.POST("/profile/:id/tasks/batch", h.PlannerHandler.CreateTaskBatch)
		planner.PUT("/profile/:id/tasks/:taskId", h.PlannerHandler.UpdateTask)
		planner.DELETE("/profile/:id/tasks/:taskId", h.PlannerHandler.DeleteTask)
		planner.POST("/profile/:id/tasks/batch-update", h.PlannerHandler.BatchUpdateTasks)
		planner.GET("/profile/:id/tasks/:taskId/comments", h.PlannerHandler.ListTaskComments)
		planner.POST("/profile/:id/tasks/:taskId/comments", h.PlannerHandler.CreateTaskComment)
		planner.GET("/profile/:id/tasks/:taskId/activities", h.PlannerHandler.ListTaskActivities)
		planner.GET("/profile/:id/tasks/:taskId/calendar", h.PlannerHandler.DownloadCalendar)
		planner.GET("/profile/:id/calendar.ics", h.PlannerHandler.DownloadCalendarFeed)
		planner.POST("/profile/:id/ai/parse", h.PlannerHandler.AIParse)
		planner.POST("/profile/:id/ai/advise", h.PlannerHandler.AIAdvise)
		planner.GET("/profile/:id/meetings", h.PlannerHandler.ListMeetingMinutes)
		planner.POST("/profile/:id/meetings", h.PlannerHandler.CreateMeetingMinutes)
		planner.GET("/profile/:id/meetings/:meetingId", h.PlannerHandler.GetMeetingMinutes)
		planner.PUT("/profile/:id/meetings/:meetingId", h.PlannerHandler.UpdateMeetingMinutes)
		planner.DELETE("/profile/:id/meetings/:meetingId", h.PlannerHandler.DeleteMeetingMinutes)
		planner.POST("/profile/:id/meetings/:meetingId/summarize", h.PlannerHandler.SummarizeMeetingMinutes)
		planner.POST("/profile/:id/recordings", h.PlannerHandler.UploadMeetingRecording)
		planner.GET("/recordings/:filename", h.PlannerHandler.ServeRecording)
		planner.GET("/admin/list", h.PlannerHandler.AdminList)
		planner.GET("/admin/:id", h.PlannerHandler.AdminGet)
		planner.PUT("/admin/:id", h.PlannerHandler.AdminUpdate)
		planner.DELETE("/admin/:id", h.PlannerHandler.AdminDelete)
	}
}
