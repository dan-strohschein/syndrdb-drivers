//go:build milestone2

package main

import (
	"context"
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

func handleCodegen(args []string) {
	if len(args) == 0 {
		printCodegenUsage()
		os.Exit(1)
	}

	subcommand := args[0]

	switch subcommand {
	case "fetch-schema":
		handleCodegenFetch(args[1:])
	case "generate":
		handleCodegenGenerate(args[1:])
	case "help", "-h", "--help":
		printCodegenUsage()
	default:
		printError(fmt.Sprintf("Unknown codegen subcommand: %s", subcommand))
		printCodegenUsage()
		os.Exit(1)
	}
}

func printCodegenUsage() {
	printHeader("Code Generation Commands")
	fmt.Println("Usage:")
	fmt.Println("  syndrdb codegen " + colorYellow("<command>") + " [options]\n")
	fmt.Println("Commands:")
	fmt.Println("  " + colorGreen("fetch-schema") + "  Fetch schema from database server")
	fmt.Println("  " + colorGreen("generate") + "     Generate code from schema")
	fmt.Println("\nExamples:")
	fmt.Println("  " + colorDim("# Fetch schema from server"))
	fmt.Println("  syndrdb codegen fetch-schema --output ./schema.json")
	fmt.Println()
	fmt.Println("  " + colorDim("# Generate TypeScript types"))
	fmt.Println("  syndrdb codegen generate --format types --language typescript")
	fmt.Println()
	fmt.Println("  " + colorDim("# Generate JSON Schema"))
	fmt.Println("  syndrdb codegen generate --format json-schema")
	fmt.Println()
	fmt.Println("  " + colorDim("# Generate GraphQL Schema"))
	fmt.Println("  syndrdb codegen generate --format graphql")
}

// handleCodegenFetch fetches schema from the server
func handleCodegenFetch(args []string) {
	fs := flag.NewFlagSet("codegen fetch-schema", flag.ExitOnError)
	connStr := fs.String("conn", os.Getenv("SYNDRDB_CONN"), "Connection string")
	output := fs.String("output", getDefaultSchemaFile(), "Output file path")
	format := fs.String("format", "json", "Output format (json, yaml)")
	fs.Parse(args)

	if *connStr == "" {
		printError("Connection string is required")
		fmt.Println("\nProvide via --conn flag or SYNDRDB_CONN environment variable")
		os.Exit(1)
	}

	printHeader("Fetch Schema from Server")

	// Connect to database
	printStep(1, 3, "Connecting to database...")
	opts := &client.ClientOptions{}
	c := client.NewClient(opts)
	ctx := context.Background()
	if err := c.Connect(ctx, *connStr); err != nil {
		printError(fmt.Sprintf("Failed to connect: %v", err))
		os.Exit(1)
	}
	defer c.Disconnect(ctx)
	printSuccess("Connected")

	// Fetch schema
	printStep(2, 3, "Fetching schema...")
	result, err := c.Query("SHOW BUNDLES;", 0)
	if err != nil {
		printError(fmt.Sprintf("Failed to fetch schema: %v", err))
		os.Exit(1)
	}

	// Parse response
	resultJSON, _ := json.Marshal(result)
	schemaDef, err := schema.ParseServerSchema(resultJSON)
	if err != nil {
		printError(fmt.Sprintf("Failed to parse schema: %v", err))
		os.Exit(1)
	}
	printSuccess(fmt.Sprintf("Found %d bundle(s)", len(schemaDef.Bundles)))

	// Write to file
	printStep(3, 3, "Writing schema file...")

	// Create directory if needed
	dir := filepath.Dir(*output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		printError(fmt.Sprintf("Failed to create directory: %v", err))
		os.Exit(1)
	}

	// Marshal based on format
	var data []byte
	switch *format {
	case "json":
		data, err = json.MarshalIndent(schemaDef, "", "  ")
	case "yaml":
		printWarning("YAML format not yet implemented, using JSON")
		data, err = json.MarshalIndent(schemaDef, "", "  ")
	default:
		printError(fmt.Sprintf("Unknown format: %s", *format))
		os.Exit(1)
	}

	if err != nil {
		printError(fmt.Sprintf("Failed to marshal schema: %v", err))
		os.Exit(1)
	}

	if err := os.WriteFile(*output, data, 0644); err != nil {
		printError(fmt.Sprintf("Failed to write file: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Schema saved to: %s", colorCyan(*output)))

	// Show bundle summary
	fmt.Println()
	printInfo("Schema Summary:")
	for _, bundle := range schemaDef.Bundles {
		fmt.Printf("  • %s (%d fields, %d indexes)\n",
			colorBold(bundle.Name),
			len(bundle.Fields),
			len(bundle.Indexes))
	}
}

// handleCodegenGenerate generates code from schema
func handleCodegenGenerate(args []string) {
	fs := flag.NewFlagSet("codegen generate", flag.ExitOnError)
	schemaFile := fs.String("schema", getDefaultSchemaFile(), "Schema file path")
	output := fs.String("output", "", "Output file path (default: stdout)")
	formatType := fs.String("format", "types", "Output format: types, json-schema, graphql")
	language := fs.String("language", "go", "Language for types: go, typescript")
	packageName := fs.String("package", "models", "Package name for generated code")
	fs.Parse(args)

	printHeader("Generate Code from Schema")

	// Read schema file
	printStep(1, 3, "Reading schema file...")
	data, err := os.ReadFile(*schemaFile)
	if err != nil {
		printError(fmt.Sprintf("Failed to read schema file: %v", err))
		printInfo("Run " + colorCyan("syndrdb codegen fetch-schema") + " to fetch schema")
		os.Exit(1)
	}

	var schemaDef schema.SchemaDefinition
	if err := json.Unmarshal(data, &schemaDef); err != nil {
		printError(fmt.Sprintf("Failed to parse schema: %v", err))
		os.Exit(1)
	}
	printSuccess(fmt.Sprintf("Loaded schema with %d bundle(s)", len(schemaDef.Bundles)))

	// Load into registry
	printStep(2, 3, "Processing schema...")
	registry := codegen.NewTypeRegistry()
	registry.LoadFromSchema(&schemaDef)
	printSuccess("Schema loaded into registry")

	// Generate output
	printStep(3, 3, "Generating code...")

	var outputData string
	switch *formatType {
	case "types":
		if *language == "typescript" {
			outputData, err = generateTypeScriptTypes(registry, *packageName)
		} else {
			outputData, err = generateGoTypes(registry, *packageName)
		}
	case "json-schema":
		outputData, err = generateJSONSchema(registry)
	case "graphql":
		outputData, err = generateGraphQLSchema(registry)
	default:
		printError(fmt.Sprintf("Unknown format: %s", *formatType))
		os.Exit(1)
	}

	if err != nil {
		printError(fmt.Sprintf("Code generation failed: %v", err))
		os.Exit(1)
	}

	// Write output
	if *output == "" {
		// Print to stdout
		fmt.Println(outputData)
	} else {
		// Create directory if needed
		dir := filepath.Dir(*output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			printError(fmt.Sprintf("Failed to create directory: %v", err))
			os.Exit(1)
		}

		if err := os.WriteFile(*output, []byte(outputData), 0644); err != nil {
			printError(fmt.Sprintf("Failed to write file: %v", err))
			os.Exit(1)
		}

		printSuccess(fmt.Sprintf("Code generated: %s", colorCyan(*output)))
	}
}

// Code generation helper functions

func generateGoTypes(registry *codegen.TypeRegistry, packageName string) (string, error) {
	bundles := registry.GetAll()
	if len(bundles) == 0 {
		return "", fmt.Errorf("no bundles found in registry")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	sb.WriteString("import \"time\"\n\n")
	sb.WriteString("// Generated by syndrdb codegen - DO NOT EDIT\n\n")

	for _, bundle := range bundles {
		// Generate struct
		structName := toPascalCase(bundle.Name)
		sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

		for _, field := range bundle.Fields {
			fieldName := toPascalCase(field.Name)
			goType := syndrdbToGoType(field.Type)

			// Add pointer for optional fields
			if !field.Required {
				goType = "*" + goType
			}

			// Add JSON tag
			jsonTag := field.Name
			if !field.Required {
				jsonTag += ",omitempty"
			}

			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, goType, jsonTag))
		}

		sb.WriteString("}\n\n")
	}

	return sb.String(), nil
}

func generateTypeScriptTypes(registry *codegen.TypeRegistry, moduleName string) (string, error) {
	bundles := registry.GetAll()
	if len(bundles) == 0 {
		return "", fmt.Errorf("no bundles found in registry")
	}

	var sb strings.Builder
	sb.WriteString("// Generated by syndrdb codegen - DO NOT EDIT\n\n")

	for _, bundle := range bundles {
		interfaceName := toPascalCase(bundle.Name)
		sb.WriteString(fmt.Sprintf("export interface %s {\n", interfaceName))

		for _, field := range bundle.Fields {
			tsType := syndrdbToTypeScriptType(field.Type)
			optional := ""
			if !field.Required {
				optional = "?"
			}

			sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", field.Name, optional, tsType))
		}

		sb.WriteString("}\n\n")
	}

	return sb.String(), nil
}

func generateJSONSchema(registry *codegen.TypeRegistry) (string, error) {
	bundles := registry.GetAll()
	if len(bundles) == 0 {
		return "", fmt.Errorf("no bundles found in registry")
	}

	gen := codegen.NewJSONSchemaGenerator()
	singleSchema := schema.SchemaDefinition{Bundles: make([]schema.BundleDefinition, 0)}
	for _, b := range bundles {
		singleSchema.Bundles = append(singleSchema.Bundles, *b)
	}
	return gen.GenerateSingle(&singleSchema)
}

func generateGraphQLSchema(registry *codegen.TypeRegistry) (string, error) {
	bundles := registry.GetAll()
	if len(bundles) == 0 {
		return "", fmt.Errorf("no bundles found in registry")
	}

	gen := codegen.NewGraphQLSchemaGenerator()
	singleSchema := schema.SchemaDefinition{Bundles: make([]schema.BundleDefinition, 0)}
	for _, b := range bundles {
		singleSchema.Bundles = append(singleSchema.Bundles, *b)
	}
	return gen.Generate(&singleSchema)
}

// Type conversion helpers

func syndrdbToGoType(fieldType schema.FieldType) string {
	switch fieldType {
	case "int":
		return "int64"
	case "float":
		return "float64"
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "timestamp":
		return "time.Time"
	case "json":
		return "interface{}"
	default:
		return "interface{}"
	}
}

func syndrdbToTypeScriptType(fieldType schema.FieldType) string {
	switch fieldType {
	case "int", "float":
		return "number"
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "timestamp":
		return "Date"
	case "json":
		return "any"
	default:
		return "any"
	}
}

func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
