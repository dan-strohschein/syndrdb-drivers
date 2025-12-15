//go:build !wasm
// +build !wasm

package tcp

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/protocol"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/transport"
)

// TCPTransportOptions configures the TCP transport
type TCPTransportOptions struct {
	// Address is the server address (host:port)
	Address string

	// Timeout for operations
	Timeout time.Duration

	// TLS configuration
	UseTLS     bool
	CertPath   string
	KeyPath    string
	SkipVerify bool

	// Pool configuration
	PoolSize        int
	PoolMinSize     int
	PoolIdleTimeout time.Duration

	// Health check interval
	HealthCheckInterval time.Duration
}

// TCPTransport implements transport.Transport for native TCP connections
type TCPTransport struct {
	opts    TCPTransportOptions
	codec   protocol.Codec
	pool    *connectionPool
	metrics transportMetrics
	mu      sync.RWMutex
}

// transportMetrics tracks transport performance
type transportMetrics struct {
	totalRequests      atomic.Int64
	totalErrors        atomic.Int64
	bytesSent          atomic.Int64
	bytesReceived      atomic.Int64
	connectionsCreated atomic.Int64
	healthChecksPassed atomic.Int64
	healthChecksFailed atomic.Int64
	lastError          error
	lastErrorTime      time.Time
	latencySum         atomic.Int64 // nanoseconds
	mu                 sync.RWMutex
}

// NewTCPTransport creates a new TCP transport with connection pooling
func NewTCPTransport(opts TCPTransportOptions) (transport.Transport, error) {
	if opts.Address == "" {
		return nil, fmt.Errorf("address is required")
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.PoolSize == 0 {
		opts.PoolSize = 10
	}
	if opts.PoolMinSize == 0 {
		opts.PoolMinSize = 2
	}
	if opts.PoolIdleTimeout == 0 {
		opts.PoolIdleTimeout = 5 * time.Minute
	}
	if opts.HealthCheckInterval == 0 {
		opts.HealthCheckInterval = 30 * time.Second
	}

	t := &TCPTransport{
		opts:  opts,
		codec: protocol.NewCodec(),
	}

	// Create connection factory
	factory := func(ctx context.Context) (*tcpConnection, error) {
		return t.createConnection(ctx)
	}

	// Initialize pool
	t.pool = newConnectionPool(factory, opts.PoolMinSize, opts.PoolSize, opts.PoolIdleTimeout, opts.HealthCheckInterval)

	// Initialize pool with minimum connections
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	if err := t.pool.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize connection pool: %w", err)
	}

	return t, nil
}

// Send implements transport.Transport
func (t *TCPTransport) Send(ctx context.Context, data []byte) error {
	start := time.Now()
	t.metrics.totalRequests.Add(1)

	conn, err := t.pool.Get(ctx)
	if err != nil {
		t.recordError(err)
		return err
	}

	// Send data
	if err := conn.write(ctx, data); err != nil {
		// Connection failed, don't return to pool
		conn.close()
		t.recordError(err)
		return err
	}

	t.metrics.bytesSent.Add(int64(len(data)))
	t.recordLatency(time.Since(start))

	// Return connection to pool
	t.pool.Put(conn)
	return nil
}

// Receive implements transport.Transport
func (t *TCPTransport) Receive(ctx context.Context) ([]byte, error) {
	start := time.Now()

	conn, err := t.pool.Get(ctx)
	if err != nil {
		t.recordError(err)
		return nil, err
	}

	// Read data
	data, err := conn.read(ctx)
	if err != nil {
		// Connection failed, don't return to pool
		conn.close()
		t.recordError(err)
		return nil, err
	}

	t.metrics.bytesReceived.Add(int64(len(data)))
	t.recordLatency(time.Since(start))

	// Return connection to pool
	t.pool.Put(conn)
	return data, nil
}

// Close implements transport.Transport
func (t *TCPTransport) Close() error {
	return t.pool.Close()
}

// IsHealthy implements transport.Transport
func (t *TCPTransport) IsHealthy() bool {
	return !t.pool.closed && t.pool.stats.totalConnections.Load() > 0
}

// GetQueueDepth implements transport.Transport
func (t *TCPTransport) GetQueueDepth() int {
	// TCP transport doesn't have a message queue
	return 0
}

// GetMetrics implements transport.Transport
func (t *TCPTransport) GetMetrics() transport.TransportMetrics {
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
		ConnectionsCreated: t.metrics.connectionsCreated.Load(),
		ConnectionsActive:  int(t.pool.stats.activeConnections.Load()),
		QueueDepth:         0,
		HealthChecksPassed: t.metrics.healthChecksPassed.Load(),
		HealthChecksFailed: t.metrics.healthChecksFailed.Load(),
	}
}

