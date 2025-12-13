package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// EnableDebugMode enables debug mode with verbose logging and stack traces.
func (c *Client) EnableDebugMode() {
	c.debugMode.Store(true)
	c.logger.Info("debug mode enabled")
}

// DisableDebugMode disables debug mode.
func (c *Client) DisableDebugMode() {
	c.debugMode.Store(false)
	c.logger.Info("debug mode disabled")
}

// IsDebugMode returns whether debug mode is currently enabled.
func (c *Client) IsDebugMode() bool {
	return c.debugMode.Load()
}

// GetDebugInfo returns a comprehensive snapshot of client state for debugging.
func (c *Client) GetDebugInfo() map[string]interface{} {
	info := map[string]interface{}{
		"version":     Version,
		"state":       c.GetState().String(),
		"debugMode":   c.IsDebugMode(),
		"poolEnabled": c.poolEnabled,
	}

	// Connection info
	if c.poolEnabled && c.pool != nil {
		stats := c.pool.Stats()
		info["pool"] = map[string]interface{}{
			"activeConnections": stats.ActiveConnections.Load(),
			"idleConnections":   stats.IdleConnections.Load(),
			"totalConnections":  stats.TotalConnections.Load(),
			"waitCount":         stats.WaitCount.Load(),
			"waitDuration":      stats.WaitDuration.Load(),
			"hits":              stats.Hits.Load(),
			"misses":            stats.Misses.Load(),
			"timeouts":          stats.Timeouts.Load(),
			"errors":            stats.Errors.Load(),
		}
	} else if c.conn != nil {
		info["connection"] = map[string]interface{}{
			"remoteAddr":   c.conn.RemoteAddr(),
			"alive":        c.conn.IsAlive(),
			"lastActivity": c.conn.LastActivity().Format("2006-01-02T15:04:05.000Z07:00"),
		}

		if tlsState := c.conn.GetTLSConnectionState(); tlsState != nil {
			info["tls"] = map[string]interface{}{
				"version":           tlsState.Version,
				"cipherSuite":       tlsState.CipherSuite,
				"serverName":        tlsState.ServerName,
				"handshakeComplete": tlsState.HandshakeComplete,
			}
		}
	}

	// Options
	info["options"] = map[string]interface{}{
		"defaultTimeoutMs":     c.opts.DefaultTimeoutMs,
		"maxRetries":           c.opts.MaxRetries,
		"poolMinSize":          c.opts.PoolMinSize,
		"poolMaxSize":          c.opts.PoolMaxSize,
		"poolIdleTimeout":      c.opts.PoolIdleTimeout.String(),
		"healthCheckInterval":  c.opts.HealthCheckInterval.String(),
		"maxReconnectAttempts": c.opts.MaxReconnectAttempts,
		"tlsEnabled":           c.opts.TLSEnabled,
	}

	// Last transition
	lastTransition := c.GetLastTransition()
	info["lastTransition"] = map[string]interface{}{
		"from":      lastTransition.From.String(),
		"to":        lastTransition.To.String(),
		"timestamp": lastTransition.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
		"duration":  lastTransition.Duration.String(),
	}

	return info
}

// DumpDebugInfoJSON returns debug info as formatted JSON string.
func (c *Client) DumpDebugInfoJSON() string {
	info := c.GetDebugInfo()
	bytes, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal debug info: %s"}`, err.Error())
	}
	return string(bytes)
}

// logCommandExecution logs a command execution with full details in debug mode.
func (c *Client) logCommandExecution(ctx context.Context, command string, response interface{}, duration int64, err error) {
	if !c.IsDebugMode() {
		return
	}

	fields := []Field{
		String("command", command),
		Int64("durationNs", duration),
	}

	if err != nil {
		fields = append(fields, Error("error", err))
	}

	if response != nil {
		// In debug mode, include response details
		if respStr, ok := response.(string); ok {
			if len(respStr) > 1000 {
				fields = append(fields, String("responsePreview", respStr[:1000]+"..."))
				fields = append(fields, Int("responseLength", len(respStr)))
			} else {
				fields = append(fields, String("response", respStr))
			}
		} else {
			respBytes, _ := json.Marshal(response)
			respStr := string(respBytes)
			if len(respStr) > 1000 {
				fields = append(fields, String("responsePreview", respStr[:1000]+"..."))
			} else {
				fields = append(fields, String("response", respStr))
			}
		}
	}

	// Log raw bytes representation in debug mode
	fields = append(fields, String("commandBytes", fmt.Sprintf("%q", command)))

	c.logger.Debug("command execution detail", fields...)
}
