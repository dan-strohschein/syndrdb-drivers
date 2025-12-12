# Migration Package

Comprehensive database migration system with automatic rollback generation for SyndrDB.

## Overview

The migration package provides:
- ✅ **Automatic Down Command Generation** - Intelligent reverse operation generation
- ✅ **Checksum Validation** - SHA-256 checksums prevent migration tampering
- ✅ **Dependency Tracking** - Ensure migrations apply in correct order
- ✅ **Conflict Detection** - Detect checksum mismatches, dependency issues, and ordering problems
- ✅ **Migration History** - Track applied migrations with timestamps and execution times
- ✅ **Safe Rollback** - Prevent rollback if other migrations depend on it

## Quick Start

```go
import "github.com/dan-strohschein/syndrdb-drivers/src/golang/migration"

// Create migration client with executor
migClient := migration.NewClient(executor)

// Define migration (Down commands optional)
migrations := []*migration.Migration{
    {
        ID:   "001_create_users",
        Name: "Create users bundle",
        Up: []string{
            `CREATE BUNDLE "users" WITH FIELDS (
                {"id", "INT", TRUE, TRUE, NULL},
                {"email", "STRING", TRUE, TRUE, NULL},
                {"name", "STRING", TRUE, FALSE, NULL}
            );`,
            `CREATE HASH INDEX "idx_users_email" ON BUNDLE "users" WITH FIELDS ("email");`,
        },
        Timestamp: time.Now(),
    },
}

// Auto-generate Down commands
migClient.GenerateAllDownCommands(migrations)

// Apply migrations
plan, _ := migClient.Plan(migrations)
migClient.Apply(plan)

// Rollback if needed
migClient.Rollback("001_create_users", migrations)
```

## Automatic Down Command Generation

### How It Works

The `RollbackGenerator` analyzes each Up command and generates the corresponding reverse operation:

1. **Parse Command Type** - Identify operation (CREATE, UPDATE, ADD, etc.)
2. **Extract Entities** - Extract bundle names, field names, index names, etc.
3. **Generate Reverse** - Create the inverse operation
4. **Validate** - Ensure reversal is safe and complete

### Supported Reversals

#### CREATE BUNDLE → DROP BUNDLE

**Up Command:**
```sql
CREATE BUNDLE "products" WITH FIELDS (
    {"id", "INT", TRUE, TRUE, NULL},
    {"name", "STRING", TRUE, FALSE, NULL},
    {"price", "FLOAT", TRUE, FALSE, NULL}
);
```

**Generated Down:**
```sql
DROP BUNDLE "products";
```

#### CREATE INDEX → DROP INDEX

**Up Command:**
```sql
CREATE HASH INDEX "idx_products_name" ON BUNDLE "products" WITH FIELDS ("name");
```

**Generated Down:**
```sql
DROP INDEX "idx_products_name";
```

**Also works for B-Tree indexes:**
```sql
CREATE B-INDEX "idx_products_price" ON BUNDLE "products" WITH FIELDS ("price");
-- Down: DROP INDEX "idx_products_price";
```

#### UPDATE BUNDLE SET ADD → UPDATE BUNDLE SET REMOVE

**Up Command:**
```sql
UPDATE BUNDLE "users"
SET (
    {ADD "age" = "age", "INT", FALSE, FALSE, NULL},
    {ADD "city" = "city", "STRING", FALSE, FALSE, NULL}
);
```

**Generated Down:**
```sql
UPDATE BUNDLE "users"
SET (
    {REMOVE "city" = "", "", FALSE, FALSE, NULL},
    {REMOVE "age" = "", "", FALSE, FALSE, NULL}
);
```

*Note: Down commands are generated in reverse order*

#### ADD RELATIONSHIP → REMOVE RELATIONSHIP

**Up Command:**
```sql
UPDATE BUNDLE "users" ADD RELATIONSHIP ("posts" {
    "1toMany",
    "users",
    "id",
    "posts",
    "user_id"
});
```

**Generated Down:**
```sql
UPDATE BUNDLE "users" REMOVE RELATIONSHIP "posts";
```

### Non-Reversible Commands

These commands **cannot** be automatically reversed and require manual Down commands:

#### DROP BUNDLE
**Why:** Cannot recreate bundle without knowing the original schema.

**Manual Down Required:**
```go
{
    Up: []string{`DROP BUNDLE "old_users";`},
    Down: []string{
        `CREATE BUNDLE "old_users" WITH FIELDS (
            {"id", "INT", TRUE, TRUE, NULL},
            {"name", "STRING", TRUE, FALSE, NULL}
        );`,
    },
}
```

#### DROP INDEX
**Why:** Cannot recreate index without knowing the type and fields.

**Manual Down Required:**
```go
{
    Up: []string{`DROP INDEX "old_idx";`},
    Down: []string{
        `CREATE HASH INDEX "old_idx" ON BUNDLE "users" WITH FIELDS ("email");`,
    },
}
```

#### UPDATE BUNDLE SET REMOVE
**Why:** Cannot restore field without knowing its original definition.

