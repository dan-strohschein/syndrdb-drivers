package client

import (
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.DefaultTimeoutMs != 10000 {
		t.Errorf("expected DefaultTimeoutMs=10000, got %d", opts.DefaultTimeoutMs)
	}

	if opts.DebugMode != false {
		t.Errorf("expected DebugMode=false, got %v", opts.DebugMode)
	}

	if opts.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", opts.MaxRetries)
	}
}

func TestCustomOptions(t *testing.T) {
	opts := ClientOptions{
		DefaultTimeoutMs: 5000,
		DebugMode:        true,
		MaxRetries:       5,
	}

	if opts.DefaultTimeoutMs != 5000 {
		t.Errorf("expected DefaultTimeoutMs=5000, got %d", opts.DefaultTimeoutMs)
	}

	if opts.DebugMode != true {
		t.Errorf("expected DebugMode=true, got %v", opts.DebugMode)
	}

	if opts.MaxRetries != 5 {
		t.Errorf("expected MaxRetries=5, got %d", opts.MaxRetries)
	}
}
