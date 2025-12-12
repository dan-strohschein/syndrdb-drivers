import * as net from 'net';

// ============================================================================
// Interfaces and Types
// ============================================================================

/**
 * Connection parameters parsed from connection string
 */
export interface ConnectionParams {
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
}

/**
 * Options for configuring the SyndrDB client connection pool
 */
export interface ConnectionOptions {
  /** Maximum number of connections in the pool (default: 5) */
  maxConnections?: number;
  /** Idle timeout in milliseconds before closing unused connections (default: 30000) */
  idleTimeout?: number;
}

/**
 * Generic response structure from SyndrDB server
 */
export interface SyndrDBResponse<T = any> {
  success: boolean;
  data?: T;
  error?: SyndrDBError;
}

/**
 * Error details from SyndrDB server
 */
export interface SyndrDBError {
  code: string;
  type: string;
  message: string;
}

const CommandTerminator = "\x04";

// ============================================================================
// Custom Error Classes
// ============================================================================

/**
 * Base error class for SyndrDB connection errors
 */
export class SyndrDBConnectionError extends Error {
  public code?: string;
  public type?: string;

  constructor(message: string, code?: string, type?: string) {
    super(message);
    this.name = 'SyndrDBConnectionError';
    this.code = code;
    this.type = type;
    Error.captureStackTrace(this, this.constructor);
  }
}

/**
 * Error class for SyndrDB protocol errors
 */
export class SyndrDBProtocolError extends Error {
  public code?: string;
  public type?: string;

  constructor(message: string, code?: string, type?: string) {
    super(message);
    this.name = 'SyndrDBProtocolError';
    this.code = code;
    this.type = type;
    Error.captureStackTrace(this, this.constructor);
  }
}

/**
 * Error class for connection pool errors
 */
// export class SyndrDBPoolError extends Error {
//   public code?: string;
//   public type?: string;

//   constructor(message: string, code?: string, type?: string) {
//     super(message);
//     this.name = 'SyndrDBPoolError';
//     this.code = code;
//     this.type = type;
//     Error.captureStackTrace(this, this.constructor);
//   }
// }

// ============================================================================
// Connection String Parser
// ============================================================================

/**
 * Parses a SyndrDB connection string into connection parameters
 * @param connectionString - Format: syndrdb://<HOST>:<PORT>:<DATABASE>:<USERNAME>:<PASSWORD>;
 * @returns Parsed connection parameters
 * @throws {SyndrDBConnectionError} If connection string format is invalid
 */
export function parseConnectionString(connectionString: string): ConnectionParams {
  // Validate format
  if (!connectionString.startsWith('syndrdb://')) {
    throw new SyndrDBConnectionError(
      'Invalid connection string: must start with "syndrdb://"'
    );
  }

  if (!connectionString.endsWith(';')) {
    throw new SyndrDBConnectionError(
      'Invalid connection string: must end with ";"'
    );
  }

  // Remove protocol prefix and trailing semicolon
  const withoutProtocol = connectionString.slice('syndrdb://'.length, -1);

  // Split by colon
  const parts = withoutProtocol.split(':');

  if (parts.length !== 5) {
    throw new SyndrDBConnectionError(
      'Invalid connection string: expected format syndrdb://<HOST>:<PORT>:<DATABASE>:<USERNAME>:<PASSWORD>;'
    );
  }

  const [host, portStr, database, username, password] = parts;

  // Validate all parts are present
  if (!host || !portStr || !database || !username || !password) {
    throw new SyndrDBConnectionError(
      'Invalid connection string: all fields (host, port, database, username, password) are required'
    );
  }

  // Parse port
  const port = parseInt(portStr, 10);
  if (isNaN(port) || port <= 0 || port > 65535) {
    throw new SyndrDBConnectionError(
      `Invalid connection string: port must be a valid number between 1 and 65535, got "${portStr}"`
    );
  }

  return {
    host,
    port,
    database,
    username,
    password,
  };
}

// ============================================================================
// SyndrDBConnection Class
// ============================================================================

/**
 * Represents a single TCP connection to a SyndrDB server
 */
