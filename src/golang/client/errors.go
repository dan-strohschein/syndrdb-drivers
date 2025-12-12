package client

import (
	"encoding/json"
	"fmt"
)

// ConnectionError represents connection-related failures.
type ConnectionError struct {
	Code    string                 `json:"code"`
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
	Cause   error                  `json:"cause,omitempty"`
}

// Error implements the error interface.
// Returns JSON representation if DebugMode is enabled globally, otherwise flattened message.
func (e *ConnectionError) Error() string {
	// TODO: Access global debug mode setting
	// For now, always return JSON format
	if e.Cause != nil {
		// Full chain with cause
		type errorJSON struct {
			Code    string                 `json:"code"`
			Type    string                 `json:"type"`
			Message string                 `json:"message"`
			Details map[string]interface{} `json:"details,omitempty"`
			Cause   interface{}            `json:"cause,omitempty"`
		}

		var cause interface{}
		if cerr, ok := e.Cause.(*ConnectionError); ok {
			// Recursive JSON serialization
			cause = map[string]interface{}{
				"code":    cerr.Code,
				"type":    cerr.Type,
				"message": cerr.Message,
				"details": cerr.Details,
			}
		} else {
			cause = map[string]interface{}{
				"message": e.Cause.Error(),
			}
		}

		errJSON := errorJSON{
			Code:    e.Code,
			Type:    e.Type,
			Message: e.Message,
			Details: e.Details,
			Cause:   cause,
		}
		b, _ := json.Marshal(errJSON)
		return string(b)
	}

	// Flattened without cause
	b, _ := json.Marshal(map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
		"details": e.Details,
	})
	return string(b)
}

// Unwrap returns the underlying cause error for errors.Is and errors.As compatibility.
func (e *ConnectionError) Unwrap() error {
	return e.Cause
}

// ProtocolError represents protocol-level errors (malformed responses, etc).
type ProtocolError struct {
	Code    string                 `json:"code"`
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
	Cause   error                  `json:"cause,omitempty"`
}

// Error implements the error interface.
func (e *ProtocolError) Error() string {
	if e.Cause != nil {
		b, _ := json.Marshal(map[string]interface{}{
			"code":    e.Code,
			"type":    e.Type,
			"message": e.Message,
			"details": e.Details,
			"cause":   map[string]interface{}{"message": e.Cause.Error()},
		})
		return string(b)
	}

	b, _ := json.Marshal(map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
		"details": e.Details,
	})
	return string(b)
}

// Unwrap returns the underlying cause error.
func (e *ProtocolError) Unwrap() error {
	return e.Cause
}

// StateError represents invalid state for an operation.
type StateError struct {
	Code    string                 `json:"code"`
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
}

// Error implements the error interface.
func (e *StateError) Error() string {
	b, _ := json.Marshal(map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
		"details": e.Details,
	})
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
	}
}

// QueryError represents query execution errors with parameter context.
type QueryError struct {
	Code    string                 `json:"code"`
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
	Query   string                 `json:"query,omitempty"`
	Params  []interface{}          `json:"params,omitempty"`
	Cause   error                  `json:"cause,omitempty"`
}

// Error implements the error interface.
func (e *QueryError) Error() string {
	if e.Cause != nil {
		b, _ := json.Marshal(map[string]interface{}{
			"code":    e.Code,
			"type":    e.Type,
			"message": e.Message,
			"details": e.Details,
			"query":   e.Query,
			"params":  e.Params,
			"cause":   map[string]interface{}{"message": e.Cause.Error()},
		})
		return string(b)
	}

	b, _ := json.Marshal(map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
		"details": e.Details,
		"query":   e.Query,
		"params":  e.Params,
	})
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
	b, _ := json.Marshal(map[string]interface{}{
		"code":           e.Code,
		"type":           "StatementError",
		"message":        e.Message,
		"details":        e.Details,
		"statement_name": e.StatementName,
		"query":          e.Query,
	})
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
}

// Error implements the error interface.
func (e *TransactionError) Error() string {
	if e.Cause != nil {
		b, _ := json.Marshal(map[string]interface{}{
			"code":           e.Code,
			"type":           e.Type,
			"message":        e.Message,
			"details":        e.Details,
			"transaction_id": e.TransactionID,
			"state":          e.State,
			"cause":          map[string]interface{}{"message": e.Cause.Error()},
		})
		return string(b)
	}

	b, _ := json.Marshal(map[string]interface{}{
		"code":           e.Code,
		"type":           e.Type,
		"message":        e.Message,
		"details":        e.Details,
		"transaction_id": e.TransactionID,
		"state":          e.State,
	})
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
		Type:    "QueryError",
		Message: fmt.Sprintf("parameter count mismatch: expected %d, got %d", expected, actual),
		Details: map[string]interface{}{
			"expected": expected,
			"actual":   actual,
		},
	}
}

// ErrStatementNotFound creates an error when a prepared statement doesn't exist.
func ErrStatementNotFound(name string) *StatementError {
	return &StatementError{
		QueryError: QueryError{
			Code:    "E_STMT_NOT_FOUND",
			Type:    "StatementError",
			Message: fmt.Sprintf("prepared statement '%s' does not exist", name),
			Details: map[string]interface{}{
				"statement_name": name,
			},
		},
		StatementName: name,
	}
}

// ErrTransactionAlreadyActive creates an error when trying to begin a transaction while one is already active.
func ErrTransactionAlreadyActive(id string) *TransactionError {
	return &TransactionError{
		Code:          "E_TX_ALREADY_ACTIVE",
		Type:          "TransactionError",
		Message:       "transaction already in progress",
		TransactionID: id,
		State:         "active",
	}
}

// ErrNoActiveTransaction creates an error when trying to commit/rollback without an active transaction.
func ErrNoActiveTransaction(operation string) *TransactionError {
	return &TransactionError{
		Code:    "E_NO_ACTIVE_TX",
		Type:    "TransactionError",
		Message: fmt.Sprintf("no active transaction to %s", operation),
		Details: map[string]interface{}{
			"operation": operation,
		},
	}
}

// ErrTransactionAlreadyCommitted creates an error for double-commit attempts.
func ErrTransactionAlreadyCommitted(id string) *TransactionError {
	return &TransactionError{
		Code:          "E_TX_ALREADY_COMMITTED",
		Type:          "TransactionError",
		Message:       "transaction has already been committed",
		TransactionID: id,
		State:         "committed",
	}
}

// ErrTransactionAlreadyRolledBack creates an error for operations on rolled-back transactions.
func ErrTransactionAlreadyRolledBack(id string) *TransactionError {
	return &TransactionError{
		Code:          "E_TX_ALREADY_ROLLEDBACK",
		Type:          "TransactionError",
		Message:       "transaction has already been rolled back",
		TransactionID: id,
		State:         "rolledback",
	}
}

// ErrTransactionTimeout creates an error for abandoned transactions.
func ErrTransactionTimeout(id string, duration int64) *TransactionError {
	return &TransactionError{
		Code:          "E_TX_TIMEOUT",
		Type:          "TransactionError",
		Message:       "transaction exceeded timeout and was rolled back",
		TransactionID: id,
		State:         "timedout",
		Details: map[string]interface{}{
			"duration_ms": duration,
		},
	}
}
