/**
 * Error context for tracking error origin
 */
export interface ErrorContext {
  /** File where error occurred */
  file?: string;

  /** Line number */
  line?: number;

  /** Function name */
  function?: string;

  /** Additional context */
  metadata?: Record<string, unknown>;
}

/**
 * Stack frame from Go or JavaScript
 */
export interface StackFrame {
  /** File path */
  file: string;

  /** Line number */
  line: number;

  /** Column number */
  column?: number;

  /** Function name */
  function: string;

  /** Whether this is from Go (true) or JavaScript (false) */
  isGo: boolean;

  /** Source code snippet (if available) */
  source?: string;
}

/**
 * DWARF source map entry
 */
export interface SourceMapEntry {
  /** Program counter address */
  pc: number;

  /** Source file path */
  file: string;

  /** Line number in source */
  line: number;

  /** Column number in source */
  column: number;

  /** Function name */
  function: string;

  /** Inlined function depth */
  inlineDepth: number;
}

/**
 * Error code categories
 */
export type ErrorCode =
  | 'CONNECTION_ERROR'
  | 'QUERY_ERROR'
  | 'MUTATION_ERROR'
  | 'MIGRATION_ERROR'
  | 'VALIDATION_ERROR'
  | 'TRANSACTION_ERROR'
  | 'HOOK_ERROR'
  | 'STATEMENT_ERROR'
  | 'WASM_ERROR'
  | 'TIMEOUT_ERROR'
  | 'UNKNOWN_ERROR';

/**
 * Error type for categorization
 */
export type ErrorType =
  | 'CONNECTION_ERROR'
  | 'QUERY_ERROR'
  | 'MUTATION_ERROR'
  | 'MIGRATION_ERROR'
  | 'VALIDATION_ERROR'
  | 'TRANSACTION_ERROR'
  | 'HOOK_ERROR'
  | 'STATEMENT_ERROR'
  | 'WASM_ERROR'
  | 'TIMEOUT_ERROR'
  | 'UNKNOWN_ERROR';
