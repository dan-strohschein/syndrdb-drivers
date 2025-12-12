const net = require('net');

const host = 'localhost';
const port = 1776;
const database = 'authortestdb';
const username = 'root';
const password = 'root';

console.log('Connecting to', `${host}:${port}...`);

const socket = new net.Socket();
let buffer = '';
let responseCount = 0;

socket.on('connect', () => {
  console.log('✓ TCP connection established');
  
  const connString = `syndrdb://${host}:${port}:${database}:${username}:${password};\n`;
  console.log('Sending auth:', connString.trim());
  
  socket.write(connString);
});

socket.on('data', (chunk) => {
  buffer += chunk.toString();
  console.log(`\n[Chunk ${responseCount + 1}]:`, JSON.stringify(chunk.toString()));
  console.log('Buffer:', JSON.stringify(buffer));
  
  // Process all complete lines in the buffer
  while (buffer.indexOf('\n') !== -1) {
    const newlineIndex = buffer.indexOf('\n');
    const line = buffer.slice(0, newlineIndex).trim();
    buffer = buffer.slice(newlineIndex + 1);
    
    if (line) {
      responseCount++;
      console.log(`\n=== Response ${responseCount} ===`);
      console.log('Line:', JSON.stringify(line));
      
      if (responseCount === 1) {
        if (line.includes('S0001')) {
          console.log('✓ Welcome message received');
        }
      } else if (responseCount === 2) {
        console.log('✓ Auth response received');
        console.log('\nNow sending: SHOW BUNDLES FOR "authortestdb"');
        socket.write('SHOW BUNDLES FOR "authortestdb";\n');
      } else if (responseCount === 3) {
        console.log('✓ SHOW BUNDLES response:', line);
        socket.end();
        process.exit(0);
      }
    }
  }
});

socket.on('error', (err) => {
  console.error('✗ Socket error:', err.message);
  process.exit(1);
});

socket.on('close', () => {
  console.log('\nConnection closed. Total responses:', responseCount);
});

socket.connect(port, host);

// Timeout after 15 seconds
setTimeout(() => {
  console.log('\n✗ Timeout');
  console.log('Responses received:', responseCount);
  console.log('Remaining buffer:', JSON.stringify(buffer));
  socket.end();
  process.exit(1);
}, 15000);
