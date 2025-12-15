/**
 * Unit tests for BatchOptimizer
 */

import { BatchOptimizer } from '../../src/utils/batch-optimizer';

describe('BatchOptimizer', () => {
  let optimizer: BatchOptimizer;

  beforeEach(() => {
    optimizer = new BatchOptimizer({
      maxBatchSize: 5,
      maxWaitMs: 50,
      enabled: true,
    });
  });

  afterEach(() => {
    optimizer.clear();
  });

  describe('add', () => {
    it('should throw when batching disabled', async () => {
      optimizer.disable();

      await expect(
        optimizer.add({
          type: 'test',
          params: [],
          resolve: () => {},
          reject: () => {},
        })
      ).rejects.toThrow('Batching disabled');
    });

    it('should queue operations', () => {
      // Don't await - just queue
      optimizer.add({
        type: 'test',
        params: [],
        resolve: () => {},
        reject: () => {},
      });

      expect(optimizer.getPendingCount('test')).toBe(1);
    });

    it('should flush when batch is full', async () => {
      const promises: Promise<unknown>[] = [];

      for (let i = 0; i < 5; i++) {
        promises.push(
          optimizer.add({
            type: 'test',
            params: [i],
            resolve: () => {},
            reject: () => {},
          })
        );
      }

      // Should auto-flush at 5
      // Wait a bit for flush to process
      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(optimizer.getPendingCount('test')).toBe(0);
    });
  });

  describe('flush', () => {
    it('should flush specific operation type', () => {
      optimizer.add({
        type: 'typeA',
        params: [],
        resolve: () => {},
        reject: () => {},
      });
      optimizer.add({
        type: 'typeB',
        params: [],
        resolve: () => {},
        reject: () => {},
      });

      const flushed = optimizer.flush('typeA');

      expect(flushed).toBe(1);
      expect(optimizer.getPendingCount('typeA')).toBe(0);
      expect(optimizer.getPendingCount('typeB')).toBe(1);
    });

    it('should return 0 for empty batch', () => {
      const flushed = optimizer.flush('nonexistent');
      expect(flushed).toBe(0);
    });
  });

  describe('flushAll', () => {
    it('should flush all pending batches', () => {
      optimizer.add({
        type: 'typeA',
        params: [],
        resolve: () => {},
        reject: () => {},
      });
      optimizer.add({
        type: 'typeB',
        params: [],
        resolve: () => {},
        reject: () => {},
      });

      const total = optimizer.flushAll();

      expect(total).toBe(2);
      expect(optimizer.getPendingCount()).toBe(0);
    });
  });

  describe('getPendingCount', () => {
    it('should return count for specific type', () => {
      optimizer.add({
        type: 'test',
        params: [],
        resolve: () => {},
        reject: () => {},
      });

      expect(optimizer.getPendingCount('test')).toBe(1);
    });

    it('should return total count without type', () => {
      optimizer.add({
        type: 'typeA',
        params: [],
        resolve: () => {},
        reject: () => {},
      });
      optimizer.add({
        type: 'typeB',
        params: [],
        resolve: () => {},
        reject: () => {},
      });

      expect(optimizer.getPendingCount()).toBe(2);
    });
  });

  describe('enable/disable', () => {
    it('should enable batching', () => {
      optimizer.disable();
      expect(optimizer.isEnabled()).toBe(false);

      optimizer.enable();
      expect(optimizer.isEnabled()).toBe(true);
    });

    it('should flush on disable', () => {
      optimizer.add({
        type: 'test',
        params: [],
        resolve: () => {},
        reject: () => {},
      });

      optimizer.disable();
      expect(optimizer.getPendingCount()).toBe(0);
    });
  });

  describe('clear', () => {
    it('should clear all batches without executing', async () => {
      const rejectSpy = jest.fn();

      optimizer.add({
        type: 'test',
        params: [],
        resolve: () => {},
        reject: rejectSpy,
      });

      optimizer.clear();

      expect(optimizer.getPendingCount()).toBe(0);
      expect(rejectSpy).toHaveBeenCalledWith(expect.objectContaining({
        message: 'Batch cleared',
      }));
    });
  });

  describe('getStats', () => {
    it('should return batch statistics', () => {
      optimizer.add({
        type: 'typeA',
        params: [],
        resolve: () => {},
        reject: () => {},
      });
      optimizer.add({
        type: 'typeA',
        params: [],
        resolve: () => {},
        reject: () => {},
      });
      optimizer.add({
        type: 'typeB',
        params: [],
        resolve: () => {},
        reject: () => {},
      });

      const stats = optimizer.getStats();

      expect(stats.totalBatches).toBe(2);
      expect(stats.totalPending).toBe(3);
      expect(stats.byType['typeA']).toBe(2);
      expect(stats.byType['typeB']).toBe(1);
    });
  });
});
