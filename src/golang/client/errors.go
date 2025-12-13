package client

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// ConnectionError represents connection-related failures.
type ConnectionError struct {
	Code        string                 `json:"code"`
	Type        string                 `json:"type"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
	Cause       error                  `json:"cause,omitempty"`
	StackTrace  []string               `json:"stack_trace,omitempty"`
	Timestamp   time.Time              `json:"timestamp,omitempty"`
	GoroutineID int                    `json:"goroutine_id,omitempty"`
}

// Error implements the error interface.
// Returns JSON format for backward compatibility.
// Use FormatError() for flexible formatting based on debug mode.
func (e *ConnectionError) Error() string {
	// For backward compatibility, Error() returns basic JSON
	errorData := map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
	}

	if len(e.Details) > 0 {
		errorData["details"] = e.Details
	}

	if e.Cause != nil {
		if cerr, ok := e.Cause.(*ConnectionError); ok {
			errorData["cause"] = map[string]interface{}{
				"code":    cerr.Code,
				"type":    cerr.Type,
				"message": cerr.Message,
			}
		} else {
			errorData["cause"] = map[string]interface{}{
				"message": e.Cause.Error(),
			}
		}
	}

	b, _ := json.Marshal(errorData)
	return string(b)
}

// FormatError formats the error based on debug mode setting.
// When debugMode=false: returns simple "CODE: message" format.
// When debugMode=true: returns full JSON with stack trace, timestamp, goroutine ID.
func (e *ConnectionError) FormatError(debugMode bool) string {
	if !debugMode {
		// Simple, concise format for production
		if e.Cause != nil {
			return fmt.Sprintf("%s: %s (caused by: %s)", e.Code, e.Message, e.Cause.Error())
		}
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	// Full debug format with all details
	errorData := map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
	}

	if len(e.Details) > 0 {
		errorData["details"] = e.Details
	}

	if e.Cause != nil {
		if cerr, ok := e.Cause.(*ConnectionError); ok {
			errorData["cause"] = map[string]interface{}{
				"code":    cerr.Code,
				"type":    cerr.Type,
				"message": cerr.Message,
				"details": cerr.Details,
			}
		} else {
			errorData["cause"] = map[string]interface{}{
				"message": e.Cause.Error(),
			}
		}
	}

	if len(e.StackTrace) > 0 {
		errorData["stack_trace"] = e.StackTrace
	}

	if !e.Timestamp.IsZero() {
		errorData["timestamp"] = e.Timestamp.Format(time.RFC3339Nano)
	}

	if e.GoroutineID > 0 {
		errorData["goroutine_id"] = e.GoroutineID
	}

	b, _ := json.MarshalIndent(errorData, "", "  ")
	return string(b)
}

// Unwrap returns the underlying cause error for errors.Is and errors.As compatibility.
func (e *ConnectionError) Unwrap() error {
	return e.Cause
}

// ProtocolError represents protocol-level errors (malformed responses, etc).
type ProtocolError struct {
	Code       string                 `json:"code"`
	Type       string                 `json:"type"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details"`
	Cause      error                  `json:"cause,omitempty"`
	StackTrace []string               `json:"stack_trace,omitempty"`
	Timestamp  time.Time              `json:"timestamp,omitempty"`
}

// Error implements the error interface.
// Returns JSON format for backward compatibility.
func (e *ProtocolError) Error() string {
	errorData := map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
	}

	if len(e.Details) > 0 {
		errorData["details"] = e.Details
	}

	if e.Cause != nil {
		errorData["cause"] = map[string]interface{}{
			"message": e.Cause.Error(),
		}
	}

	b, _ := json.Marshal(errorData)
	return string(b)
}

// FormatError formats the error based on debug mode.
func (e *ProtocolError) FormatError(debugMode bool) string {
	if !debugMode {
		if e.Cause != nil {
			return fmt.Sprintf("%s: %s (caused by: %s)", e.Code, e.Message, e.Cause.Error())
		}
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	errorData := map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
	}

	if len(e.Details) > 0 {
		errorData["details"] = e.Details
	}

	if e.Cause != nil {
		errorData["cause"] = map[string]interface{}{"message": e.Cause.Error()}
	}

	if len(e.StackTrace) > 0 {
		errorData["stack_trace"] = e.StackTrace
	}

	if !e.Timestamp.IsZero() {
		errorData["timestamp"] = e.Timestamp.Format(time.RFC3339Nano)
	}

	b, _ := json.MarshalIndent(errorData, "", "  ")
	return string(b)
}

// Unwrap returns the underlying cause error.
func (e *ProtocolError) Unwrap() error {
	return e.Cause
}

// StateError represents invalid state for an operation.
type StateError struct {
	Code       string                 `json:"code"`
	Type       string                 `json:"type"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details"`
	StackTrace []string               `json:"stack_trace,omitempty"`
}

// Error implements the error interface.
// Returns JSON format for backward compatibility.
func (e *StateError) Error() string {
	errorData := map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
	}

	if len(e.Details) > 0 {
		errorData["details"] = e.Details
	}

	b, _ := json.Marshal(errorData)
	return string(b)
}

