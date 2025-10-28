package proxy

import (
	"fmt"
	"sync"
	"time"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/ipblocklist"
	"github.com/rs/zerolog/log"
)

// Proxy is the interface for all proxy types
type Proxy interface {
	Start() error
	Stop() error
	GetStats() map[string]interface{}
	TerminateConnectionsByIP(clientIP string) int
}

// Manager handles lifecycle of all proxy instances
type Manager struct {
	configLoader     *config.Loader
	allowlistManager *ipallowlist.Manager
	blocklistManager *ipblocklist.Manager
	proxies          map[string]Proxy
	mu               sync.RWMutex
	stopStatsTicker  chan struct{}
}

// NewManager creates a new proxy manager
func NewManager(configLoader *config.Loader, allowlistManager *ipallowlist.Manager, blocklistManager *ipblocklist.Manager) *Manager {
	return &Manager{
		configLoader:     configLoader,
		allowlistManager: allowlistManager,
		blocklistManager: blocklistManager,
		proxies:          make(map[string]Proxy),
		stopStatsTicker:  make(chan struct{}),
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

		// Get connection limit from config
		maxConnections := cfg.ProxyServerConfig.MaxConnectionsPerService

		// Create appropriate proxy type
		if service.IsHTTPProtocol {
			proxy, err = NewHTTPProxy(&cfg.ProtectedServices[i], m.allowlistManager, m.blocklistManager)
			if err != nil {
				log.Error().
					Err(err).
					Str("service", service.ServiceName).
					Msg("Failed to create HTTP proxy")
				continue
			}
		} else if service.TransportProtocol == "tcp" {
			proxy = NewTCPProxy(&cfg.ProtectedServices[i], m.allowlistManager, m.blocklistManager, maxConnections)
		} else if service.TransportProtocol == "udp" {
			// Get UDP session timeout from config
			sessionTimeout := time.Duration(cfg.ProxyServerConfig.UDPSessionTimeoutSeconds) * time.Second
			proxy = NewUDPProxy(&cfg.ProtectedServices[i], m.allowlistManager, m.blocklistManager, sessionTimeout, maxConnections)
		} else if service.TransportProtocol == "both" {
			// Create both TCP and UDP proxies for the same service
			sessionTimeout := time.Duration(cfg.ProxyServerConfig.UDPSessionTimeoutSeconds) * time.Second
			
			// Start TCP proxy
			tcpProxy := NewTCPProxy(&cfg.ProtectedServices[i], m.allowlistManager, m.blocklistManager, maxConnections)
			if err := tcpProxy.Start(); err != nil {
				log.Error().
					Err(err).
					Str("service", service.ServiceName).
					Str("protocol", "tcp").
					Msg("Failed to start TCP proxy")
			} else {
				m.mu.Lock()
				m.proxies[service.ServiceID+"-tcp"] = tcpProxy
				m.mu.Unlock()
				log.Info().
					Str("service", service.ServiceName).
					Str("service_id", service.ServiceID).
					Int("listen_port", service.ProxyListenPortStart).
					Str("protocol", "tcp").
					Msg("TCP proxy started successfully")
			}
			
			// Start UDP proxy
			udpProxy := NewUDPProxy(&cfg.ProtectedServices[i], m.allowlistManager, m.blocklistManager, sessionTimeout, maxConnections)
			if err := udpProxy.Start(); err != nil {
				log.Error().
					Err(err).
					Str("service", service.ServiceName).
					Str("protocol", "udp").
					Msg("Failed to start UDP proxy")
			} else {
				m.mu.Lock()
				m.proxies[service.ServiceID+"-udp"] = udpProxy
				m.mu.Unlock()
				log.Info().
					Str("service", service.ServiceName).
					Str("service_id", service.ServiceID).
					Int("listen_port", service.ProxyListenPortStart).
					Str("protocol", "udp").
					Msg("UDP proxy started successfully")
			}
			
			continue // Skip the normal start logic below
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

	// Start periodic stats logging
	go m.statsLogger()

	return nil
}

// statsLogger logs connection statistics every 10 seconds
func (m *Manager) statsLogger() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.logStats()
		case <-m.stopStatsTicker:
			return
		}
	}
}

// logStats logs current connection statistics for all proxies
func (m *Manager) logStats() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.proxies) == 0 {
		return
	}

	totalConnections := 0
	serviceStats := []map[string]interface{}{}

	for _, proxy := range m.proxies {
		stats := proxy.GetStats()
		// Count both TCP connections and UDP sessions
		if connections, ok := stats["active_connections"].(int32); ok {
			totalConnections += int(connections)
		}
		if sessions, ok := stats["active_sessions"].(int); ok {
			totalConnections += sessions
		}
		serviceStats = append(serviceStats, stats)
	}

	log.Info().
		Int("total_active_connections", totalConnections).
		Int("active_services", len(m.proxies)).
		Interface("services", serviceStats).
		Msg("Proxy connection stats")
}

