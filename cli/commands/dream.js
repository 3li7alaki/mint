import fs from 'fs';
import path from 'path';
import { readJsonSafe } from '../lib/detect.js';
import { readJsonl, decayInstincts, topInstincts } from '../lib/jsonl.js';

export async function run(args = [], flags = {}) {
  const cwd = process.cwd();
  const mintDir = path.join(cwd, '.mint');
  const configPath = path.join(mintDir, 'config.json');

  const config = readJsonSafe(configPath);
  if (!config) {
    console.log('\n  \x1b[31mNo mint config found.\x1b[0m Run \x1b[36mmint init\x1b[0m\n');
    return;
  }

  const c = (v) => `\x1b[36m${v}\x1b[0m`;
  const g = (v) => `\x1b[32m${v}\x1b[0m`;
  const y = (v) => `\x1b[33m${v}\x1b[0m`;
  const d = (v) => `\x1b[2m${v}\x1b[0m`;

  const sub = args[0];

  if (sub === 'status') {
    return showDreamStatus(mintDir, { c, g, y, d });
  }

  if (sub === 'decay') {
    return runDecay(mintDir, flags, { c, g, y, d });
  }

  if (sub === 'instincts') {
    return showInstincts(mintDir, flags, { c, g, y, d });
  }

  // Default: show dream status + suggest running full dream via Claude
  return showDreamStatus(mintDir, { c, g, y, d }, true);
}

function showDreamStatus(mintDir, { c, g, y, d }, suggestFull = false) {
  console.log('\n  \x1b[1mDream Status\x1b[0m\n');

  // Count learning entries
  const issues = readJsonl(path.join(mintDir, 'issues.jsonl'));
  const wins = readJsonl(path.join(mintDir, 'wins.jsonl'));
  const instincts = readJsonl(path.join(mintDir, 'instincts.jsonl'));
  const metrics = readJsonl(path.join(mintDir, 'metrics.jsonl'));
  const patterns = readJsonl(path.join(mintDir, 'patterns.jsonl'));

  console.log(`  Issues:     ${c(issues.length)} ${d('entries')}`);
  console.log(`  Wins:       ${c(wins.length)} ${d('entries')}`);
  console.log(`  Instincts:  ${c(instincts.length)} ${d(`(${instincts.filter(i => (i.confidence || 0) >= 7).length} high-confidence)`)}`);
  console.log(`  Metrics:    ${c(metrics.length)} ${d('entries')}`);
  console.log(`  Patterns:   ${c(patterns.length)} ${d('promoted')}`);

  // Last dream
  const reportPath = path.join(mintDir, 'dream-report.md');
  if (fs.existsSync(reportPath)) {
    const content = fs.readFileSync(reportPath, 'utf8');
    const dateMatch = content.match(/# Dream Report — (.+)/);
    if (dateMatch) {
      const lastDream = dateMatch[1];
      const daysAgo = Math.floor((Date.now() - new Date(lastDream).getTime()) / (1000 * 60 * 60 * 24));
      const stale = daysAgo >= 7;
      console.log(`  Last dream: ${stale ? y(lastDream) : g(lastDream)} ${d(`(${daysAgo}d ago)`)}${stale ? ` ${y('— stale')}` : ''}`);
    }
  } else {
    console.log(`  Last dream: ${y('never')}`);
  }

  // Stale instincts
  const thirtyDaysAgo = Date.now() - (30 * 24 * 60 * 60 * 1000);
  const staleCount = instincts.filter(i => {
    const lastSeen = new Date(i.lastSeen || i.firstSeen || 0).getTime();
    return lastSeen < thirtyDaysAgo;
  }).length;
  if (staleCount > 0) {
    console.log(`  Stale:      ${y(`${staleCount} instincts`)} ${d('not reinforced in 30d')}`);
  }

  // Promotion candidates
  const candidates = instincts.filter(i => (i.confidence || 0) >= 7 && (i.occurrences || 0) >= 10);
  if (candidates.length > 0) {
    console.log(`  Promote:    ${g(`${candidates.length} candidates`)} ${d('ready for review')}`);
  }

  if (suggestFull) {
    const totalEntries = issues.length + wins.length + instincts.length + metrics.length;
    console.log('');
    console.log(`  ${d('Subcommands:')}`);
    console.log(`    ${c('mint dream status')}     ${d('Show this overview')}`);
    console.log(`    ${c('mint dream decay')}      ${d('Run instinct decay (30d default)')}`);
    console.log(`    ${c('mint dream instincts')}  ${d('List all instincts with scores')}`);
    console.log('');
    console.log(`  ${d('For full consolidation (issue triage, promotion, health report):')}`);
    console.log(`  ${d('Tell Claude:')} ${c('"dream"')} ${d('or')} ${c('"consolidate learning data"')}`);
  }

  console.log('');
}

function runDecay(mintDir, flags, { c, g, y, d }) {
  const instinctsPath = path.join(mintDir, 'instincts.jsonl');
  const maxAge = flags.days ? parseInt(flags.days, 10) : 30;

  console.log(`\n  Running instinct decay (${maxAge}d threshold)...\n`);

  const before = readJsonl(instinctsPath);
  const result = decayInstincts(instinctsPath, maxAge);

  console.log(`  Before:   ${c(before.length)} instincts`);
  console.log(`  Decayed:  ${result.decayed > 0 ? y(result.decayed) : d('0')}`);
  console.log(`  Removed:  ${result.removed > 0 ? y(result.removed) : d('0')}`);
  console.log(`  Active:   ${c(before.length - result.removed)}\n`);
}

function showInstincts(mintDir, flags, { c, g, y, d }) {
  const instinctsPath = path.join(mintDir, 'instincts.jsonl');
  const all = readJsonl(instinctsPath);

  if (all.length === 0) {
    console.log('\n  No instincts recorded yet.\n');
    return;
  }

  const sorted = all.sort((a, b) => (b.confidence || 0) - (a.confidence || 0));

  console.log(`\n  \x1b[1mInstincts\x1b[0m ${d(`(${all.length} total)`)}\n`);
  console.log(`  ${'CONF'.padEnd(6)}${'OCC'.padEnd(6)}${'CATEGORY'.padEnd(14)}${'OBSERVATION'.padEnd(30)}${'SOURCES'}`);
  console.log(`  ${'─'.repeat(80)}`);

  for (const inst of sorted) {
    const conf = String(inst.confidence || 0).padEnd(6);
    const occ = String(inst.occurrences || 0).padEnd(6);
    const cat = (inst.category || '').slice(0, 12).padEnd(14);
    const obs = (inst.observation || '').slice(0, 28).padEnd(30);
    const sources = (inst.sources || []).join(', ');

    const color = (inst.confidence || 0) >= 7 ? g : (inst.confidence || 0) >= 3 ? c : d;
    console.log(`  ${color(conf)}${occ}${cat}${obs}${d(sources)}`);
  }

  console.log('');
}
