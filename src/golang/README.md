# SyndrDB Go Driver

A full-featured Go driver for SyndrDB with native library support and WebAssembly compilation.

## Features

- ✅ **Single Persistent Connection** - Efficient TCP connection management with EOT delimiters
- ✅ **State Machine** - Formal connection state tracking with enriched event metadata
- ✅ **Structured Errors** - JSON-serializable errors with cause chains and debug mode
- ✅ **Schema Management** - Full schema parsing, comparison, and command serialization
- ✅ **Migration System** - Complete migration protocol with **automatic Down command generation**, checksum validation, dependency tracking, and conflict resolution
- ✅ **Code Generation** - JSON Schema and GraphQL SDL generation from SyndrDB schemas
- ✅ **Response Mapping** - Comprehensive type coercion matching Node.js implementation
- ✅ **WASM Support** - Full WebAssembly compilation with JavaScript Promise API
- ✅ **Zero Dependencies** - Uses only Go standard library

## Installation

```bash
go get github.com/dan-strohschein/syndrdb-drivers/src/golang
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
)

func main() {
    // Create client with options
    opts := &client.ClientOptions{
        DefaultTimeoutMs: 10000,
        DebugMode:        false,
        MaxRetries:       3,
    }
    
    c := client.NewClient(opts)
    
    // Register state change handler
    c.OnStateChange(func(transition client.StateTransition) {
        fmt.Printf("State: %s -> %s\n", transition.From, transition.To)
    })
    
    // Connect to database
    if err := c.Connect("syndrdb://localhost:7632/mydb"); err != nil {
        log.Fatal(err)
    }
    defer c.Disconnect()
    
    // Execute query
    result, err := c.Query("SELECT * FROM users", 5000)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(result)
}
```

## Example Applications

Five complete example applications demonstrate real-world usage patterns:

### 1. [Migration CLI](./examples/migration-cli/) - Go

Command-line migration tool with automatic rollback generation.

```bash
cd examples/migration-cli
go build -o syndr-migrate

# Initialize sample schema
./syndr-migrate init

# Generate migration with auto-rollback
./syndr-migrate generate -name "add_users_table"

# Apply migrations
./syndr-migrate up
```

**Features:**
- Automatic Down command generation
- Migration history tracking
- Safe rollbacks
- JSON migration files

### 2. [Schema Codegen](./examples/schema-codegen/) - Go

Generate TypeScript, JSON Schema, and GraphQL from SyndrDB schemas.

```bash
cd examples/schema-codegen
go build -o syndr-codegen

# Fetch schema from server
./syndr-codegen fetch -conn "syndrdb://localhost:1776:primary:root:root;" -output schema.json

# Generate TypeScript types
./syndr-codegen typescript -schema schema.json -output types.ts

# Generate GraphQL schema
./syndr-codegen graphql -schema schema.json -output schema.graphql
```

**Features:**
- Multiple output formats
- Single and multi-file modes
- Direct server connection or file input
- Perfect for CI/CD pipelines

### 3. [Node.js GraphQL API](./examples/node-graphql-api/) - Node.js + WASM

Production-ready GraphQL API using the WASM driver - **the primary use case for SyndrDB**.

```bash
cd examples/node-graphql-api
npm install
node server.js
```

**Features:**
- Full GraphQL schema with queries and mutations
- Built-in GraphQL Playground
- Health checks and graceful shutdown
- CORS enabled
- Production-ready error handling

**Example Query:**
```graphql
query {
  todos {
    id
    title
    completed
  }
}
```

### 4. [WASM Web Designer](./examples/wasm-web/) - Browser

Interactive browser-based schema designer with real-time code generation.

```bash
cd examples/wasm-web
python3 -m http.server 8080
# Open http://localhost:8080
```

**Features:**
- Beautiful, modern UI
- Real-time code generation
- TypeScript, JSON Schema, and GraphQL output
- Copy to clipboard or download
- Fully offline capable
- Pre-loaded examples

