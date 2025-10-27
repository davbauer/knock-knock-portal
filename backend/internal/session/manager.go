package session

import (
	"fmt"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Manager manages user sessions
type Manager struct {
	sessions          sync.Map // map[sessionID]*Session
	sessionsByIP      sync.Map // map[string]*sync.Map (IP -> map[sessionID]bool for O(1) operations)
	sessionsByUserID  sync.Map // map[string]*sync.Map (userID -> map[sessionID]bool for O(1) operations)
	defaultDuration   time.Duration
	maxDuration       *time.Duration
	autoExtendEnabled bool
	cleanupInterval   time.Duration
	cleanupTicker     *time.Ticker
	stopChan          chan struct{}
	maxSessions       int32 // Maximum allowed concurrent sessions (0 = unlimited)
	currentSessions   int32 // Current active session count
}

// NewManager creates a new session manager
// maxSessions: maximum allowed concurrent sessions (0 = unlimited)
func NewManager(defaultDuration time.Duration, maxDuration *time.Duration, autoExtend bool, cleanupInterval time.Duration, maxSessions int32) *Manager {
	m := &Manager{
		defaultDuration:   defaultDuration,
		maxDuration:       maxDuration,
		autoExtendEnabled: autoExtend,
		cleanupInterval:   cleanupInterval,
		stopChan:          make(chan struct{}),
		maxSessions:       maxSessions,
		currentSessions:   0,
	}

	// Start cleanup goroutine
	m.startCleanup()

	return m
}

// CreateSession creates a new session
func (m *Manager) CreateSession(userID, username string, clientIP netip.Addr, allowedServiceIDs []string) (*Session, error) {
	// Check session limit if configured (0 = unlimited)
	if m.maxSessions > 0 {
		current := atomic.LoadInt32(&m.currentSessions)
		if current >= m.maxSessions {
			return nil, fmt.Errorf("maximum sessions (%d) reached", m.maxSessions)
		}

		// Atomic increment with double-check
		newCount := atomic.AddInt32(&m.currentSessions, 1)
		if newCount > m.maxSessions {
			atomic.AddInt32(&m.currentSessions, -1)
			return nil, fmt.Errorf("maximum sessions (%d) reached", m.maxSessions)
		}
	}

	sessionID := uuid.New().String()
	now := time.Now()

	session := &Session{
		SessionID:                sessionID,
		UserID:                   userID,
		Username:                 username,
		AuthenticatedIPAddresses: []netip.Addr{clientIP}, // Start with initial IP
		AllowedServiceIDs:        allowedServiceIDs,
		CreatedAt:                now,
		LastActivityAt:           now,
		ExpiresAt:                now.Add(m.defaultDuration),
		AutoExtendEnabled:        m.autoExtendEnabled,
		MaximumDuration:          m.maxDuration,
	}

	// Store session
	m.sessions.Store(sessionID, session)

	// Index by IP
	m.addToIPIndex(clientIP.String(), sessionID)

	// Index by user ID
	m.addToUserIDIndex(userID, sessionID)

	log.Info().
		Str("session_id", sessionID).
		Str("user_id", userID).
		Str("username", username).
		Str("client_ip", clientIP.String()).
		Msg("Session created")

	return session, nil
}

// GetSessionByID retrieves a session by ID
func (m *Manager) GetSessionByID(sessionID string) (*Session, error) {
	value, ok := m.sessions.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	session := value.(*Session)
	if session.IsExpired() {
		m.TerminateSession(sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// GetSessionByIP retrieves active sessions for an IP address
func (m *Manager) GetSessionByIP(ip netip.Addr) (*Session, bool) {
	ipStr := ip.String()
	value, ok := m.sessionsByIP.Load(ipStr)
	if !ok {
		return nil, false
	}

	sessionMap := value.(*sync.Map)
	var foundSession *Session

	// Iterate through session IDs for this IP
	sessionMap.Range(func(key, _ interface{}) bool {
		sessionID := key.(string)
		if session, err := m.GetSessionByID(sessionID); err == nil {
			foundSession = session
			return false // Stop iteration
		}
		return true // Continue iteration
	})

	return foundSession, foundSession != nil
}

// RecordActivity records session activity and extends if configured
func (m *Manager) RecordActivity(sessionID string) error {
	value, ok := m.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found")
	}

	session := value.(*Session)
	if session.IsExpired() {
		return fmt.Errorf("session expired")
	}

	if session.AutoExtendEnabled && session.CanExtend() {
		session.ExtendSession(m.defaultDuration)
	} else {
		session.LastActivityAt = time.Now()
	}

	m.sessions.Store(sessionID, session)
	return nil
}

// AddIPToSession adds a new IP address to an existing session
func (m *Manager) AddIPToSession(sessionID string, clientIP netip.Addr) error {
	value, ok := m.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found")
	}

	session := value.(*Session)
	if session.IsExpired() {
		return fmt.Errorf("session expired")
	}

	// Add IP to session
	if session.AddAllowedIP(clientIP) {
		// Update session
		m.sessions.Store(sessionID, session)

		// Add to IP index
		m.addToIPIndex(clientIP.String(), sessionID)

		log.Info().
			Str("session_id", sessionID).
			Str("user_id", session.UserID).
			Str("new_ip", clientIP.String()).
			Msg("IP address added to session")

		return nil
	}

	return fmt.Errorf("IP already exists in session")
}

// TerminateSession terminates a session
func (m *Manager) TerminateSession(sessionID string) error {
	value, ok := m.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found")
	}

	session := value.(*Session)

	// Remove from all indices
	m.sessions.Delete(sessionID)

	// Decrement session counter if limit is configured
	if m.maxSessions > 0 {
		atomic.AddInt32(&m.currentSessions, -1)
	}

	// Remove from IP index for all authenticated IPs
	for _, ip := range session.AuthenticatedIPAddresses {
		m.removeFromIPIndex(ip.String(), sessionID)
	}

	m.removeFromUserIDIndex(session.UserID, sessionID)

	log.Info().
		Str("session_id", sessionID).
		Str("user_id", session.UserID).
		Str("username", session.Username).
		Msg("Session terminated")

	return nil
}

// GetAllActiveSessions returns all active sessions
func (m *Manager) GetAllActiveSessions() []*Session {
	sessions := []*Session{}

	m.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		if !session.IsExpired() {
			sessions = append(sessions, session)
		}
		return true
	})

	return sessions
}

