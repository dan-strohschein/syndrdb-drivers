/**
 * Unit tests for error classes
 */

import {
  SyndrDBError,
  ConnectionError,
  QueryError,
  MutationError,
  MigrationError,
  ValidationError,
  TransactionError,
  HookError,
  StatementError,
  WASMError,
  WASMLoadError,
  WASMNotFoundError,
  WASMTimeoutError,
  WASMVersionMismatchError,
  WASMCorruptedError,
  TimeoutError,
  mapWASMError,
  parseWASMError,
} from '../../src/errors';
import type { StackFrame, ErrorContext } from '../../src/types/errors';

describe('Error Classes', () => {
  describe('SyndrDBError', () => {
    it('should create base error', () => {
      const error = new SyndrDBError('Test error', 'TEST_CODE', 'TEST_TYPE');

      expect(error.message).toBe('Test error');
      expect(error.code).toBe('TEST_CODE');
      expect(error.type).toBe('TEST_TYPE');
      expect(error.timestamp).toBeInstanceOf(Date);
    });

    it('should include stack traces', () => {
      const goStack: StackFrame[] = [
        {
          file: 'connection.go',
          line: 123,
          function: 'Connect',
          isGo: true,
        },
      ];

      const jsStack: StackFrame[] = [
        {
          file: 'client.ts',
          line: 45,
          column: 10,
          function: 'connect',
          isGo: false,
        },
      ];

      const error = new SyndrDBError(
        'Dual stack error',
        'DUAL_STACK',
        'TEST',
        goStack,
        jsStack
      );

      expect(error.goStack).toEqual(goStack);
      expect(error.jsStack).toEqual(jsStack);
    });

    it('should format error with dual stacks', () => {
      const goStack: StackFrame[] = [
        {
          file: 'connection.go',
          line: 123,
          function: 'Connect',
          isGo: true,
          source: 'return nil, errors.New("connection failed")',
        },
      ];

      const jsStack: StackFrame[] = [
        {
          file: 'client.ts',
          line: 45,
          column: 10,
          function: 'connect',
          isGo: false,
        },
      ];

      const error = new SyndrDBError(
        'Test error',
        'TEST_CODE',
        'TEST_TYPE',
        goStack,
        jsStack
      );

      const formatted = error.format(true);

      expect(formatted).toContain('SyndrDBError: Test error');
      expect(formatted).toContain('Code: TEST_CODE');
      expect(formatted).toContain('Go Stack Trace:');
      expect(formatted).toContain('connection.go:123');
      expect(formatted).toContain('JavaScript Stack Trace:');
      expect(formatted).toContain('client.ts:45:10');
    });

    it('should format without Go stack', () => {
      const error = new SyndrDBError('Test error', 'TEST_CODE', 'TEST_TYPE');
      const formatted = error.format(false);

      expect(formatted).not.toContain('Go Stack Trace:');
    });

    it('should convert to JSON', () => {
      const context: ErrorContext = {
        file: 'test.ts',
        line: 10,
        function: 'testFunc',
        metadata: { key: 'value' },
      };

      const error = new SyndrDBError(
        'Test error',
        'TEST_CODE',
        'TEST_TYPE',
        undefined,
        undefined,
        undefined,
        context
      );

      const json = error.toJSON();

      expect(json.name).toBe('SyndrDBError');
      expect(json.message).toBe('Test error');
      expect(json.code).toBe('TEST_CODE');
      expect(json.type).toBe('TEST_TYPE');
      expect(json.context).toEqual(context);
    });

    it('should include cause in format', () => {
      const cause = new Error('Original error');
      const error = new SyndrDBError(
        'Wrapped error',
        'WRAP_CODE',
        'WRAP_TYPE',
        undefined,
        undefined,
        cause
      );

      const formatted = error.format();

      expect(formatted).toContain('Caused by:');
      expect(formatted).toContain('Error: Original error');
    });
  });

  describe('Specific error types', () => {
    it('should create ConnectionError', () => {
      const error = new ConnectionError('Connection failed');
      expect(error).toBeInstanceOf(SyndrDBError);
      expect(error.name).toBe('ConnectionError');
      expect(error.code).toBe('CONNECTION_ERROR');
    });

    it('should create QueryError', () => {
      const error = new QueryError('Query failed');
      expect(error.name).toBe('QueryError');
      expect(error.code).toBe('QUERY_ERROR');
    });

    it('should create MutationError', () => {
      const error = new MutationError('Mutation failed');
      expect(error.name).toBe('MutationError');
      expect(error.code).toBe('MUTATION_ERROR');
    });

    it('should create WASMLoadError', () => {
      const error = new WASMLoadError('Load failed');
      expect(error.name).toBe('WASMLoadError');
      expect(error.code).toBe('WASM_ERROR');
    });

    it('should create WASMNotFoundError', () => {
      const error = new WASMNotFoundError('/path/to/wasm');
      expect(error.name).toBe('WASMNotFoundError');
      expect(error.message).toContain('/path/to/wasm');
    });

    it('should create WASMTimeoutError', () => {
      const error = new WASMTimeoutError(5000);
      expect(error.name).toBe('WASMTimeoutError');
      expect(error.message).toContain('5000ms');
    });

    it('should create TimeoutError', () => {
      const error = new TimeoutError('query', 10000);
      expect(error.name).toBe('TimeoutError');
      expect(error.message).toContain('query');
      expect(error.message).toContain('10000ms');
    });
  });

  describe('mapWASMError', () => {
    it('should map to ConnectionError', () => {
      const error = mapWASMError('Connection failed', 'CONNECTION_ERROR');
      expect(error).toBeInstanceOf(ConnectionError);
    });

    it('should map to QueryError', () => {
      const error = mapWASMError('Query failed', 'QUERY_ERROR');
      expect(error).toBeInstanceOf(QueryError);
    });

    it('should map unknown type to base error', () => {
      const error = mapWASMError('Unknown error', 'UNKNOWN_TYPE');
      expect(error).toBeInstanceOf(SyndrDBError);
      expect(error.type).toBe('UNKNOWN_ERROR');
    });

    it('should include stack traces', () => {
      const goStack: StackFrame[] = [
        { file: 'test.go', line: 1, function: 'test', isGo: true },
      ];
      const jsStack: StackFrame[] = [
        { file: 'test.ts', line: 1, function: 'test', isGo: false },
      ];

      const error = mapWASMError('Test', 'QUERY_ERROR', goStack, jsStack);

      expect(error.goStack).toEqual(goStack);
      expect(error.jsStack).toEqual(jsStack);
    });
  });

  describe('parseWASMError', () => {
    it('should parse error with type prefix', () => {
      const parsed = parseWASMError('[QUERY_ERROR] Query failed');

      expect(parsed.type).toBe('QUERY_ERROR');
      expect(parsed.message).toBe('Query failed');
    });

    it('should parse error with location', () => {
      const parsed = parseWASMError(
        '[CONNECTION_ERROR] Failed at connection.go:123 in Connect()'
      );

      expect(parsed.type).toBe('CONNECTION_ERROR');
      expect(parsed.context).toBeDefined();
      expect(parsed.context?.file).toBe('connection.go');
      expect(parsed.context?.line).toBe(123);
      expect(parsed.context?.function).toBe('Connect');
    });

    it('should handle unprefixed errors', () => {
      const parsed = parseWASMError('Plain error message');

      expect(parsed.type).toBe('UNKNOWN_ERROR');
      expect(parsed.message).toBe('Plain error message');
    });
  });
});
