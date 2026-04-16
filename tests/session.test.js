import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import os from 'os';
import path from 'path';
import { spawn, spawnSync } from 'child_process';
import {
  getSessionId,
  getSessionPath,
  writeSessionState,
  readSessionState,
  deleteSessionState,
  listSessions,
  cleanStaleSessions,
  findCurrentSession,
  touchHeartbeat,
  reclaimOrphanedSessions,
  _resetSessionId,
} from '../cli/lib/session.js';

const TMP = path.join(import.meta.dir, '.tmp-session');

beforeEach(() => {
  fs.mkdirSync(path.join(TMP, '.mint'), { recursive: true });
  // Set a predictable session ID for tests
  _resetSessionId('test-session-001');
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
  _resetSessionId();
});

describe('getSessionId', () => {
  test('returns cached ID across calls', () => {
    _resetSessionId();
    const id1 = getSessionId();
    const id2 = getSessionId();
    expect(id1).toBe(id2);
  });

  test('generates timestamp-random format', () => {
    _resetSessionId();
    const id = getSessionId();
    // Format: 12-char hex timestamp + dash + 8-char hex random
    expect(id).toMatch(/^[a-f0-9]{12}-[a-f0-9]{8}$/);
  });

  test('timestamp prefix is sortable', () => {
    _resetSessionId();
    const id1 = getSessionId();
    // Small delay to ensure different timestamp
    const before = Date.now();
    _resetSessionId();
    const id2 = getSessionId();
    // Lexicographic sort should match chronological order
    expect(id1 < id2 || before === Date.now()).toBe(true);
  });

  test('respects forced ID from _resetSessionId', () => {
    _resetSessionId('my-custom-id');
    expect(getSessionId()).toBe('my-custom-id');
  });
});

describe('session CRUD', () => {
  const state = {
    mintInvoked: true,
    invokedAt: new Date().toISOString(),
    task: 'test task',
    mode: 'quick',
    autoCommitOverride: null,
    designContextLoaded: false,
  };

  test('write and read session state', () => {
    writeSessionState(TMP, state);
    const read = readSessionState(TMP);
    // State fields are preserved; writeSessionState also stamps liveness fields.
    expect(read).toMatchObject(state);
  });

  test('read returns null for non-existent session', () => {
    expect(readSessionState(TMP, 'nonexistent')).toBeNull();
  });

  test('delete session state', () => {
    writeSessionState(TMP, state);
    expect(deleteSessionState(TMP)).toBe(true);
    expect(readSessionState(TMP)).toBeNull();
  });

  test('delete non-existent returns false', () => {
    expect(deleteSessionState(TMP, 'nonexistent')).toBe(false);
  });

  test('session file goes in sessions/ dir', () => {
    writeSessionState(TMP, state);
    const sessionPath = getSessionPath(TMP);
    expect(sessionPath).toContain('.mint/sessions/test-session-001.json');
    expect(fs.existsSync(sessionPath)).toBe(true);
  });
});

describe('listSessions', () => {
  test('lists all active sessions', () => {
    writeSessionState(TMP, { mintInvoked: true, task: 'task-1', mode: 'quick' }, 'session-a');
    writeSessionState(TMP, { mintInvoked: true, task: 'task-2', mode: 'plan' }, 'session-b');

    const sessions = listSessions(TMP);
    expect(sessions).toHaveLength(2);
    expect(sessions.map(s => s.id).sort()).toEqual(['session-a', 'session-b']);
  });

  test('returns empty array when no sessions dir', () => {
    expect(listSessions(TMP)).toEqual([]);
  });
});

describe('cleanStaleSessions', () => {
  test('removes sessions older than maxAge', () => {
    const old = new Date(Date.now() - 100_000).toISOString();
    const recent = new Date().toISOString();

    writeSessionState(TMP, { mintInvoked: true, invokedAt: old, task: 'old' }, 'stale');
    writeSessionState(TMP, { mintInvoked: true, invokedAt: recent, task: 'fresh' }, 'fresh');

    const cleaned = cleanStaleSessions(TMP, 50_000);
    expect(cleaned).toBe(1);

    const remaining = listSessions(TMP);
    expect(remaining).toHaveLength(1);
    expect(remaining[0].id).toBe('fresh');
  });
});

