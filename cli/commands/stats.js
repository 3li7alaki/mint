import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { readJsonSafe, detectCodeGraph, getCodeGraphStatus } from '../lib/detect.js';
import { readJsonl, topInstincts, analyzeInstinctEffectiveness } from '../lib/jsonl.js';

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
  const r = (v) => `\x1b[31m${v}\x1b[0m`;
  const d = (v) => `\x1b[2m${v}\x1b[0m`;
  const b = (v) => `\x1b[1m${v}\x1b[0m`;

  const metrics = readJsonl(path.join(mintDir, 'metrics.jsonl'));
  const issues = readJsonl(path.join(mintDir, 'issues.jsonl'));
  const wins = readJsonl(path.join(mintDir, 'wins.jsonl'));
  const instincts = readJsonl(path.join(mintDir, 'instincts.jsonl'));

  console.log(`\n  ${b('Pipeline Stats')}\n`);

  // ─── Pipeline Health ────────────────────────────────────────────────────────
  if (metrics.length > 0) {
    showPipelineHealth(metrics, { c, g, y, r, d });
  } else {
    console.log(`  ${d('No metrics recorded yet. Stats appear after specs execute.')}\n`);
  }

  // ─── Issue Analysis ─────────────────────────────────────────────────────────
  if (issues.length > 0) {
    showIssueAnalysis(issues, { c, g, y, r, d });
  }

  // ─── Reviewer Value ─────────────────────────────────────────────────────────
  if (issues.length > 0) {
    showReviewerValue(issues, { c, g, y, r, d });
  }

  // ─── Instinct Health ────────────────────────────────────────────────────────
  if (instincts.length > 0) {
    showInstinctHealth(instincts, metrics, mintDir, { c, g, y, r, d });
  }

  // ─── Win Patterns ───────────────────────────────────────────────────────────
  if (wins.length > 0) {
    showWinPatterns(wins, { c, g, y, d });
  }

  // ─── Code Graph ─────────────────────────────────────────────────────────────
  if (config.graph?.enabled) {
    showGraphHealth(cwd, config, { c, g, y, r, d });
  }

  // ─── Git Activity ───────────────────────────────────────────────────────────
  showGitActivity(cwd, { c, g, y, d });

  console.log('');
}

function showPipelineHealth(metrics, { c, g, y, r, d }) {
  console.log(`  ${c('Pipeline Health')}`);

  const total = metrics.length;

  // Gate pass rate
  const gatesPassed = metrics.filter(m => {
    const gates = m.gateResults || {};
    return Object.values(gates).every(v => v === 'pass');
  }).length;
  const gateRate = Math.round((gatesPassed / total) * 100);

  // First-try success
  const firstTry = metrics.filter(m => (m.attempts || 1) === 1).length;
  const firstTryRate = Math.round((firstTry / total) * 100);

  // Average attempts
  const avgAttempts = (metrics.reduce((s, m) => s + (m.attempts || 1), 0) / total).toFixed(1);

  // Review pass rate
  const reviewed = metrics.filter(m => m.reviewResult && m.reviewResult !== 'skipped');
  const reviewPassed = reviewed.filter(m => m.reviewResult === 'passed').length;
  const reviewRate = reviewed.length > 0 ? Math.round((reviewPassed / reviewed.length) * 100) : null;

  // Trend (compare last half vs first half)
  const mid = Math.floor(total / 2);
  let trend = '';
  if (total >= 6) {
    const firstHalf = metrics.slice(0, mid);
    const secondHalf = metrics.slice(mid);
    const firstRate = Math.round((firstHalf.filter(m => (m.attempts || 1) === 1).length / firstHalf.length) * 100);
    const secondRate = Math.round((secondHalf.filter(m => (m.attempts || 1) === 1).length / secondHalf.length) * 100);
    if (secondRate > firstRate + 5) trend = g(' ↑');
    else if (secondRate < firstRate - 5) trend = r(' ↓');
    else trend = d(' →');
  }

  const colorRate = (rate) => rate >= 80 ? g(`${rate}%`) : rate >= 60 ? y(`${rate}%`) : r(`${rate}%`);

  console.log(`    Specs executed:    ${c(total)}`);
  console.log(`    Gate pass rate:    ${colorRate(gateRate)}`);
  console.log(`    First-try success: ${colorRate(firstTryRate)}${trend}`);
  console.log(`    Avg attempts:      ${c(avgAttempts)}`);
  if (reviewRate !== null) {
    console.log(`    Review pass rate:  ${colorRate(reviewRate)}`);
  }

  // Token usage (if available)
  const tokenMetrics = metrics.filter(m => m.tokenCount);
  if (tokenMetrics.length > 0) {
    const totalTokens = tokenMetrics.reduce((s, m) => s + m.tokenCount, 0);
    const avgTokens = Math.round(totalTokens / tokenMetrics.length);
    const formatted = avgTokens > 1000 ? `${(avgTokens / 1000).toFixed(1)}k` : String(avgTokens);
    console.log(`    Avg tokens/spec:   ${c(formatted)}`);
  }

  console.log('');
}

