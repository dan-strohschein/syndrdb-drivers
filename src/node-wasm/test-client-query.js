const { SyndrDBClient } = require('./dist/index.js');

async function test() {
  const client = new SyndrDBClient({ debugMode: true });
  
  try {
    console.log('=== Initializing client ===');
    await client.initialize();
    
    console.log('\n=== Connecting to server ===');
    await client.connect('syndrdb://127.0.0.1:1776:primary:root:root;');
    
    console.log('\n=== Sending SELECT query ===');
    const result = await client.query('SELECT 1 as value');
    console.log('Query result:', result);
    
    console.log('\n=== Disconnecting ===');
    await client.disconnect();
    console.log('Done!');
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}

test();
