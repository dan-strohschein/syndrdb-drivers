# SyndrDB WASM Driver

This directory contains the WebAssembly (WASM) export layer for the SyndrDB Go driver.

## ⚠️ Important: Browser Limitations

**The WASM driver can only connect to SyndrDB servers in Node.js environments.** Browsers do not allow direct TCP socket connections due to security restrictions. The WASM driver works perfectly in Node.js, but browser environments can only use it for:

- Schema generation (JSON Schema, GraphQL)
- Client-side type generation
- Offline schema validation
- Migration planning (without execution)

For browser-based database access, you would need a WebSocket proxy server or REST API gateway.

## Building

To build the WASM binary:

```bash
cd ..
./scripts/build-wasm.sh
```

This generates:
- `wasm/syndrdb.wasm` - The compiled WASM binary
- `wasm/syndrdb.wasm.gz` - Compressed version for web delivery

## Usage

### In the Browser

```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <script src="wasm_exec.js"></script>
    <script>
        const go = new Go();
        WebAssembly.instantiateStreaming(fetch("syndrdb.wasm"), go.importObject)
            .then((result) => {
                go.run(result.instance);
                
                // SyndrDB is now available globally
                const client = SyndrDB.createClient({
                    defaultTimeoutMs: 10000,
                    debugMode: false,
                    maxRetries: 3
                });
                
                // Connect to database
                await SyndrDB.connect("syndrdb://localhost:7632/mydb");
                
                // Execute query
                const result = await SyndrDB.query("SELECT * FROM users");
                console.log(result);
                
                // Cleanup when done
                SyndrDB.cleanup();
            });
    </script>
</head>
<body>
    <h1>SyndrDB WASM Example</h1>
</body>
</html>
```

### In Node.js

```javascript
const fs = require('fs');
const { Go } = require('./wasm_exec');

const go = new Go();
const wasmBuffer = fs.readFileSync('./syndrdb.wasm');

WebAssembly.instantiate(wasmBuffer, go.importObject).then(async (result) => {
    go.run(result.instance);
    
    // Create client
    await SyndrDB.createClient({ debugMode: true });
    
    // Connect
    await SyndrDB.connect("syndrdb://localhost:7632/mydb");
    
    // Query
    const data = await SyndrDB.query("SELECT * FROM products");
    console.log(data);
    
    // Cleanup
    SyndrDB.cleanup();
});
```

## API Reference

### Client Methods

#### `createClient(options?)`
Creates a new SyndrDB client.

```javascript
await SyndrDB.createClient({
    defaultTimeoutMs: 10000,  // Default: 10000
    debugMode: false,         // Default: false
    maxRetries: 3             // Default: 3
});
```

#### `connect(connectionString)`
Establishes a connection to the database.

```javascript
await SyndrDB.connect("syndrdb://host:port/database");
```

#### `disconnect()`
Closes the connection.

```javascript
await SyndrDB.disconnect();
```

#### `query(queryString, timeout?)`
Executes a query.

```javascript
const result = await SyndrDB.query("SELECT * FROM users", 5000);
```

#### `mutate(mutationString, timeout?)`
Executes a mutation.

```javascript
const result = await SyndrDB.mutate("INSERT INTO users ...", 5000);
```

#### `getState()`
Returns the current connection state (synchronous).

```javascript
const state = SyndrDB.getState(); // "CONNECTED", "DISCONNECTED", etc.
```

#### `onStateChange(callback)`
Registers a callback for state changes.

```javascript
SyndrDB.onStateChange((transition) => {
    console.log(`State changed: ${transition.from} -> ${transition.to}`);
    console.log(`Duration: ${transition.duration}ms`);
    if (transition.error) {
        console.error('Error:', transition.error);
    }
});
```

#### `getVersion()`
Returns the client version (synchronous).

```javascript
const version = SyndrDB.getVersion();
```

### Schema Generation

#### `generateJSONSchema(schemaJSON, mode?)`
Generates JSON Schema from a schema definition.

```javascript
const schema = await SyndrDB.generateJSONSchema(
    JSON.stringify(schemaDefinition),
    "single"  // or "multi" for separate files
);
```

#### `generateGraphQLSchema(schemaJSON)`
Generates GraphQL SDL from a schema definition.

```javascript
const sdl = await SyndrDB.generateGraphQLSchema(
    JSON.stringify(schemaDefinition)
);
```

### Cleanup

#### `cleanup()`
Releases all resources and callbacks. Call when done using the driver.

```javascript
SyndrDB.cleanup();
```

## State Transitions

The client emits state change events with the following structure:

```javascript
{
    from: "DISCONNECTED",      // Previous state
    to: "CONNECTED",           // New state
    timestamp: 1234567890,     // Unix timestamp (milliseconds)
    duration: 150,             // Time in previous state (milliseconds)
    error: "...",              // Error message (if applicable)
    metadata: {                // Additional context
        reason: "user_initiated",
        remoteAddr: "localhost:7632",
        // ... other metadata
    }
}
```

## Hooks System API

### Overview

The WASM driver supports the hooks system for intercepting and modifying command execution. You can register custom JavaScript hooks or use built-in hooks for logging, metrics, and tracing.

**Performance Note:** JavaScript hooks have ~10-50µs overhead per hook due to Go↔JavaScript boundary crossing. For performance-critical applications, consider using Go-based hooks or minimizing the number of JS hooks.

### Custom JavaScript Hooks

Register custom hooks with `before` and `after` callbacks:

```javascript
// Synchronous hook
await SyndrDB.registerHook({
    name: 'my-hook',
    before: (ctx) => {
        console.log('Executing:', ctx.command);
        // Modify command if needed
        ctx.command = ctx.command + ' /* traced */';
        return ctx;
    },
    after: (ctx) => {
        console.log('Completed in', ctx.durationMs, 'ms');
        if (ctx.error) {
            console.error('Error:', ctx.error);
        }
        return ctx;
    }
});

// Async hook with Promises
await SyndrDB.registerHook({
    name: 'async-logger',
    before: async (ctx) => {
        await logToExternalService(ctx.command);
        return ctx;
    },
    after: async (ctx) => {
        await recordMetrics(ctx.durationMs);
        return ctx;
    }
});
```

### Hook Context

The `HookContext` object passed to hooks contains:

```typescript
interface HookContext {
    command: string;          // SQL command being executed
    commandType: string;      // 'query', 'mutation', 'transaction', 'schema'
    traceId: string;          // Unique trace ID for this command
    startTime: number;        // Unix timestamp (milliseconds)
    params?: any[];           // Command parameters (if any)
    metadata: object;         // Custom metadata (shared between hooks)
    
    // Available in 'after' hook:
    result?: string;          // Command result (JSON string)
    error?: string;           // Error message (if failed)
    durationMs?: number;      // Execution duration
}
```

### Built-in Hooks

#### Logging Hook

Logs command execution with configurable detail levels:

```javascript
await SyndrDB.createLoggingHook({
    logCommands: true,   // Log raw commands
    logResults: false,   // Log results (can be verbose)
    logDurations: true   // Log execution times
});
```

#### Metrics Hook

Collects performance metrics using atomic counters:

```javascript
// Create metrics hook
await SyndrDB.createMetricsHook();

// Get current stats
const stats = await SyndrDB.getMetricsStats();
console.log(stats);
// {
//   total_commands: 150,
//   total_queries: 100,
//   total_mutations: 48,
//   total_errors: 2,
//   total_duration_ns: 45000000,
//   avg_duration_ms: 0.3,
//   total_duration_ms: 45
// }

// Reset metrics
await SyndrDB.resetMetrics();
```

#### Tracing Hook

Provides distributed tracing metadata:

```javascript
await SyndrDB.createTracingHook('my-service');

// Hook adds trace metadata to context
// Access via ctx.metadata.trace_start, trace_duration, trace_service
```

### Hook Management

```javascript
// Get list of registered hooks
const { hooks, count } = await SyndrDB.getHooks();
console.log('Registered hooks:', hooks); // ['metrics', 'logging', 'my-hook']

// Unregister a hook
await SyndrDB.unregisterHook('my-hook');
```

### Performance Comparison

| Hook Type | Overhead per Command | Use Case |
|-----------|---------------------|----------|
| **Go Native Hooks** | ~0.8% (15ns) | Production, high-throughput |
| **Go Built-in Hooks** | ~0.8% (15ns) | Metrics, logging, tracing |
| **JS Sync Hooks** | ~1-2% (10-50µs) | Development, debugging |
| **JS Async Hooks** | ~2-5% (20-100µs) | External logging, analytics |

**Recommendation:** Use Go built-in hooks (metrics, logging) in production for minimal overhead. Use JavaScript hooks for development, debugging, or when you need to integrate with external JS libraries.

### Hybrid Strategy

For optimal performance, combine Go and JavaScript hooks:

```javascript
// Fast: Use Go built-in hooks for metrics
await SyndrDB.createMetricsHook();
await SyndrDB.createLoggingHook({ logCommands: true, logResults: false, logDurations: true });

// Flexible: Add JS hook for custom logic when needed
await SyndrDB.registerHook({
    name: 'custom-analytics',
    after: async (ctx) => {
        // Only runs when you need custom JS integration
        if (ctx.durationMs > 100) {
            await sendSlowQueryAlert(ctx);
        }
    }
});
```

### Example: Query Timing Hook

```javascript
await SyndrDB.registerHook({
    name: 'timing',
    before: (ctx) => {
        ctx.metadata.clientStartTime = Date.now();
        return ctx;
    },
    after: (ctx) => {
        const clientDuration = Date.now() - ctx.metadata.clientStartTime;
        console.log(`Query: ${ctx.durationMs}ms (client: ${clientDuration}ms)`);
        return ctx;
    }
});
```

### Example: Error Notification Hook

```javascript
await SyndrDB.registerHook({
    name: 'error-notifier',
    after: async (ctx) => {
        if (ctx.error) {
            await fetch('/api/errors', {
                method: 'POST',
                body: JSON.stringify({
                    command: ctx.command,
                    error: ctx.error,
                    traceId: ctx.traceId,
                    timestamp: Date.now()
                })
            });
        }
    }
});
```

## Build Configuration

The WASM binary is built with:
- `GOOS=js GOARCH=wasm`
- Size optimization: `-ldflags "-s -w"`
- Version stamping: `-X github.com/dan-strohschein/syndrdb-drivers/src/golang/client.Version=...`
- Gzip compression for web delivery

## Memory Management

- Call `cleanup()` when done to prevent memory leaks
- State change callbacks are automatically released on cleanup
- Hooks are automatically unregistered on cleanup
- The client maintains a single global instance

## Compatibility

- Requires Go 1.24.2 or higher for building
- Compatible with modern browsers supporting WebAssembly
- Node.js 14+ with WebAssembly support
- Requires `wasm_exec.js` from your Go installation
