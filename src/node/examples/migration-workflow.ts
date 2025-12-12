/**
 * Example: Complete Migration Workflow
 * 
 * This example demonstrates a full migration workflow including:
 * - Initial schema creation
 * - Schema evolution
 * - Type generation
 * - Validation and rollback
 */

import { createMigrationClient } from '../migration-client';
import type { BundleDefinition } from '../schema/SchemaDefinition';

async function main() {
  // Create and connect client
  const client = createMigrationClient();
  await client.connect('syndrdb://localhost:7777:exampledb:admin:password;');

  console.log('Connected to SyndrDB');

  // ============================================================================
  // Step 1: Initial Schema Creation
  // ============================================================================
  
  console.log('\n=== Step 1: Creating Initial Schema ===');

  const initialSchema: BundleDefinition[] = [
    {
      name: 'User',
      fields: [
        { name: 'id', type: 'STRING', required: true, unique: true },
        { name: 'email', type: 'STRING', required: true, unique: true },
        { name: 'username', type: 'STRING', required: true, unique: true },
        { name: 'createdAt', type: 'DATETIME', required: true, unique: false }
      ],
      indexes: [
        { fieldName: 'email', type: 'hash' },
        { fieldName: 'username', type: 'hash' }
      ]
    },
    {
      name: 'Post',
      fields: [
        { name: 'id', type: 'STRING', required: true, unique: true },
        { name: 'title', type: 'STRING', required: true, unique: false },
        { name: 'content', type: 'TEXT', required: true, unique: false },
        { name: 'authorId', type: 'relationship', required: true, unique: false, relatedBundle: 'User' },
        { name: 'createdAt', type: 'DATETIME', required: true, unique: false }
      ],
      indexes: [
        { fieldName: 'authorId', type: 'hash' },
        { fieldName: 'createdAt', type: 'btree' }
      ]
    }
  ];

  let migration = await client.syncSchema(initialSchema, {
    description: 'Initial schema: User and Post bundles with indexes',
    autoApply: false
  });

  if (migration) {
    console.log(`Created migration version ${migration.version}`);
    console.log(`Description: ${migration.description}`);

    // Validate before applying
    const { formatted, hasErrors, hasWarnings } = await client.validateMigration(migration.version);
    
    console.log('\nValidation Report:');
    console.log(formatted);

    if (!hasErrors) {
      await client.applyMigration(migration.version);
      console.log('\n✓ Migration applied successfully');
    }
  }

  // ============================================================================
  // Step 2: Generate TypeScript Types
  // ============================================================================

  console.log('\n=== Step 2: Generating TypeScript Types ===');

  const typeFiles = await client.generateTypes({
    includeJSDoc: true,
    generateTypeGuards: true
  });

  console.log('Generated type files:');
  typeFiles.forEach(file => console.log(`  - ${file}`));

  // ============================================================================
  // Step 3: Schema Evolution - Add Comment Feature
  // ============================================================================

  console.log('\n=== Step 3: Evolving Schema - Adding Comments ===');

  const evolvedSchema: BundleDefinition[] = [
    ...initialSchema,
    {
      name: 'Comment',
      fields: [
        { name: 'id', type: 'STRING', required: true, unique: true },
        { name: 'content', type: 'TEXT', required: true, unique: false },
        { name: 'postId', type: 'relationship', required: true, unique: false, relatedBundle: 'Post' },
        { name: 'authorId', type: 'relationship', required: true, unique: false, relatedBundle: 'User' },
        { name: 'createdAt', type: 'DATETIME', required: true, unique: false }
      ],
      indexes: [
        { fieldName: 'postId', type: 'hash' },
        { fieldName: 'authorId', type: 'hash' }
      ]
    }
  ];

  // Get current version for conflict detection
  const currentVersion = await client.getCurrentVersion();
  console.log(`Current database version: ${currentVersion}`);

  migration = await client.syncSchema(evolvedSchema, {
    description: 'Add Comment bundle for post comments',
    expectedVersion: currentVersion, // Detect conflicts
    autoApply: false
  });

  if (migration) {
    console.log(`\nCreated migration version ${migration.version}`);
    
    const { hasErrors, hasWarnings } = await client.validateMigration(migration.version);
    
    if (!hasErrors) {
      await client.applyMigration(migration.version, hasWarnings);
      console.log('✓ Comment feature added successfully');
      
      // Regenerate types
      await client.generateTypes();
      console.log('✓ Types regenerated');
    }
  } else {
    console.log('No schema changes detected');
  }

  // ============================================================================
  // Step 4: Migration History
  // ============================================================================

  console.log('\n=== Step 4: Migration History ===');

  const { currentVersion: dbVersion, migrations } = await client.showMigrations();
  
  console.log(`\nDatabase Version: ${dbVersion}`);
  console.log('\nMigration History:');
  
  migrations.forEach(m => {
    const status = m.status === 'APPLIED' ? '✓' : '○';
    const appliedAt = m.appliedAt ? ` (applied: ${new Date(m.appliedAt).toLocaleString()})` : '';
    console.log(`  ${status} v${m.version}: ${m.description}${appliedAt}`);
  });

  // ============================================================================
  // Step 5: Using Generated Types
  // ============================================================================

  console.log('\n=== Step 5: Using Auto Type Mapping ===');

  // Enable auto-mapping
  client.enableAutoTypeMapping(true);

  // Example query (would need real data)
  try {
    const response = await client.query('SELECT * FROM User LIMIT 5');
    const { results, count, executionTime } = client.mapQueryResponse<any>('User', response);
    
    console.log(`\nQueried ${count} users in ${executionTime}ms`);
    
    results.forEach((user: any) => {
      // createdAt is automatically converted to Date object
      console.log(`  - ${user.username} (created: ${user.createdAt instanceof Date ? user.createdAt.toLocaleDateString() : user.createdAt})`);
    });
  } catch (error) {
    console.log('(Skipping query - no data in database yet)');
  }

  // ============================================================================
  // Step 6: Rollback Example (Validation Only)
  // ============================================================================

  console.log('\n=== Step 6: Rollback Validation ===');

  if (dbVersion > 1) {
    const targetVersion = dbVersion - 1;
    console.log(`\nValidating rollback from v${dbVersion} to v${targetVersion}...`);
    
    const rollbackValidation = await client.validateRollback(targetVersion);
    
    if (rollbackValidation.canRollback) {
      console.log('✓ Rollback is possible');
      console.log('\nData Impact:');
      console.log(`  Bundles to drop: ${rollbackValidation.dataImpact.bundlesDropped.join(', ') || 'none'}`);
      console.log(`  Fields to remove: ${rollbackValidation.dataImpact.fieldsRemoved.join(', ') || 'none'}`);
      console.log(`  Estimated data loss: ${rollbackValidation.dataImpact.estimatedDataLoss}`);
      
      if (rollbackValidation.warnings && rollbackValidation.warnings.length > 0) {
        console.log('\nWarnings:');
        rollbackValidation.warnings.forEach(w => console.log(`  ⚠ ${w}`));
      }

      // Note: Not actually applying rollback in example
      console.log('\n(Not applying rollback in this example)');
    } else {
      console.log('✗ Rollback not possible');
      if (rollbackValidation.errors) {
        console.log('Errors:');
        rollbackValidation.errors.forEach(e => console.log(`  - ${e}`));
      }
    }
  }

  // ============================================================================
  // Cleanup
  // ============================================================================

  await client.close();
  console.log('\n=== Example Complete ===');
}

// Run example
if (require.main === module) {
  main().catch(error => {
    console.error('Example failed:', error);
    process.exit(1);
  });
}

export { main };
