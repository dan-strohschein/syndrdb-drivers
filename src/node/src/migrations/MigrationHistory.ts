import type { MigrationInfo, MigrationStatus, MigrationHistoryResponse } from './MigrationTypes';

/**
 * Manages migration history queries and filtering
 * Follows Single Responsibility Principle - only handles migration history operations
 */
export class MigrationHistory {
  /**
   * Parses SHOW MIGRATIONS response and extracts migration history
   * @param response - Raw response from SHOW MIGRATIONS command
   * @param filter - Optional status filter
   * @returns Parsed migration history
   */
  parse(response: any, filter?: MigrationStatus): MigrationHistoryResponse {
    // Handle empty or null response
    if (!response) {
      return {
        currentVersion: 0,
        migrations: [],
        database: ''
      };
    }

    // If response is an array, it's the migrations list directly
    if (Array.isArray(response)) {
      let migrations: MigrationInfo[] = response;
      
      // Apply filter if provided
      if (filter) {
        migrations = migrations.filter((m: MigrationInfo) => m.status === filter);
      }

      // Sort by version ascending
      migrations.sort((a: MigrationInfo, b: MigrationInfo) => a.version - b.version);

      // Determine current version from migrations
      const appliedMigrations = migrations.filter(m => m.status === 'APPLIED');
      const currentVersion = appliedMigrations.length > 0
        ? Math.max(...appliedMigrations.map(m => m.version))
        : 0;

      return {
        currentVersion,
        migrations,
        database: ''
      };
    }

    // Handle object response format
    const currentVersion = response.currentVersion || 0;
    let migrations: MigrationInfo[] = response.migrations || [];

    // Apply filter if provided
    if (filter) {
      migrations = migrations.filter((m: MigrationInfo) => m.status === filter);
    }

    // Sort by version ascending
    migrations.sort((a: MigrationInfo, b: MigrationInfo) => a.version - b.version);

    return {
      currentVersion,
      migrations,
      database: response.database
    };
  }

  /**
   * Gets the current version from migration history
   */
  getCurrentVersion(response: MigrationHistoryResponse): number {
    return response.currentVersion;
  }

  /**
   * Gets all pending migrations
   */
  getPendingMigrations(response: MigrationHistoryResponse): MigrationInfo[] {
    return response.migrations.filter(m => m.status === 'PENDING');
  }

  /**
   * Gets all applied migrations
   */
  getAppliedMigrations(response: MigrationHistoryResponse): MigrationInfo[] {
    return response.migrations.filter(m => m.status === 'APPLIED');
  }

  /**
   * Gets all failed migrations
   */
  getFailedMigrations(response: MigrationHistoryResponse): MigrationInfo[] {
    return response.migrations.filter(m => m.status === 'FAILED');
  }

  /**
   * Gets all rolled back migrations
   */
  getRolledBackMigrations(response: MigrationHistoryResponse): MigrationInfo[] {
    return response.migrations.filter(m => m.status === 'ROLLED_BACK');
  }

  /**
   * Finds a migration by version
   */
  findMigrationByVersion(response: MigrationHistoryResponse, version: number): MigrationInfo | undefined {
    return response.migrations.find(m => m.version === version);
  }

  /**
   * Gets the next version number
   */
  getNextVersion(response: MigrationHistoryResponse): number {
    if (response.migrations.length === 0) {
      return 1;
    }
    const maxVersion = Math.max(...response.migrations.map(m => m.version));
    return maxVersion + 1;
  }
}
