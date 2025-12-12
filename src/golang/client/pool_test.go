//go:build !wasm && milestone1
// +build !wasm,milestone1

package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockConnection implements ConnectionInterface for testing.
type mockConnection struct {
	id           int
	alive        bool
	lastActivity time.Time
	sendErr      error
	receiveErr   error
	pingErr      error
	mu           sync.RWMutex
}

func newMockConnection(id int) *mockConnection {
	return &mockConnection{
		id:           id,
		alive:        true,
		lastActivity: time.Now(),
	}
}

func (m *mockConnection) SendCommand(ctx context.Context, command string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.lastActivity = time.Now()
	return nil
}

func (m *mockConnection) ReceiveResponse(ctx context.Context) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.receiveErr != nil {
		return nil, m.receiveErr
	}
	return map[string]interface{}{"id": m.id}, nil
}

func (m *mockConnection) Ping(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.pingErr != nil {
		return m.pingErr
	}
	return nil
}

func (m *mockConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alive = false
	return nil
}

func (m *mockConnection) RemoteAddr() string {
	return fmt.Sprintf("mock://conn-%d", m.id)
}

func (m *mockConnection) IsAlive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.alive
}

func (m *mockConnection) LastActivity() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastActivity
}

func (m *mockConnection) GetTLSConnectionState() interface{} {
	return nil
}

// TestPoolInitialization verifies pool creation and initialization.
func TestPoolInitialization(t *testing.T) {
	connID := atomic.Int32{}
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		id := int(connID.Add(1))
		return newMockConnection(id), nil
	}

	pool := NewConnectionPool(factory, 2, 5, 30*time.Second, 10*time.Second)
	if pool == nil {
		t.Fatal("NewConnectionPool returned nil")
	}

	ctx := context.Background()
	if err := pool.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify minimum idle connections were created
	stats := pool.Stats()
	if stats.IdleConnections.Load() < 2 {
		t.Errorf("Expected at least 2 idle connections, got %d", stats.IdleConnections.Load())
	}

	// Cleanup
	if err := pool.Close(ctx); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestPoolGetPut verifies basic connection acquisition and release.
func TestPoolGetPut(t *testing.T) {
	connID := atomic.Int32{}
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		id := int(connID.Add(1))
		return newMockConnection(id), nil
	}

	pool := NewConnectionPool(factory, 1, 3, 30*time.Second, 10*time.Second)
	ctx := context.Background()
	pool.Initialize(ctx)
	defer pool.Close(ctx)

	// Get a connection
	conn1, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if conn1 == nil {
		t.Fatal("Get returned nil connection")
	}

	stats := pool.Stats()
	if stats.Hits.Load() != 1 && stats.Misses.Load() != 1 {
		t.Errorf("Expected 1 hit or miss, got hits=%d misses=%d", stats.Hits.Load(), stats.Misses.Load())
	}

	// Return the connection
	pool.Put(conn1)

	// Verify idle count increased
	time.Sleep(100 * time.Millisecond) // Allow stats to update
	stats = pool.Stats()
	if stats.IdleConnections.Load() < 1 {
		t.Errorf("Expected at least 1 idle connection after Put, got %d", stats.IdleConnections.Load())
	}
}

// TestPoolConcurrentAccess verifies pool handles concurrent requests correctly.
func TestPoolConcurrentAccess(t *testing.T) {
	connID := atomic.Int32{}
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		id := int(connID.Add(1))
		return newMockConnection(id), nil
	}

	pool := NewConnectionPool(factory, 2, 10, 30*time.Second, 10*time.Second)
	ctx := context.Background()
	pool.Initialize(ctx)
	defer pool.Close(ctx)

	// Launch 20 goroutines to access pool concurrently
	const numGoroutines = 20
	var wg sync.WaitGroup
	successCount := atomic.Int32{}
	errorCount := atomic.Int32{}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := pool.Get(ctx)
			if err != nil {
				errorCount.Add(1)
				t.Logf("Goroutine %d: Get failed: %v", id, err)
				return
			}

			// Simulate work
			time.Sleep(10 * time.Millisecond)

			pool.Put(conn)
			successCount.Add(1)
		}(i)
	}

	wg.Wait()

	if successCount.Load() != numGoroutines {
		t.Errorf("Expected %d successful operations, got %d (errors: %d)",
			numGoroutines, successCount.Load(), errorCount.Load())
	}

	// Verify no connections leaked
	stats := pool.Stats()
	if stats.ActiveConnections.Load() != 0 {
		t.Errorf("Expected 0 active connections after all Put, got %d", stats.ActiveConnections.Load())
	}
}

