package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// Client is the main SyndrDB client supporting both single and pooled connections.
type Client struct {
	conn               *Connection     // Used in single-connection mode
	pool               *ConnectionPool // Used in pooled mode
	poolEnabled        bool
	connFactory        func(ctx context.Context) (ConnectionInterface, error)
	opts               ClientOptions
	stateMgr           *StateManager
	connStr            string
	logger             Logger
	debugMode          atomic.Bool
	activeTransactions sync.Map // map[string]*transactionContext
	stmtCache          *StatementCache
	schemaValidator    *SchemaValidator // Schema validation for QueryBuilder
	txMonitorDone      chan struct{}
	hooks              []hookEntry  // Registered hooks in execution order
	hooksMu            sync.RWMutex // Protects hooks slice
}

// NewClient creates a new SyndrDB client with the given options.
// If opts is nil, default options are used.
func NewClient(opts *ClientOptions) *Client {
	if opts == nil {
		defaultOpts := DefaultOptions()
		opts = &defaultOpts
	}

	// Initialize logger
	logger := opts.Logger
	if logger == nil {
		logger = NewLogger(opts.LogLevel, nil)
	}

	// Initialize statement cache
	cacheSize := opts.PreparedStatementCacheSize
	if cacheSize == 0 {
		cacheSize = 100 // Default cache size
	}

	client := &Client{
		opts:          *opts,
		stateMgr:      NewStateManager(),
		logger:        logger,
		poolEnabled:   opts.PoolMaxSize > 1,
		stmtCache:     NewStatementCache(cacheSize),
		txMonitorDone: make(chan struct{}),
	}

	client.debugMode.Store(opts.DebugMode)

	// Initialize schema validator
	client.schemaValidator = NewSchemaValidator(client, opts.SchemaCacheTTL, opts.PreloadSchema)

	// Wire up lifecycle callbacks if provided
	if opts.OnConnected != nil || opts.OnDisconnected != nil || opts.OnReconnecting != nil {
		client.stateMgr.OnStateChange(func(transition StateTransition) {
			switch transition.To {
			case CONNECTED:
				if opts.OnConnected != nil {
					opts.OnConnected(transition)
				}
			case DISCONNECTED:
				if transition.From != DISCONNECTED && opts.OnDisconnected != nil {
					opts.OnDisconnected(transition)
				}
			case CONNECTING:
				if transition.From == DISCONNECTED && opts.OnReconnecting != nil {
					opts.OnReconnecting(transition)
				}
			}
		})
	}

	return client
}

