import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { appendJsonl, readJsonl } from '../cli/lib/jsonl.js';

const TMP = path.join(import.meta.dir, '.tmp-stats');
const MINT = path.join(TMP, '.mint');

function mintFile(name) {
  return path.join(MINT, name);
}

beforeEach(() => {
  fs.mkdirSync(MINT, { recursive: true });
  fs.writeFileSync(mintFile('config.json'), JSON.stringify({ stack: 'bun', gates: { tests: 'bun test' } }));
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('stats: pipeline health metrics', () => {
  test('calculates gate pass rate correctly', () => {
    const file = mintFile('metrics.jsonl');
    appendJsonl(file, { spec: '001', gateResults: { lint: 'pass', types: 'pass', tests: 'pass' }, attempts: 1 });
    appendJsonl(file, { spec: '002', gateResults: { lint: 'pass', types: 'fail', tests: 'pass' }, attempts: 2 });
    appendJsonl(file, { spec: '003', gateResults: { lint: 'pass', types: 'pass', tests: 'pass' }, attempts: 1 });
    appendJsonl(file, { spec: '004', gateResults: { lint: 'pass', types: 'pass', tests: 'pass' }, attempts: 1 });

    const metrics = readJsonl(file);
    const gatesPassed = metrics.filter(m => {
      const gates = m.gateResults || {};
      return Object.values(gates).every(v => v === 'pass');
    });

    expect(gatesPassed.length).toBe(3);
    expect(Math.round((gatesPassed.length / metrics.length) * 100)).toBe(75);
  });

  test('calculates first-try success rate', () => {
    const file = mintFile('metrics.jsonl');
    appendJsonl(file, { spec: '001', attempts: 1 });
    appendJsonl(file, { spec: '002', attempts: 2 });
    appendJsonl(file, { spec: '003', attempts: 1 });
    appendJsonl(file, { spec: '004', attempts: 3 });
    appendJsonl(file, { spec: '005', attempts: 1 });

    const metrics = readJsonl(file);
    const firstTry = metrics.filter(m => (m.attempts || 1) === 1);
    expect(Math.round((firstTry.length / metrics.length) * 100)).toBe(60);
  });

  test('calculates average attempts', () => {
    const file = mintFile('metrics.jsonl');
    appendJsonl(file, { spec: '001', attempts: 1 });
    appendJsonl(file, { spec: '002', attempts: 2 });
    appendJsonl(file, { spec: '003', attempts: 1 });
    appendJsonl(file, { spec: '004', attempts: 2 });

    const metrics = readJsonl(file);
    const avg = metrics.reduce((s, m) => s + (m.attempts || 1), 0) / metrics.length;
    expect(avg).toBe(1.5);
  });

  test('calculates review pass rate', () => {
    const file = mintFile('metrics.jsonl');
    appendJsonl(file, { spec: '001', reviewResult: 'passed' });
    appendJsonl(file, { spec: '002', reviewResult: 'failed' });
    appendJsonl(file, { spec: '003', reviewResult: 'passed' });
    appendJsonl(file, { spec: '004', reviewResult: 'skipped' });
    appendJsonl(file, { spec: '005', reviewResult: 'passed' });

    const metrics = readJsonl(file);
    const reviewed = metrics.filter(m => m.reviewResult && m.reviewResult !== 'skipped');
    const passed = reviewed.filter(m => m.reviewResult === 'passed');
    expect(Math.round((passed.length / reviewed.length) * 100)).toBe(75);
  });
});

describe('stats: issue analysis', () => {
  test('groups issues by rootCause', () => {
    const file = mintFile('issues.jsonl');
    appendJsonl(file, { rootCause: 'scope-leak', severity: 'BLOCKING' });
    appendJsonl(file, { rootCause: 'scope-leak', severity: 'BLOCKING' });
    appendJsonl(file, { rootCause: 'missing-context', severity: 'WARNING' });
    appendJsonl(file, { rootCause: 'scope-leak', severity: 'BLOCKING' });
    appendJsonl(file, { rootCause: 'bad-spec', severity: 'BLOCKING' });

    const issues = readJsonl(file);
    const groups = {};
    for (const issue of issues) {
      const cause = issue.rootCause || 'unknown';
      groups[cause] = (groups[cause] || 0) + 1;
    }

    expect(groups['scope-leak']).toBe(3);
    expect(groups['missing-context']).toBe(1);
    expect(groups['bad-spec']).toBe(1);
  });

  test('counts blocking vs resolved', () => {
    const file = mintFile('issues.jsonl');
    appendJsonl(file, { rootCause: 'a', status: 'resolved' });
    appendJsonl(file, { rootCause: 'a' });
    appendJsonl(file, { rootCause: 'b', status: 'resolved' });

    const issues = readJsonl(file);
    const resolved = issues.filter(i => i.status === 'resolved');
    const active = issues.filter(i => i.status !== 'resolved');

    expect(resolved.length).toBe(2);
    expect(active.length).toBe(1);
  });
});

describe('stats: trend detection', () => {
  test('detects improving trend (second half better)', () => {
    const file = mintFile('metrics.jsonl');
    // First half: 2/4 first-try (50%)
    appendJsonl(file, { spec: '001', attempts: 2 });
    appendJsonl(file, { spec: '002', attempts: 1 });
    appendJsonl(file, { spec: '003', attempts: 2 });
    appendJsonl(file, { spec: '004', attempts: 1 });
    // Second half: 4/4 first-try (100%)
    appendJsonl(file, { spec: '005', attempts: 1 });
    appendJsonl(file, { spec: '006', attempts: 1 });
    appendJsonl(file, { spec: '007', attempts: 1 });
    appendJsonl(file, { spec: '008', attempts: 1 });

    const metrics = readJsonl(file);
    const mid = Math.floor(metrics.length / 2);
    const firstHalf = metrics.slice(0, mid);
    const secondHalf = metrics.slice(mid);

    const firstRate = Math.round((firstHalf.filter(m => m.attempts === 1).length / firstHalf.length) * 100);
    const secondRate = Math.round((secondHalf.filter(m => m.attempts === 1).length / secondHalf.length) * 100);

    expect(secondRate).toBeGreaterThan(firstRate + 5); // improving
  });

  test('detects declining trend (second half worse)', () => {
    const file = mintFile('metrics.jsonl');
    // First half: 3/3 first-try (100%)
    appendJsonl(file, { spec: '001', attempts: 1 });
    appendJsonl(file, { spec: '002', attempts: 1 });
    appendJsonl(file, { spec: '003', attempts: 1 });
    // Second half: 1/3 first-try (33%)
    appendJsonl(file, { spec: '004', attempts: 2 });
    appendJsonl(file, { spec: '005', attempts: 3 });
    appendJsonl(file, { spec: '006', attempts: 1 });

    const metrics = readJsonl(file);
    const mid = Math.floor(metrics.length / 2);
    const firstHalf = metrics.slice(0, mid);
    const secondHalf = metrics.slice(mid);

    const firstRate = Math.round((firstHalf.filter(m => m.attempts === 1).length / firstHalf.length) * 100);
    const secondRate = Math.round((secondHalf.filter(m => m.attempts === 1).length / secondHalf.length) * 100);

    expect(secondRate).toBeLessThan(firstRate - 5); // declining
  });
});

describe('stats: instinct health', () => {
  test('categorizes instincts by confidence level', () => {
    const file = mintFile('instincts.jsonl');
    appendJsonl(file, { category: 'naming', observation: 'a', confidence: 10 });
    appendJsonl(file, { category: 'naming', observation: 'b', confidence: 8 });
    appendJsonl(file, { category: 'naming', observation: 'c', confidence: 5 });
    appendJsonl(file, { category: 'naming', observation: 'd', confidence: 2 });
    appendJsonl(file, { category: 'naming', observation: 'e', confidence: 1 });

    const instincts = readJsonl(file);
    const high = instincts.filter(i => (i.confidence || 0) >= 7);
    const med = instincts.filter(i => (i.confidence || 0) >= 3 && (i.confidence || 0) < 7);
    const low = instincts.filter(i => (i.confidence || 0) < 3);

    expect(high.length).toBe(2);
    expect(med.length).toBe(1);
    expect(low.length).toBe(2);
  });

  test('identifies promotion candidates', () => {
    const file = mintFile('instincts.jsonl');
    appendJsonl(file, { category: 'naming', observation: 'camel', confidence: 10, occurrences: 15 });
    appendJsonl(file, { category: 'naming', observation: 'snake', confidence: 8, occurrences: 5 });

    const instincts = readJsonl(file);
    const candidates = instincts.filter(i =>
      (i.confidence || 0) >= 7 && (i.occurrences || 0) >= 10
    );

    expect(candidates.length).toBe(1);
    expect(candidates[0].observation).toBe('camel');
  });
});
