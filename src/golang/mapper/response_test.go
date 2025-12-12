package mapper

import (
	"testing"
)

func TestResponseMapper_ToString(t *testing.T) {
	mapper := NewResponseMapper()

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"float", 3.14, "3.140000"},
		{"bool", true, "true"},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapper.ToString(tt.input)
			if got != tt.expected {
				t.Errorf("ToString() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResponseMapper_ToInt(t *testing.T) {
	mapper := NewResponseMapper()

	tests := []struct {
		name     string
		input    interface{}
		expected int64
		wantErr  bool
	}{
		{"int", 42, 42, false},
		{"int32", int32(42), 42, false},
		{"int64", int64(42), 42, false},
		{"float64", 42.0, 42, false},
		{"string valid", "42", 42, false},
		{"string invalid", "not a number", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapper.ToInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ToInt() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResponseMapper_ToBool(t *testing.T) {
	mapper := NewResponseMapper()

	tests := []struct {
		name     string
		input    interface{}
		expected bool
		wantErr  bool
	}{
		{"bool true", true, true, false},
		{"bool false", false, false, false},
		{"string true", "true", true, false},
		{"string false", "false", false, false},
		{"string 1", "1", true, false},
		{"string 0", "0", false, false},
		{"int 1", 1, true, false},
		{"int 0", 0, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapper.ToBool(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ToBool() = %v, want %v", got, tt.expected)
			}
		})
	}
}
