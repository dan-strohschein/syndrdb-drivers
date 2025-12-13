//go:build milestone2

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/migration"
)

func handleTest(args []string) {
	if len(args) == 0 {
		printTestUsage()
		os.Exit(1)
	}

	subcommand := args[0]

	switch subcommand {
	case "connection":
		handleTestConnection(args[1:])
	case "migrations":
		handleTestMigrations(args[1:])
	case "all":
		handleTestAll(args[1:])
	case "help", "-h", "--help":
		printTestUsage()
	default:
		printError(fmt.Sprintf("Unknown test subcommand: %s", subcommand))
		printTestUsage()
		os.Exit(1)
	}
}

func printTestUsage() {
	printHeader("Test Commands")
	fmt.Println("Usage:")
	fmt.Println("  syndrdb test " + colorYellow("<command>") + " [options]\n")
	fmt.Println("Commands:")
	fmt.Println("  " + colorGreen("connection") + "   Test database connection and health")
	fmt.Println("  " + colorGreen("migrations") + "  Validate migration files")
	fmt.Println("  " + colorGreen("all") + "         Run all tests")
	fmt.Println("\nExamples:")
	fmt.Println("  " + colorDim("# Test connection"))
	fmt.Println("  syndrdb test connection")
	fmt.Println()
	fmt.Println("  " + colorDim("# Validate migrations"))
	fmt.Println("  syndrdb test migrations --dir ./migrations")
	fmt.Println()
	fmt.Println("  " + colorDim("# Run all tests"))
	fmt.Println("  syndrdb test all")
}

// handleTestConnection tests database connection
func handleTestConnection(args []string) {
	fs := flag.NewFlagSet("test connection", flag.ExitOnError)
	connStr := fs.String("conn", os.Getenv("SYNDRDB_CONN"), "Connection string")
	verbose := fs.Bool("verbose", false, "Show detailed connection info")
	fs.Parse(args)

	if *connStr == "" {
		printError("Connection string is required")
		fmt.Println("\nProvide via --conn flag or SYNDRDB_CONN environment variable")
		os.Exit(1)
	}

	printHeader("Test Database Connection")

	// Test 1: Parse connection string
	testsPassed := 0
	testsTotal := 4

	fmt.Println()
	fmt.Print("  1. Parse connection string... ")
	if *verbose {
		fmt.Println()
		fmt.Println(colorDim("     Connection: " + maskConnectionString(*connStr)))
	}
	// Basic validation
	if len(*connStr) < 10 || !containsString(*connStr, "syndrdb://") {
		printError("Invalid connection string format")
		os.Exit(1)
	}
	printSuccess("OK")
	testsPassed++

	// Test 2: Connect
	fmt.Print("  2. Connect to database... ")
	opts := &client.ClientOptions{}
	c := client.NewClient(opts)
	ctx := context.Background()
	err := c.Connect(ctx, *connStr)
	if err != nil {
		fmt.Println(colorRed("FAIL"))
		printError(fmt.Sprintf("Connection failed: %v", err))
		os.Exit(1)
	}
	defer c.Disconnect(ctx)
	printSuccess("OK")
	testsPassed++

	// Test 3: Ping
	fmt.Print("  3. Ping server... ")
	pingStart := time.Now()
	if err := c.Ping(ctx); err != nil {
		fmt.Println(colorRed("FAIL"))
		printError(fmt.Sprintf("Ping failed: %v", err))
		os.Exit(1)
	}
	pingDuration := time.Since(pingStart)
	if *verbose {
		fmt.Printf("%s (%s)\n", colorGreen("OK"), colorDim(fmt.Sprintf("%dms", pingDuration.Milliseconds())))
	} else {
		printSuccess(fmt.Sprintf("OK (%dms)", pingDuration.Milliseconds()))
	}
	testsPassed++

	// Test 4: Check state
	fmt.Print("  4. Check connection state... ")
	state := c.GetState()
	if state != client.CONNECTED {
		fmt.Println(colorRed("FAIL"))
		printError(fmt.Sprintf("Expected state CONNECTED, got %s", state))
		os.Exit(1)
	}
	printSuccess("OK")
	testsPassed++

	// Summary
	fmt.Println()
	if testsPassed == testsTotal {
		printSuccess(fmt.Sprintf("All tests passed! (%d/%d)", testsPassed, testsTotal))
	} else {
		printError(fmt.Sprintf("Some tests failed (%d/%d passed)", testsPassed, testsTotal))
		os.Exit(1)
	}

	if *verbose {
		fmt.Println()
		printInfo("Connection Details:")
		fmt.Println(colorDim(fmt.Sprintf("  State: %s", state)))
		fmt.Println(colorDim(fmt.Sprintf("  Ping Latency: %dms", pingDuration.Milliseconds())))
	}
}

