import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { getPendingMigrations } from '../cli/migrations/runner.js';

const TMP = path.join(import.meta.dir, '.tmp-migrations');

beforeEach(() => { fs.mkdirSync(path.join(TMP, '.mint'), { recursive: true }); });
afterEach(() => { fs.rmSync(TMP, { recursive: true, force: true }); });

describe('getPendingMigrations', () => {
  const migrations = [
    { from: '0.7.0', to: '0.7.5', description: 'First', steps: [] },
    { from: '0.7.5', to: '0.7.6', description: 'Second', steps: [] },
    { from: '0.7.6', to: '0.8.0', description: 'Third', steps: [] },
    { from: '0.8.0', to: '0.8.1', description: 'Fourth', steps: [] },
  ];

  test('finds migrations between versions', () => {
    const pending = getPendingMigrations('0.7.5', '0.8.0', migrations);
    expect(pending).toHaveLength(2);
    expect(pending[0].description).toBe('Second');
    expect(pending[1].description).toBe('Third');
  });

  test('returns empty when already up to date', () => {
    const pending = getPendingMigrations('0.8.0', '0.8.0', migrations);
    // Only 0.8.0-to-0.8.1 has from >= 0.8.0 AND to <= 0.8.0, which is false
    expect(pending).toHaveLength(0);
  });

  test('returns all when starting from 0.0.0', () => {
    const pending = getPendingMigrations('0.0.0', '0.8.1', migrations);
    expect(pending).toHaveLength(4);
  });
});

describe('0.7.6-to-0.8.0 migration', () => {
  test('migrates old instincts format to scored', async () => {
    // Write old format instincts
    const instinctsPath = path.join(TMP, '.mint', 'instincts.jsonl');
    const oldEntries = [
      { category: 'naming', observation: 'camelCase', file: 'a.js', date: '2026-03-01' },
      { category: 'naming', observation: 'camelCase', file: 'b.js', date: '2026-03-02' },
      { category: 'naming', observation: 'camelCase', file: 'c.js', date: '2026-03-03' },
      { category: 'tests', observation: 'describe-it', file: 'a.test.js', date: '2026-03-01' },
    ];
    fs.writeFileSync(instinctsPath, oldEntries.map(e => JSON.stringify(e)).join('\n') + '\n');

    // Write config
    fs.writeFileSync(path.join(TMP, '.mint', 'config.json'), JSON.stringify({ gates: {} }));

    // Import and run the specific migration step
    const migration = await import('../cli/migrations/0.7.6-to-0.8.0.js');
    const migrateStep = migration.default.steps[0];
    migrateStep.fn(TMP);

    // Check new format
    const content = fs.readFileSync(instinctsPath, 'utf8');
    const entries = content.split('\n').filter(l => l.trim()).map(l => JSON.parse(l));

    expect(entries).toHaveLength(2); // deduplicated
    const naming = entries.find(e => e.category === 'naming');
    expect(naming.confidence).toBe(3);
    expect(naming.occurrences).toBe(3);
    expect(naming.examples).toContain('a.js');
    expect(naming.sources).toContain('observer');
  });

  test('skips already-migrated instincts', async () => {
    const instinctsPath = path.join(TMP, '.mint', 'instincts.jsonl');
    const newEntry = { category: 'naming', observation: 'camelCase', confidence: 5, occurrences: 5 };
    fs.writeFileSync(instinctsPath, JSON.stringify(newEntry) + '\n');

    const migration = await import('../cli/migrations/0.7.6-to-0.8.0.js');
    migration.default.steps[0].fn(TMP);

    const entries = fs.readFileSync(instinctsPath, 'utf8').split('\n').filter(l => l.trim()).map(l => JSON.parse(l));
    expect(entries).toHaveLength(1);
    expect(entries[0].confidence).toBe(5); // unchanged
  });

  test('creates metrics.jsonl if missing', async () => {
    const migration = await import('../cli/migrations/0.7.6-to-0.8.0.js');
    migration.default.steps[1].fn(TMP);
    expect(fs.existsSync(path.join(TMP, '.mint', 'metrics.jsonl'))).toBe(true);
  });
});
