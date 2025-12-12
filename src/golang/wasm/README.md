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

## Build Configuration

The WASM binary is built with:
- `GOOS=js GOARCH=wasm`
- Size optimization: `-ldflags "-s -w"`
- Version stamping: `-X github.com/dan-strohschein/syndrdb-drivers/src/golang/client.Version=...`
- Gzip compression for web delivery

## Memory Management

- Call `cleanup()` when done to prevent memory leaks
- State change callbacks are automatically released on cleanup
- The client maintains a single global instance

## Compatibility

- Requires Go 1.24.2 or higher for building
- Compatible with modern browsers supporting WebAssembly
- Node.js 14+ with WebAssembly support
- Requires `wasm_exec.js` from your Go installation
