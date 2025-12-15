/**
 * Simple connection test
 */

const { SyndrDBClient } = require('./dist/index.js');

async function test() {
  console.log('Creating client...');
  const client = new SyndrDBClient({ debugMode: true });
  
  console.log('Initializing...');
  await client.initialize();
  console.log('Initialized!');
  
  console.log('Connecting to 127.0.0.1:1776...');
  try {
    await client.connect('syndrdb://127.0.0.1:1776');
    console.log('✅ Connected successfully!');
    console.log('Client state:', client.getState());
    
    // Try ping
    console.log('\nPinging...');
    const latency = await client.ping();
    console.log('✅ Ping latency:', latency, 'ms');
    
    await client.disconnect();
    console.log('✅ Disconnected');
  } catch (error) {
    console.error('❌ Connection failed:', error.message);
    if (error.cause) {
      console.error('  Cause:', error.cause);
    }
  }
}

test();
