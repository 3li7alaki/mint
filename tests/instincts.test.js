import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { upsertInstinct, decayInstincts, topInstincts, readJsonl } from '../cli/lib/jsonl.js';

const TMP = path.join(import.meta.dir, '.tmp-instincts');
const FILE = path.join(TMP, 'instincts.jsonl');

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('upsertInstinct', () => {
  test('creates new instinct with confidence 1', () => {
    const result = upsertInstinct(FILE, {
      category: 'naming',
      observation: 'camelCase for functions',
    });
    expect(result.action).toBe('created');
    expect(result.confidence).toBe(1);

    const entries = readJsonl(FILE);
    expect(entries).toHaveLength(1);
    expect(entries[0].category).toBe('naming');
    expect(entries[0].confidence).toBe(1);
    expect(entries[0].occurrences).toBe(1);
  });

  test('increments confidence on duplicate', () => {
    upsertInstinct(FILE, { category: 'naming', observation: 'camelCase for functions' });
    const result = upsertInstinct(FILE, { category: 'naming', observation: 'camelCase for functions' });

    expect(result.action).toBe('updated');
    expect(result.confidence).toBe(2);

    const entries = readJsonl(FILE);
    expect(entries).toHaveLength(1);
    expect(entries[0].confidence).toBe(2);
    expect(entries[0].occurrences).toBe(2);
  });

  test('normalizes for matching (case, whitespace)', () => {
    upsertInstinct(FILE, { category: 'Naming', observation: 'camelCase  for  Functions' });
    const result = upsertInstinct(FILE, { category: 'naming', observation: 'camelCase for functions' });

    expect(result.action).toBe('updated');
    expect(readJsonl(FILE)).toHaveLength(1);
  });

  test('different category = different instinct', () => {
    upsertInstinct(FILE, { category: 'naming', observation: 'camelCase' });
    upsertInstinct(FILE, { category: 'testing', observation: 'camelCase' });

    expect(readJsonl(FILE)).toHaveLength(2);
  });

  test('accumulates sources', () => {
    upsertInstinct(FILE, { category: 'naming', observation: 'camelCase', source: 'observer' });
    upsertInstinct(FILE, { category: 'naming', observation: 'camelCase', source: 'reviewer:conventions' });

    const entries = readJsonl(FILE);
    expect(entries[0].sources).toEqual(['observer', 'reviewer:conventions']);
  });

  test('accumulates examples (max 5)', () => {
    for (let i = 0; i < 7; i++) {
      upsertInstinct(FILE, {
        category: 'naming',
        observation: 'camelCase',
        examples: [`example-${i}`],
      });
    }

    const entries = readJsonl(FILE);
    expect(entries[0].examples.length).toBeLessThanOrEqual(5);
  });
});

describe('decayInstincts', () => {
  test('decays stale instincts', () => {
    // Write an old instinct
    const old = new Date(Date.now() - 40 * 24 * 60 * 60 * 1000).toISOString();
    fs.writeFileSync(FILE, JSON.stringify({
      category: 'naming', observation: 'old pattern',
      confidence: 3, occurrences: 3, firstSeen: old, lastSeen: old,
    }) + '\n');

    const result = decayInstincts(FILE, 30);
    expect(result.decayed).toBe(1);

    const entries = readJsonl(FILE);
    expect(entries[0].confidence).toBe(2);
  });

  test('removes instincts at confidence 0', () => {
    const old = new Date(Date.now() - 40 * 24 * 60 * 60 * 1000).toISOString();
    fs.writeFileSync(FILE, JSON.stringify({
      category: 'naming', observation: 'weak pattern',
      confidence: 1, occurrences: 1, firstSeen: old, lastSeen: old,
    }) + '\n');

    const result = decayInstincts(FILE, 30);
    expect(result.removed).toBe(1);
    expect(readJsonl(FILE)).toHaveLength(0);
  });

  test('leaves fresh instincts alone', () => {
    const recent = new Date().toISOString();
    fs.writeFileSync(FILE, JSON.stringify({
      category: 'naming', observation: 'fresh pattern',
      confidence: 2, occurrences: 2, firstSeen: recent, lastSeen: recent,
    }) + '\n');

    const result = decayInstincts(FILE, 30);
    expect(result.decayed).toBe(0);
    expect(result.removed).toBe(0);
  });
});

describe('topInstincts', () => {
  test('returns top N by confidence', () => {
    const entries = [
      { category: 'a', observation: 'low', confidence: 1 },
      { category: 'b', observation: 'high', confidence: 5 },
      { category: 'c', observation: 'mid', confidence: 3 },
      { category: 'd', observation: 'highest', confidence: 8 },
    ];
    fs.writeFileSync(FILE, entries.map(e => JSON.stringify(e)).join('\n') + '\n');

    const top = topInstincts(FILE, 2, 3);
    expect(top).toHaveLength(2);
    expect(top[0].observation).toBe('highest');
    expect(top[1].observation).toBe('high');
  });

  test('filters by minimum confidence', () => {
    const entries = [
      { category: 'a', observation: 'low', confidence: 1 },
      { category: 'b', observation: 'mid', confidence: 2 },
    ];
    fs.writeFileSync(FILE, entries.map(e => JSON.stringify(e)).join('\n') + '\n');

    const top = topInstincts(FILE, 20, 3);
    expect(top).toHaveLength(0);
  });
});
