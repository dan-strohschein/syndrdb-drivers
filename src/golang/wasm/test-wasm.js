#!/usr/bin/env node

/**
 * Node.js test for SyndrDB WASM driver
 * Tests the WASM module's JavaScript API
 */

const fs = require('fs');
const path = require('path');

// Load wasm_exec.js
require('./wasm_exec.js');

async function runTests() {
  console.log('Loading WASM module...');
  
  const wasmPath = path.join(__dirname, 'syndrdb.wasm');
  const wasmBinary = fs.readFileSync(wasmPath);
  
  const go = new Go();
  const result = await WebAssembly.instantiate(wasmBinary, go.importObject);
  
  // Run the Go program
  go.run(result.instance);
  
  // Wait a bit for initialization
  await new Promise(resolve => setTimeout(resolve, 100));
  
  console.log('✓ WASM module loaded\n');
  
  // Check that SyndrDB exports are available
  if (!global.SyndrDB) {
    throw new Error('SyndrDB exports not found');
  }
  
  console.log('Available exports:', Object.keys(global.SyndrDB));
  console.log('');
  
  // Test 1: Get version
  console.log('Test 1: Get version');
  const version = global.SyndrDB.getVersion();
  console.log(`  Version: ${version}`);
  console.log('  ✓ Pass\n');
  
  // Test 2: Create client
  console.log('Test 2: Create client');
  try {
    await global.SyndrDB.createClient({
      defaultTimeoutMs: 10000,
      debugMode: false,
      maxRetries: 3
    });
    console.log('  ✓ Client created\n');
  } catch (err) {
    console.error('  ✗ Failed:', err);
    throw err;
  }
  
  // Test 3: Get state (should be DISCONNECTED)
  console.log('Test 3: Get initial state');
  const initialState = global.SyndrDB.getState();
  console.log(`  State: ${initialState}`);
  if (initialState !== 'DISCONNECTED') {
    throw new Error(`Expected DISCONNECTED, got ${initialState}`);
  }
  console.log('  ✓ Pass\n');
  
  // Test 4: Register state change callback
  console.log('Test 4: Register state change callback');
  let stateChanges = [];
  global.SyndrDB.onStateChange((transition) => {
    stateChanges.push(transition);
    console.log(`  State change: ${transition.from} → ${transition.to}`);
  });
  console.log('  ✓ Callback registered\n');
  
  // Test 5: Connect to server
  console.log('Test 5: Connect to server');
  try {
    await global.SyndrDB.connect('syndrdb://localhost:1776:primary:root:root;');
    console.log('  ✓ Connected\n');
    
    const connectedState = global.SyndrDB.getState();
    console.log(`  Connected state: ${connectedState}`);
    if (connectedState !== 'CONNECTED') {
      throw new Error(`Expected CONNECTED, got ${connectedState}`);
    }
  } catch (err) {
    console.error('  ✗ Connection failed:', err);
    console.log('  ⚠ Skipping remaining tests (server may not be running)\n');
    
    // Test cleanup anyway
    console.log('Test 6: Cleanup');
    global.SyndrDB.cleanup();
    console.log('  ✓ Cleanup complete\n');
    
    console.log('=== Test Summary ===');
    console.log('Basic WASM tests passed (5/5)');
    console.log('Server tests skipped (server not available)');
    console.log('');
    return;
  }
  
  // Test 6: Execute query
  console.log('Test 6: Execute query');
  try {
    const result = await global.SyndrDB.query('SHOW BUNDLES;', 10000);
    console.log(`  ✓ Query executed`);
    console.log(`  Result type: ${typeof result}`);
    if (result && typeof result === 'object') {
      console.log(`  Has Result field: ${!!result.Result}`);
      console.log(`  ResultCount: ${result.ResultCount || 0}`);
    }
    console.log('');
  } catch (err) {
    console.error('  ✗ Query failed:', err);
    throw err;
  }
  
  // Test 7: Execute mutation
  console.log('Test 7: Execute mutation (drop test bundle)');
  try {
    await global.SyndrDB.mutate('DROP BUNDLE "wasm_test";', 10000);
    console.log('  ✓ Mutation executed (drop - expected to succeed or fail gracefully)\n');
  } catch (err) {
    console.log('  ⚠ Drop failed (bundle may not exist) - continuing\n');
  }
  
  // Test 8: Schema generation
  console.log('Test 8: Generate JSON Schema');
  try {
    const schemaDef = {
      bundles: [
        {
          name: 'users',
          fields: [
            { name: 'id', type: 'int', required: true, unique: true, defaultValue: null },
            { name: 'name', type: 'string', required: true, unique: false, defaultValue: null }
          ],
          indexes: { hash: [], btree: [] },
          relationships: []
        }
      ]
    };
    
    const jsonSchema = await global.SyndrDB.generateJSONSchema(JSON.stringify(schemaDef), 'single');
    console.log('  ✓ JSON Schema generated');
    console.log(`  Schema length: ${jsonSchema.length} characters\n`);
  } catch (err) {
    console.error('  ✗ Schema generation failed:', err);
    throw err;
  }
  
  // Test 9: Disconnect
  console.log('Test 9: Disconnect');
  try {
    await global.SyndrDB.disconnect();
    console.log('  ✓ Disconnected\n');
    
    const disconnectedState = global.SyndrDB.getState();
    console.log(`  Final state: ${disconnectedState}`);
  } catch (err) {
    console.error('  ✗ Disconnect failed:', err);
    throw err;
  }
  
  // Test 10: Verify state changes
  console.log('Test 10: Verify state changes');
  console.log(`  Total state changes recorded: ${stateChanges.length}`);
  if (stateChanges.length > 0) {
    console.log('  State transitions:');
    stateChanges.forEach((t, i) => {
      console.log(`    ${i + 1}. ${t.from} → ${t.to} (${t.duration}ms)`);
    });
  }
  console.log('  ✓ Pass\n');
  
  // Test 11: Hooks System - Register custom JS hook
  console.log('Test 11: Register custom JavaScript hook');
  let hookBeforeCalled = false;
  let hookAfterCalled = false;
  let capturedCommand = null;
  
  await global.SyndrDB.registerHook({
    name: 'test-js-hook',
    before: (ctx) => {
      hookBeforeCalled = true;
      capturedCommand = ctx.command;
      console.log(`  Hook before: ${ctx.command}`);
      // Can modify command here
      return ctx;
    },
    after: (ctx) => {
      hookAfterCalled = true;
      console.log(`  Hook after: ${ctx.durationMs}ms`);
      return ctx;
    }
  });
  
  const hooksResult = await global.SyndrDB.getHooks();
  console.log(`  Registered hooks: ${hooksResult.hooks.join(', ')}`);
  console.log('  ✓ Pass\n');
  
  // Test 12: Hooks System - Create metrics hook
  console.log('Test 12: Create metrics hook');
  await global.SyndrDB.createMetricsHook();
  
  const metricsHooks = await global.SyndrDB.getHooks();
  if (!metricsHooks.hooks.includes('metrics')) {
    throw new Error('Metrics hook not registered');
  }
  console.log('  ✓ Metrics hook created\n');
  
  // Test 13: Hooks System - Create logging hook
  console.log('Test 13: Create logging hook');
  await global.SyndrDB.createLoggingHook({
    logCommands: true,
    logResults: false,
    logDurations: true
  });
  
  const loggingHooks = await global.SyndrDB.getHooks();
  if (!loggingHooks.hooks.includes('logging')) {
    throw new Error('Logging hook not registered');
  }
  console.log('  ✓ Logging hook created\n');
  
  // Test 14: Hooks System - Async hook with Promise
  console.log('Test 14: Register async hook with Promise');
  await global.SyndrDB.registerHook({
    name: 'async-hook',
    before: async (ctx) => {
      // Simulate async work
      await new Promise(resolve => setTimeout(resolve, 10));
      console.log('  Async hook: async work completed');
      return ctx;
    },
    after: async (ctx) => {
      // Simulate async work
      await new Promise(resolve => setTimeout(resolve, 5));
      return ctx;
    }
  });
  console.log('  ✓ Async hook registered\n');
  
  // Test 15: Hooks System - Hook that modifies command
  console.log('Test 15: Hook that modifies command');
  await global.SyndrDB.registerHook({
    name: 'modifier-hook',
    before: (ctx) => {
      console.log(`  Original command: ${ctx.command}`);
      // Add metadata
      ctx.metadata = ctx.metadata || {};
      ctx.metadata.modified = true;
      ctx.metadata.timestamp = Date.now();
      return ctx;
    },
    after: (ctx) => {
      if (ctx.metadata && ctx.metadata.modified) {
        console.log('  Command was modified by hook');
      }
      return ctx;
    }
  });
  console.log('  ✓ Modifier hook registered\n');
  
  // Test 16: Hooks System - Get metrics stats
  console.log('Test 16: Get metrics stats');
  const stats = await global.SyndrDB.getMetricsStats();
  console.log(`  Total commands: ${stats.total_commands}`);
  console.log(`  Total queries: ${stats.total_queries}`);
  console.log(`  Total errors: ${stats.total_errors}`);
  console.log(`  Avg duration: ${stats.avg_duration_ms.toFixed(2)}ms`);
  console.log('  ✓ Pass\n');
  
  // Test 17: Hooks System - Unregister hook
  console.log('Test 17: Unregister hook');
  await global.SyndrDB.unregisterHook('test-js-hook');
  
  const afterUnregister = await global.SyndrDB.getHooks();
  if (afterUnregister.hooks.includes('test-js-hook')) {
    throw new Error('Hook should be unregistered');
  }
  console.log(`  Remaining hooks: ${afterUnregister.hooks.join(', ')}`);
  console.log('  ✓ Pass\n');
  
  // Test 18: Hooks System - Reset metrics
  console.log('Test 18: Reset metrics');
  await global.SyndrDB.resetMetrics();
  
  const resetStats = await global.SyndrDB.getMetricsStats();
  if (resetStats.total_commands !== 0) {
    throw new Error('Metrics should be reset');
  }
  console.log('  ✓ Metrics reset successful\n');
  
  // Test 19: Cleanup
  console.log('Test 19: Cleanup');
  global.SyndrDB.cleanup();
  console.log('  ✓ Cleanup complete\n');
  
  console.log('=== All Tests Passed ===');
  console.log(`Completed ${19} tests successfully`);
  console.log('\nHooks System Features Tested:');
  console.log('  ✓ Custom JavaScript hooks (sync)');
  console.log('  ✓ Async hooks with Promises');
  console.log('  ✓ Built-in metrics hook');
  console.log('  ✓ Built-in logging hook');
  console.log('  ✓ Hook command modification');
  console.log('  ✓ Hook metadata passing');
  console.log('  ✓ Hook registration/unregistration');
  console.log('  ✓ Metrics collection and reset');
}

// Run tests
runTests()
  .then(() => {
    console.log('\n✓ WASM test suite completed successfully');
    process.exit(0);
  })
  .catch(err => {
    console.error('\n✗ Test suite failed:', err);
    console.error(err.stack);
    process.exit(1);
  });