**Manual Down Required:**
```go
{
    Up: []string{
        `UPDATE BUNDLE "users" SET ({REMOVE "deprecated_field" = "", "", FALSE, FALSE, NULL})`,
    },
    Down: []string{
        `UPDATE BUNDLE "users" SET ({ADD "deprecated_field" = "deprecated_field", "STRING", FALSE, FALSE, NULL})`,
    },
}
```

#### UPDATE BUNDLE SET MODIFY
**Why:** Cannot revert modification without knowing the original state.

**Manual Down Required:**
```go
{
    Up: []string{
        `UPDATE BUNDLE "users" SET ({MODIFY "status" = "status", "STRING", TRUE, FALSE, "active"})`,
    },
    Down: []string{
        `UPDATE BUNDLE "users" SET ({MODIFY "status" = "status", "STRING", FALSE, FALSE, NULL})`,
    },
}
```

#### DELETE FROM
**Why:** Cannot restore deleted data.

**Manual Down Required:**
```go
{
    Up: []string{`DELETE FROM users WHERE inactive = TRUE;`},
    Down: []string{
        // Restore from backup or skip rollback
        `-- Cannot automatically restore deleted data`,
    },
}
```

## API Reference

### Migration Client

#### NewClient(executor MigrationExecutor)
Creates a new migration client.

```go
migClient := migration.NewClient(executor)
```

#### GenerateDownCommands(migration *Migration) (int, error)
Generates Down commands for a single migration if they don't exist.

```go
count, err := migClient.GenerateDownCommands(migration)
// Returns: number of Down commands generated, or error
```

#### GenerateAllDownCommands(migrations []*Migration) (map[string]int, error)
Generates Down commands for all migrations that need them.

```go
generated, err := migClient.GenerateAllDownCommands(migrations)
// Returns: map[migrationID]commandCount
```

#### CanAutoRollback(migration *Migration) bool
Checks if a migration can be automatically rolled back.

```go
if migClient.CanAutoRollback(migration) {
    fmt.Println("Can safely rollback")
}
```

#### Plan(migrations []*Migration) (*MigrationPlan, error)
Creates an execution plan for pending migrations.

```go
plan, err := migClient.Plan(migrations)
```

#### Apply(plan *MigrationPlan) error
Executes a migration plan.

```go
err := migClient.Apply(plan)
```

#### Rollback(migrationID string, allMigrations []*Migration) error
Rolls back a specific migration. Auto-generates Down commands if missing.

```go
err := migClient.Rollback("001_create_users", migrations)
```

#### Validate(migrations []*Migration) *ValidationResult
Validates migrations without executing them.

```go
result := migClient.Validate(migrations)
if !result.Valid {
    for _, conflict := range result.Conflicts {
        fmt.Println(conflict.Message)
    }
}
```

### Rollback Generator

#### NewRollbackGenerator()
Creates a new rollback generator.

```go
generator := migration.NewRollbackGenerator()
```

#### GenerateDown(upCommands []string) ([]string, error)
Generates Down commands from Up commands.

```go
downCommands, err := generator.GenerateDown(upCommands)
```

#### CanGenerateDown(upCommand string) bool
Checks if a command can be automatically reversed.

```go
if generator.CanGenerateDown(upCommand) {
    // Command is reversible
}
```

#### ValidateDownCommands(upCommands, downCommands []string) error
Validates that Down commands are proper reverses of Up commands.

```go
err := generator.ValidateDownCommands(upCommands, downCommands)
```

## Migration Workflow

### 1. Create Migration

```go
migration := &migration.Migration{
    ID:   "002_add_products",
    Name: "Add products bundle",
    Up: []string{
        `CREATE BUNDLE "products" WITH FIELDS (...)`,
        `CREATE INDEX "idx_products_name" ON BUNDLE "products" WITH FIELDS ("name")`,
    },
    Dependencies: []string{"001_create_users"},
    Timestamp:    time.Now(),
}
```

### 2. Auto-Generate Down

```go
count, err := migClient.GenerateDownCommands(migration)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Generated %d down commands\n", count)
```

### 3. Validate

```go
result := migClient.Validate([]*migration.Migration{migration})
if !result.Valid {
    for _, conflict := range result.Conflicts {
        log.Printf("Conflict: %s\n", conflict.Message)
    }
    return
}
```

### 4. Apply

```go
plan, err := migClient.Plan([]*migration.Migration{migration})
if err != nil {
    log.Fatal(err)
}

err = migClient.Apply(plan)
if err != nil {
    log.Fatal(err)
}
```

### 5. Rollback (if needed)

```go
err := migClient.Rollback("002_add_products", allMigrations)
if err != nil {
    log.Fatal(err)
}
```

## Best Practices

### 1. Use Auto-Generation for Simple Operations

✅ **Good:**
```go
{
    Up: []string{
        `CREATE BUNDLE "users" WITH FIELDS (...)`,
        `CREATE INDEX "idx_users_email" ON BUNDLE "users" WITH FIELDS ("email")`,
    },
    // Down auto-generated
}
```

### 2. Provide Manual Down for Complex Changes

