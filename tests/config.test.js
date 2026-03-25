import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

const TMP = path.join(import.meta.dir, '.tmp-config');
const CLI = path.join(import.meta.dir, '..', 'cli', 'mint.js');

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
  // Create a base config via init
  execSync(`bun "${CLI}" init --yes`, { cwd: TMP, encoding: 'utf8' });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

function run(args) {
  return execSync(`bun "${CLI}" ${args}`, {
    cwd: TMP,
    encoding: 'utf8',
    env: { ...process.env, NO_COLOR: '1' },
  });
}

function readConfig() {
  return JSON.parse(fs.readFileSync(path.join(TMP, '.mint', 'config.json'), 'utf8'));
}

describe('mint config show', () => {
  test('displays config without errors', () => {
    const output = run('config');
    expect(output).toContain('mint config');
    expect(output).toContain('Stack');
    expect(output).toContain('PM');
    expect(output).toContain('Browser');
    expect(output).toContain('TDD');
  });

  test('shows show subcommand', () => {
    const output = run('config show');
    expect(output).toContain('mint config');
  });
});

describe('mint config set', () => {
  test('sets a top-level string value', () => {
    run('config set stack react');
    expect(readConfig().stack).toBe('react');
  });

  test('sets a boolean true', () => {
    run('config set autoCommit true');
    expect(readConfig().autoCommit).toBe(true);
  });

  test('sets a boolean false', () => {
    run('config set autoCommit false');
    expect(readConfig().autoCommit).toBe(false);
  });

  test('sets nested value with dot notation', () => {
    run('config set isolation.plan worktree');
    expect(readConfig().isolation.plan).toBe('worktree');
  });

  test('sets deeply nested value', () => {
    run('config set tdd.coverageThreshold 90');
    expect(readConfig().tdd.coverageThreshold).toBe(90);
  });

  test('sets browser.enabled', () => {
    run('config set browser.enabled false');
    expect(readConfig().browser.enabled).toBe(false);
  });

  test('rejects unknown config keys', () => {
    const output = run('config set newSection.newKey hello');
    expect(output).toContain('Unknown key');
  });
});

describe('mint config errors', () => {
  test('set without args shows usage', () => {
    const output = run('config set');
    expect(output).toContain('Usage');
  });

  test('unknown subcommand shows error', () => {
    const output = run('config bogus');
    expect(output).toContain('Unknown');
  });
});
