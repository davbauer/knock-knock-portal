package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RequestSizeLimiter limits the size of incoming request bodies
func RequestSizeLimiter(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Limit request body size
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		// Check if request is too large
		if c.Request.ContentLength > maxBytes {
			log.Warn().
				Int64("content_length", c.Request.ContentLength).
				Int64("max_bytes", maxBytes).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("Request body too large")

			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
				"code":  "REQUEST_TOO_LARGE",
				"details": map[string]interface{}{
					"max_size_bytes": maxBytes,
					"max_size_mb":    float64(maxBytes) / (1024 * 1024),
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
