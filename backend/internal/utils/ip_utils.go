package utils

import (
	"fmt"
	"net"
	"net/netip"
)

// ParseRemoteAddr extracts IP from RemoteAddr (format: "ip:port")
func ParseRemoteAddr(remoteAddr string) netip.Addr {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// Try parsing as-is
		if addr, err := netip.ParseAddr(remoteAddr); err == nil {
			return addr
		}
		return netip.Addr{}
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}

	return addr
}

// ParseIPOrPrefixToPrefix parses an IP address or CIDR range and returns a netip.Prefix
// For single IPs, it creates a /32 (IPv4) or /128 (IPv6) prefix
func ParseIPOrPrefixToPrefix(ipRange string) (netip.Prefix, error) {
	// Try parsing as CIDR first
	if prefix, err := netip.ParsePrefix(ipRange); err == nil {
		return prefix, nil
	}

	// Try parsing as single IP
	if addr, err := netip.ParseAddr(ipRange); err == nil {
		// Create /32 or /128 prefix for single IP
		bits := 32
		if addr.Is6() {
			bits = 128
		}
		return netip.PrefixFrom(addr, bits), nil
	}

	return netip.Prefix{}, fmt.Errorf("invalid IP address or CIDR: %s", ipRange)
}