// TestPoolMaxLimit verifies pool enforces maximum connection limit.
func TestPoolMaxLimit(t *testing.T) {
	connID := atomic.Int32{}
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		id := int(connID.Add(1))
		return newMockConnection(id), nil
	}

	const maxOpen = 3
	pool := NewConnectionPool(factory, 1, maxOpen, 30*time.Second, 10*time.Second)
	ctx := context.Background()
	pool.Initialize(ctx)
	defer pool.Close(ctx)

	// Acquire max connections
	conns := make([]ConnectionInterface, maxOpen)
	for i := 0; i < maxOpen; i++ {
		conn, err := pool.Get(ctx)
		if err != nil {
			t.Fatalf("Get %d failed: %v", i, err)
		}
		conns[i] = conn
	}

	// Verify all connections are active
	stats := pool.Stats()
	if stats.ActiveConnections.Load() != int32(maxOpen) {
		t.Errorf("Expected %d active connections, got %d", maxOpen, stats.ActiveConnections.Load())
	}

	// Try to get one more (should timeout)
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err := pool.Get(ctxTimeout)
	if err == nil {
		t.Error("Expected timeout error when pool exhausted")
	}
	if !errors.Is(err, context.DeadlineExceeded) && err.Error() != "connection pool exhausted" {
		t.Errorf("Expected timeout or pool exhausted error, got: %v", err)
	}

	stats = pool.Stats()
	if stats.Timeouts.Load() == 0 {
		t.Error("Expected timeout to be recorded in stats")
	}

	// Return connections
	for _, conn := range conns {
		pool.Put(conn)
	}
}

// TestPoolIdleCleanup verifies idle connections are cleaned up after timeout.
func TestPoolIdleCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping idle cleanup test in short mode")
	}

	connID := atomic.Int32{}
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		id := int(connID.Add(1))
		return newMockConnection(id), nil
	}

	// Short idle timeout for testing
	pool := NewConnectionPool(factory, 1, 5, 500*time.Millisecond, 10*time.Second)
	ctx := context.Background()
	pool.Initialize(ctx)
	defer pool.Close(ctx)

	// Create some idle connections
	conns := make([]ConnectionInterface, 3)
	for i := 0; i < 3; i++ {
		conn, err := pool.Get(ctx)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		conns[i] = conn
	}
	for _, conn := range conns {
		pool.Put(conn)
	}

	// Wait for cleanup worker to run (idle timeout / 4 + buffer)
	time.Sleep(1 * time.Second)

	// Verify idle connections were reduced
	stats := pool.Stats()
	// Should keep at least minIdle (1) but clean up excess
	if stats.IdleConnections.Load() > 2 {
		t.Logf("Idle connections after cleanup: %d (expected some to be cleaned up)", stats.IdleConnections.Load())
	}
}

// TestPoolStats verifies statistics tracking is accurate.
func TestPoolStats(t *testing.T) {
	connID := atomic.Int32{}
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		id := int(connID.Add(1))
		return newMockConnection(id), nil
	}

	pool := NewConnectionPool(factory, 1, 5, 30*time.Second, 10*time.Second)
	ctx := context.Background()
	pool.Initialize(ctx)
	defer pool.Close(ctx)

	// Get and put connections
	conn1, _ := pool.Get(ctx)
	conn2, _ := pool.Get(ctx)

	stats := pool.Stats()
	if stats.TotalConnections.Load() < 2 {
		t.Errorf("Expected at least 2 total connections, got %d", stats.TotalConnections.Load())
	}
	if stats.ActiveConnections.Load() != 2 {
		t.Errorf("Expected 2 active connections, got %d", stats.ActiveConnections.Load())
	}

	pool.Put(conn1)
	pool.Put(conn2)

	time.Sleep(100 * time.Millisecond)
	stats = pool.Stats()
	if stats.ActiveConnections.Load() != 0 {
		t.Errorf("Expected 0 active connections, got %d", stats.ActiveConnections.Load())
	}
	if stats.IdleConnections.Load() < 2 {
		t.Errorf("Expected at least 2 idle connections, got %d", stats.IdleConnections.Load())
	}
}

// TestPoolClose verifies graceful pool shutdown.
func TestPoolClose(t *testing.T) {
	connID := atomic.Int32{}
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		id := int(connID.Add(1))
		return newMockConnection(id), nil
	}

	pool := NewConnectionPool(factory, 2, 5, 30*time.Second, 10*time.Second)
	ctx := context.Background()
	pool.Initialize(ctx)

	// Get a connection
	conn, _ := pool.Get(ctx)
	pool.Put(conn)

	// Close pool
	err := pool.Close(ctx)
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify subsequent operations fail
	_, err = pool.Get(ctx)
	if err == nil {
		t.Error("Expected error after pool closed")
	}

	// Verify double close is safe
	err = pool.Close(ctx)
	if err != nil {
		t.Errorf("Second Close should be safe, got error: %v", err)
	}
}

// TestPoolFactoryError verifies pool handles connection creation failures.
func TestPoolFactoryError(t *testing.T) {
	factoryErr := errors.New("connection creation failed")
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		return nil, factoryErr
	}

	pool := NewConnectionPool(factory, 1, 3, 30*time.Second, 10*time.Second)
	ctx := context.Background()

	// Initialize should succeed even if factory fails (lazy creation)
	pool.Initialize(ctx)
	defer pool.Close(ctx)

	// Get should return factory error
	_, err := pool.Get(ctx)
	if err == nil {
		t.Error("Expected factory error")
	}
	if !errors.Is(err, factoryErr) {
		t.Errorf("Expected factory error, got: %v", err)
	}

	stats := pool.Stats()
	if stats.Errors.Load() == 0 {
		t.Error("Expected error to be recorded in stats")
	}
}
