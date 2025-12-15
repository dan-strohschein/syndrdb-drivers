/**
 * Integration Tests - Migration Operations
 * 
 * Tests migration planning, application, and rollback.
 * Requires server running on localhost:1776
 */

import { SyndrDBClient } from '../../src/client';
import type { SchemaDefinition } from '../../src/types/schema';

describe('Integration: Migration Operations', () => {
  let client: SyndrDBClient;
  const TEST_TIMEOUT = 60000;
  const CONNECTION_URL = 'syndrdb://127.0.0.1:1776:primary:root:root;';

  beforeAll(async () => {
    client = new SyndrDBClient({
      debugMode: true,
    });

    await client.initialize();
    await client.connect(CONNECTION_URL);
  }, TEST_TIMEOUT);

  afterAll(async () => {
    if (client) {
      await client.disconnect();
    }
  }, TEST_TIMEOUT);

  describe('Migration Planning', () => {
    test('should generate migration plan from schema', async () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'test_users',
            fields: [
              { name: 'id', type: 'STRING', required: true, unique: false },
              { name: 'username', type: 'STRING', required: true, unique: false },
              { name: 'email', type: 'STRING', required: true, unique: false },
            ],
            indexes: [],
          },
        ],
      };

      const plan = await client.planMigration(schema);
      
      expect(plan).toBeDefined();
      expect(plan.migration).toBeDefined();
      expect(plan.migration.id).toBeDefined();
    });

    test('should detect schema changes', async () => {
      const schema1: SchemaDefinition = {
        bundles: [
          {
            name: 'test_products',
            fields: [
              { name: 'id', type: 'STRING', required: true, unique: false },
              { name: 'name', type: 'STRING', required: true, unique: false },
            ],
            indexes: [],
          },
        ],
      };

      const plan1 = await client.planMigration(schema1);
      expect(plan1.migration).toBeDefined();
    });
  });

  describe('Migration Application', () => {
    test('should apply migration', async () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'test_orders',
            fields: [
              { name: 'id', type: 'STRING', required: true, unique: false },
              { name: 'customer_id', type: 'STRING', required: true, unique: false },
              { name: 'total', type: 'FLOAT', required: true, unique: false },
            ],
            indexes: [],
          },
        ],
      };

      const plan = await client.planMigration(schema);
      expect(plan.migration).toBeDefined();

      const migrationId = plan.migration.id;

      // Apply the migration
      const applied = await client.applyMigration(migrationId);
      expect(applied).toBeDefined();
      expect(applied.id).toBe(migrationId);
    });
  });

  describe('Migration History', () => {
    test('should retrieve migration history', async () => {
      const history = await client.getMigrationHistory();

      expect(history).toBeDefined();
      expect(history.migrations).toBeDefined();
      expect(Array.isArray(history.migrations)).toBe(true);
      
      if (history.migrations.length > 0) {
        const migration = history.migrations[0];
        expect(migration.id).toBeDefined();
        expect(migration.version).toBeDefined();
        expect(typeof migration.version).toBe('number');
      }
    });
  });

  describe('Migration Validation', () => {
    test('should validate migration', async () => {
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'test_validation',
            fields: [
              { name: 'id', type: 'STRING', required: true, unique: false },
            ],
            indexes: [],
          },
        ],
      };

      const plan = await client.planMigration(schema);
      const migrationId = plan.migration.id;

      const validation = await client.validateMigration(migrationId);
      expect(validation).toBeDefined();
      expect(validation.isValid).toBeDefined();
    });
  });

  describe('Migration Rollback', () => {
    test('should rollback migration by version', async () => {
      // First apply a migration
      const schema: SchemaDefinition = {
        bundles: [
          {
            name: 'test_rollback',
            fields: [
              { name: 'id', type: 'STRING', required: true, unique: false },
            ],
            indexes: [],
          },
        ],
      };

      const plan = await client.planMigration(schema);
      const applied = await client.applyMigration(plan.migration.id);

      // Rollback using version number
      const rolledBack = await client.rollbackMigration(applied.version);
      expect(rolledBack).toBeDefined();
      expect(rolledBack.version).toBe(applied.version);
    });
  });

  describe('Schema Generation', () => {
    test('should generate JSON schema', async () => {
      const jsonSchema = await client.generateJSONSchema('single');
      expect(jsonSchema).toBeDefined();
      expect(typeof jsonSchema).toBe('string');
      expect(jsonSchema.length).toBeGreaterThan(0);
    });

    test('should generate GraphQL schema', async () => {
      const graphqlSchema = await client.generateGraphQLSchema({});
      expect(graphqlSchema).toBeDefined();
      expect(typeof graphqlSchema).toBe('string');
      expect(graphqlSchema.length).toBeGreaterThan(0);
    });
  });
});
