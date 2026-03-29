import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

const TMP = path.join(import.meta.dir, '.tmp-pipeline');
const HOOK = path.join(import.meta.dir, '..', 'hooks', 'scripts', 'pipeline-complete-check.cjs');

function setupProject(structure = {}) {
  // Create .mint directory structure
  fs.mkdirSync(path.join(TMP, '.mint', 'sessions'), { recursive: true });

  if (structure.session) {
    fs.writeFileSync(
      path.join(TMP, '.mint', 'sessions', 'test-session.json'),
      JSON.stringify(structure.session)
    );
  }

  if (structure.execution) {
    const execDir = path.join(TMP, '.mint', 'tasks', 'test-task', '001');
    fs.mkdirSync(execDir, { recursive: true });
    fs.writeFileSync(
      path.join(execDir, 'execution.json'),
      JSON.stringify(structure.execution)
    );
  }

  if (structure.pipelineState) {
    const stateDir = path.join(TMP, '.mint', 'tasks', 'test-task', '001');
    fs.mkdirSync(stateDir, { recursive: true });
    fs.writeFileSync(
      path.join(stateDir, 'pipeline-state.json'),
      JSON.stringify(structure.pipelineState)
    );
  }
}

function runHook() {
  try {
    execSync(`node "${HOOK}"`, { cwd: TMP, stdio: 'pipe' });
    return { exitCode: 0, stderr: '' };
  } catch (err) {
    return { exitCode: err.status, stderr: err.stderr?.toString() || '' };
  }
}

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('pipeline-complete-check hook', () => {
  test('allows stop when no mint project', () => {
    // No .mint directory
    const result = runHook();
    expect(result.exitCode).toBe(0);
  });

  test('allows stop when no active session', () => {
    setupProject({});
    const result = runHook();
    expect(result.exitCode).toBe(0);
  });

  test('allows stop in quick mode', () => {
    setupProject({
      session: { mintInvoked: true, mode: 'quick', task: 'fix typo' },
    });
    const result = runHook();
    expect(result.exitCode).toBe(0);
  });

  test('allows stop when plan mode spec is passed', () => {
    setupProject({
      session: { mintInvoked: true, mode: 'plan', task: 'add feature' },
      execution: {
        status: 'passed',
        gates: { lint: 'pass', types: 'pass', tests: 'pass' },
        reviews: { spec: 'passed', quality: 'passed' },
      },
    });
    const result = runHook();
    expect(result.exitCode).toBe(0);
  });

  test('blocks stop when gates ran but no reviews', () => {
    setupProject({
      session: { mintInvoked: true, mode: 'plan', task: 'add feature' },
      execution: {
        status: 'running',
        gates: { lint: 'pass', types: 'pass', tests: 'pass' },
        reviews: {},
      },
    });
    const result = runHook();
    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain('pipeline incomplete');
    expect(result.stderr).toContain('no reviews');
  });

  test('blocks stop when pipeline-state has pending steps', () => {
    setupProject({
      session: { mintInvoked: true, mode: 'plan', task: 'add feature' },
      execution: { status: 'running', gates: {}, reviews: {} },
      pipelineState: {
        currentStep: 'review-stage1',
        completedSteps: ['implement'],
        pendingSteps: ['review-stage1', 'review-stage2', 'docs', 'dod'],
        specId: '001',
      },
    });
    const result = runHook();
    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain('4 pipeline steps remaining');
  });

  test('allows stop when spec is failed', () => {
    setupProject({
      session: { mintInvoked: true, mode: 'plan', task: 'add feature' },
      execution: {
        status: 'failed',
        gates: { lint: 'pass', types: 'fail' },
        reviews: {},
      },
    });
    const result = runHook();
    expect(result.exitCode).toBe(0);
  });
});
