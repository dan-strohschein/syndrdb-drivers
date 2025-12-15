const { SyndrDBClient } = require('./dist/index.js');

async function test() {
  const client = new SyndrDBClient({ debugMode: false });
  
  try {
    await client.initialize();
    await client.connect('syndrdb://127.0.0.1:1776:primary:root:root;');
    console.log('✅ Connected');
    
    // Cleanup first
    console.log('\n0. Cleaning up any existing bundle...');
    try {
      await client.mutate('DELETE DOCUMENTS FROM "test_syndrql" CONFIRMED;');
      await client.mutate('DROP BUNDLE "test_syndrql" WITH FORCE;');
      console.log('✅ Existing bundle cleaned up');
    } catch (e) {
      console.log('   (No existing bundle)');
    }
    
    // Test 1: CREATE BUNDLE
    console.log('\n1. Testing CREATE BUNDLE...');
    await client.mutate('CREATE BUNDLE "test_syndrql" WITH FIELDS ({"id", "int", TRUE, TRUE, 0}, {"name", "string", FALSE, FALSE, ""})');
    console.log('✅ Bundle created');
    
    // Test 2: ADD DOCUMENT
    console.log('\n2. Testing ADD DOCUMENT...');
    await client.mutate('ADD DOCUMENT TO BUNDLE "test_syndrql" WITH ({"id" = 1}, {"name" = "Alice"})');
    console.log('✅ Document added');
    
    // Test 3: SELECT
    console.log('\n3. Testing SELECT...');
    const result = await client.query('SELECT * FROM "test_syndrql"');
    console.log('✅ Query result:', JSON.stringify(result, null, 2));
    
    // Test 4: SELECT with WHERE using ==
    console.log('\n4. Testing SELECT with WHERE (==)...');
    const result2 = await client.query('SELECT * FROM "test_syndrql" WHERE "id" == 1');
    console.log('✅ WHERE result:', JSON.stringify(result2, null, 2));
    
    // Test 5: Parameterized query
    console.log('\n5. Testing parameterized query...');
    await client.mutate('ADD DOCUMENT TO BUNDLE "test_syndrql" WITH ({"id" = $1}, {"name" = $2})', [2, 'Bob']);
    const result3 = await client.query('SELECT * FROM "test_syndrql" WHERE "id" == $1', [2]);
    console.log('✅ Param query result:', JSON.stringify(result3, null, 2));
    
    // Test 6: DELETE with WHERE
    console.log('\n6. Testing DELETE with WHERE...');
    await client.mutate('DELETE DOCUMENTS FROM "test_syndrql" WHERE "id" == 1;');
    const result4 = await client.query('SELECT * FROM "test_syndrql"');
    console.log('✅ After delete, remaining:', JSON.stringify(result4, null, 2));
    
    // Cleanup
    console.log('\n7. Final cleanup...');
    await client.mutate('DELETE DOCUMENTS FROM "test_syndrql" CONFIRMED;');
    await client.mutate('DROP BUNDLE "test_syndrql" WITH FORCE;');
    console.log('✅ Bundle dropped');
    
    await client.disconnect();
    console.log('\n🎉 All SyndrQL tests passed!');
  } catch (error) {
    console.error('\n❌ Error:', error.message);
    process.exit(1);
  }
}

test();
