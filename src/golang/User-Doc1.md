# SyndrDB Go Driver - Milestone 1 Features

## Overview

The SyndrDB Go driver provides a production-ready client for connecting to SyndrDB servers with advanced features including connection pooling, TLS support, health monitoring, structured logging, and debug capabilities.

## Table of Contents

- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [Connection Pooling](#connection-pooling)
- [Context Support](#context-support)
- [TLS/SSL Configuration](#tlsssl-configuration)
- [Health Monitoring](#health-monitoring)
- [Structured Logging](#structured-logging)
- [Debug Mode](#debug-mode)
- [Error Handling](#error-handling)
- [WASM Support](#wasm-support)

## Installation

```bash
go get github.com/dan-strohschein/syndrdb-drivers/src/golang/client
```

## Basic Usage

### Creating a Client

```go
import (
    "context"
    "github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
)

// Create client with default options
c := client.NewClient(nil)

// Connect to database
ctx := context.Background()
err := c.Connect(ctx, "syndrdb://localhost:1776/mydb")
if err != nil {
    log.Fatal(err)
}
defer c.Disconnect(ctx)

// Execute queries
result, err := c.Query("SELECT * FROM users", 5000) // 5 second timeout
if err != nil {
    log.Fatal(err)
}
```

### Custom Client Options

```go
opts := client.ClientOptions{
    DefaultTimeoutMs:      5000,
    MaxRetries:            3,
    DebugMode:             false,
    LogLevel:              "INFO",
    PoolMinSize:           2,
    PoolMaxSize:           10,
    PoolIdleTimeout:       30000, // milliseconds
    HealthCheckInterval:   30000, // milliseconds
    MaxReconnectAttempts:  10,
}

c := client.NewClient(&opts)
```

## Connection Pooling

Connection pooling enables concurrent database operations with automatic connection lifecycle management.

### Enabling Pooling

Pooling is automatically enabled when `PoolMaxSize > 1`:

```go
opts := client.ClientOptions{
    PoolMinSize:     2,  // Minimum idle connections
    PoolMaxSize:     10, // Maximum total connections
    PoolIdleTimeout: 30000, // Close idle connections after 30s
}

c := client.NewClient(&opts)
```

### Single Connection Mode (Default)

For applications with sequential operations:

```go
opts := client.ClientOptions{
    PoolMaxSize: 1, // Single connection mode
}

c := client.NewClient(&opts)
```

### Pool Statistics

Monitor pool performance in real-time:

```go
// Access pool stats (only available when pooling is enabled)
debugInfo := c.GetDebugInfo()
poolStats := debugInfo["pool"].(map[string]interface{})

fmt.Printf("Active connections: %d\n", poolStats["activeConnections"])
fmt.Printf("Idle connections: %d\n", poolStats["idleConnections"])
fmt.Printf("Total connections: %d\n", poolStats["totalConnections"])
fmt.Printf("Cache hits: %d\n", poolStats["hits"])
fmt.Printf("Cache misses: %d\n", poolStats["misses"])
fmt.Printf("Timeouts: %d\n", poolStats["timeouts"])
```

### Best Practices

- Set `PoolMinSize` to handle typical concurrent load
- Set `PoolMaxSize` to prevent resource exhaustion
- Use `PoolIdleTimeout` to reclaim resources during low activity
- Monitor pool statistics to optimize configuration

## Context Support

All I/O operations support context-based cancellation and timeouts.

### Query Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

result, err := c.Query("SELECT * FROM large_table", 0)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Query timed out")
    }
}
```

### Connection with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := c.Connect(ctx, "syndrdb://remote-host:1776/mydb")
if err != nil {
    log.Fatal("Connection timeout:", err)
}
```

### Graceful Shutdown

```go
// Allow 10 seconds for graceful disconnect
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := c.Disconnect(ctx); err != nil {
    log.Println("Disconnect error:", err)
}
```

## TLS/SSL Configuration

Secure connections using TLS with certificate validation.

### Connection String Configuration

```go
// Enable TLS with default settings
connStr := "syndrdb://localhost:1776/mydb?tls=true"

// Custom CA certificate
connStr := "syndrdb://localhost:1776/mydb?tls=true&tlsCAFile=/path/to/ca.pem"

// Client certificate authentication
connStr := "syndrdb://localhost:1776/mydb?tls=true&tlsCertFile=/path/to/cert.pem&tlsKeyFile=/path/to/key.pem"

// Skip certificate verification (insecure, testing only)
connStr := "syndrdb://localhost:1776/mydb?tls=true&tlsInsecureSkipVerify=true"
```

### Programmatic Configuration

```go
import "crypto/tls"

tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    // Additional TLS settings...
}

opts := client.ClientOptions{
    TLSConfig: tlsConfig,
}

c := client.NewClient(&opts)
```

### TLS State Inspection

```go
debugInfo := c.GetDebugInfo()
if tlsInfo, ok := debugInfo["tls"].(map[string]interface{}); ok {
    fmt.Printf("TLS Version: %d\n", tlsInfo["version"])
    fmt.Printf("Cipher Suite: %d\n", tlsInfo["cipherSuite"])
    fmt.Printf("Server Name: %s\n", tlsInfo["serverName"])
    fmt.Printf("Handshake Complete: %v\n", tlsInfo["handshakeComplete"])
}
```

### Error Handling

The driver provides clear error messages for common TLS issues:

- **Certificate expired**: "TLS certificate has expired"
- **Untrusted certificate**: "TLS certificate signed by unknown authority"
- **Hostname mismatch**: "TLS hostname verification failed"
- **Unknown CA**: "TLS failed to verify certificate"

## Health Monitoring

Automatic connection health checks with reconnection.

### Configuration

```go
opts := client.ClientOptions{
    HealthCheckInterval:   30000, // Ping every 30 seconds
    MaxReconnectAttempts:  10,    // Retry up to 10 times
}

c := client.NewClient(&opts)
```

### Manual Health Check

```go
ctx := context.Background()
if err := c.Ping(ctx); err != nil {
    log.Println("Connection unhealthy:", err)
}
```

### Lifecycle Callbacks

Monitor connection state changes:

```go
opts := client.ClientOptions{
    OnConnected: func(transition client.StateTransition) {
        log.Println("Connected to database")
    },
    OnDisconnected: func(transition client.StateTransition) {
        log.Println("Disconnected from database")
        if transition.Error != nil {
            log.Println("Error:", transition.Error)
        }
    },
    OnReconnecting: func(transition client.StateTransition) {
        log.Println("Attempting to reconnect...")
    },
}

c := client.NewClient(&opts)
```

### Auto-Reconnection

The driver automatically detects connection failures and attempts reconnection with exponential backoff:

- **Initial delay**: 100ms
- **Maximum delay**: 60 seconds
- **Backoff formula**: `delay = min(100ms * 2^attempt, 60s)`

Reconnection is triggered for:
- Network errors (connection reset, broken pipe, EOF)
- Timeout errors
- Protocol errors indicating connection loss

## Structured Logging

JSON-formatted logs with configurable levels and sensitive data redaction.

### Log Levels

Available levels: `DEBUG`, `INFO`, `WARN`, `ERROR`

```go
opts := client.ClientOptions{
    LogLevel: "INFO", // Only INFO and above
}

c := client.NewClient(&opts)
```

### Runtime Log Level Change

```go
c.SetLogLevel("DEBUG") // Enable verbose logging
c.SetLogLevel("ERROR") // Only critical errors
```

### Custom Logger

Implement the `Logger` interface for custom logging:

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
}

type MyLogger struct {
    // Your logger implementation
}

func (l *MyLogger) Info(msg string, fields ...client.Field) {
    // Custom logging logic
}

opts := client.ClientOptions{
    Logger: &MyLogger{},
}

c := client.NewClient(&opts)
```

### Log Output Format

```json
{
  "level": "INFO",
  "timestamp": "2025-12-11T10:30:45Z",
  "message": "connecting to database",
  "connStr": "syndrdb://localhost:1776/mydb",
  "poolEnabled": true
}
```

### Sensitive Data Redaction

The following field names are automatically redacted:
- `password`
- `token`
- `secret`
- `authorization`
- `api_key`

Values are replaced with `[REDACTED]` in log output.

## Debug Mode

Enable detailed diagnostics for troubleshooting.

### Enabling Debug Mode

```go
// At client creation
opts := client.ClientOptions{
    DebugMode: true,
}
c := client.NewClient(&opts)

// Runtime toggle
c.EnableDebugMode()
c.DisableDebugMode()
```

### Debug Information

Get comprehensive state snapshot:

```go
debugInfo := c.GetDebugInfo()

// Connection state
fmt.Printf("State: %s\n", debugInfo["state"])
fmt.Printf("Debug Enabled: %v\n", debugInfo["debugEnabled"])

// Pool information (if enabled)
if pool, ok := debugInfo["pool"].(map[string]interface{}); ok {
    fmt.Printf("Active: %d, Idle: %d\n", 
        pool["activeConnections"], 
        pool["idleConnections"])
}

// Connection details
if conn, ok := debugInfo["connection"].(map[string]interface{}); ok {
    fmt.Printf("Address: %s\n", conn["address"])
    fmt.Printf("Alive: %v\n", conn["alive"])
    fmt.Printf("Last Activity: %s\n", conn["lastActivity"])
}

// Client options
if opts, ok := debugInfo["options"].(map[string]interface{}); ok {
    fmt.Printf("Pool Max Size: %d\n", opts["poolMaxSize"])
    fmt.Printf("Health Check Interval: %d\n", opts["healthCheckInterval"])
}

// Last state transition
if trans, ok := debugInfo["lastTransition"].(map[string]interface{}); ok {
    fmt.Printf("Transition: %s -> %s\n", trans["from"], trans["to"])
    fmt.Printf("Duration: %s\n", trans["duration"])
}
```

### Debug Mode Features

When enabled, debug mode provides:
- **Stack traces** in error messages
- **Verbose command logging** with request/response data
- **Detailed timing information** for all operations
- **Connection state tracking** with transition history

### Error Stack Traces

Errors in debug mode include stack traces:

```go
c.EnableDebugMode()

_, err := c.Query("INVALID QUERY", 0)
if err != nil {
    fmt.Println(err.Error())
    // Output includes stack trace showing exact call path
}
```

## Error Handling

### Error Types

The driver defines three main error types:

**ConnectionError** - Network and connection issues:
```go
if connErr, ok := err.(*client.ConnectionError); ok {
    fmt.Printf("Code: %s\n", connErr.Code)
    fmt.Printf("Type: %s\n", connErr.Type)
    fmt.Printf("Message: %s\n", connErr.Message)
}
```

**ProtocolError** - Wire protocol violations:
```go
if protoErr, ok := err.(*client.ProtocolError); ok {
    fmt.Printf("Expected: %s\n", protoErr.Expected)
    fmt.Printf("Received: %s\n", protoErr.Received)
}
```

**StateError** - Invalid operations for current state:
```go
if stateErr, ok := err.(*client.StateError); ok {
    fmt.Printf("Current State: %s\n", stateErr.ActualState)
    fmt.Printf("Expected State: %s\n", stateErr.ExpectedState)
}
```

### Best Practices

1. **Always use contexts** for timeout and cancellation support
2. **Check errors** after every operation
3. **Handle specific error types** for better error recovery
4. **Use lifecycle callbacks** for connection state monitoring
5. **Enable debug mode** during development and troubleshooting

## WASM Support

The driver can be compiled to WebAssembly for browser usage.

### Building for WASM

```bash
cd wasm
GOOS=js GOARCH=wasm go build -o syndrdb.wasm
```

### JavaScript API

```javascript
// Load the WASM module
const go = new Go();
WebAssembly.instantiateStreaming(fetch("syndrdb.wasm"), go.importObject)
    .then(result => go.run(result.instance));

// Create client
await SyndrDB.createClient({
    defaultTimeoutMs: 5000,
    debugMode: false,
    logLevel: "INFO",
    healthCheckIntervalMs: 30000,
    maxReconnectAttempts: 10
});

// Connect
await SyndrDB.connect("syndrdb://localhost:1776/mydb");

// Execute query
const result = await SyndrDB.query("SELECT * FROM users", 5000);

// Health check
await SyndrDB.ping(2000);

// Connection health
const health = SyndrDB.getConnectionHealth();
console.log(`Connected: ${health.connected}, State: ${health.state}`);

// Logging
SyndrDB.setLogLevel("DEBUG");

// Debug mode
SyndrDB.enableDebugMode();
const debugInfo = await SyndrDB.getDebugInfo();
console.log(JSON.stringify(debugInfo, null, 2));

// Cleanup
await SyndrDB.disconnect();
await SyndrDB.cleanup();
```

### WASM Limitations

- **Connection pooling is not supported** in WASM builds (single connection mode only)
- **Background workers** (health monitoring, cleanup) are disabled
- All operations run on the main thread

## Performance Considerations

### Connection Pooling

- Pool overhead: ~10-20μs per Get/Put operation
- Ideal for concurrent workloads with multiple goroutines
- Use single connection mode for sequential operations

### Context Overhead

- Deadline propagation: ~1-2μs per operation
- Cancellation checks: negligible impact
- Always prefer context for production code

### TLS Performance

- Handshake: ~5-20ms (one-time per connection)
- Encryption overhead: ~5-10% throughput reduction
- Connection reuse amortizes handshake cost

### Logging Performance

- Structured logging: ~2-5μs per log entry
- JSON marshaling: ~1-3μs additional
- Set appropriate log level in production

### Debug Mode Impact

- Debug mode adds ~10-20μs overhead per operation
- Stack trace capture: ~20-50μs per error
- Disable in production for optimal performance

## Examples

### Complete Production Setup

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
)

func main() {
    opts := client.ClientOptions{
        DefaultTimeoutMs:      5000,
        MaxRetries:            3,
        LogLevel:              "INFO",
        PoolMinSize:           5,
        PoolMaxSize:           20,
        PoolIdleTimeout:       60000,
        HealthCheckInterval:   30000,
        MaxReconnectAttempts:  10,
        OnConnected: func(t client.StateTransition) {
            log.Println("Database connected")
        },
        OnDisconnected: func(t client.StateTransition) {
            log.Printf("Database disconnected: %v", t.Error)
        },
        OnReconnecting: func(t client.StateTransition) {
            log.Println("Reconnecting to database...")
        },
    }
    
    c := client.NewClient(&opts)
    
    ctx := context.Background()
    connStr := "syndrdb://localhost:1776/mydb?tls=true&tlsCAFile=/etc/ssl/ca.pem"
    
    if err := c.Connect(ctx, connStr); err != nil {
        log.Fatal("Connection failed:", err)
    }
    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        c.Disconnect(ctx)
    }()
    
    // Execute queries
    result, err := c.Query("SELECT * FROM users WHERE active = true", 5000)
    if err != nil {
        log.Printf("Query error: %v", err)
        return
    }
    
    log.Printf("Query result: %+v", result)
    
    // Check debug info periodically
    debugInfo := c.GetDebugInfo()
    if pool, ok := debugInfo["pool"].(map[string]interface{}); ok {
        log.Printf("Pool stats - Active: %d, Idle: %d, Hits: %d",
            pool["activeConnections"],
            pool["idleConnections"],
            pool["hits"])
    }
}
```

## Troubleshooting

### Connection Issues

1. **Enable debug mode**: `c.EnableDebugMode()`
2. **Check debug info**: `debugInfo := c.GetDebugInfo()`
3. **Verify network connectivity**: `c.Ping(ctx)`
4. **Review logs** at DEBUG level

### Pool Exhaustion

1. **Increase PoolMaxSize** if concurrent load is high
2. **Reduce operation timeout** to free connections faster
3. **Monitor pool statistics** for patterns
4. **Check for connection leaks** (not calling Put)

### TLS Errors

1. **Verify certificate paths** are correct
2. **Check certificate validity** (expiration, hostname)
3. **Test with InsecureSkipVerify** to isolate cert issues
4. **Review TLS state** in debug info

### Performance Issues

1. **Use connection pooling** for concurrent workloads
2. **Disable debug mode** in production
3. **Tune pool sizes** based on statistics
4. **Set appropriate log levels** (INFO or WARN)

## Support and Contribution

For issues, questions, or contributions, please visit:
- GitHub: https://github.com/dan-strohschein/syndrdb-drivers
- Documentation: See README.md files in each package

## Version

This documentation covers Milestone 1 features as of December 2025.
