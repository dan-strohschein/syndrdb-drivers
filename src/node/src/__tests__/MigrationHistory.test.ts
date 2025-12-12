import { MigrationHistory } from '../migrations/MigrationHistory';
import type { MigrationInfo } from '../migrations/MigrationTypes';

describe('MigrationHistory', () => {
  let history: MigrationHistory;

  beforeEach(() => {
    history = new MigrationHistory();
  });

  describe('parse', () => {
    it('should parse valid migration history', () => {
      const serverResponse = {
        currentVersion: 3,
        migrations: [
          {
            Version: 1,
            Status: 'APPLIED',
            Description: 'Initial schema',
            CreatedAt: '2024-01-01T00:00:00Z',
            AppliedAt: '2024-01-01T00:01:00Z'
          },
          {
            Version: 2,
            Status: 'APPLIED',
            Description: 'Add indexes',
            CreatedAt: '2024-01-02T00:00:00Z',
            AppliedAt: '2024-01-02T00:01:00Z'
          },
          {
            Version: 3,
            Status: 'PENDING',
            Description: 'Add user fields',
            CreatedAt: '2024-01-03T00:00:00Z',
            AppliedAt: null
          }
        ]
      };

      const result = history.parse(serverResponse);

      expect(result.currentVersion).toBe(3);
      expect(result.migrations).toHaveLength(3);
      expect(result.migrations[0].version).toBe(1);
      expect(result.migrations[0].status).toBe('APPLIED');
      expect(result.migrations[1].version).toBe(2);
      expect(result.migrations[2].status).toBe('PENDING');
    });

    it('should handle empty migration list', () => {
      const serverResponse = {
        currentVersion: 0,
        migrations: []
      };

      const result = history.parse(serverResponse);

      expect(result.currentVersion).toBe(0);
      expect(result.migrations).toHaveLength(0);
    });

    it('should filter by status', () => {
      const serverResponse = {
        currentVersion: 3,
        migrations: [
          {
            Version: 1,
            Status: 'APPLIED',
            Description: 'Migration 1',
            CreatedAt: '2024-01-01T00:00:00Z',
            AppliedAt: '2024-01-01T00:01:00Z'
          },
          {
            Version: 2,
            Status: 'PENDING',
            Description: 'Migration 2',
            CreatedAt: '2024-01-02T00:00:00Z',
            AppliedAt: null
          },
          {
            Version: 3,
            Status: 'PENDING',
            Description: 'Migration 3',
            CreatedAt: '2024-01-03T00:00:00Z',
            AppliedAt: null
          }
        ]
      };

      const result = history.parse(serverResponse, 'PENDING');

      expect(result.migrations).toHaveLength(2);
      expect(result.migrations[0].version).toBe(2);
      expect(result.migrations[1].version).toBe(3);
      expect(result.migrations.every(m => m.status === 'PENDING')).toBe(true);
    });

    it('should sort migrations by version ascending', () => {
      const serverResponse = {
        currentVersion: 3,
        migrations: [
          {
            Version: 3,
            Status: 'PENDING',
            Description: 'Migration 3',
            CreatedAt: '2024-01-03T00:00:00Z',
            AppliedAt: null
          },
          {
            Version: 1,
            Status: 'APPLIED',
            Description: 'Migration 1',
            CreatedAt: '2024-01-01T00:00:00Z',
            AppliedAt: '2024-01-01T00:01:00Z'
          },
          {
            Version: 2,
            Status: 'APPLIED',
            Description: 'Migration 2',
            CreatedAt: '2024-01-02T00:00:00Z',
            AppliedAt: '2024-01-02T00:01:00Z'
          }
        ]
      };

      const result = history.parse(serverResponse);

      expect(result.migrations[0].version).toBe(1);
      expect(result.migrations[1].version).toBe(2);
      expect(result.migrations[2].version).toBe(3);
    });
  });

  describe('getNextVersion', () => {
    it('should return 1 for empty history', () => {
      const migrations: MigrationInfo[] = [];

      const nextVersion = history.getNextVersion(migrations);

      expect(nextVersion).toBe(1);
    });

    it('should return highest version + 1', () => {
      const migrations: MigrationInfo[] = [
        {
          version: 1,
          status: 'APPLIED',
          description: 'Migration 1',
          createdAt: '2024-01-01T00:00:00Z',
          appliedBy: '2024-01-01T00:01:00Z'
        },
        {
          version: 2,
          status: 'APPLIED',
          description: 'Migration 2',
          createdAt: '2024-01-02T00:00:00Z',
          appliedBy: '2024-01-02T00:01:00Z'
        }
      ];

      const nextVersion = history.getNextVersion(migrations);

      expect(nextVersion).toBe(3);
    });

    it('should handle non-sequential versions', () => {
      const migrations: MigrationInfo[] = [
        {
          version: 1,
          status: 'APPLIED',
          description: 'Migration 1',
          createdAt: '2024-01-01T00:00:00Z',
          appliedBy: '2024-01-01T00:01:00Z'
        },
        {
          version: 5,
          status: 'APPLIED',
          description: 'Migration 5',
          createdAt: '2024-01-02T00:00:00Z',
          appliedBy: '2024-01-02T00:01:00Z'
        }
      ];

      const nextVersion = history.getNextVersion(migrations);

      expect(nextVersion).toBe(6);
    });
  });

  describe('getPendingMigrations', () => {
    it('should return only pending migrations', () => {
      const migrations: MigrationInfo[] = [
        {
          version: 1,
          status: 'APPLIED',
          description: 'Migration 1',
          createdAt: '2024-01-01T00:00:00Z',
          appliedBy: '2024-01-01T00:01:00Z'
        },
        {
          version: 2,
          status: 'PENDING',
          description: 'Migration 2',
          createdAt: '2024-01-02T00:00:00Z',
          appliedBy: null
        },
        {
          version: 3,
          status: 'PENDING',
          description: 'Migration 3',
          createdAt: '2024-01-03T00:00:00Z',
          appliedBy: null
        }
      ];

      const pending = history.getPendingMigrations(migrations);

      expect(pending).toHaveLength(2);
      expect(pending[0].version).toBe(2);
      expect(pending[1].version).toBe(3);
    });

    it('should return empty array when no pending migrations', () => {
      const migrations: MigrationInfo[] = [
        {
          version: 1,
          status: 'APPLIED',
          description: 'Migration 1',
          createdAt: '2024-01-01T00:00:00Z',
          appliedBy: '2024-01-01T00:01:00Z'
        }
      ];

      const pending = history.getPendingMigrations(migrations);

      expect(pending).toHaveLength(0);
    });
  });

  describe('getAppliedMigrations', () => {
    it('should return only applied migrations', () => {
      const migrations: MigrationInfo[] = [
        {
          version: 1,
          status: 'APPLIED',
          description: 'Migration 1',
          createdAt: '2024-01-01T00:00:00Z',
          appliedBy: '2024-01-01T00:01:00Z'
        },
        {
          version: 2,
          status: 'APPLIED',
          description: 'Migration 2',
          createdAt: '2024-01-02T00:00:00Z',
          appliedBy: '2024-01-02T00:01:00Z'
        },
        {
          version: 3,
          status: 'PENDING',
          description: 'Migration 3',
          createdAt: '2024-01-03T00:00:00Z',
          appliedBy: null
        }
      ];

      const applied = history.getAppliedMigrations(migrations);

      expect(applied).toHaveLength(2);
      expect(applied[0].version).toBe(1);
      expect(applied[1].version).toBe(2);
    });
  });
});
