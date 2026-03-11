#!/usr/bin/env node
/**
 * mint hook: console.log warning
 * Trigger: PostToolUse (Edit)
 * Warns if console.log was added to the edited file.
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
    const input = JSON.parse(data);
    const filePath = input.tool_input?.file_path;
    if (filePath && /\.(ts|tsx|js|jsx)$/.test(filePath) && fs.existsSync(filePath)) {
      const content = fs.readFileSync(filePath, 'utf8');
      const matches = content.match(/console\.log\(/g);
      if (matches && matches.length > 0) {
        console.error(`[mint] ⚠ ${path.basename(filePath)} has ${matches.length} console.log statement(s) — remove before committing`);
      }
    }
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
