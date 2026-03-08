#!/usr/bin/env node
/**
 * mint hook: Quality gate
 * Trigger: PostToolUse (Edit|Write)
 * Runs lint/format check on the edited file.
 */
'use strict';

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const MAX_STDIN = 1024 * 1024;
let data = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => {
  if (data.length < MAX_STDIN) data += chunk.substring(0, MAX_STDIN - data.length);
});

process.stdin.on('end', () => {
  try {
    const input = JSON.parse(data);
    const filePath = String(input.tool_input?.file_path || '');
    if (!filePath || !fs.existsSync(filePath)) { process.stdout.write(data); return; }

    const ext = path.extname(filePath).toLowerCase();

    if (['.ts', '.tsx', '.js', '.jsx', '.json'].includes(ext)) {
      // Prefer biome
      if (fs.existsSync('biome.json') || fs.existsSync('biome.jsonc')) {
        const r = spawnSync('npx', ['biome', 'check', filePath], { encoding: 'utf8', timeout: 15000 });
        if (r.status !== 0) console.error(`[mint] Biome issues in ${path.basename(filePath)}`);
      }
    } else if (ext === '.py') {
      const r = spawnSync('ruff', ['check', filePath], { encoding: 'utf8', timeout: 10000 });
      if (r.status !== 0) console.error(`[mint] Ruff issues in ${path.basename(filePath)}`);
    }
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
