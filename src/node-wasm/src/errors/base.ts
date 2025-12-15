/**
 * Base error class for all SyndrDB errors
 * Provides DWARF-aware stack traces combining Go and JavaScript sources
 */
export class SyndrDBError extends Error {
  /** Error code for categorization */
  public readonly code: string;

  /** Error type */
  public readonly type: string;

  /** Go stack frames (from DWARF) */
  public readonly goStack?: StackFrame[];

  /** JavaScript stack frames */
  public readonly jsStack?: StackFrame[];

  /** Original error cause */
  public readonly cause?: Error;

  /** Error context */
  public readonly context?: ErrorContext;

  /** Timestamp when error occurred */
  public readonly timestamp: Date;

  /**
   * Create a new SyndrDB error
   * @param message Error message
   * @param code Error code
   * @param type Error type
   * @param goStack Go stack frames from DWARF
   * @param jsStack JavaScript stack frames
   * @param cause Original error cause
   * @param context Additional error context
   */
  constructor(
    message: string,
    code: string,
    type: string,
    goStack?: StackFrame[],
    jsStack?: StackFrame[],
    cause?: Error,
    context?: ErrorContext
  ) {
    super(message);
    this.name = 'SyndrDBError';
    this.code = code;
    this.type = type;
    if (goStack) this.goStack = goStack;
    if (jsStack) this.jsStack = jsStack;
    if (cause) this.cause = cause;
    if (context) this.context = context;
    this.timestamp = new Date();

    // Maintains proper stack trace for where error was thrown (V8 only)
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, this.constructor);
    }
  }

  /**
   * Format error with dual stack traces
   * @param includeGoStack Whether to include Go stack trace
   * @returns Formatted error string
   */
  format(includeGoStack = true): string {
    const lines: string[] = [];

    lines.push(`${this.name}: ${this.message}`);
    lines.push(`  Code: ${this.code}`);
    lines.push(`  Type: ${this.type}`);
    lines.push(`  Time: ${this.timestamp.toISOString()}`);

    if (this.context) {
      lines.push('\nContext:');
      if (this.context.file) lines.push(`  File: ${this.context.file}:${this.context.line}`);
      if (this.context.function) lines.push(`  Function: ${this.context.function}`);
      if (this.context.metadata) {
        lines.push('  Metadata:');
        for (const [key, value] of Object.entries(this.context.metadata)) {
          lines.push(`    ${key}: ${JSON.stringify(value)}`);
        }
      }
    }

    if (includeGoStack && this.goStack && this.goStack.length > 0) {
      lines.push('\nGo Stack Trace:');
      for (const frame of this.goStack) {
        lines.push(`  at ${frame.function} (${frame.file}:${frame.line})`);
        if (frame.source) {
          lines.push(`    > ${frame.source.trim()}`);
        }
      }
    }

    if (this.jsStack && this.jsStack.length > 0) {
      lines.push('\nJavaScript Stack Trace:');
      for (const frame of this.jsStack) {
        const location = frame.column
          ? `${frame.file}:${frame.line}:${frame.column}`
          : `${frame.file}:${frame.line}`;
        lines.push(`  at ${frame.function} (${location})`);
      }
    } else if (this.stack) {
      lines.push('\nJavaScript Stack Trace:');
      lines.push(this.stack);
    }

    if (this.cause) {
      lines.push('\nCaused by:');
      lines.push(`  ${this.cause.name}: ${this.cause.message}`);
      if (this.cause.stack) {
        lines.push(this.cause.stack);
      }
    }

    return lines.join('\n');
  }

  /**
   * Convert error to JSON
   * @returns JSON representation
   */
  toJSON(): Record<string, unknown> {
    return {
      name: this.name,
      message: this.message,
      code: this.code,
      type: this.type,
      timestamp: this.timestamp.toISOString(),
      context: this.context,
      goStack: this.goStack,
      jsStack: this.jsStack,
      cause: this.cause
        ? {
            name: this.cause.name,
            message: this.cause.message,
            stack: this.cause.stack,
          }
        : undefined,
    };
  }
}

/**
 * Import types from types/errors
 */
import type { ErrorContext, StackFrame } from '../types/errors';
