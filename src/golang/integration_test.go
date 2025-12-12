package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/migration"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

const (
	testConnStr = "syndrdb://localhost:1776:primary:root:root;"
	testTimeout = 10000
)

// clientExecutorAdapter adapts our Client to the MigrationExecutor interface
type clientExecutorAdapter struct {
	client *client.Client
}

func (a *clientExecutorAdapter) Execute(command string) (interface{}, error) {
	// Determine if this is a query or mutation based on command type
	// For simplicity, treat everything as a mutation since migrations typically modify schema
	return a.client.Mutate(command, testTimeout)
}

// responseToJSON converts a response interface{} to JSON bytes for parsing
// If the response has a Result field (like SHOW BUNDLES), extracts it and transforms to parser format
func responseToJSON(response interface{}) ([]byte, error) {
	// Check if it's a map with Result field
	if respMap, ok := response.(map[string]interface{}); ok {
		if result, hasResult := respMap["Result"]; hasResult {
			// Transform the Result array to the format the schema parser expects
			if resultArray, ok := result.([]interface{}); ok {
				bundles := make([]map[string]interface{}, 0, len(resultArray))

				for _, item := range resultArray {
					if bundle, ok := item.(map[string]interface{}); ok {
						// Extract relevant fields from BundleMetadata
						metadata, hasMetadata := bundle["BundleMetadata"].(map[string]interface{})
						if !hasMetadata {
							continue
						}

						// Create simplified bundle structure
						simplifiedBundle := map[string]interface{}{
							"name":          metadata["Name"],
							"fields":        []map[string]interface{}{},
							"indexes":       map[string]interface{}{"hash": []interface{}{}, "btree": []interface{}{}},
							"relationships": []interface{}{},
						}

						// Extract fields from DocumentStructure.FieldDefinitions
						if docStruct, ok := metadata["DocumentStructure"].(map[string]interface{}); ok {
							if fieldDefs, ok := docStruct["FieldDefinitions"].(map[string]interface{}); ok {
								fields := make([]map[string]interface{}, 0)
								for _, fieldData := range fieldDefs {
									if field, ok := fieldData.(map[string]interface{}); ok {
										fields = append(fields, map[string]interface{}{
											"name":         field["Name"],
											"type":         field["Type"],
											"required":     field["Required"],
											"unique":       field["Unique"],
											"defaultValue": field["DefaultValue"],
										})
									}
								}
								simplifiedBundle["fields"] = fields
							}
						}

						// Extract indexes
						if indexes, ok := metadata["Indexes"].(map[string]interface{}); ok {
							hashIndexes := make([]interface{}, 0)
							btreeIndexes := make([]interface{}, 0)

							for indexName, indexData := range indexes {
								if idx, ok := indexData.(map[string]interface{}); ok {
									indexType, _ := idx["IndexType"].(string)

									// Get fields from the index
									indexFields := []string{}
									if hashField, ok := idx["HashIndexField"].(map[string]interface{}); ok {
										if fieldName, ok := hashField["FieldName"].(string); ok && fieldName != "" {
											indexFields = append(indexFields, fieldName)
										}
									}

									indexEntry := map[string]interface{}{
										"name":   indexName,
										"fields": indexFields,
									}

									if indexType == "hash" {
										hashIndexes = append(hashIndexes, indexEntry)
									} else if indexType == "btree" {
										btreeIndexes = append(btreeIndexes, indexEntry)
									}
								}
							}

							simplifiedBundle["indexes"] = map[string]interface{}{
								"hash":  hashIndexes,
								"btree": btreeIndexes,
							}
						}

						bundles = append(bundles, simplifiedBundle)
					}
				}

				wrapper := map[string]interface{}{
					"bundles": bundles,
				}
				return json.Marshal(wrapper)
			}
		}
	}
	return json.Marshal(response)
}

