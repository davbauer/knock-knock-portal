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
func ParseIPOrPrefix(s string) (netip.Addr, *netip.Prefix, error) {
	// Try parsing as CIDR prefix first
	if prefix, err := netip.ParsePrefix(s); err == nil {
		// Get the network address
		addr := prefix.Masked().Addr()
		return addr, &prefix, nil
	}

	// Try parsing as single IP
	if addr, err := netip.ParseAddr(s); err == nil {
		return addr, nil, nil
	}

	var zeroAddr netip.Addr
	return zeroAddr, nil, errors.New("invalid IP address or prefix")
}
