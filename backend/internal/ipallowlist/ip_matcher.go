package ipallowlist

import (
	"errors"
	"net/netip"
)

// Matcher provides IP matching functionality
type Matcher struct{}

// NewMatcher creates a new IP matcher
func NewMatcher() *Matcher {
	return &Matcher{}
}

// MatchesIP checks if an IP matches an exact IP or CIDR range
func (m *Matcher) MatchesIP(ip netip.Addr, entry *Entry) bool {
	// Exact IP match
	if entry.IPPrefix == nil {
		return ip == entry.IPAddress
	}

	// CIDR range match
	return entry.IPPrefix.Contains(ip)
}

// ParseIPOrPrefix parses an IP address or CIDR prefix
// Supports both single IPs (e.g., "127.0.0.1") and CIDR ranges (e.g., "192.168.1.0/24")
func ParseIPOrPrefix(s string) (netip.Addr, *netip.Prefix, error) {
	// Try parsing as single IP first (most common case for single hosts)
	if addr, err := netip.ParseAddr(s); err == nil {
		// Single IP - return without prefix
		return addr, nil, nil
	}

	// Try parsing as CIDR prefix
	if prefix, err := netip.ParsePrefix(s); err == nil {
		// Get the network address from the prefix
		addr := prefix.Masked().Addr()
		return addr, &prefix, nil
	}

	var zeroAddr netip.Addr
	return zeroAddr, nil, errors.New("invalid IP address or CIDR prefix: " + s)
}
