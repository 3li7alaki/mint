#!/usr/bin/env node
/**
 * mint hook: Test-on-save
 * Trigger: PostToolUse (Edit)
 * Auto-runs relevant tests after editing source files.
 *
 * Configurable via .mint/config.json:
 *   "hooks": { "testOnSave": true }    — enabled (default: false)
 *   "hooks": { "testOnSave": false }   — disabled
 *   "hooks": { "testOnSave": { "enabled": true, "timeout": 30000 } }
 *
 * Detection strategy:
 * 1. If editing a test file — run that test file directly
 * 2. If editing a source file — look for a matching test file and run it
 * 3. Uses project's configured test runner (vitest, jest, pytest, etc.)
 */
'use strict';

const { execFileSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const MAX_STDIN = 1024 * 1024;
let data = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => {
  if (data.length < MAX_STDIN) data += chunk.substring(0, MAX_STDIN - data.length);
});

function findMintConfig() {
  let dir = process.cwd();
  let depth = 0;
  while (dir !== path.parse(dir).root && depth < 20) {
    const configPath = path.join(dir, '.mint', 'config.json');
    if (fs.existsSync(configPath)) {
      try { return { config: JSON.parse(fs.readFileSync(configPath, 'utf8')), root: dir }; }
      catch { return null; }
    }
    dir = path.dirname(dir);
    depth++;
  }
  return null;
}

function isTestFile(filePath) {
  const base = path.basename(filePath);
  return /\.(test|spec)\.(ts|tsx|js|jsx|py)$/.test(base) ||
         /^test_/.test(base) ||
         filePath.includes('__tests__') ||
         filePath.includes('/tests/');
}

function findMatchingTest(filePath, root) {
  const ext = path.extname(filePath);
  const base = path.basename(filePath, ext);
  const dir = path.dirname(filePath);

  // Common test file patterns
  const candidates = [
    path.join(dir, `${base}.test${ext}`),
    path.join(dir, `${base}.spec${ext}`),
    path.join(dir, '__tests__', `${base}.test${ext}`),
    path.join(dir, '__tests__', `${base}.spec${ext}`),
    // Python
    path.join(dir, `test_${base}${ext}`),
    path.join(dir, 'tests', `test_${base}${ext}`),
  ];

  for (const candidate of candidates) {
    const resolved = path.resolve(root, candidate);
    if (fs.existsSync(resolved)) return resolved;
  }
  return null;
}

function detectTestRunner(root) {
  // Check package.json scripts
  const pkgPath = path.join(root, 'package.json');
  if (fs.existsSync(pkgPath)) {
    try {
      const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
      const scripts = pkg.scripts || {};
      const deps = { ...pkg.dependencies, ...pkg.devDependencies };

      if (deps.vitest || scripts.test?.includes('vitest')) return { cmd: 'npx', args: ['vitest', 'run'] };
      if (deps.jest || scripts.test?.includes('jest')) return { cmd: 'npx', args: ['jest', '--bail'] };
      if (deps.mocha || scripts.test?.includes('mocha')) return { cmd: 'npx', args: ['mocha'] };
    } catch { /* fall through */ }
  }

  // Check for Python
  if (fs.existsSync(path.join(root, 'pyproject.toml')) || fs.existsSync(path.join(root, 'pytest.ini'))) {
    return { cmd: 'python', args: ['-m', 'pytest', '-x'] };
  }

  return null;
}

process.stdin.on('end', () => {
  try {
    const input = JSON.parse(data);
    const filePath = input.tool_input?.file_path;
    if (!filePath) { process.stdout.write(data); return; }

    // Only for code files
    if (!/\.(ts|tsx|js|jsx|py|rb)$/.test(filePath)) { process.stdout.write(data); return; }

    const found = findMintConfig();
    if (!found) { process.stdout.write(data); return; }

    const { config, root } = found;

    // Check if test-on-save is enabled
    const hookConfig = config.hooks?.testOnSave;
    if (!hookConfig) { process.stdout.write(data); return; }
    const enabled = hookConfig === true || hookConfig?.enabled === true;
    if (!enabled) { process.stdout.write(data); return; }

    const timeout = (hookConfig?.timeout || 30) * 1000;

    // Find the test file to run
    let testFile;
    if (isTestFile(filePath)) {
      testFile = path.resolve(filePath);
    } else {
      testFile = findMatchingTest(filePath, root);
    }

    if (!testFile) { process.stdout.write(data); return; }

    const runner = detectTestRunner(root);
    if (!runner) { process.stdout.write(data); return; }

    // Run the specific test file
    const relTest = path.relative(root, testFile);
    try {
      execFileSync(runner.cmd, [...runner.args, relTest], {
        cwd: root, encoding: 'utf8', stdio: ['pipe', 'pipe', 'pipe'], timeout
      });
      console.error(`[mint] Tests passed: ${relTest}`);
    } catch (err) {
      const output = ((err.stdout || '') + (err.stderr || '')).split('\n').slice(-15).join('\n');
      console.error(`[mint] Test failures in ${relTest}:\n${output}`);
    }
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
