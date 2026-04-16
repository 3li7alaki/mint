import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';

// CJS module — use require via createRequire
import { createRequire } from 'module';
const require = createRequire(import.meta.url);
const {
  parseGitOperations,
  computeFingerprint,
  findActiveSession,
  extractExecutionOps,
  buildTrace,
} = require('../hooks/scripts/workflow-tracer.cjs');

const TMP = path.join(import.meta.dir, '.tmp-tracer');

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('parseGitOperations', () => {
  test('extracts conventional commits', () => {
    const log = 'feat(auth): add login flow\nfix: typo in header\n';
    const ops = parseGitOperations(log);
    expect(ops).toHaveLength(2);
    expect(ops[0]).toEqual({ type: 'commit', message: 'feat(auth): add login flow' });
    expect(ops[1]).toEqual({ type: 'commit', message: 'fix: typo in header' });
  });

  test('detects merge commits', () => {
    const log = 'Merge branch main into feat/auth\n';
    const ops = parseGitOperations(log);
    expect(ops).toHaveLength(1);
    expect(ops[0].type).toBe('git');
    expect(ops[0].action).toBe('merge');
  });

  test('detects cherry-pick commits', () => {
    const log = 'cherry-pick abc1234\n';
    const ops = parseGitOperations(log);
    expect(ops).toHaveLength(1);
    expect(ops[0].type).toBe('git');
    expect(ops[0].action).toBe('cherry-pick');
  });

  test('detects revert commits', () => {
    const log = 'Revert "feat: bad feature"\n';
    const ops = parseGitOperations(log);
    expect(ops).toHaveLength(1);
    expect(ops[0].type).toBe('git');
    expect(ops[0].action).toBe('revert');
  });

  test('handles empty git log', () => {
    expect(parseGitOperations('')).toEqual([]);
    expect(parseGitOperations(null)).toEqual([]);
    expect(parseGitOperations(undefined)).toEqual([]);
  });

  test('handles whitespace-only log', () => {
    expect(parseGitOperations('   \n  \n')).toEqual([]);
  });

  test('generic non-conventional commits', () => {
    const log = 'initial commit\nwip stuff\n';
    const ops = parseGitOperations(log);
    expect(ops).toHaveLength(2);
    expect(ops[0]).toEqual({ type: 'commit', message: 'initial commit' });
    expect(ops[1]).toEqual({ type: 'commit', message: 'wip stuff' });
  });
});

describe('computeFingerprint', () => {
  test('produces colon-separated type-action string', () => {
    const ops = [
      { type: 'git', action: 'fetch' },
      { type: 'edit' },
      { type: 'gate' },
      { type: 'commit' },
    ];
    expect(computeFingerprint(ops)).toBe('git-fetch:edit:gate:commit');
  });

  test('uses just type when no action', () => {
    const ops = [{ type: 'edit' }, { type: 'gate' }];
    expect(computeFingerprint(ops)).toBe('edit:gate');
  });

  test('returns empty string for empty/null/undefined', () => {
    expect(computeFingerprint([])).toBe('');
    expect(computeFingerprint(null)).toBe('');
    expect(computeFingerprint(undefined)).toBe('');
  });

  test('mixed operations produce correct fingerprint', () => {
    const ops = [
      { type: 'edit', files: ['a.ts'] },
      { type: 'gate', result: 'pass' },
      { type: 'commit', message: 'feat: something' },
      { type: 'git', action: 'merge' },
    ];
    expect(computeFingerprint(ops)).toBe('edit:gate:commit:git-merge');
  });
});

