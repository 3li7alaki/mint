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

// ─── Hook compatibility: exercise the underlying functions directly ─────
// @clack/prompts is interactive and can't be driven through execSync, so we
// test the library functions (used by `mint doctor`) against a tmp HOME.
describe('mint doctor — hook compatibility', () => {
  const RAW_CBM_HOOK = `#!/bin/bash
GATE=/tmp/cbm-code-discovery-gate-$PPID
find /tmp -name 'cbm-code-discovery-gate-*' -mtime +1 -delete 2>/dev/null
if [ -f "$GATE" ]; then exit 0; fi
touch "$GATE"
echo 'BLOCKED: For code discovery, use codebase-memory-mcp tools first.' >&2
exit 2
`;

  test('scan reports no hostile hooks when .claude/hooks is empty', async () => {
    const { scanUserHooks } = await import('../cli/lib/hook-compat.js');
    const result = scanUserHooks(TMP);
    expect(result.knownUnpatched).toHaveLength(0);
    expect(result.knownPatched).toHaveLength(0);
  });

  test('scan detects an unpatched cbm hook', async () => {
    const { scanUserHooks } = await import('../cli/lib/hook-compat.js');
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    fs.writeFileSync(path.join(hooksDir, 'cbm-code-discovery-gate'), RAW_CBM_HOOK);

    const result = scanUserHooks(TMP);
    expect(result.knownUnpatched).toHaveLength(1);
    expect(result.knownUnpatched[0].name).toBe('cbm-code-discovery-gate');
  });

  test('fix flow neutralizes an unpatched hook and writes a backup', async () => {
    const { scanUserHooks, findHookEntry, neutralizeHook, isAlreadyNeutralized } =
      await import('../cli/lib/hook-compat.js');
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    const hookPath = path.join(hooksDir, 'cbm-code-discovery-gate');
    fs.writeFileSync(hookPath, RAW_CBM_HOOK);
    fs.chmodSync(hookPath, 0o755);

    // Emulate what `mint doctor --fix` does per hook: backup + patch + chmod.
    const scan = scanUserHooks(TMP);
    expect(scan.knownUnpatched).toHaveLength(1);
    for (const h of scan.knownUnpatched) {
      const entry = findHookEntry(h.name);
      const original = fs.readFileSync(h.path, 'utf8');
      const backup = `${h.path}.mint-backup`;
      if (!fs.existsSync(backup)) fs.writeFileSync(backup, original);
      fs.writeFileSync(h.path, neutralizeHook(original, entry));
      fs.chmodSync(h.path, 0o755);
    }

    const patched = fs.readFileSync(hookPath, 'utf8');
    expect(isAlreadyNeutralized(patched)).toBe(true);
    expect(fs.existsSync(`${hookPath}.mint-backup`)).toBe(true);
    expect(fs.readFileSync(`${hookPath}.mint-backup`, 'utf8')).toBe(RAW_CBM_HOOK);
    // Executable bit preserved.
    const mode = fs.statSync(hookPath).mode & 0o777;
    expect(mode & 0o100).toBe(0o100);
  });

  test('fix flow is a no-op on an already-neutralized hook (scan reports patched)', async () => {
    const { scanUserHooks, findHookEntry, neutralizeHook } =
      await import('../cli/lib/hook-compat.js');
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    const hookPath = path.join(hooksDir, 'cbm-code-discovery-gate');
    const entry = findHookEntry('cbm-code-discovery-gate');
    const patched = neutralizeHook(RAW_CBM_HOOK, entry);
    fs.writeFileSync(hookPath, patched);

    const result = scanUserHooks(TMP);
    expect(result.knownUnpatched).toHaveLength(0);
    expect(result.knownPatched).toHaveLength(1);

    // Applying neutralize again produces the same content.
    const content = fs.readFileSync(hookPath, 'utf8');
    expect(neutralizeHook(content, entry)).toBe(content);
  });
});
