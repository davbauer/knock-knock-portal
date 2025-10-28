package handlers

import (
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/davbauer/knock-knock-portal/internal/proxy"
	"github.com/davbauer/knock-knock-portal/internal/session"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// AdminSessionsHandler handles admin session management
type AdminSessionsHandler struct {
	sessionManager   *session.Manager
	allowlistManager *ipallowlist.Manager
	proxyManager     *proxy.Manager
}

// NewAdminSessionsHandler creates a new handler
func NewAdminSessionsHandler(sessionManager *session.Manager, allowlistManager *ipallowlist.Manager, proxyManager *proxy.Manager) *AdminSessionsHandler {
	return &AdminSessionsHandler{
		sessionManager:   sessionManager,
		allowlistManager: allowlistManager,
		proxyManager:     proxyManager,
	}
}

// HandleList handles GET /api/admin/sessions
func (h *AdminSessionsHandler) HandleList(c *gin.Context) {
	sessions := h.sessionManager.GetAllActiveSessions()

	sessionList := []map[string]interface{}{}
	for _, sess := range sessions {
		// Convert IP addresses to strings
		ipStrings := make([]string, len(sess.AuthenticatedIPAddresses))
		for i, ip := range sess.AuthenticatedIPAddresses {
			ipStrings[i] = ip.String()
		}

		// Aggregate proxy stats for all IPs in this session
		var totalBytesRx, totalBytesTx, totalPacketsRx, totalPacketsTx int64
		var totalSessions int
		ipStats := []map[string]interface{}{}

		for _, ip := range sess.AuthenticatedIPAddresses {
			stats := h.proxyManager.GetStatsByIP(ip.String())
			if stats != nil {
				// Add IP to stats
				stats["ip"] = ip.String()
				ipStats = append(ipStats, stats)

				// Aggregate totals
				if pktsRx, ok := stats["total_packets_received"].(int64); ok {
					totalPacketsRx += pktsRx
				}
				if pktsTx, ok := stats["total_packets_sent"].(int64); ok {
					totalPacketsTx += pktsTx
				}
				if rx, ok := stats["total_bytes_received"].(int64); ok {
					totalBytesRx += rx
				}
				if tx, ok := stats["total_bytes_sent"].(int64); ok {
					totalBytesTx += tx
				}
				if sessions, ok := stats["total_sessions"].(int); ok {
					totalSessions += sessions
				}
			}
		}

		sessionList = append(sessionList, map[string]interface{}{
			"session_id":        sess.SessionID,
			"username":          sess.Username,
			"user_id":           sess.UserID,
			"authenticated_ips": ipStrings,
			"created_at":        sess.CreatedAt,
			"expires_at":        sess.ExpiresAt,
			"allowed_services":  sess.AllowedServiceIDs,
			"total_packets_rx":  totalPacketsRx,
			"total_packets_tx":  totalPacketsTx,
			"total_bytes_rx":    totalBytesRx,
			"total_bytes_tx":    totalBytesTx,
			"total_sessions":    totalSessions,
			"ip_stats":          ipStats,
		})
	}

	c.JSON(200, models.NewAPIResponseWithCount("Active sessions retrieved", map[string]interface{}{
		"sessions": sessionList,
	}, len(sessionList)))
}

// HandleDelete handles DELETE /api/admin/sessions/:session_id
func (h *AdminSessionsHandler) HandleDelete(c *gin.Context) {
	sessionID := c.Param("session_id")

	// Get session details before terminating (to access IPs)
	session, err := h.sessionManager.GetSessionByID(sessionID)
	if err != nil {
		c.JSON(404, models.NewErrorResponse("Session not found", "SESSION_NOT_FOUND"))
		return
	}

	// Terminate session (removes from session manager)
	if err := h.sessionManager.TerminateSession(sessionID); err != nil {
		c.JSON(404, models.NewErrorResponse("Session not found", "SESSION_NOT_FOUND"))
		return
	}

	// Remove session IPs from allowlist instantly
	h.allowlistManager.RemoveSessionIP(sessionID)

	// Terminate all active proxy sessions for these IPs instantly
	totalTerminated := 0
	for _, ip := range session.AuthenticatedIPAddresses {
		terminated := h.proxyManager.TerminateSessionsByIP(ip.String())
		totalTerminated += terminated
	}

	log.Info().
		Str("session_id", sessionID).
		Str("username", session.Username).
		Int("proxy_sessions_terminated", totalTerminated).
		Msg("Admin terminated session")

	c.JSON(200, models.NewAPIResponse("Session terminated", nil))
}
