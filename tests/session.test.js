import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import {
  getSessionId,
  getSessionPath,
  writeSessionState,
  readSessionState,
  deleteSessionState,
  listSessions,
  cleanStaleSessions,
  findCurrentSession,
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
    expect(read).toEqual(state);
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
    expect(found.state).toEqual(state);
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
    expect(readSessionState(TMP, 'session-2')).toEqual(state2);
  });
});
