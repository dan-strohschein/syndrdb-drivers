package migration

import (
	"fmt"
	"regexp"
	"strings"
)

// RollbackGenerator generates Down commands from Up commands automatically.
// This enables safe rollback without manually writing reverse operations.
type RollbackGenerator struct{}

// NewRollbackGenerator creates a new rollback generator.
func NewRollbackGenerator() *RollbackGenerator {
	return &RollbackGenerator{}
}

// GenerateDown automatically generates Down commands from Up commands.
// Returns the generated commands and any errors encountered.
func (g *RollbackGenerator) GenerateDown(upCommands []string) ([]string, error) {
	downCommands := make([]string, 0, len(upCommands))

	// Process in reverse order for proper rollback sequence
	for i := len(upCommands) - 1; i >= 0; i-- {
		down, err := g.generateSingleDown(upCommands[i])
		if err != nil {
			return nil, fmt.Errorf("failed to generate down command for up[%d]: %w", i, err)
		}
		if down != "" {
			downCommands = append(downCommands, down)
		}
	}

	return downCommands, nil
}

// generateSingleDown generates the reverse operation for a single command.
func (g *RollbackGenerator) generateSingleDown(upCommand string) (string, error) {
	normalized := strings.TrimSpace(upCommand)
	normalizedUpper := strings.ToUpper(normalized)

	// CREATE BUNDLE → DROP BUNDLE
	if strings.HasPrefix(normalizedUpper, "CREATE BUNDLE") {
		return g.reversCreateBundle(normalized)
	}

	// UPDATE BUNDLE SET → UPDATE BUNDLE SET (reverse operations)
	if strings.HasPrefix(normalizedUpper, "UPDATE BUNDLE") && strings.Contains(normalizedUpper, "SET") {
		return g.reverseUpdateBundle(normalized)
	}

	// CREATE INDEX → DROP INDEX
	if strings.HasPrefix(normalizedUpper, "CREATE") && strings.Contains(normalizedUpper, "INDEX") {
		return g.reverseCreateIndex(normalized)
	}

	// DROP BUNDLE → Not reversible (would need schema)
	if strings.HasPrefix(normalizedUpper, "DROP BUNDLE") {
		return "", fmt.Errorf("DROP BUNDLE cannot be automatically reversed (schema information required)")
	}

	// DROP INDEX → Not reversible (would need index definition)
	if strings.HasPrefix(normalizedUpper, "DROP INDEX") {
		return "", fmt.Errorf("DROP INDEX cannot be automatically reversed (index definition required)")
	}

	// ADD RELATIONSHIP → REMOVE RELATIONSHIP
	if strings.Contains(normalizedUpper, "ADD RELATIONSHIP") {
		return g.reverseAddRelationship(normalized)
	}

	// REMOVE RELATIONSHIP → ADD RELATIONSHIP (not reversible without definition)
	if strings.Contains(normalizedUpper, "REMOVE RELATIONSHIP") {
		return "", fmt.Errorf("REMOVE RELATIONSHIP cannot be automatically reversed (relationship definition required)")
	}

	// INSERT → DELETE (if we support these)
	if strings.HasPrefix(normalizedUpper, "INSERT INTO") {
		return g.reverseInsert(normalized)
	}

	// DELETE → Not reversible (would need deleted data)
	if strings.HasPrefix(normalizedUpper, "DELETE FROM") {
		return "", fmt.Errorf("DELETE FROM cannot be automatically reversed (deleted data required)")
	}

	// Unknown command type
	return "", fmt.Errorf("cannot automatically reverse command type: %s", normalized)
}

// reversCreateBundle generates DROP BUNDLE from CREATE BUNDLE
func (g *RollbackGenerator) reversCreateBundle(createCmd string) (string, error) {
	// Extract bundle name using regex
	// Pattern: CREATE BUNDLE "bundleName" or CREATE BUNDLE `bundleName`
	re := regexp.MustCompile(`(?i)CREATE\s+BUNDLE\s+["'` + "`" + `]([^"'` + "`" + `]+)["'` + "`" + `]`)
	matches := re.FindStringSubmatch(createCmd)

	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract bundle name from CREATE BUNDLE command")
	}

	bundleName := matches[1]
	return fmt.Sprintf(`DROP BUNDLE "%s";`, bundleName), nil
}

