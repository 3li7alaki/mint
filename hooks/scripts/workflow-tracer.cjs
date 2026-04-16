#!/usr/bin/env node
/**
 * mint hook: Workflow tracer
 * Trigger: Stop
 *
 * Captures structured workflow traces from session data and git history,
 * appending to .mint/workflows.jsonl for pattern mining.
 *
 * Non-blocking: always exits 0, always passes stdin through to stdout.
 */
'use strict';

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const MAX_STDIN = 1024 * 1024;
let data = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => {
  if (data.length < MAX_STDIN) data += chunk.substring(0, MAX_STDIN - data.length);
});

// --- Utility functions (exported for testing) ---

function findProjectRoot() {
  let dir = process.cwd();
  while (dir !== path.dirname(dir)) {
    if (fs.existsSync(path.join(dir, '.mint'))) return dir;
    dir = path.dirname(dir);
  }
  return null;
}

/**
 * Find the most recent active session (mintInvoked=true).
 * Returns { session, sessionId } or null.
 */
function findActiveSession(sessionsDir) {
  try {
    const files = fs.readdirSync(sessionsDir)
      .filter(f => f.endsWith('.json'))
      .sort()
      .reverse();

    for (const f of files) {
      try {
        const s = JSON.parse(fs.readFileSync(path.join(sessionsDir, f), 'utf8'));
        if (s.mintInvoked) {
          return { session: s, sessionId: f.replace('.json', '') };
        }
      } catch { /* skip invalid */ }
    }
  } catch { /* no sessions dir */ }
  return null;
}

/**
 * Parse git log output lines into operation objects.
 * Each line is a commit subject from --format="%s".
 */
function parseGitOperations(logOutput) {
  if (!logOutput || !logOutput.trim()) return [];

  const lines = logOutput.trim().split('\n').filter(l => l.trim());
  const ops = [];

  for (const line of lines) {
    const trimmed = line.trim();

    // Detect common commit message patterns
    if (/^(feat|fix|chore|refactor|docs|test|style|perf|ci|build)(\(.+\))?:/.test(trimmed)) {
      // Conventional commit — this is a code commit
      ops.push({ type: 'commit', message: trimmed });
    } else if (/^Merge\b/i.test(trimmed)) {
      ops.push({ type: 'git', action: 'merge', message: trimmed });
    } else if (/cherry.?pick/i.test(trimmed)) {
      ops.push({ type: 'git', action: 'cherry-pick', message: trimmed });
    } else if (/revert/i.test(trimmed)) {
      ops.push({ type: 'git', action: 'revert', message: trimmed });
    } else {
      // Generic commit
      ops.push({ type: 'commit', message: trimmed });
    }
  }

  return ops;
}

/**
 * Scan execution state from `.mint/tasks/<session-id>/` for gate results.
 *
 * `tasksDir` must already be scoped to a single session's namespace
 * (i.e. `.mint/tasks/<session-id>/`). The caller is responsible for
 * resolving the current session id and passing the scoped path in so
 * concurrent conversations never see each other's execution state.
 */
function extractExecutionOps(tasksDir) {
  const ops = [];
  try {
    const slugs = fs.readdirSync(tasksDir).filter(d => {
      try { return fs.statSync(path.join(tasksDir, d)).isDirectory(); } catch { return false; }
    });

    for (const slug of slugs) {
      const specDirs = fs.readdirSync(path.join(tasksDir, slug)).filter(d => {
        const fullPath = path.join(tasksDir, slug, d);
        try { return fs.statSync(fullPath).isDirectory(); } catch { return false; }
      });

      for (const specDir of specDirs) {
        const execPath = path.join(tasksDir, slug, specDir, 'execution.json');
        try {
          const exec = JSON.parse(fs.readFileSync(execPath, 'utf8'));
          const gates = exec.gates || {};

          // Only include if gates have been run
          const gateKeys = Object.keys(gates).filter(k => k !== 'tier');
          if (gateKeys.length > 0) {
            const allPass = gateKeys.every(k => gates[k] === 'pass' || gates[k] === true);
            ops.push({ type: 'gate', result: allPass ? 'pass' : 'fail' });
          }

          // Include edits if files are tracked
          if (exec.filesModified && exec.filesModified.length > 0) {
            ops.push({ type: 'edit', files: exec.filesModified });
          }
        } catch { /* skip invalid */ }
      }
    }
  } catch { /* no tasks dir */ }
  return ops;
}

