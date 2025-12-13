//go:build milestone2
// +build milestone2

package client

import (
	"context"
	"time"
)

// HookContext contains information about the command being executed.
// This is passed to hooks to allow inspection and modification.
type HookContext struct {
	// Command is the raw command string being executed
	Command string

	// CommandType categorizes the command (query, mutation, transaction, etc.)
	CommandType string

	// Params are any parameters associated with the command
	Params []interface{}

	// StartTime is when the command execution began
	StartTime time.Time

	// Metadata allows hooks to store arbitrary data for passing between Before/After
	Metadata map[string]interface{}

	// TraceID is the unique identifier for this command execution
	TraceID string

	// Result stores the command result (set after execution, available in After hook)
	Result interface{}

	// Error stores any error that occurred (available in After hook)
	Error error

	// Duration is the execution time (available in After hook)
	Duration time.Duration
}

// Hook is the interface that all hooks must implement.
// Hooks can inspect, modify, or abort command execution.
type Hook interface {
	// Name returns the unique name of this hook
	Name() string

	// Before is called before command execution.
	// Returning an error aborts the command and returns the error.
	// Hooks can modify the HookContext (e.g., change Command, add Metadata).
	Before(ctx context.Context, hookCtx *HookContext) error

	// After is called after command execution (even if it failed).
	// Returning an error replaces any existing error.
	// Hooks can inspect Result/Error and modify the HookContext.
	After(ctx context.Context, hookCtx *HookContext) error
}

// hookEntry wraps a Hook with its registration order for stable iteration.
type hookEntry struct {
	hook  Hook
	order int
}

// RegisterHook adds a hook to the client's hook chain.
// Hooks are executed in FIFO order (first registered, first executed).
// If a hook with the same name already exists, it is replaced.
func (c *Client) RegisterHook(hook Hook) {
	c.hooksMu.Lock()
	defer c.hooksMu.Unlock()

	// Check if hook already exists
	for i, entry := range c.hooks {
		if entry.hook.Name() == hook.Name() {
			// Replace existing hook, preserve order
			c.hooks[i].hook = hook
			c.logger.Info("hook replaced", String("hook", hook.Name()))
			return
		}
	}

	// Add new hook
	order := len(c.hooks)
	c.hooks = append(c.hooks, hookEntry{hook: hook, order: order})
	c.logger.Info("hook registered", String("hook", hook.Name()), Int("order", order))
}

// UnregisterHook removes a hook by name.
// Returns true if the hook was found and removed, false otherwise.
func (c *Client) UnregisterHook(name string) bool {
	c.hooksMu.Lock()
	defer c.hooksMu.Unlock()

	for i, entry := range c.hooks {
		if entry.hook.Name() == name {
			// Remove hook while preserving order of others
			c.hooks = append(c.hooks[:i], c.hooks[i+1:]...)
			c.logger.Info("hook unregistered", String("hook", name))
			return true
		}
	}

	return false
}

// GetHooks returns the names of all registered hooks in execution order.
func (c *Client) GetHooks() []string {
	c.hooksMu.RLock()
	defer c.hooksMu.RUnlock()

	names := make([]string, len(c.hooks))
	for i, entry := range c.hooks {
		names[i] = entry.hook.Name()
	}
	return names
}

// executeBeforeHooks runs all Before hooks in order.
// If any hook returns an error, execution stops and the error is returned.
func (c *Client) executeBeforeHooks(ctx context.Context, hookCtx *HookContext) error {
	c.hooksMu.RLock()
	hooks := make([]Hook, len(c.hooks))
	for i, entry := range c.hooks {
		hooks[i] = entry.hook
	}
	c.hooksMu.RUnlock()

	for _, hook := range hooks {
		if err := hook.Before(ctx, hookCtx); err != nil {
			c.logger.Debug("hook aborted command",
				String("hook", hook.Name()),
				String("command", hookCtx.Command),
				Error("error", err))
			return err
		}
	}

	return nil
}

// executeAfterHooks runs all After hooks in order.
// All hooks are executed even if one returns an error.
// The last error returned (if any) is returned.
func (c *Client) executeAfterHooks(ctx context.Context, hookCtx *HookContext) error {
	c.hooksMu.RLock()
	hooks := make([]Hook, len(c.hooks))
	for i, entry := range c.hooks {
		hooks[i] = entry.hook
	}
	c.hooksMu.RUnlock()

	var lastErr error
	for _, hook := range hooks {
		if err := hook.After(ctx, hookCtx); err != nil {
			c.logger.Debug("hook returned error in After",
				String("hook", hook.Name()),
				String("command", hookCtx.Command),
				Error("error", err))
			lastErr = err
		}
	}

	return lastErr
}

// inferCommandType attempts to determine the command type from the command string.
func inferCommandType(command string) string {
	// Simple heuristic based on command prefix
	if len(command) == 0 {
		return "unknown"
	}

	// Convert to uppercase for comparison
	cmd := command
	if len(cmd) > 10 {
		cmd = cmd[:10]
	}

	// Check common patterns
	switch {
	case len(command) >= 6 && (command[:6] == "SELECT" || command[:6] == "select"):
		return "query"
	case len(command) >= 6 && (command[:6] == "INSERT" || command[:6] == "insert"):
		return "mutation"
	case len(command) >= 6 && (command[:6] == "UPDATE" || command[:6] == "update"):
		return "mutation"
	case len(command) >= 6 && (command[:6] == "DELETE" || command[:6] == "delete"):
		return "mutation"
	case len(command) >= 5 && (command[:5] == "BEGIN" || command[:5] == "begin"):
		return "transaction"
	case len(command) >= 6 && (command[:6] == "COMMIT" || command[:6] == "commit"):
		return "transaction"
	case len(command) >= 8 && (command[:8] == "ROLLBACK" || command[:8] == "rollback"):
		return "transaction"
	case len(command) >= 6 && (command[:6] == "CREATE" || command[:6] == "create"):
		return "schema"
	case len(command) >= 4 && (command[:4] == "DROP" || command[:4] == "drop"):
		return "schema"
	case len(command) >= 4 && (command[:4] == "SHOW" || command[:4] == "show"):
		return "query"
	default:
		return "unknown"
	}
}
