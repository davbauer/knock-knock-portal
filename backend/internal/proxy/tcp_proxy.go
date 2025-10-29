package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/ipblocklist"
	"github.com/rs/zerolog/log"
)

// tcpConnection tracks an active TCP connection
type tcpConnection struct {
	clientConn        net.Conn
	clientIP          string
	cancel            context.CancelFunc
	packetsFromClient int64 // Packets received from client
	packetsToClient   int64 // Packets sent to client
	bytesFromClient   int64 // Bytes received from client
	bytesToClient     int64 // Bytes sent to client
}

// TCPProxy handles TCP connection proxying with IP filtering
type TCPProxy struct {
	service          *config.ProtectedServiceConfig
	allowlistManager *ipallowlist.Manager
	blocklistManager *ipblocklist.Manager
	listener         net.Listener
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	activeConns      sync.WaitGroup
	connCount        int64
	activeConnCount  int32 // Current active connections
	maxConns         int32 // Maximum allowed concurrent connections
	circuitBreaker   *CircuitBreaker
	connections      map[string][]*tcpConnection // clientIP -> list of connections
	connectionsMu    sync.RWMutex
	mu               sync.Mutex
}

// NewTCPProxy creates a new TCP proxy
func NewTCPProxy(service *config.ProtectedServiceConfig, allowlistManager *ipallowlist.Manager, blocklistManager *ipblocklist.Manager, maxConnections int) *TCPProxy {
	ctx, cancel := context.WithCancel(context.Background())
	return &TCPProxy{
		service:          service,
		allowlistManager: allowlistManager,
		blocklistManager: blocklistManager,
		ctx:              ctx,
		cancel:           cancel,
		maxConns:         int32(maxConnections),
		circuitBreaker:   NewCircuitBreaker(service.ServiceName, 5, 30*time.Second, 3),
		connections:      make(map[string][]*tcpConnection),
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
		Str("service_id", p.service.ServiceID).
		Int("proxy_port", p.service.ProxyListenPortStart).
		Str("backend", fmt.Sprintf("%s:%d", p.service.BackendTargetHost, p.service.BackendTargetPort)).
		Msg("Starting TCP proxy listener")

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

		// Check connection limit
		currentConns := atomic.LoadInt32(&p.activeConnCount)
		if currentConns >= p.maxConns {
			log.Warn().
				Int32("current", currentConns).
				Int32("max", p.maxConns).
				Str("service", p.service.ServiceName).
				Msg("Maximum connections reached, rejecting new connection")
			conn.Close()
			continue
		}

		atomic.AddInt32(&p.activeConnCount, 1)
		p.activeConns.Add(1)
		go p.handleConnection(p.ctx, conn)
	}
}

