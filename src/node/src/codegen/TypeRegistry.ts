import type { BundleDefinition } from '../schema/SchemaDefinition';

/**
 * Cached type information
 */
export interface CachedType {
  /** Bundle name */
  bundleName: string;
  /** Bundle definition */
  bundle: BundleDefinition;
  /** Generated TypeScript code */
  generatedCode: string;
  /** Timestamp when cached */
  cachedAt: Date;
  /** Hash of bundle definition for change detection */
  definitionHash: string;
}

/**
 * Registry for caching generated types
 * Follows Single Responsibility Principle - only handles type caching and retrieval
 */
export class TypeRegistry {
  private readonly cache = new Map<string, CachedType>();

  /**
   * Registers a generated type
   * @param bundleName - Bundle name
   * @param bundle - Bundle definition
   * @param generatedCode - Generated TypeScript code
   */
  register(bundleName: string, bundle: BundleDefinition, generatedCode: string): void {
    const definitionHash = this.hashBundleDefinition(bundle);
    
    this.cache.set(bundleName, {
      bundleName,
      bundle,
      generatedCode,
      cachedAt: new Date(),
      definitionHash
    });
  }

  /**
   * Gets cached type by bundle name
   * @param bundleName - Bundle name
   * @returns Cached type or undefined
   */
  get(bundleName: string): CachedType | undefined {
    return this.cache.get(bundleName);
  }

  /**
   * Checks if type is cached
   * @param bundleName - Bundle name
   * @returns True if cached
   */
  has(bundleName: string): boolean {
    return this.cache.has(bundleName);
  }

  /**
   * Checks if cached type is still valid for given bundle
   * @param bundleName - Bundle name
   * @param bundle - Current bundle definition
   * @returns True if cached type matches current definition
   */
  isValid(bundleName: string, bundle: BundleDefinition): boolean {
    const cached = this.cache.get(bundleName);
    if (!cached) return false;

    const currentHash = this.hashBundleDefinition(bundle);
    return cached.definitionHash === currentHash;
  }

  /**
   * Removes type from cache
   * @param bundleName - Bundle name
   * @returns True if type was removed
   */
  remove(bundleName: string): boolean {
    return this.cache.delete(bundleName);
  }

  /**
   * Clears all cached types
   */
  clear(): void {
    this.cache.clear();
  }

  /**
   * Gets all cached types
   * @returns Array of cached types
   */
  getAll(): CachedType[] {
    return Array.from(this.cache.values());
  }

  /**
   * Gets all bundle names in cache
   * @returns Array of bundle names
   */
  getBundleNames(): string[] {
    return Array.from(this.cache.keys());
  }

  /**
   * Generates hash of bundle definition for change detection
   * @param bundle - Bundle definition
   * @returns Hash string
   */
  private hashBundleDefinition(bundle: BundleDefinition): string {
    // Simple hash implementation - concatenate relevant properties
    // TODO: Consider using a proper hashing library for collision resistance if bundle definitions become very complex
    const parts: string[] = [
      bundle.name,
      bundle.fields.length.toString(),
      ...bundle.fields.map(f => `${f.name}:${f.type}:${f.required}:${f.unique}:${f.relatedBundle || ''}`),
      bundle.indexes.length.toString(),
      ...bundle.indexes.map(idx => `${idx.fieldName}:${idx.type}`)
    ];
    
    return parts.join('|');
  }

  /**
   * Gets cache statistics
   * @returns Cache statistics
   */
  getStats(): {
    totalCached: number;
    bundleNames: string[];
    oldestCache: Date | null;
    newestCache: Date | null;
  } {
    const types = this.getAll();
    
    return {
      totalCached: types.length,
      bundleNames: types.map(t => t.bundleName),
      oldestCache: types.length > 0 
        ? new Date(Math.min(...types.map(t => t.cachedAt.getTime())))
        : null,
      newestCache: types.length > 0
        ? new Date(Math.max(...types.map(t => t.cachedAt.getTime())))
        : null
    };
  }

  /**
   * Invalidates types older than specified date
   * @param beforeDate - Date threshold
   * @returns Number of invalidated types
   */
  invalidateOlderThan(beforeDate: Date): number {
    let count = 0;
    
    for (const [bundleName, cached] of this.cache.entries()) {
      if (cached.cachedAt < beforeDate) {
        this.cache.delete(bundleName);
        count++;
      }
    }
    
    return count;
  }
}
