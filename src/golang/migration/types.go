package migration

import "time"

// MigrationDirection represents the direction of a migration.
type MigrationDirection string

const (
	// Up applies a migration forward.
	Up MigrationDirection = "up"
	// Down rolls back a migration.
	Down MigrationDirection = "down"
)

// MigrationStatus represents the current state of a migration.
type MigrationStatus string

const (
	// Pending means migration has not been applied.
	Pending MigrationStatus = "pending"
	// Applied means migration has been successfully applied.
	Applied MigrationStatus = "applied"
	// Failed means migration failed during application.
	Failed MigrationStatus = "failed"
	// RolledBack means migration was successfully rolled back.
	RolledBack MigrationStatus = "rolled_back"
)

// Migration represents a single database migration.
type Migration struct {
	// ID is the unique identifier for this migration (e.g., "001_initial_schema").
	ID string `json:"id"`

	// Name is a human-readable description.
	Name string `json:"name"`

	// Up contains the SQL commands to apply this migration.
	Up []string `json:"up"`

	// Down contains the SQL commands to rollback this migration.
	Down []string `json:"down"`

	// Dependencies lists migration IDs that must be applied before this one.
	Dependencies []string `json:"dependencies,omitempty"`

	// Timestamp when this migration was created.
	Timestamp time.Time `json:"timestamp"`
}

// MigrationRecord represents a historical record of a migration execution.
type MigrationRecord struct {
	// MigrationID is the ID of the migration that was executed.
	MigrationID string `json:"migrationId"`

	// AppliedAt is when the migration was applied.
	AppliedAt time.Time `json:"appliedAt"`

	// RolledBackAt is when the migration was rolled back (nil if not rolled back).
	RolledBackAt *time.Time `json:"rolledBackAt,omitempty"`

	// Status is the current status of this migration.
	Status MigrationStatus `json:"status"`

	// ExecutionTimeMs is how long the migration took to execute.
	ExecutionTimeMs int64 `json:"executionTimeMs"`

	// Error contains error details if the migration failed.
	Error string `json:"error,omitempty"`

	// Checksum is a hash of the migration content for validation.
	Checksum string `json:"checksum"`
}

// MigrationPlan represents a planned sequence of migrations.
type MigrationPlan struct {
	// Migrations is the ordered list of migrations to execute.
	Migrations []*Migration `json:"migrations"`

	// Direction is whether we're migrating up or down.
	Direction MigrationDirection `json:"direction"`

	// TotalCount is the number of migrations in this plan.
	TotalCount int `json:"totalCount"`

	// DryRun indicates this is a preview without execution.
	DryRun bool `json:"dryRun,omitempty"`
}

// ConflictType represents the type of migration conflict.
type ConflictType string

const (
	// ChecksumMismatch indicates the migration content has changed.
	ChecksumMismatch ConflictType = "checksum_mismatch"
	// DependencyConflict indicates a dependency is missing or not applied.
	DependencyConflict ConflictType = "dependency_conflict"
	// OrderConflict indicates migrations are out of order.
	OrderConflict ConflictType = "order_conflict"
)

// MigrationConflict represents a detected issue with migrations.
type MigrationConflict struct {
	// Type is the kind of conflict.
	Type ConflictType `json:"type"`

	// MigrationID is the affected migration.
	MigrationID string `json:"migrationId"`

	// Message describes the conflict.
	Message string `json:"message"`

	// Expected is the expected value (e.g., expected checksum).
	Expected string `json:"expected,omitempty"`

	// Actual is the actual value (e.g., actual checksum).
	Actual string `json:"actual,omitempty"`
}

// ValidationResult contains the results of migration validation.
type ValidationResult struct {
	// Valid is true if all validations passed.
	Valid bool `json:"valid"`

	// Conflicts lists any detected issues.
	Conflicts []MigrationConflict `json:"conflicts"`

	// PendingMigrations lists migrations not yet applied.
	PendingMigrations []string `json:"pendingMigrations"`

	// AppliedMigrations lists migrations already applied.
	AppliedMigrations []string `json:"appliedMigrations"`
}
