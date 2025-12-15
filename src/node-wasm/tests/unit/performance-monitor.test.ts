/**
 * Unit tests for PerformanceMonitor
 */

import { PerformanceMonitor, getGlobalPerformanceMonitor, resetGlobalPerformanceMonitor } from '../../src/utils/performance-monitor';

describe('PerformanceMonitor', () => {
  let monitor: PerformanceMonitor;

  beforeEach(() => {
    monitor = new PerformanceMonitor();
  });

  describe('markStart and markEnd', () => {
    it('should record operation timing', () => {
      const markId = monitor.markStart('testOp');
      expect(markId).toBeTruthy();

      monitor.markEnd(markId, true);

      const stats = monitor.getStats();
      expect(stats.operations['testOp']).toBeDefined();
      expect(stats.operations['testOp'].count).toBe(1);
      expect(stats.operations['testOp'].successCount).toBe(1);
      expect(stats.operations['testOp'].errorCount).toBe(0);
    });

    it('should record failed operations', () => {
      const markId = monitor.markStart('failOp');
      const error = new Error('Test error');
      monitor.markEnd(markId, false, error);

      const stats = monitor.getStats();
      expect(stats.operations['failOp'].errorCount).toBe(1);
      expect(stats.operations['failOp'].successCount).toBe(0);
    });

    it('should handle empty markId gracefully', () => {
      monitor.markEnd('', true);
      const stats = monitor.getStats();
      expect(Object.keys(stats.operations)).toHaveLength(0);
    });

    it('should track multiple operations', () => {
      for (let i = 0; i < 10; i++) {
        const markId = monitor.markStart('multiOp');
        monitor.markEnd(markId, true);
      }

      const stats = monitor.getStats();
      expect(stats.operations['multiOp'].count).toBe(10);
    });

    it('should calculate percentiles correctly', () => {
      // Record operations with known durations
      for (let i = 1; i <= 100; i++) {
        const markId = monitor.markStart('percentileOp');
        // Simulate different durations by recording immediately
        monitor.markEnd(markId, true);
      }

      const stats = monitor.getStats();
      const metrics = stats.operations['percentileOp'];

      expect(metrics.p50Ms).toBeGreaterThanOrEqual(0);
      expect(metrics.p95Ms).toBeGreaterThanOrEqual(metrics.p50Ms);
      expect(metrics.p99Ms).toBeGreaterThanOrEqual(metrics.p95Ms);
    });
  });

  describe('recordBatch', () => {
    it('should track batched operations', () => {
      monitor.recordBatch(5, 4);
      monitor.recordBatch(3, 2);

      const stats = monitor.getStats();
      expect(stats.boundaries.batchedOperations).toBe(8);
      expect(stats.boundaries.crossingsSaved).toBe(6);
    });
  });

  describe('getStats', () => {
    it('should return stats with duration', () => {
      const stats = monitor.getStats();

      expect(stats.startTime).toBeInstanceOf(Date);
      expect(stats.endTime).toBeInstanceOf(Date);
      expect(stats.durationMs).toBeGreaterThanOrEqual(0);
      expect(stats.operations).toBeDefined();
      expect(stats.boundaries).toBeDefined();
      expect(stats.memory).toBeDefined();
    });

    it('should calculate operations per second', () => {
      for (let i = 0; i < 100; i++) {
        const markId = monitor.markStart('opsTest');
        monitor.markEnd(markId, true);
      }

      const stats = monitor.getStats();
      expect(stats.operations['opsTest'].opsPerSecond).toBeGreaterThan(0);
    });
  });

  describe('reset', () => {
    it('should clear all metrics', () => {
      const markId = monitor.markStart('resetTest');
      monitor.markEnd(markId, true);

      let stats = monitor.getStats();
      expect(stats.operations['resetTest']).toBeDefined();

      monitor.reset();

      stats = monitor.getStats();
      expect(stats.operations['resetTest']).toBeUndefined();
    });
  });

  describe('global instance', () => {
    it('should return singleton', () => {
      const instance1 = getGlobalPerformanceMonitor();
      const instance2 = getGlobalPerformanceMonitor();
      expect(instance1).toBe(instance2);
    });

    it('should reset global instance', () => {
      const instance1 = getGlobalPerformanceMonitor();
      resetGlobalPerformanceMonitor();
      const instance2 = getGlobalPerformanceMonitor();
      expect(instance1).not.toBe(instance2);
    });
  });

  describe('sampling', () => {
    it('should respect sample rate', () => {
      const sampledMonitor = new PerformanceMonitor({ sampleRate: 0.0 });

      for (let i = 0; i < 100; i++) {
        const markId = sampledMonitor.markStart('sampleTest');
        sampledMonitor.markEnd(markId, true);
      }

      const stats = sampledMonitor.getStats();
      // With 0% sample rate, should record no operations
      expect(stats.operations['sampleTest']).toBeUndefined();
    });
  });
});
