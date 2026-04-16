/**
 * Session state management — per-session isolation.
 *
 * Each Claude Code session gets its own state file at .mint/sessions/<session-id>.json.
 * Session ID is generated once per process: a timestamp prefix (ms since epoch, hex)
 * + random suffix — sortable by creation time, unique across concurrent sessions.
 *
 * This replaces the old single .mint/.session-state.json which caused concurrent
 * sessions to stomp on each other's state.
 */
import fs from 'fs';
import path from 'path';
import { randomBytes } from 'crypto';
import { atomicWriteJsonSync } from './atomic.js';
import { readJsonSafe } from './detect.js';

const SESSIONS_DIR = '.mint/sessions';
const CURRENT_SESSION_FILE = '.current-session-id';

/** Cached session ID — stable for the lifetime of this process. */
let _sessionId = null;

/**
 * Generate a session ID: hex timestamp (ms) + random suffix.
 * Example: "0195e3a1b2c0-a1b2c3d4" — sortable, unique, no env var needed.
 */
function generateSessionId() {
  const timestamp = Date.now().toString(16).padStart(12, '0');
  const random = randomBytes(4).toString('hex');
  return `${timestamp}-${random}`;
}

/**
 * Walk up from `start` looking for a directory that contains a `.mint/`
 * subdirectory. Mirrors the pattern used by hooks/scripts/*.cjs — same
 * semantics so CLI and hooks agree on project root.
 *
 * @param {string} [start]
 * @returns {string|null}
 */
function findProjectRoot(start) {
  let dir = start || process.cwd();
  while (dir !== path.dirname(dir)) {
    if (fs.existsSync(path.join(dir, '.mint'))) return dir;
    dir = path.dirname(dir);
  }
  return null;
}

/**
 * Try to read Claude's session_id captured by the SessionStart hook.
 * Returns the trimmed id, or null if the file is missing/empty/unreadable.
 *
 * @param {string} [projectRoot]
 * @returns {string|null}
 */
function readCapturedSessionId(projectRoot) {
  const root = projectRoot || findProjectRoot();
  if (!root) return null;
  try {
    const raw = fs.readFileSync(path.join(root, SESSIONS_DIR, CURRENT_SESSION_FILE), 'utf8');
    const id = raw.trim();
    return id.length > 0 ? id : null;
  } catch {
    return null;
  }
}

/**
 * Get the current session ID.
 *
 * Prefers Claude Code's own session_id (captured by the SessionStart hook at
 * `.mint/sessions/.current-session-id`) so mint's state files line up with
 * whatever Claude is using for the same conversation. Falls back to a
 * locally-generated id if the capture file is missing — that keeps mint
 * usable outside a live Claude session (tests, ad-hoc CLI runs).
 *
 * Cached once per process; use `_resetSessionId()` to force a re-read.
 *
 * @param {string} [projectRoot] - optional explicit root; otherwise walks up from cwd
 * @returns {string}
 */
export function getSessionId(projectRoot) {
  if (_sessionId) return _sessionId;

  const captured = readCapturedSessionId(projectRoot);
  if (captured) {
    _sessionId = captured;
    return _sessionId;
  }

  _sessionId = generateSessionId();
  return _sessionId;
}

/**
 * Reset the cached session ID. Only used for testing.
 * @param {string} [id] - Optional ID to force. If omitted, next getSessionId() generates fresh.
 */
export function _resetSessionId(id) {
  _sessionId = id || null;
}

/**
 * Get the sessions directory path for a project.
 *
 * @param {string} projectRoot
 * @returns {string}
 */
export function getSessionsDir(projectRoot) {
  return path.join(projectRoot, SESSIONS_DIR);
}

/**
 * Get the session state file path for a specific session.
 *
 * @param {string} projectRoot
 * @param {string} [sessionId] - defaults to current session
 * @returns {string}
 */
export function getSessionPath(projectRoot, sessionId) {
  const id = sessionId || getSessionId();
  return path.join(projectRoot, SESSIONS_DIR, `${id}.json`);
}

/**
 * Write session state atomically.
 *
 * Auto-stamps liveness fields (`pid`, `startTime`, `lastHeartbeat`) if absent so
 * every call-site gets a sidecar for orphan reclaim without needing to opt in.
 * Existing values are preserved — callers can override (e.g. `touchHeartbeat`).
 *
 * @param {string} projectRoot
 * @param {object} state - session state object
 * @param {string} [sessionId] - defaults to current session
 */
export function writeSessionState(projectRoot, state, sessionId) {
  const sessionPath = getSessionPath(projectRoot, sessionId);
  const stamped = {
    pid: process.pid,
    startTime: Date.now(),
    lastHeartbeat: new Date().toISOString(),
    ...state,
  };
  atomicWriteJsonSync(sessionPath, stamped);
}

/**
 * Update only the `lastHeartbeat` field on an existing session file.
 *
 * No-op if the session file does not exist. Atomic write.
 *
 * @param {string} projectRoot
 * @param {string} [sessionId] - defaults to current session
 * @returns {boolean} - true if heartbeat was updated, false if session missing
 */
