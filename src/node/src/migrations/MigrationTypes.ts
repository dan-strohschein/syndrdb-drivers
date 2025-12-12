/**
 * Migration-related type definitions
 */

/**
 * Migration status
 */
export type MigrationStatus = 'PENDING' | 'APPLIED' | 'FAILED' | 'ROLLED_BACK';

/**
 * Migration information from server
 */
export interface MigrationInfo {
  /** Migration version number */
  version: number;
  /** Current status */
  status: MigrationStatus;
  /** Human-readable description */
  description: string;
  /** When the migration was created */
  createdAt: string;
  /** When the migration was applied (if APPLIED) */
  appliedBy?: string;
  /** Number of commands in the migration */
  //commandCount: number;
  /** User who created the migration */
  //createdBy?: string;
}

/**
 * Validation phase result
 */
export interface ValidationPhase {
  /** Whether this phase passed */
  passed: boolean;
  /** Errors encountered in this phase */
  errors: string[];
  /** Warnings encountered in this phase */
  warnings?: string[];
}

/**
 * Performance validation phase result
 */
export interface PerformanceValidationPhase extends ValidationPhase {
  /** Estimated duration of migration execution */
  estimatedDuration?: string;
  /** Number of commands in migration */
  commandCount?: number;
}

/**
 * Reversibility validation phase result
 */
export interface ReversibilityValidationPhase extends ValidationPhase {
  /** Whether DOWN commands were auto-generated */
  downCommandsGenerated?: boolean;
}

/**
 * Overall validation result
 */
export type ValidationResult = 'PASS' | 'PASS_WITH_WARNINGS' | 'FAIL';

/**
 * Complete validation report from server
 */
export interface ValidationReport {
  /** Version being validated */
  migrationVersion: number;
  /** When validation was performed */
  validatedAt: string;
  /** Who performed validation */
  validatedBy?: string;
  /** Overall result */
  overallResult: string;
  /** Target version */
  targetVersion: number;
  
  // Validation results fields
  /** Whether syntax is valid */
  syntaxValid: boolean;
  /** Syntax errors if any */
  syntaxErrors: string[];
  /** Whether dependencies are valid */
  dependencyValid: boolean;
  /** Dependency errors if any */
  dependencyErrors: string[];
  /** Number of commands in migration */
  commandCount: number;
  commandCountValid: boolean;
  
  /** Data loss warnings */
  dataLossWarnings: string[];
  /** Performance warnings */
  performanceWarnings: string[];
  /** Bundles affected by migration */
  affectedBundles: string[];
  /** Number of documents impacted */
  documentsImpacted: number;
  /** Indexes affected */
  indexesAffected: string[];
  /** Estimated duration in milliseconds */
  estimatedDurationMs: number;
}

/**
 * Data impact from rollback
 */
export interface RollbackDataImpact {
  /** Bundles that will be dropped */
  bundlesDropped: string[];
  /** Fields that will be removed */
  fieldsRemoved: string[];
  /** Estimated data loss description */
  estimatedDataLoss: string;
}

/**
 * Rollback validation report from server
 */
export interface RollbackValidationReport {
  /** Whether rollback is possible */
  canRollback: boolean;
  /** Current database version */
  currentVersion: number;
  /** Target version for rollback */
  targetVersion: number;
  /** Migrations that will be reversed */
  migrationsToReverse: number[];
  /** DOWN commands for each migration */
  downCommands: Record<number, string[]>;
  /** Impact on data */
  dataImpact: RollbackDataImpact;
  /** Warnings about rollback */
  warnings?: string[];
  /** Errors preventing rollback */
  errors?: string[];
}

/**
 * Migration history response from server
 */
export interface MigrationHistoryResponse {
  /** Current database version */
  currentVersion: number;
  /** List of all migrations */
  migrations: MigrationInfo[];
  /** Database name */
  database?: string;
}

/**
 * Migration application result
 */
export interface MigrationApplicationResult {
  /** Applied migration version */
  version: number;
  /** Success message */
  message: string;
  /** Whether FORCE was used */
  force?: boolean;
}

/**
 * Rollback application result
 */
export interface RollbackApplicationResult {
  /** New current version after rollback */
  version: number;
  /** Success message */
  message: string;
  /** Versions that were rolled back */
  rolledBackVersions?: number[];
}
