package middleware

import (
	"net/netip"
	"strings"
	"sync"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RealIPExtractor extracts the real client IP considering trusted proxies
type RealIPExtractor struct {
	mu                 sync.RWMutex
	enabled            bool
	trustedProxyRanges []netip.Prefix
	headerPriority     []string
}

// NewRealIPExtractor creates a new real IP extractor
func NewRealIPExtractor(cfg *config.TrustedProxyConfiguration) (*RealIPExtractor, error) {
	e := &RealIPExtractor{}
	e.Reload(cfg)
	return e, nil
}

// Reload updates the extractor configuration dynamically (thread-safe)
func (e *RealIPExtractor) Reload(cfg *config.TrustedProxyConfiguration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.enabled = cfg.Enabled
	e.headerPriority = cfg.ClientIPHeaderPriority
	e.trustedProxyRanges = nil // Clear old ranges

	if cfg.Enabled {
		for _, ipRange := range cfg.TrustedProxyIPRanges {
			if prefix, err := utils.ParseIPOrPrefixToPrefix(ipRange); err == nil {
				e.trustedProxyRanges = append(e.trustedProxyRanges, prefix)
			} else {
				log.Warn().
					Err(err).
					Str("ip_range", ipRange).
					Msg("Failed to parse trusted proxy IP range")
			}
		}
	}

	log.Info().
		Bool("enabled", e.enabled).
		Int("trusted_ranges_count", len(e.trustedProxyRanges)).
		Msg("Real IP extractor configuration reloaded")
}

// ExtractRealIP extracts the real client IP from the request
func (e *RealIPExtractor) ExtractRealIP(c *gin.Context) netip.Addr {
	e.mu.RLock()
	enabled := e.enabled
	headerPriority := e.headerPriority
	e.mu.RUnlock()

	// Get connection IP
	connIP := utils.ParseRemoteAddr(c.Request.RemoteAddr)

	// If trusted proxy is disabled, return connection IP
	if !enabled {
		return connIP
	}

	// Check if connection is from a trusted proxy
	if !e.isTrustedProxy(connIP) {
		// Check if request has proxy headers - only warn if they tried to use proxy headers
		hasProxyHeaders := e.hasProxyHeaders(c, headerPriority)

		// Only log warning if proxy headers are present (potential spoofing attempt)
		if enabled && hasProxyHeaders {
			log.Warn().
				Str("untrusted_proxy_ip", connIP.String()).
				Str("suggestion", "Add this IP to trusted_proxy_ip_ranges in config.yml").
				Str("config_example", "trusted_proxy_config:\n  enabled: true\n  trusted_proxy_ip_ranges:\n    - \""+connIP.String()+"\"").
				Msg("Request from untrusted proxy with X-Forwarded-For headers - ignoring headers to prevent IP spoofing")
		}
		return connIP
	}

	// Extract IP from headers in priority order
	for _, header := range headerPriority {
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

// hasProxyHeaders checks if the request has any proxy headers
func (e *RealIPExtractor) hasProxyHeaders(c *gin.Context, headerPriority []string) bool {
	for _, header := range headerPriority {
		if c.GetHeader(header) != "" {
			return true
		}
	}
	return false
}

// isTrustedProxy checks if an IP is in the trusted proxy ranges
func (e *RealIPExtractor) isTrustedProxy(ip netip.Addr) bool {
	if !ip.IsValid() {
		return false
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, prefix := range e.trustedProxyRanges {
		if prefix.Contains(ip) {
			return true
		}
	}

	return false
}

// GetProxyWarning returns a warning message if there's a proxy configuration issue
func (e *RealIPExtractor) GetProxyWarning(c *gin.Context) *string {
	// Get connection IP
	connIP := utils.ParseRemoteAddr(c.Request.RemoteAddr)
	if !connIP.IsValid() {
		return nil
	}

	// Check if proxy headers exist
	e.mu.RLock()
	headerPriority := e.headerPriority
	enabled := e.enabled
	e.mu.RUnlock()

	hasProxyHeaders := e.hasProxyHeaders(c, headerPriority)

	// Only warn if proxy headers are present
	if !hasProxyHeaders {
		return nil
	}

	// Verify extraction worked
	_, hasIP := GetClientIP(c)
	if !hasIP {
		return nil
	}

	if !enabled {
		// Proxy headers present but trusted proxy disabled
		warning := "Proxy detected (" + connIP.String() + ") but trusted proxy is DISABLED. Enable 'Reverse Proxy Security' in admin settings to use real client IPs."
		return &warning
	}

	// Check if proxy is trusted
	if !e.isTrustedProxy(connIP) {
		// Untrusted proxy with headers
		warning := "Untrusted proxy detected (" + connIP.String() + "). Add this IP to 'Trusted Proxy IP Ranges' in admin settings to trust it."
		return &warning
	}

	// Proxy is trusted and working correctly
	return nil
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
