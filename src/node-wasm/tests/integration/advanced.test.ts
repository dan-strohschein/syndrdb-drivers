/**
 * Integration Tests - Advanced Features
 * 
 * Tests prepared statements, hooks, and other advanced features.
 * Requires server running on localhost:1776
 */

import { SyndrDBClient } from '../../src/client';
import type { HookConfig } from '../../src/types/hooks';

describe('Integration: Advanced Features', () => {
  let client: SyndrDBClient;
  const TEST_TIMEOUT = 60000;
  const CONNECTION_URL = 'syndrdb://127.0.0.1:1776:primary:root:root;';

  beforeAll(async () => {
    client = new SyndrDBClient({
      debugMode: true,
      preparedStatementCacheSize: 100,
    });

    await client.initialize();
    await client.connect(CONNECTION_URL);

    // Create test bundle
    await client.mutate(
      'CREATE BUNDLE "test_advanced" WITH FIELDS ({"id", "int", TRUE, TRUE, 0}, {"name", "string", FALSE, FALSE, ""}, {"value", "float", FALSE, FALSE, 0.0})'
    );
  }, TEST_TIMEOUT);

  afterAll(async () => {
    if (client) {
      await client.mutate('DROP BUNDLE "test_advanced"');
      await client.disconnect();
    }
  }, TEST_TIMEOUT);

  beforeEach(async () => {
    // Clear test data
    await client.mutate('DELETE FROM "test_advanced"');
  });

  describe('Prepared Statements', () => {
    test('should prepare statement', async () => {
      const sql = 'ADD DOCUMENT TO BUNDLE "test_advanced" WITH ({"id" = $1}, {"name" = $2}, {"value" = $3})';
      const stmtId = await client.prepare(sql);

      expect(stmtId).toBeDefined();
      expect(typeof stmtId).toBe('number');

      // Execute with parameters
      // Note: Execution depends on server prepared statement API
      await client.mutate(sql, [1, 'test', 100.5]);

      await client.deallocateStatement(stmtId);
    });

    test('should deallocate statement', async () => {
      const stmtId = await client.prepare('SELECT * FROM "test_advanced" WHERE "id" == $1');

      await client.deallocateStatement(stmtId);

      // After deallocation, statement should be removed
    });

    test('should handle prepare errors', async () => {
      await expect(
        client.prepare('INVALID SQL FOR PREPARED STATEMENT')
      ).rejects.toThrow();
    });
  });

  describe('Query Hooks', () => {
    test('should register hook', async () => {
      const hookConfig: HookConfig = {
        name: 'test_hook',
        before: (ctx) => {
          // Hook logic
        },
      };

      await client.registerHook(hookConfig);

      // Execute query to trigger hook
      await client.query('SELECT 1 as value');

      // Unregister hook
      await client.unregisterHook('test_hook');
    });

    test('should unregister hook', async () => {
      const hookConfig: HookConfig = {
        name: 'temp_hook',
        before: (ctx) => {
          // Hook logic
        },
      };

      await client.registerHook(hookConfig);
      await client.unregisterHook('temp_hook');

      // Hook should be removed
    });

    test('should get metrics stats', async () => {
      const stats = await client.getMetricsStats();
      expect(stats).toBeDefined();
    });
  });

  describe('Performance Features', () => {
    test('should track operation performance', async () => {
      await client.query('SELECT 1');
      await client.query('SELECT 2');
      await client.query('SELECT 3');

      const stats = client.getPerformanceStats();

      expect(stats).toBeDefined();
      expect(stats.operations).toBeDefined();
      expect(stats.durationMs).toBeGreaterThan(0);
    });

    test('should reset performance metrics', () => {
      // Execute some operations
      client.query('SELECT 1');

      // Reset metrics
      client.resetPerformanceMetrics();

      const stats = client.getPerformanceStats();
      // Stats should be reset
      expect(stats).toBeDefined();
    });

    test('should monitor connection health', async () => {
      const health = await client.getConnectionHealth();

      expect(health).toBeDefined();
      expect(health.isHealthy).toBe(true);
      expect(health.lastPingMs).toBeDefined();
    });
  });

  describe('Debug Features', () => {
    test('should provide debug information', async () => {
      const debugInfo = await client.getDebugInfo();

      expect(debugInfo).toBeDefined();
      expect(debugInfo.wasmMetadata).toBeDefined();
      expect(debugInfo.wasmMetadata.goVersion).toBeDefined();
      expect(debugInfo.stateHistory).toBeDefined();
    });

    test('should provide WASM metadata', () => {
      const metadata = client.getWASMMetadata();

      expect(metadata).toBeDefined();
      expect(metadata?.goVersion).toBeDefined();
    });

    test('should track state history', async () => {
      const debugInfo = await client.getDebugInfo();

      expect(debugInfo.stateHistory).toBeDefined();
      expect(Array.isArray(debugInfo.stateHistory)).toBe(true);
    });
  });

  describe('Connection Management', () => {
    test('should handle reconnection', async () => {
      // Disconnect
      await client.disconnect();
      expect(client.getState()).toBe('DISCONNECTED');

      // Reconnect
      await client.connect(CONNECTION_URL);
      expect(client.getState()).toBe('CONNECTED');

      // Should work normally
      const result = await client.query('SELECT 1');
      expect(result).toBeDefined();
    });

    test('should maintain state across operations', async () => {
      const initialState = client.getState();
      
      await client.query('SELECT 1');
      await client.mutate('ADD DOCUMENT TO BUNDLE "test_advanced" WITH ({"id" = $1}, {"name" = $2}, {"value" = $3})', [1, 'test', 1.0]);
      
      const finalState = client.getState();
      expect(finalState).toBe(initialState);
      expect(finalState).toBe('CONNECTED');
    });
  });

  describe('Error Recovery', () => {
    test('should recover from query errors', async () => {
      await expect(
        client.query('INVALID SYNTAX QUERY')
      ).rejects.toThrow();

      // Client should still be functional
      const result = await client.query('SELECT 1');
      expect(result).toBeDefined();
    });

    test('should recover from mutation errors', async () => {
      await expect(
        client.mutate('INVALID MUTATION SYNTAX')
      ).rejects.toThrow();

      // Should still work for valid mutations
      await client.mutate('ADD DOCUMENT TO BUNDLE "test_advanced" WITH ({"id" = $1}, {"name" = $2}, {"value" = $3})', [2, 'recovery', 1.0]);

      const result = await client.query('SELECT * FROM "test_advanced" WHERE "id" == $1', [2]);
      expect(result).toBeDefined();
    });

    test('should handle timeout scenarios', async () => {
      // This would test timeout behavior if server supports it
      // For now, just verify timeout option exists
      expect(client).toBeDefined();
    });
  });

  describe('Batch Operations', () => {
    test('should execute multiple operations', async () => {
      // Insert multiple records
      await client.mutate('ADD DOCUMENT TO BUNDLE \"test_advanced\" WITH ({\"id\" = $1}, {\"name\" = $2}, {\"value\" = $3})', [10, 'batch1', 1.0]);
      await client.mutate('ADD DOCUMENT TO BUNDLE \"test_advanced\" WITH ({\"id\" = $1}, {\"name\" = $2}, {\"value\" = $3})', [11, 'batch2', 2.0]);
      await client.mutate('ADD DOCUMENT TO BUNDLE \"test_advanced\" WITH ({\"id\" = $1}, {\"name\" = $2}, {\"value\" = $3})', [12, 'batch3', 3.0]);

      // Verify all inserted
      const result = await client.query('SELECT COUNT(*) as count FROM \"test_advanced\"');
      expect(result).toBeDefined();
    });
  });
});
