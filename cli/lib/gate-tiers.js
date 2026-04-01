/**
 * Gate tier classification — determines which gates to run based on changed files.
 *
 * Three tiers:
 *   skip  — docs, comments, non-code assets → no gates
 *   quick — new test files, styles, configs → types only
 *   full  — source code, existing test mods → all gates (lint + types + tests)
 *
 * Classification uses glob patterns from config. Defaults are sensible.
 * The planner/orchestrator checks this BEFORE running gates.
 */
import path from 'path';

/** @typedef {'skip'|'quick'|'full'} GateTier */

/**
 * Default patterns for each tier. Projects can override via config.gates.tiers.
 * Patterns use minimatch-style globs matched against file paths.
 */
export const DEFAULT_TIER_PATTERNS = {
  skip: [
    '*.md',
    '*.txt',
    '*.mdx',
    'LICENSE*',
    'CHANGELOG*',
    '.gitignore',
    '.gitattributes',
    '*.png',
    '*.jpg',
    '*.jpeg',
    '*.gif',
    '*.svg',
    '*.ico',
    '*.woff',
    '*.woff2',
    '*.ttf',
    '*.eot',
    'docs/**',
    '.mint/**',
  ],
  quick: [
    '*.test.ts:new',
    '*.test.js:new',
    '*.test.tsx:new',
    '*.test.jsx:new',
    '*.spec.ts:new',
    '*.spec.js:new',
    '*.css',
    '*.scss',
    '*.less',
    '*.pcss',
    '*.module.css',
    '*.module.scss',
    '*.tailwind.*',
  ],
  full: [
    'src/**',
    'lib/**',
    'app/**',
    'server/**',
    'api/**',
    'cli/**',
    'packages/**',
    '*.config.*',
    'package.json',
    'tsconfig*.json',
  ],
};

/**
 * Simple glob matcher supporting *, **, and basic extensions.
 * Not a full minimatch — covers the patterns we actually use.
 *
 * @param {string} pattern - glob pattern
 * @param {string} filePath - file path to test
 * @returns {boolean}
 */
function matchGlob(pattern, filePath) {
  // Handle :new suffix (new files only) — stripped before matching
  const cleanPattern = pattern.replace(/:new$/, '');

  // Convert glob to regex
  let regex = cleanPattern
    .replace(/\./g, '\\.')
    .replace(/\*\*\//g, '(.+/)?')
    .replace(/\*\*/g, '.*')
    .replace(/\*/g, '[^/]*');

  // If pattern has no directory separator, match against basename only
  if (!cleanPattern.includes('/')) {
    const basename = path.basename(filePath);
    return new RegExp(`^${regex}$`).test(basename);
  }

  return new RegExp(`^${regex}$`).test(filePath);
}

/**
 * Classify a list of changed files into a gate tier.
 *
 * The highest tier wins: if ANY file matches 'full', the tier is 'full'.
 * If no file matches 'full' but some match 'quick', tier is 'quick'.
 * If all files match 'skip', tier is 'skip'.
 * Unmatched files default to 'full' (safe fallback).
 *
 * @param {Array<{path: string, status?: 'new'|'modified'|'deleted'}>} changedFiles
 * @param {object} [tierConfig] - override patterns from config.gates.tiers
 * @returns {{ tier: GateTier, reason: string, breakdown: object }}
 */
export function classifyGateTier(changedFiles, tierConfig = {}) {
  const patterns = {
    skip: tierConfig.skip || DEFAULT_TIER_PATTERNS.skip,
    quick: tierConfig.quick || DEFAULT_TIER_PATTERNS.quick,
    full: tierConfig.full || DEFAULT_TIER_PATTERNS.full,
  };

  const breakdown = { skip: [], quick: [], full: [] };

  for (const file of changedFiles) {
    const filePath = file.path || file;
    const status = file.status || 'modified';
    let matched = false;

    // Check skip patterns first
    for (const pat of patterns.skip) {
      if (matchGlob(pat, filePath)) {
        breakdown.skip.push(filePath);
        matched = true;
        break;
      }
    }
    if (matched) continue;

    // Check quick patterns
    for (const pat of patterns.quick) {
      const isNewOnly = pat.endsWith(':new');
      if (isNewOnly && status !== 'new') continue;
      if (matchGlob(pat, filePath)) {
        breakdown.quick.push(filePath);
        matched = true;
        break;
      }
    }
    if (matched) continue;

    // Check full patterns (or default to full for unmatched)
    breakdown.full.push(filePath);
  }

  // Highest tier wins
  if (breakdown.full.length > 0) {
    return {
      tier: 'full',
      reason: `${breakdown.full.length} file(s) require full gates: ${breakdown.full.slice(0, 3).join(', ')}${breakdown.full.length > 3 ? '...' : ''}`,
      breakdown,
    };
  }

  if (breakdown.quick.length > 0) {
    return {
      tier: 'quick',
      reason: `${breakdown.quick.length} file(s) need quick gates (types only): ${breakdown.quick.slice(0, 3).join(', ')}${breakdown.quick.length > 3 ? '...' : ''}`,
      breakdown,
    };
  }

  if (breakdown.skip.length > 0) {
    return {
      tier: 'skip',
      reason: `all ${breakdown.skip.length} file(s) are docs/assets — gates skipped`,
      breakdown,
    };
  }

  // No files changed — shouldn't happen, but full is safe
  return {
    tier: 'full',
    reason: 'no files classified — running full gates as fallback',
    breakdown,
  };
}

/**
 * Get the gates to run for a given tier.
 *
 * @param {GateTier} tier
 * @param {object} gatesConfig - config.gates (lint, types, tests, coverage)
 * @returns {object} - { lint: cmd|false, types: cmd|false, tests: cmd|false }
 */
export function gatesForTier(tier, gatesConfig = {}) {
  switch (tier) {
    case 'skip':
      return { lint: false, types: false, tests: false };
    case 'quick':
      return { lint: false, types: gatesConfig.types || false, tests: false };
    case 'full':
      return {
        lint: gatesConfig.lint || false,
        types: gatesConfig.types || false,
        tests: gatesConfig.tests || false,
      };
    default:
      return {
        lint: gatesConfig.lint || false,
        types: gatesConfig.types || false,
        tests: gatesConfig.tests || false,
      };
  }
}
