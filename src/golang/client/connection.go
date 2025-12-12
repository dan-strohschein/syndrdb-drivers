package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// ConnectionInterface defines the contract for database connections.
// This abstraction allows for connection pooling and alternative implementations.
type ConnectionInterface interface {
	// SendCommand sends a command to the server with context support.
	SendCommand(ctx context.Context, command string) error

	// ReceiveResponse reads and parses a response from the server.
	ReceiveResponse(ctx context.Context) (interface{}, error)

	// Ping sends a minimal command to verify connection health.
	Ping(ctx context.Context) error

	// Close closes the connection gracefully.
	Close() error

	// RemoteAddr returns the remote server address.
	RemoteAddr() string

	// IsAlive checks if the connection is still valid.
	IsAlive() bool

	// LastActivity returns the timestamp of the last successful operation.
	LastActivity() time.Time
}

// Connection represents a single TCP connection to SyndrDB server.
type Connection struct {
	conn         net.Conn
	scanner      *bufio.Scanner
	remoteAddr   string
	lastActivity time.Time
	mu           sync.RWMutex
	alive        bool
	tlsState     *tls.ConnectionState
}

// NewConnection creates a new connection to the specified address with optional TLS.
func NewConnection(ctx context.Context, address string, opts ClientOptions) (*Connection, error) {
	timeout := time.Duration(opts.DefaultTimeoutMs) * time.Millisecond

	// Create TCP connection with timeout
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, &ConnectionError{
			Code:    "CONNECTION_FAILED",
			Type:    "CONNECTION_ERROR",
			Message: fmt.Sprintf("failed to connect to %s", address),
			Details: map[string]interface{}{
				"address": address,
				"timeout": opts.DefaultTimeoutMs,
			},
			Cause: err,
		}
	}

	// Extract server name from address for TLS
	serverName := address
	if idx := strings.Index(address, ":"); idx >= 0 {
		serverName = address[:idx]
	}

	// Upgrade to TLS if enabled
	if opts.TLSEnabled || opts.TLSConfig != nil {
		tlsConfig, err := buildTLSConfig(opts, serverName)
		if err != nil {
			conn.Close()
			return nil, err
		}

		tlsConn := tls.Client(conn, tlsConfig)

		// Perform TLS handshake with context
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			tlsConn.Close()
			return nil, parseTLSError(err)
		}

		// Validate connection state
		state := tlsConn.ConnectionState()
		if !state.HandshakeComplete {
			tlsConn.Close()
			return nil, &ConnectionError{
				Code:    "TLS_HANDSHAKE_INCOMPLETE",
				Type:    "CONNECTION_ERROR",
				Message: "TLS handshake did not complete",
			}
		}

		conn = tlsConn
		scanner := bufio.NewScanner(conn)

		return &Connection{
			conn:         conn,
			scanner:      scanner,
			remoteAddr:   conn.RemoteAddr().String(),
			lastActivity: time.Now(),
			alive:        true,
			tlsState:     &state,
		}, nil
	}

	// Plain TCP connection
	scanner := bufio.NewScanner(conn)

	return &Connection{
		conn:         conn,
		scanner:      scanner,
		remoteAddr:   conn.RemoteAddr().String(),
		lastActivity: time.Now(),
		alive:        true,
	}, nil
}

// SendCommand sends a command to the server with EOT terminator.
func (c *Connection) SendCommand(ctx context.Context, command string) error {
	// Check context cancellation before operation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Set deadline from context if available
	if deadline, ok := ctx.Deadline(); ok {
		if err := c.conn.SetDeadline(deadline); err != nil {
			return &ProtocolError{
				Code:    "DEADLINE_ERROR",
				Type:    "PROTOCOL_ERROR",
				Message: "failed to set connection deadline",
				Cause:   err,
			}
		}
	}

	// Append EOT terminator
	fullCmd := command + "\x04"
	_, err := c.conn.Write([]byte(fullCmd))
	if err != nil {
		c.markDead()
		return &ProtocolError{
			Code:    "SEND_FAILED",
			Type:    "PROTOCOL_ERROR",
			Message: "failed to send command to server",
			Details: map[string]interface{}{
				"command": command,
			},
			Cause: err,
		}
	}

	c.updateActivity()
	return nil
}

