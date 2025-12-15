/**
 * Hook manager for registering and managing operation hooks
 * Provides built-in hooks for logging, metrics, and tracing
 */

import type { SyndrDBClient } from '../client';
import type {
  Hook,
  HookConfig,
  HookContext,
  MetricsStats,
  LoggingHookOptions,
  RetryHookOptions,
} from '../types/hooks';
import { HookError } from '../errors';

/**
 * Hook manager for managing operation hooks
 */
export class HookManager {
  private client: SyndrDBClient;
  private registeredHooks: Map<string, Hook> = new Map();

  constructor(client: SyndrDBClient) {
    this.client = client;
  }

  /**
   * Register a hook
   * @param hook Hook configuration
   */
  async register(hook: HookConfig | Hook): Promise<void> {
    try {
      if (this.registeredHooks.has(hook.name)) {
        throw new HookError(`Hook '${hook.name}' is already registered`);
      }

      // Register with WASM
      await this.client.registerHook(hook);

      // Store locally for lifecycle management
      this.registeredHooks.set(hook.name, hook as Hook);
    } catch (error) {
      if (error instanceof HookError) {
        throw error;
      }
      throw new HookError(
        `Failed to register hook '${hook.name}': ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Unregister a hook
   * @param hookName Hook name
   */
  async unregister(hookName: string): Promise<void> {
    try {
      if (!this.registeredHooks.has(hookName)) {
        throw new HookError(`Hook '${hookName}' is not registered`);
      }

      await this.client.unregisterHook(hookName);
      this.registeredHooks.delete(hookName);
    } catch (error) {
      if (error instanceof HookError) {
        throw error;
      }
      throw new HookError(
        `Failed to unregister hook '${hookName}': ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Unregister all hooks
   */
  async unregisterAll(): Promise<void> {
    const hooks = Array.from(this.registeredHooks.keys());
    const results = await Promise.allSettled(
      hooks.map((name) => this.unregister(name))
    );

    // Log failed unregistrations
    results.forEach((result, index) => {
      if (result.status === 'rejected') {
        console.error(`Failed to unregister hook '${hooks[index]}':`, result.reason);
      }
    });
  }

  /**
   * Check if hook is registered
   * @param hookName Hook name
   * @returns True if registered
   */
  isRegistered(hookName: string): boolean {
    return this.registeredHooks.has(hookName);
  }

  /**
   * Get list of registered hook names
   * @returns Array of hook names
   */
  getRegisteredNames(): string[] {
    return Array.from(this.registeredHooks.keys());
  }

  /**
   * Get hook by name
   * @param hookName Hook name
   * @returns Hook or undefined
   */
  get(hookName: string): Hook | undefined {
    return this.registeredHooks.get(hookName);
  }

  /**
   * Create and register logging hook
   * @param options Logging options
   * @returns Hook name
   */
  async createLoggingHook(options: LoggingHookOptions = {}): Promise<string> {
    // TODO: Implement when Go exports createLoggingHook
    const hookName = 'logging';
    const hook: HookConfig = {
      name: hookName,
      before: (ctx: HookContext) => {
        if (options.logCommands !== false) {
          console.log(`[${ctx.traceId}] Executing: ${ctx.command}`);
        }
      },
      after: (ctx: HookContext) => {
        if (options.logDurations !== false && ctx.durationMs !== undefined) {
          console.log(`[${ctx.traceId}] Completed in ${ctx.durationMs.toFixed(2)}ms`);
        }
        if (options.logResults !== false && ctx.result !== undefined) {
          console.log(`[${ctx.traceId}] Result:`, ctx.result);
        }
        if (ctx.error) {
          console.error(`[${ctx.traceId}] Error:`, ctx.error);
        }
      },
    };

    await this.register(hook);
    return hookName;
  }

  /**
   * Create and register metrics hook
   * @returns Hook name
   */
  async createMetricsHook(): Promise<string> {
    // TODO: Implement when Go exports createMetricsHook
    const hookName = 'metrics';
    const hook: HookConfig = {
      name: hookName,
      before: (_ctx: HookContext) => {
        // Metrics collected in Go layer
      },
      after: (_ctx: HookContext) => {
        // Metrics collected in Go layer
      },
    };

    await this.register(hook);
    return hookName;
  }

  /**
   * Create and register tracing hook
   * @returns Hook name
   */
  async createTracingHook(): Promise<string> {
    // TODO: Implement when Go exports createTracingHook
    const hookName = 'tracing';
    const hook: HookConfig = {
      name: hookName,
      before: (_ctx: HookContext) => {
        // Tracing spans created in Go layer
      },
      after: (_ctx: HookContext) => {
        // Tracing spans finalized in Go layer
      },
    };

    await this.register(hook);
    return hookName;
  }

  /**
   * Create and register retry hook
   * @param options Retry options
   * @returns Hook name
   */
  async createRetryHook(options: RetryHookOptions = {}): Promise<string> {
    // maxAttempts, maxDelayMs, backoffMultiplier reserved for future use
    const initialDelayMs = options.initialDelayMs ?? 100;
    const retryableErrors = options.retryableErrors ?? ['TIMEOUT_ERROR', 'CONNECTION_ERROR'];

    const hookName = 'retry';
    const hook: HookConfig = {
      name: hookName,
      before: async (_ctx: HookContext) => {
        // Retry logic implemented in after hook
      },
      after: async (ctx: HookContext) => {
        if (ctx.error && ctx.metadata.retryCount === undefined) {
          const errorType = (ctx.error as any).type || 'UNKNOWN_ERROR';
          if (retryableErrors.includes(errorType)) {
            ctx.metadata.retryCount = 1;
            ctx.metadata.nextRetryDelayMs = initialDelayMs;
            // TODO: Trigger retry mechanism
          }
        }
      },
    };

    await this.register(hook);
    return hookName;
  }

  /**
   * Get metrics statistics
   * @returns Metrics stats
   */
  async getMetricsStats(): Promise<MetricsStats> {
    try {
      return await this.client.getMetricsStats();
    } catch (error) {
      throw new HookError(
        `Failed to get metrics stats: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Reset metrics
   */
  async resetMetrics(): Promise<void> {
    // TODO: Implement when Go exports resetMetrics
    // For now, no-op
  }
}
