/**
 * Unit tests for StatementManager
 */

import { StatementManager, PreparedStatement } from '../../src/features/statement-manager';
import { StatementError } from '../../src/errors';
import type { SyndrDBClient } from '../../src/client';

describe('PreparedStatement', () => {
  let mockClient: jest.Mocked<SyndrDBClient>;
  let statement: PreparedStatement;

  beforeEach(() => {
    mockClient = {
      executeStatement: jest.fn().mockResolvedValue({ result: true }),
      deallocateStatement: jest.fn().mockResolvedValue(undefined),
    } as any;

    statement = new PreparedStatement(mockClient, 1, 'SELECT * FROM "users" WHERE "id" == ?');
  });

  describe('getId', () => {
    it('should return statement ID', () => {
      expect(statement.getId()).toBe(1);
    });
  });

  describe('getQuery', () => {
    it('should return query string', () => {
      expect(statement.getQuery()).toBe('SELECT * FROM "users" WHERE "id" == ?');
    });
  });

  describe('getLastUsed', () => {
    it('should return last used date', () => {
      expect(statement.getLastUsed()).toBeInstanceOf(Date);
    });
  });

  describe('getExecutionCount', () => {
    it('should start at zero', () => {
      expect(statement.getExecutionCount()).toBe(0);
    });

    it('should increment on execution', async () => {
      await statement.execute([1]);
      await statement.execute([2]);

      expect(statement.getExecutionCount()).toBe(2);
    });
  });

  describe('execute', () => {
    it('should execute with parameters', async () => {
      const result = await statement.execute([123]);

      expect(mockClient.executeStatement).toHaveBeenCalledWith(1, [123]);
      expect(result).toEqual({ result: true });
    });

    it('should update last used time', async () => {
      const before = statement.getLastUsed();
      await new Promise((resolve) => setTimeout(resolve, 10));
      await statement.execute();

      expect(statement.getLastUsed().getTime()).toBeGreaterThan(before.getTime());
    });

    it('should throw StatementError on failure', async () => {
      mockClient.executeStatement.mockRejectedValueOnce(new Error('Execution failed'));

      await expect(statement.execute([1])).rejects.toThrow(StatementError);
    });
  });

  describe('deallocate', () => {
    it('should deallocate statement', async () => {
      await statement.deallocate();

      expect(mockClient.deallocateStatement).toHaveBeenCalledWith(1);
    });

    it('should throw on error', async () => {
      mockClient.deallocateStatement.mockRejectedValueOnce(new Error('Dealloc failed'));

      await expect(statement.deallocate()).rejects.toThrow(StatementError);
    });
  });
});

