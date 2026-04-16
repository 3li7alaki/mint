import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import os from 'os';
import path from 'path';
import { spawn } from 'child_process';

const HOOK = path.join(import.meta.dir, '..', 'hooks', 'scripts', 'pipeline-complete-check.cjs');

let TMP;

beforeEach(() => {
  TMP = fs.mkdtempSync(path.join(os.tmpdir(), 'mint-pipeline-check-'));
  fs.mkdirSync(path.join(TMP, '.mint', 'sessions'), { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

/**
 * Spawn the cjs hook with stdin JSON, return { exitCode, stderr }.
 * Matches the real Stop-hook protocol: writes JSON to stdin then closes it.
 */
function runHook({ stdinJSON = {}, cwd = TMP } = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn('node', [HOOK], {
      cwd,
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    let stderr = '';
    let stdout = '';
    child.stdout.on('data', d => { stdout += d.toString(); });
    child.stderr.on('data', d => { stderr += d.toString(); });

    child.on('error', reject);
    child.on('close', exitCode => resolve({ exitCode, stderr, stdout }));

    child.stdin.write(JSON.stringify(stdinJSON));
    child.stdin.end();
  });
}

/**
 * Write a session file at .mint/sessions/<sessionId>.json with the given state.
 */
function writeSession(sessionId, state) {
  fs.writeFileSync(
    path.join(TMP, '.mint', 'sessions', `${sessionId}.json`),
    JSON.stringify(state),
  );
}

/**
 * Write an execution.json under .mint/tasks/<sessionId>/<slug>/<spec>/.
 */
function writeExecution(sessionId, slug, specDir, payload) {
  const dir = path.join(TMP, '.mint', 'tasks', sessionId, slug, specDir);
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(path.join(dir, 'execution.json'), JSON.stringify(payload));
}

/**
 * Write a pipeline-state.json under .mint/tasks/<sessionId>/<slug>/<spec>/.
 */
function writePipelineState(sessionId, slug, specDir, payload) {
  const dir = path.join(TMP, '.mint', 'tasks', sessionId, slug, specDir);
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(path.join(dir, 'pipeline-state.json'), JSON.stringify(payload));
}

describe('pipeline-complete-check hook — stdin protocol', () => {
  test('missing session_id on stdin → exit 0', async () => {
    const result = await runHook({ stdinJSON: {} });
    expect(result.exitCode).toBe(0);
  });

  test('empty stdin → exit 0', async () => {
    // Spawn with no JSON at all (just close stdin) — must not block.
    const result = await new Promise((resolve, reject) => {
      const child = spawn('node', [HOOK], { cwd: TMP, stdio: ['pipe', 'pipe', 'pipe'] });
      let stderr = '';
      child.stderr.on('data', d => { stderr += d.toString(); });
      child.on('error', reject);
      child.on('close', exitCode => resolve({ exitCode, stderr }));
      child.stdin.end();
    });
    expect(result.exitCode).toBe(0);
  });

  test('non-string session_id → exit 0', async () => {
    const result = await runHook({ stdinJSON: { session_id: 12345 } });
    expect(result.exitCode).toBe(0);
  });

  test('session_id with disallowed characters → exit 0 (rejected)', async () => {
    // Defense-in-depth: strict allowlist check rejects path-traversal payloads.
    const result = await runHook({ stdinJSON: { session_id: '../etc/passwd' } });
    expect(result.exitCode).toBe(0);
  });

  test('unknown session_id (no session file) → exit 0', async () => {
    const result = await runHook({ stdinJSON: { session_id: 'nonexistent-session' } });
    expect(result.exitCode).toBe(0);
  });

  test('no .mint directory → exit 0', async () => {
    const bareDir = fs.mkdtempSync(path.join(os.tmpdir(), 'mint-bare-'));
    try {
      const result = await runHook({ stdinJSON: { session_id: 'whatever' }, cwd: bareDir });
      expect(result.exitCode).toBe(0);
    } finally {
      fs.rmSync(bareDir, { recursive: true, force: true });
    }
  });

  test('session file present but mintInvoked false → exit 0', async () => {
    writeSession('sess-A', { mintInvoked: false, mode: 'plan', task: 'x' });
    const result = await runHook({ stdinJSON: { session_id: 'sess-A' } });
    expect(result.exitCode).toBe(0);
  });

  test('quick mode session → exit 0', async () => {
    writeSession('sess-quick', { mintInvoked: true, mode: 'quick', task: 'fix typo' });
    // Even with execution.json present, quick mode skips pipeline checks.
    writeExecution('sess-quick', 'feat-x', '001', {
      status: 'running',
      gates: { lint: 'pass', types: 'pass', tests: 'pass' },
      reviews: {},
    });
    const result = await runHook({ stdinJSON: { session_id: 'sess-quick' } });
    expect(result.exitCode).toBe(0);
  });

  test('plan mode session with no tasks tree → exit 0', async () => {
    writeSession('sess-empty', { mintInvoked: true, mode: 'plan', task: 'x' });
    const result = await runHook({ stdinJSON: { session_id: 'sess-empty' } });
    expect(result.exitCode).toBe(0);
  });

  test('plan mode session with complete reviews → exit 0', async () => {
    writeSession('sess-done', { mintInvoked: true, mode: 'plan', task: 'add feature' });
    writeExecution('sess-done', 'feature-x', '001', {
      status: 'passed',
      gates: { lint: 'pass', types: 'pass', tests: 'pass' },
      reviews: { spec: 'passed', quality: 'passed' },
    });
    const result = await runHook({ stdinJSON: { session_id: 'sess-done' } });
    expect(result.exitCode).toBe(0);
  });

  test('plan mode session with gates but no reviews → exit 2 with slug in stderr', async () => {
    writeSession('sess-incomplete', { mintInvoked: true, mode: 'plan', task: 'add feature' });
    writeExecution('sess-incomplete', 'my-feature', '001', {
      status: 'running',
      gates: { lint: 'pass', types: 'pass', tests: 'pass' },
      reviews: {},
    });
    const result = await runHook({ stdinJSON: { session_id: 'sess-incomplete' } });
    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain('pipeline incomplete');
    expect(result.stderr).toContain('no reviews');
    expect(result.stderr).toContain('my-feature');
    expect(result.stderr).toContain('001');
  });

  test('plan mode session with pipeline-state pending → exit 2 with count', async () => {
    writeSession('sess-pending', { mintInvoked: true, mode: 'plan', task: 'x' });
    writeExecution('sess-pending', 'big-feat', '001', {
      status: 'running',
      gates: {},
      reviews: {},
    });
    writePipelineState('sess-pending', 'big-feat', '001', {
      currentStep: 'review-stage1',
      completedSteps: ['implement'],
      pendingSteps: ['review-stage1', 'review-stage2', 'docs', 'dod'],
      specId: '001',
    });
    const result = await runHook({ stdinJSON: { session_id: 'sess-pending' } });
    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain('4 pipeline steps remaining');
  });

  test('plan mode session with failed spec → exit 0 (no block on permanent failure)', async () => {
    writeSession('sess-failed', { mintInvoked: true, mode: 'plan', task: 'x' });
    writeExecution('sess-failed', 'feat-x', '001', {
      status: 'failed',
      gates: { lint: 'pass', types: 'fail' },
      reviews: {},
    });
    const result = await runHook({ stdinJSON: { session_id: 'sess-failed' } });
    expect(result.exitCode).toBe(0);
  });
});

describe('pipeline-complete-check hook — multi-session isolation', () => {
  test("session B's stop is unaffected by session A's incomplete work", async () => {
    // Session A: plan mode, has gates-but-no-reviews under its own task tree.
    writeSession('sess-A', { mintInvoked: true, mode: 'plan', task: 'A task' });
    writeExecution('sess-A', 'feat-A', '001', {
      status: 'running',
      gates: { lint: 'pass', types: 'pass', tests: 'pass' },
      reviews: {},
    });

    // Session B: plan mode but no work in its namespace.
    writeSession('sess-B', { mintInvoked: true, mode: 'plan', task: 'B task' });

    // Session B stops — must exit 0; A's incomplete work in a different namespace
    // must NOT block B.
    const resultB = await runHook({ stdinJSON: { session_id: 'sess-B' } });
    expect(resultB.exitCode).toBe(0);

    // Session A stops — must exit 2 with A's slug in stderr.
    const resultA = await runHook({ stdinJSON: { session_id: 'sess-A' } });
    expect(resultA.exitCode).toBe(2);
    expect(resultA.stderr).toContain('feat-A');
    expect(resultA.stderr).toContain('001');
  });

  test('hook never reads global tasks dir — only session-scoped namespace', async () => {
    // Legacy global tasks (no session-id wrapping) at .mint/tasks/<slug>/<spec>/
    // — present from a pre-0.8.5 project. The hook must ignore these.
    const legacyDir = path.join(TMP, '.mint', 'tasks', 'legacy-feature', '001');
    fs.mkdirSync(legacyDir, { recursive: true });
    fs.writeFileSync(path.join(legacyDir, 'execution.json'), JSON.stringify({
      status: 'running',
      gates: { lint: 'pass', types: 'pass', tests: 'pass' },
      reviews: {},
    }));

    writeSession('sess-clean', { mintInvoked: true, mode: 'plan', task: 'fresh' });

    const result = await runHook({ stdinJSON: { session_id: 'sess-clean' } });
    // Must exit 0: session-scoped scan never sees legacy-feature.
    expect(result.exitCode).toBe(0);
  });
});
