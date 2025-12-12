package migration

import (
	"testing"
)

func TestGenerateDown_CreateBundle(t *testing.T) {
	gen := NewRollbackGenerator()

	upCmd := `CREATE BUNDLE "users" WITH FIELDS (
		{"id", "INT", TRUE, TRUE, NULL},
		{"name", "STRING", TRUE, FALSE, NULL}
	);`

	downCmd, err := gen.generateSingleDown(upCmd)
	if err != nil {
		t.Fatalf("failed to generate down: %v", err)
	}

	expected := `DROP BUNDLE "users";`
	if downCmd != expected {
		t.Errorf("expected %q, got %q", expected, downCmd)
	}
}

func TestGenerateDown_CreateHashIndex(t *testing.T) {
	gen := NewRollbackGenerator()

	upCmd := `CREATE HASH INDEX "idx_users_email" ON BUNDLE "users" WITH FIELDS ("email");`

	downCmd, err := gen.generateSingleDown(upCmd)
	if err != nil {
		t.Fatalf("failed to generate down: %v", err)
	}

	expected := `DROP INDEX "idx_users_email";`
	if downCmd != expected {
		t.Errorf("expected %q, got %q", expected, downCmd)
	}
}

func TestGenerateDown_CreateBTreeIndex(t *testing.T) {
	gen := NewRollbackGenerator()

	upCmd := `CREATE B-INDEX "idx_users_name" ON BUNDLE "users" WITH FIELDS ("name");`

	downCmd, err := gen.generateSingleDown(upCmd)
	if err != nil {
		t.Fatalf("failed to generate down: %v", err)
	}

	expected := `DROP INDEX "idx_users_name";`
	if downCmd != expected {
		t.Errorf("expected %q, got %q", expected, downCmd)
	}
}

func TestGenerateDown_UpdateBundleAdd(t *testing.T) {
	gen := NewRollbackGenerator()

	upCmd := `UPDATE BUNDLE "users"
SET (
    {ADD "age" = "age", "INT", FALSE, FALSE, NULL}
);`

	downCmd, err := gen.generateSingleDown(upCmd)
	if err != nil {
		t.Fatalf("failed to generate down: %v", err)
	}

	if !contains(downCmd, `{REMOVE "age"`) {
		t.Errorf("expected REMOVE age in down command, got %q", downCmd)
	}
}

func TestGenerateDown_AddRelationship(t *testing.T) {
	gen := NewRollbackGenerator()

	upCmd := `UPDATE BUNDLE "users" ADD RELATIONSHIP ("posts" {"1toMany", "users", "id", "posts", "user_id"});`

	downCmd, err := gen.generateSingleDown(upCmd)
	if err != nil {
		t.Fatalf("failed to generate down: %v", err)
	}

	expected := `UPDATE BUNDLE "users" REMOVE RELATIONSHIP "posts";`
	if downCmd != expected {
		t.Errorf("expected %q, got %q", expected, downCmd)
	}
}

func TestGenerateDown_NonReversible_DropBundle(t *testing.T) {
	gen := NewRollbackGenerator()

	upCmd := `DROP BUNDLE "users";`

	_, err := gen.generateSingleDown(upCmd)
	if err == nil {
		t.Error("expected error for non-reversible DROP BUNDLE, got nil")
	}
}

func TestGenerateDown_NonReversible_UpdateRemove(t *testing.T) {
	gen := NewRollbackGenerator()

	upCmd := `UPDATE BUNDLE "users" SET ({REMOVE "old_field" = "", "", FALSE, FALSE, NULL});`

	_, err := gen.generateSingleDown(upCmd)
	if err == nil {
		t.Error("expected error for non-reversible UPDATE REMOVE, got nil")
	}
}

