package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/codegen"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

const version = "1.0.0"

func main() {
	// Subcommands
	fetchCmd := flag.NewFlagSet("fetch", flag.ExitOnError)
	typescriptCmd := flag.NewFlagSet("typescript", flag.ExitOnError)
	jsonSchemaCmd := flag.NewFlagSet("json-schema", flag.ExitOnError)
	graphqlCmd := flag.NewFlagSet("graphql", flag.ExitOnError)

	// Fetch flags
	fetchConn := fetchCmd.String("conn", "", "Connection string (required)")
	fetchOutput := fetchCmd.String("output", "./schema.json", "Output path for schema file")

	// TypeScript flags
	tsSchema := typescriptCmd.String("schema", "./schema.json", "Path to schema file")
	tsOutput := typescriptCmd.String("output", "./types", "Output directory")

	// JSON Schema flags
	jsonSchema := jsonSchemaCmd.String("schema", "./schema.json", "Path to schema file")
	jsonOutput := jsonSchemaCmd.String("output", "./schemas", "Output directory")
	jsonMode := jsonSchemaCmd.String("mode", "multi", "Generation mode: single or multi")

	// GraphQL flags
	gqlSchema := graphqlCmd.String("schema", "./schema.json", "Path to schema file")
	gqlOutput := graphqlCmd.String("output", "./schema.graphql", "Output file")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "fetch":
		fetchCmd.Parse(os.Args[2:])
		handleFetch(*fetchConn, *fetchOutput)
	case "typescript":
		typescriptCmd.Parse(os.Args[2:])
		handleTypeScript(*tsSchema, *tsOutput)
	case "json-schema":
		jsonSchemaCmd.Parse(os.Args[2:])
		handleJSONSchema(*jsonSchema, *jsonOutput, *jsonMode)
	case "graphql":
		graphqlCmd.Parse(os.Args[2:])
		handleGraphQL(*gqlSchema, *gqlOutput)
	case "version":
		fmt.Printf("schema-codegen v%s\n", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("SyndrDB Schema Code Generator")
	fmt.Println("\nUsage:")
	fmt.Println("  schema-codegen fetch --conn <connection-string> --output <path>")
	fmt.Println("  schema-codegen typescript --schema <path> --output <dir>")
	fmt.Println("  schema-codegen json-schema --schema <path> --output <dir> --mode <single|multi>")
	fmt.Println("  schema-codegen graphql --schema <path> --output <file>")
	fmt.Println("  schema-codegen version")
}

func handleFetch(connString, output string) {
	if connString == "" {
		fmt.Fprintf(os.Stderr, "Error: --conn is required\n")
		os.Exit(1)
	}

	// Connect to database
	c := client.NewClient(&client.ClientOptions{
		DefaultTimeoutMs: 10000,
		DebugMode:        false,
		MaxRetries:       3,
	})

	fmt.Println("Connecting to database...")
	if err := c.Connect(connString); err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		os.Exit(1)
	}
	defer c.Disconnect()
	fmt.Println("✓ Connected")

	// Fetch schema
	fmt.Println("Fetching schema...")
	result, err := c.Query("SHOW BUNDLES;", 10000)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}

	// Parse and convert result to SchemaDefinition
	// This is a simplified version - in reality you'd parse the SHOW BUNDLES output
	schemaDef := parseServerSchema(result)

	// Write to file
	dir := filepath.Dir(output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(schemaDef, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling schema: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Schema saved to %s\n", output)
	fmt.Printf("  Bundles: %d\n", len(schemaDef.Bundles))
}

func handleTypeScript(schemaPath, outputDir string) {
	schemaDef := loadSchema(schemaPath)

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generating TypeScript types...")

	// Generate types for each bundle
	for _, bundle := range schemaDef.Bundles {
		ts := generateTypeScript(&bundle)
		filename := filepath.Join(outputDir, bundle.Name+".ts")

		if err := os.WriteFile(filename, []byte(ts), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", filename, err)
			os.Exit(1)
		}

		fmt.Printf("  ✓ %s.ts\n", bundle.Name)
	}

	// Generate index file
	indexTS := generateTypeScriptIndex(schemaDef)
	indexPath := filepath.Join(outputDir, "index.ts")
	if err := os.WriteFile(indexPath, []byte(indexTS), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing index.ts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Generated %d TypeScript files in %s\n", len(schemaDef.Bundles)+1, outputDir)
}

func handleJSONSchema(schemaPath, outputDir, mode string) {
	schemaDef := loadSchema(schemaPath)

	generator := codegen.NewJSONSchemaGenerator()

	fmt.Printf("Generating JSON Schema (%s mode)...\n", mode)

	if mode == "single" {
		// Single file with all schemas
		result, err := generator.GenerateSingle(schemaDef)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Generation failed: %v\n", err)
			os.Exit(1)
		}

		outputPath := filepath.Join(outputDir, "schema.json")
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Generated schema.json\n")
	} else {
		// Multiple files, one per bundle
		result, err := generator.GenerateMulti(schemaDef)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Generation failed: %v\n", err)
			os.Exit(1)
		}

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			os.Exit(1)
		}

		for filename, content := range result {
			path := filepath.Join(outputDir, filename)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", filename, err)
				os.Exit(1)
			}
			fmt.Printf("  ✓ %s\n", filename)
		}
	}

	fmt.Printf("\n✓ JSON Schema generated in %s\n", outputDir)
}

