import type { 
  SchemaDefinition, 
  BundleDefinition, 
  FieldDefinition, 
  IndexDefinition,
  SchemaChangeType,
  SchemaDiff,
  BundleChange,
  FieldChange,
  IndexChange,
  RelationshipChange,
  IndexType,
  RelationshipDefinition
} from './SchemaDefinition';

/**
 * Server bundle metadata structure from SHOW BUNDLES response
 */
interface ServerBundleMetadata {
  BundleMetadata: {
    Name: string;
    DocumentStructure: {
      FieldDefinitions: Record<string, {
        Name: string;
        Type: string;
        Required: boolean;
        Unique: boolean;
        DefaultValue: any;
      }>;
    };
    Indexes: Record<string, {
      IndexName: string;
      IndexType: string;
      HashIndexField?: {
        FieldName: string;
        IsUnique: boolean;
      };
      BTreeIndexField?: {
        FieldName: string;
        IsUnique: boolean;
      };
    }>;
    Relationships: Record<string, any>;
  };
}

/**
 * Server response from SHOW BUNDLES command
 */
interface ShowBundlesResponse {
  ExecutionTimeMS: number;
  Result: ServerBundleMetadata[];
  ResultCount: number;
}

/**
 * Manages schema fetching and comparison
 * Follows Single Responsibility Principle - only handles schema operations
 */
export class SchemaManager {
  /**
   * Parses server SHOW BUNDLES response into SchemaDefinition
   * @param response - Raw response from SHOW BUNDLES FOR "database"
   * @returns Parsed schema definition
   */
  parseServerSchema(response: ShowBundlesResponse): SchemaDefinition {
    const bundles: BundleDefinition[] = [];
    const indexes: IndexDefinition[] = [];

    for (const item of response.Result) {
      const metadata = item.BundleMetadata;
      
      // Parse fields
      const fields: FieldDefinition[] = [];
      for (const [fieldName, fieldDef] of Object.entries(metadata.DocumentStructure.FieldDefinitions)) {
        fields.push({
          name: fieldDef.Name,
          type: fieldDef.Type as any,
          required: fieldDef.Required,
          unique: fieldDef.Unique,
          defaultValue: fieldDef.DefaultValue
        });
      }

      // Parse relationships
      const relationships = [];
      for (const [relName, relDef] of Object.entries(metadata.Relationships || {})) {
        relationships.push({
          name: relName,
          fromBundle: metadata.Name,
          toBundle: (relDef as any).ToBundle || ''
        });
      }

      bundles.push({
        name: metadata.Name,
        fields,
        indexes: [], // Will be populated below
        relationships: []
      });

      // Parse indexes and add to bundle
      const currentBundle = bundles[bundles.length - 1];
      for (const [indexName, indexDef] of Object.entries(metadata.Indexes || {})) {
        const indexType: IndexType = indexDef.IndexType as IndexType;
        let fieldName = '';

        // Extract field name based on index type
        if (indexType === 'hash' && indexDef.HashIndexField) {
          fieldName = indexDef.HashIndexField.FieldName;
        } else if (indexType === 'btree' && indexDef.BTreeIndexField) {
          fieldName = indexDef.BTreeIndexField.FieldName;
        }

        if (fieldName) {
          currentBundle.indexes.push({
            fieldName,
            type: indexType
          });
        }
      }
    }

    return {
      bundles
    };
  }

