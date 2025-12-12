import type { SyndrDBClient, SyndrDBConnection } from './index';
import { SchemaManager } from './schema/SchemaManager';
import { SchemaSerializer } from './schema/SchemaSerializer';
import { MigrationClient } from './migrations/MigrationClient';
import { MigrationHistory } from './migrations/MigrationHistory';
import { MigrationValidator } from './migrations/MigrationValidator';
import { MigrationConflictResolver, ConflictDiff } from './migrations/MigrationConflictResolver';
import { SyndrDBMigrationError } from './migrations/SyndrDBMigrationError';
import { TypeGenerator } from './codegen/TypeGenerator';
import { TypeRegistry } from './codegen/TypeRegistry';
import { TypeWriter } from './codegen/TypeWriter';
import { ResponseMapper } from './codegen/ResponseMapper';
import type {
  BundleDefinition,
  SchemaDefinition,
  SchemaDiff
} from './schema/SchemaDefinition';
import type {
  MigrationInfo,
  MigrationStatus,
  ValidationReport,
  RollbackValidationReport
} from './migrations/MigrationTypes';

/**
 * Options for syncSchema operation
 */
export interface SyncSchemaOptions {
  /** Migration description (max 500 characters) */
  description?: string;
  /** Expected server version for conflict detection */
  expectedVersion?: number;
  /** Whether to automatically apply the migration after creation */
  autoApply?: boolean;
}

/**
 * Options for type generation
 */
export interface GenerateTypesOptions {
  /** Output directory for generated types */
  outputDir?: string;
  /** Whether to include JSDoc comments */
  includeJSDoc?: boolean;
  /** Whether to generate type guards */
  generateTypeGuards?: boolean;
}

/**
 * Migration extension methods for SyndrDBClient
 * These methods provide the developer-facing API for schema migrations
 */
export class SyndrDBClientMigrationExtensions {
  private readonly schemaManager: SchemaManager;
  private readonly schemaSerializer: SchemaSerializer;
  private readonly migrationClient: MigrationClient;
  private readonly migrationHistory: MigrationHistory;
  private readonly migrationValidator: MigrationValidator;
  private readonly conflictResolver: MigrationConflictResolver;
  private readonly typeGenerator: TypeGenerator;
  private readonly typeRegistry: TypeRegistry;
  private readonly typeWriter: TypeWriter;
  private readonly responseMapper: ResponseMapper;
  
  private autoTypeMappingEnabled: boolean = false;
  private localSchema: BundleDefinition[] = [];

  constructor() {
    this.schemaManager = new SchemaManager();
    this.schemaSerializer = new SchemaSerializer();
    this.migrationClient = new MigrationClient();
    this.migrationHistory = new MigrationHistory();
    this.migrationValidator = new MigrationValidator();
    this.conflictResolver = new MigrationConflictResolver();
    this.typeGenerator = new TypeGenerator();
    this.typeRegistry = new TypeRegistry();
    this.typeWriter = new TypeWriter();
    this.responseMapper = new ResponseMapper(this.typeRegistry);
  }

