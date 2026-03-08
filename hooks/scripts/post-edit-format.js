#!/usr/bin/env node
/**
 * mint hook: Auto-format after edits
 * Trigger: PostToolUse (Edit)
 * Detects Biome or Prettier and formats the edited file.
 * Fails silently if no formatter is available.
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

function findProjectRoot(startDir) {
  let dir = startDir;
  while (dir !== path.dirname(dir)) {
    if (fs.existsSync(path.join(dir, 'package.json'))) return dir;
    dir = path.dirname(dir);
  }
  return startDir;
}

function detectFormatter(root) {
  if (['biome.json', 'biome.jsonc'].some(f => fs.existsSync(path.join(root, f)))) return 'biome';
  const prettierConfigs = ['.prettierrc', '.prettierrc.json', '.prettierrc.js', '.prettierrc.yml',
    '.prettierrc.yaml', 'prettier.config.js', 'prettier.config.cjs', 'prettier.config.mjs'];
  if (prettierConfigs.some(f => fs.existsSync(path.join(root, f)))) return 'prettier';
  return null;
}

process.stdin.on('end', () => {
  try {
    const input = JSON.parse(data);
    const filePath = input.tool_input?.file_path;
    if (filePath && /\.(ts|tsx|js|jsx|json)$/.test(filePath)) {
      const root = findProjectRoot(path.dirname(path.resolve(filePath)));
      const formatter = detectFormatter(root);
      const npx = process.platform === 'win32' ? 'npx.cmd' : 'npx';
      if (formatter === 'biome') {
        execFileSync(npx, ['@biomejs/biome', 'format', '--write', filePath], { cwd: root, stdio: 'pipe', timeout: 15000 });
      } else if (formatter === 'prettier') {
        execFileSync(npx, ['prettier', '--write', filePath], { cwd: root, stdio: 'pipe', timeout: 15000 });
      }
    }
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
