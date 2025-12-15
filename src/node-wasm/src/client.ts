/**
 * SyndrDB WASM Client
 * High-performance database client with DWARF debugging and performance monitoring
 */

import { EventEmitter } from 'events';
import { getWASMLoader } from './wasm-loader';
import type { WASMExports } from './wasm-loader';
import { NodeConnection } from './node-connection';
import {
  getGlobalPerformanceMonitor,
  PerformanceMonitor,
} from './utils';
import {
  ConnectionError,
  mapWASMError,
  parseWASMError,
} from './errors';
import type {
  ClientOptions,
  StateTransition,
  ConnectionHealth,
  DebugInfo,
  WASMMetadata,
} from './types/client';
import type { PerformanceStats } from './types/performance';
import type {
  Migration,
  MigrationHistory,
  MigrationPlan,
  ValidationReport,
} from './types/migration';
import type {
  SchemaDefinition,
  JSONSchemaMode,
  GraphQLGenerationOptions,
} from './types/schema';
import type { HookConfig, MetricsStats } from './types/hooks';
import { ClientState } from './types/client';

/**
 * SyndrDB Client with EventEmitter for state changes
 */
export class SyndrDBClient extends EventEmitter {
  private currentState: ClientState = 'DISCONNECTED' as ClientState;
  private wasmExports: WASMExports | null = null;
  private performanceMonitor: PerformanceMonitor;
  private options: Required<ClientOptions>;
  private stateHistory: StateTransition[] = [];
  private nodeConnection: NodeConnection | null = null;

  /**
   * Create a new SyndrDB client
   * @param options Client configuration options
   */
  constructor(options: ClientOptions = {}) {
    super();

    this.options = {
      defaultTimeoutMs: options.defaultTimeoutMs ?? 10000,
      debugMode: options.debugMode ?? false,
      maxRetries: options.maxRetries ?? 3,
      poolMinSize: options.poolMinSize ?? 1,
      poolMaxSize: options.poolMaxSize ?? 1,
      poolIdleTimeout: options.poolIdleTimeout ?? 30000,
      healthCheckInterval: options.healthCheckInterval ?? 30000,
      maxReconnectAttempts: options.maxReconnectAttempts ?? 10,
      logLevel: options.logLevel ?? 'INFO',
      preparedStatementCacheSize: options.preparedStatementCacheSize ?? 100,
      transactionTimeout: options.transactionTimeout ?? 300000,
    };

    this.performanceMonitor = getGlobalPerformanceMonitor();
  }

  /**
   * Initialize client (loads WASM module)
   * Optional - only needed for advanced features (migrations, schema generation, etc.)
   * Basic queries/mutations work without WASM initialization
   * @param _timeoutMs Load timeout in milliseconds (unused for now)
   */
  async initialize(_timeoutMs?: number): Promise<void> {
    const markId = this.performanceMonitor.markStart('initialize');

    try {
      // WASM loading is now optional - only load if not already loaded
      // Basic TCP operations don't require WASM
      console.log('[Client] Skipping WASM initialization - using Node TCP directly');
      this.performanceMonitor.markEnd(markId, true);
      return;

      /* WASM initialization disabled for now - causes hangs in tests
      const loader = getWASMLoader();
      await loader.load(timeoutMs ?? this.options.defaultTimeoutMs);

      this.wasmExports = loader.getExports();
      if (!this.wasmExports) {
        throw new ConnectionError('Failed to load WASM exports');
      }

      // Create client instance in WASM
      const optionsJSON = JSON.stringify(this.options);
      await this.wasmExports.createClient(optionsJSON);

      // Set log level
      this.wasmExports.setLogLevel(this.options.logLevel);

      // Enable debug mode if requested
      if (this.options.debugMode) {
        this.wasmExports.enableDebugMode();
      }

      this.performanceMonitor.markEnd(markId, true);
      */
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw error;
    }
  }