  /**
   * Syncs local schema to server, creating migration if needed
   * @param client - SyndrDB client instance
   * @param schema - Local bundle definitions
   * @param options - Sync options
   * @returns Migration info if created, or null if no changes
   */
  async syncSchema(
    client: SyndrDBClient,
    schema: BundleDefinition[],
    options: SyncSchemaOptions = {}
  ): Promise<MigrationInfo | null> {
    // Store local schema
    this.localSchema = schema;

    // Get connection
    const connection = await this.getConnection(client);

    try {
      // Fetch server schema
      const databaseName = this.getDatabaseName(client);
      const serverResponse = await this.migrationClient.showBundles(connection, databaseName);
      const serverSchema = this.schemaManager.parseServerSchema(serverResponse);

      // Compare schemas
      const localSchema: SchemaDefinition = { bundles: schema };
      const diff = this.schemaManager.compareSchemas(localSchema, serverSchema);
//console.log('DEBUG DEBUG DEBUG Diffs', JSON.stringify(diff, null, 2));

      // No changes needed
      if (!diff.hasChanges) {
        return null;
      }

      // Check for version conflicts if expectedVersion provided
      if (options.expectedVersion !== undefined) {
        const historyResponse = await this.migrationClient.showMigrations(connection, databaseName);
        const history = this.migrationHistory.parse(historyResponse);
        
        const conflict = this.conflictResolver.detectConflict(
          history.currentVersion,
          options.expectedVersion,
          diff
        );

        if (conflict) {
          throw SyndrDBMigrationError.versionConflict(
            conflict.serverVersion,
            conflict.localExpectedVersion,
            conflict.schemaDifferences
          );
        }
      }

      // Generate migration description
      let description = options.description || 'Schema sync';
      
      // Warn if description exceeds 500 characters
      if (description.length > 500) {
        // TODO: Consider logging this warning
        console.warn(
          `Migration description exceeds 500 characters (${description.length}). ` +
          `Server may truncate it.`
        );
      }

      // Serialize diff to commands
      const commands = this.schemaSerializer.serializeDiff(diff);
//console.log('DEBUG DEBUG DEBUG SERIALIZED COMMANDS', JSON.stringify(commands, null, 2));
      // Create migration
      const migrationResponse = await this.migrationClient.startMigration(
        connection,
        description,
        commands
      );

      // Handle empty or undefined response - migration might have been created but response not returned
      if (!migrationResponse) {
        const historyResponse = await this.migrationClient.showMigrations(connection, this.getDatabaseName(client));
        const history = this.migrationHistory.parse(historyResponse);
        
        if (history.migrations.length > 0) {
          // Return the most recent migration
          const latestMigration = history.migrations[history.migrations.length - 1];
        //  console.log('[syncSchema] Found latest migration from history:', JSON.stringify(latestMigration, null, 2));
          return latestMigration;
        }
        
        throw new Error('Failed to create migration: no response from server');
      }

      // Parse migration info from response
      const migrationInfo: MigrationInfo = {
        version: migrationResponse.version || migrationResponse.Version || 1,
        status: migrationResponse.migration_status, //'PENDING',
        description: '',// migrationResponse.data.description,
        createdAt: migrationResponse.created_at || '', //new Date().toISOString(),
        appliedBy: migrationResponse.applied_by || '',
       // commandCount: commands.length,
        //createdAt: 'migration-client'
      };

      // Auto-apply if requested
      if (options.autoApply) {
        await this.applyMigration(client, migrationInfo.version);
        migrationInfo.status = 'APPLIED';
        migrationInfo.appliedBy = new Date().toISOString();
      }

      return migrationInfo;
    } finally {
      await this.releaseConnection(client, connection);
    }
  }

  /**
   * Shows migration history
   * @param client - SyndrDB client instance
   * @param filter - Optional status filter
   * @returns Migration history with current version and migrations list
   */
  async showMigrations(
    client: SyndrDBClient,
    filter?: MigrationStatus
  ): Promise<{ currentVersion: number; migrations: MigrationInfo[] }> {
    const connection = await this.getConnection(client);

    try {
      const databaseName = this.getDatabaseName(client);
      const response = await this.migrationClient.showMigrations(connection, databaseName, filter);
      return this.migrationHistory.parse(response, filter);
    } finally {
      await this.releaseConnection(client, connection);
    }
  }

  /**
   * Gets current migration version
   * @param client - SyndrDB client instance
   * @returns Current version number
   */
  async getCurrentVersion(client: SyndrDBClient): Promise<number> {
    const history = await this.showMigrations(client);
    return history.currentVersion;
  }

  /**
   * Validates a migration before applying
   * @param client - SyndrDB client instance
   * @param version - Migration version to validate
   * @returns Validation report with formatted output
   */
  async validateMigration(
    client: SyndrDBClient,
    version: number
  ): Promise<{ report: ValidationReport; formatted: string; hasErrors: boolean; hasWarnings: boolean }> {
    const connection = await this.getConnection(client);

    try {
      const report = await this.migrationClient.validateMigration(connection, version);
      
      return {
        report,
        formatted: this.migrationValidator.getFormattedReport(report),
        hasErrors: this.migrationValidator.hasErrors(report),
        hasWarnings: this.migrationValidator.hasWarnings(report)
      };
    } finally {
      await this.releaseConnection(client, connection);
    }
  }

  /**
   * Validates a rollback before applying
   * @param client - SyndrDB client instance
   * @param targetVersion - Target version to rollback to
   * @returns Rollback validation report
   */
  async validateRollback(
    client: SyndrDBClient,
    targetVersion: number
  ): Promise<RollbackValidationReport> {
    const connection = await this.getConnection(client);

    try {
      return await this.migrationClient.validateRollback(connection, targetVersion);
    } finally {
      await this.releaseConnection(client, connection);
    }
  }

