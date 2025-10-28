package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/ipblocklist"
	"github.com/rs/zerolog/log"
)

// UDPProxy handles UDP packet forwarding with IP filtering and session tracking
type UDPProxy struct {
	service          *config.ProtectedServiceConfig
	allowlistManager *ipallowlist.Manager
	blocklistManager *ipblocklist.Manager
	conn             *net.UDPConn
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	sessions         map[string]*udpSession
	sessionsMu       sync.RWMutex
	sessionTimeout   time.Duration
	maxSessions      int32 // Maximum allowed concurrent sessions
	packetCount      int64
	mu               sync.Mutex
}

// udpSession represents a pseudo-connection for UDP traffic
type udpSession struct {
	clientAddr       *net.UDPAddr
	backendConn      *net.UDPConn
	backendAddr      *net.UDPAddr // Expected backend address for validation
	lastActivity     time.Time
	spoofAttempts    int32  // Counter for spoof detection
	maxSpoofAttempts int32  // Maximum allowed spoof attempts before termination
	packetsReceived  int64  // Total packets received from client
	packetsSent      int64  // Total packets sent to client
	bytesReceived    int64  // Total bytes received from client
	bytesSent        int64  // Total bytes sent to client
	mu               sync.Mutex
}

// NewUDPProxy creates a new UDP proxy
func NewUDPProxy(service *config.ProtectedServiceConfig, allowlistManager *ipallowlist.Manager, blocklistManager *ipblocklist.Manager, sessionTimeout time.Duration, maxSessions int) *UDPProxy {
	ctx, cancel := context.WithCancel(context.Background())
	return &UDPProxy{
		service:          service,
		allowlistManager: allowlistManager,
		blocklistManager: blocklistManager,
		ctx:              ctx,
		cancel:           cancel,
		sessions:         make(map[string]*udpSession),
		sessionTimeout:   sessionTimeout,
		maxSessions:      int32(maxSessions),
	}
}

// Start begins listening and forwarding UDP packets
func (p *UDPProxy) Start() error {
	listenAddr := fmt.Sprintf(":%d", p.service.ProxyListenPortStart)

	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address %s: %w", listenAddr, err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start UDP listener on %s: %w", listenAddr, err)
	}

	p.conn = conn

	log.Info().
		Str("service_id", p.service.ServiceID).
		Int("proxy_port", p.service.ProxyListenPortStart).
		Str("backend", fmt.Sprintf("%s:%d", p.service.BackendTargetHost, p.service.BackendTargetPort)).
		Msg("Starting UDP proxy listener")

	p.wg.Add(2)
	go p.receiveLoop()
	go p.cleanupLoop()

	return nil
}

// receiveLoop receives packets from clients
func (p *UDPProxy) receiveLoop() {
	defer p.wg.Done()

	bufPtr := getUDPBuffer()
	defer putUDPBuffer(bufPtr)
	buffer := *bufPtr

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		// Set read deadline to allow context cancellation
		p.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, clientAddr, err := p.conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-p.ctx.Done():
				return
			default:
				log.Error().Err(err).Msg("Failed to read UDP packet")
				continue
			}
		}

		// Extract client IP
		clientIP, ok := parseIPFromAddr(clientAddr.IP.String())
		if !ok {
			log.Warn().
				Str("addr", clientAddr.IP.String()).
				Msg("Failed to parse client IP")
			continue
		}

		// HIGHEST PRIORITY: Check IP blocklist first
		if blocked, blockReason := p.blocklistManager.IsIPBlocked(net.ParseIP(clientIP.String())); blocked {
			log.Warn().
				Str("client_ip", clientIP.String()).
				Str("service", p.service.ServiceName).
				Str("reason", blockReason).
				Msg("UDP packet denied: IP is blocked")
			continue
		}

		// Check IP allowlist
		allowed, reason := p.allowlistManager.IsIPAllowed(clientIP)
		if !allowed {
			log.Warn().
				Str("client_ip", clientIP.String()).
				Str("service", p.service.ServiceName).
				Str("reason", reason).
				Msg("UDP packet denied: IP not in allowlist")
			continue
		}

		// Track packet
		p.mu.Lock()
		p.packetCount++
		p.mu.Unlock()

		// Get or create session
		session, err := p.getOrCreateSession(clientAddr)
		if err != nil {
			log.Warn().
				Err(err).
				Str("client_addr", clientAddr.String()).
				Str("service", p.service.ServiceName).
				Msg("Failed to create UDP session (may have hit session limit)")
			continue
		}

		// Forward packet to backend
		// CRITICAL: Copy the data to avoid race condition since buffer is reused
		packetData := make([]byte, n)
		copy(packetData, buffer[:n])
		go p.forwardToBackend(session, packetData)
	}
}

