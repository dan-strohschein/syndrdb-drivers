/**
 * Main exports for @syndrdb/node-wasm
 */

// Client
export { SyndrDBClient } from './client';

// WASM Loader
export { getWASMLoader, WASMLoader } from './wasm-loader';
export type { WASMExports } from './wasm-loader';

// Types
export type {
  ClientOptions,
  StateTransition,
  ConnectionHealth,
  DebugInfo,
  WASMMetadata,
  VersionInfo,
  LogLevel,
} from './types/client';
export { ClientState } from './types/client';

export type {
  Migration,
  MigrationHistory,
  MigrationPlan,
  ValidationReport,
  RollbackValidationReport,
  SchemaDiff,
  MigrationStatus,
  DiffType,
} from './types/migration';

export type {
  FieldDefinition,
  IndexDefinition,
  BundleDefinition,
  SchemaDefinition,
  FieldType,
  IndexType,
  JSONSchemaMode,
  TypeScriptGenerationOptions,
  GraphQLGenerationOptions,
} from './types/schema';

export type {
  HookContext,
  Hook,
  HookConfig,
  MetricsStats,
  OperationTypeStats,
  TraceSpan,
  TraceEvent,
  LoggingHookOptions,
  RetryHookOptions,
} from './types/hooks';

export type {
  PerformanceStats,
  OperationMetrics,
  BoundaryMetrics,
  MemoryMetrics,
  PerformanceMonitorOptions,
  RegressionTestResult,
  PerformanceRegression,
  PerformanceImprovement,
} from './types/performance';

export type {
  ErrorContext,
  StackFrame,
  SourceMapEntry,
  ErrorCode,
  ErrorType,
} from './types/errors';

// Errors
export {
  SyndrDBError,
  ConnectionError,
  QueryError,
  MutationError,
  MigrationError,
  ValidationError,
  TransactionError,
  HookError,
  StatementError,
  WASMError,
  WASMLoadError,
  WASMNotFoundError,
  WASMTimeoutError,
  WASMVersionMismatchError,
  WASMCorruptedError,
  TimeoutError,
} from './errors';

// Utils
export {
  PerformanceMonitor,
  getGlobalPerformanceMonitor,
  BatchOptimizer,
} from './utils';
export type { BatchableOperation, BatchOptimizerOptions } from './utils';

// Features
export {
  MigrationManager,
  TransactionManager,
  Transaction,
  HookManager,
  SchemaManager,
  SchemaBuilder,
  StatementManager,
  PreparedStatement,
} from './features';
export type {
  MigrationManagerOptions,
  TransactionOptions,
  TransactionState,
  StatementCacheOptions,
} from './features';
