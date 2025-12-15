const net = require('net');

const socket = new net.Socket();

socket.connect(1776, 'localhost', () => {
  console.log('Connected to server');
  
  // Send connection string
  const connectionString = 'syndr://admin:admin@localhost:1776/testdb\x04';
  socket.write(connectionString);
});

let buffer = '';
socket.on('data', (data) => {
  buffer += data.toString();
  const lines = buffer.split('\n');
  buffer = lines.pop() || '';
  
  lines.forEach((line) => {
    if (line.trim()) {
      console.log('Received:', line);
      
      // After connected, try multi-line CREATE TABLE
      if (line.includes('"status":"success"')) {
        console.log('\nSending multi-line CREATE TABLE...');
        const createTable = `CREATE TABLE IF NOT EXISTS test_basic (
  id INTEGER PRIMARY KEY,
  value TEXT
)\x04`;
        console.log('Command:', JSON.stringify(createTable));
        socket.write(createTable);
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

// Keep alive for 5 seconds
setTimeout(() => {
  socket.end();
  process.exit(0);
}, 5000);
