package handlers

import (
	"net"
	"strings"

	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/davbauer/knock-knock-portal/internal/proxy"
	"github.com/davbauer/knock-knock-portal/internal/session"
	"github.com/gin-gonic/gin"
)

// AdminConnectionsHandler handles admin connection monitoring
type AdminConnectionsHandler struct {
	proxyManager   *proxy.Manager
	sessionManager *session.Manager
}

// NewAdminConnectionsHandler creates a new handler
func NewAdminConnectionsHandler(proxyManager *proxy.Manager, sessionManager *session.Manager) *AdminConnectionsHandler {
	return &AdminConnectionsHandler{
		proxyManager:   proxyManager,
		sessionManager: sessionManager,
	}
}

// HandleList handles GET /api/admin/connections
// Returns all active connections grouped by IP, showing both authenticated and anonymous users
func (h *AdminConnectionsHandler) HandleList(c *gin.Context) {
	// Get all active sessions to map IPs to usernames
	sessions := h.sessionManager.GetAllActiveSessions()
	
	// Create IP -> session mapping
	ipToSession := make(map[string]*session.Session)
	for _, sess := range sessions {
		for _, ip := range sess.AuthenticatedIPAddresses {
			ipToSession[ip.String()] = sess
		}
	}

	// Get all proxy stats to find ALL active IPs (including non-authenticated)
	proxyStats := h.proxyManager.GetStats()
	
	connections := []map[string]interface{}{}
	processedIPs := make(map[string]bool)

	// Get all active IPs from services
	if services, ok := proxyStats["services"].([]map[string]interface{}); ok {
		for _, service := range services {
			// Get client IPs from this service
			if clientIPsRaw, ok := service["client_ips"]; ok {
				var clientIPs []string
				
				// Handle both []string and []interface{} types
				switch v := clientIPsRaw.(type) {
				case []string:
					clientIPs = v
				case []interface{}:
					for _, ipRaw := range v {
						if ipStr, ok := ipRaw.(string); ok {
							clientIPs = append(clientIPs, ipStr)
						}
					}
				}
				
				for _, ipWithPort := range clientIPs {
					// Extract just the IP (remove port if present)
					ip := extractIP(ipWithPort)
					
					if processedIPs[ip] {
						continue
					}
					processedIPs[ip] = true

					// Get stats for this IP
					stats := h.proxyManager.GetStatsByIP(ip)
					
					// Check if this IP has an authenticated session
					var username, userID string
					var allowedServices []string
					authenticated := false
					
					if sess, exists := ipToSession[ip]; exists {
						username = sess.Username
						userID = sess.UserID
						allowedServices = sess.AllowedServiceIDs
						authenticated = true
					} else {
						username = "Anonymous"
						userID = "unauthenticated"
						allowedServices = []string{} // They can access via permanent allowlist
					}

					connections = append(connections, map[string]interface{}{
						"ip":                     ip,
						"username":               username,
						"user_id":                userID,
						"authenticated":          authenticated,
						"allowed_services":       allowedServices,
						"total_packets_rx":       stats["total_packets_received"],
						"total_packets_tx":       stats["total_packets_sent"],
						"total_bytes_rx":         stats["total_bytes_received"],
						"total_bytes_tx":         stats["total_bytes_sent"],
						"total_sessions":         stats["total_sessions"],
						"services":               stats["services"],
					})
				}
			}
		}
	}

	c.JSON(200, models.NewAPIResponseWithCount("Active connections retrieved", map[string]interface{}{
		"connections": connections,
	}, len(connections)))
}

// extractIP extracts IP from "IP:port" string or returns as-is if no port
func extractIP(ipWithPort string) string {
	// Try to parse as host:port
	host, _, err := net.SplitHostPort(ipWithPort)
	if err == nil {
		return host
	}
	
	// If SplitHostPort fails, it might be just an IP without port
	// Remove any trailing colon just in case
	return strings.TrimSuffix(ipWithPort, ":")
}
