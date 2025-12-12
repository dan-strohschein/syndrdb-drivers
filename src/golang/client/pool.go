//go:build !wasm
// +build !wasm

package client

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// PoolStats tracks connection pool statistics.
// TODO: future Prometheus metrics integration could expose these as gauges and histograms via HTTP endpoint
type PoolStats struct {
	ActiveConnections atomic.Int32
	IdleConnections   atomic.Int32
	TotalConnections  atomic.Int32
	WaitCount         atomic.Int64
	WaitDuration      atomic.Int64 // nanoseconds
	Hits              atomic.Int64
	Misses            atomic.Int64
	Timeouts          atomic.Int64
	Errors            atomic.Int64
}

// ConnectionPool manages a pool of database connections with automatic cleanup.
type ConnectionPool struct {
	conns               chan ConnectionInterface
	factory             func(ctx context.Context) (ConnectionInterface, error)
	minIdle             int
	maxOpen             int
	idleTimeout         time.Duration
	healthCheckInterval time.Duration
	stats               PoolStats
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	mu                  sync.RWMutex
	closed              bool
}

// NewConnectionPool creates a new connection pool with the specified configuration.
func NewConnectionPool(
	factory func(ctx context.Context) (ConnectionInterface, error),
	minIdle, maxOpen int,
	idleTimeout, healthCheckInterval time.Duration,
) *ConnectionPool {
	if minIdle < 0 {
		minIdle = 0
	}
	if maxOpen < 1 {
		maxOpen = 1
	}
	if minIdle > maxOpen {
		minIdle = maxOpen
	}

	pool := &ConnectionPool{
		conns:               make(chan ConnectionInterface, maxOpen),
		factory:             factory,
		minIdle:             minIdle,
		maxOpen:             maxOpen,
		idleTimeout:         idleTimeout,
		healthCheckInterval: healthCheckInterval,
		stopCh:              make(chan struct{}),
	}

	return pool
}

// Initialize starts the pool and creates minimum idle connections.
func (p *ConnectionPool) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("pool is closed")
	}

	// Create initial connections up to minIdle
	for i := 0; i < p.minIdle; i++ {
		conn, err := p.factory(ctx)
		if err != nil {
			// Close any connections created so far
			p.closeAllConnections()
			return fmt.Errorf("failed to create initial connection: %w", err)
		}

		p.conns <- conn
		p.stats.TotalConnections.Add(1)
		p.stats.IdleConnections.Add(1)
	}

	// Start background workers
	p.wg.Add(2)
	go p.cleanupWorker()
	go p.healthCheckWorker()

	return nil
}

// Get acquires a connection from the pool.
func (p *ConnectionPool) Get(ctx context.Context) (ConnectionInterface, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	startWait := time.Now()
	p.stats.WaitCount.Add(1)

	select {
	case <-ctx.Done():
		p.stats.Timeouts.Add(1)
		return nil, ctx.Err()

	case conn := <-p.conns:
		// Got connection from pool
		waitDuration := time.Since(startWait)
		p.stats.WaitDuration.Add(int64(waitDuration))
		p.stats.Hits.Add(1)
		p.stats.IdleConnections.Add(-1)
		p.stats.ActiveConnections.Add(1)

		// Validate connection is still alive
		if !conn.IsAlive() {
			p.stats.TotalConnections.Add(-1)
			p.stats.ActiveConnections.Add(-1)
			conn.Close()
			// Try to get another connection
			return p.Get(ctx)
		}

		return conn, nil

	default:
		// No idle connection available, try to create new one
		currentTotal := p.stats.TotalConnections.Load()
		if currentTotal < int32(p.maxOpen) {
			conn, err := p.factory(ctx)
			if err != nil {
				p.stats.Errors.Add(1)
				return nil, fmt.Errorf("failed to create new connection: %w", err)
			}

			waitDuration := time.Since(startWait)
			p.stats.WaitDuration.Add(int64(waitDuration))
			p.stats.Misses.Add(1)
			p.stats.TotalConnections.Add(1)
			p.stats.ActiveConnections.Add(1)

			return conn, nil
		}

		// Pool is at max capacity, wait for a connection to be released
		select {
		case <-ctx.Done():
			p.stats.Timeouts.Add(1)
			return nil, ctx.Err()

		case conn := <-p.conns:
			waitDuration := time.Since(startWait)
			p.stats.WaitDuration.Add(int64(waitDuration))
			p.stats.Hits.Add(1)
			p.stats.IdleConnections.Add(-1)
			p.stats.ActiveConnections.Add(1)

			// Validate connection is still alive
			if !conn.IsAlive() {
				p.stats.TotalConnections.Add(-1)
				p.stats.ActiveConnections.Add(-1)
				conn.Close()
				// Try to get another connection
				return p.Get(ctx)
			}

			return conn, nil
		}
	}
}