  /**
   * Compares two schemas and generates a diff
   * @param local - Local schema definition
   * @param server - Server schema definition
   * @returns Schema diff
   */
  compareSchemas(local: SchemaDefinition, server: SchemaDefinition): SchemaDiff {
    const bundleChanges: BundleChange[] = [];

    // Create maps for easier lookup
    const localBundleMap = new Map(local.bundles.map(b => [b.name, b]));
    const serverBundleMap = new Map(server.bundles.map(b => [b.name, b]));
    let relationshipChanges: RelationshipChange[] = [];
    const allRelationshipChanges: RelationshipChange[] = [];

    // Find added and modified bundles
    for (const localBundle of local.bundles) {
      const serverBundle = serverBundleMap.get(localBundle.name); //this has books
      
      if (!serverBundle) { //books isn't on the server
        // Bundle added
        const bChange: BundleChange = {
          type: 'create',
          bundleName: localBundle.name,
          newDefinition: localBundle,
          relationshipChanges: new Array<RelationshipChange>()// Will be populated below
        }
        
        
         // Collect relationship changes for this new bundle
        // If there are no relationships on the bundle, we don't need to do anything
 console.log(`FUCK: bundle : ${localBundle.name} has ${localBundle.relationships.length} Rels!!!!`)    

        if (localBundle.relationships.length > 0) {
          const relationshipChanges = this.compareRelationships(
            localBundle.relationships, 
             [] as RelationshipDefinition[]// Server has no relationships since bundle is new
          );
         
          bChange.relationshipChanges = relationshipChanges;
        }
        bundleChanges.push(bChange);
      } else {
        // Check for modifications
        const fieldChanges = this.compareFields(localBundle.fields, serverBundle.fields);
        const indexChanges = this.compareIndexes(localBundle.indexes, serverBundle.indexes, localBundle.name);
        relationshipChanges = this.compareRelationships(localBundle.relationships, serverBundle.relationships);

        // Collect all relationship changes
        //allRelationshipChanges.push(...relationshipChanges);

        if (fieldChanges.length > 0 || indexChanges.length > 0) {
          bundleChanges.push({
            type: 'modify',
            bundleName: localBundle.name,
            fieldChanges,
            indexChanges,
            relationshipChanges
          });
        }
      }
    }

    // Find removed bundles
    for (const serverBundle of server.bundles) {
      if (!localBundleMap.has(serverBundle.name)) {
        bundleChanges.push({
          type: 'delete',
          bundleName: serverBundle.name,
          oldDefinition: serverBundle,
          relationshipChanges: new Array<RelationshipChange>()
        });
      }
    }

     // Filter relationship changes to only include those where BOTH bundles exist in this migration
    // This handles the case where we're creating both Authors and Books bundles in the same migration
    const bundlesInMigration = new Set<string>();
    
    // Add all bundles that will exist after this migration
    for (const bundle of local.bundles) {
      bundlesInMigration.add(bundle.name);
    }

console.log("FUCK2 : # of rels", allRelationshipChanges.length)

    const validRelationshipChanges = allRelationshipChanges.filter(change => {

    if (change.type === 'create') {
       const relDef = change.newDefinitions;
        if (!relDef) return false;
          //Both bundles have to exist for this to work
        // Both source and destination bundles must be in the migration
        const sourceExists = bundlesInMigration.has(relDef[0].sourceBundle);
        const destExists = bundlesInMigration.has(relDef[0].destinationBundle);
        
        return sourceExists && destExists;

      } else if(change.type === 'delete') {
        const relDef = change.oldDefinitions;
        if (!relDef) return false;

        const sourceExists = bundlesInMigration.has(relDef[0].sourceBundle);
        const destExists = bundlesInMigration.has(relDef[0].destinationBundle);
        
        return sourceExists && destExists;
      } else {
        //we dont handle the modify yet, there's too many issues. Just delete and recreate
        return false;
      }
    });

console.log("FUCK3 : # of rels", validRelationshipChanges.length)

    return {
      bundleChanges,
      indexChanges: [], // Index changes are nested in bundleChanges
      hasChanges: bundleChanges.length > 0,
      relationshipChanges: validRelationshipChanges // Relationship changes are nested in bundle changes
    };
  }

  /**
   * Compares field definitions
   */
  private compareFields(localFields: FieldDefinition[], serverFields: FieldDefinition[]): FieldChange[] {
    const changes: FieldChange[] = [];
    const localFieldMap = new Map(localFields.map(f => [f.name, f]));
    const serverFieldMap = new Map(serverFields.map(f => [f.name, f]));

 //   console.log('[compareFields] Local fields:', localFields.map(f => f.name));
 //   console.log('[compareFields] Server fields:', serverFields.map(f => f.name));

    // Find added and modified fields
    for (const localField of localFields) {
      const serverField = serverFieldMap.get(localField.name);
      
      if (!serverField) {
 //       console.log(`[compareFields] Field ${localField.name} added`);
        changes.push({
          type: 'add',
          fieldName: localField.name,
          newField: localField
        });
      } else if (!this.fieldsEqual(localField, serverField)) {
//        console.log(`[compareFields] Field ${localField.name} modified`);
        changes.push({
          type: 'modify',
          fieldName: localField.name,
          newField: localField,
          oldField: serverField
        });
      }
    }

    // Find removed fields
    for (const serverField of serverFields) {
      if (!localFieldMap.has(serverField.name)) {
 //       console.log(`[compareFields] Field ${serverField.name} removed - fieldName:`, JSON.stringify(serverField.name));
        changes.push({
          type: 'remove',
          fieldName: serverField.name,
          oldField: serverField
        });
      }
    }

//    console.log('[compareFields] Total changes:', changes.length);
    return changes;
  }

