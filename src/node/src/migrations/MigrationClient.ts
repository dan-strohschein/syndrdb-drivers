import type { SyndrDBConnection } from '../index';
import type {
  MigrationInfo,
  MigrationStatus,
  ValidationReport,
  RollbackValidationReport,
  MigrationApplicationResult,
  RollbackApplicationResult
} from './MigrationTypes';

/**
 * Client for executing migration commands
 * Follows Single Responsibility Principle - only handles migration command execution
 */
export class MigrationClient {
  /**
   * Starts a new migration with commands
   * @param connection - Database connection
   * @param description - Migration description
   * @param commands - Array of SyndrDB commands
   * @returns Created migration info
   */
  async startMigration(
    connection: SyndrDBConnection,
    description: string,
    commands: string[]
  ): Promise<any> {
    // Construct START MIGRATION command
    // Each command should already be properly formatted, just add semicolon if missing
    const formattedCommands = commands.map(cmd => {
      const trimmed = cmd.trim();
      return trimmed.endsWith(';') ? trimmed : `${trimmed};`;
    });
    
    const commandBody = formattedCommands.join('\n');
    const fullCommand = `START MIGRATION WITH DESCRIPTION "${description}"\n${commandBody}\nCOMMIT;`;

  //  console.log('[startMigration] Sending command, length:', fullCommand.length);
  //  console.log('[startMigration] First 500 chars:', fullCommand.substring(0, 500));
   console.log('DEBUG DEBUG DEBUG : Full', JSON.stringify(fullCommand, null, 2));
    // Send command
    await connection.sendCommand(fullCommand);
    
    // Receive response
    const response = await connection.receiveResponse<any>();
    
 //   console.log('[startMigration] Response type:', typeof response);
 //   console.log('[startMigration] Response:', JSON.stringify(response, null, 2));
    
    return response;
  }

  /**
   * Shows migrations for a database
   * @param connection - Database connection
   * @param databaseName - Database name
   * @param filter - Optional status filter
   * @returns Migration history response
   */
  async showMigrations(
    connection: SyndrDBConnection,
    databaseName: string,
    filter?: MigrationStatus
  ): Promise<any> {
    // Construct SHOW MIGRATIONS command
    let command = `SHOW MIGRATIONS FOR "${databaseName}";`;
    // if (filter) {
    //   command += ` WHERE Status = "${filter}"`;
    // }

    // Send command
    await connection.sendCommand(command);
    
    // Receive response
    const response = await connection.receiveResponse<any>();
    return response;
  }

  /**
   * Validates a migration
   * @param connection - Database connection
   * @param version - Migration version to validate
   * @returns Validation report
   */
  async validateMigration(
    connection: SyndrDBConnection,
    version: number
  ): Promise<ValidationReport> {
    // Construct VALIDATE MIGRATION command
    const command = `VALIDATE MIGRATION WITH VERSION ${version};`;

    // Send command
    await connection.sendCommand(command);
    
    // Receive response
    const response = await connection.receiveResponse<any>();
    
    // Parse validation_results JSON string if present
    let validationResults: any = {};
    if (response) {
        if (response.report?.validation_results && Array.isArray(response.report.validation_results)) {
            try {
                validationResults = JSON.parse(response.report.validation_results[0]);
            } catch (e) {
                // If parsing fails, use empty object
            }
        }
    } else {
        console.log('Response is not valid!', response)
        return validationResults;
    }
   
// console.log("DEBUG DEBUG DEBUG REsponse :", JSON.stringify(response, null, 2));
    // Parse into ValidationReport
    return {
      migrationVersion: response.report?.migration_version || version,
      validatedAt: response.report?.generated_at || new Date().toISOString(),
      validatedBy: response.report?.generated_by || 'system',
      overallResult: response.report?.status || 'ACTIVE',
      targetVersion: response.report?.target_version || 0,
      // Additional fields from validation_results
      syntaxValid: validationResults.syntaxValid ?? true,
      syntaxErrors: validationResults.syntaxErrors || [],
      dependencyValid: validationResults.dependencyValid ?? true,
      dependencyErrors: validationResults.dependencyErrors || [],
      commandCount: validationResults.commandCount || 0,
      commandCountValid: validationResults.commandCountValid ?? true,
      dataLossWarnings: validationResults.dataLossWarnings || [],
      performanceWarnings: validationResults.performanceWarnings || [],
      affectedBundles: validationResults.affectedBundles || [],
      documentsImpacted: validationResults.documentsImpacted || 0,
      indexesAffected: validationResults.indexesAffected || [],
      estimatedDurationMs: validationResults.estimatedDurationMs || 0
    };
  }

  /**
   * Validates a rollback
   * @param connection - Database connection
   * @param targetVersion - Target version to rollback to
   * @returns Rollback validation report
   */
  async validateRollback(
    connection: SyndrDBConnection,
    targetVersion: number
  ): Promise<RollbackValidationReport> {
    // Construct VALIDATE ROLLBACK command
    const command = `VALIDATE ROLLBACK TO VERSION ${targetVersion};`;

    // Send command
    await connection.sendCommand(command);
    
    // Receive response
    const response = await connection.receiveResponse<any>();
    
    // Parse into RollbackValidationReport
    return {
      canRollback: response.canRollback ?? false,
      currentVersion: response.currentVersion || 0,
      targetVersion: response.targetVersion || targetVersion,
      migrationsToReverse: response.migrationsToReverse || [],
      downCommands: response.downCommands || {},
      dataImpact: response.dataImpact || {
        bundlesDropped: [],
        fieldsRemoved: [],
        estimatedDataLoss: ''
      },
      warnings: response.warnings,
      errors: response.errors
    };
  }

  /**
   * Applies a migration
   * @param connection - Database connection
   * @param version - Migration version to apply
   * @param force - Whether to use FORCE flag
   * @returns Application result
   */
  async applyMigration(
    connection: SyndrDBConnection,
    version: number,
    force: boolean = false
  ): Promise<MigrationApplicationResult> {
    // Construct APPLY MIGRATION command
    const command = `APPLY MIGRATION WITH VERSION ${version}${force ? ' FORCE' : ''};`;

    // Send command
    await connection.sendCommand(command);
    
    // Receive response
    const response = await connection.receiveResponse<any>();
    
    return {
      version: response.version || version,
      message: response.message || `Migration version ${version} applied successfully`,
      force
    };
  }

  /**
   * Applies a rollback
   * @param connection - Database connection
   * @param targetVersion - Target version to rollback to
   * @returns Rollback result
   */
  async applyRollback(
    connection: SyndrDBConnection,
    targetVersion: number
  ): Promise<RollbackApplicationResult> {
    // Construct APPLY ROLLBACK command
    const command = `APPLY ROLLBACK TO VERSION ${targetVersion};`;

    // Send command
    await connection.sendCommand(command);
    
    // Receive response
    const response = await connection.receiveResponse<any>();
    
    return {
      version: response.targetVersion || targetVersion,
      message: response.message || `Database rolled back to version ${targetVersion} successfully`,
      rolledBackVersions: response.rolledBackVersions
    };
  }

  /**
   * Executes SHOW BUNDLES command
   * @param connection - Database connection
   * @param databaseName - Database name
   * @returns Bundles response
   */
  async showBundles(
    connection: SyndrDBConnection,
    databaseName: string
  ): Promise<any> {
    // Construct SHOW BUNDLES command
    const command = `SHOW BUNDLES FOR "${databaseName}";`;

    // Send command
    await connection.sendCommand(command);
    
    // Receive response
    const response = await connection.receiveResponse<any>();
    return response;
  }
}
