//go:build milestone2
// +build milestone2

package client

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestLoggingHook verifies the logging hook logs commands and results.
func TestLoggingHook(t *testing.T) {
	logger := NewLogger("DEBUG", nil)
	hook := NewLoggingHook(logger, true, true, true)

	if hook.Name() != "logging" {
		t.Errorf("expected name 'logging', got %s", hook.Name())
	}

	ctx := context.Background()
	hookCtx := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		TraceID:     "test-123",
		Metadata:    make(map[string]interface{}),
		Duration:    10 * time.Millisecond,
		Result:      "result data",
	}

	// Test Before
	if err := hook.Before(ctx, hookCtx); err != nil {
		t.Errorf("Before() failed: %v", err)
	}

	// Test After (success)
	if err := hook.After(ctx, hookCtx); err != nil {
		t.Errorf("After() failed: %v", err)
	}

	// Test After (error)
	hookCtx.Error = errors.New("test error")
	if err := hook.After(ctx, hookCtx); err != nil {
		t.Errorf("After() with error failed: %v", err)
	}
}

// TestMetricsHook verifies metrics collection.
func TestMetricsHook(t *testing.T) {
	hook := NewMetricsHook()

	if hook.Name() != "metrics" {
		t.Errorf("expected name 'metrics', got %s", hook.Name())
	}

	ctx := context.Background()

	// Execute some queries
	for i := 0; i < 5; i++ {
		hookCtx := &HookContext{
			Command:     "SELECT * FROM users",
			CommandType: "query",
			Duration:    10 * time.Millisecond,
			Metadata:    make(map[string]interface{}),
		}
		hook.Before(ctx, hookCtx)
		hook.After(ctx, hookCtx)
	}

	// Execute some mutations
	for i := 0; i < 3; i++ {
		hookCtx := &HookContext{
			Command:     "INSERT INTO users VALUES (1, 'test')",
			CommandType: "mutation",
			Duration:    15 * time.Millisecond,
			Metadata:    make(map[string]interface{}),
		}
		hook.Before(ctx, hookCtx)
		hook.After(ctx, hookCtx)
	}

	// Execute error
	errorCtx := &HookContext{
		Command:     "SELECT * FROM invalid",
		CommandType: "query",
		Duration:    5 * time.Millisecond,
		Error:       errors.New("table not found"),
		Metadata:    make(map[string]interface{}),
	}
	hook.Before(ctx, errorCtx)
	hook.After(ctx, errorCtx)

	// Check metrics
	stats := hook.GetStats()

	if stats["total_commands"].(uint64) != 9 {
		t.Errorf("expected 9 total commands, got %v", stats["total_commands"])
	}

	if stats["total_queries"].(uint64) != 6 {
		t.Errorf("expected 6 queries, got %v", stats["total_queries"])
	}

	if stats["total_mutations"].(uint64) != 3 {
		t.Errorf("expected 3 mutations, got %v", stats["total_mutations"])
	}

	if stats["total_errors"].(uint64) != 1 {
		t.Errorf("expected 1 error, got %v", stats["total_errors"])
	}

	if stats["avg_duration_ns"].(int64) <= 0 {
		t.Error("expected positive average duration")
	}

	// Test reset
	hook.Reset()
	stats = hook.GetStats()
	if stats["total_commands"].(uint64) != 0 {
		t.Errorf("expected 0 commands after reset, got %v", stats["total_commands"])
	}
}

// TestTracingHook verifies tracing metadata is set.
func TestTracingHook(t *testing.T) {
	hook := NewTracingHook("test-service")

	if hook.Name() != "tracing" {
		t.Errorf("expected name 'tracing', got %s", hook.Name())
	}

	ctx := context.Background()
	hookCtx := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		Metadata:    make(map[string]interface{}),
	}

	// Test Before
	if err := hook.Before(ctx, hookCtx); err != nil {
		t.Errorf("Before() failed: %v", err)
	}

	if hookCtx.Metadata["trace_service"] != "test-service" {
		t.Error("expected trace_service metadata to be set")
	}

	if _, ok := hookCtx.Metadata["trace_start"].(time.Time); !ok {
		t.Error("expected trace_start metadata to be set")
	}

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	// Test After
	if err := hook.After(ctx, hookCtx); err != nil {
		t.Errorf("After() failed: %v", err)
	}

	if duration, ok := hookCtx.Metadata["trace_duration"].(time.Duration); !ok || duration <= 0 {
		t.Error("expected trace_duration metadata to be set")
	}
}

