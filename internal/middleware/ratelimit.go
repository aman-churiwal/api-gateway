package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/ratelimit"
	"github.com/gin-gonic/gin"
)

func RateLimiter(limiter *ratelimit.FixedWindowLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract API key from header
		key := c.GetHeader("X-API-Key")
		if key == "" {
			key = c.ClientIP() // Fallback to IP
		}

		ctx := c.Request.Context()

		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Rate limit check failed",
			})
			c.Abort()
			return
		}

		remaining, _ := limiter.Remaining(c.Request.Context(), key)
		resetTime, _ := limiter.Reset(ctx, key)

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.Limit()))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

		if !allowed {
			retryAfter := int(time.Until(resetTime).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}

			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": resetTime.Unix(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
