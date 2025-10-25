package handlers

import (
	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/davbauer/knock-knock-portal/internal/session"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// AdminSessionsHandler handles admin session management
type AdminSessionsHandler struct {
	sessionManager *session.Manager
}

// NewAdminSessionsHandler creates a new handler
func NewAdminSessionsHandler(sessionManager *session.Manager) *AdminSessionsHandler {
	return &AdminSessionsHandler{
		sessionManager: sessionManager,
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
		
		sessionList = append(sessionList, map[string]interface{}{
			"session_id":         sess.SessionID,
			"username":           sess.Username,
			"user_id":            sess.UserID,
			"authenticated_ips":  ipStrings,
			"created_at":         sess.CreatedAt,
			"expires_at":         sess.ExpiresAt,
			"allowed_services":   sess.AllowedServiceIDs,
		})
	}

	c.JSON(200, models.NewAPIResponseWithCount("Active sessions retrieved", map[string]interface{}{
		"sessions": sessionList,
	}, len(sessionList)))
}

// HandleDelete handles DELETE /api/admin/sessions/:session_id
func (h *AdminSessionsHandler) HandleDelete(c *gin.Context) {
	sessionID := c.Param("session_id")

	if err := h.sessionManager.TerminateSession(sessionID); err != nil {
		c.JSON(404, models.NewErrorResponse("Session not found", "SESSION_NOT_FOUND"))
		return
	}

	log.Info().Str("session_id", sessionID).Msg("Admin terminated session")

	c.JSON(200, models.NewAPIResponse("Session terminated", nil))
}
