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

		// Get client IP from our middleware (respects trusted proxy config)
		clientIP := "unknown"
		if addr, ok := GetClientIP(c); ok {
			clientIP = addr.String()
		}

		logEvent := log.Info().
			Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Dur("duration", duration).
			Str("client_ip", clientIP)

		// Add error if present
		if len(c.Errors) > 0 {
			logEvent = logEvent.Str("error", c.Errors.String())
		}

		logEvent.Msg("HTTP request")
	}
}
