package schema

import (
	"encoding/json"
	"fmt"
)

// ParseServerSchema parses the response from SHOW BUNDLES command.
// The response format is: {"bundles": [{...}, {...}]}
func ParseServerSchema(response []byte) (*SchemaDefinition, error) {
	var rawResponse struct {
		Bundles []struct {
			Name   string `json:"name"`
			Fields []struct {
				Name         string      `json:"name"`
				Type         string      `json:"type"`
				Required     bool        `json:"required"`
				Unique       bool        `json:"unique"`
				DefaultValue interface{} `json:"defaultValue"`
			} `json:"fields"`
			Indexes struct {
				Hash []struct {
					Name   string   `json:"name"`
					Fields []string `json:"fields"`
				} `json:"hash"`
				Btree []struct {
					Name   string   `json:"name"`
					Fields []string `json:"fields"`
				} `json:"btree"`
			} `json:"indexes"`
			Relationships []struct {
				Name         string `json:"name"`
				Type         string `json:"type"`
				SourceBundle string `json:"sourceBundle"`
				SourceField  string `json:"sourceField"`
				DestBundle   string `json:"destBundle"`
				DestField    string `json:"destField"`
			} `json:"relationships"`
		} `json:"bundles"`
	}

	if err := json.Unmarshal(response, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to parse server schema: %w", err)
	}

	schema := &SchemaDefinition{
		Bundles: make([]BundleDefinition, 0, len(rawResponse.Bundles)),
	}

	for _, rawBundle := range rawResponse.Bundles {
		bundle := BundleDefinition{
			Name:          rawBundle.Name,
			Fields:        make([]FieldDefinition, 0, len(rawBundle.Fields)),
			Indexes:       make([]IndexDefinition, 0),
			Relationships: make([]RelationshipDefinition, 0, len(rawBundle.Relationships)),
		}

		// Parse fields
		for _, rawField := range rawBundle.Fields {
			field := FieldDefinition{
				Name:         rawField.Name,
				Type:         FieldType(rawField.Type),
				Required:     rawField.Required,
				Unique:       rawField.Unique,
				DefaultValue: rawField.DefaultValue,
			}
			bundle.Fields = append(bundle.Fields, field)
		}

		// Parse hash indexes
		for _, rawIndex := range rawBundle.Indexes.Hash {
			index := IndexDefinition{
				Name:   rawIndex.Name,
				Type:   HASH,
				Fields: rawIndex.Fields,
			}
			bundle.Indexes = append(bundle.Indexes, index)
		}

		// Parse btree indexes
		for _, rawIndex := range rawBundle.Indexes.Btree {
			index := IndexDefinition{
				Name:   rawIndex.Name,
				Type:   BTREE,
				Fields: rawIndex.Fields,
			}
			bundle.Indexes = append(bundle.Indexes, index)
		}

		// Parse relationships
		for _, rawRel := range rawBundle.Relationships {
			rel := RelationshipDefinition{
				Name:         rawRel.Name,
				Type:         rawRel.Type,
				SourceBundle: rawRel.SourceBundle,
				SourceField:  rawRel.SourceField,
				DestBundle:   rawRel.DestBundle,
				DestField:    rawRel.DestField,
			}
			bundle.Relationships = append(bundle.Relationships, rel)
		}

		schema.Bundles = append(schema.Bundles, bundle)
	}

	return schema, nil
}

// CompareSchemas compares local and server schemas to generate a diff.
// This matches the Node.js implementation in SchemaManager.ts lines 85-200.
func CompareSchemas(local, server *SchemaDefinition) *SchemaDiff {
	diff := &SchemaDiff{
		BundleChanges:       make([]BundleChange, 0),
		IndexChanges:        make([]IndexChange, 0),
		RelationshipChanges: make([]RelationshipChange, 0),
		HasChanges:          false,
	}

	// Create maps for efficient lookup
	localBundles := make(map[string]*BundleDefinition)
	serverBundles := make(map[string]*BundleDefinition)

	for i := range local.Bundles {
		localBundles[local.Bundles[i].Name] = &local.Bundles[i]
	}
	for i := range server.Bundles {
		serverBundles[server.Bundles[i].Name] = &server.Bundles[i]
	}

	// Find added and modified bundles
	for name, localBundle := range localBundles {
		serverBundle, exists := serverBundles[name]
		if !exists {
			// Bundle created
			diff.BundleChanges = append(diff.BundleChanges, BundleChange{
				Type:          "create",
				BundleName:    name,
				NewDefinition: localBundle,
			})
			diff.HasChanges = true
		} else {
			// Check for modifications
			fieldChanges := compareFields(localBundle.Fields, serverBundle.Fields)
			indexChanges := compareIndexes(localBundle.Indexes, serverBundle.Indexes)

			if len(fieldChanges) > 0 || len(indexChanges) > 0 {
				diff.BundleChanges = append(diff.BundleChanges, BundleChange{
					Type:          "modify",
					BundleName:    name,
					OldDefinition: serverBundle,
					NewDefinition: localBundle,
					FieldChanges:  fieldChanges,
					IndexChanges:  indexChanges,
				})
				diff.HasChanges = true
			}
		}
	}

	// Find removed bundles
	for name, serverBundle := range serverBundles {
		if _, exists := localBundles[name]; !exists {
			diff.BundleChanges = append(diff.BundleChanges, BundleChange{
				Type:          "delete",
				BundleName:    name,
				OldDefinition: serverBundle,
			})
			diff.HasChanges = true
		}
	}

	// Compare relationships (only include valid cross-bundle relationships)
	diff.RelationshipChanges = compareRelationships(local, server, localBundles, serverBundles)
	if len(diff.RelationshipChanges) > 0 {
		diff.HasChanges = true
	}

	return diff
}

