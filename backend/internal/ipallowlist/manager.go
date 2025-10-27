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
	exactIPEntries sync.Map // map[string]*Entry (IP string -> Entry) - Permanent + Session IPs only
	dnsIPEntries   sync.Map // map[string]*Entry (IP string -> Entry) - DNS-resolved IPs only
	cidrEntries    []*Entry
	cidrMutex      sync.RWMutex
	matcher        *Matcher
	dnsResolver    *DNSResolver
	config         *config.NetworkAccessControlConfig
	configMutex    sync.RWMutex
	sessionIPIndex sync.Map // map[sessionID]string (sessionID -> IP) for O(1) removal
	ctx            context.Context
	cancel         context.CancelFunc
	dnsCancel      context.CancelFunc // Separate cancel for DNS refresh
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
	m.configMutex.RLock()
	cfg := m.config
	m.configMutex.RUnlock()

	for _, ipRange := range cfg.PermanentlyAllowedIPRanges {
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
	m.configMutex.RLock()
	cfg := m.config
	m.configMutex.RUnlock()

	interval := time.Duration(cfg.DNSRefreshIntervalSeconds) * time.Second

	// Create a separate context for DNS refresh so we can restart it
	dnsCtx, dnsCancel := context.WithCancel(m.ctx)
	m.dnsCancel = dnsCancel

	m.dnsResolver.StartPeriodicRefresh(
		dnsCtx,
		cfg.AllowedDynamicDNSHostnames,
		interval,
		func(results map[string][]netip.Addr) {
			m.updateDNSEntries(results)
		},
	)

	log.Info().
		Int("count", len(cfg.AllowedDynamicDNSHostnames)).
		Dur("interval", interval).
		Msg("Started DNS refresh")
}

// updateDNSEntries updates DNS-resolved IP entries
func (m *Manager) updateDNSEntries(results map[string][]netip.Addr) {
	now := time.Now()

	// O(1) operation: Clear the entire DNS map by replacing it
	// This is much faster than removeEntriesByType() which scans all entries
	m.dnsIPEntries = sync.Map{}

	// Add new DNS entries to dedicated DNS map
	totalIPs := 0
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

			m.dnsIPEntries.Store(addr.String(), entry)
			totalIPs++

			log.Info().
				Str("hostname", hostname).
				Str("ip", addr.String()).
				Bool("ipv4", addr.Is4()).
				Bool("ipv6", addr.Is6()).
				Msg("Added DNS-resolved IP to allowlist")
		}
	}

	log.Info().
		Int("hostnames", len(results)).
		Int("total_ips", totalIPs).
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

	ipStr := ip.String()
	m.exactIPEntries.Store(ipStr, entry)
	m.sessionIPIndex.Store(sessionID, ipStr) // Add to index for O(1) removal

	log.Info().
		Str("session_id", sessionID).
		Str("ip", ipStr).
		Time("expires_at", expiresAt).
		Msg("Added session IP to allowlist")
}

// RemoveSessionIP removes a session-based IP from the allowlist
func (m *Manager) RemoveSessionIP(sessionID string) {
	// O(1) lookup using index instead of O(n) iteration
	if ipValue, ok := m.sessionIPIndex.Load(sessionID); ok {
		ipStr := ipValue.(string)
		m.exactIPEntries.Delete(ipStr)
		m.sessionIPIndex.Delete(sessionID)
		log.Debug().
			Str("session_id", sessionID).
			Str("ip", ipStr).
			Msg("Removed session IP from allowlist")
		return
	}

	// Fallback: if not in index, search (shouldn't happen in normal operation)
	m.exactIPEntries.Range(func(key, value interface{}) bool {
		entry := value.(*Entry)
		if entry.SourceType == EntryTypeSession && entry.SessionID == sessionID {
			m.exactIPEntries.Delete(key)
			log.Debug().
				Str("session_id", sessionID).
				Str("ip", entry.IPAddress.String()).
				Msg("Removed session IP from allowlist (fallback path)")
		}
		return true
	})
}

