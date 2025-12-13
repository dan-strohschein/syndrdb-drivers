//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"syscall/js"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/codegen"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/migration"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

var (
	// Global client instance
	globalClient *client.Client

	// Global migration client instance
	globalMigrationClient *migration.Client

	// State change callbacks
	stateChangeCallbacks []js.Value

	// Prepared statements (Milestone 2)
	preparedStatements = make(map[string]*client.Statement)

	// Active transactions (Milestone 2)
	activeTransactions = make(map[string]*client.Transaction)

	// Registered JS hooks (Milestone 5)
	jsHooks = make(map[string]*jsHook)

	// Built-in hooks instances (Milestone 5)
	builtinHooks = make(map[string]client.Hook)
)

// clientExecutorAdapter adapts client.Client to migration.MigrationExecutor interface
type clientExecutorAdapter struct {
	client *client.Client
}

// Execute implements migration.MigrationExecutor
func (a *clientExecutorAdapter) Execute(command string) (interface{}, error) {
	// Use Query for SELECT, SHOW, etc. and Mutate for DDL/DML
	// For migrations, most commands will be mutations (CREATE TABLE, ALTER, etc.)
	return a.client.Mutate(command, 0)
}

// convertJSValueToInterface converts a JavaScript value to a Go interface{}
func convertJSValueToInterface(v js.Value) interface{} {
	switch v.Type() {
	case js.TypeNull, js.TypeUndefined:
		return nil
	case js.TypeBoolean:
		return v.Bool()
	case js.TypeNumber:
		return v.Float()
	case js.TypeString:
		return v.String()
	case js.TypeObject:
		if v.Get("length").Type() != js.TypeUndefined {
			// Array
			length := v.Length()
			result := make([]interface{}, length)
			for i := 0; i < length; i++ {
				result[i] = convertJSValueToInterface(v.Index(i))
			}
			return result
		}
		// Object
		result := make(map[string]interface{})
		keys := js.Global().Get("Object").Call("keys", v)
		for i := 0; i < keys.Length(); i++ {
			key := keys.Index(i).String()
			result[key] = convertJSValueToInterface(v.Get(key))
		}
		return result
	default:
		return nil
	}
}

func main() {
	// Export functions to JavaScript
	js.Global().Set("SyndrDB", makeExports())

	// Keep the Go program running
	select {}
}

func makeExports() js.Value {
	exports := make(map[string]interface{})

	// Client methods
	exports["createClient"] = js.FuncOf(createClient)
	exports["connect"] = js.FuncOf(connect)
	exports["disconnect"] = js.FuncOf(disconnect)
	exports["query"] = js.FuncOf(query)
	exports["mutate"] = js.FuncOf(mutate)
	exports["getState"] = js.FuncOf(getState)
	exports["onStateChange"] = js.FuncOf(onStateChange)
	exports["getVersion"] = js.FuncOf(getVersion)

	// Health and monitoring (Milestone 1)
	exports["ping"] = js.FuncOf(ping)
	exports["getConnectionHealth"] = js.FuncOf(getConnectionHealth)

	// Logging (Milestone 1)
	exports["setLogLevel"] = js.FuncOf(setLogLevel)

	// Debug mode (Milestone 1)
	exports["enableDebugMode"] = js.FuncOf(enableDebugMode)
	exports["disableDebugMode"] = js.FuncOf(disableDebugMode)
	exports["getDebugInfo"] = js.FuncOf(getDebugInfo)

	// Schema methods
	exports["generateJSONSchema"] = js.FuncOf(generateJSONSchema)
	exports["generateGraphQLSchema"] = js.FuncOf(generateGraphQLSchema)

	// Migration methods
	exports["createMigrationClient"] = js.FuncOf(createMigrationClient)
	exports["planMigration"] = js.FuncOf(planMigration)
	exports["applyMigration"] = js.FuncOf(applyMigration)
	exports["getMigrationHistory"] = js.FuncOf(getMigrationHistory)
	exports["validateMigration"] = js.FuncOf(validateMigration)
	exports["rollbackMigration"] = js.FuncOf(rollbackMigration)
	exports["previewMigration"] = js.FuncOf(previewMigration)

	// Migration file operations (Node.js only)
	exports["saveMigrationFile"] = js.FuncOf(nodeOnlyExport("saveMigrationFile", saveMigrationFile))
	exports["loadMigrationFile"] = js.FuncOf(nodeOnlyExport("loadMigrationFile", loadMigrationFile))
	exports["listMigrations"] = js.FuncOf(nodeOnlyExport("listMigrations", listMigrations))
	exports["acquireMigrationLock"] = js.FuncOf(nodeOnlyExport("acquireMigrationLock", acquireMigrationLock))
	exports["releaseMigrationLock"] = js.FuncOf(nodeOnlyExport("releaseMigrationLock", releaseMigrationLock))

	// Environment info
	exports["getEnvironmentInfo"] = js.FuncOf(getEnvironmentInfo)

	// Parameterized queries (Milestone 2)
	exports["prepare"] = js.FuncOf(prepare)
	exports["executeStatement"] = js.FuncOf(executeStatement)
	exports["deallocateStatement"] = js.FuncOf(deallocateStatement)
	exports["queryWithParams"] = js.FuncOf(queryWithParams)

	// Transactions (Milestone 2)
	exports["beginTransaction"] = js.FuncOf(beginTransaction)
	exports["commitTransaction"] = js.FuncOf(commitTransaction)
	exports["rollbackTransaction"] = js.FuncOf(rollbackTransaction)
	exports["inTransaction"] = js.FuncOf(inTransaction)

	// Hooks System (Milestone 5)
	exports["registerHook"] = js.FuncOf(registerHook)
	exports["unregisterHook"] = js.FuncOf(unregisterHook)
	exports["getHooks"] = js.FuncOf(getHooks)
	exports["createLoggingHook"] = js.FuncOf(createLoggingHook)
	exports["createMetricsHook"] = js.FuncOf(createMetricsHook)
	exports["getMetricsStats"] = js.FuncOf(getMetricsStats)
	exports["resetMetrics"] = js.FuncOf(resetMetrics)
	exports["createTracingHook"] = js.FuncOf(createTracingHook)

	// Cleanup
	exports["cleanup"] = js.FuncOf(cleanup)

	return js.ValueOf(exports)
}

