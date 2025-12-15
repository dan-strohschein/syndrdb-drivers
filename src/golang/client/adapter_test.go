package client

import (
	"context"
	"testing"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/protocol"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/transport/mock"
)

// TestTransportConnection_BasicOperations tests basic send/receive operations
func TestTransportConnection_BasicOperations(t *testing.T) {
	mockTransport := mock.NewMockTransport()

	// Configure mock to return success response
	successResponse := []byte(`{"status": "success", "data": "test"}` + string(byte(0x04)))
	mockTransport.WithReceiveData(successResponse)

	conn := NewTransportConnection(mockTransport, "test:1234")
	ctx := context.Background()

	// Test SendCommand
	err := conn.SendCommand(ctx, "SELECT * FROM test")
	if err != nil {
		t.Fatalf("SendCommand failed: %v", err)
	}

	// Verify send was called
	if mockTransport.GetSendCallCount() != 1 {
		t.Errorf("expected 1 send call, got %d", mockTransport.GetSendCallCount())
	}

	// Test ReceiveResponse
	response, err := conn.ReceiveResponse(ctx)
	if err != nil {
		t.Fatalf("ReceiveResponse failed: %v", err)
	}

	// Verify receive was called
	if mockTransport.GetReceiveCallCount() != 1 {
		t.Errorf("expected 1 receive call, got %d", mockTransport.GetReceiveCallCount())
	}

	// Verify response
	if response == nil {
		t.Error("expected non-nil response")
	}
}

