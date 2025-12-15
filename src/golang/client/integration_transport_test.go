package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/protocol"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/transport/mock"
)

// TestNewArchitecture_BasicQueryFlow demonstrates the new transport architecture
func TestNewArchitecture_BasicQueryFlow(t *testing.T) {
	// Step 1: Create a mock transport
	mockTransport := mock.NewMockTransport()

	// Step 2: Configure mock responses for the connection flow
	welcomeResponse := []byte("S0001 Welcome to SyndrDB" + string(byte(0x04)))
	queryResponse := []byte(`{"status": "success", "data": [{"id": 1, "name": "test"}]}` + string(byte(0x04)))

	// Queue responses in order
	mockTransport.WithReceiveData(welcomeResponse)

	// Step 3: Create a transport connection adapter
	conn := client.NewTransportConnection(mockTransport, "localhost:1776")

	// Step 4: Verify basic operations work
	ctx := context.Background()

	// Send a query
	err := conn.SendCommand(ctx, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("SendCommand failed: %v", err)
	}

	// Receive response
	mockTransport.WithReceiveData(queryResponse)
	response, err := conn.ReceiveResponse(ctx)
	if err != nil {
		t.Fatalf("ReceiveResponse failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}

	// Verify metrics
	metrics := mockTransport.GetMetrics()
	if metrics.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", metrics.TotalRequests)
	}
}

// TestNewArchitecture_ErrorHandling demonstrates error handling with new architecture
func TestNewArchitecture_ErrorHandling(t *testing.T) {
	mockTransport := mock.NewMockTransport()

	// Configure transport to return an error
	mockTransport.WithSendError(protocol.ConnectionError("connection refused", map[string]interface{}{
		"host": "localhost",
		"port": 1776,
	}))

	conn := client.NewTransportConnection(mockTransport, "localhost:1776")
	ctx := context.Background()

	// Attempt to send
	err := conn.SendCommand(ctx, "SELECT 1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify connection is marked as dead
	if conn.IsAlive() {
		t.Error("connection should not be alive after error")
	}

	// Verify error metrics
	metrics := mockTransport.GetMetrics()
	if metrics.TotalErrors != 1 {
		t.Errorf("expected 1 error, got %d", metrics.TotalErrors)
	}
}

// TestNewArchitecture_Retries demonstrates retry logic with transient errors
func TestNewArchitecture_Retries(t *testing.T) {
	mockTransport := mock.NewMockTransport()

	// First two attempts fail, third succeeds
	var callCount int
	originalSend := mockTransport.Send
	mockTransport.WithSendError(protocol.TimeoutError("timeout", nil))

	conn := client.NewTransportConnection(mockTransport, "localhost:1776")
	ctx := context.Background()

	// First attempt fails
	err := conn.SendCommand(ctx, "SELECT 1")
	if err == nil {
		t.Fatal("expected error on first attempt")
	}
	callCount++

	// Reset mock for retry - this time succeeds
	mockTransport.Reset()
	mockTransport.WithReceiveData([]byte(`{"status": "success"}` + string(byte(0x04))))

	// Recreate connection for second attempt
	_ = client.NewTransportConnection(mockTransport, "localhost:1776")
	err = conn.SendCommand(ctx, "SELECT 1")
	if err != nil {
		t.Fatalf("expected success on retry, got %v", err)
	}

	_ = originalSend
}

// TestNewArchitecture_HealthChecks demonstrates health check functionality
func TestNewArchitecture_HealthChecks(t *testing.T) {
	tests := []struct {
		name        string
		healthy     bool
		expectError bool
	}{
		{"healthy connection", true, false},
		{"unhealthy connection", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := mock.NewMockTransport().WithHealthy(tt.healthy)
			conn := client.NewTransportConnection(mockTransport, "localhost:1776")

			// Ping should use health check
			err := conn.Ping(context.Background())

			if tt.expectError && err == nil {
				t.Error("expected error for unhealthy connection")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error for healthy connection, got %v", err)
			}

			// Verify health check metrics
			metrics := mockTransport.GetMetrics()
			if tt.healthy {
				if metrics.HealthChecksPassed == 0 {
					t.Error("expected health check to pass")
				}
			} else {
				if metrics.HealthChecksFailed == 0 {
					t.Error("expected health check to fail")
				}
			}
		})
	}
}

// TestNewArchitecture_Backpressure demonstrates backpressure handling
func TestNewArchitecture_Backpressure(t *testing.T) {
	mockTransport := mock.NewMockTransport()

	// Simulate high queue depth
	mockTransport.WithQueueDepth(85) // Above 80% threshold

	// Queue depth should be visible
	depth := mockTransport.GetQueueDepth()
	if depth != 85 {
		t.Errorf("expected queue depth 85, got %d", depth)
	}

	// In a real scenario, high queue depth would trigger backpressure
	// The transport implementation would reject new requests
	// This is a demonstration of the metric being tracked
}

// TestNewArchitecture_Metrics demonstrates comprehensive metrics tracking
func TestNewArchitecture_Metrics(t *testing.T) {
	mockTransport := mock.NewMockTransport()
	successResponse := []byte(`{"status": "success"}` + string(byte(0x04)))
	mockTransport.WithReceiveData(successResponse)

	conn := client.NewTransportConnection(mockTransport, "localhost:1776")
	ctx := context.Background()

	// Perform multiple operations
	for i := 0; i < 5; i++ {
		conn.SendCommand(ctx, "SELECT 1")
		conn.ReceiveResponse(ctx)
	}

	// Check health
	conn.IsAlive()

	// Verify metrics
	metrics := mockTransport.GetMetrics()

	if metrics.TotalRequests != 5 {
		t.Errorf("expected 5 requests, got %d", metrics.TotalRequests)
	}

	if metrics.BytesSent == 0 {
		t.Error("expected bytes sent to be tracked")
	}

	if metrics.BytesReceived == 0 {
		t.Error("expected bytes received to be tracked")
	}

	if metrics.HealthChecksPassed == 0 {
		t.Error("expected health checks to be tracked")
	}
}

// TestNewArchitecture_ContextCancellation demonstrates context handling
func TestNewArchitecture_ContextCancellation(t *testing.T) {
	mockTransport := mock.NewMockTransport()

	// Add delay to allow cancellation
	mockTransport.WithSendDelay(100 * time.Millisecond)

	conn := client.NewTransportConnection(mockTransport, "localhost:1776")

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	// This should fail due to context cancellation
	err := conn.SendCommand(ctx, "SELECT 1")
	if err == nil {
		t.Fatal("expected context deadline exceeded error")
	}

	// Error should be context.DeadlineExceeded
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", ctx.Err())
	}
}

