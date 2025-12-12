package migration

import (
	"fmt"
)

// MigrationValidator validates migrations against history and dependencies.
type MigrationValidator struct {
	history *MigrationHistory
}

// NewMigrationValidator creates a new migration validator.
func NewMigrationValidator(history *MigrationHistory) *MigrationValidator {
	return &MigrationValidator{
		history: history,
	}
}

// Validate performs comprehensive validation on a set of migrations.
func (v *MigrationValidator) Validate(migrations []*Migration) *ValidationResult {
	result := &ValidationResult{
		Valid:             true,
		Conflicts:         make([]MigrationConflict, 0),
		PendingMigrations: make([]string, 0),
		AppliedMigrations: v.history.GetAppliedMigrations(),
	}

	// Build migration map for quick lookup
	migrationMap := make(map[string]*Migration)
	for _, m := range migrations {
		migrationMap[m.ID] = m
	}

	// Check each migration
	for _, migration := range migrations {
		// Check if already applied
		if v.history.IsApplied(migration.ID) {
			// Validate checksum
			if err := v.history.ValidateChecksum(migration); err != nil {
				if checksumErr, ok := err.(*MigrationError); ok && checksumErr.Code == "CHECKSUM_MISMATCH" {
					result.Valid = false
					result.Conflicts = append(result.Conflicts, MigrationConflict{
						Type:        ChecksumMismatch,
						MigrationID: migration.ID,
						Message:     checksumErr.Message,
						Expected:    checksumErr.Details["expected"].(string),
						Actual:      checksumErr.Details["actual"].(string),
					})
				}
			}
		} else {
			// Not applied yet - it's pending
			result.PendingMigrations = append(result.PendingMigrations, migration.ID)

			// Check dependencies
			conflicts := v.validateDependencies(migration, migrationMap)
			if len(conflicts) > 0 {
				result.Valid = false
				result.Conflicts = append(result.Conflicts, conflicts...)
			}
		}
	}

	// Check for ordering conflicts
	orderConflicts := v.validateOrdering(migrations)
	if len(orderConflicts) > 0 {
		result.Valid = false
		result.Conflicts = append(result.Conflicts, orderConflicts...)
	}

	return result
}

// validateDependencies checks if all dependencies for a migration are satisfied.
func (v *MigrationValidator) validateDependencies(migration *Migration, allMigrations map[string]*Migration) []MigrationConflict {
	conflicts := make([]MigrationConflict, 0)

	for _, depID := range migration.Dependencies {
		// Check if dependency exists
		if _, exists := allMigrations[depID]; !exists {
			conflicts = append(conflicts, MigrationConflict{
				Type:        DependencyConflict,
				MigrationID: migration.ID,
				Message:     fmt.Sprintf("dependency '%s' does not exist", depID),
				Expected:    depID,
				Actual:      "not_found",
			})
			continue
		}

		// Check if dependency is applied
		if !v.history.IsApplied(depID) {
			conflicts = append(conflicts, MigrationConflict{
				Type:        DependencyConflict,
				MigrationID: migration.ID,
				Message:     fmt.Sprintf("dependency '%s' has not been applied", depID),
				Expected:    "applied",
				Actual:      "pending",
			})
		}
	}

	return conflicts
}

// validateOrdering ensures migrations are applied in the correct order.
// This checks that migration IDs maintain sequential ordering when applied.
func (v *MigrationValidator) validateOrdering(migrations []*Migration) []MigrationConflict {
	conflicts := make([]MigrationConflict, 0)
	appliedMigrations := v.history.GetAppliedMigrations()

	if len(appliedMigrations) == 0 {
		return conflicts
	}

	// Check that no pending migration has an ID less than the latest applied
	lastApplied := appliedMigrations[len(appliedMigrations)-1]

	for _, migration := range migrations {
		if !v.history.IsApplied(migration.ID) {
			// Simple lexicographic comparison for ordering
			if migration.ID < lastApplied {
				conflicts = append(conflicts, MigrationConflict{
					Type:        OrderConflict,
					MigrationID: migration.ID,
					Message:     fmt.Sprintf("migration ID '%s' is out of order (last applied: '%s')", migration.ID, lastApplied),
					Expected:    fmt.Sprintf("> %s", lastApplied),
					Actual:      migration.ID,
				})
			}
		}
	}

	return conflicts
}

// CanRollback checks if a migration can be safely rolled back.
func (v *MigrationValidator) CanRollback(migrationID string, allMigrations []*Migration) error {
	// Check if migration is applied
	if !v.history.IsApplied(migrationID) {
		return ErrMigrationNotFound(migrationID)
	}

	// Find migrations that depend on this one
	dependents := make([]string, 0)
	for _, migration := range allMigrations {
		if v.history.IsApplied(migration.ID) {
			for _, depID := range migration.Dependencies {
				if depID == migrationID {
					dependents = append(dependents, migration.ID)
					break
				}
			}
		}
	}

	if len(dependents) > 0 {
		return &MigrationError{
			Code:    "CANNOT_ROLLBACK",
			Type:    "MIGRATION_ERROR",
			Message: fmt.Sprintf("migration '%s' cannot be rolled back - other migrations depend on it", migrationID),
			Details: map[string]interface{}{
				"migrationId": migrationID,
				"dependents":  dependents,
			},
		}
	}

	return nil
}
