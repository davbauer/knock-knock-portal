package ipallowlist

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/rs/zerolog/log"
)

// Manager manages the IP allowlist
type Manager struct {
	exactIPEntries sync.Map // map[string]*Entry (IP string -> Entry)
	cidrEntries    []*Entry
	cidrMutex      sync.RWMutex
	matcher        *Matcher
	dnsResolver    *DNSResolver
	config         *config.NetworkAccessControlConfig
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewManager creates a new IP allowlist manager
func NewManager(cfg *config.NetworkAccessControlConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		matcher:     NewMatcher(),
		dnsResolver: NewDNSResolver(),
		config:      cfg,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Load permanent IP ranges
	m.loadPermanentIPRanges()

	// Start DNS refresh
	if len(cfg.AllowedDynamicDNSHostnames) > 0 {
		m.startDNSRefresh()
	}

	return m
}

// loadPermanentIPRanges loads permanently allowed IP ranges from config
func (m *Manager) loadPermanentIPRanges() {
	for _, ipRange := range m.config.PermanentlyAllowedIPRanges {
		addr, prefix, err := ParseIPOrPrefix(ipRange)
		if err != nil {
			log.Error().
				Err(err).
				Str("ip_range", ipRange).
				Msg("Failed to parse permanent IP range")
			continue
		}

		entry := &Entry{
			IPAddress:  addr,
			IPPrefix:   prefix,
			SourceType: EntryTypePermanent,
			AddedAt:    time.Now(),
		}

		if prefix == nil {
			// Exact IP - store in map
			m.exactIPEntries.Store(addr.String(), entry)
		} else {
			// CIDR range - store in slice
			m.cidrMutex.Lock()
			m.cidrEntries = append(m.cidrEntries, entry)
			m.cidrMutex.Unlock()
		}

		log.Info().
			Str("ip_range", ipRange).
			Msg("Added permanent IP allowlist entry")
	}
}

// startDNSRefresh starts periodic DNS resolution
func (m *Manager) startDNSRefresh() {
	interval := time.Duration(m.config.DNSRefreshIntervalSeconds) * time.Second

	m.dnsResolver.StartPeriodicRefresh(
		m.ctx,
		m.config.AllowedDynamicDNSHostnames,
		interval,
		func(results map[string][]netip.Addr) {
			m.updateDNSEntries(results)
		},
	)

	log.Info().
		Int("count", len(m.config.AllowedDynamicDNSHostnames)).
		Dur("interval", interval).
		Msg("Started DNS refresh")
}

// updateDNSEntries updates DNS-resolved IP entries
func (m *Manager) updateDNSEntries(results map[string][]netip.Addr) {
	now := time.Now()

	// Remove old DNS entries
	m.removeEntriesByType(EntryTypeDNSResolved)

	// Add new DNS entries
	for hostname, addrs := range results {
		for _, addr := range addrs {
			entry := &Entry{
				IPAddress:        addr,
				IPPrefix:         nil,
				SourceType:       EntryTypeDNSResolved,
				AddedAt:          now,
				LastVerifiedAt:   now,
				OriginalHostname: hostname,
			}

			m.exactIPEntries.Store(addr.String(), entry)
		}
	}

	log.Debug().
		Int("hostnames", len(results)).
		Msg("Updated DNS-resolved IP entries")
}

// AddSessionIP adds a session-based IP to the allowlist
func (m *Manager) AddSessionIP(sessionID string, ip netip.Addr, expiresAt time.Time) {
	entry := &Entry{
		IPAddress:  ip,
		IPPrefix:   nil,
		SourceType: EntryTypeSession,
		SessionID:  sessionID,
		AddedAt:    time.Now(),
		ExpiresAt:  &expiresAt,
	}

	m.exactIPEntries.Store(ip.String(), entry)

	log.Info().
		Str("session_id", sessionID).
		Str("ip", ip.String()).
		Time("expires_at", expiresAt).
		Msg("Added session IP to allowlist")
}

// RemoveSessionIP removes a session-based IP from the allowlist
func (m *Manager) RemoveSessionIP(sessionID string) {
	m.exactIPEntries.Range(func(key, value interface{}) bool {
		entry := value.(*Entry)
		if entry.SourceType == EntryTypeSession && entry.SessionID == sessionID {
			m.exactIPEntries.Delete(key)
			log.Debug().
				Str("session_id", sessionID).
				Str("ip", entry.IPAddress.String()).
				Msg("Removed session IP from allowlist")
		}
		return true
	})
}

// IsIPAllowed checks if an IP is allowed
func (m *Manager) IsIPAllowed(ip netip.Addr) (allowed bool, reason string) {
	// Fast path: exact IP lookup
	if value, ok := m.exactIPEntries.Load(ip.String()); ok {
		entry := value.(*Entry)
		if !entry.IsExpired() {
			return true, string(entry.SourceType)
		}
		// Remove expired entry
		m.exactIPEntries.Delete(ip.String())
	}

	// Slow path: CIDR range matching
	m.cidrMutex.RLock()
	defer m.cidrMutex.RUnlock()

	for _, entry := range m.cidrEntries {
		if !entry.IsExpired() && m.matcher.MatchesIP(ip, entry) {
			return true, string(entry.SourceType)
		}
	}

	return false, "not_allowed"
}

// IsIPAllowedForService checks if an IP is allowed for a specific service
func (m *Manager) IsIPAllowedForService(ip netip.Addr, serviceID string, allowedServiceIDs []string) (allowed bool, reason string) {
	// First check if IP is allowed at all
	ipAllowed, ipReason := m.IsIPAllowed(ip)
	if !ipAllowed {
		return false, ipReason
	}

	// If permanent or DNS-resolved, always allow
	if ipReason == string(EntryTypePermanent) || ipReason == string(EntryTypeDNSResolved) {
		return true, ipReason
	}

	// For session-based access, check service restrictions
	if len(allowedServiceIDs) == 0 {
		// Empty list = all services allowed
		return true, "session_all_services"
	}

	for _, allowedID := range allowedServiceIDs {
		if allowedID == serviceID {
			return true, "session_service_allowed"
		}
	}

	return false, "service_not_allowed"
}

// removeEntriesByType removes all entries of a specific type
func (m *Manager) removeEntriesByType(entryType EntryType) {
	// Remove from exact IPs
	m.exactIPEntries.Range(func(key, value interface{}) bool {
		entry := value.(*Entry)
		if entry.SourceType == entryType {
			m.exactIPEntries.Delete(key)
		}
		return true
	})

	// Remove from CIDR entries
	m.cidrMutex.Lock()
	defer m.cidrMutex.Unlock()

	newCIDREntries := []*Entry{}
	for _, entry := range m.cidrEntries {
		if entry.SourceType != entryType {
			newCIDREntries = append(newCIDREntries, entry)
		}
	}
	m.cidrEntries = newCIDREntries
}

// GetAllowlistStats returns statistics about the allowlist
func (m *Manager) GetAllowlistStats() map[string]interface{} {
	exactCount := 0
	m.exactIPEntries.Range(func(key, value interface{}) bool {
		exactCount++
		return true
	})

	m.cidrMutex.RLock()
	cidrCount := len(m.cidrEntries)
	m.cidrMutex.RUnlock()

	return map[string]interface{}{
		"exact_ip_count": exactCount,
		"cidr_count":     cidrCount,
		"total_count":    exactCount + cidrCount,
	}
}

// Close stops the allowlist manager
func (m *Manager) Close() {
	m.cancel()
}
