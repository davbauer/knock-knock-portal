package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RequestLogger logs HTTP requests with structured logging
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Log after request
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		logEvent := log.Info().
			Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Dur("duration", duration).
			Str("client_ip", c.ClientIP())

		// Add error if present
		if len(c.Errors) > 0 {
			logEvent = logEvent.Str("error", c.Errors.String())
		}

		logEvent.Msg("HTTP request")
	}
}
