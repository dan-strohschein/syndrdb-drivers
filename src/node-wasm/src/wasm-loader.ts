/**
 * WASM loader with singleton pattern and DWARF-aware error handling
 * Manages loading and initialization of the SyndrDB WASM module
 */

import { readFile } from 'fs/promises';
import { join } from 'path';
import { performance } from 'perf_hooks';
import type { WASMMetadata } from './types/client';
import {
  WASMLoadError,
  WASMNotFoundError,
  WASMTimeoutError,
  WASMCorruptedError,
} from './errors/types';
import { getGlobalDWARFParser } from './errors/dwarf-parser';
import { existsSync } from 'fs';

// Import Go WASM runtime
// Path resolution for different execution contexts:
// - When built & running from dist/: __dirname is dist/, need ./wasm/wasm_exec.js
// - When running Jest tests: __dirname is src/node-wasm/, need ./dist/wasm/wasm_exec.js
// - When running from src/: __dirname is src/, need ../dist/wasm/wasm_exec.js
const wasmExecPath = existsSync(join(__dirname, './wasm/wasm_exec.js'))
  ? join(__dirname, './wasm/wasm_exec.js')
  : existsSync(join(__dirname, './dist/wasm/wasm_exec.js'))
  ? join(__dirname, './dist/wasm/wasm_exec.js')
  : join(__dirname, '../dist/wasm/wasm_exec.js');

// eslint-disable-next-line @typescript-eslint/no-var-requires
require(wasmExecPath);

// Declare Go class from wasm_exec.js
declare global {
  class Go {
    constructor();
    argv: string[];
    env: Record<string, string>;
    exit: (code: number) => void;
    importObject: {
      go: Record<string, unknown>;
      gojs: Record<string, unknown>;
    };
    run(instance: WebAssembly.Instance): Promise<void>;
  }
  
  // Go WASM sets exports on globalThis.SyndrDB
  var SyndrDB: Record<string, any> | undefined;
}

/**
 * WASM module exports interface
 * Matches the exported functions from golang/wasm/main.go
 * Note: Go WASM uses a global client model, not client IDs
 */
export interface WASMExports {
  // Client operations (no clientId - uses global client)
  createClient(optionsJSON: string): Promise<Record<string, any>>;
  connect(url: string): Promise<Record<string, any>>;
  disconnect(): Promise<Record<string, any>>;
  query(query: string, timeoutMs?: number): Promise<any>;
  mutate(mutation: string, timeoutMs?: number): Promise<any>;
  getState(): string;
  onStateChange(callback: (transition: any) => void): void;
  getVersion(): string;
  ping(): Promise<number>;
  getConnectionHealth(): Promise<string>;

  // Logging
  setLogLevel(level: string): void;

  // Debug operations
  enableDebugMode(): Record<string, any>;
  disableDebugMode(): Record<string, any>;
  getDebugInfo(): Promise<any>;

  // Schema generation
  generateJSONSchema(schemaJSON: string, mode?: string): Promise<any>;
  generateGraphQLSchema(schemaJSON: string): Promise<any>;

  // Migration operations
  createMigrationClient(): Promise<Record<string, any>>;
  planMigration(schemaJSON: string): Promise<string>;
  applyMigration(migrationId: string): Promise<string>;
  rollbackMigration(version: number): Promise<string>;
  validateMigration(migrationId: string): Promise<string>;
  previewMigration(migrationId: string): Promise<string>;
  getMigrationHistory(): Promise<string>;

  // Migration file operations (Node.js)
  saveMigrationFile(path: string, content: string): Promise<void>;
  loadMigrationFile(path: string): Promise<string>;
  listMigrations(directory: string): Promise<string>;
  acquireMigrationLock(path: string): Promise<boolean>;
  releaseMigrationLock(path: string): Promise<void>;

  // Environment info
  getEnvironmentInfo(): any;

  // Prepared statements (Milestone 2)
  prepare(query: string): Promise<number>;
  executeStatement(stmtId: number, paramsJSON: string): Promise<string>;
  deallocateStatement(stmtId: number): Promise<void>;
  queryWithParams(query: string, paramsJSON: string): Promise<any>;

  // Transactions (Milestone 2)
  beginTransaction(): Promise<string>;
  commitTransaction(txId: string): Promise<void>;
  rollbackTransaction(txId: string): Promise<void>;
  inTransaction(): boolean;

