/**
 * JSONL (JSON Lines) utilities for mint.
 *
 * JSONL is append-only, concurrent-safe, grep-able, streamable.
 * One JSON object per line. Append = fs.appendFileSync(file, JSON.stringify(entry) + '\n').
 * No read-parse-modify-write cycle. Two agents appending simultaneously just add two lines.
 */
import fs from 'fs';

/**
 * Append a single entry to a JSONL file.
 * Atomic for lines under ~4KB on POSIX (always true for our entries).
 * @param {string} filePath
 * @param {object} entry
 */
export function appendJsonl(filePath, entry) {
  fs.appendFileSync(filePath, JSON.stringify(entry) + '\n');
}

/**
 * Read all entries from a JSONL file.
 * Skips blank lines and invalid JSON (graceful).
 * @param {string} filePath
 * @returns {object[]}
 */
export function readJsonl(filePath) {
  try {
    const content = fs.readFileSync(filePath, 'utf8');
    return content
      .split('\n')
      .filter(line => line.trim())
      .map(line => {
        try { return JSON.parse(line); }
        catch { return null; }
      })
      .filter(Boolean);
  } catch {
    return [];
  }
}

/**
 * Read entries from a JSONL file matching a filter.
 * @param {string} filePath
 * @param {function} predicate - (entry) => boolean
 * @returns {object[]}
 */
export function queryJsonl(filePath, predicate) {
  return readJsonl(filePath).filter(predicate);
}

/**
 * Get the most recent entry matching a filter.
 * @param {string} filePath
 * @param {function} predicate
 * @returns {object|null}
 */
export function lastJsonl(filePath, predicate) {
  const entries = readJsonl(filePath);
  for (let i = entries.length - 1; i >= 0; i--) {
    if (!predicate || predicate(entries[i])) return entries[i];
  }
  return null;
}

/**
 * Migrate a markdown table file to JSONL.
 * Reads a markdown table with | delimiters, extracts headers and rows,
 * converts to JSONL entries.
 * @param {string} mdPath - path to markdown file
 * @param {string} jsonlPath - path to output JSONL file
 * @returns {{ migrated: number, skipped: number }}
 */
export function migrateMarkdownTableToJsonl(mdPath, jsonlPath) {
  try {
    const content = fs.readFileSync(mdPath, 'utf8');
    const lines = content.split('\n');

    // Find header row (first line with |)
    let headerIdx = -1;
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].trim().startsWith('|') && lines[i].includes('|')) {
        headerIdx = i;
        break;
      }
    }
    if (headerIdx === -1) return { migrated: 0, skipped: 0 };

    const headers = lines[headerIdx]
      .split('|')
      .map(h => h.trim())
      .filter(Boolean)
      .map(h => h.toLowerCase().replace(/\s+/g, '_'));

    // Skip separator row (|---|---|)
    let migrated = 0;
    let skipped = 0;

    for (let i = headerIdx + 2; i < lines.length; i++) {
      const line = lines[i].trim();
      if (!line || !line.startsWith('|')) continue;

      const cells = line.split('|').map(c => c.trim()).filter(Boolean);
      if (cells.length < headers.length) { skipped++; continue; }

      const entry = {};
      for (let j = 0; j < headers.length; j++) {
        entry[headers[j]] = cells[j] || '';
      }
      appendJsonl(jsonlPath, entry);
      migrated++;
    }

    return { migrated, skipped };
  } catch {
    return { migrated: 0, skipped: 0 };
  }
}

/**
 * Upsert an instinct — deduplicate by category + observation match.
 * If a matching instinct exists, increment confidence and occurrences.
 * If not, append a new entry with confidence 1.
 *
 * @param {string} filePath - path to instincts.jsonl
 * @param {object} instinct - { category, observation, source?, examples? }
 * @returns {{ action: 'created'|'updated', confidence: number }}
 */
