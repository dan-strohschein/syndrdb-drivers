package client

import (
	"fmt"
	"sync"
	"time"
)

// ConnectionState represents the current state of the client connection.
type ConnectionState int

const (
	// DISCONNECTED indicates no active connection.
	DISCONNECTED ConnectionState = iota
	// CONNECTING indicates connection attempt in progress.
	CONNECTING
	// CONNECTED indicates active, established connection.
	CONNECTED
	// DISCONNECTING indicates graceful disconnect in progress.
	DISCONNECTING
)

// String returns the string representation of the connection state.
func (cs ConnectionState) String() string {
	switch cs {
	case DISCONNECTED:
		return "DISCONNECTED"
	case CONNECTING:
		return "CONNECTING"
	case CONNECTED:
		return "CONNECTED"
	case DISCONNECTING:
		return "DISCONNECTING"
	default:
		return "UNKNOWN"
	}
}

// StateTransition represents a change in connection state with enriched context.
//
// Standard Metadata Keys (conventions for consistency):
//   - reason: string - "user_initiated" | "error" | "timeout" | "server_closed"
//   - attempt: int - Retry attempt number (1-indexed)
//   - remoteAddr: string - Remote server address (e.g., "localhost:7632")
//   - connectionString: string - Original connection string
//
// These are conventions, not enforced. Custom metadata can be added as needed.
type StateTransition struct {
	// From is the previous state.
	From ConnectionState

	// To is the new current state.
	To ConnectionState

	// Timestamp is when the transition occurred.
	Timestamp time.Time

	// Error is the error that caused the transition (if any).
	// Non-nil for failed connection attempts or unexpected disconnects.
	Error error

	// Duration is how long the previous state was held.
	Duration time.Duration

	// Metadata contains additional context about the transition.
	// See StateTransition godoc for standard key conventions.
	Metadata map[string]interface{}
}

// StateChangeHandler is called when the connection state changes.
type StateChangeHandler func(transition StateTransition)

// StateManager manages connection state transitions and event handlers.
type StateManager struct {
	current        ConnectionState
	lastTransition time.Time
	handlers       []StateChangeHandler
	mu             sync.RWMutex
}

// NewStateManager creates a new state manager in DISCONNECTED state.
func NewStateManager() *StateManager {
	return &StateManager{
		current:        DISCONNECTED,
		lastTransition: time.Now(),
		handlers:       make([]StateChangeHandler, 0),
	}
}

// TransitionTo attempts to transition to a new state.
// Returns error if the transition is illegal.
//
// Legal transitions:
//   - DISCONNECTED → CONNECTING
//   - CONNECTING → CONNECTED
//   - CONNECTING → DISCONNECTED (failed connection)
//   - CONNECTED → DISCONNECTING
//   - DISCONNECTING → DISCONNECTED
func (sm *StateManager) TransitionTo(newState ConnectionState, err error, metadata map[string]interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate transition legality
	if !sm.isLegalTransition(sm.current, newState) {
		return fmt.Errorf("illegal state transition: %s → %s", sm.current, newState)
	}

	// Calculate duration in previous state
	now := time.Now()
	duration := now.Sub(sm.lastTransition)

	// Create transition event
	transition := StateTransition{
		From:      sm.current,
		To:        newState,
		Timestamp: now,
		Error:     err,
		Duration:  duration,
		Metadata:  metadata,
	}

	// Update state
	sm.current = newState
	sm.lastTransition = now

	// Notify handlers (call without lock to prevent deadlocks)
	handlers := make([]StateChangeHandler, len(sm.handlers))
	copy(handlers, sm.handlers)

	sm.mu.Unlock()
	for _, handler := range handlers {
		handler(transition)
	}
	sm.mu.Lock()

	return nil
}

// isLegalTransition checks if a state transition is allowed.
func (sm *StateManager) isLegalTransition(from, to ConnectionState) bool {
	switch from {
	case DISCONNECTED:
		return to == CONNECTING
	case CONNECTING:
		return to == CONNECTED || to == DISCONNECTED
	case CONNECTED:
		return to == DISCONNECTING
	case DISCONNECTING:
		return to == DISCONNECTED
	default:
		return false
	}
}

// OnStateChange registers a handler to be called on state transitions.
func (sm *StateManager) OnStateChange(handler StateChangeHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.handlers = append(sm.handlers, handler)
}

// GetState returns the current connection state (thread-safe).
func (sm *StateManager) GetState() ConnectionState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// GetLastTransition returns the most recent state transition.
func (sm *StateManager) GetLastTransition() StateTransition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return StateTransition{
		From:      sm.current, // Current state was the "to" of last transition
		To:        sm.current,
		Timestamp: sm.lastTransition,
		Duration:  time.Since(sm.lastTransition),
	}
}