// TestRetryHook verifies retry logic.
func TestRetryHook(t *testing.T) {
	hook := NewRetryHook(3, 10*time.Millisecond, 100*time.Millisecond)

	if hook.Name() != "retry" {
		t.Errorf("expected name 'retry', got %s", hook.Name())
	}

	ctx := context.Background()

	// Test with retryable error
	hookCtx := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		Error:       errors.New("CONNECTION_TIMEOUT"),
		Metadata:    make(map[string]interface{}),
	}

	hook.Before(ctx, hookCtx)

	if retryCount, ok := hookCtx.Metadata["retry_count"].(int); !ok || retryCount != 0 {
		t.Errorf("expected retry_count to be initialized to 0, got %v", hookCtx.Metadata["retry_count"])
	}

	// Test After with retryable error
	hook.After(ctx, hookCtx)

	// Test with non-retryable error
	nonRetryableCtx := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		Error:       errors.New("SYNTAX_ERROR"),
		Metadata:    make(map[string]interface{}),
	}

	hook.Before(ctx, nonRetryableCtx)
	hook.After(ctx, nonRetryableCtx)

	// Test with no error
	successCtx := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		Metadata:    make(map[string]interface{}),
	}

	hook.Before(ctx, successCtx)
	if err := hook.After(ctx, successCtx); err != nil {
		t.Errorf("After() with no error should not fail: %v", err)
	}
}

// TestCacheHook verifies caching behavior.
func TestCacheHook(t *testing.T) {
	hook := NewCacheHook(5 * time.Minute)

	if hook.Name() != "cache" {
		t.Errorf("expected name 'cache', got %s", hook.Name())
	}

	ctx := context.Background()

	// First execution - cache miss
	hookCtx := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		Result:      "query result",
		Metadata:    make(map[string]interface{}),
	}

	hook.Before(ctx, hookCtx)
	if hookCtx.Metadata["cache_hit"] != nil {
		t.Error("expected cache miss on first execution")
	}

	// Store result in cache
	hook.After(ctx, hookCtx)
	if hookCtx.Metadata["cached"] != true {
		t.Error("expected result to be cached")
	}

	// Second execution - cache hit
	hookCtx2 := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		Metadata:    make(map[string]interface{}),
	}

	hook.Before(ctx, hookCtx2)
	if hookCtx2.Metadata["cache_hit"] != true {
		t.Error("expected cache hit on second execution")
	}

	if hookCtx2.Result == nil {
		t.Error("expected cached result to be set")
	}

	// Test mutations are not cached
	mutationCtx := &HookContext{
		Command:     "INSERT INTO users VALUES (1, 'test')",
		CommandType: "mutation",
		Result:      "mutation result",
		Metadata:    make(map[string]interface{}),
	}

	hook.Before(ctx, mutationCtx)
	hook.After(ctx, mutationCtx)

	if mutationCtx.Metadata["cached"] == true {
		t.Error("mutations should not be cached")
	}

	// Test clear cache
	hook.ClearCache()

	hookCtx3 := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		Metadata:    make(map[string]interface{}),
	}

	hook.Before(ctx, hookCtx3)
	if hookCtx3.Metadata["cache_hit"] != nil {
		t.Error("expected cache miss after clear")
	}
}

