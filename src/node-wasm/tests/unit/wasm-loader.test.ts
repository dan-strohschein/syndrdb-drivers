/**
 * Unit tests for WASMLoader
 */

import { WASMLoader, getWASMLoader } from '../../src/wasm-loader';
import { WASMLoadError, WASMNotFoundError, WASMTimeoutError } from '../../src/errors';

describe('WASMLoader', () => {
  beforeEach(() => {
    // Reset singleton between tests
    WASMLoader.reset();
  });

  describe('singleton pattern', () => {
    it('should return same instance', () => {
      const instance1 = WASMLoader.getInstance();
      const instance2 = WASMLoader.getInstance();
      expect(instance1).toBe(instance2);
    });

    it('should return same instance via getWASMLoader', () => {
      const instance1 = getWASMLoader();
      const instance2 = WASMLoader.getInstance();
      expect(instance1).toBe(instance2);
    });
  });

  describe('load', () => {
    it('should not be loaded initially', () => {
      const loader = getWASMLoader();
      expect(loader.isLoaded()).toBe(false);
    });

    it('should return null metadata when not loaded', () => {
      const loader = getWASMLoader();
      expect(loader.getMetadata()).toBeNull();
    });

    it('should return null exports when not loaded', () => {
      const loader = getWASMLoader();
      expect(loader.getExports()).toBeNull();
    });

    // TODO: Add load tests once WASM binary is available
    // These will test:
    // - Successful load with valid WASM binary
    // - DWARF metadata extraction
    // - Integrity verification
    // - Timeout handling
    // - Error cases (missing file, corrupted binary)
  });

  describe('reset', () => {
    it('should reset singleton', () => {
      const instance1 = WASMLoader.getInstance();
      WASMLoader.reset();
      const instance2 = WASMLoader.getInstance();
      expect(instance1).not.toBe(instance2);
    });
  });
});