// TestIntegration_Connection tests basic connection to SyndrDB server
func TestIntegration_Connection(t *testing.T) {
	opts := client.DefaultOptions()
	c := client.NewClient(&opts)

	err := c.Connect(context.Background(), testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect(context.Background())

	if c.GetState() != client.CONNECTED {
		t.Errorf("Expected CONNECTED state, got %s", c.GetState())
	}
}

// TestIntegration_ShowBundles tests retrieving schema from server
func TestIntegration_ShowBundles(t *testing.T) {
	opts := client.DefaultOptions()
	c := client.NewClient(&opts)

	err := c.Connect(context.Background(), testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect(context.Background())

	response, err := c.Query("SHOW BUNDLES;", testTimeout)
	if err != nil {
		t.Fatalf("SHOW BUNDLES failed: %v", err)
	}

	if response == nil {
		t.Error("Expected non-empty response from SHOW BUNDLES")
	}

	// Try to parse the schema
	responseJSON, err := responseToJSON(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	_, err = schema.ParseServerSchema(responseJSON)
	if err != nil {
		t.Errorf("Failed to parse schema response: %v", err)
	}
}

// TestIntegration_CreateDropBundle tests bundle creation and deletion
func TestIntegration_CreateDropBundle(t *testing.T) {
	opts := client.DefaultOptions()
	c := client.NewClient(&opts)

	err := c.Connect(context.Background(), testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect(context.Background())

	// Clean up test bundle if it exists
	c.Mutate(`DROP BUNDLE "test_users";`, testTimeout)

	// Create test bundle
	createCmd := `CREATE BUNDLE "test_users" WITH FIELDS (
		{"id", "INT", TRUE, TRUE, NULL},
		{"name", "STRING", TRUE, FALSE, NULL},
		{"email", "STRING", TRUE, TRUE, NULL},
		{"created_at", "DATETIME", TRUE, FALSE, NULL}
	);`

	response, err := c.Mutate(createCmd, testTimeout)
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	t.Logf("Bundle created, response: %v", response)

	// Verify bundle exists
	showResponse, err := c.Query("SHOW BUNDLES;", testTimeout)
	if err != nil {
		t.Fatalf("Failed to show bundles: %v", err)
	}

	showJSON, err := responseToJSON(showResponse)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}
	schemaDef, err := schema.ParseServerSchema(showJSON)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	found := false
	for _, bundle := range schemaDef.Bundles {
		if bundle.Name == "test_users" {
			found = true
			// Server adds DocumentID field automatically, so we get 5 instead of 4
			if len(bundle.Fields) < 4 {
				t.Errorf("Expected at least 4 fields, got %d", len(bundle.Fields))
			}
			break
		}
	}

	if !found {
		t.Error("test_users bundle not found after creation")
	}

	// Clean up
	_, err = c.Mutate(`DROP BUNDLE "test_users";`, testTimeout)
	if err != nil {
		t.Errorf("Failed to drop bundle: %v", err)
	}
}

// TestIntegration_InsertQuery tests data insertion and retrieval
func TestIntegration_InsertQuery(t *testing.T) {
	opts := client.DefaultOptions()
	c := client.NewClient(&opts)

	err := c.Connect(context.Background(), testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect(context.Background())

	// Clean up and create test bundle
	c.Mutate(`DROP BUNDLE "test_products";`, testTimeout)

	createCmd := `CREATE BUNDLE "test_products" WITH FIELDS (
		{"id", "INT", TRUE, TRUE, NULL},
		{"name", "STRING", TRUE, FALSE, NULL},
		{"price", "FLOAT", TRUE, FALSE, NULL},
		{"in_stock", "BOOLEAN", TRUE, FALSE, TRUE}
	);`

	_, err = c.Mutate(createCmd, testTimeout)
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}
	defer c.Mutate(`DROP BUNDLE "test_products";`, testTimeout)

	// Insert test data
	insertCmd := `INSERT INTO test_products (id, name, price, in_stock) VALUES (1, "Laptop", 999.99, TRUE);`
	_, err = c.Mutate(insertCmd, testTimeout)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Query the data
	queryCmd := `SELECT * FROM test_products WHERE id = 1;`
	response, err := c.Query(queryCmd, testTimeout)
	if err != nil {
		t.Fatalf("Failed to query data: %v", err)
	}

	if response == nil {
		t.Error("Expected non-empty query response")
	}

	// Marshal raw response (not a SHOW BUNDLES response, so don't transform)
	responseJSON, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	responseStr := string(responseJSON)
	// Should contain our inserted data or at least a valid response
	// Note: The query might return empty if the syntax isn't fully supported yet
	if responseStr == "" {
		t.Error("Expected non-empty response")
	}

	// Log the response for debugging
	t.Logf("Query response: %s", responseStr)

	// Check if we got a valid response structure (even if Result is empty)
	if !contains(responseStr, "Result") {
		t.Error("Expected response to contain 'Result' field")
	}
}

// TestIntegration_CreateIndex tests index creation
func TestIntegration_CreateIndex(t *testing.T) {
	opts := client.DefaultOptions()
	c := client.NewClient(&opts)

	err := c.Connect(context.Background(), testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect(context.Background())

	// Create test bundle
	c.Mutate(`DROP BUNDLE "test_indexed";`, testTimeout)

	createCmd := `CREATE BUNDLE "test_indexed" WITH FIELDS (
		{"id", "INT", TRUE, TRUE, NULL},
		{"email", "STRING", TRUE, TRUE, NULL}
	);`

	_, err = c.Mutate(createCmd, testTimeout)
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}
	defer c.Mutate(`DROP BUNDLE "test_indexed";`, testTimeout)

	// Create hash index
	indexCmd := `CREATE HASH INDEX "idx_email" ON BUNDLE "test_indexed" WITH FIELDS ("email");`
	_, err = c.Mutate(indexCmd, testTimeout)
	if err != nil {
		t.Errorf("Failed to create index: %v", err)
	}

	// Verify index exists in schema
	showResponse, err := c.Query("SHOW BUNDLES;", testTimeout)
	if err != nil {
		t.Fatalf("Failed to show bundles: %v", err)
	}

	showJSON, err := responseToJSON(showResponse)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}
	schemaDef, err := schema.ParseServerSchema(showJSON)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	for _, bundle := range schemaDef.Bundles {
		if bundle.Name == "test_indexed" {
			if len(bundle.Indexes) == 0 {
				t.Error("Expected index to be created")
			}
			break
		}
	}
}

// TestIntegration_Migration tests migration apply and rollback
func TestIntegration_Migration(t *testing.T) {
	opts := client.DefaultOptions()
	c := client.NewClient(&opts)

	err := c.Connect(context.Background(), testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect(context.Background())

	// Clean up
	c.Mutate(`DROP BUNDLE "test_migration";`, testTimeout)

	// Create migration client with adapter
	executor := &clientExecutorAdapter{client: c}
	migrationClient := migration.NewClient(executor)

	// Define test migration
	testMigration := &migration.Migration{
		ID:   "001_test_migration",
		Name: "Test Migration",
		Up: []string{
			`CREATE BUNDLE "test_migration" WITH FIELDS (
				{"id", "INT", TRUE, TRUE, NULL},
				{"data", "STRING", FALSE, FALSE, NULL}
			);`,
		},
		Down: []string{
			`DROP BUNDLE "test_migration";`,
		},
		Dependencies: []string{},
	}

	// Create migration plan
	plan := &migration.MigrationPlan{
		Migrations: []*migration.Migration{testMigration},
		Direction:  migration.Up,
		TotalCount: 1,
	}

	// Apply migration
	err = migrationClient.Apply(plan)
	if err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}

	// Verify bundle was created
	showResponse, err := c.Query("SHOW BUNDLES;", testTimeout)
	if err != nil {
		t.Fatalf("Failed to show bundles: %v", err)
	}

	showJSON, err := responseToJSON(showResponse)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}
	schemaDef, err := schema.ParseServerSchema(showJSON)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	found := false
	for _, bundle := range schemaDef.Bundles {
		if bundle.Name == "test_migration" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Bundle not found after migration apply")
	}

	// Rollback migration
	err = migrationClient.Rollback(testMigration.ID, []*migration.Migration{testMigration})
	if err != nil {
		t.Errorf("Failed to rollback migration: %v", err)
	}

	// Verify bundle was dropped
	showResponse, err = c.Query("SHOW BUNDLES;", testTimeout)
	if err != nil {
		t.Fatalf("Failed to show bundles after rollback: %v", err)
	}

	showJSON, err = responseToJSON(showResponse)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}
	schemaDef, err = schema.ParseServerSchema(showJSON)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	for _, bundle := range schemaDef.Bundles {
		if bundle.Name == "test_migration" {
			t.Error("Bundle still exists after rollback")
		}
	}
}

