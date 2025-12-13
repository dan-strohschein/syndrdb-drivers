//go:build milestone2

package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
)

// MockClient is a mock implementation of the SyndrDB client for testing.
// It provides a fluent API for setting up expectations and verifying calls.
//
// Example usage:
//
//	mock := NewMockClient()
//	mock.ExpectQuery("SELECT * FROM users").
//	    WillReturn(map[string]interface{}{"users": []interface{}{...}})
//
//	result, err := mock.Query(ctx, "SELECT * FROM users", 0)
//	mock.VerifyExpectations(t)
type MockClient struct {
	expectations []*Expectation
	calls        []Call
	mu           sync.RWMutex
	strict       bool // If true, unexpected calls will panic
}

// Expectation represents an expected method call and its response.
type Expectation struct {
	method      string // "Query", "Mutate", "Connect", etc.
	command     string // SQL command (for Query/Mutate)
	response    interface{}
	err         error
	times       int  // Expected number of calls (-1 = any)
	actualCalls int  // Actual number of calls
	matched     bool // Whether this expectation was matched
}

// Call represents an actual method call that was made.
type Call struct {
	Method  string
	Command string
	Args    []interface{}
}

// NewMockClient creates a new mock client for testing.
func NewMockClient() *MockClient {
	return &MockClient{
		expectations: make([]*Expectation, 0),
		calls:        make([]Call, 0),
		strict:       false,
	}
}

// Strict enables strict mode where unexpected calls will panic.
// This helps catch unintended interactions during tests.
func (m *MockClient) Strict() *MockClient {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strict = true
	return m
}

// ExpectQuery sets up an expectation for a Query call.
// Returns the expectation for chaining WillReturn/WillReturnError.
func (m *MockClient) ExpectQuery(command string) *Expectation {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp := &Expectation{
		method:  "Query",
		command: command,
		times:   1,
	}
	m.expectations = append(m.expectations, exp)
	return exp
}

// ExpectMutate sets up an expectation for a Mutate call.
func (m *MockClient) ExpectMutate(command string) *Expectation {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp := &Expectation{
		method:  "Mutate",
		command: command,
		times:   1,
	}
	m.expectations = append(m.expectations, exp)
	return exp
}

// ExpectConnect sets up an expectation for a Connect call.
func (m *MockClient) ExpectConnect() *Expectation {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp := &Expectation{
		method: "Connect",
		times:  1,
	}
	m.expectations = append(m.expectations, exp)
	return exp
}

// ExpectDisconnect sets up an expectation for a Disconnect call.
func (m *MockClient) ExpectDisconnect() *Expectation {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp := &Expectation{
		method: "Disconnect",
		times:  1,
	}
	m.expectations = append(m.expectations, exp)
	return exp
}

// ExpectPing sets up an expectation for a Ping call.
func (m *MockClient) ExpectPing() *Expectation {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp := &Expectation{
		method: "Ping",
		times:  1,
	}
	m.expectations = append(m.expectations, exp)
	return exp
}

// WillReturn sets the return value for this expectation.
// Can be chained after ExpectQuery/ExpectMutate.
func (e *Expectation) WillReturn(response interface{}) *Expectation {
	e.response = response
	return e
}

// WillReturnError sets the error to return for this expectation.
func (e *Expectation) WillReturnError(err error) *Expectation {
	e.err = err
	return e
}

// WillReturnJSON sets a JSON response that will be marshaled.
// Useful for testing JSON responses.
func (e *Expectation) WillReturnJSON(v interface{}) *Expectation {
	e.response = v
	return e
}

// Times sets the expected number of times this call should occur.
// Use -1 for "any number of times".
func (e *Expectation) Times(n int) *Expectation {
	e.times = n
	return e
}

// Once is a shorthand for Times(1).
func (e *Expectation) Once() *Expectation {
	return e.Times(1)
}

// Twice is a shorthand for Times(2).
func (e *Expectation) Twice() *Expectation {
	return e.Times(2)
}

// AnyTimes allows this expectation to match any number of times.
func (e *Expectation) AnyTimes() *Expectation {
	return e.Times(-1)
}

