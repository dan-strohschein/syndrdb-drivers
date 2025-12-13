//go:build milestone2
// +build milestone2

package client

import (
	"context"
	"testing"
	"time"
)

// NoOpHook is a minimal hook that does nothing (for baseline benchmarking).
type NoOpHook struct {
	name string
}

func (h *NoOpHook) Name() string {
	return h.name
}

func (h *NoOpHook) Before(ctx context.Context, hookCtx *HookContext) error {
	return nil
}

func (h *NoOpHook) After(ctx context.Context, hookCtx *HookContext) error {
	return nil
}

// SimpleLoggingHook logs basic info (representative of real hook overhead).
type SimpleLoggingHook struct {
	name    string
	counter int
}

func (h *SimpleLoggingHook) Name() string {
	return h.name
}

func (h *SimpleLoggingHook) Before(ctx context.Context, hookCtx *HookContext) error {
	h.counter++
	// Simulate simple logging work
	_ = hookCtx.Command
	_ = hookCtx.TraceID
	return nil
}

func (h *SimpleLoggingHook) After(ctx context.Context, hookCtx *HookContext) error {
	h.counter++
	// Simulate timing calculation
	_ = hookCtx.Duration
	return nil
}

// BenchmarkQuery_NoHooks establishes baseline performance without hooks.
func BenchmarkQuery_NoHooks(b *testing.B) {
	opts := DefaultOptions()
	opts.DebugMode = false
	opts.LogLevel = "ERROR"
	client := NewClient(&opts)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = client.Query("SELECT * FROM users", 100)
	}
}

// BenchmarkQuery_1Hook benchmarks with a single no-op hook.
func BenchmarkQuery_1Hook(b *testing.B) {
	opts := DefaultOptions()
	opts.DebugMode = false
	opts.LogLevel = "ERROR"
	client := NewClient(&opts)

	client.RegisterHook(&NoOpHook{name: "noop1"})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = client.Query("SELECT * FROM users", 100)
	}
}

// BenchmarkQuery_3Hooks benchmarks with 3 no-op hooks (target <2% overhead).
func BenchmarkQuery_3Hooks(b *testing.B) {
	opts := DefaultOptions()
	opts.DebugMode = false
	opts.LogLevel = "ERROR"
	client := NewClient(&opts)

	client.RegisterHook(&NoOpHook{name: "noop1"})
	client.RegisterHook(&NoOpHook{name: "noop2"})
	client.RegisterHook(&NoOpHook{name: "noop3"})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = client.Query("SELECT * FROM users", 100)
	}
}

// BenchmarkQuery_5Hooks benchmarks with 5 no-op hooks (stress test).
func BenchmarkQuery_5Hooks(b *testing.B) {
	opts := DefaultOptions()
	opts.DebugMode = false
	opts.LogLevel = "ERROR"
	client := NewClient(&opts)

	client.RegisterHook(&NoOpHook{name: "noop1"})
	client.RegisterHook(&NoOpHook{name: "noop2"})
	client.RegisterHook(&NoOpHook{name: "noop3"})
	client.RegisterHook(&NoOpHook{name: "noop4"})
	client.RegisterHook(&NoOpHook{name: "noop5"})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = client.Query("SELECT * FROM users", 100)
	}
}

// BenchmarkQuery_3SimpleHooks benchmarks with realistic simple logging hooks.
func BenchmarkQuery_3SimpleHooks(b *testing.B) {
	opts := DefaultOptions()
	opts.DebugMode = false
	opts.LogLevel = "ERROR"
	client := NewClient(&opts)

	client.RegisterHook(&SimpleLoggingHook{name: "log1"})
	client.RegisterHook(&SimpleLoggingHook{name: "log2"})
	client.RegisterHook(&SimpleLoggingHook{name: "log3"})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = client.Query("SELECT * FROM users", 100)
	}
}

// BenchmarkHookExecution_Before benchmarks just the Before hook execution.
func BenchmarkHookExecution_Before(b *testing.B) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	client.RegisterHook(&NoOpHook{name: "noop1"})
	client.RegisterHook(&NoOpHook{name: "noop2"})
	client.RegisterHook(&NoOpHook{name: "noop3"})

	ctx := context.Background()
	hookCtx := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		StartTime:   time.Now(),
		Metadata:    make(map[string]interface{}),
		TraceID:     "test-trace",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = client.executeBeforeHooks(ctx, hookCtx)
	}
}

// BenchmarkHookExecution_After benchmarks just the After hook execution.
func BenchmarkHookExecution_After(b *testing.B) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	client.RegisterHook(&NoOpHook{name: "noop1"})
	client.RegisterHook(&NoOpHook{name: "noop2"})
	client.RegisterHook(&NoOpHook{name: "noop3"})

	ctx := context.Background()
	hookCtx := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		StartTime:   time.Now(),
		Metadata:    make(map[string]interface{}),
		TraceID:     "test-trace",
		Result:      "test result",
		Duration:    100 * time.Millisecond,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = client.executeAfterHooks(ctx, hookCtx)
	}
}

// BenchmarkHookRegistration benchmarks hook registration overhead.
func BenchmarkHookRegistration(b *testing.B) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		hook := &NoOpHook{name: "test"}
		client.RegisterHook(hook)
		client.UnregisterHook("test")
	}
}

// BenchmarkInferCommandType benchmarks command type inference.
func BenchmarkInferCommandType(b *testing.B) {
	commands := []string{
		"SELECT * FROM users",
		"INSERT INTO users VALUES (1, 'test')",
		"UPDATE users SET name='test'",
		"DELETE FROM users WHERE id=1",
		"BEGIN TRANSACTION",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = inferCommandType(commands[i%len(commands)])
	}
}

// Run these benchmarks with:
// go test -tags milestone2 -bench=BenchmarkQuery -benchmem ./client/
//
// Expected results:
// - NoHooks: baseline
// - 1Hook: <1% overhead
// - 3Hooks: <2% overhead (CRITICAL THRESHOLD)
// - 5Hooks: acceptable if <5% (stress test)
//
// If 3Hooks overhead >2%:
// - Document actual percentage
// - Add TODO comment with optimization options
// - DO NOT implement optimizations without user approval