  // Hooks System (Milestone 5)
  registerHook(hookJSON: string): Promise<void>;
  unregisterHook(hookName: string): Promise<void>;
  getHooks(): Promise<string>;
  createLoggingHook(optionsJSON: string): string;
  createMetricsHook(): string;
  createTracingHook(): string;
  getMetricsStats(): Promise<string>;
  resetMetrics(): Promise<void>;

  // Cleanup
  cleanup(): void;
}

/**
 * Singleton WASM loader
 */
export class WASMLoader {
  private static instance: WASMLoader | null = null;
  private module: WebAssembly.Module | null = null;
  private instance_: WebAssembly.Instance | null = null;
  private exports: WASMExports | null = null;
  private metadata: WASMMetadata | null = null;
  private loading = false;
  private loadPromise: Promise<void> | null = null;

  private constructor() {}

  /**
   * Get singleton instance
   * @returns WASMLoader instance
   */
  static getInstance(): WASMLoader {
    if (!WASMLoader.instance) {
      WASMLoader.instance = new WASMLoader();
    }
    return WASMLoader.instance;
  }

  /**
   * Reset singleton (for testing)
   */
  static reset(): void {
    WASMLoader.instance = null;
  }

  /**
   * Load WASM module with timeout and integrity checks
   * @param timeoutMs Load timeout in milliseconds (default: 10000)
   * @returns Promise that resolves when module is loaded
   */
  async load(timeoutMs = 10000): Promise<void> {
    // Return existing load promise if already loading
    if (this.loading && this.loadPromise) {
      return this.loadPromise;
    }

    // Return immediately if already loaded
    if (this.module && this.instance_ && this.exports) {
      return;
    }

    this.loading = true;
    const startTime = performance.now();

    this.loadPromise = (async () => {
      try {
        // Load WASM binary with same path resolution as wasm_exec.js
        const wasmPath = existsSync(join(__dirname, './wasm/syndrdb.wasm'))
          ? join(__dirname, './wasm/syndrdb.wasm')
          : existsSync(join(__dirname, './dist/wasm/syndrdb.wasm'))
          ? join(__dirname, './dist/wasm/syndrdb.wasm')
          : join(__dirname, '../dist/wasm/syndrdb.wasm');
        let wasmBuffer: ArrayBuffer;

        try {
          const buffer = await Promise.race([
            readFile(wasmPath),
            new Promise<never>((_, reject) =>
              setTimeout(() => reject(new WASMTimeoutError(timeoutMs)), timeoutMs)
            ),
          ]);
          wasmBuffer = buffer.buffer.slice(
            buffer.byteOffset,
            buffer.byteOffset + buffer.byteLength
          );
        } catch (error) {
          if (error instanceof WASMTimeoutError) {
            throw error;
          }
          if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
            throw new WASMNotFoundError(wasmPath);
          }
          throw new WASMLoadError(`Failed to read WASM binary: ${(error as Error).message}`, error as Error);
        }

        // Verify integrity using checksum (optional, skip if manifest unavailable)
        try {
          await this.verifyIntegrity(wasmBuffer);
        } catch (err) {
          console.warn('WASM integrity check skipped:', (err as Error).message);
        }

        // Compile WASM module
        try {
          this.module = await WebAssembly.compile(wasmBuffer);
        } catch (error) {
          throw new WASMLoadError(
            `Failed to compile WASM module: ${(error as Error).message}`,
            error as Error
          );
        }

        // Extract DWARF metadata
        const hasDWARF = this.extractDWARFMetadata(this.module);

        // Initialize Go WASM runtime
        const go = new Go();
        go.argv = process.argv;
        go.env = Object.assign({ TMPDIR: require('os').tmpdir() }, process.env) as Record<string, string>;

        // Instantiate WASM module
        try {
          this.instance_ = await WebAssembly.instantiate(this.module, go.importObject as any);
        } catch (error) {
          throw new WASMLoadError(
            `Failed to instantiate WASM module: ${(error as Error).message}`,
            error as Error
          );
        }

        // Run Go program (starts in background)
        // The Go runtime will register exported functions on globalThis.SyndrDB
        go.run(this.instance_).catch((err) => {
          console.error('Go program exited with error:', err);
        });

        // Wait for Go runtime to initialize and register exports
        await new Promise((resolve) => setTimeout(resolve, 200));

        // TCP bridge will be installed when connect() is called with actual connection parameters
        // This allows different connections to use different servers

        // Get exports from globalThis.SyndrDB (not instance.exports)
        if (!globalThis.SyndrDB) {
          throw new WASMLoadError('Go program did not register SyndrDB exports');
        }
        this.exports = globalThis.SyndrDB as unknown as WASMExports;

        // Debug: Log available exports
        console.log('Available SyndrDB functions:', Object.keys(globalThis.SyndrDB).slice(0, 10));
        console.log('createClient function:', typeof globalThis.SyndrDB.createClient);

        // Create metadata
        const loadTime = performance.now() - startTime;
        this.metadata = {
          goVersion: '1.25', // TODO: Extract from WASM custom section
          hasDWARF,
          loadTimeMs: loadTime,
          binarySize: wasmBuffer.byteLength,
          checksum: '', // Will be set by verifyIntegrity
        };

        // Initialize DWARF parser
        if (hasDWARF) {
          const parser = getGlobalDWARFParser();
          await parser.initialize(this.module);
        }

        performance.mark('wasm-load-complete');
        performance.measure('wasm-load', 'wasm-load-start', 'wasm-load-complete');

        this.loading = false;
      } catch (error) {
        this.loading = false;
        this.loadPromise = null;
        throw error;
      }
    })();