// createConnection creates a new TCP connection with optional TLS
func (t *TCPTransport) createConnection(ctx context.Context) (*tcpConnection, error) {
	t.metrics.connectionsCreated.Add(1)

	// Create TCP connection with timeout
	conn, err := net.DialTimeout("tcp", t.opts.Address, t.opts.Timeout)
	if err != nil {
		return nil, protocol.ConnectionError(fmt.Sprintf("failed to connect to %s", t.opts.Address), map[string]interface{}{
			"address": t.opts.Address,
			"timeout": t.opts.Timeout.String(),
		})
	}

	// Upgrade to TLS if enabled
	if t.opts.UseTLS {
		tlsConfig, err := t.buildTLSConfig()
		if err != nil {
			conn.Close()
			return nil, err
		}

		tlsConn := tls.Client(conn, tlsConfig)

		// Perform TLS handshake
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			tlsConn.Close()
			return nil, protocol.ConnectionError("TLS handshake failed", map[string]interface{}{
				"error": err.Error(),
			})
		}

		conn = tlsConn
	}

	scanner := bufio.NewScanner(conn)
	// Set custom split function to read until EOT
	scanner.Split(splitAtEOT)

	return &tcpConnection{
		conn:         conn,
		scanner:      scanner,
		lastActivity: time.Now(),
		alive:        true,
	}, nil
}

// buildTLSConfig creates a TLS configuration
func (t *TCPTransport) buildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: t.opts.SkipVerify,
	}

	// Extract server name from address
	serverName := t.opts.Address
	if idx := strings.Index(t.opts.Address, ":"); idx >= 0 {
		serverName = t.opts.Address[:idx]
	}
	tlsConfig.ServerName = serverName

	// Load client certificate if provided
	if t.opts.CertPath != "" && t.opts.KeyPath != "" {
		cert, err := tls.LoadX509KeyPair(t.opts.CertPath, t.opts.KeyPath)
		if err != nil {
			return nil, protocol.ConnectionError("failed to load TLS certificate", map[string]interface{}{
				"certPath": t.opts.CertPath,
				"keyPath":  t.opts.KeyPath,
				"error":    err.Error(),
			})
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// recordError records an error in metrics
func (t *TCPTransport) recordError(err error) {
	t.metrics.totalErrors.Add(1)
	t.metrics.mu.Lock()
	t.metrics.lastError = err
	t.metrics.lastErrorTime = time.Now()
	t.metrics.mu.Unlock()
}

// recordLatency records latency in metrics
func (t *TCPTransport) recordLatency(latency time.Duration) {
	t.metrics.latencySum.Add(int64(latency))
}

// splitAtEOT is a custom scanner split function that splits on EOT (0x04)
func splitAtEOT(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for EOT delimiter
	if i := indexOf(data, protocol.EOT); i >= 0 {
		// Return everything up to and including EOT
		return i + 1, data[0:i], nil
	}

	// If at EOF, return all remaining data
	if atEOF {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}

// indexOf finds the first occurrence of byte b in slice s
func indexOf(s []byte, b byte) int {
	for i, v := range s {
		if v == b {
			return i
		}
	}
	return -1
}

// tcpConnection represents a single TCP connection
type tcpConnection struct {
	conn         net.Conn
	scanner      *bufio.Scanner
	lastActivity time.Time
	alive        bool
	mu           sync.RWMutex
}

// write sends data to the connection
func (c *tcpConnection) write(ctx context.Context, data []byte) error {
	if deadline, ok := ctx.Deadline(); ok {
		if err := c.conn.SetDeadline(deadline); err != nil {
			return err
		}
	}

	_, err := c.conn.Write(data)
	if err != nil {
		c.markDead()
		return err
	}

	c.updateActivity()
	return nil
}

// read reads data from the connection
func (c *tcpConnection) read(ctx context.Context) ([]byte, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if err := c.conn.SetDeadline(deadline); err != nil {
			return nil, err
		}
	}

	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			c.markDead()
			return nil, err
		}
		c.markDead()
		return nil, fmt.Errorf("no data received")
	}

	data := c.scanner.Bytes()
	c.updateActivity()

	// Return a copy since scanner reuses the buffer
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// close closes the connection
func (c *tcpConnection) close() error {
	c.mu.Lock()
	c.alive = false
	c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// isAlive checks if the connection is alive
func (c *tcpConnection) isAlive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.alive
}

// lastActivityTime returns the last activity time
func (c *tcpConnection) lastActivityTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastActivity
}

// updateActivity updates the last activity timestamp
func (c *tcpConnection) updateActivity() {
	c.mu.Lock()
	c.lastActivity = time.Now()
	c.mu.Unlock()
}

// markDead marks the connection as dead
func (c *tcpConnection) markDead() {
	c.mu.Lock()
	c.alive = false
	c.mu.Unlock()
}

// ping sends a ping to verify connection health
func (c *tcpConnection) ping(ctx context.Context, codec protocol.Codec) error {
	// Send a simple protocol version check
	data := codec.EncodeVersionHandshake()
	if err := c.write(ctx, data); err != nil {
		return err
	}

	// Read response
	respData, err := c.read(ctx)
	if err != nil {
		return err
	}

	// Decode version response
	return codec.DecodeVersionResponse(respData)
}
