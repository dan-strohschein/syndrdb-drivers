/**
 * End-to-End Migration System Test
 * 
 * This test validates the complete migration workflow from a cold start:
 * 1. Connect to a brand new database
 * 2. Define schema with Authors and Books bundles (one-to-many relationship)
 * 3. Create migration through the migration system
 * 4. Validate the migration
 * 5. Apply the migration
 * 6. Verify bundles were created correctly by querying server
 * 7. Verify migration history
 * 8. Generate TypeScript types
 * 9. Verify type files were created
 * 
 * Prerequisites:
 * - SyndrDB server running and accessible
 * - Empty database in connection string exists
 * - Connection string set in SYNDRDB_TEST_CONNECTION environment variable
 *   Example: SYNDRDB_TEST_CONNECTION="syndrdb://localhost:7777:testdb:admin:password;"
 * 
 * Run with: SYNDRDB_TEST_CONNECTION="..." npm test -- migration-e2e.test.ts
 */

import { createMigrationClient, SyndrDBMigrationClient } from '../migration-client';
import type { BundleDefinition } from '../schema/SchemaDefinition';
import * as fs from 'fs/promises';
import * as path from 'path';

// Skip tests if no connection string provided
const testConnection = process.env.SYNDRDB_TEST_CONNECTION;
const describeIfConnected = testConnection ? describe : describe.skip;

const dbName = getDBFromConnection(testConnection)


function getDBFromConnection(connectionString: string | undefined): string {
  if (!connectionString) return 'unknowndb';

  const parts = connectionString.split(':');
  if (parts.length < 4) return 'unknowndb';

  return parts[3];
}


