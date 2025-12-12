/**
 * Basic usage example for SyndrDB Node.js driver
 */

import { SyndrDBClient } from '../src/index';

async function main() {
  // Create a new client instance
  const client = new SyndrDBClient();

  try {
    // Connect to SyndrDB with connection pooling
    console.log('Connecting to SyndrDB...');
    await client.connect('syndrdb://localhost:1776:mydb:admin:password;', {
      maxConnections: 5,    // Maximum 5 concurrent connections
      idleTimeout: 30000,   // Close idle connections after 30 seconds
    });
    console.log('Connected successfully!');

    // Check pool statistics
    const poolStats = client.getPoolStats();
    console.log('Pool stats:', poolStats);

    // Example: Query (will throw NOT_IMPLEMENTED until protocol is ready)
    try {
      const users = await client.query('SELECT * FROM users');
      console.log('Users:', users);
    } catch (error: any) {
      console.log('Expected error - query not implemented yet:', error.message);
    }

    // Example: Transaction (will throw NOT_IMPLEMENTED until protocol is ready)
    try {
      await client.beginTransaction();
      await client.mutate('INSERT INTO users (name) VALUES ("John")');
      await client.commit();
    } catch (error: any) {
      console.log('Expected error - transactions not implemented yet:', error.message);
    }

    // Example: Schema operations (will throw NOT_IMPLEMENTED until protocol is ready)
    try {
      await client.addBundle({
        name: 'users_bundle',
        schema: {
          id: 'uuid',
          name: 'string',
          email: 'string',
        },
      });
    } catch (error: any) {
      console.log('Expected error - schema operations not implemented yet:', error.message);
    }

  } catch (error) {
    console.error('Error:', error);
  } finally {
    // Always close the client when done
    console.log('Closing connection...');
    await client.close();
    console.log('Connection closed.');
  }
}

// Run the example
main().catch(console.error);
