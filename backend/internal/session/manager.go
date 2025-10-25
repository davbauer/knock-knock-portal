package session

import (
	"fmt"
	"net/netip"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Manager manages user sessions
type Manager struct {
	sessions          sync.Map // map[sessionID]*Session
	sessionsByIP      sync.Map // map[string][]string (IP -> sessionIDs)
	sessionsByUserID  sync.Map // map[string][]string (userID -> sessionIDs)
	defaultDuration   time.Duration
	maxDuration       *time.Duration
	autoExtendEnabled bool
	cleanupInterval   time.Duration
	cleanupTicker     *time.Ticker
	stopChan          chan struct{}
}

// NewManager creates a new session manager
func NewManager(defaultDuration time.Duration, maxDuration *time.Duration, autoExtend bool, cleanupInterval time.Duration) *Manager {
	m := &Manager{
		defaultDuration:   defaultDuration,
		maxDuration:       maxDuration,
		autoExtendEnabled: autoExtend,
		cleanupInterval:   cleanupInterval,
		stopChan:          make(chan struct{}),
	}

	// Start cleanup goroutine
	m.startCleanup()

	return m
}

// CreateSession creates a new session
func (m *Manager) CreateSession(userID, username string, clientIP netip.Addr, allowedServiceIDs []string) (*Session, error) {
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
	value, ok := m.sessionsByIP.Load(ip.String())
	if !ok {
		return nil, false
	}

	sessionIDs := value.([]string)
	for _, sessionID := range sessionIDs {
		if session, err := m.GetSessionByID(sessionID); err == nil {
			return session, true
		}
	}

	return nil, false
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
	value, _ := m.sessionsByIP.LoadOrStore(ip, []string{})
	sessionIDs := value.([]string)
	sessionIDs = append(sessionIDs, sessionID)
	m.sessionsByIP.Store(ip, sessionIDs)
}

func (m *Manager) removeFromIPIndex(ip, sessionID string) {
	value, ok := m.sessionsByIP.Load(ip)
	if !ok {
		return
	}

	sessionIDs := value.([]string)
	newSessionIDs := []string{}
	for _, id := range sessionIDs {
		if id != sessionID {
			newSessionIDs = append(newSessionIDs, id)
		}
	}

	if len(newSessionIDs) > 0 {
		m.sessionsByIP.Store(ip, newSessionIDs)
	} else {
		m.sessionsByIP.Delete(ip)
	}
}

func (m *Manager) addToUserIDIndex(userID, sessionID string) {
	value, _ := m.sessionsByUserID.LoadOrStore(userID, []string{})
	sessionIDs := value.([]string)
	sessionIDs = append(sessionIDs, sessionID)
	m.sessionsByUserID.Store(userID, sessionIDs)
}

func (m *Manager) removeFromUserIDIndex(userID, sessionID string) {
	value, ok := m.sessionsByUserID.Load(userID)
	if !ok {
		return
	}

	sessionIDs := value.([]string)
	newSessionIDs := []string{}
	for _, id := range sessionIDs {
		if id != sessionID {
			newSessionIDs = append(newSessionIDs, id)
		}
	}

	if len(newSessionIDs) > 0 {
		m.sessionsByUserID.Store(userID, newSessionIDs)
	} else {
		m.sessionsByUserID.Delete(userID)
	}
}
