import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import os from 'os';
import path from 'path';
import {
  KNOWN_HOOKS,
  MINT_PATCH_MARKER_BEGIN,
  MINT_PATCH_MARKER_END,
  detectHook,
  isAlreadyNeutralized,
  neutralizeHook,
  restoreHook,
  scanUserHooks,
  findHookEntry,
  applyHookPatch,
} from '../cli/lib/hook-compat.js';

// A faithful reproduction of the cbm-code-discovery-gate hook body.
// Keep this in sync with the real installer output.
const RAW_CBM_HOOK = `#!/bin/bash
GATE=/tmp/cbm-code-discovery-gate-$PPID
find /tmp -name 'cbm-code-discovery-gate-*' -mtime +1 -delete 2>/dev/null
if [ -f "$GATE" ]; then exit 0; fi
touch "$GATE"
echo 'BLOCKED: For code discovery, use codebase-memory-mcp tools first: search_graph(name_pattern) to find functions/classes, trace_path() for call chains, get_code_snippet(qualified_name) to read source. If the graph is not indexed yet, call index_repository first. Fall back to Grep/Glob/Read only for text content search. If you need Grep, retry.' >&2
exit 2
`;

// The live disk copy currently has a defunct bypass block from an earlier
// abandoned design. Any neutralization must remove it as part of the patch.
const CBM_HOOK_WITH_DEFUNCT_BYPASS = `#!/bin/bash
# mint-bypass-begin (v1)
if [ "\${MINT_SUBAGENT:-}" = "1" ]; then exit 0; fi
# mint-bypass-end
GATE=/tmp/cbm-code-discovery-gate-$PPID
find /tmp -name 'cbm-code-discovery-gate-*' -mtime +1 -delete 2>/dev/null
if [ -f "$GATE" ]; then exit 0; fi
touch "$GATE"
echo 'BLOCKED: For code discovery, use codebase-memory-mcp tools first: search_graph(name_pattern) to find functions/classes, trace_path() for call chains, get_code_snippet(qualified_name) to read source. If the graph is not indexed yet, call index_repository first. Fall back to Grep/Glob/Read only for text content search. If you need Grep, retry.' >&2
exit 2
`;

let TMP;
beforeEach(() => {
  TMP = fs.mkdtempSync(path.join(os.tmpdir(), 'mint-hookcompat-'));
});
afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('KNOWN_HOOKS registry', () => {
  test('contains cbm-code-discovery-gate entry', () => {
    const entry = KNOWN_HOOKS.find((h) => h.name === 'cbm-code-discovery-gate');
    expect(entry).toBeDefined();
    expect(entry.relPath).toBe('.claude/hooks/cbm-code-discovery-gate');
    expect(typeof entry.detect).toBe('function');
    expect(typeof entry.neutralize).toBe('function');
    expect(typeof entry.reason).toBe('string');
  });

  test('findHookEntry returns the cbm entry by name', () => {
    expect(findHookEntry('cbm-code-discovery-gate')).toBeDefined();
    expect(findHookEntry('nonexistent-hook')).toBeUndefined();
  });
});

describe('detectHook', () => {
  const entry = KNOWN_HOOKS.find((h) => h.name === 'cbm-code-discovery-gate');

  test('matches the cbm hook body', () => {
    expect(detectHook(RAW_CBM_HOOK, entry)).toBe(true);
  });

  test('rejects unrelated bash', () => {
    expect(detectHook('#!/bin/bash\necho hello\n', entry)).toBe(false);
  });

  test('rejects empty content', () => {
    expect(detectHook('', entry)).toBe(false);
  });

  test('still matches after neutralization (marker + detect strings remain)', () => {
    // detectHook is not the same as "unpatched" — isAlreadyNeutralized is.
    const patched = neutralizeHook(RAW_CBM_HOOK, entry);
    expect(detectHook(patched, entry)).toBe(true);
  });
});