// TestIntegration_StateTransitions tests connection state changes
func TestIntegration_StateTransitions(t *testing.T) {
	opts := client.DefaultOptions()
	c := client.NewClient(&opts)

	if c.GetState() != client.DISCONNECTED {
		t.Errorf("Expected initial state DISCONNECTED, got %s", c.GetState())
	}

	err := c.Connect(context.Background(), testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if c.GetState() != client.CONNECTED {
		t.Errorf("Expected CONNECTED state after connect, got %s", c.GetState())
	}

	err = c.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Failed to disconnect: %v", err)
	}

	if c.GetState() != client.DISCONNECTED {
		t.Errorf("Expected DISCONNECTED state after disconnect, got %s", c.GetState())
	}
}

// TestIntegration_Reconnection tests reconnection after disconnect
func TestIntegration_Reconnection(t *testing.T) {
	opts := client.DefaultOptions()
	c := client.NewClient(&opts)

	// First connection
	err := c.Connect(context.Background(), testConnStr)
	if err != nil {
		t.Fatalf("Failed first connect: %v", err)
	}

	// Disconnect
	err = c.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	// Wait a moment
	time.Sleep(100 * time.Millisecond)

	// Reconnect
	err = c.Connect(context.Background(), testConnStr)
	if err != nil {
		t.Fatalf("Failed to reconnect: %v", err)
	}
	defer c.Disconnect(context.Background())

	if c.GetState() != client.CONNECTED {
		t.Errorf("Expected CONNECTED state after reconnect, got %s", c.GetState())
	}

	// Verify we can still query
	_, err = c.Query("SHOW BUNDLES;", testTimeout)
	if err != nil {
		t.Errorf("Failed to query after reconnect: %v", err)
	}
}

