#!/usr/bin/env node
/**
 * mint hook: Instinct observer
 * Trigger: PostToolUse (Edit|Write)
 * Observes patterns in edited files and upserts instincts to .mint/instincts.jsonl
 *
 * Watches for naming conventions, import styles, test patterns, and framework usage.
 * Deduplicates by category+observation — confidence grows with each new observation.
 * Patterns with confidence >= 3 are treated as project conventions by the planner.
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
 * Upsert an instinct — deduplicate by category + observation.
 * If a matching entry exists, increment confidence. Otherwise create new.
 */
function recordInstinct(mintDir, pattern) {
  const instinctsPath = path.join(mintDir, 'instincts.jsonl');
  const fileName = path.basename(pattern.file);

  // Read existing entries
  let entries = [];
  try {
    entries = fs.readFileSync(instinctsPath, 'utf8')
      .split('\n').filter(l => l.trim())
      .map(l => { try { return JSON.parse(l); } catch { return null; } })
      .filter(Boolean);
  } catch { /* file doesn't exist yet */ }

  const normalize = s => (s || '').toLowerCase().trim().replace(/\s+/g, ' ');
  const matchCat = normalize(pattern.category);
  const matchObs = normalize(pattern.observation);
  const now = new Date().toISOString();

  const idx = entries.findIndex(e =>
    normalize(e.category) === matchCat && normalize(e.observation) === matchObs
  );

  if (idx >= 0) {
    // Update existing
    entries[idx].confidence = (entries[idx].confidence || 1) + 1;
    entries[idx].occurrences = (entries[idx].occurrences || 1) + 1;
    entries[idx].lastSeen = now;
    const examples = entries[idx].examples || [];
    if (!examples.includes(fileName)) {
      entries[idx].examples = [...examples, fileName].slice(0, 5);
    }
    fs.writeFileSync(instinctsPath, entries.map(e => JSON.stringify(e)).join('\n') + '\n');
  } else {
    // Append new
    const entry = {
      category: pattern.category,
      observation: pattern.observation,
      confidence: 1,
      occurrences: 1,
      firstSeen: now,
      lastSeen: now,
      sources: ['observer'],
      examples: [fileName],
    };
    fs.appendFileSync(instinctsPath, JSON.stringify(entry) + '\n');
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
