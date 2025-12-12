/**
 * Schema definition types for SyndrDB
 * These interfaces define the structure for describing database schemas in plain objects
 */

/**
 * SyndrDB field types
 */
export type SyndrDBFieldType = 
  | 'STRING'
  | 'INT'
  | 'INTEGER'
  | 'FLOAT'
  | 'BOOLEAN'
  | 'DATETIME'
  | 'JSON'
  | 'TEXT'
  | 'relationship';

/**
 * Field definition for a bundle
 */
export interface FieldDefinition {
  /** Field name */
  name: string;
  /** Field data type */
  type: SyndrDBFieldType;
  /** Whether the field is required */
  required: boolean;
  /** Whether the field must be unique */
  unique: boolean;
  /** Default value for the field */
  defaultValue?: any;
  /** Related bundle name (for relationship fields) */
  relatedBundle?: string;
}

/**
 * Relationship definition between bundles
 */
export interface RelationshipDefinition {
  /** Relationship name */
  //name: string;
 

  name: string;

    // Source field for the relationship (e.g., "DocumentID")
    sourceField: string;
    // Source bundle name
    sourceBundle: string;

    // Destination bundle name
    destinationBundle: string;

    // Destination field for the relationship (e.g., "OrderID")
    destinationField: string;

    // Type is the type of the relationship (e.g., "0toMany", "1toMany", "ManyToMany").
    relationshipType: string;
}

/**
 * Index type for SyndrDB
 */
export type IndexType = 'hash' | 'btree';

/**
 * Index definition for a bundle  
 */
export interface IndexDefinition {
  /** Field name for the index */
  fieldName: string;
  /** Type of index */
  type: IndexType;
}

/**
 * View definition
 */
export interface ViewDefinition {
  /** View name */
  name: string;
  /** Query that defines the view */
  query: string;
}

/**
 * Bundle definition
 */
export interface BundleDefinition {
  /** Bundle name */
  name: string;
  /** Fields in this bundle */
  fields: FieldDefinition[];
  /** Indexes for this bundle */
  indexes: IndexDefinition[];

  relationships: RelationshipDefinition[];
}

/**
 * Complete schema definition for a database
 */
export interface SchemaDefinition {
  /** Bundles in the schema */
  bundles: BundleDefinition[];
  /** Views in the schema */
  views?: ViewDefinition[];
  /** Optional local version for informational purposes (server is authoritative) */
  version?: number;
  /** Optional description of this schema */
  description?: string;
}

/**
 * Schema change types
 */
export type SchemaChangeType = 'ADDED' | 'REMOVED' | 'MODIFIED';

/**
 * Details of a bundle change
 */
export interface BundleChange {
  type: 'create' | 'delete' | 'modify';
  bundleName: string;
  /** For create: the new bundle definition */
  newDefinition?: BundleDefinition;
  /** For delete: the old bundle definition */
  oldDefinition?: BundleDefinition;
  /** For modify: specific field changes */
  fieldChanges?: FieldChange[];
  /** For modify: specific index changes within the bundle */
  indexChanges?: IndexChange[];

  relationshipChanges: RelationshipChange[];
}

/**
 * Details of a field change
 */
export interface FieldChange {
  type: 'add' | 'remove' | 'modify';
  fieldName: string;
  /** For add: new field definition */
  newField?: FieldDefinition;
  /** For remove: old field definition */
  oldField?: FieldDefinition;
}

/**
 * Details of a relationship change
 */
export interface RelationshipChange {
  type: 'create' | 'delete' | 'modify';
  relationshipName: string;
  /** For ADDED/MODIFIED: new relationship definition */
  newDefinitions?: RelationshipDefinition[];
  /** For REMOVED/MODIFIED: old relationship definition */
  oldDefinitions?: RelationshipDefinition[];
}

/**
 * Details of an index change
 */
export interface IndexChange {
  type: 'create' | 'delete';
  bundleName: string;
  index: IndexDefinition;
}



/**
 * Complete schema diff representing changes between two schemas
 */
export interface SchemaDiff {
  /** Bundle changes */
  bundleChanges: BundleChange[];
  /** Index changes */
  indexChanges: IndexChange[];
  
  /** relationship changes */
  relationshipChanges: RelationshipChange[];

  /** Whether there are any changes */


  hasChanges: boolean;
}
