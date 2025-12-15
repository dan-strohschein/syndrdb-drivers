package mock

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/protocol"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/transport"
)

// MockTransport implements transport.Transport for testing
type MockTransport struct {
	// Behavior configuration
	sendErr     error
	receiveErr  error
	receiveData []byte
	healthy     bool
	queueDepth  int

	// Call tracking
	sendCalls    atomic.Int32
	receiveCalls atomic.Int32
	closeCalls   atomic.Int32

	// Metrics
	metrics     mockMetrics
	mu          sync.RWMutex
	closed      bool
	sendDelay   time.Duration
	recvDelay   time.Duration
	sendHistory [][]byte
	recvHistory [][]byte
}

type mockMetrics struct {
	totalRequests      atomic.Int64
	totalErrors        atomic.Int64
	bytesSent          atomic.Int64
	bytesReceived      atomic.Int64
	healthChecksPassed atomic.Int64
	healthChecksFailed atomic.Int64
	latencySum         atomic.Int64
}

// NewMockTransport creates a new mock transport
func NewMockTransport() *MockTransport {
	return &MockTransport{
		healthy:     true,
		sendHistory: make([][]byte, 0),
		recvHistory: make([][]byte, 0),
	}
}

// WithSendError configures the transport to return an error on Send
func (m *MockTransport) WithSendError(err error) *MockTransport {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendErr = err
	return m
}

// WithReceiveError configures the transport to return an error on Receive
func (m *MockTransport) WithReceiveError(err error) *MockTransport {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receiveErr = err
	return m
}

// WithReceiveData configures the data to return on Receive
func (m *MockTransport) WithReceiveData(data []byte) *MockTransport {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receiveData = data
	return m
}

// WithHealthy configures the health status
func (m *MockTransport) WithHealthy(healthy bool) *MockTransport {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
	return m
}

// WithQueueDepth configures the queue depth
func (m *MockTransport) WithQueueDepth(depth int) *MockTransport {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queueDepth = depth
	return m
}

// WithSendDelay adds a delay to Send operations
func (m *MockTransport) WithSendDelay(delay time.Duration) *MockTransport {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendDelay = delay
	return m
}

// WithReceiveDelay adds a delay to Receive operations
func (m *MockTransport) WithReceiveDelay(delay time.Duration) *MockTransport {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recvDelay = delay
	return m
}

// Send implements transport.Transport
func (m *MockTransport) Send(ctx context.Context, data []byte) error {
	m.sendCalls.Add(1)
	m.metrics.totalRequests.Add(1)

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("transport is closed")
	}

	// Apply delay if configured
	delay := m.sendDelay
	sendErr := m.sendErr
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	if sendErr != nil {
		m.metrics.totalErrors.Add(1)
		return sendErr
	}

	// Record send
	m.mu.Lock()
	m.sendHistory = append(m.sendHistory, data)
	m.mu.Unlock()

	m.metrics.bytesSent.Add(int64(len(data)))
	return nil
}

// Receive implements transport.Transport
func (m *MockTransport) Receive(ctx context.Context) ([]byte, error) {
	m.receiveCalls.Add(1)

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, fmt.Errorf("transport is closed")
	}

	// Apply delay if configured
	delay := m.recvDelay
	receiveErr := m.receiveErr
	receiveData := m.receiveData
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	if receiveErr != nil {
		m.metrics.totalErrors.Add(1)
		return nil, receiveErr
	}

	if receiveData == nil {
		return nil, protocol.TimeoutError("no data available", nil)
	}

	// Record receive
	m.mu.Lock()
	m.recvHistory = append(m.recvHistory, receiveData)
	m.mu.Unlock()

	m.metrics.bytesReceived.Add(int64(len(receiveData)))
	return receiveData, nil
}

// Close implements transport.Transport
func (m *MockTransport) Close() error {
	m.closeCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// IsHealthy implements transport.Transport
func (m *MockTransport) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.healthy {
		m.metrics.healthChecksPassed.Add(1)
	} else {
		m.metrics.healthChecksFailed.Add(1)
	}

	return m.healthy
}

// GetQueueDepth implements transport.Transport
func (m *MockTransport) GetQueueDepth() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.queueDepth
}

// GetMetrics implements transport.Transport
func (m *MockTransport) GetMetrics() transport.TransportMetrics {
	totalReqs := m.metrics.totalRequests.Load()
	avgLatency := time.Duration(0)
	if totalReqs > 0 {
		avgLatency = time.Duration(m.metrics.latencySum.Load() / totalReqs)
	}

	return transport.TransportMetrics{
		TotalRequests:      totalReqs,
		TotalErrors:        m.metrics.totalErrors.Load(),
		AverageLatency:     avgLatency,
		BytesSent:          m.metrics.bytesSent.Load(),
		BytesReceived:      m.metrics.bytesReceived.Load(),
		QueueDepth:         m.queueDepth,
		HealthChecksPassed: m.metrics.healthChecksPassed.Load(),
		HealthChecksFailed: m.metrics.healthChecksFailed.Load(),
	}
}

// GetSendCallCount returns the number of times Send was called
func (m *MockTransport) GetSendCallCount() int {
	return int(m.sendCalls.Load())
}

// GetReceiveCallCount returns the number of times Receive was called
func (m *MockTransport) GetReceiveCallCount() int {
	return int(m.receiveCalls.Load())
}

// GetCloseCallCount returns the number of times Close was called
func (m *MockTransport) GetCloseCallCount() int {
	return int(m.closeCalls.Load())
}

// GetSendHistory returns all data sent through this transport
func (m *MockTransport) GetSendHistory() [][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	history := make([][]byte, len(m.sendHistory))
	copy(history, m.sendHistory)
	return history
}

// GetReceiveHistory returns all data received through this transport
func (m *MockTransport) GetReceiveHistory() [][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	history := make([][]byte, len(m.recvHistory))
	copy(history, m.recvHistory)
	return history
}

// Reset clears all state and call counts
func (m *MockTransport) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sendErr = nil
	m.receiveErr = nil
	m.receiveData = nil
	m.healthy = true
	m.queueDepth = 0
	m.closed = false
	m.sendDelay = 0
	m.recvDelay = 0

	m.sendCalls.Store(0)
	m.receiveCalls.Store(0)
	m.closeCalls.Store(0)

	m.metrics.totalRequests.Store(0)
	m.metrics.totalErrors.Store(0)
	m.metrics.bytesSent.Store(0)
	m.metrics.bytesReceived.Store(0)
	m.metrics.healthChecksPassed.Store(0)
	m.metrics.healthChecksFailed.Store(0)
	m.metrics.latencySum.Store(0)

	m.sendHistory = make([][]byte, 0)
	m.recvHistory = make([][]byte, 0)
}

// IsClosed returns whether the transport has been closed
func (m *MockTransport) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}
