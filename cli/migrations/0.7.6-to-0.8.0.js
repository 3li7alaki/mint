/**
 * Migration: 0.7.6 → 0.8.0
 *
 * - Migrate instincts to scored format (dedup, confidence, occurrences)
 * - Add mintVersion to config
 * - Add metrics.jsonl
 * - Update gitignore for new files
 */
import fs from 'fs';
import path from 'path';
import { readJsonl } from '../lib/jsonl.js';
import { ensureGitignore } from '../lib/gitignore.js';

function migrateInstincts(projectRoot) {
  const instinctsPath = path.join(projectRoot, '.mint', 'instincts.jsonl');
  if (!fs.existsSync(instinctsPath)) return;

  const entries = readJsonl(instinctsPath);
  if (entries.length === 0) return;

  // Check if already in new format (has confidence field)
  if (entries[0].confidence !== undefined) return;

  // Group old format entries by category+observation
  const groups = {};
  for (const e of entries) {
    const key = `${(e.category || '').toLowerCase().trim()}:${(e.observation || '').toLowerCase().trim()}`;
    if (!groups[key]) {
      groups[key] = { category: e.category, observation: e.observation, files: [], dates: [] };
    }
    if (e.file) groups[key].files.push(e.file);
    if (e.date) groups[key].dates.push(e.date);
  }

  // Write new format
  const now = new Date().toISOString();
  const newEntries = Object.values(groups).map(g => ({
    category: g.category,
    observation: g.observation,
    confidence: Math.min(g.files.length, 10), // cap at 10 for migration
    occurrences: g.files.length,
    firstSeen: g.dates.length > 0 ? new Date(Math.min(...g.dates.map(d => new Date(d)))).toISOString() : now,
    lastSeen: g.dates.length > 0 ? new Date(Math.max(...g.dates.map(d => new Date(d)))).toISOString() : now,
    sources: ['observer'],
    examples: [...new Set(g.files)].slice(0, 5),
  }));

  fs.writeFileSync(instinctsPath, newEntries.map(e => JSON.stringify(e)).join('\n') + '\n');
}

function ensureMetricsFile(projectRoot) {
  const metricsPath = path.join(projectRoot, '.mint', 'metrics.jsonl');
  if (!fs.existsSync(metricsPath)) {
    fs.writeFileSync(metricsPath, '');
  }
}

function addMintVersion(projectRoot, config) {
  if (!config.mintVersion) {
    config.mintVersion = '0.7.6'; // their current version before migration
  }
}

function updateGitignore(projectRoot) {
  ensureGitignore(projectRoot);
}

export default {
  from: '0.7.6',
  to: '0.8.0',
  description: 'Scored instincts, execution metrics, pipeline state machine',
  steps: [
    { name: 'Migrate instincts to scored format', fn: migrateInstincts },
    { name: 'Add metrics.jsonl', fn: ensureMetricsFile },
    { name: 'Add mintVersion to config', fn: addMintVersion },
    { name: 'Update .gitignore', fn: updateGitignore },
  ],
};
