package middleware

import (
	"context"
	"net/http"
	"time"

	"devtools/state"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	store         state.TransientStore
	fallbackStore state.TransientStore
	limit         int
	window        time.Duration
}

func NewRateLimiter(limit int, window time.Duration, store state.TransientStore) *RateLimiter {
	if store == nil {
		store = state.NewMemoryStore()
	}
	return &RateLimiter{
		store:         store,
		fallbackStore: state.NewMemoryStore(),
		limit:         limit,
		window:        window,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	allowed, err := rl.store.AllowRateLimit(context.Background(), ip, rl.limit, rl.window)
	if err != nil {
		allowed, fallbackErr := rl.fallbackStore.AllowRateLimit(context.Background(), ip, rl.limit, rl.window)
		if fallbackErr != nil {
			return false
		}
		return allowed
	}
	if !allowed {
		return false
	}
	return true
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
				"code":  429,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ContentSizeLimiter 限制请求体大小
func ContentSizeLimiter(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "内容大小超过限制",
				"code":  413,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