✅ **Good:**
```go
{
    Up: []string{
        `UPDATE BUNDLE "users" SET ({MODIFY "role" = "role", "STRING", TRUE, FALSE, "user"})`,
    },
    Down: []string{
        `UPDATE BUNDLE "users" SET ({MODIFY "role" = "role", "STRING", FALSE, FALSE, NULL})`,
    },
}
```

### 3. Test Rollbacks in Development

```go
// Apply migration
migClient.Apply(plan)

// Verify it works
// ... run tests ...

// Test rollback
migClient.Rollback("001_test", migrations)

// Verify rollback worked
// ... run tests ...
```

### 4. Use Dependencies for Ordering

```go
{
    ID:   "002_add_posts",
    Dependencies: []string{"001_create_users"}, // Ensure users exist first
    Up: []string{
        `CREATE BUNDLE "posts" WITH FIELDS (...)`,
    },
}
```

### 5. Include Checksums in Version Control

```go
// Migration history tracks checksums
// Prevents accidental modification of applied migrations
record, _ := migClient.GetMigrationRecord("001_create_users")
fmt.Println(record.Checksum) // SHA-256 hash
```

## Error Handling

All migration operations return structured errors:

```go
err := migClient.Apply(plan)
if err != nil {
    if migErr, ok := err.(*migration.MigrationError); ok {
        fmt.Printf("Code: %s\n", migErr.Code)
        fmt.Printf("Type: %s\n", migErr.Type)
        fmt.Printf("Message: %s\n", migErr.Message)
        fmt.Printf("Details: %v\n", migErr.Details)
    }
}
```

**Error Codes:**
- `MIGRATION_NOT_FOUND` - Migration ID doesn't exist
- `MIGRATION_FAILED` - Execution failed
- `CHECKSUM_MISMATCH` - Migration was modified after being applied
- `DEPENDENCY_NOT_MET` - Required dependencies not satisfied
- `ROLLBACK_NOT_SUPPORTED` - Cannot rollback (missing Down commands)
- `MIGRATION_CONFLICT` - Validation detected conflicts

## Performance

- **Checksum Calculation:** SHA-256 hashing is fast (~1ms per migration)
- **Auto-Generation:** Regex parsing adds minimal overhead (~0.1ms per command)
- **History Tracking:** In-memory with JSON serialization
- **Validation:** O(n) where n = number of migrations

## Examples

See `examples/migration-workflow.go` for complete working examples.

## Related

- [Schema Package](../schema/) - Schema management and comparison
- [Client Package](../client/) - Database connection and execution
- [Main README](../README.md) - Full driver documentation

---

## Migration File Persistence

### Overview
Migrations can be saved to and loaded from timestamped JSON files for version control and collaboration.

### File Format
\`\`\`json
{
  "formatVersion": "1.0",
  "migration": {
    "id": "001_create_users",
    "name": "Create users bundle",
    "up": ["CREATE BUNDLE \"users\" WITH FIELDS (...)"],
    "down": ["DROP BUNDLE \"users\";"],
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
\`\`\`

**File Naming**: \`YYYYMMDDHHMMSS_migration_id.json\`

**Permissions**: Migration files \`0644\`, directory \`0755\`

### Usage
\`\`\`go
// Initialize directory
migration.InitMigrationDirectory("./migrations")

// Save/load migrations
filePath, _ := migration.WriteMigrationFile(mig, "./migrations")
mig, _ := migration.ReadMigrationFile(filePath)
migrations, _ := migration.ListMigrationFiles("./migrations")

// Apply from directory
client.ApplyFromDirectory("./migrations")
\`\`\`

---

## Migration Locking

### Configuration
**Default Timeout**: 1 hour  
**CI/CD Recommended**: 5-10 minutes

\`\`\`bash
export SYNDR_LOCK_TIMEOUT="5m"  # CI/CD
export SYNDR_LOCK_TIMEOUT="1h"  # Local
\`\`\`

### Usage
\`\`\`go
client.WithLocking("./migrations", time.Hour)
client.WithLockRetry(3, 2*time.Second)
result, _ := client.Apply(migrations)
\`\`\`

**Lock File**: \`.syndr_migration.lock\` (permissions: \`0600\`)  
**Metadata**: Holder, Hostname, PID, Timestamp, Note

See \`lock.go\` for details on retry logic, stale detection, and distributed coordination.

---

## Dry-Run Mode

\`\`\`go
plan, _ := client.Preview(migrations)
preview := client.FormatPreview(plan)
fmt.Println(preview)
\`\`\`

---

## WASM / JavaScript

### Node.js
\`\`\`javascript
await SyndrDB.createMigrationClient();
const plan = await SyndrDB.planMigration(migrations);
await SyndrDB.applyMigration(plan);

// Node.js only
await SyndrDB.saveMigrationFile(migration, "./migrations");
await SyndrDB.acquireMigrationLock("./migrations", 3600000);
\`\`\`

### Browser Detection
File/lock operations return error in browser. Check runtime:
\`\`\`javascript
const info = await SyndrDB.getEnvironmentInfo();
// { runtime: "nodejs"|"browser", fileSystemSupport: bool }
\`\`\`

---

## Documentation

- [Implementation Summary](./IMPLEMENTATION_SUMMARY.md) - Technical details and test results

