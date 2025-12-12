package client

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
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

// captureStackTrace captures the current stack trace for error reporting.
func captureStackTrace() []string {
	const maxDepth = 32
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(3, pcs) // Skip captureStackTrace, the error constructor, and runtime.Callers

	frames := make([]string, 0, n)
	callersFrames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := callersFrames.Next()

		// Format: function (file:line)
		frames = append(frames, fmt.Sprintf("%s (%s:%d)",
			frame.Function,
			frame.File,
			frame.Line,
		))

		if !more {
			break
		}
	}

	return frames
}

// ErrorWithDebug formats an error with debug information if debug mode is enabled.
func (e *ConnectionError) ErrorWithDebug(debugEnabled bool) string {
	if !debugEnabled {
		// Simple format without cause chain
		if e.Cause == nil {
			b, _ := json.Marshal(map[string]interface{}{
				"code":    e.Code,
				"type":    e.Type,
				"message": e.Message,
				"details": e.Details,
			})
			return string(b)
		}

		// Flatten cause into message
		return fmt.Sprintf("%s: %s", e.Message, e.Cause.Error())
	}

	// Debug mode: full detail with stack trace
	errorData := map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
		"details": e.Details,
	}

	if e.Cause != nil {
		errorData["cause"] = e.Cause.Error()
	}

	// Add stack trace in debug mode
	errorData["stackTrace"] = captureStackTrace()

	b, _ := json.Marshal(errorData)
	return string(b)
}

// ErrorWithDebug formats an error with debug information if debug mode is enabled.
func (e *ProtocolError) ErrorWithDebug(debugEnabled bool) string {
	if !debugEnabled {
		if e.Cause == nil {
			b, _ := json.Marshal(map[string]interface{}{
				"code":    e.Code,
				"type":    e.Type,
				"message": e.Message,
				"details": e.Details,
			})
			return string(b)
		}

		return fmt.Sprintf("%s: %s", e.Message, e.Cause.Error())
	}

	errorData := map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
		"details": e.Details,
	}

	if e.Cause != nil {
		errorData["cause"] = e.Cause.Error()
	}

	errorData["stackTrace"] = captureStackTrace()

	b, _ := json.Marshal(errorData)
	return string(b)
}

// ErrorWithDebug formats an error with debug information if debug mode is enabled.
func (e *StateError) ErrorWithDebug(debugEnabled bool) string {
	if !debugEnabled {
		return e.Message
	}

	errorData := map[string]interface{}{
		"code":       e.Code,
		"type":       e.Type,
		"message":    e.Message,
		"stackTrace": captureStackTrace(),
	}

	b, _ := json.Marshal(errorData)
	return string(b)
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
