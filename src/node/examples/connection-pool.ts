/**
 * Advanced example: Direct connection pool usage
 */

import {
  SyndrDBConnectionPool,
  SyndrDBConnection,
  parseConnectionString,
  ConnectionParams,
} from '../src/index';

async function demonstrateConnectionPool() {
  // Parse the connection string
  const connectionString = 'syndrdb://localhost:1776:mydb:admin:password;';
  const params: ConnectionParams = parseConnectionString(connectionString);

  console.log('Parsed connection parameters:', params);

  // Create a connection pool
  const pool = new SyndrDBConnectionPool(params, {
    maxConnections: 3,
    idleTimeout: 15000, // 15 seconds
  });

  console.log('Initial pool stats:', pool.getStats());

  try {
    // Acquire connections from the pool
    console.log('\n--- Acquiring connections ---');
    const conn1 = await pool.acquire();
    console.log('Acquired connection 1');
    console.log('Pool stats:', pool.getStats());

    const conn2 = await pool.acquire();
    console.log('Acquired connection 2');
    console.log('Pool stats:', pool.getStats());

    const conn3 = await pool.acquire();
    console.log('Acquired connection 3');
    console.log('Pool stats:', pool.getStats());

    // Release connections back to the pool
    console.log('\n--- Releasing connections ---');
    await pool.release(conn1);
    console.log('Released connection 1');
    console.log('Pool stats:', pool.getStats());

    await pool.release(conn2);
    console.log('Released connection 2');
    console.log('Pool stats:', pool.getStats());

    // Demonstrate transaction binding
    console.log('\n--- Transaction binding ---');
    conn3.isInTransaction = true;
    console.log('Connection 3 marked as in transaction');

    // Release it (will stay in pool but marked)
    await pool.release(conn3);
    console.log('Released connection 3 (still in transaction)');
    console.log('Pool stats:', pool.getStats());

    // Simulate idle timeout
    console.log('\n--- Idle timeout simulation ---');
    console.log('Waiting for idle timeout (15 seconds)...');
    console.log('(Non-transaction connections will be cleaned up)');

    // In a real scenario, the pool's background cleanup would handle this
    // For demo purposes, we just show the concept

  } catch (error) {
    console.error('Error:', error);
  } finally {
    // Clean up: close all connections
    console.log('\n--- Cleanup ---');
    await pool.closeAll();
    console.log('All connections closed');
    console.log('Final pool stats:', pool.getStats());
  }
}

async function demonstrateDirectConnection() {
  console.log('\n\n=== Direct Connection Usage ===\n');

  const params = parseConnectionString('syndrdb://localhost:1776:mydb:admin:password;');
  const conn = new SyndrDBConnection(params);

  console.log('Connection created');
  console.log('Is connected:', conn.isConnected());
  console.log('Is in transaction:', conn.isInTransaction);
  console.log('Last used at:', new Date(conn.lastUsedAt).toISOString());

  try {
    // Attempt to connect (will fail if no server running)
    console.log('\nAttempting to connect...');
    await conn.connect();
    console.log('Connected successfully!');

    // Send a command (placeholder)
    console.log('\nSending test command...');
    await conn.sendCommand('TEST_COMMAND');

    // Receive response
    console.log('Waiting for response...');
    const response = await conn.receiveResponse();
    console.log('Received:', response);

  } catch (error: any) {
    console.error('Expected error (server not running):', error.message);
  } finally {
    await conn.close();
    console.log('\nConnection closed');
    console.log('Is connected:', conn.isConnected());
  }
}

// Run the examples
async function main() {
  await demonstrateConnectionPool();
  await demonstrateDirectConnection();
}

main().catch(console.error);