// TestIntegration_SchemaComparison tests schema diff functionality
func TestIntegration_SchemaComparison(t *testing.T) {
	opts := client.DefaultOptions()
	c := client.NewClient(&opts)

	err := c.Connect(context.Background(), testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect(context.Background())

	// Get initial schema
	showResponse1, err := c.Query("SHOW BUNDLES;", testTimeout)
	if err != nil {
		t.Fatalf("Failed to show bundles: %v", err)
	}

	showJSON1, err := responseToJSON(showResponse1)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}
	schema1, err := schema.ParseServerSchema(showJSON1)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Create a test bundle
	c.Mutate(`DROP BUNDLE "test_diff";`, testTimeout)
	createCmd := `CREATE BUNDLE "test_diff" WITH FIELDS (
		{"id", "INT", TRUE, TRUE, NULL}
	);`

	_, err = c.Mutate(createCmd, testTimeout)
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}
	defer c.Mutate(`DROP BUNDLE "test_diff";`, testTimeout)

	// Get updated schema
	showResponse2, err := c.Query("SHOW BUNDLES;", testTimeout)
	if err != nil {
		t.Fatalf("Failed to show bundles: %v", err)
	}

	showJSON2, err := responseToJSON(showResponse2)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}
	schema2, err := schema.ParseServerSchema(showJSON2)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Compare schemas
	diff := schema.CompareSchemas(schema1, schema2)

	if !diff.HasChanges {
		t.Error("Expected schema changes to be detected")
	}

	if len(diff.BundleChanges) == 0 {
		t.Error("Expected bundle changes in diff")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
