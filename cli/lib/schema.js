/**
 * Config schema definition for mint.
 *
 * Used for:
 * - Validation on `mint config set`
 * - `mint config list` to show all available keys
 * - Shell completions
 * - Doctor checks
 */

export const CONFIG_SCHEMA = {
  // ─── Core ──────────────────────────────────────────────────────────────────
  'stack': {
    type: 'string',
    description: 'Detected framework (nuxt, react, typescript, python, etc.)',
    scope: 'project',
  },
  'packageManager': {
    type: 'string',
    description: 'Package manager (npm, pnpm, yarn, bun)',
    scope: 'project',
  },
  'autoCommit': {
    type: 'boolean',
    default: true,
    description: 'Auto-commit after gates pass',
    scope: 'global',
  },
  'signature': {
    type: 'boolean',
    default: false,
    description: '"Minted with mint" signature in README',
    scope: 'project',
  },

  // ─── Isolation ─────────────────────────────────────────────────────────────
  'isolation.plan': {
    type: 'enum',
    values: ['none', 'branch', 'worktree'],
    default: 'none',
    description: 'Isolation mode for plan tasks',
    scope: 'global',
  },
  'isolation.ship': {
    type: 'enum',
    values: ['none', 'branch', 'worktree'],
    default: 'none',
    description: 'Isolation mode for ship tasks',
    scope: 'global',
  },
  'isolation.quick': {
    type: 'enum',
    values: ['none', 'branch', 'worktree'],
    default: 'none',
    description: 'Isolation mode for quick tasks',
    scope: 'global',
  },

  // ─── TDD ───────────────────────────────────────────────────────────────────
  'tdd.default': {
    type: 'boolean',
    default: false,
    description: 'Enable TDD by default for all specs',
    scope: 'global',
  },
  'tdd.desloppify': {
    type: 'boolean',
    default: true,
    description: 'Run de-sloppifier after TDD implementation',
    scope: 'global',
  },
  'tdd.coverageThreshold': {
    type: 'number',
    min: 0,
    max: 100,
    default: 80,
    description: 'Minimum test coverage percentage',
    scope: 'global',
  },

  // ─── Gates ─────────────────────────────────────────────────────────────────
  'gates.lint': {
    type: 'string',
    default: false,
    description: 'Lint command (e.g., "eslint .")',
    scope: 'project',
  },
  'gates.types': {
    type: 'string',
    default: false,
    description: 'Type check command (e.g., "tsc --noEmit")',
    scope: 'project',
  },
  'gates.tests': {
    type: 'string',
    default: false,
    description: 'Test command (e.g., "bun test")',
    scope: 'project',
  },
  'gates.coverage': {
    type: 'string',
    default: false,
    description: 'Coverage command',
    scope: 'project',
  },
  'gates.tiered': {
    type: 'boolean',
    default: true,
    description: 'Enable gate tier classification (skip/quick/full based on changed files)',
    scope: 'global',
  },

  // ─── Browser ───────────────────────────────────────────────────────────────
  'browser.enabled': {
    type: 'boolean',
    default: false,
    description: 'Enable PinchTab browser automation',
    scope: 'project',
  },
  'browser.baseUrl': {
    type: 'string',
    default: 'http://localhost:9867',
    description: 'PinchTab API base URL',
    scope: 'project',
  },
  'browser.devServer': {
    type: 'string',
    default: 'http://localhost:3000',
    description: 'Dev server URL for browser-reviewer',
    scope: 'project',
  },
  'browser.timeout': {
    type: 'number',
    min: 1,
    max: 300,
    default: 30,
    description: 'Per-action timeout in seconds',
    scope: 'project',
  },
  'browser.headless': {
    type: 'boolean',
    default: true,
    description: 'Run browser in headless mode',
    scope: 'project',
  },
  'browser.blockImages': {
    type: 'boolean',
    default: false,
    description: 'Block image loading for faster pages',
    scope: 'project',
  },
  'browser.persistSessions': {
    type: 'boolean',
    default: true,
    description: 'Persist login cookies between browser tasks',
    scope: 'project',
  },
  'browser.autoStart': {
    type: 'boolean',
    default: true,
    description: 'Auto-start PinchTab when needed',
    scope: 'project',
  },

  // ─── Context Mode ──────────────────────────────────────────────────────────
  'context.enabled': {
    type: 'boolean',
    default: false,
    description: 'Enable context-mode (sandboxed execution, FTS5 search)',
    scope: 'project',
  },
  'context.autoRoute': {
    type: 'boolean',
    default: true,
    description: 'Auto-route heavy operations to context-mode',
    scope: 'project',
  },

  // ─── Design ────────────────────────────────────────────────────────────────
  'design.enabled': {
    type: 'boolean',
    default: false,
    description: 'Enable design intelligence (profiling, review, anti-patterns)',
    scope: 'project',
  },
  'design.stack': {
    type: 'string',
    default: 'auto',
    description: 'UI framework for design context (auto-detected)',
    scope: 'project',
  },
  'design.review.accessibility': {
    type: 'boolean',
    default: true,
    description: 'Check WCAG 2.1 AA accessibility in design review',
    scope: 'project',
  },
  'design.review.consistency': {
    type: 'boolean',
    default: true,
    description: 'Check design token and spacing consistency',
    scope: 'project',
  },
  'design.review.performance': {
    type: 'boolean',
    default: true,
    description: 'Check animation and bundle performance',
    scope: 'project',
  },
  'design.review.rtl': {
    type: 'boolean',
    default: false,
    description: 'Check RTL layout compliance',
    scope: 'project',
  },
  'design.review.i18n': {
    type: 'boolean',
    default: false,
    description: 'Check internationalization compliance',
    scope: 'project',
  },
  'design.review.brand': {
    type: 'boolean',
    default: false,
    description: 'Check brand guide compliance',
    scope: 'project',
  },

  // ─── Reviewers ─────────────────────────────────────────────────────────────
  'reviewers.spec': {
    type: 'reviewer',
    default: true,
    description: 'Spec compliance reviewer (stage 1)',
    scope: 'global',
  },
  'reviewers.quality': {
    type: 'reviewer',
    default: true,
    description: 'Code quality reviewer (stage 2)',
    scope: 'global',
  },
  'reviewers.security': {
    type: 'reviewer',
    default: false,
    description: 'Security auditor — OWASP top 10 (stage 2)',
    scope: 'global',
  },
  'reviewers.conventions': {
    type: 'reviewer',
    default: true,
    description: 'Project conventions enforcer (stage 2)',
    scope: 'global',
  },
  'reviewers.tests': {
    type: 'reviewer',
    default: false,
    description: 'Test quality auditor (stage 2)',
    scope: 'global',
  },
  'reviewers.business': {
    type: 'reviewer',
    default: true,
    description: 'Business requirements reviewer (stage 2)',
    scope: 'global',
  },
  'reviewers.performance': {
    type: 'reviewer',
    default: false,
    description: 'Performance reviewer (stage 2)',
    scope: 'global',
  },

  // ─── Model Routing ─────────────────────────────────────────────────────────
  'modelRouting.enabled': {
    type: 'boolean',
    default: true,
    description: 'Route specs to different models by complexity',
    scope: 'global',
  },

  // ─── Instincts ─────────────────────────────────────────────────────────────
  'instincts.enabled': {
    type: 'boolean',
    default: true,
    description: 'Auto-learn project conventions from edits',
    scope: 'global',
  },

  // ─── Parallel ──────────────────────────────────────────────────────────────
  'parallel.concurrency': {
    type: 'number',
    min: 1,
    max: 10,
    default: 3,
    description: 'Max parallel Claude Code sessions',
    scope: 'global',
  },
  'parallel.maxBudgetPerSpec': {
    type: 'number',
    min: 0.1,
    max: 50,
    default: 5.0,
    description: 'Max USD budget per spec in parallel mode',
    scope: 'global',
  },

  // ─── Repo Mode ─────────────────────────────────────────────────────────────
  'repoMode': {
    type: 'enum',
    values: ['solo', 'collaborative'],
    default: 'collaborative',
    description: 'Solo: fix incidental issues proactively. Collaborative: flag and move on.',
    scope: 'project',
  },

  // ─── Challenge ─────────────────────────────────────────────────────────────
  'challenge': {
    type: 'enum',
    values: ['true', 'false', 'auto'],
    default: 'auto',
    description: 'Challenge task assumptions before planning (auto = large tasks only)',
    scope: 'global',
  },

  // ─── Review Scaling ────────────────────────────────────────────────────────
  'reviewScaling': {
    type: 'boolean',
    default: true,
    description: 'Scale review intensity by diff size (small diffs = lighter review)',
    scope: 'global',
  },

  // ─── Definition of Done ────────────────────────────────────────────────────
  'definitionOfDone.gatesPassing': {
    type: 'boolean',
    default: true,
    description: 'Require all gates passing for completion',
    scope: 'global',
  },
  'definitionOfDone.specReviewPassed': {
    type: 'boolean',
    default: true,
    description: 'Require spec review passing for completion',
    scope: 'global',
  },
  'definitionOfDone.stage2ReviewsPassed': {
    type: 'boolean',
    default: true,
    description: 'Require all stage 2 reviews passing for completion',
    scope: 'global',
  },
  'definitionOfDone.docCheckPassed': {
    type: 'boolean',
    default: true,
    description: 'Require doc-manifest check for completion',
    scope: 'global',
  },
  'definitionOfDone.screenshotReminder': {
    type: 'enum',
    values: ['ui-changes', 'always', 'false'],
    default: 'ui-changes',
    description: 'When to remind about screenshots',
    scope: 'global',
  },
};

