# SyndrDB Go Driver Examples

Complete collection of example applications demonstrating the SyndrDB Go driver in different contexts.

## Overview

| Example | Language | Type | Use Case |
|---------|----------|------|----------|
| [migration-cli](#1-migration-cli) | Go | CLI Tool | Database migration management |
| [schema-codegen](#2-schema-codegen) | Go | CLI Tool | Code generation from schemas |
| [node-graphql-api](#3-nodejs-graphql-api) | Node.js + WASM | API Server | Production GraphQL API |
| [wasm-web](#4-wasm-web-designer) | Browser + WASM | Web App | Interactive schema design |
| [node-migration](#5-nodejs-migration-script) | Node.js + WASM | Script | CI/CD automation |

## 1. Migration CLI

**Path:** `examples/migration-cli/`  
**Language:** Go  
**Purpose:** Full-featured migration management tool

### Features

- ✅ Automatic Down command generation
- ✅ Migration history tracking
- ✅ Safe rollbacks with validation
- ✅ JSON migration files
- ✅ Checksum verification
- ✅ Dependency tracking

### Quick Start

```bash
cd examples/migration-cli
go build -o syndr-migrate

# Initialize with sample schema
./syndr-migrate init

# Generate new migration
./syndr-migrate generate -name "add_users_table"

# Apply migrations
./syndr-migrate up

# Check status
./syndr-migrate status

# Rollback last migration
./syndr-migrate down
```

### Example Output

```
Migration Status
================

Applied Migrations:
✓ 001_initial_schema.json
  Applied: 2024-01-15 10:30:45
  Checksum: abc123...

Pending Migrations:
○ 002_add_posts_table.json
  Checksum: def456...

Total: 1 applied, 1 pending
```

### Use Cases

- Development database setup
- Production schema changes
- Team collaboration with version control
- Safe rollback capabilities

[Full Documentation →](./migration-cli/README.md)

---

## 2. Schema Codegen

**Path:** `examples/schema-codegen/`  
**Language:** Go  
**Purpose:** Generate code from SyndrDB schemas

### Features

- ✅ TypeScript interface generation
- ✅ JSON Schema (single/multi-file)
- ✅ GraphQL SDL generation
- ✅ Fetch from server or file
- ✅ Multiple output formats

### Quick Start

```bash
cd examples/schema-codegen
go build -o syndr-codegen

# Fetch schema from server
./syndr-codegen fetch -conn "syndrdb://localhost:1776:primary:root:root;" -output schema.json

# Generate TypeScript types
./syndr-codegen typescript -schema schema.json -output types.ts

# Generate JSON Schema
./syndr-codegen json-schema -schema schema.json -mode multi -output ./schemas/

# Generate GraphQL
./syndr-codegen graphql -schema schema.json -output schema.graphql
```

### Generated Output Examples

**TypeScript:**
```typescript
export interface User {
  id: number;
  email: string;
  username: string;
  created_at: Date;
  updated_at?: Date;
}
```

**GraphQL:**
```graphql
type User {
  id: Int!
  email: String!
  username: String!
  created_at: String!
  updated_at: String
}

type Query {
  users: [User!]!
  user(id: Int!): User
}
```

### Use Cases

- Frontend type generation
- GraphQL API schema
- JSON Schema validation
- Documentation generation
- CI/CD pipelines

[Full Documentation →](./schema-codegen/README.md)

---

## 3. Node.js GraphQL API

**Path:** `examples/node-graphql-api/`  
**Language:** Node.js + WASM  
**Purpose:** Production-ready GraphQL API

### Features

- ✅ Complete GraphQL schema
- ✅ Built-in GraphQL Playground
- ✅ WASM driver integration
- ✅ Health checks
- ✅ Graceful shutdown
- ✅ CORS enabled
- ✅ Error handling

### Quick Start

```bash
cd examples/node-graphql-api
bash setup.sh  # Copies WASM files and installs dependencies
node server.js
```

Server starts on http://localhost:4000

### Example Queries

```graphql
# Get all todos
query {
  todos {
    id
    title
    completed
    created_at
  }
}

# Create todo
mutation {
  createTodo(title: "Learn SyndrDB", description: "Complete tutorial") {
    id
    title
    completed
  }
}

# Update todo
mutation {
  updateTodo(id: 1, completed: true) {
    id
    completed
  }
}
```

### Architecture

```
HTTP Request → Express Router → GraphQL Parser
                                      ↓
                              Resolver Functions
                                      ↓
                              WASM Driver (SyndrDB.query/mutate)
                                      ↓
                              SyndrDB Server
```

### Use Cases

- Production APIs
- Full-stack applications
- Microservices
- Real-time data access
- Mobile app backends

[Full Documentation →](./node-graphql-api/README.md)

---

## 4. WASM Web Designer

**Path:** `examples/wasm-web/`  
**Language:** Browser + WASM  
**Purpose:** Interactive schema designer

### Features

- ✅ Beautiful modern UI
- ✅ Real-time code generation
- ✅ TypeScript + JSON Schema + GraphQL
- ✅ Copy to clipboard
- ✅ Download generated files
- ✅ Fully offline capable
- ✅ Pre-loaded examples

### Quick Start

```bash
cd examples/wasm-web
python3 -m http.server 8080
# Open http://localhost:8080
```

Or just open `index.html` directly in your browser!

### Features Demo

1. **Edit Schema** - Type or paste JSON schema
2. **Auto-Generate** - Code updates as you type (1s debounce)
3. **Switch Tabs** - View JSON Schema, GraphQL, or TypeScript
4. **Copy/Download** - Export generated code
5. **Load Example** - Pre-loaded users/posts schema

### Screenshot

```
┌────────────────────────────────────────────────────┐
│  🎨 SyndrDB Schema Designer         ✓ Driver v1.0  │
├────────────────────────────────────────────────────┤
│                                                     │
│  Schema Editor          │  Generated Code          │
│  ┌─────────────────┐   │  [JSON] [GraphQL] [TS]  │
│  │ {               │   │  ┌────────────────────┐  │
│  │   "bundles": [  │   │  │ type User {        │  │
│  │     {           │   │  │   id: Int!         │  │
│  │       "name":   │   │  │   email: String!   │  │
│  │       ...       │   │  │   ...              │  │
│  └─────────────────┘   │  └────────────────────┘  │
│                                                     │
│  [Load Example] [Generate] [Clear]  [Copy] [↓]    │
└────────────────────────────────────────────────────┘
```

### Use Cases

- Schema prototyping
- Learning SyndrDB
- Quick code generation
- Documentation
- Presentations
- Educational demos

[Full Documentation →](./wasm-web/README.md)

---

## 5. Node.js Migration Script

**Path:** `examples/node-migration/`  
**Language:** Node.js + WASM  
**Purpose:** CI/CD migration automation

### Features

- ✅ Schema comparison
- ✅ Automatic migration generation
- ✅ Dry-run mode
- ✅ Environment variables
- ✅ CI/CD ready
- ✅ Verification steps

### Quick Start

```bash
cd examples/node-migration
npm install

# Run migration
node migrate.js

# Dry run (preview only)
DRY_RUN=true node migrate.js

# Custom connection
SYNDR_CONN="syndrdb://prod:1776:mydb:user:pass;" node migrate.js
```

### Example Output

```
🔧 SyndrDB Migration Script
============================

📦 Loading WASM driver...
✓ WASM driver loaded

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

🚀 Applying migration...
   Executing: CREATE BUNDLE "users" WITH FIELDS...
✓ Successfully applied 4 command(s)

✅ Migration complete!
```

### CI/CD Examples

**GitHub Actions:**
```yaml
- name: Migrate Database
  env:
    SYNDR_CONN: ${{ secrets.SYNDR_CONN }}
  run: |
    cd examples/node-migration
    npm install
    node migrate.js
```

**GitLab CI:**
```yaml
migrate:
  script:
    - cd examples/node-migration
    - npm install
    - node migrate.js
```

**Docker:**
```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY . .
RUN npm install
CMD ["node", "migrate.js"]
```

### Use Cases

- Automated deployments
- CI/CD pipelines
- Infrastructure as code
- Multi-environment management
- Database initialization

[Full Documentation →](./node-migration/README.md)

---

## Comparison Matrix

| Feature | migration-cli | schema-codegen | node-graphql-api | wasm-web | node-migration |
|---------|---------------|----------------|------------------|----------|----------------|
| **Platform** | Go Binary | Go Binary | Node.js | Browser | Node.js |
| **Database Connection** | ✅ Yes | ✅ Yes | ✅ Yes | ❌ No | ✅ Yes |
| **Code Generation** | ❌ No | ✅ Yes | ❌ No | ✅ Yes | ❌ No |
| **Migration Management** | ✅ Full | ❌ No | ❌ No | ❌ No | ✅ Basic |
| **GraphQL** | ❌ No | ✅ SDL Only | ✅ Full API | ✅ SDL Only | ❌ No |
| **TypeScript** | ❌ No | ✅ Yes | ❌ No | ✅ Yes | ❌ No |
| **Interactive UI** | ❌ CLI | ❌ CLI | ✅ Playground | ✅ Full | ❌ CLI |
| **CI/CD Ready** | ✅ Yes | ✅ Yes | ❌ Server | ❌ Manual | ✅ Yes |
| **Offline Capable** | ❌ No | ❌ No | ❌ No | ✅ Yes | ❌ No |
| **Production Ready** | ✅ Yes | ✅ Yes | ✅ Yes | ⚠️ Demo | ✅ Yes |

## Getting Started

### Prerequisites

**All Examples:**
- SyndrDB server running (except wasm-web)
- Connection string: `syndrdb://host:port:database:user:pass;`

**Go Examples (migration-cli, schema-codegen):**
- Go 1.24 or higher
- No external dependencies

**Node.js Examples (node-graphql-api, node-migration):**
- Node.js 18 or higher
- WASM driver files (copied automatically)

**Browser Example (wasm-web):**
- Modern browser with WASM support
- HTTP server (for loading WASM files)

### Installation

```bash
# Clone repository
git clone <repo-url>
cd src/golang/examples

# Build Go examples
cd migration-cli && go build && cd ..
cd schema-codegen && go build && cd ..

# Install Node.js examples
cd node-graphql-api && npm install && cd ..
cd node-migration && npm install && cd ..

# WASM web example (no installation needed)
cd wasm-web
python3 -m http.server 8080
```

## Common Workflows

### Workflow 1: New Project Setup

```bash
# 1. Design schema in browser
cd examples/wasm-web && open index.html

# 2. Download schema.json

# 3. Generate types for frontend
cd examples/schema-codegen
./syndr-codegen typescript -schema schema.json -output types.ts

# 4. Create initial migration
cd examples/migration-cli
./syndr-migrate generate -name "initial_schema"

# 5. Apply to database
./syndr-migrate up
```

### Workflow 2: Schema Changes

```bash
# 1. Modify schema.json

# 2. Generate new migration
cd examples/migration-cli
./syndr-migrate generate -name "add_new_fields"

# 3. Test in development
./syndr-migrate up

# 4. Update types
cd examples/schema-codegen
./syndr-codegen typescript -schema ../migration-cli/schema.json -output types.ts

# 5. Commit changes
git add migrations/ types.ts schema.json
git commit -m "Add new fields"
```

### Workflow 3: Production Deployment

```bash
# 1. Test locally with dry-run
cd examples/node-migration
DRY_RUN=true SYNDR_CONN="$PROD_CONN" node migrate.js

# 2. Review generated commands

# 3. Apply to production
SYNDR_CONN="$PROD_CONN" node migrate.js

# 4. Verify with API
cd examples/node-graphql-api
SYNDR_CONN="$PROD_CONN" node server.js

# 5. Test GraphQL queries
curl -X POST http://localhost:4000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"{ users { id email } }"}'
```

## Best Practices

### Development

- ✅ Use `wasm-web` for schema design and prototyping
- ✅ Generate types with `schema-codegen` after schema changes
- ✅ Test migrations with `migration-cli` before committing
- ✅ Version control migration files
- ✅ Use meaningful migration names

### Testing

- ✅ Test migrations in development first
- ✅ Use dry-run mode before production
- ✅ Verify with integration tests
- ✅ Test API with GraphQL playground
- ✅ Check health endpoints

### Production

- ✅ Use `node-migration` in CI/CD pipelines
- ✅ Always backup before migrations
- ✅ Use read-only connections for code generation
- ✅ Monitor API health endpoints
- ✅ Log all migration operations

### Security

- ✅ Store connection strings in environment variables
- ✅ Use secrets management for CI/CD
- ✅ Never commit credentials
- ✅ Use read-only users for queries
- ✅ Validate input in API resolvers

## Troubleshooting

### Common Issues

**Connection Failed:**
```bash
# Check server is running
nc -zv localhost 1776

# Verify connection string format
echo "syndrdb://host:port:database:user:pass;"

# Test with migration-cli
cd examples/migration-cli
./syndr-migrate status -conn "syndrdb://localhost:1776:primary:root:root;"
```

**WASM Won't Load:**
```bash
# Verify WASM files exist
ls -lh ../wasm/syndrdb.wasm
ls -lh ../wasm/wasm_exec.js

# Use HTTP server (not file://)
python3 -m http.server 8080
```

**TypeScript Errors:**
```bash
# Regenerate types
cd examples/schema-codegen
./syndr-codegen typescript -conn "syndrdb://..." -output types.ts

# Check schema format
cat schema.json | jq .
```

**Migration Failed:**
```bash
# Check migration history
cd examples/migration-cli
./syndr-migrate status

# Validate migration file
cat migrations/001_*.json | jq .

# Try dry-run first
cd examples/node-migration
DRY_RUN=true node migrate.js
```

## Performance

| Operation | Time | Notes |
|-----------|------|-------|
| WASM Load | 100-300ms | One-time initialization |
| Schema Fetch | <100ms | Network dependent |
| Code Generation | <10ms | All formats |
| Migration Apply | Varies | Command dependent |
| GraphQL Query | 10-50ms | Query complexity dependent |

## Next Steps

1. **Read the docs** - Each example has detailed README
2. **Try examples** - Run through each example locally
3. **Customize** - Adapt examples to your needs
4. **Contribute** - Submit improvements or new examples
5. **Build** - Create your own SyndrDB applications

## Support

- 📖 [Main Documentation](../README.md)
- 🐛 [Report Issues](https://github.com/dan-strohschein/syndrdb-drivers/issues)
- 💬 [Discussions](https://github.com/dan-strohschein/syndrdb-drivers/discussions)

## License

All examples are licensed under the same terms as the main driver. See LICENSE file.
