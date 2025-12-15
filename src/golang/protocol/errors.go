// Package protocol provides error codes and types for SyndrDB protocol
package protocol

import (
	"encoding/json"
	"fmt"
)

// ErrorCode represents standardized error codes across transport layers
type ErrorCode int

const (
	// Connection errors (1000-1099)
	ErrorCodeConnectionRefused       ErrorCode = 1001
	ErrorCodeTimeout                 ErrorCode = 1002
	ErrorCodeAuthFailed              ErrorCode = 1003
	ErrorCodeProtocolVersionMismatch ErrorCode = 1004
	ErrorCodeBackpressure            ErrorCode = 1010

	// Protocol errors (2000-2099)
	ErrorCodeProtocolError ErrorCode = 2001

	// Query errors (3000-3099)
	ErrorCodeQueryError ErrorCode = 3001

	// Bridge errors (9000-9999)
	ErrorCodeBridgeBusy            ErrorCode = 9001
	ErrorCodeBridgeCallbackMissing ErrorCode = 9002
	ErrorCodeBridgeInitFailed      ErrorCode = 9999
)

// TransportError represents an error with structured error code
type TransportError struct {
	Code        ErrorCode              `json:"code"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	IsRetryable bool                   `json:"isRetryable"`
}

// Error implements the error interface
func (e *TransportError) Error() string {
	if len(e.Details) > 0 {
		detailsJSON, _ := json.Marshal(e.Details)
		return fmt.Sprintf("[%d] %s (details: %s)", e.Code, e.Message, string(detailsJSON))
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewTransportError creates a new transport error
func NewTransportError(code ErrorCode, message string, details map[string]interface{}) *TransportError {
	return &TransportError{
		Code:        code,
		Message:     message,
		Details:     details,
		IsRetryable: isRetryable(code),
	}
}

// isRetryable determines if an error code represents a retryable error
func isRetryable(code ErrorCode) bool {
	switch code {
	case ErrorCodeTimeout,
		ErrorCodeBackpressure,
		ErrorCodeBridgeBusy,
		ErrorCodeBridgeCallbackMissing:
		return true
	default:
		return false
	}
}

// ConnectionError creates a connection-related transport error
func ConnectionError(message string, details map[string]interface{}) *TransportError {
	return NewTransportError(ErrorCodeConnectionRefused, message, details)
}

// TimeoutError creates a timeout transport error
func TimeoutError(message string, details map[string]interface{}) *TransportError {
	return NewTransportError(ErrorCodeTimeout, message, details)
}

// AuthError creates an authentication transport error
func AuthError(message string, details map[string]interface{}) *TransportError {
	return NewTransportError(ErrorCodeAuthFailed, message, details)
}

// ProtocolVersionError creates a protocol version mismatch error
func ProtocolVersionMismatchError(message string, details map[string]interface{}) *TransportError {
	return NewTransportError(ErrorCodeProtocolVersionMismatch, message, details)
}

// BackpressureError creates a backpressure transport error
func BackpressureError(queueDepth int) *TransportError {
	return NewTransportError(ErrorCodeBackpressure, "message queue full", map[string]interface{}{
		"queueDepth": queueDepth,
	})
}

// BridgeBusyError creates a bridge busy error
func BridgeBusyError() *TransportError {
	return NewTransportError(ErrorCodeBridgeBusy, "bridge busy, retry later", nil)
}

// BridgeCallbackMissingError creates a bridge callback missing error
func BridgeCallbackMissingError(callback string) *TransportError {
	return NewTransportError(ErrorCodeBridgeCallbackMissing, "bridge callback missing", map[string]interface{}{
		"callback": callback,
	})
}

// BridgeInitError creates a bridge initialization error
func BridgeInitError(message string) *TransportError {
	return NewTransportError(ErrorCodeBridgeInitFailed, message, nil)
}

// ToJSON serializes the error to JSON for cross-language transmission
func (e *TransportError) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON deserializes a transport error from JSON
func FromJSON(data []byte) (*TransportError, error) {
	var err TransportError
	if unmarshalErr := json.Unmarshal(data, &err); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	return &err, nil
}
