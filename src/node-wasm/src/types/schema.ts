/**
 * Field type enumeration
 */
export type FieldType =
  | 'STRING'
  | 'INT'
  | 'FLOAT'
  | 'DOUBLE'
  | 'BOOL'
  | 'DATETIME'
  | 'TEXT'
  | 'BYTES'
  | 'relationship';

/**
 * Index type enumeration
 */
export type IndexType = 'hash' | 'btree' | 'unique';

/**
 * Field definition in a bundle
 */
export interface FieldDefinition {
  /** Field name */
  name: string;

  /** Field data type */
  type: FieldType;

  /** Whether field is required */
  required: boolean;

  /** Whether field values must be unique */
  unique: boolean;

  /** Related bundle name (for relationship fields) */
  relatedBundle?: string;

  /** Default value */
  defaultValue?: unknown;

  /** Field metadata */
  metadata?: Record<string, unknown>;
}

/**
 * Index definition
 */
export interface IndexDefinition {
  /** Field name to index */
  fieldName: string;

  /** Index type */
  type: IndexType;

  /** Index name (auto-generated if not provided) */
  name?: string;
}

/**
 * Bundle (table) definition
 */
export interface BundleDefinition {
  /** Bundle name */
  name: string;

  /** Bundle fields */
  fields: FieldDefinition[];

  /** Bundle indexes */
  indexes?: IndexDefinition[];

  /** Bundle metadata */
  metadata?: Record<string, unknown>;
}

/**
 * Complete schema definition
 */
export interface SchemaDefinition {
  /** List of bundles */
  bundles: BundleDefinition[];

  /** Schema version */
  version?: number;

  /** Schema metadata */
  metadata?: Record<string, unknown>;
}

/**
 * JSON Schema generation mode
 */
export type JSONSchemaMode = 'single' | 'multiple';

/**
 * TypeScript type generation options
 */
export interface TypeScriptGenerationOptions {
  /** Output directory for generated files */
  outputDir?: string;

  /** Whether to include JSDoc comments */
  includeJSDoc?: boolean;

  /** Whether to generate type guards */
  generateTypeGuards?: boolean;

  /** Custom type mappings */
  typeMappings?: Record<string, string>;
}

/**
 * GraphQL schema generation options
 */
export interface GraphQLGenerationOptions {
  /** Whether to include mutations */
  includeMutations?: boolean;

  /** Whether to include subscriptions */
  includeSubscriptions?: boolean;

  /** Custom scalar mappings */
  scalarMappings?: Record<string, string>;
}