export class SyndrDBConnection {
  private static nextId = 1;
  public readonly id: number;
  private socket: net.Socket | null = null;
  private params: ConnectionParams;
  private buffer: string = '';
  private connected: boolean = false;
  private readTimeout: number = 10000; // 10 seconds

  /** Timestamp of last usage for idle timeout tracking */
  public lastUsedAt: number = Date.now();
  
  /** Flag indicating if this connection is currently in a transaction */
  public isInTransaction: boolean = false;

  constructor(params: ConnectionParams) {
    this.params = params;
    this.id = SyndrDBConnection.nextId++;
  }

  /**
   * Establishes TCP connection and authenticates with SyndrDB server
   * @returns True if connection successful
   * @throws {SyndrDBConnectionError} If connection fails
   */
  async connect(): Promise<boolean> {
    return new Promise((resolve, reject) => {
      this.socket = new net.Socket();

      // Set up error handler
      this.socket.on('error', (err) => {
        reject(new SyndrDBConnectionError(
          `Failed to connect to ${this.params.host}:${this.params.port}: ${err.message}`
        ));
      });

      // Connect to server
      this.socket.connect(this.params.port, this.params.host, async () => {
        try {
          // Build connection string
          const connString = `syndrdb://${this.params.host}:${this.params.port}:${this.params.database}:${this.params.username}:${this.params.password};\n`;
          
          // Send connection string (skip connected check since we're not authenticated yet)
          await this.sendCommand(connString, true);
          
          // Receive welcome response (skip connected check since we're not authenticated yet)
          const welcomeResponse = await this.receiveResponse(true);
          
          // Check for success code S0001
          if (!welcomeResponse.includes('S0001')) {
            reject(new SyndrDBConnectionError(
              `Authentication failed: unexpected welcome response "${welcomeResponse}"`
            ));
            return;
          }

          // Receive authentication success JSON response
          const authResponse = await this.receiveResponse(true);
          
          // Check if authentication was successful
          try {
            const authData = JSON.parse(authResponse);
            if (authData.status !== 'success') {
              reject(new SyndrDBConnectionError(
                `Authentication failed: ${authData.message || 'unknown error'}`
              ));
              return;
            }
          } catch (err) {
            // If not JSON, assume it's an error message
            reject(new SyndrDBConnectionError(
              `Authentication failed: unexpected response "${authResponse}"`
            ));
            return;
          }

          // Now we're fully authenticated
          this.connected = true;
          this.lastUsedAt = Date.now();
          resolve(true);
        } catch (err) {
          reject(err);
        }
      });
    });
  }

  /**
   * Sends a command to the SyndrDB server
   * @param command - Command string to send
   * @param skipConnectedCheck - Skip the connected check (for initial auth)
   * @throws {SyndrDBConnectionError} If not connected or send fails
   */
  async sendCommand(command: string, skipConnectedCheck: boolean = false): Promise<void> {
    if (!this.socket) {
      throw new SyndrDBConnectionError('Socket not initialized');
    }
    
    if (!skipConnectedCheck && !this.connected) {
      throw new SyndrDBConnectionError('Not connected to SyndrDB server');
    }

    // CommandTerminator is a non-printable character signaling end of complete command batch.
    // Using ASCII EOT (End of Transmission) for semantic clarity and protocol framing.
    // This allows multi-statement commands (migrations, transactions) to be processed as a unit.
    
    const cmd = command + CommandTerminator;



    return new Promise((resolve, reject) => {
      this.socket!.write(cmd, (err) => {
        if (err) {
          reject(new SyndrDBConnectionError(`Failed to send command: ${err.message}`));
        } else {
          this.lastUsedAt = Date.now();
          resolve();
        }
      });
    });
  }

