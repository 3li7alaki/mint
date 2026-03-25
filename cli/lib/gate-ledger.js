/**
 * Gate ledger — prevents redundant lint/test/type-check runs.
 *
 * Tracks when each gate ran, what state it covered, and where the work was done.
 * Any agent (or orchestrator) checks the ledger before running a gate.
 *
 * Uses JSONL for concurrent-safe append. One entry per gate run.
 */
import { appendJsonl, readJsonl } from './jsonl.js';
import path from 'path';
import fs from 'fs';

const LEDGER_FILE = '.mint/.gate-ledger.jsonl';
const STALE_MS = 10 * 60 * 1000; // 10 minutes

/**
 * Check if a gate result already covers the current state.
 *
 * @param {string} projectRoot
 * @param {string} gate - 'lint', 'types', or 'tests'
 * @param {object} options
 * @param {string|null} [options.worktree] - worktree path (null = main working dir)
 * @param {string[]} [options.specIds] - spec IDs whose work should be covered
 * @returns {{ covered: boolean, lastRun: object|null, reason: string }}
 */
export function checkGateCoverage(projectRoot, gate, options = {}) {
  const ledgerPath = path.join(projectRoot, LEDGER_FILE);
  const entries = readJsonl(ledgerPath);
  const now = Date.now();

  // Find most recent passing entry for this gate
  let lastRun = null;
  for (let i = entries.length - 1; i >= 0; i--) {
    const entry = entries[i];
    if (entry.gate !== gate) continue;
    if (entry.result === 'running') {
      // Another agent is running this gate — wait for it
      const age = now - new Date(entry.ranAt).getTime();
      if (age < 60_000) { // Less than 60s old
        return { covered: false, lastRun: entry, reason: `${gate} is currently running (started by ${entry.startedBy})` };
      }
      // Stale "running" entry — ignore it
      continue;
    }
    if (entry.result === 'pass') {
      lastRun = entry;
      break;
    }
    if (entry.result === 'fail') {
      // Last run failed — must re-run
      return { covered: false, lastRun: entry, reason: `last ${gate} run failed` };
    }
  }

  if (!lastRun) {
    return { covered: false, lastRun: null, reason: `no previous ${gate} run found` };
  }

  // Check staleness
  const runAge = now - new Date(lastRun.ranAt).getTime();
  if (runAge > STALE_MS) {
    return { covered: false, lastRun, reason: `last ${gate} run is stale (${Math.round(runAge / 60000)}m ago)` };
  }

  // Check worktree match
  const worktree = options.worktree || null;
  if (lastRun.worktree !== worktree) {
    return { covered: false, lastRun, reason: `last run was in different worktree` };
  }

  // Check if current specs are covered
  if (options.specIds && lastRun.coveredSpecs) {
    const uncovered = options.specIds.filter(id => !lastRun.coveredSpecs.includes(id));
    if (uncovered.length > 0) {
      return { covered: false, lastRun, reason: `specs not covered: ${uncovered.join(', ')}` };
    }
  }

  return { covered: true, lastRun, reason: `covered by run at ${lastRun.ranAt}` };
}

/**
 * Record a gate run starting.
 *
 * @param {string} projectRoot
 * @param {string} gate
 * @param {object} options
 * @param {string} options.startedBy - spec ID or agent name
 * @param {string[]} [options.coveredSpecs]
 * @param {string|null} [options.worktree]
 */
export function recordGateStart(projectRoot, gate, options) {
  const ledgerPath = path.join(projectRoot, LEDGER_FILE);
  appendJsonl(ledgerPath, {
    gate,
    ranAt: new Date().toISOString(),
    result: 'running',
    startedBy: options.startedBy,
    coveredSpecs: options.coveredSpecs || [],
    worktree: options.worktree || null,
  });
}

/**
 * Record a gate run completing.
 *
 * @param {string} projectRoot
 * @param {string} gate
 * @param {object} options
 * @param {'pass'|'fail'} options.result
 * @param {string[]} [options.coveredSpecs]
 * @param {string|null} [options.worktree]
 * @param {string} [options.scope] - 'full' or specific path
 */
export function recordGateResult(projectRoot, gate, options) {
  const ledgerPath = path.join(projectRoot, LEDGER_FILE);
  appendJsonl(ledgerPath, {
    gate,
    ranAt: new Date().toISOString(),
    result: options.result,
    coveredSpecs: options.coveredSpecs || [],
    worktree: options.worktree || null,
    scope: options.scope || 'full',
  });
}

/**
 * Clear the gate ledger (on session start or stop signal).
 * @param {string} projectRoot
 */
export function clearGateLedger(projectRoot) {
  const ledgerPath = path.join(projectRoot, LEDGER_FILE);
  try { fs.writeFileSync(ledgerPath, ''); } catch { /* ignore */ }
}
