package proxy

import (
	"fmt"
	"sync"
	"time"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/rs/zerolog/log"
)

// Proxy is the interface for all proxy types
type Proxy interface {
	Start() error
	Stop() error
	GetStats() map[string]interface{}
}

// Manager handles lifecycle of all proxy instances
type Manager struct {
	configLoader     *config.Loader
	allowlistManager *ipallowlist.Manager
	proxies          map[string]Proxy
	mu               sync.RWMutex
}

// NewManager creates a new proxy manager
func NewManager(configLoader *config.Loader, allowlistManager *ipallowlist.Manager) *Manager {
	return &Manager{
		configLoader:     configLoader,
		allowlistManager: allowlistManager,
		proxies:          make(map[string]Proxy),
	}
}

// Start initializes and starts all enabled proxies
func (m *Manager) Start() error {
	cfg := m.configLoader.GetConfig()
	
	log.Info().
		Int("total_services", len(cfg.ProtectedServices)).
		Msg("Starting proxy manager")
	
	for i, service := range cfg.ProtectedServices {
		if !service.Enabled {
			log.Info().
				Str("service", service.ServiceName).
				Msg("Service disabled, skipping")
			continue
		}
		
		// Validate service configuration
		if err := m.validateService(&service); err != nil {
			log.Error().
				Err(err).
				Str("service", service.ServiceName).
				Msg("Invalid service configuration, skipping")
			continue
		}
		
		var proxy Proxy
		var err error
		
		// Create appropriate proxy type
		if service.IsHTTPProtocol {
			proxy, err = NewHTTPProxy(&cfg.ProtectedServices[i], m.allowlistManager)
			if err != nil {
				log.Error().
					Err(err).
					Str("service", service.ServiceName).
					Msg("Failed to create HTTP proxy")
				continue
			}
		} else if service.TransportProtocol == "tcp" {
			proxy = NewTCPProxy(&cfg.ProtectedServices[i], m.allowlistManager)
		} else if service.TransportProtocol == "udp" {
			// Get UDP session timeout from config
			sessionTimeout := time.Duration(cfg.ProxyServerConfig.UDPSessionTimeoutSeconds) * time.Second
			proxy = NewUDPProxy(&cfg.ProtectedServices[i], m.allowlistManager, sessionTimeout)
		} else if service.TransportProtocol == "both" {
			log.Warn().
				Str("service", service.ServiceName).
				Msg("'both' protocol requires two service definitions (one TCP, one UDP), skipping")
			continue
		} else {
			log.Warn().
				Str("service", service.ServiceName).
				Str("protocol", service.TransportProtocol).
				Msg("Unknown protocol, skipping")
			continue
		}
		
		// Start the proxy
		if err := proxy.Start(); err != nil {
			log.Error().
				Err(err).
				Str("service", service.ServiceName).
				Msg("Failed to start proxy")
			continue
		}
		
		m.mu.Lock()
		m.proxies[service.ServiceID] = proxy
		m.mu.Unlock()
		
		log.Info().
			Str("service", service.ServiceName).
			Str("service_id", service.ServiceID).
			Int("listen_port", service.ProxyListenPortStart).
			Str("protocol", service.TransportProtocol).
			Bool("is_http", service.IsHTTPProtocol).
			Msg("Proxy started successfully")
	}
	
	m.mu.RLock()
	activeCount := len(m.proxies)
	m.mu.RUnlock()
	
	log.Info().
		Int("active_proxies", activeCount).
		Msg("Proxy manager started")
	
	return nil
}

// Stop gracefully shuts down all proxies
func (m *Manager) Stop() error {
	log.Info().Msg("Stopping proxy manager")
	
	m.mu.Lock()
	proxies := make(map[string]Proxy)
	for k, v := range m.proxies {
		proxies[k] = v
	}
	m.mu.Unlock()
	
	var wg sync.WaitGroup
	for serviceID, proxy := range proxies {
		wg.Add(1)
		go func(sid string, p Proxy) {
			defer wg.Done()
			if err := p.Stop(); err != nil {
				log.Error().
					Err(err).
					Str("service_id", sid).
					Msg("Error stopping proxy")
			}
		}(serviceID, proxy)
	}
	
	wg.Wait()
	
	m.mu.Lock()
	m.proxies = make(map[string]Proxy)
	m.mu.Unlock()
	
	log.Info().Msg("Proxy manager stopped")
	
	return nil
}

// GetStats returns statistics for all active proxies
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := make(map[string]interface{})
	stats["active_proxies"] = len(m.proxies)
	
	services := make([]map[string]interface{}, 0, len(m.proxies))
	for _, proxy := range m.proxies {
		services = append(services, proxy.GetStats())
	}
	stats["services"] = services
	
	return stats
}

// validateService checks if a service configuration is valid
func (m *Manager) validateService(service *config.ProtectedServiceConfig) error {
	if service.ServiceID == "" {
		return fmt.Errorf("service_id is required")
	}
	
	if service.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	
	if service.ProxyListenPortStart < 1 || service.ProxyListenPortStart > 65535 {
		return fmt.Errorf("invalid proxy_listen_port_start: %d", service.ProxyListenPortStart)
	}
	
	if service.BackendTargetPortStart < 1 || service.BackendTargetPortStart > 65535 {
		return fmt.Errorf("invalid backend_target_port_start: %d", service.BackendTargetPortStart)
	}
	
	if service.BackendTargetHost == "" {
		return fmt.Errorf("backend_target_host is required")
	}
	
	if service.TransportProtocol != "tcp" && service.TransportProtocol != "udp" && service.TransportProtocol != "both" {
		return fmt.Errorf("invalid transport_protocol: %s (must be tcp, udp, or both)", service.TransportProtocol)
	}
	
	return nil
}