// getOrCreateSession retrieves or creates a UDP session
func (p *UDPProxy) getOrCreateSession(clientAddr *net.UDPAddr) (*udpSession, error) {
	sessionKey := clientAddr.String()

	p.sessionsMu.RLock()
	session, exists := p.sessions[sessionKey]
	p.sessionsMu.RUnlock()

	if exists {
		session.mu.Lock()
		session.lastActivity = time.Now()
		session.mu.Unlock()
		return session, nil
	}

	// Check session limit before creating new session
	p.sessionsMu.RLock()
	currentSessions := int32(len(p.sessions))
	p.sessionsMu.RUnlock()

	if currentSessions >= p.maxSessions {
		return nil, fmt.Errorf("maximum sessions (%d) reached", p.maxSessions)
	}

	// Create new session
	backendAddr := fmt.Sprintf("%s:%d", p.service.BackendTargetHost, p.service.BackendTargetPort)
	backendUDPAddr, err := net.ResolveUDPAddr("udp", backendAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve backend address: %w", err)
	}

	backendConn, err := net.DialUDP("udp", nil, backendUDPAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to backend: %w", err)
	}

	session = &udpSession{
		clientAddr:       clientAddr,
		backendConn:      backendConn,
		backendAddr:      backendUDPAddr,
		lastActivity:     time.Now(),
		spoofAttempts:    0,
		maxSpoofAttempts: 5, // Terminate after 5 spoof attempts
	}

	p.sessionsMu.Lock()
	p.sessions[sessionKey] = session
	p.sessionsMu.Unlock()

	log.Debug().
		Str("client_addr", clientAddr.String()).
		Str("backend_addr", backendAddr).
		Str("service", p.service.ServiceName).
		Int("active_sessions", int(currentSessions)+1).
		Msg("Created UDP session")

	// Start receiving responses from backend with context
	go p.receiveFromBackend(p.ctx, session)

	return session, nil
}

// forwardToBackend sends a packet to the backend
func (p *UDPProxy) forwardToBackend(session *udpSession, data []byte) {
	if session == nil || len(data) == 0 {
		return
	}

	session.mu.Lock()
	conn := session.backendConn
	session.mu.Unlock()

	if conn == nil {
		log.Error().
			Str("client_addr", session.clientAddr.String()).
			Msg("Backend connection is nil, cannot forward packet")
		return
	}

	n, err := conn.Write(data)
	if err != nil {
		log.Error().
			Err(err).
			Str("client_addr", session.clientAddr.String()).
			Msg("Failed to forward UDP packet to backend")
		return
	}

	// Track stats atomically
	atomic.AddInt64(&session.packetsReceived, 1)
	atomic.AddInt64(&session.bytesReceived, int64(n))
}

