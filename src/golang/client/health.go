//go:build !wasm
// +build !wasm

package client

import (
	"context"
	"errors"
	"io"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// HealthMonitor periodically checks connection health and triggers reconnection if needed.
type HealthMonitor struct {
	client           *Client
	interval         time.Duration
	failureThreshold int
	failureCount     atomic.Int32
	stopCh           chan struct{}
	wg               sync.WaitGroup
	logger           Logger
}

// NewHealthMonitor creates a new health monitor for the client.
func NewHealthMonitor(client *Client, interval time.Duration, threshold int) *HealthMonitor {
	return &HealthMonitor{
		client:           client,
		interval:         interval,
		failureThreshold: threshold,
		stopCh:           make(chan struct{}),
		logger:           client.logger.WithFields(String("component", "health_monitor")),
	}
}

// Start begins the health check monitoring in a background goroutine.
func (h *HealthMonitor) Start() {
	h.wg.Add(1)
	go h.monitorLoop()
	h.logger.Info("health monitor started", Duration("interval", h.interval))
}

// Stop stops the health monitor gracefully.
func (h *HealthMonitor) Stop() {
	close(h.stopCh)
	h.wg.Wait()
	h.logger.Info("health monitor stopped")
}

// monitorLoop is the main monitoring loop.
func (h *HealthMonitor) monitorLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return

		case <-ticker.C:
			if h.client.GetState() != CONNECTED {
				continue
			}

			if err := h.performHealthCheck(); err != nil {
				h.logger.Warn("health check failed",
					Error("error", err),
					Int("failureCount", int(h.failureCount.Add(1))))

				if int(h.failureCount.Load()) >= h.failureThreshold {
					h.logger.Error("health check failure threshold exceeded, triggering reconnection")
					go h.client.attemptReconnect(context.Background())
					h.failureCount.Store(0)
				}
			} else {
				// Reset failure count on success
				if prev := h.failureCount.Swap(0); prev > 0 {
					h.logger.Info("health check recovered", Int("previousFailures", int(prev)))
				}
			}
		}
	}
}

// performHealthCheck executes a ping on the connection.
func (h *HealthMonitor) performHealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if h.client.poolEnabled && h.client.pool != nil {
		// In pool mode, check by trying to get a connection
		conn, err := h.client.pool.Get(ctx)
		if err != nil {
			return err
		}
		defer h.client.pool.Put(conn)

		return conn.Ping(ctx)
	}

	// Single connection mode
	if h.client.conn == nil {
		return errors.New("no active connection")
	}

	return h.client.conn.Ping(ctx)
}

// detectConnectionDrop checks if an error indicates a connection drop.
func detectConnectionDrop(err error) bool {
	if err == nil {
		return false
	}

	// Check for common connection drop indicators
	if errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EPIPE) {
		return true
	}

	// Check for net.OpError types
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}

	// Check error string for common patterns
	errStr := err.Error()
	dropPatterns := []string{
		"connection reset",
		"broken pipe",
		"connection refused",
		"connection closed",
		"EOF",
	}

	for _, pattern := range dropPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0))
}

// indexOf returns the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// attemptReconnect tries to reconnect with exponential backoff.
func (c *Client) attemptReconnect(ctx context.Context) error {
	c.logger.Warn("attempting automatic reconnection")

	// Transition to CONNECTING state
	c.stateMgr.TransitionTo(CONNECTING, nil, map[string]interface{}{
		"reason": "auto_reconnect",
	})

	backoff := 100 * time.Millisecond
	maxBackoff := 60 * time.Second

	for attempt := 1; attempt <= c.opts.MaxReconnectAttempts; attempt++ {
		c.logger.Info("reconnection attempt",
			Int("attempt", attempt),
			Int("maxAttempts", c.opts.MaxReconnectAttempts),
			Duration("backoff", backoff))

		// Check context cancellation
		select {
		case <-ctx.Done():
			c.stateMgr.TransitionTo(DISCONNECTED, ctx.Err(), map[string]interface{}{
				"reason": "context_cancelled",
			})
			return ctx.Err()
		default:
		}

		// Try to reconnect
		if c.poolEnabled && c.pool != nil {
			// Reinitialize the pool
			c.pool.Close(ctx)
			c.pool = NewConnectionPool(
				c.connFactory,
				c.opts.PoolMinSize,
				c.opts.PoolMaxSize,
				c.opts.PoolIdleTimeout,
				c.opts.HealthCheckInterval,
			)

			if err := c.pool.Initialize(ctx); err == nil {
				c.logger.Info("reconnection successful via pool")
				c.stateMgr.TransitionTo(CONNECTED, nil, map[string]interface{}{
					"reason":  "auto_reconnect",
					"attempt": attempt,
				})
				return nil
			}
		} else {
			// Single connection mode
			conn, err := c.connFactory(ctx)
			if err == nil {
				if c.conn != nil {
					c.conn.Close()
				}
				c.conn = conn.(*Connection)
				c.logger.Info("reconnection successful")
				c.stateMgr.TransitionTo(CONNECTED, nil, map[string]interface{}{
					"reason":  "auto_reconnect",
					"attempt": attempt,
				})
				return nil
			}
		}

		// Calculate next backoff with exponential growth
		if attempt < c.opts.MaxReconnectAttempts {
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * math.Pow(2, float64(attempt)))
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	// All reconnection attempts failed
	c.logger.Error("reconnection failed after all attempts",
		Int("maxAttempts", c.opts.MaxReconnectAttempts))
	c.stateMgr.TransitionTo(DISCONNECTED, errors.New("reconnection failed"), map[string]interface{}{
		"reason":   "reconnect_failed",
		"attempts": c.opts.MaxReconnectAttempts,
	})

	return errors.New("reconnection failed after maximum attempts")
}