// IsIPAllowed checks if an IP is allowed
func (m *Manager) IsIPAllowed(ip netip.Addr) (allowed bool, reason string) {
	ipStr := ip.String()

	// Fast path 1: Check DNS-resolved IPs
	if value, ok := m.dnsIPEntries.Load(ipStr); ok {
		entry := value.(*Entry)
		if !entry.IsExpired() {
			log.Debug().
				Str("ip", ipStr).
				Str("source_type", string(entry.SourceType)).
				Str("original_hostname", entry.OriginalHostname).
				Msg("IP allowed - DNS-resolved match")
			return true, string(entry.SourceType)
		}
		// DNS entries shouldn't expire, but handle it just in case
		go m.dnsIPEntries.Delete(ipStr)
	}

	// Fast path 2: Check permanent + session IPs
	if value, ok := m.exactIPEntries.Load(ipStr); ok {
		entry := value.(*Entry)
		if entry.IsExpired() {
			// Remove expired entry in background to avoid blocking
			go m.exactIPEntries.Delete(ipStr)
			log.Debug().
				Str("ip", ipStr).
				Msg("IP found in allowlist but entry is expired")
			// Continue to check CIDR ranges
		} else {
			log.Debug().
				Str("ip", ipStr).
				Str("source_type", string(entry.SourceType)).
				Msg("IP allowed - exact match")
			return true, string(entry.SourceType)
		}
	}

	// Slow path: CIDR range matching
	m.cidrMutex.RLock()
	defer m.cidrMutex.RUnlock()

	for _, entry := range m.cidrEntries {
		if !entry.IsExpired() && m.matcher.MatchesIP(ip, entry) {
			log.Debug().
				Str("ip", ipStr).
				Str("source_type", string(entry.SourceType)).
				Str("cidr", entry.IPPrefix.String()).
				Msg("IP allowed - CIDR match")
			return true, string(entry.SourceType)
		}
	}

	log.Debug().
		Str("ip", ipStr).
		Msg("IP not allowed - no match found")
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

	dnsCount := 0
	m.dnsIPEntries.Range(func(key, value interface{}) bool {
		dnsCount++
		return true
	})

	m.cidrMutex.RLock()
	cidrCount := len(m.cidrEntries)
	m.cidrMutex.RUnlock()

	return map[string]interface{}{
		"exact_ip_count": exactCount, // Permanent + Session IPs
		"dns_ip_count":   dnsCount,   // DNS-resolved IPs
		"cidr_count":     cidrCount,
		"total_count":    exactCount + dnsCount + cidrCount,
	}
}

// Close stops the allowlist manager
func (m *Manager) Close() {
	m.cancel()
}

// Reload updates the manager with new configuration (thread-safe, instant reload)
func (m *Manager) Reload(newCfg *config.NetworkAccessControlConfig) {
	m.configMutex.Lock()
	oldCfg := m.config
	m.config = newCfg
	m.configMutex.Unlock()

	log.Info().Msg("Reloading IP allowlist configuration...")

	// Step 1: Clear all permanent IP entries (both exact and CIDR)
	m.exactIPEntries.Range(func(key, value interface{}) bool {
		entry := value.(*Entry)
		if entry.SourceType == EntryTypePermanent {
			m.exactIPEntries.Delete(key)
		}
		return true
	})

	m.cidrMutex.Lock()
	newCIDREntries := []*Entry{}
	for _, entry := range m.cidrEntries {
		if entry.SourceType != EntryTypePermanent {
			newCIDREntries = append(newCIDREntries, entry)
		}
	}
	m.cidrEntries = newCIDREntries
	m.cidrMutex.Unlock()

	// Step 2: Load new permanent IP ranges
	m.loadPermanentIPRanges()

	// Step 3: Restart DNS refresh if hostnames changed
	oldHostnames := oldCfg.AllowedDynamicDNSHostnames
	newHostnames := newCfg.AllowedDynamicDNSHostnames

	hostnamesChanged := len(oldHostnames) != len(newHostnames)
	if !hostnamesChanged {
		hostnameMap := make(map[string]bool)
		for _, h := range oldHostnames {
			hostnameMap[h] = true
		}
		for _, h := range newHostnames {
			if !hostnameMap[h] {
				hostnamesChanged = true
				break
			}
		}
	}

	if hostnamesChanged {
		// Stop old DNS refresh
		if m.dnsCancel != nil {
			m.dnsCancel()
		}

		// Clear DNS entries
		m.dnsIPEntries = sync.Map{}

		// Start new DNS refresh
		if len(newHostnames) > 0 {
			m.startDNSRefresh()
		}

		log.Info().
			Int("old_count", len(oldHostnames)).
			Int("new_count", len(newHostnames)).
			Msg("DNS hostnames changed - restarted DNS refresh")
	}

	log.Info().
		Int("permanent_ip_ranges", len(newCfg.PermanentlyAllowedIPRanges)).
		Int("dns_hostnames", len(newCfg.AllowedDynamicDNSHostnames)).
		Msg("IP allowlist configuration reloaded successfully")
}
