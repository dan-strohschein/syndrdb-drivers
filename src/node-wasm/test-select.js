const net = require('net');

const socket = new net.Socket();

socket.connect(1776, 'localhost', () => {
  console.log('Connected');
  socket.write('syndr://admin:admin@localhost:1776/testdb\x04');
});

let buffer = '';
let stepCount = 0;
socket.on('data', (data) => {
  buffer += data.toString();
  const lines = buffer.split('\n');
  buffer = lines.pop() || '';
  
  lines.forEach((line) => {
    if (line.trim()) {
      stepCount++;
      console.log(`[${stepCount}] Received:`, line);
      
      // After auth, send SELECT query
      if (stepCount >= 2 && line.includes('"status":"success"')) {
        setTimeout(() => {
          console.log('\n>>> Sending: SELECT 1 as value');
          socket.write('SELECT 1 as value\x04');
        }, 200);
      }
    }
  });
});

socket.on('error', (err) => console.error('Error:', err));
socket.on('close', () => process.exit(0));
setTimeout(() => socket.end(), 3000);
