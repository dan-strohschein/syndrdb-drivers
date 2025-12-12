# SyndrDB Node.js Driver - Export Reference

This document lists all public exports from the `@syndrdb/node-driver` package.

## Main Classes

### SyndrDBClient
Main client class for interacting with SyndrDB. Provides connection pooling, transaction support, and all database operations.

```typescript
import { SyndrDBClient } from '@syndrdb/node-driver';
const client = new SyndrDBClient();
```

### SyndrDBConnection
Low-level class representing a single TCP connection to SyndrDB.

```typescript
import { SyndrDBConnection } from '@syndrdb/node-driver';
const conn = new SyndrDBConnection(params);
```

### SyndrDBConnectionPool
Connection pool manager for handling multiple SyndrDB connections.

```typescript
import { SyndrDBConnectionPool } from '@syndrdb/node-driver';
const pool = new SyndrDBConnectionPool(params, options);
```

## Error Classes

### SyndrDBConnectionError
Thrown for connection-related errors (network failures, authentication failures, etc.).

```typescript
import { SyndrDBConnectionError } from '@syndrdb/node-driver';
```

Properties:
- `name: string` - "SyndrDBConnectionError"
- `message: string` - Error message
- `code?: string` - Error code (if available)
- `type?: string` - Error type (if available)

### SyndrDBProtocolError
Thrown for protocol-level errors (invalid responses, unsupported operations, etc.).

```typescript
import { SyndrDBProtocolError } from '@syndrdb/node-driver';
```

Properties:
- `name: string` - "SyndrDBProtocolError"
- `message: string` - Error message
- `code?: string` - Error code (if available)
- `type?: string` - Error type (if available)

### SyndrDBPoolError
Thrown for connection pool errors (pool exhausted, transaction conflicts, etc.).

```typescript
import { SyndrDBPoolError } from '@syndrdb/node-driver';
```

Properties:
- `name: string` - "SyndrDBPoolError"
- `message: string` - Error message
- `code?: string` - Error code (if available)
- `type?: string` - Error type (if available)

## Interfaces

### ConnectionParams
Parameters extracted from a connection string.

```typescript
interface ConnectionParams {
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
}
```

### ConnectionOptions
Options for configuring the connection pool.

```typescript
interface ConnectionOptions {
  maxConnections?: number;  // Default: 5
  idleTimeout?: number;     // Default: 30000 (30 seconds)
}
```

### SyndrDBResponse<T>
Generic response structure from SyndrDB server.

```typescript
interface SyndrDBResponse<T = any> {
  success: boolean;
  data?: T;
  error?: SyndrDBError;
}
```

### SyndrDBError
Error details from SyndrDB server.

```typescript
interface SyndrDBError {
  code: string;
  type: string;
  message: string;
}
```

## Utility Functions

### parseConnectionString
Parses a SyndrDB connection string into connection parameters.

```typescript
import { parseConnectionString } from '@syndrdb/node-driver';

const params = parseConnectionString('syndrdb://localhost:1776:mydb:admin:password;');
// Returns: ConnectionParams
```

## Complete Import Example

```typescript
import {
  // Main classes
  SyndrDBClient,
  SyndrDBConnection,
  SyndrDBConnectionPool,
  
  // Error classes
  SyndrDBConnectionError,
  SyndrDBProtocolError,
  SyndrDBPoolError,
  
  // Interfaces
  ConnectionParams,
  ConnectionOptions,
  SyndrDBResponse,
  SyndrDBError,
  
  // Utility functions
  parseConnectionString,
} from '@syndrdb/node-driver';
```

## TypeScript Support

All exports include full TypeScript type definitions. The package supports both CommonJS and ES Module imports:

```typescript
// ESM
import { SyndrDBClient } from '@syndrdb/node-driver';

// CommonJS
const { SyndrDBClient } = require('@syndrdb/node-driver');
```

## Package Metadata

- **Package Name**: `@syndrdb/node-driver`
- **Version**: 0.1.0
- **Node Version**: >= 18
- **License**: MIT
- **Formats**: CommonJS (CJS) + ES Modules (ESM)
- **Type Definitions**: Included (`.d.ts`)
