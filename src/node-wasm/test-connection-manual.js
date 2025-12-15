#!/usr/bin/env node
/**
 * Manual test script to verify Node TCP connection to SyndrDB server
 */

const net = require('net');

console.log('Attempting to connect to 127.0.0.1:1776...');

const socket = net.connect({
  host: '127.0.0.1',
  port: 1776
});

let buffer = '';

socket.on('connect', () => {
  console.log('✓ Socket connected!');
  
  const connectionString = 'syndrdb://127.0.0.1:1776:primary:root:root;';
  const message = connectionString + '\x04';
  
  console.log('Sending connection string:', connectionString);
  socket.write(message);
});

socket.on('data', (data) => {
  console.log('✓ Received data:', data.length, 'bytes');
  console.log('  Raw:', data.toString().substring(0, 200));
  
  buffer += data.toString();
  
  // Split by EOT
  const messages = buffer.split('\x04');
  buffer = messages.pop() || '';
  
  for (const msg of messages) {
    const trimmed = msg.trim();
    if (trimmed) {
      console.log('✓ Complete message:', trimmed);
    }
  }
});

socket.on('error', (err) => {
  console.error('✗ Socket error:', err.message);
  process.exit(1);
});

socket.on('close', () => {
  console.log('Socket closed');
  process.exit(0);
});

// Timeout after 5 seconds
setTimeout(() => {
  console.log('✗ Timeout - no response received');
  socket.destroy();
  process.exit(1);
}, 5000);