  private compareRelationships(local: RelationshipDefinition[], server: RelationshipDefinition[]): RelationshipChange[] {
    const changes: RelationshipChange[] = [];
    const localRelationshipMap = new Map(local.map(i => [i.name, i]));
    const serverRelationshipMap = new Map(server.map(i => [i.name, i]));


    for (const localRelationship of local) {
      const serverRelationship = serverRelationshipMap.get(localRelationship.name);
      
      if (!serverRelationship) {
        changes.push({
          type: 'create',
          oldDefinitions: undefined,
          newDefinitions: [localRelationship],
          relationshipName: localRelationship.name
        });
         
      } else if (!this.relationshipsEqual(localRelationship, serverRelationship)) {
        changes.push({
          type: 'delete', // Delete old and will create new
          oldDefinitions: [localRelationship],
          newDefinitions: undefined,
          relationshipName: localRelationship.name
        });
        changes.push({
          type: 'create',
          oldDefinitions: undefined,
          newDefinitions: [localRelationship],
          relationshipName: localRelationship.name
        });
      }
    }

     // Find removed indexes
    for (const serverRelationship of server) {
         const localRelationship = localRelationshipMap.get(serverRelationship.name);
      
      if (localRelationship && !localRelationshipMap.has(serverRelationship.name)) {
        changes.push({
          type: 'delete', // Delete old and will create new
          oldDefinitions: [localRelationship],
          newDefinitions: undefined,
          relationshipName: serverRelationship.name
        });
      }
    }

    return changes;
}
  /**
   * Compares index definitions within a bundle
   */
  private compareIndexes(local: IndexDefinition[], server: IndexDefinition[], bundleName: string): IndexChange[] {
    const changes: IndexChange[] = [];
    const localIndexMap = new Map(local.map(i => [i.fieldName, i]));
    const serverIndexMap = new Map(server.map(i => [i.fieldName, i]));

    // Find added and modified indexes
    for (const localIndex of local) {
      const serverIndex = serverIndexMap.get(localIndex.fieldName);
      
      if (!serverIndex) {
        changes.push({
          type: 'create',
          bundleName,
          index: localIndex
        });
      } else if (!this.indexesEqual(localIndex, serverIndex)) {
        changes.push({
          type: 'delete', // Delete old and will create new
          bundleName,
          index: serverIndex
        });
        changes.push({
          type: 'create',
          bundleName,
          index: localIndex
        });
      }
    }

    // Find removed indexes
    for (const serverIndex of server) {
      if (!localIndexMap.has(serverIndex.fieldName)) {
        changes.push({
          type: 'delete',
          bundleName,
          index: serverIndex
        });
      }
    }

    return changes;
  }

  /**
   * Checks if two fields are equal
   */
  private fieldsEqual(a: FieldDefinition, b: FieldDefinition): boolean {
    return a.type === b.type &&
           a.required === b.required &&
           a.unique === b.unique &&
           JSON.stringify(a.defaultValue) === JSON.stringify(b.defaultValue);
  }

  /**
   * Checks if two indexes are equal
   */
  private indexesEqual(a: IndexDefinition, b: IndexDefinition): boolean {
    return a.fieldName === b.fieldName &&
           a.type === b.type;
  }

  private relationshipsEqual(a: RelationshipDefinition, b: RelationshipDefinition): boolean {
    return a.name === b.name && a.destinationBundle === b.destinationBundle 
    && a.sourceBundle === b.sourceBundle 
    && a.relationshipType === b.relationshipType;
  }
}
