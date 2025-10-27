package utils

import "github.com/davbauer/knock-knock-portal/internal/config"

// GetServiceNames returns service names for allowed service IDs
// If allowedServiceIDs is empty, returns all enabled service names
func GetServiceNames(cfg *config.ApplicationConfig, allowedServiceIDs []string) []string {
	if len(allowedServiceIDs) == 0 {
		// Return all enabled service names (pre-allocate for efficiency)
		names := make([]string, 0, len(cfg.ProtectedServices))
		for _, svc := range cfg.ProtectedServices {
			if svc.Enabled {
				names = append(names, svc.ServiceName)
			}
		}
		return names
	}

	// Return specific service names (pre-allocate for efficiency)
	names := make([]string, 0, len(allowedServiceIDs))
	for _, allowedID := range allowedServiceIDs {
		for _, svc := range cfg.ProtectedServices {
			if svc.ServiceID == allowedID && svc.Enabled {
				names = append(names, svc.ServiceName)
				break
			}
		}
	}
	return names
}

// GetServiceByID retrieves a service configuration by its ID
// Returns nil if service not found
func GetServiceByID(cfg *config.ApplicationConfig, serviceID string) *config.ProtectedServiceConfig {
	for i := range cfg.ProtectedServices {
		if cfg.ProtectedServices[i].ServiceID == serviceID {
			return &cfg.ProtectedServices[i]
		}
	}
	return nil
}

// GetEnabledServices returns all enabled services
func GetEnabledServices(cfg *config.ApplicationConfig) []config.ProtectedServiceConfig {
	enabled := make([]config.ProtectedServiceConfig, 0, len(cfg.ProtectedServices))
	for _, svc := range cfg.ProtectedServices {
		if svc.Enabled {
			enabled = append(enabled, svc)
		}
	}
	return enabled
}
