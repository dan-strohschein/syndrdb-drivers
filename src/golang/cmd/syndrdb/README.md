# SyndrDB CLI

A beautiful, developer-friendly command-line tool for managing SyndrDB databases, migrations, and code generation.

## Installation

```bash
# Build from source
cd src/golang
go build -tags milestone2 -o bin/syndrdb ./cmd/syndrdb

# Add to PATH (optional)
export PATH="$PATH:$(pwd)/bin"
```

## Features

✨ **Beautiful Output** - Colored text, progress indicators, and formatted tables  
🎯 **Smart Defaults** - Automatically reads from environment variables  
🔒 **Safe Operations** - Interactive confirmations and dry-run modes  
📖 **Comprehensive Help** - Detailed examples and usage instructions  
⚡ **Fast & Efficient** - Built with Go for speed and reliability  

## Quick Start

```bash
# 1. Initialize a new project
syndrdb migrate init

# 2. Edit your schema
vim schema.json

# 3. Generate migration
syndrdb migrate generate --name initial_schema

# 4. Preview changes
syndrdb migrate up --dry-run

# 5. Apply migrations
syndrdb migrate up

# 6. Generate TypeScript types
syndrdb codegen generate --format types --language typescript --output types.ts
```

## Commands

### `syndrdb migrate` - Database Migrations

Manage database schema changes with version-controlled migrations.

#### `migrate init`

Initialize a new migration project with directory structure and sample schema.

```bash
syndrdb migrate init
syndrdb migrate init --dir ./db/migrations --schema ./db/schema.json
```

**Creates:**
- `migrations/` directory
- `schema.json` with sample bundle
- `migrations/README.md` with workflow guide

#### `migrate generate`

Generate a new migration from schema changes.

```bash
syndrdb migrate generate --name add_users_table
syndrdb migrate generate --name add_email_index --schema ./db/schema.json
```

**Options:**
- `--name` (required) - Migration name
- `--schema` - Path to schema file (default: `./schema.json`)
- `--dir` - Output directory (default: `./migrations`)

**Output:**
- Creates timestamped migration file (e.g., `20251212164744_add_users_table.json`)
- Includes UP commands (schema changes)
- Auto-generates DOWN commands (rollback) when possible

#### `migrate up`

Apply pending migrations to the database.

```bash
# Preview changes without applying
syndrdb migrate up --dry-run

# Apply all pending migrations
syndrdb migrate up

# Apply specific number of migrations
syndrdb migrate up --steps 1

# Skip confirmation prompt
syndrdb migrate up --force
```

**Options:**
- `--conn` - Connection string (or use `SYNDRDB_CONN` env var)
- `--dir` - Migrations directory (default: `./migrations`)
- `--dry-run` - Show what would be applied without executing
- `--steps` - Number of migrations to apply (0 = all)
- `--force` - Skip confirmation prompt

**Features:**
- Shows migration plan before applying
- Interactive confirmation
- Progress tracking
- Detailed error messages

#### `migrate down`

Rollback the last applied migration.

```bash
# Preview rollback
syndrdb migrate down --dry-run

# Rollback last migration
syndrdb migrate down

# Force rollback without confirmation
syndrdb migrate down --force
```

**Options:**
- `--conn` - Connection string
- `--dir` - Migrations directory
- `--dry-run` - Preview without executing
- `--force` - Skip confirmation

#### `migrate status`

Show the status of all migrations.

```bash
syndrdb migrate status
syndrdb migrate status --conn $SYNDRDB_CONN
```

**Output:**
- Table of all migrations
- Status: pending / applied / failed
- Creation timestamps
- Connection to database optional (shows file status only without connection)

#### `migrate validate`

Validate migration files for common issues.

```bash
syndrdb migrate validate
syndrdb migrate validate --dir ./db/migrations
```

**Checks:**
- File format validity
- Migration structure
- Dependency order
- Checksum integrity
- Common issues (missing DOWN commands, etc.)

### `syndrdb codegen` - Code Generation

Generate type-safe code from your database schema.

#### `codegen fetch-schema`

Fetch the current schema from a running SyndrDB server.

```bash
syndrdb codegen fetch-schema --output ./schema.json
syndrdb codegen fetch-schema --conn $SYNDRDB_CONN
```

**Options:**
- `--conn` - Connection string (or use `SYNDRDB_CONN`)
- `--output` - Output file path (default: `./schema.json`)
- `--format` - Output format: json, yaml (default: `json`)

#### `codegen generate`

Generate code from schema definition.

```bash
# Generate Go types
syndrdb codegen generate --format types --language go --output models/types.go

# Generate TypeScript interfaces
syndrdb codegen generate --format types --language typescript --output types.ts

# Generate JSON Schema
syndrdb codegen generate --format json-schema --output schema.json

# Generate GraphQL Schema
syndrdb codegen generate --format graphql --output schema.graphql
```

**Options:**
- `--schema` - Schema file path (default: `./schema.json`)
- `--output` - Output file path (default: stdout)
- `--format` - Output format:
  - `types` - Type definitions (Go or TypeScript)
  - `json-schema` - JSON Schema specification
  - `graphql` - GraphQL SDL
- `--language` - For types: `go`, `typescript`
- `--package` - Package name for generated code (Go only)

**Generated Code Examples:**

**Go Types:**
```go
package models

import "time"

type Users struct {
    Id        int64     `json:"id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}
