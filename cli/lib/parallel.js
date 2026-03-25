/**
 * Parallel spec execution via Claude Code CLI.
 *
 * Spawns isolated Claude Code sessions — one per spec — each in its own
 * git worktree. Collects results, merges back, reports.
 *
 * Uses `claude -p` (headless mode) with JSON output for structured results.
 * Falls back to raw spawn if the Agent SDK is not available.
 */
import { spawn } from 'child_process';
import fs from 'fs';
import path from 'path';
import { createWorktree, removeWorktree, mergeWorktree } from './worktree.js';

/** @typedef {{ specId: string, specPath: string, prompt: string, model?: string, maxTurns?: number, maxBudget?: number }} SpecTask */
/** @typedef {{ specId: string, status: 'passed'|'failed'|'timeout', result?: any, error?: string, worktree: string, duration: number }} SpecResult */

/**
 * Run multiple specs in parallel, each in its own worktree + Claude Code session.
 *
 * @param {string} projectRoot - absolute path to project root
 * @param {SpecTask[]} specs - specs to execute in parallel
 * @param {object} options
 * @param {number} [options.concurrency=3] - max parallel sessions
 * @param {number} [options.timeoutMs=600000] - per-spec timeout (10 min default)
 * @param {string} [options.baseBranch='HEAD'] - branch to base worktrees off
 * @param {function} [options.onProgress] - callback(specId, event, data)
 * @returns {Promise<SpecResult[]>}
 */
export async function runSpecsInParallel(projectRoot, specs, options = {}) {
  const {
    concurrency = 3,
    timeoutMs = 600_000,
    baseBranch = 'HEAD',
    onProgress = () => {},
  } = options;

  const results = [];
  const queue = [...specs];

  // Process specs with concurrency limit
  const running = new Set();

  async function runNext() {
    if (queue.length === 0) return;
    const spec = queue.shift();

    running.add(spec.specId);
    onProgress(spec.specId, 'start', { remaining: queue.length });

    try {
      const result = await runSingleSpec(projectRoot, spec, { timeoutMs, baseBranch, onProgress });
      results.push(result);
      onProgress(spec.specId, 'done', result);
    } catch (err) {
      results.push({
        specId: spec.specId,
        status: 'failed',
        error: err.message,
        worktree: null,
        duration: 0,
      });
      onProgress(spec.specId, 'error', { error: err.message });
    } finally {
      running.delete(spec.specId);
    }
  }

  // Run with concurrency limit using a simple pool
  while (queue.length > 0 || running.size > 0) {
    // Fill up to concurrency limit
    const toStart = Math.min(concurrency - running.size, queue.length);
    const promises = [];
    for (let i = 0; i < toStart; i++) {
      promises.push(runNext());
    }
    if (promises.length > 0) {
      await Promise.race(promises);
    } else {
      // All slots full, wait for any to finish
      await new Promise(resolve => setTimeout(resolve, 500));
    }
  }

  return results;
}

/**
 * Run a single spec in an isolated worktree via `claude -p`.
 *
 * @param {string} projectRoot
 * @param {SpecTask} spec
 * @param {object} options
 * @returns {Promise<SpecResult>}
 */
async function runSingleSpec(projectRoot, spec, options) {
  const { timeoutMs, baseBranch, onProgress } = options;
  const slug = `spec-${spec.specId}`;
  const startTime = Date.now();

  // 1. Create worktree
  const { worktreePath, branch } = createWorktree(projectRoot, slug, { baseBranch });
  onProgress(spec.specId, 'worktree', { path: worktreePath, branch });

  try {
    // 2. Copy .mint config into worktree (specs, config, hard-blocks)
    const mintSrc = path.join(projectRoot, '.mint');
    const mintDest = path.join(worktreePath, '.mint');
    if (!fs.existsSync(mintDest)) fs.mkdirSync(mintDest, { recursive: true });

    // Copy essential mint files
    for (const file of ['config.json', 'hard-blocks.md', 'issues.md', 'wins.md', 'instincts.md']) {
      const src = path.join(mintSrc, file);
      if (fs.existsSync(src)) {
        fs.copyFileSync(src, path.join(mintDest, file));
      }
    }

    // Copy the spec file
    const specSrc = path.join(projectRoot, spec.specPath);
    const specDir = path.join(mintDest, 'tasks', path.dirname(path.relative(path.join(mintSrc, 'tasks'), specSrc)));
    fs.mkdirSync(specDir, { recursive: true });
    fs.copyFileSync(specSrc, path.join(specDir, path.basename(specSrc)));

    // 3. Spawn claude -p in the worktree
    const result = await spawnClaudeSession(worktreePath, spec, timeoutMs);

    // 4. Merge worktree changes back if successful
    if (result.status === 'passed') {
      const mergeResult = mergeWorktree(projectRoot, slug, baseBranch);
      result.merge = mergeResult;
    }

    result.duration = Date.now() - startTime;
    result.worktree = worktreePath;
    return result;

  } finally {
    // 5. Cleanup worktree (keep on failure for debugging)
    // removeWorktree(projectRoot, slug);
    // Note: cleanup is deferred — orchestrator decides when to clean up
  }
}