// createClient creates a new SyndrDB client with options
func createClient(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		var opts *client.ClientOptions

		if len(args) > 0 && !args[0].IsNull() && !args[0].IsUndefined() {
			optsArg := args[0]
			opts = &client.ClientOptions{
				DefaultTimeoutMs: optsArg.Get("defaultTimeoutMs").Int(),
				DebugMode:        optsArg.Get("debugMode").Bool(),
				MaxRetries:       optsArg.Get("maxRetries").Int(),
			}

			// Parse log level if provided
			if logLevel := optsArg.Get("logLevel"); !logLevel.IsUndefined() && !logLevel.IsNull() {
				opts.LogLevel = logLevel.String()
			}

			// Parse health check interval if provided
			if interval := optsArg.Get("healthCheckIntervalMs"); !interval.IsUndefined() && interval.Int() > 0 {
				opts.HealthCheckInterval = time.Duration(interval.Int()) * time.Millisecond
			}

			// Parse max reconnect attempts if provided
			if maxAttempts := optsArg.Get("maxReconnectAttempts"); !maxAttempts.IsUndefined() && maxAttempts.Int() > 0 {
				opts.MaxReconnectAttempts = maxAttempts.Int()
			}
		}

		globalClient = client.NewClient(opts)

		// Setup state change forwarding
		globalClient.OnStateChange(func(transition client.StateTransition) {
			transitionJS := map[string]interface{}{
				"from":      transition.From.String(),
				"to":        transition.To.String(),
				"timestamp": transition.Timestamp.UnixMilli(),
				"duration":  transition.Duration.Milliseconds(),
			}

			if transition.Error != nil {
				transitionJS["error"] = transition.Error.Error()
			}

			if transition.Metadata != nil {
				transitionJS["metadata"] = transition.Metadata
			}

			// Call all registered callbacks
			for _, callback := range stateChangeCallbacks {
				callback.Invoke(js.ValueOf(transitionJS))
			}
		})

		return map[string]interface{}{"success": true}, nil
	})
}

// connect establishes a connection to the database
func connect(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "connect", Type: js.TypeNull}
		}

		connStr := args[0].String()
		ctx := context.Background()
		err := globalClient.Connect(ctx, connStr)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{"success": true}, nil
	})
}

// disconnect closes the database connection
func disconnect(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "disconnect", Type: js.TypeNull}
		}

		ctx := context.Background()
		err := globalClient.Disconnect(ctx)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{"success": true}, nil
	})
}

// query executes a database query
func query(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "query", Type: js.TypeNull}
		}

		queryStr := args[0].String()
		timeout := 0
		if len(args) > 1 {
			timeout = args[1].Int()
		}

		result, err := globalClient.Query(queryStr, timeout)
		if err != nil {
			return nil, err
		}

		return result, nil
	})
}

// mutate executes a database mutation
func mutate(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "mutate", Type: js.TypeNull}
		}

		mutationStr := args[0].String()
		timeout := 0
		if len(args) > 1 {
			timeout = args[1].Int()
		}

		result, err := globalClient.Mutate(mutationStr, timeout)
		if err != nil {
			return nil, err
		}

		return result, nil
	})
}

// getState returns the current connection state
func getState(this js.Value, args []js.Value) interface{} {
	if globalClient == nil {
		return js.ValueOf("DISCONNECTED")
	}
	return js.ValueOf(globalClient.GetState().String())
}

// onStateChange registers a callback for state changes
func onStateChange(this js.Value, args []js.Value) interface{} {
	if len(args) == 0 || !args[0].InstanceOf(js.Global().Get("Function")) {
		return js.ValueOf(map[string]interface{}{"error": "callback must be a function"})
	}

	stateChangeCallbacks = append(stateChangeCallbacks, args[0])
	return js.ValueOf(map[string]interface{}{"success": true})
}

// getVersion returns the client version
func getVersion(this js.Value, args []js.Value) interface{} {
	if globalClient == nil {
		return js.ValueOf(client.Version)
	}
	return js.ValueOf(globalClient.GetVersion())
}

