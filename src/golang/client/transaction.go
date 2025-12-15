package client

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// IsolationLevel represents transaction isolation levels.
type IsolationLevel int

const (
	// ReadUncommitted allows dirty reads.
	ReadUncommitted IsolationLevel = iota
	// ReadCommitted prevents dirty reads.
	ReadCommitted
	// RepeatableRead prevents non-repeatable reads.
	RepeatableRead
	// Serializable provides full isolation.
	Serializable
)

// String returns the string representation of the isolation level.
func (l IsolationLevel) String() string {
	switch l {
	case ReadUncommitted:
		return "READ UNCOMMITTED"
	case ReadCommitted:
		return "READ COMMITTED"
	case RepeatableRead:
		return "REPEATABLE READ"
	case Serializable:
		return "SERIALIZABLE"
	default:
		return "UNKNOWN"
	}
}

// Transaction represents a database transaction with ACID properties.
// Binds to a specific connection for the transaction lifetime.
type Transaction struct {
	id         string
	connID     string // Connection identifier for affinity tracking
	conn       ConnectionInterface
	client     *Client
	isolation  IsolationLevel
	committed  bool
	rolledBack bool
	startedAt  time.Time
	mu         sync.Mutex
}

// Query executes a query within the transaction context.
func (tx *Transaction) Query(query string, timeoutMs int) (interface{}, error) {
	tx.mu.Lock()
	if tx.committed {
		tx.mu.Unlock()
		return nil, ErrTransactionAlreadyCommitted(tx.id)
	}
	if tx.rolledBack {
		tx.mu.Unlock()
		return nil, ErrTransactionAlreadyRolledBack(tx.id)
	}
	tx.mu.Unlock()

	ctx := context.Background()
	if timeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
	}

	if err := tx.conn.SendCommand(ctx, query); err != nil {
		return nil, &QueryError{
			Code:    "E_TX_QUERY_FAILED",
			Type:    "QueryError",
			Message: "failed to execute query in transaction",
			Details: map[string]interface{}{
				"transaction_id": tx.id,
			},
			Query: query,
			Cause: err,
		}
	}

	return tx.conn.ReceiveResponse(ctx)
}

// QueryWithParams executes a parameterized query within the transaction.
func (tx *Transaction) QueryWithParams(query string, params ...interface{}) (interface{}, error) {
	tx.mu.Lock()
	if tx.committed {
		tx.mu.Unlock()
		return nil, ErrTransactionAlreadyCommitted(tx.id)
	}
	if tx.rolledBack {
		tx.mu.Unlock()
		return nil, ErrTransactionAlreadyRolledBack(tx.id)
	}
	tx.mu.Unlock()

	// Prepare statement within transaction
	stmt, err := tx.prepareInternal(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	return stmt.Execute(params...)
}

// Prepare creates a prepared statement within the transaction context.
func (tx *Transaction) Prepare(query string) (*Statement, error) {
	tx.mu.Lock()
	if tx.committed {
		tx.mu.Unlock()
		return nil, ErrTransactionAlreadyCommitted(tx.id)
	}
	if tx.rolledBack {
		tx.mu.Unlock()
		return nil, ErrTransactionAlreadyRolledBack(tx.id)
	}
	tx.mu.Unlock()

	return tx.prepareInternal(query)
}

// prepareInternal handles statement preparation without state checks.
func (tx *Transaction) prepareInternal(query string) (*Statement, error) {
	// Generate statement name with transaction prefix
	stmtName := fmt.Sprintf("tx_%s_stmt_%d", tx.id[:8], time.Now().UnixNano())

	if err := validateStatementName(stmtName); err != nil {
		return nil, err
	}

	command := fmt.Sprintf("PREPARE %s AS %s", stmtName, query)
	ctx := context.Background()

	if err := tx.conn.SendCommand(ctx, command); err != nil {
		return nil, &StatementError{
			QueryError: QueryError{
				Code:    "E_PREPARE_FAILED",
				Type:    "StatementError",
				Message: "failed to prepare statement in transaction",
				Details: map[string]interface{}{
					"transaction_id": tx.id,
				},
				Query: query,
				Cause: err,
			},
			StatementName: stmtName,
		}
	}

	response, err := tx.conn.ReceiveResponse(ctx)
	if err != nil {
		return nil, err
	}

	// Parse parameter count from response
	paramCount := countPlaceholders(query)

	stmt := &Statement{
		name:       stmtName,
		query:      query,
		paramCount: paramCount,
		conn:       tx.conn,
		closed:     false,
		createdAt:  time.Now(),
	}

	// Log success for debugging
	if tx.client != nil && tx.client.logger != nil {
		tx.client.logger.Debug("prepared statement in transaction",
			String("tx_id", tx.id),
			String("stmt_name", stmtName),
			Int("param_count", paramCount))
	}

	_ = response // TODO: Parse server response for validation

	return stmt, nil
}

// Commit commits the transaction and releases the connection back to the pool.
func (tx *Transaction) Commit() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return ErrTransactionAlreadyCommitted(tx.id)
	}
	if tx.rolledBack {
		return ErrTransactionAlreadyRolledBack(tx.id)
	}

	ctx := context.Background()
	if err := tx.conn.SendCommand(ctx, "COMMIT;"); err != nil {
		return &TransactionError{
			Code:          "E_COMMIT_FAILED",
			Type:          "TransactionError",
			Message:       "failed to commit transaction",
			TransactionID: tx.id,
			State:         "active",
			Cause:         err,
		}
	}

	if _, err := tx.conn.ReceiveResponse(ctx); err != nil {
		return &TransactionError{
			Code:          "E_COMMIT_RESPONSE_FAILED",
			Type:          "TransactionError",
			Message:       "failed to receive commit response",
			TransactionID: tx.id,
			Cause:         err,
		}
	}

	tx.committed = true

	// Remove from active transactions and return connection to pool
	if tx.client != nil {
		tx.client.activeTransactions.Delete(tx.id)
		if tx.client.poolEnabled && tx.client.pool != nil {
			tx.client.pool.Put(tx.conn)
		}
	}

	return nil
}

