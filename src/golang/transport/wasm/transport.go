//go:build js && wasm
// +build js,wasm

package wasm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"syscall/js"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/protocol"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/transport"
)

// WASMTransportOptions configures the WASM bridge transport
type WASMTransportOptions struct {
	// PoolSize for the Node.js side connection pool
	PoolSize int

	// ProtocolVersion to negotiate
	ProtocolVersion int

	// Node.js TLS options (passed to bridge)
	NodeTLSOptions map[string]interface{}

	// MaxRetries for transient bridge errors
	MaxRetries int

	// RetryBackoff for bridge operations
	RetryBackoff time.Duration
}

// WASMTransport implements transport.Transport using JavaScript bridge
type WASMTransport struct {
	opts       WASMTransportOptions
	codec      protocol.Codec
	bridge     js.Value
	metrics    wasmMetrics
	sendChan   chan *sendRequest
	recvChan   chan *recvResponse
	queueDepth atomic.Int32
	stopCh     chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
	closed     bool
}

// wasmMetrics tracks transport performance
type wasmMetrics struct {
	totalRequests      atomic.Int64
	totalErrors        atomic.Int64
	bytesSent          atomic.Int64
	bytesReceived      atomic.Int64
	bridgeRetries      atomic.Int64
	healthChecksPassed atomic.Int64
	healthChecksFailed atomic.Int64
	lastError          error
	lastErrorTime      time.Time
	latencySum         atomic.Int64
	mu                 sync.RWMutex
}

// sendRequest represents a send operation
type sendRequest struct {
	data     []byte
	resultCh chan error
}

// recvResponse represents a receive operation
type recvResponse struct {
	data     []byte
	err      error
	resultCh chan struct{}
}

// NewWASMTransport creates a new WASM bridge transport
func NewWASMTransport(opts WASMTransportOptions) (transport.Transport, error) {
	if opts.PoolSize == 0 {
		opts.PoolSize = 5
	}
	if opts.ProtocolVersion == 0 {
		opts.ProtocolVersion = protocol.PROTOCOL_VERSION
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}
	if opts.RetryBackoff == 0 {
		opts.RetryBackoff = 10 * time.Millisecond
	}

	// Get bridge from global scope
	bridge := js.Global().Get("SyndrDBBridge")
	if bridge.IsUndefined() {
		return nil, protocol.BridgeInitError("SyndrDBBridge not found on globalThis")
	}

	// Verify required callbacks exist
	requiredCallbacks := []string{"goRequestConnection", "goSend", "goReceive", "goReleaseConnection"}
	for _, callback := range requiredCallbacks {
		if bridge.Get(callback).IsUndefined() {
			return nil, protocol.BridgeCallbackMissingError(callback)
		}
	}

	t := &WASMTransport{
		opts:     opts,
		codec:    protocol.NewCodec(),
		bridge:   bridge,
		sendChan: make(chan *sendRequest, 100), // Bounded channel for backpressure
		recvChan: make(chan *recvResponse, 100),
		stopCh:   make(chan struct{}),
	}

	// Register Go callbacks for Node.js to invoke
	t.registerCallbacks()

	return t, nil
}

// Send implements transport.Transport
func (t *WASMTransport) Send(ctx context.Context, data []byte) error {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return fmt.Errorf("transport is closed")
	}
	t.mu.RUnlock()

	start := time.Now()
	t.metrics.totalRequests.Add(1)

	// Check queue depth for backpressure
	depth := int(t.queueDepth.Load())
	if depth > 80 {
		// Queue is getting full, apply backpressure
		return protocol.BackpressureError(depth)
	}

	// Try to send with retries for transient errors
	var lastErr error
	for attempt := 0; attempt < t.opts.MaxRetries; attempt++ {
		if attempt > 0 {
			t.metrics.bridgeRetries.Add(1)
			// Exponential backoff: 10ms, 100ms, 1s
			backoff := t.opts.RetryBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		err := t.sendViaBridge(ctx, data)
		if err == nil {
			t.metrics.bytesSent.Add(int64(len(data)))
			t.recordLatency(time.Since(start))
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if te, ok := err.(*protocol.TransportError); ok {
			if !te.IsRetryable {
				t.recordError(err)
				return err
			}
		} else {
			// Unknown error type, don't retry
			t.recordError(err)
			return err
		}
	}

	t.recordError(lastErr)
	return fmt.Errorf("send failed after %d retries: %w", t.opts.MaxRetries, lastErr)
}

// Receive implements transport.Transport
func (t *WASMTransport) Receive(ctx context.Context) ([]byte, error) {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return nil, fmt.Errorf("transport is closed")
	}
	t.mu.RUnlock()

	start := time.Now()

	// Try to receive with retries for transient errors
	var lastErr error
	for attempt := 0; attempt < t.opts.MaxRetries; attempt++ {
		if attempt > 0 {
			t.metrics.bridgeRetries.Add(1)
			backoff := t.opts.RetryBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		data, err := t.receiveViaBridge(ctx)
		if err == nil {
			t.metrics.bytesReceived.Add(int64(len(data)))
			t.recordLatency(time.Since(start))
			return data, nil
		}

		lastErr = err

		// Check if error is retryable
		if te, ok := err.(*protocol.TransportError); ok {
			if !te.IsRetryable {
				t.recordError(err)
				return nil, err
			}
		} else {
			t.recordError(err)
			return nil, err
		}
	}

	t.recordError(lastErr)
	return nil, fmt.Errorf("receive failed after %d retries: %w", t.opts.MaxRetries, lastErr)
}

// Close implements transport.Transport
func (t *WASMTransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()

	close(t.stopCh)
	t.wg.Wait()

	return nil
}

// IsHealthy implements transport.Transport
func (t *WASMTransport) IsHealthy() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return !t.closed
}