  /**
   * Receives and parses a JSON response from the SyndrDB server
   * @param skipConnectedCheck - Skip the connected check (for initial auth)
   * @returns Parsed JSON response
   * @throws {SyndrDBConnectionError} If read fails or timeout occurs
   * @throws {SyndrDBProtocolError} If response indicates an error
   */
  async receiveResponse<T = any>(skipConnectedCheck: boolean = false): Promise<T> {
    if (!this.socket) {
      throw new SyndrDBConnectionError('Socket not initialized');
    }
    
    if (!skipConnectedCheck && !this.connected) {
      throw new SyndrDBConnectionError('Not connected to SyndrDB server');
    }

    return new Promise((resolve, reject) => {
      let timeoutHandle: NodeJS.Timeout;
     
      // First, check if there's already a complete line in the buffer
      const existingNewlineIndex = this.buffer.indexOf('\n');
      if (existingNewlineIndex !== -1) {
        const line = this.buffer.slice(0, existingNewlineIndex).trim();
        this.buffer = this.buffer.slice(existingNewlineIndex + 1);
        this.lastUsedAt = Date.now();
        


        // For initial connection, return raw response
        if (line.includes('S0001')) {
          resolve(line as any);
          return;
        }
        
        // Parse JSON response
        try {
          const parsed = JSON.parse(line);
          
          // Check if it's an authentication response (has 'message' field with auth text)
          if (parsed.status && parsed.message && typeof parsed.message === 'string' && 
              parsed.message.includes('Authentication')) {
            resolve(line as any); // Return raw JSON string for auth responses
            return;
          }
          
        // Check if it's a VALIDATE MIGRATION response
        if (parsed.status === 'success' && parsed.message && parsed.report) {
        // Return the whole parsed object
        resolve(parsed as any);
        return;
        }


          // Check if it's a data response with status field (like SHOW MIGRATIONS)
          if (parsed.status === 'success' && !parsed.message) {
            // Return the whole parsed object
            resolve(parsed as any);
            return;
          }
          
          // It's a standard SyndrDB response with success/error/data fields
          const response: SyndrDBResponse<T> = parsed;
          
          // Check for error in response
          if (!response.success && response.error) {
            reject(new SyndrDBProtocolError(
              response.error.message,
              response.error.code,
              response.error.type
            ));
          } else {
            resolve(response.data as T);
          }
        } catch (err) {
          // If not valid JSON, return raw line
          resolve(line as any);
        }
        return;
      }
      
      
      const onData = (chunk: Buffer) => {
        this.buffer += chunk.toString();
        // console.log("ret: ", this.buffer)
        // Check for complete line (newline-terminated)
        const newlineIndex = this.buffer.indexOf('\n');
       
        if (newlineIndex !== -1) {
          clearTimeout(timeoutHandle);
         // this.socket!.removeListener('data', onData);
          
          // Extract the line
          const line = this.buffer.slice(0, newlineIndex).trim();
          this.buffer = this.buffer.slice(newlineIndex + 1);
          this.lastUsedAt = Date.now();
          
          // For initial connection, return raw response
          if (line.includes('S0001')) {
            resolve(line as any);
            return;
          }
          
          // Parse JSON response
          try {
            const parsed = JSON.parse(line);
            
            // Check if it's an authentication response (has 'message' field with auth text)
            if (parsed.status && parsed.message && typeof parsed.message === 'string' && 
                parsed.message.includes('Authentication')) {
              resolve(line as any); // Return raw JSON string for auth responses
              return;
            }
            

       // Check if it's a VALIDATE MIGRATION response
        if (parsed.status === 'success' && parsed.message && parsed.report) {
        // Return the whole parsed object
        resolve(parsed as any);
        return;
        }


            // Check if it's a data response with status field (like SHOW MIGRATIONS)
            if (parsed.status === 'success' && !parsed.message) {
              // Return the whole parsed object
              resolve(parsed as any);
              return;
            }

            // Check if it's a data response with status field (like SHOW MIGRATIONS)
            if (parsed.status === 'success' && parsed.message) {
              // Return the whole parsed object
              resolve(parsed as any);
              return;
            }
            
            // Check if it's a SHOW BUNDLES response (has Result/ResultCount/ExecutionTimeMS)
            if (parsed.Result !== undefined && parsed.ResultCount !== undefined) {
              // Return the whole parsed object
              resolve(parsed as any);
              return;
            }
            
            // It's a standard SyndrDB response with success/error/data fields
            const response: SyndrDBResponse<T> = parsed;
            
            // Check for error in response
            if (!response.success && response.error) {
              reject(new SyndrDBProtocolError(
                response.error.message,
                response.error.code,
                response.error.type
              ));
            } else {
              resolve(response.data as T);
            }
          } catch (err) {
            // If not valid JSON, return raw line
            console.log('[receiveResponse] Not JSON, returning raw line:', line);
            resolve(line as any);
          }
        }
      };

      // Set timeout
      timeoutHandle = setTimeout(() => {
        this.socket!.removeListener('data', onData);
        reject(new SyndrDBConnectionError('Read timeout: no response received within 10 seconds'));
      }, this.readTimeout);

      // Listen for data
      this.socket!.once('data', onData);
    });
  }

