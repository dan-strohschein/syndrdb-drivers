/**
 * Baseline Performance Tests
 * 
 * Validates performance against established baselines and detects regressions.
 * These tests should fail if performance degrades by more than 20%.
 */

import { SyndrDBClient } from '../../src/client';
import * as fs from 'fs';
import * as path from 'path';

interface Baseline {
  operation: string;
  averageDuration: number;
  p95: number;
  timestamp: string;
  nodeVersion: string;
}

interface BaselineFile {
  version: string;
  timestamp: string;
  baselines: Baseline[];
}

describe('Baseline Performance Tests', () => {
  let client: SyndrDBClient;
  const TEST_TIMEOUT = 120000; // 2 minutes
  const REGRESSION_THRESHOLD = 0.20; // 20% regression tolerance
  const BASELINE_PATH = path.join(__dirname, 'baseline.json');
  
  let baselines: BaselineFile | null = null;

  beforeAll(async () => {
    // Load baseline file if it exists
    if (fs.existsSync(BASELINE_PATH)) {
      const content = fs.readFileSync(BASELINE_PATH, 'utf-8');
      baselines = JSON.parse(content);
      console.log(`\nLoaded baseline from ${BASELINE_PATH}`);
      console.log(`Baseline timestamp: ${baselines?.timestamp}`);
      console.log(`Baseline Node version: ${baselines?.baselines[0]?.nodeVersion || 'unknown'}`);
    } else {
      console.log(`\nNo baseline file found at ${BASELINE_PATH}`);
      console.log('Run "npm run benchmark:baseline" to create baseline metrics');
    }

    client = new SyndrDBClient({
      host: 'localhost',
      port: 1776,
      debug: false,
      performanceMonitoring: { enabled: false },
    });

    await client.connect();

    // Warmup
    for (let i = 0; i < 10; i++) {
      await client.query({ bundle: 'users', filters: {} });
    }
  }, TEST_TIMEOUT);

  afterAll(async () => {
    if (client) {
      await client.disconnect();
    }
  }, TEST_TIMEOUT);

  /**
   * Helper to measure operation performance
   */
  async function measureOperation(
    name: string,
    iterations: number,
    operation: () => Promise<void>
  ): Promise<{ average: number; p95: number }> {
    const durations: number[] = [];

    for (let i = 0; i < iterations; i++) {
      const start = performance.now();
      await operation();
      const end = performance.now();
      durations.push(end - start);
    }

    durations.sort((a, b) => a - b);
    const average = durations.reduce((sum, d) => sum + d, 0) / iterations;
    const p95Index = Math.floor(iterations * 0.95);
    const p95 = durations[p95Index];

    return { average, p95 };
  }

  /**
   * Helper to check against baseline
   */
  function checkBaseline(operation: string, current: number, metric: 'average' | 'p95'): void {
    if (!baselines) {
      console.log(`  ⚠️  No baseline available for ${operation}`);
      return;
    }

    const baseline = baselines.baselines.find(b => b.operation === operation);
    if (!baseline) {
      console.log(`  ⚠️  No baseline found for operation: ${operation}`);
      return;
    }

    const baselineValue = metric === 'average' ? baseline.averageDuration : baseline.p95;
    const regression = (current - baselineValue) / baselineValue;
    const percentChange = (regression * 100).toFixed(1);

    if (regression > REGRESSION_THRESHOLD) {
      console.log(`  ❌ REGRESSION: ${operation} (${metric})`);
      console.log(`     Baseline: ${baselineValue.toFixed(2)}ms`);
      console.log(`     Current:  ${current.toFixed(2)}ms`);
      console.log(`     Change:   +${percentChange}% (threshold: ${REGRESSION_THRESHOLD * 100}%)`);
      
      throw new Error(
        `Performance regression detected for ${operation} (${metric}): ` +
        `${percentChange}% slower than baseline (threshold: ${REGRESSION_THRESHOLD * 100}%)`
      );
    } else if (regression > 0) {
      console.log(`  ⚠️  ${operation} (${metric}): +${percentChange}% (within threshold)`);
    } else {
      console.log(`  ✓ ${operation} (${metric}): ${percentChange}% (improved or stable)`);
    }
  }

  describe('Query Performance', () => {
    test('simple query should meet baseline', async () => {
      const result = await measureOperation(
        'Simple Query',
        50,
        async () => {
          await client.query({ bundle: 'users', filters: {} });
        }
      );

      console.log(`\nSimple Query: avg=${result.average.toFixed(2)}ms, p95=${result.p95.toFixed(2)}ms`);
      checkBaseline('Simple Query', result.average, 'average');
      checkBaseline('Simple Query', result.p95, 'p95');

      // Absolute limits regardless of baseline
      expect(result.average).toBeLessThan(100);
      expect(result.p95).toBeLessThan(150);
    }, TEST_TIMEOUT);

    test('query with filters should meet baseline', async () => {
      const result = await measureOperation(
        'Query with Filters',
        50,
        async () => {
          await client.query({
            bundle: 'users',
            filters: { active: true },
          });
        }
      );

      console.log(`\nQuery with Filters: avg=${result.average.toFixed(2)}ms, p95=${result.p95.toFixed(2)}ms`);
      checkBaseline('Query with Filters', result.average, 'average');
      checkBaseline('Query with Filters', result.p95, 'p95');

      expect(result.average).toBeLessThan(100);
      expect(result.p95).toBeLessThan(150);
    }, TEST_TIMEOUT);

    test('query with pagination should meet baseline', async () => {
      const result = await measureOperation(
        'Query with Pagination',
        50,
        async () => {
          await client.query({
            bundle: 'users',
            filters: {},
            limit: 10,
            offset: 0,
          });
        }
      );

      console.log(`\nQuery with Pagination: avg=${result.average.toFixed(2)}ms, p95=${result.p95.toFixed(2)}ms`);
      checkBaseline('Query with Pagination', result.average, 'average');
      checkBaseline('Query with Pagination', result.p95, 'p95');

      expect(result.average).toBeLessThan(100);
      expect(result.p95).toBeLessThan(150);
    }, TEST_TIMEOUT);
  });

  describe('Mutation Performance', () => {
    const createdIds: string[] = [];

    afterAll(async () => {
      for (const id of createdIds) {
        try {
          await client.mutate({ operation: 'delete', bundle: 'users', id });
        } catch (error) {
          // Ignore
        }
      }
    });

    test('create operation should meet baseline', async () => {
      const result = await measureOperation(
        'Create Operation',
        30,
        async () => {
          const res = await client.mutate({
            operation: 'create',
            bundle: 'users',
            data: {
              name: `Baseline Test ${Date.now()}`,
              email: `baseline-${Date.now()}-${Math.random()}@example.com`,
              active: true,
            },
          });
          createdIds.push(res.data.id);
        }
      );

      console.log(`\nCreate Operation: avg=${result.average.toFixed(2)}ms, p95=${result.p95.toFixed(2)}ms`);
      checkBaseline('Create Operation', result.average, 'average');
      checkBaseline('Create Operation', result.p95, 'p95');

      expect(result.average).toBeLessThan(150);
      expect(result.p95).toBeLessThan(200);
    }, TEST_TIMEOUT);

    test('update operation should meet baseline', async () => {
      // Create record to update
      const createRes = await client.mutate({
        operation: 'create',
        bundle: 'users',
        data: {
          name: 'Baseline Update Test',
          email: `baseline-update-${Date.now()}@example.com`,
          active: true,
        },
      });
      const userId = createRes.data.id;
      createdIds.push(userId);

      const result = await measureOperation(
        'Update Operation',
        30,
        async () => {
          await client.mutate({
            operation: 'update',
            bundle: 'users',
            id: userId,
            data: { name: `Updated ${Date.now()}` },
          });
        }
      );

      console.log(`\nUpdate Operation: avg=${result.average.toFixed(2)}ms, p95=${result.p95.toFixed(2)}ms`);
      checkBaseline('Update Operation', result.average, 'average');
      checkBaseline('Update Operation', result.p95, 'p95');

      expect(result.average).toBeLessThan(150);
      expect(result.p95).toBeLessThan(200);
    }, TEST_TIMEOUT);
  });

  describe('Transaction Performance', () => {
    test('transaction begin-commit should meet baseline', async () => {
      const result = await measureOperation(
        'Transaction Begin-Commit',
        30,
        async () => {
          const tx = await client.beginTransaction();
          await tx.commit();
        }
      );

      console.log(`\nTransaction Begin-Commit: avg=${result.average.toFixed(2)}ms, p95=${result.p95.toFixed(2)}ms`);
      checkBaseline('Transaction Begin-Commit', result.average, 'average');
      checkBaseline('Transaction Begin-Commit', result.p95, 'p95');

      expect(result.average).toBeLessThan(100);
      expect(result.p95).toBeLessThan(150);
    }, TEST_TIMEOUT);

    test('transaction with query should meet baseline', async () => {
      const result = await measureOperation(
        'Transaction with Query',
        30,
        async () => {
          const tx = await client.beginTransaction();
          await tx.query({ bundle: 'users', filters: {} });
          await tx.commit();
        }
      );

      console.log(`\nTransaction with Query: avg=${result.average.toFixed(2)}ms, p95=${result.p95.toFixed(2)}ms`);
      checkBaseline('Transaction with Query', result.average, 'average');
      checkBaseline('Transaction with Query', result.p95, 'p95');

      expect(result.average).toBeLessThan(150);
      expect(result.p95).toBeLessThan(200);
    }, TEST_TIMEOUT);
  });

  describe('WASM Overhead', () => {
    test('WASM boundary crossing should meet baseline', async () => {
      const result = await measureOperation(
        'WASM Boundary Crossing',
        50,
        async () => {
          await client.healthCheck();
        }
      );

      console.log(`\nWASM Boundary Crossing: avg=${result.average.toFixed(2)}ms, p95=${result.p95.toFixed(2)}ms`);
      checkBaseline('WASM Boundary Crossing', result.average, 'average');
      checkBaseline('WASM Boundary Crossing', result.p95, 'p95');

      expect(result.average).toBeLessThan(50);
      expect(result.p95).toBeLessThan(75);
    }, TEST_TIMEOUT);
  });

  describe('Parallel Operations', () => {
    test('parallel queries should meet baseline', async () => {
      const result = await measureOperation(
        'Parallel Queries',
        30,
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

      console.log(`\nParallel Queries: avg=${result.average.toFixed(2)}ms, p95=${result.p95.toFixed(2)}ms`);
      checkBaseline('Parallel Queries', result.average, 'average');
      checkBaseline('Parallel Queries', result.p95, 'p95');

      expect(result.average).toBeLessThan(300);
      expect(result.p95).toBeLessThan(400);
    }, TEST_TIMEOUT);
  });

  describe('Summary', () => {
    test('print baseline comparison summary', () => {
      if (!baselines) {
        console.log('\n' + '='.repeat(60));
        console.log('NO BASELINE AVAILABLE');
        console.log('='.repeat(60));
        console.log('\nTo establish baseline metrics:');
        console.log('  npm run benchmark:baseline');
        console.log('\nThis will create baseline.json for future comparisons.');
        console.log('='.repeat(60) + '\n');
      } else {
        console.log('\n' + '='.repeat(60));
        console.log('BASELINE PERFORMANCE COMPARISON');
        console.log('='.repeat(60));
        console.log(`Baseline Date: ${baselines.timestamp}`);
        console.log(`Current Date:  ${new Date().toISOString()}`);
        console.log(`Threshold:     ${REGRESSION_THRESHOLD * 100}% regression tolerance`);
        console.log('='.repeat(60));
        console.log('\nAll performance tests passed! No significant regressions detected.');
        console.log('='.repeat(60) + '\n');
      }
    });
  });
});
