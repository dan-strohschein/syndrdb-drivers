/**
 * Node.js TCP connection handler for SyndrDB
 * Handles raw TCP communication - WASM only handles protocol logic
 */

import * as net from 'net';
import * as tls from 'tls';
import { EventEmitter } from 'events';

export interface NodeConnectionOptions {
  host: string;
  port: number;
  connectionTimeout?: number;
  tls?: {
    enabled: boolean;
    rejectUnauthorized?: boolean;
    ca?: string;
    cert?: string;
    key?: string;
  };
}

/**
 * Node.js TCP connection that handles raw socket I/O
 * Go WASM layer handles protocol logic (building commands, parsing responses)
 */
export class NodeConnection extends EventEmitter {
  private socket: net.Socket | tls.TLSSocket | null = null;
  private buffer: string = '';
  private messageQueue: string[] = [];
  private waitingForMessage: ((message: string) => void)[] = [];
  private connected: boolean = false;

  constructor(private options: NodeConnectionOptions) {
    super();
  }

  /**
   * Connect to the server and perform handshake
   * @param connectionString Full connection string to send to server
   */
  async connect(connectionString: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const timeout = this.options.connectionTimeout || 10000;
      let timeoutHandle: NodeJS.Timeout;

      const cleanup = () => {
        if (timeoutHandle) clearTimeout(timeoutHandle);
      };

      console.log('[NodeConnection] Attempting to connect to', this.options.host, this.options.port);

      timeoutHandle = setTimeout(() => {
        console.log('[NodeConnection] Connection timeout!');
        reject(new Error(`Connection timeout after ${timeout}ms`));
        if (this.socket) {
          this.socket.destroy();
        }
      }, timeout);

      // Create socket based on TLS settings
      if (this.options.tls?.enabled) {
        console.log('[NodeConnection] Creating TLS socket');
        this.socket = tls.connect({
          host: this.options.host,
          port: this.options.port,
          rejectUnauthorized: this.options.tls.rejectUnauthorized ?? true,
          ca: this.options.tls.ca,
          cert: this.options.tls.cert,
          key: this.options.tls.key,
        });
      } else {
        console.log('[NodeConnection] Creating plain TCP socket');
        this.socket = net.connect({
          host: this.options.host,
          port: this.options.port,
        });
      }

      console.log('[NodeConnection] Socket created, setting up event handlers');

      // Set up data handler FIRST, before connection completes
      this.socket.on('data', (data) => {
        console.log('[NodeConnection] Received data:', data.length, 'bytes:', data.toString().substring(0, 100));
        this.handleData(data);
      });

      this.socket.on('error', (err) => {
        console.log('[NodeConnection] Socket error:', err.message);
        cleanup();
        this.connected = false;
        this.emit('error', err);
        reject(err);
      });

      this.socket.on('close', () => {
        console.log('[NodeConnection] Socket closed');
        cleanup();
        this.connected = false;
        this.emit('close');
      });

      this.socket.on('connect', () => {
        console.log('[NodeConnection] Socket connected');
        this.connected = true;
        this.emit('connect');

        // Send connection string with EOT terminator
        const message = connectionString + '\x04';
        console.log('[NodeConnection] Sending connection string:', connectionString);
        this.socket!.write(message, (err) => {
          if (err) {
            console.log('[NodeConnection] Write error:', err.message);
            cleanup();
            reject(new Error(`Failed to send connection string: ${err.message}`));
            return;
          }

          console.log('[NodeConnection] Connection string sent, waiting for welcome...');
          // Wait for two messages: welcome (S0001) and auth response
          this.receiveMessage()
            .then((welcome) => {
              console.log('[NodeConnection] Received welcome:', welcome);
              if (!welcome.includes('S0001')) {
                cleanup();
                reject(new Error(`Authentication failed: unexpected welcome response "${welcome}"`));
                return Promise.reject(new Error('Auth failed'));
              }

              // Receive auth response
              console.log('[NodeConnection] Waiting for auth response...');
              return this.receiveMessage();
            })
            .then((authResp) => {
              console.log('[NodeConnection] Received auth response:', authResp);
              cleanup();
              
              // Parse auth response
              try {
                const authData = JSON.parse(authResp);
                // Server sends {"status":"success"} or {"status":"error"}
                if (authData.status !== 'success') {
                  reject(new Error(`Authentication failed: ${authData.message || authData.error || 'unknown error'}`));
                  return;
                }
                console.log('[NodeConnection] Authentication successful!');
                resolve();
              } catch (e) {
                reject(new Error(`Failed to parse auth response: ${authResp}`));
              }
            })
            .catch((err) => {
              console.log('[NodeConnection] Error during handshake:', err);
              cleanup();
              reject(err);
            });
        });
      });
    });
  }

  /**
   * Handle incoming data from socket
   */
  private handleData(data: Buffer): void {
    console.log('[NodeConnection] handleData called with', data.length, 'bytes');
    this.buffer += data.toString();
    console.log('[NodeConnection] Buffer now contains:', this.buffer.length, 'chars');

    // Server sends responses as newline-delimited, NOT EOT-delimited
    // Only client commands use EOT terminator
    const messages = this.buffer.split('\n');
    console.log('[NodeConnection] Split into', messages.length, 'parts');
    
    // Last element is incomplete message (or empty if buffer ended with newline)
    this.buffer = messages.pop() || '';

    // Process complete messages
    for (const message of messages) {
      const trimmed = message.trim();
      console.log('[NodeConnection] Processing message:', trimmed.substring(0, 100));
      if (trimmed) {
        if (this.waitingForMessage.length > 0) {
          console.log('[NodeConnection] Resolving waiting promise with message');
          const resolver = this.waitingForMessage.shift()!;
          resolver(trimmed);
        } else {
          console.log('[NodeConnection] Queueing message');
          this.messageQueue.push(trimmed);
        }
      }
    }
  }

  /**
   * Receive a single message from the server
   */
  private receiveMessage(): Promise<string> {
    // Check if we have queued messages
    if (this.messageQueue.length > 0) {
      return Promise.resolve(this.messageQueue.shift()!);
    }

    // Wait for next message
    return new Promise((resolve) => {
      this.waitingForMessage.push(resolve);
    });
  }

  /**
   * Send a command to the server
   * @param command Command string (will be terminated with EOT)
   */
  async sendCommand(command: string): Promise<void> {
    if (!this.socket || !this.connected) {
      throw new Error('Not connected');
    }

    return new Promise((resolve, reject) => {
      const message = command + '\x04';
      this.socket!.write(message, (err) => {
        if (err) {
          reject(new Error(`Failed to send command: ${err.message}`));
        } else {
          resolve();
        }
      });
    });
  }

  /**
   * Receive a response from the server
   */
  async receiveResponse(): Promise<string> {
    if (!this.socket || !this.connected) {
      throw new Error('Not connected');
    }

    return this.receiveMessage();
  }

  /**
   * Close the connection
   */
  async close(): Promise<void> {
    if (this.socket) {
      return new Promise((resolve) => {
        this.socket!.once('close', () => resolve());
        this.socket!.end();
      });
    }
  }

  /**
   * Check if connection is alive
   */
  isConnected(): boolean {
    return this.connected && this.socket !== null;
  }
}
