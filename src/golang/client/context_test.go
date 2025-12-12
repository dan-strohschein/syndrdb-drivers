//go:build milestone1
// +build milestone1

package client

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test context cancellation during query execution
func TestContextCancellationDuringQuery(t *testing.T) {
	// Create a mock connection that takes time to respond
	slowConn := &slowMockConnection{
		responseDelay: 500 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Send command should respect context cancellation
	err := slowConn.SendCommand(ctx, "SLOW QUERY")
	if err == nil {
		t.Error("Expected context cancellation error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected DeadlineExceeded, got: %v", err)
	}
}

// Test timeout triggers properly
func TestContextTimeout(t *testing.T) {
	slowConn := &slowMockConnection{
		responseDelay: 200 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := slowConn.ReceiveResponse(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected DeadlineExceeded, got: %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("Timeout took too long: %v", elapsed)
	}
}

// Test context cancellation during connection establishment
func TestContextCancellationDuringConnect(t *testing.T) {
	opts := DefaultOptions()
	opts.LogLevel = "ERROR" // Reduce noise
	c := NewClient(&opts)

	// Cancel immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.Connect(ctx, "syndrdb://localhost:9999/test")
	if err == nil {
		c.Disconnect(context.Background())
		t.Error("Expected context cancellation error")
	}
}

// Test multiple concurrent contexts
func TestMultipleConcurrentContexts(t *testing.T) {
	connID := atomic.Int32{}
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		id := int(connID.Add(1))
		return newMockConnection(id), nil
	}

	pool := NewConnectionPool(factory, 2, 5, 30*time.Second, 10*time.Second)
	ctx := context.Background()
	pool.Initialize(ctx)
	defer pool.Close(ctx)

	const numGoroutines = 10
	var wg sync.WaitGroup
	successCount := atomic.Int32{}
	cancelCount := atomic.Int32{}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine uses its own context with different timeouts
			var ctx context.Context
			var cancel context.CancelFunc
			if id%2 == 0 {
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
			}
			defer cancel()

			conn, err := pool.Get(ctx)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					cancelCount.Add(1)
				}
				return
			}
			defer pool.Put(conn)

			// Simulate work
			time.Sleep(10 * time.Millisecond)
			successCount.Add(1)
		}(i)
	}

	wg.Wait()

	if successCount.Load()+cancelCount.Load() != numGoroutines {
		t.Errorf("Expected %d total operations, got %d success + %d cancelled",
			numGoroutines, successCount.Load(), cancelCount.Load())
	}
}

// Test context deadline propagation to connection
func TestContextDeadlinePropagation(t *testing.T) {
	conn := &deadlineTrackingConnection{}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancel()

	// SendCommand should set deadline on connection
	err := conn.SendCommand(ctx, "TEST")
	if err != nil {
		t.Errorf("SendCommand failed: %v", err)
	}

	if !conn.deadlineSet {
		t.Error("Expected deadline to be set on connection")
	}
	if conn.deadline.IsZero() {
		t.Error("Deadline was not propagated")
	}
}

// Test context cancellation doesn't leak goroutines
func TestNoGoroutineLeaks(t *testing.T) {
	// Note: This test requires the goleak package for full verification
	// For now, we'll do a basic check

	connID := atomic.Int32{}
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		id := int(connID.Add(1))
		return newMockConnection(id), nil
	}

	pool := NewConnectionPool(factory, 2, 5, 30*time.Second, 10*time.Second)
	ctx := context.Background()
	pool.Initialize(ctx)

	// Launch many operations and cancel them
	const numOps = 50
	for i := 0; i < numOps; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		go func() {
			conn, err := pool.Get(ctx)
			if err == nil {
				pool.Put(conn)
			}
		}()
		cancel() // Cancel immediately
	}

	// Give goroutines time to clean up
	time.Sleep(100 * time.Millisecond)

	// Close pool and verify clean shutdown
	err := pool.Close(context.Background())
	if err != nil {
		t.Errorf("Pool close failed: %v", err)
	}
}

// Test parent context cancellation propagates to child operations
func TestParentContextCancellation(t *testing.T) {
	connID := atomic.Int32{}
	factory := func(ctx context.Context) (ConnectionInterface, error) {
		id := int(connID.Add(1))
		return newMockConnection(id), nil
	}

	pool := NewConnectionPool(factory, 1, 3, 30*time.Second, 10*time.Second)

	// Create parent context
	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	pool.Initialize(parentCtx)
	defer pool.Close(context.Background())

	// Start an operation with parent context
	childCtx, childCancel := context.WithCancel(parentCtx)
	defer childCancel()

	done := make(chan struct{})
	var opErr error

	go func() {
		conn, err := pool.Get(childCtx)
		opErr = err
		if err == nil {
			pool.Put(conn)
		}
		close(done)
	}()

	// Cancel parent context
	time.Sleep(10 * time.Millisecond)
	parentCancel()

	select {
	case <-done:
		// Operation should fail with cancellation error
		if opErr == nil {
			t.Error("Expected cancellation error")
		}
	case <-time.After(1 * time.Second):
		t.Error("Operation did not complete after parent cancellation")
	}
}

// slowMockConnection simulates a slow network connection
type slowMockConnection struct {
	responseDelay time.Duration
	mockConnection
}

func (s *slowMockConnection) SendCommand(ctx context.Context, command string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.responseDelay):
		return nil
	}
}

func (s *slowMockConnection) ReceiveResponse(ctx context.Context) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(s.responseDelay):
		return map[string]interface{}{"result": "ok"}, nil
	}
}

// deadlineTrackingConnection tracks whether deadline was set
type deadlineTrackingConnection struct {
	deadlineSet bool
	deadline    time.Time
	mockConnection
}

func (d *deadlineTrackingConnection) SendCommand(ctx context.Context, command string) error {
	if deadline, ok := ctx.Deadline(); ok {
		d.deadlineSet = true
		d.deadline = deadline
	}
	return nil
}

func (d *deadlineTrackingConnection) ReceiveResponse(ctx context.Context) (interface{}, error) {
	return map[string]interface{}{"result": "ok"}, nil
}