func TestGenerateDown_NonReversible_UpdateModify(t *testing.T) {
	gen := NewRollbackGenerator()

	upCmd := `UPDATE BUNDLE "users" SET ({MODIFY "status" = "status", "STRING", TRUE, FALSE, "active"});`

	_, err := gen.generateSingleDown(upCmd)
	if err == nil {
		t.Error("expected error for non-reversible UPDATE MODIFY, got nil")
	}
}

func TestGenerateDown_MultipleCommands(t *testing.T) {
	gen := NewRollbackGenerator()

	upCommands := []string{
		`CREATE BUNDLE "users" WITH FIELDS ({"id", "INT", TRUE, TRUE, NULL});`,
		`CREATE HASH INDEX "idx_email" ON BUNDLE "users" WITH FIELDS ("email");`,
		`UPDATE BUNDLE "users" SET ({ADD "age" = "age", "INT", FALSE, FALSE, NULL});`,
	}

	downCommands, err := gen.GenerateDown(upCommands)
	if err != nil {
		t.Fatalf("failed to generate down: %v", err)
	}

	// Should be in reverse order
	if len(downCommands) != 3 {
		t.Fatalf("expected 3 down commands, got %d", len(downCommands))
	}

	// First down should be last up reversed
	if !contains(downCommands[0], "REMOVE") {
		t.Errorf("expected first down to be REMOVE (from last up ADD), got %q", downCommands[0])
	}

	// Second down should be second up reversed
	if !contains(downCommands[1], `DROP INDEX "idx_email"`) {
		t.Errorf("expected second down to be DROP INDEX, got %q", downCommands[1])
	}

	// Third down should be first up reversed
	if !contains(downCommands[2], `DROP BUNDLE "users"`) {
		t.Errorf("expected third down to be DROP BUNDLE, got %q", downCommands[2])
	}
}

func TestCanGenerateDown(t *testing.T) {
	gen := NewRollbackGenerator()

	tests := []struct {
		cmd      string
		expected bool
	}{
		{`CREATE BUNDLE "users" WITH FIELDS (...)`, true},
		{`CREATE HASH INDEX "idx" ON BUNDLE "users" WITH FIELDS ("email")`, true},
		{`CREATE B-INDEX "idx" ON BUNDLE "users" WITH FIELDS ("name")`, true},
		{`UPDATE BUNDLE "users" SET ({ADD "age" = ...})`, true},
		{`UPDATE BUNDLE "users" ADD RELATIONSHIP ("posts" {...})`, true},
		{`DROP BUNDLE "users"`, false},
		{`DROP INDEX "idx"`, false},
		{`UPDATE BUNDLE "users" SET ({REMOVE "field" = ...})`, false},
		{`UPDATE BUNDLE "users" SET ({MODIFY "field" = ...})`, false},
		{`DELETE FROM users WHERE id = 1`, false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := gen.CanGenerateDown(tt.cmd)
			if got != tt.expected {
				t.Errorf("CanGenerateDown(%q) = %v, want %v", tt.cmd, got, tt.expected)
			}
		})
	}
}

func TestValidateDownCommands(t *testing.T) {
	gen := NewRollbackGenerator()

	upCommands := []string{
		`CREATE BUNDLE "users" WITH FIELDS (...)`,
		`CREATE INDEX "idx" ON BUNDLE "users" WITH FIELDS ("email")`,
	}

	downCommands := []string{
		`DROP INDEX "idx";`,
		`DROP BUNDLE "users";`,
	}

	err := gen.ValidateDownCommands(upCommands, downCommands)
	if err != nil {
		t.Errorf("expected validation to pass, got error: %v", err)
	}
}

func TestValidateDownCommands_TooMany(t *testing.T) {
	gen := NewRollbackGenerator()

	upCommands := []string{
		`CREATE BUNDLE "users" WITH FIELDS (...)`,
	}

	downCommands := []string{
		`DROP BUNDLE "users";`,
		`DROP INDEX "extra";`,
	}

	err := gen.ValidateDownCommands(upCommands, downCommands)
	if err == nil {
		t.Error("expected validation error for too many down commands, got nil")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
