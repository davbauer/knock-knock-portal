package ipblocklist

import (
	"net"
	"sync"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/rs/zerolog/log"
)

// Manager handles IP blocklist checking with HIGHEST priority
// Blocked IPs cannot login, authenticate, or use any proxy services
type Manager struct {
	mu              sync.RWMutex
	blockedIPs      map[string]bool // Specific IPs that are blocked
	blockedCIDRs    []*net.IPNet    // CIDR ranges that are blocked
}

// NewManager creates a new blocklist manager
func NewManager(cfg *config.NetworkAccessControlConfig) *Manager {
	m := &Manager{
		blockedIPs:   make(map[string]bool),
		blockedCIDRs: make([]*net.IPNet, 0),
	}
	
	m.Reload(cfg)
	return m
}

// Reload updates the blocklist from configuration
func (m *Manager) Reload(cfg *config.NetworkAccessControlConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Clear existing blocklists
	m.blockedIPs = make(map[string]bool)
	m.blockedCIDRs = make([]*net.IPNet, 0)
	
	// Parse blocked IP addresses (supports both individual IPs and CIDR ranges)
	for _, ipStr := range cfg.BlockedIPAddresses {
		// Try parsing as CIDR first
		_, ipNet, err := net.ParseCIDR(ipStr)
		if err == nil {
			// It's a valid CIDR range
			m.blockedCIDRs = append(m.blockedCIDRs, ipNet)
			log.Info().
				Str("cidr", ipStr).
				Msg("Added CIDR range to blocklist")
			continue
		}
		
		// Try parsing as individual IP
		ip := net.ParseIP(ipStr)
		if ip == nil {
			log.Warn().
				Str("ip", ipStr).
				Msg("Invalid blocked IP address or CIDR range, skipping")
			continue
		}
		
		m.blockedIPs[ip.String()] = true
		log.Info().
			Str("ip", ip.String()).
			Msg("Added IP to blocklist")
	}
	
	log.Info().
		Int("blocked_ips", len(m.blockedIPs)).
		Int("blocked_cidrs", len(m.blockedCIDRs)).
		Msg("Blocklist reloaded")
}

// IsIPBlocked checks if an IP is on the blocklist (HIGHEST PRIORITY CHECK)
// Returns true if blocked, false if allowed
// Also returns reason for blocking
func (m *Manager) IsIPBlocked(ip net.IP) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if ip == nil {
		return true, "invalid IP address"
	}
	
	// Check specific blocked IPs first
	if m.blockedIPs[ip.String()] {
		log.Warn().
			Str("ip", ip.String()).
			Msg("IP blocked: matches blocklist")
		return true, "IP is on blocklist"
	}
	
	// Check blocked CIDR ranges
	for _, cidr := range m.blockedCIDRs {
		if cidr.Contains(ip) {
			log.Warn().
				Str("ip", ip.String()).
				Str("cidr", cidr.String()).
				Msg("IP blocked: matches blocked CIDR range")
			return true, "IP is in blocked CIDR range: " + cidr.String()
		}
	}
	
	// Not blocked
	return false, ""
}

// GetStats returns blocklist statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]interface{}{
		"blocked_ips_count":   len(m.blockedIPs),
		"blocked_cidrs_count": len(m.blockedCIDRs),
	}
}
