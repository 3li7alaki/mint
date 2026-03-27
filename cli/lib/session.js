/**
 * Session state management — per-session isolation.
 *
 * Each Claude Code session gets its own state file at .mint/sessions/<session-id>.json.
 * Session ID comes from $CLAUDE_SESSION_ID (set by Claude Code) or a generated fallback.
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

/**
 * Get the current session ID.
 * Uses CLAUDE_SESSION_ID env var if available, otherwise generates a random one.
 *
 * @returns {string}
 */
export function getSessionId() {
  return process.env.CLAUDE_SESSION_ID || `local-${randomBytes(4).toString('hex')}`;
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
 * @param {string} projectRoot
 * @param {object} state - session state object
 * @param {string} [sessionId] - defaults to current session
 */
export function writeSessionState(projectRoot, state, sessionId) {
  const sessionPath = getSessionPath(projectRoot, sessionId);
  atomicWriteJsonSync(sessionPath, state);
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
