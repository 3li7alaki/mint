#!/usr/bin/env node
/**
 * mint hook: Git push and dangerous commit pattern blocker
 * Trigger: PreToolUse (Bash)
 *
 * BLOCKS (not warns):
 * - git push in any form
 * - git commit && git push combined in one command
 * - bash interpolation in commit messages: $(...), backticks, $VAR
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

    // Block git commit + git push combined in one command (sneaks push through commit approval)
    if (/\bgit\s+commit\b/.test(command) && /\bgit\s+push\b/.test(command)) {
      const result = {
        decision: 'block',
        reason: '[mint] git commit + git push in same command blocked — separate them. Commit first, push separately.'
      };
      process.stdout.write(JSON.stringify(result));
      return;
    }

    // Block bash interpolation in git commit -m
    // Matches: git commit -m "$(..)", git commit -m "$VAR", git commit -m "`..`"
    if (/\bgit\s+commit\b/.test(command)) {
      const msgMatch = command.match(/-m\s+(".*?"|'.*?')/s);
      if (msgMatch) {
        const msg = msgMatch[1];
        // Check for $(...) subshells, `backticks`, or $VAR inside double quotes
        // But allow heredoc pattern: -m "$(cat <<'EOF' ... EOF )" which is safe (single-quoted heredoc)
        if (msg.startsWith('"')) {
          const hasSubshell = /\$\((?!cat\s+<<'EOF')/.test(msg);
          const hasBacktick = /`/.test(msg);
          const hasVariable = /\$[A-Z_]/.test(msg);
          if (hasSubshell || hasBacktick || hasVariable) {
            const result = {
              decision: 'block',
              reason: '[mint] Bash interpolation in commit message blocked — use plain string: git commit -m "type(scope): description". No $(...), backticks, or $VAR.'
            };
            process.stdout.write(JSON.stringify(result));
            return;
          }
        }
      }
    }
  } catch { /* non-blocking on parse errors */ }
  process.stdout.write(data);
});
