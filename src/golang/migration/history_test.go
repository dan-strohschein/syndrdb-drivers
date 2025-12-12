package migration

import (
	"testing"
	"time"
)

func TestNewMigrationHistory(t *testing.T) {
	history := NewMigrationHistory()

	if history == nil {
		t.Fatal("NewMigrationHistory returned nil")
	}

	if len(history.records) != 0 {
		t.Errorf("expected empty history, got %d records", len(history.records))
	}
}

func TestRecordMigration(t *testing.T) {
	history := NewMigrationHistory()

	history.RecordMigration("001_test", Applied, 150, "abc123", nil)

	record, exists := history.GetRecord("001_test")
	if !exists {
		t.Fatal("expected record to exist")
	}

	if record.MigrationID != "001_test" {
		t.Errorf("expected ID=001_test, got %s", record.MigrationID)
	}

	if record.Status != Applied {
		t.Errorf("expected status=Applied, got %s", record.Status)
	}

	if record.ExecutionTimeMs != 150 {
		t.Errorf("expected execution time=150, got %d", record.ExecutionTimeMs)
	}

	if record.Checksum != "abc123" {
		t.Errorf("expected checksum=abc123, got %s", record.Checksum)
	}
}

func TestRecordMigration_WithError(t *testing.T) {
	history := NewMigrationHistory()

	testErr := &MigrationError{
		Code:    "TEST_ERROR",
		Type:    "MIGRATION_ERROR",
		Message: "test error",
		Details: map[string]interface{}{},
	}

	history.RecordMigration("001_test", Failed, 150, "abc123", testErr)

	record, exists := history.GetRecord("001_test")
	if !exists {
		t.Fatal("expected record to exist")
	}

	if record.Status != Failed {
		t.Errorf("expected status=Failed, got %s", record.Status)
	}

	if record.Error == "" {
		t.Error("expected error message in record, got empty string")
	}
}

func TestRecordRollback(t *testing.T) {
	history := NewMigrationHistory()

	// First record a migration
	history.RecordMigration("001_test", Applied, 150, "abc123", nil)

	// Then rollback
	err := history.RecordRollback("001_test")
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	record, _ := history.GetRecord("001_test")

	if record.Status != RolledBack {
		t.Errorf("expected status=RolledBack, got %s", record.Status)
	}

	if record.RolledBackAt == nil {
		t.Error("expected RolledBackAt to be set, got nil")
	}
}

func TestRecordRollback_NotFound(t *testing.T) {
	history := NewMigrationHistory()

	err := history.RecordRollback("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent migration, got nil")
	}
}

func TestIsApplied(t *testing.T) {
	history := NewMigrationHistory()

	// Not applied yet
	if history.IsApplied("001_test") {
		t.Error("expected IsApplied=false for non-applied migration")
	}

	// Apply it
	history.RecordMigration("001_test", Applied, 150, "abc123", nil)

	if !history.IsApplied("001_test") {
		t.Error("expected IsApplied=true for applied migration")
	}

	// Rollback
	history.RecordRollback("001_test")

	if history.IsApplied("001_test") {
		t.Error("expected IsApplied=false for rolled back migration")
	}
}

func TestGetAppliedMigrations(t *testing.T) {
	history := NewMigrationHistory()

	history.RecordMigration("001_first", Applied, 100, "abc1", nil)
	history.RecordMigration("002_second", Applied, 100, "abc2", nil)
	history.RecordMigration("003_third", Failed, 100, "abc3", nil)

	applied := history.GetAppliedMigrations()

	if len(applied) != 2 {
		t.Fatalf("expected 2 applied migrations, got %d", len(applied))
	}

	// Should be sorted
	if applied[0] != "001_first" {
		t.Errorf("expected first to be 001_first, got %s", applied[0])
	}

	if applied[1] != "002_second" {
		t.Errorf("expected second to be 002_second, got %s", applied[1])
	}
}