function showIssueAnalysis(issues, { c, g, y, r, d }) {
  console.log(`  ${c('Top Issues')} ${d(`(${issues.length} total)`)}`);

  // Group by rootCause
  const groups = {};
  for (const issue of issues) {
    const cause = issue.rootCause || 'unknown';
    if (!groups[cause]) groups[cause] = { count: 0, resolved: 0 };
    groups[cause].count++;
    if (issue.status === 'resolved') groups[cause].resolved++;
  }

  // Sort by count descending, show top 5
  const sorted = Object.entries(groups)
    .sort(([, a], [, b]) => b.count - a.count)
    .slice(0, 5);

  for (const [cause, data] of sorted) {
    const status = data.resolved === data.count
      ? g('all resolved')
      : data.resolved > 0
        ? y(`${data.count - data.resolved} active`)
        : r(`${data.count} active`);
    console.log(`    ${String(data.count).padStart(3)} × ${cause.padEnd(25)} ${status}`);
  }

  console.log('');
}

function showReviewerValue(issues, { c, g, y, r, d }) {
  // Count BLOCKINGs caught per severity
  const blocking = issues.filter(i => i.severity === 'BLOCKING');
  if (blocking.length === 0) return;

  // Try to identify which reviewer caught each (from rootCause or issue text)
  const reviewerCatches = {};
  for (const issue of blocking) {
    // Heuristic: check if issue mentions a reviewer type
    const text = `${issue.issue || ''} ${issue.rootCause || ''}`.toLowerCase();
    let reviewer = 'spec';
    if (text.includes('security') || text.includes('injection') || text.includes('xss')) reviewer = 'security';
    else if (text.includes('convention') || text.includes('naming') || text.includes('import')) reviewer = 'conventions';
    else if (text.includes('mock') || text.includes('test') || text.includes('assertion')) reviewer = 'tests';
    else if (text.includes('performance') || text.includes('render') || text.includes('n+1')) reviewer = 'performance';
    else if (text.includes('quality') || text.includes('type') || text.includes('pattern')) reviewer = 'quality';
    else if (text.includes('scope') || text.includes('extra') || text.includes('yagni')) reviewer = 'spec';

    reviewerCatches[reviewer] = (reviewerCatches[reviewer] || 0) + 1;
  }

  if (Object.keys(reviewerCatches).length === 0) return;

  console.log(`  ${c('Reviewer Value')} ${d(`(${blocking.length} BLOCKINGs caught)`)}`);

  const sorted = Object.entries(reviewerCatches)
    .sort(([, a], [, b]) => b - a);

  for (const [reviewer, count] of sorted) {
    const bar = '█'.repeat(Math.min(count, 20));
    console.log(`    ${reviewer.padEnd(15)} ${String(count).padStart(3)} ${g(bar)}`);
  }

  console.log('');
}

