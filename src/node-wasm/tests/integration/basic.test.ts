/**
 * Integration Tests - Basic Client Operations
 * 
 * Tests basic client functionality against a real SyndrDB server.
 * Requires server running on localhost:1776
 */

import { SyndrDBClient } from '../../src/client';
import { ConnectionError } from '../../src/errors';

describe('Integration: Basic Client Operations', () => {
  let client: SyndrDBClient;
  const TEST_TIMEOUT = 60000;
  const CONNECTION_URL = 'syndrdb://127.0.0.1:1776:primary:root:root;';

  beforeAll(async () => {
    client = new SyndrDBClient({
      debugMode: true,
      defaultTimeoutMs: 10000,
    });

    await client.initialize();
    await client.connect(CONNECTION_URL);
  }, TEST_TIMEOUT);

  afterAll(async () => {
    if (client) {
      await client.disconnect();
    }
  }, TEST_TIMEOUT);

  describe('Connection Management', () => {
    test('should be connected after initialization', () => {
      expect(client.getState()).toBe('CONNECTED');
    });

    test.skip('should ping server', async () => {
      const latency = await client.ping();
      expect(latency).toBeGreaterThan(0);
      expect(latency).toBeLessThan(5000); // Should be under 5 seconds
    });

    test.skip('should get connection health', async () => {
      const health = await client.getConnectionHealth();
      expect(health).toBeDefined();
      expect(health.isHealthy).toBe(true);
      expect(health.lastPingMs).toBeDefined();
    });
  });

  describe('Query Operations', () => {
    test('should execute simple query', async () => {
      const result = await client.query('SELECT 1 as value');
      expect(result).toBeDefined();
    });

    test('should execute query with parameters', async () => {
      const result = await client.query('SELECT $1 as value', [42]);
      expect(result).toBeDefined();
    });

    test.skip('should handle query errors', async () => {
      await expect(
        client.query('INVALID SQL QUERY')
      ).rejects.toThrow();
    });
  });

  describe('Mutation Operations', () => {
    beforeAll(async () => {
      // Clean up first
      try {
        await client.mutate('DELETE DOCUMENTS FROM "test_basic" CONFIRMED;');
        await client.mutate('DROP BUNDLE "test_basic" WITH FORCE;');
      } catch (e) {
        // Ignore errors if bundle doesn't exist
      }
      // Create test bundle
      await client.mutate('CREATE BUNDLE "test_basic" WITH FIELDS ({"id", "int", TRUE, FALSE, 0}, {"value", "string", FALSE, FALSE, ""})');
    }, TEST_TIMEOUT);

    afterAll(async () => {
      // Clean up after tests
      try {
        await client.mutate('DELETE DOCUMENTS FROM "test_basic" CONFIRMED;');
        await client.mutate('DROP BUNDLE "test_basic" WITH FORCE;');
      } catch (e) {
        // Ignore cleanup errors
      }
    }, TEST_TIMEOUT);

    beforeEach(async () => {
      // Clear data before each test
      await client.mutate('DELETE DOCUMENTS FROM "test_basic" CONFIRMED;');
    });

    test('should execute mutation', async () => {
      const result = await client.mutate(
        'ADD DOCUMENT TO BUNDLE "test_basic" WITH ({"id" = $1}, {"value" = $2})',
        [1, 'test']
      );
      expect(result).toBeDefined();
    });

    test('should read mutated data', async () => {
      await client.mutate('ADD DOCUMENT TO BUNDLE "test_basic" WITH ({"id" = $1}, {"value" = $2})', [1, 'test']);

      const result = await client.query('SELECT * FROM "test_basic" WHERE "id" == $1', [1]);
      expect(result).toBeDefined();
    });

    test.skip('should handle mutation errors', async () => {
      await expect(
        client.mutate('INVALID MUTATION')
      ).rejects.toThrow();
    });
  });

  describe('State Management', () => {
    test('should track state transitions', async () => {
      const initialState = client.getState();
      expect(initialState).toBe('CONNECTED');

      await client.disconnect();
      expect(client.getState()).toBe('DISCONNECTED');

      await client.connect(CONNECTION_URL);
      expect(client.getState()).toBe('CONNECTED');
    });

    test('should reject operations when disconnected', async () => {
      await client.disconnect();

      await expect(
        client.query('SELECT 1')
      ).rejects.toThrow(ConnectionError);

      // Reconnect for other tests
      await client.connect(CONNECTION_URL);
    });
  });

  describe('Performance Monitoring', () => {
    test('should track operation metrics', async () => {
      await client.query('SELECT 1');
      await client.query('SELECT 2');

      const stats = client.getPerformanceStats();
      expect(stats).toBeDefined();
      expect(stats.operations).toBeDefined();
      expect(stats.durationMs).toBeGreaterThan(0);
    });

    test.skip('should provide debug info', async () => {
      const debugInfo = await client.getDebugInfo();
      expect(debugInfo).toBeDefined();
      expect(debugInfo.wasmMetadata).toBeDefined();
      expect(debugInfo.stateHistory).toBeDefined();
    });
  });
});
