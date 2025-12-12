# SyndrDB Schema Codegen Tool

A command-line tool for generating code from SyndrDB schemas. Fetch schemas from a running database and generate TypeScript types, JSON Schema, or GraphQL SDL.

## Features

- 🔄 **Fetch Schemas** - Pull schema definitions from SyndrDB server
- 📝 **TypeScript Generation** - Create typed interfaces
- 📋 **JSON Schema** - Generate single or multi-file schemas
- 🎯 **GraphQL SDL** - Create GraphQL type definitions
- ⚡ **Fast** - Zero dependencies, pure Go implementation
- 🔧 **Flexible** - Multiple output formats and modes

## Installation

```bash
cd examples/schema-codegen
go build -o syndr-codegen
```

## Usage

### Fetch Schema from Server

```bash
# Fetch and display schema
./syndr-codegen fetch -conn "syndrdb://127.0.0.1:1776:primary:root:root;"

# Save to file
./syndr-codegen fetch -conn "syndrdb://127.0.0.1:1776:primary:root:root;" -output schema.json
```

### Generate TypeScript Types

```bash
# From server
./syndr-codegen typescript -conn "syndrdb://127.0.0.1:1776:primary:root:root;"

# From file
./syndr-codegen typescript -schema schema.json -output types.ts
```

**Example Output:**

```typescript
// Generated TypeScript types for SyndrDB schema

export interface User {
  id: number;
  email: string;
  username: string;
  password_hash: string;
  created_at: Date;
  updated_at?: Date;
}

export interface Post {
  id: number;
  user_id: number;
  title: string;
  content: string;
  published: boolean;
  created_at: Date;
}
```

### Generate JSON Schema

```bash
# Single file mode (all bundles in one schema)
./syndr-codegen json-schema -schema schema.json -mode single -output schema.json

# Multi-file mode (one file per bundle)
./syndr-codegen json-schema -schema schema.json -mode multi -output ./schemas/
```

**Single File Output:**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "users": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "email", "username"],
        "properties": {
          "id": { "type": "integer" },
          "email": { "type": "string" },
          "username": { "type": "string" }
        }
      }
    }
  }
}
```

**Multi-File Output:**

Creates `users.schema.json`, `posts.schema.json`, etc.

### Generate GraphQL Schema

```bash
# From server
./syndr-codegen graphql -conn "syndrdb://127.0.0.1:1776:primary:root:root;"

# From file
./syndr-codegen graphql -schema schema.json -output schema.graphql
```

**Example Output:**

```graphql
type User {
  id: Int!
  email: String!
  username: String!
  password_hash: String!
  created_at: String!
  updated_at: String
}

type Post {
  id: Int!
  user_id: Int!
  title: String!
  content: String!
  published: Boolean!
  created_at: String!
}

type Query {
  users: [User!]!
  user(id: Int!): User
  posts: [Post!]!
  post(id: Int!): Post
}
```

## Commands

### fetch

Fetch schema from SyndrDB server and optionally save to file.

**Flags:**
- `-conn` - Connection string (required)
- `-output` - Output file path (optional, defaults to stdout)

**Examples:**

```bash
# Display on screen
./syndr-codegen fetch -conn "syndrdb://localhost:1776:mydb:user:pass;"

# Save to file
./syndr-codegen fetch -conn "syndrdb://localhost:1776:mydb:user:pass;" -output myschema.json
```

### typescript

Generate TypeScript type definitions from schema.

**Flags:**
- `-conn` - Connection string (fetch from server)
- `-schema` - Schema file path (read from file)
- `-output` - Output file path (optional, defaults to stdout)

**Note:** Must specify either `-conn` or `-schema`.

**Examples:**

```bash
# From server
./syndr-codegen typescript -conn "syndrdb://localhost:1776:mydb:user:pass;" -output types.ts

# From file
./syndr-codegen typescript -schema schema.json -output types.ts

# Display on screen
./syndr-codegen typescript -schema schema.json
```

### json-schema

Generate JSON Schema definitions.

**Flags:**
- `-conn` - Connection string (fetch from server)
- `-schema` - Schema file path (read from file)
- `-mode` - Generation mode: `single` or `multi` (default: single)
- `-output` - Output path (file for single, directory for multi)

**Examples:**

```bash
# Single file from server
./syndr-codegen json-schema -conn "syndrdb://localhost:1776:mydb:user:pass;" -mode single -output schema.json

# Multi-file from local schema
./syndr-codegen json-schema -schema schema.json -mode multi -output ./schemas/

# Display single file on screen
./syndr-codegen json-schema -schema schema.json -mode single
```

### graphql

Generate GraphQL SDL (Schema Definition Language).

**Flags:**
- `-conn` - Connection string (fetch from server)
- `-schema` - Schema file path (read from file)
- `-output` - Output file path (optional, defaults to stdout)

**Examples:**

```bash
# From server
./syndr-codegen graphql -conn "syndrdb://localhost:1776:mydb:user:pass;" -output schema.graphql

# From file
./syndr-codegen graphql -schema schema.json -output schema.graphql

# Display on screen
./syndr-codegen graphql -schema schema.json
```

## Workflow Examples

### 1. Full Development Workflow

```bash
# 1. Fetch current schema from development server
./syndr-codegen fetch -conn "syndrdb://dev:1776:myapp:root:root;" -output dev-schema.json

# 2. Generate TypeScript types for frontend
./syndr-codegen typescript -schema dev-schema.json -output ../frontend/src/types/db.ts

# 3. Generate GraphQL schema for API
./syndr-codegen graphql -schema dev-schema.json -output ../api/schema.graphql

# 4. Generate JSON Schema for validation
./syndr-codegen json-schema -schema dev-schema.json -mode multi -output ../api/schemas/
```

### 2. CI/CD Pipeline

```bash
#!/bin/bash
# Generate all code artifacts from production schema