// handleTestMigrations validates migration files
func handleTestMigrations(args []string) {
	fs := flag.NewFlagSet("test migrations", flag.ExitOnError)
	dir := fs.String("dir", getDefaultMigrationsDir(), "Migration directory")
	verbose := fs.Bool("verbose", false, "Show detailed validation info")
	fs.Parse(args)

	printHeader("Test Migration Files")

	// Test 1: Check directory exists
	fmt.Println()
	fmt.Print("  1. Check migrations directory... ")
	if _, err := os.Stat(*dir); os.IsNotExist(err) {
		fmt.Println(colorRed("FAIL"))
		printError(fmt.Sprintf("Directory not found: %s", *dir))
		printInfo("Run " + colorCyan("syndrdb migrate init") + " to initialize")
		os.Exit(1)
	}
	printSuccess("OK")

	// Test 2: Load migrations
	fmt.Print("  2. Load migration files... ")
	migrations, err := migration.ListMigrationFiles(*dir)
	if err != nil {
		fmt.Println(colorRed("FAIL"))
		printError(fmt.Sprintf("Failed to load migrations: %v", err))
		os.Exit(1)
	}
	printSuccess(fmt.Sprintf("OK (%d found)", len(migrations)))

	if len(migrations) == 0 {
		printWarning("No migration files found")
		fmt.Println()
		printInfo("Run " + colorCyan("syndrdb migrate generate") + " to create migrations")
		return
	}

	// Test 3: Validate migrations
	fmt.Print("  3. Validate migration structure... ")
	validator := migration.NewMigrationValidator(migration.NewMigrationHistory())
	validation := validator.Validate(migrations)

	if !validation.Valid {
		fmt.Println(colorRed("FAIL"))
		printError("Validation errors found:")
		for _, conflict := range validation.Conflicts {
			fmt.Println("  " + colorRed("•") + " " + conflict.Message)
		}
		os.Exit(1)
	}
	printSuccess("OK")

	// Test 4: Check for issues
	fmt.Print("  4. Check for common issues... ")
	issues := checkMigrationIssues(migrations)
	if len(issues) > 0 {
		fmt.Println(colorYellow("WARN"))
		for _, issue := range issues {
			printWarning(issue)
		}
	} else {
		printSuccess("OK")
	}

	// Summary
	fmt.Println()
	printSuccess("All migration tests passed!")

	if *verbose {
		fmt.Println()
		printInfo("Migration Summary:")
		for i, mig := range migrations {
			fmt.Printf("  %d. %s\n", i+1, colorBold(mig.Name))
			fmt.Println(colorDim(fmt.Sprintf("     ID: %s", mig.ID)))
			fmt.Println(colorDim(fmt.Sprintf("     Up commands: %d", len(mig.Up))))
			fmt.Println(colorDim(fmt.Sprintf("     Down commands: %d", len(mig.Down))))
			fmt.Println(colorDim(fmt.Sprintf("     Created: %s", mig.Timestamp.Format("2006-01-02 15:04"))))
		}
	}
}

