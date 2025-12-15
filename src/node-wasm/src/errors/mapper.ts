/**
 * Error mapping utilities to convert WASM errors to typed errors
 */

import { SyndrDBError } from './base';
import {
  ConnectionError,
  QueryError,
  MutationError,
  MigrationError,
  ValidationError,
  TransactionError,
  HookError,
  StatementError,
  WASMError,
  TimeoutError,
} from './types';
import type { ErrorContext, StackFrame } from '../types/errors';

/**
 * Map WASM error to appropriate error type
 * @param message Error message from WASM
 * @param errorType Error type from WASM
 * @param goStack Go stack frames from DWARF
 * @param jsStack JavaScript stack frames
 * @param cause Original error
 * @param context Error context
 * @returns Typed error instance
 */
export function mapWASMError(
  message: string,
  errorType: string,
  goStack?: StackFrame[],
  jsStack?: StackFrame[],
  cause?: Error,
  context?: ErrorContext
): SyndrDBError {
  // TODO: Enhance error type detection logic as more error patterns emerge
  switch (errorType) {
    case 'CONNECTION_ERROR':
      return new ConnectionError(message, goStack, jsStack, cause, context);

    case 'QUERY_ERROR':
      return new QueryError(message, goStack, jsStack, cause, context);

    case 'MUTATION_ERROR':
      return new MutationError(message, goStack, jsStack, cause, context);

    case 'MIGRATION_ERROR':
      return new MigrationError(message, goStack, jsStack, cause, context);

    case 'VALIDATION_ERROR':
      return new ValidationError(message, goStack, jsStack, cause, context);

    case 'TRANSACTION_ERROR':
      return new TransactionError(message, goStack, jsStack, cause, context);

    case 'HOOK_ERROR':
      return new HookError(message, goStack, jsStack, cause, context);

    case 'STATEMENT_ERROR':
      return new StatementError(message, goStack, jsStack, cause, context);

    case 'WASM_ERROR':
      return new WASMError(message, goStack, jsStack, cause, context);

    case 'TIMEOUT_ERROR':
      return new TimeoutError('operation', 0, goStack, jsStack, cause, context);

    default:
      return new SyndrDBError(
        message,
        'UNKNOWN_ERROR',
        'UNKNOWN_ERROR',
        goStack,
        jsStack,
        cause,
        context
      );
  }
}

/**
 * Parse error message from WASM to extract type and context
 * @param rawMessage Raw error message from WASM
 * @returns Parsed error information
 */
export function parseWASMError(rawMessage: string): {
  message: string;
  type: string;
  context?: ErrorContext | undefined;
} {
  // TODO: Implement robust parsing once Go error format is finalized
  // Expected format: "[ERROR_TYPE] message at file.go:line in function()"

  const typeMatch = rawMessage.match(/^\[([A-Z_]+)\]\s+/);
  let type = '';
  type = (typeMatch ? typeMatch[1] : 'UNKNOWN_ERROR') as string;
  let message = '';
  if (!rawMessage) {
    rawMessage = '';
   }
  message = typeMatch ? rawMessage.replace(/^\[([A-Z_]+)\]\s+/, '') : rawMessage;

  const locationMatch = rawMessage.match(/at\s+([^:]+):(\d+)\s+in\s+([^()]+)/);
  let context: ErrorContext | undefined = undefined;
  
  if (locationMatch && locationMatch[1] && locationMatch[2] && locationMatch[3]) {
    context = {
      file: locationMatch[1],
      line: parseInt(locationMatch[2], 10),
      function: locationMatch[3],
    };
  }

  return { message, type, context };
}