  /**
   * Checks for incoming messages without blocking (1ms timeout)
   * @returns Message string if available, empty string otherwise
   */
  async checkForMessage(): Promise<string> {
    if (!this.socket || !this.connected) {
      return '';
    }

    return new Promise((resolve) => {
      const timeoutHandle = setTimeout(() => {
        this.socket!.removeListener('data', onData);
        resolve('');
      }, 1);

      const onData = (chunk: Buffer) => {
        clearTimeout(timeoutHandle);
        this.socket!.removeListener('data', onData);
        this.lastUsedAt = Date.now();
        resolve(chunk.toString());
      };

      this.socket!.once('data', onData);
    });
  }

  /**
   * Closes the TCP connection
   */
  async close(): Promise<void> {
    if (this.socket) {
      return new Promise((resolve) => {
        this.socket!.end(() => {
          this.socket = null;
          this.connected = false;
          this.buffer = '';
          resolve();
        });
      });
    }
  }

  /**
   * Checks if the connection is currently established
   */
  isConnected(): boolean {
    return this.connected && this.socket !== null;
  }
}

// // ============================================================================
// // SyndrDBConnectionPool Class
// // ============================================================================

// /**
//  * Manages a pool of SyndrDB connections with idle timeout handling
//  */
// export class SyndrDBConnectionPool {
//   private params: ConnectionParams;
//   private maxConnections: number;
//   private idleTimeout: number;
//   private availableConnections: SyndrDBConnection[] = [];
//   private activeConnections: Set<SyndrDBConnection> = new Set();
//   private idleCheckInterval: NodeJS.Timeout | null = null;

//   constructor(params: ConnectionParams, options: ConnectionOptions = {}) {
//     this.params = params;
//     this.maxConnections = options.maxConnections ?? 5;
//     this.idleTimeout = options.idleTimeout ?? 30000;
//   }

//   /**
//    * Starts the idle connection cleanup interval
//    */
//   private startIdleChecker(): void {
//     if (this.idleCheckInterval) {
//       return;
//     }

//     this.idleCheckInterval = setInterval(() => {
//       this.cleanupIdleConnections();
//     }, this.idleTimeout);
//   }

//   /**
//    * Stops the idle connection cleanup interval
//    */
//   private stopIdleChecker(): void {
//     if (this.idleCheckInterval) {
//       clearInterval(this.idleCheckInterval);
//       this.idleCheckInterval = null;
//     }
//   }

//   /**
//    * Cleans up idle connections that have exceeded the timeout
//    */
//   private async cleanupIdleConnections(): Promise<void> {
//     const now = Date.now();
//     const connectionsToClose: SyndrDBConnection[] = [];

//     // Find idle connections to close (skip those in transactions)
//     for (let i = this.availableConnections.length - 1; i >= 0; i--) {
//       const conn = this.availableConnections[i];
//       const idleTime = now - conn.lastUsedAt;

//       if (idleTime >= this.idleTimeout && !conn.isInTransaction) {
//         connectionsToClose.push(conn);
//         this.availableConnections.splice(i, 1);
//       }
//     }

//     // Close idle connections
//     await Promise.all(connectionsToClose.map(conn => conn.close()));
//   }

//   /**
//    * Acquires a connection from the pool
//    * @returns A ready-to-use connection
//    * @throws {SyndrDBPoolError} If unable to acquire connection
//    */
//   async acquire(): Promise<SyndrDBConnection> {
//     // Start idle checker if not already running
//     this.startIdleChecker();

