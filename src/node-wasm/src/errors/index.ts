/**
 * Error exports
 */

export { SyndrDBError } from './base';
export {
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
} from './types';
export { mapWASMError, parseWASMError } from './mapper';
