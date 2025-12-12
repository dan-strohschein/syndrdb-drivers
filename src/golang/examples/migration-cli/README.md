# Migration CLI Example

A command-line tool demonstrating SyndrDB's migration workflow with automatic rollback generation.

## Features

- 📝 Initialize projects with sample schema
- 🔄 Generate migrations with auto-generated DOWN commands
- ⬆️ Apply pending migrations
- ⬇️ Rollback migrations safely
- 📊 View migration status and history

## Installation

```bash
cd examples/migration-cli
go build -o migration-cli
```

## Usage

### 1. Initialize a New Project

```bash
./migration-cli init --output ./db/schema.json
```

Creates a sample schema file with a `users` bundle.

### 2. Generate a Migration

Edit `./db/schema.json` to add/modify your schema, then:

```bash
./migration-cli generate --name "add_users_table" --schema ./db/schema.json --output ./migrations
```

This generates:
- UP commands from your schema
- DOWN commands automatically (using the rollback generator)
- A timestamped migration file in `./migrations/`

### 3. Apply Migrations

```bash
./migration-cli up --conn "syndrdb://localhost:1776:primary:root:root;" --dir ./migrations
```

Applies all pending migrations to the database.

### 4. Rollback Last Migration

```bash
./migration-cli down --conn "syndrdb://localhost:1776:primary:root:root;" --dir ./migrations
```

Safely rolls back the last applied migration using auto-generated DOWN commands.

### 5. Check Migration Status

```bash
./migration-cli status --conn "syndrdb://localhost:1776:primary:root:root;" --dir ./migrations
```

Shows:
- Total migrations
- Applied migrations with timestamps
- Pending migrations

## Example Workflow

```bash
# 1. Initialize
./migration-cli init

# 2. Edit db/schema.json to add your bundles

# 3. Generate first migration
./migration-cli generate --name "initial_schema"

# 4. Apply to database
./migration-cli up --conn "syndrdb://localhost:1776:primary:root:root;"

# 5. Check status
./migration-cli status --conn "syndrdb://localhost:1776:primary:root:root;"

# Output:
# Migration Status:
#   Total migrations: 1
#   Applied: 1
#   Pending: 0
#
# Applied migrations:
#   ✓ 20251210120000_initial_schema (applied 2025-12-10 12:00:05)

# 6. Make schema changes (add fields, bundles, etc.)

# 7. Generate another migration
./migration-cli generate --name "add_posts_bundle"

# 8. Apply new migration
./migration-cli up --conn "syndrdb://localhost:1776:primary:root:root;"

# 9. Oops, need to rollback
./migration-cli down --conn "syndrdb://localhost:1776:primary:root:root;"
```

## Migration File Format

Generated migrations are JSON files:

```json
{
  "id": "20251210120000_initial_schema",
  "name": "initial_schema",
  "description": "Migration: initial_schema",
  "up": [
    "CREATE BUNDLE \"users\" WITH FIELDS (id int REQUIRED UNIQUE, email string REQUIRED UNIQUE, name string REQUIRED, created_at timestamp REQUIRED);"
  ],
  "down": [
    "DROP BUNDLE \"users\";"
  ],
  "createdAt": "2025-12-10T12:00:00Z"
}
```

## Key Features Demonstrated

### 1. Automatic Rollback Generation

The tool uses SyndrDB's `RollbackGenerator` to automatically create DOWN commands:

```go
rollbackGen := migration.NewRollbackGenerator()
downCommands, err := rollbackGen.GenerateDown(upCommands)
```

This means you never have to manually write reverse operations!

### 2. Migration History Tracking

The migration client tracks:
- Which migrations have been applied
- When they were applied
- The order of application

### 3. Safe Rollbacks

Rollbacks execute DOWN commands in reverse order to safely undo changes.

### 4. Validation

The tool validates migrations before applying them to prevent errors.

## Connection String Format

```
syndrdb://HOST:PORT:DATABASE:USERNAME:PASSWORD;
```

Example:
```
syndrdb://localhost:1776:primary:root:root;
```

## Tips

1. **Always generate migrations** instead of manually creating them - this ensures DOWN commands are generated correctly

2. **Test rollbacks** in a development environment before applying to production

3. **Keep migrations small** - one logical change per migration makes rollbacks safer

4. **Version control** your migrations directory - this is your database history

5. **Run status** regularly to ensure your local migrations match the database state

## What This Demonstrates

- ✅ Real-world migration workflow
- ✅ Automatic rollback generation (killer feature!)
- ✅ Migration history tracking
- ✅ Safe up/down migrations
- ✅ CLI design patterns
- ✅ Error handling
- ✅ Connection management

## Next Steps

After exploring this example, check out:
- `../schema-codegen/` - Generate types from schema
- `../node-graphql-api/` - Node.js GraphQL API using WASM driver
- `../wasm-web/` - Browser-based schema designer