describe('findCurrentSession', () => {
  test('finds session by cached ID', () => {
    const state = { mintInvoked: true, task: 'my task', mode: 'plan' };
    writeSessionState(TMP, state);

    const found = findCurrentSession(TMP);
    expect(found.state).toMatchObject(state);
    expect(found.sessionId).toBe('test-session-001');
  });

  test('returns null when no session exists', () => {
    const found = findCurrentSession(TMP);
    expect(found.state).toBeNull();
    expect(found.sessionId).toBeNull();
  });
});

describe('concurrent session isolation', () => {
  test('two sessions do not interfere', () => {
    const state1 = { mintInvoked: true, task: 'task-1', mode: 'quick', activeSpec: 'spec-001.xml' };
    const state2 = { mintInvoked: true, task: 'task-2', mode: 'plan', activeSpec: 'spec-002.xml' };

    writeSessionState(TMP, state1, 'session-1');
    writeSessionState(TMP, state2, 'session-2');

    // Each session reads its own state
    const read1 = readSessionState(TMP, 'session-1');
    const read2 = readSessionState(TMP, 'session-2');

    expect(read1.task).toBe('task-1');
    expect(read1.activeSpec).toBe('spec-001.xml');
    expect(read2.task).toBe('task-2');
    expect(read2.activeSpec).toBe('spec-002.xml');

    // Deleting one doesn't affect the other
    deleteSessionState(TMP, 'session-1');
    expect(readSessionState(TMP, 'session-1')).toBeNull();
    expect(readSessionState(TMP, 'session-2')).toMatchObject(state2);
  });
});

describe('getSessionId capture-file integration', () => {
  let root;

  beforeEach(() => {
    // Build a real project root with a .mint/sessions/ dir — no mocks.
    root = fs.mkdtempSync(path.join(os.tmpdir(), 'mint-session-'));
    fs.mkdirSync(path.join(root, '.mint', 'sessions'), { recursive: true });
    // Clear any cached id so getSessionId re-reads.
    _resetSessionId();
  });

  afterEach(() => {
    fs.rmSync(root, { recursive: true, force: true });
    _resetSessionId();
  });

  test('getSessionId reads .current-session-id when present', () => {
    const captured = 'abc123-claude-session';
    fs.writeFileSync(path.join(root, '.mint', 'sessions', '.current-session-id'), captured + '\n');

    expect(getSessionId(root)).toBe(captured);
    // Cached on second call too.
    expect(getSessionId(root)).toBe(captured);
  });

  test('getSessionId falls back to generated id when capture file missing', () => {
    const id = getSessionId(root);
    expect(id).toMatch(/^[a-f0-9]{12}-[a-f0-9]{8}$/);
  });

  test('getSessionId falls back when capture file is empty', () => {
    fs.writeFileSync(path.join(root, '.mint', 'sessions', '.current-session-id'), '   \n');
    const id = getSessionId(root);
    expect(id).toMatch(/^[a-f0-9]{12}-[a-f0-9]{8}$/);
  });
});

describe('session-start-capture hook', () => {
  let projectDir;
  const hookPath = path.join(import.meta.dir, '..', 'hooks', 'scripts', 'session-start-capture.cjs');

  beforeEach(() => {
    projectDir = fs.mkdtempSync(path.join(os.tmpdir(), 'mint-hook-'));
  });

  afterEach(() => {
    fs.rmSync(projectDir, { recursive: true, force: true });
  });

  test('SessionStart hook writes stdin session_id to .current-session-id', () => {
    const sessionId = 'claude-sess-xyz-789';
    const result = spawnSync('node', [hookPath], {
      input: JSON.stringify({ session_id: sessionId, hook_event_name: 'SessionStart' }),
      env: { ...process.env, CLAUDE_PROJECT_DIR: projectDir },
      encoding: 'utf8',
    });

    expect(result.status).toBe(0);
    const captured = fs.readFileSync(
      path.join(projectDir, '.mint', 'sessions', '.current-session-id'),
      'utf8',
    ).trim();
    expect(captured).toBe(sessionId);
  });

  test('SessionStart hook is non-blocking on bad stdin', () => {
    const result = spawnSync('node', [hookPath], {
      input: 'not json at all',
      env: { ...process.env, CLAUDE_PROJECT_DIR: projectDir },
      encoding: 'utf8',
    });
    // Must exit 0 — SessionStart hooks are never allowed to block.
    expect(result.status).toBe(0);
    expect(fs.existsSync(path.join(projectDir, '.mint', 'sessions', '.current-session-id'))).toBe(false);
  });
});