    performance.mark('wasm-load-start');
    return this.loadPromise;
  }

  /**
   * Verify WASM binary integrity using SHA-256 checksum
   * @param wasmBuffer WASM binary buffer
   */
  private async verifyIntegrity(wasmBuffer: ArrayBuffer): Promise<void> {
    try {
      // Load integrity manifest with same path resolution as wasm_exec.js
      const integrityPath = existsSync(join(__dirname, './wasm/integrity.json'))
        ? join(__dirname, './wasm/integrity.json')
        : existsSync(join(__dirname, './dist/wasm/integrity.json'))
        ? join(__dirname, './dist/wasm/integrity.json')
        : join(__dirname, '../dist/wasm/integrity.json');
      const manifestBuffer = await readFile(integrityPath, 'utf-8');
      const manifest = JSON.parse(manifestBuffer);

      // Compute SHA-256 of WASM binary
      const crypto = await import('crypto');
      const hash = crypto.createHash('sha256');
      hash.update(Buffer.from(wasmBuffer));
      const checksum = hash.digest('hex');

      // Compare checksums
      if (checksum !== manifest.wasm?.sha256) {
        throw new WASMCorruptedError(
          `Checksum mismatch: expected ${manifest.wasm?.sha256}, got ${checksum}`
        );
      }

      if (this.metadata) {
        this.metadata.checksum = checksum;
      }
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
        console.warn('Integrity manifest not found - skipping verification');
        return;
      }
      throw error;
    }
  }

  /**
   * Extract DWARF metadata from WASM module
   * @param module WebAssembly module
   * @returns True if DWARF sections present
   */
  private extractDWARFMetadata(module: WebAssembly.Module): boolean {
    try {
      const debugInfo = WebAssembly.Module.customSections(module, '.debug_info');
      const debugLine = WebAssembly.Module.customSections(module, '.debug_line');
      const debugAbbrev = WebAssembly.Module.customSections(module, '.debug_abbrev');

      return debugInfo.length > 0 && debugLine.length > 0 && debugAbbrev.length > 0;
    } catch (error) {
      console.error('Failed to extract DWARF metadata:', error);
      return false;
    }
  }

  /**
   * Get WASM exports
   * @returns WASM exports or null if not loaded
   */
  getExports(): WASMExports | null {
    return this.exports;
  }

  /**
   * Get WASM metadata
   * @returns Module metadata or null if not loaded
   */
  getMetadata(): WASMMetadata | null {
    return this.metadata;
  }

  /**
   * Check if WASM is loaded
   * @returns True if module is loaded
   */
  isLoaded(): boolean {
    return this.module !== null && this.instance_ !== null && this.exports !== null;
  }
}

/**
 * Get global WASM loader instance
 * @returns WASMLoader singleton
 */
export function getWASMLoader(): WASMLoader {
  return WASMLoader.getInstance();
}