// Rollback rolls back the transaction and releases the connection.
func (tx *Transaction) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return ErrTransactionAlreadyCommitted(tx.id)
	}
	if tx.rolledBack {
		return nil // Already rolled back, no-op
	}

	ctx := context.Background()
	if err := tx.conn.SendCommand(ctx, "ROLLBACK;"); err != nil {
		return &TransactionError{
			Code:          "E_ROLLBACK_FAILED",
			Type:          "TransactionError",
			Message:       "failed to rollback transaction",
			TransactionID: tx.id,
			State:         "active",
			Cause:         err,
		}
	}

	if _, err := tx.conn.ReceiveResponse(ctx); err != nil {
		// Log but don't fail - rollback intent is clear
		if tx.client != nil && tx.client.logger != nil {
			tx.client.logger.Warn("failed to receive rollback response",
				String("tx_id", tx.id),
				Error("error", err))
		}
	}

	tx.rolledBack = true

	// Remove from active transactions and return connection to pool
	if tx.client != nil {
		tx.client.activeTransactions.Delete(tx.id)
		if tx.client.poolEnabled && tx.client.pool != nil {
			tx.client.pool.Put(tx.conn)
		}
	}

	return nil
}

// ID returns the transaction ID.
func (tx *Transaction) ID() string {
	return tx.id
}

// ConnectionID returns the connection ID this transaction is bound to
func (tx *Transaction) ConnectionID() string {
	return tx.connID
}

// getState returns the current transaction state as a string.
func (tx *Transaction) getState() string {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return "committed"
	}
	if tx.rolledBack {
		return "rolledback"
	}
	return "active"
}

// transactionContext holds transaction metadata for monitoring.
type transactionContext struct {
	tx        *Transaction
	conn      ConnectionInterface
	startedAt time.Time
}

// InTransaction executes a function within a transaction with automatic commit/rollback.
// Commits on success, rolls back on error or panic.
func (c *Client) InTransaction(ctx context.Context, fn func(*Transaction) error) error {
	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}

	// Set up panic recovery with rollback
	defer func() {
		if r := recover(); r != nil {
			rollbackErr := tx.Rollback()

			state := tx.getState()
			duration := time.Since(tx.startedAt)

			c.logger.Warn("transaction rolled back due to panic",
				String("transaction_id", tx.id),
				String("state", state),
				Duration("duration", duration),
				Error("panic", fmt.Errorf("%v", r)),
				Error("rollback_error", rollbackErr),
				String("stack", string(debug.Stack())))

			// Re-throw panic to preserve stack trace
			panic(r)
		}
	}()

	// Execute user function
	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			c.logger.Error("failed to rollback transaction after error",
				String("tx_id", tx.id),
				Error("original_error", err),
				Error("rollback_error", rollbackErr))
		}
		return err
	}

	// Commit on success
	return tx.Commit()
}

// TODO: Implement savepoints with SAVEPOINT/ROLLBACK TO/RELEASE commands when
// server supports nested transactions. Design: tx.Savepoint(name), tx.RollbackTo(name),
// tx.ReleaseSavepoint(name). Track savepoint stack per transaction for proper nesting.
// NOTE: Server currently doesn't support savepoints (see limitations.md)

// TODO: Add transaction isolation level configuration when server protocol is extended.
// Currently server provides READ COMMITTED isolation only.
// Monitor server roadmap for configurable isolation levels.
