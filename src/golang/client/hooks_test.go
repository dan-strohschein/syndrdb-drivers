//go:build milestone2
// +build milestone2

package client

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestHook is a simple hook for testing.
type TestHook struct {
	name         string
	beforeCalled bool
	afterCalled  bool
	beforeError  error
	afterError   error
	modifyCmd    string
}

func (h *TestHook) Name() string {
	return h.name
}

func (h *TestHook) Before(ctx context.Context, hookCtx *HookContext) error {
	h.beforeCalled = true
	if h.modifyCmd != "" {
		hookCtx.Command = h.modifyCmd
	}
	return h.beforeError
}

func (h *TestHook) After(ctx context.Context, hookCtx *HookContext) error {
	h.afterCalled = true
	return h.afterError
}

// TestHookRegistration verifies hooks can be registered and unregistered.
func TestHookRegistration(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	hook1 := &TestHook{name: "hook1"}
	hook2 := &TestHook{name: "hook2"}

	// Register hooks
	client.RegisterHook(hook1)
	client.RegisterHook(hook2)

	hooks := client.GetHooks()
	if len(hooks) != 2 {
		t.Errorf("expected 2 hooks, got %d", len(hooks))
	}

	if hooks[0] != "hook1" || hooks[1] != "hook2" {
		t.Errorf("unexpected hook order: %v", hooks)
	}

	// Unregister hook1
	if !client.UnregisterHook("hook1") {
		t.Error("expected UnregisterHook to return true")
	}

	hooks = client.GetHooks()
	if len(hooks) != 1 {
		t.Errorf("expected 1 hook after unregister, got %d", len(hooks))
	}

	if hooks[0] != "hook2" {
		t.Errorf("expected hook2, got %s", hooks[0])
	}

	// Unregister non-existent hook
	if client.UnregisterHook("nonexistent") {
		t.Error("expected UnregisterHook to return false for non-existent hook")
	}
}

// TestHookReplacement verifies replacing a hook with the same name.
func TestHookReplacement(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	hook1 := &TestHook{name: "test", beforeError: errors.New("error1")}
	hook2 := &TestHook{name: "test", beforeError: errors.New("error2")}

	client.RegisterHook(hook1)
	client.RegisterHook(hook2) // Should replace hook1

	hooks := client.GetHooks()
	if len(hooks) != 1 {
		t.Errorf("expected 1 hook after replacement, got %d", len(hooks))
	}

	// Verify hook2 is active by checking error
	ctx := context.Background()
	hookCtx := &HookContext{Command: "test", Metadata: make(map[string]interface{})}

	err := client.executeBeforeHooks(ctx, hookCtx)
	if err == nil || err.Error() != "error2" {
		t.Errorf("expected error2, got %v", err)
	}
}

// OrderTrackingHook tracks execution order for testing.
type OrderTrackingHook struct {
	name  string
	order *[]string
}

func (h *OrderTrackingHook) Name() string {
	return h.name
}

func (h *OrderTrackingHook) Before(ctx context.Context, hookCtx *HookContext) error {
	*h.order = append(*h.order, h.name)
	return nil
}

func (h *OrderTrackingHook) After(ctx context.Context, hookCtx *HookContext) error {
	return nil
}

// TestHookExecutionOrder verifies hooks execute in FIFO order.
func TestHookExecutionOrder(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	var order []string

	hook1 := &OrderTrackingHook{name: "first", order: &order}
	hook2 := &OrderTrackingHook{name: "second", order: &order}
	hook3 := &OrderTrackingHook{name: "third", order: &order}

	client.RegisterHook(hook1)
	client.RegisterHook(hook2)
	client.RegisterHook(hook3)

	ctx := context.Background()
	hookCtx := &HookContext{Command: "test", Metadata: make(map[string]interface{})}
	client.executeBeforeHooks(ctx, hookCtx)

	if len(order) != 3 {
		t.Errorf("expected 3 hook executions, got %d", len(order))
	}

	if order[0] != "first" || order[1] != "second" || order[2] != "third" {
		t.Errorf("unexpected execution order: %v", order)
	}
}

