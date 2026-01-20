package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		requestID := c.GetString("request_id")

		log.Printf("[%s] %s %s - %d - %v - %s",
			requestID,
			method,
			path,
			statusCode,
			latency,
			c.ClientIP(),
		)
	}
}
