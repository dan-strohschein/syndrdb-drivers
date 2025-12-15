const net = require('net');

const socket = new net.Socket();

socket.connect(1776, 'localhost', () => {
  console.log('Connected to server');
  
  setTimeout(() => {
    // Send connection string
    console.log('Sending connection string...');
    const connectionString = 'syndr://admin:admin@localhost:1776/testdb\x04';
    socket.write(connectionString);
  }, 100);
});

let buffer = '';
let responseCount = 0;
socket.on('data', (data) => {
  buffer += data.toString();
  const lines = buffer.split('\n');
  buffer = lines.pop() || '';
  
  lines.forEach((line) => {
    if (line.trim()) {
      responseCount++;
      console.log(`[${responseCount}] Received:`, line);
      
      // After auth response, try simple query
      if (responseCount >= 2 && line.includes('"status":"success"')) {
        setTimeout(() => {
          console.log('\nSending simple SELECT query...');
          socket.write('SELECT 1 as value\x04');
        }, 500);
      }
    }
  });
});

socket.on('error', (err) => {
  console.error('Socket error:', err);
});

socket.on('close', () => {
  console.log('Connection closed');
});

setTimeout(() => {
  console.log('Timeout - closing socket');
  socket.end();
  process.exit(0);
}, 5000);
