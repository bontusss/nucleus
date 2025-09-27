package ratelimit

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func IPRateLimitMiddleware(limiter *RateLimiter, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("middleware:ip:%s", ip)

		exceeded, count, timeLeft, err := limiter.Check(c.Request.Context(), key, limit, window)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		if exceeded {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%.0f", timeLeft.Seconds()))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many requests",
				"retry_after": timeLeft.Seconds(),
			})
			c.Abort()
			return
		}

		// Increment counter
		limiter.Increment(c.Request.Context(), key, window)

		remaining := limit - count - 1
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%.0f", window.Seconds()))

		c.Next()
	}
}