describe('writeSessionState liveness auto-stamp', () => {
  test('fresh session file includes pid, startTime, lastHeartbeat', () => {
    writeSessionState(TMP, { mintInvoked: true, task: 't', mode: 'quick' });
    const read = readSessionState(TMP);
    expect(read.pid).toBe(process.pid);
    expect(typeof read.startTime).toBe('number');
    expect(read.startTime).toBeGreaterThan(0);
    expect(typeof read.lastHeartbeat).toBe('string');
    // Valid ISO timestamp.
    expect(Number.isNaN(new Date(read.lastHeartbeat).getTime())).toBe(false);
  });

  test('preserves caller-provided liveness fields', () => {
    writeSessionState(
      TMP,
      { mintInvoked: true, pid: 42, startTime: 12345, lastHeartbeat: '2020-01-01T00:00:00.000Z' },
      'forced',
    );
    const read = readSessionState(TMP, 'forced');
    expect(read.pid).toBe(42);
    expect(read.startTime).toBe(12345);
    expect(read.lastHeartbeat).toBe('2020-01-01T00:00:00.000Z');
  });
});

describe('touchHeartbeat', () => {
  test('updates only lastHeartbeat', () => {
    writeSessionState(
      TMP,
      {
        mintInvoked: true,
        task: 'orig',
        mode: 'plan',
        pid: 999,
        startTime: 111,
        lastHeartbeat: '2020-01-01T00:00:00.000Z',
      },
      'heartbeat-target',
    );

    const before = readSessionState(TMP, 'heartbeat-target');
    const ok = touchHeartbeat(TMP, 'heartbeat-target');
    const after = readSessionState(TMP, 'heartbeat-target');

    expect(ok).toBe(true);
    // Only lastHeartbeat changed.
    expect(after.task).toBe(before.task);
    expect(after.mode).toBe(before.mode);
    expect(after.pid).toBe(before.pid);
    expect(after.startTime).toBe(before.startTime);
    expect(after.mintInvoked).toBe(before.mintInvoked);
    expect(after.lastHeartbeat).not.toBe(before.lastHeartbeat);
    expect(new Date(after.lastHeartbeat).getTime()).toBeGreaterThan(
      new Date(before.lastHeartbeat).getTime(),
    );
  });

  test('returns false when session file missing', () => {
    expect(touchHeartbeat(TMP, 'does-not-exist')).toBe(false);
  });
});