// Query implements the Query method for the mock client.
func (m *MockClient) Query(ctx context.Context, command string, timeout int) (interface{}, error) {
	m.recordCall("Query", command)

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find matching expectation
	for _, exp := range m.expectations {
		if exp.method == "Query" && exp.command == command {
			if exp.times == -1 || exp.actualCalls < exp.times {
				exp.actualCalls++
				exp.matched = true
				if exp.err != nil {
					return nil, exp.err
				}
				return exp.response, nil
			}
		}
	}

	// No matching expectation found
	if m.strict {
		panic(fmt.Sprintf("unexpected Query call: %s", command))
	}
	return nil, fmt.Errorf("no expectation set for Query: %s", command)
}

// Mutate implements the Mutate method for the mock client.
func (m *MockClient) Mutate(ctx context.Context, command string, timeout int) (interface{}, error) {
	m.recordCall("Mutate", command)

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, exp := range m.expectations {
		if exp.method == "Mutate" && exp.command == command {
			if exp.times == -1 || exp.actualCalls < exp.times {
				exp.actualCalls++
				exp.matched = true
				if exp.err != nil {
					return nil, exp.err
				}
				return exp.response, nil
			}
		}
	}

	if m.strict {
		panic(fmt.Sprintf("unexpected Mutate call: %s", command))
	}
	return nil, fmt.Errorf("no expectation set for Mutate: %s", command)
}

// Connect implements the Connect method for the mock client.
func (m *MockClient) Connect(ctx context.Context, connStr string) error {
	m.recordCall("Connect", connStr)

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, exp := range m.expectations {
		if exp.method == "Connect" {
			if exp.times == -1 || exp.actualCalls < exp.times {
				exp.actualCalls++
				exp.matched = true
				return exp.err
			}
		}
	}

	if m.strict {
		panic("unexpected Connect call")
	}
	return nil
}

// Disconnect implements the Disconnect method for the mock client.
func (m *MockClient) Disconnect(ctx context.Context) error {
	m.recordCall("Disconnect", "")

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, exp := range m.expectations {
		if exp.method == "Disconnect" {
			if exp.times == -1 || exp.actualCalls < exp.times {
				exp.actualCalls++
				exp.matched = true
				return exp.err
			}
		}
	}

	if m.strict {
		panic("unexpected Disconnect call")
	}
	return nil
}

// Ping implements the Ping method for the mock client.
func (m *MockClient) Ping(ctx context.Context) error {
	m.recordCall("Ping", "")

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, exp := range m.expectations {
		if exp.method == "Ping" {
			if exp.times == -1 || exp.actualCalls < exp.times {
				exp.actualCalls++
				exp.matched = true
				return exp.err
			}
		}
	}

	if m.strict {
		panic("unexpected Ping call")
	}
	return nil
}

// GetState returns a mock connection state.
func (m *MockClient) GetState() client.ConnectionState {
	return client.CONNECTED
}

// VerifyExpectations checks that all expectations were met.
// Should be called at the end of each test.
func (m *MockClient) VerifyExpectations(t *testing.T) {
	t.Helper()
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i, exp := range m.expectations {
		if exp.times != -1 && exp.actualCalls != exp.times {
			t.Errorf("expectation %d (%s %s): expected %d calls, got %d",
				i, exp.method, exp.command, exp.times, exp.actualCalls)
		}
	}
}

// AssertExpectations is an alias for VerifyExpectations (Jest-style naming).
func (m *MockClient) AssertExpectations(t *testing.T) {
	m.VerifyExpectations(t)
}

// GetCalls returns all recorded method calls.
// Useful for custom assertions.
func (m *MockClient) GetCalls() []Call {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Call{}, m.calls...)
}

// GetCallCount returns the number of times a method was called.
func (m *MockClient) GetCallCount(method string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, call := range m.calls {
		if call.Method == method {
			count++
		}
	}
	return count
}

// Reset clears all expectations and recorded calls.
// Useful when reusing a mock across multiple test cases.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expectations = make([]*Expectation, 0)
	m.calls = make([]Call, 0)
}

// recordCall adds a call to the call history.
func (m *MockClient) recordCall(method, command string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, Call{
		Method:  method,
		Command: command,
	})
}

// ToJSON is a helper that converts an object to JSON for testing.
// Useful with WillReturn to create JSON responses.
func ToJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// MockResponse creates a properly formatted mock response.
// This is useful for mocking complex query results.
func MockResponse(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"data": data,
	}
}

// MockError creates a mock error response.
func MockError(code, message string) error {
	return fmt.Errorf(`{"code": "%s", "message": "%s"}`, code, message)
}