/**
 * Validate a config key-value pair.
 * @param {string} key - dot-notation key (e.g., "isolation.plan")
 * @param {any} value - the value to validate
 * @returns {{ valid: boolean, error?: string, coerced?: any }}
 */
export function validateConfigValue(key, value) {
  const schema = CONFIG_SCHEMA[key];

  if (!schema) {
    // Check if it's a prefix of any known key
    const suggestions = Object.keys(CONFIG_SCHEMA)
      .filter(k => k.startsWith(key.split('.')[0]))
      .slice(0, 5);
    return {
      valid: false,
      error: suggestions.length
        ? `Unknown key "${key}". Did you mean: ${suggestions.join(', ')}?`
        : `Unknown key "${key}". Run "mint config list" to see available keys.`,
    };
  }

  switch (schema.type) {
    case 'boolean': {
      if (value === 'true' || value === true) return { valid: true, coerced: true };
      if (value === 'false' || value === false) return { valid: true, coerced: false };
      return { valid: false, error: `"${value}" is not valid for ${key}. Expected: true or false` };
    }
    case 'number': {
      const num = Number(value);
      if (isNaN(num)) return { valid: false, error: `"${value}" is not a number for ${key}` };
      if (schema.min !== undefined && num < schema.min) return { valid: false, error: `${key} must be >= ${schema.min}` };
      if (schema.max !== undefined && num > schema.max) return { valid: false, error: `${key} must be <= ${schema.max}` };
      return { valid: true, coerced: num };
    }
    case 'enum': {
      if (!schema.values.includes(String(value))) {
        return { valid: false, error: `"${value}" is not valid for ${key}. Allowed: ${schema.values.join(', ')}` };
      }
      return { valid: true, coerced: String(value) };
    }
    case 'string': {
      return { valid: true, coerced: String(value) };
    }
    case 'reviewer': {
      // Reviewer accepts: true, false, or object with enabled + model
      if (value === 'true' || value === true) return { valid: true, coerced: true };
      if (value === 'false' || value === false) return { valid: true, coerced: false };
      if (typeof value === 'string' && ['opus', 'sonnet', 'haiku'].includes(value)) {
        return { valid: true, coerced: { enabled: true, model: value } };
      }
      return { valid: false, error: `${key} accepts: true, false, opus, sonnet, or haiku` };
    }
    default:
      return { valid: true, coerced: value };
  }
}

/**
 * Get all config keys, optionally filtered by scope.
 * @param {'global'|'project'|null} scope
 * @returns {Array<{ key: string, type: string, default: any, description: string, scope: string }>}
 */
export function listConfigKeys(scope = null) {
  return Object.entries(CONFIG_SCHEMA)
    .filter(([, v]) => !scope || v.scope === scope)
    .map(([key, v]) => ({
      key,
      type: v.type,
      default: v.default,
      description: v.description,
      scope: v.scope,
      values: v.values || null,
    }));
}
