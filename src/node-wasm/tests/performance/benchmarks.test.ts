/**
 * Performance Benchmarks
 * 
 * Measures performance of key operations and WASM boundary crossing overhead.
 * Results can be used to establish baseline metrics for regression testing.
 */

import { performance } from 'perf_hooks';
import { SyndrDBClient } from '../../src/client';
import type { QueryResult, MutationResult } from '../../src/types';

interface BenchmarkResult {
  operation: string;
  iterations: number;
  totalDuration: number;
  averageDuration: number;
  opsPerSecond: number;
  p50: number;
  p95: number;
  p99: number;
}

describe('Performance Benchmarks', () => {
  let client: SyndrDBClient;
  const TEST_TIMEOUT = 300000; // 5 minutes for benchmarks
  const WARMUP_ITERATIONS = 10;
  const BENCHMARK_ITERATIONS = 100;

  beforeAll(async () => {
    client = new SyndrDBClient({
      host: 'localhost',
      port: 1776,
      debug: false, // Disable debug logging for accurate benchmarks
      performanceMonitoring: {
        enabled: false, // Disable to avoid interference
      },
    });

    await client.connect();

    // Warmup: ensure connection is stable and caches are primed
    for (let i = 0; i < WARMUP_ITERATIONS; i++) {
      await client.query({ bundle: 'users', filters: {} });
    }
  }, TEST_TIMEOUT);

  afterAll(async () => {
    if (client) {
      await client.disconnect();
    }
  }, TEST_TIMEOUT);

  /**
   * Helper to run a benchmark and calculate statistics
   */
  function runBenchmark(
    name: string,
    iterations: number,
    operation: () => Promise<void>
  ): Promise<BenchmarkResult> {
    return new Promise(async (resolve) => {
      const durations: number[] = [];

      for (let i = 0; i < iterations; i++) {
        const start = performance.now();
        await operation();
        const end = performance.now();
        durations.push(end - start);
      }

      // Calculate statistics
      durations.sort((a, b) => a - b);
      const totalDuration = durations.reduce((sum, d) => sum + d, 0);
      const averageDuration = totalDuration / iterations;
      const opsPerSecond = 1000 / averageDuration;

      const p50Index = Math.floor(iterations * 0.5);
      const p95Index = Math.floor(iterations * 0.95);
      const p99Index = Math.floor(iterations * 0.99);

      resolve({
        operation: name,
        iterations,
        totalDuration,
        averageDuration,
        opsPerSecond,
        p50: durations[p50Index],
        p95: durations[p95Index],
        p99: durations[p99Index],
      });
    });
  }

  /**
   * Print benchmark results in a readable format
   */
  function printResults(result: BenchmarkResult): void {
    console.log(`\n${'='.repeat(60)}`);
    console.log(`Benchmark: ${result.operation}`);
    console.log(`${'='.repeat(60)}`);
    console.log(`Iterations:      ${result.iterations}`);
    console.log(`Total Duration:  ${result.totalDuration.toFixed(2)}ms`);
    console.log(`Average:         ${result.averageDuration.toFixed(2)}ms`);
    console.log(`Ops/Second:      ${result.opsPerSecond.toFixed(2)}`);
    console.log(`P50 (median):    ${result.p50.toFixed(2)}ms`);
    console.log(`P95:             ${result.p95.toFixed(2)}ms`);
    console.log(`P99:             ${result.p99.toFixed(2)}ms`);
    console.log(`${'='.repeat(60)}\n`);
  }

  describe('Query Operations', () => {
    test('benchmark simple query', async () => {
      const result = await runBenchmark(
        'Simple Query (no filters)',
        BENCHMARK_ITERATIONS,
        async () => {
          await client.query({ bundle: 'users', filters: {} });
        }
      );

      printResults(result);

      // Assertions - queries should be reasonably fast
      expect(result.averageDuration).toBeLessThan(100); // < 100ms average
      expect(result.p95).toBeLessThan(150); // < 150ms at p95
    }, TEST_TIMEOUT);

    test('benchmark query with filters', async () => {
      const result = await runBenchmark(
        'Query with Filters',
        BENCHMARK_ITERATIONS,
        async () => {
          await client.query({
            bundle: 'users',
            filters: { active: true },
          });
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(100);
    }, TEST_TIMEOUT);

    test('benchmark query with select fields', async () => {
      const result = await runBenchmark(
        'Query with Select Fields',
        BENCHMARK_ITERATIONS,
        async () => {
          await client.query({
            bundle: 'users',
            filters: {},
            select: ['id', 'name', 'email'],
          });
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(100);
    }, TEST_TIMEOUT);

    test('benchmark query with pagination', async () => {
      const result = await runBenchmark(
        'Query with Pagination',
        BENCHMARK_ITERATIONS,
        async () => {
          await client.query({
            bundle: 'users',
            filters: {},
            limit: 10,
            offset: 0,
          });
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(100);
    }, TEST_TIMEOUT);

    test('benchmark query with sorting', async () => {
      const result = await runBenchmark(
        'Query with Sorting',
        BENCHMARK_ITERATIONS,
        async () => {
          await client.query({
            bundle: 'users',
            filters: {},
            sort: [{ field: 'created_at', direction: 'desc' }],
          });
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(100);
    }, TEST_TIMEOUT);
  });

  describe('Mutation Operations', () => {
    const createdIds: string[] = [];

    afterAll(async () => {
      // Clean up created records
      for (const id of createdIds) {
        try {
          await client.mutate({
            operation: 'delete',
            bundle: 'users',
            id,
          });
        } catch (error) {
          // Ignore cleanup errors
        }
      }
    });

    test('benchmark create operation', async () => {
      const result = await runBenchmark(
        'Create Operation',
        50, // Fewer iterations for mutations
        async () => {
          const res = await client.mutate({
            operation: 'create',
            bundle: 'users',
            data: {
              name: `Benchmark User ${Date.now()}`,
              email: `bench-${Date.now()}-${Math.random()}@example.com`,
              active: true,
            },
          });
          createdIds.push(res.data.id);
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(150); // Mutations can be slower
    }, TEST_TIMEOUT);

    test('benchmark update operation', async () => {
      // Create a record to update
      const createRes = await client.mutate({
        operation: 'create',
        bundle: 'users',
        data: {
          name: 'Update Benchmark User',
          email: `update-bench-${Date.now()}@example.com`,
          active: true,
        },
      });
      const userId = createRes.data.id;
      createdIds.push(userId);

      const result = await runBenchmark(
        'Update Operation',
        50,
        async () => {
          await client.mutate({
            operation: 'update',
            bundle: 'users',
            id: userId,
            data: { name: `Updated ${Date.now()}` },
          });
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(150);
    }, TEST_TIMEOUT);

    test('benchmark delete operation', async () => {
      // Create records to delete
      const idsToDelete: string[] = [];
      for (let i = 0; i < 50; i++) {
        const res = await client.mutate({
          operation: 'create',
          bundle: 'users',
          data: {
            name: `Delete Benchmark ${i}`,
            email: `delete-bench-${i}-${Date.now()}@example.com`,
            active: true,
          },
        });
        idsToDelete.push(res.data.id);
      }

      const result = await runBenchmark(
        'Delete Operation',
        50,
        async () => {
          const id = idsToDelete.pop();
          if (id) {
            await client.mutate({
              operation: 'delete',
              bundle: 'users',
              id,
            });
          }
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(150);
    }, TEST_TIMEOUT);
  });

  describe('Transaction Operations', () => {
    test('benchmark transaction begin-commit', async () => {
      const result = await runBenchmark(
        'Transaction Begin-Commit',
        50,
        async () => {
          const tx = await client.beginTransaction();
          await tx.commit();
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(100);
    }, TEST_TIMEOUT);

    test('benchmark transaction with query', async () => {
      const result = await runBenchmark(
        'Transaction with Query',
        50,
        async () => {
          const tx = await client.beginTransaction();
          await tx.query({ bundle: 'users', filters: {} });
          await tx.commit();
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(150);
    }, TEST_TIMEOUT);

    test('benchmark transaction rollback', async () => {
      const result = await runBenchmark(
        'Transaction Rollback',
        50,
        async () => {
          const tx = await client.beginTransaction();
          await tx.rollback();
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(100);
    }, TEST_TIMEOUT);
  });

  describe('Prepared Statements', () => {
    test('benchmark prepared statement creation', async () => {
      const statements: string[] = [];

      const result = await runBenchmark(
        'Prepared Statement Creation',
        50,
        async () => {
          const query = `SELECT * FROM \"users\" WHERE \"id\" == $1 -- ${Date.now()}-${Math.random()}`;
          const stmt = await client.prepareStatement(query);
          statements.push(stmt.getId());
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(100);

      // Cleanup
      for (const stmtId of statements) {
        try {
          await client.deallocateStatement(stmtId);
        } catch (error) {
          // Ignore
        }
      }
    }, TEST_TIMEOUT);

    test('benchmark prepared statement execution', async () => {
      const stmt = await client.prepareStatement(
        'SELECT * FROM \"users\" WHERE \"active\" == $1'
      );

      const result = await runBenchmark(
        'Prepared Statement Execution',
        BENCHMARK_ITERATIONS,
        async () => {
          await stmt.execute([true]);
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(100);

      await client.deallocateStatement(stmt.getId());
    }, TEST_TIMEOUT);
  });

  describe('WASM Boundary Overhead', () => {
    test('benchmark WASM call overhead', async () => {
      // Measure overhead of crossing WASM boundary with minimal operation
      const result = await runBenchmark(
        'WASM Boundary Crossing (health check)',
        BENCHMARK_ITERATIONS,
        async () => {
          await client.healthCheck();
        }
      );

      printResults(result);

      // Health check should be very fast - mostly WASM overhead
      expect(result.averageDuration).toBeLessThan(50);
      
      console.log(`\nWASM Boundary Overhead: ~${result.averageDuration.toFixed(2)}ms per call`);
    }, TEST_TIMEOUT);

    test('benchmark JSON serialization overhead', async () => {
      // Measure overhead of JSON serialization across boundary
      const largeFilter = {
        id: 'test-id',
        active: true,
        tags: ['tag1', 'tag2', 'tag3', 'tag4', 'tag5'],
        metadata: {
          field1: 'value1',
          field2: 'value2',
          field3: 'value3',
        },
      };

      const result = await runBenchmark(
        'JSON Serialization (complex filter)',
        BENCHMARK_ITERATIONS,
        async () => {
          await client.query({
            bundle: 'users',
            filters: largeFilter,
          });
        }
      );

      printResults(result);
      expect(result.averageDuration).toBeLessThan(150);
    }, TEST_TIMEOUT);
  });

  describe('Batch Operations', () => {
    test('benchmark parallel queries', async () => {
      const result = await runBenchmark(
        'Parallel Queries (5 concurrent)',
        50,
        async () => {
          await Promise.all([
            client.query({ bundle: 'users', filters: {} }),
            client.query({ bundle: 'posts', filters: {} }),
            client.query({ bundle: 'comments', filters: {} }),
            client.query({ bundle: 'users', filters: { active: true } }),
            client.query({ bundle: 'posts', filters: { published: true } }),
          ]);
        }
      );

      printResults(result);
      
      // Parallel should be faster than sequential
      expect(result.averageDuration).toBeLessThan(300);
    }, TEST_TIMEOUT);

    test('benchmark sequential vs parallel', async () => {
      // Sequential
      const sequentialResult = await runBenchmark(
        'Sequential Queries (5)',
        20,
        async () => {
          await client.query({ bundle: 'users', filters: {} });
          await client.query({ bundle: 'posts', filters: {} });
          await client.query({ bundle: 'comments', filters: {} });
          await client.query({ bundle: 'users', filters: { active: true } });
          await client.query({ bundle: 'posts', filters: { published: true } });
        }
      );

      printResults(sequentialResult);

      // Parallel
      const parallelResult = await runBenchmark(
        'Parallel Queries (5)',
        20,
        async () => {
          await Promise.all([
            client.query({ bundle: 'users', filters: {} }),
            client.query({ bundle: 'posts', filters: {} }),
            client.query({ bundle: 'comments', filters: {} }),
            client.query({ bundle: 'users', filters: { active: true } }),
            client.query({ bundle: 'posts', filters: { published: true } }),
          ]);
        }
      );

      printResults(parallelResult);

      // Parallel should be significantly faster
      const speedup = sequentialResult.averageDuration / parallelResult.averageDuration;
      console.log(`\nSpeedup from parallelization: ${speedup.toFixed(2)}x`);
      
      expect(speedup).toBeGreaterThan(1.5); // At least 1.5x faster
    }, TEST_TIMEOUT);
  });

  describe('Summary Report', () => {
    test('generate performance summary', () => {
      console.log('\n' + '='.repeat(60));
      console.log('PERFORMANCE BENCHMARK SUMMARY');
      console.log('='.repeat(60));
      console.log(`Date: ${new Date().toISOString()}`);
      console.log(`Node Version: ${process.version}`);
      console.log(`Platform: ${process.platform}`);
      console.log(`Architecture: ${process.arch}`);
      console.log('='.repeat(60));
      console.log('\nAll benchmarks completed successfully!');
      console.log('\nThese results can be used as baseline metrics for CI regression testing.');
      console.log('Store these values and compare future runs to detect performance degradation.');
      console.log('='.repeat(60) + '\n');
    });
  });
});
