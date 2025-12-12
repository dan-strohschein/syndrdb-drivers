const { createMigrationClient } = require('./src/migration-client.ts');

async function testQueries() {
  const client = createMigrationClient();
  
  try {
    console.log('Connecting...');
    await client.connect('syndrdb://localhost:1776:authortestdb:root:root;');
    console.log('✓ Connected\n');

    // Test 1: Raw connection query
    console.log('=== Test 1: Raw Query - SHOW MIGRATIONS ===');
    try {
      const conn = await client['pool'].acquire();
      await conn.sendCommand('SHOW MIGRATIONS FOR "authortestdb"');
      const rawResponse = await conn.receiveResponse();
      console.log('Raw Response:', JSON.stringify(rawResponse, null, 2));
      await client['pool'].release(conn);
    } catch (err) {
      console.error('Error:', err.message);
    }

    // Test 2: Raw query - SHOW BUNDLES
    console.log('\n=== Test 2: Raw Query - SHOW BUNDLES ===');
    try {
      const conn = await client['pool'].acquire();
      await conn.sendCommand('SHOW BUNDLES FOR "authortestdb"');
      const rawResponse = await conn.receiveResponse();
      console.log('Raw Response:', JSON.stringify(rawResponse, null, 2));
      await client['pool'].release(conn);
    } catch (err) {
      console.error('Error:', err.message);
    }

    await client.close();
  } catch (err) {
    console.error('Fatal error:', err);
    process.exit(1);
  }
}

testQueries();
