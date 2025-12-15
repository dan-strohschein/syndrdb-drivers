const { SyndrDBClient } = require('./dist/index.js');

async function test() {
  const client = new SyndrDBClient({ debugMode: false });
  
  try {
    await client.initialize();
    await client.connect('syndrdb://127.0.0.1:1776:primary:root:root;');
    console.log('✅ Connected');
    
    // beforeAll cleanup
    console.log('\n1. Cleaning up...');
    try {
      await client.mutate('DELETE DOCUMENTS FROM "test_basic" CONFIRMED;');
      await client.mutate('DROP BUNDLE "test_basic" WITH FORCE;');
      console.log('✅ Cleaned up existing bundle');
    } catch (e) {
      console.log('   (No existing bundle)');
    }
    
    // beforeAll create
    console.log('\n2. Creating test bundle...');
    await client.mutate('CREATE BUNDLE "test_basic" WITH FIELDS ({"id", "int", TRUE, FALSE, 0}, {"value", "string", FALSE, FALSE", ""})');
    console.log('✅ Bundle created');
    
    // beforeEach cleanup
    console.log('\n3. Clearing data...');
    await client.mutate('DELETE DOCUMENTS FROM "test_basic" CONFIRMED;');
    console.log('✅ Data cleared');
    
    // Test mutation
    console.log('\n4. Adding document...');
    await client.mutate('ADD DOCUMENT TO BUNDLE "test_basic" WITH ({"id" = 1}, {"value" = "test"})');
    console.log('✅ Document added');
    
    // afterAll cleanup
    console.log('\n5. Final cleanup...');
    await client.mutate('DELETE DOCUMENTS FROM "test_basic" CONFIRMED;');
    await client.mutate('DROP BUNDLE "test_basic" WITH FORCE;');
    console.log('✅ Cleaned up');
    
    await client.disconnect();
    console.log('\n🎉 All operations succeeded!');
  } catch (error) {
    console.error('\n❌ Error:', error.message);
    process.exit(1);
  }
}

test();
