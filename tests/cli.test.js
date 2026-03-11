import { describe, test, expect } from 'bun:test';
import { execSync } from 'child_process';
import path from 'path';

const CLI = path.join(import.meta.dir, '..', 'cli', 'mint.js');

function run(args) {
  return execSync(`bun "${CLI}" ${args}`, {
    encoding: 'utf8',
    env: { ...process.env, NO_COLOR: '1' },
  });
}

describe('mint CLI entry point', () => {
  test('--version prints version', () => {
    const output = run('--version');
    expect(output.trim()).toMatch(/^mint v\d+\.\d+\.\d+$/);
  });

  test('-v prints version', () => {
    const output = run('-v');
    expect(output.trim()).toMatch(/^mint v\d+\.\d+\.\d+$/);
  });

  test('help shows commands', () => {
    const output = run('help');
    expect(output).toContain('mint init');
    expect(output).toContain('mint config');
    expect(output).toContain('mint doctor');
    expect(output).toContain('mint update');
  });

  test('--help shows commands', () => {
    const output = run('--help');
    expect(output).toContain('mint init');
  });

  test('no args shows help', () => {
    const output = run('');
    expect(output).toContain('Commands');
  });

  test('unknown command exits with error', () => {
    try {
      run('foobar');
      expect(false).toBe(true); // should not reach
    } catch (e) {
      expect(e.status).toBe(1);
    }
  });

  test('version matches package.json', () => {
    const pkg = require('../package.json');
    const output = run('--version').trim();
    expect(output).toBe(`mint v${pkg.version}`);
  });
});
