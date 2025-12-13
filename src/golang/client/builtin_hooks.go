//go:build milestone2
// +build milestone2

package client

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// ============================================================================
// LoggingHook - Logs command execution details
// ============================================================================

// LoggingHook logs command execution with configurable detail levels.
type LoggingHook struct {
	logger       Logger
	logCommands  bool // Log raw commands
	logResults   bool // Log results
	logDurations bool // Log execution times
}

// NewLoggingHook creates a new logging hook with the given logger.
func NewLoggingHook(logger Logger, logCommands, logResults, logDurations bool) *LoggingHook {
	return &LoggingHook{
		logger:       logger,
		logCommands:  logCommands,
		logResults:   logResults,
		logDurations: logDurations,
	}
}

func (h *LoggingHook) Name() string {
	return "logging"
}

func (h *LoggingHook) Before(ctx context.Context, hookCtx *HookContext) error {
	if h.logCommands {
		h.logger.Debug("executing command",
			String("command", hookCtx.Command),
			String("type", hookCtx.CommandType),
			String("trace_id", hookCtx.TraceID))
	}
	return nil
}

func (h *LoggingHook) After(ctx context.Context, hookCtx *HookContext) error {
	fields := []Field{
		String("command_type", hookCtx.CommandType),
		String("trace_id", hookCtx.TraceID),
	}

	if h.logDurations {
		fields = append(fields, Duration("duration", hookCtx.Duration))
	}

	if hookCtx.Error != nil {
		fields = append(fields, Error("error", hookCtx.Error))
		h.logger.Error("command failed", fields...)
	} else {
		if h.logResults && hookCtx.Result != nil {
			fields = append(fields, String("result", fmt.Sprintf("%v", hookCtx.Result)))
		}
		h.logger.Debug("command completed", fields...)
	}

	return nil
}

// ============================================================================
// MetricsHook - Collects performance metrics
// ============================================================================

// MetricsHook collects command execution metrics using atomic counters.
type MetricsHook struct {
	TotalCommands   atomic.Uint64
	TotalQueries    atomic.Uint64
	TotalMutations  atomic.Uint64
	TotalErrors     atomic.Uint64
	TotalDurationNs atomic.Uint64
}

// NewMetricsHook creates a new metrics collection hook.
func NewMetricsHook() *MetricsHook {
	return &MetricsHook{}
}

func (h *MetricsHook) Name() string {
	return "metrics"
}

func (h *MetricsHook) Before(ctx context.Context, hookCtx *HookContext) error {
	return nil
}

func (h *MetricsHook) After(ctx context.Context, hookCtx *HookContext) error {
	h.TotalCommands.Add(1)
	h.TotalDurationNs.Add(uint64(hookCtx.Duration.Nanoseconds()))

	switch hookCtx.CommandType {
	case "query":
		h.TotalQueries.Add(1)
	case "mutation":
		h.TotalMutations.Add(1)
	}

	if hookCtx.Error != nil {
		h.TotalErrors.Add(1)
	}

	return nil
}

// GetStats returns current metrics as a map.
func (h *MetricsHook) GetStats() map[string]interface{} {
	totalCmds := h.TotalCommands.Load()
	totalDur := h.TotalDurationNs.Load()

	avgDuration := int64(0)
	if totalCmds > 0 {
		avgDuration = int64(totalDur / totalCmds)
	}

	return map[string]interface{}{
		"total_commands":    totalCmds,
		"total_queries":     h.TotalQueries.Load(),
		"total_mutations":   h.TotalMutations.Load(),
		"total_errors":      h.TotalErrors.Load(),
		"total_duration_ns": totalDur,
		"avg_duration_ns":   avgDuration,
		"avg_duration_ms":   float64(avgDuration) / 1_000_000,
		"total_duration_ms": float64(totalDur) / 1_000_000,
	}
}

// Reset clears all metrics.
func (h *MetricsHook) Reset() {
	h.TotalCommands.Store(0)
	h.TotalQueries.Store(0)
	h.TotalMutations.Store(0)
	h.TotalErrors.Store(0)
	h.TotalDurationNs.Store(0)
}

// ============================================================================
// TracingHook - Distributed tracing support
// ============================================================================

// TracingHook provides distributed tracing integration.
// TODO: Add OpenTelemetry integration when dependency is approved.
type TracingHook struct {
	serviceName string
}

// NewTracingHook creates a new tracing hook.
func NewTracingHook(serviceName string) *TracingHook {
	return &TracingHook{
		serviceName: serviceName,
	}
}

func (h *TracingHook) Name() string {
	return "tracing"
}

func (h *TracingHook) Before(ctx context.Context, hookCtx *HookContext) error {
	// TODO: Start OpenTelemetry span
	// span, ctx := otel.Tracer(h.serviceName).Start(ctx, hookCtx.CommandType)
	// hookCtx.Metadata["trace_span"] = span

	// For now, just record start time
	hookCtx.Metadata["trace_start"] = time.Now()
	hookCtx.Metadata["trace_service"] = h.serviceName
	return nil
}

