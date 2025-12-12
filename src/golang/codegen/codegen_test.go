package codegen

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

func TestJSONSchemaGenerator_GenerateSingle(t *testing.T) {
	gen := NewJSONSchemaGenerator()

	schemaDef := &schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name: "users",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: schema.INT, Required: true, Unique: true},
					{Name: "email", Type: schema.STRING, Required: true},
				},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}

	result, err := gen.GenerateSingle(schemaDef)
	if err != nil {
		t.Fatalf("GenerateSingle failed: %v", err)
	}

	var parsed map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(result), &parsed); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}

	if parsed["type"] != "object" {
		t.Errorf("expected type=object, got %v", parsed["type"])
	}

	definitions := parsed["definitions"].(map[string]interface{})
	if _, exists := definitions["users"]; !exists {
		t.Error("expected users definition to exist")
	}
}

func TestJSONSchemaGenerator_GenerateMulti(t *testing.T) {
	gen := NewJSONSchemaGenerator()

	schemaDef := &schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name:          "users",
				Fields:        []schema.FieldDefinition{{Name: "id", Type: schema.INT}},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
			{
				Name:          "posts",
				Fields:        []schema.FieldDefinition{{Name: "id", Type: schema.INT}},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}

	result, err := gen.GenerateMulti(schemaDef)
	if err != nil {
		t.Fatalf("GenerateMulti failed: %v", err)
	}

	// Should have both bundles as separate files
	if len(result) != 2 {
		t.Errorf("expected 2 files, got %d", len(result))
	}

	if _, exists := result["users"]; !exists {
		t.Error("expected users schema to exist")
	}

	if _, exists := result["posts"]; !exists {
		t.Error("expected posts schema to exist")
	}
}

func TestGraphQLSchemaGenerator_Generate(t *testing.T) {
	gen := NewGraphQLSchemaGenerator()

	schemaDef := &schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name: "users",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: schema.INT, Required: true},
					{Name: "email", Type: schema.STRING, Required: true},
				},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}

	result, err := gen.Generate(schemaDef)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should contain type definition
	if !strings.Contains(result, "type") {
		t.Error("expected type definition")
	}

	// Should contain Query
	if !strings.Contains(result, "Query") {
		t.Error("expected Query type")
	}

	// Should contain Mutation
	if !strings.Contains(result, "Mutation") {
		t.Error("expected Mutation type")
	}
}