// ping performs an explicit health check on the connection
func ping(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "ping", Type: js.TypeNull}
		}

		ctx := context.Background()
		if len(args) > 0 && args[0].Int() > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(args[0].Int())*time.Millisecond)
			defer cancel()
		}

		err := globalClient.Ping(ctx)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{"success": true}, nil
	})
}

// getConnectionHealth returns the connection health status
func getConnectionHealth(this js.Value, args []js.Value) interface{} {
	if globalClient == nil {
		return js.ValueOf(map[string]interface{}{
			"connected": false,
			"state":     "DISCONNECTED",
		})
	}

	state := globalClient.GetState()
	return js.ValueOf(map[string]interface{}{
		"connected": state == client.CONNECTED,
		"state":     state.String(),
	})
}

// setLogLevel changes the logging level at runtime
func setLogLevel(this js.Value, args []js.Value) interface{} {
	if len(args) == 0 {
		return js.ValueOf(map[string]interface{}{
			"error": "log level required (DEBUG, INFO, WARN, ERROR)",
		})
	}

	level := args[0].String()
	if globalClient != nil {
		globalClient.SetLogLevel(level)
	}

	return js.ValueOf(map[string]interface{}{"success": true, "level": level})
}

// enableDebugMode enables debug mode with verbose logging
func enableDebugMode(this js.Value, args []js.Value) interface{} {
	if globalClient == nil {
		return js.ValueOf(map[string]interface{}{"error": "client not initialized"})
	}

	globalClient.EnableDebugMode()
	return js.ValueOf(map[string]interface{}{"success": true, "debugEnabled": true})
}

// disableDebugMode disables debug mode
func disableDebugMode(this js.Value, args []js.Value) interface{} {
	if globalClient == nil {
		return js.ValueOf(map[string]interface{}{"error": "client not initialized"})
	}

	globalClient.DisableDebugMode()
	return js.ValueOf(map[string]interface{}{"success": true, "debugEnabled": false})
}

// getDebugInfo returns current debug information
func getDebugInfo(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "getDebugInfo", Type: js.TypeNull}
		}

		debugInfo := globalClient.GetDebugInfo()
		return debugInfo, nil
	})
}

// generateJSONSchema generates JSON Schema from a schema definition
func generateJSONSchema(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		schemaJSON := args[0].String()
		mode := "single"
		if len(args) > 1 {
			mode = args[1].String()
		}

		var schemaDef schema.SchemaDefinition
		if err := json.Unmarshal([]byte(schemaJSON), &schemaDef); err != nil {
			return nil, err
		}

		generator := codegen.NewJSONSchemaGenerator()

		if mode == "multi" {
			schemas, err := generator.GenerateMulti(&schemaDef)
			if err != nil {
				return nil, err
			}
			return schemas, nil
		}

		schemaStr, err := generator.GenerateSingle(&schemaDef)
		if err != nil {
			return nil, err
		}

		return schemaStr, nil
	})
}

// generateGraphQLSchema generates GraphQL SDL from a schema definition
func generateGraphQLSchema(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		schemaJSON := args[0].String()

		var schemaDef schema.SchemaDefinition
		if err := json.Unmarshal([]byte(schemaJSON), &schemaDef); err != nil {
			return nil, err
		}

		generator := codegen.NewGraphQLSchemaGenerator()
		schemaStr, err := generator.Generate(&schemaDef)
		if err != nil {
			return nil, err
		}

		return schemaStr, nil
	})
}

// createMigrationClient creates a migration client
func createMigrationClient(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "createMigrationClient", Type: js.TypeNull}
		}

		adapter := &clientExecutorAdapter{client: globalClient}
		globalMigrationClient = migration.NewClient(adapter)

		return map[string]interface{}{
			"success": true,
			"message": "Migration client created successfully",
		}, nil
	})
}

// planMigration creates a migration plan
func planMigration(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalMigrationClient == nil {
			return nil, &js.ValueError{Method: "planMigration", Type: js.TypeNull}
		}

		if len(args) < 1 {
			return nil, &js.ValueError{Method: "planMigration", Type: js.TypeNull}
		}

		// Parse migrations array from JavaScript
		migrationsData := convertJSValueToInterface(args[0])
		migrationsJSON, err := json.Marshal(migrationsData)
		if err != nil {
			return nil, err
		}

		var migrations []*migration.Migration
		if err := json.Unmarshal(migrationsJSON, &migrations); err != nil {
			return nil, err
		}

		// Create plan
		plan, err := globalMigrationClient.Plan(migrations)
		if err != nil {
			return nil, err
		}

		// Serialize plan back to JavaScript
		planJSON, err := json.Marshal(plan)
		if err != nil {
			return nil, err
		}

		var result interface{}
		if err := json.Unmarshal(planJSON, &result); err != nil {
			return nil, err
		}

		return result, nil
	})
}

