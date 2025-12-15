/**
 * Statement manager for prepared statement lifecycle
 * Provides automatic caching and cleanup
 */

import type { SyndrDBClient } from '../client';
import { StatementError } from '../errors';

/**
 * Statement cache options
 */
export interface StatementCacheOptions {
  /** Maximum number of cached statements (default: 100) */
  maxSize?: number;

  /** Enable automatic cleanup of unused statements (default: true) */
  autoCleanup?: boolean;

  /** Time in ms before unused statements are cleaned up (default: 300000) */
  cleanupIntervalMs?: number;
}

/**
 * Prepared statement wrapper
 */
export class PreparedStatement {
  private client: SyndrDBClient;
  private stmtId: number;
  private query: string;
  private lastUsed: Date;
  private executionCount = 0;

  constructor(client: SyndrDBClient, stmtId: number, query: string) {
    this.client = client;
    this.stmtId = stmtId;
    this.query = query;
    this.lastUsed = new Date();
  }

  /**
   * Get statement ID
   * @returns Statement ID
   */
  getId(): number {
    return this.stmtId;
  }

  /**
   * Get query string
   * @returns Query
   */
  getQuery(): string {
    return this.query;
  }

  /**
   * Get last used timestamp
   * @returns Last used date
   */
  getLastUsed(): Date {
    return this.lastUsed;
  }

  /**
   * Get execution count
   * @returns Number of executions
   */
  getExecutionCount(): number {
    return this.executionCount;
  }

  /**
   * Execute statement with parameters
   * @param params Parameters
   * @returns Execution result
   */
  async execute<T = unknown>(params: unknown[] = []): Promise<T> {
    try {
      this.lastUsed = new Date();
      this.executionCount++;
      return await this.client.executeStatement<T>(this.stmtId, params);
    } catch (error) {
      throw new StatementError(
        `Failed to execute statement: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Deallocate statement
   */
  async deallocate(): Promise<void> {
    try {
      await this.client.deallocateStatement(this.stmtId);
    } catch (error) {
      throw new StatementError(
        `Failed to deallocate statement: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }
}

/**
 * Statement manager with caching
 */
export class StatementManager {
  private client: SyndrDBClient;
  private cache: Map<string, PreparedStatement> = new Map();
  private options: Required<StatementCacheOptions>;
  private cleanupTimer: NodeJS.Timeout | null = null;

  constructor(client: SyndrDBClient, options: StatementCacheOptions = {}) {
    this.client = client;
    this.options = {
      maxSize: options.maxSize ?? 100,
      autoCleanup: options.autoCleanup ?? true,
      cleanupIntervalMs: options.cleanupIntervalMs ?? 300000,
    };

    if (this.options.autoCleanup) {
      this.startCleanupTimer();
    }
  }

  /**
   * Prepare a statement (with caching)
   * @param query Query string
   * @returns Prepared statement
   */
  async prepare(query: string): Promise<PreparedStatement> {
    // Check cache first
    const cached = this.cache.get(query);
    if (cached) {
      return cached;
    }

    // Evict oldest if cache is full
    if (this.cache.size >= this.options.maxSize) {
      await this.evictOldest();
    }

    try {
      const stmtId = await this.client.prepare(query);
      const stmt = new PreparedStatement(this.client, stmtId, query);
      this.cache.set(query, stmt);
      return stmt;
    } catch (error) {
      throw new StatementError(
        `Failed to prepare statement: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Execute query with automatic statement caching
   * @param query Query string
   * @param params Parameters
   * @returns Execution result
   */
  async execute<T = unknown>(query: string, params: unknown[] = []): Promise<T> {
    const stmt = await this.prepare(query);
    return await stmt.execute<T>(params);
  }

  /**
   * Deallocate a specific statement
   * @param query Query string
   */
  async deallocate(query: string): Promise<void> {
    const stmt = this.cache.get(query);
    if (!stmt) {
      return;
    }

    await stmt.deallocate();
    this.cache.delete(query);
  }

  /**
   * Deallocate all cached statements
   */
  async deallocateAll(): Promise<void> {
    const statements = Array.from(this.cache.values());
    const results = await Promise.allSettled(
      statements.map((stmt) => stmt.deallocate())
    );

    // Log failures
    results.forEach((result) => {
      if (result.status === 'rejected') {
        console.error(`Failed to deallocate statement:`, result.reason);
      }
    });

    this.cache.clear();
  }

  /**
   * Get cache size
   * @returns Number of cached statements
   */
  getCacheSize(): number {
    return this.cache.size;
  }

  /**
   * Get cache statistics
   * @returns Cache stats
   */
  getCacheStats(): {
    size: number;
    maxSize: number;
    statements: Array<{
      query: string;
      id: number;
      executions: number;
      lastUsed: Date;
    }>;
  } {
    const statements = Array.from(this.cache.entries()).map(([query, stmt]) => ({
      query,
      id: stmt.getId(),
      executions: stmt.getExecutionCount(),
      lastUsed: stmt.getLastUsed(),
    }));

    return {
      size: this.cache.size,
      maxSize: this.options.maxSize,
      statements,
    };
  }

  /**
   * Evict oldest unused statement
   */
  private async evictOldest(): Promise<void> {
    let oldest: [string, PreparedStatement] | null = null;

    for (const entry of this.cache.entries()) {
      if (!oldest || entry[1].getLastUsed() < oldest[1].getLastUsed()) {
        oldest = entry;
      }
    }

    if (oldest) {
      await oldest[1].deallocate();
      this.cache.delete(oldest[0]);
    }
  }

  /**
   * Start automatic cleanup timer
   */
  private startCleanupTimer(): void {
    this.cleanupTimer = setInterval(() => {
      this.cleanup();
    }, this.options.cleanupIntervalMs);
  }

  /**
   * Stop automatic cleanup timer
   */
  stopCleanup(): void {
    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer);
      this.cleanupTimer = null;
    }
  }

  /**
   * Cleanup unused statements
   */
  private async cleanup(): Promise<void> {
    const now = Date.now();
    const toRemove: string[] = [];

    // Find statements not used in last cleanup interval
    for (const [query, stmt] of this.cache.entries()) {
      const timeSinceLastUse = now - stmt.getLastUsed().getTime();
      if (timeSinceLastUse > this.options.cleanupIntervalMs) {
        toRemove.push(query);
      }
    }

    // Deallocate unused statements
    for (const query of toRemove) {
      await this.deallocate(query);
    }
  }

  /**
   * Destroy manager and cleanup resources
   */
  async destroy(): Promise<void> {
    this.stopCleanup();
    await this.deallocateAll();
  }
}
