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

const HEADER = `# Instincts

Auto-extracted patterns from code observations. The planner reads this to match
existing project conventions. Confidence grows as patterns repeat across files.

**High-confidence (>= 3):** Follow by default when writing new code.
**Low-confidence (1-2):** Observed but not yet established — use judgment.

| Category | Pattern | Files Seen | Confidence | Last Updated |
|----------|---------|-----------|------------|--------------|
`;

function recordInstinct(mintDir, pattern) {
  const instinctsPath = path.join(mintDir, 'instincts.md');
  let existing = '';
  try { existing = fs.readFileSync(instinctsPath, 'utf8'); } catch { /* new file */ }

  const key = `| ${pattern.category} | ${pattern.observation} |`;
  const fileName = path.basename(pattern.file);
  const today = new Date().toISOString().split('T')[0];

  if (existing.includes(key)) {
    // Pattern exists — increment confidence and update files seen
    const lines = existing.split('\n');
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].startsWith(key)) {
        const cols = lines[i].split('|').map(c => c.trim());
        // cols: ['', category, observation, filesSeen, confidence, date, '']
        const seenFiles = cols[3] || '';
        const confidence = parseInt(cols[4]) || 1;

        // Only bump confidence if this is a different file
        if (!seenFiles.includes(fileName)) {
          const newSeen = seenFiles ? `${seenFiles}, ${fileName}` : fileName;
          // Cap the files list to avoid mega-long rows
          const truncatedSeen = newSeen.length > 60 ? newSeen.substring(0, 57) + '...' : newSeen;
          cols[3] = truncatedSeen;
          cols[4] = String(confidence + 1);
          cols[5] = today;
          lines[i] = `| ${cols[1]} | ${cols[2]} | ${cols[3]} | ${cols[4]} | ${cols[5]} |`;
        }
        break;
      }
    }
    fs.writeFileSync(instinctsPath, lines.join('\n'));
  } else if (!existing) {
    // New file
    const entry = `| ${pattern.category} | ${pattern.observation} | ${fileName} | 1 | ${today} |\n`;
    fs.writeFileSync(instinctsPath, HEADER + entry);
  } else {
    // Append new pattern
    const entry = `| ${pattern.category} | ${pattern.observation} | ${fileName} | 1 | ${today} |\n`;
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