describe('reclaimOrphanedSessions', () => {
  // All tests use forced non-current session ids (current is 'test-session-001')
  // so reclaim never treats them as the current session.

  test('removes session with dead pid', async () => {
    // Spawn a short-lived child and wait for it to exit so we have a dead pid.
    const child = spawn(process.execPath, ['-e', 'process.exit(0)']);
    const deadPid = child.pid;
    await new Promise(resolve => child.on('exit', resolve));

    writeSessionState(
      TMP,
      {
        mintInvoked: true,
        task: 'dead',
        pid: deadPid,
        startTime: Date.now(),
        lastHeartbeat: new Date().toISOString(),
      },
      'dead-session',
    );

    const count = reclaimOrphanedSessions(TMP);
    expect(count).toBe(1);
    expect(readSessionState(TMP, 'dead-session')).toBeNull();
  });

  test('preserves session with live pid and fresh heartbeat', () => {
    writeSessionState(
      TMP,
      {
        mintInvoked: true,
        task: 'live',
        pid: process.pid,
        startTime: Date.now(),
        lastHeartbeat: new Date().toISOString(),
      },
      'live-session',
    );

    const count = reclaimOrphanedSessions(TMP);
    expect(count).toBe(0);
    expect(readSessionState(TMP, 'live-session')).not.toBeNull();
  });

  test('reclaims session with live pid but stale heartbeat', () => {
    const stale = new Date(Date.now() - 2 * 3_600_000).toISOString();
    writeSessionState(
      TMP,
      {
        mintInvoked: true,
        task: 'stale-beat',
        pid: process.pid,
        startTime: Date.now(),
        lastHeartbeat: stale,
      },
      'stale-beat-session',
    );

    const count = reclaimOrphanedSessions(TMP);
    expect(count).toBe(1);
    expect(readSessionState(TMP, 'stale-beat-session')).toBeNull();
  });

  test('reclaims session with missing pid field', () => {
    // Manually write a pre-migration state: no pid, no liveness fields.
    // Use fs directly to bypass writeSessionState auto-stamping.
    const dir = path.join(TMP, '.mint', 'sessions');
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(
      path.join(dir, 'pre-migration.json'),
      JSON.stringify({ mintInvoked: true, task: 'legacy' }, null, 2),
    );

    const count = reclaimOrphanedSessions(TMP);
    expect(count).toBe(1);
    expect(fs.existsSync(path.join(dir, 'pre-migration.json'))).toBe(false);
  });

  test('never reclaims the current session even with stale heartbeat', () => {
    // The current session id (forced by _resetSessionId in beforeEach) is
    // 'test-session-001'. Write it with our live pid but stale heartbeat.
    const stale = new Date(Date.now() - 2 * 3_600_000).toISOString();
    writeSessionState(TMP, {
      mintInvoked: true,
      task: 'self',
      pid: process.pid,
      startTime: Date.now(),
      lastHeartbeat: stale,
    });

    const count = reclaimOrphanedSessions(TMP);
    expect(count).toBe(0);
    expect(readSessionState(TMP)).not.toBeNull();
  });

  test('removes .mint/tasks/<session-id>/ tree on orphan', async () => {
    const child = spawn(process.execPath, ['-e', 'process.exit(0)']);
    const deadPid = child.pid;
    await new Promise(resolve => child.on('exit', resolve));

    const sessionId = 'orphan-with-tasks';
    writeSessionState(
      TMP,
      {
        mintInvoked: true,
        task: 'orphan-tasks',
        pid: deadPid,
        startTime: Date.now(),
        lastHeartbeat: new Date().toISOString(),
      },
      sessionId,
    );

    // Seed a tasks tree for this session.
    const tasksDir = path.join(TMP, '.mint', 'tasks', sessionId);
    fs.mkdirSync(path.join(tasksDir, 'nested'), { recursive: true });
    fs.writeFileSync(path.join(tasksDir, 'spec.xml'), '<task/>');
    fs.writeFileSync(path.join(tasksDir, 'nested', 'artifact.json'), '{}');

    const count = reclaimOrphanedSessions(TMP);
    expect(count).toBe(1);
    expect(fs.existsSync(tasksDir)).toBe(false);
  });

  test('treats sentinel pids (0, -1, 1) as dead', () => {
    // Bypass writeSessionState auto-stamping: write raw files with sentinel pids.
    const dir = path.join(TMP, '.mint', 'sessions');
    fs.mkdirSync(dir, { recursive: true });
    const fresh = new Date().toISOString();
    for (const sentinel of [0, -1, 1]) {
      fs.writeFileSync(
        path.join(dir, `sentinel-${sentinel}.json`),
        JSON.stringify({
          mintInvoked: true,
          task: `sentinel-${sentinel}`,
          pid: sentinel,
          startTime: Date.now(),
          lastHeartbeat: fresh,
        }),
      );
    }

    const count = reclaimOrphanedSessions(TMP);
    expect(count).toBe(3);
    expect(fs.existsSync(path.join(dir, 'sentinel-0.json'))).toBe(false);
    expect(fs.existsSync(path.join(dir, 'sentinel--1.json'))).toBe(false);
    expect(fs.existsSync(path.join(dir, 'sentinel-1.json'))).toBe(false);
  });

  test('treats future-dated heartbeat as stale', () => {
    const tomorrow = new Date(Date.now() + 86_400_000).toISOString();
    writeSessionState(
      TMP,
      {
        mintInvoked: true,
        task: 'future-beat',
        pid: process.pid,
        startTime: Date.now(),
        lastHeartbeat: tomorrow,
      },
      'future-beat-session',
    );

    const count = reclaimOrphanedSessions(TMP);
    expect(count).toBe(1);
    expect(readSessionState(TMP, 'future-beat-session')).toBeNull();
  });

  test('integration — dead pid session with populated tasks tree fully cleaned', async () => {
    // End-to-end: simulate an abandoned session that left behind a session file
    // AND a tasks tree with execution.json + pipeline-state.json. After
    // reclaimOrphanedSessions, BOTH the session file and the entire per-session
    // tasks tree must be gone — nothing left for the Stop hook or the
    // workflow tracer to trip over.
    const child = spawn(process.execPath, ['-e', 'process.exit(0)']);
    const deadPid = child.pid;
    await new Promise(resolve => child.on('exit', resolve));

    const sessionId = 'integration-orphan-session';
    writeSessionState(
      TMP,
      {
        mintInvoked: true,
        task: 'half-finished feature',
        mode: 'plan',
        pid: deadPid,
        startTime: Date.now(),
        lastHeartbeat: new Date().toISOString(),
      },
      sessionId,
    );

    // Seed a realistic per-session tasks tree.
    const featureDir = path.join(TMP, '.mint', 'tasks', sessionId, 'half-feature', '001');
    fs.mkdirSync(featureDir, { recursive: true });
    fs.writeFileSync(path.join(featureDir, 'execution.json'), JSON.stringify({
      status: 'running',
      gates: { lint: 'pass', types: 'pass' },
      reviews: {},
      filesModified: ['src/half.ts'],
    }));
    fs.writeFileSync(path.join(featureDir, 'pipeline-state.json'), JSON.stringify({
      currentStep: 'review-stage1',
      pendingSteps: ['review-stage1', 'docs', 'dod'],
    }));
    fs.writeFileSync(path.join(featureDir, 'spec.xml'), '<task><id>001</id></task>');

    // A second session in the same project that should NOT be reclaimed.
    writeSessionState(
      TMP,
      {
        mintInvoked: true,
        task: 'live work',
        mode: 'plan',
        pid: process.pid,
        startTime: Date.now(),
        lastHeartbeat: new Date().toISOString(),
      },
      'integration-live-sibling',
    );
    fs.mkdirSync(path.join(TMP, '.mint', 'tasks', 'integration-live-sibling', 'live-feat'), { recursive: true });

    // Run the reclaim.
    const count = reclaimOrphanedSessions(TMP);
    expect(count).toBe(1);

    // Dead session: file + entire tasks subtree are gone.
    expect(readSessionState(TMP, sessionId)).toBeNull();
    expect(fs.existsSync(path.join(TMP, '.mint', 'tasks', sessionId))).toBe(false);

    // Live sibling: untouched.
    expect(readSessionState(TMP, 'integration-live-sibling')).not.toBeNull();
    expect(fs.existsSync(path.join(TMP, '.mint', 'tasks', 'integration-live-sibling'))).toBe(true);
  });

  test('respects staleHeartbeatMs option', () => {
    const slightlyOld = new Date(Date.now() - 5_000).toISOString();
    writeSessionState(
      TMP,
      {
        mintInvoked: true,
        task: 'short-tolerance',
        pid: process.pid,
        startTime: Date.now(),
        lastHeartbeat: slightlyOld,
      },
      'tight-window',
    );

    // 1s tolerance — a 5s-old heartbeat counts as stale.
    const count = reclaimOrphanedSessions(TMP, { staleHeartbeatMs: 1_000 });
    expect(count).toBe(1);
    expect(readSessionState(TMP, 'tight-window')).toBeNull();
  });
});
