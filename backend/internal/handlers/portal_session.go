package handlers

import (
	"time"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/middleware"
	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/davbauer/knock-knock-portal/internal/session"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// PortalSessionHandler handles session operations
type PortalSessionHandler struct {
	sessionManager     *session.Manager
	configLoader       *config.Loader
	ipAllowListManager *ipallowlist.Manager
}

// NewPortalSessionHandler creates a new handler
func NewPortalSessionHandler(sessionManager *session.Manager, configLoader *config.Loader, ipAllowListManager *ipallowlist.Manager) *PortalSessionHandler {
	return &PortalSessionHandler{
		sessionManager:     sessionManager,
		configLoader:       configLoader,
		ipAllowListManager: ipAllowListManager,
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

	// Get client IP
	clientIP, ok := middleware.GetClientIP(c)
	if !ok || !clientIP.IsValid() {
		c.JSON(400, models.NewErrorResponse("Could not determine client IP", "INVALID_IP"))
		return
	}

	// Convert authenticated IP addresses to strings
	ipStrings := make([]string, len(sess.AuthenticatedIPAddresses))
	for i, ip := range sess.AuthenticatedIPAddresses {
		ipStrings[i] = ip.String()
	}

	expiresIn := time.Until(sess.ExpiresAt).Seconds()
	if expiresIn < 0 {
		expiresIn = 0
	}

	// Get service information
	cfg := h.configLoader.GetConfig()
	_, ipAllowlistReason := h.ipAllowListManager.IsIPAllowed(clientIP)
	serviceAccessList := BuildServiceAccessList(cfg, clientIP, sess, ipAllowlistReason)

	// Extract simplified service details for user's allowed services only
	allowedServiceDetails := ExtractAllowedServiceDetails(serviceAccessList, sess.AllowedServiceIDs)

	response := map[string]interface{}{
		"session": map[string]interface{}{
			"session_id":              sess.SessionID,
			"username":                sess.Username,
			"user_id":                 sess.UserID,
			"authenticated_ips":       ipStrings,
			"current_ip":              clientIP.String(),
			"current_ip_allowed":      sess.IsIPAllowed(clientIP),
			"created_at":              sess.CreatedAt,
			"last_activity_at":        sess.LastActivityAt,
			"expires_at":              sess.ExpiresAt,
			"expires_in_seconds":      int(expiresIn),
			"auto_extend_enabled":     sess.AutoExtendEnabled,
			"allowed_service_ids":     sess.AllowedServiceIDs,
			"allowed_service_details": allowedServiceDetails,
			"services":                serviceAccessList,
			"total_services":          len(serviceAccessList),
			"active":                  !sess.IsExpired(),
		},
	}

	c.JSON(200, models.NewAPIResponse("Session status retrieved", response))
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

// HandleAddIP handles POST /api/portal/session/add-ip
func (h *PortalSessionHandler) HandleAddIP(c *gin.Context) {
	claims, ok := middleware.GetJWTClaims(c)
	if !ok {
		c.JSON(401, models.NewErrorResponse("Unauthorized", "UNAUTHORIZED"))
		return
	}

	// Get client IP
	clientIP, ok := middleware.GetClientIP(c)
	if !ok || !clientIP.IsValid() {
		c.JSON(400, models.NewErrorResponse("Could not determine client IP", "INVALID_IP"))
		return
	}

	// Add IP to session
	if err := h.sessionManager.AddIPToSession(claims.SessionID, clientIP); err != nil {
		if err.Error() == "IP already exists in session" {
			c.JSON(400, models.NewErrorResponse("IP already authorized for this session", "IP_ALREADY_EXISTS"))
			return
		}
		c.JSON(400, models.NewErrorResponse(err.Error(), "ADD_IP_FAILED"))
		return
	}

	log.Info().
		Str("session_id", claims.SessionID).
		Str("user_id", claims.UserID).
		Str("new_ip", clientIP.String()).
		Msg("User added new IP to session")

	c.JSON(200, models.NewAPIResponse("IP address added to session", map[string]interface{}{
		"added_ip": clientIP.String(),
	}))
}

// HandleExtendSession handles POST /api/portal/session/extend
func (h *PortalSessionHandler) HandleExtendSession(c *gin.Context) {
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

	// Get the default session duration from config
	cfg := h.configLoader.GetConfig()
	extendDuration := time.Duration(cfg.SessionConfig.DefaultSessionDurationSeconds) * time.Second

	// Extend the session
	sess.ExtendSession(extendDuration)

	// Calculate new expiry time
	expiresIn := time.Until(sess.ExpiresAt).Seconds()
	if expiresIn < 0 {
		expiresIn = 0
	}

	log.Info().
		Str("session_id", claims.SessionID).
		Str("user_id", claims.UserID).
		Str("new_expiry", sess.ExpiresAt.Format(time.RFC3339)).
		Msg("User manually extended session")

	c.JSON(200, models.NewAPIResponse("Session extended successfully", map[string]interface{}{
		"expires_at":         sess.ExpiresAt,
		"expires_in_seconds": int(expiresIn),
	}))
}
