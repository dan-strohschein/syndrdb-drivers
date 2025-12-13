//go:build milestone2

package testutil

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
)

var testDBCounter uint64

// NewTestClient creates a client configured for testing.
// It reads the connection string from SYNDRDB_TEST_CONN environment variable.
// If not set, it returns an error.
//
// Example:
//
//	export SYNDRDB_TEST_CONN="syndrdb://localhost:1776/testdb"
//	client, cleanup := testutil.NewTestClient(t)
//	defer cleanup()
func NewTestClient(t *testing.T) (*client.Client, func()) {
	t.Helper()

	connStr := os.Getenv("SYNDRDB_TEST_CONN")
	if connStr == "" {
		t.Skip("SYNDRDB_TEST_CONN not set, skipping integration test")
		return nil, func() {}
	}

	opts := &client.ClientOptions{
		DebugMode: testing.Verbose(),
	}

	c := client.NewClient(opts)
	ctx := context.Background()

	if err := c.Connect(ctx, connStr); err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	cleanup := func() {
		if err := c.Disconnect(ctx); err != nil {
			t.Logf("warning: failed to disconnect: %v", err)
		}
	}

	return c, cleanup
}

// NewTestClientOrSkip creates a test client or skips the test if unavailable.
// This is useful for optional integration tests.
func NewTestClientOrSkip(t *testing.T) (*client.Client, func()) {
	t.Helper()

	connStr := os.Getenv("SYNDRDB_TEST_CONN")
	if connStr == "" {
		t.Skip("SYNDRDB_TEST_CONN not set, skipping integration test")
		return nil, func() {}
	}

	return NewTestClient(t)
}

// TestDBName generates a unique database name for testing.
// Format: test_db_<timestamp>_<counter>
func TestDBName(prefix string) string {
	if prefix == "" {
		prefix = "test"
	}
	n := atomic.AddUint64(&testDBCounter, 1)
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s_db_%d_%d", prefix, timestamp, n)
}

// TestBundleName generates a unique bundle name for testing.
func TestBundleName(prefix string) string {
	if prefix == "" {
		prefix = "test"
	}
	n := atomic.AddUint64(&testDBCounter, 1)
	return fmt.Sprintf("%s_bundle_%d", prefix, n)
}

// WithTimeout creates a context with timeout for tests.
// Default timeout is 10 seconds.
func WithTimeout(t *testing.T, timeout ...time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()

	duration := 10 * time.Second
	if len(timeout) > 0 {
		duration = timeout[0]
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	t.Cleanup(cancel)

	return ctx, cancel
}

// RequireNoError fails the test if err is not nil.
// This is similar to testify's require.NoError.
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("Unexpected error: %v - %v", err, msgAndArgs)
		} else {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
}

// RequireError fails the test if err is nil.
func RequireError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("Expected error but got nil - %v", msgAndArgs)
		} else {
			t.Fatal("Expected error but got nil")
		}
	}
}

// AssertEqual checks if two values are equal.
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if expected != actual {
		if len(msgAndArgs) > 0 {
			t.Errorf("Not equal: expected=%v, actual=%v - %v", expected, actual, msgAndArgs)
		} else {
			t.Errorf("Not equal: expected=%v, actual=%v", expected, actual)
		}
	}
}

// AssertNotEqual checks if two values are not equal.
func AssertNotEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if expected == actual {
		if len(msgAndArgs) > 0 {
			t.Errorf("Should not be equal: value=%v - %v", actual, msgAndArgs)
		} else {
			t.Errorf("Should not be equal: value=%v", actual)
		}
	}
}

// AssertContains checks if a string contains a substring.
func AssertContains(t *testing.T, str, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if !containsStr(str, substr) {
		if len(msgAndArgs) > 0 {
			t.Errorf("String does not contain substring: str=%q, substr=%q - %v", str, substr, msgAndArgs)
		} else {
			t.Errorf("String does not contain substring: str=%q, substr=%q", str, substr)
		}
	}
}

// SetupTestBundle creates a test bundle and returns a cleanup function.
// The cleanup function will drop the bundle after the test.
//
// Example:
//
//	cleanup := testutil.SetupTestBundle(t, client, "users", []string{
//	    "id int REQUIRED UNIQUE",
//	    "name string REQUIRED",
//	})
//	defer cleanup()
func SetupTestBundle(t *testing.T, c *client.Client, bundleName string, fields []string) func() {
	t.Helper()

	// Create bundle
	fieldsStr := ""
	for i, field := range fields {
		if i > 0 {
			fieldsStr += ", "
		}
		fieldsStr += field
	}

	createCmd := fmt.Sprintf("CREATE BUNDLE %s (%s);", bundleName, fieldsStr)
	_, err := c.Mutate(createCmd, 0)
	RequireNoError(t, err, "failed to create test bundle")

	// Return cleanup function
	return func() {
		dropCmd := fmt.Sprintf("DROP BUNDLE %s;", bundleName)
		_, err := c.Mutate(dropCmd, 0)
		if err != nil {
			t.Logf("warning: failed to drop test bundle %s: %v", bundleName, err)
		}
	}
}

