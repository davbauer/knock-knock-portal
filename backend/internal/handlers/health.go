package handlers

import (
	"time"

	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/gin-gonic/gin"
)

// HealthHandler handles health checks
type HealthHandler struct {
	startTime time.Time
	version   string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
		version:   version,
	}
}

// Handle processes health check requests
func (h *HealthHandler) Handle(c *gin.Context) {
	uptime := time.Since(h.startTime).Seconds()

	response := map[string]interface{}{
		"status":         "healthy",
		"version":        h.version,
		"uptime_seconds": int(uptime),
	}

	c.JSON(200, models.NewAPIResponse("Service healthy", response))
}
