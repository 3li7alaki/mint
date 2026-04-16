#!/usr/bin/env node
/**
 * mint hook: Pipeline completion check
 * Trigger: Stop
 *
 * Blocks Claude from stopping if THIS Claude session has incomplete pipeline steps.
 * Exit code 0 = allow stop. Exit code 2 = block stop (pipeline incomplete).
 *
 * Stop hook stdin: { session_id, transcript_path, last_assistant_message, cwd, ... }
 *
 * Scoping rules:
 * 1. Read session_id from stdin JSON. If missing/empty → exit 0 (safety).
 * 2. Open `.mint/sessions/<session_id>.json`. If missing OR `mintInvoked !== true` → exit 0.
 * 3. If `mode !== 'plan'` → exit 0 (quick mode has no pipeline-state).
 * 4. Scan ONLY `.mint/tasks/<session_id>/**` — never global tasks.
 * 5. Flag specs with execution.json gates but no reviews.
 * 6. Flag pipeline-state.json with pendingSteps.
 *
 * All FS/parse errors are swallowed and default to exit 0 (safety default — never block on error).
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

// Find project root (walk up looking for .mint/)
function findProjectRoot() {
  let dir = process.cwd();
  while (dir !== path.dirname(dir)) {
    if (fs.existsSync(path.join(dir, '.mint'))) return dir;
    dir = path.dirname(dir);
  }
  return null;
}

function safeReadJson(p) {
  try {
    return JSON.parse(fs.readFileSync(p, 'utf8'));
  } catch {
    return null;
  }
}

function safeReadDir(p) {
  try {
    return fs.readdirSync(p);
  } catch {
    return [];
  }
}

function safeIsDir(p) {
  try {
    return fs.statSync(p).isDirectory();
  } catch {
    return false;
  }
}

function collectIssues(sessionTasksDir) {
  const issues = [];

  const featureSlugs = safeReadDir(sessionTasksDir).filter(d =>
    safeIsDir(path.join(sessionTasksDir, d))
  );

  for (const slug of featureSlugs) {
    const featureDir = path.join(sessionTasksDir, slug);

    const specDirs = safeReadDir(featureDir).filter(d => {
      const fullPath = path.join(featureDir, d);
      return safeIsDir(fullPath) && fs.existsSync(path.join(fullPath, 'execution.json'));
    });

    for (const specDir of specDirs) {
      const execPath = path.join(featureDir, specDir, 'execution.json');
      const exec = safeReadJson(execPath);
      if (!exec) continue;

      // Skip completed specs
      if (exec.status === 'passed' || exec.status === 'failed') continue;

      // Check for missing reviews (the most common skipped step)
      const gates = exec.gates || {};
      const reviews = exec.reviews || {};
      const gateTier = gates.tier || 'full';
      const hasGates = Object.keys(gates).length > 1 || gateTier === 'skip'; // tier field counts
      const hasReviews = Object.keys(reviews).length > 0;

      if (hasGates && !hasReviews) {
        issues.push(`${slug}/${specDir}: gates ran but no reviews — pipeline incomplete`);
      }

      // Check pipeline-state.json if it exists
      const pipelinePath = path.join(featureDir, specDir, 'pipeline-state.json');
      if (fs.existsSync(pipelinePath)) {
        const pipeline = safeReadJson(pipelinePath);
        if (pipeline) {
          const pending = pipeline.pendingSteps || [];
          if (pending.length > 0) {
            issues.push(`${slug}/${specDir}: ${pending.length} pipeline steps remaining (${pending.join(', ')})`);
          }
        }
      }
    }
  }

  return issues;
}

process.stdin.on('end', () => {
  try {
    // 1. Parse stdin and extract session_id
    const input = data.trim() ? JSON.parse(data) : {};
    const sessionId = input.session_id;
    if (!sessionId || typeof sessionId !== 'string') process.exit(0);
    if (!isValidSessionId(sessionId)) process.exit(0);

    // 2. Resolve project root
    const projectRoot = findProjectRoot();
    if (!projectRoot) process.exit(0);

    const mintDir = path.join(projectRoot, '.mint');
    const sessionPath = path.join(mintDir, 'sessions', `${sessionId}.json`);

    // 3. Open session file. Missing or not invoked → exit 0
    const session = safeReadJson(sessionPath);
    if (!session || session.mintInvoked !== true) process.exit(0);

    // 4. Quick mode has no pipeline-state — allow stop
    if (session.mode !== 'plan') process.exit(0);

    // 5. Scan ONLY this session's task tree
    const sessionTasksDir = path.join(mintDir, 'tasks', sessionId);
    if (!fs.existsSync(sessionTasksDir)) process.exit(0);

    const issues = collectIssues(sessionTasksDir);

    if (issues.length > 0) {
      console.error(`\n[mint] Pipeline incomplete — ${issues.length} issue(s):\n`);
      for (const issue of issues) {
        console.error(`  - ${issue}`);
      }
      console.error('\nComplete the pipeline steps or delete the session to stop.');
      console.error(`To force stop: delete .mint/sessions/${sessionId}.json\n`);
      process.exit(2); // Block stop
    }

    process.exit(0); // All clear
  } catch (err) {
    // Never throw — safety default is to allow stop
    try { console.error(`[mint] pipeline-complete-check warning: ${err && err.message ? err.message : err}`); } catch { /* noop */ }
    process.exit(0);
  }
});
