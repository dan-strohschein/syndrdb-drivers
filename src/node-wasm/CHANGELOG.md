# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial implementation of WASM-powered Node.js driver
- DWARF v5 source map support for enhanced debugging
- Dual stack traces (Go + JavaScript) in debug mode
- Operation batching for optimized WASM boundary crossing
- Comprehensive performance monitoring with regression testing
- Hook system for extensibility (logging, metrics, tracing)
- Connection pooling with automatic health checks
- Prepared statements with automatic cleanup
- Transaction support with ACID guarantees
- Migration system with automatic rollback generation
- Schema generation (JSON Schema, GraphQL, TypeScript)
- TypeScript type generation from Go source
- Comprehensive test suite with 85%+ coverage
- Performance benchmarks with CI regression detection

### Changed
- N/A (initial release)

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- N/A

## [2.0.0-alpha.1] - 2025-12-12

### Added
- Initial alpha release for testing
- Core WASM wrapper implementation
- Basic error handling and type safety
- Documentation and examples

---

[Unreleased]: https://github.com/dan-strohschein/syndrdb-drivers/compare/v2.0.0-alpha.1...HEAD
[2.0.0-alpha.1]: https://github.com/dan-strohschein/syndrdb-drivers/releases/tag/v2.0.0-alpha.1