// reverseUpdateBundle generates reverse UPDATE BUNDLE SET operations
func (g *RollbackGenerator) reverseUpdateBundle(updateCmd string) (string, error) {
	// Extract bundle name
	re := regexp.MustCompile(`(?i)UPDATE\s+BUNDLE\s+["'` + "`" + `]([^"'` + "`" + `]+)["'` + "`" + `]\s+SET`)
	matches := re.FindStringSubmatch(updateCmd)

	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract bundle name from UPDATE BUNDLE command")
	}

	bundleName := matches[1]

	// Parse field operations
	// Look for ADD, REMOVE, MODIFY operations
	var reverseOps []string

	// ADD field → REMOVE field
	addRe := regexp.MustCompile(`(?i)\{ADD\s+["']([^"']+)["']\s+=\s+["']([^"']+)["'],\s+["']([^"']+)["'],\s+([^,]+),\s+([^,]+),\s+([^}]+)\}`)
	addMatches := addRe.FindAllStringSubmatch(updateCmd, -1)
	for _, match := range addMatches {
		if len(match) > 1 {
			fieldName := match[1]
			reverseOps = append(reverseOps, fmt.Sprintf(`    {REMOVE "%s" = "", "", FALSE, FALSE, NULL}`, fieldName))
		}
	}

	// REMOVE field → Cannot auto-reverse (need field definition)
	if strings.Contains(strings.ToUpper(updateCmd), "{REMOVE") {
		return "", fmt.Errorf("UPDATE BUNDLE SET REMOVE cannot be automatically reversed (field definition required)")
	}

	// MODIFY field → Store original and reverse (complex, skip for now)
	if strings.Contains(strings.ToUpper(updateCmd), "{MODIFY") {
		return "", fmt.Errorf("UPDATE BUNDLE SET MODIFY cannot be automatically reversed (original field state required)")
	}

	if len(reverseOps) == 0 {
		return "", fmt.Errorf("no reversible operations found in UPDATE BUNDLE command")
	}

	return fmt.Sprintf("UPDATE BUNDLE \"%s\"\nSET (\n%s\n);", bundleName, strings.Join(reverseOps, ",\n")), nil
}

// reverseCreateIndex generates DROP INDEX from CREATE INDEX
func (g *RollbackGenerator) reverseCreateIndex(createCmd string) (string, error) {
	// Pattern: CREATE [HASH|B-]INDEX "indexName" ON BUNDLE "bundleName"
	re := regexp.MustCompile(`(?i)CREATE\s+(?:HASH\s+|B-)?INDEX\s+["'` + "`" + `]([^"'` + "`" + `]+)["'` + "`" + `]`)
	matches := re.FindStringSubmatch(createCmd)

	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract index name from CREATE INDEX command")
	}

	indexName := matches[1]
	return fmt.Sprintf(`DROP INDEX "%s";`, indexName), nil
}

// reverseAddRelationship generates REMOVE RELATIONSHIP from ADD RELATIONSHIP
func (g *RollbackGenerator) reverseAddRelationship(addCmd string) (string, error) {
	// Pattern: UPDATE BUNDLE "bundleName" ADD RELATIONSHIP ("relName" {...})
	bundleRe := regexp.MustCompile(`(?i)UPDATE\s+BUNDLE\s+["'` + "`" + `]([^"'` + "`" + `]+)["'` + "`" + `]`)
	bundleMatches := bundleRe.FindStringSubmatch(addCmd)

	relRe := regexp.MustCompile(`(?i)ADD\s+RELATIONSHIP\s+\(["']([^"']+)["']`)
	relMatches := relRe.FindStringSubmatch(addCmd)

	if len(bundleMatches) < 2 || len(relMatches) < 2 {
		return "", fmt.Errorf("could not extract bundle name or relationship name from ADD RELATIONSHIP command")
	}

	bundleName := bundleMatches[1]
	relName := relMatches[1]

	return fmt.Sprintf(`UPDATE BUNDLE "%s" REMOVE RELATIONSHIP "%s";`, bundleName, relName), nil
}

// reverseInsert generates DELETE from INSERT (if we support data operations)
func (g *RollbackGenerator) reverseInsert(insertCmd string) (string, error) {
	// This would require tracking inserted IDs
	// For now, return error as it needs runtime information
	return "", fmt.Errorf("INSERT cannot be automatically reversed without tracking inserted record IDs")
}

// CanGenerateDown checks if a command can be automatically reversed.
func (g *RollbackGenerator) CanGenerateDown(upCommand string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(upCommand))

	// Commands we CAN reverse
	reversible := []string{
		"CREATE BUNDLE",
		"CREATE HASH INDEX",
		"CREATE B-INDEX",
		"CREATE INDEX",
	}

	for _, prefix := range reversible {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}

	// UPDATE BUNDLE SET ADD (specific case)
	if strings.HasPrefix(normalized, "UPDATE BUNDLE") && strings.Contains(normalized, "SET") && strings.Contains(normalized, "{ADD") {
		return true
	}

	// ADD RELATIONSHIP
	if strings.Contains(normalized, "ADD RELATIONSHIP") {
		return true
	}

	return false
}

// ValidateDownCommands checks if generated Down commands are valid reverses of Up commands.
func (g *RollbackGenerator) ValidateDownCommands(upCommands, downCommands []string) error {
	if len(downCommands) > len(upCommands) {
		return fmt.Errorf("more down commands (%d) than up commands (%d)", len(downCommands), len(upCommands))
	}

	// Basic validation: ensure we have reverses for reversible commands
	reversibleCount := 0
	for _, up := range upCommands {
		if g.CanGenerateDown(up) {
			reversibleCount++
		}
	}

	if len(downCommands) != reversibleCount {
		return fmt.Errorf("expected %d down commands for %d reversible up commands, got %d", reversibleCount, reversibleCount, len(downCommands))
	}

	return nil
}