// Connect establishes a connection to the SyndrDB server.
// Connection string format: syndrdb://host:port/database
func (c *Client) Connect(ctx context.Context, connStr string) error {
	c.logger.Info("connecting to database", String("connStr", connStr), Bool("poolEnabled", c.poolEnabled))

	// Transition to CONNECTING state
	err := c.stateMgr.TransitionTo(CONNECTING, nil, map[string]interface{}{
		"reason":           "user_initiated",
		"connectionString": connStr,
		"attempt":          1,
	})
	if err != nil {
		return err
	}

	// Validate connection string format
	if !strings.HasPrefix(connStr, "syndrdb://") {
		c.stateMgr.TransitionTo(DISCONNECTED, nil, map[string]interface{}{
			"reason": "error",
		})
		return &ConnectionError{
			Code:    "INVALID_SCHEME",
			Type:    "CONNECTION_ERROR",
			Message: "connection string must use 'syndrdb://' scheme",
			Details: map[string]interface{}{
				"connectionString": connStr,
				"expected":         "syndrdb://",
			},
		}
	}

	// Extract host:port from connection string
	// Format: syndrdb://HOST:PORT:DATABASE:USERNAME:PASSWORD;
	withoutScheme := strings.TrimPrefix(connStr, "syndrdb://")
	parts := strings.Split(withoutScheme, ":")
	if len(parts) < 2 {
		c.stateMgr.TransitionTo(DISCONNECTED, nil, map[string]interface{}{
			"reason": "error",
		})
		return &ConnectionError{
			Code:    "INVALID_CONNECTION_STRING",
			Type:    "CONNECTION_ERROR",
			Message: "invalid connection string format",
			Details: map[string]interface{}{
				"connectionString": connStr,
				"expected":         "syndrdb://HOST:PORT:DATABASE:USERNAME:PASSWORD;",
			},
		}
	}

	address := parts[0] + ":" + parts[1] // HOST:PORT
	c.connStr = connStr

	// Parse TLS options from connection string query parameters
	tlsOpts := parseTLSOptions(connStr)
	if val, ok := tlsOpts["tls"]; ok && (val == "true" || val == "require") {
		c.opts.TLSEnabled = true
		c.logger.Info("TLS enabled via connection string")
	}
	if val, ok := tlsOpts["tlsCAFile"]; ok {
		c.opts.TLSCAFile = val
	}
	if val, ok := tlsOpts["tlsCert"]; ok {
		c.opts.TLSCertFile = val
	}
	if val, ok := tlsOpts["tlsKey"]; ok {
		c.opts.TLSKeyFile = val
	}
	if val, ok := tlsOpts["tlsInsecureSkipVerify"]; ok && val == "true" {
		c.opts.TLSInsecureSkipVerify = true
		c.logger.Warn("TLS certificate verification disabled - USE ONLY FOR TESTING")
	}

	// Create connection factory that will be reused for reconnection
	c.connFactory = func(ctx context.Context) (ConnectionInterface, error) {
		return c.createAndAuthenticateConnection(ctx, address, connStr)
	}

	// Use pool mode if configured
	if c.poolEnabled {
		return c.connectWithPool(ctx)
	}

	// Use single connection mode
	return c.connectSingle(ctx)
}

// createAndAuthenticateConnection creates a new connection and performs authentication.
func (c *Client) createAndAuthenticateConnection(ctx context.Context, address, connStr string) (ConnectionInterface, error) {
	conn, err := NewConnection(ctx, address, c.opts)
	if err != nil {
		return nil, err
	}

	// Send connection string
	err = conn.SendCommand(ctx, connStr)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Read welcome response (should contain S0001)
	welcomeResp, err := conn.ReceiveResponse(ctx)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Check for S0001 success code
	welcomeStr := fmt.Sprintf("%v", welcomeResp)
	if !strings.Contains(welcomeStr, "S0001") {
		conn.Close()
		return nil, &ConnectionError{
			Code:    "AUTH_FAILED",
			Type:    "CONNECTION_ERROR",
			Message: fmt.Sprintf("authentication failed: unexpected welcome response \"%s\"", welcomeStr),
			Details: map[string]interface{}{
				"response": welcomeStr,
			},
		}
	}

	// Read authentication success JSON response
	authResp, err := conn.ReceiveResponse(ctx)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Parse and validate authentication response
	authData, ok := authResp.(map[string]interface{})
	if !ok {
		conn.Close()
		return nil, &ConnectionError{
			Code:    "AUTH_FAILED",
			Type:    "CONNECTION_ERROR",
			Message: fmt.Sprintf("authentication failed: unexpected response type %T", authResp),
			Details: map[string]interface{}{
				"response": authResp,
			},
		}
	}

	status, ok := authData["status"].(string)
	if !ok || status != "success" {
		conn.Close()
		message := "unknown error"
		if msg, ok := authData["message"].(string); ok {
			message = msg
		}
		return nil, &ConnectionError{
			Code:    "AUTH_FAILED",
			Type:    "CONNECTION_ERROR",
			Message: fmt.Sprintf("authentication failed: %s", message),
			Details: map[string]interface{}{
				"response": authData,
			},
		}
	}

	return conn, nil
}