//     // Try to get an available connection
//     if (this.availableConnections.length > 0) {
//       const conn = this.availableConnections.pop()!;
      
//       // Verify connection is still alive
//       if (!conn.isConnected()) {
//         // Try to reconnect
//         try {
//           await conn.connect();
//         } catch (err) {
//           // If reconnection fails, create a new one
//           return this.createNewConnection();
//         }
//       }
      
//       this.activeConnections.add(conn);
//       conn.lastUsedAt = Date.now();
//       return conn;
//     }

//     // Check if we can create a new connection
//     const totalConnections = this.availableConnections.length + this.activeConnections.size;
//     if (totalConnections < this.maxConnections) {
//       return this.createNewConnection();
//     }

//     // Pool is exhausted, wait for a connection to be released
//     throw new SyndrDBPoolError(
//       `Connection pool exhausted: maximum ${this.maxConnections} connections reached`
//     );
//   }

//   /**
//    * Creates a new connection and adds it to the active set
//    */
//   private async createNewConnection(): Promise<SyndrDBConnection> {
//     const conn = new SyndrDBConnection(this.params);
//     await conn.connect();
//     this.activeConnections.add(conn);
//     conn.lastUsedAt = Date.now();
//     return conn;
//   }

//   /**
//    * Releases a connection back to the pool
//    * @param connection - The connection to release
//    */
//   async release(connection: SyndrDBConnection): Promise<void> {
//     this.activeConnections.delete(connection);
    
//     // Update last used timestamp
//     connection.lastUsedAt = Date.now();
    
//     // Only return to pool if still connected
//     if (connection.isConnected()) {
//       this.availableConnections.push(connection);
//     } else {
//       // Connection is dead, close it
//       await connection.close();
//     }
//   }

//   /**
//    * Closes all connections in the pool
//    */
//   async closeAll(): Promise<void> {
//     this.stopIdleChecker();

//     // Close all connections
//     const allConnections = [
//       ...this.availableConnections,
//       ...Array.from(this.activeConnections)
//     ];

//     await Promise.all(allConnections.map(conn => conn.close()));

//     this.availableConnections = [];
//     this.activeConnections.clear();
//   }

//   /**
//    * Gets the current pool statistics
//    */
//   getStats() {
//     return {
//       available: this.availableConnections.length,
//       active: this.activeConnections.size,
//       total: this.availableConnections.length + this.activeConnections.size,
//       max: this.maxConnections,
//     };
//   }
// }

// ============================================================================
// SyndrDBClient Class
// ============================================================================

/**
 * Main client class for interacting with SyndrDB
 */
export class SyndrDBClient {
  private connection: SyndrDBConnection | null = null;
  private params: ConnectionParams | null = null;
  private activeTransaction: SyndrDBConnection | null = null;

  /**
   * Connects to SyndrDB server with a single persistent connection
   * @param connectionString - Format: syndrdb://<HOST>:<PORT>:<DATABASE>:<USERNAME>:<PASSWORD>;
   * @param options - Connection options (currently unused for single connection mode)
   * @throws {SyndrDBConnectionError} If connection string is invalid or connection fails
   */
  async connect(connectionString: string, options?: ConnectionOptions): Promise<void> {
    // Parse connection string
    this.params = parseConnectionString(connectionString);

    // Create and connect a single persistent connection
    this.connection = new SyndrDBConnection(this.params);
    await this.connection.connect();
  }

  /**
   * Closes the connection
   */
  async close(): Promise<void> {
    // If in transaction, rollback first
    if (this.activeTransaction) {
      await this.rollback();
    }

    if (this.connection) {
      await this.connection.close();
      this.connection = null;
    }
    this.params = null;
  }

  /**
   * Gets the connection (single connection mode)
   */
  private async getConnection(): Promise<{ connection: SyndrDBConnection; shouldRelease: boolean }> {
    if (!this.connection) {
      throw new SyndrDBConnectionError('Not connected to SyndrDB server. Call connect() first.');
    }

    if (this.activeTransaction) {
      return { connection: this.activeTransaction, shouldRelease: false };
    }

    return { connection: this.connection, shouldRelease: false };
  }

