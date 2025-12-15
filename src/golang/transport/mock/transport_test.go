package mock

import (
	"context"
	"testing"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/protocol"
)

func TestMockTransport_Send(t *testing.T) {
	mock := NewMockTransport()
	ctx := context.Background()
	data := []byte("test data")

	err := mock.Send(ctx, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mock.GetSendCallCount() != 1 {
		t.Errorf("expected 1 send call, got %d", mock.GetSendCallCount())
	}

	history := mock.GetSendHistory()
	if len(history) != 1 {
		t.Errorf("expected 1 item in history, got %d", len(history))
	}
	if string(history[0]) != string(data) {
		t.Errorf("expected %q in history, got %q", data, history[0])
	}
}

func TestMockTransport_SendError(t *testing.T) {
	mock := NewMockTransport().WithSendError(protocol.ConnectionError("test error", nil))
	ctx := context.Background()

	err := mock.Send(ctx, []byte("test"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	metrics := mock.GetMetrics()
	if metrics.TotalErrors != 1 {
		t.Errorf("expected 1 error, got %d", metrics.TotalErrors)
	}
}

func TestMockTransport_SendWithDelay(t *testing.T) {
	mock := NewMockTransport().WithSendDelay(50 * time.Millisecond)
	ctx := context.Background()

	start := time.Now()
	err := mock.Send(ctx, []byte("test"))
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if duration < 50*time.Millisecond {
		t.Errorf("expected delay of at least 50ms, got %v", duration)
	}
}

func TestMockTransport_SendContextCancellation(t *testing.T) {
	mock := NewMockTransport().WithSendDelay(100 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := mock.Send(ctx, []byte("test"))
	if err == nil {
		t.Fatal("expected context deadline exceeded error")
	}
}

func TestMockTransport_Receive(t *testing.T) {
	expectedData := []byte("response data")
	mock := NewMockTransport().WithReceiveData(expectedData)
	ctx := context.Background()

	data, err := mock.Receive(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if string(data) != string(expectedData) {
		t.Errorf("expected %q, got %q", expectedData, data)
	}

	if mock.GetReceiveCallCount() != 1 {
		t.Errorf("expected 1 receive call, got %d", mock.GetReceiveCallCount())
	}
}

func TestMockTransport_ReceiveError(t *testing.T) {
	mock := NewMockTransport().WithReceiveError(protocol.TimeoutError("test timeout", nil))
	ctx := context.Background()

	_, err := mock.Receive(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMockTransport_ReceiveNoData(t *testing.T) {
	mock := NewMockTransport()
	ctx := context.Background()

	_, err := mock.Receive(ctx)
	if err == nil {
		t.Fatal("expected error when no data configured, got nil")
	}
}

func TestMockTransport_Close(t *testing.T) {
	mock := NewMockTransport()

	err := mock.Close()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !mock.IsClosed() {
		t.Error("expected transport to be closed")
	}

	if mock.GetCloseCallCount() != 1 {
		t.Errorf("expected 1 close call, got %d", mock.GetCloseCallCount())
	}

	// Operations after close should fail
	err = mock.Send(context.Background(), []byte("test"))
	if err == nil {
		t.Error("expected error when sending to closed transport")
	}
}

func TestMockTransport_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		healthy  bool
		expected bool
	}{
		{"healthy transport", true, true},
		{"unhealthy transport", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockTransport().WithHealthy(tt.healthy)
			result := mock.IsHealthy()

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMockTransport_GetQueueDepth(t *testing.T) {
	mock := NewMockTransport().WithQueueDepth(42)

	depth := mock.GetQueueDepth()
	if depth != 42 {
		t.Errorf("expected queue depth 42, got %d", depth)
	}
}

func TestMockTransport_GetMetrics(t *testing.T) {
	mock := NewMockTransport().WithReceiveData([]byte("test"))

	// Perform some operations
	mock.Send(context.Background(), []byte("hello"))
	mock.Send(context.Background(), []byte("world"))
	mock.Receive(context.Background())
	mock.IsHealthy()

	metrics := mock.GetMetrics()

	if metrics.TotalRequests != 2 {
		t.Errorf("expected 2 requests, got %d", metrics.TotalRequests)
	}

	if metrics.BytesSent != 10 { // "hello" + "world"
		t.Errorf("expected 10 bytes sent, got %d", metrics.BytesSent)
	}

	if metrics.BytesReceived != 4 { // "test"
		t.Errorf("expected 4 bytes received, got %d", metrics.BytesReceived)
	}

	if metrics.HealthChecksPassed != 1 {
		t.Errorf("expected 1 health check passed, got %d", metrics.HealthChecksPassed)
	}
}

func TestMockTransport_Reset(t *testing.T) {
	mock := NewMockTransport().
		WithSendError(protocol.ConnectionError("error", nil)).
		WithReceiveData([]byte("data")).
		WithQueueDepth(10)

	// Perform some operations
	mock.Send(context.Background(), []byte("test"))
	mock.Close()

	// Reset
	mock.Reset()

	// Verify reset state
	if mock.GetSendCallCount() != 0 {
		t.Errorf("expected 0 send calls after reset, got %d", mock.GetSendCallCount())
	}

	if mock.IsClosed() {
		t.Error("expected transport to not be closed after reset")
	}

	if mock.GetQueueDepth() != 0 {
		t.Errorf("expected queue depth 0 after reset, got %d", mock.GetQueueDepth())
	}

	if len(mock.GetSendHistory()) != 0 {
		t.Errorf("expected empty send history after reset, got %d items", len(mock.GetSendHistory()))
	}
}

func TestMockTransport_Chaining(t *testing.T) {
	// Test that configuration methods can be chained
	mock := NewMockTransport().
		WithSendDelay(10 * time.Millisecond).
		WithReceiveDelay(20 * time.Millisecond).
		WithHealthy(true).
		WithQueueDepth(5)

	if mock.GetQueueDepth() != 5 {
		t.Errorf("expected queue depth 5, got %d", mock.GetQueueDepth())
	}

	if !mock.IsHealthy() {
		t.Error("expected healthy transport")
	}
}
