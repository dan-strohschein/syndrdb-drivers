/**
 * Schema manager for code generation and schema operations
 * Provides utilities for TypeScript, JSON Schema, and GraphQL generation
 */

import type { SyndrDBClient } from '../client';
import type {
  SchemaDefinition,
  BundleDefinition,
  FieldDefinition,
  IndexDefinition,
  JSONSchemaMode,
  TypeScriptGenerationOptions,
  GraphQLGenerationOptions,
} from '../types/schema';
import { ValidationError } from '../errors';

/**
 * Schema manager for code generation
 */
export class SchemaManager {
  private client: SyndrDBClient;

  constructor(client: SyndrDBClient) {
    this.client = client;
  }

  /**
   * Generate JSON Schema from current database schema
   * @param mode Generation mode (single or multiple files)
   * @returns JSON Schema string
   */
  async generateJSONSchema(mode: JSONSchemaMode = 'single'): Promise<string> {
    try {
      return await this.client.generateJSONSchema(mode);
    } catch (error) {
      throw new ValidationError(
        `Failed to generate JSON Schema: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Generate GraphQL schema from current database schema
   * @param options Generation options
   * @returns GraphQL schema string
   */
  async generateGraphQLSchema(options: GraphQLGenerationOptions = {}): Promise<string> {
    try {
      return await this.client.generateGraphQLSchema(options);
    } catch (error) {
      throw new ValidationError(
        `Failed to generate GraphQL schema: ${(error as Error).message}`,
        undefined,
        undefined,
        error as Error
      );
    }
  }

  /**
   * Generate TypeScript types from schema definition
   * @param schema Schema definition
   * @param options Generation options
   * @returns TypeScript code
   */
  generateTypeScript(
    schema: SchemaDefinition,
    options: TypeScriptGenerationOptions = {}
  ): string {
    const lines: string[] = [];
    const includeJSDoc = options.includeJSDoc ?? true;
    const generateTypeGuards = options.generateTypeGuards ?? false;
    const typeMappings = options.typeMappings ?? this.getDefaultTypeMappings();

    lines.push('/**');
    lines.push(' * Auto-generated TypeScript types from SyndrDB schema');
    lines.push(` * Generated: ${new Date().toISOString()}`);
    lines.push(' * Do not edit manually');
    lines.push(' */');
    lines.push('');

    for (const bundle of schema.bundles) {
      this.generateBundleType(bundle, lines, includeJSDoc, typeMappings);
      lines.push('');

      if (generateTypeGuards) {
        this.generateTypeGuard(bundle, lines);
        lines.push('');
      }
    }

    return lines.join('\n');
  }

  /**
   * Generate TypeScript interface for a bundle
   * @param bundle Bundle definition
   * @param lines Output lines
   * @param includeJSDoc Whether to include JSDoc
   * @param typeMappings Type mappings
   */
  private generateBundleType(
    bundle: BundleDefinition,
    lines: string[],
    includeJSDoc: boolean,
    typeMappings: Record<string, string>
  ): void {
    if (includeJSDoc) {
      lines.push('/**');
      lines.push(` * ${bundle.name} bundle`);
      if (bundle.metadata?.description) {
        lines.push(` * ${bundle.metadata.description}`);
      }
      lines.push(' */');
    }

    lines.push(`export interface ${bundle.name} {`);

    for (const field of bundle.fields) {
      if (includeJSDoc && field.metadata?.description) {
        lines.push(`  /** ${field.metadata.description} */`);
      }

      const tsType = this.mapFieldType(field, typeMappings);
      const optional = field.required ? '' : '?';
      lines.push(`  ${field.name}${optional}: ${tsType};`);
    }

    lines.push('}');
  }

  /**
   * Generate type guard function
   * @param bundle Bundle definition
   * @param lines Output lines
   */
  private generateTypeGuard(bundle: BundleDefinition, lines: string[]): void {
    lines.push(`/**`);
    lines.push(` * Type guard for ${bundle.name}`);
    lines.push(` */`);
    lines.push(`export function is${bundle.name}(obj: unknown): obj is ${bundle.name} {`);
    lines.push(`  if (typeof obj !== 'object' || obj === null) return false;`);
    lines.push(`  const record = obj as Record<string, unknown>;`);

    for (const field of bundle.fields.filter((f) => f.required)) {
      lines.push(`  if (!('${field.name}' in record)) return false;`);
    }

    lines.push(`  return true;`);
    lines.push(`}`);
  }

  /**
   * Map field type to TypeScript type
   * @param field Field definition
   * @param typeMappings Custom type mappings
   * @returns TypeScript type
   */
  private mapFieldType(field: FieldDefinition, typeMappings: Record<string, string>): string {
    const mappedType = typeMappings[field.type];
    if (mappedType) {
      return mappedType;
    }

    switch (field.type) {
      case 'STRING':
      case 'TEXT':
        return 'string';
      case 'INT':
      case 'FLOAT':
      case 'DOUBLE':
        return 'number';
      case 'BOOL':
        return 'boolean';
      case 'DATETIME':
        return 'Date';
      case 'BYTES':
        return 'Uint8Array';
      case 'relationship':
        return field.relatedBundle || 'unknown';
      default:
        return 'unknown';
    }
  }

  /**
   * Get default type mappings
   * @returns Default mappings
   */
  private getDefaultTypeMappings(): Record<string, string> {
    return {
      STRING: 'string',
      TEXT: 'string',
      INT: 'number',
      FLOAT: 'number',
      DOUBLE: 'number',
      BOOL: 'boolean',
      DATETIME: 'Date',
      BYTES: 'Uint8Array',
    };
  }

  /**
   * Validate schema definition
   * @param schema Schema to validate
   * @returns Validation errors
   */
  validateSchema(schema: SchemaDefinition): string[] {
    const errors: string[] = [];

    if (!schema.bundles || schema.bundles.length === 0) {
      errors.push('Schema must contain at least one bundle');
      return errors;
    }

    const bundleNames = new Set<string>();

    for (const bundle of schema.bundles) {
      // Check for duplicate bundle names
      if (bundleNames.has(bundle.name)) {
        errors.push(`Duplicate bundle name: ${bundle.name}`);
      }
      bundleNames.add(bundle.name);

      // Validate fields
      if (!bundle.fields || bundle.fields.length === 0) {
        errors.push(`Bundle '${bundle.name}' must contain at least one field`);
        continue;
      }

      const fieldNames = new Set<string>();
      for (const field of bundle.fields) {
        // Check for duplicate field names
        if (fieldNames.has(field.name)) {
          errors.push(`Duplicate field name in bundle '${bundle.name}': ${field.name}`);
        }
        fieldNames.add(field.name);

        // Validate relationship fields
        if (field.type === 'relationship') {
          if (!field.relatedBundle) {
            errors.push(
              `Relationship field '${bundle.name}.${field.name}' must specify relatedBundle`
            );
          } else if (
            !schema.bundles.find((b) => b.name === field.relatedBundle) &&
            field.relatedBundle !== bundle.name // Allow self-references
          ) {
            errors.push(
              `Relationship field '${bundle.name}.${field.name}' references unknown bundle: ${field.relatedBundle}`
            );
          }
        }
      }

      // Validate indexes
      if (bundle.indexes) {
        for (const index of bundle.indexes) {
          if (!bundle.fields.find((f) => f.name === index.fieldName)) {
            errors.push(
              `Index in bundle '${bundle.name}' references unknown field: ${index.fieldName}`
            );
          }
        }
      }
    }

    return errors;
  }

  /**
   * Create schema builder
   * @returns Schema builder instance
   */
  static builder(): SchemaBuilder {
    return new SchemaBuilder();
  }
}

/**
 * Fluent schema builder
 */
export class SchemaBuilder {
  private schema: SchemaDefinition = { bundles: [] };
  private currentBundle: BundleDefinition | null = null;

  /**
   * Add a bundle
   * @param name Bundle name
   * @returns Builder instance
   */
  bundle(name: string): this {
    this.currentBundle = { name, fields: [], indexes: [] };
    this.schema.bundles.push(this.currentBundle);
    return this;
  }

  /**
   * Add a field to current bundle
   * @param field Field definition
   * @returns Builder instance
   */
  field(field: FieldDefinition): this {
    if (!this.currentBundle) {
      throw new Error('Must call bundle() before adding fields');
    }
    this.currentBundle.fields.push(field);
    return this;
  }

  /**
   * Add an index to current bundle
   * @param index Index definition
   * @returns Builder instance
   */
  index(index: IndexDefinition): this {
    if (!this.currentBundle) {
      throw new Error('Must call bundle() before adding indexes');
    }
    this.currentBundle.indexes?.push(index);
    return this;
  }

  /**
   * Set schema version
   * @param version Version number
   * @returns Builder instance
   */
  version(version: number): this {
    this.schema.version = version;
    return this;
  }

  /**
   * Set schema metadata
   * @param metadata Metadata object
   * @returns Builder instance
   */
  metadata(metadata: Record<string, unknown>): this {
    this.schema.metadata = metadata;
    return this;
  }

  /**
   * Build final schema
   * @returns Schema definition
   */
  build(): SchemaDefinition {
    return this.schema;
  }
}