// receiveFromBackend receives responses from the backend and forwards to client
func (p *UDPProxy) receiveFromBackend(ctx context.Context, session *udpSession) {
	bufPtr := getUDPBuffer()
	defer putUDPBuffer(bufPtr)
	buffer := *bufPtr

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		session.mu.Lock()
		conn := session.backendConn
		expectedBackend := session.backendAddr
		session.mu.Unlock()

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(p.sessionTimeout))

		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Session timed out, will be cleaned up
				return
			}
			log.Error().
				Err(err).
				Str("client_addr", session.clientAddr.String()).
				Msg("Failed to read from backend")
			return
		}

		// SECURITY: Validate response is from expected backend to prevent amplification attacks
		if addr.String() != expectedBackend.String() {
			// Increment spoof counter atomically
			attempts := atomic.AddInt32(&session.spoofAttempts, 1)

			log.Warn().
				Str("expected_backend", expectedBackend.String()).
				Str("actual_source", addr.String()).
				Str("client_addr", session.clientAddr.String()).
				Str("service", p.service.ServiceName).
				Int32("spoof_attempts", attempts).
				Msg("UDP response from unexpected source - possible spoofing/amplification attack attempt")

			// Terminate session after max spoof attempts to prevent amplification
			if attempts >= session.maxSpoofAttempts {
				log.Error().
					Str("client_addr", session.clientAddr.String()).
					Str("service", p.service.ServiceName).
					Int32("spoof_attempts", attempts).
					Msg("Maximum spoof attempts reached - terminating UDP session for security")

				// Close backend connection to terminate the session
				session.mu.Lock()
				session.backendConn.Close()
				session.mu.Unlock()

				// Remove from sessions map
				p.sessionsMu.Lock()
				delete(p.sessions, session.clientAddr.String())
				p.sessionsMu.Unlock()

				return
			}

			continue // Drop the packet
		}

		// Update activity time
		session.mu.Lock()
		session.lastActivity = time.Now()
		session.mu.Unlock()

		// Forward response to client
		written, err := p.conn.WriteToUDP(buffer[:n], session.clientAddr)
		if err != nil {
			log.Error().
				Err(err).
				Str("client_addr", session.clientAddr.String()).
				Msg("Failed to forward UDP packet to client")
		} else {
			// Track stats
			atomic.AddInt64(&session.packetsSent, 1)
			atomic.AddInt64(&session.bytesSent, int64(written))
		}
	}
}

// cleanupLoop periodically removes expired sessions
func (p *UDPProxy) cleanupLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.cleanupExpiredSessions()
		}
	}
}

// cleanupExpiredSessions removes sessions that have timed out
func (p *UDPProxy) cleanupExpiredSessions() {
	now := time.Now()
	expired := make([]string, 0)

	p.sessionsMu.RLock()
	for key, session := range p.sessions {
		session.mu.Lock()
		if now.Sub(session.lastActivity) > p.sessionTimeout {
			expired = append(expired, key)
		}
		session.mu.Unlock()
	}
	p.sessionsMu.RUnlock()

	if len(expired) == 0 {
		return
	}

	p.sessionsMu.Lock()
	for _, key := range expired {
		if session, exists := p.sessions[key]; exists {
			session.backendConn.Close()
			delete(p.sessions, key)
			log.Debug().
				Str("client_addr", key).
				Str("service", p.service.ServiceName).
				Msg("Cleaned up expired UDP session")
		}
	}
	p.sessionsMu.Unlock()

	log.Debug().
	Int("expired_count", len(expired)).
	Str("service", p.service.ServiceName).
	Msg("UDP session cleanup completed")
}

// TerminateSessionsByIP closes all UDP sessions for a specific IP address
func (p *UDPProxy) TerminateSessionsByIP(clientIP string) int {
	p.sessionsMu.Lock()
	defer p.sessionsMu.Unlock()

	terminated := 0

	// Safely iterate and terminate matching sessions
	for key, session := range p.sessions {
		if session != nil && session.clientAddr.IP.String() == clientIP {
			// Close backend connection
			if session.backendConn != nil {
				session.backendConn.Close()
			}
			// Remove from map
			delete(p.sessions, key)
			terminated++

			log.Debug().
				Str("client_ip", clientIP).
				Str("session_key", key).
				Str("service", p.service.ServiceName).
				Msg("Terminated UDP session for IP")
		}
	}

	return terminated
}