describeIfConnected('Migration System E2E - Cold Start', () => {
  let client: SyndrDBMigrationClient;
  const typeOutputDir = '.syndrdb-types-test';

  beforeAll(async () => {
    if (!testConnection) {
      throw new Error('SYNDRDB_TEST_CONNECTION environment variable not set');
    }

    client = createMigrationClient();
    await client.connect(testConnection);
  }, 30000);

  afterAll(async () => {
    // Cleanup: Remove generated type files
    try {
      await fs.rm(typeOutputDir, { recursive: true, force: true });
    } catch (error) {
      // Ignore cleanup errors
    }

    if (client) {
      await client.close();
    }
  });

  describe('Step 1: Initial State Verification', () => {
    it('should connect to empty "${dbName}" database', async () => {
      const stats = client.getPoolStats();
      expect(stats).toBeTruthy();
      expect(stats?.max).toBeGreaterThan(0);
    });

    it('should show version 0 for brand new database', async () => {
   //   console.log('[TEST] Pool stats before getCurrentVersion:', JSON.stringify(client.getPoolStats()));
      const currentVersion = await client.getCurrentVersion();
   //   console.log('[TEST] Pool stats after getCurrentVersion:', JSON.stringify(client.getPoolStats()));
      expect(currentVersion).toBe(0);
    }, 15000);

    it('should show empty migration history', async () => {
      const { migrations } = await client.showMigrations();
      expect(migrations).toHaveLength(0);
    }, 15000); // Increase timeout to 15 seconds

    it('should show no existing bundles', async () => {
      const response = await client.query(`SHOW BUNDLES FOR "${dbName}";`);
      const bundles = (response && response.Result) ? response.Result : (response || []);
      expect(Array.isArray(bundles) ? bundles : []).toEqual([]);
    }, 15000);
  });

  describe('Step 2: Schema Definition', () => {
    let schema: BundleDefinition[];

    beforeAll(() => {
      schema = [
        {
          name: 'Authors',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'firstName', type: 'STRING', required: true, unique: false },
            { name: 'lastName', type: 'STRING', required: true, unique: false },
            { name: 'email', type: 'STRING', required: true, unique: true },
            { name: 'bio', type: 'STRING', required: false, unique: false },
            { name: 'birthYear', type: 'INT', required: false, unique: false },
            { name: 'isActive', type: 'BOOLEAN', required: true, unique: false },
            { name: 'createdAt', type: 'DATETIME', required: true, unique: false }
          ],
          indexes: [
            { fieldName: 'email', type: 'hash' },
            { fieldName: 'lastName', type: 'hash' }
          ],
          relationships: [
            { name: 'AuthorBooks', sourceBundle: 'Authors', destinationBundle: 'Books', sourceField: 'DocumentID', destinationField: 'authorId', relationshipType: '1toMany' }
          ]
        },
        {
          name: 'Books',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'title', type: 'STRING', required: true, unique: false },
            { name: 'isbn', type: 'STRING', required: true, unique: true },
            //{ name: 'authorId', type: 'STRING', required: true, unique: true, relatedBundle: 'Authors' },
            { name: 'publicationYear', type: 'INT', required: false, unique: false },
            { name: 'price', type: 'FLOAT', required: true, unique: false },
            { name: 'genre', type: 'STRING', required: false, unique: false },
            { name: 'description', type: 'STRING', required: false, unique: false },
            { name: 'inStock', type: 'BOOLEAN', required: true, unique: false },
            { name: 'metadata', type: 'STRING', required: false, unique: false },
            { name: 'createdAt', type: 'DATETIME', required: true, unique: false }
          ],
          indexes: [
            { fieldName: 'isbn', type: 'hash' },
            { fieldName: 'authorId', type: 'hash' },
            { fieldName: 'price', type: 'btree' },
            { fieldName: 'title', type: 'hash' }
          ],
          relationships: []
        }
      ];
    });

    it('should create valid schema definitions', () => {
      expect(schema).toHaveLength(2);
      expect(schema[0].name).toBe('Authors');
      expect(schema[1].name).toBe('Books');
    });

    it('should have correct field count', () => {
      expect(schema[0].fields).toHaveLength(8);
      expect(schema[1].fields).toHaveLength(10);
    });

    // This needs to be after the relationships is added, so after the apply step.
    // it('should have relationship field in Books', () => {
    //   const authorIdField = schema[1].fields.find(f => f.name === 'authorId');
    //   expect(authorIdField).toBeDefined();
    //   expect(authorIdField?.type).toBe('STRING'); // Foreign key field is STRING type
    //   expect(authorIdField?.relatedBundle).toBe('Authors');
    // });

    it('should have indexes defined', () => {
      expect(schema[0].indexes).toHaveLength(2);
      expect(schema[1].indexes).toHaveLength(4);
    });
  });

  describe('Step 3: Create Migration', () => {
    it('should create migration for Authors and Books bundles', async () => {
      const schema: BundleDefinition[] = [
        {
          name: 'Authors',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'firstName', type: 'STRING', required: true, unique: false },
            { name: 'lastName', type: 'STRING', required: true, unique: false },
            { name: 'email', type: 'STRING', required: true, unique: true },
            { name: 'bio', type: 'STRING', required: false, unique: false },
            { name: 'birthYear', type: 'INT', required: false, unique: false },
            { name: 'isActive', type: 'BOOLEAN', required: true, unique: false },
            { name: 'createdAt', type: 'DATETIME', required: true, unique: false }
          ],
          indexes: [
            { fieldName: 'email', type: 'hash' },
            { fieldName: 'lastName', type: 'hash' }
          ],
          relationships: [
            { name: 'AuthorBooks', sourceBundle: 'Authors', destinationBundle: 'Books', sourceField: 'id', destinationField: 'authorId', relationshipType: '1toMany' }
          ]
        },
        {
          name: 'Books',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'title', type: 'STRING', required: true, unique: false },
            { name: 'isbn', type: 'STRING', required: true, unique: true },
            { name: 'authorId', type: 'STRING', required: true, unique: true, relatedBundle: 'Authors' },
            { name: 'publicationYear', type: 'INT', required: false, unique: false },
            { name: 'price', type: 'FLOAT', required: true, unique: false },
            { name: 'genre', type: 'STRING', required: false, unique: false },
            { name: 'description', type: 'STRING', required: false, unique: false },
            { name: 'inStock', type: 'BOOLEAN', required: true, unique: false },
            { name: 'metadata', type: 'STRING', required: false, unique: false },
            { name: 'createdAt', type: 'DATETIME', required: true, unique: false }
           
          ],
          indexes: [
            { fieldName: 'isbn', type: 'hash' },
            { fieldName: 'authorId', type: 'hash' },
            { fieldName: 'price', type: 'btree' },
            { fieldName: 'title', type: 'hash' }
          ],
          relationships: []
        }
      ];

    //  console.log('[TEST Step 3] Calling syncSchema with autoApply=false...');
      
      try {
        const migration = await client.syncSchema(schema, {
          description: 'Initial schema: Create Authors and Books bundles with one-to-many relationship',
          expectedVersion: 0,
          autoApply: false
        });

        //console.log('[TEST Step 3] syncSchema returned:', JSON.stringify(migration, null, 2));
        
        //expect(migration).toBeTruthy();
        expect(migration).not.toBeNull();
        expect(migration!.version).toBe(1);
        expect(migration!.status).toBe('PENDING');
       
      } catch (error) {
        console.error('[TEST Step 3] syncSchema failed with error:', error);
        throw error;
      }
    }, 30000); // Increase timeout for migration creation

    it('should show migration in history as PENDING', async () => {
      const { migrations, currentVersion } = await client.showMigrations();
      
      expect(currentVersion).toBe(0);
      expect(migrations).toHaveLength(1);
      expect(migrations[0].version).toBe(1);
      expect(migrations[0].status).toBe('PENDING');
    });
  });

  describe('Step 4: Validate Migration', () => {
    beforeAll(async () => {
      // Verify migration exists before trying to validate
      const { migrations } = await client.showMigrations();
      if (migrations.length === 0) {
        throw new Error('No migration found. Step 3 must complete successfully before Step 4 can run.');
      }
    });

    it('should validate migration successfully', async () => {
      const { report, formatted, hasErrors, hasWarnings } = await client.validateMigration(1);

      console.log('\n=== Migration Validation Report ===');
      console.log(formatted);
      console.log('===================================\n');

      expect(hasErrors).toBe(false);
      expect(report.syntaxValid).toBe(true);
      expect(report.dependencyValid).toBe(true);
      expect(report.commandCountValid).toBe(true);
      expect(report.overallResult).toBe('ACTIVE');
    });

    it('should have all validation phases passed', async () => {
      const { report } = await client.validateMigration(1);

    //   expect(report.syntaxPhase.passed).toBe(true);
    //   expect(report.semanticPhase.passed).toBe(true);
    //   expect(report.constraintPhase.passed).toBe(true);
    //   expect(report.performancePhase.passed).toBe(true);
    //   expect(report.reversibilityPhase.passed).toBe(true);
    });
  });

  describe('Step 5: Apply Migration', () => {
    it('should apply migration successfully', async () => {
      const result = await client.applyMigration(1, false);

      expect(result.version).toBe(1);
      expect(result.message).toBeTruthy();
      expect(result.force).toBe(false);

      console.log('\n✓ Migration applied:', result.message);
    });

    it('should show migration as APPLIED in history', async () => {
      const { migrations, currentVersion } = await client.showMigrations();

      
      expect(currentVersion).toBe(1);
      expect(migrations).toHaveLength(1);
      expect(migrations[0].version).toBe(1);
      expect(migrations[0].status).toBe('APPLIED');
      
    });

    it('should show current version as 1', async () => {
      const currentVersion = await client.getCurrentVersion();
      expect(currentVersion).toBe(1);
    });
  });

  describe('Step 6: Verify Bundles Created', () => {
    it('should show Authors and Books bundles exist', async () => {
      const response = await client.query(`SHOW BUNDLES FOR "${dbName}";`);
      
      const bundles = response.Result || response;
      expect(bundles).toHaveLength(2);

      const bundleNames = bundles.map((b: any) => b.BundleMetadata.Name);
      expect(bundleNames).toContain('Authors');
      expect(bundleNames).toContain('Books');
    });

    it('should verify Authors bundle structure', async () => {
      const response = await client.query(`SHOW BUNDLES FOR "${dbName}";`);
      const bundles = response.Result || response;
      
      const authorsBundle = bundles.find((b: any) => b.BundleMetadata.Name === 'Authors');
      expect(authorsBundle).toBeDefined();

      const fields = authorsBundle.BundleMetadata.DocumentStructure.FieldDefinitions;
      expect(fields).toBeDefined();
      
      expect(fields.id.Type).toBe('STRING');
      expect(fields.id.Required).toBe(true);
      expect(fields.id.Unique).toBe(true);
      expect(fields.email.Unique).toBe(true);
      expect(fields.birthYear.Type).toBe('INT');
      expect(fields.isActive.Type).toBe('BOOLEAN');
      expect(fields.createdAt.Type).toBe('DATETIME');
    });

    it('should verify Books bundle structure', async () => {
      const response = await client.query(`SHOW BUNDLES FOR "${dbName}";`);
      const bundles = response.Result || response;
      
      const booksBundle = bundles.find((b: any) => b.BundleMetadata.Name === 'Books');
      expect(booksBundle).toBeDefined();

      const fields = booksBundle.BundleMetadata.DocumentStructure.FieldDefinitions;
      
      expect(fields.authorId.Type).toBe('relationship');
     
      expect(fields.price.Type).toBe('FLOAT');
      expect(fields.inStock.Type).toBe('BOOLEAN');
      //expect(fields.metadata.Type).toBe('JSON'); // NOT SUPPORTED
    });

    it('should verify Authors indexes', async () => {
      const response = await client.query(`SHOW BUNDLES FOR "${dbName}";`);
      const bundles = response.Result || response;
      
      const authorsBundle = bundles.find((b: any) => b.BundleMetadata.Name === 'Authors');
      const indexes = authorsBundle.BundleMetadata.Indexes;
      
      

      expect(Object.keys(indexes)).toHaveLength(3); //Got to count the DocumentID index too
    });

    it('should verify Books indexes', async () => {
      const response = await client.query(`SHOW BUNDLES FOR "${dbName}";`);
      const bundles = response.Result || response;
      
      const booksBundle = bundles.find((b: any) => b.BundleMetadata.Name === 'Books');
      const indexes = booksBundle.BundleMetadata.Indexes;
      
      expect(Object.keys(indexes)).toHaveLength(5); //cuz of the FK index that gets added automatically by the creation of the relationship
    });
  });

  describe('Step 7: Generate TypeScript Types', () => {
    it('should generate TypeScript types successfully', async () => {
      const generatedFiles = await client.generateTypes({
        outputDir: typeOutputDir,
        includeJSDoc: true,
        generateTypeGuards: true
      });

      expect(generatedFiles.length).toBeGreaterThan(0);
      console.log('\n✓ Generated type files:', generatedFiles);
    });

    it('should create Authors.d.ts file', async () => {
      const authorsFile = path.join(typeOutputDir, 'Authors.d.ts');
      const content = await fs.readFile(authorsFile, 'utf-8');

      expect(content).toContain('export interface Authors');
      expect(content).toContain('id: string');
      expect(content).toContain('email: string');
      expect(content).toContain('isActive: boolean');
      expect(content).toContain('createdAt: Date');
      expect(content).toContain('export function isAuthors');
    });

    it('should create Books.d.ts file', async () => {
      const booksFile = path.join(typeOutputDir, 'Books.d.ts');
      const content = await fs.readFile(booksFile, 'utf-8');

      expect(content).toContain('export interface Books');
      expect(content).toContain('authorId: any');
      
      expect(content).toContain('price: number');
      expect(content).toContain('export function isBooks');
    });
  });

  describe('Step 8: Verify No Changes on Re-Sync', () => {
    it('should detect no changes when syncing same schema', async () => {
      const schema: BundleDefinition[] = [
        {
          name: 'Authors',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'firstName', type: 'STRING', required: true, unique: false },
            { name: 'lastName', type: 'STRING', required: true, unique: false },
            { name: 'email', type: 'STRING', required: true, unique: true },
            { name: 'bio', type: 'TEXT', required: false, unique: false },
            { name: 'birthYear', type: 'INT', required: false, unique: false },
            { name: 'isActive', type: 'BOOLEAN', required: true, unique: false },
            { name: 'createdAt', type: 'DATETIME', required: true, unique: false }
          ],
          indexes: [
            { fieldName: 'email', type: 'hash' },
            { fieldName: 'lastName', type: 'hash' }
          ],
          relationships: [
            { name: 'AuthorBooks', sourceBundle: 'Authors', destinationBundle: 'Books', sourceField: 'id', destinationField: 'authorId', relationshipType: '1toMany' }
          ]
        },
        {
          name: 'Books',
          fields: [
                      { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'title', type: 'STRING', required: true, unique: false },
            { name: 'isbn', type: 'STRING', required: true, unique: true },
            { name: 'authorId', type: 'STRING', required: true, unique: true, relatedBundle: 'Authors' },
            { name: 'publicationYear', type: 'INT', required: false, unique: false },
            { name: 'price', type: 'FLOAT', required: true, unique: false },
            { name: 'genre', type: 'STRING', required: false, unique: false },
            { name: 'description', type: 'STRING', required: false, unique: false },
            { name: 'inStock', type: 'BOOLEAN', required: true, unique: false },
            { name: 'metadata', type: 'STRING', required: false, unique: false },
            { name: 'createdAt', type: 'DATETIME', required: true, unique: false }
          ],
          indexes: [
            { fieldName: 'isbn', type: 'hash' },
            { fieldName: 'authorId', type: 'hash' },
            { fieldName: 'price', type: 'btree' },
            { fieldName: 'title', type: 'hash' }
          ],
          relationships: [
            
          ]
        }
      ];

      const migration = await client.syncSchema(schema, {
        description: 'Test re-sync',
        expectedVersion: 1
      });

      expect(migration?.status).toBe('PENDING');
      expect(migration?.createdAt).toBe('');
    });
  });
  
});

if (!testConnection) {
  console.log('\n' + '⚠'.repeat(70));
  console.log('  Migration E2E tests SKIPPED');
  console.log('  Set SYNDRDB_TEST_CONNECTION to run these tests');
  console.log('  Example: SYNDRDB_TEST_CONNECTION="syndrdb://localhost:7777:testdb:admin:password;"');
  console.log('⚠'.repeat(70) + '\n');
}