func handleGraphQL(schemaPath, output string) {
	schemaDef := loadSchema(schemaPath)

	generator := codegen.NewGraphQLSchemaGenerator()

	fmt.Println("Generating GraphQL schema...")
	result, err := generator.Generate(schemaDef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Generation failed: %v\n", err)
		os.Exit(1)
	}

	dir := filepath.Dir(output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(output, []byte(result), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ GraphQL schema saved to %s\n", output)
}

func loadSchema(path string) *schema.SchemaDefinition {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading schema: %v\n", err)
		os.Exit(1)
	}

	var schemaDef schema.SchemaDefinition
	if err := json.Unmarshal(data, &schemaDef); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing schema: %v\n", err)
		os.Exit(1)
	}

	return &schemaDef
}

func parseServerSchema(result interface{}) *schema.SchemaDefinition {
	// Simplified parser - in reality you'd parse the actual SHOW BUNDLES response
	return &schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name: "example",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: "int", Required: true, Unique: true},
					{Name: "name", Type: "string", Required: true},
				},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}
}

func generateTypeScript(bundle *schema.BundleDefinition) string {
	var sb strings.Builder

	// Interface
	sb.WriteString(fmt.Sprintf("/**\n * %s bundle\n */\n", bundle.Name))
	sb.WriteString(fmt.Sprintf("export interface %s {\n", capitalize(bundle.Name)))

	for _, field := range bundle.Fields {
		tsType := mapToTypeScript(field.Type)
		optional := ""
		if !field.Required {
			optional = "?"
		}
		sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", field.Name, optional, tsType))
	}

	sb.WriteString("}\n\n")

	// Input type
	sb.WriteString(fmt.Sprintf("export interface %sInput {\n", capitalize(bundle.Name)))
	for _, field := range bundle.Fields {
		tsType := mapToTypeScript(field.Type)
		optional := "?"
		if field.Required {
			optional = ""
		}
		sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", field.Name, optional, tsType))
	}
	sb.WriteString("}\n")

	return sb.String()
}

func generateTypeScriptIndex(schemaDef *schema.SchemaDefinition) string {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by schema-codegen\n\n")

	for _, bundle := range schemaDef.Bundles {
		sb.WriteString(fmt.Sprintf("export * from './%s';\n", bundle.Name))
	}

	return sb.String()
}

func mapToTypeScript(syndrType string) string {
	switch syndrType {
	case "int", "float", "double":
		return "number"
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "bytes":
		return "Uint8Array"
	case "timestamp":
		return "Date"
	default:
		return "any"
	}
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
