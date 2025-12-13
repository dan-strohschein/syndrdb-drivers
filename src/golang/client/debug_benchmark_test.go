package client

import (
	"testing"
	"time"
)

// BenchmarkQuery_DebugOff benchmarks query execution with debug mode disabled.
// Measures baseline overhead without debug logging.
func BenchmarkQuery_DebugOff(b *testing.B) {
	opts := DefaultOptions()
	opts.DebugMode = false
	opts.LogLevel = "ERROR" // Minimize logging overhead
	client := NewClient(&opts)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Query will fail (no connection) but will exercise debug code paths
		_, _ = client.Query("SELECT * FROM users", 100)
	}
}

// BenchmarkQuery_DebugOn benchmarks query execution with debug mode enabled.
// Expected overhead: <5% compared to DebugOff baseline.
func BenchmarkQuery_DebugOn(b *testing.B) {
	opts := DefaultOptions()
	opts.DebugMode = true
	opts.LogLevel = "DEBUG"
	client := NewClient(&opts)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = client.Query("SELECT * FROM users", 100)
	}
}

// BenchmarkErrorFormatting_DebugOff benchmarks error formatting without debug info.
func BenchmarkErrorFormatting_DebugOff(b *testing.B) {
	err := &ConnectionError{
		Code:       "TEST_ERROR",
		Type:       "CONNECTION_ERROR",
		Message:    "test error message",
		Details:    map[string]interface{}{"key": "value"},
		StackTrace: captureStackTrace(),
		Timestamp:  time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = err.FormatError(false)
	}
}

// BenchmarkErrorFormatting_DebugOn benchmarks error formatting with debug info.
func BenchmarkErrorFormatting_DebugOn(b *testing.B) {
	err := &ConnectionError{
		Code:       "TEST_ERROR",
		Type:       "CONNECTION_ERROR",
		Message:    "test error message",
		Details:    map[string]interface{}{"key": "value"},
		StackTrace: captureStackTrace(),
		Timestamp:  time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = err.FormatError(true)
	}
}

// BenchmarkStackTraceCapture benchmarks the cost of capturing stack traces.
func BenchmarkStackTraceCapture(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = captureStackTrace()
	}
}

// BenchmarkGoroutineIDCapture benchmarks the cost of getting goroutine ID.
func BenchmarkGoroutineIDCapture(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = getGoroutineID()
	}
}

// Run these benchmarks with:
// go test -bench=Debug -benchmem
//
// Expected results:
// - DebugOff should be baseline
// - DebugOn overhead should be <5% in ns/op
// - If overhead >5%, add TODO comment for optimization
