package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// MigrationHistory tracks the history of applied migrations.
type MigrationHistory struct {
	records map[string]*MigrationRecord
}

// NewMigrationHistory creates a new migration history tracker.
func NewMigrationHistory() *MigrationHistory {
	return &MigrationHistory{
		records: make(map[string]*MigrationRecord),
	}
}

// RecordMigration records a migration execution.
func (h *MigrationHistory) RecordMigration(migrationID string, status MigrationStatus, executionTimeMs int64, checksum string, err error) {
	record := &MigrationRecord{
		MigrationID:     migrationID,
		AppliedAt:       time.Now(),
		Status:          status,
		ExecutionTimeMs: executionTimeMs,
		Checksum:        checksum,
	}

	if err != nil {
		record.Error = err.Error()
	}

	h.records[migrationID] = record
}

// RecordRollback records a migration rollback.
func (h *MigrationHistory) RecordRollback(migrationID string) error {
	record, exists := h.records[migrationID]
	if !exists {
		return ErrMigrationNotFound(migrationID)
	}

	now := time.Now()
	record.RolledBackAt = &now
	record.Status = RolledBack

	return nil
}

// GetRecord retrieves the record for a specific migration.
func (h *MigrationHistory) GetRecord(migrationID string) (*MigrationRecord, bool) {
	record, exists := h.records[migrationID]
	return record, exists
}

// IsApplied checks if a migration has been successfully applied.
func (h *MigrationHistory) IsApplied(migrationID string) bool {
	record, exists := h.records[migrationID]
	return exists && record.Status == Applied && record.RolledBackAt == nil
}

// GetAppliedMigrations returns a sorted list of all applied migration IDs.
func (h *MigrationHistory) GetAppliedMigrations() []string {
	var applied []string
	for id, record := range h.records {
		if record.Status == Applied && record.RolledBackAt == nil {
			applied = append(applied, id)
		}
	}
	sort.Strings(applied)
	return applied
}

// GetAllRecords returns all migration records sorted by application time.
func (h *MigrationHistory) GetAllRecords() []*MigrationRecord {
	records := make([]*MigrationRecord, 0, len(h.records))
	for _, record := range h.records {
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].AppliedAt.Before(records[j].AppliedAt)
	})

	return records
}

// LoadFromJSON deserializes migration history from JSON.
func (h *MigrationHistory) LoadFromJSON(data []byte) error {
	var records []*MigrationRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("failed to parse migration history: %w", err)
	}

	h.records = make(map[string]*MigrationRecord)
	for _, record := range records {
		h.records[record.MigrationID] = record
	}

	return nil
}

// ToJSON serializes the migration history to JSON.
func (h *MigrationHistory) ToJSON() ([]byte, error) {
	records := h.GetAllRecords()
	return json.MarshalIndent(records, "", "  ")
}

// CalculateChecksum computes a SHA-256 checksum for a migration.
func CalculateChecksum(migration *Migration) string {
	// Concatenate all commands for checksumming
	content := migration.ID + migration.Name
	for _, cmd := range migration.Up {
		content += cmd
	}
	for _, cmd := range migration.Down {
		content += cmd
	}

	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// ValidateChecksum verifies that a migration's checksum matches the recorded one.
func (h *MigrationHistory) ValidateChecksum(migration *Migration) error {
	record, exists := h.records[migration.ID]
	if !exists {
		// No record exists, so no checksum to validate
		return nil
	}

	actualChecksum := CalculateChecksum(migration)
	if actualChecksum != record.Checksum {
		return ErrChecksumMismatch(migration.ID, record.Checksum, actualChecksum)
	}

	return nil
}

// Clear removes all records from the history.
func (h *MigrationHistory) Clear() {
	h.records = make(map[string]*MigrationRecord)
}
