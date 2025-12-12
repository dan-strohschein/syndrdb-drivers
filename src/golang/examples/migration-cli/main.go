package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/migration"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

const version = "1.0.0"

func main() {
	// Subcommands
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
	upCmd := flag.NewFlagSet("up", flag.ExitOnError)
	downCmd := flag.NewFlagSet("down", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)

	// Init flags
	initOutput := initCmd.String("output", "./db/schema.json", "Output path for schema file")

	// Generate flags
	genName := generateCmd.String("name", "", "Migration name (required)")
	genSchema := generateCmd.String("schema", "./db/schema.json", "Path to schema file")
	genOutput := generateCmd.String("output", "./migrations", "Output directory for migrations")

	// Up/Down/Status flags
	upConn := upCmd.String("conn", "", "Connection string (required)")
	upDir := upCmd.String("dir", "./migrations", "Migrations directory")
	downConn := downCmd.String("conn", "", "Connection string (required)")
	downDir := downCmd.String("dir", "./migrations", "Migrations directory")
	statusConn := statusCmd.String("conn", "", "Connection string (required)")
	statusDir := statusCmd.String("dir", "./migrations", "Migrations directory")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		initCmd.Parse(os.Args[2:])
		handleInit(*initOutput)
	case "generate":
		generateCmd.Parse(os.Args[2:])
		handleGenerate(*genName, *genSchema, *genOutput)
	case "up":
		upCmd.Parse(os.Args[2:])
		handleUp(*upConn, *upDir)
	case "down":
		downCmd.Parse(os.Args[2:])
		handleDown(*downConn, *downDir)
	case "status":
		statusCmd.Parse(os.Args[2:])
		handleStatus(*statusConn, *statusDir)
	case "version":
		fmt.Printf("migration-cli v%s\n", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("SyndrDB Migration CLI")
	fmt.Println("\nUsage:")
	fmt.Println("  migration-cli init --output <path>")
	fmt.Println("  migration-cli generate --name <name> --schema <path> --output <dir>")
	fmt.Println("  migration-cli up --conn <connection-string> --dir <migrations-dir>")
	fmt.Println("  migration-cli down --conn <connection-string> --dir <migrations-dir>")
	fmt.Println("  migration-cli status --conn <connection-string> --dir <migrations-dir>")
	fmt.Println("  migration-cli version")
}

func handleInit(output string) {
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
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}

	// Create directory if needed
	dir := filepath.Dir(output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Write schema file
	data, err := json.MarshalIndent(sampleSchema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling schema: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing schema file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Initialized schema at %s\n", output)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Edit the schema file to match your needs")
	fmt.Println("  2. Run: migration-cli generate --name initial_schema")
	fmt.Println("  3. Run: migration-cli up --conn <your-connection-string>")
}

func handleGenerate(name, schemaPath, outputDir string) {
	if name == "" {
		fmt.Fprintf(os.Stderr, "Error: --name is required\n")
		os.Exit(1)
	}

	// Read schema file
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading schema: %v\n", err)
		os.Exit(1)
	}

	var newSchema schema.SchemaDefinition
	if err := json.Unmarshal(data, &newSchema); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing schema: %v\n", err)
		os.Exit(1)
	}

	// For simplicity, we'll generate UP commands from the schema
	// In a real app, you'd compare with the current server schema
	upCommands := generateUpCommands(&newSchema)

	// Generate DOWN commands automatically
	rollbackGen := migration.NewRollbackGenerator()
	downCommands, err := rollbackGen.GenerateDown(upCommands)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating down commands: %v\n", err)
		os.Exit(1)
	}

	// Create migration file
	timestamp := time.Now().Format("20060102150405")
	migrationID := fmt.Sprintf("%s_%s", timestamp, name)

	mig := &migration.Migration{
		ID:          migrationID,
		Name:        name,
		Description: fmt.Sprintf("Migration: %s", name),
		Up:          upCommands,
		Down:        downCommands,
		CreatedAt:   time.Now(),
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Write migration file
	migPath := filepath.Join(outputDir, migrationID+".json")
	migData, err := json.MarshalIndent(mig, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling migration: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(migPath, migData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing migration file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Generated migration: %s\n", migrationID)
	fmt.Printf("  Location: %s\n", migPath)
	fmt.Printf("  Up commands: %d\n", len(upCommands))
	fmt.Printf("  Down commands: %d (auto-generated)\n", len(downCommands))
	fmt.Println("\nTo apply: migration-cli up --conn <connection-string>")
}

func handleUp(connString, migrationsDir string) {
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

	// Load migrations
	migrations, err := loadMigrations(migrationsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading migrations: %v\n", err)
		os.Exit(1)
	}

	if len(migrations) == 0 {
		fmt.Println("No migrations found")
		return
	}

	// Create migration client
	migClient := migration.NewClient(c)

	// Plan migrations
	fmt.Println("\nPlanning migrations...")
	plan, err := migClient.Plan(migrations)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Planning failed: %v\n", err)
		os.Exit(1)
	}

	if len(plan.ToApply) == 0 {
		fmt.Println("✓ No pending migrations")
		return
	}

	fmt.Printf("\nWill apply %d migration(s):\n", len(plan.ToApply))
	for _, mig := range plan.ToApply {
		fmt.Printf("  - %s: %s\n", mig.ID, mig.Name)
	}

	// Apply migrations
	fmt.Println("\nApplying migrations...")
	if err := migClient.Apply(plan); err != nil {
		fmt.Fprintf(os.Stderr, "Apply failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Successfully applied %d migration(s)\n", len(plan.ToApply))
}

func handleDown(connString, migrationsDir string) {
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

	// Load migrations
	migrations, err := loadMigrations(migrationsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading migrations: %v\n", err)
		os.Exit(1)
	}

	// Create migration client
	migClient := migration.NewClient(c)

	// Get last applied migration
	applied := migClient.GetAppliedMigrations()
	if len(applied) == 0 {
		fmt.Println("No migrations to rollback")
		return
	}

	lastMigID := applied[len(applied)-1]
	fmt.Printf("\nRolling back: %s\n", lastMigID)

	// Rollback
	if err := migClient.Rollback(lastMigID, migrations); err != nil {
		fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Successfully rolled back migration")
}

func handleStatus(connString, migrationsDir string) {
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

	// Load migrations
	migrations, err := loadMigrations(migrationsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading migrations: %v\n", err)
		os.Exit(1)
	}

	// Create migration client
	migClient := migration.NewClient(c)

	// Get applied migrations
	applied := migClient.GetAppliedMigrations()

	fmt.Printf("\nMigration Status:\n")
	fmt.Printf("  Total migrations: %d\n", len(migrations))
	fmt.Printf("  Applied: %d\n", len(applied))
	fmt.Printf("  Pending: %d\n", len(migrations)-len(applied))

	if len(applied) > 0 {
		fmt.Println("\nApplied migrations:")
		for _, migID := range applied {
			record, ok := migClient.GetMigrationRecord(migID)
			if ok {
				fmt.Printf("  ✓ %s (applied %s)\n", migID, record.AppliedAt.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("  ✓ %s\n", migID)
			}
		}
	}

	// Show pending
	appliedMap := make(map[string]bool)
	for _, id := range applied {
		appliedMap[id] = true
	}

	var pending []*migration.Migration
	for _, mig := range migrations {
		if !appliedMap[mig.ID] {
			pending = append(pending, mig)
		}
	}

	if len(pending) > 0 {
		fmt.Println("\nPending migrations:")
		for _, mig := range pending {
			fmt.Printf("  ○ %s: %s\n", mig.ID, mig.Name)
		}
	}
}

func loadMigrations(dir string) ([]*migration.Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var migrations []*migration.Migration
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var mig migration.Migration
		if err := json.Unmarshal(data, &mig); err != nil {
			return nil, err
		}

		migrations = append(migrations, &mig)
	}

	return migrations, nil
}

func generateUpCommands(schemaDef *schema.SchemaDefinition) []string {
	var commands []string

	for _, bundle := range schemaDef.Bundles {
		// CREATE BUNDLE command
		cmd := fmt.Sprintf(`CREATE BUNDLE "%s" WITH FIELDS (`, bundle.Name)

		for i, field := range bundle.Fields {
			if i > 0 {
				cmd += ", "
			}
			cmd += fmt.Sprintf("%s %s", field.Name, field.Type)
			if field.Required {
				cmd += " REQUIRED"
			}
			if field.Unique {
				cmd += " UNIQUE"
			}
		}

		cmd += ");"
		commands = append(commands, cmd)

		// CREATE INDEX commands
		for _, idx := range bundle.Indexes {
			idxCmd := fmt.Sprintf(`CREATE INDEX "%s" ON "%s" (%s) TYPE %s;`,
				idx.Name, bundle.Name, idx.Fields[0], idx.Type)
			commands = append(commands, idxCmd)
		}
	}

	return commands
}
