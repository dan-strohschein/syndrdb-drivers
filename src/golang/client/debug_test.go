package client

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestErrorFormatting_DebugMode verifies error formatting includes stack trace when debug enabled.
func TestErrorFormatting_DebugMode(t *testing.T) {
	err := &ConnectionError{
		Code:        "TEST_ERROR",
		Type:        "CONNECTION_ERROR",
		Message:     "test error message",
		Details:     map[string]interface{}{"key": "value"},
		StackTrace:  captureStackTrace(),
		Timestamp:   time.Now(),
		GoroutineID: getGoroutineID(),
	}

	// Test debug mode ON - should return full JSON with stack trace
	debugOutput := err.FormatError(true)

	if !strings.Contains(debugOutput, "stack_trace") {
		t.Error("Debug output should contain stack_trace")
	}
	if !strings.Contains(debugOutput, "timestamp") {
		t.Error("Debug output should contain timestamp")
	}
	if !strings.Contains(debugOutput, "goroutine_id") {
		t.Error("Debug output should contain goroutine_id")
	}
	if !strings.Contains(debugOutput, "TEST_ERROR") {
		t.Error("Debug output should contain error code")
	}

	// Verify JSON format
	if !strings.HasPrefix(debugOutput, "{") {
		t.Error("Debug output should be JSON format")
	}
}

// TestErrorFormatting_NormalMode verifies concise output without debug info.
func TestErrorFormatting_NormalMode(t *testing.T) {
	err := &ConnectionError{
		Code:        "TEST_ERROR",
		Type:        "CONNECTION_ERROR",
		Message:     "test error message",
		Details:     map[string]interface{}{"key": "value"},
		StackTrace:  captureStackTrace(),
		Timestamp:   time.Now(),
		GoroutineID: getGoroutineID(),
	}

	// Test debug mode OFF - should return simple format
	normalOutput := err.FormatError(false)

	if strings.Contains(normalOutput, "stack_trace") {
		t.Error("Normal output should NOT contain stack_trace")
	}
	if strings.Contains(normalOutput, "timestamp") {
		t.Error("Normal output should NOT contain timestamp")
	}
	if strings.Contains(normalOutput, "goroutine_id") {
		t.Error("Normal output should NOT contain goroutine_id")
	}
	if !strings.Contains(normalOutput, "TEST_ERROR") {
		t.Error("Normal output should contain error code")
	}
	if !strings.Contains(normalOutput, "test error message") {
		t.Error("Normal output should contain error message")
	}

	// Should be simple string format
	expected := "TEST_ERROR: test error message"
	if normalOutput != expected {
		t.Errorf("Expected %q, got %q", expected, normalOutput)
	}
}

// TestErrorFormatting_WithCause verifies cause chain formatting.
func TestErrorFormatting_WithCause(t *testing.T) {
	causeErr := &ConnectionError{
		Code:    "CAUSE_ERROR",
		Type:    "CONNECTION_ERROR",
		Message: "underlying cause",
	}

	err := &ConnectionError{
		Code:       "TEST_ERROR",
		Type:       "CONNECTION_ERROR",
		Message:    "test error message",
		Cause:      causeErr,
		StackTrace: captureStackTrace(),
	}

	// Test normal mode with cause
	normalOutput := err.FormatError(false)
	if !strings.Contains(normalOutput, "caused by") {
		t.Error("Normal output should contain 'caused by' for errors with cause")
	}
	if !strings.Contains(normalOutput, "underlying cause") {
		t.Error("Normal output should contain cause message")
	}

	// Test debug mode with cause
	debugOutput := err.FormatError(true)
	if !strings.Contains(debugOutput, "\"cause\"") {
		t.Error("Debug output should contain cause object")
	}
}

// TestQueryError_FormatError verifies QueryError formatting.
func TestQueryError_FormatError(t *testing.T) {
	err := &QueryError{
		Code:       "QUERY_ERROR",
		Type:       "QUERY_ERROR",
		Message:    "query failed",
		Query:      "SELECT * FROM users",
		Params:     []interface{}{1, "test"},
		StackTrace: captureStackTrace(),
		Timestamp:  time.Now(),
	}

	// Debug mode should include query and params
	debugOutput := err.FormatError(true)
	if !strings.Contains(debugOutput, "SELECT * FROM users") {
		t.Error("Debug output should contain query text")
	}
	if !strings.Contains(debugOutput, "\"params\"") {
		t.Error("Debug output should contain params")
	}

	// Normal mode should be concise
	normalOutput := err.FormatError(false)
	if strings.Contains(normalOutput, "SELECT * FROM users") {
		t.Error("Normal output should NOT contain query text for brevity")
	}
}