func (h *TracingHook) After(ctx context.Context, hookCtx *HookContext) error {
	// TODO: End OpenTelemetry span with attributes
	// if span, ok := hookCtx.Metadata["trace_span"].(trace.Span); ok {
	//     span.SetAttributes(
	//         attribute.String("db.system", "syndrdb"),
	//         attribute.String("db.statement", hookCtx.Command),
	//         attribute.String("db.operation", hookCtx.CommandType),
	//     )
	//     if hookCtx.Error != nil {
	//         span.RecordError(hookCtx.Error)
	//         span.SetStatus(codes.Error, hookCtx.Error.Error())
	//     }
	//     span.End()
	// }

	// For now, calculate duration manually
	if start, ok := hookCtx.Metadata["trace_start"].(time.Time); ok {
		duration := time.Since(start)
		hookCtx.Metadata["trace_duration"] = duration
	}
	return nil
}

// ============================================================================
// RetryHook - Automatic retry with exponential backoff
// ============================================================================

// RetryHook automatically retries failed commands with exponential backoff.
type RetryHook struct {
	maxRetries      int
	initialBackoff  time.Duration
	maxBackoff      time.Duration
	retryableErrors map[string]bool
}

// NewRetryHook creates a new retry hook with exponential backoff.
func NewRetryHook(maxRetries int, initialBackoff, maxBackoff time.Duration) *RetryHook {
	return &RetryHook{
		maxRetries:     maxRetries,
		initialBackoff: initialBackoff,
		maxBackoff:     maxBackoff,
		retryableErrors: map[string]bool{
			"CONNECTION_TIMEOUT": true,
			"CONNECTION_LOST":    true,
			"NETWORK_ERROR":      true,
		},
	}
}

func (h *RetryHook) Name() string {
	return "retry"
}

func (h *RetryHook) Before(ctx context.Context, hookCtx *HookContext) error {
	// Initialize retry counter
	if _, exists := hookCtx.Metadata["retry_count"]; !exists {
		hookCtx.Metadata["retry_count"] = 0
	}
	return nil
}

func (h *RetryHook) After(ctx context.Context, hookCtx *HookContext) error {
	// Only retry on specific errors
	if hookCtx.Error == nil {
		return nil
	}

	// Check if error is retryable
	// TODO: Improve error type detection
	errorStr := hookCtx.Error.Error()
	isRetryable := false
	for errCode := range h.retryableErrors {
		if containsErrorCode(errorStr, errCode) {
			isRetryable = true
			break
		}
	}

	if !isRetryable {
		return nil
	}

	// Check retry count
	retryCount, _ := hookCtx.Metadata["retry_count"].(int)
	if retryCount >= h.maxRetries {
		return nil
	}

	// Calculate backoff
	backoff := h.initialBackoff * time.Duration(1<<uint(retryCount))
	if backoff > h.maxBackoff {
		backoff = h.maxBackoff
	}

	// Wait with context cancellation support
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Increment retry count for next attempt
		hookCtx.Metadata["retry_count"] = retryCount + 1
		// TODO: Implement actual retry logic - needs access to Client.sendCommand
		// For now, just log that we would retry
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ============================================================================
// CacheHook - Query result caching
// ============================================================================

// CacheHook caches query results for read operations.
type CacheHook struct {
	cache   map[string]interface{}
	mu      atomic.Value // stores *sync.RWMutex
	enabled bool
	ttl     time.Duration
}

// NewCacheHook creates a new caching hook.
func NewCacheHook(ttl time.Duration) *CacheHook {
	return &CacheHook{
		cache:   make(map[string]interface{}),
		enabled: true,
		ttl:     ttl,
	}
}

func (h *CacheHook) Name() string {
	return "cache"
}

func (h *CacheHook) Before(ctx context.Context, hookCtx *HookContext) error {
	if !h.enabled || hookCtx.CommandType != "query" {
		return nil
	}

	// Check cache for result
	// TODO: Implement proper cache key generation and TTL checking
	cacheKey := hookCtx.Command
	if result, exists := h.cache[cacheKey]; exists {
		// Cache hit - set result and skip execution
		hookCtx.Metadata["cache_hit"] = true
		hookCtx.Result = result
		// TODO: Need mechanism to skip actual command execution
	}

	return nil
}

func (h *CacheHook) After(ctx context.Context, hookCtx *HookContext) error {
	if !h.enabled || hookCtx.CommandType != "query" || hookCtx.Error != nil {
		return nil
	}

	// Store result in cache
	// TODO: Implement proper cache invalidation strategy
	cacheKey := hookCtx.Command
	h.cache[cacheKey] = hookCtx.Result
	hookCtx.Metadata["cached"] = true

	return nil
}

// ClearCache clears all cached results.
func (h *CacheHook) ClearCache() {
	h.cache = make(map[string]interface{})
}

// Helper function to check if error string contains error code.
// Checks for exact substring match anywhere in the string.
func containsErrorCode(s, substr string) bool {
	if len(substr) == 0 {
		return len(s) == 0
	}
	if len(s) < len(substr) {
		return false
	}
	// Simple substring search
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