// FormatError formats the error based on debug mode.
func (e *StateError) FormatError(debugMode bool) string {
	if !debugMode {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	errorData := map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
		"details": e.Details,
	}

	if len(e.StackTrace) > 0 {
		errorData["stack_trace"] = e.StackTrace
	}

	b, _ := json.MarshalIndent(errorData, "", "  ")
	return string(b)
}

// ErrInvalidState creates a StateError for operations attempted in wrong state.
func ErrInvalidState(operation string, required, actual ConnectionState) error {
	return &StateError{
		Code:    "INVALID_STATE",
		Type:    "STATE_ERROR",
		Message: fmt.Sprintf("%s requires %s state, currently %s", operation, required, actual),
		Details: map[string]interface{}{
			"operation":     operation,
			"requiredState": required.String(),
			"currentState":  actual.String(),
		},
		StackTrace: captureStackTrace(),
	}
}

// QueryError represents query execution errors with parameter context.
type QueryError struct {
	Code       string                 `json:"code"`
	Type       string                 `json:"type"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details"`
	Query      string                 `json:"query,omitempty"`
	Params     []interface{}          `json:"params,omitempty"`
	Cause      error                  `json:"cause,omitempty"`
	StackTrace []string               `json:"stack_trace,omitempty"`
	Timestamp  time.Time              `json:"timestamp,omitempty"`
}

// Error implements the error interface.
func (e *QueryError) Error() string {
	return e.FormatError(false)
}

// FormatError formats the error based on debug mode.
func (e *QueryError) FormatError(debugMode bool) string {
	if !debugMode {
		if e.Cause != nil {
			return fmt.Sprintf("%s: %s (caused by: %s)", e.Code, e.Message, e.Cause.Error())
		}
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	errorData := map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
	}

	if e.Query != "" {
		errorData["query"] = e.Query
	}

	if len(e.Params) > 0 {
		errorData["params"] = e.Params
	}

	if len(e.Details) > 0 {
		errorData["details"] = e.Details
	}

	if e.Cause != nil {
		errorData["cause"] = map[string]interface{}{"message": e.Cause.Error()}
	}

	if len(e.StackTrace) > 0 {
		errorData["stack_trace"] = e.StackTrace
	}

	if !e.Timestamp.IsZero() {
		errorData["timestamp"] = e.Timestamp.Format(time.RFC3339Nano)
	}

	b, _ := json.MarshalIndent(errorData, "", "  ")
	return string(b)
}

// Unwrap returns the underlying cause error.
func (e *QueryError) Unwrap() error {
	return e.Cause
}

// StatementError represents prepared statement errors.
type StatementError struct {
	QueryError
	StatementName string `json:"statement_name,omitempty"`
}

// Error implements the error interface for StatementError.
func (e *StatementError) Error() string {
	return e.FormatError(false)
}

// FormatError formats the error based on debug mode.
func (e *StatementError) FormatError(debugMode bool) string {
	if !debugMode {
		return fmt.Sprintf("%s: %s (statement: %s)", e.Code, e.Message, e.StatementName)
	}

	errorData := map[string]interface{}{
		"code":           e.Code,
		"type":           "STATEMENT_ERROR",
		"message":        e.Message,
		"statement_name": e.StatementName,
	}

	if e.Query != "" {
		errorData["query"] = e.Query
	}

	if len(e.Details) > 0 {
		errorData["details"] = e.Details
	}

	if len(e.StackTrace) > 0 {
		errorData["stack_trace"] = e.StackTrace
	}

	b, _ := json.MarshalIndent(errorData, "", "  ")
	return string(b)
}

// TransactionError represents transaction-related errors.
type TransactionError struct {
	Code          string                 `json:"code"`
	Type          string                 `json:"type"`
	Message       string                 `json:"message"`
	Details       map[string]interface{} `json:"details"`
	TransactionID string                 `json:"transaction_id,omitempty"`
	State         string                 `json:"state,omitempty"`
	Cause         error                  `json:"cause,omitempty"`
	StackTrace    []string               `json:"stack_trace,omitempty"`
	Timestamp     time.Time              `json:"timestamp,omitempty"`
}

// Error implements the error interface.
func (e *TransactionError) Error() string {
	return e.FormatError(false)
}

// FormatError formats the error based on debug mode.
func (e *TransactionError) FormatError(debugMode bool) string {
	if !debugMode {
		if e.Cause != nil {
			return fmt.Sprintf("%s: %s (TX: %s, caused by: %s)", e.Code, e.Message, e.TransactionID, e.Cause.Error())
		}
		return fmt.Sprintf("%s: %s (TX: %s)", e.Code, e.Message, e.TransactionID)
	}

	errorData := map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
	}

	if e.TransactionID != "" {
		errorData["transaction_id"] = e.TransactionID
	}

	if e.State != "" {
		errorData["state"] = e.State
	}

	if len(e.Details) > 0 {
		errorData["details"] = e.Details
	}

	if e.Cause != nil {
		errorData["cause"] = map[string]interface{}{"message": e.Cause.Error()}
	}

	if len(e.StackTrace) > 0 {
		errorData["stack_trace"] = e.StackTrace
	}

	if !e.Timestamp.IsZero() {
		errorData["timestamp"] = e.Timestamp.Format(time.RFC3339Nano)
	}

	b, _ := json.MarshalIndent(errorData, "", "  ")
	return string(b)
}

