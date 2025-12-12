import type { SchemaDiff, BundleChange, FieldChange, IndexChange } from '../schema/SchemaDefinition';

/**
 * Schema difference detail for conflict resolution
 */
export interface SchemaDifferenceDetail {
  /** Type of change */
  type: 'ADDED' | 'REMOVED' | 'MODIFIED';
  /** Bundle name */
  bundleName: string;
  /** Description of the change */
  change: string;
  /** Server's value/state */
  serverValue: any;
  /** Local value/state */
  localValue: any;
}

/**
 * Conflict diff showing version mismatch and schema differences
 * Provides both structured data and human-readable output
 */
export class ConflictDiff {
  constructor(
    public serverVersion: number,
    public localExpectedVersion: number,
    public schemaDifferences: SchemaDifferenceDetail[]
  ) {}

  /**
   * Formats the conflict as a human-readable string
   */
  toString(): string {
    const lines: string[] = [];
    
    lines.push('CONFLICT DETECTED:');
    lines.push('');
    lines.push(`Server is at migration version ${this.serverVersion}`);
    lines.push(`You are attempting to apply changes from version ${this.localExpectedVersion}`);
    lines.push('');
    lines.push('=== Schema Differences ===');
    lines.push('');

    // Group differences by bundle
    const bundleGroups = new Map<string, SchemaDifferenceDetail[]>();
    for (const diff of this.schemaDifferences) {
      if (!bundleGroups.has(diff.bundleName)) {
        bundleGroups.set(diff.bundleName, []);
      }
      bundleGroups.get(diff.bundleName)!.push(diff);
    }

    // Format each bundle's differences
    for (const [bundleName, diffs] of bundleGroups) {
      lines.push(`Bundle: ${bundleName}`);
      for (const diff of diffs) {
        lines.push(`  ${diff.change}`);
        lines.push(`    Server: ${this.formatValue(diff.serverValue)}`);
        lines.push(`    Local:  ${this.formatValue(diff.localValue)}`);
      }
      lines.push('');
    }

    lines.push('RESOLUTION OPTIONS:');
    lines.push('  1. Pull latest server migrations and rebase your local schema');
    lines.push('  2. Force apply your changes (WARNING: may cause data loss or conflicts)');
    lines.push('  3. Manually merge differences and create new migration');
    lines.push('');
    lines.push('Recommendation: Option 1 is safest for collaborative environments');

    return lines.join('\n');
  }

  /**
   * Formats a value for display
   */
  private formatValue(value: any): string {
    if (value === null || value === undefined) {
      return 'not present';
    }
    if (typeof value === 'object') {
      return JSON.stringify(value);
    }
    return String(value);
  }

  // TODO: Consider adding toJSON() method for serialization and toMarkdown() for rich CLI output if developer feedback requests enhanced reporting
}

/**
 * Resolves conflicts between local schema changes and server state
 * Follows Single Responsibility Principle - only handles conflict detection and formatting
 */
export class MigrationConflictResolver {
  /**
   * Detects if there's a version conflict
   * @param serverVersion - Current server version
   * @param localExpectedVersion - Version local schema expects to be at
   * @param schemaDiff - Local schema changes
   * @returns ConflictDiff if conflict exists, null otherwise
   */
  detectConflict(
    serverVersion: number,
    localExpectedVersion: number,
    schemaDiff: SchemaDiff
  ): ConflictDiff | null {
    // No conflict if server version matches expected and there are changes
    // Or if there are no changes regardless of version
    if (!schemaDiff.hasChanges || serverVersion === localExpectedVersion) {
      return null;
    }

    // Conflict exists if server is ahead and we have changes
    if (serverVersion > localExpectedVersion && schemaDiff.hasChanges) {
      const differences = this.generateDifferences(schemaDiff);
      return new ConflictDiff(serverVersion, localExpectedVersion, differences);
    }

    return null;
  }

  /**
   * Generates detailed difference descriptions from schema diff
   */
  private generateDifferences(schemaDiff: SchemaDiff): SchemaDifferenceDetail[] {
    const differences: SchemaDifferenceDetail[] = [];

    // Process bundle changes
    for (const bundleChange of schemaDiff.bundleChanges) {
      differences.push(...this.processBundleChange(bundleChange));
    }

    // Process index changes
    for (const indexChange of schemaDiff.indexChanges) {
      differences.push(...this.processIndexChange(indexChange));
    }

    return differences;
  }

  /**
   * Processes a bundle change into difference details
   */
  private processBundleChange(change: BundleChange): SchemaDifferenceDetail[] {
    const details: SchemaDifferenceDetail[] = [];

    if (change.type === 'create') {
      details.push({
        type: 'ADDED',
        bundleName: change.bundleName,
        change: 'Bundle added',
        serverValue: null,
        localValue: {
          fields: change.newDefinition?.fields.length,
          indexes: change.newDefinition?.indexes?.length || 0
        }
      });
    } else if (change.type === 'delete') {
      details.push({
        type: 'REMOVED',
        bundleName: change.bundleName,
        change: 'Bundle removed',
        serverValue: {
          fields: change.oldDefinition?.fields.length,
          indexes: change.oldDefinition?.indexes?.length || 0
        },
        localValue: null
      });
    } else if (change.type === 'modify') {
      // Process field changes
      for (const fieldChange of change.fieldChanges || []) {
        details.push(this.processFieldChange(change.bundleName, fieldChange));
      }

      // Process index changes
      for (const indexChange of change.indexChanges || []) {
        details.push(this.processIndexChangeDetail(change.bundleName, indexChange));
      }
    }

    return details;
  }

  /**
   * Processes a field change into difference detail
   */
  private processFieldChange(bundleName: string, change: FieldChange): SchemaDifferenceDetail {
    const formatField = (field: any) => {
      if (!field) return null;
      return `${field.type}, required=${field.required}, unique=${field.unique}`;
    };

    return {
      type: change.type === 'add' ? 'ADDED' : change.type === 'remove' ? 'REMOVED' : 'MODIFIED',
      bundleName,
      change: `Field: ${change.fieldName}`,
      serverValue: formatField(change.oldField),
      localValue: formatField(change.newField)
    };
  }

  /**
   * Processes an index change detail for a specific bundle
   */
  private processIndexChangeDetail(bundleName: string, change: IndexChange): SchemaDifferenceDetail {
    const formatIndex = (index: any) => {
      if (!index) return null;
      return `${index.type} on field ${index.fieldName}`;
    };

    return {
      type: change.type === 'create' ? 'ADDED' : 'REMOVED',
      bundleName,
      change: `Index: ${change.index?.fieldName || 'unknown'}`,
      serverValue: change.type === 'delete' ? formatIndex(change.index) : null,
      localValue: change.type === 'create' ? formatIndex(change.index) : null
    };
  }

  /**
   * Processes an index change into difference detail
   */
  private processIndexChange(change: IndexChange): SchemaDifferenceDetail[] {
    const formatIndex = (index: any) => {
      if (!index) return null;
      return `${index.type} on field ${index.fieldName}`;
    };

    return [{
      type: change.type === 'create' ? 'ADDED' : 'REMOVED',
      bundleName: change.bundleName,
      change: `Index: ${change.index?.fieldName || 'unknown'}`,
      serverValue: change.type === 'delete' ? formatIndex(change.index) : null,
      localValue: change.type === 'create' ? formatIndex(change.index) : null
    }];
  }
}
