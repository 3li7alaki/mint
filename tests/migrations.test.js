import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import os from 'os';
import path from 'path';
import { getPendingMigrations, runMigrations } from '../cli/migrations/runner.js';

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

describe('0.8.4-to-0.8.5 migration', () => {
  // Use a fresh tmp project per test (separate from the shared TMP) so we never
  // pollute the suite-level fixture and so the migration's symlink-aware code
  // can operate on a real isolated tree.
  let projectRoot;

  beforeEach(() => {
    projectRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'mint-mig-085-'));
    fs.mkdirSync(path.join(projectRoot, '.mint'), { recursive: true });
  });

  afterEach(() => {
    fs.rmSync(projectRoot, { recursive: true, force: true });
  });

  function seedFakeProject(extra = {}) {
    // 0.8.4 config.
    fs.writeFileSync(
      path.join(projectRoot, '.mint', 'config.json'),
      JSON.stringify({
        mintVersion: '0.8.4',
        gates: { lint: false, types: false, tests: 'bun test', coverage: false },
        appliedMigrations: ['0.7.6-to-0.8.0'],
        ...extra.config,
      }, null, 2) + '\n',
    );

    // Old session files (self-generated ids — must be cleared).
    const sessionsDir = path.join(projectRoot, '.mint', 'sessions');
    fs.mkdirSync(sessionsDir, { recursive: true });
    fs.writeFileSync(
      path.join(sessionsDir, 'old-session-aaa.json'),
      JSON.stringify({ mintInvoked: true, task: 'old', mode: 'plan' }),
    );
    fs.writeFileSync(
      path.join(sessionsDir, 'old-session-bbb.json'),
      JSON.stringify({ mintInvoked: false, task: 'older', mode: 'quick' }),
    );

    // Old flat task tree: .mint/tasks/<feature>/<spec>/execution.json (no session-id wrapper).
    const featDir = path.join(projectRoot, '.mint', 'tasks', 'old-feature', '001');
    fs.mkdirSync(featDir, { recursive: true });
    fs.writeFileSync(
      path.join(featDir, 'execution.json'),
      JSON.stringify({ status: 'passed', gates: { tests: 'pass' }, reviews: { spec: 'passed' } }),
    );
    fs.writeFileSync(path.join(featDir, 'spec.xml'), '<task><id>001</id></task>');

    const featDir2 = path.join(projectRoot, '.mint', 'tasks', 'second-feature', '001');
    fs.mkdirSync(featDir2, { recursive: true });
    fs.writeFileSync(path.join(featDir2, 'execution.json'), JSON.stringify({ status: 'failed' }));
  }

  test('clears old sessions and quarantines legacy tasks under _legacy-pre-0.8.5/', async () => {
    seedFakeProject();
    // Add the capture file — must be preserved across the migration.
    fs.writeFileSync(
      path.join(projectRoot, '.mint', 'sessions', '.current-session-id'),
      'claude-real-session-xyz\n',
    );

    const result = await runMigrations(projectRoot, '0.8.5');
    expect(result.success).toBe(true);
    expect(result.applied.length).toBeGreaterThan(0);

    // Old session JSONs are gone.
    const sessionsDir = path.join(projectRoot, '.mint', 'sessions');
    const remaining = fs.readdirSync(sessionsDir);
    expect(remaining).not.toContain('old-session-aaa.json');
    expect(remaining).not.toContain('old-session-bbb.json');
    // .current-session-id (dot-prefixed) preserved.
    expect(remaining).toContain('.current-session-id');
    expect(
      fs.readFileSync(path.join(sessionsDir, '.current-session-id'), 'utf8').trim(),
    ).toBe('claude-real-session-xyz');

    // Legacy feature dirs moved under _legacy-pre-0.8.5/.
    const tasksDir = path.join(projectRoot, '.mint', 'tasks');
    expect(fs.existsSync(path.join(tasksDir, 'old-feature'))).toBe(false);
    expect(fs.existsSync(path.join(tasksDir, 'second-feature'))).toBe(false);
    expect(fs.existsSync(path.join(tasksDir, '_legacy-pre-0.8.5', 'old-feature', '001', 'execution.json'))).toBe(true);
    expect(fs.existsSync(path.join(tasksDir, '_legacy-pre-0.8.5', 'second-feature', '001', 'execution.json'))).toBe(true);
  });

  test('config.mintVersion bumps to 0.8.5 after successful migration', async () => {
    seedFakeProject();

    const result = await runMigrations(projectRoot, '0.8.5');
    expect(result.success).toBe(true);

    const config = JSON.parse(fs.readFileSync(path.join(projectRoot, '.mint', 'config.json'), 'utf8'));
    expect(config.mintVersion).toBe('0.8.5');
    expect(config.appliedMigrations).toContain('0.8.4-to-0.8.5');
    // Pre-existing migration history preserved.
    expect(config.appliedMigrations).toContain('0.7.6-to-0.8.0');
  });

  test('idempotent — running migrations a second time is a no-op', async () => {
    seedFakeProject();

    const first = await runMigrations(projectRoot, '0.8.5');
    expect(first.success).toBe(true);
    expect(first.applied.length).toBeGreaterThan(0);

    // Snapshot post-migration state.
    const tasksBefore = fs.readdirSync(path.join(projectRoot, '.mint', 'tasks')).sort();
    const sessionsBefore = fs.readdirSync(path.join(projectRoot, '.mint', 'sessions')).sort();
    const configBefore = fs.readFileSync(path.join(projectRoot, '.mint', 'config.json'), 'utf8');

    // Run again.
    const second = await runMigrations(projectRoot, '0.8.5');
    expect(second.success).toBe(true);
    // No pending migrations after first run.
    expect(second.applied).toEqual([]);

    // Filesystem state unchanged.
    expect(fs.readdirSync(path.join(projectRoot, '.mint', 'tasks')).sort()).toEqual(tasksBefore);
    expect(fs.readdirSync(path.join(projectRoot, '.mint', 'sessions')).sort()).toEqual(sessionsBefore);
    expect(fs.readFileSync(path.join(projectRoot, '.mint', 'config.json'), 'utf8')).toBe(configBefore);
  });

  test('symlink entry in .mint/tasks/ is skipped (not moved) with stderr warning', async () => {
    seedFakeProject();

    // Replace one feature dir with a symlink pointing somewhere outside the project.
    const tasksDir = path.join(projectRoot, '.mint', 'tasks');
    fs.rmSync(path.join(tasksDir, 'second-feature'), { recursive: true });
    const linkTarget = fs.mkdtempSync(path.join(os.tmpdir(), 'mint-mig-link-target-'));
    try {
      fs.symlinkSync(linkTarget, path.join(tasksDir, 'second-feature'), 'dir');

      // Capture stderr by stubbing process.stderr.write briefly.
      const origWrite = process.stderr.write.bind(process.stderr);
      let captured = '';
      process.stderr.write = (chunk, ...rest) => {
        captured += typeof chunk === 'string' ? chunk : chunk.toString();
        return origWrite(chunk, ...rest);
      };

      try {
        const result = await runMigrations(projectRoot, '0.8.5');
        expect(result.success).toBe(true);
      } finally {
        process.stderr.write = origWrite;
      }

      // Symlink must not have been quarantined.
      expect(fs.existsSync(path.join(tasksDir, '_legacy-pre-0.8.5', 'second-feature'))).toBe(false);
      // Original symlink still in place (was skipped, not moved).
      expect(fs.lstatSync(path.join(tasksDir, 'second-feature')).isSymbolicLink()).toBe(true);
      // Non-symlink feature was migrated.
      expect(fs.existsSync(path.join(tasksDir, '_legacy-pre-0.8.5', 'old-feature', '001', 'execution.json'))).toBe(true);
      // Warning surfaced to stderr.
      expect(captured).toMatch(/symlink/i);
    } finally {
      fs.rmSync(linkTarget, { recursive: true, force: true });
    }
  });
});
