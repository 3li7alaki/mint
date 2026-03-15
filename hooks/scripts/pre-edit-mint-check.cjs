#!/usr/bin/env node
/**
 * mint hook: Auto-invocation enforcement
 * Trigger: PreToolUse (Edit|Write)
 * Checks if mint has been invoked before file modifications.
 * Emits a warning if mint wasn't invoked — non-blocking.
 */
'use strict';

const fs = require('fs');
const path = require('path');

const MAX_STDIN = 1024 * 1024;
let data = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => {
  if (data.length < MAX_STDIN) data += chunk.substring(0, MAX_STDIN - data.length);
});

process.stdin.on('end', () => {
  try {
    const input = JSON.parse(data);
    const filePath = String(input.tool_input?.file_path || '');

    // Skip non-project files (hooks editing themselves, home dir configs, etc.)
    if (!filePath || filePath.includes('/.claude/') || filePath.includes('/node_modules/')) {
      process.stdout.write(data);
      return;
    }

    // Find .mint directory by walking up from the file being edited
    let dir = path.dirname(filePath);
    let mintDir = null;
    for (let i = 0; i < 10; i++) {
      const candidate = path.join(dir, '.mint');
      if (fs.existsSync(candidate) && fs.statSync(candidate).isDirectory()) {
        mintDir = candidate;
        break;
      }
      const parent = path.dirname(dir);
      if (parent === dir) break;
      dir = parent;
    }

    if (!mintDir) {
      // No .mint directory — not a mint project, skip silently
      process.stdout.write(data);
      return;
    }

    const sessionStatePath = path.join(mintDir, '.session-state.json');

    // Check if session state exists and mint was invoked
    let mintInvoked = false;
    try {
      const state = JSON.parse(fs.readFileSync(sessionStatePath, 'utf8'));
      mintInvoked = state.mintInvoked === true;
    } catch {
      // No session state file — mint hasn't been invoked
    }

    if (!mintInvoked) {
      console.error(
        '[mint] File modification without mint invocation detected. ' +
        'Invoke the mint skill first for quality gates, reviews, and disciplined execution. ' +
        'Use: Skill tool → "mint" with task description.'
      );
    }
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
