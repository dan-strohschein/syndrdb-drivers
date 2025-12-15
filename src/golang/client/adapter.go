package client

import (
	"context"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/protocol"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/transport"
)

// TransportConnection adapts transport.Transport to ConnectionInterface
// This allows gradual migration to the new architecture
type TransportConnection struct {
	transport transport.Transport
	codec     protocol.Codec
	addr      string
	alive     bool
	lastUsed  time.Time
}

// NewTransportConnection creates a ConnectionInterface from a Transport
func NewTransportConnection(t transport.Transport, addr string) ConnectionInterface {
	return &TransportConnection{
		transport: t,
		codec:     protocol.NewCodec(),
		addr:      addr,
		alive:     true,
		lastUsed:  time.Now(),
	}
}

// SendCommand implements ConnectionInterface.SendCommand
func (tc *TransportConnection) SendCommand(ctx context.Context, command string) error {
	// Encode command using protocol codec
	encoded := tc.codec.Encode(command, nil)

	// Send via transport
	err := tc.transport.Send(ctx, encoded)
	if err != nil {
		tc.alive = false
		return err
	}

	tc.lastUsed = time.Now()
	return nil
}

// ReceiveResponse implements ConnectionInterface.ReceiveResponse
func (tc *TransportConnection) ReceiveResponse(ctx context.Context) (interface{}, error) {
	// Receive raw bytes via transport
	data, err := tc.transport.Receive(ctx)
	if err != nil {
		tc.alive = false
		return nil, err
	}

	// Decode using protocol codec
	response, err := tc.codec.Decode(data)
	if err != nil {
		return nil, err
	}

	tc.lastUsed = time.Now()

	// Check for error in response
	if response.Error != "" {
		// Create error from response error string
		return response.Data, &ConnectionError{
			Code:    response.Code,
			Type:    "PROTOCOL_ERROR",
			Message: response.Error,
			Details: response.Details,
		}
	}

	// Return Data if present, otherwise return Message (for plain text responses)
	if response.Data != nil {
		return response.Data, nil
	}
	if response.Message != "" {
		return response.Message, nil
	}

	return nil, nil
}

// Ping implements ConnectionInterface.Ping
func (tc *TransportConnection) Ping(ctx context.Context) error {
	if !tc.transport.IsHealthy() {
		tc.alive = false
		return &ConnectionError{
			Code:    "CONNECTION_UNHEALTHY",
			Type:    "CONNECTION_ERROR",
			Message: "transport is not healthy",
		}
	}

	// Send a simple PING command
	return tc.SendCommand(ctx, "PING")
}

// Close implements ConnectionInterface.Close
func (tc *TransportConnection) Close() error {
	tc.alive = false
	return tc.transport.Close()
}

// RemoteAddr implements ConnectionInterface.RemoteAddr
func (tc *TransportConnection) RemoteAddr() string {
	return tc.addr
}

// IsAlive implements ConnectionInterface.IsAlive
func (tc *TransportConnection) IsAlive() bool {
	return tc.alive && tc.transport.IsHealthy()
}

// LastActivity implements ConnectionInterface.LastActivity
func (tc *TransportConnection) LastActivity() time.Time {
	return tc.lastUsed
}