// TestTransactionError_FormatError verifies TransactionError formatting.
func TestTransactionError_FormatError(t *testing.T) {
	err := &TransactionError{
		Code:          "TX_ERROR",
		Type:          "TRANSACTION_ERROR",
		Message:       "transaction failed",
		TransactionID: "TX_123456",
		State:         "active",
		StackTrace:    captureStackTrace(),
		Timestamp:     time.Now(),
	}

	// Debug mode should include TX ID
	debugOutput := err.FormatError(true)
	if !strings.Contains(debugOutput, "TX_123456") {
		t.Error("Debug output should contain transaction ID")
	}

	// Normal mode should include TX ID in concise format
	normalOutput := err.FormatError(false)
	if !strings.Contains(normalOutput, "TX_123456") {
		t.Error("Normal output should contain transaction ID")
	}
	if !strings.Contains(normalOutput, "TX:") {
		t.Error("Normal output should have TX: prefix")
	}
}

// TestDebugModeToggle verifies debug mode can be toggled at runtime.
func TestDebugModeToggle(t *testing.T) {
	opts := DefaultOptions()
	opts.DebugMode = false
	client := NewClient(&opts)

	// Verify initial state
	if client.IsDebugMode() {
		t.Error("Debug mode should be off initially")
	}

	// Enable debug mode
	client.EnableDebugMode()
	if !client.IsDebugMode() {
		t.Error("Debug mode should be on after enabling")
	}

	// Disable debug mode
	client.DisableDebugMode()
	if client.IsDebugMode() {
		t.Error("Debug mode should be off after disabling")
	}

	// Verify no reconnection needed (client state unchanged)
	if client.GetState() != DISCONNECTED {
		t.Error("Client state should remain unchanged")
	}
}

// TestGoroutineIDCapture verifies goroutine ID is captured correctly.
func TestGoroutineIDCapture(t *testing.T) {
	gid := getGoroutineID()

	if gid <= 0 {
		t.Errorf("Expected positive goroutine ID, got %d", gid)
	}

	// Capture in error
	err := &ConnectionError{
		Code:        "TEST",
		Type:        "CONNECTION_ERROR",
		Message:     "test",
		GoroutineID: getGoroutineID(),
	}

	if err.GoroutineID <= 0 {
		t.Error("Error should have valid goroutine ID")
	}
}

// TestStackTraceCapture verifies stack traces are captured.
func TestStackTraceCapture(t *testing.T) {
	stack := captureStackTrace()

	if len(stack) == 0 {
		t.Error("Stack trace should not be empty")
	}

	// Verify stack frames have expected format: "function (file:line)"
	for _, frame := range stack {
		if !strings.Contains(frame, "(") || !strings.Contains(frame, ":") {
			t.Errorf("Invalid stack frame format: %s", frame)
		}
	}

	// Verify at least one frame contains the client package
	// (The test function itself may be skipped due to captureStackTrace skip count)
	found := false
	for _, frame := range stack {
		if strings.Contains(frame, "client.") || strings.Contains(frame, "testing.") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Stack trace should contain recognizable frames, got: %v", stack)
	}
}

// TestFormatErrorHelper verifies the FormatError helper function.
func TestFormatErrorHelper(t *testing.T) {
	// Test with custom error
	customErr := &ConnectionError{
		Code:       "TEST",
		Type:       "CONNECTION_ERROR",
		Message:    "test",
		StackTrace: captureStackTrace(),
	}

	// Should use FormatError method
	debugOutput := FormatError(customErr, true)
	if !strings.Contains(debugOutput, "stack_trace") {
		t.Error("FormatError should use custom FormatError method in debug mode")
	}

	normalOutput := FormatError(customErr, false)
	if strings.Contains(normalOutput, "stack_trace") {
		t.Error("FormatError should use custom FormatError method in normal mode")
	}

	// Test with standard error
	stdErr := context.DeadlineExceeded
	output := FormatError(stdErr, true)
	if output != stdErr.Error() {
		t.Error("FormatError should fallback to Error() for standard errors")
	}

	// Test with nil
	nilOutput := FormatError(nil, true)
	if nilOutput != "" {
		t.Error("FormatError should return empty string for nil error")
	}
}

// TestErrorFactory_StackTraces verifies all error factory functions create errors with stack traces.
func TestErrorFactory_StackTraces(t *testing.T) {
	tests := []struct {
		name string
		err  interface{ FormatError(bool) string }
	}{
		{"InvalidParameterCount", ErrInvalidParameterCount(3, 2)},
		{"StatementNotFound", ErrStatementNotFound("test_stmt")},
		{"TransactionAlreadyActive", ErrTransactionAlreadyActive("TX_123")},
		{"NoActiveTransaction", ErrNoActiveTransaction("commit")},
		{"TransactionAlreadyCommitted", ErrTransactionAlreadyCommitted("TX_123")},
		{"TransactionAlreadyRolledBack", ErrTransactionAlreadyRolledBack("TX_123")},
		{"TransactionTimeout", ErrTransactionTimeout("TX_123", 5000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.err.FormatError(true)
			if !strings.Contains(output, "stack_trace") {
				t.Errorf("%s should include stack trace in debug mode", tt.name)
			}
		})
	}
}
