package session

import (
	"net/netip"
	"time"
)

// Session represents an authenticated user session
type Session struct {
	SessionID         string
	UserID            string
	Username          string
	ClientIPAddress   netip.Addr
	AllowedServiceIDs []string // Empty = all services allowed
	CreatedAt         time.Time
	LastActivityAt    time.Time
	ExpiresAt         time.Time
	AutoExtendEnabled bool
	MaximumDuration   *time.Duration // nil = unlimited
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
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