// handleConnection handles a single TCP connection with context-aware cancellation
func (p *TCPProxy) handleConnection(ctx context.Context, clientConn net.Conn) {
	defer func() {
		p.activeConns.Done()
		atomic.AddInt32(&p.activeConnCount, -1)
	}()
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

	clientIPStr := clientIP.String()

	// HIGHEST PRIORITY: Check IP blocklist first
	if blocked, blockReason := p.blocklistManager.IsIPBlocked(net.ParseIP(clientIPStr)); blocked {
		log.Warn().
			Str("client_ip", clientIPStr).
			Str("service", p.service.ServiceName).
			Str("reason", blockReason).
			Msg("Connection denied: IP is blocked")
		return
	}

	// Check IP allowlist
	allowed, reason := p.allowlistManager.IsIPAllowed(clientIP)
	if !allowed {
		log.Warn().
			Str("client_ip", clientIPStr).
			Str("service", p.service.ServiceName).
			Str("reason", reason).
			Msg("Connection denied: IP not in allowlist")
		return
	}

	// Create connection-specific context for instant termination
	connCtx, connCancel := context.WithCancel(ctx)
	defer connCancel()

	// Track this connection
	conn := &tcpConnection{
		clientConn:        clientConn,
		clientIP:          clientIPStr,
		cancel:            connCancel,
		packetsFromClient: 0,
		packetsToClient:   0,
		bytesFromClient:   0,
		bytesToClient:     0,
	}

	p.connectionsMu.Lock()
	p.connections[clientIPStr] = append(p.connections[clientIPStr], conn)
	p.connectionsMu.Unlock()

	// Ensure cleanup when connection ends (safe removal)
	defer func() {
		p.connectionsMu.Lock()
		defer p.connectionsMu.Unlock()

		conns := p.connections[clientIPStr]
		if conns == nil {
			return
		}

		// Find and remove this specific connection
		for i := len(conns) - 1; i >= 0; i-- {
			if conns[i] == conn {
				// Safe removal by replacing with last element
				conns[i] = conns[len(conns)-1]
				conns = conns[:len(conns)-1]
				p.connections[clientIPStr] = conns
				break
			}
		}

		// Clean up empty map entry
		if len(p.connections[clientIPStr]) == 0 {
			delete(p.connections, clientIPStr)
		}
	}()

	// Check circuit breaker
	if !p.circuitBreaker.Allow() {
		log.Warn().
			Str("client_ip", clientIPStr).
			Str("service", p.service.ServiceName).
			Str("circuit_state", p.circuitBreaker.GetState().String()).
			Msg("Connection denied: circuit breaker is open")
		return
	}

	// Connect to backend
	backendAddr := net.JoinHostPort(p.service.BackendTargetHost, fmt.Sprintf("%d", p.service.BackendTargetPort))
	backendConn, err := net.DialTimeout("tcp", backendAddr, 10*time.Second)
	if err != nil {
		p.circuitBreaker.RecordFailure()
		log.Error().
			Err(err).
			Str("backend", backendAddr).
			Str("circuit_state", p.circuitBreaker.GetState().String()).
			Msg("Failed to connect to backend")
		return
	}
	defer backendConn.Close()

	// Record success
	p.circuitBreaker.RecordSuccess()

	log.Info().
		Str("client_ip", clientIPStr).
		Str("service", p.service.ServiceName).
		Str("backend", backendAddr).
		Msg("Proxying TCP connection")

	// Track connection count
	p.mu.Lock()
	p.connCount++
	connID := p.connCount
	p.mu.Unlock()

	// Bidirectional copy with context awareness and buffer pooling
	errChan := make(chan error, 2)
	done := make(chan struct{})

	// Get buffers from pool
	clientToBackendBuf := getTCPBuffer()
	backendToClientBuf := getTCPBuffer()
	defer putTCPBuffer(clientToBackendBuf)
	defer putTCPBuffer(backendToClientBuf)

	go func() {
		_, err := copyWithStats(backendConn, clientConn, *clientToBackendBuf, &conn.bytesFromClient, &conn.packetsFromClient)
		errChan <- err
	}()

	go func() {
		_, err := copyWithStats(clientConn, backendConn, *backendToClientBuf, &conn.bytesToClient, &conn.packetsToClient)
		errChan <- err
	}()

	// Wait for either direction to close or context cancellation
	go func() {
		<-errChan
		close(done)
	}()

	select {
	case <-done:
		// Normal completion - one direction finished
	case <-connCtx.Done():
		// Connection cancelled (instant termination via TerminateSessionsByIP)
		log.Info().
			Int64("conn_id", connID).
			Str("client_ip", clientIPStr).
			Str("service", p.service.ServiceName).
			Msg("TCP connection terminated instantly (session deleted)")
		return
	case <-ctx.Done():
		// Context cancelled - graceful shutdown
		log.Debug().
			Int64("conn_id", connID).
			Str("client_ip", clientIPStr).
			Str("service", p.service.ServiceName).
			Msg("TCP connection cancelled due to shutdown")
		return
	}

	log.Debug().
		Int64("conn_id", connID).
		Str("client_ip", clientIPStr).
		Str("service", p.service.ServiceName).
		Msg("TCP connection closed")
}

