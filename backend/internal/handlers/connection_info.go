package handlers

import (
	"net/netip"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/middleware"
	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/davbauer/knock-knock-portal/internal/session"
	"github.com/gin-gonic/gin"
)

// ConnectionInfoHandler handles connection information requests
type ConnectionInfoHandler struct {
	ipAllowListManager *ipallowlist.Manager
	sessionManager     *session.Manager
	configLoader       *config.Loader
	ipExtractor        *middleware.RealIPExtractor
}

// NewConnectionInfoHandler creates a new connection info handler
func NewConnectionInfoHandler(ipAllowListManager *ipallowlist.Manager, sessionManager *session.Manager, configLoader *config.Loader, ipExtractor *middleware.RealIPExtractor) *ConnectionInfoHandler {
	return &ConnectionInfoHandler{
		ipAllowListManager: ipAllowListManager,
		sessionManager:     sessionManager,
		configLoader:       configLoader,
		ipExtractor:        ipExtractor,
	}
}

// HandleCheck processes GET /api/connection-info
// This is a public endpoint that provides connection information for the client
func (h *ConnectionInfoHandler) HandleCheck(c *gin.Context) {
	// Get client IP (properly extracted by middleware, handles trusted proxies)
	clientIP, hasIP := middleware.GetClientIP(c)

	if !hasIP || !clientIP.IsValid() {
		c.JSON(400, models.NewErrorResponse("Could not determine client IP", "INVALID_IP"))
		return
	}

	clientIPStr := clientIP.String()

	// Check if there's an untrusted proxy warning using middleware's detection
	proxyWarning := h.ipExtractor.GetProxyWarning(c)

	// Check if IP is allowed at the base level (permanent/DNS/session)
	allowed, baseReason := h.ipAllowListManager.IsIPAllowed(clientIP)

	// Build detailed response
	response := map[string]interface{}{
		"client_ip": clientIPStr,
		"allowed":   allowed,
	}

	// Add proxy warning if present
	if proxyWarning != nil {
		response["proxy_warning"] = *proxyWarning
	}

	// Get all protected services from config
	cfg := h.configLoader.GetConfig()
	serviceAccessList := []map[string]interface{}{}

	// Check if user has an active session (optional - works without auth too)
	var userSession *session.Session
	var sessionAllowedServiceIDs []string
	claims, hasSession := middleware.GetJWTClaims(c)

	if hasSession {
		sess, err := h.sessionManager.GetSessionByID(claims.SessionID)
		if err == nil && !sess.IsExpired() {
			userSession = sess
			sessionAllowedServiceIDs = sess.AllowedServiceIDs
		}
	}

	// Build service access information
	for _, service := range cfg.ProtectedServices {
		if !service.Enabled {
			continue
		}

		serviceAccess := map[string]interface{}{
			"service_id":   service.ServiceID,
			"service_name": service.ServiceName,
			"description":  service.Description,
		}

		// Determine access and why
		accessGranted := false
		accessReasons := []map[string]string{}

		// 1. Check if IP is allowed via permanent IP range
		if baseReason == "permanent" {
			accessGranted = true
			accessReasons = append(accessReasons, map[string]string{
				"method":      "permanent_ip_range",
				"description": "Your IP is in the permanently allowed IP ranges (unrestricted access to all services)",
			})
		}

		// 2. Check if IP is allowed via DNS hostname
		if baseReason == "dns_resolved" {
			accessGranted = true
			accessReasons = append(accessReasons, map[string]string{
				"method":      "dynamic_dns_hostname",
				"description": "Your IP matches an allowed dynamic DNS hostname (unrestricted access to all services)",
			})
		}

		// 3. Check session-based access (if user is logged in)
		if userSession != nil && userSession.IsIPAllowed(clientIP) {
			// Check if user has access to this specific service
			hasServiceAccess := len(sessionAllowedServiceIDs) == 0 // Empty = all services
			if !hasServiceAccess {
				for _, allowedID := range sessionAllowedServiceIDs {
					if allowedID == service.ServiceID {
						hasServiceAccess = true
						break
					}
				}
			}

			if hasServiceAccess {
				accessGranted = true
				sessionScope := "all services"
				if len(sessionAllowedServiceIDs) > 0 {
					sessionScope = "specific services only"
				}
				accessReasons = append(accessReasons, map[string]string{
					"method":      "authenticated_session",
					"description": "Access granted via authenticated session (user: " + userSession.Username + ", scope: " + sessionScope + ")",
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

	// Add overall access method (backward compatible)
	if allowed {
		switch baseReason {
		case "permanent":
			response["access_method"] = "permanent_ip_range"
			response["access_description"] = "Your IP is in the permanently allowed IP ranges"
		case "dns_resolved":
			response["access_method"] = "dynamic_dns_hostname"
			response["access_description"] = "Your IP matches an allowed dynamic DNS hostname"
		case "session":
			response["access_method"] = "authenticated_session"
			response["access_description"] = "Access granted via authenticated session"
		default:
			response["access_method"] = "allowed"
			response["access_description"] = "IP is allowed"
		}
	} else {
		response["access_method"] = "not_allowed"
		response["access_description"] = "Your IP is not in the allowlist. Please login to gain access."
	}

	// Add service-level details
	response["services"] = serviceAccessList
	response["total_services"] = len(serviceAccessList)

	// Add session info if present
	if userSession != nil {
		response["session_username"] = userSession.Username
		response["session_active"] = true
	} else {
		response["session_active"] = false
	}

	c.JSON(200, models.NewAPIResponse("Connection info retrieved", response))
}
