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
    env: { ...process.env, NO_COLOR: '1' },
  });
}

describe('mint doctor', () => {
  test('reports missing config', () => {
    const output = run('doctor');
    expect(output).toContain('missing');
  });

  test('reports valid config after init', () => {
    run('init --yes');
    const output = run('doctor');
    expect(output).toContain('config.json valid');
  });

  test('reports hard-blocks present', () => {
    run('init --yes');
    const output = run('doctor');
    expect(output).toContain('hard-blocks.md present');
  });

  test('reports browser enabled', () => {
    run('init --yes');
    const output = run('doctor');
    expect(output).toContain('Browser: enabled');
  });

  test('does not report browser when disabled', () => {
    run('init --yes --browser false');
    const output = run('doctor');
    expect(output).not.toContain('Browser: enabled');
  });

  test('reports not a git repository', () => {
    run('init --yes');
    const output = run('doctor');
    expect(output).toContain('Not a git repository');
  });

  test('reports git initialized when .git exists', () => {
    execSync('git init', { cwd: TMP, stdio: 'pipe' });
    run('init --yes');
    const output = run('doctor');
    expect(output).toContain('Git initialized');
  });

  test('reports doc-manifest after init', () => {
    run('init --yes');
    const output = run('doctor');
    expect(output).toContain('Doc-manifest');
  });

  test('shows summary counts', () => {
    run('init --yes');
    const output = run('doctor');
    expect(output).toMatch(/\d+ passed/);
  });
});
