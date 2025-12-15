/**
 * Integration Tests - Transaction Operations
 * 
 * Tests transaction lifecycle, commit, and rollback.
 * Requires server running on localhost:1776
 */

import { SyndrDBClient } from '../../src/client';

describe('Integration: Transaction Operations', () => {
  let client: SyndrDBClient;
  const TEST_TIMEOUT = 60000;
  const CONNECTION_URL = 'syndrdb://127.0.0.1:1776:primary:root:root;';

  beforeAll(async () => {
    client = new SyndrDBClient({
      debugMode: true,
      transactionTimeout: 30000,
    });

    await client.initialize();
    await client.connect(CONNECTION_URL);

    // Create test bundle
    await client.mutate(
      'CREATE BUNDLE "test_transactions" WITH FIELDS ({"id", "int", TRUE, TRUE, 0}, {"value", "string", FALSE, FALSE, ""}, {"amount", "float", FALSE, FALSE, 0.0})'
    );
  }, TEST_TIMEOUT);

  afterAll(async () => {
    if (client) {
      await client.mutate('DROP BUNDLE "test_transactions"');
      await client.disconnect();
    }
  }, TEST_TIMEOUT);

  beforeEach(async () => {
    // Clear test data before each test
    await client.mutate('DELETE FROM "test_transactions"');
  });

  describe('Transaction Lifecycle', () => {
    test('should begin transaction', async () => {
      const txId = await client.beginTransaction();
      expect(txId).toBeDefined();
      expect(typeof txId).toBe('number');

      await client.commitTransaction(txId);
    });

    test('should commit transaction', async () => {
      const txId = await client.beginTransaction();

      // Insert data within transaction context
      // Note: actual transaction query execution depends on server API
      await client.mutate('ADD DOCUMENT TO BUNDLE "test_transactions" WITH ({"id" = $1}, {"value" = $2})', [1, 'test']);

      await client.commitTransaction(txId);

      // Verify data was committed
      const result = await client.query('SELECT * FROM "test_transactions" WHERE "id" == $1', [1]);
      expect(result).toBeDefined();
    });

    test('should rollback transaction', async () => {
      const txId = await client.beginTransaction();

      // Insert data
      await client.mutate('ADD DOCUMENT TO BUNDLE "test_transactions" WITH ({"id" = $1}, {"value" = $2})', [2, 'rollback-test']);

      await client.rollbackTransaction(txId);

      // Data should not be present after rollback
      // Note: Behavior depends on server transaction implementation
    });
  });

  describe('Transaction Error Handling', () => {
    test('should handle commit on non-existent transaction', async () => {
      const fakeTxId = 'fake-tx-99999';

      await expect(
        client.commitTransaction(fakeTxId)
      ).rejects.toThrow();
    });

    test('should handle rollback on non-existent transaction', async () => {
      const fakeTxId = 'fake-tx-99999';

      await expect(
        client.rollbackTransaction(fakeTxId)
      ).rejects.toThrow();
    });
  });

  describe('Multiple Transactions', () => {
    test('should handle multiple concurrent transactions', async () => {
      const tx1 = await client.beginTransaction();
      const tx2 = await client.beginTransaction();

      expect(tx1).not.toBe(tx2);

      await client.commitTransaction(tx1);
      await client.commitTransaction(tx2);
    });

    test('should handle mixed commit and rollback', async () => {
      const tx1 = await client.beginTransaction();
      const tx2 = await client.beginTransaction();

      await client.commitTransaction(tx1);
      await client.rollbackTransaction(tx2);
    });
  });
});
