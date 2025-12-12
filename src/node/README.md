# @syndrdb/node-driver

Node.js/TypeScript driver for SyndrDB database with connection pooling, transaction support, and comprehensive database operations.

## Installation

```bash
npm install @syndrdb/node-driver
```

## Features

- 🔌 TCP-based connection to SyndrDB servers
- 🏊 Connection pooling with configurable size and idle timeout
- 🔄 Sticky transaction support
- 📊 Query, Mutation, and GraphQL operations (protocol placeholders)
- 🗂️ Schema management operations (Bundle/Index/View)
- 🔒 Type-safe TypeScript API
- ⚡ Dual CommonJS and ESM support

## Quick Start

```typescript
import { SyndrDBClient } from '@syndrdb/node-driver';

// Create client
const client = new SyndrDBClient();

// Connect to SyndrDB
await client.connect('syndrdb://localhost:1776:mydb:admin:password;', {
  maxConnections: 10,    // Optional: default 5
  idleTimeout: 60000     // Optional: default 30000ms
});

// Use the client (when protocol is implemented)
try {
  const results = await client.query('SELECT * FROM users');
  console.log(results);
} catch (error) {
  console.error('Query failed:', error);
}

// Close when done
await client.close();
```

## Connection String Format

```
syndrdb://<HOST>:<PORT>:<DATABASE>:<USERNAME>:<PASSWORD>;
```

Example:
```
syndrdb://localhost:1776:mydb:admin:secret123;
```

## API Reference

### SyndrDBClient

Main client class for database operations.

#### Methods

**Connection Management**

- `connect(connectionString: string, options?: ConnectionOptions): Promise<void>`
  - Establishes connection pool to SyndrDB server
  - Options: `maxConnections` (default: 5), `idleTimeout` (default: 30000ms)

- `close(): Promise<void>`
  - Closes all connections and shuts down the client

- `getPoolStats(): { available: number; active: number; total: number; max: number } | null`
  - Returns current connection pool statistics

**Transaction Support**

- `beginTransaction(): Promise<void>` *(placeholder)*
  - Begins a new transaction with sticky connection binding

- `commit(): Promise<void>` *(placeholder)*
  - Commits the active transaction

- `rollback(): Promise<void>` *(placeholder)*
  - Rolls back the active transaction

**Database Operations**

- `query<T>(sql: string): Promise<T>` *(placeholder)*
  - Executes a SQL query

- `mutate<T>(mutation: string): Promise<T>` *(placeholder)*
  - Executes a mutation

- `graphql<T>(query: string): Promise<T>` *(placeholder)*
  - Executes a GraphQL query

- `migrate(migration: string): Promise<void>` *(placeholder)*
  - Executes a migration script

**Schema Management**

- `addBundle(definition: Record<string, any>): Promise<void>` *(placeholder)*
- `addIndex(definition: Record<string, any>): Promise<void>` *(placeholder)*
- `addView(definition: Record<string, any>): Promise<void>` *(placeholder)*
- `changeBundle(name: string, definition: Record<string, any>): Promise<void>` *(placeholder)*
- `changeIndex(name: string, definition: Record<string, any>): Promise<void>` *(placeholder)*
- `changeView(name: string, definition: Record<string, any>): Promise<void>` *(placeholder)*
- `dropBundle(name: string): Promise<void>` *(placeholder)*
- `dropIndex(name: string): Promise<void>` *(placeholder)*
- `dropView(name: string): Promise<void>` *(placeholder)*

### Advanced Usage

#### Direct Connection Access

```typescript
import { SyndrDBConnection, parseConnectionString } from '@syndrdb/node-driver';

const params = parseConnectionString('syndrdb://localhost:1776:mydb:admin:password;');
const conn = new SyndrDBConnection(params);

await conn.connect();
await conn.sendCommand('SOME_COMMAND\n');
const response = await conn.receiveResponse();
await conn.close();
```

#### Custom Connection Pool

```typescript
import { SyndrDBConnectionPool, parseConnectionString } from '@syndrdb/node-driver';

const params = parseConnectionString('syndrdb://localhost:1776:mydb:admin:password;');
const pool = new SyndrDBConnectionPool(params, {
  maxConnections: 20,
  idleTimeout: 120000  // 2 minutes
});

const conn = await pool.acquire();
try {
  // Use connection
} finally {
  await pool.release(conn);
}

await pool.closeAll();
```

## Error Handling

The driver provides three custom error types:

```typescript
import {
  SyndrDBConnectionError,
  SyndrDBProtocolError,
  SyndrDBPoolError
} from '@syndrdb/node-driver';

try {
  await client.connect('invalid-connection-string');
} catch (error) {
  if (error instanceof SyndrDBConnectionError) {
    console.error('Connection failed:', error.message);
    console.error('Error code:', error.code);
    console.error('Error type:', error.type);
  }
}
```

- **SyndrDBConnectionError**: Connection-related errors (network, authentication, etc.)
- **SyndrDBProtocolError**: Protocol-level errors (unsupported operations, malformed responses)
- **SyndrDBPoolError**: Connection pool errors (exhausted pool, transaction conflicts)

## Development Status

**Current Status**: v0.1.0 - Core infrastructure complete

**Implemented**:
- ✅ Connection string parsing and validation
- ✅ TCP socket connection handling
- ✅ Connection pooling with idle timeout
- ✅ Sticky transaction connection binding
- ✅ Error handling and custom error types
- ✅ TypeScript type definitions
- ✅ Comprehensive test suite

**Pending** (awaiting SyndrDB protocol implementation):
- ⏳ Query execution
- ⏳ Mutation execution
- ⏳ GraphQL support
- ⏳ Migration support
- ⏳ Transaction BEGIN/COMMIT/ROLLBACK
- ⏳ Schema management operations

All placeholder methods throw `SyndrDBProtocolError` with code `NOT_IMPLEMENTED` until the server protocol is finalized.

## Testing

```bash
npm test
```

## Building

```bash
npm run build
```

Produces:
- `dist/cjs/` - CommonJS build
- `dist/esm/` - ES Module build
- `dist/*.d.ts` - TypeScript declarations

## License

MIT