function showInstinctHealth(instincts, metrics, mintDir, { c, g, y, r, d }) {
  console.log(`  ${c('Instinct Health')} ${d(`(${instincts.length} total)`)}`);

  const highConf = instincts.filter(i => (i.confidence || 0) >= 7);
  const medConf = instincts.filter(i => (i.confidence || 0) >= 3 && (i.confidence || 0) < 7);
  const lowConf = instincts.filter(i => (i.confidence || 0) < 3);

  const thirtyDaysAgo = Date.now() - (30 * 24 * 60 * 60 * 1000);
  const stale = instincts.filter(i => {
    const lastSeen = new Date(i.lastSeen || i.firstSeen || 0).getTime();
    return lastSeen < thirtyDaysAgo;
  });

  const promotionCandidates = instincts.filter(i =>
    (i.confidence || 0) >= 7 && (i.occurrences || 0) >= 10
  );

  console.log(`    High confidence (≥7):  ${g(highConf.length)}${promotionCandidates.length > 0 ? ` ${y(`(${promotionCandidates.length} promotion candidates)`)}` : ''}`);
  console.log(`    Active (3-6):          ${c(medConf.length)}`);
  console.log(`    Low (<3):              ${d(lowConf.length)}`);
  if (stale.length > 0) {
    console.log(`    Stale (30d+):          ${y(stale.length)} ${d('candidates for decay')}`);
  }

  // Effectiveness (if metrics exist)
  if (metrics.length > 0) {
    const effectiveness = analyzeInstinctEffectiveness(path.join(mintDir, 'metrics.jsonl'));
    const proven = effectiveness.filter(e => e.uses >= 3 && e.passRate >= 80);
    if (proven.length > 0) {
      console.log(`    Proven effective:       ${g(proven.length)} ${d('(≥3 uses, ≥80% pass rate)')}`);
    }
  }

  // Show top 3 instincts
  if (highConf.length > 0) {
    const top3 = highConf
      .sort((a, b) => (b.confidence || 0) - (a.confidence || 0))
      .slice(0, 3);
    console.log(`    ${d('Top:')}`);
    for (const inst of top3) {
      console.log(`      ${g(String(inst.confidence).padStart(3))} ${inst.category}/${inst.observation}`);
    }
  }

  console.log('');
}

function showWinPatterns(wins, { c, g, y, d }) {
  console.log(`  ${c('Win Patterns')} ${d(`(${wins.length} total)`)}`);

  // Group by pattern, show top 3
  const groups = {};
  for (const win of wins) {
    const key = win.pattern || 'unspecified';
    groups[key] = (groups[key] || 0) + 1;
  }

  const sorted = Object.entries(groups)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 3);

  for (const [pattern, count] of sorted) {
    console.log(`    ${g(String(count).padStart(3))} × ${pattern.slice(0, 60)}`);
  }

  console.log('');
}

function showGraphHealth(cwd, config, { c, g, y, r, d }) {
  if (!detectCodeGraph()) {
    console.log(`  ${c('Code Graph')} ${r('not installed')}`);
    console.log(`    ${d('Install: curl -fsSL .../install.sh | bash')}\n`);
    return;
  }

  const status = getCodeGraphStatus(cwd);
  if (!status) {
    console.log(`  ${c('Code Graph')} ${y('not indexed')}`);
    console.log(`    ${d('Run: codebase-memory-mcp cli index_repository \'{"repo_path": "."}\'')}\n`);
    return;
  }

  console.log(`  ${c('Code Graph')} ${d(`(${config.graph?.provider || 'codebase-memory-mcp'})`)}`);

  // Try to get schema for node/edge counts
  try {
    const projectName = path.basename(cwd);
    const schema = execSync(
      `codebase-memory-mcp cli get_graph_schema '{"project": "${projectName}"}'`,
      { encoding: 'utf8', stdio: 'pipe', timeout: 5000 }
    );
    const data = JSON.parse(schema);
    if (data.node_count !== undefined) {
      console.log(`    Nodes:    ${c(data.node_count)}`);
    }
    if (data.edge_count !== undefined) {
      console.log(`    Edges:    ${c(data.edge_count)}`);
    }
    if (data.languages) {
      console.log(`    Languages: ${d(data.languages.join(', '))}`);
    }
  } catch {
    console.log(`    ${g('indexed')} ${d('(run get_graph_schema for details)')}`);
  }

  console.log('');
}

function showGitActivity(cwd, { c, g, y, d }) {
  try {
    // Recent mint-related commits (last 30 days)
    const output = require('child_process').execSync(
      'git log --oneline --since="30 days ago" --no-merges 2>/dev/null',
      { cwd, encoding: 'utf8', timeout: 5000 }
    ).trim();

    if (!output) return;

    const lines = output.split('\n');
    const total = lines.length;

    // Count by type
    const types = {};
    for (const line of lines) {
      const match = line.match(/^\w+ (\w+)[\(:]?/);
      if (match) {
        const type = match[1].toLowerCase();
        types[type] = (types[type] || 0) + 1;
      }
    }

    console.log(`  ${c('Git Activity')} ${d('(last 30 days)')}`);
    console.log(`    Commits: ${c(total)}`);

    if (Object.keys(types).length > 0) {
      const sorted = Object.entries(types)
        .sort(([, a], [, b]) => b - a)
        .slice(0, 5);
      const breakdown = sorted.map(([type, count]) => `${type}:${count}`).join('  ');
      console.log(`    Types:   ${d(breakdown)}`);
    }
  } catch {
    // Not a git repo or git not available — skip
  }
}
