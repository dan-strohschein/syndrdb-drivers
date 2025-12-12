package client

import (
	"testing"
	"time"
)

func TestConnectionStateString(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{DISCONNECTED, "DISCONNECTED"},
		{CONNECTING, "CONNECTING"},
		{CONNECTED, "CONNECTED"},
		{DISCONNECTING, "DISCONNECTING"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestNewStateManager(t *testing.T) {
	sm := NewStateManager()

	if sm == nil {
		t.Fatal("NewStateManager returned nil")
	}

	if sm.GetState() != DISCONNECTED {
		t.Errorf("expected initial state DISCONNECTED, got %s", sm.GetState())
	}
}

func TestLegalStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     ConnectionState
		to       ConnectionState
		shouldOK bool
	}{
		{"DISCONNECTED to CONNECTING", DISCONNECTED, CONNECTING, true},
		{"CONNECTING to CONNECTED", CONNECTING, CONNECTED, true},
		{"CONNECTING to DISCONNECTED", CONNECTING, DISCONNECTED, true},
		{"CONNECTED to DISCONNECTING", CONNECTED, DISCONNECTING, true},
		{"DISCONNECTING to DISCONNECTED", DISCONNECTING, DISCONNECTED, true},
		// Illegal transitions
		{"DISCONNECTED to CONNECTED", DISCONNECTED, CONNECTED, false},
		{"DISCONNECTED to DISCONNECTING", DISCONNECTED, DISCONNECTING, false},
		{"CONNECTING to DISCONNECTING", CONNECTING, DISCONNECTING, false},
		{"CONNECTED to CONNECTING", CONNECTED, CONNECTING, false},
		{"CONNECTED to DISCONNECTED", CONNECTED, DISCONNECTED, false},
		{"DISCONNECTING to CONNECTING", DISCONNECTING, CONNECTING, false},
		{"DISCONNECTING to CONNECTED", DISCONNECTING, CONNECTED, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateManager()

			// Set initial state by transitioning to it
			if tt.from != DISCONNECTED {
				// Get to the desired starting state
				switch tt.from {
				case CONNECTING:
					sm.TransitionTo(CONNECTING, nil, nil)
				case CONNECTED:
					sm.TransitionTo(CONNECTING, nil, nil)
					sm.TransitionTo(CONNECTED, nil, nil)
				case DISCONNECTING:
					sm.TransitionTo(CONNECTING, nil, nil)
					sm.TransitionTo(CONNECTED, nil, nil)
					sm.TransitionTo(DISCONNECTING, nil, nil)
				}
			}

			err := sm.TransitionTo(tt.to, nil, nil)

			if tt.shouldOK && err != nil {
				t.Errorf("expected legal transition, got error: %v", err)
			}

			if !tt.shouldOK && err == nil {
				t.Errorf("expected illegal transition error, got none")
			}
		})
	}
}

func TestStateChangeHandlers(t *testing.T) {
	sm := NewStateManager()

	var capturedTransitions []StateTransition

	sm.OnStateChange(func(transition StateTransition) {
		capturedTransitions = append(capturedTransitions, transition)
	})

	// Perform transition
	err := sm.TransitionTo(CONNECTING, nil, map[string]interface{}{
		"reason": "test",
	})

	if err != nil {
		t.Fatalf("transition failed: %v", err)
	}

	if len(capturedTransitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(capturedTransitions))
	}

	trans := capturedTransitions[0]

	if trans.From != DISCONNECTED {
		t.Errorf("expected From=DISCONNECTED, got %s", trans.From)
	}

	if trans.To != CONNECTING {
		t.Errorf("expected To=CONNECTING, got %s", trans.To)
	}

	if reason, ok := trans.Metadata["reason"].(string); !ok || reason != "test" {
		t.Errorf("expected metadata reason='test', got %v", trans.Metadata["reason"])
	}
}

func TestMultipleHandlers(t *testing.T) {
	sm := NewStateManager()

	count1 := 0
	count2 := 0

	sm.OnStateChange(func(transition StateTransition) {
		count1++
	})

	sm.OnStateChange(func(transition StateTransition) {
		count2++
	})

	sm.TransitionTo(CONNECTING, nil, nil)

	if count1 != 1 {
		t.Errorf("expected handler 1 called once, got %d", count1)
	}

	if count2 != 1 {
		t.Errorf("expected handler 2 called once, got %d", count2)
	}
}

func TestTransitionDuration(t *testing.T) {
	sm := NewStateManager()

	var duration time.Duration

	sm.OnStateChange(func(transition StateTransition) {
		duration = transition.Duration
	})

	// Small sleep to ensure measurable duration
	time.Sleep(10 * time.Millisecond)

	sm.TransitionTo(CONNECTING, nil, nil)

	if duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", duration)
	}
}

func TestGetState(t *testing.T) {
	sm := NewStateManager()

	if sm.GetState() != DISCONNECTED {
		t.Errorf("expected DISCONNECTED, got %s", sm.GetState())
	}

	sm.TransitionTo(CONNECTING, nil, nil)

	if sm.GetState() != CONNECTING {
		t.Errorf("expected CONNECTING, got %s", sm.GetState())
	}
}

func TestTransitionWithError(t *testing.T) {
	sm := NewStateManager()

	var capturedError error

	sm.OnStateChange(func(transition StateTransition) {
		capturedError = transition.Error
	})

	testErr := &ConnectionError{
		Code:    "TEST_ERROR",
		Type:    "TEST",
		Message: "test error",
		Details: map[string]interface{}{},
	}

	sm.TransitionTo(CONNECTING, testErr, nil)

	if capturedError == nil {
		t.Fatal("expected error in transition, got nil")
	}

	if capturedError.Error() != testErr.Error() {
		t.Errorf("expected error %v, got %v", testErr, capturedError)
	}
}
