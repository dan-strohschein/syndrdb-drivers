#!/usr/bin/env node

/**
 * SyndrDB Migration Script for CI/CD
 * 
 * This script demonstrates using the WASM driver in Node.js for automated migrations.
 * Perfect for CI/CD pipelines, deployment scripts, and automated database management.
 */

import fs from 'fs';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Configuration
const SYNDR_CONN = process.env.SYNDR_CONN || 'syndrdb://127.0.0.1:1776:primary:root:root;';
const SCHEMA_FILE = process.env.SCHEMA_FILE || './schema.json';
const DRY_RUN = process.env.DRY_RUN === 'true';

console.log('🔧 SyndrDB Migration Script');
console.log('============================\n');

// Load WASM driver
console.log('📦 Loading WASM driver...');
const wasmPath = join(__dirname, '../../wasm/syndrdb.wasm');
const wasmBinary = fs.readFileSync(wasmPath);

const go = new Go();
const result = await WebAssembly.instantiate(wasmBinary, go.importObject);
go.run(result.instance);

// Wait for initialization
await new Promise(resolve => setTimeout(resolve, 100));

console.log('✓ WASM driver loaded\n');

// Initialize client
console.log('🔌 Initializing SyndrDB client...');
await SyndrDB.createClient({
  defaultTimeoutMs: 10000,
  debugMode: false,
  maxRetries: 3
});
console.log('✓ Client initialized\n');

// Connect to database
console.log(\`🌐 Connecting to database...\`);
try {
  await SyndrDB.connect(SYNDR_CONN);
  console.log('✓ Connected successfully\n');
} catch (err) {
  console.error('❌ Connection failed:', err);
  process.exit(1);
}

// Load local schema
console.log(\`📄 Loading schema from \${SCHEMA_FILE}...\`);
let localSchema;
try {
  const schemaContent = fs.readFileSync(SCHEMA_FILE, 'utf8');
  localSchema = JSON.parse(schemaContent);
  console.log(\`✓ Loaded schema with \${localSchema.bundles.length} bundle(s)\n\`);
} catch (err) {
  console.error('❌ Failed to load schema:', err.message);
  await SyndrDB.disconnect();
  process.exit(1);
}

// Fetch current schema from server
console.log('🔍 Fetching current schema from server...');
let serverBundles = [];
try {
  const result = await SyndrDB.query('SHOW BUNDLES;', 10000);
  serverBundles = parseBundles(result);
  console.log(\`✓ Server has \${serverBundles.length} bundle(s)\n\`);
} catch (err) {
  console.error('⚠️  Could not fetch server schema:', err.message);
  console.log('   Assuming empty database\n');
}

// Compare schemas
console.log('🔄 Comparing schemas...');
const changes = compareSchemas(localSchema.bundles, serverBundles);

if (changes.length === 0) {
  console.log('✓ No changes detected - database is up to date!\n');
  await SyndrDB.disconnect();
  process.exit(0);
}

console.log(\`📋 Found \${changes.length} change(s):\n\`);
changes.forEach((change, i) => {
  console.log(\`   \${i + 1}. \${change.description}\`);
});
console.log('');

// Generate migration commands
console.log('⚙️  Generating migration commands...');
const upCommands = generateUpCommands(changes);
console.log(\`✓ Generated \${upCommands.length} command(s)\n\`);

if (DRY_RUN) {
  console.log('🔍 DRY RUN MODE - Commands to be executed:\n');
  upCommands.forEach((cmd, i) => {
    console.log(\`   \${i + 1}. \${cmd}\`);
  });
  console.log('');
  await SyndrDB.disconnect();
  process.exit(0);
}

// Apply migration
console.log('🚀 Applying migration...');
let applied = 0;
try {
  for (const cmd of upCommands) {
    console.log(\`   Executing: \${cmd.substring(0, 60)}...\`);
    await SyndrDB.mutate(cmd, 10000);
    applied++;
  }
  console.log(\`✓ Successfully applied \${applied} command(s)\n\`);
} catch (err) {
  console.error(\`❌ Failed after \${applied} command(s):`, err);
  await SyndrDB.disconnect();
  process.exit(1);
}

// Verify migration
console.log('✅ Verifying migration...');
try {
  const verifyResult = await SyndrDB.query('SHOW BUNDLES;', 10000);
  const newBundles = parseBundles(verifyResult);
  console.log(\`✓ Database now has \${newBundles.length} bundle(s)\n\`);
} catch (err) {
  console.error('⚠️  Could not verify migration:', err.message);
}

// Cleanup
await SyndrDB.disconnect();
console.log('✅ Migration complete!\n');

// Helper functions

function parseBundles(result) {
  // Simplified parser - in production, parse actual SHOW BUNDLES output
  if (!result || !Array.isArray(result)) {
    return [];
  }
  return result.map(r => ({
    name: r.name || r.bundle_name,
    fields: []
  }));
}

function compareSchemas(localBundles, serverBundles) {
  const changes = [];
  const serverBundleNames = new Set(serverBundles.map(b => b.name));
  
  for (const localBundle of localBundles) {
    if (!serverBundleNames.has(localBundle.name)) {
      changes.push({
        type: 'CREATE_BUNDLE',
        bundle: localBundle,
        description: \`Create bundle "\${localBundle.name}"\`
      });
    } else {
      // In production, you'd compare fields, indexes, etc.
      // For this example, we only detect new bundles
    }
  }
  
  return changes;
}

function generateUpCommands(changes) {
  const commands = [];
  
  for (const change of changes) {
    if (change.type === 'CREATE_BUNDLE') {
      const bundle = change.bundle;
      let cmd = \`CREATE BUNDLE "\${bundle.name}" WITH FIELDS (\`;
      
      const fieldDefs = bundle.fields.map(field => {
        let def = \`\${field.name} \${field.type.toUpperCase()}\`;
        if (field.required) def += ' REQUIRED';
        if (field.unique) def += ' UNIQUE';
        return def;
      });
      
      cmd += fieldDefs.join(', ');
      cmd += ');';
      commands.push(cmd);
      
      // Add indexes
      for (const index of bundle.indexes || []) {
        const idxCmd = \`CREATE INDEX "\${index.name}" ON "\${bundle.name}" (\${index.fields.join(', ')}) TYPE \${index.type.toUpperCase()};\`;
        commands.push(idxCmd);
      }
    }
  }
  
  return commands;
}
