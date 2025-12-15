package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/migration"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

func handleMigrate(args []string) {
	if len(args) == 0 {
		printMigrateUsage()
		os.Exit(1)
	}

	subcommand := args[0]

	switch subcommand {
	case "init":
		handleMigrateInit(args[1:])
	case "generate":
		handleMigrateGenerate(args[1:])
	case "up":
		handleMigrateUp(args[1:])
	case "down":
		handleMigrateDown(args[1:])
	case "status":
		handleMigrateStatus(args[1:])
	case "validate":
		handleMigrateValidate(args[1:])
	case "help", "-h", "--help":
		printMigrateUsage()
	default:
		printError(fmt.Sprintf("Unknown migrate subcommand: %s", subcommand))
		printMigrateUsage()
		os.Exit(1)
	}
}

func printMigrateUsage() {
	printHeader("Migration Commands")
	fmt.Println("Usage:")
	fmt.Println("  syndrdb migrate " + colorYellow("<command>") + " [options]\n")
	fmt.Println("Commands:")
	fmt.Println("  " + colorGreen("init") + "       Initialize migration directory and sample schema")
	fmt.Println("  " + colorGreen("generate") + "   Generate a new migration from schema changes")
	fmt.Println("  " + colorGreen("up") + "         Apply pending migrations")
	fmt.Println("  " + colorGreen("down") + "       Rollback the last migration")
	fmt.Println("  " + colorGreen("status") + "     Show migration status")
	fmt.Println("  " + colorGreen("validate") + "   Validate migration files")
	fmt.Println("\nExamples:")
	fmt.Println("  " + colorDim("# Initialize project"))
	fmt.Println("  syndrdb migrate init")
	fmt.Println()
	fmt.Println("  " + colorDim("# Create a new migration"))
	fmt.Println("  syndrdb migrate generate --name add_users_table")
	fmt.Println()
	fmt.Println("  " + colorDim("# Apply migrations (with preview)"))
	fmt.Println("  syndrdb migrate up --dry-run")
	fmt.Println("  syndrdb migrate up")
	fmt.Println()
	fmt.Println("  " + colorDim("# Check status"))
	fmt.Println("  syndrdb migrate status")
}

// handleMigrateInit initializes a new migration project
func handleMigrateInit(args []string) {
	fs := flag.NewFlagSet("migrate init", flag.ExitOnError)
	dir := fs.String("dir", getDefaultMigrationsDir(), "Migration directory")
	schemaFile := fs.String("schema", getDefaultSchemaFile(), "Schema file path")
	force := fs.Bool("force", false, "Overwrite existing files")
	fs.Parse(args)

	printHeader("Initialize Migration Project")

	// Check if directory already exists
	if _, err := os.Stat(*dir); err == nil && !*force {
		printError(fmt.Sprintf("Directory %s already exists. Use --force to overwrite.", *dir))
		os.Exit(1)
	}

	// Create migration directory
	if err := os.MkdirAll(*dir, 0755); err != nil {
		printError(fmt.Sprintf("Failed to create directory: %v", err))
		os.Exit(1)
	}
	printSuccess(fmt.Sprintf("Created migration directory: %s", colorCyan(*dir)))

	// Create sample schema
	sampleSchema := schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name: "users",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: "int", Required: true, Unique: true},
					{Name: "email", Type: "string", Required: true, Unique: true},
					{Name: "name", Type: "string", Required: true},
					{Name: "created_at", Type: "timestamp", Required: true},
				},
				Indexes: []schema.IndexDefinition{
					{Name: "idx_email", Type: "btree", Fields: []string{"email"}},
				},
			},
		},
	}

	// Create schema directory if needed
	schemaDir := filepath.Dir(*schemaFile)
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		printError(fmt.Sprintf("Failed to create schema directory: %v", err))
		os.Exit(1)
	}

	// Write schema file
	data, _ := json.MarshalIndent(sampleSchema, "", "  ")
	if err := os.WriteFile(*schemaFile, data, 0644); err != nil {
		printError(fmt.Sprintf("Failed to write schema file: %v", err))
		os.Exit(1)
	}
	printSuccess(fmt.Sprintf("Created sample schema: %s", colorCyan(*schemaFile)))

	// Create README
	readmePath := filepath.Join(*dir, "README.md")
	readme := `# Database Migrations

This directory contains database migrations for your SyndrDB project.

## Workflow

1. Edit your schema in ` + "`schema.json`" + `
2. Generate migration: ` + "`syndrdb migrate generate --name <description>`" + `
3. Review the generated migration file
4. Apply migrations: ` + "`syndrdb migrate up`" + `

## Commands

- ` + "`syndrdb migrate status`" + ` - View migration status
- ` + "`syndrdb migrate up --dry-run`" + ` - Preview changes
- ` + "`syndrdb migrate down`" + ` - Rollback last migration
- ` + "`syndrdb migrate validate`" + ` - Validate migration files
`
	if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
		printWarning(fmt.Sprintf("Failed to create README: %v", err))
	} else {
		printSuccess(fmt.Sprintf("Created README: %s", colorCyan(readmePath)))
	}

	fmt.Println()
	printInfo("Next steps:")
	fmt.Println("  1. Edit " + colorCyan(*schemaFile) + " to define your schema")
	fmt.Println("  2. Run " + colorCyan("syndrdb migrate generate --name initial_schema"))
	fmt.Println("  3. Run " + colorCyan("syndrdb migrate up") + " to apply migrations")
}

