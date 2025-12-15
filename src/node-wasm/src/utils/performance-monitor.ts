/**
 * Performance monitoring for WASM operations
 * Tracks operation metrics, boundary crossings, and memory usage
 */

import { performance } from 'perf_hooks';
import type {
  PerformanceStats,
  OperationMetrics,
  BoundaryMetrics,
  MemoryMetrics,
  PerformanceMonitorOptions,
} from '../types/performance';

/**
 * Operation timing entry
 */
interface OperationEntry {
  name: string;
  startTime: number;
  endTime?: number;
  success?: boolean;
  error?: Error;
}

/**
 * Performance monitor for tracking operation metrics
 */
export class PerformanceMonitor {
  private operations: Map<string, number[]> = new Map();
  private errors: Map<string, number> = new Map();
  private boundaryCrossings = 0;
  private batchedOperations = 0;
  private crossingsSaved = 0;
  private startTime: Date;
  private options: Required<PerformanceMonitorOptions>;
  private activeOperations: Map<string, OperationEntry> = new Map();

  constructor(options: PerformanceMonitorOptions = {}) {
    this.startTime = new Date();
    this.options = {
      maxOperations: options.maxOperations ?? 10000,
      trackMemory: options.trackMemory ?? true,
      trackBoundaries: options.trackBoundaries ?? true,
      sampleRate: options.sampleRate ?? 1.0,
    };
  }

  /**
   * Mark the start of an operation
   * @param operationName Operation identifier
   * @returns Mark ID for later measurement
   */
  markStart(operationName: string): string {
    if (Math.random() > this.options.sampleRate) {
      return ''; // Skip sampling
    }

    const markId = `${operationName}-${Date.now()}-${Math.random()}`;
    performance.mark(`${markId}-start`);

    this.activeOperations.set(markId, {
      name: operationName,
      startTime: performance.now(),
    });

    return markId;
  }

  /**
   * Mark the end of an operation and record metrics
   * @param markId Mark ID from markStart
   * @param success Whether operation succeeded
   * @param error Error if operation failed
   */
  markEnd(markId: string, success = true, error?: Error): void {
    if (!markId) return; // Skipped sample

    const entry = this.activeOperations.get(markId);
    if (!entry) return;

    const endTime = performance.now();
    entry.endTime = endTime;
    entry.success = success;
    if (error) {
      entry.error = error;
    }

    performance.mark(`${markId}-end`);
    performance.measure(markId, `${markId}-start`, `${markId}-end`);

    const duration = endTime - entry.startTime;
    this.recordOperation(entry.name, duration, success);

    this.activeOperations.delete(markId);

    // Track boundary crossing
    if (this.options.trackBoundaries) {
      this.boundaryCrossings++;
    }
  }

  /**
   * Record operation timing
   * @param operationName Operation name
   * @param durationMs Duration in milliseconds
   * @param success Whether operation succeeded
   */
  private recordOperation(operationName: string, durationMs: number, success: boolean): void {
    // Initialize arrays if needed
    if (!this.operations.has(operationName)) {
      this.operations.set(operationName, []);
      this.errors.set(operationName, 0);
    }

    const timings = this.operations.get(operationName)!;
    timings.push(durationMs);

    // Limit stored timings to prevent memory bloat
    if (timings.length > this.options.maxOperations) {
      timings.shift();
    }

    if (!success) {
      this.errors.set(operationName, (this.errors.get(operationName) ?? 0) + 1);
    }
  }

  /**
   * Record batched operations
   * @param count Number of operations in batch
   * @param crossingsSaved Boundary crossings avoided
   */
  recordBatch(count: number, crossingsSaved: number): void {
    this.batchedOperations += count;
    this.crossingsSaved += crossingsSaved;
  }

