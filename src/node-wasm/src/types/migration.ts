/**
 * Migration status enumeration
 */
export type MigrationStatus = 'PENDING' | 'APPLIED' | 'FAILED' | 'ROLLED_BACK';

/**
 * Migration definition
 */
export interface Migration {
  /** Unique migration identifier (timestamp-based) */
  id: string;

  /** Migration version number */
  version: number;

  /** Human-readable name */
  name: string;

  /** Description of changes */
  description?: string;

  /** SHA-256 checksum of migration commands */
  checksum: string;

  /** Array of forward migration commands */
  upCommands: string[];

  /** Array of rollback commands (auto-generated) */
  downCommands: string[];

  /** When the migration was applied */
  appliedAt?: Date;

  /** Current migration status */
  status: MigrationStatus;

  /** Error message if migration failed */
  error?: string;

  /** Migration metadata */
  metadata?: Record<string, unknown>;
}

/**
 * Migration history response
 */
export interface MigrationHistory {
  /** Current database version */
  currentVersion: number;

  /** List of all migrations */
  migrations: Migration[];

  /** Total number of applied migrations */
  appliedCount: number;

  /** Total number of pending migrations */
  pendingCount: number;

  /** Total number of failed migrations */
  failedCount: number;
}

/**
 * Migration plan for schema changes
 */
export interface MigrationPlan {
  /** Planned migration */
  migration: Migration;

  /** Schema differences detected */
  diffs: SchemaDiff[];

  /** Estimated execution time (ms) */
  estimatedTimeMs: number;

  /** Whether changes are breaking */
  hasBreakingChanges: boolean;

  /** Warnings about the migration */
  warnings: string[];
}

/**
 * Migration validation report
 */
export interface ValidationReport {
  /** Whether validation passed */
  isValid: boolean;

  /** List of errors found */
  errors: ValidationError[];

  /** List of warnings */
  warnings: string[];

  /** Formatted report for display */
  formatted: string;

  /** Whether breaking changes detected */
  hasBreakingChanges: boolean;

  /** Whether warnings exist */
  hasWarnings: boolean;
}

/**
 * Validation error details
 */
export interface ValidationError {
  /** Error severity */
  severity: 'ERROR' | 'WARNING';

  /** Error message */
  message: string;

  /** Affected command */
  command?: string;

  /** Command index in migration */
  commandIndex?: number;

  /** Error type */
  type: string;
}

/**
 * Rollback validation report
 */
export interface RollbackValidationReport {
  /** Whether rollback is safe */
  isSafe: boolean;

  /** Potential risks */
  risks: string[];

  /** Affected data count */
  affectedRecords?: number;

  /** Formatted report */
  formatted: string;
}

/**
 * Schema difference type
 */
export type DiffType =
  | 'BUNDLE_ADDED'
  | 'BUNDLE_REMOVED'
  | 'FIELD_ADDED'
  | 'FIELD_REMOVED'
  | 'FIELD_MODIFIED'
  | 'INDEX_ADDED'
  | 'INDEX_REMOVED';

/**
 * Schema difference entry
 */
export interface SchemaDiff {
  /** Type of difference */
  type: DiffType;

  /** Affected bundle name */
  bundleName: string;

  /** Affected field name (if applicable) */
  fieldName?: string;

  /** Old value */
  oldValue?: unknown;

  /** New value */
  newValue?: unknown;

  /** Whether this is a breaking change */
  isBreaking: boolean;

  /** Human-readable description */
  description: string;
}
