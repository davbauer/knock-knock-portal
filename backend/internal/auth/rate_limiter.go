package auth

import (
	"container/list"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter provides IP-based rate limiting with LRU eviction
type RateLimiter struct {
	limiters   map[string]*list.Element // map[IP]*list.Element (points to lruEntry)
	lruList    *list.List               // LRU list for eviction
	mu         sync.Mutex
	rate       rate.Limit
	burst      int
	maxEntries int           // Maximum number of limiters to cache
	maxIdleAge time.Duration
}

// lruEntry represents an entry in the LRU cache
type lruEntry struct {
	ip    string
	entry *limiterEntry
}

// limiterEntry tracks a rate limiter and its metadata
type limiterEntry struct {
	limiter      *rate.Limiter
	lastAccessed time.Time
	failCount    int // Track consecutive failures for exponential backoff
}

// NewRateLimiter creates a new rate limiter with LRU eviction
// rate: requests per second
// burst: maximum burst size
// maxEntries: maximum number of IP limiters to cache (0 = unlimited, recommended: 10000)
func NewRateLimiter(requestsPerMinute int, burst int, maxEntries int) *RateLimiter {
	if maxEntries == 0 {
		maxEntries = 10000 // Default safe limit
	}
	return &RateLimiter{
		limiters:   make(map[string]*list.Element),
		lruList:    list.New(),
		rate:       rate.Limit(float64(requestsPerMinute) / 60.0), // Convert to per-second
		burst:      burst,
		maxEntries: maxEntries,
		maxIdleAge: 15 * time.Minute,
	}
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	elem, exists := rl.limiters[ip]
	if !exists {
		// Check if we need to evict oldest entry
		if rl.lruList.Len() >= rl.maxEntries {
			rl.evictOldest()
		}

		// Create new entry
		entry := &limiterEntry{
			limiter:      rate.NewLimiter(rl.rate, rl.burst),
			lastAccessed: time.Now(),
			failCount:    0,
		}
		lru := &lruEntry{
			ip:    ip,
			entry: entry,
		}
		elem = rl.lruList.PushFront(lru)
		rl.limiters[ip] = elem
	} else {
		// Move to front (most recently used)
		rl.lruList.MoveToFront(elem)
		lru := elem.Value.(*lruEntry)
		lru.entry.lastAccessed = time.Now()
	}

	lru := elem.Value.(*lruEntry)
	return lru.entry.limiter.Allow()
}

// evictOldest removes the least recently used entry
func (rl *RateLimiter) evictOldest() {
	elem := rl.lruList.Back()
	if elem != nil {
		rl.lruList.Remove(elem)
		lru := elem.Value.(*lruEntry)
		delete(rl.limiters, lru.ip)
	}
}

// RecordFailure tracks a failed authentication attempt and applies exponential backoff
func (rl *RateLimiter) RecordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	elem, exists := rl.limiters[ip]
	if !exists {
		// Check if we need to evict oldest entry
		if rl.lruList.Len() >= rl.maxEntries {
			rl.evictOldest()
		}

		entry := &limiterEntry{
			limiter:      rate.NewLimiter(rl.rate, rl.burst),
			lastAccessed: time.Now(),
			failCount:    0,
		}
		lru := &lruEntry{
			ip:    ip,
			entry: entry,
		}
		elem = rl.lruList.PushFront(lru)
		rl.limiters[ip] = elem
	}

	lru := elem.Value.(*lruEntry)
	lru.entry.failCount++
	lru.entry.lastAccessed = time.Now()

	// Exponential backoff: reduce rate after repeated failures
	if lru.entry.failCount >= 5 {
		// Severely limit after 5 failures: 1 request per 100 seconds
		lru.entry.limiter = rate.NewLimiter(rate.Limit(0.01), 1)
	} else if lru.entry.failCount >= 3 {
		// Reduce after 3 failures: 1 request per 10 seconds
		lru.entry.limiter = rate.NewLimiter(rate.Limit(0.1), 2)
	}

	// Move to front
	rl.lruList.MoveToFront(elem)
}

// RecordSuccess resets the failure count on successful authentication
func (rl *RateLimiter) RecordSuccess(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if elem, exists := rl.limiters[ip]; exists {
		lru := elem.Value.(*lruEntry)
		lru.entry.failCount = 0 // Reset failures on success
		lru.entry.lastAccessed = time.Now()
		rl.lruList.MoveToFront(elem)
	}
}

// Cleanup removes old limiters (should be called periodically)
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	toRemove := []*list.Element{}

	// Iterate from back (oldest) to front
	for elem := rl.lruList.Back(); elem != nil; elem = elem.Prev() {
		lru := elem.Value.(*lruEntry)
		// Only remove entries that haven't been accessed recently
		if now.Sub(lru.entry.lastAccessed) > rl.maxIdleAge {
			toRemove = append(toRemove, elem)
		} else {
			// Since list is ordered by access time, we can stop
			break
		}
	}

	for _, elem := range toRemove {
		lru := elem.Value.(*lruEntry)
		delete(rl.limiters, lru.ip)
		rl.lruList.Remove(elem)
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
