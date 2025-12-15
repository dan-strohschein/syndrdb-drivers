/**
 * Unit tests for SyndrDBClient
 */

import { SyndrDBClient } from '../../src/client';
import { ConnectionError } from '../../src/errors';
import { ClientState } from '../../src/types/client';

// Mock WASM loader
jest.mock('../../src/wasm-loader', () => ({
  getWASMLoader: jest.fn(() => ({
    load: jest.fn().mockResolvedValue(undefined),
    getExports: jest.fn(() => mockWASMExports),
    getMetadata: jest.fn(() => ({
      goVersion: '1.25',
      hasDWARF: true,
      loadTimeMs: 100,
      binarySize: 1000000,
      checksum: 'abc123',
    })),
    isLoaded: jest.fn(() => true),
  })),
  WASMLoader: {
    getInstance: jest.fn(),
    reset: jest.fn(),
  },
}));

// Mock WASM exports
const mockWASMExports = {
  createClient: jest.fn(() => 1),
  connect: jest.fn().mockResolvedValue(undefined),
  disconnect: jest.fn().mockResolvedValue(undefined),
  query: jest.fn().mockResolvedValue(JSON.stringify({ data: [] })),
  mutate: jest.fn().mockResolvedValue(JSON.stringify({ id: 1 })),
  getState: jest.fn(() => 'CONNECTED'),
  ping: jest.fn().mockResolvedValue(5),
  getConnectionHealth: jest.fn(() =>
    JSON.stringify({
      isHealthy: true,
      lastPingMs: 5,
      uptime: 1000,
      state: 'CONNECTED',
    })
  ),
  planMigration: jest.fn().mockResolvedValue(JSON.stringify({ migration: {} })),
  applyMigration: jest.fn().mockResolvedValue(JSON.stringify({ id: 'mig1' })),
  rollbackMigration: jest.fn().mockResolvedValue(JSON.stringify({ version: 0 })),
  validateMigration: jest.fn().mockResolvedValue(JSON.stringify({ isValid: true })),
  getMigrationHistory: jest.fn().mockResolvedValue(
    JSON.stringify({ currentVersion: 1, migrations: [] })
  ),
  generateJSONSchema: jest.fn().mockResolvedValue('{}'),
  generateGraphQLSchema: jest.fn().mockResolvedValue('type Query {}'),
  beginTransaction: jest.fn().mockResolvedValue(1),
  commitTransaction: jest.fn().mockResolvedValue(undefined),
  rollbackTransaction: jest.fn().mockResolvedValue(undefined),
  prepare: jest.fn().mockResolvedValue(1),
  executeStatement: jest.fn().mockResolvedValue(JSON.stringify({ result: true })),
  deallocateStatement: jest.fn().mockResolvedValue(undefined),
  registerHook: jest.fn().mockResolvedValue(undefined),
  unregisterHook: jest.fn().mockResolvedValue(undefined),
  getMetricsStats: jest.fn().mockResolvedValue(
    JSON.stringify({
      totalOperations: 100,
      successCount: 95,
      errorCount: 5,
    })
  ),
  setLogLevel: jest.fn(),
  enableDebugMode: jest.fn(),
  disableDebugMode: jest.fn(),
  getDebugInfo: jest.fn(() => JSON.stringify({ version: '2.0.0' })),
};

