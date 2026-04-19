/**
 * Hook compatibility registry and neutralizer.
 *
 * User PreToolUse hooks (notably `~/.claude/hooks/cbm-code-discovery-gate` from
 * codebase-memory-mcp) can `exit 2` to block Read/Grep/Glob in mint subagents.
 * Subagents inherit the hook but may not have the MCP server it redirects to.
 *
 * This module detects known-hostile hooks and rewrites their blocking
 * enforcement into an advisory warning (`exit 0`) guarded by idempotent
 * marker comments. The rest of the hook (shebang, PPID gates, cache
 * sweeps, cache-hit shortcuts) is preserved intact.
 *
 * Pattern mirrors `cli/lib/gitignore.js` — a registry of entries plus
 * pure functions that take content and return content.
 */
import fs from 'fs';
import os from 'os';
import path from 'path';

export const MINT_PATCH_MARKER_BEGIN = '# mint-patch-begin (v1)';
export const MINT_PATCH_MARKER_END = '# mint-patch-end';

// Defunct marker from an earlier (abandoned) env-var bypass design.
// The neutralizer strips any leftover block wrapped in these markers.
const DEFUNCT_BYPASS_BEGIN = '# mint-bypass-begin (v1)';
const DEFUNCT_BYPASS_END = '# mint-bypass-end';

/**
 * Detect the line ending used in `content`. Returns '\r\n' if the first
 * newline is CRLF, otherwise '\n'. Empty/one-line content defaults to '\n'.
 */
function detectEol(content) {
  const idx = content.indexOf('\n');
  if (idx > 0 && content[idx - 1] === '\r') return '\r\n';
  return '\n';
}

/**
 * Strip any defunct `# mint-bypass-begin (v1)` ... `# mint-bypass-end`
 * block (plus the trailing newline after the end marker, if any). The
 * block may appear anywhere in the file.
 */
function stripDefunctBypass(content) {
  if (!content.includes(DEFUNCT_BYPASS_BEGIN)) return content;

  const eol = detectEol(content);
  const lines = content.split(eol);
  const out = [];
  let inBlock = false;
  for (const line of lines) {
    if (!inBlock && line.trim().startsWith(DEFUNCT_BYPASS_BEGIN)) {
      inBlock = true;
      continue;
    }
    if (inBlock) {
      if (line.trim().startsWith(DEFUNCT_BYPASS_END)) {
        inBlock = false;
      }
      continue;
    }
    out.push(line);
  }
  return out.join(eol);
}

/**
 * Neutralize the cbm-code-discovery-gate hook. Replaces the final
 * enforcement block (`echo 'BLOCKED...' >&2` followed by `exit 2`) with
 * an advisory echo + `exit 0`, wrapped in mint-patch markers so the
 * patch is idempotent and reversible. Also strips any defunct
 * mint-bypass block left over from the abandoned earlier design.
 *
 * Throws when the content has no mint-patch marker AND no matching
 * enforcement block — i.e. upstream has shipped a new format we don't
 * recognize. Failing loud prevents `mint doctor --fix` from silently
 * reporting success while leaving a hostile hook in place.
 */
function neutralizeCbmHook(content) {
  if (isAlreadyNeutralized(content)) return content;

  // Always strip defunct bypass first so the remaining content is clean.
  let patched = stripDefunctBypass(content);

  const eol = detectEol(patched);
  const lines = patched.split(eol);

  // Find the enforcement block: an `echo '...BLOCKED...' >&2` line
  // followed (possibly after blank lines) by `exit 2`.
  let blockedIdx = -1;
  let exitIdx = -1;
  for (let i = 0; i < lines.length; i++) {
    const trimmed = lines[i].trim();
    if (blockedIdx === -1 && trimmed.startsWith('echo') && trimmed.includes('BLOCKED') && trimmed.includes('>&2')) {
      blockedIdx = i;
      continue;
    }
    if (blockedIdx !== -1 && trimmed === 'exit 2') {
      exitIdx = i;
      break;
    }
  }

  if (blockedIdx === -1 || exitIdx === -1) {
    throw new Error(
      'cbm-code-discovery-gate: upstream format drift — enforcement block not found',
    );
  }

  const replacement = [
    MINT_PATCH_MARKER_BEGIN,
    "echo '[mint] cbm discovery gate neutralized — use codebase-memory-mcp tools for better code exploration' >&2",
    'exit 0',
    MINT_PATCH_MARKER_END,
  ];

  const next = [
    ...lines.slice(0, blockedIdx),
    ...replacement,
    ...lines.slice(exitIdx + 1),
  ];

  return next.join(eol);
}

