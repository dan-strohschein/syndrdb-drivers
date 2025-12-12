import type { BundleDefinition, FieldDefinition } from '../schema/SchemaDefinition';
import type { TypeRegistry } from './TypeRegistry';

/**
 * Maps raw SyndrDB responses to typed objects
 * Follows Single Responsibility Principle - only handles runtime data mapping
 */
export class ResponseMapper {
  private readonly typeRegistry: TypeRegistry;
  private readonly autoConvertDates: boolean;

  constructor(typeRegistry: TypeRegistry, autoConvertDates: boolean = true) {
    this.typeRegistry = typeRegistry;
    this.autoConvertDates = autoConvertDates;
  }

  /**
   * Maps a single result object to typed object
   * @param bundleName - Bundle name
   * @param rawData - Raw data from server
   * @returns Mapped object with proper types
   */
  mapResult<T = any>(bundleName: string, rawData: any): T {
    const cached = this.typeRegistry.get(bundleName);
    if (!cached) {
      // No type info available, return as-is
      // TODO: Consider logging warning when auto-mapping is enabled but type info is missing
      return rawData as T;
    }

    return this.mapObject(cached.bundle, rawData) as T;
  }

  /**
   * Maps multiple result objects
   * @param bundleName - Bundle name
   * @param rawDataArray - Array of raw data from server
   * @returns Array of mapped objects
   */
  mapResults<T = any>(bundleName: string, rawDataArray: any[]): T[] {
    return rawDataArray.map(raw => this.mapResult<T>(bundleName, raw));
  }

  /**
   * Maps object based on bundle definition
   * @param bundle - Bundle definition
   * @param rawData - Raw data
   * @returns Mapped object
   */
  private mapObject(bundle: BundleDefinition, rawData: any): any {
    if (!rawData || typeof rawData !== 'object') {
      return rawData;
    }

    const mapped: any = {};

    for (const field of bundle.fields) {
      const rawValue = rawData[field.name];
      
      if (rawValue === undefined || rawValue === null) {
        mapped[field.name] = rawValue;
        continue;
      }

      mapped[field.name] = this.convertFieldValue(field, rawValue);
    }

    // Copy any additional fields not in bundle definition
    // TODO: Consider adding strict mode that throws on unknown fields
    for (const key in rawData) {
      if (!(key in mapped)) {
        mapped[key] = rawData[key];
      }
    }

    return mapped;
  }

  /**
   * Converts field value to appropriate TypeScript type
   * @param field - Field definition
   * @param value - Raw value
   * @returns Converted value
   */
  private convertFieldValue(field: FieldDefinition, value: any): any {
    switch (field.type) {
      case 'DATETIME':
        if (this.autoConvertDates) {
          return this.parseDate(value);
        }
        return value;

      case 'INT':
      case 'INTEGER':
        return typeof value === 'string' ? parseInt(value, 10) : value;

      case 'FLOAT':
        return typeof value === 'string' ? parseFloat(value) : value;

      case 'BOOLEAN':
        if (typeof value === 'string') {
          return value.toLowerCase() === 'true' || value === '1';
        }
        return Boolean(value);
        // This is not yet supported, but soon will be
      // case 'JSON':
      //   if (typeof value === 'string') {
      //     try {
      //       return JSON.parse(value);
      //     } catch {
      //       // TODO: Consider logging JSON parse errors
      //       return value;
      //     }
      //   }
      //   return value;

      case 'relationship':
        // TODO: Handle nested object mapping when relationship data is included
        // For now, assume it's already an object or an ID
        if (typeof value === 'object' && field.relatedBundle) {
          const relatedCached = this.typeRegistry.get(field.relatedBundle);
          if (relatedCached) {
            return this.mapObject(relatedCached.bundle, value);
          }
        }
        return value;

      default:
        return value;
    }
  }

  /**
   * Parses date from various formats
   * @param value - Date value (string, number, or Date)
   * @returns Date object or original value if parsing fails
   */
  private parseDate(value: any): Date | any {
    if (value instanceof Date) {
      return value;
    }

    if (typeof value === 'string' || typeof value === 'number') {
      const date = new Date(value);
      if (!isNaN(date.getTime())) {
        return date;
      }
    }

    // TODO: Consider supporting additional date formats (ISO 8601, custom formats)
    return value;
  }

  /**
   * Maps a generic query response with Result array
   * @param bundleName - Bundle name for results
   * @param response - Server response with Result array
   * @returns Mapped response with typed results
   */
  mapQueryResponse<T = any>(
    bundleName: string,
    response: { Result?: any[]; ResultCount?: number; ExecutionTimeMS?: number }
  ): {
    results: T[];
    count: number;
    executionTime: number;
  } {
    return {
      results: this.mapResults<T>(bundleName, response.Result || []),
      count: response.ResultCount ?? response.Result?.length ?? 0,
      executionTime: response.ExecutionTimeMS ?? 0
    };
  }
}
