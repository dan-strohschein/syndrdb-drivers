/**
 * Performance statistics collected by PerformanceMonitor
 */
export interface PerformanceStats {
  /** Statistics by operation name */
  operations: Record<string, OperationMetrics>;

  /** WASM boundary crossing metrics */
  boundaries: BoundaryMetrics;

  /** Memory usage metrics */
  memory: MemoryMetrics;

  /** Collection start time */
  startTime: Date;

  /** Collection end time */
  endTime: Date;

  /** Total collection duration (ms) */
  durationMs: number;
}

/**
 * Metrics for a specific operation type
 */
export interface OperationMetrics {
  /** Total number of operations */
  count: number;

  /** Number of successful operations */
  successCount: number;

  /** Number of failed operations */
  errorCount: number;

  /** Total execution time (ms) */
  totalMs: number;

  /** Average execution time (ms) */
  avgMs: number;

  /** Minimum execution time (ms) */
  minMs: number;

  /** Maximum execution time (ms) */
  maxMs: number;

  /** 50th percentile (median) */
  p50Ms: number;

  /** 95th percentile */
  p95Ms: number;

  /** 99th percentile */
  p99Ms: number;

  /** Standard deviation */
  stdDevMs: number;

  /** Operations per second */
  opsPerSecond: number;
}

/**
 * WASM boundary crossing metrics
 */
export interface BoundaryMetrics {
  /** Total number of boundary crossings */
  totalCrossings: number;

  /** Average crossing overhead (ms) */
  avgOverheadMs: number;

  /** Total overhead (ms) */
  totalOverheadMs: number;

  /** Percentage of time spent in boundary crossing */
  overheadPercentage: number;

  /** Number of batched operations */
  batchedOperations: number;

  /** Boundary crossings saved by batching */
  crossingsSaved: number;
}

/**
 * Memory usage metrics
 */
export interface MemoryMetrics {
  /** Current heap used (bytes) */
  heapUsed: number;

  /** Current heap total (bytes) */
  heapTotal: number;

  /** Peak heap used (bytes) */
  peakHeapUsed: number;

  /** External memory (bytes) */
  external: number;

  /** RSS (Resident Set Size) in bytes */
  rss: number;

  /** Number of garbage collections */
  gcCount: number;

  /** Total GC pause time (ms) */
  gcPauseMs: number;
}

/**
 * Performance monitoring options
 */
export interface PerformanceMonitorOptions {
  /** Maximum number of operations to track */
  maxOperations?: number;

  /** Whether to track memory usage */
  trackMemory?: boolean;

  /** Whether to track boundary crossings */
  trackBoundaries?: boolean;

  /** Sample rate (0.0 to 1.0) */
  sampleRate?: number;
}

/**
 * Performance regression test result
 */
export interface RegressionTestResult {
  /** Whether test passed */
  passed: boolean;

  /** Baseline metrics */
  baseline: PerformanceStats;

  /** Current metrics */
  current: PerformanceStats;

  /** Regressions detected */
  regressions: PerformanceRegression[];

  /** Improvements detected */
  improvements: PerformanceImprovement[];

  /** Overall regression percentage */
  overallRegressionPercentage: number;
}

/**
 * Performance regression details
 */
export interface PerformanceRegression {
  /** Operation name */
  operation: string;

  /** Metric that regressed */
  metric: string;

  /** Baseline value */
  baselineValue: number;

  /** Current value */
  currentValue: number;

  /** Regression percentage */
  regressionPercentage: number;

  /** Whether this is above threshold */
  aboveThreshold: boolean;
}

/**
 * Performance improvement details
 */
export interface PerformanceImprovement {
  /** Operation name */
  operation: string;

  /** Metric that improved */
  metric: string;

  /** Baseline value */
  baselineValue: number;

  /** Current value */
  currentValue: number;

  /** Improvement percentage */
  improvementPercentage: number;
}