// compareFields compares two field lists and returns the changes.
func compareFields(localFields, serverFields []FieldDefinition) []FieldChange {
	changes := make([]FieldChange, 0)

	localMap := make(map[string]*FieldDefinition)
	serverMap := make(map[string]*FieldDefinition)

	for i := range localFields {
		localMap[localFields[i].Name] = &localFields[i]
	}
	for i := range serverFields {
		serverMap[serverFields[i].Name] = &serverFields[i]
	}

	// Find added and modified fields
	for name, localField := range localMap {
		serverField, exists := serverMap[name]
		if !exists {
			changes = append(changes, FieldChange{
				Type:      "add",
				FieldName: name,
				NewField:  localField,
			})
		} else if !fieldsEqual(localField, serverField) {
			changes = append(changes, FieldChange{
				Type:      "modify",
				FieldName: name,
				OldField:  serverField,
				NewField:  localField,
			})
		}
	}

	// Find removed fields
	for name, serverField := range serverMap {
		if _, exists := localMap[name]; !exists {
			changes = append(changes, FieldChange{
				Type:      "remove",
				FieldName: name,
				OldField:  serverField,
			})
		}
	}

	return changes
}

// fieldsEqual compares two fields for equality.
func fieldsEqual(a, b *FieldDefinition) bool {
	return a.Type == b.Type &&
		a.Required == b.Required &&
		a.Unique == b.Unique &&
		fmt.Sprintf("%v", a.DefaultValue) == fmt.Sprintf("%v", b.DefaultValue) &&
		a.RelatedBundle == b.RelatedBundle
}

// compareIndexes compares two index lists and returns the changes.
func compareIndexes(localIndexes, serverIndexes []IndexDefinition) []IndexChange {
	changes := make([]IndexChange, 0)

	localMap := make(map[string]*IndexDefinition)
	serverMap := make(map[string]*IndexDefinition)

	for i := range localIndexes {
		localMap[localIndexes[i].Name] = &localIndexes[i]
	}
	for i := range serverIndexes {
		serverMap[serverIndexes[i].Name] = &serverIndexes[i]
	}

	// Find added and modified indexes
	for name, localIndex := range localMap {
		serverIndex, exists := serverMap[name]
		if !exists {
			changes = append(changes, IndexChange{
				Type:     "add",
				NewIndex: localIndex,
			})
		} else if !indexesEqual(localIndex, serverIndex) {
			changes = append(changes, IndexChange{
				Type:     "modify",
				OldIndex: serverIndex,
				NewIndex: localIndex,
			})
		}
	}

	// Find removed indexes
	for name, serverIndex := range serverMap {
		if _, exists := localMap[name]; !exists {
			changes = append(changes, IndexChange{
				Type:     "remove",
				OldIndex: serverIndex,
			})
		}
	}

	return changes
}

// indexesEqual compares two indexes for equality.
func indexesEqual(a, b *IndexDefinition) bool {
	if a.Type != b.Type || len(a.Fields) != len(b.Fields) {
		return false
	}
	for i := range a.Fields {
		if a.Fields[i] != b.Fields[i] {
			return false
		}
	}
	return true
}

// compareRelationships compares relationships between schemas.
// Only includes relationships where both source and destination bundles exist.
func compareRelationships(local, server *SchemaDefinition, localBundles, serverBundles map[string]*BundleDefinition) []RelationshipChange {
	changes := make([]RelationshipChange, 0)

	// Collect all relationships from both schemas
	localRels := make(map[string]*RelationshipDefinition)
	serverRels := make(map[string]*RelationshipDefinition)

	for _, bundle := range local.Bundles {
		for i := range bundle.Relationships {
			rel := &bundle.Relationships[i]
			// Only include if both bundles exist
			if _, srcExists := localBundles[rel.SourceBundle]; srcExists {
				if _, destExists := localBundles[rel.DestBundle]; destExists {
					key := fmt.Sprintf("%s.%s", bundle.Name, rel.Name)
					localRels[key] = rel
				}
			}
		}
	}

	for _, bundle := range server.Bundles {
		for i := range bundle.Relationships {
			rel := &bundle.Relationships[i]
			// Only include if both bundles exist
			if _, srcExists := serverBundles[rel.SourceBundle]; srcExists {
				if _, destExists := serverBundles[rel.DestBundle]; destExists {
					key := fmt.Sprintf("%s.%s", bundle.Name, rel.Name)
					serverRels[key] = rel
				}
			}
		}
	}

	// Find added relationships
	for key, localRel := range localRels {
		if _, exists := serverRels[key]; !exists {
			changes = append(changes, RelationshipChange{
				Type:            "add",
				BundleName:      localRel.SourceBundle,
				NewRelationship: localRel,
			})
		}
	}

	// Find removed relationships
	for key, serverRel := range serverRels {
		if _, exists := localRels[key]; !exists {
			changes = append(changes, RelationshipChange{
				Type:            "remove",
				BundleName:      serverRel.SourceBundle,
				OldRelationship: serverRel,
			})
		}
	}

	return changes
}
