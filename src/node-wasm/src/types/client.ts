/**
 * Client configuration options
 * Matches Go client.ClientOptions structure
 */
import type { PerformanceStats } from './performance';

export interface ClientOptions {
  /** Default timeout in milliseconds for operations. Default: 10000 */
  defaultTimeoutMs?: number;

  /** Enable verbose error serialization with full cause chains. Default: false */
  debugMode?: boolean;

  /** Maximum number of connection retry attempts. Default: 3 */
  maxRetries?: number;

  /** Minimum number of idle connections to maintain. Default: 1 */
  poolMinSize?: number;

  /** Maximum number of open connections. Default: 1 */
  poolMaxSize?: number;

  /** Duration after which idle connections are closed (ms). Default: 30000 */
  poolIdleTimeout?: number;

  /** How often to ping idle connections (ms). Default: 30000 */
  healthCheckInterval?: number;

  /** Maximum number of automatic reconnection attempts. Default: 10 */
  maxReconnectAttempts?: number;

  /** Minimum log level (DEBUG, INFO, WARN, ERROR). Default: INFO */
  logLevel?: LogLevel;

  /** Maximum number of prepared statements to cache. Default: 100 */
  preparedStatementCacheSize?: number;

  /** Maximum duration a transaction can remain active (ms). Default: 300000 */
  transactionTimeout?: number;
}

/**
 * Client connection states
 */
export enum ClientState {
  DISCONNECTED = 'DISCONNECTED',
  CONNECTING = 'CONNECTING',
  CONNECTED = 'CONNECTED',
  RECONNECTING = 'RECONNECTING',
  DISCONNECTING = 'DISCONNECTING',
}

/**
 * State transition information
 */
export interface StateTransition {
  /** Previous state */
  from: ClientState;

  /** New state */
  to: ClientState;

  /** Transition timestamp (Unix milliseconds) */
  timestamp: number;

  /** Time spent in previous state (milliseconds) */
  duration: number;

  /** Error message if transition failed */
  error?: string;

  /** Additional context about the transition */
  metadata?: Record<string, unknown>;
}

/**
 * Connection health information
 */
export interface ConnectionHealth {
  /** Whether the connection is healthy */
  isHealthy: boolean;

  /** Last ping latency in milliseconds */
  lastPingMs: number;

  /** Connection uptime in milliseconds */
  uptime: number;

  /** Current connection state */
  state: ClientState;

  /** Remote server address */
  remoteAddr?: string;

  /** Local client address */
  localAddr?: string;
}

/**
 * Debug information
 */
export interface DebugInfo {
  /** Driver version */
  version: string;

  /** WASM metadata */
  wasmMetadata: WASMMetadata;

  /** Recent state transitions */
  stateHistory: StateTransition[];

  /** Performance statistics */
  performanceStats: PerformanceStats;
}

/**
 * WASM module metadata
 */
export interface WASMMetadata {
  /** Go version used to compile WASM */
  goVersion: string;

  /** Whether DWARF debug info is present */
  hasDWARF: boolean;

  /** WASM module load time in milliseconds */
  loadTimeMs: number;

  /** WASM binary size in bytes */
  binarySize: number;

  /** SHA-256 checksum of WASM binary */
  checksum: string;
}

/**
 * Version information
 */
export interface VersionInfo {
  /** Wrapper version */
  version: string;

  /** Go core version */
  goVersion: string;

  /** Build timestamp */
  buildTime: string;

  /** Git commit hash */
  gitCommit: string;
}

/**
 * Log level enumeration
 */
export type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
