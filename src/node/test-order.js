const net = require('net');

const host = 'localhost';
const port = 1776;
const database = 'authortestdb';
const username = 'root';
const password = 'root';

console.log('=== Testing Connection Order ===\n');

const socket = new net.Socket();
let buffer = '';
let step = 0;

socket.on('connect', () => {
  step++;
  console.log(`${step}. ✓ TCP Connected`);
  
  const connString = `syndrdb://${host}:${port}:${database}:${username}:${password};\n`;
  console.log(`${step}. → Sending connection string: ${connString.trim()}\n`);
  socket.write(connString);
});

socket.on('data', (chunk) => {
  buffer += chunk.toString();
  
  // Process all complete lines
  while (buffer.indexOf('\n') !== -1) {
    const newlineIndex = buffer.indexOf('\n');
    const line = buffer.slice(0, newlineIndex).trim();
    buffer = buffer.slice(newlineIndex + 1);
    
    if (!line) continue;
    
    step++;
    console.log(`${step}. ← Received: ${line.substring(0, 100)}${line.length > 100 ? '...' : ''}\n`);
    
    if (line.includes('S0001')) {
      console.log(`${step}. ✓ Welcome message received\n`);
    } else if (line.includes('Authentication successful')) {
      step++;
      console.log(`${step}. ✓ Authentication successful\n`);
      
      step++;
      console.log(`${step}. → Sending: SHOW BUNDLES FOR "authortestdb";\n`);
      socket.write('SHOW BUNDLES FOR "authortestdb";\n');
    } else if (line.includes('"Result"') && line.includes('"ExecutionTimeMS"')) {
      try {
        const parsed = JSON.parse(line);
        console.log(`${step}. ✓ Bundles result:`, JSON.stringify(parsed, null, 2), '\n');
      } catch (e) {
        console.log(`${step}. ✓ Bundles result (raw): ${line}\n`);
      }
      
      step++;
      console.log(`${step}. → Sending: SHOW MIGRATIONS FOR "authortestdb";\n`);
      socket.write('SHOW MIGRATIONS FOR "authortestdb";\n');
    } else if (line.includes('migrations') || line.includes('currentVersion')) {
      try {
        const parsed = JSON.parse(line);
        console.log(`${step}. ✓ Migrations result:`, JSON.stringify(parsed, null, 2), '\n');
      } catch (e) {
        console.log(`${step}. ✓ Migrations result (raw): ${line}\n`);
      }
      
      console.log('=== Test Complete ===');
      socket.end();
      process.exit(0);
    }
  }
});

socket.on('error', (err) => {
  console.error('✗ Socket error:', err.message);
  process.exit(1);
});

socket.on('close', () => {
  console.log('\n=== Connection Closed ===');
});

socket.connect(port, host);

setTimeout(() => {
  console.log('\n✗ Timeout - Connection took too long');
  socket.end();
  process.exit(1);
}, 15000);
