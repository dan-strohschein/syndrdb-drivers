package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TODO: File-based locks work for shared filesystem (NFS, EFS) and serverless functions
// on same host/container. For true distributed coordination across shared-nothing architectures,
// future enhancement should implement database-backed locks using PostgreSQL pg_advisory_lock
// or MySQL GET_LOCK functions. Migration path: add LockProvider interface with FileLockProvider
// and DBLockProvider implementations allowing runtime selection based on deployment environment.

// LockMetadata contains information about who holds the migration lock.
type LockMetadata struct {
	Holder    string    `json:"holder"`    // Username from environment
	Hostname  string    `json:"hostname"`  // Hostname for distributed detection
	PID       int       `json:"pid"`       // Process ID
	Timestamp time.Time `json:"timestamp"` // When lock was acquired
	Note      string    `json:"note,omitempty"` // Optional context (CI job ID, etc.)
}

// MigrationLock provides file-based locking for migration operations.
type MigrationLock struct {
	lockPath     string
	staleTimeout time.Duration
	maxRetries   int
	retryBackoff time.Duration
	metadata     *LockMetadata
}

// NewMigrationLock creates a new migration lock instance.
// Timeout defaults to 1 hour if zero. Checks SYNDR_LOCK_TIMEOUT env var.
func NewMigrationLock(dir string, timeout time.Duration) (*MigrationLock, error) {
	if dir == "" {
		return nil, fmt.Errorf("directory path cannot be empty")
	}

	// Parse timeout from environment if not provided
	if timeout == 0 {
		var err error
		timeout, err = parseLockTimeout()
		if err != nil {
			return nil, fmt.Errorf("failed to parse lock timeout: %w", err)
		}
	}

	lockPath := filepath.Join(dir, ".syndr_migration.lock")
	
	return &MigrationLock{
		lockPath:     lockPath,
		staleTimeout: timeout,
		maxRetries:   0, // Default: no retries, fail immediately
		retryBackoff: 0,
	}, nil
}

// SetRetry configures retry behavior for lock acquisition.
// Useful for CI/CD environments with brief contention.
func (l *MigrationLock) SetRetry(maxRetries int, backoff time.Duration) error {
	if maxRetries < 0 {
		return fmt.Errorf("maxRetries cannot be negative")
	}
	if maxRetries > 10 {
		return fmt.Errorf("maxRetries cannot exceed 10")
	}
	if backoff < 0 {
		return fmt.Errorf("backoff cannot be negative")
	}
	if backoff > time.Minute {
		return fmt.Errorf("backoff cannot exceed 1 minute")
	}

	l.maxRetries = maxRetries
	l.retryBackoff = backoff
	return nil
}

// AcquireLock attempts to acquire the migration lock.
// Automatically cleans up stale locks and retries if configured.
func (l *MigrationLock) AcquireLock() error {
	return l.acquireLockWithRetry(0)
}

// acquireLockWithRetry implements the retry logic for lock acquisition.
func (l *MigrationLock) acquireLockWithRetry(attempt int) error {
	// Prepare metadata
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME") // Windows fallback
		if user == "" {
			user = "unknown"
		}
	}

	l.metadata = &LockMetadata{
		Holder:    user,
		Hostname:  hostname,
		PID:       os.Getpid(),
		Timestamp: time.Now(),
	}

	// Try to create lock file exclusively
	file, err := os.OpenFile(l.lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create lock file: %w", err)
		}

		// Lock file exists, check if stale
		if l.isLockStale() {
			if err := l.cleanupStaleLock(); err != nil {
				return fmt.Errorf("failed to cleanup stale lock: %w", err)
			}
			// Retry immediately after cleanup
			return l.acquireLockWithRetry(attempt)
		}

		// Lock is held by active process
		metadata, _ := l.readLockMetadata()
		
		// Check if we should retry
		if attempt < l.maxRetries {
			// Calculate backoff with exponential increase
			backoff := l.retryBackoff * time.Duration(1<<uint(attempt))
			if backoff > time.Minute {
				backoff = time.Minute
			}
			
			fmt.Fprintf(os.Stderr, "Lock held by %s@%s (PID %d), retrying in %s (attempt %d/%d)\n",
				metadata.Holder, metadata.Hostname, metadata.PID, backoff, attempt+1, l.maxRetries)
			
			time.Sleep(backoff)
			return l.acquireLockWithRetry(attempt + 1)
		}

		// All retries exhausted
		return l.createLockConflictError(metadata)
	}
	defer file.Close()

	// Write metadata to lock file
	data, err := json.MarshalIndent(l.metadata, "", "  ")
	if err != nil {
		// Clean up lock file if we can't write metadata
		os.Remove(l.lockPath)
		return fmt.Errorf("failed to marshal lock metadata: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		os.Remove(l.lockPath)
		return fmt.Errorf("failed to write lock metadata: %w", err)
	}

	return nil
}