// handleTestAll runs all tests
func handleTestAll(args []string) {
	fs := flag.NewFlagSet("test all", flag.ExitOnError)
	connStr := fs.String("conn", os.Getenv("SYNDRDB_CONN"), "Connection string")
	dir := fs.String("dir", getDefaultMigrationsDir(), "Migration directory")
	verbose := fs.Bool("verbose", false, "Show detailed test info")
	fs.Parse(args)

	printHeader("Run All Tests")

	testsRun := 0
	testsPassed := 0

	// Test connection
	fmt.Println()
	fmt.Println(colorBold("Connection Tests"))
	fmt.Println(colorDim("────────────────────────────────────────"))
	testsRun++
	if runConnectionTests(*connStr, *verbose) {
		testsPassed++
	}

	// Test migrations
	fmt.Println()
	fmt.Println(colorBold("Migration Tests"))
	fmt.Println(colorDim("────────────────────────────────────────"))
	testsRun++
	if runMigrationTests(*dir, *verbose) {
		testsPassed++
	}

	// Summary
	fmt.Println()
	fmt.Println(colorBold("Test Summary"))
	fmt.Println(colorDim("────────────────────────────────────────"))
	if testsPassed == testsRun {
		printSuccess(fmt.Sprintf("All test suites passed! (%d/%d)", testsPassed, testsRun))
	} else {
		printError(fmt.Sprintf("Some test suites failed (%d/%d passed)", testsPassed, testsRun))
		os.Exit(1)
	}
}

// Helper functions

func runConnectionTests(connStr string, verbose bool) bool {
	if connStr == "" {
		printWarning("Skipping connection tests (no connection string)")
		return true
	}

	opts := &client.ClientOptions{}
	c := client.NewClient(opts)
	ctx := context.Background()
	err := c.Connect(ctx, connStr)
	if err != nil {
		printError(fmt.Sprintf("Connection failed: %v", err))
		return false
	}
	defer c.Disconnect(ctx)

	if err := c.Ping(ctx); err != nil {
		printError(fmt.Sprintf("Ping failed: %v", err))
		return false
	}

	printSuccess("Connection tests passed")
	return true
}

func runMigrationTests(dir string, verbose bool) bool {
	migrations, err := migration.ListMigrationFiles(dir)
	if err != nil {
		printError(fmt.Sprintf("Failed to load migrations: %v", err))
		return false
	}

	if len(migrations) == 0 {
		printWarning("No migration files found")
		return true
	}

	validator := migration.NewMigrationValidator(migration.NewMigrationHistory())
	validation := validator.Validate(migrations)

	if !validation.Valid {
		printError("Migration validation failed")
		for _, conflict := range validation.Conflicts {
			fmt.Println("  " + colorRed("•") + " " + conflict.Message)
		}
		return false
	}

	printSuccess(fmt.Sprintf("Migration tests passed (%d files)", len(migrations)))
	return true
}

func maskConnectionString(connStr string) string {
	// Mask password in connection string
	// syndrdb://host:port:database:user:password;
	parts := []rune(connStr)
	inPassword := false
	colonCount := 0
	for i, ch := range parts {
		if ch == ':' {
			colonCount++
			if colonCount == 4 {
				inPassword = true
			}
		} else if ch == ';' {
			inPassword = false
		} else if inPassword && i > 0 {
			parts[i] = '*'
		}
	}
	return string(parts)
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || containsString(s[1:], substr)))
}

func checkMigrationIssues(migrations []*migration.Migration) []string {
	issues := make([]string, 0)

	for _, mig := range migrations {
		// Check for empty migrations
		if len(mig.Up) == 0 {
			issues = append(issues, fmt.Sprintf("Migration %s has no UP commands", mig.ID))
		}

		// Check for missing down migrations
		if len(mig.Down) == 0 {
			issues = append(issues, fmt.Sprintf("Migration %s has no DOWN commands (rollback may not work)", mig.ID))
		}

		// Check for very old migrations (>1 year)
		if time.Since(mig.Timestamp) > 365*24*time.Hour {
			issues = append(issues, fmt.Sprintf("Migration %s is very old (%s)", mig.ID, mig.Timestamp.Format("2006-01-02")))
		}
	}

	return issues
}
