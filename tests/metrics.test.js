import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { logMetric, analyzeInstinctEffectiveness, readJsonl } from '../cli/lib/jsonl.js';

const TMP = path.join(import.meta.dir, '.tmp-metrics');
const FILE = path.join(TMP, 'metrics.jsonl');

beforeEach(() => { fs.mkdirSync(TMP, { recursive: true }); });
afterEach(() => { fs.rmSync(TMP, { recursive: true, force: true }); });

describe('logMetric', () => {
  test('logs a metric entry', () => {
    logMetric(FILE, {
      task: 'auth', spec: '001',
      instinctsApplied: ['camelCase-functions'],
      reviewResult: 'passed',
      gateResults: { lint: 'pass', types: 'pass', tests: 'pass' },
      attempts: 1,
    });

    const entries = readJsonl(FILE);
    expect(entries).toHaveLength(1);
    expect(entries[0].task).toBe('auth');
    expect(entries[0].reviewResult).toBe('passed');
  });
});

describe('analyzeInstinctEffectiveness', () => {
  test('calculates pass rate per instinct', () => {
    logMetric(FILE, { task: 'a', spec: '001', instinctsApplied: ['camelCase'], reviewResult: 'passed' });
    logMetric(FILE, { task: 'a', spec: '002', instinctsApplied: ['camelCase'], reviewResult: 'passed' });
    logMetric(FILE, { task: 'a', spec: '003', instinctsApplied: ['camelCase'], reviewResult: 'failed' });
    logMetric(FILE, { task: 'b', spec: '001', instinctsApplied: ['barrel-imports'], reviewResult: 'passed' });

    const results = analyzeInstinctEffectiveness(FILE);
    expect(results).toHaveLength(2);

    const camel = results.find(r => r.observation === 'camelCase');
    expect(camel.uses).toBe(3);
    expect(camel.passRate).toBe(67);

    const barrel = results.find(r => r.observation === 'barrel-imports');
    expect(barrel.uses).toBe(1);
    expect(barrel.passRate).toBe(100);
  });

  test('returns empty array for no data', () => {
    expect(analyzeInstinctEffectiveness(FILE)).toEqual([]);
  });
});