SYNDR_CONN="syndrdb://prod:1776:myapp:readonly:pass;"

echo "Fetching schema..."
./syndr-codegen fetch -conn "$SYNDR_CONN" -output prod-schema.json

echo "Generating TypeScript..."
./syndr-codegen typescript -schema prod-schema.json -output types.ts

echo "Generating GraphQL..."
./syndr-codegen graphql -schema prod-schema.json -output schema.graphql

echo "Generating JSON Schema..."
./syndr-codegen json-schema -schema prod-schema.json -mode multi -output ./schemas/

echo "Code generation complete!"
```

### 3. Multi-Environment Setup

```bash
# Development
./syndr-codegen fetch -conn "syndrdb://dev:1776:app:root:root;" -output schemas/dev.json
./syndr-codegen typescript -schema schemas/dev.json -output types/dev.ts

# Staging
./syndr-codegen fetch -conn "syndrdb://staging:1776:app:user:pass;" -output schemas/staging.json
./syndr-codegen typescript -schema schemas/staging.json -output types/staging.ts

# Production
./syndr-codegen fetch -conn "syndrdb://prod:1776:app:readonly:pass;" -output schemas/prod.json
./syndr-codegen typescript -schema schemas/prod.json -output types/prod.ts
```

## Integration Examples

### With Frontend Frameworks

**React/Vue/Svelte:**

```bash
# Generate types for frontend
./syndr-codegen typescript -conn "syndrdb://localhost:1776:myapp:root:root;" -output src/types/database.ts
```

```typescript
// Use in your components
import { User, Post } from './types/database';

const user: User = {
  id: 1,
  email: 'user@example.com',
  username: 'johndoe',
  password_hash: '...',
  created_at: new Date(),
};
```

### With GraphQL Servers

**Apollo Server:**

```bash
# Generate GraphQL schema
./syndr-codegen graphql -conn "syndrdb://localhost:1776:myapp:root:root;" -output schema.graphql
```

```javascript
import { loadSchemaSync } from '@graphql-tools/load';
import { GraphQLFileLoader } from '@graphql-tools/graphql-file-loader';

const schema = loadSchemaSync('./schema.graphql', {
  loaders: [new GraphQLFileLoader()]
});
```

### With Validation Libraries

**Ajv (JSON Schema validation):**

```bash
# Generate JSON schemas
./syndr-codegen json-schema -schema schema.json -mode multi -output ./schemas/
```

```javascript
import Ajv from 'ajv';
import userSchema from './schemas/users.schema.json';

const ajv = new Ajv();
const validate = ajv.compile(userSchema);

const valid = validate(data);
if (!valid) console.log(validate.errors);
```

## Type Mapping

### SyndrDB → TypeScript

| SyndrDB Type | TypeScript Type |
|--------------|-----------------|
| int          | number          |
| float        | number          |
| string       | string          |
| bool         | boolean         |
| timestamp    | Date            |
| json         | any             |

### SyndrDB → JSON Schema

| SyndrDB Type | JSON Schema Type |
|--------------|------------------|
| int          | integer          |
| float        | number           |
| string       | string           |
| bool         | boolean          |
| timestamp    | string           |
| json         | object           |

### SyndrDB → GraphQL

| SyndrDB Type | GraphQL Type |
|--------------|--------------|
| int          | Int          |
| float        | Float        |
| string       | String       |
| bool         | Boolean      |
| timestamp    | String       |
| json         | JSON (custom scalar) |

## Production Tips

### Performance

- Schema fetching is fast (typically <100ms)
- Code generation is instant (sub-millisecond)
- Safe to run in CI/CD pipelines

### Best Practices

✅ **Version control generated files** - Commit types for team consistency  
✅ **Automate generation** - Run in pre-commit hooks or CI/CD  
✅ **Use read-only credentials** - When fetching from production  
✅ **Validate output** - Check generated files compile/validate  
✅ **Document types** - Add JSDoc comments manually if needed

### Common Patterns

**Git Hook (pre-commit):**

```bash
#!/bin/bash
# .git/hooks/pre-commit

./syndr-codegen fetch -conn "$SYNDR_CONN" -output schema.json
./syndr-codegen typescript -schema schema.json -output types.ts

git add schema.json types.ts
```

**Makefile:**

```makefile
.PHONY: codegen
codegen:
	cd examples/schema-codegen && \
	./syndr-codegen fetch -conn "$(SYNDR_CONN)" -output schema.json && \
	./syndr-codegen typescript -schema schema.json -output ../../frontend/types.ts && \
	./syndr-codegen graphql -schema schema.json -output ../../api/schema.graphql
```

**NPM Script:**

```json
{
  "scripts": {
    "codegen": "syndr-codegen typescript -schema schema.json -output src/types/db.ts"
  }
}
```

## What This Demonstrates

- ✅ Fetching schemas from SyndrDB
- ✅ Multiple code generation formats
- ✅ CLI tool design patterns
- ✅ Type mapping across languages
- ✅ Single and multi-file outputs
- ✅ Development workflow integration

## Limitations

- GraphQL generation is basic (no resolvers, just types)
- TypeScript generation doesn't include relationships
- JSON Schema generation uses draft-07

For more complex code generation needs, extend the `codegen` package.

## Next Steps

- Add support for GraphQL resolvers
- Generate ORM models (Prisma, TypeORM, etc.)
- Add support for other languages (Python, C#, Java)
- Generate API documentation
- Add validation rules to generated schemas

## Related Examples

- `../migration-cli/` - Migration management tool
- `../node-graphql-api/` - GraphQL API using generated types
- `../node-migration/` - Automated migration script
- `../wasm-web/` - Browser-based code generation
