package migration

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestLockAcquireAndRelease tests basic lock operations
func TestLockAcquireAndRelease(t *testing.T) {
	tmpDir := t.TempDir()

	lock, err := NewMigrationLock(tmpDir, time.Hour)
	if err != nil {
		t.Fatalf("NewMigrationLock failed: %v", err)
	}

	// Acquire lock
	err = lock.AcquireLock()
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}

	// Verify lock file exists
	lockFile := filepath.Join(tmpDir, ".syndr_migration.lock")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Error("Lock file not created")
	}

	// Release lock
	err = lock.ReleaseLock()
	if err != nil {
		t.Fatalf("ReleaseLock failed: %v", err)
	}

	// Verify lock file removed
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		t.Error("Lock file should be removed after release")
	}
}

// TestLockConcurrency tests that only one goroutine can acquire lock
func TestLockConcurrency(t *testing.T) {
	tmpDir := t.TempDir()

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Try to acquire from 5 goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lock, _ := NewMigrationLock(tmpDir, time.Second)
			err := lock.AcquireLock()

			mu.Lock()
			if err == nil {
				successCount++
				time.Sleep(50 * time.Millisecond)
				lock.ReleaseLock()
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Only one should succeed
	if successCount != 1 {
		t.Errorf("Expected 1 successful acquisition, got %d", successCount)
	}
}

// TestStaleLockCleanup tests that stale locks are cleaned up
func TestStaleLockCleanup(t *testing.T) {
	t.Skip("Skipping test: requires simulating dead process which is complex in tests")
	// TODO: This test requires mocking or using a different PID
	// to properly test stale lock detection. For now we skip it.
}

// TestLockRetry tests retry logic with exponential backoff
func TestLockRetry(t *testing.T) {
	tmpDir := t.TempDir()

	// Acquire lock first
	lock1, _ := NewMigrationLock(tmpDir, time.Hour)
	err := lock1.AcquireLock()
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}

	// Try to acquire with retry
	lock2, _ := NewMigrationLock(tmpDir, time.Hour)
	lock2.SetRetry(3, 50*time.Millisecond)

	start := time.Now()
	err = lock2.AcquireLock()
	duration := time.Since(start)

	// Should fail after retries
	if err == nil {
		t.Error("Expected error after retry attempts")
	}

	// Should have taken at least 200ms (3 retries with backoff)
	if duration < 200*time.Millisecond {
		t.Errorf("Expected duration >= 200ms, got %v", duration)
	}

	lock1.ReleaseLock()
}

// TestForceUnlock tests forced lock removal
func TestForceUnlock(t *testing.T) {
	tmpDir := t.TempDir()

	lock, _ := NewMigrationLock(tmpDir, time.Hour)
	err := lock.AcquireLock()
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}

	// Force unlock
	err = lock.ForceUnlock()
	if err != nil {
		t.Fatalf("ForceUnlock failed: %v", err)
	}

	// Verify lock is removed
	lockFile := filepath.Join(tmpDir, ".syndr_migration.lock")
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		t.Error("Lock file should be removed after force unlock")
	}
}

// TestLockFilePermissions tests that lock file has 0600 permissions
func TestLockFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	lock, _ := NewMigrationLock(tmpDir, time.Hour)
	err := lock.AcquireLock()
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}

	lockFile := filepath.Join(tmpDir, ".syndr_migration.lock")
	info, err := os.Stat(lockFile)
	if err != nil {
		t.Fatalf("Failed to stat lock file: %v", err)
	}

	expectedMode := os.FileMode(0600)
	if info.Mode().Perm() != expectedMode {
		t.Errorf("Expected permissions %s, got %s", expectedMode, info.Mode().Perm())
	}

	lock.ReleaseLock()
}

// TestSetRetryValidation tests retry parameter validation
func TestSetRetryValidation(t *testing.T) {
	tmpDir := t.TempDir()
	lock, _ := NewMigrationLock(tmpDir, time.Hour)

	// Test max retries > 10
	err := lock.SetRetry(15, time.Second)
	if err == nil {
		t.Error("Expected error for maxRetries > 10")
	}

	// Test backoff > 1 minute
	err = lock.SetRetry(3, 2*time.Minute)
	if err == nil {
		t.Error("Expected error for backoff > 1 minute")
	}

	// Test valid values
	err = lock.SetRetry(5, 30*time.Second)
	if err != nil {
		t.Errorf("Expected no error for valid parameters, got: %v", err)
	}
}

// TestParseLockTimeout tests environment variable parsing
func TestParseLockTimeout(t *testing.T) {
	tests := []struct {
		envValue string
		wantErr  bool
	}{
		{"", false},                 // Default
		{"5m", false},               // Valid
		{"1h", false},               // Valid
		{"invalid", true},           // Invalid - should return error
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("SYNDR_LOCK_TIMEOUT", tt.envValue)
				defer os.Unsetenv("SYNDR_LOCK_TIMEOUT")
			}

			timeout, err := parseLockTimeout()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLockTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && timeout <= 0 {
				t.Error("Expected positive timeout")
			}
		})
	}
}