// applyMigration applies a migration plan
func applyMigration(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalMigrationClient == nil {
			return nil, &js.ValueError{Method: "applyMigration", Type: js.TypeNull}
		}

		if len(args) < 1 {
			return nil, &js.ValueError{Method: "applyMigration", Type: js.TypeNull}
		}

		// Parse migration plan from JavaScript
		planData := convertJSValueToInterface(args[0])
		planJSON, err := json.Marshal(planData)
		if err != nil {
			return nil, err
		}

		var plan migration.MigrationPlan
		if err := json.Unmarshal(planJSON, &plan); err != nil {
			return nil, err
		}

		// Apply migration
		if err := globalMigrationClient.Apply(&plan); err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"success": true,
			"message": "Migration applied successfully",
		}, nil
	})
}

// cleanup releases resources and callbacks
func cleanup(this js.Value, args []js.Value) interface{} {
	if globalClient != nil {
		globalClient.Disconnect(context.Background())
		globalClient = nil
	}
	stateChangeCallbacks = nil
	preparedStatements = make(map[string]*client.Statement)
	activeTransactions = make(map[string]*client.Transaction)
	return js.ValueOf(map[string]interface{}{"success": true})
}

// prepare creates a prepared statement with parameter placeholders.
// Args: statementName (string), query (string)
// Returns: Promise<{statementId: string, paramCount: number}>
func prepare(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "prepare", Type: js.TypeNull}
		}

		if len(args) < 2 {
			return nil, &js.ValueError{Method: "prepare", Type: js.TypeUndefined}
		}

		stmtName := args[0].String()
		query := args[1].String()

		ctx := context.Background()
		stmt, err := globalClient.Prepare(ctx, stmtName, query)
		if err != nil {
			return nil, err
		}

		// Store statement reference
		preparedStatements[stmtName] = stmt

		return map[string]interface{}{
			"statementId": stmtName,
			"paramCount":  stmt.ParamCount(),
			"query":       stmt.Query(),
		}, nil
	})
}

// executeStatement executes a prepared statement with parameters.
// Args: statementId (string), params (array)
// Returns: Promise<result>
func executeStatement(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "executeStatement", Type: js.TypeNull}
		}

		if len(args) < 1 {
			return nil, &js.ValueError{Method: "executeStatement", Type: js.TypeUndefined}
		}

		stmtID := args[0].String()
		stmt, exists := preparedStatements[stmtID]
		if !exists {
			return nil, &js.ValueError{Method: "executeStatement", Type: js.TypeUndefined}
		}

		// Extract parameters from JavaScript array
		var params []interface{}
		if len(args) > 1 && !args[1].IsNull() && !args[1].IsUndefined() {
			paramsArray := args[1]
			length := paramsArray.Length()
			params = make([]interface{}, length)
			for i := 0; i < length; i++ {
				params[i] = jsValueToGo(paramsArray.Index(i))
			}
		}

		result, err := stmt.Execute(params...)
		if err != nil {
			return nil, err
		}

		return result, nil
	})
}

// deallocateStatement deallocates a prepared statement.
// Args: statementId (string)
// Returns: Promise<{success: boolean}>
func deallocateStatement(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if len(args) < 1 {
			return nil, &js.ValueError{Method: "deallocateStatement", Type: js.TypeUndefined}
		}

		stmtID := args[0].String()
		stmt, exists := preparedStatements[stmtID]
		if !exists {
			return map[string]interface{}{"success": false, "message": "statement not found"}, nil
		}

		if err := stmt.Close(); err != nil {
			return nil, err
		}

		delete(preparedStatements, stmtID)

		return map[string]interface{}{"success": true}, nil
	})
}

// queryWithParams executes a parameterized query with automatic statement management.
// Args: query (string), params (array)
// Returns: Promise<result>
func queryWithParams(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "queryWithParams", Type: js.TypeNull}
		}

		if len(args) < 1 {
			return nil, &js.ValueError{Method: "queryWithParams", Type: js.TypeUndefined}
		}

		query := args[0].String()

		// Extract parameters from JavaScript array
		var params []interface{}
		if len(args) > 1 && !args[1].IsNull() && !args[1].IsUndefined() {
			paramsArray := args[1]
			length := paramsArray.Length()
			params = make([]interface{}, length)
			for i := 0; i < length; i++ {
				params[i] = jsValueToGo(paramsArray.Index(i))
			}
		}

		ctx := context.Background()
		result, err := globalClient.QueryWithParams(ctx, query, params...)
		if err != nil {
			return nil, err
		}

		return result, nil
	})
}

// beginTransaction starts a new transaction.
// Args: none
// Returns: Promise<{transactionId: string}>
func beginTransaction(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "beginTransaction", Type: js.TypeNull}
		}

		ctx := context.Background()
		tx, err := globalClient.Begin(ctx)
		if err != nil {
			return nil, err
		}

		// Store transaction reference
		txID := tx.ID()
		activeTransactions[txID] = tx

		return map[string]interface{}{
			"transactionId": txID,
		}, nil
	})
}

// commitTransaction commits a transaction.
// Args: transactionId (string)
// Returns: Promise<{success: boolean}>
func commitTransaction(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if len(args) < 1 {
			return nil, &js.ValueError{Method: "commitTransaction", Type: js.TypeUndefined}
		}

		txID := args[0].String()
		tx, exists := activeTransactions[txID]
		if !exists {
			return nil, &js.ValueError{Method: "commitTransaction", Type: js.TypeUndefined}
		}

		if err := tx.Commit(); err != nil {
			return nil, err
		}

		delete(activeTransactions, txID)

		return map[string]interface{}{"success": true}, nil
	})
}

