package ipallowlist

import (
	"net/netip"
	"time"
)

// EntryType represents the type of allowlist entry
type EntryType string

const (
	EntryTypePermanent   EntryType = "permanent"
	EntryTypeDNSResolved EntryType = "dns_resolved"
	EntryTypeSession     EntryType = "session"
)

// Entry represents an IP allowlist entry
type Entry struct {
	IPAddress        netip.Addr
	IPPrefix         *netip.Prefix // nil for exact IPs
	SourceType       EntryType
	SessionID        string // Only for session entries
	AddedAt          time.Time
	ExpiresAt        *time.Time // nil for permanent/DNS entries
	LastVerifiedAt   time.Time  // For DNS entries
	OriginalHostname string     // For DNS entries
}

// IsExpired checks if the entry is expired
func (e *Entry) IsExpired() bool {
	if e.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*e.ExpiresAt)
}
