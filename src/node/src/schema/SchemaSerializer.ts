import type { SchemaDiff, BundleDefinition, FieldDefinition, IndexDefinition, RelationshipDefinition, RelationshipChange, BundleChange } from '../schema/SchemaDefinition';

/**
 * Serializes schema changes into SyndrDB command strings
 * Follows the Single Responsibility Principle - only handles command generation
 */
export class SchemaSerializer {
  /**
   * Converts a SchemaDiff into an array of SyndrDB commands
   * @param diff - The schema differences to serialize
   * @returns Array of command strings
   */
  serializeDiff(diff: SchemaDiff): string[] {
    const commands: string[] = [];
    const relCommands: string[] = [];

    // Process bundle changes
    for (const bundleChange of diff.bundleChanges) {
      switch (bundleChange.type) {
        case 'create':
          if (bundleChange.newDefinition) {
            // Create the bundle first
            commands.push(this.serializeCreateBundle(bundleChange.newDefinition));
            
            // Note: Relationships should be added AFTER all bundles are created
            // They will be handled in a separate step or migration
            if(bundleChange.relationshipChanges && bundleChange.relationshipChanges.length > 0) {
              relCommands.push(this.serializeRelationships(bundleChange));
            }
          }
          break;
        case 'delete':
          commands.push(this.serializeDropBundle(bundleChange.bundleName));
          break;
        case 'modify':
          if (bundleChange.fieldChanges) {
            commands.push(...this.serializeFieldChanges(bundleChange.bundleName, bundleChange.fieldChanges));
          }
          if (bundleChange.indexChanges) {
            commands.push(...this.serializeIndexChangesForBundle(bundleChange.bundleName, bundleChange.indexChanges));
          }
          break;
      }
    }

    //There is this idea that during bundle create, we may have relationships to add. 
    // if so, we need to add them AFTER all of the bundles have been built.
    if (relCommands.length > 0) {
      commands.push(...relCommands);
    }

    // Top-level index changes (if any - now handled per-bundle)
    for (const indexChange of diff.indexChanges) {
      switch (indexChange.type) {
        case 'create':
          if (indexChange.index) {
            commands.push(this.serializeCreateIndexForBundle(indexChange.bundleName, indexChange.index));
          }
          break;
        case 'delete':
          if (indexChange.index) {
            commands.push(this.serializeDropIndexForBundle(indexChange.bundleName, indexChange.index.fieldName));
          }
          break;
      }
    }

    return commands;
  }

  /**
   * Serializes a CREATE BUNDLE command
   */
  private serializeCreateBundle(bundle: BundleDefinition): string {
    const fieldStrings = bundle.fields
      .map(field => this.serializeFieldForCreate(field));
    return `CREATE BUNDLE "${bundle.name}"\nWITH FIELDS (\n    ${fieldStrings.join(',\n    ')}\n);`;
  }

  /**
   * Serializes a field definition for CREATE BUNDLE
   * Format: {"fieldName", "type", required, unique, defaultValue}
   */
  private serializeFieldForCreate(field: FieldDefinition): string {
    const defaultValue = this.serializeDefaultValue(field);
    return `{"${field.name}", "${field.type}", ${field.required ? 'TRUE' : 'FALSE'}, ${field.unique ? 'TRUE' : 'FALSE'}, ${defaultValue}}`;
  }

  /**
   * Serializes a field definition for CREATE BUNDLE or UPDATE BUNDLE (deprecated - use specific methods)
   */
  private serializeField(field: FieldDefinition): string {
    return this.serializeFieldForCreate(field);
  }

  /**
   * Serializes a DROP BUNDLE command
   */
  private serializeDropBundle(bundleName: string): string {
    return `DROP BUNDLE "${bundleName}";`;
  }

  /**
   * Serializes relationship fields into ADD RELATIONSHIP commands
   * Format: UPDATE BUNDLE "bundle" ADD RELATIONSHIP ("1toMany", "source", "sourceField", "dest", "destField")
   */
  private serializeRelationships(bundleChange: BundleChange): string {
   
    let relationshipCommands = `UPDATE BUNDLE "${bundleChange.bundleName}"\n`;

    for (const relationshipChange of bundleChange.relationshipChanges || []) {
      
      if (relationshipChange.type === 'create') {
        relationshipCommands += `ADD RELATIONSHIP (\n `;
        let counter = 1;
        if (relationshipChange.newDefinitions) {
          for (const relationshipAddChange of relationshipChange.newDefinitions || []) {
            const relationshipName = relationshipAddChange.name || '';
            // Determine relationship type - default to 1toMany
            const relationshipType = relationshipAddChange.relationshipType ||  '1toMany';
            
            // Source bundle is the related bundle, source field is typically DocumentID
            const sourceBundle = relationshipAddChange.sourceBundle || '';
            const sourceField = relationshipAddChange.sourceField || '';
            
            // Destination bundle is current bundle, destination field is the relationship field
            const destBundle = relationshipAddChange.destinationBundle || '';
            const destField = relationshipAddChange.destinationField || '';

            // The true syntax for this in SyndrQL is:
            // UPDATE BUNDLE "SourceBundle"
            // ADD RELATIONSHIP (
            //      "relationshipName"
            //      {
            //        "relationshipType",
            //        "sourceBundle",
            //        "sourceField",
            //        "destBundle",
            //        "destField"
            //      },
            //      "relationshipName2"
            //      {
            //        "relationshipType",
            //        "sourceBundle2",
            //        "sourceField2",
            //        "destBundle2",
            //        "destField2"
            //      },
            // );
            // This technically allows you to add multiple relationships in one command. We will implement that in the migration test
            // Later. For this
            // , we will do one command per relationship.
            relationshipCommands +=  `     \"${relationshipName}\" ` +
              `   {\n` +
              `       "${relationshipType}",\n` +
              `       "${sourceBundle}",\n` +
              `       "${sourceField}",\n` +
              `       "${destBundle}",\n` +
              `       "${destField}"\n` +
              `   }\n` 
            ;
            if (counter < relationshipChange.newDefinitions.length) {
              relationshipCommands += ",";
            }
          }
          relationshipCommands += `)\n `;
        }
      }
    }

  

    return relationshipCommands;
  }