// connectWithPool initializes connection pool.
func (c *Client) connectWithPool(ctx context.Context) error {
	c.logger.Info("initializing connection pool",
		Int("minIdle", c.opts.PoolMinSize),
		Int("maxOpen", c.opts.PoolMaxSize))

	c.pool = NewConnectionPool(
		c.connFactory,
		c.opts.PoolMinSize,
		c.opts.PoolMaxSize,
		c.opts.PoolIdleTimeout,
		c.opts.HealthCheckInterval,
	)

	if err := c.pool.Initialize(ctx); err != nil {
		c.logger.Error("failed to initialize connection pool", Error("error", err))
		c.stateMgr.TransitionTo(DISCONNECTED, err, map[string]interface{}{
			"reason": "pool_init_failed",
		})
		return err
	}

	c.logger.Info("connection pool initialized successfully")

	// Recreate transaction monitor channel (in case of reconnect)
	c.txMonitorDone = make(chan struct{})

	// Start transaction timeout monitor
	go c.transactionTimeoutMonitor()

	c.stateMgr.TransitionTo(CONNECTED, nil, map[string]interface{}{
		"reason": "user_initiated",
		"mode":   "pool",
	})
	return nil
}

// connectSingle establishes a single persistent connection with retries.
func (c *Client) connectSingle(ctx context.Context) error {
	var lastErr error
	backoff := 100 * time.Millisecond

	for attempt := 1; attempt <= c.opts.MaxRetries; attempt++ {
		c.logger.Debug("attempting connection", Int("attempt", attempt))

		// Check context cancellation
		select {
		case <-ctx.Done():
			c.stateMgr.TransitionTo(DISCONNECTED, ctx.Err(), map[string]interface{}{
				"reason": "context_cancelled",
			})
			return ctx.Err()
		default:
		}

		conn, err := c.connFactory(ctx)
		if err == nil {
			// Success - fully authenticated
			c.conn = conn.(*Connection)
			c.logger.Info("connection established", String("remoteAddr", conn.RemoteAddr()))

			// Recreate transaction monitor channel (in case of reconnect)
			c.txMonitorDone = make(chan struct{})

			// Start transaction timeout monitor
			go c.transactionTimeoutMonitor()

			c.stateMgr.TransitionTo(CONNECTED, nil, map[string]interface{}{
				"reason":     "user_initiated",
				"remoteAddr": conn.RemoteAddr(),
				"mode":       "single",
			})
			return nil
		}

		lastErr = err
		c.logger.Warn("connection attempt failed",
			Int("attempt", attempt),
			Error("error", err))

		// If not last attempt, wait and retry
		if attempt < c.opts.MaxRetries {
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff

			// Update metadata with retry attempt
			c.stateMgr.TransitionTo(CONNECTING, nil, map[string]interface{}{
				"reason":           "error",
				"connectionString": c.connStr,
				"attempt":          attempt + 1,
			})
		}
	}

	// All retries failed
	c.logger.Error("all connection attempts failed", Error("error", lastErr))
	c.stateMgr.TransitionTo(DISCONNECTED, lastErr, map[string]interface{}{
		"reason":  "error",
		"attempt": c.opts.MaxRetries,
	})

	return lastErr
}

