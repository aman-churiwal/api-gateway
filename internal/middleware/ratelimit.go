package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/config"
	"github.com/aman-churiwal/api-gateway/internal/models"
	"github.com/aman-churiwal/api-gateway/internal/ratelimit"
	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/gin-gonic/gin"
)

func RateLimitWithTier(redis *storage.RedisClient, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tier string
		var limit int
		var algorithm string
		var key string

		// Check if API key exists in context
		apiKeyInterface, exists := c.Get("api_key")

		if exists && apiKeyInterface != nil {
			apiKey := apiKeyInterface.(*models.APIKey)
			tier = apiKey.Tier
			key = apiKey.ID.String() // Use API key ID as the rate limit key

			// Find Tier Configuration
			tierConfig := findTierConfig(cfg, tier)
			if tierConfig != nil {
				limit = tierConfig.RequestsPerMinute
				algorithm = tierConfig.Algorithm
			} else {
				limit = 60
				algorithm = "fixed_window"
			}
		} else {
			tier = "basic"
			key = c.ClientIP()

			// Use first tier as default
			if len(cfg.RateLimitTiers) > 0 {
				limit = cfg.RateLimitTiers[0].RequestsPerMinute
				algorithm = cfg.RateLimitTiers[0].Algorithm
			} else {
				limit = 60
				algorithm = "fixed_window"
			}
		}

		// Create Rate Limiter based on algorithm
		limiter := ratelimit.NewLimiter(redis, algorithm, limit, time.Minute)

		// Check Rate Limit
		ctx := c.Request.Context()
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Rate limit check failed",
			})
			c.Abort()
			return
		}

		// Get remaining count
		remaining, _ := limiter.Remaining(c.Request.Context(), key)

		// Get reset time
		resetTime, _ := limiter.Reset(ctx, key)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.Limit()))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
		c.Header("X-RateLimit-Tier", tier)

		if !allowed {
			retryAfter := int(time.Until(resetTime).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}

			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"tier":        tier,
				"limit":       limit,
				"retry_after": resetTime.Unix(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func findTierConfig(cfg *config.Config, tierName string) *config.RateLimiterTier {
	for _, tier := range cfg.RateLimitTiers {
		if tier.Name == tierName {
			return &tier
		}
	}

	return nil
}
