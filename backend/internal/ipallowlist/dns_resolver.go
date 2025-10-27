package ipallowlist

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/rs/zerolog/log"
)

// DNSResolver resolves DNS hostnames to IPs
type DNSResolver struct {
	resolver *net.Resolver
}

// NewDNSResolver creates a new DNS resolver
func NewDNSResolver() *DNSResolver {
	return &DNSResolver{
		resolver: &net.Resolver{},
	}
}

// ResolveHostname resolves a hostname to IP addresses (both IPv4 and IPv6)
// This handles CNAME chains automatically - the net.Resolver follows CNAMEs
// and returns the final resolved IPs (both A and AAAA records)
func (r *DNSResolver) ResolveHostname(ctx context.Context, hostname string) ([]netip.Addr, error) {
	// LookupIP follows CNAME chains automatically and returns all IPs
	ips, err := r.resolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return nil, err
	}

	addrs := []netip.Addr{}
	ipv4Count := 0
	ipv6Count := 0

	for _, ip := range ips {
		if addr, ok := netip.AddrFromSlice(ip); ok {
			addrs = append(addrs, addr)
			if addr.Is4() {
				ipv4Count++
			} else if addr.Is6() {
				ipv6Count++
			}
		}
	}

	log.Info().
		Str("hostname", hostname).
		Int("total_ips", len(addrs)).
		Int("ipv4_count", ipv4Count).
		Int("ipv6_count", ipv6Count).
		Msg("Resolved DNS hostname (including CNAME chain)")

	return addrs, nil
}

// ResolveHostnames resolves multiple hostnames
func (r *DNSResolver) ResolveHostnames(ctx context.Context, hostnames []string) map[string][]netip.Addr {
	results := make(map[string][]netip.Addr)

	for _, hostname := range hostnames {
		addrs, err := r.ResolveHostname(ctx, hostname)
		if err != nil {
			log.Warn().
				Err(err).
				Str("hostname", hostname).
				Msg("Failed to resolve DNS hostname")
			continue
		}
		results[hostname] = addrs
	}

	return results
}

// StartPeriodicRefresh starts periodic DNS resolution
func (r *DNSResolver) StartPeriodicRefresh(
	ctx context.Context,
	hostnames []string,
	interval time.Duration,
	callback func(map[string][]netip.Addr),
) {
	ticker := time.NewTicker(interval)
	go func() {
		// Initial refresh
		results := r.ResolveHostnames(ctx, hostnames)
		callback(results)

		for {
			select {
			case <-ticker.C:
				results := r.ResolveHostnames(ctx, hostnames)
				callback(results)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