// Disconnect closes the connection gracefully.
func (c *Client) Disconnect(ctx context.Context) error {
	c.logger.Info("disconnecting from database")

	if c.stateMgr.GetState() != CONNECTED {
		return ErrInvalidState("Disconnect", CONNECTED, c.stateMgr.GetState())
	}

	// Transition to DISCONNECTING
	err := c.stateMgr.TransitionTo(DISCONNECTING, nil, map[string]interface{}{
		"reason": "user_initiated",
	})
	if err != nil {
		return err
	}

	// Stop transaction timeout monitor
	close(c.txMonitorDone)

	// Rollback any active transactions
	c.activeTransactions.Range(func(key, value interface{}) bool {
		txCtx := value.(*transactionContext)
		if err := txCtx.tx.Rollback(); err != nil {
			c.logger.Error("failed to rollback transaction during disconnect",
				String("tx_id", txCtx.tx.id),
				Error("error", err))
		}
		return true
	})

	// Clear prepared statement cache
	if c.stmtCache != nil {
		if err := c.stmtCache.Clear(); err != nil {
			c.logger.Warn("failed to clear statement cache", Error("error", err))
		}
	}

	// Check context with timeout for graceful shutdown
	select {
	case <-ctx.Done():
		c.logger.Warn("disconnect context cancelled, forcing shutdown")
		// Force close if context expired
		if c.poolEnabled && c.pool != nil {
			c.pool.Close(ctx)
			c.pool = nil
		} else if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.stateMgr.TransitionTo(DISCONNECTED, ctx.Err(), map[string]interface{}{
			"reason": "context_timeout",
		})
		return ctx.Err()
	default:
	}

	// Close connection or pool
	var closeErr error
	if c.poolEnabled && c.pool != nil {
		c.logger.Debug("closing connection pool")
		closeErr = c.pool.Close(ctx)
		c.pool = nil
	} else if c.conn != nil {
		c.logger.Debug("closing single connection")
		closeErr = c.conn.Close()
		c.conn = nil
	}

	if closeErr != nil {
		c.logger.Error("error during disconnect", Error("error", closeErr))
	} else {
		c.logger.Info("disconnected successfully")
	}

	// Transition to DISCONNECTED
	c.stateMgr.TransitionTo(DISCONNECTED, closeErr, map[string]interface{}{
		"reason": "user_initiated",
	})

	return closeErr
}

// GetState returns the current connection state.
func (c *Client) GetState() ConnectionState {
	return c.stateMgr.GetState()
}

// GetLastTransition returns the most recent state transition.
func (c *Client) GetLastTransition() StateTransition {
	return c.stateMgr.GetLastTransition()
}

// OnStateChange registers a handler to be called on state transitions.
func (c *Client) OnStateChange(handler StateChangeHandler) {
	c.stateMgr.OnStateChange(handler)
}

// GetVersion returns the build version of the client.
func (c *Client) GetVersion() string {
	return Version
}

