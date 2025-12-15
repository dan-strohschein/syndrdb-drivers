// Package transport defines the transport layer abstraction for SyndrDB
package transport

import (
	"context"
	"time"
)

// Transport defines the interface for sending and receiving messages
type Transport interface {
	// Send transmits data to the server
	Send(ctx context.Context, data []byte) error

	// Receive reads data from the server
	Receive(ctx context.Context) ([]byte, error)

	// Close closes the transport connection
	Close() error

	// IsHealthy returns whether the transport is healthy
	IsHealthy() bool

	// GetQueueDepth returns the current message queue depth (for backpressure monitoring)
	GetQueueDepth() int

	// GetMetrics returns transport performance metrics
	GetMetrics() TransportMetrics
}

// TransportMetrics contains performance and health metrics
type TransportMetrics struct {
	// TotalRequests is the total number of requests sent
	TotalRequests int64

	// TotalErrors is the total number of errors encountered
	TotalErrors int64

	// AverageLatency is the average round-trip latency
	AverageLatency time.Duration

	// LastError is the most recent error encountered
	LastError error

	// LastErrorTime is when the last error occurred
	LastErrorTime time.Time

	// BytesSent is the total bytes sent
	BytesSent int64

	// BytesReceived is the total bytes received
	BytesReceived int64

	// ConnectionsCreated is the total number of connections created
	ConnectionsCreated int64

	// ConnectionsActive is the current number of active connections
	ConnectionsActive int

	// QueueDepth is the current message queue depth
	QueueDepth int

	// HealthChecksPassed is the number of successful health checks
	HealthChecksPassed int64

	// HealthChecksFailed is the number of failed health checks
	HealthChecksFailed int64
}

// Factory creates new transport instances
type Factory func(ctx context.Context) (Transport, error)
