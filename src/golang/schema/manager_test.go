package schema

import (
	"testing"
)

func TestParseServerSchema(t *testing.T) {
	schemaJSON := `{
		"bundles": [
			{
				"name": "users",
				"fields": [
					{
						"name": "id",
						"type": "INT",
						"required": true,
						"unique": true,
						"defaultValue": null
					},
					{
						"name": "email",
						"type": "STRING",
						"required": true,
						"unique": true,
						"defaultValue": null
					}
				],
				"indexes": {
					"hash": [
						{
							"name": "idx_email",
							"fields": ["email"]
						}
					],
					"btree": []
				},
				"relationships": []
			}
		]
	}`

	schemaDef, err := ParseServerSchema([]byte(schemaJSON))
	if err != nil {
		t.Fatalf("ParseServerSchema failed: %v", err)
	}

	if len(schemaDef.Bundles) != 1 {
		t.Fatalf("expected 1 bundle, got %d", len(schemaDef.Bundles))
	}

	usersBundle := schemaDef.Bundles[0]
	if usersBundle.Name != "users" {
		t.Errorf("expected name=users, got %s", usersBundle.Name)
	}

	if len(usersBundle.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(usersBundle.Fields))
	}

	// Find email field
	var emailField *FieldDefinition
	for _, f := range usersBundle.Fields {
		if f.Name == "email" {
			emailField = &f
			break
		}
	}

	if emailField == nil {
		t.Fatal("expected email field to exist")
	}

	if emailField.Type != STRING {
		t.Errorf("expected email type=STRING, got %s", emailField.Type)
	}

	if len(usersBundle.Indexes) != 1 {
		t.Fatalf("expected 1 index, got %d", len(usersBundle.Indexes))
	}
}

func TestParseServerSchema_InvalidJSON(t *testing.T) {
	invalidJSON := `{invalid json`

	_, err := ParseServerSchema([]byte(invalidJSON))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestCompareSchemas_NoChanges(t *testing.T) {
	schema1 := &SchemaDefinition{
		Bundles: []BundleDefinition{
			{
				Name: "users",
				Fields: []FieldDefinition{
					{Name: "id", Type: INT, Required: true, Unique: true},
				},
				Indexes:       []IndexDefinition{},
				Relationships: []RelationshipDefinition{},
			},
		},
	}

	schema2 := &SchemaDefinition{
		Bundles: []BundleDefinition{
			{
				Name: "users",
				Fields: []FieldDefinition{
					{Name: "id", Type: INT, Required: true, Unique: true},
				},
				Indexes:       []IndexDefinition{},
				Relationships: []RelationshipDefinition{},
			},
		},
	}

	diff := CompareSchemas(schema1, schema2)

	if diff.HasChanges {
		t.Errorf("expected no changes")
	}
}

func TestCompareSchemas_CreatedBundle(t *testing.T) {
	schema1 := &SchemaDefinition{
		Bundles: []BundleDefinition{},
	}

	schema2 := &SchemaDefinition{
		Bundles: []BundleDefinition{
			{
				Name: "users",
				Fields: []FieldDefinition{
					{Name: "id", Type: INT, Required: true, Unique: true},
				},
				Indexes:       []IndexDefinition{},
				Relationships: []RelationshipDefinition{},
			},
		},
	}

	diff := CompareSchemas(schema1, schema2)

	if !diff.HasChanges {
		t.Error("expected changes to be detected")
	}

	// Just check that we detected changes
	if len(diff.BundleChanges) == 0 {
		t.Error("expected bundle changes to be detected")
	}
}

func TestCompareSchemas_DeletedBundle(t *testing.T) {
	schema1 := &SchemaDefinition{
		Bundles: []BundleDefinition{
			{
				Name: "users",
				Fields: []FieldDefinition{
					{Name: "id", Type: INT, Required: true, Unique: true},
				},
				Indexes:       []IndexDefinition{},
				Relationships: []RelationshipDefinition{},
			},
		},
	}

	schema2 := &SchemaDefinition{
		Bundles: []BundleDefinition{},
	}

	diff := CompareSchemas(schema1, schema2)

	if !diff.HasChanges {
		t.Error("expected changes to be detected")
	}

	// Just check that we detected changes
	if len(diff.BundleChanges) == 0 {
		t.Error("expected bundle changes to be detected")
	}
}

func TestCompareSchemas_ModifiedFields(t *testing.T) {
	schema1 := &SchemaDefinition{
		Bundles: []BundleDefinition{
			{
				Name: "users",
				Fields: []FieldDefinition{
					{Name: "id", Type: INT, Required: true, Unique: true},
				},
				Indexes:       []IndexDefinition{},
				Relationships: []RelationshipDefinition{},
			},
		},
	}

	schema2 := &SchemaDefinition{
		Bundles: []BundleDefinition{
			{
				Name: "users",
				Fields: []FieldDefinition{
					{Name: "id", Type: INT, Required: true, Unique: true},
					{Name: "email", Type: STRING, Required: true, Unique: false},
				},
				Indexes:       []IndexDefinition{},
				Relationships: []RelationshipDefinition{},
			},
		},
	}

	diff := CompareSchemas(schema1, schema2)

	if !diff.HasChanges {
		t.Error("expected changes to be detected")
	}

	// Should have at least one bundle change
	if len(diff.BundleChanges) == 0 {
		t.Error("expected bundle changes to be detected")
	}
}