// rollbackTransaction rolls back a transaction.
// Args: transactionId (string)
// Returns: Promise<{success: boolean}>
func rollbackTransaction(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if len(args) < 1 {
			return nil, &js.ValueError{Method: "rollbackTransaction", Type: js.TypeUndefined}
		}

		txID := args[0].String()
		tx, exists := activeTransactions[txID]
		if !exists {
			return nil, &js.ValueError{Method: "rollbackTransaction", Type: js.TypeUndefined}
		}

		if err := tx.Rollback(); err != nil {
			return nil, err
		}

		delete(activeTransactions, txID)

		return map[string]interface{}{"success": true}, nil
	})
}

// inTransaction executes a function within a transaction with automatic commit/rollback.
// Args: callback (function)
// Returns: Promise<result>
func inTransaction(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, &js.ValueError{Method: "inTransaction", Type: js.TypeNull}
		}

		if len(args) < 1 || args[0].Type() != js.TypeFunction {
			return nil, &js.ValueError{Method: "inTransaction", Type: js.TypeUndefined}
		}

		callback := args[0]

		ctx := context.Background()
		var result interface{}

		err := globalClient.InTransaction(ctx, func(tx *client.Transaction) error {
			// Store transaction temporarily for callback access
			txID := tx.ID()
			activeTransactions[txID] = tx
			defer delete(activeTransactions, txID)

			// Create transaction object for JavaScript
			txObj := map[string]interface{}{
				"transactionId": txID,
			}

			// Invoke callback with transaction object
			callbackResult := callback.Invoke(js.ValueOf(txObj))

			// Handle promise returned from callback
			if callbackResult.Type() == js.TypeObject && callbackResult.Get("then").Type() == js.TypeFunction {
				// Wait for promise to resolve
				done := make(chan error, 1)
				callbackResult.Call("then",
					js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						if len(args) > 0 {
							result = jsValueToGo(args[0])
						}
						done <- nil
						return nil
					}),
					js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						_ = args // Ignore rejection reason
						done <- &js.ValueError{Method: "inTransaction"}
						return nil
					}),
				)
				return <-done
			}

			// Synchronous callback
			result = jsValueToGo(callbackResult)
			return nil
		})

		if err != nil {
			return nil, err
		}

		return result, nil
	})
}

// jsValueToGo converts a JavaScript value to a Go interface{}.
func jsValueToGo(val js.Value) interface{} {
	switch val.Type() {
	case js.TypeUndefined, js.TypeNull:
		return nil
	case js.TypeBoolean:
		return val.Bool()
	case js.TypeNumber:
		return val.Float()
	case js.TypeString:
		return val.String()
	case js.TypeObject:
		// Check if it's an array
		if val.Get("length").Type() == js.TypeNumber {
			length := val.Get("length").Int()
			arr := make([]interface{}, length)
			for i := 0; i < length; i++ {
				arr[i] = jsValueToGo(val.Index(i))
			}
			return arr
		}
		// Convert object to map
		obj := make(map[string]interface{})
		keys := js.Global().Get("Object").Call("keys", val)
		for i := 0; i < keys.Length(); i++ {
			key := keys.Index(i).String()
			obj[key] = jsValueToGo(val.Get(key))
		}
		return obj
	default:
		return val.String()
	}
}

// promiseWrapper wraps a function in a JavaScript Promise
func promiseWrapper(fn func() (interface{}, error)) js.Value {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]

		go func() {
			result, err := fn()
			if err != nil {
				errorObj := map[string]interface{}{
					"message": err.Error(),
					"error":   err.Error(),
				}
				reject.Invoke(js.ValueOf(errorObj))
			} else {
				resolve.Invoke(js.ValueOf(result))
			}
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// Migration helper methods

// getMigrationHistory retrieves migration history
func getMigrationHistory(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalMigrationClient == nil {
			return nil, &js.ValueError{Method: "getMigrationHistory", Type: js.TypeNull}
		}

		historyJSON, err := globalMigrationClient.GetHistory()
		if err != nil {
			return nil, err
		}

		var result interface{}
		if err := json.Unmarshal(historyJSON, &result); err != nil {
			return nil, err
		}

		return result, nil
	})
}

// validateMigration validates migrations
func validateMigration(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalMigrationClient == nil {
			return nil, &js.ValueError{Method: "validateMigration", Type: js.TypeNull}
		}

		if len(args) < 1 {
			return nil, &js.ValueError{Method: "validateMigration", Type: js.TypeNull}
		}

		// Parse migrations array
		migrationsData := convertJSValueToInterface(args[0])
		migrationsJSON, err := json.Marshal(migrationsData)
		if err != nil {
			return nil, err
		}

		var migrations []*migration.Migration
		if err := json.Unmarshal(migrationsJSON, &migrations); err != nil {
			return nil, err
		}

		// Validate
		result := globalMigrationClient.Validate(migrations)

		// Serialize result
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}

		var output interface{}
		if err := json.Unmarshal(resultJSON, &output); err != nil {
			return nil, err
		}

		return output, nil
	})
}