```

**TypeScript:**
```typescript
export interface Users {
  id: number;
  email: string;
  name: string;
  created_at: Date;
}
```

### `syndrdb test` - Testing

Test your database connection, schema, and migrations.

#### `test connection`

Test database connection and health.

```bash
syndrdb test connection
syndrdb test connection --conn $SYNDRDB_CONN --verbose
```

**Tests:**
1. Parse connection string
2. Connect to database
3. Ping server
4. Check connection state

**Options:**
- `--conn` - Connection string
- `--verbose` - Show detailed connection info (latency, state, etc.)

#### `test migrations`

Validate all migration files.

```bash
syndrdb test migrations
syndrdb test migrations --dir ./migrations --verbose
```

**Tests:**
1. Check migrations directory exists
2. Load migration files
3. Validate structure
4. Check for common issues

**Options:**
- `--dir` - Migrations directory
- `--verbose` - Show detailed validation info

#### `test all`

Run all test suites.

```bash
syndrdb test all
syndrdb test all --conn $SYNDRDB_CONN --dir ./migrations
```

**Runs:**
- Connection tests
- Migration validation tests

## Environment Variables

Set these environment variables to avoid repeating flags:

```bash
# Database connection string
export SYNDRDB_CONN="syndrdb://localhost:1776/mydb"

# Migrations directory (default: ./migrations)
export SYNDRDB_MIGRATIONS_DIR="./db/migrations"

# Schema file path (default: ./schema.json)
export SYNDRDB_SCHEMA_FILE="./db/schema.json"

# Disable colored output
export NO_COLOR=1
```

## Workflow Examples

### New Project Setup

```bash
# 1. Initialize project
syndrdb migrate init

# 2. Edit schema
cat > schema.json << 'EOF'
{
  "bundles": [
    {
      "name": "users",
      "fields": [
        {"name": "id", "type": "int", "required": true, "unique": true},
        {"name": "email", "type": "string", "required": true, "unique": true},
        {"name": "username", "type": "string", "required": true}
      ],
      "indexes": [
        {"name": "idx_email", "type": "btree", "fields": ["email"]}
      ]
    }
  ]
}
EOF

# 3. Generate migration
syndrdb migrate generate --name initial_schema

# 4. Review migration
cat migrations/*.json

# 5. Apply migration
syndrdb migrate up

# 6. Verify status
syndrdb migrate status
```

### Adding a New Table

```bash
# 1. Update schema.json (add new bundle)
vim schema.json

# 2. Generate migration
syndrdb migrate generate --name add_posts_table

# 3. Preview changes
syndrdb migrate up --dry-run

# 4. Apply migration
syndrdb migrate up

# 5. Update generated types
syndrdb codegen generate --format types --language typescript --output types.ts
```

### CI/CD Integration

```bash
#!/bin/bash
set -e

# Validate migrations
echo "Validating migrations..."
syndrdb test migrations

# Run migrations
echo "Applying migrations..."
syndrdb migrate up --force

# Generate types
echo "Generating types..."
syndrdb codegen generate --format types --language go --output models/db.go
```

### Development Workflow

```bash
# Start development
export SYNDRDB_CONN="syndrdb://localhost:1776/dev"

# Check connection
syndrdb test connection

# Apply migrations
syndrdb migrate up

# Generate types for frontend
syndrdb codegen generate --format types --language typescript --output ../frontend/src/types/db.ts

# Generate GraphQL schema
syndrdb codegen generate --format graphql --output schema.graphql
```

## Tips & Best Practices

### Migration Best Practices

1. **Use Descriptive Names**: `add_users_table` not `migration_001`
2. **One Change Per Migration**: Keep migrations focused
3. **Test Rollbacks**: Always include DOWN commands
4. **Version Control**: Commit migrations with code changes
5. **Preview First**: Use `--dry-run` before applying

### Schema Design Tips

1. **Define Required Fields**: Mark critical fields as `required: true`
2. **Add Indexes**: Index frequently queried fields
3. **Use Unique Constraints**: Prevent duplicates
4. **Timestamp Fields**: Include `created_at` and `updated_at`

### Code Generation Tips

1. **Regenerate Often**: Run after schema changes
2. **Check Into Git**: Generated types should be versioned
3. **Use Type Safety**: Leverage generated types in your app
4. **Format Output**: Use `--output` to write to files

## Troubleshooting

### Connection Issues

```bash
# Test connection first
syndrdb test connection --verbose

# Check environment variable
echo $SYNDRDB_CONN

# Use explicit connection string
syndrdb migrate up --conn "syndrdb://localhost:1776/mydb"
```

### Migration Conflicts

```bash
# Validate migrations
syndrdb migrate validate

# Check status
syndrdb migrate status

# Review migration files
ls -la migrations/
```

### Code Generation Errors

```bash
# Validate schema file
cat schema.json | jq '.'

# Fetch latest schema from server
syndrdb codegen fetch-schema --output schema.json

# Try different formats
syndrdb codegen generate --format types --language typescript
```

## Output Examples

### Successful Migration

```
Apply Migrations
────────────────────────────────────────

ℹ Pending migrations: 2
  1. add_users_table [pending]
     001_add_users (2 up, 1 down)
  2. add_posts_table [pending]
     002_add_posts (3 up, 2 down)

Apply 2 migration(s)? [y/N]: y

Applying Migrations
────────────────────────────────────────
✓ All migrations applied successfully!
```

### Migration Status

```
Migration Status
────────────────────────────────────────

ID                Name              Status   Created
────────────────  ────────────────  ───────  ────────────────
initial_schema    initial_schema    applied  2025-12-12 16:47
add_posts         add_posts_table   pending  2025-12-12 17:15

ℹ Total migrations: 2
ℹ Connected to database - showing actual status
```

## Version

```bash
syndrdb version
# Output: syndrdb v1.0.0
```

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/dan-strohschein/syndrdb-drivers).
