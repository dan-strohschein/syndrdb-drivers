/**
 * Hook context passed to hook functions
 */
export interface HookContext {
  /** The command being executed */
  command: string;

  /** Command type (query, mutation, etc.) */
  commandType: string;

  /** Trace ID for distributed tracing */
  traceId: string;

  /** Operation start timestamp */
  startTime: Date;

  /** Query parameters (if applicable) */
  params?: unknown[];

  /** Additional metadata */
  metadata: Record<string, unknown>;

  /** Operation result (populated in after hooks) */
  result?: unknown;

  /** Error if operation failed */
  error?: Error;

  /** Operation duration in milliseconds */
  durationMs?: number;
}

/**
 * Hook interface for custom hooks
 */
export interface Hook {
  /** Hook name (must be unique) */
  readonly name: string;

  /**
   * Called before operation executes
   * Can modify context or abort operation
   */
  before?(ctx: HookContext): Promise<void> | void;

  /**
   * Called after operation completes
   * Can inspect/log results
   */
  after?(ctx: HookContext): Promise<void> | void;
}

/**
 * Hook configuration for registration
 */
export interface HookConfig {
  /** Hook name */
  name: string;

  /** Before hook function */
  before?: (ctx: HookContext) => Promise<void> | void;

  /** After hook function */
  after?: (ctx: HookContext) => Promise<void> | void;
}

/**
 * Metrics statistics from MetricsHook
 */
export interface MetricsStats {
  /** Total number of operations */
  totalOperations: number;

  /** Number of successful operations */
  successCount: number;

  /** Number of failed operations */
  errorCount: number;

  /** Total execution time (ms) */
  totalDurationMs: number;

  /** Average execution time (ms) */
  avgDurationMs: number;

  /** Minimum execution time (ms) */
  minDurationMs: number;

  /** Maximum execution time (ms) */
  maxDurationMs: number;

  /** 95th percentile execution time (ms) */
  p95DurationMs: number;

  /** 99th percentile execution time (ms) */
  p99DurationMs: number;

  /** Operations per second */
  operationsPerSecond: number;

  /** Error rate (percentage) */
  errorRate: number;

  /** Statistics by operation type */
  byType: Record<string, OperationTypeStats>;
}

/**
 * Statistics for a specific operation type
 */
export interface OperationTypeStats {
  /** Number of operations */
  count: number;

  /** Number of errors */
  errors: number;

  /** Average duration (ms) */
  avgDurationMs: number;

  /** Minimum duration (ms) */
  minDurationMs: number;

  /** Maximum duration (ms) */
  maxDurationMs: number;
}

/**
 * Tracing span for distributed tracing
 */
export interface TraceSpan {
  /** Span ID */
  spanId: string;

  /** Trace ID */
  traceId: string;

  /** Parent span ID */
  parentSpanId?: string;

  /** Operation name */
  operationName: string;

  /** Start timestamp */
  startTime: Date;

  /** End timestamp */
  endTime?: Date;

  /** Span attributes */
  attributes: Record<string, unknown>;

  /** Span events */
  events: TraceEvent[];

  /** Span status */
  status: 'OK' | 'ERROR';

  /** Error message if failed */
  error?: string;
}

/**
 * Event within a trace span
 */
export interface TraceEvent {
  /** Event name */
  name: string;

  /** Event timestamp */
  timestamp: Date;

  /** Event attributes */
  attributes: Record<string, unknown>;
}

/**
 * Logging hook options
 */
export interface LoggingHookOptions {
  /** Whether to log commands */
  logCommands?: boolean;

  /** Whether to log results */
  logResults?: boolean;

  /** Whether to log durations */
  logDurations?: boolean;

  /** Minimum log level */
  logLevel?: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
}

/**
 * Retry hook options
 */
export interface RetryHookOptions {
  /** Maximum number of retry attempts */
  maxAttempts?: number;

  /** Initial retry delay (ms) */
  initialDelayMs?: number;

  /** Maximum retry delay (ms) */
  maxDelayMs?: number;

  /** Backoff multiplier */
  backoffMultiplier?: number;

  /** Whether to retry on specific errors only */
  retryableErrors?: string[];
}