// sendCommand sends a command and validates connection state.
func (c *Client) sendCommand(ctx context.Context, command string) (interface{}, error) {
	if c.stateMgr.GetState() != CONNECTED {
		return nil, ErrInvalidState("sendCommand", CONNECTED, c.stateMgr.GetState())
	}

	start := time.Now()
	traceID := uuid.New().String()
	debugMode := c.IsDebugMode()

	// Initialize hook context
	hookCtx := &HookContext{
		Command:     command,
		CommandType: inferCommandType(command),
		Params:      nil,
		StartTime:   start,
		Metadata:    make(map[string]interface{}),
		TraceID:     traceID,
	}

	// Execute before hooks
	if err := c.executeBeforeHooks(ctx, hookCtx); err != nil {
		return nil, err
	}

	// Use potentially modified command from hooks
	command = hookCtx.Command

	// Debug logging: log raw command before sending
	if debugMode {
		c.logger.Debug("sending raw command",
			String("command", command),
			String("trace_id", traceID),
			String("timestamp", start.Format(time.RFC3339Nano)))
	}

	// Use pool mode if enabled
	if c.poolEnabled && c.pool != nil {
		// Debug logging: acquiring connection from pool
		if debugMode {
			poolStats := c.pool.Stats()
			c.logger.Debug("acquiring connection from pool",
				String("trace_id", traceID),
				Int("active_connections", int(poolStats.ActiveConnections.Load())),
				Int("idle_connections", int(poolStats.IdleConnections.Load())))
		}

		conn, err := c.pool.Get(ctx)
		if err != nil {
			c.logger.Error("failed to acquire connection from pool", Error("error", err))

			// Execute after hooks with error
			hookCtx.Error = err
			hookCtx.Duration = time.Since(start)
			c.executeAfterHooks(ctx, hookCtx)

			return nil, err
		}
		defer func() {
			c.pool.Put(conn)
			// Debug logging: returning connection to pool
			if debugMode {
				c.logger.Debug("returned connection to pool",
					String("trace_id", traceID),
					String("remote_addr", conn.RemoteAddr()))
			}
		}()

		if err := conn.SendCommand(ctx, command); err != nil {
			c.logger.Error("failed to send command", Error("error", err))

			// Execute after hooks with error
			hookCtx.Error = err
			hookCtx.Duration = time.Since(start)
			c.executeAfterHooks(ctx, hookCtx)

			return nil, err
		}

		result, err := conn.ReceiveResponse(ctx)
		duration := time.Since(start)

		// Update hook context with result
		hookCtx.Result = result
		hookCtx.Error = err
		hookCtx.Duration = duration

		// Debug logging: log raw response
		if debugMode {
			c.logger.Debug("received raw response",
				String("trace_id", traceID),
				String("response", fmt.Sprintf("%v", result)),
				Duration("elapsed", duration),
				Bool("success", err == nil))
		}

		// Execute after hooks
		if hookErr := c.executeAfterHooks(ctx, hookCtx); hookErr != nil {
			// Hook error replaces original error
			err = hookErr
		}

		if err != nil {
			c.logger.Error("failed to receive response",
				Error("error", err),
				Duration("duration", duration))
			return nil, err
		}

		c.logger.Debug("command executed",
			String("command", command),
			String("trace_id", traceID),
			Duration("duration", duration))
		return result, nil
	}

	// Use single connection mode
	if c.conn == nil {
		err := &ConnectionError{
			Code:    "NO_CONNECTION",
			Type:    "CONNECTION_ERROR",
			Message: "no active connection",
		}

		// Execute after hooks with error
		hookCtx.Error = err
		hookCtx.Duration = time.Since(start)
		c.executeAfterHooks(ctx, hookCtx)

		return nil, err
	}

	err := c.conn.SendCommand(ctx, command)
	if err != nil {
		c.logger.Error("failed to send command", Error("error", err))

		// Execute after hooks with error
		hookCtx.Error = err
		hookCtx.Duration = time.Since(start)
		c.executeAfterHooks(ctx, hookCtx)

		return nil, err
	}

	result, err := c.conn.ReceiveResponse(ctx)
	duration := time.Since(start)

	// Update hook context with result
	hookCtx.Result = result
	hookCtx.Error = err
	hookCtx.Duration = duration

	// Debug logging: log raw response
	if debugMode {
		c.logger.Debug("received raw response",
			String("trace_id", traceID),
			String("response", fmt.Sprintf("%v", result)),
			Duration("elapsed", duration),
			Bool("success", err == nil))
	}

	// Execute after hooks
	if hookErr := c.executeAfterHooks(ctx, hookCtx); hookErr != nil {
		// Hook error replaces original error
		err = hookErr
	}

	// Detect DDL operations and invalidate schema cache
	if err == nil && c.schemaValidator != nil && DetectDDL(command) {
		c.logger.Debug("DDL operation detected, invalidating schema cache",
			String("command", command),
			String("trace_id", traceID))
		c.schemaValidator.InvalidateCache()

		// Trigger background schema refresh if auto-refresh is enabled
		if c.schemaValidator.autoRefresh {
			go func() {
				refreshCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if refreshErr := c.schemaValidator.fetchSchema(refreshCtx); refreshErr != nil {
					c.logger.Warn("failed to refresh schema after DDL",
						Error("error", refreshErr),
						String("command", command))
				} else {
					c.logger.Debug("schema refreshed after DDL",
						String("command", command))
				}
			}()
		}
	}

	if err != nil {
		c.logger.Error("failed to receive response",
			Error("error", err),
			Duration("duration", duration))
		return nil, err
	}

	c.logger.Debug("command executed",
		String("command", command),
		String("trace_id", traceID),
		Duration("duration", duration))
	return result, nil
}

