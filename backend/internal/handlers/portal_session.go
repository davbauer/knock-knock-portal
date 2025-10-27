package handlers

import (
	"net/netip"
	"time"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/middleware"
	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/davbauer/knock-knock-portal/internal/session"
	"github.com/davbauer/knock-knock-portal/internal/utils"
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

	// Get service names and detailed access info
	cfg := h.configLoader.GetConfig()
	allowedServices := utils.GetServiceNames(cfg, sess.AllowedServiceIDs)
	serviceAccessList := h.getServiceAccessDetails(cfg, clientIP, sess)

	response := map[string]interface{}{
		"session": map[string]interface{}{
			"session_id":          sess.SessionID,
			"username":            sess.Username,
			"user_id":             sess.UserID,
			"authenticated_ips":   ipStrings,
			"current_ip":          clientIP.String(),
			"current_ip_allowed":  sess.IsIPAllowed(clientIP),
			"created_at":          sess.CreatedAt,
			"last_activity_at":    sess.LastActivityAt,
			"expires_at":          sess.ExpiresAt,
			"expires_in_seconds":  int(expiresIn),
			"auto_extend_enabled": sess.AutoExtendEnabled,
			"allowed_service_ids": sess.AllowedServiceIDs,
			"allowed_services":    allowedServices,
			"services":            serviceAccessList,
			"total_services":      len(serviceAccessList),
			"active":              !sess.IsExpired(),
		},
	}

	c.JSON(200, models.NewAPIResponse("Session status retrieved", response))
}

// getServiceAccessDetails returns detailed service access information
func (h *PortalSessionHandler) getServiceAccessDetails(cfg *config.ApplicationConfig, clientIP netip.Addr, sess *session.Session) []map[string]interface{} {
	serviceAccessList := []map[string]interface{}{}

	// Check base IP allowlist status
	_, baseReason := h.ipAllowListManager.IsIPAllowed(clientIP)

	for _, service := range cfg.ProtectedServices {
		if !service.Enabled {
			continue
		}

		serviceAccess := map[string]interface{}{
			"service_id":   service.ServiceID,
			"service_name": service.ServiceName,
			"description":  service.Description,
		}

		accessGranted := false
		accessReasons := []map[string]string{}

		// 1. Check if IP is allowed via permanent IP range
		if baseReason == "permanent" {
			accessGranted = true
			accessReasons = append(accessReasons, map[string]string{
				"method":      "permanent_ip_range",
				"description": "Your IP is in the permanently allowed IP ranges (unrestricted access)",
			})
		}

		// 2. Check if IP is allowed via DNS hostname
		if baseReason == "dns_resolved" {
			accessGranted = true
			accessReasons = append(accessReasons, map[string]string{
				"method":      "dynamic_dns_hostname",
				"description": "Your IP matches an allowed dynamic DNS hostname (unrestricted access)",
			})
		}

		// 3. Check session-based access
		if sess.IsIPAllowed(clientIP) {
			// Check if user has access to this specific service
			hasServiceAccess := len(sess.AllowedServiceIDs) == 0 // Empty = all services
			if !hasServiceAccess {
				for _, allowedID := range sess.AllowedServiceIDs {
					if allowedID == service.ServiceID {
						hasServiceAccess = true
						break
					}
				}
			}

			if hasServiceAccess {
				accessGranted = true
				sessionScope := "all services"
				if len(sess.AllowedServiceIDs) > 0 {
					sessionScope = "specific services only"
				}
				accessReasons = append(accessReasons, map[string]string{
					"method":      "authenticated_session",
					"description": "Session access (user: " + sess.Username + ", scope: " + sessionScope + ")",
				})
			}
		}

		serviceAccess["access_granted"] = accessGranted
		serviceAccess["access_reasons"] = accessReasons

		if !accessGranted {
			serviceAccess["access_denied_reason"] = "No access method grants permission to this service"
		}

		serviceAccessList = append(serviceAccessList, serviceAccess)
	}

	return serviceAccessList
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
