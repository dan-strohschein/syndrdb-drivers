import type { ValidationReport } from './MigrationTypes';

/**
 * Validates and formats migration validation reports
 * Follows Single Responsibility Principle - only handles validation report processing
 */
export class MigrationValidator {
  /**
   * Checks if the validation report has any errors
   */
  hasErrors(report: ValidationReport): boolean {
    return !report.syntaxValid || !report.dependencyValid || report.syntaxErrors.length > 0 || report.dependencyErrors.length > 0;
  }

  /**
   * Checks if the validation report has any warnings
   */
  hasWarnings(report: ValidationReport): boolean {
    return report.dataLossWarnings.length > 0 || report.performanceWarnings.length > 0;
  }

  /**
   * Formats the validation report into a human-readable string
   */
  getFormattedReport(report: ValidationReport): string {
    const lines: string[] = [];
    
    lines.push(`Migration Version ${report.migrationVersion} Validation Report`);
    lines.push(`Validated at: ${report.validatedAt}`);
    lines.push(`Validated by: ${report.validatedBy || 'system'}`);
    lines.push(`Overall Result: ${report.overallResult}`);
    lines.push('');

    // Syntax Validation
    lines.push('=== Syntax Validation ===');
    lines.push(`Status: ${report.syntaxValid ? 'PASSED' : 'FAILED'}`);
    if (report.syntaxErrors.length > 0) {
      lines.push('  Errors:');
      for (const error of report.syntaxErrors) {
        lines.push(`    • ${error}`);
      }
    } else {
      lines.push('  No issues found');
    }
    lines.push('');

    // Dependency Validation
    lines.push('=== Dependency Validation ===');
    lines.push(`Status: ${report.dependencyValid ? 'PASSED' : 'FAILED'}`);
    if (report.dependencyErrors.length > 0) {
      lines.push('  Errors:');
      for (const error of report.dependencyErrors) {
        lines.push(`    • ${error}`);
      }
    } else {
      lines.push('  No issues found');
    }
    lines.push('');

    // Performance Estimation
    lines.push('=== Performance Estimation ===');
    lines.push(`Status: ${report.performanceWarnings.length === 0 ? 'PASSED' : 'PASSED WITH WARNINGS'}`);
    lines.push(`  Command Count: ${report.commandCount}`);
    lines.push(`  Estimated Duration: ${report.estimatedDurationMs}ms`);
    lines.push(`  Documents Impacted: ${report.documentsImpacted}`);
    if (report.performanceWarnings.length > 0) {
      lines.push('  Warnings:');
      for (const warning of report.performanceWarnings) {
        lines.push(`    • ${warning}`);
      }
    } else {
      lines.push('  No issues found');
    }
    lines.push('');

    // Data Loss Check
    lines.push('=== Data Loss Check ===');
    lines.push(`Status: ${report.dataLossWarnings.length === 0 ? 'PASSED' : 'PASSED WITH WARNINGS'}`);
    if (report.dataLossWarnings.length > 0) {
      lines.push('  Warnings:');
      for (const warning of report.dataLossWarnings) {
        lines.push(`    • ${warning}`);
      }
    } else {
      lines.push('  No data loss expected');
    }
    lines.push('');

    // Impact Summary
    lines.push('=== Impact Summary ===');
    lines.push(`Affected Bundles: ${report.affectedBundles.length > 0 ? report.affectedBundles.join(', ') : 'None'}`);
    lines.push(`Indexes Affected: ${report.indexesAffected.length > 0 ? report.indexesAffected.join(', ') : 'None'}`);
    lines.push('');

    // Summary
    lines.push('=== Summary ===');
    const canApply = report.syntaxValid && report.dependencyValid;
    lines.push(`Can Apply: ${canApply ? 'YES' : 'NO'}`);
    lines.push(`Can Apply with FORCE: ${canApply || report.dataLossWarnings.length > 0 ? 'YES' : 'NO'}`);

    return lines.join('\n');
  }
}
