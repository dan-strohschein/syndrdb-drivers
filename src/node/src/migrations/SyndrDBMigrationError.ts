/**
 * Custom error class for migration-specific errors
 */
export class SyndrDBMigrationError extends Error {
  public code: string;
  public type: string;
  public details?: any;

  constructor(message: string, code: string, type: string, details?: any) {
    super(message);
    this.name = 'SyndrDBMigrationError';
    this.code = code;
    this.type = type;
    this.details = details;
    Error.captureStackTrace(this, this.constructor);
  }

  /**
   * Creates a version conflict error
   */
  static versionConflict(serverVersion: number, localVersion: number, details?: any): SyndrDBMigrationError {
    return new SyndrDBMigrationError(
      `Version conflict: server is at version ${serverVersion}, local expects version ${localVersion}`,
      'VERSION_CONFLICT',
      'MIGRATION_ERROR',
      details
    );
  }

  /**
   * Creates a validation failed error
   */
  static validationFailed(version: number, errors: string[]): SyndrDBMigrationError {
    return new SyndrDBMigrationError(
      `Migration version ${version} validation failed: ${errors.join(', ')}`,
      'VALIDATION_FAILED',
      'MIGRATION_ERROR',
      { version, errors }
    );
  }

  /**
   * Creates a rollback failed error
   */
  static rollbackFailed(targetVersion: number, reason: string): SyndrDBMigrationError {
    return new SyndrDBMigrationError(
      `Rollback to version ${targetVersion} failed: ${reason}`,
      'ROLLBACK_FAILED',
      'MIGRATION_ERROR',
      { targetVersion, reason }
    );
  }

  /**
   * Creates a migration pending error
   */
  static migrationPending(version: number): SyndrDBMigrationError {
    return new SyndrDBMigrationError(
      `Migration version ${version} is pending and must be applied or validated first`,
      'MIGRATION_PENDING',
      'MIGRATION_ERROR',
      { version }
    );
  }

  /**
   * Creates a description too long error
   */
  static descriptionTooLong(originalLength: number, maxLength: number = 500): SyndrDBMigrationError {
    return new SyndrDBMigrationError(
      `Migration description length ${originalLength} exceeds maximum ${maxLength} characters`,
      'DESCRIPTION_TOO_LONG',
      'MIGRATION_ERROR',
      { originalLength, maxLength }
    );
  }
}