describe('isAlreadyNeutralized', () => {
  const entry = KNOWN_HOOKS.find((h) => h.name === 'cbm-code-discovery-gate');

  test('returns false on raw content', () => {
    expect(isAlreadyNeutralized(RAW_CBM_HOOK)).toBe(false);
  });

  test('returns true after neutralize', () => {
    const patched = neutralizeHook(RAW_CBM_HOOK, entry);
    expect(isAlreadyNeutralized(patched)).toBe(true);
  });

  test('returns false for non-string', () => {
    expect(isAlreadyNeutralized(null)).toBe(false);
    expect(isAlreadyNeutralized(undefined)).toBe(false);
  });
});

describe('neutralizeHook', () => {
  const entry = KNOWN_HOOKS.find((h) => h.name === 'cbm-code-discovery-gate');

  test('replaces the exit-2 block with an advisory exit-0 block wrapped in markers', () => {
    const patched = neutralizeHook(RAW_CBM_HOOK, entry);
    expect(patched).toContain(MINT_PATCH_MARKER_BEGIN);
    expect(patched).toContain(MINT_PATCH_MARKER_END);
    expect(patched).toContain('exit 0');
    expect(patched).not.toMatch(/\nexit 2\n?$/);
  });

  test('is idempotent — applying twice equals applying once', () => {
    const once = neutralizeHook(RAW_CBM_HOOK, entry);
    const twice = neutralizeHook(once, entry);
    expect(twice).toBe(once);
  });

  test('removes the defunct mint-bypass block', () => {
    const patched = neutralizeHook(CBM_HOOK_WITH_DEFUNCT_BYPASS, entry);
    expect(patched).not.toContain('mint-bypass-begin');
    expect(patched).not.toContain('mint-bypass-end');
    expect(patched).not.toContain('MINT_SUBAGENT');
    // And still contains the mint-patch block.
    expect(patched).toContain(MINT_PATCH_MARKER_BEGIN);
  });

  test('preserves shebang, PPID gate, and cache logic above the enforcement block', () => {
    const patched = neutralizeHook(RAW_CBM_HOOK, entry);
    expect(patched.startsWith('#!/bin/bash')).toBe(true);
    expect(patched).toContain('GATE=/tmp/cbm-code-discovery-gate-$PPID');
    expect(patched).toContain("find /tmp -name 'cbm-code-discovery-gate-*'");
    expect(patched).toContain('if [ -f "$GATE" ]; then exit 0; fi');
    expect(patched).toContain('touch "$GATE"');
  });

  test('preserves CRLF line endings when present', () => {
    const crlf = RAW_CBM_HOOK.replace(/\n/g, '\r\n');
    const patched = neutralizeHook(crlf, entry);
    // The first line break should still be CRLF.
    const firstNewline = patched.indexOf('\n');
    expect(firstNewline).toBeGreaterThan(0);
    expect(patched[firstNewline - 1]).toBe('\r');
    expect(patched).toContain(MINT_PATCH_MARKER_BEGIN);
  });

  test('throws on upstream format drift — no markers and no enforcement block', () => {
    // Content that mentions cbm-code-discovery-gate and "BLOCKED: For code
    // discovery" (so `detect()` still matches) but lacks the `echo ... >&2`
    // + `exit 2` enforcement pair the neutralizer knows how to replace.
    const drifted = `#!/bin/bash
# cbm-code-discovery-gate — refactored upstream
# Comment mentioning BLOCKED: For code discovery so detect() matches
exit 0
`;
    // Sanity: detect() still recognizes this as "the hook".
    expect(detectHook(drifted, entry)).toBe(true);
    // But the enforcement block is missing, so neutralization must throw.
    expect(() => neutralizeHook(drifted, entry)).toThrow(/format drift/);
  });
});

describe('restoreHook', () => {
  const entry = KNOWN_HOOKS.find((h) => h.name === 'cbm-code-discovery-gate');

  test('removes the mint-patch block', () => {
    const patched = neutralizeHook(RAW_CBM_HOOK, entry);
    const restored = restoreHook(patched);
    expect(restored).not.toContain(MINT_PATCH_MARKER_BEGIN);
    expect(restored).not.toContain(MINT_PATCH_MARKER_END);
  });

  test('no-op on content without markers', () => {
    expect(restoreHook(RAW_CBM_HOOK)).toBe(RAW_CBM_HOOK);
  });
});

