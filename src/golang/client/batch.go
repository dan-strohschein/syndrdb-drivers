package client

// TODO: Implement batch operations when server extends parameterized query support
// to DML operations. Current server limitation per parameterized_queries.md section
// 'Current Limitations': INSERT/UPDATE/DELETE with parameters not yet supported,
// only SELECT queries work.
//
// Planned design: Batch.Add(operation, params) accumulates operations, Execute()
// sends all in transaction for atomicity. Use PREPARE once, EXECUTE multiple times
// pattern from server best practices documentation.
//
// Example usage:
//   batch := client.NewBatch()
//   for _, user := range users {
//       batch.Add("INSERT INTO Users (Name, Email) VALUES ($1, $2)", user.Name, user.Email)
//   }
//   results, err := batch.Execute(ctx)
//
// Support BulkInsert(bundle, records) helper generating single INSERT with multiple
// VALUES clauses when protocol supports it. Add partial failure handling returning
// detailed error info per operation. Reference task2.md Feature 2.5 acceptance
// criteria expecting 10x performance improvement over individual operations.
//
// Implementation considerations:
// - Batch size limits to prevent memory exhaustion
// - Transaction isolation: all succeed or all fail
// - Progress reporting for long-running batches
// - Parallel execution where operations don't conflict
// - Retry logic for transient failures
//
// TODO: Add support for batch SELECT operations when server implements batch protocol.
// Design: prepare single statement, execute with array of parameter sets, receive
// array of result sets. Reduces network round-trips significantly.