// Query executes a query command.
func (c *Client) Query(query string, timeoutMs int) (interface{}, error) {
	if c.stateMgr.GetState() != CONNECTED {
		return nil, ErrInvalidState("Query", CONNECTED, c.stateMgr.GetState())
	}

	ctx := context.Background()
	if timeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
	}

	return c.sendCommand(ctx, query)
}

// Mutate executes a mutation command.
func (c *Client) Mutate(mutation string, timeoutMs int) (interface{}, error) {
	if c.stateMgr.GetState() != CONNECTED {
		return nil, ErrInvalidState("Mutate", CONNECTED, c.stateMgr.GetState())
	}

	ctx := context.Background()
	if timeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
	}

	return c.sendCommand(ctx, mutation)
}

// Ping performs a health check on the connection.
// Returns nil if the connection is healthy, an error otherwise.
func (c *Client) Ping(ctx context.Context) error {
	if c.stateMgr.GetState() != CONNECTED {
		return ErrInvalidState("Ping", CONNECTED, c.stateMgr.GetState())
	}

	// Use pool mode if enabled
	if c.poolEnabled && c.pool != nil {
		conn, err := c.pool.Get(ctx)
		if err != nil {
			return err
		}
		defer c.pool.Put(conn)
		return conn.Ping(ctx)
	}

	// Use single connection mode
	if c.conn == nil {
		return &ConnectionError{
			Code:    "NO_CONNECTION",
			Type:    "CONNECTION_ERROR",
			Message: "no active connection",
		}
	}

	return c.conn.Ping(ctx)
}

// SetLogLevel changes the logging level at runtime.
// Valid levels: DEBUG, INFO, WARN, ERROR.
func (c *Client) SetLogLevel(level string) {
	parsedLevel := ParseLogLevel(level)

	// Update options
	c.opts.LogLevel = level

	// If using default logger, update its level via recreating
	if _, ok := c.logger.(*defaultLogger); ok {
		c.logger = NewLogger(parsedLevel.String(), nil)
		c.logger.Info("log level changed", String("newLevel", level))
	}
}

// Prepare creates a prepared statement with parameter placeholders.
// Statement names must be alphanumeric with underscores only.
// Sends PREPARE command to server following parameterized_queries.md protocol.
func (c *Client) Prepare(ctx context.Context, name, query string) (*Statement, error) {
	if c.stateMgr.GetState() != CONNECTED {
		return nil, ErrInvalidState("Prepare", CONNECTED, c.stateMgr.GetState())
	}

	// Validate statement name
	if err := validateStatementName(name); err != nil {
		return nil, err
	}

	// Count expected parameters
	paramCount := countPlaceholders(query)

	command := fmt.Sprintf("PREPARE %s AS %s", name, query)

	// Get connection
	var conn ConnectionInterface
	var err error
	returnConn := false

	if c.poolEnabled && c.pool != nil {
		conn, err = c.pool.Get(ctx)
		if err != nil {
			return nil, err
		}
		returnConn = true
	} else {
		conn = c.conn
	}

	// Send PREPARE command
	if err := conn.SendCommand(ctx, command); err != nil {
		if returnConn {
			c.pool.Put(conn)
		}
		return nil, &StatementError{
			QueryError: QueryError{
				Code:    "E_PREPARE_FAILED",
				Type:    "StatementError",
				Message: fmt.Sprintf("failed to prepare statement %s", name),
				Query:   query,
				Cause:   err,
			},
			StatementName: name,
		}
	}

	// Receive response
	response, err := conn.ReceiveResponse(ctx)
	if err != nil {
		if returnConn {
			c.pool.Put(conn)
		}
		return nil, &StatementError{
			QueryError: QueryError{
				Code:    "E_PREPARE_RESPONSE_FAILED",
				Type:    "StatementError",
				Message: fmt.Sprintf("failed to receive prepare response for %s", name),
				Query:   query,
				Cause:   err,
			},
			StatementName: name,
		}
	}

	stmt := &Statement{
		name:       name,
		query:      query,
		paramCount: paramCount,
		conn:       conn,
		closed:     false,
		createdAt:  time.Now(),
	}

	// Add to cache
	if err := c.stmtCache.Add(stmt); err != nil {
		c.logger.Warn("failed to cache prepared statement",
			String("stmt_name", name),
			Error("error", err))
	}

	c.logger.Debug("prepared statement",
		String("name", name),
		Int("param_count", paramCount),
		String("query", query))

	_ = response // TODO: Parse server response for validation

	// Don't return connection yet - statement needs it for Execute
	return stmt, nil
}

