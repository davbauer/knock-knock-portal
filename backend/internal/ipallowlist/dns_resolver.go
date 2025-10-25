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
func (r *DNSResolver) ResolveHostname(ctx context.Context, hostname string) ([]netip.Addr, error) {
	ips, err := r.resolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return nil, err
	}

	addrs := []netip.Addr{}
	for _, ip := range ips {
		if addr, ok := netip.AddrFromSlice(ip); ok {
			addrs = append(addrs, addr)
		}
	}

	log.Debug().
		Str("hostname", hostname).
		Int("count", len(addrs)).
		Msg("Resolved DNS hostname")

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