// TerminateSessionsByIP closes all TCP connections for a specific IP address
func (p *TCPProxy) TerminateSessionsByIP(clientIP string) int {
	p.connectionsMu.Lock()
	defer p.connectionsMu.Unlock()

	conns, exists := p.connections[clientIP]
	if !exists || len(conns) == 0 {
		return 0
	}

	terminated := 0

	// Terminate all connections for this IP
	for _, conn := range conns {
		if conn != nil {
			// Cancel context first
			if conn.cancel != nil {
				conn.cancel()
			}
			// Force close socket
			if conn.clientConn != nil {
				conn.clientConn.Close()
			}
			terminated++
		}
	}

	// Remove all connections for this IP from tracking
	delete(p.connections, clientIP)

	log.Debug().
		Str("client_ip", clientIP).
		Int("connections_terminated", terminated).
		Str("service", p.service.ServiceName).
		Msg("Terminated TCP connections for IP")

	return terminated
}

// GetStatsByIP returns statistics for a specific client IP
func (p *TCPProxy) GetStatsByIP(clientIP string) map[string]interface{} {
	stats := map[string]interface{}{
		"protocol":         "tcp",
		"packets_received": int64(0),
		"packets_sent":     int64(0),
		"bytes_received":   int64(0),
		"bytes_sent":       int64(0),
		"active_sessions":  0, // Renamed from "connections" for consistency
	}

	p.connectionsMu.RLock()
	defer p.connectionsMu.RUnlock()

	conns, exists := p.connections[clientIP]
	if !exists || len(conns) == 0 {
		return stats
	}

	var totalPacketsRx, totalPacketsTx, totalBytesRx, totalBytesTx int64
	for _, conn := range conns {
		totalPacketsRx += atomic.LoadInt64(&conn.packetsFromClient)
		totalPacketsTx += atomic.LoadInt64(&conn.packetsToClient)
		totalBytesRx += atomic.LoadInt64(&conn.bytesFromClient)
		totalBytesTx += atomic.LoadInt64(&conn.bytesToClient)
	}

	stats["active_sessions"] = len(conns)
	stats["packets_received"] = totalPacketsRx
	stats["packets_sent"] = totalPacketsTx
	stats["bytes_received"] = totalBytesRx
	stats["bytes_sent"] = totalBytesTx

	return stats
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

	// Collect unique client IPs from active connections
	clientIPs := make([]string, 0, len(p.connections))
	for clientIP := range p.connections {
		clientIPs = append(clientIPs, clientIP)
	}

	stats := map[string]interface{}{
		"total_connections":  p.connCount,
		"active_connections": atomic.LoadInt32(&p.activeConnCount),
		"client_ips":         clientIPs,
		"max_connections":    p.maxConns,
		"service_name":       p.service.ServiceName,
		"listen_port":        p.service.ProxyListenPortStart,
		"backend_addr":       fmt.Sprintf("%s:%d", p.service.BackendTargetHost, p.service.BackendTargetPort),
		"circuit_breaker":    p.circuitBreaker.GetStats(),
	}

	return stats
}

// TerminateConnectionsByIP forcefully closes all active connections from a specific IP
func (p *TCPProxy) TerminateConnectionsByIP(clientIP string) int {
	p.connectionsMu.Lock()
	defer p.connectionsMu.Unlock()

	conns, exists := p.connections[clientIP]
	if !exists || len(conns) == 0 {
		return 0
	}

	terminated := 0
	for _, conn := range conns {
		if conn.cancel != nil {
			conn.cancel() // Cancel context to stop proxy goroutines
		}
		if conn.clientConn != nil {
			conn.clientConn.Close() // Force close the connection
		}
		terminated++
	}

	// Remove from active connections map
	delete(p.connections, clientIP)

	log.Info().
		Str("service", p.service.ServiceName).
		Str("client_ip", clientIP).
		Int("terminated_count", terminated).
		Msg("Terminated TCP connections for IP")

	return terminated
}

// copyWithStats performs buffered copy while tracking bytes and packet counts
// This is optimized for performance and safety
func copyWithStats(dst io.Writer, src io.Reader, buf []byte, bytesCounter, packetsCounter *int64) (written int64, err error) {
	if buf == nil || len(buf) == 0 {
		return 0, fmt.Errorf("invalid buffer")
	}

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			// Count each successful read as a packet
			atomic.AddInt64(packetsCounter, 1)

			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = fmt.Errorf("invalid write result")
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	// Update total bytes transferred atomically
	atomic.AddInt64(bytesCounter, written)
	return written, err
}
