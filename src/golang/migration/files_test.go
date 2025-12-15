package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWriteAndReadMigrationFile tests the complete write-read cycle
func TestWriteAndReadMigrationFile(t *testing.T) {
	tmpDir := t.TempDir()

	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	migration := &Migration{
		ID:        "create_users_bundle",
		Name:      "Create users bundle",
		Timestamp: timestamp,
		Up:        []string{`CREATE BUNDLE "users" WITH FIELDS ({"id", "int", TRUE, TRUE, 0})`},
		Down:      []string{`DROP BUNDLE "users";`},
	}

	// Write migration
	filePath, err := WriteMigrationFile(migration, tmpDir)
	if err != nil {
		t.Fatalf("WriteMigrationFile failed: %v", err)
	}

	// Read it back
	readMigration, err := ReadMigrationFile(filePath)
	if err != nil {
		t.Fatalf("ReadMigrationFile failed: %v", err)
	}

	// Verify content
	if readMigration.ID != migration.ID {
		t.Errorf("ID mismatch: expected %s, got %s", migration.ID, readMigration.ID)
	}

	if len(readMigration.Up) != len(migration.Up) {
		t.Errorf("Up commands mismatch: expected %d, got %d", len(migration.Up), len(readMigration.Up))
	}
}

// TestFormatVersion tests that all files have formatVersion "1.0"
func TestFormatVersion(t *testing.T) {
	tmpDir := t.TempDir()

	migration := &Migration{
		ID:        "test",
		Name:      "Test",
		Timestamp: time.Now(),
		Up:        []string{`CREATE BUNDLE "test" WITH FIELDS ({"id", "int", TRUE, FALSE, 0})`},
		Down:      []string{`DROP BUNDLE "test";`},
	}

	filePath, err := WriteMigrationFile(migration, tmpDir)
	if err != nil {
		t.Fatalf("WriteMigrationFile failed: %v", err)
	}

	// Read raw JSON to verify formatVersion field
	data, _ := os.ReadFile(filePath)
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	version, ok := raw["formatVersion"]
	if !ok {
		t.Error("formatVersion field missing")
	}

	if version != "1.0" {
		t.Errorf("Expected formatVersion '1.0', got '%v'", version)
	}
}

// TestListMigrationFiles tests directory listing
func TestListMigrationFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple migrations
	timestamps := []time.Time{
		time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
	}

	for i, ts := range timestamps {
		migration := &Migration{
			ID:        fmt.Sprintf("mig_%d", i),
			Name:      "Test",
			Timestamp: ts,
			Up:        []string{`CREATE BUNDLE "test" WITH FIELDS ({"id", "int", TRUE, FALSE, 0})`},
			Down:      []string{`DROP BUNDLE "test";`},
		}
		WriteMigrationFile(migration, tmpDir)
	}

	// List migrations
	migrations, err := ListMigrationFiles(tmpDir)
	if err != nil {
		t.Fatalf("ListMigrationFiles failed: %v", err)
	}

	if len(migrations) != 3 {
		t.Errorf("Expected 3 migrations, got %d", len(migrations))
	}

	// Verify sorting by timestamp (earliest first)
	if !migrations[0].Timestamp.Before(migrations[1].Timestamp) {
		t.Error("Migrations not sorted by timestamp")
	}
}

// TestInitMigrationDirectory tests directory creation
func TestInitMigrationDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	migDir := filepath.Join(tmpDir, "migrations")

	err := InitMigrationDirectory(migDir)
	if err != nil {
		t.Fatalf("InitMigrationDirectory failed: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(migDir)
	if err != nil {
		t.Fatalf("Directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Path is not a directory")
	}

	// Verify permissions (0755)
	expectedMode := os.FileMode(0755)
	if info.Mode().Perm() != expectedMode {
		t.Errorf("Expected permissions %s, got %s", expectedMode, info.Mode().Perm())
	}
}
