const { SyndrDBClient } = require('./dist/index.js');

async function cleanup() {
  const client = new SyndrDBClient({ debugMode: false });
  try {
    await client.initialize();
    await client.connect('syndrdb://127.0.0.1:1776:primary:root:root;');
    console.log('Connected');
    
    // Try DROP with FORCE directly
    try {
      await client.mutate('DROP BUNDLE "test_syndrql" FORCE;');
      console.log('✅ Dropped bundle with FORCE');
    } catch (e) {
      console.log('Drop error:', e.message);
    }
    
    await client.disconnect();
  } catch (error) {
    console.error('Error:', error.message);
  }
}

cleanup();
