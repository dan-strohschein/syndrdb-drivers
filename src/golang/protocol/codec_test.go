package protocol

import (
	"testing"
)

func TestCodecEncode(t *testing.T) {
	codec := NewCodec()

	tests := []struct {
		name     string
		command  string
		params   []string
		expected string
	}{
		{
			name:     "simple command",
			command:  "SELECT * FROM users",
			params:   nil,
			expected: "SELECT * FROM users\x04",
		},
		{
			name:     "command with one parameter",
			command:  "EXECUTE stmt",
			params:   []string{"value1"},
			expected: "EXECUTE stmt\x05value1\x04",
		},
		{
			name:     "command with multiple parameters",
			command:  "EXECUTE stmt",
			params:   []string{"value1", "value2", "value3"},
			expected: "EXECUTE stmt\x05value1\x05value2\x05value3\x04",
		},
		{
			name:     "parameter with EOT needs escaping",
			command:  "EXECUTE stmt",
			params:   []string{"value\x04with\x04eot"},
			expected: "EXECUTE stmt\x05value\x04\x04with\x04\x04eot\x04",
		},
		{
			name:     "parameter with ENQ needs escaping",
			command:  "EXECUTE stmt",
			params:   []string{"value\x05with\x05enq"},
			expected: "EXECUTE stmt\x05value\x05\x05with\x05\x05enq\x04",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := codec.Encode(tt.command, tt.params)
			if string(result) != tt.expected {
				t.Errorf("Encode() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestCodecDecode(t *testing.T) {
	codec := NewCodec()

	tests := []struct {
		name        string
		input       []byte
		wantSuccess bool
		wantMessage string
	}{
		{
			name:        "JSON response with data",
			input:       []byte(`{"success":true,"data":{"id":1,"name":"test"}}`),
			wantSuccess: true,
		},
		{
			name:        "JSON response with error",
			input:       []byte(`{"success":false,"error":"test error"}`),
			wantSuccess: false,
		},
		{
			name:        "plain text response",
			input:       []byte(`S0001:: Welcome to SyndrDB`),
			wantSuccess: true,
			wantMessage: "S0001:: Welcome to SyndrDB",
		},
		{
			name:        "response with trailing EOT",
			input:       []byte("OK\x04"),
			wantSuccess: true,
			wantMessage: "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := codec.Decode(tt.input)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if resp.Success != tt.wantSuccess {
				t.Errorf("Decode() success = %v, want %v", resp.Success, tt.wantSuccess)
			}
			if tt.wantMessage != "" && resp.Message != tt.wantMessage {
				t.Errorf("Decode() message = %q, want %q", resp.Message, tt.wantMessage)
			}
		})
	}
}

func TestVersionHandshake(t *testing.T) {
	codec := NewCodec()

	// Test encoding version handshake
	encoded := codec.EncodeVersionHandshake()
	expected := "PROTOCOL_VERSION 2\x04"
	if string(encoded) != expected {
		t.Errorf("EncodeVersionHandshake() = %q, want %q", string(encoded), expected)
	}

	// Test decoding successful version response
	t.Run("successful version response", func(t *testing.T) {
		response := []byte("PROTOCOL_OK 2\x04")
		err := codec.DecodeVersionResponse(response)
		if err != nil {
			t.Errorf("DecodeVersionResponse() error = %v, want nil", err)
		}
	})

	// Test decoding version mismatch
	t.Run("version mismatch", func(t *testing.T) {
		response := []byte("PROTOCOL_ERROR unsupported_version\x04")
		err := codec.DecodeVersionResponse(response)
		if err == nil {
			t.Error("DecodeVersionResponse() error = nil, want error")
		}
		if _, ok := err.(*ProtocolVersionError); !ok {
			t.Errorf("DecodeVersionResponse() error type = %T, want *ProtocolVersionError", err)
		}
	})
}

func TestEscapeParameter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no escaping needed",
			input:    "normal text",
			expected: "normal text",
		},
		{
			name:     "escape EOT",
			input:    "text\x04with\x04eot",
			expected: "text\x04\x04with\x04\x04eot",
		},
		{
			name:     "escape ENQ",
			input:    "text\x05with\x05enq",
			expected: "text\x05\x05with\x05\x05enq",
		},
		{
			name:     "escape both",
			input:    "\x04\x05mixed\x04\x05",
			expected: "\x04\x04\x05\x05mixed\x04\x04\x05\x05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeParameter(tt.input)
			if result != tt.expected {
				t.Errorf("escapeParameter() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func BenchmarkCodecEncode(b *testing.B) {
	codec := NewCodec()
	command := "SELECT * FROM users WHERE age > $1 AND name = $2"
	params := []string{"25", "John Doe"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Encode(command, params)
	}
}

func BenchmarkCodecDecode(b *testing.B) {
	codec := NewCodec()
	data := []byte(`{"success":true,"data":{"id":1,"name":"test","email":"test@example.com"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Decode(data)
	}
}
