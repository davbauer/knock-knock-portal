package middleware

import (
	"net"
	"net/netip"
	"strings"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RealIPExtractor extracts the real client IP considering trusted proxies
type RealIPExtractor struct {
	enabled              bool
	trustedProxyRanges   []netip.Prefix
	headerPriority       []string
}

// NewRealIPExtractor creates a new real IP extractor
func NewRealIPExtractor(cfg *config.TrustedProxyConfiguration) (*RealIPExtractor, error) {
	e := &RealIPExtractor{
		enabled:        cfg.Enabled,
		headerPriority: cfg.ClientIPHeaderPriority,
	}

	if cfg.Enabled {
		for _, ipRange := range cfg.TrustedProxyIPRanges {
			if prefix, err := netip.ParsePrefix(ipRange); err == nil {
				e.trustedProxyRanges = append(e.trustedProxyRanges, prefix)
			} else if addr, err := netip.ParseAddr(ipRange); err == nil {
				// Single IP - create /32 or /128 prefix
				bits := 32
				if addr.Is6() {
					bits = 128
				}
				prefix := netip.PrefixFrom(addr, bits)
				e.trustedProxyRanges = append(e.trustedProxyRanges, prefix)
			}
		}
	}

	return e, nil
}

// ExtractRealIP extracts the real client IP from the request
func (e *RealIPExtractor) ExtractRealIP(c *gin.Context) netip.Addr {
	// Get connection IP
	connIP := e.parseRemoteAddr(c.Request.RemoteAddr)

	// If trusted proxy is disabled, return connection IP
	if !e.enabled {
		return connIP
	}

	// Check if connection is from a trusted proxy
	if !e.isTrustedProxy(connIP) {
		// Check if request has proxy headers - only warn if they tried to use proxy headers
		hasProxyHeaders := false
		for _, header := range e.headerPriority {
			if c.GetHeader(header) != "" {
				hasProxyHeaders = true
				break
			}
		}
		
		// Only log warning if proxy headers are present (potential spoofing attempt)
		if e.enabled && hasProxyHeaders {
			log.Warn().
				Str("untrusted_proxy_ip", connIP.String()).
				Str("suggestion", "Add this IP to trusted_proxy_ip_ranges in config.yml").
				Str("config_example", "trusted_proxy_config:\n  enabled: true\n  trusted_proxy_ip_ranges:\n    - \""+connIP.String()+"\"").
				Msg("Request from untrusted proxy with X-Forwarded-For headers - ignoring headers to prevent IP spoofing")
		}
		return connIP
	}

	// Extract IP from headers in priority order
	for _, header := range e.headerPriority {
		value := c.GetHeader(header)
		if value == "" {
			continue
		}

		// Handle X-Forwarded-For (can contain multiple IPs)
		if header == "X-Forwarded-For" {
			// Take the first IP (original client)
			parts := strings.Split(value, ",")
			if len(parts) > 0 {
				value = strings.TrimSpace(parts[0])
			}
		}

		// Try parsing the IP
		if addr, err := netip.ParseAddr(value); err == nil {
			return addr
		}
	}

	// Fallback to connection IP
	return connIP
}

// parseRemoteAddr extracts IP from RemoteAddr (format: "ip:port")
func (e *RealIPExtractor) parseRemoteAddr(remoteAddr string) netip.Addr {
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

// isTrustedProxy checks if an IP is in the trusted proxy ranges
func (e *RealIPExtractor) isTrustedProxy(ip netip.Addr) bool {
	if !ip.IsValid() {
		return false
	}

	for _, prefix := range e.trustedProxyRanges {
		if prefix.Contains(ip) {
			return true
		}
	}

	return false
}

// Middleware returns a Gin middleware that extracts and stores the real IP
func (e *RealIPExtractor) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		realIP := e.ExtractRealIP(c)
		c.Set("client_ip", realIP)
		c.Next()
	}
}

// GetClientIP retrieves the stored client IP from context
func GetClientIP(c *gin.Context) (netip.Addr, bool) {
	if ip, exists := c.Get("client_ip"); exists {
		if addr, ok := ip.(netip.Addr); ok {
			return addr, true
		}
	}
	return netip.Addr{}, false
}
