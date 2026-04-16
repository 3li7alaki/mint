#!/usr/bin/env node
/**
 * mint hook: SessionStart session_id capture
 * Trigger: SessionStart (startup|resume|clear|compact)
 *
 * Reads the SessionStart stdin JSON, extracts `session_id`, and writes it to
 * `$CLAUDE_PROJECT_DIR/.mint/sessions/.current-session-id` so the mint CLI can
 * use the same id Claude Code hooks receive (instead of a self-generated one).
 *
 * Non-blocking: any error is swallowed and the hook always exits 0.
 * The file is a bare single line (no JSON wrapper) for cheap reads from shell
 * and node alike.
 */
'use strict';

const fs = require('fs');
const path = require('path');

const MAX_STDIN = 1024 * 1024;
let data = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => {
  if (data.length < MAX_STDIN) {
    data += chunk.substring(0, MAX_STDIN - data.length);
    if (data.length >= MAX_STDIN) process.stdin.destroy();
  }
});

// Strict allowlist for session ids — defense-in-depth against path traversal.
// Claude Code session ids are alphanumeric/dash/underscore; anything else is rejected.
function isValidSessionId(id) {
  return typeof id === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(id);
}

process.stdin.on('end', () => {
  try {
    const projectDir = process.env.CLAUDE_PROJECT_DIR;
    if (!projectDir) return;

    const input = data.trim() ? JSON.parse(data) : {};
    const sessionId = input.session_id;
    if (!sessionId || typeof sessionId !== 'string') return;
    if (!isValidSessionId(sessionId)) return;

    const sessionsDir = path.join(projectDir, '.mint', 'sessions');
    fs.mkdirSync(sessionsDir, { recursive: true });
    fs.writeFileSync(path.join(sessionsDir, '.current-session-id'), sessionId + '\n');
  } catch { /* non-blocking — never fail SessionStart */ }
  process.exit(0);
});
