import { SyndrDBClient } from './index';
import { SyndrDBClientMigrationExtensions } from './SyndrDBClientMigrationExtensions';
import type { BundleDefinition } from './schema/SchemaDefinition';
import type { SyncSchemaOptions, GenerateTypesOptions } from './SyndrDBClientMigrationExtensions';
import type { MigrationStatus } from './migrations/MigrationTypes';

/**
 * Extended SyndrDB client with migration support
 * This provides a convenient wrapper around the migration extensions
 */
export class SyndrDBMigrationClient extends SyndrDBClient {
  private readonly migrations: SyndrDBClientMigrationExtensions;

  constructor() {
    super();
    this.migrations = new SyndrDBClientMigrationExtensions();
  }

  /**
   * Syncs local schema to server, creating migration if needed
   * Automatically detects schema changes and creates migrations
   * 
   * @example
   * ```typescript
   * const schema: BundleDefinition[] = [
   *   {
   *     name: 'User',
   *     fields: [
   *       { name: 'id', type: 'STRING', required: true, unique: true },
   *       { name: 'email', type: 'STRING', required: true, unique: true },
   *       { name: 'name', type: 'STRING', required: true, unique: false }
   *     ],
   *     indexes: [
   *       { fieldName: 'email', type: 'hash' }
   *     ]
   *   }
   * ];
   * 
   * const migration = await client.syncSchema(schema, {
   *   description: 'Add User bundle with email index',
   *   autoApply: true
   * });
   * ```
   */
  async syncSchema(schema: BundleDefinition[], options?: SyncSchemaOptions) {
    return this.migrations.syncSchema(this, schema, options);
  }

  /**
   * Shows migration history for the current database
   * 
   * @example
   * ```typescript
   * const { currentVersion, migrations } = await client.showMigrations();
   * console.log(`Current version: ${currentVersion}`);
   * 
   * for (const migration of migrations) {
   *   console.log(`Version ${migration.version}: ${migration.description} - ${migration.status}`);
   * }
   * ```
   */
  async showMigrations(filter?: MigrationStatus) {
    return this.migrations.showMigrations(this, filter);
  }

  /**
   * Gets current migration version
   * 
   * @example
   * ```typescript
   * const version = await client.getCurrentVersion();
   * console.log(`Database is at version ${version}`);
   * ```
   */
  async getCurrentVersion() {
    return this.migrations.getCurrentVersion(this);
  }

  /**
   * Validates a migration before applying
   * Checks syntax, semantics, constraints, performance, and reversibility
   * 
   * @example
   * ```typescript
   * const { formatted, hasErrors, hasWarnings } = await client.validateMigration(5);
   * 
   * if (hasErrors) {
   *   console.error('Validation failed:');
   *   console.log(formatted);
   * } else if (hasWarnings) {
   *   console.warn('Validation passed with warnings:');
   *   console.log(formatted);
   * } else {
   *   console.log('Validation passed!');
   * }
   * ```
   */
  async validateMigration(version: number) {
    return this.migrations.validateMigration(this, version);
  }

  /**
   * Validates a rollback before applying
   * Shows data impact and potential data loss
   * 
   * @example
   * ```typescript
   * const validation = await client.validateRollback(3);
   * 
   * if (!validation.canRollback) {
   *   console.error('Cannot rollback:', validation.errors);
   * } else {
   *   console.log(`Will rollback from v${validation.currentVersion} to v${validation.targetVersion}`);
   *   console.log('Data impact:', validation.dataImpact);
   * }
   * ```
   */
  async validateRollback(targetVersion: number) {
    return this.migrations.validateRollback(this, targetVersion);
  }

  /**
   * Applies a pending migration
   * Can optionally force apply despite warnings
   * 
   * @example
   * ```typescript
   * // Validate first
   * const { hasErrors, hasWarnings } = await client.validateMigration(5);
   * 
   * if (!hasErrors) {
   *   const result = await client.applyMigration(5, hasWarnings); // Force if warnings
   *   console.log(result.message);
   * }
   * ```
   */
  async applyMigration(version: number, force: boolean = false) {
    return this.migrations.applyMigration(this, version, force);
  }

  /**
   * Applies a rollback to target version
   * Automatically validates before rolling back
   * 
   * @example
   * ```typescript
   * try {
   *   const result = await client.applyRollback(3);
   *   console.log(result.message);
   *   console.log('Rolled back versions:', result.rolledBackVersions);
   * } catch (error) {
   *   console.error('Rollback failed:', error.message);
   * }
   * ```
   */
  async applyRollback(targetVersion: number) {
    return this.migrations.applyRollback(this, targetVersion);
  }

  /**
   * Generates TypeScript type definitions from current server schema
   * Creates .d.ts files in .syndrdb-types/ directory by default
   * 
   * @example
   * ```typescript
   * const files = await client.generateTypes({
   *   outputDir: './types/syndrdb',
   *   includeJSDoc: true,
   *   generateTypeGuards: true
   * });
   * 
   * console.log('Generated types:', files);
   * // Now you can import: import type { User, Product } from './types/syndrdb';
   * ```
   */
  async generateTypes(options?: GenerateTypesOptions) {
    return this.migrations.generateTypes(this, options);
  }

  /**
   * Enables automatic type mapping for query responses
   * When enabled, query results are automatically converted to proper TypeScript types
   * 
   * @example
   * ```typescript
   * // Generate types first
   * await client.generateTypes();
   * 
   * // Enable auto-mapping
   * client.enableAutoTypeMapping(true);
   * 
   * // Query results are now typed and converted
   * const response = await client.query('SELECT * FROM User WHERE email = "test@example.com"');
   * const mapped = client.mapQueryResponse<User>('User', response);
   * 
   * // mapped.results[0].createdAt is a Date object, not a string
   * console.log(mapped.results[0].createdAt instanceof Date); // true
   * ```
   */
  enableAutoTypeMapping(enabled: boolean = true) {
    this.migrations.enableAutoTypeMapping(enabled);
  }

  /**
   * Maps a query response to typed objects
   * Converts field values to appropriate TypeScript types (dates, numbers, etc.)
   * 
   * @example
   * ```typescript
   * const response = await client.query('SELECT * FROM User');
   * const { results, count, executionTime } = client.mapQueryResponse<User>('User', response);
   * 
   * for (const user of results) {
   *   console.log(`${user.name} - ${user.email}`);
   * }
   * ```
   */
  mapQueryResponse<T = any>(
    bundleName: string,
    response: { Result?: any[]; ResultCount?: number; ExecutionTimeMS?: number }
  ) {
    return this.migrations.mapQueryResponse<T>(bundleName, response);
  }
}

/**
 * Creates a new SyndrDB client with migration support
 * This is the recommended way to create a client instance
 * 
 * @example
 * ```typescript
 * import { createMigrationClient } from 'syndrdb-driver';
 * 
 * const client = createMigrationClient();
 * await client.connect('syndrdb://localhost:7777:mydb:admin:password;');
 * 
 * // Use migration features
 * const schema = [...];
 * await client.syncSchema(schema, { autoApply: true });
 * await client.generateTypes();
 * ```
 */
export function createMigrationClient(): SyndrDBMigrationClient {
  return new SyndrDBMigrationClient();
}
