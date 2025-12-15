#!/usr/bin/env node

/**
 * Validates that Go 1.25+ is installed for DWARF v5 support
 */

const { execSync } = require('child_process');

const MIN_GO_VERSION = '1.25';

function validateGoVersion() {
  try {
    const goVersionOutput = execSync('go version', { encoding: 'utf-8' });
    const match = goVersionOutput.match(/go(\d+)\.(\d+)/);
    
    if (!match) {
      console.error('❌ Could not parse Go version');
      process.exit(1);
    }
    
    const major = parseInt(match[1], 10);
    const minor = parseInt(match[2], 10);
    const version = `${major}.${minor}`;
    
    console.log(`✓ Found Go ${version}`);
    
    // Check minimum version
    const [minMajor, minMinor] = MIN_GO_VERSION.split('.').map(Number);
    
    if (major < minMajor || (major === minMajor && minor < minMinor)) {
      console.error(`❌ Go ${MIN_GO_VERSION}+ required for DWARF v5 support`);
      console.error(`   Current version: ${version}`);
      console.error(`   Please upgrade Go: https://go.dev/dl/`);
      process.exit(1);
    }
    
    console.log(`✓ Go version ${version} meets minimum requirement (${MIN_GO_VERSION}+)`);
    console.log('✓ DWARF v5 debug info will be embedded in WASM binary');
    
  } catch (error) {
    console.error('❌ Go not found in PATH');
    console.error('   Please install Go 1.25+: https://go.dev/dl/');
    process.exit(1);
  }
}

validateGoVersion();
