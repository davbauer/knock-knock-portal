package proxy

import (
	"net/netip"
	"strings"
)

// parseIPFromAddr extracts an IP address from a network address string
func parseIPFromAddr(addr string) (netip.Addr, bool) {
	// Remove port if present (e.g., "127.0.0.1:12345" -> "127.0.0.1")
	host, _, err := strings.Cut(addr, ":")
	if err {
		host = addr
	}
	
	// Remove brackets from IPv6 addresses (e.g., "[::1]" -> "::1")
	host = strings.Trim(host, "[]")
	
	ip, parseErr := netip.ParseAddr(host)
	if parseErr != nil {
		return netip.Addr{}, false
	}
	
	return ip, true
}
