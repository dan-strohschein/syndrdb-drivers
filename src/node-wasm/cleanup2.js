const { SyndrDBClient } = require('./dist/index.js');

async function cleanup() {
  const client = new SyndrDBClient({ debugMode: false });
  try {
    await client.initialize();
    await client.connect('syndrdb://127.0.0.1:1776:primary:root:root;');
    
    // Get all documents
    const data = await client.query('SELECT * FROM "test_syndrql"');
    console.log(`Found ${data.length} documents`);
    
    // Delete each one by ID
    for (const doc of data) {
      try {
        await client.mutate(`DELETE DOCUMENTS FROM "test_syndrql" WHERE "id" == ${doc.id};`);
        console.log(`✅ Deleted document with id=${doc.id}`);
      } catch (e) {
        console.log(`❌ Failed to delete id=${doc.id}:`, e.message);
      }
    }
    
    // Check if empty
    const remaining = await client.query('SELECT * FROM "test_syndrql"');
    console.log(`Remaining documents: ${remaining.length}`);
    
    // Try to drop
    if (remaining.length === 0) {
      await client.mutate('DROP BUNDLE "test_syndrql";');
      console.log('✅ Dropped bundle');
    }
    
    await client.disconnect();
  } catch (error) {
    console.error('Error:', error.message);
  }
}

cleanup();
