/**
 * Transaction manager for ACID operations
 * Provides high-level transaction control with automatic rollback on errors
 */

import type { SyndrDBClient } from '../client';
import { TransactionError } from '../errors';

/**
 * Transaction options
 */
export interface TransactionOptions {
  /** Transaction timeout in milliseconds (default: 300000) */
  timeoutMs?: number;

  /** Whether to auto-rollback on errors (default: true) */
  autoRollback?: boolean;

  /** Isolation level (future use) */
  isolationLevel?: 'READ_UNCOMMITTED' | 'READ_COMMITTED' | 'REPEATABLE_READ' | 'SERIALIZABLE';
}

/**
 * Transaction state
 */
export type TransactionState = 'ACTIVE' | 'COMMITTED' | 'ROLLED_BACK' | 'FAILED';

/**
 * Transaction wrapper with automatic cleanup
 */
export class Transaction {
  private client: SyndrDBClient;
  private txId: string;
  private state: TransactionState = 'ACTIVE';
  private options: Required<Omit<TransactionOptions, 'isolationLevel'>>;
  private startTime: Date;
  private timeoutTimer: NodeJS.Timeout | null = null;

  constructor(client: SyndrDBClient, txId: string, options: TransactionOptions = {}) {
    this.client = client;
    this.txId = txId;
    this.options = {
      timeoutMs: options.timeoutMs ?? 300000,
      autoRollback: options.autoRollback ?? true,
    };
    this.startTime = new Date();

    // Set timeout timer
    this.timeoutTimer = setTimeout(() => {
      this.handleTimeout();
    }, this.options.timeoutMs);
  }

  /**
   * Get transaction ID
   * @returns Transaction ID
   */
  getId(): string {
    return this.txId;
  }

  /**
   * Get transaction state
   * @returns Current state
   */
  getState(): TransactionState {
    return this.state;
  }

  /**
   * Check if transaction is active
   * @returns True if active
   */
  isActive(): boolean {
    return this.state === 'ACTIVE';
  }

  /**
   * Execute query within transaction
   * @param query Query string
   * @param params Query parameters
   * @returns Query result
   */
  async query<T = unknown>(query: string, params: unknown[] = []): Promise<T> {
    this.ensureActive();

    try {
      // TODO: Implement transaction-scoped query when Go exports it
      // For now, use regular query (not transaction-safe)
      return await this.client.query<T>(query, params);
    } catch (error) {
      if (this.options.autoRollback) {
        await this.rollback();
      }
      throw error;
    }
  }

  /**
   * Execute mutation within transaction
   * @param mutation Mutation string
   * @param params Mutation parameters
   * @returns Mutation result
   */
  async mutate<T = unknown>(mutation: string, params: unknown[] = []): Promise<T> {
    this.ensureActive();

    try {
      // TODO: Implement transaction-scoped mutate when Go exports it
      // For now, use regular mutate (not transaction-safe)
      return await this.client.mutate<T>(mutation, params);
    } catch (error) {
      if (this.options.autoRollback) {
        await this.rollback();
      }
      throw error;
    }
  }

  /**
   * Commit transaction
   */
  async commit(): Promise<void> {
    this.ensureActive();

    try {
      await this.client.commitTransaction(this.txId);
      this.state = 'COMMITTED';
      this.clearTimeout();
    } catch (error) {
      this.state = 'FAILED';
      this.clearTimeout();
      throw new TransactionError(
        `Failed to commit transaction: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Rollback transaction
   */
  async rollback(): Promise<void> {
    if (this.state !== 'ACTIVE' && this.state !== 'FAILED') {
      return; // Already finalized
    }

    try {
      await this.client.rollbackTransaction(this.txId);
      this.state = 'ROLLED_BACK';
      this.clearTimeout();
    } catch (error) {
      this.state = 'FAILED';
      this.clearTimeout();
      throw new TransactionError(
        `Failed to rollback transaction: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Get transaction duration in milliseconds
   * @returns Duration
   */
  getDuration(): number {
    return Date.now() - this.startTime.getTime();
  }

  /**
   * Ensure transaction is active
   */
  private ensureActive(): void {
    if (this.state !== 'ACTIVE') {
      throw new TransactionError(`Transaction is not active - state: ${this.state}`);
    }
  }

  /**
   * Handle transaction timeout
   */
  private async handleTimeout(): Promise<void> {
    if (this.state === 'ACTIVE') {
      this.state = 'FAILED';
      try {
        await this.client.rollbackTransaction(this.txId);
      } catch (error) {
        console.error('Failed to rollback timed-out transaction:', error);
      }
    }
  }

  /**
   * Clear timeout timer
   */
  private clearTimeout(): void {
    if (this.timeoutTimer) {
      clearTimeout(this.timeoutTimer);
      this.timeoutTimer = null;
    }
  }
}

/**
 * Transaction manager for creating and managing transactions
 */
export class TransactionManager {
  private client: SyndrDBClient;
  private activeTransactions: Map<string, Transaction> = new Map();

  constructor(client: SyndrDBClient) {
    this.client = client;
  }

  /**
   * Begin a new transaction
   * @param options Transaction options
   * @returns Transaction instance
   */
  async begin(options: TransactionOptions = {}): Promise<Transaction> {
    try {
      const txId = await this.client.beginTransaction();
      const tx = new Transaction(this.client, txId, options);
      this.activeTransactions.set(txId, tx);
      return tx;
    } catch (error) {
      throw new TransactionError(
        `Failed to begin transaction: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Execute callback within transaction with auto-commit/rollback
   * @param callback Transaction callback
   * @param options Transaction options
   * @returns Callback result
   */
  async execute<T>(
    callback: (tx: Transaction) => Promise<T>,
    options: TransactionOptions = {}
  ): Promise<T> {
    const tx = await this.begin(options);

    try {
      const result = await callback(tx);
      await tx.commit();
      this.activeTransactions.delete(tx.getId());
      return result;
    } catch (error) {
      if (tx.isActive()) {
        await tx.rollback();
      }
      this.activeTransactions.delete(tx.getId());
      throw error;
    }
  }

  /**
   * Get active transaction count
   * @returns Number of active transactions
   */
  getActiveCount(): number {
    return this.activeTransactions.size;
  }

  /**
   * Rollback all active transactions
   */
  async rollbackAll(): Promise<void> {
    const transactions = Array.from(this.activeTransactions.values());
    const results = await Promise.allSettled(
      transactions.map((tx) => tx.rollback())
    );

    // Log failed rollbacks
    results.forEach((result, index) => {
      if (result.status === 'rejected') {
        const tx = transactions[index];
        const txId = tx ? tx.getId() : 'unknown';
        console.error(`Failed to rollback transaction ${txId}:`, result.reason);
      }
    });

    this.activeTransactions.clear();
  }
}
