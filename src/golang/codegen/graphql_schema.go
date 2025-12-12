package codegen

import (
	"fmt"
	"strings"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

// GraphQLSchemaGenerator generates GraphQL SDL from SyndrDB schema definitions.
type GraphQLSchemaGenerator struct {
	registry *TypeRegistry
}

// NewGraphQLSchemaGenerator creates a new GraphQL schema generator.
func NewGraphQLSchemaGenerator() *GraphQLSchemaGenerator {
	return &GraphQLSchemaGenerator{
		registry: NewTypeRegistry(),
	}
}

// Generate creates a complete GraphQL SDL schema.
func (g *GraphQLSchemaGenerator) Generate(schemaDef *schema.SchemaDefinition) (string, error) {
	var builder strings.Builder

	// Write schema header
	builder.WriteString("# Generated GraphQL Schema for SyndrDB\n\n")

	// Generate type definitions for each bundle
	for _, bundle := range schemaDef.Bundles {
		g.generateType(&builder, &bundle)
		builder.WriteString("\n")
	}

	// Generate input types for mutations
	for _, bundle := range schemaDef.Bundles {
		g.generateInputType(&builder, &bundle)
		builder.WriteString("\n")
	}

	// Generate root Query type
	g.generateQueryType(&builder, schemaDef)
	builder.WriteString("\n")

	// Generate root Mutation type
	g.generateMutationType(&builder, schemaDef)
	builder.WriteString("\n")

	// Generate Subscription type (placeholder)
	g.generateSubscriptionType(&builder, schemaDef)

	return builder.String(), nil
}

// generateType creates a GraphQL type definition for a bundle.
func (g *GraphQLSchemaGenerator) generateType(builder *strings.Builder, bundle *schema.BundleDefinition) {
	builder.WriteString(fmt.Sprintf("type %s {\n", bundle.Name))

	// Add ID field (standard for GraphQL)
	builder.WriteString("  id: ID!\n")

	// Add fields
	for _, field := range bundle.Fields {
		if field.Type == schema.RELATIONSHIP {
			// Handle relationships
			g.generateRelationshipField(builder, &field)
		} else {
			// Regular fields
			graphqlType := g.mapToGraphQLType(field.Type)
			required := ""
			if field.Required {
				required = "!"
			}
			builder.WriteString(fmt.Sprintf("  %s: %s%s\n", field.Name, graphqlType, required))
		}
	}

	builder.WriteString("}\n")
}

// generateRelationshipField creates a field definition for relationships.
func (g *GraphQLSchemaGenerator) generateRelationshipField(builder *strings.Builder, field *schema.FieldDefinition) {
	if field.RelatedBundle == "" {
		return
	}

	// Determine if it's a list relationship or single reference
	// For now, assume single reference (can be enhanced based on relationship type)
	builder.WriteString(fmt.Sprintf("  %s: %s\n", field.Name, field.RelatedBundle))
}

// generateInputType creates a GraphQL input type for mutations.
func (g *GraphQLSchemaGenerator) generateInputType(builder *strings.Builder, bundle *schema.BundleDefinition) {
	// Create input for bundle
	builder.WriteString(fmt.Sprintf("input %sInput {\n", bundle.Name))

	for _, field := range bundle.Fields {
		// Skip relationship fields in input types
		if field.Type == schema.RELATIONSHIP {
			continue
		}

		graphqlType := g.mapToGraphQLType(field.Type)
		required := ""
		if field.Required && field.DefaultValue == nil {
			required = "!"
		}
		builder.WriteString(fmt.Sprintf("  %s: %s%s\n", field.Name, graphqlType, required))
	}

	builder.WriteString("}\n")

	// Create update input (all fields optional)
	builder.WriteString(fmt.Sprintf("input %sUpdateInput {\n", bundle.Name))

	for _, field := range bundle.Fields {
		// Skip relationship fields
		if field.Type == schema.RELATIONSHIP {
			continue
		}

		graphqlType := g.mapToGraphQLType(field.Type)
		builder.WriteString(fmt.Sprintf("  %s: %s\n", field.Name, graphqlType))
	}

	builder.WriteString("}\n")
}

// generateQueryType creates the root Query type.
func (g *GraphQLSchemaGenerator) generateQueryType(builder *strings.Builder, schemaDef *schema.SchemaDefinition) {
	builder.WriteString("type Query {\n")

	for _, bundle := range schemaDef.Bundles {
		// Single item query
		builder.WriteString(fmt.Sprintf("  %s(id: ID!): %s\n",
			g.toLowerFirst(bundle.Name), bundle.Name))

		// List query
		builder.WriteString(fmt.Sprintf("  %s(limit: Int, offset: Int): [%s!]!\n",
			g.toPlural(g.toLowerFirst(bundle.Name)), bundle.Name))
	}

	builder.WriteString("}\n")
}

// generateMutationType creates the root Mutation type.
func (g *GraphQLSchemaGenerator) generateMutationType(builder *strings.Builder, schemaDef *schema.SchemaDefinition) {
	builder.WriteString("type Mutation {\n")

	for _, bundle := range schemaDef.Bundles {
		// Create mutation
		builder.WriteString(fmt.Sprintf("  create%s(input: %sInput!): %s!\n",
			bundle.Name, bundle.Name, bundle.Name))

		// Update mutation
		builder.WriteString(fmt.Sprintf("  update%s(id: ID!, input: %sUpdateInput!): %s!\n",
			bundle.Name, bundle.Name, bundle.Name))

		// Delete mutation
		builder.WriteString(fmt.Sprintf("  delete%s(id: ID!): Boolean!\n",
			bundle.Name))
	}

	builder.WriteString("}\n")
}

// generateSubscriptionType creates the root Subscription type (placeholder).
func (g *GraphQLSchemaGenerator) generateSubscriptionType(builder *strings.Builder, schemaDef *schema.SchemaDefinition) {
	builder.WriteString("type Subscription {\n")

	for _, bundle := range schemaDef.Bundles {
		// Subscription for changes to a specific bundle
		builder.WriteString(fmt.Sprintf("  %sChanged(id: ID): %s\n",
			g.toLowerFirst(bundle.Name), bundle.Name))
	}

	builder.WriteString("}\n")
}

// mapToGraphQLType maps SyndrDB types to GraphQL types.
func (g *GraphQLSchemaGenerator) mapToGraphQLType(fieldType schema.FieldType) string {
	switch fieldType {
	case schema.STRING, schema.TEXT, schema.DATETIME:
		return "String"
	case schema.INT:
		return "Int"
	case schema.FLOAT:
		return "Float"
	case schema.BOOLEAN:
		return "Boolean"
	case schema.JSON:
		return "JSON" // Assumes JSON scalar is defined
	default:
		return "String"
	}
}

// toLowerFirst converts the first character to lowercase.
func (g *GraphQLSchemaGenerator) toLowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// toPlural adds 's' to make a simple plural (can be enhanced).
func (g *GraphQLSchemaGenerator) toPlural(s string) string {
	if strings.HasSuffix(s, "s") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}

// GetTypeRegistry returns the type registry used by this generator.
func (g *GraphQLSchemaGenerator) GetTypeRegistry() *TypeRegistry {
	return g.registry
}