// TestBeforeHookAbort verifies a Before hook can abort command execution.
func TestBeforeHookAbort(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	hook1 := &TestHook{name: "abort", beforeError: errors.New("aborted")}
	hook2 := &TestHook{name: "never-called"}

	client.RegisterHook(hook1)
	client.RegisterHook(hook2)

	ctx := context.Background()
	hookCtx := &HookContext{Command: "test", Metadata: make(map[string]interface{})}

	err := client.executeBeforeHooks(ctx, hookCtx)
	if err == nil || err.Error() != "aborted" {
		t.Errorf("expected abort error, got %v", err)
	}

	// hook2 should not have been called
	if hook2.beforeCalled {
		t.Error("expected second hook to not be called after abort")
	}
}

// TestAfterHookErrorReplacement verifies After hook errors replace original errors.
func TestAfterHookErrorReplacement(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	hook := &TestHook{name: "replacer", afterError: errors.New("replaced")}
	client.RegisterHook(hook)

	ctx := context.Background()
	hookCtx := &HookContext{
		Command:  "test",
		Metadata: make(map[string]interface{}),
		Error:    errors.New("original"),
	}

	err := client.executeAfterHooks(ctx, hookCtx)
	if err == nil || err.Error() != "replaced" {
		t.Errorf("expected replaced error, got %v", err)
	}
}

// TestAfterHookAllExecute verifies all After hooks execute even if one errors.
func TestAfterHookAllExecute(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	hook1 := &TestHook{name: "first", afterError: errors.New("error1")}
	hook2 := &TestHook{name: "second"}
	hook3 := &TestHook{name: "third", afterError: errors.New("error3")}

	client.RegisterHook(hook1)
	client.RegisterHook(hook2)
	client.RegisterHook(hook3)

	ctx := context.Background()
	hookCtx := &HookContext{Command: "test", Metadata: make(map[string]interface{})}

	err := client.executeAfterHooks(ctx, hookCtx)

	// Should return last error
	if err == nil || err.Error() != "error3" {
		t.Errorf("expected error3, got %v", err)
	}

	// All hooks should have been called
	if !hook1.afterCalled || !hook2.afterCalled || !hook3.afterCalled {
		t.Error("expected all After hooks to be called")
	}
}

// TestHookCommandModification verifies hooks can modify the command.
func TestHookCommandModification(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	hook := &TestHook{name: "modifier", modifyCmd: "SELECT modified"}
	client.RegisterHook(hook)

	ctx := context.Background()
	hookCtx := &HookContext{
		Command:  "SELECT original",
		Metadata: make(map[string]interface{}),
	}

	client.executeBeforeHooks(ctx, hookCtx)

	if hookCtx.Command != "SELECT modified" {
		t.Errorf("expected modified command, got %s", hookCtx.Command)
	}
}

// MetadataBeforeHook writes to metadata.
type MetadataBeforeHook struct{}

func (h *MetadataBeforeHook) Name() string { return "metadata-before" }
func (h *MetadataBeforeHook) Before(ctx context.Context, hookCtx *HookContext) error {
	hookCtx.Metadata["test_key"] = "test_value"
	return nil
}
func (h *MetadataBeforeHook) After(ctx context.Context, hookCtx *HookContext) error { return nil }

// MetadataAfterHook reads from metadata.
type MetadataAfterHook struct{}

func (h *MetadataAfterHook) Name() string                                           { return "metadata-after" }
func (h *MetadataAfterHook) Before(ctx context.Context, hookCtx *HookContext) error { return nil }
func (h *MetadataAfterHook) After(ctx context.Context, hookCtx *HookContext) error {
	value, ok := hookCtx.Metadata["test_key"]
	if !ok || value != "test_value" {
		return errors.New("metadata not found")
	}
	return nil
}

// TestHookMetadata verifies hooks can use metadata for state passing.
func TestHookMetadata(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	client.RegisterHook(&MetadataBeforeHook{})
	client.RegisterHook(&MetadataAfterHook{})

	ctx := context.Background()
	hookCtx := &HookContext{
		Command:  "test",
		Metadata: make(map[string]interface{}),
	}

	client.executeBeforeHooks(ctx, hookCtx)
	err := client.executeAfterHooks(ctx, hookCtx)

	if err != nil {
		t.Errorf("metadata passing failed: %v", err)
	}
}

