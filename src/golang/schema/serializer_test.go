package schema

import (
	"strings"
	"testing"
)

func TestSerializeCreateBundle(t *testing.T) {
	bundle := &BundleDefinition{
		Name: "users",
		Fields: []FieldDefinition{
			{
				Name:     "id",
				Type:     INT,
				Required: true,
				Unique:   true,
			},
		},
		Indexes:       []IndexDefinition{},
		Relationships: []RelationshipDefinition{},
	}

	cmd := SerializeCreateBundle(bundle)

	if !strings.Contains(cmd, "CREATE BUNDLE") {
		t.Errorf("expected CREATE BUNDLE in command, got: %s", cmd)
	}

	if !strings.Contains(cmd, "users") {
		t.Errorf("expected users in command, got: %s", cmd)
	}
}

func TestSerializeDeleteBundle(t *testing.T) {
	cmd := SerializeDeleteBundle("users")

	expected := `DROP BUNDLE "users";`
	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestSerializeCreateIndex_Hash(t *testing.T) {
	index := &IndexDefinition{
		Name:   "idx_email",
		Type:   HASH,
		Fields: []string{"email"},
	}

	cmd := SerializeCreateIndex(index, "users")

	if !strings.Contains(cmd, "CREATE HASH INDEX") {
		t.Errorf("expected CREATE HASH INDEX, got: %s", cmd)
	}

	if !strings.Contains(cmd, "idx_email") {
		t.Errorf("expected idx_email, got: %s", cmd)
	}
}

func TestSerializeCreateIndex_BTree(t *testing.T) {
	index := &IndexDefinition{
		Name:   "idx_name",
		Type:   BTREE,
		Fields: []string{"name"},
	}

	cmd := SerializeCreateIndex(index, "users")

	if !strings.Contains(cmd, "CREATE B-INDEX") {
		t.Errorf("expected CREATE B-INDEX, got: %s", cmd)
	}
}

func TestSerializeDropIndex(t *testing.T) {
	cmd := SerializeDropIndex("idx_email")

	expected := `DROP INDEX "idx_email";`
	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}