export function upsertInstinct(filePath, instinct) {
  const entries = readJsonl(filePath);
  const now = new Date().toISOString();

  // Normalize for matching: lowercase, trim whitespace
  const normalize = (s) => (s || '').toLowerCase().trim().replace(/\s+/g, ' ');
  const matchCat = normalize(instinct.category);
  const matchObs = normalize(instinct.observation);

  // Find existing match (same category + similar observation)
  const existingIdx = entries.findIndex(e =>
    normalize(e.category) === matchCat &&
    normalize(e.observation) === matchObs
  );

  if (existingIdx >= 0) {
    // Update existing — rewrite the file with updated entry
    const existing = entries[existingIdx];
    existing.confidence = (existing.confidence || 1) + 1;
    existing.occurrences = (existing.occurrences || 1) + 1;
    existing.lastSeen = now;
    if (instinct.source && !existing.sources?.includes(instinct.source)) {
      existing.sources = [...(existing.sources || []), instinct.source];
    }
    if (instinct.examples) {
      existing.examples = [...new Set([...(existing.examples || []), ...instinct.examples])].slice(0, 5);
    }

    // Rewrite file (atomic enough for our use case — single-writer per session)
    const content = entries.map(e => JSON.stringify(e)).join('\n') + '\n';
    fs.writeFileSync(filePath, content);

    return { action: 'updated', confidence: existing.confidence };
  }

  // New instinct
  const entry = {
    category: instinct.category,
    observation: instinct.observation,
    confidence: 1,
    occurrences: 1,
    firstSeen: now,
    lastSeen: now,
    sources: instinct.source ? [instinct.source] : [],
    examples: (instinct.examples || []).slice(0, 5),
  };
  appendJsonl(filePath, entry);

  return { action: 'created', confidence: 1 };
}

/**
 * Decay instincts not reinforced within a given period.
 * Reduces confidence by 1 for stale instincts. Removes those at confidence 0.
 *
 * @param {string} filePath
 * @param {number} [maxAgeDays=30] - days without reinforcement before decay
 * @returns {{ decayed: number, removed: number }}
 */
export function decayInstincts(filePath, maxAgeDays = 30) {
  const entries = readJsonl(filePath);
  if (entries.length === 0) return { decayed: 0, removed: 0 };

  const cutoff = Date.now() - (maxAgeDays * 24 * 60 * 60 * 1000);
  let decayed = 0;
  let removed = 0;

  const surviving = entries.filter(e => {
    const lastSeen = new Date(e.lastSeen || e.firstSeen || 0).getTime();
    if (lastSeen < cutoff) {
      const newConf = (e.confidence || 1) - 1;
      if (newConf <= 0) { removed++; return false; }
      e.confidence = newConf;
      decayed++;
    }
    return true;
  });

  if (decayed > 0 || removed > 0) {
    const content = surviving.map(e => JSON.stringify(e)).join('\n') + '\n';
    fs.writeFileSync(filePath, content);
  }

  return { decayed, removed };
}

/**
 * Get top instincts by confidence, capped at a maximum count.
 *
 * @param {string} filePath
 * @param {number} [maxCount=20]
 * @param {number} [minConfidence=3]
 * @returns {object[]}
 */
export function topInstincts(filePath, maxCount = 20, minConfidence = 3) {
  return readJsonl(filePath)
    .filter(e => (e.confidence || 0) >= minConfidence)
    .sort((a, b) => (b.confidence || 0) - (a.confidence || 0))
    .slice(0, maxCount);
}

/**
 * Log execution metrics for a completed spec.
 * Correlates instincts/patterns applied with outcomes for evidence-based evolution.
 *
 * @param {string} filePath - path to metrics.jsonl
 * @param {object} metric
 * @param {string} metric.date - ISO date
 * @param {string} metric.task - task slug
 * @param {string} metric.spec - spec ID
 * @param {string[]} metric.instinctsApplied - instinct observations used
 * @param {string} metric.reviewResult - 'passed' | 'failed' | 'skipped'
 * @param {object} metric.gateResults - { lint, types, tests }
 * @param {number} metric.attempts - number of attempts
 * @param {number} [metric.tokenCount] - tokens used (if available)
 */
