# Changelog

All notable changes to the SyndrDB Node.js Driver will be documented in this file.

## [0.1.0] - 2025-12-07

### Added

#### Core Infrastructure
- **Connection String Parser**: Parse and validate `syndrdb://<HOST>:<PORT>:<DATABASE>:<USERNAME>:<PASSWORD>;` format
- **SyndrDBConnection Class**: Low-level TCP socket connection handler
  - Async connect/disconnect
  - Line-buffered reading (newline-terminated messages)
  - 10-second read timeout for normal operations
  - 1ms timeout for non-blocking message checks
  - JSON response parsing with error handling
  - Idle timestamp tracking
  - Transaction state flag

#### Connection Pooling
- **SyndrDBConnectionPool Class**: Manages multiple connections
  - Configurable max connections (default: 5)
  - Configurable idle timeout (default: 30 seconds)
  - Automatic idle connection cleanup
  - Connection health checking and auto-reconnection
  - Pool statistics tracking
  - Transaction-aware (doesn't evict connections in transactions)

#### Main Client API
- **SyndrDBClient Class**: High-level database client
  - Connection pooling support
  - Sticky transaction connections
  - Pool statistics access

#### Database Operations (Placeholders)
- `query(sql: string)`: Execute SQL queries
- `mutate(mutation: string)`: Execute mutations
- `graphql(query: string)`: Execute GraphQL queries
- `migrate(migration: string)`: Run migrations
- `beginTransaction()`: Start transaction with sticky connection
- `commit()`: Commit active transaction
- `rollback()`: Rollback active transaction

#### Schema Management (Placeholders)
- `addBundle(definition)`: Add new bundle
- `addIndex(definition)`: Add new index
- `addView(definition)`: Add new view
- `changeBundle(name, definition)`: Modify bundle
- `changeIndex(name, definition)`: Modify index
- `changeView(name, definition)`: Modify view
- `dropBundle(name)`: Remove bundle
- `dropIndex(name)`: Remove index
- `dropView(name)`: Remove view

#### Error Handling
- **SyndrDBConnectionError**: Connection failures, network errors
- **SyndrDBProtocolError**: Protocol errors, unimplemented operations
- **SyndrDBPoolError**: Pool exhaustion, transaction conflicts
- All errors include optional `code` and `type` properties

#### TypeScript Support
- Full type definitions for all classes and interfaces
- Generic typing for query/mutation responses
- Type-safe error handling
- Dual package support (CommonJS + ESM)

#### Testing
- 24 comprehensive unit tests
- Connection string parsing tests
- Error class tests
- Connection and pool instantiation tests
- Client API error handling tests
- All tests passing

#### Documentation
- Comprehensive README with API reference
- Quick start guide
- Usage examples
- Error handling documentation
- Export reference documentation
- Example scripts (basic usage, connection pool demo)

### Implementation Notes

All database operation and schema management methods currently throw `SyndrDBProtocolError` with code `NOT_IMPLEMENTED` as placeholders. These will be implemented once the SyndrDB server protocol is finalized.

The driver infrastructure is complete and ready for protocol implementation:
- TCP connection handling ✅
- Connection pooling ✅
- Transaction binding ✅
- JSON response parsing ✅
- Error handling ✅
- Type safety ✅

### Technical Details

- **Protocol**: TCP sockets (Node.js `net` module)
- **Message Format**: Newline-delimited text/JSON
- **Authentication**: Connection string sent on initial connection
- **Success Code**: Server responds with `S0001` on successful connection
- **Idle Handling**: Client-side cleanup (30s), server-side timeout (30 minutes)
- **Build System**: tsup (fast TypeScript bundler)
- **Test Framework**: Jest with ts-jest
- **Node Version**: >= 18
- **License**: MIT

### Breaking Changes

None (initial release)

### Deprecated

None

### Security

- Credentials transmitted in connection string during initial TCP handshake
- No credential storage or caching
- Connections closed on client shutdown

### Dependencies

Runtime Dependencies:
- None (uses built-in Node.js modules only)

Development Dependencies:
- TypeScript
- tsup (bundler)
- Jest (testing)
- ts-jest (TypeScript testing)

---

## [Unreleased]

### Planned

- Actual protocol implementation for:
  - Query execution
  - Mutation execution
  - GraphQL operations
  - Migration support
  - Transaction control (BEGIN/COMMIT/ROLLBACK)
  - Schema operations (Bundle/Index/View management)
- Response data type definitions
- Connection retry logic
- Prepared statement support
- Streaming query results
- WebSocket protocol support (if needed)
