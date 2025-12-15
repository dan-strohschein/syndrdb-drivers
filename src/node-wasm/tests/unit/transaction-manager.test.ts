/**
 * Unit tests for TransactionManager and Transaction
 */

import { TransactionManager, Transaction } from '../../src/features/transaction-manager';
import { TransactionError } from '../../src/errors';
import type { SyndrDBClient } from '../../src/client';

describe('Transaction', () => {
  let mockClient: jest.Mocked<SyndrDBClient>;
  let transaction: Transaction;

  beforeEach(() => {
    mockClient = {
      query: jest.fn().mockResolvedValue({ data: [] }),
      mutate: jest.fn().mockResolvedValue({ id: 1 }),
      commitTransaction: jest.fn().mockResolvedValue(undefined),
      rollbackTransaction: jest.fn().mockResolvedValue(undefined),
    } as any;

    transaction = new Transaction(mockClient, 1, {
      timeoutMs: 1000,
      autoRollback: true,
    });
  });

  describe('getId', () => {
    it('should return transaction ID', () => {
      expect(transaction.getId()).toBe(1);
    });
  });

  describe('getState', () => {
    it('should start in ACTIVE state', () => {
      expect(transaction.getState()).toBe('ACTIVE');
    });
  });

  describe('isActive', () => {
    it('should return true when active', () => {
      expect(transaction.isActive()).toBe(true);
    });

    it('should return false after commit', async () => {
      await transaction.commit();
      expect(transaction.isActive()).toBe(false);
    });
  });

  describe('query', () => {
    it('should execute query', async () => {
      const result = await transaction.query('SELECT 1');

      expect(mockClient.query).toHaveBeenCalledWith('SELECT 1', []);
      expect(result).toEqual({ data: [] });
    });

    it('should throw when not active', async () => {
      await transaction.commit();

      await expect(transaction.query('SELECT 1')).rejects.toThrow(TransactionError);
    });

    it('should auto-rollback on error', async () => {
      mockClient.query.mockRejectedValueOnce(new Error('Query failed'));

      await expect(transaction.query('INVALID')).rejects.toThrow();
      expect(mockClient.rollbackTransaction).toHaveBeenCalled();
    });
  });

  describe('mutate', () => {
    it('should execute mutation', async () => {
      const result = await transaction.mutate('ADD DOCUMENT TO BUNDLE "users" WITH ({"name" = ?})', ['Alice']);

      expect(mockClient.mutate).toHaveBeenCalledWith(
        'ADD DOCUMENT TO BUNDLE "users" WITH ({"name" = ?})',
        ['Alice']
      );
      expect(result).toEqual({ id: 1 });
    });

    it('should auto-rollback on error', async () => {
      mockClient.mutate.mockRejectedValueOnce(new Error('Mutation failed'));

      await expect(transaction.mutate('INVALID')).rejects.toThrow();
      expect(mockClient.rollbackTransaction).toHaveBeenCalled();
    });
  });

  describe('commit', () => {
    it('should commit transaction', async () => {
      await transaction.commit();

      expect(mockClient.commitTransaction).toHaveBeenCalledWith(1);
      expect(transaction.getState()).toBe('COMMITTED');
    });

    it('should throw when not active', async () => {
      await transaction.commit();

      await expect(transaction.commit()).rejects.toThrow(TransactionError);
    });

    it('should set FAILED state on error', async () => {
      mockClient.commitTransaction.mockRejectedValueOnce(new Error('Commit failed'));

      await expect(transaction.commit()).rejects.toThrow(TransactionError);
      expect(transaction.getState()).toBe('FAILED');
    });
  });

  describe('rollback', () => {
    it('should rollback transaction', async () => {
      await transaction.rollback();

      expect(mockClient.rollbackTransaction).toHaveBeenCalledWith(1);
      expect(transaction.getState()).toBe('ROLLED_BACK');
    });

    it('should be idempotent', async () => {
      await transaction.rollback();
      await transaction.rollback(); // Second call should not throw

      expect(mockClient.rollbackTransaction).toHaveBeenCalledTimes(1);
    });
  });

  describe('getDuration', () => {
    it('should return transaction duration', async () => {
      await new Promise((resolve) => setTimeout(resolve, 10));

      const duration = transaction.getDuration();
      expect(duration).toBeGreaterThanOrEqual(10);
    });
  });

  describe('timeout', () => {
    it('should rollback on timeout', async () => {
      const shortTimeoutTx = new Transaction(mockClient, 2, {
        timeoutMs: 50,
        autoRollback: true,
      });

      await new Promise((resolve) => setTimeout(resolve, 100));

      expect(mockClient.rollbackTransaction).toHaveBeenCalledWith(2);
    });
  });
});