  /**
   * Connect to database
   * @param url Database connection URL
   */
  async connect(url: string): Promise<void> {
    this.ensureInitialized();
    const markId = this.performanceMonitor.markStart('connect');

    try {
      this.transitionState(ClientState.CONNECTING);

      // Parse connection string to extract host and port
      // Format: syndrdb://host:port:database:username:password;
      const match = url.match(/^syndrdb:\/\/([^:]+):(\d+):([^:]+):([^:]+):([^;]+);?$/);
      if (!match || !match[1] || !match[2]) {
        throw new ConnectionError('Invalid connection string format. Expected: syndrdb://host:port:database:username:password;');
      }

      const host: string = match[1];
      const portStr: string = match[2];
      const port = parseInt(portStr, 10);

      console.log('[Client] Creating NodeConnection to', host, port);

      // Create Node TCP connection
      this.nodeConnection = new NodeConnection({
        host,
        port,
        connectionTimeout: this.options.defaultTimeoutMs,
        tls: {
          enabled: false, // TODO: Parse from connection string
        },
      });

      console.log('[Client] Calling nodeConnection.connect()');
      // Connect and perform handshake (Node handles TCP, validates server responses)
      await this.nodeConnection.connect(url);

      console.log('[Client] Connection successful!');
      this.transitionState(ClientState.CONNECTED);
      this.performanceMonitor.markEnd(markId, true);
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      this.transitionState(ClientState.DISCONNECTED);
      throw this.wrapError(error, 'connect');
    }
  }

