package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConnectionError(t *testing.T) {
	err := &ConnectionError{
		Code:    "CONNECTION_FAILED",
		Type:    "CONNECTION_ERROR",
		Message: "failed to connect",
		Details: map[string]interface{}{
			"address": "localhost:7632",
		},
	}

	errStr := err.Error()

	// Should be valid JSON
	var parsed map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(errStr), &parsed); jsonErr != nil {
		t.Fatalf("error should be valid JSON: %v", jsonErr)
	}

	if parsed["code"] != "CONNECTION_FAILED" {
		t.Errorf("expected code=CONNECTION_FAILED, got %v", parsed["code"])
	}

	if parsed["type"] != "CONNECTION_ERROR" {
		t.Errorf("expected type=CONNECTION_ERROR, got %v", parsed["type"])
	}

	if parsed["message"] != "failed to connect" {
		t.Errorf("expected message='failed to connect', got %v", parsed["message"])
	}
}

func TestConnectionErrorWithCause(t *testing.T) {
	cause := &ConnectionError{
		Code:    "NETWORK_ERROR",
		Type:    "CONNECTION_ERROR",
		Message: "connection refused",
		Details: map[string]interface{}{},
	}

	err := &ConnectionError{
		Code:    "CONNECTION_FAILED",
		Type:    "CONNECTION_ERROR",
		Message: "failed to connect",
		Details: map[string]interface{}{},
		Cause:   cause,
	}

	errStr := err.Error()

	// Should contain cause
	if !strings.Contains(errStr, "cause") {
		t.Errorf("error should contain cause, got: %s", errStr)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(errStr), &parsed)

	if parsed["cause"] == nil {
		t.Error("expected cause field in JSON")
	}
}

func TestConnectionErrorUnwrap(t *testing.T) {
	cause := &ConnectionError{
		Code:    "NETWORK_ERROR",
		Type:    "CONNECTION_ERROR",
		Message: "connection refused",
		Details: map[string]interface{}{},
	}

	err := &ConnectionError{
		Code:    "CONNECTION_FAILED",
		Type:    "CONNECTION_ERROR",
		Message: "failed to connect",
		Details: map[string]interface{}{},
		Cause:   cause,
	}

	unwrapped := err.Unwrap()

	if unwrapped != cause {
		t.Errorf("expected unwrapped to be cause, got %v", unwrapped)
	}
}

func TestProtocolError(t *testing.T) {
	err := &ProtocolError{
		Code:    "PROTOCOL_ERROR",
		Type:    "PROTOCOL_ERROR",
		Message: "malformed response",
		Details: map[string]interface{}{
			"response": "invalid",
		},
	}

	errStr := err.Error()

	var parsed map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(errStr), &parsed); jsonErr != nil {
		t.Fatalf("error should be valid JSON: %v", jsonErr)
	}

	if parsed["code"] != "PROTOCOL_ERROR" {
		t.Errorf("expected code=PROTOCOL_ERROR, got %v", parsed["code"])
	}
}

func TestStateError(t *testing.T) {
	err := &StateError{
		Code:    "INVALID_STATE",
		Type:    "STATE_ERROR",
		Message: "invalid state",
		Details: map[string]interface{}{
			"operation":     "Query",
			"requiredState": "CONNECTED",
			"currentState":  "DISCONNECTED",
		},
	}

	errStr := err.Error()

	var parsed map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(errStr), &parsed); jsonErr != nil {
		t.Fatalf("error should be valid JSON: %v", jsonErr)
	}

	details := parsed["details"].(map[string]interface{})
	if details["operation"] != "Query" {
		t.Errorf("expected operation=Query, got %v", details["operation"])
	}
}

func TestErrInvalidState(t *testing.T) {
	err := ErrInvalidState("Query", CONNECTED, DISCONNECTED)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	stateErr, ok := err.(*StateError)
	if !ok {
		t.Fatalf("expected *StateError, got %T", err)
	}

	if stateErr.Code != "INVALID_STATE" {
		t.Errorf("expected code=INVALID_STATE, got %s", stateErr.Code)
	}

	details := stateErr.Details
	if details["operation"] != "Query" {
		t.Errorf("expected operation=Query, got %v", details["operation"])
	}

	if details["requiredState"] != "CONNECTED" {
		t.Errorf("expected requiredState=CONNECTED, got %v", details["requiredState"])
	}

	if details["currentState"] != "DISCONNECTED" {
		t.Errorf("expected currentState=DISCONNECTED, got %v", details["currentState"])
	}
}