// rollbackMigration rolls back a migration
func rollbackMigration(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalMigrationClient == nil {
			return nil, &js.ValueError{Method: "rollbackMigration", Type: js.TypeNull}
		}

		if len(args) < 2 {
			return nil, &js.ValueError{Method: "rollbackMigration", Type: js.TypeNull}
		}

		migrationID := args[0].String()

		// Parse all migrations array
		migrationsData := convertJSValueToInterface(args[1])
		migrationsJSON, err := json.Marshal(migrationsData)
		if err != nil {
			return nil, err
		}

		var allMigrations []*migration.Migration
		if err := json.Unmarshal(migrationsJSON, &allMigrations); err != nil {
			return nil, err
		}

		// Rollback
		if err := globalMigrationClient.Rollback(migrationID, allMigrations); err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"success": true,
			"message": "Migration rolled back successfully",
		}, nil
	})
}

// previewMigration creates a dry-run preview
func previewMigration(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalMigrationClient == nil {
			return nil, &js.ValueError{Method: "previewMigration", Type: js.TypeNull}
		}

		if len(args) < 1 {
			return nil, &js.ValueError{Method: "previewMigration", Type: js.TypeNull}
		}

		// Parse migrations array
		migrationsData := convertJSValueToInterface(args[0])
		migrationsJSON, err := json.Marshal(migrationsData)
		if err != nil {
			return nil, err
		}

		var migrations []*migration.Migration
		if err := json.Unmarshal(migrationsJSON, &migrations); err != nil {
			return nil, err
		}

		// Create preview plan
		plan, err := globalMigrationClient.Preview(migrations)
		if err != nil {
			return nil, err
		}

		// Format for human reading
		preview := migration.FormatPreview(plan)

		return map[string]interface{}{
			"preview": preview,
			"plan":    plan,
		}, nil
	})
}

// Node.js-only file operations

// isNodeJS checks if running in Node.js environment
func isNodeJS() bool {
	process := js.Global().Get("process")
	return process.Truthy() && process.Get("version").Truthy()
}

// nodeOnlyExport wraps a function to check for Node.js environment
func nodeOnlyExport(name string, fn func(js.Value, []js.Value) interface{}) func(js.Value, []js.Value) interface{} {
	return func(this js.Value, args []js.Value) interface{} {
		return promiseWrapper(func() (interface{}, error) {
			if !isNodeJS() {
				return map[string]interface{}{
					"error":   "This feature requires Node.js environment",
					"feature": name,
				}, nil
			}

			// Unwrap the promise from the inner function
			_ = fn(this, args)

			// If it's already a promise, we need to handle it differently
			// For now, return an error indicating implementation needed
			return map[string]interface{}{
				"error": "Node.js file operations not yet fully implemented in WASM",
			}, nil
		})
	}
}

// saveMigrationFile saves a migration to file (Node.js only)
func saveMigrationFile(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if len(args) < 2 {
			return nil, &js.ValueError{Method: "saveMigrationFile", Type: js.TypeNull}
		}

		// Parse migration
		migrationData := convertJSValueToInterface(args[0])
		migrationJSON, err := json.Marshal(migrationData)
		if err != nil {
			return nil, err
		}

		var mig migration.Migration
		if err := json.Unmarshal(migrationJSON, &mig); err != nil {
			return nil, err
		}

		dir := args[1].String()

		// Write file
		path, err := migration.WriteMigrationFile(&mig, dir)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"success": true,
			"path":    path,
		}, nil
	})
}

// loadMigrationFile loads a migration from file (Node.js only)
func loadMigrationFile(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if len(args) < 1 {
			return nil, &js.ValueError{Method: "loadMigrationFile", Type: js.TypeNull}
		}

		path := args[0].String()

		// Read file
		mig, err := migration.ReadMigrationFile(path)
		if err != nil {
			return nil, err
		}

		// Serialize to JavaScript
		migJSON, err := json.Marshal(mig)
		if err != nil {
			return nil, err
		}

		var result interface{}
		if err := json.Unmarshal(migJSON, &result); err != nil {
			return nil, err
		}

		return result, nil
	})
}

// listMigrations lists migration files in directory (Node.js only)
func listMigrations(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if len(args) < 1 {
			return nil, &js.ValueError{Method: "listMigrations", Type: js.TypeNull}
		}

		dir := args[0].String()

		// List files
		migrations, err := migration.ListMigrationFiles(dir)
		if err != nil {
			return nil, err
		}

		// Serialize to JavaScript
		migsJSON, err := json.Marshal(migrations)
		if err != nil {
			return nil, err
		}

		var result interface{}
		if err := json.Unmarshal(migsJSON, &result); err != nil {
			return nil, err
		}

		return result, nil
	})
}

// acquireMigrationLock acquires migration lock (Node.js only)
func acquireMigrationLock(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalMigrationClient == nil {
			return nil, &js.ValueError{Method: "acquireMigrationLock", Type: js.TypeNull}
		}

		if len(args) < 1 {
			return nil, &js.ValueError{Method: "acquireMigrationLock", Type: js.TypeNull}
		}

		dir := args[0].String()

		// Configure locking
		if err := globalMigrationClient.WithLocking(dir, 0); err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"success": true,
			"message": "Locking configured",
		}, nil
	})
}