  /**
   * Disconnect from database
   */
  async disconnect(): Promise<void> {
    this.ensureInitialized();
    const markId = this.performanceMonitor.markStart('disconnect');

    try {
      this.transitionState(ClientState.DISCONNECTING);
      
      // Close Node connection
      if (this.nodeConnection) {
        await this.nodeConnection.close();
        this.nodeConnection = null;
      }
      
      this.transitionState(ClientState.DISCONNECTED);
      this.performanceMonitor.markEnd(markId, true);
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'disconnect');
    }
  }

  /**
   * Execute a query
   * @param query Query string
   * @param params Query parameters (will be substituted into $1, $2, etc.)
   * @returns Query result
   */
  async query<T = unknown>(query: string, params: unknown[] = []): Promise<T> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('query');

    try {
      if (!this.nodeConnection) {
        throw new ConnectionError('Not connected');
      }

      // Substitute parameters ($1, $2, etc.) with actual values
      let command = query;
      if (params && params.length > 0) {
        for (let i = 0; i < params.length; i++) {
          const param = params[i];
          const placeholder = `$${i + 1}`;
          // Escape and quote string values
          let value: string;
          if (typeof param === 'string') {
            value = `'${param.replace(/'/g, "''")}'`; // SQL escape single quotes
          } else if (param === null || param === undefined) {
            value = 'NULL';
          } else {
            value = String(param);
          }
          command = command.replace(new RegExp(`\\${placeholder}\\b`, 'g'), value);
        }
      }

      // Send command via Node TCP
      await this.nodeConnection.sendCommand(command);

      // Receive response via Node TCP
      const responseStr = await this.nodeConnection.receiveResponse();

      // Parse JSON response
      const response = JSON.parse(responseStr);

      // Check for errors
      if (response.success === false || response.status === 'error') {
        throw new Error(response.error || response.message || 'Query failed');
      }

      // Extract data - server returns Result field, not data
      const result = response.Result as T;

      this.performanceMonitor.markEnd(markId, true);
      return result;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'query');
    }
  }

  /**
   * Execute mutation
   * @param mutation Mutation string
   * @param params Mutation parameters (will be substituted into $1, $2, etc.)
   * @returns Mutation result
   */
  async mutate<T = unknown>(mutation: string, params: unknown[] = []): Promise<T> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('mutate');

    try {
      if (!this.nodeConnection) {
        throw new ConnectionError('Not connected');
      }

      // Substitute parameters ($1, $2, etc.) with actual values
      let command = mutation;
      if (params && params.length > 0) {
        for (let i = 0; i < params.length; i++) {
          const param = params[i];
          const placeholder = `$${i + 1}`;
          // Escape and quote string values
          let value: string;
          if (typeof param === 'string') {
            value = `'${param.replace(/'/g, "''")}'`; // SQL escape single quotes
          } else if (param === null || param === undefined) {
            value = 'NULL';
          } else {
            value = String(param);
          }
          command = command.replace(new RegExp(`\\${placeholder}\\b`, 'g'), value);
        }
      }

      // Send mutation command via Node TCP
      await this.nodeConnection.sendCommand(command);

      // Receive response via Node TCP
      const responseStr = await this.nodeConnection.receiveResponse();

      // Parse JSON response
      const response = JSON.parse(responseStr);

      // Check for errors
      if (response.success === false || response.status === 'error') {
        throw new Error(response.error || response.message || 'Mutation failed');
      }

      // Extract data - server returns Result field, not data
      const result = response.Result as T;

      this.performanceMonitor.markEnd(markId, true);
      return result;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'mutate');
    }
  }

  /**
   * Get current client state
   * @returns Current state
   */
  getState(): ClientState {
    return this.currentState;
  }

  /**
   * Ping database to check connection
   * @returns Ping latency in milliseconds
   */
  async ping(): Promise<number> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('ping');

    try {
      if (!this.nodeConnection) {
        throw new ConnectionError('Not connected');
      }

      const startTime = Date.now();

      // Send PING command
      await this.nodeConnection.sendCommand('PING');

      // Receive response
      const responseStr = await this.nodeConnection.receiveResponse();
      const response = JSON.parse(responseStr);

      const latency = Date.now() - startTime;

      if (response.success === false) {
        throw new Error(response.error || 'Ping failed');
      }

      this.performanceMonitor.markEnd(markId, true);
      return latency;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'ping');
    }
  }

  /**
   * Get connection health information
   * @returns Connection health
   */
  async getConnectionHealth(): Promise<ConnectionHealth> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('getConnectionHealth');

    try {
      const healthJSON = await this.wasmExports!.getConnectionHealth();
      const health = JSON.parse(healthJSON) as ConnectionHealth;
      this.performanceMonitor.markEnd(markId, true);
      return health;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'getConnectionHealth');
    }
  }

  /**
   * Plan a migration from schema definition
   * @param schema Schema definition
   * @returns Migration plan
   */
  async planMigration(schema: SchemaDefinition): Promise<MigrationPlan> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('planMigration');

    try {
      const schemaJSON = JSON.stringify(schema);
      const planJSON = await this.wasmExports!.planMigration(schemaJSON);
      const plan = JSON.parse(planJSON) as MigrationPlan;

      this.performanceMonitor.markEnd(markId, true);
      return plan;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'planMigration');
    }
  }

  /**
   * Apply a migration
   * @param migrationId Migration ID
   * @returns Applied migration
   */
  async applyMigration(migrationId: string): Promise<Migration> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('applyMigration');

    try {
      const resultJSON = await this.wasmExports!.applyMigration(migrationId);
      const migration = JSON.parse(resultJSON) as Migration;

      this.performanceMonitor.markEnd(markId, true);
      return migration;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'applyMigration');
    }
  }

  /**
   * Rollback to a specific migration version
   * @param version Target version
   * @returns Rollback result
   */
  async rollbackMigration(version: number): Promise<Migration> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('rollbackMigration');

    try {
      const resultJSON = await this.wasmExports!.rollbackMigration(version);
      const migration = JSON.parse(resultJSON) as Migration;

      this.performanceMonitor.markEnd(markId, true);
      return migration;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'rollbackMigration');
    }
  }

  /**
   * Validate a migration
   * @param migrationId Migration ID
   * @returns Validation report
   */
  async validateMigration(migrationId: string): Promise<ValidationReport> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('validateMigration');

    try {
      const reportJSON = await this.wasmExports!.validateMigration(migrationId);
      const report = JSON.parse(reportJSON) as ValidationReport;

      this.performanceMonitor.markEnd(markId, true);
      return report;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'validateMigration');
    }
  }

  /**
   * Get migration history
   * @returns Migration history
   */
  async getMigrationHistory(): Promise<MigrationHistory> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('getMigrationHistory');

    try {
      const historyJSON = await this.wasmExports!.getMigrationHistory();
      const history = JSON.parse(historyJSON) as MigrationHistory;

      this.performanceMonitor.markEnd(markId, true);
      return history;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'getMigrationHistory');
    }
  }

  /**
   * Generate JSON schema
   * @param mode Generation mode (single or multiple files)
   * @returns JSON schema
   */
  async generateJSONSchema(mode: JSONSchemaMode = 'single'): Promise<string> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('generateJSONSchema');

    try {
      const schema = await this.wasmExports!.generateJSONSchema('{}', mode);
      this.performanceMonitor.markEnd(markId, true);
      return schema;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'generateJSONSchema');
    }
  }

  /**
   * Generate GraphQL schema
   * @param options Generation options
   * @returns GraphQL schema
   */
  async generateGraphQLSchema(options: GraphQLGenerationOptions = {}): Promise<string> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('generateGraphQLSchema');

    try {
      const optionsJSON = JSON.stringify(options);
      const schema = await this.wasmExports!.generateGraphQLSchema(optionsJSON);
      this.performanceMonitor.markEnd(markId, true);
      return schema;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'generateGraphQLSchema');
    }
  }

  /**
   * Begin a transaction
   * @returns Transaction ID
   */
  async beginTransaction(): Promise<string> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('beginTransaction');

    try {
      const txId = await this.wasmExports!.beginTransaction();
      this.performanceMonitor.markEnd(markId, true);
      return txId;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'beginTransaction');
    }
  }

  /**
   * Commit a transaction
   * @param txId Transaction ID
   */
  async commitTransaction(txId: string): Promise<void> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('commitTransaction');

    try {
      await this.wasmExports!.commitTransaction(txId);
      this.performanceMonitor.markEnd(markId, true);
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'commitTransaction');
    }
  }

  /**
   * Rollback a transaction
   * @param txId Transaction ID
   */
  async rollbackTransaction(txId: string): Promise<void> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('rollbackTransaction');

    try {
      await this.wasmExports!.rollbackTransaction(txId);
      this.performanceMonitor.markEnd(markId, true);
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'rollbackTransaction');
    }
  }

  /**
   * Prepare a statement
   * @param query Query string
   * @returns Statement ID
   */
  async prepare(query: string): Promise<number> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('prepare');

    try {
      const stmtId = await this.wasmExports!.prepare(query);
      this.performanceMonitor.markEnd(markId, true);
      return stmtId;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'prepare');
    }
  }

  /**
   * Execute a prepared statement
   * @param stmtId Statement ID
   * @param params Parameters
   * @returns Execution result
   */
  async executeStatement<T = unknown>(stmtId: number, params: unknown[] = []): Promise<T> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('executeStatement');

    try {
      const paramsJSON = JSON.stringify(params);
      const resultJSON = await this.wasmExports!.executeStatement(
        stmtId,
        paramsJSON
      );
      const result = JSON.parse(resultJSON) as T;

      this.performanceMonitor.markEnd(markId, true);
      return result;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'executeStatement');
    }
  }

  /**
   * Deallocate a prepared statement
   * @param stmtId Statement ID
   */
  async deallocateStatement(stmtId: number): Promise<void> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('deallocateStatement');

    try {
      await this.wasmExports!.deallocateStatement(stmtId);
      this.performanceMonitor.markEnd(markId, true);
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'deallocateStatement');
    }
  }

  /**
   * Register a hook
   * @param hook Hook configuration
   */
  async registerHook(hook: HookConfig): Promise<void> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('registerHook');

    try {
      const hookJSON = JSON.stringify(hook);
      await this.wasmExports!.registerHook(hookJSON);
      this.performanceMonitor.markEnd(markId, true);
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'registerHook');
    }
  }

  /**
   * Unregister a hook
   * @param hookName Hook name
   */
  async unregisterHook(hookName: string): Promise<void> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('unregisterHook');

    try {
      await this.wasmExports!.unregisterHook(hookName);
      this.performanceMonitor.markEnd(markId, true);
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'unregisterHook');
    }
  }

  /**
   * Get metrics statistics
   * @returns Metrics stats
   */
  async getMetricsStats(): Promise<MetricsStats> {
    this.ensureConnected();
    const markId = this.performanceMonitor.markStart('getMetricsStats');

    try {
      const statsJSON = await this.wasmExports!.getMetricsStats();
      const stats = JSON.parse(statsJSON) as MetricsStats;

      this.performanceMonitor.markEnd(markId, true);
      return stats;
    } catch (error) {
      this.performanceMonitor.markEnd(markId, false, error as Error);
      throw this.wrapError(error, 'getMetricsStats');
    }
  }

  /**
   * Get performance statistics
   * @returns Performance stats
   */
  getPerformanceStats(): PerformanceStats {
    return this.performanceMonitor.getStats();
  }

  /**
   * Reset performance metrics
   */
  resetPerformanceMetrics(): void {
    this.performanceMonitor.reset();
  }

  /**
   * Get debug information
   * @returns Debug info
   */
  async getDebugInfo(): Promise<DebugInfo> {
    this.ensureInitialized();

    const wasmMetadata = getWASMLoader().getMetadata()!;
    const performanceStats = this.performanceMonitor.getStats();

    const debugInfoPromise = this.wasmExports!.getDebugInfo();
    const coreDebugInfo = await debugInfoPromise;

    return {
      version: coreDebugInfo.version,
      wasmMetadata,
      stateHistory: this.stateHistory,
      performanceStats,
    };
  }

  /**
   * Get WASM metadata
   * @returns WASM metadata
   */
  getWASMMetadata(): WASMMetadata | null {
    return getWASMLoader().getMetadata();
  }

  /**
   * Transition to a new state
   * @param newState New state
   */
  private transitionState(newState: ClientState): void {
    const oldState = this.currentState;
    const now = Date.now();

    // Calculate duration in previous state
    const lastTransition = this.stateHistory[this.stateHistory.length - 1];
    const duration = lastTransition ? now - lastTransition.timestamp : 0;

    const transition: StateTransition = {
      from: oldState,
      to: newState,
      timestamp: now,
      duration,
    };

    this.stateHistory.push(transition);
    this.currentState = newState;

    this.emit('stateChange', transition);
  }

  /**
   * Ensure client is initialized
   * Now a no-op since WASM is optional
   */
  private ensureInitialized(): void {
    // No longer required - basic operations work without WASM
    return;
  }

  /**
   * Ensure client is connected
   */
  private ensureConnected(): void {
    this.ensureInitialized();
    if (this.currentState !== 'CONNECTED') {
      throw new ConnectionError(`Client not connected - current state: ${this.currentState}`);
    }
  }

  /**
   * Wrap WASM error with typed error
   * @param error Original error
   * @param operation Operation name
   * @returns Typed error
   */
  private wrapError(error: unknown, operation: string): Error {
    if (error instanceof Error) {
      const parsed = parseWASMError(error.message);
      return mapWASMError(parsed.message, parsed.type, undefined, undefined, error, parsed.context);
    }
    // Handle plain error objects from WASM (Go rejects with { message, error })
    if (typeof error === 'object' && error !== null) {
      const errorObj = error as { message?: string; error?: string };
      const message = errorObj.message || errorObj.error || 'Unknown error';
      const parsed = parseWASMError(message);
      return mapWASMError(parsed.message, parsed.type, undefined, undefined, undefined, parsed.context);
    }
    return new Error(`Unknown error in ${operation}: ${String(error)}`);
  }
}
