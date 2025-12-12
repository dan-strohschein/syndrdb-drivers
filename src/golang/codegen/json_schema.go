package codegen

import (
	"encoding/json"
	"fmt"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

// JSONSchemaGenerator generates JSON Schema from SyndrDB schema definitions.
type JSONSchemaGenerator struct {
	registry *TypeRegistry
}

// NewJSONSchemaGenerator creates a new JSON Schema generator.
func NewJSONSchemaGenerator() *JSONSchemaGenerator {
	return &JSONSchemaGenerator{
		registry: NewTypeRegistry(),
	}
}

// GenerateSingle generates a single JSON Schema file containing all bundle definitions.
func (g *JSONSchemaGenerator) GenerateSingle(schemaDef *schema.SchemaDefinition) (string, error) {
	definitions := make(map[string]interface{})

	for _, bundle := range schemaDef.Bundles {
		bundleSchema := g.generateBundleSchema(&bundle)
		definitions[bundle.Name] = bundleSchema
	}

	rootSchema := map[string]interface{}{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"title":       "SyndrDB Schema",
		"type":        "object",
		"definitions": definitions,
	}

	data, err := json.MarshalIndent(rootSchema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON schema: %w", err)
	}

	return string(data), nil
}

// GenerateMulti generates separate JSON Schema files for each bundle.
// Returns a map of bundle name to JSON Schema content.
func (g *JSONSchemaGenerator) GenerateMulti(schemaDef *schema.SchemaDefinition) (map[string]string, error) {
	schemas := make(map[string]string)

	for _, bundle := range schemaDef.Bundles {
		bundleSchema := g.generateBundleSchema(&bundle)

		rootSchema := map[string]interface{}{
			"$schema":     "http://json-schema.org/draft-07/schema#",
			"title":       bundle.Name,
			"type":        "object",
			"description": fmt.Sprintf("Schema for %s bundle", bundle.Name),
			"properties":  bundleSchema["properties"],
			"required":    bundleSchema["required"],
		}

		data, err := json.MarshalIndent(rootSchema, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schema for bundle %s: %w", bundle.Name, err)
		}

		schemas[bundle.Name] = string(data)
	}

	return schemas, nil
}

// generateBundleSchema creates a JSON Schema object for a single bundle.
func (g *JSONSchemaGenerator) generateBundleSchema(bundle *schema.BundleDefinition) map[string]interface{} {
	properties := make(map[string]interface{})
	required := make([]string, 0)

	for _, field := range bundle.Fields {
		properties[field.Name] = g.generateFieldSchema(&field)

		if field.Required {
			required = append(required, field.Name)
		}
	}

	bundleSchema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		bundleSchema["required"] = required
	}

	return bundleSchema
}

// generateFieldSchema creates a JSON Schema type definition for a field.
func (g *JSONSchemaGenerator) generateFieldSchema(field *schema.FieldDefinition) map[string]interface{} {
	fieldSchema := make(map[string]interface{})

	// Map SyndrDB types to JSON Schema types
	switch field.Type {
	case schema.STRING, schema.TEXT:
		fieldSchema["type"] = "string"
	case schema.INT:
		fieldSchema["type"] = "integer"
	case schema.FLOAT:
		fieldSchema["type"] = "number"
	case schema.BOOLEAN:
		fieldSchema["type"] = "boolean"
	case schema.DATETIME:
		fieldSchema["type"] = "string"
		fieldSchema["format"] = "date-time"
	case schema.JSON:
		fieldSchema["type"] = "object"
	case schema.RELATIONSHIP:
		// For relationships, reference the related bundle
		if field.RelatedBundle != "" {
			fieldSchema["$ref"] = fmt.Sprintf("#/definitions/%s", field.RelatedBundle)
		} else {
			fieldSchema["type"] = "object"
		}
	default:
		fieldSchema["type"] = "string"
	}

	// Add default value if present
	if field.DefaultValue != nil {
		fieldSchema["default"] = field.DefaultValue
	}

	return fieldSchema
}

// GetTypeRegistry returns the type registry used by this generator.
func (g *JSONSchemaGenerator) GetTypeRegistry() *TypeRegistry {
	return g.registry
}
