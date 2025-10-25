package config

import (
	"fmt"
	"net/netip"
	"strings"
)

// ValidateConfig validates the configuration for errors
func ValidateConfig(cfg *ApplicationConfig) error {
	// Validate session config
	if cfg.SessionConfig.DefaultSessionDurationSeconds < 1 {
		return fmt.Errorf("default_session_duration_seconds must be >= 1")
	}
	if cfg.SessionConfig.SessionCleanupIntervalSeconds < 1 {
		return fmt.Errorf("session_cleanup_interval_seconds must be >= 1")
	}

	// Validate network access control
	if cfg.NetworkAccessControl.DNSRefreshIntervalSeconds < 1 {
		return fmt.Errorf("dns_refresh_interval_seconds must be >= 1")
	}

	// Validate IP ranges
	for _, ipRange := range cfg.NetworkAccessControl.PermanentlyAllowedIPRanges {
		if _, err := netip.ParsePrefix(ipRange); err != nil {
			// Try parsing as single IP
			if _, err := netip.ParseAddr(ipRange); err != nil {
				return fmt.Errorf("invalid IP range '%s': %w", ipRange, err)
			}
		}
	}

	// Validate trusted proxy IP ranges
	if cfg.TrustedProxyConfig.Enabled {
		for _, ipRange := range cfg.TrustedProxyConfig.TrustedProxyIPRanges {
			if _, err := netip.ParsePrefix(ipRange); err != nil {
				if _, err := netip.ParseAddr(ipRange); err != nil {
					return fmt.Errorf("invalid trusted proxy IP range '%s': %w", ipRange, err)
				}
			}
		}
	}

	// Validate portal users
	for i, user := range cfg.PortalUserAccounts {
		if user.UserID == "" {
			return fmt.Errorf("portal user %d: user_id is required", i)
		}
		if user.Username == "" {
			return fmt.Errorf("portal user %d: username is required", i)
		}
		if user.BcryptHashedPassword == "" {
			return fmt.Errorf("portal user %s: bcrypt_hashed_password is required", user.Username)
		}
		if !strings.HasPrefix(user.BcryptHashedPassword, "$2a$") &&
			!strings.HasPrefix(user.BcryptHashedPassword, "$2b$") &&
			!strings.HasPrefix(user.BcryptHashedPassword, "$2y$") {
			return fmt.Errorf("portal user %s: bcrypt_hashed_password does not appear to be a valid bcrypt hash", user.Username)
		}
	}

	// Validate protected services
	for i, service := range cfg.ProtectedServices {
		if service.ServiceID == "" {
			return fmt.Errorf("protected service %d: service_id is required", i)
		}
		if service.ServiceName == "" {
			return fmt.Errorf("protected service %s: service_name is required", service.ServiceID)
		}

		// Validate port ranges
		if service.ProxyListenPortStart < 1 || service.ProxyListenPortStart > 65535 {
			return fmt.Errorf("service %s: invalid proxy_listen_port_start %d", service.ServiceID, service.ProxyListenPortStart)
		}
		if service.ProxyListenPortEnd < service.ProxyListenPortStart {
			return fmt.Errorf("service %s: proxy_listen_port_end must be >= proxy_listen_port_start", service.ServiceID)
		}
		if service.ProxyListenPortEnd > 65535 {
			return fmt.Errorf("service %s: invalid proxy_listen_port_end %d", service.ServiceID, service.ProxyListenPortEnd)
		}

		// Backend port validation
		if service.BackendTargetHost == "" {
			return fmt.Errorf("service %s: backend_target_host is required", service.ServiceID)
		}
		if service.BackendTargetPort < 1 || service.BackendTargetPort > 65535 {
			return fmt.Errorf("service %s: invalid backend_target_port %d", service.ServiceID, service.BackendTargetPort)
		}

		// Validate protocol
		protocol := strings.ToLower(service.TransportProtocol)
		if protocol != "tcp" && protocol != "udp" && protocol != "both" {
			return fmt.Errorf("service %s: transport_protocol must be 'tcp', 'udp', or 'both'", service.ServiceID)
		}

		// Validate backend host
		if service.BackendTargetHost == "" {
			return fmt.Errorf("service %s: backend_target_host is required", service.ServiceID)
		}
	}

	// Check for port conflicts between services
	if err := checkPortConflicts(cfg.ProtectedServices); err != nil {
		return err
	}

	return nil
}

// checkPortConflicts ensures no two services listen on the same port
func checkPortConflicts(services []ProtectedServiceConfig) error {
	portMap := make(map[int]string) // port -> service_id

	for _, service := range services {
		if !service.Enabled {
			continue
		}

		for port := service.ProxyListenPortStart; port <= service.ProxyListenPortEnd; port++ {
			if existingServiceID, exists := portMap[port]; exists {
				return fmt.Errorf("port conflict: port %d is used by both service %s and %s",
					port, existingServiceID, service.ServiceID)
			}
			portMap[port] = service.ServiceID
		}
	}

	return nil
}
