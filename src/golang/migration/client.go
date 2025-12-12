package migration

import (
	"fmt"
	"strings"
	"time"
)

// Client provides migration operations for SyndrDB.
// It wraps a base client and adds migration-specific functionality.
type Client struct {
	history   *MigrationHistory
	validator *MigrationValidator
	executor  MigrationExecutor
	generator *RollbackGenerator
	lock      *MigrationLock
}

// MigrationExecutor defines the interface for executing migration commands.
// This allows the migration client to work with any client that can execute SQL.
type MigrationExecutor interface {
	// Execute runs a SQL command and returns the result.
	Execute(command string) (interface{}, error)
}

// NewClient creates a new migration client.
func NewClient(executor MigrationExecutor) *Client {
	history := NewMigrationHistory()
	return &Client{
		history:   history,
		validator: NewMigrationValidator(history),
		executor:  executor,
		generator: NewRollbackGenerator(),
	}
}

// LoadHistory loads migration history from the database.
// This should be called before performing migration operations.
func (c *Client) LoadHistory(historyJSON []byte) error {
	return c.history.LoadFromJSON(historyJSON)
}

// GetHistory returns the current migration history as JSON.
func (c *Client) GetHistory() ([]byte, error) {
	return c.history.ToJSON()
}

// Plan creates a migration plan for the given migrations.
func (c *Client) Plan(migrations []*Migration) (*MigrationPlan, error) {
	// Validate migrations first
	validation := c.validator.Validate(migrations)
	if !validation.Valid {
		return nil, ErrMigrationConflict(validation.Conflicts)
	}

	// Build plan with pending migrations in order
	pending := make([]*Migration, 0)
	for _, migration := range migrations {
		if !c.history.IsApplied(migration.ID) {
			pending = append(pending, migration)
		}
	}

	return &MigrationPlan{
		Migrations: pending,
		Direction:  Up,
		TotalCount: len(pending),
	}, nil
}

// Apply executes a migration plan.
// TODO: Future enhancement: support parallel execution of migrations with non-overlapping dependencies to improve performance for large migration sets
func (c *Client) Apply(plan *MigrationPlan) error {
	if plan.Direction != Up {
		return fmt.Errorf("only 'up' migrations are currently supported")
	}

	// Handle dry-run mode
	if plan.DryRun {
		// In dry-run mode, skip execution but preserve validation
		return nil
	}

	// Acquire lock if configured
	if c.lock != nil {
		if err := c.lock.AcquireLock(); err != nil {
			return err
		}
		defer func() {
			if err := c.lock.ReleaseLock(); err != nil {
				fmt.Printf("Warning: failed to release lock: %v\n", err)
			}
		}()
	}

	for _, migration := range plan.Migrations {
		if err := c.applyMigration(migration); err != nil {
			return err
		}
	}

	return nil
}

// applyMigration executes a single migration's "up" commands.
func (c *Client) applyMigration(migration *Migration) error {
	startTime := time.Now()
	checksum := CalculateChecksum(migration)

	// Execute each command in sequence
	for i, command := range migration.Up {
		if _, err := c.executor.Execute(command); err != nil {
			// Record failure
			executionTime := time.Since(startTime).Milliseconds()
			c.history.RecordMigration(migration.ID, Failed, executionTime, checksum, err)
			return ErrMigrationFailed(migration.ID, fmt.Errorf("command %d failed: %w", i+1, err))
		}
	}

	// Record success
	executionTime := time.Since(startTime).Milliseconds()
	c.history.RecordMigration(migration.ID, Applied, executionTime, checksum, nil)

	return nil
}

// Rollback rolls back a specific migration.
// If the migration doesn't have Down commands, attempts to generate them automatically.
func (c *Client) Rollback(migrationID string, allMigrations []*Migration) error {
	// Validate rollback is safe
	if err := c.validator.CanRollback(migrationID, allMigrations); err != nil {
		return err
	}

	// Find the migration
	var migration *Migration
	for _, m := range allMigrations {
		if m.ID == migrationID {
			migration = m
			break
		}
	}

	if migration == nil {
		return ErrMigrationNotFound(migrationID)
	}

	// Check if it has rollback commands, generate if missing
	if len(migration.Down) == 0 {
		// Attempt automatic generation
		count, err := c.GenerateDownCommands(migration)
		if err != nil {
			return fmt.Errorf("cannot rollback '%s': %w", migrationID, err)
		}
		if count == 0 {
			return ErrRollbackNotSupported(migrationID)
		}
	}

	// Execute rollback commands
	for i, command := range migration.Down {
		if _, err := c.executor.Execute(command); err != nil {
			return ErrMigrationFailed(migrationID, fmt.Errorf("rollback command %d failed: %w", i+1, err))
		}
	}

	// Record rollback
	return c.history.RecordRollback(migrationID)
}

// Validate performs validation on migrations without executing them.
func (c *Client) Validate(migrations []*Migration) *ValidationResult {
	return c.validator.Validate(migrations)
}

// GetAppliedMigrations returns a list of all applied migration IDs.
func (c *Client) GetAppliedMigrations() []string {
	return c.history.GetAppliedMigrations()
}

// GetMigrationRecord retrieves the record for a specific migration.
func (c *Client) GetMigrationRecord(migrationID string) (*MigrationRecord, bool) {
	return c.history.GetRecord(migrationID)
}