  /**
   * Begins a new transaction (binds a connection for the transaction lifetime)
   * @throws {SyndrDBProtocolError} Transaction support not yet implemented in protocol
   */
  async beginTransaction(): Promise<void> {
    // if (this.activeTransaction) {
    //   throw new SyndrDBPoolError('Transaction already active');
    // }

    if (!this.connection) {
      throw new SyndrDBConnectionError('Not connected to SyndrDB server. Call connect() first.');
    }

    // Use the single connection for the transaction
    this.activeTransaction = this.connection;
    this.activeTransaction.isInTransaction = true;

    // Send BEGIN command (placeholder)
    throw new SyndrDBProtocolError(
      'Transaction support not yet implemented in SyndrDB protocol',
      'NOT_IMPLEMENTED',
      'PROTOCOL_ERROR'
    );
  }

  /**
   * Commits the active transaction
   * @throws {SyndrDBProtocolError} Transaction support not yet implemented in protocol
   */
  async commit(): Promise<void> {
    if (!this.activeTransaction) {
       throw new SyndrDBConnectionError('No active transaction to commit');
    }

    try {
      // Send COMMIT command (placeholder)
      throw new SyndrDBProtocolError(
        'Transaction support not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
      // Clear the transaction flag
      this.activeTransaction.isInTransaction = false;
      this.activeTransaction = null;
    }
  }

  /**
   * Rolls back the active transaction
   * @throws {SyndrDBProtocolError} Transaction support not yet implemented in protocol
   */
  async rollback(): Promise<void> {
    if (!this.activeTransaction) {
      throw new SyndrDBConnectionError('No active transaction to rollback');
    }

    try {
      // Send ROLLBACK command (placeholder)
      throw new SyndrDBProtocolError(
        'Transaction support not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
      // Clear the transaction flag
      this.activeTransaction.isInTransaction = false;
      this.activeTransaction = null;
    }
  }

  /**
   * Executes a SQL query
   * @param sql - SQL query string
   * @returns Query results
   * @throws {SyndrDBConnectionError} If connection fails
   * @throws {SyndrDBProtocolError} If server returns an error
   */
  async query<T = any>(sql: string): Promise<T> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      // Send the query command to the server
      await connection.sendCommand(sql);
      
      // Receive and return the response
      const response = await connection.receiveResponse<T>();
      return response;
    } finally {
    }
  }

  /**
   * Executes a mutation
   * @param mutation - Mutation string
   * @returns Mutation result
   * @throws {SyndrDBConnectionError} If connection fails
   * @throws {SyndrDBProtocolError} If server returns an error
   */
  async mutate<T = any>(mutation: string): Promise<T> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      // Send the mutation command to the server
      await connection.sendCommand(mutation);
      
