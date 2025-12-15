package client

// Server Protocol Limitations
//
// This document tracks current server protocol limitations affecting client implementation.
// Information sourced from: https://github.com/dan-strohschein/SyndrDB/blob/main/docs/user/parameterized_queries.md
//
// Last Updated: December 2025

// Parameterized Query Limitations

// TODO: Only SELECT queries support parameters currently. INSERT/UPDATE/DELETE are blocked
// until server adds DML support. Track server issue/PR for DML parameterization feature.
// Workaround: Use string concatenation with manual escaping for mutations (security risk).

// TODO: LIKE/ILIKE pattern matching with wildcards not supported with parameters.
// Only exact equality matching works: WHERE field = $1
// Pattern matching like WHERE field LIKE '%' || $1 || '%' not available.
// Workaround: Use client-side filtering or exact match queries only.

// TODO: Named parameters (:name syntax) not available, only positional $1, $2, $3.
// Go clients typically prefer named parameters for readability with many params.
// Monitor server roadmap for named parameter support.

// TODO: Type hints not implemented ($1::INTEGER explicit casting syntax).
// Parameters are passed as strings and converted based on comparison context.
// This can cause ambiguity with numeric strings vs actual numbers.
// Workaround: Ensure parameter types match expected field types in application code.

// TODO: Batch execution protocol not available. Each EXECUTE command runs one query.
// Cannot execute prepared statement multiple times with different parameter sets
// in single round-trip. This limits bulk operation performance.
// Expected feature: EXECUTE_BATCH stmt_name WITH [[p1, p2], [p3, p4], ...]

// TODO: Cross-session prepared statement sharing not supported. All statements
// are session-scoped and cannot be shared between connections/users.
// This means connection pooling requires re-preparing statements per connection.
// Consider: Add client-side cache keyed by (connection_id, query_hash).

// Transaction Limitations

// ✅ UPDATED: Server now implements transaction protocol (BEGIN TRANSACTION/COMMIT/ROLLBACK).
// Full ACID guarantees are provided through Write-Ahead Logging (WAL) and buffer-aware rollback.
// Transaction ID format: TX_<timestamp>_<random> assigned by server.
// Reference: https://github.com/dan-strohschein/SyndrDB/blob/main/docs/user/transactions.md

// Current transaction limitations:

// TODO: Nested transactions not supported. Each session supports one active transaction at a time.
// Attempting to BEGIN while transaction is active returns "transaction already in progress" error.
// Workaround: Commit or rollback existing transaction before starting new one.

// TODO: Savepoints not supported (SAVEPOINT/ROLLBACK TO/RELEASE commands).
// Cannot implement partial rollback within transaction.
// Limits error recovery strategies in complex transaction workflows.

// TODO: Isolation levels not configurable. Server provides READ COMMITTED isolation only.
// SET TRANSACTION ISOLATION LEVEL command not available.
// Transactions see only committed data from other transactions.

// TODO: Two-phase commit (2PC) protocol not available for distributed transactions.
// Cannot coordinate transactions across multiple SyndrDB instances.
// Blocks distributed system architectures requiring atomic cross-shard operations.

// TODO: DDL operations (CREATE BUNDLE, DROP BUNDLE, etc.) not supported within transactions.
// Schema modifications cannot be rolled back.
// Workaround: Perform schema changes outside of transaction scope.

// Schema and Metadata Limitations

// TODO: Schema introspection queries not available. Cannot query available bundles,
// fields, relationships, indexes programmatically.
// Workaround: Maintain schema definitions in client code or external files.

// TODO: Bundle version tracking not exposed. Cannot detect schema changes to
// invalidate prepared statement cache automatically.
// Risk: Cached statements become invalid after schema migration without notification.

// TODO: Query execution plans (EXPLAIN output) not available for optimization.
// Cannot analyze slow queries or verify index usage from client.
// Limits performance tuning capabilities.

// Performance and Resource Limitations

// TODO: Query memory limits not configurable per-query from client.
// Server enforces global limits (64 MB default, 256 MB admin).
// Cannot request higher limits for known large result sets.

// TODO: Query timeout not configurable per-query from client.
// Server enforces global timeouts (300s default, 600s admin).
// Cannot extend timeout for known long-running analytical queries.

// TODO: Streaming result sets not supported. All query results loaded into memory.
// Cannot process large result sets incrementally with cursor/iterator pattern.
// Blocks processing of multi-GB result sets that exceed memory limits.

// TODO: Compression not available for protocol messages.
// Large parameter values or result sets consume significant bandwidth.
// Consider: Client-side compression before sending if protocol adds support.

// Security Limitations

// TODO: Parameter value encryption not supported. Sensitive values (passwords,
// SSNs, credit cards) sent as plaintext parameters even over TLS.
// Risk: Man-in-the-middle attacks could expose sensitive data if TLS compromised.
// Workaround: Hash sensitive values before sending, store hashes server-side.

// TODO: Prepared statement permissions not granular. Cannot restrict which
// statements a user can prepare/execute based on roles.
// All authenticated users can prepare any valid SELECT query.

// Feature Availability Matrix
//
// | Feature                    | Status      | Server Version | Client Support |
// |----------------------------|-------------|----------------|----------------|
// | SELECT with parameters     | ✅ Available | v0.1.0+        | Implemented    |
// | INSERT with parameters     | ❌ Blocked   | Planned        | TODO           |
// | UPDATE with parameters     | ❌ Blocked   | Planned        | TODO           |
// | DELETE with parameters     | ❌ Blocked   | Planned        | TODO           |
// | LIKE/ILIKE with parameters | ❌ Blocked   | Planned        | TODO           |
// | Named parameters (:name)   | ❌ Blocked   | Planned        | TODO           |
// | Type hints ($1::type)      | ❌ Blocked   | Planned        | TODO           |
// | Batch execution            | ❌ Blocked   | Planned        | TODO           |
// | BEGIN TRANSACTION          | ✅ Available | Current        | Implemented    |
// | COMMIT                     | ✅ Available | Current        | Implemented    |
// | ROLLBACK                   | ✅ Available | Current        | Implemented    |
// | Nested transactions        | ❌ Blocked   | Planned        | TODO           |
// | Isolation levels           | ❌ Blocked   | Planned        | Stubbed        |
// | Savepoints                 | ❌ Blocked   | Planned        | TODO           |
// | Query streaming            | ❌ Blocked   | Not Started    | TODO           |
// | Schema introspection       | ❌ Blocked   | Not Started    | TODO           |
//
// Refer to SyndrDB server documentation for transaction details:
// https://github.com/dan-strohschein/SyndrDB/blob/main/docs/user/transactions.md
//
// Refer to SyndrDB server roadmap for planned feature timeline:
// https://github.com/dan-strohschein/SyndrDB/blob/main/ROADMAP.md