// Unwrap returns the underlying cause error.
func (e *TransactionError) Unwrap() error {
	return e.Cause
}

// ErrInvalidParameterCount creates an error for parameter count mismatches.
func ErrInvalidParameterCount(expected, actual int) *QueryError {
	return &QueryError{
		Code:    "E_PARAM_COUNT_MISMATCH",
		Type:    "QUERY_ERROR",
		Message: fmt.Sprintf("parameter count mismatch: expected %d, got %d", expected, actual),
		Details: map[string]interface{}{
			"expected": expected,
			"actual":   actual,
		},
		StackTrace: captureStackTrace(),
		Timestamp:  time.Now(),
	}
}

// ErrStatementNotFound creates an error when a prepared statement doesn't exist.
func ErrStatementNotFound(name string) *StatementError {
	return &StatementError{
		QueryError: QueryError{
			Code:    "E_STMT_NOT_FOUND",
			Type:    "STATEMENT_ERROR",
			Message: fmt.Sprintf("prepared statement '%s' does not exist", name),
			Details: map[string]interface{}{
				"statement_name": name,
			},
			StackTrace: captureStackTrace(),
			Timestamp:  time.Now(),
		},
		StatementName: name,
	}
}

// ErrTransactionAlreadyActive creates an error when trying to begin a transaction while one is already active.
func ErrTransactionAlreadyActive(id string) *TransactionError {
	return &TransactionError{
		Code:          "E_TX_ALREADY_ACTIVE",
		Type:          "TRANSACTION_ERROR",
		Message:       "transaction already in progress",
		TransactionID: id,
		State:         "active",
		StackTrace:    captureStackTrace(),
		Timestamp:     time.Now(),
	}
}

// ErrNoActiveTransaction creates an error when trying to commit/rollback without an active transaction.
func ErrNoActiveTransaction(operation string) *TransactionError {
	return &TransactionError{
		Code:    "E_NO_ACTIVE_TX",
		Type:    "TRANSACTION_ERROR",
		Message: fmt.Sprintf("no active transaction to %s", operation),
		Details: map[string]interface{}{
			"operation": operation,
		},
		StackTrace: captureStackTrace(),
		Timestamp:  time.Now(),
	}
}

// ErrTransactionAlreadyCommitted creates an error for double-commit attempts.
func ErrTransactionAlreadyCommitted(id string) *TransactionError {
	return &TransactionError{
		Code:          "E_TX_ALREADY_COMMITTED",
		Type:          "TRANSACTION_ERROR",
		Message:       "transaction has already been committed",
		TransactionID: id,
		State:         "committed",
		StackTrace:    captureStackTrace(),
		Timestamp:     time.Now(),
	}
}

// ErrTransactionAlreadyRolledBack creates an error for operations on rolled-back transactions.
func ErrTransactionAlreadyRolledBack(id string) *TransactionError {
	return &TransactionError{
		Code:          "E_TX_ALREADY_ROLLEDBACK",
		Type:          "TRANSACTION_ERROR",
		Message:       "transaction has already been rolled back",
		TransactionID: id,
		State:         "rolledback",
		StackTrace:    captureStackTrace(),
		Timestamp:     time.Now(),
	}
}

// ErrTransactionTimeout creates an error for abandoned transactions.
func ErrTransactionTimeout(id string, duration int64) *TransactionError {
	return &TransactionError{
		Code:          "E_TX_TIMEOUT",
		Type:          "TRANSACTION_ERROR",
		Message:       "transaction exceeded timeout and was rolled back",
		TransactionID: id,
		State:         "timedout",
		Details: map[string]interface{}{
			"duration_ms": duration,
		},
		StackTrace: captureStackTrace(),
		Timestamp:  time.Now(),
	}
}

// Helper functions

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

// getGoroutineID extracts the goroutine ID for debugging.
// Note: This uses runtime stack parsing and is intended for debug purposes only.
func getGoroutineID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// Stack trace format: "goroutine <id> [<status>]:"
	// Extract the ID
	var id int
	fmt.Sscanf(string(buf[:n]), "goroutine %d ", &id)
	return id
}

// FormatError is a helper to format any error with debug mode support.
func FormatError(err error, debugMode bool) string {
	if err == nil {
		return ""
	}

	// Check if error implements our custom format interface
	type debugFormatter interface {
		FormatError(bool) string
	}

	if formatter, ok := err.(debugFormatter); ok {
		return formatter.FormatError(debugMode)
	}

	// Fallback to standard error string
	return err.Error()
}
