import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import os from 'os';
import path from 'path';
import { scanUserHooks, neutralizeHook, findHookEntry } from '../cli/lib/hook-compat.js';

// `mint update` consults scanUserHooks after the migration loop and emits
// a warning when `knownUnpatched.length > 0`. The end-to-end CLI run is
// not tested here (update touches ~/.mint, the git remote, npm/bun, etc),
// so we verify the scan contract that the warning branch depends on.

const RAW_CBM_HOOK = `#!/bin/bash
GATE=/tmp/cbm-code-discovery-gate-$PPID
if [ -f "$GATE" ]; then exit 0; fi
touch "$GATE"
echo 'BLOCKED: For code discovery, use codebase-memory-mcp tools first.' >&2
exit 2
`;

let TMP;
beforeEach(() => {
  TMP = fs.mkdtempSync(path.join(os.tmpdir(), 'mint-update-'));
});
afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('mint update — hook compatibility warning', () => {
  test('scan flags unpatched hostile hook so update can warn', () => {
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    fs.writeFileSync(path.join(hooksDir, 'cbm-code-discovery-gate'), RAW_CBM_HOOK);

    const result = scanUserHooks(TMP);
    expect(result.knownUnpatched.length).toBeGreaterThan(0);
    expect(result.knownUnpatched[0].name).toBe('cbm-code-discovery-gate');
  });

  test('scan reports zero unpatched when hook is missing (no warning would fire)', () => {
    const result = scanUserHooks(TMP);
    expect(result.knownUnpatched).toHaveLength(0);
  });

  test('scan reports zero unpatched when hook is already patched (no warning would fire)', () => {
    const hooksDir = path.join(TMP, '.claude', 'hooks');
    fs.mkdirSync(hooksDir, { recursive: true });
    const entry = findHookEntry('cbm-code-discovery-gate');
    fs.writeFileSync(
      path.join(hooksDir, 'cbm-code-discovery-gate'),
      neutralizeHook(RAW_CBM_HOOK, entry),
    );

    const result = scanUserHooks(TMP);
    expect(result.knownUnpatched).toHaveLength(0);
    expect(result.knownPatched).toHaveLength(1);
  });
});
