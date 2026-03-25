import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

const TMP = path.join(import.meta.dir, '.tmp-doctor');
const CLI = path.join(import.meta.dir, '..', 'cli', 'mint.js');

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

function run(args = '') {
  return execSync(`bun "${CLI}" ${args}`, {
    cwd: TMP,
    encoding: 'utf8',
    env: { ...process.env, NO_COLOR: '1', HOME: TMP },
  });
}

describe('mint doctor', () => {
  test('reports missing config as critical', () => {
    const output = run('doctor');
    expect(output).toContain('missing');
    expect(output).toContain('critical');
  });

  test('no critical issues after init with git', () => {
    execSync('git init', { cwd: TMP, stdio: 'pipe' });
    run('init --yes');
    const output = run('doctor --quick');
    expect(output).not.toContain('CRITICAL');
  });

  test('reports not a git repo as critical', () => {
    run('init --yes');
    const output = run('doctor --quick');
    expect(output).toContain('Not a git repository');
  });

  test('does not report browser warning when disabled', () => {
    execSync('git init', { cwd: TMP, stdio: 'pipe' });
    run('init --yes --browser false');
    const output = run('doctor --quick');
    expect(output).not.toContain('PinchTab');
  });

  test('shows tiered output with summary', () => {
    run('init --yes');
    const output = run('doctor --quick');
    // Should have some output categories
    expect(output).toMatch(/critical|warning|info|healthy/i);
  });

  test('quick mode skips gate execution', () => {
    execSync('git init', { cwd: TMP, stdio: 'pipe' });
    run('init --yes');
    // Quick should be fast — no gate timeouts
    const start = Date.now();
    run('doctor --quick');
    const elapsed = Date.now() - start;
    expect(elapsed).toBeLessThan(5000); // Should be < 5s without gates
  });

  test('--fix repairs missing learning files', () => {
    execSync('git init', { cwd: TMP, stdio: 'pipe' });
    run('init --yes');
    // Delete a learning file
    fs.unlinkSync(path.join(TMP, '.mint', 'issues.jsonl'));
    const output = run('doctor --fix --quick');
    expect(output).toContain('fixed');
    // File should be restored
    expect(fs.existsSync(path.join(TMP, '.mint', 'issues.jsonl'))).toBe(true);
  });
});