describe('StatementManager', () => {
  let mockClient: jest.Mocked<SyndrDBClient>;
  let manager: StatementManager;

  beforeEach(() => {
    mockClient = {
      prepare: jest.fn().mockResolvedValue(1),
      executeStatement: jest.fn().mockResolvedValue({ result: true }),
      deallocateStatement: jest.fn().mockResolvedValue(undefined),
    } as any;

    manager = new StatementManager(mockClient, {
      maxSize: 3,
      autoCleanup: false,
    });
  });

  afterEach(async () => {
    await manager.destroy();
  });

  describe('prepare', () => {
    it('should prepare and cache statement', async () => {
      const stmt = await manager.prepare('SELECT 1');

      expect(mockClient.prepare).toHaveBeenCalledWith('SELECT 1');
      expect(stmt.getQuery()).toBe('SELECT 1');
      expect(manager.getCacheSize()).toBe(1);
    });

    it('should return cached statement', async () => {
      const stmt1 = await manager.prepare('SELECT 1');
      const stmt2 = await manager.prepare('SELECT 1');

      expect(mockClient.prepare).toHaveBeenCalledTimes(1);
      expect(stmt1).toBe(stmt2);
    });

    it('should evict oldest when cache is full', async () => {
      await manager.prepare('SELECT 1');
      await manager.prepare('SELECT 2');
      await manager.prepare('SELECT 3');
      await manager.prepare('SELECT 4'); // Should evict oldest

      expect(manager.getCacheSize()).toBe(3);
      expect(mockClient.deallocateStatement).toHaveBeenCalledTimes(1);
    });

    it('should throw on prepare error', async () => {
      mockClient.prepare.mockRejectedValueOnce(new Error('Prepare failed'));

      await expect(manager.prepare('INVALID')).rejects.toThrow(StatementError);
    });
  });

  describe('execute', () => {
    it('should prepare and execute in one call', async () => {
      const result = await manager.execute('SELECT * FROM "users" WHERE "id" == ?', [123]);

      expect(mockClient.prepare).toHaveBeenCalled();
      expect(mockClient.executeStatement).toHaveBeenCalled();
      expect(result).toEqual({ result: true });
    });

    it('should reuse cached statement', async () => {
      await manager.execute('SELECT 1');
      await manager.execute('SELECT 1');

      expect(mockClient.prepare).toHaveBeenCalledTimes(1);
      expect(mockClient.executeStatement).toHaveBeenCalledTimes(2);
    });
  });

  describe('deallocate', () => {
    it('should deallocate specific statement', async () => {
      await manager.prepare('SELECT 1');
      await manager.deallocate('SELECT 1');

      expect(mockClient.deallocateStatement).toHaveBeenCalled();
      expect(manager.getCacheSize()).toBe(0);
    });

    it('should be idempotent', async () => {
      await manager.deallocate('nonexistent');

      expect(mockClient.deallocateStatement).not.toHaveBeenCalled();
    });
  });

  describe('deallocateAll', () => {
    it('should deallocate all cached statements', async () => {
      await manager.prepare('SELECT 1');
      await manager.prepare('SELECT 2');
      await manager.prepare('SELECT 3');

      await manager.deallocateAll();

      expect(mockClient.deallocateStatement).toHaveBeenCalledTimes(3);
      expect(manager.getCacheSize()).toBe(0);
    });

    it('should handle deallocation failures', async () => {
      await manager.prepare('SELECT 1');
      await manager.prepare('SELECT 2');

      mockClient.deallocateStatement.mockRejectedValueOnce(new Error('Failed'));

      await expect(manager.deallocateAll()).resolves.not.toThrow();
      expect(manager.getCacheSize()).toBe(0);
    });
  });

  describe('getCacheSize', () => {
    it('should return zero initially', () => {
      expect(manager.getCacheSize()).toBe(0);
    });

    it('should reflect cached statements', async () => {
      await manager.prepare('SELECT 1');
      await manager.prepare('SELECT 2');

      expect(manager.getCacheSize()).toBe(2);
    });
  });

  describe('getCacheStats', () => {
    it('should return cache statistics', async () => {
      await manager.prepare('SELECT 1');
      const stmt = await manager.prepare('SELECT 2');
      await stmt.execute();

      const stats = manager.getCacheStats();

      expect(stats.size).toBe(2);
      expect(stats.maxSize).toBe(3);
      expect(stats.statements).toHaveLength(2);
      expect(stats.statements[1].executions).toBe(1);
    });
  });

  describe('automatic cleanup', () => {
    it('should cleanup unused statements', async () => {
      const cleanupManager = new StatementManager(mockClient, {
        autoCleanup: true,
        cleanupIntervalMs: 100,
      });

      await cleanupManager.prepare('SELECT 1');

      // Wait for cleanup interval
      await new Promise((resolve) => setTimeout(resolve, 150));

      expect(mockClient.deallocateStatement).toHaveBeenCalled();

      await cleanupManager.destroy();
    });

    it('should not cleanup recently used statements', async () => {
      const cleanupManager = new StatementManager(mockClient, {
        autoCleanup: true,
        cleanupIntervalMs: 100,
      });

      const stmt = await cleanupManager.prepare('SELECT 1');

      // Keep using the statement
      const interval = setInterval(() => stmt.execute(), 50);

      await new Promise((resolve) => setTimeout(resolve, 150));

      clearInterval(interval);

      // Should not have been cleaned up
      expect(mockClient.deallocateStatement).not.toHaveBeenCalled();

      await cleanupManager.destroy();
    });
  });

  describe('destroy', () => {
    it('should stop cleanup and deallocate all', async () => {
      const cleanupManager = new StatementManager(mockClient, {
        autoCleanup: true,
        cleanupIntervalMs: 100,
      });

      await cleanupManager.prepare('SELECT 1');
      await cleanupManager.destroy();

      expect(mockClient.deallocateStatement).toHaveBeenCalled();
      expect(cleanupManager.getCacheSize()).toBe(0);
    });
  });
});