      // Receive and return the response
      const response = await connection.receiveResponse<T>();
      return response;
    } finally {
    }
  }

  /**
   * Executes a GraphQL query
   * @param query - GraphQL query string
   * @returns Query results
   * @throws {SyndrDBProtocolError} GraphQL support not yet implemented in protocol
   */
  async graphql<T = any>(query: string): Promise<T> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
        // Send the graphQL command to the server
        // Add the command prefix if not already present
        if(!query.trim().toLowerCase().startsWith('graphql::')) {
            query = `graphql::${query}`;
        }

      await connection.sendCommand(query);
      
      // Receive and return the response
      const response = await connection.receiveResponse<T>();
      return response;
    } finally {
    }
  }

  /**
   * Executes a migration
   * @param migration - Migration script
   * @throws {SyndrDBProtocolError} Migration support not yet implemented in protocol
   */
  async migrate(migration: string): Promise<void> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      // Send migrate command (placeholder)
      throw new SyndrDBProtocolError(
        'Migration support not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
    }
  }

  /**
   * Adds a new bundle to the schema
   * @param definition - Bundle definition
   * @throws {SyndrDBProtocolError} Schema operations not yet implemented in protocol
   */
  async addBundle(definition: Record<string, any>): Promise<void> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      throw new SyndrDBProtocolError(
        'AddBundle not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
    }
  }

  /**
   * Adds a new index to the schema
   * @param definition - Index definition
   * @throws {SyndrDBProtocolError} Schema operations not yet implemented in protocol
   */
  async addIndex(definition: Record<string, any>): Promise<void> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      throw new SyndrDBProtocolError(
        'AddIndex not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
    }
  }

  /**
   * Adds a new view to the schema
   * @param definition - View definition
   * @throws {SyndrDBProtocolError} Schema operations not yet implemented in protocol
   */
  async addView(definition: Record<string, any>): Promise<void> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      throw new SyndrDBProtocolError(
        'AddView not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
    }
  }

  /**
   * Modifies an existing bundle
   * @param name - Bundle name
   * @param definition - New bundle definition
   * @throws {SyndrDBProtocolError} Schema operations not yet implemented in protocol
   */
  async changeBundle(name: string, definition: Record<string, any>): Promise<void> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      throw new SyndrDBProtocolError(
        'ChangeBundle not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
    }
  }

  /**
   * Modifies an existing index
   * @param name - Index name
   * @param definition - New index definition
   * @throws {SyndrDBProtocolError} Schema operations not yet implemented in protocol
   */
  async changeIndex(name: string, definition: Record<string, any>): Promise<void> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      throw new SyndrDBProtocolError(
        'ChangeIndex not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
    }
  }

  /**
   * Modifies an existing view
   * @param name - View name
   * @param definition - New view definition
   * @throws {SyndrDBProtocolError} Schema operations not yet implemented in protocol
   */
  async changeView(name: string, definition: Record<string, any>): Promise<void> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      throw new SyndrDBProtocolError(
        'ChangeView not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
    }
  }

  /**
   * Removes a bundle from the schema
   * @param name - Bundle name
   * @throws {SyndrDBProtocolError} Schema operations not yet implemented in protocol
   */
  async dropBundle(name: string): Promise<void> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      throw new SyndrDBProtocolError(
        'DropBundle not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
    }
  }

  /**
   * Removes an index from the schema
   * @param name - Index name
   * @throws {SyndrDBProtocolError} Schema operations not yet implemented in protocol
   */
  async dropIndex(name: string): Promise<void> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      throw new SyndrDBProtocolError(
        'DropIndex not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
    }
  }

  /**
   * Removes a view from the schema
   * @param name - View name
   * @throws {SyndrDBProtocolError} Schema operations not yet implemented in protocol
   */
  async dropView(name: string): Promise<void> {
    const { connection, shouldRelease } = await this.getConnection();

    try {
      throw new SyndrDBProtocolError(
        'DropView not yet implemented in SyndrDB protocol',
        'NOT_IMPLEMENTED',
        'PROTOCOL_ERROR'
      );
    } finally {
    }
  }

  /**
   * Gets connection statistics (single connection mode)
   */
  getPoolStats() {
    if (!this.connection) {
      return null;
    }
    return {
      available: this.activeTransaction ? 0 : 1,
      active: this.activeTransaction ? 1 : 0,
      total: 1,
      max: 1,
    };
  }
}

// ============================================================================
// Re-export Migration Types and Extensions
// ============================================================================

export * from './schema/SchemaDefinition';
export * from './migrations/MigrationTypes';
export * from './migrations/SyndrDBMigrationError';
export { SchemaManager } from './schema/SchemaManager';
export { SchemaSerializer } from './schema/SchemaSerializer';
export { MigrationClient } from './migrations/MigrationClient';
export { MigrationHistory } from './migrations/MigrationHistory';
export { MigrationValidator } from './migrations/MigrationValidator';
export { MigrationConflictResolver, ConflictDiff } from './migrations/MigrationConflictResolver';
export { TypeGenerator, TypeGeneratorOptions } from './codegen/TypeGenerator';
export { TypeRegistry, CachedType } from './codegen/TypeRegistry';
export { TypeWriter, TypeWriterOptions } from './codegen/TypeWriter';
export { ResponseMapper } from './codegen/ResponseMapper';
export { SyndrDBClientMigrationExtensions, SyncSchemaOptions, GenerateTypesOptions } from './SyndrDBClientMigrationExtensions';
export { SyndrDBMigrationClient, createMigrationClient } from './migration-client';
