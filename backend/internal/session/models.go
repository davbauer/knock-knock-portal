package session

import (
	"net/netip"
	"time"
)

// Session represents an authenticated user session
type Session struct {
	SessionID                string
	UserID                   string
	Username                 string
	AuthenticatedIPAddresses []netip.Addr // Multiple IPs authenticated for the same session
	AllowedServiceIDs        []string     // Empty = all services allowed
	CreatedAt                time.Time
	LastActivityAt           time.Time
	ExpiresAt                time.Time
	AutoExtendEnabled        bool
	MaximumDuration          *time.Duration // nil = unlimited
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsIPAllowed checks if an IP is in the authenticated list
func (s *Session) IsIPAllowed(ip netip.Addr) bool {
	for _, authenticatedIP := range s.AuthenticatedIPAddresses {
		if authenticatedIP == ip {
			return true
		}
	}
	return false
}

// AddAllowedIP adds an IP to the authenticated list if not already present
// Returns true if the IP was added, false if already present
func (s *Session) AddAllowedIP(ip netip.Addr) bool {
	if s.IsIPAllowed(ip) {
		return false
	}
	s.AuthenticatedIPAddresses = append(s.AuthenticatedIPAddresses, ip)
	return true
}

// CanExtend checks if the session can be extended
func (s *Session) CanExtend() bool {
	if !s.AutoExtendEnabled {
		return false
	}
	if s.MaximumDuration == nil {
		return true
	}
	maxExpiry := s.CreatedAt.Add(*s.MaximumDuration)
	return time.Now().Before(maxExpiry)
}

// ExtendSession extends the session expiration time
func (s *Session) ExtendSession(duration time.Duration) {
	newExpiry := time.Now().Add(duration)

	// Check maximum duration limit
	if s.MaximumDuration != nil {
		maxExpiry := s.CreatedAt.Add(*s.MaximumDuration)
		if newExpiry.After(maxExpiry) {
			newExpiry = maxExpiry
		}
	}

	s.ExpiresAt = newExpiry
	s.LastActivityAt = time.Now()
}