  /**
   * Serializes field changes for a bundle using UPDATE BUNDLE SET syntax
   */
  private serializeFieldChanges(bundleName: string, changes: Array<any>): string[] {
    if (changes.length === 0) {
      return [];
    }

    // Group all changes into a single UPDATE BUNDLE SET command
    const fieldModifications: string[] = [];

    for (const change of changes) {
      switch (change.type) {
        case 'add':
          if (change.newField) {
            fieldModifications.push(this.serializeFieldAdd(change.newField));
          }
          break;
        case 'remove':
          fieldModifications.push(this.serializeFieldRemove(change.fieldName));
          break;
        case 'modify':
          if (change.newField) {
            fieldModifications.push(this.serializeFieldModify(change.fieldName, change.newField));
          }
          break;
      }
    }

    if (fieldModifications.length === 0) {
      return [];
    }

    // Combine all modifications into a single UPDATE BUNDLE SET command
    return [`UPDATE BUNDLE "${bundleName}"\nSET (\n    ${fieldModifications.join(',\n    ')}\n);`];
  }

  /**
   * Serializes an ADD field modification
   * Format: {ADD "fieldName" = "fieldName", "type", required, unique, defaultValue}
   */
  private serializeFieldAdd(field: FieldDefinition): string {
    const defaultValue = this.serializeDefaultValue(field);
    return `{ADD "${field.name}" = "${field.name}", "${field.type}", ${field.required ? 'TRUE' : 'FALSE'}, ${field.unique ? 'TRUE' : 'FALSE'}, ${defaultValue}}`;
  }

  /**
   * Serializes a MODIFY field modification
   * Format: {MODIFY "oldName" = "newName", "type", required, unique, defaultValue}
   */
  private serializeFieldModify(oldName: string, newField: FieldDefinition): string {
    const defaultValue = this.serializeDefaultValue(newField);
    return `{MODIFY "${oldName}" = "${newField.name}", "${newField.type}", ${newField.required ? 'TRUE' : 'FALSE'}, ${newField.unique ? 'TRUE' : 'FALSE'}, ${defaultValue}}`;
  }

  /**
   * Serializes a REMOVE field modification
   * Format: {REMOVE "fieldName" = "", "", FALSE, FALSE, NULL}
   */
  private serializeFieldRemove(fieldName: string): string {
    return `{REMOVE "${fieldName}" = "", "", FALSE, FALSE, NULL}`;
  }

  /**
   * Serializes a default value for field definitions
   */
  private serializeDefaultValue(field: FieldDefinition): string {
    if (field.defaultValue === undefined || field.defaultValue === null) {
      return 'NULL';
    }

    // Handle different types appropriately
    switch (field.type.toLowerCase()) {
      case 'string':
      case 'text':
        return `"${field.defaultValue}"`;
      case 'int':
      case 'float':
        return String(field.defaultValue);
      case 'boolean':
      case 'bool':
        return field.defaultValue ? 'TRUE' : 'FALSE';
      case 'datetime':
        return `"${field.defaultValue}"`;
      case 'json':
        return typeof field.defaultValue === 'string' 
          ? `"${field.defaultValue}"` 
          : JSON.stringify(field.defaultValue);
      default:
        return 'NULL';
    }
  }

  /**
   * Serializes index changes for a bundle
   */
  private serializeIndexChangesForBundle(bundleName: string, changes: Array<any>): string[] {
    const commands: string[] = [];

    for (const change of changes) {
      switch (change.type) {
        case 'create':
          if (change.index) {
            commands.push(this.serializeCreateIndexForBundle(bundleName, change.index));
          }
          break;
        case 'delete':
          if (change.index) {
            commands.push(this.serializeDropIndexForBundle(bundleName, change.index.fieldName));
          }
          break;
      }
    }

    return commands;
  }

  /**
   * Serializes a CREATE INDEX command for a specific bundle
   * Format: CREATE HASH INDEX "name" ON BUNDLE "bundle" WITH FIELDS ({"field", required, unique})
   * Or: CREATE B-INDEX "name" ON BUNDLE "bundle" WITH FIELDS ({"field", required, unique})
   */
  private serializeCreateIndexForBundle(bundleName: string, index: IndexDefinition): string {
    // Determine index type - hash or btree
    const indexCommand = index.type.toLowerCase() === 'btree' ? 'B-INDEX' : 'HASH INDEX';
    
    // Index names should be descriptive
    const indexName = `idx_${bundleName.toLowerCase()}_${index.fieldName.toLowerCase()}`;
    
    // Field definition: {"fieldName", required, unique}
    // For now, we'll assume indexed fields are not required and not unique unless it's a unique index
    const required = false;
    const unique = false;
    
    return `CREATE ${indexCommand} "${indexName}" ON BUNDLE "${bundleName}"\nWITH FIELDS (\n    {"${index.fieldName}", ${required ? 'true' : 'false'}, ${unique ? 'true' : 'false'}}\n)`;
  }

  /**
   * Serializes a DROP INDEX command for a bundle
   */
  private serializeDropIndexForBundle(bundleName: string, fieldName: string): string {
    const indexName = `idx_${bundleName.toLowerCase()}_${fieldName.toLowerCase()}`;
    return `DROP INDEX "${indexName}";`;
  }
}
