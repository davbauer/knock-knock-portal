package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/rs/zerolog/log"
)

// HTTPProxy handles HTTP reverse proxying with IP filtering
type HTTPProxy struct {
	service          *config.ProtectedServiceConfig
	allowlistManager *ipallowlist.Manager
	server           *http.Server
	proxy            *httputil.ReverseProxy
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	requestCount     int64
	mu               sync.Mutex
}

// NewHTTPProxy creates a new HTTP reverse proxy
func NewHTTPProxy(service *config.ProtectedServiceConfig, allowlistManager *ipallowlist.Manager) (*HTTPProxy, error) {
	backendURL, err := url.Parse(fmt.Sprintf("http://%s:%d", service.BackendTargetHost, service.BackendTargetPortStart))
	if err != nil {
		return nil, fmt.Errorf("invalid backend URL: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	hp := &HTTPProxy{
		service:          service,
		allowlistManager: allowlistManager,
		ctx:              ctx,
		cancel:           cancel,
		proxy:            httputil.NewSingleHostReverseProxy(backendURL),
	}
	
	// Customize the reverse proxy
	hp.proxy.ErrorHandler = hp.errorHandler
	hp.proxy.ModifyResponse = hp.modifyResponse
	
	return hp, nil
}

// Start begins the HTTP proxy server
func (p *HTTPProxy) Start() error {
	listenAddr := fmt.Sprintf(":%d", p.service.ProxyListenPortStart)
	
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleRequest)
	
	p.server = &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	log.Info().
		Str("service", p.service.ServiceName).
		Str("listen", listenAddr).
		Str("backend", fmt.Sprintf("http://%s:%d", p.service.BackendTargetHost, p.service.BackendTargetPortStart)).
		Msg("HTTP proxy started")
	
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP proxy server error")
		}
	}()
	
	return nil
}

// handleRequest processes incoming HTTP requests
func (p *HTTPProxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Extract client IP
	clientIP, ok := parseIPFromAddr(r.RemoteAddr)
	if !ok {
		log.Warn().Str("addr", r.RemoteAddr).Msg("Failed to parse client IP")
		http.Error(w, "Invalid client address", http.StatusBadRequest)
		return
	}
	
	// Check IP allowlist
	allowed, reason := p.allowlistManager.IsIPAllowed(clientIP)
	if !allowed {
		log.Warn().
			Str("client_ip", clientIP.String()).
			Str("service", p.service.ServiceName).
			Str("path", r.URL.Path).
			Str("reason", reason).
			Msg("HTTP request denied: IP not in allowlist")
		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}
	
	// Track request
	p.mu.Lock()
	p.requestCount++
	reqID := p.requestCount
	p.mu.Unlock()
	
	log.Info().
		Int64("req_id", reqID).
		Str("client_ip", clientIP.String()).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("service", p.service.ServiceName).
		Msg("Proxying HTTP request")
	
	// Proxy the request
	p.proxy.ServeHTTP(w, r)
}

// errorHandler handles reverse proxy errors
func (p *HTTPProxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Error().
		Err(err).
		Str("service", p.service.ServiceName).
		Str("path", r.URL.Path).
		Msg("HTTP proxy error")
	
	http.Error(w, "Bad Gateway", http.StatusBadGateway)
}

// modifyResponse allows modifying the backend response
func (p *HTTPProxy) modifyResponse(resp *http.Response) error {
	// Could add custom headers, logging, etc.
	return nil
}

// Stop gracefully shuts down the HTTP proxy
func (p *HTTPProxy) Stop() error {
	log.Info().
		Str("service", p.service.ServiceName).
		Msg("Stopping HTTP proxy")
	
	p.cancel()
	
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := p.server.Shutdown(ctx); err != nil {
			log.Warn().
				Err(err).
				Str("service", p.service.ServiceName).
				Msg("HTTP proxy shutdown error")
			return err
		}
	}
	
	p.wg.Wait()
	
	log.Info().
		Str("service", p.service.ServiceName).
		Msg("HTTP proxy stopped")
	
	return nil
}

// GetStats returns proxy statistics
func (p *HTTPProxy) GetStats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	return map[string]interface{}{
		"total_requests": p.requestCount,
		"service_name":   p.service.ServiceName,
		"listen_port":    p.service.ProxyListenPortStart,
		"backend_addr":   fmt.Sprintf("http://%s:%d", p.service.BackendTargetHost, p.service.BackendTargetPortStart),
	}
}