// InsertTestData inserts test data into a bundle and returns the data.
// This is useful for setting up test fixtures.
//
// Example:
//
//	data := testutil.InsertTestData(t, client, "users", []map[string]interface{}{
//	    {"id": 1, "name": "Alice"},
//	    {"id": 2, "name": "Bob"},
//	})
func InsertTestData(t *testing.T, c *client.Client, bundleName string, records []map[string]interface{}) []map[string]interface{} {
	t.Helper()

	for _, record := range records {
		// Build INSERT command
		fields := make([]string, 0, len(record))
		values := make([]string, 0, len(record))

		for k, v := range record {
			fields = append(fields, k)
			values = append(values, formatValue(v))
		}

		insertCmd := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
			bundleName,
			joinStrings(fields, ", "),
			joinStrings(values, ", "))

		_, err := c.Mutate(insertCmd, 0)
		RequireNoError(t, err, "failed to insert test data")
	}

	return records
}

// CleanupTestData removes all data from a bundle.
func CleanupTestData(t *testing.T, c *client.Client, bundleName string) {
	t.Helper()

	deleteCmd := fmt.Sprintf("DELETE FROM %s;", bundleName)
	_, err := c.Mutate(deleteCmd, 0)
	if err != nil {
		t.Logf("warning: failed to cleanup test data from %s: %v", bundleName, err)
	}
}

// WaitFor polls a condition until it returns true or times out.
// This is useful for testing eventual consistency.
//
// Example:
//
//	testutil.WaitFor(t, 5*time.Second, 100*time.Millisecond, func() bool {
//	    result, _ := client.Query(ctx, "SELECT * FROM users WHERE id = 1", 0)
//	    return result != nil
//	})
func WaitFor(t *testing.T, timeout, interval time.Duration, condition func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}

	t.Errorf("condition not met within timeout %v", timeout)
	return false
}

// Eventually is an alias for WaitFor (Jest-style naming).
func Eventually(t *testing.T, timeout, interval time.Duration, condition func() bool) bool {
	return WaitFor(t, timeout, interval, condition)
}

// Parallel marks the test to run in parallel and returns the test instance.
// This is a convenience wrapper that returns t for chaining.
func Parallel(t *testing.T) *testing.T {
	t.Parallel()
	return t
}

// SkipIf skips the test if the condition is true.
func SkipIf(t *testing.T, condition bool, reason string) {
	t.Helper()
	if condition {
		t.Skip(reason)
	}
}

// SkipUnless skips the test unless the condition is true.
func SkipUnless(t *testing.T, condition bool, reason string) {
	t.Helper()
	if !condition {
		t.Skip(reason)
	}
}

// Helper functions

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("'%s'", val)
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%f", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case time.Time:
		return fmt.Sprintf("'%s'", val.Format(time.RFC3339))
	default:
		return fmt.Sprintf("'%v'", val)
	}
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// BenchmarkHelper provides utilities for benchmark tests.
type BenchmarkHelper struct {
	b *testing.B
	c *client.Client
}

// NewBenchmarkHelper creates a new benchmark helper.
func NewBenchmarkHelper(b *testing.B) *BenchmarkHelper {
	b.Helper()

	connStr := os.Getenv("SYNDRDB_TEST_CONN")
	if connStr == "" {
		b.Skip("SYNDRDB_TEST_CONN not set, skipping benchmark")
		return nil
	}

	opts := &client.ClientOptions{}
	c := client.NewClient(opts)
	ctx := context.Background()

	if err := c.Connect(ctx, connStr); err != nil {
		b.Fatalf("failed to connect: %v", err)
	}

	b.Cleanup(func() {
		c.Disconnect(ctx)
	})

	return &BenchmarkHelper{
		b: b,
		c: c,
	}
}

// Client returns the test client.
func (h *BenchmarkHelper) Client() *client.Client {
	return h.c
}

// ResetTimer resets the benchmark timer.
func (h *BenchmarkHelper) ResetTimer() {
	h.b.ResetTimer()
}

// RunParallel runs the benchmark in parallel.
func (h *BenchmarkHelper) RunParallel(body func(*testing.PB)) {
	h.b.RunParallel(body)
}
