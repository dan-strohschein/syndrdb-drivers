/**
 * Error type definitions extending base SyndrDBError
 */

import { SyndrDBError } from './base';
import type { ErrorContext, StackFrame } from '../types/errors';

/**
 * Connection-related errors
 */
export class ConnectionError extends SyndrDBError {
  constructor(
    message: string,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(message, 'CONNECTION_ERROR', 'CONNECTION_ERROR', goStack, jsStack, cause, context);
    this.name = 'ConnectionError';
  }
}

/**
 * Query execution errors
 */
export class QueryError extends SyndrDBError {
  constructor(
    message: string,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(message, 'QUERY_ERROR', 'QUERY_ERROR', goStack, jsStack, cause, context);
    this.name = 'QueryError';
  }
}

/**
 * Mutation execution errors
 */
export class MutationError extends SyndrDBError {
  constructor(
    message: string,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(message, 'MUTATION_ERROR', 'MUTATION_ERROR', goStack, jsStack, cause, context);
    this.name = 'MutationError';
  }
}

/**
 * Migration-related errors
 */
export class MigrationError extends SyndrDBError {
  constructor(
    message: string,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(message, 'MIGRATION_ERROR', 'MIGRATION_ERROR', goStack, jsStack, cause, context);
    this.name = 'MigrationError';
  }
}

/**
 * Validation errors
 */
export class ValidationError extends SyndrDBError {
  constructor(
    message: string,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(message, 'VALIDATION_ERROR', 'VALIDATION_ERROR', goStack, jsStack, cause, context);
    this.name = 'ValidationError';
  }
}

/**
 * Transaction errors
 */
export class TransactionError extends SyndrDBError {
  constructor(
    message: string,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(message, 'TRANSACTION_ERROR', 'TRANSACTION_ERROR', goStack, jsStack, cause, context);
    this.name = 'TransactionError';
  }
}

/**
 * Hook execution errors
 */
export class HookError extends SyndrDBError {
  constructor(
    message: string,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(message, 'HOOK_ERROR', 'HOOK_ERROR', goStack, jsStack, cause, context);
    this.name = 'HookError';
  }
}

/**
 * Prepared statement errors
 */
export class StatementError extends SyndrDBError {
  constructor(
    message: string,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(message, 'STATEMENT_ERROR', 'STATEMENT_ERROR', goStack, jsStack, cause, context);
    this.name = 'StatementError';
  }
}

/**
 * WASM-related errors
 */
export class WASMError extends SyndrDBError {
  constructor(
    message: string,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(message, 'WASM_ERROR', 'WASM_ERROR', goStack, jsStack, cause, context);
    this.name = 'WASMError';
  }
}

/**
 * WASM load error
 */
export class WASMLoadError extends WASMError {
  constructor(message: string, cause?: Error) {
    super(message, undefined, undefined, cause);
    this.name = 'WASMLoadError';
  }
}

/**
 * WASM not found error
 */
export class WASMNotFoundError extends WASMError {
  constructor(path: string) {
    super(`WASM binary not found at path: ${path}`);
    this.name = 'WASMNotFoundError';
  }
}

/**
 * WASM load timeout error
 */
export class WASMTimeoutError extends WASMError {
  constructor(timeoutMs: number) {
    super(`WASM load timeout after ${timeoutMs}ms`);
    this.name = 'WASMTimeoutError';
  }
}

/**
 * WASM version mismatch error
 */
export class WASMVersionMismatchError extends WASMError {
  constructor(expected: string, actual: string) {
    super(`WASM version mismatch: expected ${expected}, got ${actual}`);
    this.name = 'WASMVersionMismatchError';
  }
}

/**
 * WASM corrupted error
 */
export class WASMCorruptedError extends WASMError {
  constructor(reason: string) {
    super(`WASM binary corrupted: ${reason}`);
    this.name = 'WASMCorruptedError';
  }
}

/**
 * Timeout errors
 */
export class TimeoutError extends SyndrDBError {
  constructor(
    operation: string,
    timeoutMs: number,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(
      `Operation '${operation}' timed out after ${timeoutMs}ms`,
      'TIMEOUT_ERROR',
      'TIMEOUT_ERROR',
      goStack,
      jsStack,
      cause,
      context
    );
    this.name = 'TimeoutError';
  }
}
