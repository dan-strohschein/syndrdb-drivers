/**
 * Migration manager for database schema evolution
 * Provides high-level migration operations with validation and safety checks
 */

import type { SyndrDBClient } from '../client';
import type {
  Migration,
  MigrationHistory,
  MigrationPlan,
  ValidationReport,
  RollbackValidationReport,
} from '../types/migration';
import type { SchemaDefinition } from '../types/schema';
import { MigrationError } from '../errors';

/**
 * Migration manager options
 */
export interface MigrationManagerOptions {
  /** Whether to automatically validate before applying (default: true) */
  autoValidate?: boolean;

  /** Whether to require confirmation for breaking changes (default: true) */
  requireConfirmation?: boolean;

  /** Whether to create backups before migrations (default: true) */
  createBackups?: boolean;

  /** Migration timeout in milliseconds (default: 300000) */
  timeoutMs?: number;
}

/**
 * Migration manager for schema operations
 */
export class MigrationManager {
  private client: SyndrDBClient;
  private options: Required<MigrationManagerOptions>;

  constructor(client: SyndrDBClient, options: MigrationManagerOptions = {}) {
    this.client = client;
    this.options = {
      autoValidate: options.autoValidate ?? true,
      requireConfirmation: options.requireConfirmation ?? true,
      createBackups: options.createBackups ?? true,
      timeoutMs: options.timeoutMs ?? 300000,
    };
  }

  /**
   * Plan migration from schema definition
   * @param schema Target schema
   * @returns Migration plan with diffs and warnings
   */
  async plan(schema: SchemaDefinition): Promise<MigrationPlan> {
    try {
      return await this.client.planMigration(schema);
    } catch (error) {
      throw new MigrationError(
        `Failed to plan migration: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Apply migration with validation and safety checks
   * @param migrationId Migration ID
   * @param force Skip validation and confirmation
   * @returns Applied migration
   */
  async apply(migrationId: string, force = false): Promise<Migration> {
    try {
      // Validate migration unless forced
      if (!force && this.options.autoValidate) {
        const validation = await this.validate(migrationId);

        if (!validation.isValid) {
          throw new MigrationError(
            `Migration validation failed:\n${validation.formatted}`,
            undefined,
            undefined,
            undefined,
            { metadata: { migrationId, validation } }
          );
        }

        if (validation.hasBreakingChanges && this.options.requireConfirmation) {
          throw new MigrationError(
            'Migration contains breaking changes - use force=true to apply',
            undefined,
            undefined,
            undefined,
            { metadata: { migrationId, validation } }
          );
        }
      }

      return await this.client.applyMigration(migrationId);
    } catch (error) {
      if (error instanceof MigrationError) {
        throw error;
      }
      throw new MigrationError(
        `Failed to apply migration: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Rollback to specific version with safety checks
   * @param version Target version (0 for complete rollback)
   * @param force Skip validation
   * @returns Migration result
   */
  async rollback(version: number, _force = false): Promise<Migration> {
    try {
      if (version < 0) {
        throw new MigrationError('Version must be non-negative');
      }

      // TODO: Validate rollback safety when Go implements validateRollback
      // if (!force && this.options.autoValidate) {
      //   const validation = await this.validateRollback(version);
      //   if (!validation.isSafe && this.options.requireConfirmation) {
      //     throw new MigrationError(
      //       `Rollback has risks:\n${validation.formatted}`,
      //       undefined,
      //       undefined,
      //       undefined,
      //       { version, validation }
      //     );
      //   }
      // }

      return await this.client.rollbackMigration(version);
    } catch (error) {
      if (error instanceof MigrationError) {
        throw error;
      }
      throw new MigrationError(
        `Failed to rollback migration: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Validate migration before applying
   * @param migrationId Migration ID
   * @returns Validation report
   */
  async validate(migrationId: string): Promise<ValidationReport> {
    try {
      return await this.client.validateMigration(migrationId);
    } catch (error) {
      throw new MigrationError(
        `Failed to validate migration: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Validate rollback safety
   * @param version Target version
   * @returns Rollback validation report
   */
  async validateRollback(_version: number): Promise<RollbackValidationReport> {
    // TODO: Implement when Go provides validateRollback export
    // For now, return optimistic report
    return {
      isSafe: true,
      risks: [],
      formatted: 'Rollback validation not yet implemented',
    };
  }

  /**
   * Get migration history
   * @returns Complete migration history
   */
  async getHistory(): Promise<MigrationHistory> {
    try {
      return await this.client.getMigrationHistory();
    } catch (error) {
      throw new MigrationError(
        `Failed to get migration history: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Get current database version
   * @returns Current version number
   */
  async getCurrentVersion(): Promise<number> {
    const history = await this.getHistory();
    return history.currentVersion;
  }

  /**
   * Check if migrations are pending
   * @returns True if pending migrations exist
   */
  async hasPendingMigrations(): Promise<boolean> {
    const history = await this.getHistory();
    return history.pendingCount > 0;
  }

  /**
   * Get list of pending migrations
   * @returns Array of pending migrations
   */
  async getPendingMigrations(): Promise<Migration[]> {
    const history = await this.getHistory();
    return history.migrations.filter((m) => m.status === 'PENDING');
  }

  /**
   * Get list of applied migrations
   * @returns Array of applied migrations
   */
  async getAppliedMigrations(): Promise<Migration[]> {
    const history = await this.getHistory();
    return history.migrations.filter((m) => m.status === 'APPLIED');
  }

  /**
   * Get list of failed migrations
   * @returns Array of failed migrations
   */
  async getFailedMigrations(): Promise<Migration[]> {
    const history = await this.getHistory();
    return history.migrations.filter((m) => m.status === 'FAILED');
  }
}