describe('TransactionManager', () => {
  let mockClient: jest.Mocked<SyndrDBClient>;
  let manager: TransactionManager;

  beforeEach(() => {
    mockClient = {
      beginTransaction: jest.fn().mockResolvedValue(1),
      commitTransaction: jest.fn().mockResolvedValue(undefined),
      rollbackTransaction: jest.fn().mockResolvedValue(undefined),
    } as any;

    manager = new TransactionManager(mockClient);
  });

  describe('begin', () => {
    it('should begin transaction', async () => {
      const tx = await manager.begin();

      expect(mockClient.beginTransaction).toHaveBeenCalled();
      expect(tx.getId()).toBe(1);
      expect(tx.isActive()).toBe(true);
    });

    it('should throw on error', async () => {
      mockClient.beginTransaction.mockRejectedValueOnce(new Error('Begin failed'));

      await expect(manager.begin()).rejects.toThrow(TransactionError);
    });

    it('should track active transactions', async () => {
      await manager.begin();

      expect(manager.getActiveCount()).toBe(1);
    });
  });

  describe('execute', () => {
    it('should execute callback in transaction', async () => {
      const callback = jest.fn().mockResolvedValue('result');

      const result = await manager.execute(callback);

      expect(callback).toHaveBeenCalled();
      expect(mockClient.commitTransaction).toHaveBeenCalled();
      expect(result).toBe('result');
    });

    it('should rollback on callback error', async () => {
      const callback = jest.fn().mockRejectedValue(new Error('Callback failed'));

      await expect(manager.execute(callback)).rejects.toThrow('Callback failed');
      expect(mockClient.rollbackTransaction).toHaveBeenCalled();
    });

    it('should remove from active transactions after completion', async () => {
      const callback = jest.fn().mockResolvedValue('result');

      await manager.execute(callback);

      expect(manager.getActiveCount()).toBe(0);
    });

    it('should remove from active transactions after error', async () => {
      const callback = jest.fn().mockRejectedValue(new Error('Error'));

      await expect(manager.execute(callback)).rejects.toThrow();
      expect(manager.getActiveCount()).toBe(0);
    });
  });

  describe('getActiveCount', () => {
    it('should return zero initially', () => {
      expect(manager.getActiveCount()).toBe(0);
    });

    it('should increment on begin', async () => {
      await manager.begin();
      await manager.begin();

      expect(manager.getActiveCount()).toBe(2);
    });
  });

  describe('rollbackAll', () => {
    it('should rollback all active transactions', async () => {
      await manager.begin();
      await manager.begin();

      await manager.rollbackAll();

      expect(mockClient.rollbackTransaction).toHaveBeenCalledTimes(2);
      expect(manager.getActiveCount()).toBe(0);
    });

    it('should handle rollback failures gracefully', async () => {
      mockClient.rollbackTransaction.mockRejectedValueOnce(new Error('Rollback failed'));

      await manager.begin();
      await manager.begin();

      await expect(manager.rollbackAll()).resolves.not.toThrow();
      expect(manager.getActiveCount()).toBe(0);
    });
  });
});