// ClearHistory clears all migration history (use with caution).
func (c *Client) ClearHistory() {
	c.history.Clear()
}

// GenerateDownCommands automatically generates Down commands for a migration.
// This should be called before applying migrations if Down commands are missing.
// Returns the number of Down commands generated.
func (c *Client) GenerateDownCommands(migration *Migration) (int, error) {
	if len(migration.Down) > 0 {
		// Already has down commands, skip generation
		return 0, nil
	}

	if len(migration.Up) == 0 {
		// No up commands to reverse
		return 0, nil
	}

	downCommands, err := c.generator.GenerateDown(migration.Up)
	if err != nil {
		return 0, fmt.Errorf("failed to generate down commands for migration '%s': %w", migration.ID, err)
	}

	migration.Down = downCommands
	return len(downCommands), nil
}

// GenerateAllDownCommands generates Down commands for all migrations that don't have them.
// Returns a map of migration ID to number of Down commands generated.
func (c *Client) GenerateAllDownCommands(migrations []*Migration) (map[string]int, error) {
	result := make(map[string]int)

	for _, migration := range migrations {
		count, err := c.GenerateDownCommands(migration)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			result[migration.ID] = count
		}
	}

	return result, nil
}

// CanAutoRollback checks if a migration can be automatically rolled back.
func (c *Client) CanAutoRollback(migration *Migration) bool {
	if len(migration.Down) > 0 {
		// Already has down commands
		return true
	}

	// Check if all up commands can be reversed
	for _, upCmd := range migration.Up {
		if !c.generator.CanGenerateDown(upCmd) {
			return false
		}
	}

	return len(migration.Up) > 0
}

// WithLocking configures the client to use file-based locking.
// Timeout defaults to 1 hour if zero. Checks SYNDR_LOCK_TIMEOUT env var.
func (c *Client) WithLocking(dir string, timeout time.Duration) error {
	lock, err := NewMigrationLock(dir, timeout)
	if err != nil {
		return err
	}
	c.lock = lock
	return nil
}

// WithLockRetry configures retry behavior for lock acquisition.
// Useful for CI/CD environments with brief contention.
func (c *Client) WithLockRetry(maxRetries int, backoff time.Duration) error {
	if c.lock == nil {
		return fmt.Errorf("locking not configured, call WithLocking first")
	}
	return c.lock.SetRetry(maxRetries, backoff)
}

// Preview creates a migration plan in dry-run mode for preview.
func (c *Client) Preview(migrations []*Migration) (*MigrationPlan, error) {
	plan, err := c.Plan(migrations)
	if err != nil {
		return nil, err
	}
	plan.DryRun = true
	return plan, nil
}

// FormatPreview formats a migration plan for human-readable output.
func FormatPreview(plan *MigrationPlan) string {
	var sb strings.Builder
	
	sb.WriteString("=== Migration Preview ===\n\n")
	
	if len(plan.Migrations) == 0 {
		sb.WriteString("No migrations to apply.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Total migrations: %d\n\n", plan.TotalCount))

	for i, migration := range plan.Migrations {
		sb.WriteString(fmt.Sprintf("Migration %d: %s\n", i+1, migration.ID))
		sb.WriteString(fmt.Sprintf("  Name: %s\n", migration.Name))
		sb.WriteString(fmt.Sprintf("  Timestamp: %s\n", migration.Timestamp.Format(time.RFC3339)))
		
		if len(migration.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("  Dependencies: %v\n", migration.Dependencies))
		}

		sb.WriteString("\n  Up Commands:\n")
		for j, cmd := range migration.Up {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", j+1, cmd))
		}

		if len(migration.Down) > 0 {
			sb.WriteString("\n  Down Commands:\n")
			for j, cmd := range migration.Down {
				sb.WriteString(fmt.Sprintf("    %d. %s\n", j+1, cmd))
			}
		} else {
			sb.WriteString("\n  Down Commands: (will be auto-generated if needed)\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// GenerateFile generates migration files in the specified directory.
func (c *Client) GenerateFile(migrations []*Migration, dir string) ([]string, error) {
	if len(migrations) == 0 {
		return nil, fmt.Errorf("no migrations to generate")
	}

	// Filter to pending migrations only
	plan, err := c.Plan(migrations)
	if err != nil {
		return nil, fmt.Errorf("failed to plan migrations: %w", err)
	}

	var filePaths []string
	for _, migration := range plan.Migrations {
		path, err := WriteMigrationFile(migration, dir)
		if err != nil {
			return filePaths, fmt.Errorf("failed to write migration %s: %w", migration.ID, err)
		}
		filePaths = append(filePaths, path)
	}

	return filePaths, nil
}

// LoadFromFile loads a migration from a file.
func (c *Client) LoadFromFile(path string) (*Migration, error) {
	return ReadMigrationFile(path)
}

// ApplyFromDirectory scans a directory and applies pending migrations.
func (c *Client) ApplyFromDirectory(dir string) error {
	// List all migration files
	migrations, err := ListMigrationFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to list migration files: %w", err)
	}

	if len(migrations) == 0 {
		return nil // No migrations to apply
	}

	// Create plan with pending migrations
	plan, err := c.Plan(migrations)
	if err != nil {
		return fmt.Errorf("failed to plan migrations: %w", err)
	}

	// Apply the plan
	return c.Apply(plan)
}
