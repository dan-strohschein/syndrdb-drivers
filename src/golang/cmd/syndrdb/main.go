//go:build milestone2

package main

import (
	"fmt"
	"os"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "migrate":
		handleMigrate(os.Args[2:])
	case "codegen":
		handleCodegen(os.Args[2:])
	case "test":
		handleTest(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("syndrdb v%s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		printError(fmt.Sprintf("Unknown command: %s", command))
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(colorBold(colorCyan("SyndrDB CLI")) + " - Database migrations, codegen, and testing\n")
	fmt.Println("Usage:")
	fmt.Println("  syndrdb " + colorYellow("<command>") + " [options]\n")
	fmt.Println("Commands:")
	fmt.Println("  " + colorGreen("migrate") + "   Manage database migrations")
	fmt.Println("  " + colorGreen("codegen") + "   Generate code from schema")
	fmt.Println("  " + colorGreen("test") + "      Test database connection and schema")
	fmt.Println("  " + colorGreen("version") + "   Show version information")
	fmt.Println("  " + colorGreen("help") + "      Show this help message\n")
	fmt.Println("Run '" + colorCyan("syndrdb <command> --help") + "' for more information on a command.\n")
	fmt.Println("Environment Variables:")
	fmt.Println("  SYNDRDB_CONN             Database connection string")
	fmt.Println("  SYNDRDB_MIGRATIONS_DIR   Directory for migration files (default: ./migrations)")
	fmt.Println("  SYNDRDB_SCHEMA_FILE      Path to schema file (default: ./schema.json)")
}
