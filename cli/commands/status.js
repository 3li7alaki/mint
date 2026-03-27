import fs from 'fs';
import path from 'path';
import { readJsonSafe, fileExists, loadGlobalConfig } from '../lib/detect.js';
import { readJsonl } from '../lib/jsonl.js';
import { listWorktrees } from '../lib/worktree.js';
import { listSessions } from '../lib/session.js';

export async function run(args = [], flags = {}) {
  const cwd = process.cwd();
  const mintDir = path.join(cwd, '.mint');
  const configPath = path.join(mintDir, 'config.json');

  const config = readJsonSafe(configPath);
  if (!config) {
    console.log('\n  \x1b[31mNo mint config found.\x1b[0m Run \x1b[36mmint init\x1b[0m\n');
    return;
  }

  // Version
  let version = 'unknown';
  try {
    const home = process.env.HOME || process.env.USERPROFILE || '';
    const pkgPaths = [
      path.join(home, '.mint', 'package.json'),
      path.join(home, '.claude', 'plugins', 'marketplaces', 'mint', 'package.json'),
    ];
    for (const p of pkgPaths) {
      if (fs.existsSync(p)) {
        version = JSON.parse(fs.readFileSync(p, 'utf8')).version;
        break;
      }
    }
  } catch { /* ignore */ }

  const c = (v) => `\x1b[36m${v}\x1b[0m`;
  const g = (v) => `\x1b[32m✓\x1b[0m ${v}`;
  const r = (v) => `\x1b[31m✗\x1b[0m ${v}`;
  const d = (v) => `\x1b[2m${v}\x1b[0m`;

  console.log(`\n  \x1b[1mmint v${version}\x1b[0m — ${c(config.stack || 'unknown')} ${d(`(${config.packageManager || 'unknown'})`)}\n`);

  // Config status
  const globalConfig = loadGlobalConfig();
  const hasGlobal = Object.keys(globalConfig).length > 0;
  console.log(`  Config:     ${g('valid')}${hasGlobal ? ` ${d('(global defaults active)')}` : ''}`);

  // Gates
  const gates = config.gates || {};
  const gateStatus = ['lint', 'types', 'tests'].map(g => {
    const cmd = gates[g];
    return cmd && cmd !== false ? `\x1b[32m${g}\x1b[0m` : `\x1b[2m${g}\x1b[0m`;
  }).join('  ');
  console.log(`  Gates:      ${gateStatus}`);

  // Features
  console.log(`  Browser:    ${config.browser?.enabled ? g('on') : d('off')}`);
  console.log(`  Context:    ${config.context?.enabled ? g('on') : d('off')}`);
  console.log(`  Design:     ${config.design?.enabled ? g('on') : d('off')}`);
  console.log(`  Isolation:  ${c(config.isolation?.plan || 'none')}`);
  console.log(`  Repo mode:  ${c(config.repoMode || 'collaborative')}`);

  // Reviewers
  const reviewers = config.reviewers || {};
  const enabledReviewers = Object.entries(reviewers)
    .filter(([, v]) => v === true || (typeof v === 'object' && v.enabled))
    .map(([k]) => k);
  console.log(`  Reviewers:  ${enabledReviewers.length ? enabledReviewers.join(', ') : d('none')}`);

  // Plugins
  const plugins = (config.plugins || []).map(p => path.basename(p));
  console.log(`  Plugins:    ${plugins.length ? plugins.join(', ') : d('none')}`);

  // Learning stats
  const issueCount = readJsonl(path.join(mintDir, 'issues.jsonl')).length;
  const winCount = readJsonl(path.join(mintDir, 'wins.jsonl')).length;
  const patternCount = readJsonl(path.join(mintDir, 'patterns.jsonl')).length;
  const instinctCount = readJsonl(path.join(mintDir, 'instincts.jsonl')).length;
  console.log(`  Learning:   ${d(`${issueCount} issues, ${winCount} wins, ${patternCount} patterns, ${instinctCount} instincts`)}`);

  // Worktrees
  const worktrees = listWorktrees(cwd);
  if (worktrees.length > 0) {
    console.log(`  Worktrees:  ${worktrees.length} active`);
  }

  // Sessions (per-session isolation)
  const sessions = listSessions(cwd);
  if (sessions.length > 0) {
    console.log(`  Sessions:   ${sessions.length} active`);
    for (const { id, state } of sessions) {
      const shortId = id.length > 12 ? id.slice(0, 12) + '…' : id;
      console.log(`              ${d(shortId)} ${c(state.mode)} — ${state.task || 'unknown task'}`);
    }
  }

  console.log('');
}