// releaseMigrationLock releases migration lock (Node.js only)
func releaseMigrationLock(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		// Lock is automatically released after migration
		return map[string]interface{}{
			"success": true,
			"message": "Lock released automatically after migration",
		}, nil
	})
}

// getEnvironmentInfo returns environment information
func getEnvironmentInfo(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		isNode := isNodeJS()

		return map[string]interface{}{
			"runtime":           map[bool]string{true: "nodejs", false: "browser"}[isNode],
			"fileSystemSupport": isNode,
			"lockingSupport":    isNode,
		}, nil
	})
}

// ============================================================================
// Hooks System (Milestone 5)
// ============================================================================

// jsHook wraps JavaScript hook functions to implement the Go Hook interface
type jsHook struct {
	name       string
	beforeFunc js.Value
	afterFunc  js.Value
}

func (h *jsHook) Name() string {
	return h.name
}

func (h *jsHook) Before(ctx context.Context, hookCtx *client.HookContext) error {
	if h.beforeFunc.IsUndefined() || h.beforeFunc.IsNull() {
		return nil
	}

	// Convert HookContext to JS object
	jsCtx := convertHookContextToJS(hookCtx)

	// Call JavaScript before function
	result := h.beforeFunc.Invoke(jsCtx)

	// Handle Promise return
	if result.Type() == js.TypeObject && result.Get("then").Type() == js.TypeFunction {
		// It's a Promise - wait for it
		resultChan := make(chan error, 1)

		result.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// Success - update hookCtx if modified
			if len(args) > 0 && args[0].Type() == js.TypeObject {
				updateHookContextFromJS(hookCtx, args[0])
			}
			resultChan <- nil
			return nil
		}))

		result.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// Error
			if len(args) > 0 {
				resultChan <- js.Error{Value: args[0]}
			} else {
				resultChan <- js.Error{Value: js.ValueOf("hook error")}
			}
			return nil
		}))

		return <-resultChan
	}

	// Synchronous return - check for error
	if result.Type() == js.TypeObject && !result.Get("error").IsUndefined() {
		return js.Error{Value: result.Get("error")}
	}

	// Update hookCtx if modified
	if result.Type() == js.TypeObject {
		updateHookContextFromJS(hookCtx, result)
	}

	return nil
}

func (h *jsHook) After(ctx context.Context, hookCtx *client.HookContext) error {
	if h.afterFunc.IsUndefined() || h.afterFunc.IsNull() {
		return nil
	}

	// Convert HookContext to JS object
	jsCtx := convertHookContextToJS(hookCtx)

	// Call JavaScript after function
	result := h.afterFunc.Invoke(jsCtx)

	// Handle Promise return
	if result.Type() == js.TypeObject && result.Get("then").Type() == js.TypeFunction {
		// It's a Promise - wait for it
		resultChan := make(chan error, 1)

		result.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resultChan <- nil
			return nil
		}))

		result.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				resultChan <- js.Error{Value: args[0]}
			} else {
				resultChan <- js.Error{Value: js.ValueOf("hook error")}
			}
			return nil
		}))

		return <-resultChan
	}

	// Synchronous return - check for error
	if result.Type() == js.TypeObject && !result.Get("error").IsUndefined() {
		return js.Error{Value: result.Get("error")}
	}

	return nil
}

// convertHookContextToJS converts a Go HookContext to a JavaScript object
func convertHookContextToJS(hookCtx *client.HookContext) js.Value {
	obj := js.Global().Get("Object").New()

	obj.Set("command", hookCtx.Command)
	obj.Set("commandType", hookCtx.CommandType)
	obj.Set("traceId", hookCtx.TraceID)
	obj.Set("startTime", hookCtx.StartTime.UnixMilli())

	if hookCtx.Params != nil {
		paramsArray := make([]interface{}, len(hookCtx.Params))
		copy(paramsArray, hookCtx.Params)
		obj.Set("params", paramsArray)
	}

	if hookCtx.Metadata != nil {
		metadataObj := js.Global().Get("Object").New()
		for k, v := range hookCtx.Metadata {
			metadataObj.Set(k, v)
		}
		obj.Set("metadata", metadataObj)
	}

	if hookCtx.Result != nil {
		resultJSON, _ := json.Marshal(hookCtx.Result)
		obj.Set("result", string(resultJSON))
	}

	if hookCtx.Error != nil {
		obj.Set("error", hookCtx.Error.Error())
	}

	if hookCtx.Duration > 0 {
		obj.Set("durationMs", float64(hookCtx.Duration.Milliseconds()))
	}

	return obj
}

// updateHookContextFromJS updates Go HookContext from JavaScript object
func updateHookContextFromJS(hookCtx *client.HookContext, jsObj js.Value) {
	if !jsObj.Get("command").IsUndefined() {
		hookCtx.Command = jsObj.Get("command").String()
	}

	if !jsObj.Get("metadata").IsUndefined() {
		metadata := jsObj.Get("metadata")
		keys := js.Global().Get("Object").Call("keys", metadata)
		for i := 0; i < keys.Length(); i++ {
			key := keys.Index(i).String()
			value := convertJSValueToInterface(metadata.Get(key))
			hookCtx.Metadata[key] = value
		}
	}
}