// TestInferCommandType verifies command type inference.
func TestInferCommandType(t *testing.T) {
	tests := []struct {
		command  string
		expected string
	}{
		{"SELECT * FROM users", "query"},
		{"select * from users", "query"},
		{"INSERT INTO users VALUES (1, 'test')", "mutation"},
		{"UPDATE users SET name='test'", "mutation"},
		{"DELETE FROM users WHERE id=1", "mutation"},
		{"BEGIN TRANSACTION", "transaction"},
		{"COMMIT", "transaction"},
		{"ROLLBACK", "transaction"},
		{"CREATE BUNDLE users", "schema"},
		{"DROP BUNDLE users", "schema"},
		{"SHOW BUNDLES", "query"},
		{"", "unknown"},
		{"UNKNOWN COMMAND", "unknown"},
	}

	for _, tt := range tests {
		result := inferCommandType(tt.command)
		if result != tt.expected {
			t.Errorf("inferCommandType(%q) = %q, expected %q", tt.command, result, tt.expected)
		}
	}
}

// CaptureHook captures the HookContext for inspection.
type CaptureHook struct {
	captured **HookContext
}

func (h *CaptureHook) Name() string { return "capture" }
func (h *CaptureHook) Before(ctx context.Context, hookCtx *HookContext) error {
	*h.captured = hookCtx
	return nil
}
func (h *CaptureHook) After(ctx context.Context, hookCtx *HookContext) error { return nil }

// TestHookContextFields verifies HookContext contains expected fields.
func TestHookContextFields(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	var capturedCtx *HookContext
	hook := &CaptureHook{captured: &capturedCtx}

	client.RegisterHook(hook)

	ctx := context.Background()
	hookCtx := &HookContext{
		Command:     "SELECT * FROM users",
		CommandType: "query",
		Params:      []interface{}{1, "test"},
		Metadata:    make(map[string]interface{}),
		TraceID:     "test-trace-id",
	}

	client.executeBeforeHooks(ctx, hookCtx)

	if capturedCtx.Command != "SELECT * FROM users" {
		t.Errorf("unexpected Command: %s", capturedCtx.Command)
	}

	if capturedCtx.CommandType != "query" {
		t.Errorf("unexpected CommandType: %s", capturedCtx.CommandType)
	}

	if capturedCtx.TraceID != "test-trace-id" {
		t.Errorf("unexpected TraceID: %s", capturedCtx.TraceID)
	}

	if len(capturedCtx.Params) != 2 {
		t.Errorf("unexpected Params length: %d", len(capturedCtx.Params))
	}
}

// ReplaceUsersHook replaces 'users' with 'customers'.
type ReplaceUsersHook struct{}

func (h *ReplaceUsersHook) Name() string { return "replace-users" }
func (h *ReplaceUsersHook) Before(ctx context.Context, hookCtx *HookContext) error {
	hookCtx.Command = strings.ReplaceAll(hookCtx.Command, "users", "customers")
	return nil
}
func (h *ReplaceUsersHook) After(ctx context.Context, hookCtx *HookContext) error { return nil }

// WrapCountHook wraps query in COUNT(*).
type WrapCountHook struct{}

func (h *WrapCountHook) Name() string { return "wrap-count" }
func (h *WrapCountHook) Before(ctx context.Context, hookCtx *HookContext) error {
	hookCtx.Command = strings.ReplaceAll(hookCtx.Command, "SELECT", "SELECT COUNT(*) FROM (SELECT")
	hookCtx.Command += ")"
	return nil
}
func (h *WrapCountHook) After(ctx context.Context, hookCtx *HookContext) error { return nil }

// TestMultipleHooksModifyCommand verifies multiple hooks can chain modifications.
func TestMultipleHooksModifyCommand(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(&opts)

	client.RegisterHook(&ReplaceUsersHook{})
	client.RegisterHook(&WrapCountHook{})

	ctx := context.Background()
	hookCtx := &HookContext{
		Command:  "SELECT * FROM users",
		Metadata: make(map[string]interface{}),
	}

	client.executeBeforeHooks(ctx, hookCtx)

	expected := "SELECT COUNT(*) FROM (SELECT * FROM customers)"
	if hookCtx.Command != expected {
		t.Errorf("expected %q, got %q", expected, hookCtx.Command)
	}
}
