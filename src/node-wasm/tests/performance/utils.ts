/**
 * Performance Test Utilities
 * 
 * Helper functions and utilities for performance testing and benchmarking.
 */

import { performance } from 'perf_hooks';

/**
 * Benchmark result statistics
 */
export interface BenchmarkStats {
  operation: string;
  iterations: number;
  totalDuration: number;
  averageDuration: number;
  minDuration: number;
  maxDuration: number;
  opsPerSecond: number;
  percentiles: {
    p50: number;
    p90: number;
    p95: number;
    p99: number;
  };
}

/**
 * Run a benchmark and collect detailed statistics
 */
export async function benchmark(
  name: string,
  iterations: number,
  operation: () => Promise<void>,
  options?: {
    warmup?: number;
    onProgress?: (current: number, total: number) => void;
  }
): Promise<BenchmarkStats> {
  const warmupIterations = options?.warmup ?? 0;
  
  // Warmup phase
  if (warmupIterations > 0) {
    for (let i = 0; i < warmupIterations; i++) {
      await operation();
    }
  }

  // Benchmark phase
  const durations: number[] = [];
  
  for (let i = 0; i < iterations; i++) {
    const start = performance.now();
    await operation();
    const end = performance.now();
    durations.push(end - start);
    
    if (options?.onProgress) {
      options.onProgress(i + 1, iterations);
    }
  }

  // Calculate statistics
  durations.sort((a, b) => a - b);
  const totalDuration = durations.reduce((sum, d) => sum + d, 0);
  const averageDuration = totalDuration / iterations;
  const minDuration = durations[0];
  const maxDuration = durations[durations.length - 1];
  const opsPerSecond = 1000 / averageDuration;

  return {
    operation: name,
    iterations,
    totalDuration,
    averageDuration,
    minDuration,
    maxDuration,
    opsPerSecond,
    percentiles: {
      p50: percentile(durations, 0.5),
      p90: percentile(durations, 0.9),
      p95: percentile(durations, 0.95),
      p99: percentile(durations, 0.99),
    },
  };
}

/**
 * Calculate percentile from sorted array
 */
export function percentile(sortedValues: number[], p: number): number {
  const index = Math.floor(sortedValues.length * p);
  return sortedValues[Math.min(index, sortedValues.length - 1)];
}

/**
 * Compare two benchmark results and calculate regression
 */
export function compareResults(
  baseline: BenchmarkStats,
  current: BenchmarkStats
): {
  operation: string;
  averageRegression: number;
  p95Regression: number;
  opsImprovement: number;
  verdict: 'improved' | 'stable' | 'regressed';
} {
  const averageRegression = (current.averageDuration - baseline.averageDuration) / baseline.averageDuration;
  const p95Regression = (current.percentiles.p95 - baseline.percentiles.p95) / baseline.percentiles.p95;
  const opsImprovement = (current.opsPerSecond - baseline.opsPerSecond) / baseline.opsPerSecond;

  let verdict: 'improved' | 'stable' | 'regressed';
  if (averageRegression > 0.1 || p95Regression > 0.15) {
    verdict = 'regressed';
  } else if (averageRegression < -0.05 && p95Regression < -0.05) {
    verdict = 'improved';
  } else {
    verdict = 'stable';
  }

  return {
    operation: current.operation,
    averageRegression,
    p95Regression,
    opsImprovement,
    verdict,
  };
}

/**
 * Format benchmark results for display
 */
export function formatResults(stats: BenchmarkStats): string {
  const lines = [
    '='.repeat(70),
    `Benchmark: ${stats.operation}`,
    '='.repeat(70),
    `Iterations:        ${stats.iterations.toLocaleString()}`,
    `Total Duration:    ${stats.totalDuration.toFixed(2)}ms`,
    `Average Duration:  ${stats.averageDuration.toFixed(2)}ms`,
    `Min Duration:      ${stats.minDuration.toFixed(2)}ms`,
    `Max Duration:      ${stats.maxDuration.toFixed(2)}ms`,
    `Ops/Second:        ${stats.opsPerSecond.toFixed(2)}`,
    '',
    'Percentiles:',
    `  P50 (median):    ${stats.percentiles.p50.toFixed(2)}ms`,
    `  P90:             ${stats.percentiles.p90.toFixed(2)}ms`,
    `  P95:             ${stats.percentiles.p95.toFixed(2)}ms`,
    `  P99:             ${stats.percentiles.p99.toFixed(2)}ms`,
    '='.repeat(70),
  ];

  return lines.join('\n');
}

/**
 * Format comparison results for display
 */