// TestNewArchitecture_ConnectionPooling demonstrates connection pooling concepts
func TestNewArchitecture_ConnectionPooling(t *testing.T) {
	// Create multiple connections with different transports
	connections := make([]client.ConnectionInterface, 5)

	for i := 0; i < 5; i++ {
		mockTransport := mock.NewMockTransport()
		successResponse := []byte(`{"status": "success"}` + string(byte(0x04)))
		mockTransport.WithReceiveData(successResponse)

		connections[i] = client.NewTransportConnection(mockTransport, "localhost:1776")
	}

	// All connections should be independent
	for i, conn := range connections {
		if !conn.IsAlive() {
			t.Errorf("connection %d should be alive", i)
		}
	}

	// Close one connection
	connections[0].Close()

	// Others should still be alive
	for i := 1; i < 5; i++ {
		if !connections[i].IsAlive() {
			t.Errorf("connection %d should still be alive", i)
		}
	}
}

// TestNewArchitecture_MultipleTransports demonstrates using different transports
func TestNewArchitecture_MultipleTransports(t *testing.T) {
	// Simulate TCP transport
	tcpTransport := mock.NewMockTransport()
	tcpTransport.WithReceiveData([]byte(`{"source": "tcp"}` + string(byte(0x04))))
	tcpConn := client.NewTransportConnection(tcpTransport, "tcp:1776")

	// Simulate WASM transport
	wasmTransport := mock.NewMockTransport()
	wasmTransport.WithReceiveData([]byte(`{"source": "wasm"}` + string(byte(0x04))))
	wasmConn := client.NewTransportConnection(wasmTransport, "wasm:bridge")

	// Both should work identically
	ctx := context.Background()

	tcpConn.SendCommand(ctx, "SELECT 1")
	tcpResp, err := tcpConn.ReceiveResponse(ctx)
	if err != nil {
		t.Fatalf("TCP connection failed: %v", err)
	}

	wasmConn.SendCommand(ctx, "SELECT 1")
	wasmResp, err := wasmConn.ReceiveResponse(ctx)
	if err != nil {
		t.Fatalf("WASM connection failed: %v", err)
	}

	// Both should return valid responses
	// Both should complete without errors
	_ = tcpResp
	_ = wasmResp
}

// BenchmarkNewArchitecture_SendReceive benchmarks the new architecture
func BenchmarkNewArchitecture_SendReceive(b *testing.B) {
	mockTransport := mock.NewMockTransport()
	successResponse := []byte(`{"status": "success"}` + string(byte(0x04)))
	mockTransport.WithReceiveData(successResponse)

	conn := client.NewTransportConnection(mockTransport, "localhost:1776")
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn.SendCommand(ctx, "SELECT 1")
		conn.ReceiveResponse(ctx)
	}
}

// BenchmarkNewArchitecture_ProtocolEncoding benchmarks protocol encoding overhead
func BenchmarkNewArchitecture_ProtocolEncoding(b *testing.B) {
	mockTransport := mock.NewMockTransport()
	successResponse := []byte(`{"status": "success"}` + string(byte(0x04)))
	mockTransport.WithReceiveData(successResponse)

	conn := client.NewTransportConnection(mockTransport, "localhost:1776")
	ctx := context.Background()

	command := "SELECT * FROM users WHERE id = 1"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn.SendCommand(ctx, command)
	}
}
