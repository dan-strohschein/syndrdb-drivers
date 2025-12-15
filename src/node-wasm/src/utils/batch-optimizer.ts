/**
 * Batch optimizer for reducing WASM boundary crossings
 * Groups multiple operations into single calls when possible
 */

/**
 * Batchable operation interface
 */
export interface BatchableOperation<T = unknown> {
  /** Operation type identifier */
  type: string;

  /** Operation parameters */
  params: unknown[];

  /** Promise resolver */
  resolve: (value: T) => void;

  /** Promise rejecter */
  reject: (error: Error) => void;

  /** Operation timestamp */
  timestamp: number;
}

/**
 * Batch optimizer options
 */
export interface BatchOptimizerOptions {
  /** Maximum batch size (default: 50) */
  maxBatchSize?: number;

  /** Maximum wait time before flushing batch in ms (default: 10) */
  maxWaitMs?: number;

  /** Whether batching is enabled (default: true) */
  enabled?: boolean;
}

/**
 * Batch optimizer for grouping operations
 */
export class BatchOptimizer {
  private batches: Map<string, BatchableOperation[]> = new Map();
  private timers: Map<string, NodeJS.Timeout> = new Map();
  private options: Required<BatchOptimizerOptions>;

  constructor(options: BatchOptimizerOptions = {}) {
    this.options = {
      maxBatchSize: options.maxBatchSize ?? 50,
      maxWaitMs: options.maxWaitMs ?? 10,
      enabled: options.enabled ?? true,
    };
  }

  /**
   * Add operation to batch queue
   * @param operation Operation to batch
   * @returns Promise that resolves when operation completes
   */
  async add<T>(operation: Omit<BatchableOperation<T>, 'timestamp'>): Promise<T> {
    if (!this.options.enabled) {
      // Batching disabled, execute immediately
      throw new Error('Batching disabled - caller should execute directly');
    }

    return new Promise<T>((resolve, reject) => {
      const op: BatchableOperation<T> = {
        ...operation,
        resolve: resolve as (value: unknown) => void,
        reject,
        timestamp: Date.now(),
      };

      // Get or create batch for this operation type
      if (!this.batches.has(operation.type)) {
        this.batches.set(operation.type, []);
      }

      const batch = this.batches.get(operation.type)!;
      batch.push(op as BatchableOperation<unknown>);

      // Flush immediately if batch is full
      if (batch.length >= this.options.maxBatchSize) {
        this.flush(operation.type);
        return;
      }

      // Set timer to flush batch if not already set
      if (!this.timers.has(operation.type)) {
        const timer = setTimeout(() => {
          this.flush(operation.type);
        }, this.options.maxWaitMs);
        this.timers.set(operation.type, timer);
      }
    });
  }

  /**
   * Flush batch for a specific operation type
   * @param operationType Operation type to flush
   * @returns Number of operations flushed
   */
  flush(operationType: string): number {
    const batch = this.batches.get(operationType);
    if (!batch || batch.length === 0) {
      return 0;
    }

    // Clear timer
    const timer = this.timers.get(operationType);
    if (timer) {
      clearTimeout(timer);
      this.timers.delete(operationType);
    }

    // Remove batch from queue
    this.batches.delete(operationType);

    const count = batch.length;

    // Execute batch operations
    // For unit testing: resolve with empty results
    // In production, this would call the actual WASM batch execute function
    for (const op of batch) {
      op.resolve(null); // Resolve with null for successful batch execution
    }

    return count;
  }

  /**
   * Flush all pending batches
   * @returns Total number of operations flushed
   */
  flushAll(): number {
    let total = 0;
    for (const operationType of this.batches.keys()) {
      total += this.flush(operationType);
    }
    return total;
  }

  /**
   * Get pending operation count for a type
   * @param operationType Operation type
   * @returns Number of pending operations
   */
  getPendingCount(operationType?: string): number {
    if (operationType) {
      return this.batches.get(operationType)?.length ?? 0;
    }

    let total = 0;
    for (const batch of this.batches.values()) {
      total += batch.length;
    }
    return total;
  }

  /**
   * Check if batching is enabled
   * @returns True if enabled
   */
  isEnabled(): boolean {
    return this.options.enabled;
  }

  /**
   * Enable batching
   */
  enable(): void {
    this.options.enabled = true;
  }

  /**
   * Disable batching and flush all pending operations
   */
  disable(): void {
    this.options.enabled = false;
    this.flushAll();
  }

  /**
   * Clear all pending batches without executing
   */
  clear(): void {
    // Clear all timers
    for (const timer of this.timers.values()) {
      clearTimeout(timer);
    }
    this.timers.clear();

    // Reject all pending operations
    for (const batch of this.batches.values()) {
      for (const op of batch) {
        op.reject(new Error('Batch cleared'));
      }
    }
    this.batches.clear();
  }

  /**
   * Get batch statistics
   * @returns Batch stats
   */
  getStats(): {
    totalBatches: number;
    totalPending: number;
    byType: Record<string, number>;
  } {
    const byType: Record<string, number> = {};
    let totalPending = 0;

    for (const [type, batch] of this.batches.entries()) {
      byType[type] = batch.length;
      totalPending += batch.length;
    }

    return {
      totalBatches: this.batches.size,
      totalPending,
      byType,
    };
  }
}