// ReleaseLock removes the lock file.
func (l *MigrationLock) ReleaseLock() error {
	if err := os.Remove(l.lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

// ForceUnlock forcibly removes the lock file after safety checks.
// Checks hostname to prevent accidental cross-machine unlocks.
func (l *MigrationLock) ForceUnlock() error {
	metadata, err := l.readLockMetadata()
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already unlocked
		}
		// If we can't read metadata, allow force unlock
		fmt.Fprintf(os.Stderr, "Warning: forcing unlock without metadata validation\n")
		return l.ReleaseLock()
	}

	// Check hostname to prevent cross-machine unlocks
	currentHostname, _ := os.Hostname()
	if currentHostname != "" && metadata.Hostname != "" && currentHostname != metadata.Hostname {
		return fmt.Errorf("cannot force unlock: lock held on different host (%s), current host is %s",
			metadata.Hostname, currentHostname)
	}

	// Check if process is still active (best effort)
	if isProcessActive(metadata.PID) {
		return fmt.Errorf("cannot force unlock: process %d appears to be active on this host", metadata.PID)
	}

	fmt.Fprintf(os.Stderr, "Force unlocking migration lock held by %s@%s (PID %d)\n",
		metadata.Holder, metadata.Hostname, metadata.PID)
	
	return l.ReleaseLock()
}

// isLockStale checks if the lock file is older than the stale timeout.
func (l *MigrationLock) isLockStale() bool {
	info, err := os.Stat(l.lockPath)
	if err != nil {
		return false
	}

	age := time.Since(info.ModTime())
	return age > l.staleTimeout
}

// cleanupStaleLock removes a stale lock file with logging.
func (l *MigrationLock) cleanupStaleLock() error {
	metadata, _ := l.readLockMetadata()
	
	fmt.Fprintf(os.Stderr, "Warning: cleaning up stale lock (held for >%s by %s@%s)\n",
		l.staleTimeout, metadata.Holder, metadata.Hostname)
	
	return l.ReleaseLock()
}

// readLockMetadata reads metadata from an existing lock file.
func (l *MigrationLock) readLockMetadata() (*LockMetadata, error) {
	data, err := os.ReadFile(l.lockPath)
	if err != nil {
		return nil, err
	}

	var metadata LockMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lock metadata: %w", err)
	}

	return &metadata, nil
}

// createLockConflictError creates a detailed error for lock conflicts.
func (l *MigrationLock) createLockConflictError(metadata *LockMetadata) error {
	age := time.Since(metadata.Timestamp)
	
	return fmt.Errorf("migration lock is held by %s@%s (PID %d) since %s ago. "+
		"Wait for the migration to complete or use force unlock if the process is stuck",
		metadata.Holder, metadata.Hostname, metadata.PID, age.Round(time.Second))
}

// parseLockTimeout parses lock timeout from SYNDR_LOCK_TIMEOUT env var.
// Returns 1 hour default if not set or invalid.
func parseLockTimeout() (time.Duration, error) {
	envTimeout := os.Getenv("SYNDR_LOCK_TIMEOUT")
	if envTimeout == "" {
		return time.Hour, nil // Default: 1 hour
	}

	timeout, err := time.ParseDuration(envTimeout)
	if err != nil {
		return 0, fmt.Errorf("invalid SYNDR_LOCK_TIMEOUT value '%s': %w", envTimeout, err)
	}

	if timeout <= 0 {
		return 0, fmt.Errorf("SYNDR_LOCK_TIMEOUT must be positive, got %s", timeout)
	}

	return timeout, nil
}

// isProcessActive checks if a process with the given PID is active.
// This is a best-effort check and may not be accurate across all platforms.
func isProcessActive(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Try to find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds. Try to signal it.
	// Signal 0 checks if we can send a signal without actually sending one.
	err = process.Signal(os.Signal(nil))
	return err == nil
}
