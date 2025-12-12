import { createMigrationClient } from '../migration-client';
import type { BundleDefinition } from '../schema/SchemaDefinition';

/**
 * Integration tests for SyndrDBMigrationClient
 * Note: These tests require a running SyndrDB instance
 * Skip these tests if no server is available
 */
describe('SyndrDBMigrationClient Integration', () => {
  const connectionString = process.env.SYNDRDB_TEST_CONNECTION || 'syndrdb://localhost:7777:testdb:admin:password;';
  let client: ReturnType<typeof createMigrationClient>;

  beforeAll(async () => {
    // Skip tests if no test server configured
    if (!process.env.SYNDRDB_TEST_CONNECTION) {
      console.warn('Skipping integration tests - set SYNDRDB_TEST_CONNECTION to run');
      return;
    }

    client = createMigrationClient();
    await client.connect(connectionString);
  });

  afterAll(async () => {
    if (client) {
      await client.close();
    }
  });

  describe('syncSchema', () => {
    it('should create migration for new bundle', async () => {
      if (!process.env.SYNDRDB_TEST_CONNECTION) {
        return;
      }

      const schema: BundleDefinition[] = [
        {
          name: 'TestUser',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'email', type: 'STRING', required: true, unique: true },
            { name: 'name', type: 'STRING', required: true, unique: false }
          ],
          indexes: [
            { fieldName: 'email', type: 'hash' }
          ]
        }
      ];

      const migration = await client.syncSchema(schema, {
        description: 'Test migration - create TestUser bundle',
        autoApply: false
      });

      expect(migration).not.toBeNull();
      expect(migration!.description).toContain('Test migration');
      expect(migration!.status).toBe('PENDING');
    });

    it('should detect no changes when schema matches', async () => {
      if (!process.env.SYNDRDB_TEST_CONNECTION) {
        return;
      }

      // First sync
      const schema: BundleDefinition[] = [
        {
          name: 'TestProduct',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true }
          ],
          indexes: []
        }
      ];

      await client.syncSchema(schema, { autoApply: true });

      // Second sync with same schema
      const migration = await client.syncSchema(schema);

      expect(migration).toBeNull();
    });
  });

  describe('migration workflow', () => {
    it('should complete full migration workflow', async () => {
      if (!process.env.SYNDRDB_TEST_CONNECTION) {
        return;
      }

      // 1. Create schema
      const schema: BundleDefinition[] = [
        {
          name: 'TestOrder',
          fields: [
            { name: 'id', type: 'STRING', required: true, unique: true },
            { name: 'amount', type: 'FLOAT', required: true, unique: false }
          ],
          indexes: []
        }
      ];

      // 2. Sync schema
      const migration = await client.syncSchema(schema, {
        description: 'Create TestOrder bundle'
      });

      expect(migration).not.toBeNull();
      const version = migration!.version;

      // 3. Validate migration
      const validation = await client.validateMigration(version);

      expect(validation.hasErrors).toBe(false);

      // 4. Apply migration
      const result = await client.applyMigration(version);

      expect(result.version).toBe(version);
      expect(result.message).toBeTruthy();

      // 5. Check migration status
      const { migrations } = await client.showMigrations();
      const appliedMigration = migrations.find(m => m.version === version);

      expect(appliedMigration).toBeDefined();
      expect(appliedMigration!.status).toBe('APPLIED');
    });
  });

  describe('type generation', () => {
    it('should generate TypeScript types', async () => {
      if (!process.env.SYNDRDB_TEST_CONNECTION) {
        return;
      }

      const files = await client.generateTypes({
        includeJSDoc: true,
        generateTypeGuards: true
      });

      expect(files.length).toBeGreaterThan(0);
      expect(files.some(f => f.endsWith('.d.ts'))).toBe(true);
    });
  });

  describe('error handling', () => {
    it('should throw error for invalid migration version', async () => {
      if (!process.env.SYNDRDB_TEST_CONNECTION) {
        return;
      }

      await expect(client.validateMigration(99999)).rejects.toThrow();
    });

    it('should throw error when applying migration without validation', async () => {
      if (!process.env.SYNDRDB_TEST_CONNECTION) {
        return;
      }

      // Create invalid migration (if possible) and test force flag
      // This depends on server implementation
    });
  });
});

describe('SyndrDBMigrationClient Unit Tests', () => {
  let client: ReturnType<typeof createMigrationClient>;

  beforeEach(() => {
    client = createMigrationClient();
  });

  describe('initialization', () => {
    it('should create client instance', () => {
      expect(client).toBeDefined();
      expect(client.syncSchema).toBeDefined();
      expect(client.showMigrations).toBeDefined();
      expect(client.validateMigration).toBeDefined();
      expect(client.applyMigration).toBeDefined();
      expect(client.generateTypes).toBeDefined();
    });

    it('should throw error when not connected', async () => {
      const schema: BundleDefinition[] = [];

      await expect(client.syncSchema(schema)).rejects.toThrow('Not connected');
    });
  });

  describe('type mapping', () => {
    it('should map query response without auto-mapping', () => {
      const response = {
        Result: [
          { id: '1', name: 'Test' },
          { id: '2', name: 'Test2' }
        ],
        ResultCount: 2,
        ExecutionTimeMS: 10
      };

      const mapped = client.mapQueryResponse('User', response);

      expect(mapped.results).toHaveLength(2);
      expect(mapped.count).toBe(2);
      expect(mapped.executionTime).toBe(10);
    });

    it('should enable auto-mapping', () => {
      expect(() => client.enableAutoTypeMapping(true)).not.toThrow();
      expect(() => client.enableAutoTypeMapping(false)).not.toThrow();
    });
  });
});
