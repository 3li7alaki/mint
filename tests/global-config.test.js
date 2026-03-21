import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

const TMP = path.join(import.meta.dir, '.tmp-global');
const FAKE_HOME = path.join(TMP, 'home');
const PROJECT = path.join(TMP, 'project');
const CLI = path.join(import.meta.dir, '..', 'cli', 'mint.js');

beforeEach(() => {
  fs.mkdirSync(FAKE_HOME, { recursive: true });
  fs.mkdirSync(PROJECT, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

function run(args = '', { cwd = PROJECT } = {}) {
  return execSync(`bun "${CLI}" ${args}`, {
    cwd,
    encoding: 'utf8',
    env: { ...process.env, HOME: FAKE_HOME, NO_COLOR: '1' },
  });
}

function readProjectConfig() {
  return JSON.parse(fs.readFileSync(path.join(PROJECT, '.mint', 'config.json'), 'utf8'));
}

function writeGlobalConfig(config) {
  const dir = path.join(FAKE_HOME, '.mint');
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(path.join(dir, 'config.json'), JSON.stringify(config, null, 2));
}

function readGlobalConfig() {
  return JSON.parse(fs.readFileSync(path.join(FAKE_HOME, '.mint', 'config.json'), 'utf8'));
}

// ─── Global config utilities ─────────────────────────────────────────────────

const { mergeConfigs } = await import('../cli/lib/detect.js');

describe('mergeConfigs', () => {

  test('project overrides global', () => {
    const global = { autoCommit: false, tdd: { default: true } };
    const project = { stack: 'react', autoCommit: true, tdd: { default: false } };
    const merged = mergeConfigs(global, project);
    expect(merged.autoCommit).toBe(true);
    expect(merged.tdd.default).toBe(false);
    expect(merged.stack).toBe('react');
  });

  test('global fills in missing project keys', () => {
    const global = { autoCommit: false, isolation: { plan: 'worktree', ship: 'worktree', quick: 'branch' } };
    const project = { stack: 'react' };
    const merged = mergeConfigs(global, project);
    expect(merged.autoCommit).toBe(false);
    expect(merged.isolation.plan).toBe('worktree');
    expect(merged.stack).toBe('react');
  });

  test('returns project config when global is empty', () => {
    const project = { stack: 'react', autoCommit: true };
    const merged = mergeConfigs({}, project);
    expect(merged).toEqual(project);
  });

  test('returns null when project is null', () => {
    const merged = mergeConfigs({ autoCommit: false }, null);
    expect(merged).toBeNull();
  });

  test('deep merges nested objects', () => {
    const global = { reviewers: { spec: { enabled: true, model: 'opus' }, quality: { enabled: true, model: 'sonnet' } } };
    const project = { reviewers: { spec: { enabled: true, model: 'haiku' } } };
    const merged = mergeConfigs(global, project);
    expect(merged.reviewers.spec.model).toBe('haiku');
    expect(merged.reviewers.quality.model).toBe('sonnet');
  });

  test('only merges GLOBAL_KEYS from global config', () => {
    const global = { stack: 'vue', autoCommit: false };
    const project = { stack: 'react' };
    const merged = mergeConfigs(global, project);
    // stack is not a GLOBAL_KEY, so global stack should not leak
    expect(merged.stack).toBe('react');
  });
});

// ─── CLI integration ─────────────────────────────────────────────────────────

describe('mint config --global', () => {
  test('shows empty state when no global config', () => {
    const output = run('config --global');
    expect(output).toContain('No global config found');
  });

  test('set --global creates global config', () => {
    run('config set --global autoCommit false');
    const config = readGlobalConfig();
    expect(config.autoCommit).toBe(false);
  });

  test('set --global with dot notation', () => {
    run('config set --global isolation.plan worktree');
    const config = readGlobalConfig();
    expect(config.isolation.plan).toBe('worktree');
  });

  test('show --global displays set values', () => {
    writeGlobalConfig({ autoCommit: false, isolation: { plan: 'worktree' } });
    const output = run('config --global');
    expect(output).toContain('global');
    expect(output).toContain('Auto-commit');
    expect(output).toContain('Isolation');
  });
});

describe('init seeds from global config', () => {
  test('uses global autoCommit as default', () => {
    writeGlobalConfig({ autoCommit: false });
    run('init --yes');
    const config = readProjectConfig();
    expect(config.autoCommit).toBe(false);
  });

  test('uses global isolation as default', () => {
    writeGlobalConfig({ isolation: { plan: 'worktree', ship: 'worktree', quick: 'branch' } });
    run('init --yes');
    const config = readProjectConfig();
    expect(config.isolation.plan).toBe('worktree');
  });

  test('uses global tdd as default', () => {
    writeGlobalConfig({ tdd: { default: true, desloppify: true, coverageThreshold: 90 } });
    run('init --yes');
    const config = readProjectConfig();
    expect(config.tdd.default).toBe(true);
    expect(config.tdd.coverageThreshold).toBe(90);
  });

  test('uses global reviewer models as defaults', () => {
    writeGlobalConfig({
      reviewers: {
        spec: { enabled: true, model: 'sonnet' },
        quality: { enabled: true, model: 'opus' },
      },
    });
    run('init --yes');
    const config = readProjectConfig();
    expect(config.reviewers.spec.model).toBe('sonnet');
    expect(config.reviewers.quality.model).toBe('opus');
  });

  test('CLI flags override global defaults', () => {
    writeGlobalConfig({ autoCommit: false, isolation: { plan: 'worktree' } });
    run('init --yes --autocommit true --isolation branch');
    const config = readProjectConfig();
    expect(config.autoCommit).toBe(true);
    expect(config.isolation.plan).toBe('branch');
  });

  test('init works fine without global config', () => {
    run('init --yes');
    const config = readProjectConfig();
    expect(config).toBeDefined();
    expect(config.autoCommit).toBe(true);
  });
});
