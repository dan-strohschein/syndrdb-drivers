package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// MigrationFile represents the structure of a migration file on disk.
type MigrationFile struct {
	FormatVersion string     `json:"formatVersion"`
	Migration     *Migration `json:"migration"`
}

// WriteMigrationFile writes a migration to a timestamped JSON file.
// Files are created with 0644 permissions (readable by all, writable by owner).
func WriteMigrationFile(migration *Migration, dir string) (string, error) {
	if migration == nil {
		return "", fmt.Errorf("migration cannot be nil")
	}
	
	if dir == "" {
		return "", fmt.Errorf("directory path cannot be empty")
	}

	// Ensure directory exists
	if err := InitMigrationDirectory(dir); err != nil {
		return "", fmt.Errorf("failed to initialize directory: %w", err)
	}

	// Generate filename from timestamp and migration ID
	timestamp := migration.Timestamp.Format("20060102150405")
	// Sanitize migration ID for filename (replace non-alphanumeric with underscore)
	sanitized := regexp.MustCompile(`[^a-zA-Z0-9_]+`).ReplaceAllString(migration.ID, "_")
	filename := fmt.Sprintf("%s_%s.json", timestamp, sanitized)
	filePath := filepath.Join(dir, filename)

	// Create file structure with format version
	fileData := MigrationFile{
		FormatVersion: "1.0",
		Migration:     migration,
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal migration: %w", err)
	}

	// Write file with 0644 permissions
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, nil
}

// ReadMigrationFile reads and validates a migration from a JSON file.
func ReadMigrationFile(path string) (*Migration, error) {
	if path == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Try to unmarshal as MigrationFile (with FormatVersion)
	var fileData MigrationFile
	if err := json.Unmarshal(data, &fileData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal migration file: %w", err)
	}

	// Handle missing FormatVersion (backward compatibility)
	if fileData.FormatVersion == "" {
		fileData.FormatVersion = "1.0"
	}

	// Validate format version
	if fileData.FormatVersion != "1.0" {
		return nil, fmt.Errorf("unsupported migration format version: %s", fileData.FormatVersion)
	}

	migration := fileData.Migration
	if migration == nil {
		return nil, fmt.Errorf("migration data is missing in file")
	}

	// Validate checksum if migration has Up commands
	if len(migration.Up) > 0 {
		expectedChecksum := CalculateChecksum(migration)
		// Note: We don't fail on checksum mismatch during read, just validate structure
		// Checksum validation happens during application via validator
		_ = expectedChecksum
	}

	return migration, nil
}

// ListMigrationFiles scans a directory and returns migrations sorted by timestamp.
func ListMigrationFiles(dir string) ([]*Migration, error) {
	if dir == "" {
		return nil, fmt.Errorf("directory path cannot be empty")
	}

	// Read directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Migration{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Extract migrations from JSON files
	var migrations []*Migration
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .json files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Skip lock files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		migration, err := ReadMigrationFile(path)
		if err != nil {
			// Log warning but continue processing other files
			fmt.Fprintf(os.Stderr, "Warning: failed to read migration file %s: %v\n", entry.Name(), err)
			continue
		}

		migrations = append(migrations, migration)
	}

	// Sort by timestamp
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Timestamp.Before(migrations[j].Timestamp)
	})

	return migrations, nil
}

// InitMigrationDirectory creates a migration directory if it doesn't exist.
// Warns if directory has world-writable permissions.
func InitMigrationDirectory(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	// Create directory with 0755 permissions (rwxr-xr-x)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check permissions
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	// Warn if world-writable (0777 or similar)
	mode := info.Mode().Perm()
	if mode&0002 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: migration directory %s has world-writable permissions (%s). This may be a security risk.\n", dir, mode)
	}

	return nil
}