// TestBuiltinHooksIntegration tests multiple hooks working together.
func TestBuiltinHooksIntegration(t *testing.T) {
	opts := DefaultOptions()
	opts.LogLevel = "ERROR" // Reduce noise
	client := NewClient(&opts)

	// Register multiple built-in hooks
	metricsHook := NewMetricsHook()
	tracingHook := NewTracingHook("integration-test")
	loggingHook := NewLoggingHook(client.logger, false, false, true)

	client.RegisterHook(metricsHook)
	client.RegisterHook(tracingHook)
	client.RegisterHook(loggingHook)

	hooks := client.GetHooks()
	if len(hooks) != 3 {
		t.Errorf("expected 3 hooks, got %d", len(hooks))
	}

	// Execute some commands through the hook chain
	ctx := context.Background()
	hookCtx := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		StartTime:   time.Now(),
		Metadata:    make(map[string]interface{}),
		TraceID:     "test-trace",
	}

	// Execute before hooks
	if err := client.executeBeforeHooks(ctx, hookCtx); err != nil {
		t.Errorf("executeBeforeHooks failed: %v", err)
	}

	// Simulate command execution
	time.Sleep(5 * time.Millisecond)
	hookCtx.Duration = time.Since(hookCtx.StartTime)
	hookCtx.Result = "test result"

	// Execute after hooks
	if err := client.executeAfterHooks(ctx, hookCtx); err != nil {
		t.Errorf("executeAfterHooks failed: %v", err)
	}

	// Verify metrics were collected
	stats := metricsHook.GetStats()
	if stats["total_commands"].(uint64) != 1 {
		t.Errorf("expected 1 command in metrics, got %v", stats["total_commands"])
	}

	// Verify tracing metadata was set
	if _, ok := hookCtx.Metadata["trace_duration"]; !ok {
		t.Error("expected trace_duration to be set")
	}
}

// TestHookNames verifies all built-in hooks have unique names.
func TestHookNames(t *testing.T) {
	hooks := []Hook{
		NewLoggingHook(NewLogger("ERROR", nil), false, false, false),
		NewMetricsHook(),
		NewTracingHook("test"),
		NewRetryHook(3, time.Second, time.Minute),
		NewCacheHook(5 * time.Minute),
	}

	names := make(map[string]bool)
	for _, hook := range hooks {
		name := hook.Name()
		if names[name] {
			t.Errorf("duplicate hook name: %s", name)
		}
		names[name] = true
	}

	expectedNames := []string{"logging", "metrics", "tracing", "retry", "cache"}
	for _, expected := range expectedNames {
		if !names[expected] {
			t.Errorf("expected hook name %s not found", expected)
		}
	}
}

// TestContainsErrorCodeHelper tests the containsErrorCode helper function.
func TestContainsErrorCodeHelper(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"CONNECTION_TIMEOUT", "CONNECTION_TIMEOUT", true},
		{"CONNECTION_TIMEOUT", "TIMEOUT", true}, // Substring match
		{"TIMEOUT", "TIMEOUT", true},
		{"", "", true},
		{"test", "", false},
		{"abc", "xyz", false},
		{"error: CONNECTION_TIMEOUT occurred", "CONNECTION_TIMEOUT", true},
	}

	for _, tt := range tests {
		got := containsErrorCode(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("containsErrorCode(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

// TestLoggingHookOptions verifies different logging configurations.
func TestLoggingHookOptions(t *testing.T) {
	logger := NewLogger("ERROR", nil)

	// Test with all options disabled
	hook1 := NewLoggingHook(logger, false, false, false)
	ctx := context.Background()
	hookCtx := &HookContext{
		Command:     "SELECT",
		CommandType: "query",
		Metadata:    make(map[string]interface{}),
	}

	if err := hook1.Before(ctx, hookCtx); err != nil {
		t.Errorf("Before() failed: %v", err)
	}

	if err := hook1.After(ctx, hookCtx); err != nil {
		t.Errorf("After() failed: %v", err)
	}

	// Test with all options enabled
	hook2 := NewLoggingHook(logger, true, true, true)
	if err := hook2.Before(ctx, hookCtx); err != nil {
		t.Errorf("Before() failed: %v", err)
	}

	if err := hook2.After(ctx, hookCtx); err != nil {
		t.Errorf("After() failed: %v", err)
	}
}

// TestMetricsHookAverageDuration verifies average duration calculation.
func TestMetricsHookAverageDuration(t *testing.T) {
	hook := NewMetricsHook()
	ctx := context.Background()

	// Execute commands with known durations
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
	}

	for _, duration := range durations {
		hookCtx := &HookContext{
			Command:     "SELECT",
			CommandType: "query",
			Duration:    duration,
			Metadata:    make(map[string]interface{}),
		}
		hook.After(ctx, hookCtx)
	}

	stats := hook.GetStats()
	avgNs := stats["avg_duration_ns"].(int64)
	expectedAvg := int64(20 * time.Millisecond)

	if avgNs != expectedAvg {
		t.Errorf("expected avg duration %d ns, got %d ns", expectedAvg, avgNs)
	}
}
