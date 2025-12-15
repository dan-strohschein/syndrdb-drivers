const { SyndrDBClient } = require('./dist/index.js');

async function test() {
  const client = new SyndrDBClient({ debugMode: false });
  
  try {
    await client.initialize();
    await client.connect('syndrdb://127.0.0.1:1776:primary:root:root;');
    console.log('✅ Connected');
    
    // Check current state
    const before = await client.query('SELECT * FROM "test_syndrql"');
    console.log(`\nBefore: ${before.length} documents`);
    before.forEach(doc => console.log(`  - id=${doc.id}, name=${doc.name}`));
    
    // Try DELETE with WHERE
    console.log('\nExecuting: DELETE DOCUMENTS FROM "test_syndrql" WHERE "id" == 1;');
    const result = await client.mutate('DELETE DOCUMENTS FROM "test_syndrql" WHERE "id" == 1;');
    console.log('Result:', JSON.stringify(result));
    
    // Check after
    const after = await client.query('SELECT * FROM "test_syndrql"');
    console.log(`\nAfter: ${after.length} documents`);
    after.forEach(doc => console.log(`  - id=${doc.id}, name=${doc.name}`));
    
    await client.disconnect();
  } catch (error) {
    console.error('❌ Error:', error.message);
  }
}

test();