  /**
   * Applies a pending migration
   * @param client - SyndrDB client instance
   * @param version - Migration version to apply
   * @param force - Whether to force apply despite warnings
   * @returns Application result
   */
  async applyMigration(
    client: SyndrDBClient,
    version: number,
    force: boolean = false
  ): Promise<{ version: number; message: string; force: boolean }> {
    const connection = await this.getConnection(client);

    try {
      // Validate first unless forcing
      if (!force) {
        const validation = await this.validateMigration(client, version);
        
        // if (validation.hasErrors) {
        //   const errors = [
        //     // ...validation.report.syntaxPhase.errors,
        //     // ...validation.report.semanticPhase.errors,
        //     // ...validation.report.constraintPhase.errors
        //   ];
        //   throw SyndrDBMigrationError.validationFailed(version, errors);
        // }
      }

      const result = await this.migrationClient.applyMigration(connection, version, force);
      return {
        version: result.version,
        message: result.message,
        force: force
      };
    } finally {
      await this.releaseConnection(client, connection);
    }
  }

  /**
   * Applies a rollback to target version
   * @param client - SyndrDB client instance
   * @param targetVersion - Target version to rollback to
   * @returns Rollback result
   */
  async applyRollback(
    client: SyndrDBClient,
    targetVersion: number
  ): Promise<{ version: number; message: string; rolledBackVersions?: number[] }> {
    const connection = await this.getConnection(client);

    try {
      // Validate rollback first
      const validation = await this.validateRollback(client, targetVersion);
      
      if (!validation.canRollback) {
        throw SyndrDBMigrationError.rollbackFailed(
          targetVersion,
          validation.errors?.join(', ') || 'Rollback validation failed'
        );
      }

      return await this.migrationClient.applyRollback(connection, targetVersion);
    } finally {
      await this.releaseConnection(client, connection);
    }
  }

  /**
   * Generates TypeScript types from current schema
   * @param client - SyndrDB client instance
   * @param options - Generation options
   * @returns Array of generated filenames
   */
  async generateTypes(
    client: SyndrDBClient,
    options: GenerateTypesOptions = {}
  ): Promise<string[]> {
    const connection = await this.getConnection(client);

    try {
      // Fetch current server schema
      const databaseName = this.getDatabaseName(client);
      const serverResponse = await this.migrationClient.showBundles(connection, databaseName);
      const serverSchema = this.schemaManager.parseServerSchema(serverResponse);
      const bundles = serverSchema.bundles;

      // Configure type generator
      const generator = new TypeGenerator({
        includeJSDoc: options.includeJSDoc ?? true
      });

      // Configure type writer
      const writer = new TypeWriter({
        outputDir: options.outputDir
      });

      // Generate and cache types
      const typeMap = new Map<string, string>();
      
      for (const bundle of bundles) {
        let code = generator.generateInterface(bundle);
        
        if (options.generateTypeGuards) {
          code += '\n\n' + generator.generateTypeGuard(bundle);
        }

        typeMap.set(bundle.name, code);
        this.typeRegistry.register(bundle.name, bundle, code);
      }

      // Write type files
      await writer.writeMultipleTypes(typeMap);
      await writer.writeIndexFile(bundles.map(b => b.name));

      return await writer.listGeneratedFiles();
    } finally {
      await this.releaseConnection(client, connection);
    }
  }

  /**
   * Enables automatic type mapping for query responses
   * @param enabled - Whether to enable auto-mapping
   */
  enableAutoTypeMapping(enabled: boolean = true): void {
    this.autoTypeMappingEnabled = enabled;
  }

  /**
   * Maps a query response to typed objects
   * @param bundleName - Bundle name
   * @param response - Server response
   * @returns Mapped response with typed results
   */
  mapQueryResponse<T = any>(
    bundleName: string,
    response: { Result?: any[]; ResultCount?: number; ExecutionTimeMS?: number }
  ): { results: T[]; count: number; executionTime: number } {
    if (!this.autoTypeMappingEnabled) {
      return {
        results: (response.Result || []) as T[],
        count: response.ResultCount ?? response.Result?.length ?? 0,
        executionTime: response.ExecutionTimeMS ?? 0
      };
    }

    return this.responseMapper.mapQueryResponse<T>(bundleName, response);
  }

  /**
   * Gets connection from client
   * TODO: This is a helper that accesses private client internals - consider making this part of the public API
   */
  private async getConnection(client: any): Promise<SyndrDBConnection> {
    // Access private connection through any cast
    // This is a workaround until we can properly extend SyndrDBClient
    if (!client.connection) {
      throw new Error('Not connected to SyndrDB server. Call connect() first.');
    }
    
    return client.connection;
  }

  /**
   * Releases connection back to pool
   */
  private async releaseConnection(client: any, connection: SyndrDBConnection): Promise<void> {
    if (client.pool) {
      await client.pool.release(connection);
    }
  }

  /**
   * Gets database name from client params
   * TODO: This accesses private client internals - consider making this part of the public API
   */
  private getDatabaseName(client: any): string {
    if (!client.params || !client.params.database) {
      throw new Error('Database name not available. Ensure connect() was called.');
    }
    return client.params.database;
  }
}
