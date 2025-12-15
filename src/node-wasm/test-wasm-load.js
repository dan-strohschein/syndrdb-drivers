/**
 * Simple test script to debug WASM loading
 */

const { SyndrDBClient } = require('./dist/index.js');

async function test() {
  console.log('Creating client...');
  const client = new SyndrDBClient({ debugMode: true });
  
  console.log('Initializing...');
  try {
    await client.initialize();
    console.log('Initialization successful!');
    console.log('Client state:', client.getState());
  } catch (error) {
    console.error('Initialization failed:', error.message);
    if (error.cause) {
      console.error('Cause:', error.cause.message);
    }
  }
}

test();