describe('SyndrDBClient', () => {
  let client: SyndrDBClient;

  beforeEach(() => {
    jest.clearAllMocks();
    client = new SyndrDBClient({
      debugMode: false,
      defaultTimeoutMs: 5000,
    });
  });

  describe('constructor', () => {
    it('should create client with default options', () => {
      const defaultClient = new SyndrDBClient();
      expect(defaultClient).toBeInstanceOf(SyndrDBClient);
    });

    it('should create client with custom options', () => {
      const customClient = new SyndrDBClient({
        debugMode: true,
        maxRetries: 5,
        logLevel: 'DEBUG',
      });
      expect(customClient).toBeInstanceOf(SyndrDBClient);
    });
  });

  describe('initialize', () => {
    it('should initialize client and load WASM', async () => {
      await client.initialize();

      expect(mockWASMExports.createClient).toHaveBeenCalled();
      expect(mockWASMExports.setLogLevel).toHaveBeenCalled();
    });

    it('should enable debug mode when configured', async () => {
      const debugClient = new SyndrDBClient({ debugMode: true });
      await debugClient.initialize();

      expect(mockWASMExports.enableDebugMode).toHaveBeenCalled();
    });

    it('should throw ConnectionError on failure', async () => {
      const { getWASMLoader } = require('../../src/wasm-loader');
      getWASMLoader.mockReturnValueOnce({
        load: jest.fn().mockRejectedValue(new Error('Load failed')),
        getExports: jest.fn(() => null),
      });

      await expect(client.initialize()).rejects.toThrow();
    });
  });

  describe('state management', () => {
    it('should start in DISCONNECTED state', () => {
      expect(client.getState()).toBe('DISCONNECTED');
    });

    it('should transition to CONNECTED on connect', async () => {
      await client.initialize();
      await client.connect('localhost:1776');

      expect(client.getState()).toBe('CONNECTED');
    });

    it('should emit stateChange events', async () => {
      const stateChangeSpy = jest.fn();
      client.on('stateChange', stateChangeSpy);

      await client.initialize();
      await client.connect('localhost:1776');

      expect(stateChangeSpy).toHaveBeenCalled();
      const transition = stateChangeSpy.mock.calls[0][0];
      expect(transition.from).toBe('DISCONNECTED');
      expect(transition.to).toBe('CONNECTING');
    });

    it('should track state history', async () => {
      await client.initialize();
      await client.connect('localhost:1776');

      const debugInfo = client.getDebugInfo();
      expect(debugInfo.stateHistory.length).toBeGreaterThan(0);
    });
  });

  describe('connect', () => {
    beforeEach(async () => {
      await client.initialize();
    });

    it('should connect to database', async () => {
      await client.connect('localhost:1776');

      expect(mockWASMExports.connect).toHaveBeenCalledWith(1, 'localhost:1776');
      expect(client.getState()).toBe('CONNECTED');
    });

    it('should throw if not initialized', async () => {
      const uninitializedClient = new SyndrDBClient();
      await expect(uninitializedClient.connect('localhost:1776')).rejects.toThrow(
        ConnectionError
      );
    });

    it('should transition to DISCONNECTED on error', async () => {
      mockWASMExports.connect.mockRejectedValueOnce(new Error('Connection failed'));

      await expect(client.connect('localhost:1776')).rejects.toThrow();
      expect(client.getState()).toBe('DISCONNECTED');
    });
  });

  describe('disconnect', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should disconnect from database', async () => {
      await client.disconnect();

      expect(mockWASMExports.disconnect).toHaveBeenCalledWith(1);
      expect(client.getState()).toBe('DISCONNECTED');
    });
  });

  describe('query', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should execute query', async () => {
      const result = await client.query('SELECT * FROM "users"');

      expect(mockWASMExports.query).toHaveBeenCalledWith(
        1,
        'SELECT * FROM "users"',
        JSON.stringify([])
      );
      expect(result).toEqual({ data: [] });
    });

    it('should execute query with parameters', async () => {
      await client.query('SELECT * FROM "users" WHERE "id" == ?', [123]);

      expect(mockWASMExports.query).toHaveBeenCalledWith(
        1,
        'SELECT * FROM "users" WHERE "id" == ?',
        JSON.stringify([123])
      );
    });

    it('should throw if not connected', async () => {
      await client.disconnect();
      await expect(client.query('SELECT 1')).rejects.toThrow(ConnectionError);
    });

    it('should wrap WASM errors', async () => {
      mockWASMExports.query.mockRejectedValueOnce(
        new Error('[QUERY_ERROR] Invalid query')
      );

      await expect(client.query('INVALID')).rejects.toThrow();
    });
  });

  describe('mutate', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should execute mutation', async () => {
      const result = await client.mutate('ADD DOCUMENT TO BUNDLE "users" WITH ({"name" = ?})', ['Alice']);

      expect(mockWASMExports.mutate).toHaveBeenCalledWith(
        1,
        'ADD DOCUMENT TO BUNDLE "users" WITH ({"name" = ?})',
        JSON.stringify(['Alice'])
      );
      expect(result).toEqual({ id: 1 });
    });
  });

  describe('ping', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should ping database', async () => {
      const latency = await client.ping();

      expect(mockWASMExports.ping).toHaveBeenCalledWith(1);
      expect(latency).toBe(5);
    });
  });

  describe('getConnectionHealth', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should get connection health', async () => {
      const health = await client.getConnectionHealth();

      expect(health.isHealthy).toBe(true);
      expect(health.lastPingMs).toBe(5);
      expect(health.uptime).toBe(1000);
    });
  });

  describe('migrations', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should plan migration', async () => {
      const plan = await client.planMigration({ bundles: [] });

      expect(mockWASMExports.planMigration).toHaveBeenCalled();
      expect(plan).toHaveProperty('migration');
    });

    it('should apply migration', async () => {
      const result = await client.applyMigration('mig1');

      expect(mockWASMExports.applyMigration).toHaveBeenCalledWith(1, 'mig1');
      expect(result).toHaveProperty('id', 'mig1');
    });

    it('should rollback migration', async () => {
      const result = await client.rollbackMigration(0);

      expect(mockWASMExports.rollbackMigration).toHaveBeenCalledWith(1, 0);
      expect(result).toHaveProperty('version', 0);
    });

    it('should validate migration', async () => {
      const report = await client.validateMigration('mig1');

      expect(mockWASMExports.validateMigration).toHaveBeenCalledWith(1, 'mig1');
      expect(report.isValid).toBe(true);
    });

    it('should get migration history', async () => {
      const history = await client.getMigrationHistory();

      expect(mockWASMExports.getMigrationHistory).toHaveBeenCalledWith(1);
      expect(history.currentVersion).toBe(1);
    });
  });

  describe('schema generation', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should generate JSON schema', async () => {
      const schema = await client.generateJSONSchema('single');

      expect(mockWASMExports.generateJSONSchema).toHaveBeenCalledWith(1, 'single');
      expect(schema).toBe('{}');
    });

    it('should generate GraphQL schema', async () => {
      const schema = await client.generateGraphQLSchema({
        includeMutations: true,
      });

      expect(mockWASMExports.generateGraphQLSchema).toHaveBeenCalled();
      expect(schema).toContain('type Query');
    });
  });

  describe('transactions', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should begin transaction', async () => {
      const txId = await client.beginTransaction();

      expect(mockWASMExports.beginTransaction).toHaveBeenCalledWith(1);
      expect(txId).toBe(1);
    });

    it('should commit transaction', async () => {
      const txId = await client.beginTransaction();
      await client.commitTransaction(txId);

      expect(mockWASMExports.commitTransaction).toHaveBeenCalledWith(1, 1);
    });

    it('should rollback transaction', async () => {
      const txId = await client.beginTransaction();
      await client.rollbackTransaction(txId);

      expect(mockWASMExports.rollbackTransaction).toHaveBeenCalledWith(1, 1);
    });
  });

  describe('prepared statements', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should prepare statement', async () => {
      const stmtId = await client.prepare('SELECT * FROM "users" WHERE "id" == ?');

      expect(mockWASMExports.prepare).toHaveBeenCalledWith(
        1,
        'SELECT * FROM "users" WHERE "id" == ?'
      );
      expect(stmtId).toBe(1);
    });

    it('should execute statement', async () => {
      const stmtId = await client.prepare('SELECT * FROM "users" WHERE "id" == ?');
      const result = await client.executeStatement(stmtId, [123]);

      expect(mockWASMExports.executeStatement).toHaveBeenCalledWith(
        1,
        1,
        JSON.stringify([123])
      );
      expect(result).toEqual({ result: true });
    });

    it('should deallocate statement', async () => {
      const stmtId = await client.prepare('SELECT 1');
      await client.deallocateStatement(stmtId);

      expect(mockWASMExports.deallocateStatement).toHaveBeenCalledWith(1, 1);
    });
  });

  describe('hooks', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should register hook', async () => {
      await client.registerHook({
        name: 'testHook',
        before: () => {},
        after: () => {},
      });

      expect(mockWASMExports.registerHook).toHaveBeenCalled();
    });

    it('should unregister hook', async () => {
      await client.unregisterHook('testHook');

      expect(mockWASMExports.unregisterHook).toHaveBeenCalledWith(1, 'testHook');
    });

    it('should get metrics stats', async () => {
      const stats = await client.getMetricsStats();

      expect(mockWASMExports.getMetricsStats).toHaveBeenCalledWith(1);
      expect(stats.totalOperations).toBe(100);
      expect(stats.successCount).toBe(95);
    });
  });

  describe('performance', () => {
    beforeEach(async () => {
      await client.initialize();
      await client.connect('localhost:1776');
    });

    it('should track operation performance', async () => {
      await client.query('SELECT 1');

      const stats = client.getPerformanceStats();
      expect(stats.operations['query']).toBeDefined();
      expect(stats.operations['query'].count).toBeGreaterThan(0);
    });

    it('should reset performance metrics', async () => {
      await client.query('SELECT 1');

      client.resetPerformanceMetrics();

      const stats = client.getPerformanceStats();
      expect(stats.operations['query']).toBeUndefined();
    });

    it('should get performance stats', () => {
      const stats = client.getPerformanceStats();

      expect(stats).toHaveProperty('operations');
      expect(stats).toHaveProperty('boundaries');
      expect(stats).toHaveProperty('memory');
    });
  });

  describe('debug', () => {
    beforeEach(async () => {
      await client.initialize();
    });

    it('should get debug info', () => {
      const debugInfo = client.getDebugInfo();

      expect(debugInfo).toHaveProperty('version');
      expect(debugInfo).toHaveProperty('wasmMetadata');
      expect(debugInfo).toHaveProperty('stateHistory');
      expect(debugInfo).toHaveProperty('performanceStats');
    });

    it('should get WASM metadata', () => {
      const metadata = client.getWASMMetadata();

      expect(metadata).toHaveProperty('goVersion', '1.25');
      expect(metadata).toHaveProperty('hasDWARF', true);
    });
  });
});