describe('scanUserHooks', () => {
  test('returns empty-ish result when .claude/hooks dir is missing', () => {
    const result = scanUserHooks(TMP);
    expect(result.knownUnpatched).toEqual([]);
    expect(result.knownPatched).toEqual([]);
    expect(result.missing).toHaveLength(KNOWN_HOOKS.length);
    expect(result.unknown).toEqual([]);
  });

  test('categorizes an unpatched cbm hook correctly', () => {
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    fs.writeFileSync(path.join(hooksDir, 'cbm-code-discovery-gate'), RAW_CBM_HOOK);

    const result = scanUserHooks(TMP);
    expect(result.knownUnpatched).toHaveLength(1);
    expect(result.knownUnpatched[0].name).toBe('cbm-code-discovery-gate');
    expect(result.knownUnpatched[0].entry).toBeDefined();
    expect(result.knownUnpatched[0].entry.name).toBe('cbm-code-discovery-gate');
    expect(result.knownPatched).toHaveLength(0);
    expect(result.missing).toHaveLength(0);
  });

  test('categorizes a neutralized cbm hook as patched', () => {
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    const entry = KNOWN_HOOKS.find((h) => h.name === 'cbm-code-discovery-gate');
    fs.writeFileSync(
      path.join(hooksDir, 'cbm-code-discovery-gate'),
      neutralizeHook(RAW_CBM_HOOK, entry),
    );

    const result = scanUserHooks(TMP);
    expect(result.knownPatched).toHaveLength(1);
    expect(result.knownUnpatched).toHaveLength(0);
  });

  test('treats an unrelated file at the path as missing', () => {
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    fs.writeFileSync(path.join(hooksDir, 'cbm-code-discovery-gate'), '#!/bin/bash\necho hi\n');

    const result = scanUserHooks(TMP);
    expect(result.missing).toHaveLength(1);
    expect(result.knownUnpatched).toHaveLength(0);
  });

  test('rejects a path-traversal relPath as missing (containment check)', () => {
    // Monkey-patch the registry with a hostile entry. KNOWN_HOOKS is an
    // exported `const` array, but arrays are mutable — push + pop is safe
    // here because nothing else mutates the registry.
    const hostile = {
      name: 'hostile-traversal-test',
      relPath: '../../etc/passwd',
      detect: () => true,
      reason: 'test-only: defense-in-depth path traversal probe',
      neutralize: (c) => c,
    };
    KNOWN_HOOKS.push(hostile);
    try {
      const result = scanUserHooks(TMP);
      // The hostile entry should land in `missing`, never in `knownUnpatched`
      // or `knownPatched` — even if `/etc/passwd` exists on the host.
      const inMissing = result.missing.some((h) => h.name === 'hostile-traversal-test');
      const inUnpatched = result.knownUnpatched.some((h) => h.name === 'hostile-traversal-test');
      const inPatched = result.knownPatched.some((h) => h.name === 'hostile-traversal-test');
      expect(inMissing).toBe(true);
      expect(inUnpatched).toBe(false);
      expect(inPatched).toBe(false);
    } finally {
      // Always restore the registry so other tests see the original state.
      const idx = KNOWN_HOOKS.indexOf(hostile);
      if (idx !== -1) KNOWN_HOOKS.splice(idx, 1);
    }
  });
});

