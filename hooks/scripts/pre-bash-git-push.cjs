#!/usr/bin/env node
/**
 * mint hook: Git push safety check
 * Trigger: PreToolUse (Bash)
 * Reminds before git push — agents should not push.
 */
'use strict';

const MAX_STDIN = 1024 * 1024;
let data = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => {
  if (data.length < MAX_STDIN) data += chunk.substring(0, MAX_STDIN - data.length);
});

process.stdin.on('end', () => {
  try {
    const input = JSON.parse(data);
    const command = String(input.tool_input?.command || '');
    if (/\bgit\s+push\b/.test(command)) {
      console.error('[mint] ⚠ git push detected — mint agents commit only, humans push. Verify this is intentional.');
    }
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
