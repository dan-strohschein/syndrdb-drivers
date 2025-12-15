const { SyndrDBClient } = require('./dist/index.js');

async function cleanup() {
  const client = new SyndrDBClient({ debugMode: false });
  try {
    await client.initialize();
    await client.connect('syndrdb://127.0.0.1:1776:primary:root:root;');
    
    // Check what's in the bundle
    try {
      const data = await client.query('SELECT * FROM "test_syndrql"');
      console.log('Current data:', JSON.stringify(data, null, 2));
    } catch (e) {
      console.log('Query error:', e.message);
    }
    
    // Try to delete all
    try {
      await client.mutate('DELETE DOCUMENTS FROM "test_syndrql" CONFIRMED;');
      console.log('✅ Deleted all documents');
    } catch (e) {
      console.log('Delete error:', e.message);
    }
    
    // Try to drop
    try {
      await client.mutate('DROP BUNDLE "test_syndrql";');
      console.log('✅ Dropped bundle');
    } catch (e) {
      console.log('Drop error:', e.message);
    }
    
    await client.disconnect();
  } catch (error) {
    console.error('Error:', error.message);
  }
}

cleanup();
