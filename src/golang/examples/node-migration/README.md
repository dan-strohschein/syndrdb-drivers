# Node.js Migration Script

A lightweight CI/CD migration script using the SyndrDB WASM driver in Node.js. Perfect for deployment pipelines, automated database setup, and schema synchronization.

## Features

- 🚀 **Automated Migrations** - Compare local schema with server and apply changes
- 🔍 **Dry Run Mode** - Preview changes without applying them
- ⚡ **Fast Execution** - WASM driver for optimal performance
- 📦 **Zero Dependencies** - Only uses SyndrDB WASM driver
- 🔧 **CI/CD Ready** - Perfect for deployment scripts
- 🛡️ **Safe** - Verifies migration after applying

## Installation

```bash
cd examples/node-migration
npm install
```

## Usage

### Basic Migration

```bash
# Apply migration
node migrate.js

# or use npm script
npm run migrate
```

### Dry Run (Preview Changes)

```bash
# Preview what would be executed
DRY_RUN=true node migrate.js

# or use npm script
npm run migrate:dry-run
```

### Environment Variables

```bash
# Custom connection string
SYNDR_CONN="syndrdb://localhost:1776:mydb:user:pass;" node migrate.js

# Custom schema file
SCHEMA_FILE="./my-schema.json" node migrate.js

# Combine options
SYNDR_CONN="syndrdb://prod:1776:prod:user:pass;" \\
SCHEMA_FILE="./prod-schema.json" \\
DRY_RUN=true \\
node migrate.js
```

## Schema File Format

The script reads from `schema.json` (or custom path via `SCHEMA_FILE`):

```json
{
  "bundles": [
    {
      "name": "users",
      "fields": [
        { "name": "id", "type": "int", "required": true, "unique": true },
        { "name": "email", "type": "string", "required": true, "unique": true },
        { "name": "name", "type": "string", "required": true }
      ],
      "indexes": [
        { "name": "idx_email", "type": "hash", "fields": ["email"] }
      ],
      "relationships": []
    }
  ]
}
```

## Example Output

```
🔧 SyndrDB Migration Script
============================

📦 Loading WASM driver...
✓ WASM driver loaded

🔌 Initializing SyndrDB client...
✓ Client initialized

🌐 Connecting to database...
✓ Connected successfully

📄 Loading schema from ./schema.json...
✓ Loaded schema with 2 bundle(s)

🔍 Fetching current schema from server...
✓ Server has 0 bundle(s)

🔄 Comparing schemas...
📋 Found 2 change(s):

   1. Create bundle "users"
   2. Create bundle "posts"

⚙️  Generating migration commands...
✓ Generated 4 command(s)

🚀 Applying migration...
   Executing: CREATE BUNDLE "users" WITH FIELDS (id INT REQUIRED UNI...
   Executing: CREATE INDEX "idx_email" ON "users" (email) TYPE HASH;...
   Executing: CREATE BUNDLE "posts" WITH FIELDS (id INT REQUIRED UNI...
✓ Successfully applied 4 command(s)

✅ Verifying migration...
✓ Database now has 2 bundle(s)

✅ Migration complete!
```

## Dry Run Output

```
🔧 SyndrDB Migration Script
============================

... (connection steps) ...

🔄 Comparing schemas...
📋 Found 2 change(s):

   1. Create bundle "users"
   2. Create bundle "posts"

⚙️  Generating migration commands...
✓ Generated 4 command(s)

🔍 DRY RUN MODE - Commands to be executed:

   1. CREATE BUNDLE "users" WITH FIELDS (id INT REQUIRED UNIQUE, email STRING REQUIRED UNIQUE, ...);
   2. CREATE INDEX "idx_email" ON "users" (email) TYPE HASH;
   3. CREATE BUNDLE "posts" WITH FIELDS (id INT REQUIRED UNIQUE, user_id INT REQUIRED, ...);
   4. CREATE INDEX "idx_user_id" ON "posts" (user_id) TYPE BTREE;
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Deploy Database Schema

on:
  push:
    branches: [main]

jobs:
  migrate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Run migration
        env:
          SYNDR_CONN: \${{ secrets.SYNDR_CONN }}
        run: |
          cd examples/node-migration
          npm install
          node migrate.js
```

