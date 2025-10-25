package handlers

import (
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/middleware"
	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/gin-gonic/gin"
)

// ConnectionInfoHandler handles connection information requests
type ConnectionInfoHandler struct {
	ipAllowListManager *ipallowlist.Manager
}

// NewConnectionInfoHandler creates a new connection info handler
func NewConnectionInfoHandler(ipAllowListManager *ipallowlist.Manager) *ConnectionInfoHandler {
	return &ConnectionInfoHandler{
		ipAllowListManager: ipAllowListManager,
	}
}

// Handle processes GET /api/connection-info
// This is a public endpoint that returns the client's IP and allowlist status
func (h *ConnectionInfoHandler) Handle(c *gin.Context) {
	// Get client IP (properly extracted by middleware, handles trusted proxies)
	clientIP, hasIP := middleware.GetClientIP(c)
	
	response := map[string]interface{}{
		"client_ip": "",
		"allowed":   false,
		"reason":    "unknown",
	}

	if hasIP && clientIP.IsValid() {
		response["client_ip"] = clientIP.String()
		
		// Check if IP is allowed
		if h.ipAllowListManager != nil {
			allowed, reason := h.ipAllowListManager.IsIPAllowed(clientIP)
			response["allowed"] = allowed
			response["reason"] = reason
		}
	}

	c.JSON(200, models.NewAPIResponse("Connection info retrieved", response))
}