describe('findActiveSession', () => {
  test('returns null when sessions dir does not exist', () => {
    const result = findActiveSession(path.join(TMP, 'nonexistent'));
    expect(result).toBeNull();
  });

  test('returns null when no sessions have mintInvoked', () => {
    const sessDir = path.join(TMP, 'sessions');
    fs.mkdirSync(sessDir, { recursive: true });
    fs.writeFileSync(
      path.join(sessDir, '20260403-abc.json'),
      JSON.stringify({ mintInvoked: false, mode: 'quick' })
    );
    const result = findActiveSession(sessDir);
    expect(result).toBeNull();
  });

  test('finds most recent active session', () => {
    const sessDir = path.join(TMP, 'sessions');
    fs.mkdirSync(sessDir, { recursive: true });
    fs.writeFileSync(
      path.join(sessDir, '20260401-old.json'),
      JSON.stringify({ mintInvoked: true, mode: 'quick', task: 'old task' })
    );
    fs.writeFileSync(
      path.join(sessDir, '20260403-new.json'),
      JSON.stringify({ mintInvoked: true, mode: 'plan', task: 'new task' })
    );
    const result = findActiveSession(sessDir);
    expect(result).not.toBeNull();
    expect(result.sessionId).toBe('20260403-new');
    expect(result.session.task).toBe('new task');
  });

  test('handles empty sessions directory', () => {
    const sessDir = path.join(TMP, 'sessions');
    fs.mkdirSync(sessDir, { recursive: true });
    const result = findActiveSession(sessDir);
    expect(result).toBeNull();
  });

  test('skips invalid JSON files gracefully', () => {
    const sessDir = path.join(TMP, 'sessions');
    fs.mkdirSync(sessDir, { recursive: true });
    fs.writeFileSync(path.join(sessDir, '20260403-bad.json'), 'not json');
    fs.writeFileSync(
      path.join(sessDir, '20260402-good.json'),
      JSON.stringify({ mintInvoked: true, mode: 'quick', task: 'good' })
    );
    const result = findActiveSession(sessDir);
    expect(result).not.toBeNull();
    expect(result.session.task).toBe('good');
  });
});

describe('extractExecutionOps', () => {
  test('returns empty array when tasks dir does not exist', () => {
    const ops = extractExecutionOps(path.join(TMP, 'nonexistent'));
    expect(ops).toEqual([]);
  });

  test('extracts gate results from execution.json', () => {
    const specDir = path.join(TMP, 'tasks', 'my-task', '001');
    fs.mkdirSync(specDir, { recursive: true });
    fs.writeFileSync(
      path.join(specDir, 'execution.json'),
      JSON.stringify({
        gates: { lint: 'pass', types: 'pass', tests: 'pass' },
      })
    );
    const ops = extractExecutionOps(path.join(TMP, 'tasks'));
    const gateOps = ops.filter(o => o.type === 'gate');
    expect(gateOps).toHaveLength(1);
    expect(gateOps[0].result).toBe('pass');
  });

  test('detects gate failure', () => {
    const specDir = path.join(TMP, 'tasks', 'my-task', '001');
    fs.mkdirSync(specDir, { recursive: true });
    fs.writeFileSync(
      path.join(specDir, 'execution.json'),
      JSON.stringify({
        gates: { lint: 'pass', types: 'fail' },
      })
    );
    const ops = extractExecutionOps(path.join(TMP, 'tasks'));
    const gateOps = ops.filter(o => o.type === 'gate');
    expect(gateOps).toHaveLength(1);
    expect(gateOps[0].result).toBe('fail');
  });

  test('extracts filesModified as edit operation', () => {
    const specDir = path.join(TMP, 'tasks', 'my-task', '001');
    fs.mkdirSync(specDir, { recursive: true });
    fs.writeFileSync(
      path.join(specDir, 'execution.json'),
      JSON.stringify({
        gates: { tests: 'pass' },
        filesModified: ['src/foo.ts', 'src/bar.ts'],
      })
    );
    const ops = extractExecutionOps(path.join(TMP, 'tasks'));
    const editOps = ops.filter(o => o.type === 'edit');
    expect(editOps).toHaveLength(1);
    expect(editOps[0].files).toEqual(['src/foo.ts', 'src/bar.ts']);
  });

  test('skips execution with no gates', () => {
    const specDir = path.join(TMP, 'tasks', 'my-task', '001');
    fs.mkdirSync(specDir, { recursive: true });
    fs.writeFileSync(
      path.join(specDir, 'execution.json'),
      JSON.stringify({ status: 'running' })
    );
    const ops = extractExecutionOps(path.join(TMP, 'tasks'));
    expect(ops).toEqual([]);
  });

  test('handles invalid execution.json gracefully', () => {
    const specDir = path.join(TMP, 'tasks', 'my-task', '001');
    fs.mkdirSync(specDir, { recursive: true });
    fs.writeFileSync(path.join(specDir, 'execution.json'), 'not json');
    const ops = extractExecutionOps(path.join(TMP, 'tasks'));
    expect(ops).toEqual([]);
  });
});

