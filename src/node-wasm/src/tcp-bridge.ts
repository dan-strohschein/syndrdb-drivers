import * as tls from 'tls';
import * as net from 'net';
import { EventEmitter } from 'events';

/**
 * Error codes matching protocol/errors.go
 */
export enum ErrorCode {
  // Connection errors (1001-1004)
  ErrorCodeConnectionFailed = 1001,
  ErrorCodeConnectionTimeout = 1002,
  ErrorCodeAuthFailed = 1003,
  ErrorCodeConnectionClosed = 1004,
  
  // Protocol errors (2001)
  ErrorCodeProtocolVersion = 2001,
  
  // Query errors (3001)
  ErrorCodeQueryFailed = 3001,
  
  // Backpressure (1010)
  ErrorCodeBackpressure = 1010,
  
  // Bridge errors (9001-9999)
  ErrorCodeBridgeNotReady = 9001,
  ErrorCodeBridgeBusy = 9002,
  ErrorCodeBridgeTimeout = 9003,
}

/**
 * Transport error matching protocol.TransportError
 */
export interface TransportError {
  code: ErrorCode;
  message: string;
  details?: Record<string, any>;
  isRetryable: boolean;
}

/**
 * TLS options for Node.js TCP connections
 */
export interface TLSOptions {
  enabled: boolean;
  rejectUnauthorized?: boolean;
  ca?: Buffer | string;
  cert?: Buffer | string;
  key?: Buffer | string;
  minVersion?: string;
  maxVersion?: string;
}

/**
 * TCP Bridge options
 */
export interface TCPBridgeOptions {
  host: string;
  port: number;
  tls?: TLSOptions;
  poolSize?: number;
  connectionTimeout?: number;
  idleTimeout?: number;
  healthCheckInterval?: number;
  maxQueueSize?: number;
}

/**
 * TCP connection wrapper
 */
class TCPConnection extends EventEmitter {
  private socket: net.Socket | tls.TLSSocket | null = null;
  private buffer: Buffer = Buffer.alloc(0);
  private messageQueue: Buffer[] = [];
  private isConnected: boolean = false;
  private lastUsed: number = Date.now();
  
  constructor(
    public readonly id: string,
    private readonly options: TCPBridgeOptions
  ) {
    super();
  }
  
  /**
   * Connect to the server
   */
  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.socket?.destroy();
        reject(this.createError(
          ErrorCode.ErrorCodeConnectionTimeout,
          `Connection timeout after ${this.options.connectionTimeout}ms`
        ));
      }, this.options.connectionTimeout || 5000);
      
      try {
        if (this.options.tls?.enabled) {
          this.socket = tls.connect({
            host: this.options.host,
            port: this.options.port,
            rejectUnauthorized: this.options.tls.rejectUnauthorized ?? true,
            ca: this.options.tls.ca,
            cert: this.options.tls.cert,
            key: this.options.tls.key,
            minVersion: this.options.tls.minVersion as any,
            maxVersion: this.options.tls.maxVersion as any,
          });
        } else {
          this.socket = net.connect({
            host: this.options.host,
            port: this.options.port,
          });
        }
        
        this.socket.on('connect', () => {
          clearTimeout(timeout);
          this.isConnected = true;
          this.emit('connect');
          resolve();
        });
        
        this.socket.on('data', (chunk: Buffer) => {
          this.handleData(chunk);
        });
        
        this.socket.on('error', (err: Error) => {
          clearTimeout(timeout);
          this.emit('error', err);
          reject(this.createError(
            ErrorCode.ErrorCodeConnectionFailed,
            err.message
          ));
        });
        
        this.socket.on('close', () => {
          this.isConnected = false;
          this.emit('close');
        });
        
      } catch (err) {
        clearTimeout(timeout);
        reject(this.createError(
          ErrorCode.ErrorCodeConnectionFailed,
          err instanceof Error ? err.message : String(err)
        ));
      }
    });
  }
  
  /**
   * Handle incoming data
   */
  private handleData(chunk: Buffer): void {
    this.buffer = Buffer.concat([this.buffer, chunk]);
    
    // Split on EOT (0x04)
    while (true) {
      const eotIndex = this.buffer.indexOf(0x04);
      if (eotIndex === -1) break;
      
      const message = this.buffer.slice(0, eotIndex);
      this.buffer = this.buffer.slice(eotIndex + 1);
      
      this.messageQueue.push(message);
      this.emit('message', message);
    }
  }
  
  /**
   * Send data to the server
   */
  async send(data: Uint8Array): Promise<void> {
    if (!this.isConnected || !this.socket) {
      throw this.createError(
        ErrorCode.ErrorCodeConnectionClosed,
        'Connection is not established'
      );
    }
    
    return new Promise((resolve, reject) => {
      this.socket!.write(Buffer.from(data), (err) => {
        if (err) {
          reject(this.createError(
            ErrorCode.ErrorCodeConnectionFailed,
            err.message
          ));
        } else {
          this.lastUsed = Date.now();
          resolve();
        }
      });
    });
  }
  
  /**
   * Receive a message from the queue
   */
  receive(): Uint8Array | null {
    const message = this.messageQueue.shift();
    if (message) {
      this.lastUsed = Date.now();
      return new Uint8Array(message);
    }
    return null;
  }
  
  /**
   * Check if connection is healthy
   */
  isHealthy(): boolean {
    if (!this.isConnected || !this.socket) {
      return false;
    }
    
    // Check if idle for too long
    const idleTime = Date.now() - this.lastUsed;
    if (this.options.idleTimeout && idleTime > this.options.idleTimeout) {
      return false;
    }
    
    return true;
  }
  
  /**
   * Close the connection
   */
  close(): void {
    if (this.socket) {
      this.socket.destroy();
      this.socket = null;
    }
    this.isConnected = false;
    this.messageQueue = [];
  }
  
  /**
   * Create a transport error
   */
  private createError(code: ErrorCode, message: string): TransportError {
    return {
      code,
      message,
      isRetryable: code === ErrorCode.ErrorCodeBridgeBusy || 
                   code === ErrorCode.ErrorCodeConnectionTimeout,
    };
  }
}