### 5. [Node.js Migration Script](./examples/node-migration/) - Node.js + WASM

Lightweight CI/CD migration script for automated deployments.

```bash
cd examples/node-migration
npm install
node migrate.js
```

**Features:**
- Schema comparison and diff
- Automatic migration generation
- Dry-run mode
- Perfect for CI/CD pipelines
- Environment variable configuration

**CI/CD Example:**
```yaml
# GitHub Actions
- name: Run migration
  env:
    SYNDR_CONN: ${{ secrets.SYNDR_CONN }}
  run: |
    cd examples/node-migration
    npm install
    node migrate.js
```

See each example's README for detailed documentation and usage.

## Architecture

### Package Structure

```
github.com/dan-strohschein/syndrdb-drivers/src/golang/
├── client/          # Core client with connection management
│   ├── client.go    # Main Client API
│   ├── connection.go # TCP connection with EOT scanner
│   ├── state.go     # State machine and transitions
│   ├── errors.go    # Structured error types
│   ├── options.go   # Client configuration
│   └── version.go   # Build version tracking
├── schema/          # Schema management
│   ├── types.go     # Schema type definitions
│   ├── manager.go   # Schema parsing and comparison
│   └── serializer.go # Command generation
├── migration/       # Migration system
│   ├── types.go     # Migration type definitions
│   ├── client.go    # Migration operations
│   ├── history.go   # Migration tracking
│   ├── validator.go # Validation and conflict detection
│   └── errors.go    # Migration-specific errors
├── codegen/         # Code generation
│   ├── json_schema.go    # JSON Schema generation
│   ├── graphql_schema.go # GraphQL SDL generation
│   └── registry.go       # Type registry with caching
├── mapper/          # Response mapping
│   └── response.go  # Type coercion
├── wasm/            # WebAssembly exports
│   ├── main.go      # WASM entry point
│   └── README.md    # WASM documentation
└── scripts/         # Build scripts
    ├── build-lib.sh  # Library build script
    └── build-wasm.sh # WASM build script
```

### Connection State Machine

```
DISCONNECTED ──▶ CONNECTING ──▶ CONNECTED
                     │               │
                     │               ▼
                     └──────▶ DISCONNECTING
                                     │
                                     ▼
                              DISCONNECTED
```

Legal transitions:
- `DISCONNECTED → CONNECTING` (user initiated)
- `CONNECTING → CONNECTED` (successful)
- `CONNECTING → DISCONNECTED` (failed)
- `CONNECTED → DISCONNECTING` (user disconnect)
- `DISCONNECTING → DISCONNECTED` (completed)

## API Reference

### Client

#### Creating a Client

```go
import "github.com/dan-strohschein/syndrdb-drivers/src/golang/client"

// With default options
c := client.NewClient(nil)

// With custom options
opts := &client.ClientOptions{
    DefaultTimeoutMs: 10000,  // Default: 10000
    DebugMode:        false,  // Default: false
    MaxRetries:       3,      // Default: 3
}
c := client.NewClient(opts)
```

#### Connection Methods

```go
// Connect with retry logic (exponential backoff: 100ms, 200ms, 400ms)
err := c.Connect("syndrdb://host:port/database")

// Disconnect gracefully
err := c.Disconnect()

// Get current state
state := c.GetState() // DISCONNECTED, CONNECTING, CONNECTED, DISCONNECTING
```

#### Query Methods

```go
// Execute query with timeout
result, err := c.Query("SELECT * FROM users", 5000)

// Execute mutation
result, err := c.Mutate("INSERT INTO users ...", 5000)
```

#### State Change Events

```go
c.OnStateChange(func(transition client.StateTransition) {
    fmt.Printf("%s -> %s\n", transition.From, transition.To)
    fmt.Printf("Duration: %v\n", transition.Duration)
    
    if transition.Error != nil {
        fmt.Printf("Error: %v\n", transition.Error)
    }
    
    // Standard metadata keys
    if reason, ok := transition.Metadata["reason"].(string); ok {
        fmt.Printf("Reason: %s\n", reason)
    }
})
```