/**
 * Compute fingerprint from operations (colon-separated operation types).
 * Inlined from cli/lib/jsonl.js since this is CJS and cannot import ESM.
 */
function computeFingerprint(operations) {
  if (!operations || operations.length === 0) return '';
  return operations
    .map(op => op.action ? `${op.type}-${op.action}` : op.type)
    .join(':');
}

/**
 * Build the full workflow trace for this session.
 */
function buildTrace(projectRoot, sessionData, sessionId) {
  const { session } = sessionData;
  const mintDir = path.join(projectRoot, '.mint');
  // Scope execution-state scanning to the current session's namespace so
  // concurrent conversations don't get their gate results cross-pollinated.
  const sessionTasksDir = path.join(mintDir, 'tasks', sessionId);

  // Determine session start time
  const startTime = session.invokedAt || session.startedAt || session.createdAt;
  const now = new Date();
  const durationSec = startTime
    ? Math.round((now.getTime() - new Date(startTime).getTime()) / 1000)
    : 0;

  // Collect operations from git log
  let gitOps = [];
  if (startTime) {
    try {
      const logOutput = execSync(
        `git log --after="${startTime}" --oneline --format="%s"`,
        { cwd: projectRoot, stdio: ['pipe', 'pipe', 'pipe'], timeout: 5000 }
      ).toString();
      gitOps = parseGitOperations(logOutput);
    } catch { /* git log failed — proceed without */ }
  }

  // Collect operations from execution state (scoped to this session)
  let execOps = [];
  if (fs.existsSync(sessionTasksDir)) {
    execOps = extractExecutionOps(sessionTasksDir);
  }

  // Merge operations: exec ops (edit, gate) come before git ops (commit)
  const operations = [...execOps, ...gitOps];

  // Determine outcome
  const hasFailure = operations.some(op =>
    (op.type === 'gate' && op.result === 'fail')
  );
  const outcome = hasFailure ? 'failure' : 'success';

  return {
    date: now.toISOString(),
    sessionId,
    task: session.task || '',
    mode: session.mode || 'unknown',
    duration: durationSec,
    operations,
    outcome,
    fingerprint: computeFingerprint(operations),
  };
}

// --- Main ---

function main() {
  try {
    const projectRoot = findProjectRoot();
    if (!projectRoot) return;

    const mintDir = path.join(projectRoot, '.mint');

    // Check automation.enabled in config
    try {
      const config = JSON.parse(fs.readFileSync(path.join(mintDir, 'config.json'), 'utf8'));
      if (config.automation && config.automation.enabled === false) return;
    } catch { return; /* no config = skip */ }

    // Find active session
    const sessionsDir = path.join(mintDir, 'sessions');
    const sessionData = findActiveSession(sessionsDir);
    if (!sessionData) return;

    // Build and append trace
    const trace = buildTrace(projectRoot, sessionData, sessionData.sessionId);
    const workflowsPath = path.join(mintDir, 'workflows.jsonl');
    fs.appendFileSync(workflowsPath, JSON.stringify(trace) + '\n');
  } catch { /* non-blocking — never fail */ }
}

process.stdin.on('end', () => {
  main();
  process.stdout.write(data);
});

// Export for testing
module.exports = {
  findProjectRoot,
  findActiveSession,
  parseGitOperations,
  extractExecutionOps,
  computeFingerprint,
  buildTrace,
};