// Stop gracefully shuts down all proxies
func (m *Manager) Stop() error {
	log.Info().Msg("Stopping proxy manager")

	// Stop stats logger safely
	select {
	case <-m.stopStatsTicker:
		// Already closed
	default:
		close(m.stopStatsTicker)
	}

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

// TerminateSessionsByIP closes all active TCP/UDP sessions for a specific IP
// This is called when a session is terminated and IPs need to be instantly disconnected
func (m *Manager) TerminateSessionsByIP(clientIP string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalTerminated := 0

	for serviceID, proxy := range m.proxies {
		var terminated int

		// Check proxy type and terminate accordingly
		if udpProxy, ok := proxy.(*UDPProxy); ok {
			terminated = udpProxy.TerminateSessionsByIP(clientIP)
		} else if tcpProxy, ok := proxy.(*TCPProxy); ok {
			terminated = tcpProxy.TerminateSessionsByIP(clientIP)
		}

		if terminated > 0 {
			log.Info().
				Str("service_id", serviceID).
				Str("client_ip", clientIP).
				Int("connections_terminated", terminated).
				Msg("Terminated sessions for IP")
			totalTerminated += terminated
		}
	}

	return totalTerminated
}

// GetStatsByIP returns aggregated statistics for a specific client IP across all proxies
func (m *Manager) GetStatsByIP(clientIP string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	aggregated := map[string]interface{}{
		"total_packets_received": int64(0),
		"total_packets_sent":     int64(0),
		"total_bytes_received":   int64(0),
		"total_bytes_sent":       int64(0),
		"total_sessions":         0, // Combined TCP + UDP sessions
		"services":               []map[string]interface{}{},
	}

	var totalPacketsRx, totalPacketsTx, totalBytesRx, totalBytesTx int64
	var totalSessions int
	services := []map[string]interface{}{}

	for serviceID, proxy := range m.proxies {
		var stats map[string]interface{}

		if udpProxy, ok := proxy.(*UDPProxy); ok {
			stats = udpProxy.GetStatsByIP(clientIP)
		} else if tcpProxy, ok := proxy.(*TCPProxy); ok {
			stats = tcpProxy.GetStatsByIP(clientIP)
		}

		if stats != nil {
			if sessions, ok := stats["active_sessions"].(int); ok && sessions > 0 {
				stats["service_id"] = serviceID
				services = append(services, stats)
				totalSessions += sessions
			}

			if pktsRx, ok := stats["packets_received"].(int64); ok {
				totalPacketsRx += pktsRx
			}
			if pktsTx, ok := stats["packets_sent"].(int64); ok {
				totalPacketsTx += pktsTx
			}
			if rx, ok := stats["bytes_received"].(int64); ok {
				totalBytesRx += rx
			}
			if tx, ok := stats["bytes_sent"].(int64); ok {
				totalBytesTx += tx
			}
		}
	}

	aggregated["total_packets_received"] = totalPacketsRx
	aggregated["total_packets_sent"] = totalPacketsTx
	aggregated["total_bytes_received"] = totalBytesRx
	aggregated["total_bytes_sent"] = totalBytesTx
	aggregated["total_sessions"] = totalSessions
	aggregated["services"] = services

	return aggregated
}

// Reload stops all existing proxies and restarts them with new config
func (m *Manager) Reload() error {
	log.Info().Msg("Reloading proxy manager with new configuration")

	// Stop all existing proxies
	if err := m.Stop(); err != nil {
		log.Error().Err(err).Msg("Error during proxy manager stop")
	}

	// Recreate the stopStatsTicker channel for the new stats logger
	m.stopStatsTicker = make(chan struct{})

	// Start with new configuration
	return m.Start()
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

	if service.BackendTargetPort < 1 || service.BackendTargetPort > 65535 {
		return fmt.Errorf("invalid backend_target_port: %d", service.BackendTargetPort)
	}

	if service.BackendTargetHost == "" {
		return fmt.Errorf("backend_target_host is required")
	}

	if service.TransportProtocol != "tcp" && service.TransportProtocol != "udp" && service.TransportProtocol != "both" {
		return fmt.Errorf("invalid transport_protocol: %s (must be tcp, udp, or both)", service.TransportProtocol)
	}

	return nil
}

// TerminateConnectionsByIP terminates all active connections from a specific IP across all proxies
func (m *Manager) TerminateConnectionsByIP(clientIP string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if clientIP == "" {
		return fmt.Errorf("client IP is required")
	}
	
	totalTerminated := 0
	for _, proxy := range m.proxies {
		terminated := proxy.TerminateConnectionsByIP(clientIP)
		totalTerminated += terminated
	}
	
	log.Info().
		Str("client_ip", clientIP).
		Int("total_terminated", totalTerminated).
		Msg("Terminated connections across all proxies")
	
	return nil
}
