package proxy

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int32

const (
	CircuitClosed   CircuitState = 0 // Normal operation
	CircuitOpen     CircuitState = 1 // Failing, rejecting requests
	CircuitHalfOpen CircuitState = 2 // Testing if service recovered
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	maxFailures      int32         // Max consecutive failures before opening
	failureCount     int32         // Current consecutive failure count
	successCount     int32         // Success count in half-open state
	state            int32         // Current state (CircuitState)
	lastFailureTime  time.Time     // When the last failure occurred
	lastStateChange  time.Time     // When state last changed
	timeout          time.Duration // How long to wait before half-open
	halfOpenAttempts int32         // Max attempts in half-open before closing
	mu               sync.RWMutex
	serviceName      string // For logging
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(serviceName string, maxFailures int32, timeout time.Duration, halfOpenAttempts int32) *CircuitBreaker {
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if halfOpenAttempts <= 0 {
		halfOpenAttempts = 3
	}

	return &CircuitBreaker{
		maxFailures:      maxFailures,
		failureCount:     0,
		successCount:     0,
		state:            int32(CircuitClosed),
		lastStateChange:  time.Now(),
		timeout:          timeout,
		halfOpenAttempts: halfOpenAttempts,
		serviceName:      serviceName,
	}
}

// Allow checks if a request is allowed
func (cb *CircuitBreaker) Allow() bool {
	state := CircuitState(atomic.LoadInt32(&cb.state))

	switch state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if timeout has elapsed
		cb.mu.RLock()
		elapsed := time.Since(cb.lastStateChange)
		cb.mu.RUnlock()

		if elapsed >= cb.timeout {
			// Try to transition to half-open
			if atomic.CompareAndSwapInt32(&cb.state, int32(CircuitOpen), int32(CircuitHalfOpen)) {
				cb.mu.Lock()
				cb.lastStateChange = time.Now()
				cb.successCount = 0
				cb.mu.Unlock()

				log.Info().
					Str("service", cb.serviceName).
					Str("previous_state", CircuitOpen.String()).
					Str("new_state", CircuitHalfOpen.String()).
					Msg("Circuit breaker transitioning to half-open")

				return true
			}
		}
		return false

	case CircuitHalfOpen:
		// Allow limited requests through
		return true

	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	state := CircuitState(atomic.LoadInt32(&cb.state))

	switch state {
	case CircuitClosed:
		// Reset failure count on success
		atomic.StoreInt32(&cb.failureCount, 0)

	case CircuitHalfOpen:
		// Increment success count
		successes := atomic.AddInt32(&cb.successCount, 1)

		// If enough successes, close the circuit
		if successes >= cb.halfOpenAttempts {
			if atomic.CompareAndSwapInt32(&cb.state, int32(CircuitHalfOpen), int32(CircuitClosed)) {
				cb.mu.Lock()
				cb.lastStateChange = time.Now()
				cb.failureCount = 0
				cb.successCount = 0
				cb.mu.Unlock()

				log.Info().
					Str("service", cb.serviceName).
					Str("previous_state", CircuitHalfOpen.String()).
					Str("new_state", CircuitClosed.String()).
					Int32("consecutive_successes", successes).
					Msg("Circuit breaker closed - service recovered")
			}
		}

	case CircuitOpen:
		// Shouldn't happen, but handle it
		log.Warn().
			Str("service", cb.serviceName).
			Msg("Recorded success while circuit is open - unexpected state")
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	state := CircuitState(atomic.LoadInt32(&cb.state))

	cb.mu.Lock()
	cb.lastFailureTime = time.Now()
	cb.mu.Unlock()

	switch state {
	case CircuitClosed:
		failures := atomic.AddInt32(&cb.failureCount, 1)

		// Open circuit if max failures reached
		if failures >= cb.maxFailures {
			if atomic.CompareAndSwapInt32(&cb.state, int32(CircuitClosed), int32(CircuitOpen)) {
				cb.mu.Lock()
				cb.lastStateChange = time.Now()
				cb.mu.Unlock()

				log.Error().
					Str("service", cb.serviceName).
					Str("previous_state", CircuitClosed.String()).
					Str("new_state", CircuitOpen.String()).
					Int32("consecutive_failures", failures).
					Dur("timeout", cb.timeout).
					Msg("Circuit breaker opened - too many failures")
			}
		}

	case CircuitHalfOpen:
		// Any failure in half-open state reopens the circuit
		if atomic.CompareAndSwapInt32(&cb.state, int32(CircuitHalfOpen), int32(CircuitOpen)) {
			cb.mu.Lock()
			cb.lastStateChange = time.Now()
			cb.failureCount = 1
			cb.successCount = 0
			cb.mu.Unlock()

			log.Warn().
				Str("service", cb.serviceName).
				Str("previous_state", CircuitHalfOpen.String()).
				Str("new_state", CircuitOpen.String()).
				Msg("Circuit breaker reopened - service still failing")
		}

	case CircuitOpen:
		// Already open, just increment counter
		atomic.AddInt32(&cb.failureCount, 1)
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitState {
	return CircuitState(atomic.LoadInt32(&cb.state))
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	state := CircuitState(atomic.LoadInt32(&cb.state))

	stats := map[string]interface{}{
		"state":            state.String(),
		"failure_count":    atomic.LoadInt32(&cb.failureCount),
		"max_failures":     cb.maxFailures,
		"timeout_seconds":  cb.timeout.Seconds(),
		"last_state_change": cb.lastStateChange,
	}

	if !cb.lastFailureTime.IsZero() {
		stats["last_failure_time"] = cb.lastFailureTime
		stats["time_since_failure_seconds"] = time.Since(cb.lastFailureTime).Seconds()
	}

	if state == CircuitHalfOpen {
		stats["success_count"] = atomic.LoadInt32(&cb.successCount)
		stats["half_open_attempts"] = cb.halfOpenAttempts
	}

	return stats
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := CircuitState(atomic.LoadInt32(&cb.state))
	atomic.StoreInt32(&cb.state, int32(CircuitClosed))
	atomic.StoreInt32(&cb.failureCount, 0)
	atomic.StoreInt32(&cb.successCount, 0)
	cb.lastStateChange = time.Now()

	log.Info().
		Str("service", cb.serviceName).
		Str("previous_state", oldState.String()).
		Str("new_state", CircuitClosed.String()).
		Msg("Circuit breaker manually reset")
}
