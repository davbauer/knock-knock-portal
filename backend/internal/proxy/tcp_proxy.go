package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/rs/zerolog/log"
)

// TCPProxy handles TCP connection proxying with IP filtering
type TCPProxy struct {
	service          *config.ProtectedServiceConfig
	allowlistManager *ipallowlist.Manager
	listener         net.Listener
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	activeConns      sync.WaitGroup
	connCount        int64
	mu               sync.Mutex
}

// NewTCPProxy creates a new TCP proxy
func NewTCPProxy(service *config.ProtectedServiceConfig, allowlistManager *ipallowlist.Manager) *TCPProxy {
	ctx, cancel := context.WithCancel(context.Background())
	return &TCPProxy{
		service:          service,
		allowlistManager: allowlistManager,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start begins listening and proxying connections
func (p *TCPProxy) Start() error {
	listenAddr := fmt.Sprintf(":%d", p.service.ProxyListenPortStart)
	
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to start TCP listener on %s: %w", listenAddr, err)
	}
	
	p.listener = listener
	
	log.Info().
		Str("service", p.service.ServiceName).
		Str("listen", listenAddr).
		Str("backend", fmt.Sprintf("%s:%d", p.service.BackendTargetHost, p.service.BackendTargetPortStart)).
		Msg("TCP proxy started")
	
	p.wg.Add(1)
	go p.acceptLoop()
	
	return nil
}

// acceptLoop accepts incoming connections
func (p *TCPProxy) acceptLoop() {
	defer p.wg.Done()
	
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.ctx.Done():
				return
			default:
				log.Error().Err(err).Msg("Failed to accept connection")
				continue
			}
		}
		
		p.activeConns.Add(1)
		go p.handleConnection(conn)
	}
}

// handleConnection handles a single TCP connection
func (p *TCPProxy) handleConnection(clientConn net.Conn) {
	defer p.activeConns.Done()
	defer clientConn.Close()
	
	// Extract client IP
	clientAddr := clientConn.RemoteAddr().(*net.TCPAddr)
	clientIP, ok := parseIPFromAddr(clientAddr.IP.String())
	if !ok {
		log.Warn().
			Str("addr", clientAddr.IP.String()).
			Msg("Failed to parse client IP")
		return
	}
	
	// Check IP allowlist
	allowed, reason := p.allowlistManager.IsIPAllowed(clientIP)
	if !allowed {
		log.Warn().
			Str("client_ip", clientIP.String()).
			Str("service", p.service.ServiceName).
			Str("reason", reason).
			Msg("Connection denied: IP not in allowlist")
		return
	}
	
	// Connect to backend
	backendAddr := fmt.Sprintf("%s:%d", p.service.BackendTargetHost, p.service.BackendTargetPortStart)
	backendConn, err := net.DialTimeout("tcp", backendAddr, 10*time.Second)
	if err != nil {
		log.Error().
			Err(err).
			Str("backend", backendAddr).
			Msg("Failed to connect to backend")
		return
	}
	defer backendConn.Close()
	
	log.Info().
		Str("client_ip", clientIP.String()).
		Str("service", p.service.ServiceName).
		Str("backend", backendAddr).
		Msg("Proxying TCP connection")
	
	// Track connection count
	p.mu.Lock()
	p.connCount++
	connID := p.connCount
	p.mu.Unlock()
	
	// Bidirectional copy
	errChan := make(chan error, 2)
	
	go func() {
		_, err := io.Copy(backendConn, clientConn)
		errChan <- err
	}()
	
	go func() {
		_, err := io.Copy(clientConn, backendConn)
		errChan <- err
	}()
	
	// Wait for either direction to close
	<-errChan
	
	log.Debug().
		Int64("conn_id", connID).
		Str("client_ip", clientIP.String()).
		Str("service", p.service.ServiceName).
		Msg("TCP connection closed")
}

// Stop gracefully shuts down the proxy
func (p *TCPProxy) Stop() error {
	log.Info().
		Str("service", p.service.ServiceName).
		Msg("Stopping TCP proxy")
	
	p.cancel()
	
	if p.listener != nil {
		p.listener.Close()
	}
	
	// Wait for accept loop to stop
	p.wg.Wait()
	
	// Wait for all active connections to finish (with timeout)
	done := make(chan struct{})
	go func() {
		p.activeConns.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Info().
			Str("service", p.service.ServiceName).
			Msg("TCP proxy stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Warn().
			Str("service", p.service.ServiceName).
			Msg("TCP proxy stopped with timeout (some connections may have been terminated)")
	}
	
	return nil
}

// GetStats returns proxy statistics
func (p *TCPProxy) GetStats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	return map[string]interface{}{
		"total_connections": p.connCount,
		"service_name":      p.service.ServiceName,
		"listen_port":       p.service.ProxyListenPortStart,
		"backend_addr":      fmt.Sprintf("%s:%d", p.service.BackendTargetHost, p.service.BackendTargetPortStart),
	}
}