// CleanupExpiredSessions removes all expired sessions
func (m *Manager) CleanupExpiredSessions() int {
	count := 0
	expiredSessionIDs := []string{}

	m.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		if session.IsExpired() {
			expiredSessionIDs = append(expiredSessionIDs, session.SessionID)
		}
		return true
	})

	for _, sessionID := range expiredSessionIDs {
		if err := m.TerminateSession(sessionID); err == nil {
			count++
		}
	}

	if count > 0 {
		log.Debug().Int("count", count).Msg("Cleaned up expired sessions")
	}

	return count
}

// startCleanup starts the cleanup goroutine
func (m *Manager) startCleanup() {
	m.cleanupTicker = time.NewTicker(m.cleanupInterval)
	go func() {
		for {
			select {
			case <-m.cleanupTicker.C:
				m.CleanupExpiredSessions()
			case <-m.stopChan:
				m.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// Close stops the session manager
func (m *Manager) Close() {
	close(m.stopChan)
}

// Helper methods for indexing

func (m *Manager) addToIPIndex(ip, sessionID string) {
	// Create or load the session map for this IP
	value, _ := m.sessionsByIP.LoadOrStore(ip, &sync.Map{})
	sessionMap := value.(*sync.Map)

	// Add session ID to the map (O(1) operation)
	sessionMap.Store(sessionID, true)
}

func (m *Manager) removeFromIPIndex(ip, sessionID string) {
	value, ok := m.sessionsByIP.Load(ip)
	if !ok {
		return
	}

	sessionMap := value.(*sync.Map)
	sessionMap.Delete(sessionID)

	// Atomic check-and-delete to prevent race condition
	hasEntries := false
	sessionMap.Range(func(_, _ interface{}) bool {
		hasEntries = true
		return false // Stop iteration, we found an entry
	})

	if !hasEntries {
		// Use LoadAndDelete for atomic operation to prevent TOCTOU race
		m.sessionsByIP.LoadAndDelete(ip)
	}
}

func (m *Manager) addToUserIDIndex(userID, sessionID string) {
	// Create or load the session map for this user
	value, _ := m.sessionsByUserID.LoadOrStore(userID, &sync.Map{})
	sessionMap := value.(*sync.Map)

	// Add session ID to the map (O(1) operation)
	sessionMap.Store(sessionID, true)
}

func (m *Manager) removeFromUserIDIndex(userID, sessionID string) {
	value, ok := m.sessionsByUserID.Load(userID)
	if !ok {
		return
	}

	sessionMap := value.(*sync.Map)
	sessionMap.Delete(sessionID)

	// Atomic check-and-delete to prevent race condition
	hasEntries := false
	sessionMap.Range(func(_, _ interface{}) bool {
		hasEntries = true
		return false // Stop iteration, we found an entry
	})

	if !hasEntries {
		// Use LoadAndDelete for atomic operation to prevent TOCTOU race
		m.sessionsByUserID.LoadAndDelete(userID)
	}
}
