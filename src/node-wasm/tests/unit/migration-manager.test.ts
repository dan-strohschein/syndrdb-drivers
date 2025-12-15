/**
 * Unit tests for MigrationManager
 */

import { MigrationManager } from '../../src/features/migration-manager';
import { MigrationError } from '../../src/errors';
import type { SyndrDBClient } from '../../src/client';

describe('MigrationManager', () => {
  let mockClient: jest.Mocked<SyndrDBClient>;
  let manager: MigrationManager;

  beforeEach(() => {
    mockClient = {
      planMigration: jest.fn().mockResolvedValue({
        migration: { id: 'mig1', version: 1 },
        diffs: [],
        estimatedTimeMs: 100,
        hasBreakingChanges: false,
        warnings: [],
      }),
      applyMigration: jest.fn().mockResolvedValue({
        id: 'mig1',
        version: 1,
        status: 'APPLIED',
      }),
      rollbackMigration: jest.fn().mockResolvedValue({
        id: 'rollback1',
        version: 0,
        status: 'ROLLED_BACK',
      }),
      validateMigration: jest.fn().mockResolvedValue({
        isValid: true,
        errors: [],
        warnings: [],
        formatted: 'Validation passed',
        hasBreakingChanges: false,
        hasWarnings: false,
      }),
      getMigrationHistory: jest.fn().mockResolvedValue({
        currentVersion: 2,
        migrations: [
          { id: 'mig1', version: 1, status: 'APPLIED' },
          { id: 'mig2', version: 2, status: 'APPLIED' },
          { id: 'mig3', version: 3, status: 'PENDING' },
        ],
        appliedCount: 2,
        pendingCount: 1,
        failedCount: 0,
      }),
    } as any;

    manager = new MigrationManager(mockClient, {
      autoValidate: true,
      requireConfirmation: true,
    });
  });

  describe('plan', () => {
    it('should plan migration from schema', async () => {
      const schema = { bundles: [] };
      const plan = await manager.plan(schema);

      expect(mockClient.planMigration).toHaveBeenCalledWith(schema);
      expect(plan.migration.id).toBe('mig1');
    });

    it('should throw MigrationError on failure', async () => {
      mockClient.planMigration.mockRejectedValueOnce(new Error('Plan failed'));

      await expect(manager.plan({ bundles: [] })).rejects.toThrow(MigrationError);
    });
  });

  describe('apply', () => {
    it('should apply migration with validation', async () => {
      const result = await manager.apply('mig1');

      expect(mockClient.validateMigration).toHaveBeenCalledWith('mig1');
      expect(mockClient.applyMigration).toHaveBeenCalledWith('mig1');
      expect(result.status).toBe('APPLIED');
    });

    it('should skip validation when forced', async () => {
      await manager.apply('mig1', true);

      expect(mockClient.validateMigration).not.toHaveBeenCalled();
      expect(mockClient.applyMigration).toHaveBeenCalledWith('mig1');
    });

    it('should throw on validation failure', async () => {
      mockClient.validateMigration.mockResolvedValueOnce({
        isValid: false,
        errors: [{ severity: 'ERROR', message: 'Invalid', type: 'SYNTAX' }],
        warnings: [],
        formatted: 'Validation failed',
        hasBreakingChanges: false,
        hasWarnings: false,
      });

      await expect(manager.apply('mig1')).rejects.toThrow(MigrationError);
      expect(mockClient.applyMigration).not.toHaveBeenCalled();
    });

    it('should require confirmation for breaking changes', async () => {
      mockClient.validateMigration.mockResolvedValueOnce({
        isValid: true,
        errors: [],
        warnings: [],
        formatted: 'Breaking changes detected',
        hasBreakingChanges: true,
        hasWarnings: false,
      });

      await expect(manager.apply('mig1')).rejects.toThrow(MigrationError);
      await expect(manager.apply('mig1')).rejects.toThrow('breaking changes');
    });

    it('should allow breaking changes when forced', async () => {
      mockClient.validateMigration.mockResolvedValueOnce({
        isValid: true,
        errors: [],
        warnings: [],
        formatted: 'Breaking changes detected',
        hasBreakingChanges: true,
        hasWarnings: false,
      });

      await manager.apply('mig1', true);

      expect(mockClient.applyMigration).toHaveBeenCalled();
    });
  });

  describe('rollback', () => {
    it('should rollback to version', async () => {
      const result = await manager.rollback(0);

      expect(mockClient.rollbackMigration).toHaveBeenCalledWith(0);
      expect(result.status).toBe('ROLLED_BACK');
    });

    it('should throw on negative version', async () => {
      await expect(manager.rollback(-1)).rejects.toThrow(MigrationError);
    });

    it('should skip validation when forced', async () => {
      await manager.rollback(0, true);

      expect(mockClient.rollbackMigration).toHaveBeenCalledWith(0);
    });
  });

  describe('validate', () => {
    it('should validate migration', async () => {
      const report = await manager.validate('mig1');

      expect(mockClient.validateMigration).toHaveBeenCalledWith('mig1');
      expect(report.isValid).toBe(true);
    });

    it('should throw on validation error', async () => {
      mockClient.validateMigration.mockRejectedValueOnce(new Error('Validation failed'));

      await expect(manager.validate('mig1')).rejects.toThrow(MigrationError);
    });
  });

  describe('getHistory', () => {
    it('should get migration history', async () => {
      const history = await manager.getHistory();

      expect(mockClient.getMigrationHistory).toHaveBeenCalled();
      expect(history.currentVersion).toBe(2);
      expect(history.migrations).toHaveLength(3);
    });
  });

  describe('getCurrentVersion', () => {
    it('should return current version', async () => {
      const version = await manager.getCurrentVersion();

      expect(version).toBe(2);
    });
  });

  describe('hasPendingMigrations', () => {
    it('should return true when pending exist', async () => {
      const hasPending = await manager.hasPendingMigrations();

      expect(hasPending).toBe(true);
    });

    it('should return false when no pending', async () => {
      mockClient.getMigrationHistory.mockResolvedValueOnce({
        currentVersion: 2,
        migrations: [],
        appliedCount: 2,
        pendingCount: 0,
        failedCount: 0,
      });

      const hasPending = await manager.hasPendingMigrations();

      expect(hasPending).toBe(false);
    });
  });

  describe('getPendingMigrations', () => {
    it('should return pending migrations', async () => {
      const pending = await manager.getPendingMigrations();

      expect(pending).toHaveLength(1);
      expect(pending[0].status).toBe('PENDING');
    });
  });

  describe('getAppliedMigrations', () => {
    it('should return applied migrations', async () => {
      const applied = await manager.getAppliedMigrations();

      expect(applied).toHaveLength(2);
      expect(applied.every((m) => m.status === 'APPLIED')).toBe(true);
    });
  });

  describe('getFailedMigrations', () => {
    it('should return failed migrations', async () => {
      mockClient.getMigrationHistory.mockResolvedValueOnce({
        currentVersion: 1,
        migrations: [
          { id: 'mig1', version: 1, status: 'APPLIED', name: 'test1', checksum: 'abc', upCommands: [], downCommands: [] } as any,
          { id: 'mig2', version: 2, status: 'FAILED', name: 'test2', checksum: 'def', upCommands: [], downCommands: [] } as any,
        ],
        appliedCount: 1,
        pendingCount: 0,
        failedCount: 1,
      });

      const failed = await manager.getFailedMigrations();

      expect(failed).toHaveLength(1);
      expect(failed[0].status).toBe('FAILED');
    });
  });
});