// QueryWithParams executes a parameterized query with automatic statement management.
// Generates a UUID-based statement name, prepares, executes once, and deallocates.
func (c *Client) QueryWithParams(ctx context.Context, query string, params ...interface{}) (interface{}, error) {
	if c.stateMgr.GetState() != CONNECTED {
		return nil, ErrInvalidState("QueryWithParams", CONNECTED, c.stateMgr.GetState())
	}

	// Generate unique statement name
	stmtName := fmt.Sprintf("stmt_%s", uuid.New().String())

	// Prepare statement
	stmt, err := c.Prepare(ctx, stmtName, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	c.logger.Debug("auto-prepared temporary statement",
		String("stmt_name", stmtName),
		String("query", query),
		Int("param_count", len(params)))

	// Execute with parameters
	return stmt.Execute(params...)
}

// ============================================================================
// QueryBuilder Factory Methods
// ============================================================================

// Query returns a new QueryBuilder for constructing SELECT queries.
// Schema validation is disabled by default for maximum performance.
// Use WithValidation(true) on the builder to enable schema-based validation.
func (c *Client) QueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		client:           c,
		schemaValidation: false,
		queryType:        selectQuery,
	}
}

// Insert returns a new InsertBuilder for constructing INSERT queries.
func (c *Client) InsertBuilder(bundle string) *InsertBuilder {
	return &InsertBuilder{
		client:           c,
		bundle:           bundle,
		schemaValidation: false,
	}
}

// Update returns a new UpdateBuilder for constructing UPDATE queries.
func (c *Client) UpdateBuilder(bundle string) *UpdateBuilder {
	return &UpdateBuilder{
		client:           c,
		bundle:           bundle,
		schemaValidation: false,
		setFields:        make(map[string]interface{}),
	}
}

// Delete returns a new DeleteBuilder for constructing DELETE queries.
func (c *Client) DeleteBuilder(bundle string) *DeleteBuilder {
	return &DeleteBuilder{
		client:           c,
		bundle:           bundle,
		schemaValidation: false,
	}
}

// PreloadSchema eagerly fetches and caches the database schema.
// This is useful when PreloadSchema option is enabled or when you want
// to warm the schema cache before executing queries with validation.
func (c *Client) PreloadSchema(ctx context.Context) error {
	if c.stateMgr.GetState() != CONNECTED {
		return ErrInvalidState("PreloadSchema", CONNECTED, c.stateMgr.GetState())
	}

	if c.schemaValidator == nil {
		return &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "schema validator not initialized",
		}
	}

	return c.schemaValidator.fetchSchema(ctx)
}

