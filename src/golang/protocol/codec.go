// Package protocol provides encoding/decoding for SyndrDB wire protocol
package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
)

const (
	// EOT is the End of Transmission character used for message framing
	EOT byte = 0x04

	// ENQ is the Enquiry character used for parameter delimiter
	ENQ byte = 0x05

	// PROTOCOL_VERSION is the current wire protocol version
	PROTOCOL_VERSION = 2
)

// Codec handles encoding and decoding of protocol messages
type Codec interface {
	// Encode encodes a command with optional parameters into wire format
	Encode(command string, params []string) []byte

	// Decode parses a raw message into a Response
	Decode(data []byte) (*Response, error)

	// EncodeVersionHandshake creates the protocol version message
	EncodeVersionHandshake() []byte

	// DecodeVersionResponse parses the server's version response
	DecodeVersionResponse(data []byte) error
}

// Response represents a decoded protocol response
type Response struct {
	Data    interface{}            `json:"data,omitempty"`
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// SyndrDBCodec implements the SyndrDB wire protocol codec
type SyndrDBCodec struct {
	// Buffer pool for encoding operations
	bufferPool sync.Pool
}

// NewCodec creates a new SyndrDB protocol codec
func NewCodec() Codec {
	return &SyndrDBCodec{
		bufferPool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Encode encodes a command with optional parameters
func (c *SyndrDBCodec) Encode(command string, params []string) []byte {
	buf := c.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer c.bufferPool.Put(buf)

	// Write command
	buf.WriteString(command)

	// Write parameters if present
	if len(params) > 0 {
		for _, param := range params {
			buf.WriteByte(ENQ)
			// Escape EOT and ENQ in parameters
			escaped := escapeParameter(param)
			buf.WriteString(escaped)
		}
	}

	// Add EOT terminator
	buf.WriteByte(EOT)

	// Return a copy since we're reusing the buffer
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result
}

// escapeParameter escapes EOT and ENQ characters in parameter values
func escapeParameter(param string) string {
	// Fast path: no escaping needed
	needsEscape := false
	for i := 0; i < len(param); i++ {
		if param[i] == EOT || param[i] == ENQ {
			needsEscape = true
			break
		}
	}
	if !needsEscape {
		return param
	}

	// Slow path: escape special characters
	var buf bytes.Buffer
	buf.Grow(len(param) + 10) // Pre-allocate with some extra space
	for i := 0; i < len(param); i++ {
		b := param[i]
		if b == EOT {
			buf.WriteByte(EOT)
			buf.WriteByte(EOT) // Double EOT
		} else if b == ENQ {
			buf.WriteByte(ENQ)
			buf.WriteByte(ENQ) // Double ENQ
		} else {
			buf.WriteByte(b)
		}
	}
	return buf.String()
}

// Decode parses a raw message into a Response
func (c *SyndrDBCodec) Decode(data []byte) (*Response, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty response data")
	}

	// Remove trailing EOT if present
	if data[len(data)-1] == EOT {
		data = data[:len(data)-1]
	}

	// Try to parse as JSON
	var response Response
	if err := json.Unmarshal(data, &response); err != nil {
		// If JSON parsing fails, treat as plain text message
		return &Response{
			Success: true,
			Message: string(data),
		}, nil
	}

	return &response, nil
}

// EncodeVersionHandshake creates the protocol version message
func (c *SyndrDBCodec) EncodeVersionHandshake() []byte {
	return []byte(fmt.Sprintf("PROTOCOL_VERSION %d%c", PROTOCOL_VERSION, EOT))
}

// DecodeVersionResponse parses the server's version response
func (c *SyndrDBCodec) DecodeVersionResponse(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty version response")
	}

	// Remove trailing EOT if present
	if data[len(data)-1] == EOT {
		data = data[:len(data)-1]
	}

	msg := string(data)

	// Check for success response
	if len(msg) >= 11 && msg[:11] == "PROTOCOL_OK" {
		// Expected format: "PROTOCOL_OK 2"
		return nil
	}

	// Check for error response
	if len(msg) >= 14 && msg[:14] == "PROTOCOL_ERROR" {
		// Expected format: "PROTOCOL_ERROR unsupported_version"
		return &ProtocolVersionError{
			Message: msg[15:], // Skip "PROTOCOL_ERROR "
		}
	}

	return fmt.Errorf("unexpected version response: %s", msg)
}

// ProtocolVersionError indicates a protocol version mismatch
type ProtocolVersionError struct {
	Message string
}

func (e *ProtocolVersionError) Error() string {
	return fmt.Sprintf("protocol version mismatch: %s", e.Message)
}
