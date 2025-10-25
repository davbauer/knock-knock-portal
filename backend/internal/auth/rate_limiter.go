package auth

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter provides IP-based rate limiting
type RateLimiter struct {
	limiters   map[string]*limiterEntry
	mu         sync.Mutex
	rate       rate.Limit
	burst      int
	maxIdleAge time.Duration
}

// limiterEntry tracks a rate limiter and its metadata
type limiterEntry struct {
	limiter      *rate.Limiter
	lastAccessed time.Time
	failCount    int // Track consecutive failures for exponential backoff
}

// NewRateLimiter creates a new rate limiter
// rate: requests per second
// burst: maximum burst size
func NewRateLimiter(requestsPerMinute int, burst int) *RateLimiter {
	return &RateLimiter{
		limiters:   make(map[string]*limiterEntry),
		rate:       rate.Limit(float64(requestsPerMinute) / 60.0), // Convert to per-second
		burst:      burst,
		maxIdleAge: 15 * time.Minute,
	}
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	entry, exists := rl.limiters[ip]
	if !exists {
		entry = &limiterEntry{
			limiter:      rate.NewLimiter(rl.rate, rl.burst),
			lastAccessed: time.Now(),
			failCount:    0,
		}
		rl.limiters[ip] = entry
	}
	entry.lastAccessed = time.Now()
	rl.mu.Unlock()

	return entry.limiter.Allow()
}

// RecordFailure tracks a failed authentication attempt and applies exponential backoff
func (rl *RateLimiter) RecordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		entry = &limiterEntry{
			limiter:      rate.NewLimiter(rl.rate, rl.burst),
			lastAccessed: time.Now(),
			failCount:    0,
		}
		rl.limiters[ip] = entry
	}

	entry.failCount++
	entry.lastAccessed = time.Now()

	// Exponential backoff: reduce rate after repeated failures
	if entry.failCount >= 5 {
		// Severely limit after 5 failures: 1 request per 100 seconds
		entry.limiter = rate.NewLimiter(rate.Limit(0.01), 1)
	} else if entry.failCount >= 3 {
		// Reduce after 3 failures: 1 request per 10 seconds
		entry.limiter = rate.NewLimiter(rate.Limit(0.1), 2)
	}
}

// RecordSuccess resets the failure count on successful authentication
func (rl *RateLimiter) RecordSuccess(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if entry, exists := rl.limiters[ip]; exists {
		entry.failCount = 0 // Reset failures on success
		entry.lastAccessed = time.Now()
	}
}

// Cleanup removes old limiters (should be called periodically)
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, entry := range rl.limiters {
		// Only remove entries that haven't been accessed recently
		if now.Sub(entry.lastAccessed) > rl.maxIdleAge {
			delete(rl.limiters, ip)
		}
	}
}

// StartCleanup starts a goroutine that periodically cleans up limiters
func (rl *RateLimiter) StartCleanup(interval time.Duration, stopChan <-chan struct{}) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				rl.Cleanup()
			case <-stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}