export function logMetric(filePath, metric) {
  appendJsonl(filePath, {
    date: metric.date || new Date().toISOString(),
    task: metric.task,
    spec: metric.spec,
    instinctsApplied: metric.instinctsApplied || [],
    reviewResult: metric.reviewResult || 'skipped',
    gateResults: metric.gateResults || {},
    attempts: metric.attempts || 1,
    tokenCount: metric.tokenCount || null,
  });
}

/**
 * Analyze instinct effectiveness from metrics.
 * Returns instincts sorted by success rate (specs that applied them and passed review).
 *
 * @param {string} metricsPath
 * @param {number} [minSamples=3] - minimum uses before reporting
 * @returns {Array<{ observation: string, uses: number, passRate: number }>}
 */
export function analyzeInstinctEffectiveness(metricsPath) {
  const metrics = readJsonl(metricsPath);
  const stats = {};

  for (const m of metrics) {
    for (const inst of (m.instinctsApplied || [])) {
      if (!stats[inst]) stats[inst] = { uses: 0, passes: 0 };
      stats[inst].uses++;
      if (m.reviewResult === 'passed') stats[inst].passes++;
    }
  }

  return Object.entries(stats)
    .map(([observation, s]) => ({
      observation,
      uses: s.uses,
      passRate: s.uses > 0 ? Math.round((s.passes / s.uses) * 100) : 0,
    }))
    .sort((a, b) => b.passRate - a.passRate || b.uses - a.uses);
}

/**
 * Format JSONL entries as a pretty table for CLI display.
 * @param {object[]} entries
 * @param {string[]} columns - keys to display
 * @param {object} [options]
 * @param {object} [options.widths] - { key: maxWidth }
 * @param {object} [options.labels] - { key: 'Display Label' }
 * @returns {string}
 */
export function formatTable(entries, columns, options = {}) {
  const { widths = {}, labels = {} } = options;

  // Calculate column widths
  const colWidths = {};
  for (const col of columns) {
    const label = labels[col] || col.toUpperCase();
    const maxData = entries.reduce((max, e) => Math.max(max, String(e[col] || '').length), 0);
    colWidths[col] = Math.min(widths[col] || 60, Math.max(label.length, maxData));
  }

  // Header
  const header = columns
    .map(col => (labels[col] || col.toUpperCase()).padEnd(colWidths[col]))
    .join('  ');

  // Rows
  const rows = entries.map(entry =>
    columns
      .map(col => String(entry[col] || '').slice(0, colWidths[col]).padEnd(colWidths[col]))
      .join('  ')
  );

  return `  ${header}\n  ${'─'.repeat(header.length)}\n${rows.map(r => `  ${r}`).join('\n')}`;
}

/**
 * Compute a fingerprint from a workflow trace's operations array.
 * Each operation becomes "type-action" (if action exists) or just "type".
 * Segments are joined with colons: "git-fetch:edit:gate:commit"
 *
 * @param {Array<{type: string, action?: string}>} operations
 * @returns {string}
 */
export function computeFingerprint(operations) {
  if (!operations || operations.length === 0) return '';
  return operations
    .map(op => op.action ? `${op.type}-${op.action}` : op.type)
    .join(':');
}

/**
 * Levenshtein edit distance on colon-separated fingerprint segments.
 * Operates on segments (not characters) — split by ":" first.
 *
 * @param {string} fp1
 * @param {string} fp2
 * @returns {number}
 */
export function fingerprintDistance(fp1, fp2) {
  const a = fp1 ? fp1.split(':') : [];
  const b = fp2 ? fp2.split(':') : [];

  const m = a.length;
  const n = b.length;

  // Build distance matrix
  const dp = Array.from({ length: m + 1 }, () => new Array(n + 1).fill(0));
  for (let i = 0; i <= m; i++) dp[i][0] = i;
  for (let j = 0; j <= n; j++) dp[0][j] = j;

  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1;
      dp[i][j] = Math.min(
        dp[i - 1][j] + 1,       // deletion
        dp[i][j - 1] + 1,       // insertion
        dp[i - 1][j - 1] + cost  // substitution
      );
    }
  }

  return dp[m][n];
}