/** Known-hostile user hooks that mint can detect and neutralize. */
export const KNOWN_HOOKS = [
  {
    name: 'cbm-code-discovery-gate',
    relPath: '.claude/hooks/cbm-code-discovery-gate',
    detect: (content) =>
      content.includes('cbm-code-discovery-gate') &&
      (content.includes('BLOCKED: For code discovery') ||
        content.includes(MINT_PATCH_MARKER_BEGIN)),
    reason:
      'blocks Read/Grep/Glob via exit 2; redirects to codebase-memory-mcp tools that may not be reachable from mint subagents',
    neutralize: (content) => neutralizeCbmHook(content),
  },
];

/** Returns true if `entry.detect` matches `content`. */
export function detectHook(content, entry) {
  if (!content || !entry || typeof entry.detect !== 'function') return false;
  return entry.detect(content);
}

/** Returns true if the content already carries the mint-patch marker. */
export function isAlreadyNeutralized(content) {
  return typeof content === 'string' && content.includes(MINT_PATCH_MARKER_BEGIN);
}

/**
 * Apply `entry.neutralize` to `content` and return the patched string.
 * Idempotent — applying twice yields the same result as applying once.
 * May throw if the neutralizer detects upstream format drift.
 */
export function neutralizeHook(content, entry) {
  if (!entry || typeof entry.neutralize !== 'function') return content;
  return entry.neutralize(content);
}

/**
 * Restore a neutralized hook by removing the mint-patch block. The
 * returned content still lacks the original `exit 2` enforcement — this
 * is the inverse of `neutralizeHook` (strip our patch), not a full
 * rollback to a pristine vendor copy. Use the `.mint-backup` file for
 * that.
 */
export function restoreHook(content) {
  if (!isAlreadyNeutralized(content)) return content;

  const eol = detectEol(content);
  const lines = content.split(eol);
  const out = [];
  let inBlock = false;
  for (const line of lines) {
    if (!inBlock && line.trim() === MINT_PATCH_MARKER_BEGIN) {
      inBlock = true;
      continue;
    }
    if (inBlock) {
      if (line.trim() === MINT_PATCH_MARKER_END) {
        inBlock = false;
      }
      continue;
    }
    out.push(line);
  }
  return out.join(eol);
}

/**
 * Returns true if `full` resolves to a path that lives under `homeDir`.
 * Uses `path.resolve` on both sides so symlink components aren't
 * consulted (`realpath` would chase them, which we explicitly don't
 * want here). Defense-in-depth: today `relPath` values in KNOWN_HOOKS
 * are hard-coded, but a future change could move them to config.
 */
function isUnderHome(full, homeDir) {
  const resolvedHome = path.resolve(homeDir);
  const resolvedFull = path.resolve(full);
  if (resolvedFull === resolvedHome) return true;
  const withSep = resolvedHome.endsWith(path.sep)
    ? resolvedHome
    : resolvedHome + path.sep;
  return resolvedFull.startsWith(withSep);
}

/**
 * Scan `<homeDir>/.claude/hooks` for known hostile hooks and categorize
 * them. Always returns an object with `knownUnpatched`, `knownPatched`,
 * `missing`, and `unknown` arrays. `unknown` is reserved for heuristic
 * detection (future feature) and is always `[]` today.
 *
 * Each entry is `{ name, path, entry }` where `entry` is the matching
 * registry object. Entries whose resolved path escapes `homeDir` are
 * treated as missing (defense-in-depth against future path-traversal
 * if the registry ever comes from user config).
 */
