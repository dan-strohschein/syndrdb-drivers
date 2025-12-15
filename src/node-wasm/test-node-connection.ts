#!/usr/bin/env ts-node
/**
 * Test NodeConnection class directly
 */

import { NodeConnection } from './src/node-connection';

async function test() {
  console.log('Creating NodeConnection...');
  
  const conn = new NodeConnection({
    host: '127.0.0.1',
    port: 1776,
    connectionTimeout: 10000,
    tls: { enabled: false },
  });

  try {
    console.log('Connecting...');
    await conn.connect('syndrdb://127.0.0.1:1776:primary:root:root;');
    console.log('✓ Connected successfully!');
    
    console.log('Sending PING...');
    await conn.sendCommand('PING');
    
    console.log('Receiving response...');
    const response = await conn.receiveResponse();
    console.log('✓ Response:', response);
    
    await conn.close();
    console.log('✓ Connection closed');
    process.exit(0);
  } catch (error: any) {
    console.error('✗ Error:', error.message);
    process.exit(1);
  }
}

test();