/**
 * Spawn a headless Claude Code session.
 *
 * @param {string} cwd - working directory (worktree path)
 * @param {SpecTask} spec
 * @param {number} timeoutMs
 * @returns {Promise<SpecResult>}
 */
function spawnClaudeSession(cwd, spec, timeoutMs) {
  return new Promise((resolve, reject) => {
    const args = [
      '-p', spec.prompt,
      '--output-format', 'json',
      '--max-turns', String(spec.maxTurns || 50),
    ];

    if (spec.model) {
      args.push('--model', spec.model);
    }

    if (spec.maxBudget) {
      args.push('--max-budget-usd', String(spec.maxBudget));
    }

    // Allow essential tools
    args.push('--allowedTools', 'Bash,Read,Edit,Write,Glob,Grep');

    const proc = spawn('claude', args, {
      cwd,
      env: { ...process.env },
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    let stdout = '';
    let stderr = '';

    proc.stdout.on('data', (data) => { stdout += data.toString(); });
    proc.stderr.on('data', (data) => { stderr += data.toString(); });

    // Timeout
    const timer = setTimeout(() => {
      proc.kill('SIGTERM');
      resolve({
        specId: spec.specId,
        status: 'timeout',
        error: `Spec timed out after ${timeoutMs / 1000}s`,
        worktree: cwd,
        duration: timeoutMs,
      });
    }, timeoutMs);

    proc.on('close', (code) => {
      clearTimeout(timer);

      try {
        const output = JSON.parse(stdout);
        resolve({
          specId: spec.specId,
          status: code === 0 ? 'passed' : 'failed',
          result: output.result || output.structured_output || null,
          sessionId: output.session_id || null,
          error: code !== 0 ? (stderr || `Exit code ${code}`) : null,
          worktree: cwd,
          duration: 0, // filled by caller
        });
      } catch {
        // JSON parse failed — return raw output
        resolve({
          specId: spec.specId,
          status: code === 0 ? 'passed' : 'failed',
          result: stdout.slice(0, 2000),
          error: code !== 0 ? (stderr.slice(0, 1000) || `Exit code ${code}`) : null,
          worktree: cwd,
          duration: 0,
        });
      }
    });

    proc.on('error', (err) => {
      clearTimeout(timer);
      reject(new Error(`Failed to spawn claude: ${err.message}`));
    });
  });
}

/**
 * Build a prompt for a spec execution session.
 * Reads the spec XML and wraps it with mint planner instructions.
 *
 * @param {string} specPath - path to spec XML file
 * @param {object} options
 * @param {boolean} [options.autoCommit=true]
 * @param {boolean} [options.tdd=false]
 * @returns {string}
 */
export function buildSpecPrompt(specPath, options = {}) {
  const specXml = fs.readFileSync(specPath, 'utf8');
  const { autoCommit = true, tdd = false } = options;

  return `You are a mint planner agent executing a single spec. Follow the spec exactly.

## Spec XML

${specXml}

## Instructions

1. Read the spec completely — understand every field
2. Declare scope: only modify files listed in <can-modify>
3. Implement according to <steps> — follow them exactly
${tdd ? '4. Follow TDD: write tests first (must fail), implement, refactor\n' : ''}
5. Run gates: lint, types, tests (from .mint/config.json)
6. ${autoCommit ? 'If gates pass, commit: git add <files> && git commit -m "<commit message from spec>"' : 'If gates pass, leave changes staged (autocommit is off)'}
7. Report result

## Rules
- NEVER git push
- NEVER use bash interpolation in commit messages
- NEVER modify files outside <can-modify>
- Use plain git commit -m "message" only
`;
}