### Schema Management

```go
import "github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"

// Parse server schema from SHOW BUNDLES response
schemaDef, err := schema.ParseServerSchema(responseJSON)

// Compare local and server schemas
diff := schema.CompareSchemas(localSchema, serverSchema)

if diff.HasChanges {
    for _, change := range diff.BundleChanges {
        switch change.Type {
        case "create":
            cmd := schema.SerializeCreateBundle(change.NewDefinition)
            // Execute cmd
        case "modify":
            cmd := schema.SerializeUpdateBundle(change.BundleName, &change)
            // Execute cmd
        }
    }
}
```

### Migration System

```go
import "github.com/dan-strohschein/syndrdb-drivers/src/golang/migration"

// Create migration client
migClient := migration.NewClient(executor)

// Define migrations (Down commands can be omitted for auto-generation)
migrations := []*migration.Migration{
    {
        ID:   "001_initial_schema",
        Name: "Initial Schema",
        Up: []string{
            `CREATE BUNDLE "users" WITH FIELDS (...)`,
            `CREATE HASH INDEX "idx_users_email" ON BUNDLE "users" WITH FIELDS ("email")`,
        },
        // Down commands omitted - will be auto-generated
        Timestamp: time.Now(),
    },
}

// Auto-generate Down commands for all migrations
generated, err := migClient.GenerateAllDownCommands(migrations)
if err != nil {
    log.Fatal(err)
}
for migID, count := range generated {
    fmt.Printf("Generated %d down commands for %s\n", count, migID)
}

// Validate migrations
validation := migClient.Validate(migrations)
if !validation.Valid {
    for _, conflict := range validation.Conflicts {
        fmt.Printf("Conflict: %s\n", conflict.Message)
    }
}

// Create migration plan
plan, err := migClient.Plan(migrations)

// Apply migrations
err = migClient.Apply(plan)

// Rollback specific migration (auto-generates Down if needed)
err = migClient.Rollback("001_initial_schema", migrations)
```

#### Automatic Down Command Generation

The migration system automatically generates rollback commands from Up commands, eliminating manual reverse operation writing:

**Supported Reversals:**

| Up Command | Generated Down Command |
|------------|----------------------|
| `CREATE BUNDLE "users"` | `DROP BUNDLE "users";` |
| `CREATE HASH INDEX "idx_email" ON BUNDLE "users"` | `DROP INDEX "idx_email";` |
| `CREATE B-INDEX "idx_name" ON BUNDLE "users"` | `DROP INDEX "idx_name";` |
| `UPDATE BUNDLE "users" SET ({ADD "age" = ...})` | `UPDATE BUNDLE "users" SET ({REMOVE "age" = ...})` |
| `UPDATE BUNDLE "users" ADD RELATIONSHIP ("posts" {...})` | `UPDATE BUNDLE "users" REMOVE RELATIONSHIP "posts";` |

**Non-Reversible Commands** (require manual Down commands):
- `DROP BUNDLE` - Cannot recreate without schema
- `DROP INDEX` - Cannot recreate without index definition
- `UPDATE BUNDLE SET REMOVE` - Cannot restore without original field definition
- `UPDATE BUNDLE SET MODIFY` - Cannot revert without original state
- `DELETE FROM` - Cannot restore deleted data
- `REMOVE RELATIONSHIP` - Cannot recreate without relationship definition

**Usage:**

```go
// Check if migration can be auto-rolled back
canRollback := migClient.CanAutoRollback(migration)

// Generate Down for single migration
count, err := migClient.GenerateDownCommands(migration)

// Generate Down for all migrations
generated, err := migClient.GenerateAllDownCommands(migrations)

// Rollback automatically generates Down if missing
err = migClient.Rollback("001_initial_schema", migrations)
```

**Manual Override:**

You can always provide manual Down commands to override auto-generation:

```go
{
    ID: "002_complex_change",
    Up: []string{
        `UPDATE BUNDLE "users" SET ({MODIFY "status" = ...})`,
    },
    Down: []string{
        // Manual rollback for complex change
        `UPDATE BUNDLE "users" SET ({MODIFY "status" = "active", "STRING", TRUE, FALSE, NULL})`,
    },
}
```

**Best Practices:**

1. **Let auto-generation handle simple operations** (CREATE, ADD)
2. **Provide manual Down for destructive operations** (DELETE, DROP, MODIFY)
3. **Test rollbacks** in development before production
4. **Always backup data** before running migrations in production

📖 **[See complete migration documentation →](migration/README.md)**

### Code Generation

#### JSON Schema

```go
import "github.com/dan-strohschein/syndrdb-drivers/src/golang/codegen"

generator := codegen.NewJSONSchemaGenerator()

// Generate single file
schemaJSON, err := generator.GenerateSingle(schemaDef)

// Generate multiple files (one per bundle)
schemas, err := generator.GenerateMulti(schemaDef)
for bundleName, schemaJSON := range schemas {
    // Write to file
}
```

#### GraphQL SDL

```go
generator := codegen.NewGraphQLSchemaGenerator()
sdl, err := generator.Generate(schemaDef)
```

### Response Mapping

```go
import "github.com/dan-strohschein/syndrdb-drivers/src/golang/mapper"

mapper := mapper.NewResponseMapper()

// Map single value
intVal, err := mapper.ToInt(rawValue)
floatVal, err := mapper.ToFloat(rawValue)
boolVal, err := mapper.ToBool(rawValue)
timeVal, err := mapper.ToDateTime(rawValue)

// Map object fields
fieldTypes := map[string]string{
    "id":        "int",
    "name":      "string",
    "createdAt": "datetime",
}
mappedObj, err := mapper.MapObject(rawObject, fieldTypes)
```

## WebAssembly

Build the WASM binary:

```bash
./scripts/build-wasm.sh
```

See `wasm/README.md` for complete WASM usage documentation.

## Building

### Library

```bash
./scripts/build-lib.sh
```

This runs tests, builds the library, and runs `go vet`.

### WASM

```bash
./scripts/build-wasm.sh
```

Generates:
- `wasm/syndrdb.wasm` - WASM binary
- `wasm/syndrdb.wasm.gz` - Compressed for web
- `wasm/wasm_exec.js` - Go WASM runtime

### Version Stamping

Both scripts support version stamping:

```bash
VERSION=v1.0.0 ./scripts/build-lib.sh
VERSION=$(git describe --tags) ./scripts/build-wasm.sh
```

## Error Handling

All errors are JSON-serializable with structured fields:

```go
{
    "code": "CONNECTION_FAILED",
    "type": "CONNECTION_ERROR",
    "message": "failed to connect to localhost:7632",
    "details": {
        "address": "localhost:7632",
        "timeout": 10000
    },
    "cause": {
        "message": "connection refused"
    }
}
```

Error types:
- `ConnectionError` - Connection-related failures
- `ProtocolError` - Protocol violations or malformed responses
- `StateError` - Invalid state for operation
- `MigrationError` - Migration-specific errors

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./client
```

## Performance

- **Single connection**: No connection pool overhead
- **Zero allocations** in hot paths where possible
- **Efficient scanning**: Custom EOT delimiter scanner
- **Small binary**: ~2MB WASM (gzipped: ~600KB)

## Compatibility

- **Go**: 1.24.2 or higher
- **WASM**: Modern browsers and Node.js 14+
- **Protocol**: EOT-delimited TCP (`\x04`)

## Contributing

1. Run tests: `go test ./...`
2. Format code: `go fmt ./...`
3. Run linter: `go vet ./...`
4. Build: `./scripts/build-lib.sh`

## License

[Your License Here]

## Related

- [Node.js Driver](../node/)
- [Python Driver](../python/)
- [C# Driver](../csharp/)
- [Java Driver](../java/)
