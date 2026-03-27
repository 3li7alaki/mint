#!/usr/bin/env node
/**
 * mint hook: Instinct observer
 * Trigger: PostToolUse (Edit|Write)
 * Observes patterns in agent behavior and records instincts to .mint/instincts.md
 *
 * Watches for recurring code patterns, naming conventions, import styles, and test
 * patterns. Confidence grows as the same pattern is seen across different files.
 * Patterns with confidence >= 3 are high-confidence conventions.
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

function getMintDir() {
  let dir = process.cwd();
  let depth = 0;
  while (dir !== path.parse(dir).root && depth < 20) {
    if (fs.existsSync(path.join(dir, '.mint', 'config.json'))) return path.join(dir, '.mint');
    dir = path.dirname(dir);
    depth++;
  }
  return null;
}

function extractPatterns(filePath, content) {
  const patterns = [];
  const ext = path.extname(filePath);

  if (/\.(ts|tsx|js|jsx|mjs|cjs)$/.test(ext) && content) {
    const lines = content.split('\n').slice(0, 50);
    const imports = lines.filter(l => l.startsWith('import '));
    if (imports.length > 0) {
      const hasBarrel = imports.some(l => /from ['"][.\/]+['"]$/.test(l) || /from ['"]@\//.test(l));
      const hasAbsolute = imports.some(l => /from ['"]@\//.test(l) || /from ['"]~\//.test(l));
      if (hasBarrel) patterns.push({ category: 'imports', observation: 'barrel-imports', file: filePath });
      if (hasAbsolute) patterns.push({ category: 'imports', observation: 'alias-imports', file: filePath });
    }

    const funcDefs = lines.filter(l => /^(export )?(const|function|async function) /.test(l.trim()));
    const hasCamel = funcDefs.some(l => /\b[a-z][a-zA-Z]+\b/.test(l));
    const hasSnake = funcDefs.some(l => /\b[a-z]+_[a-z]+\b/.test(l));
    if (hasCamel && !hasSnake) patterns.push({ category: 'naming', observation: 'camelCase-functions', file: filePath });
    if (hasSnake && !hasCamel) patterns.push({ category: 'naming', observation: 'snake_case-functions', file: filePath });

    if (filePath.includes('test') || filePath.includes('spec')) {
      if (content.includes('describe(') && (content.includes('test(') || content.includes('it('))) {
        patterns.push({ category: 'tests', observation: 'describe-it-pattern', file: filePath });
      }
      if (content.includes('expect(')) patterns.push({ category: 'tests', observation: 'expect-assertions', file: filePath });
      if (content.includes('assert.')) patterns.push({ category: 'tests', observation: 'assert-assertions', file: filePath });
    }
  }

  if (/\.vue$/.test(ext)) patterns.push({ category: 'framework', observation: 'vue-sfc', file: filePath });
  if (/\.(tsx|jsx)$/.test(ext) && content && content.includes('React')) {
    patterns.push({ category: 'framework', observation: 'react-component', file: filePath });
  }
  if (/\.py$/.test(ext)) patterns.push({ category: 'framework', observation: 'python', file: filePath });
  if (/\.go$/.test(ext)) patterns.push({ category: 'framework', observation: 'go', file: filePath });
  if (/\.rs$/.test(ext)) patterns.push({ category: 'framework', observation: 'rust', file: filePath });

  return patterns;
}

/**
 * Record an instinct observation as a JSONL append.
 *
 * Instead of read-modify-write on a markdown table (which races under concurrent
 * sessions), we append one JSON line per observation. Confidence is computed at
 * read time by counting distinct files per (category, observation) pair.
 *
 * Entry format: { category, observation, file, date }
 * To compute confidence: group by (category, observation), count distinct files.
 */
function recordInstinct(mintDir, pattern) {
  const instinctsPath = path.join(mintDir, 'instincts.jsonl');
  const fileName = path.basename(pattern.file);
  const today = new Date().toISOString().split('T')[0];

  const entry = JSON.stringify({
    category: pattern.category,
    observation: pattern.observation,
    file: fileName,
    date: today,
  });

  // appendFileSync is atomic for writes under 4KB on POSIX
  fs.appendFileSync(instinctsPath, entry + '\n');
}

process.stdin.on('end', () => {
  try {
    const input = JSON.parse(data);
    const filePath = input.tool_input?.file_path;
    if (!filePath) { process.stdout.write(data); return; }

    const mintDir = getMintDir();
    if (!mintDir) { process.stdout.write(data); return; }

    // Check if instinct learning is enabled
    try {
      const config = JSON.parse(fs.readFileSync(path.join(mintDir, 'config.json'), 'utf8'));
      if (config.instincts?.enabled === false) { process.stdout.write(data); return; }
    } catch { /* proceed with defaults */ }

    const resolved = path.resolve(filePath);
    let content = '';
    try { content = fs.readFileSync(resolved, 'utf8'); } catch { /* file may not exist yet */ }

    const patterns = extractPatterns(filePath, content);
    for (const p of patterns) {
      recordInstinct(mintDir, p);
    }
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
