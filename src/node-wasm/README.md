# @syndrdb/node-wasm

> SyndrDB Node.js driver powered by Go 1.25 WASM with DWARF v5 debugging support

[![npm version](https://img.shields.io/npm/v/@syndrdb/node-wasm.svg)](https://www.npmjs.com/package/@syndrdb/node-wasm)
[![Build Status](https://img.shields.io/github/actions/workflow/status/dan-strohschein/syndrdb-drivers/test.yml?branch=main)](https://github.com/dan-strohschein/syndrdb-drivers/actions)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://go.dev/dl/)
[![License](https://img.shields.io/npm/l/@syndrdb/node-wasm.svg)](LICENSE)

A high-performance Node.js driver for SyndrDB, built as a thin TypeScript wrapper around the Go core driver compiled to WebAssembly. This architecture provides:

- **5-10x faster** CPU-bound operations (schema diffing, migration generation)
- **Automatic rollback generation** for migrations
- **Single source of truth** - all features from Go driver
- **DWARF v5 debugging** - Full source maps with dual stack traces (Go + JavaScript)
- **Battle-tested** - Same core used by production Go applications

## Installation

```bash
npm install @syndrdb/node-wasm
```

**Requirements:**
- Node.js 18.0.0 or higher
- For building from source: Go 1.25+ (for DWARF v5 support)

## Quick Start

```typescript
import { SyndrDBClient } from '@syndrdb/node-wasm';

// Create client
const client = new SyndrDBClient({
  defaultTimeoutMs: 10000,
  debugMode: true, // Enable dual stack traces
  poolSize: 10,
});

// Connect to database
await client.connect('syndrdb://localhost:7632:mydb:admin:password;');

// Execute queries
const users = await client.query<User[]>('SELECT * FROM users WHERE age > 21');
console.log(users);

// Execute mutations
await client.mutate('INSERT INTO users (name, age) VALUES ("Alice", 30)');

// Disconnect
await client.disconnect();
```

## Features

### 🚀 High Performance

- **6-7x faster** schema comparison and diffing
- **Operation batching** minimizes JS↔WASM boundary crossings
- **Connection pooling** with automatic health checks
- **Performance monitoring** built-in with detailed metrics

### 🐛 Enhanced Debugging

- **DWARF v5 source maps** embedded in WASM binary
- **Dual stack traces** show both Go and JavaScript call stacks
- **Debug mode** with verbose logging and performance tracking
- **Error context** preserved across WASM boundary

### 🔄 Advanced Migrations

- **Automatic rollback** generation from schema diffs
- **Migration validation** detects breaking changes
- **File-based migrations** with checksums and locking
- **Preview mode** for dry-run testing

### 🪝 Extensible Hook System

- **Custom hooks** for logging, metrics, tracing
- **Built-in hooks** for common patterns
- **Async support** with Promise-based API
- **Performance hooks** for operation timing

## Documentation

- [API Reference](./docs/API.md)
- [Architecture Guide](./docs/ARCHITECTURE.md)
- [Performance Guide](./docs/PERFORMANCE.md)
- [Debugging Guide](./docs/DEBUGGING.md)
- [Migration from Native Driver](./docs/MIGRATION_GUIDE.md)

## Examples

See the [examples/](./examples/) directory for complete working examples:

- [Basic Usage](./examples/basic-usage.ts)
- [Migration Workflow](./examples/migration-workflow.ts)
- [Transactions](./examples/transactions.ts)
- [Custom Hooks](./examples/custom-hooks.ts)
- [Batch Operations](./examples/batch-operations.ts)
- [Performance Monitoring](./examples/performance-monitoring.ts)

## Performance Benchmarks

| Operation | Native TS | WASM (Go) | Speedup |
|-----------|-----------|-----------|---------|
| Schema Comparison | ~4µs | 681ns | **6x faster** |
| Migration Generation | ~50µs | 8.38µs | **6x faster** |
| JSON Schema Gen | ~45µs | 7.41µs | **6x faster** |
| GraphQL Schema Gen | ~30µs | 4.78µs | **6.3x faster** |

*Network I/O operations have similar performance (0-5% difference)*

## Debugging

When `debugMode: true`, errors include both Go and JavaScript stack traces:

```typescript
try {
  await client.connect('invalid://connection');
} catch (err) {
  console.error(err);
  // Error: Failed to connect
  //   at SyndrDBClient.connect (client.ts:45)
  //   at async main (index.ts:10)
  // 
  // --- Go Stack Trace (from WASM) ---
  //   at connection.go:123 (github.com/dan-strohschein/syndrdb-drivers/client.Connect)
  //   at client.go:89 (github.com/dan-strohschein/syndrdb-drivers/client.NewConnection)
}
```

## Development

```bash
# Install dependencies
npm install

# Build WASM from Go source (requires Go 1.25+)
npm run build:wasm

# Build TypeScript wrapper
npm run build

# Run tests
npm test

# Run integration tests (requires SyndrDB server)
npm run test:integration

# Run benchmarks
npm run benchmark

# Generate type definitions from Go source
npm run generate-types
```

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for development setup and guidelines.

## License

MIT © Dan Strohschein

## Related

- [SyndrDB Go Driver](../golang/) - Core Go driver
- [SyndrDB Node Driver (Native)](../node/) - Native TypeScript implementation (deprecated)
- [SyndrDB Server](https://github.com/dan-strohschein/syndrdb) - Database server