/**
 * TCP Bridge for WASM
 * Manages a pool of TCP connections and provides a bridge for Go WASM code
 */
export class TCPBridge {
  private connections: Map<string, TCPConnection> = new Map();
  private availableConnections: Set<string> = new Set();
  private healthCheckTimer: NodeJS.Timeout | null = null;
  private isReady: boolean = false;
  
  constructor(private readonly options: TCPBridgeOptions) {
    if (!options.poolSize) {
      this.options.poolSize = 5;
    }
    if (!options.maxQueueSize) {
      this.options.maxQueueSize = 100;
    }
    if (!options.healthCheckInterval) {
      this.options.healthCheckInterval = 30000; // 30 seconds
    }
  }
  
  /**
   * Initialize the bridge
   */
  async initialize(): Promise<void> {
    // Create initial pool of connections
    const promises: Promise<void>[] = [];
    for (let i = 0; i < (this.options.poolSize || 5); i++) {
      const connId = `conn-${i}`;
      promises.push(this.createConnection(connId));
    }
    
    await Promise.all(promises);
    
    // Start health check timer
    this.startHealthChecks();
    
    this.isReady = true;
  }
  
  /**
   * Request a connection (called from Go WASM)
   */
  goRequestConnection(connId: string): Uint8Array | TransportError {
    if (!this.isReady) {
      return this.createError(
        ErrorCode.ErrorCodeBridgeNotReady,
        'Bridge is not initialized'
      );
    }
    
    // Try to get an available connection
    let targetConnId: string | undefined;
    for (const id of this.availableConnections) {
      targetConnId = id;
      break;
    }
    
    if (!targetConnId) {
      // No available connections
      if (this.connections.size >= (this.options.poolSize || 5)) {
        return this.createError(
          ErrorCode.ErrorCodeBackpressure,
          `Connection pool is at capacity (${this.connections.size})`
        );
      }
      
      // Create a new connection synchronously (will connect in background)
      this.createConnection(connId).catch((err) => {
        console.error(`Failed to create connection ${connId}:`, err);
      });
      
      return new Uint8Array([0]); // Success indicator
    }
    
    // Mark connection as in use
    this.availableConnections.delete(targetConnId);
    
    return new Uint8Array([0]); // Success indicator
  }
  
  /**
   * Send data via a connection (called from Go WASM)
   */
  goSend(connIdOrData: string | Uint8Array, data?: Uint8Array): TransportError | void {
    let connId: string;
    let sendData: Uint8Array;
    
    // Handle both signatures: goSend(data) and goSend(connId, data)
    if (typeof connIdOrData === 'string') {
      connId = connIdOrData;
      sendData = data!;
    } else {
      // Find any available connection
      const availConn = this.availableConnections.values().next().value;
      if (!availConn) {
        return this.createError(
          ErrorCode.ErrorCodeBridgeBusy,
          'No available connections'
        );
      }
      connId = availConn;
      sendData = connIdOrData;
    }
    
    const conn = this.connections.get(connId);
    if (!conn) {
      return this.createError(
        ErrorCode.ErrorCodeConnectionClosed,
        `Connection ${connId} not found`
      );
    }
    
    conn.send(sendData).catch((err) => {
      console.error(`Failed to send data on ${connId}:`, err);
    });
  }
  
