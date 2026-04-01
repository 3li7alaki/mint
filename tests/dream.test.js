import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { readJsonl, appendJsonl, upsertInstinct, decayInstincts } from '../cli/lib/jsonl.js';

const TMP = path.join(import.meta.dir, '.tmp-dream');
const MINT = path.join(TMP, '.mint');

function mintFile(name) {
  return path.join(MINT, name);
}

beforeEach(() => {
  fs.mkdirSync(MINT, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('dream: issue triage', () => {
  test('identifies duplicate issues by rootCause', () => {
    const file = mintFile('issues.jsonl');
    const issues = [
      { date: '2026-03-01', task: 'auth-001', rootCause: 'scope-leak', issue: 'Modified files outside scope' },
      { date: '2026-03-05', task: 'api-002', rootCause: 'scope-leak', issue: 'Scope leak in handler' },
      { date: '2026-03-10', task: 'db-003', rootCause: 'scope-leak', issue: 'Files outside can-modify' },
    ];
    for (const issue of issues) appendJsonl(file, issue);

    const entries = readJsonl(file);
    const grouped = {};
    for (const e of entries) {
      const key = e.rootCause || 'unknown';
      grouped[key] = (grouped[key] || 0) + 1;
    }

    expect(grouped['scope-leak']).toBe(3);
    // 3+ occurrences = escalation candidate
    expect(grouped['scope-leak']).toBeGreaterThanOrEqual(3);
  });

  test('separates resolved from active issues', () => {
    const file = mintFile('issues.jsonl');
    appendJsonl(file, { date: '2026-03-01', task: 'a', status: 'resolved', issue: 'old' });
    appendJsonl(file, { date: '2026-03-20', task: 'b', issue: 'active' });
    appendJsonl(file, { date: '2026-03-25', task: 'c', status: 'resolved', issue: 'fixed' });

    const entries = readJsonl(file);
    const resolved = entries.filter(e => e.status === 'resolved');
    const active = entries.filter(e => e.status !== 'resolved');

    expect(resolved.length).toBe(2);
    expect(active.length).toBe(1);
  });
});

describe('dream: instinct lifecycle', () => {
  test('high-confidence instincts are promotion candidates', () => {
    const file = mintFile('instincts.jsonl');

    // Build up a high-confidence instinct
    for (let i = 0; i < 10; i++) {
      upsertInstinct(file, { category: 'naming', observation: 'camelCase functions', source: 'observer' });
    }

    const instincts = readJsonl(file);
    const highConf = instincts.filter(i => (i.confidence || 0) >= 7 && (i.occurrences || 0) >= 10);

    expect(highConf.length).toBe(1);
    expect(highConf[0].confidence).toBe(10);
    expect(highConf[0].occurrences).toBe(10);
  });

  test('decay reduces stale instinct confidence', () => {
    const file = mintFile('instincts.jsonl');

    // Create an instinct with old lastSeen
    const oldDate = new Date(Date.now() - 45 * 24 * 60 * 60 * 1000).toISOString();
    appendJsonl(file, {
      category: 'naming',
      observation: 'snake_case',
      confidence: 3,
      occurrences: 3,
      firstSeen: oldDate,
      lastSeen: oldDate,
      sources: ['observer'],
      examples: [],
    });

    const result = decayInstincts(file, 30);
    expect(result.decayed).toBe(1);

    const after = readJsonl(file);
    expect(after[0].confidence).toBe(2);
  });

  test('decay removes instinct at confidence 0', () => {
    const file = mintFile('instincts.jsonl');

    const oldDate = new Date(Date.now() - 45 * 24 * 60 * 60 * 1000).toISOString();
    appendJsonl(file, {
      category: 'naming',
      observation: 'old-pattern',
      confidence: 1,
      occurrences: 1,
      firstSeen: oldDate,
      lastSeen: oldDate,
      sources: ['observer'],
      examples: [],
    });

    const result = decayInstincts(file, 30);
    expect(result.removed).toBe(1);

    const after = readJsonl(file);
    expect(after.length).toBe(0);
  });
});

describe('dream: win archival', () => {
  test('keeps recent wins, archives old ones', () => {
    const winsFile = mintFile('wins.jsonl');
    const archiveFile = mintFile('wins-archive.jsonl');

    // Add 55 wins
    for (let i = 0; i < 55; i++) {
      appendJsonl(winsFile, {
        date: `2026-03-${String(i % 28 + 1).padStart(2, '0')}`,
        task: `task-${i}`,
        pattern: `pattern-${i}`,
        why: 'test',
      });
    }

    const all = readJsonl(winsFile);
    expect(all.length).toBe(55);

    // Simulate archival: keep last 50, move rest
    const active = all.slice(-50);
    const archived = all.slice(0, all.length - 50);

    for (const w of archived) appendJsonl(archiveFile, w);
    fs.writeFileSync(winsFile, active.map(e => JSON.stringify(e)).join('\n') + '\n');

    expect(readJsonl(winsFile).length).toBe(50);
    expect(readJsonl(archiveFile).length).toBe(5);
  });
});

describe('dream: metrics analysis', () => {
  test('calculates gate pass rate', () => {
    const file = mintFile('metrics.jsonl');

    appendJsonl(file, { spec: '001', gateResults: { lint: 'pass', types: 'pass', tests: 'pass' }, attempts: 1 });
    appendJsonl(file, { spec: '002', gateResults: { lint: 'pass', types: 'fail', tests: 'pass' }, attempts: 2 });
    appendJsonl(file, { spec: '003', gateResults: { lint: 'pass', types: 'pass', tests: 'pass' }, attempts: 1 });

    const metrics = readJsonl(file);
    const allPassed = metrics.filter(m => {
      const gates = m.gateResults || {};
      return Object.values(gates).every(v => v === 'pass');
    });

    expect(allPassed.length).toBe(2);
    expect(Math.round((allPassed.length / metrics.length) * 100)).toBe(67);
  });

  test('calculates first-try success rate', () => {
    const file = mintFile('metrics.jsonl');

    appendJsonl(file, { spec: '001', attempts: 1 });
    appendJsonl(file, { spec: '002', attempts: 2 });
    appendJsonl(file, { spec: '003', attempts: 1 });
    appendJsonl(file, { spec: '004', attempts: 1 });

    const metrics = readJsonl(file);
    const firstTry = metrics.filter(m => (m.attempts || 1) === 1);
    const rate = Math.round((firstTry.length / metrics.length) * 100);

    expect(rate).toBe(75);
  });
});

describe('dream: lock mechanism', () => {
  test('creates and reads dream lock', () => {
    const lockFile = mintFile('.dream-lock');

    fs.writeFileSync(lockFile, JSON.stringify({ timestamp: new Date().toISOString() }));
    expect(fs.existsSync(lockFile)).toBe(true);

    const lock = JSON.parse(fs.readFileSync(lockFile, 'utf8'));
    const lockAge = Date.now() - new Date(lock.timestamp).getTime();
    expect(lockAge).toBeLessThan(5000); // less than 5 seconds old
  });

  test('detects stale lock (>1 hour)', () => {
    const lockFile = mintFile('.dream-lock');

    const staleTime = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
    fs.writeFileSync(lockFile, JSON.stringify({ timestamp: staleTime }));

    const lock = JSON.parse(fs.readFileSync(lockFile, 'utf8'));
    const lockAge = Date.now() - new Date(lock.timestamp).getTime();
    const isStale = lockAge > 60 * 60 * 1000;

    expect(isStale).toBe(true);
  });
});