// ReceiveResponse reads and parses a response from the server.
func (c *Connection) ReceiveResponse(ctx context.Context) (interface{}, error) {
	// Check context cancellation before operation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Set deadline from context if available
	if deadline, ok := ctx.Deadline(); ok {
		if err := c.conn.SetDeadline(deadline); err != nil {
			return nil, &ProtocolError{
				Code:    "DEADLINE_ERROR",
				Type:    "PROTOCOL_ERROR",
				Message: "failed to set connection deadline",
				Cause:   err,
			}
		}
	}

	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			c.markDead()
			return nil, &ProtocolError{
				Code:    "RECEIVE_FAILED",
				Type:    "PROTOCOL_ERROR",
				Message: "failed to read response from server",
				Details: map[string]interface{}{},
				Cause:   err,
			}
		}
		c.markDead()
		return nil, &ProtocolError{
			Code:    "NO_RESPONSE",
			Type:    "PROTOCOL_ERROR",
			Message: "no response from server",
			Details: map[string]interface{}{},
		}
	}

	line := strings.TrimSpace(c.scanner.Text())

	// Check for welcome message (S0001)
	if strings.Contains(line, "S0001") {
		return line, nil
	}

	// Try to parse as JSON
	var result interface{}
	if err := json.Unmarshal([]byte(line), &result); err != nil {
		// Not JSON, return raw string
		return line, nil
	}

	// Check for error in JSON response
	if respMap, ok := result.(map[string]interface{}); ok {
		if success, hasSuccess := respMap["success"].(bool); hasSuccess && !success {
			// Error response
			errMsg := "unknown error"
			if errData, ok := respMap["error"]; ok {
				errMsg = fmt.Sprintf("%v", errData)
			}
			return nil, &ProtocolError{
				Code:    "SERVER_ERROR",
				Type:    "PROTOCOL_ERROR",
				Message: errMsg,
				Details: respMap,
			}
		}

		// Return data field if present
		if data, ok := respMap["data"]; ok {
			c.updateActivity()
			return data, nil
		}
	}

	c.updateActivity()
	return result, nil
}

// Ping sends a minimal status check command to verify connection health.
func (c *Connection) Ping(ctx context.Context) error {
	if !c.IsAlive() {
		return &ConnectionError{
			Code:    "CONNECTION_DEAD",
			Type:    "CONNECTION_ERROR",
			Message: "connection is not alive",
		}
	}

	// Send a lightweight status command
	// TODO: replace with actual ping command when server supports it
	if err := c.SendCommand(ctx, "STATUS"); err != nil {
		return err
	}

	// Read response to confirm connection is working
	if _, err := c.ReceiveResponse(ctx); err != nil {
		return err
	}

	return nil
}

// Close closes the connection gracefully.
func (c *Connection) Close() error {
	c.mu.Lock()
	c.alive = false
	c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// RemoteAddr returns the remote server address.
func (c *Connection) RemoteAddr() string {
	return c.remoteAddr
}

// IsAlive checks if the connection is still valid.
func (c *Connection) IsAlive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.alive
}

// LastActivity returns the timestamp of the last successful operation.
func (c *Connection) LastActivity() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastActivity
}

// updateActivity updates the last activity timestamp.
func (c *Connection) updateActivity() {
	c.mu.Lock()
	c.lastActivity = time.Now()
	c.mu.Unlock()
}

// markDead marks the connection as dead.
func (c *Connection) markDead() {
	c.mu.Lock()
	c.alive = false
	c.mu.Unlock()
}

// GetTLSConnectionState returns the TLS connection state if TLS is enabled.
func (c *Connection) GetTLSConnectionState() *tls.ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tlsState
}