describe('buildTrace session scoping', () => {
  // Per .mint/issues.jsonl follow-up from spec 003: confirm buildTrace reads from
  // .mint/tasks/<session-id>/ only, never the global tasks tree.
  //
  // We don't init a real git repo in TMP, so git log inside buildTrace fails
  // and gitOps come back empty — that's intentional. We only assert on
  // execOps which come from the session-scoped tasks dir.

  function seedSession(root, sessionId, slug, gates) {
    const specDir = path.join(root, '.mint', 'tasks', sessionId, slug, '001');
    fs.mkdirSync(specDir, { recursive: true });
    fs.writeFileSync(
      path.join(specDir, 'execution.json'),
      JSON.stringify({ gates, filesModified: [`src/${slug}.ts`] }),
    );
  }

  test('buildTrace scopes execution scan to .mint/tasks/<session-id>/ only', () => {
    fs.mkdirSync(path.join(TMP, '.mint'), { recursive: true });
    seedSession(TMP, 'sess-A', 'feat-a', { lint: 'pass', types: 'pass' });
    // Drop a global / legacy entry that should NOT bleed into A's trace.
    const legacyDir = path.join(TMP, '.mint', 'tasks', 'legacy-feat', '001');
    fs.mkdirSync(legacyDir, { recursive: true });
    fs.writeFileSync(
      path.join(legacyDir, 'execution.json'),
      JSON.stringify({ gates: { tests: 'fail' }, filesModified: ['src/leak.ts'] }),
    );

    const trace = buildTrace(
      TMP,
      { session: { task: 'A', mode: 'plan', invokedAt: new Date().toISOString() } },
      'sess-A',
    );

    // Trace must reference the session id and contain only A's files.
    expect(trace.sessionId).toBe('sess-A');
    const editOps = trace.operations.filter(o => o.type === 'edit');
    const allFiles = editOps.flatMap(o => o.files);
    expect(allFiles).toContain('src/feat-a.ts');
    expect(allFiles).not.toContain('src/leak.ts');

    // Outcome derives from A's gates only (all pass).
    expect(trace.outcome).toBe('success');
  });

  test('two sessions do not cross-pollinate execution operations', () => {
    fs.mkdirSync(path.join(TMP, '.mint'), { recursive: true });
    // A is clean; B has a failing gate.
    seedSession(TMP, 'sess-A', 'feat-a', { lint: 'pass', types: 'pass' });
    seedSession(TMP, 'sess-B', 'feat-b', { lint: 'pass', types: 'fail' });

    const traceA = buildTrace(
      TMP,
      { session: { task: 'A', mode: 'plan', invokedAt: new Date().toISOString() } },
      'sess-A',
    );
    const traceB = buildTrace(
      TMP,
      { session: { task: 'B', mode: 'plan', invokedAt: new Date().toISOString() } },
      'sess-B',
    );

    const filesA = traceA.operations.filter(o => o.type === 'edit').flatMap(o => o.files);
    const filesB = traceB.operations.filter(o => o.type === 'edit').flatMap(o => o.files);

    expect(filesA).toEqual(['src/feat-a.ts']);
    expect(filesB).toEqual(['src/feat-b.ts']);

    // A's outcome is success (no failing gates), B's is failure.
    expect(traceA.outcome).toBe('success');
    expect(traceB.outcome).toBe('failure');
  });

  test('buildTrace returns empty execOps when session has no tasks dir', () => {
    fs.mkdirSync(path.join(TMP, '.mint'), { recursive: true });
    // Sibling session has work but the queried session does not — must come back clean.
    seedSession(TMP, 'sess-other', 'feat-other', { tests: 'pass' });

    const trace = buildTrace(
      TMP,
      { session: { task: 'mine', mode: 'plan', invokedAt: new Date().toISOString() } },
      'sess-mine',
    );

    const execOps = trace.operations.filter(o => o.type === 'edit' || o.type === 'gate');
    expect(execOps).toEqual([]);
  });
});
