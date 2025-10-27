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
	"github.com/rs/zerolog/log"
)

// UDPProxy handles UDP packet forwarding with IP filtering and session tracking
type UDPProxy struct {
	service          *config.ProtectedServiceConfig
	allowlistManager *ipallowlist.Manager
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
	spoofAttempts    int32 // Counter for spoof detection
	maxSpoofAttempts int32 // Maximum allowed spoof attempts before termination
	mu               sync.Mutex
}

// NewUDPProxy creates a new UDP proxy
func NewUDPProxy(service *config.ProtectedServiceConfig, allowlistManager *ipallowlist.Manager, sessionTimeout time.Duration, maxSessions int) *UDPProxy {
	ctx, cancel := context.WithCancel(context.Background())
	return &UDPProxy{
		service:          service,
		allowlistManager: allowlistManager,
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
	session.mu.Lock()
	conn := session.backendConn
	session.mu.Unlock()

	_, err := conn.Write(data)
	if err != nil {
		log.Error().
			Err(err).
			Str("client_addr", session.clientAddr.String()).
			Msg("Failed to forward UDP packet to backend")
	}
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
		_, err = p.conn.WriteToUDP(buffer[:n], session.clientAddr)
		if err != nil {
			log.Error().
				Err(err).
				Str("client_addr", session.clientAddr.String()).
				Msg("Failed to forward UDP packet to client")
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
