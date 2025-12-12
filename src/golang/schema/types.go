package schema

// FieldType represents the type of a bundle field.
type FieldType string

const (
	STRING       FieldType = "STRING"
	INT          FieldType = "INT"
	FLOAT        FieldType = "FLOAT"
	BOOLEAN      FieldType = "BOOLEAN"
	DATETIME     FieldType = "DATETIME"
	JSON         FieldType = "JSON"
	TEXT         FieldType = "TEXT"
	RELATIONSHIP FieldType = "relationship"
)

// IndexType represents the type of index.
type IndexType string

const (
	HASH  IndexType = "hash"
	BTREE IndexType = "btree"
)

// FieldDefinition defines a single field within a bundle.
type FieldDefinition struct {
	Name          string      `json:"name"`
	Type          FieldType   `json:"type"`
	Required      bool        `json:"required"`
	Unique        bool        `json:"unique"`
	DefaultValue  interface{} `json:"defaultValue,omitempty"`
	RelatedBundle string      `json:"relatedBundle,omitempty"` // For relationship fields
}

// IndexDefinition defines an index on a bundle.
type IndexDefinition struct {
	Name   string    `json:"name"`
	Type   IndexType `json:"type"`
	Fields []string  `json:"fields"`
}

// RelationshipDefinition defines a relationship between bundles.
type RelationshipDefinition struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // "1toMany", "ManytoMany", etc.
	SourceBundle string `json:"sourceBundle"`
	SourceField  string `json:"sourceField"`
	DestBundle   string `json:"destBundle"`
	DestField    string `json:"destField"`
}

// BundleDefinition defines the structure of a bundle (table).
type BundleDefinition struct {
	Name          string                   `json:"name"`
	Fields        []FieldDefinition        `json:"fields"`
	Indexes       []IndexDefinition        `json:"indexes"`
	Relationships []RelationshipDefinition `json:"relationships"`
}

// SchemaDefinition represents the complete database schema.
type SchemaDefinition struct {
	Bundles []BundleDefinition `json:"bundles"`
}

// FieldChange represents a change to a field in a bundle.
type FieldChange struct {
	Type      string           `json:"type"` // "add", "remove", "modify"
	FieldName string           `json:"fieldName"`
	OldField  *FieldDefinition `json:"oldField,omitempty"`
	NewField  *FieldDefinition `json:"newField,omitempty"`
}

// IndexChange represents a change to an index.
type IndexChange struct {
	Type     string           `json:"type"` // "add", "remove", "modify"
	OldIndex *IndexDefinition `json:"oldIndex,omitempty"`
	NewIndex *IndexDefinition `json:"newIndex,omitempty"`
}

// BundleChange represents a change to a bundle.
type BundleChange struct {
	Type          string            `json:"type"` // "create", "delete", "modify"
	BundleName    string            `json:"bundleName"`
	OldDefinition *BundleDefinition `json:"oldDefinition,omitempty"`
	NewDefinition *BundleDefinition `json:"newDefinition,omitempty"`
	FieldChanges  []FieldChange     `json:"fieldChanges,omitempty"`
	IndexChanges  []IndexChange     `json:"indexChanges,omitempty"`
}

// RelationshipChange represents a change to a relationship.
type RelationshipChange struct {
	Type            string                  `json:"type"` // "add", "remove"
	BundleName      string                  `json:"bundleName"`
	OldRelationship *RelationshipDefinition `json:"oldRelationship,omitempty"`
	NewRelationship *RelationshipDefinition `json:"newRelationship,omitempty"`
}

// SchemaDiff represents the differences between two schemas.
type SchemaDiff struct {
	BundleChanges       []BundleChange       `json:"bundleChanges"`
	IndexChanges        []IndexChange        `json:"indexChanges"`
	RelationshipChanges []RelationshipChange `json:"relationshipChanges"`
	HasChanges          bool                 `json:"hasChanges"`
}