export function formatComparison(
  comparison: ReturnType<typeof compareResults>,
  baseline: BenchmarkStats,
  current: BenchmarkStats
): string {
  const avgChange = (comparison.averageRegression * 100).toFixed(1);
  const p95Change = (comparison.p95Regression * 100).toFixed(1);
  const opsChange = (comparison.opsImprovement * 100).toFixed(1);

  const verdictEmoji = {
    improved: '✓',
    stable: '→',
    regressed: '✗',
  }[comparison.verdict];

  const lines = [
    '',
    `${verdictEmoji} ${comparison.operation}`,
    `  Average: ${baseline.averageDuration.toFixed(2)}ms → ${current.averageDuration.toFixed(2)}ms (${avgChange > 0 ? '+' : ''}${avgChange}%)`,
    `  P95:     ${baseline.percentiles.p95.toFixed(2)}ms → ${current.percentiles.p95.toFixed(2)}ms (${p95Change > 0 ? '+' : ''}${p95Change}%)`,
    `  Ops/s:   ${baseline.opsPerSecond.toFixed(2)} → ${current.opsPerSecond.toFixed(2)} (${opsChange > 0 ? '+' : ''}${opsChange}%)`,
  ];

  return lines.join('\n');
}

/**
 * Measure memory usage of an operation
 */
export async function measureMemory(operation: () => Promise<void>): Promise<{
  heapUsedBefore: number;
  heapUsedAfter: number;
  heapUsedDelta: number;
  externalBefore: number;
  externalAfter: number;
  externalDelta: number;
}> {
  // Force garbage collection if available
  if (global.gc) {
    global.gc();
  }

  const before = process.memoryUsage();
  await operation();
  const after = process.memoryUsage();

  return {
    heapUsedBefore: before.heapUsed,
    heapUsedAfter: after.heapUsed,
    heapUsedDelta: after.heapUsed - before.heapUsed,
    externalBefore: before.external,
    externalAfter: after.external,
    externalDelta: after.external - before.external,
  };
}

/**
 * Format memory usage for display
 */
export function formatMemory(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB'];
  let value = bytes;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex++;
  }

  return `${value.toFixed(2)} ${units[unitIndex]}`;
}

/**
 * Progress bar for long-running benchmarks
 */
export class ProgressBar {
  private total: number;
  private current: number = 0;
  private barLength: number = 40;
  private startTime: number;

  constructor(total: number) {
    this.total = total;
    this.startTime = Date.now();
  }

  update(current: number): void {
    this.current = current;
    this.render();
  }

  private render(): void {
    const percentage = (this.current / this.total) * 100;
    const filledLength = Math.round((this.barLength * this.current) / this.total);
    const bar = '█'.repeat(filledLength) + '░'.repeat(this.barLength - filledLength);
    
    const elapsed = Date.now() - this.startTime;
    const eta = this.current > 0 ? (elapsed / this.current) * (this.total - this.current) : 0;
    
    process.stdout.write(
      `\r  Progress: [${bar}] ${percentage.toFixed(1)}% (${this.current}/${this.total}) ETA: ${Math.round(eta / 1000)}s`
    );

    if (this.current >= this.total) {
      process.stdout.write('\n');
    }
  }

  finish(): void {
    this.current = this.total;
    this.render();
  }
}

/**
 * Create a simple timer for measuring durations
 */
export class Timer {
  private startTime: number = 0;
  private endTime: number = 0;

  start(): void {
    this.startTime = performance.now();
  }

  stop(): number {
    this.endTime = performance.now();
    return this.duration();
  }

  duration(): number {
    return this.endTime - this.startTime;
  }

  reset(): void {
    this.startTime = 0;
    this.endTime = 0;
  }
}

/**
 * Run operations in parallel and measure aggregate performance
 */
export async function benchmarkParallel(
  name: string,
  concurrency: number,
  operations: Array<() => Promise<void>>
): Promise<BenchmarkStats> {
  const startTime = performance.now();
  
  await Promise.all(operations.map(op => op()));
  
  const endTime = performance.now();
  const totalDuration = endTime - startTime;
  const operationCount = operations.length;
  const averageDuration = totalDuration / operationCount;
  const opsPerSecond = (1000 * operationCount) / totalDuration;

  return {
    operation: name,
    iterations: operationCount,
    totalDuration,
    averageDuration,
    minDuration: totalDuration, // Not accurate for parallel
    maxDuration: totalDuration, // Not accurate for parallel
    opsPerSecond,
    percentiles: {
      p50: averageDuration,
      p90: averageDuration,
      p95: averageDuration,
      p99: averageDuration,
    },
  };
}

/**
 * Statistical helpers
 */
export function mean(values: number[]): number {
  return values.reduce((sum, v) => sum + v, 0) / values.length;
}

export function median(values: number[]): number {
  const sorted = [...values].sort((a, b) => a - b);
  const mid = Math.floor(sorted.length / 2);
  return sorted.length % 2 === 0 
    ? (sorted[mid - 1] + sorted[mid]) / 2 
    : sorted[mid];
}

export function standardDeviation(values: number[]): number {
  const avg = mean(values);
  const squareDiffs = values.map(v => Math.pow(v - avg, 2));
  const avgSquareDiff = mean(squareDiffs);
  return Math.sqrt(avgSquareDiff);
}

export function variance(values: number[]): number {
  const avg = mean(values);
  const squareDiffs = values.map(v => Math.pow(v - avg, 2));
  return mean(squareDiffs);
}
