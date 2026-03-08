#!/usr/bin/env node
/**
 * mint hook: TypeScript check after edits
 * Trigger: PostToolUse (Edit)
 * Runs tsc --noEmit and reports only errors in the edited file.
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

function findTsconfig(startDir) {
  let dir = startDir;
  let depth = 0;
  while (dir !== path.parse(dir).root && depth < 20) {
    if (fs.existsSync(path.join(dir, 'tsconfig.json'))) return dir;
    dir = path.dirname(dir);
    depth++;
  }
  return null;
}

process.stdin.on('end', () => {
  try {
    const input = JSON.parse(data);
    const filePath = input.tool_input?.file_path;
    if (filePath && /\.(ts|tsx)$/.test(filePath)) {
      const resolved = path.resolve(filePath);
      if (!fs.existsSync(resolved)) { process.stdout.write(data); return; }
      const tsconfigDir = findTsconfig(path.dirname(resolved));
      if (tsconfigDir) {
        try {
          const npx = process.platform === 'win32' ? 'npx.cmd' : 'npx';
          execFileSync(npx, ['tsc', '--noEmit', '--pretty', 'false'], {
            cwd: tsconfigDir, encoding: 'utf8', stdio: ['pipe', 'pipe', 'pipe'], timeout: 30000
          });
        } catch (err) {
          const output = (err.stdout || '') + (err.stderr || '');
          const relPath = path.relative(tsconfigDir, resolved);
          const candidates = new Set([filePath, resolved, relPath]);
          const relevant = output.split('\n')
            .filter(line => [...candidates].some(c => line.includes(c)))
            .slice(0, 10);
          if (relevant.length > 0) {
            console.error(`[mint] TypeScript errors in ${path.basename(filePath)}:`);
            relevant.forEach(line => console.error(line));
          }
        }
      }
    }
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