func TestCalculateChecksum(t *testing.T) {
	migration := &Migration{
		ID:   "001_test",
		Name: "Test Migration",
		Up:   []string{`CREATE BUNDLE "users" WITH FIELDS (...)`},
		Down: []string{`DROP BUNDLE "users";`},
	}

	checksum := CalculateChecksum(migration)

	if checksum == "" {
		t.Error("expected non-empty checksum")
	}

	// Should be consistent
	checksum2 := CalculateChecksum(migration)
	if checksum != checksum2 {
		t.Errorf("expected consistent checksums, got %s and %s", checksum, checksum2)
	}

	// Different migration should have different checksum
	migration2 := &Migration{
		ID:   "002_different",
		Name: "Different Migration",
		Up:   []string{`CREATE BUNDLE "products" WITH FIELDS (...)`},
		Down: []string{`DROP BUNDLE "products";`},
	}

	checksum3 := CalculateChecksum(migration2)
	if checksum == checksum3 {
		t.Error("expected different checksums for different migrations")
	}
}

func TestValidateChecksum(t *testing.T) {
	history := NewMigrationHistory()

	migration := &Migration{
		ID:   "001_test",
		Name: "Test Migration",
		Up:   []string{`CREATE BUNDLE "users" WITH FIELDS (...)`},
		Down: []string{`DROP BUNDLE "users";`},
	}

	checksum := CalculateChecksum(migration)
	history.RecordMigration("001_test", Applied, 100, checksum, nil)

	// Should validate successfully
	err := history.ValidateChecksum(migration)
	if err != nil {
		t.Errorf("expected validation to pass, got error: %v", err)
	}

	// Modify migration
	migration.Up[0] = `CREATE BUNDLE "modified" WITH FIELDS (...)`

	// Should fail validation
	err = history.ValidateChecksum(migration)
	if err == nil {
		t.Error("expected checksum mismatch error, got nil")
	}
}

func TestToJSON_LoadFromJSON(t *testing.T) {
	history := NewMigrationHistory()

	history.RecordMigration("001_first", Applied, 100, "abc1", nil)
	history.RecordMigration("002_second", Applied, 150, "abc2", nil)

	// Export to JSON
	data, err := history.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Create new history and load
	history2 := NewMigrationHistory()
	err = history2.LoadFromJSON(data)
	if err != nil {
		t.Fatalf("LoadFromJSON failed: %v", err)
	}

	// Should have same records
	if len(history2.records) != 2 {
		t.Errorf("expected 2 records, got %d", len(history2.records))
	}

	record, exists := history2.GetRecord("001_first")
	if !exists {
		t.Error("expected 001_first to exist")
	}

	if record.Checksum != "abc1" {
		t.Errorf("expected checksum=abc1, got %s", record.Checksum)
	}
}

func TestGetAllRecords_Sorted(t *testing.T) {
	history := NewMigrationHistory()

	// Add in non-chronological order
	time.Sleep(1 * time.Millisecond)
	history.RecordMigration("003_third", Applied, 100, "abc3", nil)

	time.Sleep(1 * time.Millisecond)
	history.RecordMigration("001_first", Applied, 100, "abc1", nil)

	time.Sleep(1 * time.Millisecond)
	history.RecordMigration("002_second", Applied, 100, "abc2", nil)

	records := history.GetAllRecords()

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	// Should be sorted by application time
	if records[0].MigrationID != "003_third" {
		t.Errorf("expected first record to be 003_third, got %s", records[0].MigrationID)
	}

	if records[1].MigrationID != "001_first" {
		t.Errorf("expected second record to be 001_first, got %s", records[1].MigrationID)
	}

	if records[2].MigrationID != "002_second" {
		t.Errorf("expected third record to be 002_second, got %s", records[2].MigrationID)
	}
}

func TestClear(t *testing.T) {
	history := NewMigrationHistory()

	history.RecordMigration("001_test", Applied, 100, "abc1", nil)
	history.RecordMigration("002_test", Applied, 100, "abc2", nil)

	if len(history.records) != 2 {
		t.Fatalf("expected 2 records before clear, got %d", len(history.records))
	}

	history.Clear()

	if len(history.records) != 0 {
		t.Errorf("expected 0 records after clear, got %d", len(history.records))
	}
}
