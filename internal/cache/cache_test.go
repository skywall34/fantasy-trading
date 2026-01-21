package cache

import (
	"context"
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	c := NewCache(1*time.Second, 500*time.Millisecond)
	defer c.Stop()

	// Test Set and Get
	c.Set("test_key", "test_value")
	data, found, isStale := c.Get("test_key")
	if !found {
		t.Error("Expected to find cached value")
	}
	if data != "test_value" {
		t.Errorf("Expected 'test_value', got %v", data)
	}
	if isStale {
		t.Error("Value should not be stale immediately after set")
	}

	// Test cache miss
	_, found, _ = c.Get("nonexistent_key")
	if found {
		t.Error("Expected cache miss for nonexistent key")
	}
}

func TestCacheExpiration(t *testing.T) {
	c := NewCache(100*time.Millisecond, 50*time.Millisecond)
	defer c.Stop()

	c.Set("expiring_key", "expiring_value")

	// Should be found immediately
	_, found, _ := c.Get("expiring_key")
	if !found {
		t.Error("Expected to find cached value")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found, _ = c.Get("expiring_key")
	if found {
		t.Error("Expected cache miss after expiration")
	}
}

func TestCacheStaleDetection(t *testing.T) {
	c := NewCache(1*time.Second, 500*time.Millisecond)
	defer c.Stop()

	c.Set("stale_key", "stale_value")

	// Should not be stale immediately
	_, found, isStale := c.Get("stale_key")
	if !found {
		t.Error("Expected to find cached value")
	}
	if isStale {
		t.Error("Value should not be stale immediately")
	}

	// Wait to enter stale window
	time.Sleep(600 * time.Millisecond)

	// Should be stale but still found
	_, found, isStale = c.Get("stale_key")
	if !found {
		t.Error("Expected to find cached value even if stale")
	}
	if !isStale {
		t.Error("Value should be stale after refresh threshold")
	}
}

func TestCacheGetOrSet(t *testing.T) {
	c := NewCache(1*time.Second, 500*time.Millisecond)
	defer c.Stop()

	callCount := 0
	fetchFunc := func() (any, error) {
		callCount++
		return "fetched_value", nil
	}

	// First call should fetch
	data, err := c.GetOrSet("fetch_key", 1*time.Second, fetchFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if data != "fetched_value" {
		t.Errorf("Expected 'fetched_value', got %v", data)
	}
	if callCount != 1 {
		t.Errorf("Expected fetch function to be called once, called %d times", callCount)
	}

	// Second call should use cache
	data, err = c.GetOrSet("fetch_key", 1*time.Second, fetchFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if data != "fetched_value" {
		t.Errorf("Expected 'fetched_value', got %v", data)
	}
	if callCount != 1 {
		t.Errorf("Expected fetch function not to be called again, called %d times", callCount)
	}
}

func TestCacheRefreshFunction(t *testing.T) {
	c := NewCache(1*time.Second, 500*time.Millisecond)
	defer c.Stop()

	callCount := 0
	refreshFunc := func(ctx context.Context) (any, error) {
		callCount++
		return "refreshed_value", nil
	}

	// First call should fetch
	data, err := c.GetOrSetWithRefresh("refresh_key", 1*time.Second, refreshFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if data != "refreshed_value" {
		t.Errorf("Expected 'refreshed_value', got %v", data)
	}
	if callCount != 1 {
		t.Errorf("Expected refresh function to be called once, called %d times", callCount)
	}

	// Second call should use cache
	data, err = c.GetOrSetWithRefresh("refresh_key", 1*time.Second, refreshFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected refresh function not to be called again, called %d times", callCount)
	}
}

func TestCacheDelete(t *testing.T) {
	c := NewCache(1*time.Second, 500*time.Millisecond)
	defer c.Stop()

	c.Set("delete_key", "delete_value")

	// Should be found
	_, found, _ := c.Get("delete_key")
	if !found {
		t.Error("Expected to find cached value")
	}

	// Delete
	c.Delete("delete_key")

	// Should not be found
	_, found, _ = c.Get("delete_key")
	if found {
		t.Error("Expected cache miss after delete")
	}
}

func TestCacheInvalidatePattern(t *testing.T) {
	c := NewCache(1*time.Second, 500*time.Millisecond)
	defer c.Stop()

	c.Set("account:1", "account1")
	c.Set("account:2", "account2")
	c.Set("activities:1", "activities1")

	// Invalidate all account entries
	c.InvalidatePattern("account:")

	// Account entries should be gone
	_, found, _ := c.Get("account:1")
	if found {
		t.Error("Expected account:1 to be invalidated")
	}
	_, found, _ = c.Get("account:2")
	if found {
		t.Error("Expected account:2 to be invalidated")
	}

	// Activities entry should still exist
	_, found, _ = c.Get("activities:1")
	if !found {
		t.Error("Expected activities:1 to still exist")
	}
}

func TestCacheStats(t *testing.T) {
	c := NewCache(1*time.Second, 500*time.Millisecond)
	defer c.Stop()

	c.Set("stats_key", "stats_value")

	// Generate some hits
	c.Get("stats_key")
	c.Get("stats_key")

	// Generate some misses
	c.Get("nonexistent1")
	c.Get("nonexistent2")
	c.Get("nonexistent3")

	stats := c.GetStats()
	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 3 {
		t.Errorf("Expected 3 misses, got %d", stats.Misses)
	}
}