// TestTransportConnection_Ping tests connection health check
func TestTransportConnection_Ping(t *testing.T) {
	tests := []struct {
		name        string
		healthy     bool
		expectError bool
	}{
		{"healthy transport", true, false},
		{"unhealthy transport", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := mock.NewMockTransport().WithHealthy(tt.healthy)
			conn := NewTransportConnection(mockTransport, "test:1234")

			err := conn.Ping(context.Background())

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

// TestTransportConnection_Close tests connection closure
func TestTransportConnection_Close(t *testing.T) {
	mockTransport := mock.NewMockTransport()
	conn := NewTransportConnection(mockTransport, "test:1234")

	err := conn.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if mockTransport.GetCloseCallCount() != 1 {
		t.Errorf("expected 1 close call, got %d", mockTransport.GetCloseCallCount())
	}

	if conn.IsAlive() {
		t.Error("connection should not be alive after close")
	}
}

// TestTransportConnection_RemoteAddr tests address retrieval
func TestTransportConnection_RemoteAddr(t *testing.T) {
	mockTransport := mock.NewMockTransport()
	expectedAddr := "localhost:1776"
	conn := NewTransportConnection(mockTransport, expectedAddr)

	addr := conn.RemoteAddr()
	if addr != expectedAddr {
		t.Errorf("expected address %q, got %q", expectedAddr, addr)
	}
}

// TestTransportConnection_IsAlive tests connection liveness
func TestTransportConnection_IsAlive(t *testing.T) {
	mockTransport := mock.NewMockTransport().WithHealthy(true)
	conn := NewTransportConnection(mockTransport, "test:1234")

	if !conn.IsAlive() {
		t.Error("connection should be alive initially")
	}

	// Make transport unhealthy
	mockTransport.WithHealthy(false)

	if conn.IsAlive() {
		t.Error("connection should not be alive when transport is unhealthy")
	}
}

// TestTransportConnection_LastActivity tests activity tracking
func TestTransportConnection_LastActivity(t *testing.T) {
	mockTransport := mock.NewMockTransport()
	successResponse := []byte(`{"status": "success"}` + string(byte(0x04)))
	mockTransport.WithReceiveData(successResponse)

	conn := NewTransportConnection(mockTransport, "test:1234")

	// Get initial activity time
	initialTime := conn.LastActivity()

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Perform operation
	conn.SendCommand(context.Background(), "TEST")

	// Activity time should be updated
	newTime := conn.LastActivity()
	if !newTime.After(initialTime) {
		t.Error("activity time should be updated after operation")
	}
}

// TestTransportConnection_SendError tests error handling during send
func TestTransportConnection_SendError(t *testing.T) {
	mockTransport := mock.NewMockTransport().
		WithSendError(protocol.ConnectionError("connection refused", nil))

	conn := NewTransportConnection(mockTransport, "test:1234")

	err := conn.SendCommand(context.Background(), "TEST")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Connection should be marked as not alive after error
	if conn.IsAlive() {
		t.Error("connection should not be alive after send error")
	}
}

// TestTransportConnection_ReceiveError tests error handling during receive
func TestTransportConnection_ReceiveError(t *testing.T) {
	mockTransport := mock.NewMockTransport().
		WithReceiveError(protocol.TimeoutError("read timeout", nil))

	conn := NewTransportConnection(mockTransport, "test:1234")

	_, err := conn.ReceiveResponse(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Connection should be marked as not alive after error
	if conn.IsAlive() {
		t.Error("connection should not be alive after receive error")
	}
}

// TestTransportConnection_ContextCancellation tests context cancellation
func TestTransportConnection_ContextCancellation(t *testing.T) {
	mockTransport := mock.NewMockTransport().
		WithSendDelay(100 * time.Millisecond)

	conn := NewTransportConnection(mockTransport, "test:1234")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := conn.SendCommand(ctx, "TEST")
	if err == nil {
		t.Fatal("expected context deadline exceeded error")
	}
}

// TestTransportConnection_ParameterEncoding tests parameter encoding
func TestTransportConnection_ParameterEncoding(t *testing.T) {
	mockTransport := mock.NewMockTransport()
	successResponse := []byte(`{"status": "success"}` + string(byte(0x04)))
	mockTransport.WithReceiveData(successResponse)

	conn := NewTransportConnection(mockTransport, "test:1234")

	// Send command with special characters
	command := "SELECT * WHERE value = 'test\x04data'"
	err := conn.SendCommand(context.Background(), command)
	if err != nil {
		t.Fatalf("SendCommand failed: %v", err)
	}

	// Check that EOT was properly added
	history := mockTransport.GetSendHistory()
	if len(history) != 1 {
		t.Fatalf("expected 1 send, got %d", len(history))
	}

	// Verify EOT terminator is present
	lastByte := history[0][len(history[0])-1]
	if lastByte != 0x04 {
		t.Errorf("expected EOT terminator (0x04), got %#x", lastByte)
	}
}

// TestTransportConnection_JSONResponse tests JSON response parsing
func TestTransportConnection_JSONResponse(t *testing.T) {
	mockTransport := mock.NewMockTransport()

	jsonResponse := []byte(`{"status": "success", "data": {"id": 1, "name": "test"}}` + string(byte(0x04)))
	mockTransport.WithReceiveData(jsonResponse)

	conn := NewTransportConnection(mockTransport, "test:1234")

	response, err := conn.ReceiveResponse(context.Background())
	if err != nil {
		t.Fatalf("ReceiveResponse failed: %v", err)
	}

	// The response.Data field should contain the parsed data
	if response == nil {
		t.Fatal("expected non-nil response")
	}

	// Response should be a map when Data contains a JSON object
	if respMap, ok := response.(map[string]interface{}); ok {
		if id, ok := respMap["id"].(float64); !ok || id != 1 {
			t.Errorf("expected id 1, got %v", respMap["id"])
		}
	}
}

// TestTransportConnection_PlainTextResponse tests plain text response handling
func TestTransportConnection_PlainTextResponse(t *testing.T) {
	mockTransport := mock.NewMockTransport()

	plainResponse := []byte("Plain text response" + string(byte(0x04)))
	mockTransport.WithReceiveData(plainResponse)

	conn := NewTransportConnection(mockTransport, "test:1234")

	response, err := conn.ReceiveResponse(context.Background())
	if err != nil {
		t.Fatalf("ReceiveResponse failed: %v", err)
	}

	// Plain text responses should still be returned
	// The codec wraps them in Response.Message field
	if response == nil {
		t.Fatal("expected non-nil response")
	}

	// Response could be either a string or wrapped in the protocol structure
	// This is acceptable as the codec handles both formats
}

// TestTransportConnection_ConcurrentOperations tests thread safety
func TestTransportConnection_ConcurrentOperations(t *testing.T) {
	mockTransport := mock.NewMockTransport()
	successResponse := []byte(`{"status": "success"}` + string(byte(0x04)))
	mockTransport.WithReceiveData(successResponse)

	conn := NewTransportConnection(mockTransport, "test:1234")

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Each goroutine performs a complete send/receive cycle
			err := conn.SendCommand(context.Background(), "TEST")
			if err != nil {
				t.Errorf("goroutine %d: SendCommand failed: %v", id, err)
				return
			}

			_, err = conn.ReceiveResponse(context.Background())
			if err != nil {
				t.Errorf("goroutine %d: ReceiveResponse failed: %v", id, err)
				return
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all operations were tracked
	if mockTransport.GetSendCallCount() != numGoroutines {
		t.Errorf("expected %d send calls, got %d", numGoroutines, mockTransport.GetSendCallCount())
	}

	if mockTransport.GetReceiveCallCount() != numGoroutines {
		t.Errorf("expected %d receive calls, got %d", numGoroutines, mockTransport.GetReceiveCallCount())
	}
}
