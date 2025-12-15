const { join } = require('path');
const { existsSync } = require('fs');

console.log('__dirname:', __dirname);
console.log('Looking for wasm_exec.js...');

const paths = [
  join(__dirname, './wasm/wasm_exec.js'),
  join(__dirname, '../dist/wasm/wasm_exec.js'),
  join(__dirname, '../../dist/wasm/wasm_exec.js'),
  join(__dirname, './dist/wasm/wasm_exec.js'),
];

paths.forEach(p => {
  console.log(`  ${existsSync(p) ? '✓' : '✗'} ${p}`);
});