describe('applyHookPatch', () => {
  const entry = KNOWN_HOOKS.find((h) => h.name === 'cbm-code-discovery-gate');

  test('neutralizes an unpatched hook and writes a backup', () => {
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    const hookPath = path.join(hooksDir, 'cbm-code-discovery-gate');
    fs.writeFileSync(hookPath, RAW_CBM_HOOK);
    fs.chmodSync(hookPath, 0o755);

    const result = applyHookPatch(entry, { homeDir: TMP });
    expect(result.patched).toBe(true);

    // File content is now neutralized.
    const patched = fs.readFileSync(hookPath, 'utf8');
    expect(isAlreadyNeutralized(patched)).toBe(true);

    // Backup exists and holds the original bytes.
    const backup = `${hookPath}.mint-backup`;
    expect(fs.existsSync(backup)).toBe(true);
    expect(fs.readFileSync(backup, 'utf8')).toBe(RAW_CBM_HOOK);

    // Executable bit preserved.
    const mode = fs.statSync(hookPath).mode & 0o777;
    expect(mode & 0o100).toBe(0o100);
  });

  test('is a no-op on an already-neutralized hook', () => {
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    const hookPath = path.join(hooksDir, 'cbm-code-discovery-gate');
    fs.writeFileSync(hookPath, neutralizeHook(RAW_CBM_HOOK, entry));

    const result = applyHookPatch(entry, { homeDir: TMP });
    expect(result.patched).toBe(false);
    expect(result.reason).toMatch(/already neutralized/);
  });

  test('returns missing when the hook file does not exist', () => {
    const result = applyHookPatch(entry, { homeDir: TMP });
    expect(result.patched).toBe(false);
    expect(result.reason).toMatch(/not found/);
  });

  test('refuses to patch a symlinked hook path', () => {
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    // Target of the symlink: an unrelated file we must not touch.
    const realTarget = path.join(TMP, 'real-target');
    fs.writeFileSync(realTarget, RAW_CBM_HOOK);
    const hookPath = path.join(hooksDir, 'cbm-code-discovery-gate');
    fs.symlinkSync(realTarget, hookPath);

    const result = applyHookPatch(entry, { homeDir: TMP });
    expect(result.patched).toBe(false);
    expect(result.reason).toMatch(/symbolic link|symlink/i);

    // Target is untouched.
    expect(fs.readFileSync(realTarget, 'utf8')).toBe(RAW_CBM_HOOK);
    // And no backup was created along the symlink path.
    expect(fs.existsSync(`${hookPath}.mint-backup`)).toBe(false);
  });

  test('refuses when the backup destination is itself a symlink', () => {
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    const hookPath = path.join(hooksDir, 'cbm-code-discovery-gate');
    fs.writeFileSync(hookPath, RAW_CBM_HOOK);

    // Pre-create the backup path AS a symlink pointing to an unrelated
    // file. If applyHookPatch ignored this, the backup write would
    // clobber the target.
    const innocentTarget = path.join(TMP, 'innocent');
    fs.writeFileSync(innocentTarget, 'do-not-touch\n');
    fs.symlinkSync(innocentTarget, `${hookPath}.mint-backup`);

    const result = applyHookPatch(entry, { homeDir: TMP });
    expect(result.patched).toBe(false);
    expect(result.reason).toMatch(/symbolic link|symlink/i);

    // Innocent target is untouched.
    expect(fs.readFileSync(innocentTarget, 'utf8')).toBe('do-not-touch\n');
    // Original hook file is unchanged too.
    expect(fs.readFileSync(hookPath, 'utf8')).toBe(RAW_CBM_HOOK);
  });

  test('does not clobber a pre-existing (regular-file) backup', () => {
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    const hookPath = path.join(hooksDir, 'cbm-code-discovery-gate');
    fs.writeFileSync(hookPath, RAW_CBM_HOOK);

    // Pre-existing backup with sentinel contents. We must not overwrite it.
    const backupPath = `${hookPath}.mint-backup`;
    fs.writeFileSync(backupPath, 'pre-existing backup contents\n');

    const result = applyHookPatch(entry, { homeDir: TMP });
    expect(result.patched).toBe(true);
    expect(fs.readFileSync(backupPath, 'utf8')).toBe('pre-existing backup contents\n');
    // And the hook is in fact patched.
    expect(isAlreadyNeutralized(fs.readFileSync(hookPath, 'utf8'))).toBe(true);
  });
});
