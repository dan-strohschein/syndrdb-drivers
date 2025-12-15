#!/usr/bin/env node

/**
 * Copies WASM binary and runtime from Go build to dist/wasm/
 * Validates DWARF sections and computes integrity checksums
 */

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

const SOURCE_WASM = path.join(__dirname, '../../../src/golang/wasm/syndrdb.wasm');
const SOURCE_EXEC = path.join(__dirname, '../../../src/golang/wasm/wasm_exec.js');
const DEST_DIR = path.join(__dirname, '../dist/wasm');
const DEST_WASM = path.join(DEST_DIR, 'syndrdb.wasm');
const DEST_EXEC = path.join(DEST_DIR, 'wasm_exec.js');

function computeSHA256(filePath) {
  const content = fs.readFileSync(filePath);
  return crypto.createHash('sha256').update(content).digest('hex');
}

function checkDWARFSections(wasmPath) {
  const buffer = fs.readFileSync(wasmPath);
  
  // Check for DWARF section markers in WASM custom sections
  const hasDWARF = buffer.includes('.debug_info') || 
                   buffer.includes('.debug_line') ||
                   buffer.includes('.debug_abbrev');
  
  return hasDWARF;
}

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

function copyWASM() {
  console.log('📦 Copying WASM artifacts...\n');
  
  // Check source files exist
  if (!fs.existsSync(SOURCE_WASM)) {
    console.error(`❌ WASM binary not found: ${SOURCE_WASM}`);
    console.error('   Run: npm run build:wasm');
    process.exit(1);
  }
  
  if (!fs.existsSync(SOURCE_EXEC)) {
    console.error(`❌ wasm_exec.js not found: ${SOURCE_EXEC}`);
    process.exit(1);
  }
  
  // Create destination directory
  if (!fs.existsSync(DEST_DIR)) {
    fs.mkdirSync(DEST_DIR, { recursive: true });
  }
  
  // Copy WASM binary
  fs.copyFileSync(SOURCE_WASM, DEST_WASM);
  const wasmSize = fs.statSync(DEST_WASM).size;
  const wasmHash = computeSHA256(DEST_WASM);
  
  console.log(`✓ Copied: syndrdb.wasm`);
  console.log(`  Size: ${formatBytes(wasmSize)}`);
  console.log(`  SHA256: ${wasmHash}`);
  
  // Check DWARF sections
  const hasDWARF = checkDWARFSections(DEST_WASM);
  if (hasDWARF) {
    console.log(`  ✓ DWARF v5 debug info present`);
  } else {
    console.warn(`  ⚠ Warning: DWARF sections not detected`);
    console.warn(`    Rebuild with: GOOS=js GOARCH=wasm go build -ldflags="-w=0"`);
  }
  
  // Copy wasm_exec.js
  fs.copyFileSync(SOURCE_EXEC, DEST_EXEC);
  const execSize = fs.statSync(DEST_EXEC).size;
  const execHash = computeSHA256(DEST_EXEC);
  
  console.log(`\n✓ Copied: wasm_exec.js`);
  console.log(`  Size: ${formatBytes(execSize)}`);
  console.log(`  SHA256: ${execHash}`);
  
  // Write integrity manifest
  const manifest = {
    timestamp: new Date().toISOString(),
    wasm: {
      file: 'syndrdb.wasm',
      size: wasmSize,
      sha256: wasmHash,
      hasDWARF,
    },
    exec: {
      file: 'wasm_exec.js',
      size: execSize,
      sha256: execHash,
    },
  };
  
  const manifestPath = path.join(DEST_DIR, 'integrity.json');
  fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));
  
  console.log(`\n✓ Integrity manifest: ${manifestPath}`);
  console.log(`\n✅ WASM artifacts copied successfully`);
  
  // Check bundle size
  const MAX_SIZE = 5 * 1024 * 1024; // 5MB uncompressed
  if (wasmSize > MAX_SIZE) {
    console.warn(`\n⚠ Warning: WASM binary > ${formatBytes(MAX_SIZE)}`);
    console.warn(`  Consider using TinyGo or optimizing build flags`);
  }
}

copyWASM();