// handleMigrateGenerate creates a new migration
func handleMigrateGenerate(args []string) {
	fs := flag.NewFlagSet("migrate generate", flag.ExitOnError)
	name := fs.String("name", "", "Migration name (required)")
	schemaFile := fs.String("schema", getDefaultSchemaFile(), "Schema file path")
	dir := fs.String("dir", getDefaultMigrationsDir(), "Migration directory")
	fs.Parse(args)

	if *name == "" {
		printError("Migration name is required")
		fmt.Println("\nUsage: syndrdb migrate generate --name <name>")
		os.Exit(1)
	}

	printHeader(fmt.Sprintf("Generate Migration: %s", *name))

	// Read schema file
	data, err := os.ReadFile(*schemaFile)
	if err != nil {
		printError(fmt.Sprintf("Failed to read schema file: %v", err))
		printInfo("Run " + colorCyan("syndrdb migrate init") + " to create a schema file")
		os.Exit(1)
	}

	var newSchema schema.SchemaDefinition
	if err := json.Unmarshal(data, &newSchema); err != nil {
		printError(fmt.Sprintf("Failed to parse schema: %v", err))
		os.Exit(1)
	}

	printInfo(fmt.Sprintf("Found %d bundle(s) in schema", len(newSchema.Bundles)))

	// Generate UP commands from schema
	upCommands := generateUpCommands(&newSchema)

	// Generate DOWN commands (drop bundles in reverse order)
	rollbackGen := migration.NewRollbackGenerator()
	downCommands, err := rollbackGen.GenerateDown(upCommands)
	if err != nil {
		printWarning(fmt.Sprintf("Could not auto-generate down commands: %v", err))
		downCommands = []string{} // Empty down commands if auto-generation fails
	} // Create migration
	mig := &migration.Migration{
		ID:           generateMigrationID(*name),
		Name:         *name,
		Up:           upCommands,
		Down:         downCommands,
		Dependencies: []string{},
		Timestamp:    time.Now(),
	}

	// Write migration file
	filePath, err := migration.WriteMigrationFile(mig, *dir)
	if err != nil {
		printError(fmt.Sprintf("Failed to write migration file: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Created migration: %s", colorCyan(filepath.Base(filePath))))
	fmt.Println()
	printInfo("Migration preview:")
	fmt.Println(colorDim("  UP commands:   " + fmt.Sprintf("%d", len(upCommands))))
	fmt.Println(colorDim("  DOWN commands: " + fmt.Sprintf("%d", len(downCommands))))
	fmt.Println()
	printInfo("Next steps:")
	fmt.Println("  1. Review the migration file: " + colorCyan(filePath))
	fmt.Println("  2. Run " + colorCyan("syndrdb migrate up --dry-run") + " to preview")
	fmt.Println("  3. Run " + colorCyan("syndrdb migrate up") + " to apply")
}

// handleMigrateUp applies pending migrations
func handleMigrateUp(args []string) {
	fs := flag.NewFlagSet("migrate up", flag.ExitOnError)
	connStr := fs.String("conn", os.Getenv("SYNDRDB_CONN"), "Connection string")
	dir := fs.String("dir", getDefaultMigrationsDir(), "Migration directory")
	dryRun := fs.Bool("dry-run", false, "Show what would be applied without executing")
	steps := fs.Int("steps", 0, "Number of migrations to apply (0 = all)")
	force := fs.Bool("force", false, "Skip confirmation prompt")
	fs.Parse(args)

	if *connStr == "" {
		printError("Connection string is required")
		fmt.Println("\nProvide via --conn flag or SYNDRDB_CONN environment variable")
		os.Exit(1)
	}

	printHeader("Apply Migrations")

	// Load migrations from directory
	migrations, err := migration.ListMigrationFiles(*dir)
	if err != nil {
		printError(fmt.Sprintf("Failed to list migrations: %v", err))
		os.Exit(1)
	}

	if len(migrations) == 0 {
		printWarning("No migration files found in " + *dir)
		printInfo("Run " + colorCyan("syndrdb migrate generate") + " to create a migration")
		return
	}

	printInfo(fmt.Sprintf("Found %d migration(s)", len(migrations)))

	// Connect to database
	opts := &client.ClientOptions{}
	c := client.NewClient(opts)
	ctx := context.Background()
	if err := c.Connect(ctx, *connStr); err != nil {
		printError(fmt.Sprintf("Failed to connect: %v", err))
		os.Exit(1)
	}
	defer c.Disconnect(ctx)

	// Create migration client
	migrationClient := migration.NewClient(&clientExecutorAdapter{client: c})

	// TODO: Load migration history from server
	// For now, we'll track in memory (in production, would use a migrations table)

	// Plan migrations
	plan, err := migrationClient.Plan(migrations)
	if err != nil {
		printError(fmt.Sprintf("Failed to create migration plan: %v", err))
		os.Exit(1)
	}

	if len(plan.Migrations) == 0 {
		printSuccess("All migrations are up to date!")
		return
	}

	// Limit to specified steps
	if *steps > 0 && len(plan.Migrations) > *steps {
		plan.Migrations = plan.Migrations[:*steps]
		plan.TotalCount = *steps
	}

	// Show plan
	fmt.Println()
	printInfo(fmt.Sprintf("Pending migrations: %d", plan.TotalCount))
	for i, mig := range plan.Migrations {
		status := colorYellow("pending")
		fmt.Printf("  %d. %s [%s]\n", i+1, colorBold(mig.Name), status)
		fmt.Printf("     %s (%d up, %d down)\n", colorDim(mig.ID), len(mig.Up), len(mig.Down))
	}

	if *dryRun {
		fmt.Println()
		printInfo(colorYellow("DRY RUN") + " - no changes will be applied")
		return
	}

	// Confirm before applying
	if !*force {
		fmt.Println()
		if !promptConfirm(fmt.Sprintf("Apply %d migration(s)?", plan.TotalCount)) {
			printInfo("Cancelled")
			return
		}
	}

	// Apply migrations
	fmt.Println()
	printHeader("Applying Migrations")

	plan.DryRun = false
	if err := migrationClient.Apply(plan); err != nil {
		printError(fmt.Sprintf("Migration failed: %v", err))
		os.Exit(1)
	}

	printSuccess("All migrations applied successfully!")
}

// handleMigrateDown rolls back the last migration
func handleMigrateDown(args []string) {
	fs := flag.NewFlagSet("migrate down", flag.ExitOnError)
	connStr := fs.String("conn", os.Getenv("SYNDRDB_CONN"), "Connection string")
	dir := fs.String("dir", getDefaultMigrationsDir(), "Migration directory")
	dryRun := fs.Bool("dry-run", false, "Show what would be rolled back without executing")
	force := fs.Bool("force", false, "Skip confirmation prompt")
	fs.Parse(args)

	if *connStr == "" {
		printError("Connection string is required")
		fmt.Println("\nProvide via --conn flag or SYNDRDB_CONN environment variable")
		os.Exit(1)
	}

	printHeader("Rollback Migration")

	// Load migrations
	migrations, err := migration.ListMigrationFiles(*dir)
	if err != nil {
		printError(fmt.Sprintf("Failed to list migrations: %v", err))
		os.Exit(1)
	}

	if len(migrations) == 0 {
		printWarning("No migrations found")
		return
	}

	// Get last migration (TODO: track which are applied)
	lastMigration := migrations[len(migrations)-1]

	printInfo(fmt.Sprintf("Rolling back: %s", colorBold(lastMigration.Name)))
	fmt.Println(colorDim("  ID: " + lastMigration.ID))
	fmt.Println(colorDim(fmt.Sprintf("  DOWN commands: %d", len(lastMigration.Down))))

	if *dryRun {
		fmt.Println()
		printInfo(colorYellow("DRY RUN") + " - no changes will be applied")
		return
	}

	// Confirm
	if !*force {
		fmt.Println()
		if !promptConfirm("Rollback this migration?") {
			printInfo("Cancelled")
			return
		}
	}

	// Connect and rollback
	opts := &client.ClientOptions{}
	c := client.NewClient(opts)
	ctx := context.Background()
	if err := c.Connect(ctx, *connStr); err != nil {
		printError(fmt.Sprintf("Failed to connect: %v", err))
		os.Exit(1)
	}
	defer c.Disconnect(ctx)

	migrationClient := migration.NewClient(&clientExecutorAdapter{client: c})

	fmt.Println()
	printHeader("Rolling Back")

	if err := migrationClient.Rollback(lastMigration.ID, migrations); err != nil {
		printError(fmt.Sprintf("Rollback failed: %v", err))
		os.Exit(1)
	}

	printSuccess("Migration rolled back successfully!")
}

// handleMigrateStatus shows the status of migrations
func handleMigrateStatus(args []string) {
	fs := flag.NewFlagSet("migrate status", flag.ExitOnError)
	connStr := fs.String("conn", os.Getenv("SYNDRDB_CONN"), "Connection string (optional)")
	dir := fs.String("dir", getDefaultMigrationsDir(), "Migration directory")
	fs.Parse(args)

	printHeader("Migration Status")

	// Load migrations
	migrations, err := migration.ListMigrationFiles(*dir)
	if err != nil {
		printError(fmt.Sprintf("Failed to list migrations: %v", err))
		os.Exit(1)
	}

	if len(migrations) == 0 {
		printWarning("No migration files found in " + *dir)
		printInfo("Run " + colorCyan("syndrdb migrate generate") + " to create a migration")
		return
	}

	// Show all migrations
	fmt.Println()
	rows := make([][]string, 0, len(migrations))
	for _, mig := range migrations {
		status := colorYellow("pending") // TODO: check if applied
		rows = append(rows, []string{
			mig.ID,
			mig.Name,
			status,
			mig.Timestamp.Format("2006-01-02 15:04"),
		})
	}

	printTable(
		[]string{"ID", "Name", "Status", "Created"},
		rows,
	)

	fmt.Println()
	printInfo(fmt.Sprintf("Total migrations: %d", len(migrations)))

	if *connStr != "" {
		printInfo("Connected to database - showing actual status")
	} else {
		printInfo("Not connected - showing file status only")
		fmt.Println(colorDim("  Use --conn to check applied migrations"))
	}
}

// handleMigrateValidate validates migration files
func handleMigrateValidate(args []string) {
	fs := flag.NewFlagSet("migrate validate", flag.ExitOnError)
	dir := fs.String("dir", getDefaultMigrationsDir(), "Migration directory")
	fs.Parse(args)

	printHeader("Validate Migrations")

	// Load migrations
	migrations, err := migration.ListMigrationFiles(*dir)
	if err != nil {
		printError(fmt.Sprintf("Failed to list migrations: %v", err))
		os.Exit(1)
	}

	if len(migrations) == 0 {
		printWarning("No migration files found")
		return
	}

	printInfo(fmt.Sprintf("Validating %d migration(s)...", len(migrations)))
	fmt.Println()

	// Validate each migration
	validator := migration.NewMigrationValidator(migration.NewMigrationHistory())
	validation := validator.Validate(migrations)

	if validation.Valid {
		printSuccess("All migrations are valid!")
		return
	}

	// Show conflicts
	printError("Validation failed!")
	fmt.Println()
	for _, conflict := range validation.Conflicts {
		fmt.Println(colorRed("✗") + " " + conflict.Message)
	}

	os.Exit(1)
}

// Helper functions

func getDefaultMigrationsDir() string {
	if dir := os.Getenv("SYNDRDB_MIGRATIONS_DIR"); dir != "" {
		return dir
	}
	return "./migrations"
}

func getDefaultSchemaFile() string {
	if file := os.Getenv("SYNDRDB_SCHEMA_FILE"); file != "" {
		return file
	}
	return "./schema.json"
}

func promptConfirm(message string) bool {
	fmt.Printf("%s [y/N]: ", message)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func generateMigrationID(name string) string {
	// Convert name to valid ID format
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "_")
	id = strings.ReplaceAll(id, "-", "_")
	return id
}

func generateUpCommands(schema *schema.SchemaDefinition) []string {
	commands := make([]string, 0)
	for _, bundle := range schema.Bundles {
		// Generate CREATE BUNDLE command
		fields := make([]string, 0, len(bundle.Fields))
		for _, field := range bundle.Fields {
			fieldDef := field.Name + " " + string(field.Type)
			if field.Required {
				fieldDef += " REQUIRED"
			}
			if field.Unique {
				fieldDef += " UNIQUE"
			}
			fields = append(fields, fieldDef)
		}

		cmd := fmt.Sprintf("CREATE BUNDLE %s (%s);", bundle.Name, strings.Join(fields, ", "))
		commands = append(commands, cmd)

		// Generate CREATE INDEX commands
		for _, index := range bundle.Indexes {
			indexCmd := fmt.Sprintf("CREATE INDEX %s ON %s USING %s (%s);",
				index.Name,
				bundle.Name,
				strings.ToUpper(string(index.Type)),
				strings.Join(index.Fields, ", "))
			commands = append(commands, indexCmd)
		}
	}
	return commands
}

// clientExecutorAdapter adapts client.Client to migration.MigrationExecutor
type clientExecutorAdapter struct {
	client *client.Client
}

func (a *clientExecutorAdapter) Execute(command string) (interface{}, error) {
	// Determine if query or mutation based on command
	cmdUpper := strings.ToUpper(strings.TrimSpace(command))
	if strings.HasPrefix(cmdUpper, "SELECT") || strings.HasPrefix(cmdUpper, "SHOW") {
		return a.client.Query(command, 0)
	}
	return a.client.Mutate(command, 0)
}