/**
 * Cluster traces by fingerprint similarity.
 * Groups traces where fingerprintDistance <= maxDistance.
 * Uses simple single-linkage: a trace joins a cluster if it matches any member.
 *
 * @param {Array<{fingerprint: string}>} traces
 * @param {number} [maxDistance=2]
 * @returns {Array<Array<object>>} array of clusters
 */
export function clusterByFingerprint(traces, maxDistance = 2) {
  const clusters = [];

  for (const trace of traces) {
    let merged = false;
    for (const cluster of clusters) {
      const matches = cluster.some(
        member => fingerprintDistance(member.fingerprint, trace.fingerprint) <= maxDistance
      );
      if (matches) {
        cluster.push(trace);
        merged = true;
        break;
      }
    }
    if (!merged) {
      clusters.push([trace]);
    }
  }

  return clusters;
}

/**
 * Upsert a workflow candidate — deduplicate by fingerprint similarity.
 * If a candidate with fingerprintDistance <= 1 exists, increment confidence/occurrences
 * and update lastSeen. Otherwise create a new entry.
 *
 * @param {string} filePath - path to workflow-candidates.jsonl
 * @param {object} candidate - { fingerprint, variableSlots?, exampleTraces?, proposedName?, trustLevel? }
 * @returns {{ action: 'created'|'updated', confidence: number }}
 */
export function upsertWorkflowCandidate(filePath, candidate) {
  const entries = readJsonl(filePath);
  const now = new Date().toISOString();

  // Find existing match by fingerprint similarity
  const existingIdx = entries.findIndex(
    e => fingerprintDistance(e.fingerprint, candidate.fingerprint) <= 1
  );

  if (existingIdx >= 0) {
    const existing = entries[existingIdx];
    existing.confidence = (existing.confidence || 1) + 1;
    existing.occurrences = (existing.occurrences || 1) + 1;
    existing.lastSeen = now;

    // Merge example traces (deduplicated, capped)
    if (candidate.exampleTraces) {
      existing.exampleTraces = [
        ...new Set([...(existing.exampleTraces || []), ...candidate.exampleTraces])
      ].slice(0, 10);
    }

    const content = entries.map(e => JSON.stringify(e)).join('\n') + '\n';
    fs.writeFileSync(filePath, content);

    return { action: 'updated', confidence: existing.confidence };
  }

  // New candidate
  const entry = {
    fingerprint: candidate.fingerprint,
    occurrences: 1,
    confidence: 1,
    variableSlots: candidate.variableSlots || [],
    exampleTraces: (candidate.exampleTraces || []).slice(0, 10),
    proposedName: candidate.proposedName || '',
    trustLevel: candidate.trustLevel || 0,
    firstSeen: now,
    lastSeen: now,
  };
  appendJsonl(filePath, entry);

  return { action: 'created', confidence: 1 };
}

/**
 * Decay workflow candidates not reinforced within a given period.
 * Reduces confidence by 1 for stale candidates. Removes those at confidence 0.
 *
 * @param {string} filePath
 * @param {number} [maxAgeDays=30]
 * @returns {{ decayed: number, removed: number }}
 */
export function decayWorkflowCandidates(filePath, maxAgeDays = 30) {
  const entries = readJsonl(filePath);
  if (entries.length === 0) return { decayed: 0, removed: 0 };

  const cutoff = Date.now() - (maxAgeDays * 24 * 60 * 60 * 1000);
  let decayed = 0;
  let removed = 0;

  const surviving = entries.filter(e => {
    const lastSeen = new Date(e.lastSeen || e.firstSeen || 0).getTime();
    if (lastSeen < cutoff) {
      const newConf = (e.confidence || 1) - 1;
      if (newConf <= 0) { removed++; return false; }
      e.confidence = newConf;
      decayed++;
    }
    return true;
  });

  if (decayed > 0 || removed > 0) {
    const content = surviving.map(e => JSON.stringify(e)).join('\n') + '\n';
    fs.writeFileSync(filePath, content);
  }

  return { decayed, removed };
}