export function scanUserHooks(homeDir = os.homedir()) {
  const result = {
    knownUnpatched: [],
    knownPatched: [],
    missing: [],
    unknown: [],
  };

  for (const entry of KNOWN_HOOKS) {
    const full = path.join(homeDir, entry.relPath);

    // Containment check. If the resolved path escapes homeDir, treat
    // the entry as missing rather than throwing.
    if (!isUnderHome(full, homeDir)) {
      result.missing.push({ name: entry.name, path: full, entry });
      continue;
    }

    let content;
    try {
      content = fs.readFileSync(full, 'utf8');
    } catch {
      result.missing.push({ name: entry.name, path: full, entry });
      continue;
    }

    if (!detectHook(content, entry)) {
      // File exists but isn't the hostile hook we know about — treat as missing.
      result.missing.push({ name: entry.name, path: full, entry });
      continue;
    }

    if (isAlreadyNeutralized(content)) {
      result.knownPatched.push({ name: entry.name, path: full, entry });
    } else {
      result.knownUnpatched.push({ name: entry.name, path: full, entry });
    }
  }

  return result;
}

/** Lookup a registry entry by name. Returns undefined if not found. */
export function findHookEntry(name) {
  return KNOWN_HOOKS.find((h) => h.name === name);
}

/**
 * Apply patch to a known hook on disk. Includes safety checks:
 *   - containment check: hook path must live under homeDir
 *   - symlink TOCTOU check: refuses a hook file that is a symbolic link
 *   - backup-path symlink check: refuses if `.mint-backup` exists AS a symlink
 *
 * Creates a `.mint-backup` copy of the original before patching (only
 * if no backup exists yet — never clobbers an existing backup). Preserves
 * the executable bit on the patched file.
 *
 * Returns `{ patched: true }` on success, `{ patched: false, reason }`
 * when a safety check or pre-condition rejects the target. Throws if
 * read/write fails or if the neutralizer detects upstream format drift.
 */
export function applyHookPatch(entry, { homeDir = os.homedir() } = {}) {
  if (!entry || typeof entry.neutralize !== 'function') {
    return { patched: false, reason: 'invalid registry entry' };
  }

  const full = path.join(homeDir, entry.relPath);

  // Containment check.
  if (!isUnderHome(full, homeDir)) {
    return { patched: false, reason: 'hook path escapes homeDir' };
  }

  // Symlink-safety check on the hook file itself. A malicious symlink
  // could redirect the later `writeFileSync` to an unrelated target.
  let linkStat;
  try {
    linkStat = fs.lstatSync(full);
  } catch (err) {
    return { patched: false, reason: `hook not found: ${err.message}` };
  }
  if (linkStat.isSymbolicLink()) {
    return {
      patched: false,
      reason: 'hook is a symbolic link, skipping for safety',
    };
  }

  // Symlink-safety check on the backup destination: if something
  // already occupies that path AS a symlink, skip the entire hook so
  // we never clobber an unrelated target via the backup write.
  const backupPath = `${full}.mint-backup`;
  try {
    const backupLinkStat = fs.lstatSync(backupPath);
    if (backupLinkStat.isSymbolicLink()) {
      return {
        patched: false,
        reason: 'backup path exists as a symbolic link, skipping for safety',
      };
    }
  } catch {
    // Backup doesn't exist yet — that's the normal case.
  }

  const original = fs.readFileSync(full, 'utf8');

  if (!detectHook(original, entry)) {
    return {
      patched: false,
      reason: 'hook content did not match registry detect()',
    };
  }

  if (isAlreadyNeutralized(original)) {
    return { patched: false, reason: 'already neutralized' };
  }

  // Preserve executable bit before we overwrite.
  const origMode = fs.statSync(full).mode;

  // Write backup first (only if none exists) so the user can recover
  // the vendor-shipped content even after we patch.
  if (!fs.existsSync(backupPath)) {
    fs.writeFileSync(backupPath, original);
  }

  // Neutralize — may throw on upstream format drift.
  const patched = neutralizeHook(original, entry);

  fs.writeFileSync(full, patched);
  fs.chmodSync(full, origMode & 0o777);

  return { patched: true };
}
