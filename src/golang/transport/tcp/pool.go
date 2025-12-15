//go:build !wasm
// +build !wasm

package tcp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// poolStats tracks connection pool statistics
type poolStats struct {
	activeConnections atomic.Int32
	idleConnections   atomic.Int32
	totalConnections  atomic.Int32
	waitCount         atomic.Int64
	hits              atomic.Int64
	misses            atomic.Int64
	timeouts          atomic.Int64
	errors            atomic.Int64
}

// connectionPool manages a pool of TCP connections
type connectionPool struct {
	conns               chan *tcpConnection
	factory             func(ctx context.Context) (*tcpConnection, error)
	minIdle             int
	maxOpen             int
	idleTimeout         time.Duration
	healthCheckInterval time.Duration
	stats               poolStats
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	mu                  sync.RWMutex
	closed              bool
}

// newConnectionPool creates a new connection pool
func newConnectionPool(
	factory func(ctx context.Context) (*tcpConnection, error),
	minIdle, maxOpen int,
	idleTimeout, healthCheckInterval time.Duration,
) *connectionPool {
	if minIdle < 0 {
		minIdle = 0
	}
	if maxOpen < 1 {
		maxOpen = 1
	}
	if minIdle > maxOpen {
		minIdle = maxOpen
	}

	return &connectionPool{
		conns:               make(chan *tcpConnection, maxOpen),
		factory:             factory,
		minIdle:             minIdle,
		maxOpen:             maxOpen,
		idleTimeout:         idleTimeout,
		healthCheckInterval: healthCheckInterval,
		stopCh:              make(chan struct{}),
	}
}

// Initialize starts the pool and creates minimum idle connections
func (p *connectionPool) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("pool is closed")
	}

	// Create initial connections
	for i := 0; i < p.minIdle; i++ {
		conn, err := p.factory(ctx)
		if err != nil {
			p.closeAllConnections()
			return fmt.Errorf("failed to create initial connection: %w", err)
		}

		p.conns <- conn
		p.stats.totalConnections.Add(1)
		p.stats.idleConnections.Add(1)
	}

	// Start background workers
	p.wg.Add(2)
	go p.cleanupWorker()
	go p.healthCheckWorker()

	return nil
}

// Get acquires a connection from the pool
func (p *connectionPool) Get(ctx context.Context) (*tcpConnection, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	p.stats.waitCount.Add(1)

	select {
	case <-ctx.Done():
		p.stats.timeouts.Add(1)
		return nil, ctx.Err()

	case conn := <-p.conns:
		// Got connection from pool
		p.stats.hits.Add(1)
		p.stats.idleConnections.Add(-1)
		p.stats.activeConnections.Add(1)

		// Validate connection is still alive
		if !conn.isAlive() {
			p.stats.totalConnections.Add(-1)
			p.stats.activeConnections.Add(-1)
			conn.close()
			// Try to get another connection
			return p.Get(ctx)
		}

		return conn, nil

	default:
		// No idle connection available, try to create new one
		currentTotal := p.stats.totalConnections.Load()
		if currentTotal < int32(p.maxOpen) {
			conn, err := p.factory(ctx)
			if err != nil {
				p.stats.errors.Add(1)
				return nil, fmt.Errorf("failed to create new connection: %w", err)
			}

			p.stats.misses.Add(1)
			p.stats.totalConnections.Add(1)
			p.stats.activeConnections.Add(1)
			return conn, nil
		}

		// Pool is full, wait for a connection
		p.stats.misses.Add(1)
		select {
		case <-ctx.Done():
			p.stats.timeouts.Add(1)
			return nil, ctx.Err()
		case conn := <-p.conns:
			p.stats.idleConnections.Add(-1)
			p.stats.activeConnections.Add(1)

			if !conn.isAlive() {
				p.stats.totalConnections.Add(-1)
				p.stats.activeConnections.Add(-1)
				conn.close()
				return p.Get(ctx)
			}

			return conn, nil
		}
	}
}

// Put returns a connection to the pool
func (p *connectionPool) Put(conn *tcpConnection) {
	if conn == nil {
		return
	}

	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed || !conn.isAlive() {
		p.stats.totalConnections.Add(-1)
		p.stats.activeConnections.Add(-1)
		conn.close()
		return
	}

	p.stats.activeConnections.Add(-1)

	select {
	case p.conns <- conn:
		p.stats.idleConnections.Add(1)
	default:
		// Pool is full, close the connection
		p.stats.totalConnections.Add(-1)
		conn.close()
	}
}

// Close closes the pool and all connections
func (p *connectionPool) Close() error {
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

// closeAllConnections closes all connections in the pool
func (p *connectionPool) closeAllConnections() {
	close(p.conns)
	for conn := range p.conns {
		conn.close()
	}
}

// cleanupWorker periodically removes idle connections
func (p *connectionPool) cleanupWorker() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.idleTimeout / 2)
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

// cleanupIdleConnections removes connections that have been idle too long
func (p *connectionPool) cleanupIdleConnections() {
	now := time.Now()
	currentIdle := p.stats.idleConnections.Load()

	// Keep at least minIdle connections
	toRemove := int(currentIdle) - p.minIdle
	if toRemove <= 0 {
		return
	}

	removed := 0
	for i := 0; i < toRemove; i++ {
		select {
		case conn := <-p.conns:
			idleTime := now.Sub(conn.lastActivityTime())
			if idleTime > p.idleTimeout {
				conn.close()
				p.stats.totalConnections.Add(-1)
				p.stats.idleConnections.Add(-1)
				removed++
			} else {
				// Connection is not idle enough, put it back
				p.conns <- conn
			}
		default:
			// No more connections to check
			return
		}
	}
}

// healthCheckWorker periodically checks connection health
func (p *connectionPool) healthCheckWorker() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.healthCheckConnections()
		}
	}
}

// healthCheckConnections checks health of idle connections
func (p *connectionPool) healthCheckConnections() {
	currentIdle := int(p.stats.idleConnections.Load())
	if currentIdle == 0 {
		return
	}

	// Check up to half of idle connections to avoid blocking
	toCheck := currentIdle / 2
	if toCheck < 1 {
		toCheck = 1
	}

	for i := 0; i < toCheck; i++ {
		select {
		case conn := <-p.conns:
			if !conn.isAlive() {
				// Connection is dead, don't put it back
				conn.close()
				p.stats.totalConnections.Add(-1)
				p.stats.idleConnections.Add(-1)
			} else {
				// Connection is alive, put it back
				p.conns <- conn
			}
		default:
			// No more connections to check
			return
		}
	}
}