  /**
   * Get comprehensive performance statistics
   * @returns Performance stats
   */
  getStats(): PerformanceStats {
    const endTime = new Date();
    const durationMs = endTime.getTime() - this.startTime.getTime();

    const operations: Record<string, OperationMetrics> = {};

    for (const [name, timings] of this.operations.entries()) {
      if (timings.length === 0) continue;

      const sorted = [...timings].sort((a, b) => a - b);
      const count = timings.length;
      const errorCount = this.errors.get(name) ?? 0;
      const successCount = count - errorCount;
      const total = timings.reduce((sum, t) => sum + t, 0);
      const avg = total / count;
      const min = sorted[0];
      const max = sorted[sorted.length - 1];

      // Calculate percentiles
      const p50Idx = Math.floor(count * 0.5);
      const p95Idx = Math.floor(count * 0.95);
      const p99Idx = Math.floor(count * 0.99);

      const p50Ms = sorted[p50Idx] ?? avg;
      const p95Ms = sorted[p95Idx] ?? max;
      const p99Ms = sorted[p99Idx] ?? max;

      // Calculate standard deviation
      const variance = timings.reduce((sum, t) => sum + Math.pow(t - avg, 2), 0) / count;
      const stdDevMs = Math.sqrt(variance);

      const opsPerSecond = (count / durationMs) * 1000;

      operations[name] = {
        count,
        successCount,
        errorCount,
        totalMs: total,
        avgMs: avg,
        minMs: min ?? 0,
        maxMs: max ?? 0,
        p50Ms: p50Ms ?? 0,
        p95Ms: p95Ms ?? 0,
        p99Ms: p99Ms ?? 0,
        stdDevMs,
        opsPerSecond,
      };
    }

    const boundaries = this.getBoundaryMetrics(durationMs);
    const memory = this.options.trackMemory ? this.getMemoryMetrics() : this.getEmptyMemoryMetrics();

    return {
      operations,
      boundaries,
      memory,
      startTime: this.startTime,
      endTime,
      durationMs,
    };
  }

  /**
   * Get boundary crossing metrics
   * @param totalDurationMs Total monitoring duration
   * @returns Boundary metrics
   */
  private getBoundaryMetrics(totalDurationMs: number): BoundaryMetrics {
    const avgOverheadMs = this.boundaryCrossings > 0 ? 0.05 : 0; // Estimated 0.05ms per crossing
    const totalOverheadMs = this.boundaryCrossings * avgOverheadMs;
    const overheadPercentage = totalDurationMs > 0 ? (totalOverheadMs / totalDurationMs) * 100 : 0;

    return {
      totalCrossings: this.boundaryCrossings,
      avgOverheadMs,
      totalOverheadMs,
      overheadPercentage,
      batchedOperations: this.batchedOperations,
      crossingsSaved: this.crossingsSaved,
    };
  }

  /**
   * Get memory usage metrics
   * @returns Memory metrics
   */
  private getMemoryMetrics(): MemoryMetrics {
    const memUsage = process.memoryUsage();

    return {
      heapUsed: memUsage.heapUsed,
      heapTotal: memUsage.heapTotal,
      peakHeapUsed: memUsage.heapUsed, // TODO: Track actual peak
      external: memUsage.external,
      rss: memUsage.rss,
      gcCount: 0, // TODO: Track via PerformanceObserver
      gcPauseMs: 0, // TODO: Track via PerformanceObserver
    };
  }

  /**
   * Get empty memory metrics
   * @returns Empty memory metrics
   */
  private getEmptyMemoryMetrics(): MemoryMetrics {
    return {
      heapUsed: 0,
      heapTotal: 0,
      peakHeapUsed: 0,
      external: 0,
      rss: 0,
      gcCount: 0,
      gcPauseMs: 0,
    };
  }

  /**
   * Reset all metrics
   */
  reset(): void {
    this.operations.clear();
    this.errors.clear();
    this.boundaryCrossings = 0;
    this.batchedOperations = 0;
    this.crossingsSaved = 0;
    this.startTime = new Date();
    this.activeOperations.clear();
  }

  /**
   * Export metrics to JSON file
   * @param filePath Output file path
   */
  async exportToFile(filePath: string): Promise<void> {
    const stats = this.getStats();
    const { writeFile } = await import('fs/promises');
    await writeFile(filePath, JSON.stringify(stats, null, 2), 'utf-8');
  }
}

/**
 * Global performance monitor instance
 */
let globalMonitor: PerformanceMonitor | null = null;

/**
 * Get global performance monitor
 * @returns Performance monitor instance
 */
export function getGlobalPerformanceMonitor(): PerformanceMonitor {
  if (!globalMonitor) {
    globalMonitor = new PerformanceMonitor();
  }
  return globalMonitor;
}

/**
 * Reset global monitor (for testing)
 */
export function resetGlobalPerformanceMonitor(): void {
  globalMonitor = null;
}
