/**
 * Core type exports for @syndrdb/node-wasm
 * 
 * This module re-exports all type definitions used by the driver.
 */

// Client types
export type {
  ClientOptions,
  StateTransition,
  ConnectionHealth,
  DebugInfo,
  WASMMetadata,
  VersionInfo,
  LogLevel,
} from './client';
export { ClientState } from './client';

// Migration types
export type {
  Migration,
  MigrationHistory,
  MigrationPlan,
  ValidationReport,
  ValidationError,
  RollbackValidationReport,
  SchemaDiff,
  MigrationStatus,
  DiffType,
} from './migration';

// Schema types
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
} from './schema';

// Hook types
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
} from './hooks';

// Performance types
export type {
  PerformanceStats,
  OperationMetrics,
  BoundaryMetrics,
  MemoryMetrics,
  PerformanceMonitorOptions,
  RegressionTestResult,
  PerformanceRegression,
  PerformanceImprovement,
} from './performance';

// Error types
export type {
  ErrorContext,
  StackFrame,
  SourceMapEntry,
  ErrorCode,
  ErrorType,
} from './errors';
