package migration

import (
	"testing"
)

func TestMigrationValidator_Validate(t *testing.T) {
	history := NewMigrationHistory()
	validator := NewMigrationValidator(history)

	migrations := []*Migration{
		{
			ID:   "001_test",
			Name: "Test Migration",
			Up:   []string{`CREATE BUNDLE "users" WITH FIELDS (...)`},
			Down: []string{`DROP BUNDLE "users";`},
		},
	}

	result := validator.Validate(migrations)

	if result == nil {
		t.Fatal("expected validation result, got nil")
	}

	// Should have pending migrations
	if len(result.PendingMigrations) == 0 {
		t.Error("expected pending migrations")
	}
}

func TestMigrationValidator_DetectChecksumMismatch(t *testing.T) {
	history := NewMigrationHistory()

	migration := &Migration{
		ID:   "001_test",
		Name: "Test Migration",
		Up:   []string{`CREATE BUNDLE "users" WITH FIELDS (...)`},
		Down: []string{`DROP BUNDLE "users";`},
	}

	// Record with one checksum
	checksum := CalculateChecksum(migration)
	history.RecordMigration("001_test", Applied, 100, checksum, nil)

	// Modify migration
	migration.Up[0] = `CREATE BUNDLE "modified" WITH FIELDS (...)`

	validator := NewMigrationValidator(history)
	result := validator.Validate([]*Migration{migration})

	// Should detect checksum mismatch
	if len(result.Conflicts) == 0 {
		t.Error("expected conflicts for checksum mismatch")
	}

	foundMismatch := false
	for _, conflict := range result.Conflicts {
		if conflict.Type == ChecksumMismatch {
			foundMismatch = true
			break
		}
	}

	if !foundMismatch {
		t.Error("expected ChecksumMismatch conflict type")
	}
}

func TestMigrationValidator_ValidDependencies(t *testing.T) {
	history := NewMigrationHistory()

	migrations := []*Migration{
		{
			ID:           "001_first",
			Name:         "First Migration",
			Up:           []string{`CREATE BUNDLE "users" WITH FIELDS (...)`},
			Down:         []string{`DROP BUNDLE "users";`},
			Dependencies: []string{},
		},
		{
			ID:           "002_second",
			Name:         "Second Migration",
			Up:           []string{`CREATE BUNDLE "posts" WITH FIELDS (...)`},
			Down:         []string{`DROP BUNDLE "posts";`},
			Dependencies: []string{"001_first"},
		},
	}

	// Apply first migration with correct checksum
	checksum := CalculateChecksum(migrations[0])
	history.RecordMigration("001_first", Applied, 100, checksum, nil)

	validator := NewMigrationValidator(history)
	result := validator.Validate(migrations)

	// Should be valid with no conflicts (001 already applied with correct checksum, 002 pending with valid dep)
	if !result.Valid {
		t.Errorf("expected valid result, got conflicts: %v", result.Conflicts)
	}
}

func TestMigrationValidator_MissingDependency(t *testing.T) {
	history := NewMigrationHistory()
	validator := NewMigrationValidator(history)

	migrations := []*Migration{
		{
			ID:           "002_second",
			Name:         "Second Migration",
			Up:           []string{`CREATE BUNDLE "posts" WITH FIELDS (...)`},
			Down:         []string{`DROP BUNDLE "posts";`},
			Dependencies: []string{"001_missing"},
		},
	}

	result := validator.Validate(migrations)

	// Should have dependency conflict
	if result.Valid {
		t.Error("expected invalid result for missing dependency")
	}

	foundDepConflict := false
	for _, conflict := range result.Conflicts {
		if conflict.Type == DependencyConflict {
			foundDepConflict = true
			break
		}
	}

	if !foundDepConflict {
		t.Error("expected DependencyConflict")
	}
}