// GetStatsByIP returns statistics for a specific client IP
func (p *UDPProxy) GetStatsByIP(clientIP string) map[string]interface{} {
	stats := map[string]interface{}{
		"protocol":         "udp",
		"packets_received": int64(0),
		"packets_sent":     int64(0),
		"bytes_received":   int64(0),
		"bytes_sent":       int64(0),
		"active_sessions":  0, // Renamed from "sessions" for consistency
	}

	p.sessionsMu.RLock()
	defer p.sessionsMu.RUnlock()

	sessionCount := 0
	var totalPacketsRx, totalPacketsTx, totalBytesRx, totalBytesTx int64

	for _, session := range p.sessions {
		if session.clientAddr.IP.String() == clientIP {
			sessionCount++
			totalPacketsRx += atomic.LoadInt64(&session.packetsReceived)
			totalPacketsTx += atomic.LoadInt64(&session.packetsSent)
			totalBytesRx += atomic.LoadInt64(&session.bytesReceived)
			totalBytesTx += atomic.LoadInt64(&session.bytesSent)
		}
	}

	stats["active_sessions"] = sessionCount
	stats["packets_received"] = totalPacketsRx
	stats["packets_sent"] = totalPacketsTx
	stats["bytes_received"] = totalBytesRx
	stats["bytes_sent"] = totalBytesTx

	return stats
}

// Stop gracefully shuts down the proxy
func (p *UDPProxy) Stop() error {
	log.Info().
		Str("service", p.service.ServiceName).
		Msg("Stopping UDP proxy")

	p.cancel()

	if p.conn != nil {
		p.conn.Close()
	}

	// Close all backend connections
	p.sessionsMu.Lock()
	for _, session := range p.sessions {
		session.backendConn.Close()
	}
	p.sessions = make(map[string]*udpSession)
	p.sessionsMu.Unlock()

	p.wg.Wait()

	log.Info().
		Str("service", p.service.ServiceName).
		Msg("UDP proxy stopped")

	return nil
}

// GetStats returns proxy statistics
func (p *UDPProxy) GetStats() map[string]interface{} {
	p.mu.Lock()
	packetCount := p.packetCount
	p.mu.Unlock()

	p.sessionsMu.RLock()
	sessionCount := len(p.sessions)
	clientIPs := make([]string, 0, sessionCount)
	for sessionKey := range p.sessions {
		clientIPs = append(clientIPs, sessionKey)
	}
	p.sessionsMu.RUnlock()

	return map[string]interface{}{
		"total_packets":   packetCount,
		"active_sessions": sessionCount,
		"client_ips":      clientIPs,
		"max_sessions":    p.maxSessions,
		"service_name":    p.service.ServiceName,
		"listen_port":     p.service.ProxyListenPortStart,
		"backend_addr":    fmt.Sprintf("%s:%d", p.service.BackendTargetHost, p.service.BackendTargetPort),
		"session_timeout": p.sessionTimeout.String(),
	}
}

// TerminateConnectionsByIP forcefully closes all active UDP sessions from a specific IP
func (p *UDPProxy) TerminateConnectionsByIP(clientIP string) int {
	p.sessionsMu.Lock()
	defer p.sessionsMu.Unlock()
	
	terminated := 0
	
	// Find and close all sessions for this IP
	for sessionKey, session := range p.sessions {
		if session.clientAddr.IP.String() == clientIP {
			// Close the backend connection
			if session.backendConn != nil {
				session.backendConn.Close()
			}
			// Remove from sessions map
			delete(p.sessions, sessionKey)
			terminated++
		}
	}
	
	if terminated > 0 {
		log.Info().
			Str("service", p.service.ServiceName).
			Str("client_ip", clientIP).
			Int("terminated_count", terminated).
			Msg("Terminated UDP sessions for IP")
	}
	
	return terminated
}
