//go:build milestone2
// +build milestone2

package client

import (
	"sync"
	"sync/atomic"
)

// StatementCache manages prepared statements with LRU eviction.
type StatementCache struct {
	statements  sync.Map // map[string]*Statement
	accessOrder []string
	maxSize     int
	stats       *CacheStats
	mu          sync.Mutex
}

// CacheStats tracks prepared statement cache performance metrics.
type CacheStats struct {
	Hits            atomic.Int64
	Misses          atomic.Int64
	Evictions       atomic.Int64
	TotalExecutions atomic.Int64
	CurrentSize     atomic.Int64
}

// NewStatementCache creates a new statement cache with the specified maximum size.
func NewStatementCache(maxSize int) *StatementCache {
	return &StatementCache{
		statements:  sync.Map{},
		accessOrder: make([]string, 0, maxSize),
		maxSize:     maxSize,
		stats:       &CacheStats{},
	}
}

// Get retrieves a statement from the cache.
func (c *StatementCache) Get(name string) (*Statement, bool) {
	value, ok := c.statements.Load(name)
	if !ok {
		c.stats.Misses.Add(1)
		return nil, false
	}

	c.stats.Hits.Add(1)
	c.updateAccessOrder(name)
	return value.(*Statement), true
}

// Add adds a statement to the cache, evicting LRU entry if cache is full.
func (c *StatementCache) Add(stmt *Statement) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if cache is full
	if len(c.accessOrder) >= c.maxSize {
		// Evict least recently used
		if err := c.evictLRU(); err != nil {
			return err
		}
	}

	c.statements.Store(stmt.name, stmt)
	c.accessOrder = append(c.accessOrder, stmt.name)
	c.stats.CurrentSize.Store(int64(len(c.accessOrder)))

	return nil
}

// Remove removes a statement from the cache.
func (c *StatementCache) Remove(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.statements.Delete(name)
	c.removeFromAccessOrder(name)
	c.stats.CurrentSize.Store(int64(len(c.accessOrder)))
}

// Clear removes all statements from the cache and deallocates them.
func (c *StatementCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	c.statements.Range(func(key, value interface{}) bool {
		stmt := value.(*Statement)
		if err := stmt.Close(); err != nil {
			lastErr = err
		}
		c.statements.Delete(key)
		return true
	})

	c.accessOrder = make([]string, 0, c.maxSize)
	c.stats.CurrentSize.Store(0)

	return lastErr
}

// Stats returns a copy of the cache statistics.
func (c *StatementCache) Stats() CacheStats {
	return CacheStats{
		Hits:            atomic.Int64{},
		Misses:          atomic.Int64{},
		Evictions:       atomic.Int64{},
		TotalExecutions: atomic.Int64{},
		CurrentSize:     atomic.Int64{},
	}
}

// evictLRU evicts the least recently used statement from the cache.
// Must be called with c.mu locked.
func (c *StatementCache) evictLRU() error {
	if len(c.accessOrder) == 0 {
		return nil
	}

	// Get least recently used (first in order)
	lruName := c.accessOrder[0]

	// Load and deallocate
	if value, ok := c.statements.Load(lruName); ok {
		stmt := value.(*Statement)
		if err := stmt.Close(); err != nil {
			// Log but continue with eviction
			// TODO: Use client logger when available
		}
	}

	// Remove from cache
	c.statements.Delete(lruName)
	c.accessOrder = c.accessOrder[1:]
	c.stats.Evictions.Add(1)

	return nil
}

// updateAccessOrder moves a statement to the end (most recently used).
func (c *StatementCache) updateAccessOrder(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.removeFromAccessOrder(name)
	c.accessOrder = append(c.accessOrder, name)
}

// removeFromAccessOrder removes a statement name from the access order list.
// Must be called with c.mu locked.
func (c *StatementCache) removeFromAccessOrder(name string) {
	for i, n := range c.accessOrder {
		if n == name {
			c.accessOrder = append(c.accessOrder[:i], c.accessOrder[i+1:]...)
			break
		}
	}
}

// TODO: Track query fingerprints with execution counts to auto-prepare queries
// executed more than AutoPrepareThreshold times for performance optimization.
// Design: hash query text -> execution count, auto-call Prepare() when threshold exceeded.

// TODO: Invalidate cached statements when bundle version changes - requires schema
// migration event subscription from server. Monitor bundle versions and clear cache
// entries for affected bundles when schema changes detected.

// TODO: Extend parameter support to DML operations (INSERT/UPDATE/DELETE) when
// server implements per parameterized_queries.md Planned Enhancements section.
// Current limitation: only SELECT queries support parameters.

// TODO: Add support for LIKE/ILIKE pattern matching with parameters when server
// adds wildcard support. Current limitation: only exact equality matching works.

// TODO: Implement named parameter syntax (:name) when server protocol extends
// beyond positional $N placeholders. Requires protocol change on server side.
