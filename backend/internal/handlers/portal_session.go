package handlers

import (
	"time"

	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/davbauer/knock-knock-portal/internal/middleware"
	"github.com/davbauer/knock-knock-portal/internal/session"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// PortalSessionHandler handles session operations
type PortalSessionHandler struct {
	sessionManager *session.Manager
}

// NewPortalSessionHandler creates a new handler
func NewPortalSessionHandler(sessionManager *session.Manager) *PortalSessionHandler {
	return &PortalSessionHandler{
		sessionManager: sessionManager,
	}
}

// HandleStatus handles GET /api/portal/session/status
func (h *PortalSessionHandler) HandleStatus(c *gin.Context) {
	claims, ok := middleware.GetJWTClaims(c)
	if !ok {
		c.JSON(401, models.NewErrorResponse("Unauthorized", "UNAUTHORIZED"))
		return
	}

	sess, err := h.sessionManager.GetSessionByID(claims.SessionID)
	if err != nil {
		c.JSON(404, models.NewErrorResponse("Session not found or expired", "SESSION_NOT_FOUND"))
		return
	}

	expiresIn := time.Until(sess.ExpiresAt).Seconds()
	if expiresIn < 0 {
		expiresIn = 0
	}

	response := map[string]interface{}{
		"active":             !sess.IsExpired(),
		"expires_in_seconds": int(expiresIn),
		"authenticated_ip":   sess.ClientIPAddress.String(),
		"allowed_services":   sess.AllowedServiceIDs,
		"auto_extend_enabled": sess.AutoExtendEnabled,
	}

	c.JSON(200, models.NewAPIResponse("Session active", response))
}

// HandleLogout handles POST /api/portal/session/logout
func (h *PortalSessionHandler) HandleLogout(c *gin.Context) {
	claims, ok := middleware.GetJWTClaims(c)
	if !ok {
		c.JSON(401, models.NewErrorResponse("Unauthorized", "UNAUTHORIZED"))
		return
	}

	if err := h.sessionManager.TerminateSession(claims.SessionID); err != nil {
		log.Warn().Err(err).Str("session_id", claims.SessionID).Msg("Failed to terminate session")
	}

	c.JSON(200, models.NewAPIResponse("Session terminated successfully", nil))
}