  /**
   * Receive data from a connection (called from Go WASM)
   */
  goReceive(connId?: string): Uint8Array | TransportError {
    let targetConnId: string;
    
    if (connId) {
      targetConnId = connId;
    } else {
      // Find any connection with messages
      for (const [id, conn] of this.connections) {
        const data = conn.receive();
        if (data) {
          return data;
        }
      }
      return this.createError(
        ErrorCode.ErrorCodeBridgeBusy,
        'No messages available'
      );
    }
    
    const conn = this.connections.get(targetConnId);
    if (!conn) {
      return this.createError(
        ErrorCode.ErrorCodeConnectionClosed,
        `Connection ${targetConnId} not found`
      );
    }
    
    const data = conn.receive();
    if (!data) {
      return this.createError(
        ErrorCode.ErrorCodeBridgeBusy,
        'No messages available'
      );
    }
    
    return data;
  }
  
  /**
   * Release a connection (called from Go WASM)
   */
  goReleaseConnection(connId: string): void {
    const conn = this.connections.get(connId);
    if (conn && conn.isHealthy()) {
      this.availableConnections.add(connId);
    } else if (conn) {
      // Connection is unhealthy, close it
      conn.close();
      this.connections.delete(connId);
    }
  }
  
  /**
   * Check connection health (called from Go WASM)
   */
  goCheckConnectionHealth(connId: string): boolean {
    const conn = this.connections.get(connId);
    return conn ? conn.isHealthy() : false;
  }
  
  /**
   * Create a new connection
   */
  private async createConnection(connId: string): Promise<void> {
    const conn = new TCPConnection(connId, this.options);
    
    conn.on('error', (err: Error) => {
      console.error(`Connection ${connId} error:`, err);
      this.connections.delete(connId);
      this.availableConnections.delete(connId);
    });
    
    conn.on('close', () => {
      this.connections.delete(connId);
      this.availableConnections.delete(connId);
    });
    
    await conn.connect();
    
    this.connections.set(connId, conn);
    this.availableConnections.add(connId);
  }
  
  /**
   * Start health check timer
   */
  private startHealthChecks(): void {
    this.healthCheckTimer = setInterval(() => {
      for (const [connId, conn] of this.connections) {
        if (!conn.isHealthy()) {
          console.log(`Connection ${connId} is unhealthy, removing`);
          conn.close();
          this.connections.delete(connId);
          this.availableConnections.delete(connId);
          
          // Notify Go WASM
          if ((globalThis as any).SyndrDBBridge?.goMarkConnectionUnhealthy) {
            (globalThis as any).SyndrDBBridge.goMarkConnectionUnhealthy(connId, false);
          }
        }
      }
    }, this.options.healthCheckInterval);
  }
  
  /**
   * Close all connections
   */
  close(): void {
    if (this.healthCheckTimer) {
      clearInterval(this.healthCheckTimer);
      this.healthCheckTimer = null;
    }
    
    for (const conn of this.connections.values()) {
      conn.close();
    }
    
    this.connections.clear();
    this.availableConnections.clear();
    this.isReady = false;
  }
  
  /**
   * Create a transport error
   */
  private createError(code: ErrorCode, message: string): TransportError {
    return {
      code,
      message,
      isRetryable: code === ErrorCode.ErrorCodeBridgeBusy || 
                   code === ErrorCode.ErrorCodeConnectionTimeout,
    };
  }
  
  /**
   * Get pool statistics
   */
  getStats(): Record<string, any> {
    return {
      totalConnections: this.connections.size,
      availableConnections: this.availableConnections.size,
      poolSize: this.options.poolSize,
      isReady: this.isReady,
    };
  }
}

/**
 * Create and install the TCP bridge on globalThis
 */
export function installTCPBridge(options: TCPBridgeOptions): TCPBridge {
  const bridge = new TCPBridge(options);
  
  // Install on globalThis for Go WASM to access
  (globalThis as any).SyndrDBBridge = {
    goRequestConnection: bridge.goRequestConnection.bind(bridge),
    goSend: bridge.goSend.bind(bridge),
    goReceive: bridge.goReceive.bind(bridge),
    goReleaseConnection: bridge.goReleaseConnection.bind(bridge),
    goCheckConnectionHealth: bridge.goCheckConnectionHealth.bind(bridge),
    getStats: bridge.getStats.bind(bridge),
    close: bridge.close.bind(bridge),
  };
  
  return bridge;
}
