//go:build js && wasm
// +build js,wasm

package wasm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"syscall/js"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/protocol"
)

// VirtualPool manages virtual connections that delegate to Node.js TCP bridge
// The actual connection pooling happens on the Node.js side
type VirtualPool struct {
	bridge       js.Value
	connections  sync.Map // connId -> *virtualConnection
	nextConnId   atomic.Int64
	poolSize     int
	mu           sync.RWMutex
	closed       bool
	activeConns  atomic.Int32
	totalCreated atomic.Int64
	totalReused  atomic.Int64
}

// virtualConnection represents a virtual connection to Node.js bridge
type virtualConnection struct {
	id       string
	bridge   js.Value
	inUse    atomic.Bool
	lastUsed atomic.Int64 // Unix timestamp
	mu       sync.Mutex
}

// NewVirtualPool creates a new virtual connection pool
func NewVirtualPool(bridge js.Value, poolSize int) *VirtualPool {
	return &VirtualPool{
		bridge:   bridge,
		poolSize: poolSize,
	}
}

// Get acquires a virtual connection from the pool
func (p *VirtualPool) Get(ctx context.Context) (*virtualConnection, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	// Try to find an idle connection
	var idleConn *virtualConnection
	p.connections.Range(func(key, value interface{}) bool {
		vc := value.(*virtualConnection)
		if !vc.inUse.Load() {
			if vc.inUse.CompareAndSwap(false, true) {
				idleConn = vc
				p.totalReused.Add(1)
				return false // Stop iteration
			}
		}
		return true
	})

	if idleConn != nil {
		return idleConn, nil
	}

	// Check if we can create a new connection
	activeCount := int(p.activeConns.Load())
	if activeCount >= p.poolSize {
		// Pool is at capacity, wait for a connection to become available
		// In a real implementation, this would block with context cancellation support
		return nil, protocol.BackpressureError(activeCount)
	}

	// Create a new virtual connection
	return p.createConnection()
}

// Put returns a virtual connection to the pool
func (p *VirtualPool) Put(conn *virtualConnection) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return p.closeConnection(conn)
	}

	// Mark as not in use
	conn.inUse.Store(false)
	return nil
}

// Close closes all virtual connections and the pool
func (p *VirtualPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Close all connections
	var errs []error
	p.connections.Range(func(key, value interface{}) bool {
		vc := value.(*virtualConnection)
		if err := p.closeConnection(vc); err != nil {
			errs = append(errs, err)
		}
		return true
	})

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// Stats returns pool statistics
func (p *VirtualPool) Stats() map[string]interface{} {
	var idleCount, activeCount int
	p.connections.Range(func(key, value interface{}) bool {
		vc := value.(*virtualConnection)
		if vc.inUse.Load() {
			activeCount++
		} else {
			idleCount++
		}
		return true
	})

	return map[string]interface{}{
		"poolSize":     p.poolSize,
		"activeConns":  activeCount,
		"idleConns":    idleCount,
		"totalCreated": p.totalCreated.Load(),
		"totalReused":  p.totalReused.Load(),
	}
}

// createConnection creates a new virtual connection via the bridge
func (p *VirtualPool) createConnection() (*virtualConnection, error) {
	// Generate unique connection ID
	connId := fmt.Sprintf("vc-%d", p.nextConnId.Add(1))

	// Request connection from Node.js bridge
	result := p.bridge.Call("goRequestConnection", connId)

	// Check for errors
	if result.Type() == js.TypeObject && !result.Get("code").IsUndefined() {
		code := result.Get("code").Int()
		message := result.Get("message").String()
		return nil, &protocol.TransportError{
			Code:        protocol.ErrorCode(code),
			Message:     message,
			IsRetryable: false,
		}
	}

	// Create virtual connection
	vc := &virtualConnection{
		id:     connId,
		bridge: p.bridge,
	}
	vc.inUse.Store(true)

	// Store in map
	p.connections.Store(connId, vc)
	p.activeConns.Add(1)
	p.totalCreated.Add(1)

	return vc, nil
}

// closeConnection closes a virtual connection
func (p *VirtualPool) closeConnection(conn *virtualConnection) error {
	// Remove from map
	p.connections.Delete(conn.id)
	p.activeConns.Add(-1)

	// Notify Node.js bridge to release the connection
	result := p.bridge.Call("goReleaseConnection", conn.id)

	// Check for errors
	if result.Type() == js.TypeObject && !result.Get("code").IsUndefined() {
		code := result.Get("code").Int()
		message := result.Get("message").String()
		return &protocol.TransportError{
			Code:    protocol.ErrorCode(code),
			Message: message,
		}
	}

	return nil
}

// Send sends data through a virtual connection
func (vc *virtualConnection) Send(data []byte) error {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Copy bytes to JavaScript
	jsData := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(jsData, data)

	// Call bridge send function with connection ID
	result := vc.bridge.Call("goSend", vc.id, jsData)

	// Check for errors
	if result.Type() == js.TypeObject && !result.Get("code").IsUndefined() {
		code := result.Get("code").Int()
		message := result.Get("message").String()
		return &protocol.TransportError{
			Code:        protocol.ErrorCode(code),
			Message:     message,
			IsRetryable: protocol.ErrorCode(code) == protocol.ErrorCodeBridgeBusy,
		}
	}

	return nil
}

// Receive receives data from a virtual connection
func (vc *virtualConnection) Receive() ([]byte, error) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Call bridge receive function with connection ID
	result := vc.bridge.Call("goReceive", vc.id)

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

	return data, nil
}

// IsHealthy checks if the virtual connection is healthy
func (vc *virtualConnection) IsHealthy() bool {
	// Query Node.js bridge for connection health
	result := vc.bridge.Call("goCheckConnectionHealth", vc.id)
	if result.Type() == js.TypeBoolean {
		return result.Bool()
	}
	return false
}
