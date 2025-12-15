const { SyndrDBClient } = require('./dist/index.js');

async function test() {
  const client = new SyndrDBClient({ debugMode: false });
  
  try {
    await client.initialize();
    await client.connect('syndrdb://127.0.0.1:1776:primary:root:root;');
    console.log('Connected');
    
    // Try simple query
    console.log('Executing: SELECT 1 as value');
    const result1 = await client.query('SELECT 1 as value');
    console.log('Result 1:', result1);
    
    // Try parameterized query
    console.log('\nExecuting: SELECT $1 as value with param [42]');
    const result2 = await client.query('SELECT $1 as value', [42]);
    console.log('Result 2:', result2);
    
    await client.disconnect();
    console.log('\n✅ All queries successful!');
  } catch (error) {
    console.error('❌ Error:', error.message);
    process.exit(1);
  }
}

test();
