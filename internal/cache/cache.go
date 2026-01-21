package cache

import (
	"context"
	"log"
	"sync"
	"time"
)

// CacheEntry holds cached data with metadata
type CacheEntry struct {
	Data          any
	ExpiresAt     time.Time
	LastRefreshAt time.Time
	RefreshFunc   RefreshFunc
}

// RefreshFunc defines how to refresh cached data
type RefreshFunc func(ctx context.Context) (any, error)

// Cache is a thread-safe in-memory cache with TTL and background refresh
type Cache struct {
	store         map[string]*CacheEntry
	mu            sync.RWMutex
	defaultTTL    time.Duration
	refreshBuffer time.Duration
	stats         CacheStats
	stopChan      chan bool
}

type CacheStats struct {
	Hits      int64
	Misses    int64
	Refreshes int64
	mu        sync.RWMutex
}

// NewCache creates a new cache with default TTL and refresh buffer
func NewCache(defaultTTL time.Duration, refreshBuffer time.Duration) *Cache {
	c := &Cache{
		store:         make(map[string]*CacheEntry),
		defaultTTL:    defaultTTL,
		refreshBuffer: refreshBuffer,
		stopChan:      make(chan bool),
	}

	go c.cleanupExpired()
	go c.backgroundRefresh()
	go c.logStats()

	return c
}

// Get retrieves a value from cache
// Returns (data, found, isStale)
func (c *Cache) Get(key string) (any, bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.store[key]
	if !exists {
		c.recordMiss()
		return nil, false, false
	}

	now := time.Now()

	// Check if expired
	if now.After(entry.ExpiresAt) {
		c.recordMiss()
		return nil, false, false
	}

	c.recordHit()

	// Check if stale (needs refresh soon)
	refreshThreshold := entry.ExpiresAt.Add(-c.refreshBuffer)
	isStale := now.After(refreshThreshold)

	return entry.Data, true, isStale
}

// Set stores a value in cache with default TTL
func (c *Cache) Set(key string, value any) {
	c.SetWithTTL(key, value, c.defaultTTL, nil)
}

// SetWithRefresh stores a value and registers a refresh function
func (c *Cache) SetWithRefresh(key string, value any, ttl time.Duration, refreshFunc RefreshFunc) {
	c.SetWithTTL(key, value, ttl, refreshFunc)
}

// SetWithTTL stores a value with custom TTL and optional refresh function
func (c *Cache) SetWithTTL(key string, value any, ttl time.Duration, refreshFunc RefreshFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[key] = &CacheEntry{
		Data:          value,
		ExpiresAt:     time.Now().Add(ttl),
		LastRefreshAt: time.Now(),
		RefreshFunc:   refreshFunc,
	}
}

// GetOrSet returns cached value or sets it using the provided function
func (c *Cache) GetOrSet(key string, ttl time.Duration, fetchFunc func() (any, error)) (any, error) {
	// Try to get from cache
	if data, found, isStale := c.Get(key); found {
		// If stale, trigger background refresh
		if isStale {
			go c.refreshKey(key)
		}
		return data, nil
	}

	// Cache miss - fetch data
	data, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	c.SetWithTTL(key, data, ttl, nil)

	return data, nil
}

// GetOrSetWithRefresh is like GetOrSet but registers a refresh function
func (c *Cache) GetOrSetWithRefresh(key string, ttl time.Duration, refreshFunc RefreshFunc) (any, error) {
	// Try to get from cache
	if data, found, isStale := c.Get(key); found {
		// If stale, trigger background refresh
		if isStale {
			go c.refreshKey(key)
		}
		return data, nil
	}

	// Cache miss - fetch data
	ctx := context.Background()
	data, err := refreshFunc(ctx)
	if err != nil {
		return nil, err
	}

	c.SetWithRefresh(key, data, ttl, refreshFunc)

	return data, nil
}

// refreshKey refreshes a specific cache key using its refresh function
func (c *Cache) refreshKey(key string) {
	c.mu.RLock()
	entry, exists := c.store[key]
	if !exists || entry.RefreshFunc == nil {
		c.mu.RUnlock()
		return
	}
	refreshFunc := entry.RefreshFunc
	c.mu.RUnlock()

	// Fetch new data
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	newData, err := refreshFunc(ctx)
	if err != nil {
		log.Printf("Cache refresh failed for key %s: %v", key, err)
		return
	}

	// Update cache
	c.mu.Lock()
	if entry, exists := c.store[key]; exists {
		entry.Data = newData
		entry.ExpiresAt = time.Now().Add(c.defaultTTL)
		entry.LastRefreshAt = time.Now()
	}
	c.mu.Unlock()

	c.stats.mu.Lock()
	c.stats.Refreshes++
	c.stats.mu.Unlock()
}

// Delete removes a value from cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
}

// Clear removes all entries
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[string]*CacheEntry)
}

// InvalidatePattern removes all keys matching a pattern (prefix match)
func (c *Cache) InvalidatePattern(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.store {
		if matchesPattern(key, pattern) {
			delete(c.store, key)
		}
	}
}

// cleanupExpired removes expired entries every minute
func (c *Cache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, entry := range c.store {
				if now.After(entry.ExpiresAt) {
					delete(c.store, key)
				}
			}
			c.mu.Unlock()
		case <-c.stopChan:
			return
		}
	}
}

// backgroundRefresh proactively refreshes stale entries
func (c *Cache) backgroundRefresh() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.RLock()
			now := time.Now()
			var keysToRefresh []string

			for key, entry := range c.store {
				if entry.RefreshFunc == nil {
					continue
				}

				// Check if entry needs refresh (within refresh buffer)
				refreshThreshold := entry.ExpiresAt.Add(-c.refreshBuffer)
				if now.After(refreshThreshold) && now.Before(entry.ExpiresAt) {
					keysToRefresh = append(keysToRefresh, key)
				}
			}
			c.mu.RUnlock()

			// Refresh in background
			for _, key := range keysToRefresh {
				go c.refreshKey(key)
			}

		case <-c.stopChan:
			return
		}
	}
}

// GetStats returns current cache statistics
func (c *Cache) GetStats() CacheStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	return CacheStats{
		Hits:      c.stats.Hits,
		Misses:    c.stats.Misses,
		Refreshes: c.stats.Refreshes,
	}
}

// logStats logs cache statistics periodically
func (c *Cache) logStats() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := c.GetStats()
			total := stats.Hits + stats.Misses
			if total == 0 {
				continue
			}

			hitRate := float64(stats.Hits) / float64(total) * 100

			c.mu.RLock()
			entries := len(c.store)
			c.mu.RUnlock()

			log.Printf("Cache Stats - Hits: %d, Misses: %d, Hit Rate: %.2f%%, Refreshes: %d, Entries: %d",
				stats.Hits, stats.Misses, hitRate, stats.Refreshes, entries)

		case <-c.stopChan:
			return
		}
	}
}

func (c *Cache) recordHit() {
	c.stats.mu.Lock()
	c.stats.Hits++
	c.stats.mu.Unlock()
}

func (c *Cache) recordMiss() {
	c.stats.mu.Lock()
	c.stats.Misses++
	c.stats.mu.Unlock()
}

// Stop gracefully shuts down background goroutines
func (c *Cache) Stop() {
	close(c.stopChan)
}

// matchesPattern performs simple prefix matching
func matchesPattern(key, pattern string) bool {
	return len(pattern) > 0 && len(key) >= len(pattern) && key[:len(pattern)] == pattern
}
