/**
 * Unit tests for HookManager
 */

import { HookManager } from '../../src/features/hook-manager';
import { HookError } from '../../src/errors';
import type { SyndrDBClient } from '../../src/client';
import type { HookConfig } from '../../src/types/hooks';

describe('HookManager', () => {
  let mockClient: jest.Mocked<SyndrDBClient>;
  let manager: HookManager;

  beforeEach(() => {
    mockClient = {
      registerHook: jest.fn().mockResolvedValue(undefined),
      unregisterHook: jest.fn().mockResolvedValue(undefined),
      getMetricsStats: jest.fn().mockResolvedValue({
        totalOperations: 100,
        successCount: 95,
        errorCount: 5,
        totalDurationMs: 1000,
        avgDurationMs: 10,
        minDurationMs: 1,
        maxDurationMs: 50,
        p95DurationMs: 40,
        p99DurationMs: 48,
        operationsPerSecond: 100,
        errorRate: 5,
        byType: {},
      }),
    } as any;

    manager = new HookManager(mockClient);
  });

  describe('register', () => {
    it('should register hook', async () => {
      const hook: HookConfig = {
        name: 'testHook',
        before: () => {},
        after: () => {},
      };

      await manager.register(hook);

      expect(mockClient.registerHook).toHaveBeenCalledWith(hook);
      expect(manager.isRegistered('testHook')).toBe(true);
    });

    it('should throw when hook already registered', async () => {
      const hook: HookConfig = {
        name: 'testHook',
        before: () => {},
      };

      await manager.register(hook);

      await expect(manager.register(hook)).rejects.toThrow(HookError);
      await expect(manager.register(hook)).rejects.toThrow('already registered');
    });

    it('should throw on registration error', async () => {
      mockClient.registerHook.mockRejectedValueOnce(new Error('Registration failed'));

      const hook: HookConfig = {
        name: 'testHook',
        before: () => {},
      };

      await expect(manager.register(hook)).rejects.toThrow(HookError);
    });
  });

  describe('unregister', () => {
    beforeEach(async () => {
      await manager.register({
        name: 'testHook',
        before: () => {},
      });
    });

    it('should unregister hook', async () => {
      await manager.unregister('testHook');

      expect(mockClient.unregisterHook).toHaveBeenCalledWith('testHook');
      expect(manager.isRegistered('testHook')).toBe(false);
    });

    it('should throw when hook not registered', async () => {
      await expect(manager.unregister('nonexistent')).rejects.toThrow(HookError);
      await expect(manager.unregister('nonexistent')).rejects.toThrow('not registered');
    });

    it('should throw on unregistration error', async () => {
      mockClient.unregisterHook.mockRejectedValueOnce(new Error('Unregister failed'));

      await expect(manager.unregister('testHook')).rejects.toThrow(HookError);
    });
  });

  describe('unregisterAll', () => {
    it('should unregister all hooks', async () => {
      await manager.register({ name: 'hook1', before: () => {} });
      await manager.register({ name: 'hook2', before: () => {} });

      await manager.unregisterAll();

      expect(mockClient.unregisterHook).toHaveBeenCalledTimes(2);
      expect(manager.getRegisteredNames()).toHaveLength(0);
    });

    it('should handle unregistration failures', async () => {
      await manager.register({ name: 'hook1', before: () => {} });
      await manager.register({ name: 'hook2', before: () => {} });

      mockClient.unregisterHook.mockRejectedValueOnce(new Error('Failed'));

      await expect(manager.unregisterAll()).resolves.not.toThrow();
    });
  });

  describe('isRegistered', () => {
    it('should return false for unregistered hook', () => {
      expect(manager.isRegistered('nonexistent')).toBe(false);
    });

    it('should return true for registered hook', async () => {
      await manager.register({ name: 'testHook', before: () => {} });

      expect(manager.isRegistered('testHook')).toBe(true);
    });
  });

  describe('getRegisteredNames', () => {
    it('should return empty array initially', () => {
      expect(manager.getRegisteredNames()).toEqual([]);
    });

    it('should return all registered hook names', async () => {
      await manager.register({ name: 'hook1', before: () => {} });
      await manager.register({ name: 'hook2', before: () => {} });

      const names = manager.getRegisteredNames();
      expect(names).toContain('hook1');
      expect(names).toContain('hook2');
      expect(names).toHaveLength(2);
    });
  });

  describe('get', () => {
    it('should return undefined for unregistered hook', () => {
      expect(manager.get('nonexistent')).toBeUndefined();
    });

    it('should return hook for registered name', async () => {
      const hook: HookConfig = {
        name: 'testHook',
        before: () => {},
      };

      await manager.register(hook);

      const retrieved = manager.get('testHook');
      expect(retrieved).toBeDefined();
      expect(retrieved?.name).toBe('testHook');
    });
  });

  describe('createLoggingHook', () => {
    it('should create and register logging hook', async () => {
      const hookName = await manager.createLoggingHook({
        logCommands: true,
        logDurations: true,
      });

      expect(hookName).toBe('logging');
      expect(manager.isRegistered('logging')).toBe(true);
    });
  });

  describe('createMetricsHook', () => {
    it('should create and register metrics hook', async () => {
      const hookName = await manager.createMetricsHook();

      expect(hookName).toBe('metrics');
      expect(manager.isRegistered('metrics')).toBe(true);
    });
  });

  describe('createTracingHook', () => {
    it('should create and register tracing hook', async () => {
      const hookName = await manager.createTracingHook();

      expect(hookName).toBe('tracing');
      expect(manager.isRegistered('tracing')).toBe(true);
    });
  });

  describe('createRetryHook', () => {
    it('should create and register retry hook', async () => {
      const hookName = await manager.createRetryHook({
        maxAttempts: 3,
        initialDelayMs: 100,
      });

      expect(hookName).toBe('retry');
      expect(manager.isRegistered('retry')).toBe(true);
    });
  });

  describe('getMetricsStats', () => {
    it('should get metrics statistics', async () => {
      const stats = await manager.getMetricsStats();

      expect(mockClient.getMetricsStats).toHaveBeenCalled();
      expect(stats.totalOperations).toBe(100);
      expect(stats.successCount).toBe(95);
      expect(stats.errorRate).toBe(5);
    });

    it('should throw on error', async () => {
      mockClient.getMetricsStats.mockRejectedValueOnce(new Error('Stats failed'));

      await expect(manager.getMetricsStats()).rejects.toThrow(HookError);
    });
  });
});
