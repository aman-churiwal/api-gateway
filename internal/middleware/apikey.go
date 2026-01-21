package middleware

import (
	"net/http"
	"strings"

	"github.com/aman-churiwal/api-gateway/internal/service"
	"github.com/gin-gonic/gin"
)

func APIKeyValidator(apiKeyService *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKeyHeader := c.GetHeader("X-API-Key")

		if apiKeyHeader == "" {
			c.Next()
			return
		}

		// Trimming whitespace
		apiKeyHeader = strings.TrimSpace(apiKeyHeader)

		// Validate API key
		ctx := c.Request.Context()
		apiKey, err := apiKeyService.Validate(ctx, apiKeyHeader)

		if err != nil || apiKey == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
			})
			c.Abort()
			return
		}

		c.Set("api_key", apiKey)
		c.Set("api_key_id", apiKey.ID)
		c.Set("api_key_tier", apiKey.Tier)

		go apiKeyService.UpdateLastUsed(ctx, apiKey.ID)

		c.Next()
	}
}
