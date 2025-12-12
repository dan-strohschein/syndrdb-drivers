//go:build js && wasm
// +build js,wasm

package client

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

// ConnectionPool is a stub for WASM builds where pooling is not supported.
// WASM environments don't support goroutines reliably, so pool operations
// are no-ops that always return errors.
type ConnectionPool struct{}

// PoolStats tracks connection pool statistics (stub for WASM).
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

// NewConnectionPool returns an error in WASM builds as pooling is not supported.
func NewConnectionPool(
	factory func(ctx context.Context) (ConnectionInterface, error),
	minIdle, maxOpen int,
	idleTimeout, healthCheckInterval time.Duration,
) *ConnectionPool {
	// In WASM, we always return nil pool - single connection mode will be used
	return nil
}

// Get always returns an error in WASM builds.
func (p *ConnectionPool) Get(ctx context.Context) (ConnectionInterface, error) {
	return nil, errors.New("connection pooling is not supported in WASM builds")
}

// Put is a no-op in WASM builds.
func (p *ConnectionPool) Put(conn ConnectionInterface) {}

// Stats returns empty statistics in WASM builds.
func (p *ConnectionPool) Stats() PoolStats {
	return PoolStats{}
}

// Initialize is a no-op in WASM builds.
func (p *ConnectionPool) Initialize(ctx context.Context) error {
	return errors.New("connection pooling is not supported in WASM builds")
}

// Close is a no-op in WASM builds.
func (p *ConnectionPool) Close(ctx context.Context) error {
	return nil
}