export function touchHeartbeat(projectRoot, sessionId) {
  const id = sessionId || getSessionId(projectRoot);
  const state = readSessionState(projectRoot, id);
  if (!state) return false;
  const updated = { ...state, lastHeartbeat: new Date().toISOString() };
  const sessionPath = getSessionPath(projectRoot, id);
  atomicWriteJsonSync(sessionPath, updated);
  return true;
}

/**
 * Reclaim orphaned sessions — session files whose owning process is dead or
 * whose heartbeat is stale. Removes the session file AND its per-session tasks
 * tree at `.mint/tasks/<id>/`.
 *
 * Never reclaims the current session (defense against self-eviction).
 *
 * A session is considered orphaned when any of:
 *   - `pid` field is missing (pre-migration state)
 *   - `process.kill(pid, 0)` throws (process is dead)
 *   - `lastHeartbeat` is older than `staleHeartbeatMs`
 *
 * @param {string} projectRoot
 * @param {{ staleHeartbeatMs?: number }} [opts]
 * @returns {number} - count of reclaimed sessions
 */
export function reclaimOrphanedSessions(projectRoot, { staleHeartbeatMs = 3_600_000 } = {}) {
  const sessions = listSessions(projectRoot);
  const currentId = getSessionId(projectRoot);
  const now = Date.now();
  let reclaimed = 0;

  for (const { id, state } of sessions) {
    if (id === currentId) continue;

    // Only accept real user pids. Sentinel values (0, -1) target process groups
    // and 1 is init — `process.kill(pid, 0)` would always succeed for these,
    // pinning a corrupt/malicious session as "alive" forever.
    const pid = Number.isInteger(state?.pid) && state.pid > 1 ? state.pid : null;

    let pidAlive = false;
    if (pid !== null) {
      try {
        process.kill(pid, 0);
        pidAlive = true;
      } catch {
        pidAlive = false;
      }
    }

    const heartbeatMs = state && state.lastHeartbeat
      ? new Date(state.lastHeartbeat).getTime()
      : 0;
    // Clamp negative ages (future-dated heartbeats) to "stale" so an attacker
    // can't pin a dead session live by writing a far-future timestamp.
    const age = now - heartbeatMs;
    const heartbeatFresh = Number.isFinite(heartbeatMs) && age >= 0 && age <= staleHeartbeatMs;

    const orphan = pid === null || !pidAlive || !heartbeatFresh;
    if (!orphan) continue;

    // Remove session file.
    deleteSessionState(projectRoot, id);
    // Remove per-session tasks tree if present.
    const tasksDir = path.join(projectRoot, '.mint', 'tasks', id);
    try {
      fs.rmSync(tasksDir, { recursive: true, force: true });
    } catch {
      /* ignore — best effort cleanup */
    }
    reclaimed++;
  }

  return reclaimed;
}

/**
 * Read session state for a specific session.
 *
 * @param {string} projectRoot
 * @param {string} [sessionId] - defaults to current session
 * @returns {object|null}
 */
export function readSessionState(projectRoot, sessionId) {
  const sessionPath = getSessionPath(projectRoot, sessionId);
  return readJsonSafe(sessionPath);
}

/**
 * Delete session state file (cleanup on task completion).
 *
 * @param {string} projectRoot
 * @param {string} [sessionId] - defaults to current session
 * @returns {boolean} - true if deleted, false if didn't exist
 */
export function deleteSessionState(projectRoot, sessionId) {
  const sessionPath = getSessionPath(projectRoot, sessionId);
  try {
    fs.unlinkSync(sessionPath);
    return true;
  } catch {
    return false;
  }
}

/**
 * List all active sessions for a project.
 * Returns session state objects with their IDs.
 *
 * @param {string} projectRoot
 * @returns {Array<{ id: string, state: object }>}
 */
export function listSessions(projectRoot) {
  const sessionsDir = getSessionsDir(projectRoot);
  try {
    const files = fs.readdirSync(sessionsDir).filter(f => f.endsWith('.json'));
    return files.map(f => {
      const id = f.replace('.json', '');
      const state = readJsonSafe(path.join(sessionsDir, f));
      return state ? { id, state } : null;
    }).filter(Boolean);
  } catch {
    return [];
  }
}

/**
 * Clean up stale sessions (older than maxAge).
 *
 * @param {string} projectRoot
 * @param {number} [maxAgeMs=86400000] - max age in ms (default: 24h)
 * @returns {number} - number of sessions cleaned
 */
export function cleanStaleSessions(projectRoot, maxAgeMs = 86_400_000) {
  const sessions = listSessions(projectRoot);
  const now = Date.now();
  let cleaned = 0;

  for (const { id, state } of sessions) {
    const age = now - new Date(state.invokedAt || 0).getTime();
    if (age > maxAgeMs) {
      deleteSessionState(projectRoot, id);
      cleaned++;
    }
  }

  return cleaned;
}

/**
 * Find the active session state for the current session.
 *
 * @param {string} projectRoot
 * @returns {{ state: object|null, sessionId: string|null }}
 */
export function findCurrentSession(projectRoot) {
  const sessionId = getSessionId();
  const state = readSessionState(projectRoot, sessionId);
  if (state) return { state, sessionId };
  return { state: null, sessionId: null };
}
