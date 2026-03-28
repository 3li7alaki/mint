#!/usr/bin/env node
/**
 * mint hook: Edit/Write gatekeeper
 * Trigger: PreToolUse (Edit|Write)
 *
 * Three checks in order:
 * 1. Freeze/guard — BLOCKS writes to frozen/guarded paths
 * 2. Scope enforcement — BLOCKS writes outside current spec's <can-modify> (future)
 * 3. Mint invocation — WARNS if mint wasn't invoked before file modifications
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

    const projectRoot = path.dirname(mintDir);

    // ── Check 1: Freeze/Guard ─────────────────────────────────────────────
    const freezeListPath = path.join(mintDir, '.freeze-list.json');
    try {
      const freezeData = JSON.parse(fs.readFileSync(freezeListPath, 'utf8'));
      const entries = freezeData.entries || [];

      for (const entry of entries) {
        const frozenPath = entry.path;
        const reason = entry.reason || null;
        const type = entry.type || 'freeze'; // 'freeze' or 'guard'

        // Resolve the frozen path relative to project root if not absolute
        const resolvedFrozen = path.isAbsolute(frozenPath)
          ? frozenPath
          : path.join(projectRoot, frozenPath);

        // Check if the file being edited falls under a frozen path
        const normalizedFile = path.resolve(filePath);
        const normalizedFrozen = path.resolve(resolvedFrozen);

        // Match: exact file, or file is inside a frozen directory
        const isMatch = normalizedFile === normalizedFrozen
          || normalizedFile.startsWith(normalizedFrozen + path.sep);

        // Glob matching: if frozen path contains * or **, use minimatch-style check
        if (!isMatch && (frozenPath.includes('*'))) {
          const globMatch = matchGlob(frozenPath, filePath, projectRoot);
          if (globMatch) {
            const label = type === 'guard' ? 'GUARDED' : 'FROZEN';
            const msg = reason
              ? `[mint] ${label}: ${filePath} — ${reason}`
              : `[mint] ${label}: ${filePath} is ${type === 'guard' ? 'guarded' : 'frozen'}. Use /unfreeze to remove.`;
            const result = { decision: 'block', reason: msg };
            process.stdout.write(JSON.stringify(result));
            return;
          }
        }

        if (isMatch) {
          const label = type === 'guard' ? 'GUARDED' : 'FROZEN';
          const msg = reason
            ? `[mint] ${label}: ${filePath} — ${reason}`
            : `[mint] ${label}: ${filePath} is ${type === 'guard' ? 'guarded' : 'frozen'}. Use /unfreeze to remove.`;
          const result = { decision: 'block', reason: msg };
          process.stdout.write(JSON.stringify(result));
          return;
        }
      }
    } catch {
      // No freeze list or invalid JSON — no freezes active, continue
    }

    // ── Check 2: Spec scope enforcement ─────────────────────────────────
    // If a spec is being executed, block writes outside its <can-modify>
    // Session isolation: each session has its own state file in .mint/sessions/
    let mintInvoked = false;
    let sessionState = null;

    // Scan session files for any active session (most recent first by filename — IDs are timestamp-prefixed)
    const sessionsDir = path.join(mintDir, 'sessions');
    try {
      const files = fs.readdirSync(sessionsDir).filter(f => f.endsWith('.json')).sort().reverse();
      for (const f of files) {
        try {
          const s = JSON.parse(fs.readFileSync(path.join(sessionsDir, f), 'utf8'));
          if (s.mintInvoked) {
            sessionState = s;
            mintInvoked = true;
            break;
          }
        } catch { /* skip invalid */ }
      }
    } catch { /* no sessions dir yet */ }

    if (sessionState && sessionState.activeSpec) {
      const specPath = sessionState.activeSpec;
      try {
        const specXml = fs.readFileSync(path.join(projectRoot, specPath), 'utf8');
        const canModifyMatch = specXml.match(/<can-modify>([\s\S]*?)<\/can-modify>/);
        if (canModifyMatch) {
          const allowedPaths = canModifyMatch[1]
            .split(',')
            .map(p => p.trim())
            .filter(Boolean);

          const relFile = path.relative(projectRoot, path.resolve(filePath));
          const isAllowed = allowedPaths.some(allowed => {
            if (allowed.includes('*')) {
              return matchGlob(allowed, filePath, projectRoot);
            }
            // Exact file match or directory prefix
            const normalizedAllowed = path.normalize(allowed);
            return relFile === normalizedAllowed
              || relFile.startsWith(normalizedAllowed + path.sep);
          });

          if (!isAllowed) {
            const result = {
              decision: 'block',
              reason: `[mint] SCOPE VIOLATION: ${relFile} is outside spec scope. Allowed: ${allowedPaths.join(', ')}. Request scope expansion or find an alternative approach.`
            };
            process.stdout.write(JSON.stringify(result));
            return;
          }
        }
      } catch {
        // Can't read spec — don't block, just continue
      }
    }

    // ── Check 3: Mint invocation warning ──────────────────────────────────
    if (!mintInvoked) {
      console.error(
        '[mint] File modification without mint invocation detected. ' +
        'Invoke the mint skill first for quality gates, reviews, and disciplined execution. ' +
        'Use: Skill tool → "mint" with task description.'
      );
    }
  } catch { /* non-blocking on parse errors */ }
  process.stdout.write(data);
});

/**
 * Simple glob matcher for * and ** patterns.
 * Supports: src/*.ts, src/**\/*.test.ts, *.md
 */
function matchGlob(pattern, filePath, projectRoot) {
  const relFile = path.relative(projectRoot, path.resolve(filePath));
  // Convert glob to regex
  let regex = pattern
    .replace(/[.+^${}()|[\]\\]/g, '\\$&')  // Escape regex special chars except * and ?
    .replace(/\*\*/g, '{{DOUBLESTAR}}')     // Placeholder for **
    .replace(/\*/g, '[^/]*')                 // * matches anything except /
    .replace(/{{DOUBLESTAR}}/g, '.*')        // ** matches anything including /
    .replace(/\?/g, '[^/]');                 // ? matches single char except /
  try {
    return new RegExp('^' + regex + '$').test(relFile);
  } catch {
    return false;
  }
}