// Begin starts a new transaction, reserving a connection until commit/rollback.
// Sends BEGIN TRANSACTION command to server and parses the returned TX_ID.
func (c *Client) Begin(ctx context.Context) (*Transaction, error) {
	if c.stateMgr.GetState() != CONNECTED {
		return nil, ErrInvalidState("Begin", CONNECTED, c.stateMgr.GetState())
	}

	// Get connection from pool or use single connection
	var conn ConnectionInterface
	var err error

	if c.poolEnabled && c.pool != nil {
		conn, err = c.pool.Get(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		conn = c.conn
	}

	// Send BEGIN TRANSACTION command
	if err := conn.SendCommand(ctx, "BEGIN TRANSACTION;"); err != nil {
		if c.poolEnabled && c.pool != nil {
			c.pool.Put(conn)
		}
		return nil, &TransactionError{
			Code:    "E_BEGIN_FAILED",
			Type:    "TransactionError",
			Message: "failed to begin transaction",
			Cause:   err,
		}
	}

	// Receive response with TX_ID
	response, err := conn.ReceiveResponse(ctx)
	if err != nil {
		if c.poolEnabled && c.pool != nil {
			c.pool.Put(conn)
		}
		return nil, &TransactionError{
			Code:    "E_BEGIN_RESPONSE_FAILED",
			Type:    "TransactionError",
			Message: "failed to receive begin response",
			Cause:   err,
		}
	}

	// Parse TX_ID from response
	// Expected format: "Transaction started with ID: TX_<timestamp>_<random>"
	var txID string
	if respStr, ok := response.(string); ok {
		// Extract TX_ID using simple parsing
		if strings.Contains(respStr, "Transaction started with ID:") {
			parts := strings.Split(respStr, "ID:")
			if len(parts) == 2 {
				txID = strings.TrimSpace(parts[1])
			}
		}
	}

	if txID == "" {
		if c.poolEnabled && c.pool != nil {
			c.pool.Put(conn)
		}
		return nil, &TransactionError{
			Code:    "E_BEGIN_PARSE_FAILED",
			Type:    "TransactionError",
			Message: fmt.Sprintf("failed to parse transaction ID from response: %v", response),
			Details: map[string]interface{}{"response": response},
		}
	}

	tx := &Transaction{
		id:        txID,
		connID:    conn.RemoteAddr(), // Track connection for affinity
		conn:      conn,
		client:    c,
		isolation: ReadCommitted, // Default isolation level
		startedAt: time.Now(),
	}

	// Register active transaction
	c.activeTransactions.Store(txID, &transactionContext{
		tx:        tx,
		conn:      conn,
		startedAt: time.Now(),
	})

	c.logger.Info("transaction started",
		String("tx_id", txID))

	return tx, nil
}

// BeginWithIsolation starts a transaction with a specific isolation level.
// Note: Server currently only supports READ COMMITTED isolation (not configurable).
// The isolation parameter is accepted but ignored; all transactions use READ COMMITTED.
func (c *Client) BeginWithIsolation(ctx context.Context, level IsolationLevel) (*Transaction, error) {
	c.logger.Warn("transaction isolation levels not yet configurable, using READ COMMITTED",
		String("requested_level", level.String()))

	// Begin transaction with default isolation (server will use READ COMMITTED)
	return c.Begin(ctx)
}

// transactionTimeoutMonitor runs in the background checking for abandoned transactions.
// Automatically rolls back and releases connections for transactions exceeding the timeout.
func (c *Client) transactionTimeoutMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkAbandonedTransactions()
		case <-c.txMonitorDone:
			return
		}
	}
}

// checkAbandonedTransactions scans active transactions and rolls back timed-out ones.
func (c *Client) checkAbandonedTransactions() {
	timeout := c.opts.TransactionTimeout
	if timeout == 0 {
		timeout = 5 * time.Minute // Default 5 minutes
	}

	c.activeTransactions.Range(func(key, value interface{}) bool {
		txID := key.(string)
		txCtx := value.(*transactionContext)

		age := time.Since(txCtx.startedAt)
		if age > timeout {
			c.logger.Error("transaction exceeded timeout, forcing rollback",
				String("tx_id", txID),
				Duration("age", age),
				Duration("timeout", timeout))

			// Force rollback
			if err := txCtx.tx.Rollback(); err != nil {
				c.logger.Error("failed to rollback timed-out transaction",
					String("tx_id", txID),
					Error("error", err))
			}

			// Remove from active transactions (Rollback already does this, but double-check)
			c.activeTransactions.Delete(txID)
		}

		return true // Continue iteration
	})
}