// GetQueueDepth implements transport.Transport
func (t *WASMTransport) GetQueueDepth() int {
	return int(t.queueDepth.Load())
}

// GetMetrics implements transport.Transport
func (t *WASMTransport) GetMetrics() transport.TransportMetrics {
	t.metrics.mu.RLock()
	lastErr := t.metrics.lastError
	lastErrTime := t.metrics.lastErrorTime
	t.metrics.mu.RUnlock()

	totalReqs := t.metrics.totalRequests.Load()
	avgLatency := time.Duration(0)
	if totalReqs > 0 {
		avgLatency = time.Duration(t.metrics.latencySum.Load() / totalReqs)
	}

	return transport.TransportMetrics{
		TotalRequests:      totalReqs,
		TotalErrors:        t.metrics.totalErrors.Load(),
		AverageLatency:     avgLatency,
		LastError:          lastErr,
		LastErrorTime:      lastErrTime,
		BytesSent:          t.metrics.bytesSent.Load(),
		BytesReceived:      t.metrics.bytesReceived.Load(),
		QueueDepth:         int(t.queueDepth.Load()),
		HealthChecksPassed: t.metrics.healthChecksPassed.Load(),
		HealthChecksFailed: t.metrics.healthChecksFailed.Load(),
	}
}

// sendViaBridge sends data via the JavaScript bridge
func (t *WASMTransport) sendViaBridge(ctx context.Context, data []byte) error {
	// Copy bytes to JavaScript
	jsData := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(jsData, data)

	// Call bridge send function
	result := t.bridge.Call("goSend", jsData)

	// Check for immediate errors
	if result.Type() == js.TypeObject && !result.Get("code").IsUndefined() {
		// Error object returned
		code := result.Get("code").Int()
		message := result.Get("message").String()
		return &protocol.TransportError{
			Code:        protocol.ErrorCode(code),
			Message:     message,
			IsRetryable: protocol.ErrorCode(code) == protocol.ErrorCodeBridgeBusy,
		}
	}

	t.queueDepth.Add(1)
	return nil
}

// receiveViaBridge receives data via the JavaScript bridge
func (t *WASMTransport) receiveViaBridge(ctx context.Context) ([]byte, error) {
	// Call bridge receive function
	result := t.bridge.Call("goReceive")

	// Check for errors
	if result.Type() == js.TypeObject && !result.Get("code").IsUndefined() {
		code := result.Get("code").Int()
		message := result.Get("message").String()
		return nil, &protocol.TransportError{
			Code:        protocol.ErrorCode(code),
			Message:     message,
			IsRetryable: protocol.ErrorCode(code) == protocol.ErrorCodeBridgeBusy,
		}
	}

	// Extract data from Uint8Array
	length := result.Length()
	data := make([]byte, length)
	js.CopyBytesToGo(data, result)

	t.queueDepth.Add(-1)
	return data, nil
}

// registerCallbacks registers Go callbacks for Node.js to invoke
func (t *WASMTransport) registerCallbacks() {
	// Register callback for connection health notifications
	healthCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			connId := args[0].String()
			isHealthy := args[1].Bool()

			if !isHealthy {
				t.metrics.healthChecksFailed.Add(1)
			} else {
				t.metrics.healthChecksPassed.Add(1)
			}

			// Log health status change
			js.Global().Get("console").Call("log",
				fmt.Sprintf("Connection %s health: %v", connId, isHealthy))
		}
		return nil
	})

	t.bridge.Set("goMarkConnectionUnhealthy", healthCallback)
}

// recordError records an error in metrics
func (t *WASMTransport) recordError(err error) {
	t.metrics.totalErrors.Add(1)
	t.metrics.mu.Lock()
	t.metrics.lastError = err
	t.metrics.lastErrorTime = time.Now()
	t.metrics.mu.Unlock()
}

// recordLatency records latency in metrics
func (t *WASMTransport) recordLatency(latency time.Duration) {
	t.metrics.latencySum.Add(int64(latency))
}
