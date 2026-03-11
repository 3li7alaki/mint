import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

const TMP = path.join(import.meta.dir, '.tmp-init');
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

function readConfig() {
  return JSON.parse(fs.readFileSync(path.join(TMP, '.mint', 'config.json'), 'utf8'));
}

describe('mint init --yes', () => {
  test('creates .mint directory', () => {
    run('init --yes');
    expect(fs.existsSync(path.join(TMP, '.mint'))).toBe(true);
  });

  test('creates config.json', () => {
    run('init --yes');
    const config = readConfig();
    expect(config).toBeDefined();
    expect(config.stack).toBe('none');
    expect(config.packageManager).toBe('none');
  });

  test('creates hard-blocks.md', () => {
    run('init --yes');
    expect(fs.existsSync(path.join(TMP, '.mint', 'hard-blocks.md'))).toBe(true);
  });

  test('creates issues.md', () => {
    run('init --yes');
    expect(fs.existsSync(path.join(TMP, '.mint', 'issues.md'))).toBe(true);
  });

  test('creates wins.md', () => {
    run('init --yes');
    expect(fs.existsSync(path.join(TMP, '.mint', 'wins.md'))).toBe(true);
  });

  test('browser enabled by default', () => {
    run('init --yes');
    const config = readConfig();
    expect(config.browser.enabled).toBe(true);
  });

  test('browser can be disabled', () => {
    run('init --yes --browser false');
    const config = readConfig();
    expect(config.browser.enabled).toBe(false);
  });

  test('autoCommit defaults to true', () => {
    run('init --yes');
    const config = readConfig();
    expect(config.autoCommit).toBe(true);
  });

  test('tdd defaults to false', () => {
    run('init --yes');
    const config = readConfig();
    expect(config.tdd.default).toBe(false);
  });

  test('tdd can be enabled', () => {
    run('init --yes --tdd true');
    const config = readConfig();
    expect(config.tdd.default).toBe(true);
  });

  test('isolation defaults to none', () => {
    run('init --yes');
    const config = readConfig();
    expect(config.isolation.plan).toBe('none');
  });

  test('isolation can be set', () => {
    run('init --yes --isolation branch');
    const config = readConfig();
    expect(config.isolation.plan).toBe('branch');
    expect(config.isolation.ship).toBe('branch');
    expect(config.isolation.quick).toBe('branch');
  });

  test('plugins can be set', () => {
    run('init --yes --plugins mint-nuxt,mint-e2e');
    const config = readConfig();
    expect(config.plugins).toContain('plugins/mint-nuxt');
    expect(config.plugins).toContain('plugins/mint-e2e');
  });

  test('default reviewers are configured', () => {
    run('init --yes');
    const config = readConfig();
    expect(config.reviewers.spec.enabled).toBe(true);
    expect(config.reviewers.quality.enabled).toBe(true);
    expect(config.reviewers.conventions.enabled).toBe(true);
    expect(config.reviewers.business.enabled).toBe(true);
    expect(config.reviewers.security).toBe(false);
    expect(config.reviewers.tests).toBe(false);
    expect(config.reviewers.performance).toBe(false);
  });

  test('gates default to false in empty dir', () => {
    run('init --yes');
    const config = readConfig();
    expect(config.gates.lint).toBe(false);
    expect(config.gates.types).toBe(false);
    expect(config.gates.tests).toBe(false);
  });

  test('detects stack from nuxt.config', () => {
    fs.writeFileSync(path.join(TMP, 'nuxt.config.ts'), '');
    run('init --yes');
    const config = readConfig();
    expect(config.stack).toBe('nuxt');
  });

  test('detects package manager from bun.lock', () => {
    fs.writeFileSync(path.join(TMP, 'bun.lock'), '');
    run('init --yes');
    const config = readConfig();
    expect(config.packageManager).toBe('bun');
  });

  test('detects gates from tsconfig', () => {
    fs.writeFileSync(path.join(TMP, 'package.json'), '{}');
    fs.writeFileSync(path.join(TMP, 'tsconfig.json'), '{}');
    run('init --yes');
    const config = readConfig();
    expect(config.gates.types).toBeTruthy();
  });

  test('does not overwrite existing hard-blocks.md', () => {
    fs.mkdirSync(path.join(TMP, '.mint'), { recursive: true });
    fs.writeFileSync(path.join(TMP, '.mint', 'hard-blocks.md'), 'custom rules');
    run('init --yes');
    const content = fs.readFileSync(path.join(TMP, '.mint', 'hard-blocks.md'), 'utf8');
    expect(content).toBe('custom rules');
  });

  test('adds mint state files to .gitignore', () => {
    run('init --yes');
    const gitignore = fs.readFileSync(path.join(TMP, '.gitignore'), 'utf8');
    expect(gitignore).toContain('.mint/tasks/');
    expect(gitignore).toContain('.mint/ssh-cache.json');
    // instincts.md should NOT be gitignored — it's learned knowledge
    expect(gitignore).not.toContain('.mint/instincts.md');
  });

  test('does not duplicate gitignore entries on re-init', () => {
    run('init --yes');
    run('init --yes');
    const gitignore = fs.readFileSync(path.join(TMP, '.gitignore'), 'utf8');
    const matches = gitignore.match(/mint local state/g);
    expect(matches.length).toBe(1);
  });

  test('appends to existing .gitignore', () => {
    fs.writeFileSync(path.join(TMP, '.gitignore'), 'node_modules/\n');
    run('init --yes');
    const gitignore = fs.readFileSync(path.join(TMP, '.gitignore'), 'utf8');
    expect(gitignore).toContain('node_modules/');
    expect(gitignore).toContain('.mint/tasks/');
  });

  test('output includes version', () => {
    const output = run('init --yes');
    expect(output).toContain('mint configured');
    expect(output).toMatch(/v\d+\.\d+\.\d+/);
  });
});

describe('stack detection in init', () => {
  test('detects react from package.json', () => {
    fs.writeFileSync(path.join(TMP, 'package.json'), JSON.stringify({
      dependencies: { react: '^18' },
    }));
    run('init --yes');
    expect(readConfig().stack).toBe('react');
  });

  test('detects go', () => {
    fs.writeFileSync(path.join(TMP, 'go.mod'), 'module test');
    run('init --yes');
    expect(readConfig().stack).toBe('go');
  });

  test('stack flag overrides detection', () => {
    fs.writeFileSync(path.join(TMP, 'go.mod'), 'module test');
    run('init --yes --stack python');
    expect(readConfig().stack).toBe('python');
  });
});
