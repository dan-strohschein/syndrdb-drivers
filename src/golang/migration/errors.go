package migration

import (
	"encoding/json"
	"fmt"
)

// MigrationError represents migration-specific errors.
type MigrationError struct {
	Code    string                 `json:"code"`
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
	Cause   error                  `json:"cause,omitempty"`
}

// Error implements the error interface.
func (e *MigrationError) Error() string {
	if e.Cause != nil {
		b, _ := json.Marshal(map[string]interface{}{
			"code":    e.Code,
			"type":    e.Type,
			"message": e.Message,
			"details": e.Details,
			"cause":   map[string]interface{}{"message": e.Cause.Error()},
		})
		return string(b)
	}

	b, _ := json.Marshal(map[string]interface{}{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
		"details": e.Details,
	})
	return string(b)
}

// Unwrap returns the underlying cause error.
func (e *MigrationError) Unwrap() error {
	return e.Cause
}

// ErrMigrationNotFound creates an error for when a migration doesn't exist.
func ErrMigrationNotFound(migrationID string) error {
	return &MigrationError{
		Code:    "MIGRATION_NOT_FOUND",
		Type:    "MIGRATION_ERROR",
		Message: fmt.Sprintf("migration '%s' not found", migrationID),
		Details: map[string]interface{}{
			"migrationId": migrationID,
		},
	}
}

// ErrMigrationFailed creates an error for when a migration execution fails.
func ErrMigrationFailed(migrationID string, cause error) error {
	return &MigrationError{
		Code:    "MIGRATION_FAILED",
		Type:    "MIGRATION_ERROR",
		Message: fmt.Sprintf("migration '%s' failed to execute", migrationID),
		Details: map[string]interface{}{
			"migrationId": migrationID,
		},
		Cause: cause,
	}
}

// ErrChecksumMismatch creates an error for when migration checksums don't match.
func ErrChecksumMismatch(migrationID, expected, actual string) error {
	return &MigrationError{
		Code:    "CHECKSUM_MISMATCH",
		Type:    "MIGRATION_ERROR",
		Message: fmt.Sprintf("migration '%s' has been modified (checksum mismatch)", migrationID),
		Details: map[string]interface{}{
			"migrationId": migrationID,
			"expected":    expected,
			"actual":      actual,
		},
	}
}

// ErrDependencyNotMet creates an error for when migration dependencies aren't satisfied.
func ErrDependencyNotMet(migrationID string, missingDeps []string) error {
	return &MigrationError{
		Code:    "DEPENDENCY_NOT_MET",
		Type:    "MIGRATION_ERROR",
		Message: fmt.Sprintf("migration '%s' has unmet dependencies", migrationID),
		Details: map[string]interface{}{
			"migrationId":         migrationID,
			"missingDependencies": missingDeps,
		},
	}
}

// ErrInvalidMigrationFile creates an error for malformed migration files.
func ErrInvalidMigrationFile(filename string, cause error) error {
	return &MigrationError{
		Code:    "INVALID_MIGRATION_FILE",
		Type:    "MIGRATION_ERROR",
		Message: fmt.Sprintf("migration file '%s' is invalid", filename),
		Details: map[string]interface{}{
			"filename": filename,
		},
		Cause: cause,
	}
}

// ErrRollbackNotSupported creates an error for migrations that cannot be rolled back.
func ErrRollbackNotSupported(migrationID string) error {
	return &MigrationError{
		Code:    "ROLLBACK_NOT_SUPPORTED",
		Type:    "MIGRATION_ERROR",
		Message: fmt.Sprintf("migration '%s' does not support rollback", migrationID),
		Details: map[string]interface{}{
			"migrationId": migrationID,
		},
	}
}

// ErrMigrationConflict creates an error for when validation detects conflicts.
func ErrMigrationConflict(conflicts []MigrationConflict) error {
	conflictDetails := make([]map[string]interface{}, len(conflicts))
	for i, c := range conflicts {
		conflictDetails[i] = map[string]interface{}{
			"type":        c.Type,
			"migrationId": c.MigrationID,
			"message":     c.Message,
			"expected":    c.Expected,
			"actual":      c.Actual,
		}
	}

	return &MigrationError{
		Code:    "MIGRATION_CONFLICT",
		Type:    "MIGRATION_ERROR",
		Message: fmt.Sprintf("found %d migration conflict(s)", len(conflicts)),
		Details: map[string]interface{}{
			"conflicts": conflictDetails,
			"count":     len(conflicts),
		},
	}
}
