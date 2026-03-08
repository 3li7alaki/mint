#!/usr/bin/env node
/**
 * mint hook: Instinct observer
 * Trigger: PostToolUse (Edit|Write)
 * Observes patterns in agent behavior and records instincts to .mint/instincts.md
 *
 * Lightweight version of pattern extraction — watches for recurring code patterns,
 * naming conventions, import styles, and test patterns. Records observations that
 * the planner can use to write better specs.
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

  // Detect import style
  if (/\.(ts|tsx|js|jsx)$/.test(ext) && content) {
    const lines = content.split('\n').slice(0, 50);
    const imports = lines.filter(l => l.startsWith('import '));
    if (imports.length > 0) {
      const hasBarrel = imports.some(l => /from ['"][.\/]+['"]$/.test(l) || /from ['"]@\//.test(l));
      const hasAbsolute = imports.some(l => /from ['"]@\//.test(l) || /from ['"]~\//.test(l));
      if (hasBarrel) patterns.push({ category: 'imports', observation: 'barrel-imports', file: filePath });
      if (hasAbsolute) patterns.push({ category: 'imports', observation: 'alias-imports', file: filePath });
    }

    // Detect naming convention
    const funcDefs = lines.filter(l => /^(export )?(const|function|async function) /.test(l.trim()));
    const hasCamel = funcDefs.some(l => /\b[a-z][a-zA-Z]+\b/.test(l));
    const hasSnake = funcDefs.some(l => /\b[a-z]+_[a-z]+\b/.test(l));
    if (hasCamel && !hasSnake) patterns.push({ category: 'naming', observation: 'camelCase-functions', file: filePath });
    if (hasSnake && !hasCamel) patterns.push({ category: 'naming', observation: 'snake_case-functions', file: filePath });

    // Detect test patterns
    if (filePath.includes('test') || filePath.includes('spec')) {
      const hasDescribe = content.includes('describe(');
      const hasTest = content.includes('test(') || content.includes('it(');
      const hasExpect = content.includes('expect(');
      const hasAssert = content.includes('assert.');
      if (hasDescribe && hasTest) patterns.push({ category: 'tests', observation: 'describe-it-pattern', file: filePath });
      if (hasExpect) patterns.push({ category: 'tests', observation: 'expect-assertions', file: filePath });
      if (hasAssert) patterns.push({ category: 'tests', observation: 'assert-assertions', file: filePath });
    }
  }

  // Detect Vue/React patterns
  if (/\.vue$/.test(ext)) patterns.push({ category: 'framework', observation: 'vue-sfc', file: filePath });
  if (/\.(tsx|jsx)$/.test(ext) && content && content.includes('React')) {
    patterns.push({ category: 'framework', observation: 'react-component', file: filePath });
  }

  return patterns;
}

function recordInstinct(mintDir, pattern) {
  const instinctsPath = path.join(mintDir, 'instincts.md');
  let existing = '';
  try { existing = fs.readFileSync(instinctsPath, 'utf8'); } catch { /* new file */ }

  // Check if this pattern already exists — increment confidence if so
  const key = `${pattern.category}:${pattern.observation}`;
  if (existing.includes(key)) {
    // Pattern already recorded — don't duplicate
    return;
  }

  // Append new observation
  const entry = `| ${pattern.category} | ${pattern.observation} | ${path.basename(pattern.file)} | 1 | ${new Date().toISOString().split('T')[0]} |\n`;

  if (!existing) {
    // Create the file with header
    const header = `# Instincts

Auto-extracted patterns from agent observations. The planner reads this file to match
existing project conventions when writing new code.

Confidence increases when the same pattern is observed across multiple files.
Patterns with confidence >= 3 are high-confidence and should be followed by default.

## Observations

| Category | Pattern | Last Seen In | Confidence | Date |
|----------|---------|-------------|------------|------|
`;
    fs.writeFileSync(instinctsPath, header + entry);
  } else {
    fs.appendFileSync(instinctsPath, entry);
  }
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
    } catch { /* no config or parse error — proceed with defaults */ }

    // Read the file content for pattern extraction
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