### GitLab CI

```yaml
migrate:
  stage: deploy
  image: node:18
  script:
    - cd examples/node-migration
    - npm install
    - node migrate.js
  variables:
    SYNDR_CONN: $SYNDR_CONN
  only:
    - main
```

### Docker

```dockerfile
FROM node:18-alpine

WORKDIR /app

COPY package*.json ./
COPY migrate.js schema.json wasm_exec.js ./
COPY ../../wasm/syndrdb.wasm ../../wasm/

RUN npm install --production

CMD ["node", "migrate.js"]
```

```bash
docker build -t syndr-migrate .
docker run -e SYNDR_CONN="syndrdb://..." syndr-migrate
```

## How It Works

1. **Load WASM** - Initializes the SyndrDB WASM driver
2. **Connect** - Establishes connection to the database
3. **Fetch Current Schema** - Queries `SHOW BUNDLES` to get current state
4. **Compare** - Diffs local schema file against server schema
5. **Generate Commands** - Creates SQL commands for changes
6. **Apply** - Executes commands (unless dry-run)
7. **Verify** - Confirms migration was successful

## Production Considerations

### Safety

✅ **Always test in staging first**  
✅ **Use dry-run before applying to production**  
✅ **Backup your database before running migrations**  
✅ **Review generated commands carefully**

### Error Handling

The script exits with error codes:
- `0` - Success (or no changes needed)
- `1` - Connection failed, schema invalid, or migration failed

### Extending

You can extend this script to:
- Handle field modifications (not just new bundles)
- Support rollback generation
- Save migration history
- Send notifications on completion
- Integrate with your logging system

## Example Use Cases

### 1. Initial Database Setup

```bash
# First deployment
SYNDR_CONN="syndrdb://prod:1776:myapp:admin:pass;" node migrate.js
```

### 2. Adding New Tables

Edit `schema.json` to add new bundles, then:

```bash
# Preview changes
DRY_RUN=true node migrate.js

# Apply changes
node migrate.js
```

### 3. Multi-Environment Deployment

```bash
# Development
SYNDR_CONN="syndrdb://dev:1776:myapp:root:root;" \\
SCHEMA_FILE="./schema.dev.json" \\
node migrate.js

# Staging
SYNDR_CONN="syndrdb://staging:1776:myapp:user:pass;" \\
SCHEMA_FILE="./schema.staging.json" \\
node migrate.js

# Production (dry-run first!)
SYNDR_CONN="syndrdb://prod:1776:myapp:user:pass;" \\
SCHEMA_FILE="./schema.prod.json" \\
DRY_RUN=true \\
node migrate.js
```

## What This Demonstrates

- ✅ WASM driver in Node.js environment
- ✅ Automated schema management
- ✅ CI/CD integration patterns
- ✅ Safe migration practices
- ✅ Environment configuration
- ✅ Error handling
- ✅ Verification steps

## Limitations

This example script:
- Only detects **new bundles** (not field changes)
- Does not handle **rollbacks**
- Does not maintain **migration history**
- Uses simplified schema comparison

For production, consider using the full migration CLI (`../migration-cli/`) which has:
- Complete schema diffing
- Automatic rollback generation
- Migration history tracking
- Validation and conflict resolution

## Next Steps

- Add support for field modifications
- Implement rollback generation
- Add migration history tracking
- Integrate with monitoring tools
- Add webhook notifications
- Support for multiple databases

## Related Examples

- `../migration-cli/` - Full-featured Go migration tool
- `../schema-codegen/` - Generate types from schema
- `../node-graphql-api/` - GraphQL API example
- `../wasm-web/` - Browser-based schema designer