// Put returns a connection to the pool.
func (p *ConnectionPool) Put(conn ConnectionInterface) {
	if conn == nil {
		return
	}

	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		conn.Close()
		return
	}

	p.stats.ActiveConnections.Add(-1)

	// Validate connection health before returning to pool
	if !conn.IsAlive() {
		p.stats.TotalConnections.Add(-1)
		conn.Close()
		return
	}

	// Try to return connection to pool
	select {
	case p.conns <- conn:
		p.stats.IdleConnections.Add(1)
	default:
		// Pool is full, close the connection
		p.stats.TotalConnections.Add(-1)
		conn.Close()
	}
}

// Stats returns a snapshot of pool statistics.
func (p *ConnectionPool) Stats() PoolStats {
	stats := PoolStats{}
	stats.ActiveConnections.Store(p.stats.ActiveConnections.Load())
	stats.IdleConnections.Store(p.stats.IdleConnections.Load())
	stats.TotalConnections.Store(p.stats.TotalConnections.Load())
	stats.WaitCount.Store(p.stats.WaitCount.Load())
	stats.WaitDuration.Store(p.stats.WaitDuration.Load())
	stats.Hits.Store(p.stats.Hits.Load())
	stats.Misses.Store(p.stats.Misses.Load())
	stats.Timeouts.Store(p.stats.Timeouts.Load())
	stats.Errors.Store(p.stats.Errors.Load())
	return stats
}

// Close closes all connections in the pool gracefully.
// Context is currently not used but reserved for future graceful shutdown with deadlines.
func (p *ConnectionPool) Close(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Signal workers to stop
	close(p.stopCh)

	// Wait for workers to finish
	p.wg.Wait()

	// Close all connections
	p.closeAllConnections()

	return nil
}

// cleanupWorker periodically removes idle connections that exceed idleTimeout.
func (p *ConnectionPool) cleanupWorker() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.idleTimeout / 4)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return

		case <-ticker.C:
			p.cleanupIdleConnections()
		}
	}
}

// cleanupIdleConnections removes stale idle connections while maintaining minIdle.
func (p *ConnectionPool) cleanupIdleConnections() {
	now := time.Now()
	currentIdle := int(p.stats.IdleConnections.Load())

	for currentIdle > p.minIdle {
		select {
		case conn := <-p.conns:
			// Check if connection has been idle too long
			if now.Sub(conn.LastActivity()) > p.idleTimeout {
				p.stats.IdleConnections.Add(-1)
				p.stats.TotalConnections.Add(-1)
				conn.Close()
				currentIdle--
			} else {
				// Connection is still fresh, return it
				p.conns <- conn
				return
			}

		default:
			return
		}
	}
}

// healthCheckWorker periodically pings idle connections.
func (p *ConnectionPool) healthCheckWorker() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return

		case <-ticker.C:
			p.healthCheckIdleConnections()
		}
	}
}

// healthCheckIdleConnections pings idle connections and removes dead ones.
func (p *ConnectionPool) healthCheckIdleConnections() {
	idleCount := int(p.stats.IdleConnections.Load())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check up to all idle connections
	for i := 0; i < idleCount; i++ {
		select {
		case conn := <-p.conns:
			// Try to ping the connection
			if err := conn.Ping(ctx); err != nil || !conn.IsAlive() {
				// Connection is dead, don't return it
				p.stats.IdleConnections.Add(-1)
				p.stats.TotalConnections.Add(-1)
				conn.Close()
			} else {
				// Connection is healthy, return it
				p.conns <- conn
			}

		default:
			return
		}
	}
}

// closeAllConnections closes all connections in the pool.
func (p *ConnectionPool) closeAllConnections() {
	for {
		select {
		case conn := <-p.conns:
			conn.Close()
		default:
			return
		}
	}
}
