package handlers

import (
	"net/netip"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/session"
)

// BuildServiceAccessList generates detailed service access information for a client IP
// It checks access permissions via permanent IP ranges, DNS hostnames, and authenticated sessions
// Returns a list of services with their access status, reasons, and port information
func BuildServiceAccessList(cfg *config.ApplicationConfig, clientIP netip.Addr, userSession *session.Session, ipAllowlistReason string) []map[string]interface{} {
	serviceAccessList := make([]map[string]interface{}, 0, len(cfg.ProtectedServices))

	for _, service := range cfg.ProtectedServices {
		if !service.Enabled {
			continue
		}

		// Build service detail with port information
		serviceInfo := map[string]interface{}{
			"service_id":              service.ServiceID,
			"service_name":            service.ServiceName,
			"description":             service.Description,
			"proxy_listen_port_start": service.ProxyListenPortStart,
			"proxy_listen_port_end":   service.ProxyListenPortEnd,
			"transport_protocol":      service.TransportProtocol,
		}

		accessGranted := false
		accessReasons := make([]map[string]string, 0, 3)

		// Priority 1: Check permanent IP range access
		if ipAllowlistReason == "permanent" {
			accessGranted = true
			accessReasons = append(accessReasons, map[string]string{
				"method":      "permanent_ip_range",
				"description": "Your IP is in the permanently allowed IP ranges (unrestricted access)",
			})
		}

		// Priority 2: Check dynamic DNS hostname access
		if ipAllowlistReason == "dns_resolved" {
			accessGranted = true
			accessReasons = append(accessReasons, map[string]string{
				"method":      "dynamic_dns_hostname",
				"description": "Your IP matches an allowed dynamic DNS hostname (unrestricted access)",
			})
		}

		// Priority 3: Check session-based access
		if userSession != nil && userSession.IsIPAllowed(clientIP) {
			hasServiceAccess := isServiceAllowedForSession(userSession, service.ServiceID)

			if hasServiceAccess {
				accessGranted = true
				sessionScope := "all services"
				if len(userSession.AllowedServiceIDs) > 0 {
					sessionScope = "specific services only"
				}
				accessReasons = append(accessReasons, map[string]string{
					"method":      "authenticated_session",
					"description": "Session access (user: " + userSession.Username + ", scope: " + sessionScope + ")",
				})
			}
		}

		serviceInfo["access_granted"] = accessGranted
		serviceInfo["access_reasons"] = accessReasons

		if !accessGranted {
			serviceInfo["access_denied_reason"] = "No access method grants permission to this service"
		}

		serviceAccessList = append(serviceAccessList, serviceInfo)
	}

	return serviceAccessList
}

// isServiceAllowedForSession checks if a session has access to a specific service
// Returns true if the session has unrestricted access (empty AllowedServiceIDs) or explicitly includes the service
func isServiceAllowedForSession(userSession *session.Session, serviceID string) bool {
	// Empty AllowedServiceIDs means unrestricted access to all services
	if len(userSession.AllowedServiceIDs) == 0 {
		return true
	}

	// Check if service is in the allowed list
	for _, allowedID := range userSession.AllowedServiceIDs {
		if allowedID == serviceID {
			return true
		}
	}

	return false
}

// ExtractAllowedServiceDetails filters service access list to only include services the user can access
// Returns a simplified list with essential fields (id, name, ports, protocol, description)
func ExtractAllowedServiceDetails(serviceAccessList []map[string]interface{}, allowedServiceIDs []string) []map[string]interface{} {
	details := make([]map[string]interface{}, 0, len(serviceAccessList))

	// Empty allowedServiceIDs means user has access to all services
	allServicesAllowed := len(allowedServiceIDs) == 0

	for _, service := range serviceAccessList {
		serviceID, _ := service["service_id"].(string)

		// Check if user has access to this service
		hasAccess := allServicesAllowed
		if !hasAccess {
			for _, allowedID := range allowedServiceIDs {
				if allowedID == serviceID {
					hasAccess = true
					break
				}
			}
		}

		if hasAccess {
			// Extract only essential fields for simplified service list
			details = append(details, map[string]interface{}{
				"service_id":              service["service_id"],
				"service_name":            service["service_name"],
				"proxy_listen_port_start": service["proxy_listen_port_start"],
				"proxy_listen_port_end":   service["proxy_listen_port_end"],
				"transport_protocol":      service["transport_protocol"],
				"description":             service["description"],
			})
		}
	}

	return details
}