// registerHook registers a custom JavaScript hook
func registerHook(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, js.Error{Value: js.ValueOf("client not connected")}
		}

		if len(args) < 1 {
			return nil, &js.ValueError{Method: "registerHook", Type: js.TypeUndefined}
		}

		hookConfig := args[0]

		// Extract hook configuration
		name := hookConfig.Get("name").String()
		if name == "" {
			return nil, js.Error{Value: js.ValueOf("hook name is required")}
		}

		beforeFunc := hookConfig.Get("before")
		afterFunc := hookConfig.Get("after")

		// Create JS hook wrapper
		hook := &jsHook{
			name:       name,
			beforeFunc: beforeFunc,
			afterFunc:  afterFunc,
		}

		// Store reference
		jsHooks[name] = hook

		// Register with client
		globalClient.RegisterHook(hook)

		return map[string]interface{}{
			"success": true,
			"message": "Hook registered: " + name,
		}, nil
	})
}

// unregisterHook removes a registered hook
func unregisterHook(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, js.Error{Value: js.ValueOf("client not connected")}
		}

		if len(args) < 1 {
			return nil, &js.ValueError{Method: "unregisterHook", Type: js.TypeUndefined}
		}

		name := args[0].String()

		// Remove from client
		removed := globalClient.UnregisterHook(name)

		// Remove from our map
		delete(jsHooks, name)
		delete(builtinHooks, name)

		return map[string]interface{}{
			"success": removed,
			"message": map[bool]string{true: "Hook removed", false: "Hook not found"}[removed],
		}, nil
	})
}

// getHooks returns list of registered hooks
func getHooks(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, js.Error{Value: js.ValueOf("client not connected")}
		}

		hooks := globalClient.GetHooks()

		return map[string]interface{}{
			"hooks": hooks,
			"count": len(hooks),
		}, nil
	})
}

// createLoggingHook creates a built-in logging hook
func createLoggingHook(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, js.Error{Value: js.ValueOf("client not connected")}
		}

		// Parse options
		logCommands := true
		logResults := false
		logDurations := true

		if len(args) > 0 && args[0].Type() == js.TypeObject {
			opts := args[0]
			if !opts.Get("logCommands").IsUndefined() {
				logCommands = opts.Get("logCommands").Bool()
			}
			if !opts.Get("logResults").IsUndefined() {
				logResults = opts.Get("logResults").Bool()
			}
			if !opts.Get("logDurations").IsUndefined() {
				logDurations = opts.Get("logDurations").Bool()
			}
		}

		// Create and register hook
		logger := client.NewLogger("DEBUG", nil)
		hook := client.NewLoggingHook(logger, logCommands, logResults, logDurations)

		builtinHooks["logging"] = hook
		globalClient.RegisterHook(hook)

		return map[string]interface{}{
			"success": true,
			"message": "Logging hook created and registered",
			"name":    "logging",
		}, nil
	})
}

// createMetricsHook creates a built-in metrics hook
func createMetricsHook(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, js.Error{Value: js.ValueOf("client not connected")}
		}

		hook := client.NewMetricsHook()

		builtinHooks["metrics"] = hook
		globalClient.RegisterHook(hook)

		return map[string]interface{}{
			"success": true,
			"message": "Metrics hook created and registered",
			"name":    "metrics",
		}, nil
	})
}

// getMetricsStats returns current metrics
func getMetricsStats(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		hook, exists := builtinHooks["metrics"]
		if !exists {
			return nil, js.Error{Value: js.ValueOf("metrics hook not registered")}
		}

		metricsHook, ok := hook.(*client.MetricsHook)
		if !ok {
			return nil, js.Error{Value: js.ValueOf("invalid metrics hook")}
		}

		return metricsHook.GetStats(), nil
	})
}

// resetMetrics resets metrics counters
func resetMetrics(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		hook, exists := builtinHooks["metrics"]
		if !exists {
			return nil, js.Error{Value: js.ValueOf("metrics hook not registered")}
		}

		metricsHook, ok := hook.(*client.MetricsHook)
		if !ok {
			return nil, js.Error{Value: js.ValueOf("invalid metrics hook")}
		}

		metricsHook.Reset()

		return map[string]interface{}{
			"success": true,
			"message": "Metrics reset",
		}, nil
	})
}

// createTracingHook creates a built-in tracing hook
func createTracingHook(this js.Value, args []js.Value) interface{} {
	return promiseWrapper(func() (interface{}, error) {
		if globalClient == nil {
			return nil, js.Error{Value: js.ValueOf("client not connected")}
		}

		serviceName := "syndrdb-wasm"
		if len(args) > 0 {
			serviceName = args[0].String()
		}

		hook := client.NewTracingHook(serviceName)

		builtinHooks["tracing"] = hook
		globalClient.RegisterHook(hook)

		return map[string]interface{}{
			"success": true,
			"message": "Tracing hook created and registered",
			"name":    "tracing",
		}, nil
	})
}
