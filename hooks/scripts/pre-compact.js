#!/usr/bin/env node
/**
 * mint hook: Pre-compact state save
 * Trigger: PreCompact
 * Saves current execution state before context compaction so nothing is lost.
 */
'use strict';

const fs = require('fs');
const path = require('path');

const MAX_STDIN = 1024 * 1024;
let data = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => {
  if (data.length < MAX_STDIN) data += chunk.substring(0, MAX_STDIN - data.length);
});

process.stdin.on('end', () => {
  try {
    const mintDir = path.join(process.cwd(), '.mint');
    if (!fs.existsSync(mintDir)) { process.stdout.write(data); return; }

    const stateFile = path.join(mintDir, 'pre-compact-state.json');
    const state = {
      timestamp: new Date().toISOString(),
      cwd: process.cwd(),
      git_branch: (() => {
        try {
          return require('child_process').execFileSync('git', ['branch', '--show-current'], { encoding: 'utf8' }).trim();
        } catch { return 'unknown'; }
      })(),
      open_tasks: (() => {
        try {
          const tasksDir = path.join(mintDir, 'tasks');
          if (!fs.existsSync(tasksDir)) return [];
          return fs.readdirSync(tasksDir).filter(d =>
            fs.statSync(path.join(tasksDir, d)).isDirectory()
          );
        } catch { return []; }
      })()
    };
    fs.writeFileSync(stateFile, JSON.stringify(state, null, 2));
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
