import * as p from '@clack/prompts';
import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';
import {
  detectStack,
  detectPackageManager,
  detectGates,
  detectPlugins,
  detectTool,
  detectContextMode,
  installContextMode,
  fileExists,
  readJsonSafe,
  ensureClaudeMd,
  generateDocManifest,
  loadGlobalConfig,
  registerProject,
} from '../lib/detect.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const VERSION = JSON.parse(fs.readFileSync(join(__dirname, '..', '..', 'package.json'), 'utf8')).version;

const HARD_BLOCKS_TEMPLATE = `# Hard Blocks — What Agents Can NEVER Do

## Universal
- NEVER \`git push\` — human reviews and pushes manually
- NEVER modify files outside declared task scope
- NEVER fix bad output directly — reset and fix the spec
- NEVER continue after 2 failures on the same spec

## Context Protection
- NEVER read large files in the main orchestrator context
- Subagents return summaries only

## Project-Specific
- Add project-specific constraints here
`;

export async function run(flags = {}) {
  const cwd = process.cwd();
  const mintDir = path.join(cwd, '.mint');
  const configPath = path.join(mintDir, 'config.json');
  // Never interactive — Claude handles all decisions. --yes is for CI without Claude.
  const headless = flags.yes === true;

  // Smart init — Claude reads the project and configures mint perfectly.
  // Always runs when Claude is available. Only --yes skips it (CI/scripts).
  if (detectTool('claude') && !flags.yes) {
    p.intro('\x1b[32m mint init --smart \x1b[0m');
    const s = p.spinner();
    s.start('Analyzing project with Claude...');
    try {
      const { smartInit } = await import('../lib/smart-session.js');
      const result = await smartInit(cwd);
      s.stop(result.success ? 'Setup complete' : 'Setup finished with issues');
      if (result.result) {
        console.log('');
        for (const line of result.result.split('\n')) {
          if (line.trim()) console.log(`  ${line}`);
        }
        console.log('');
      }
    } catch (err) {
      s.stop(`Smart init failed: ${err.message}`);
      p.log.info('Falling back to standard init...');
      // Fall through to standard init
    }
    return;
  }

  // Load existing config for defaults, with global config as base layer
  const globalConfig = loadGlobalConfig();
  let existing = null;
  if (fileExists(configPath)) {
    existing = readJsonSafe(configPath);
  }
  const defaults = existing || globalConfig || {};

  // ─── Auto-detect silently ──────────────────────────────────────────────────

  const stack = flags.stack || detectStack(cwd);
  const packageManager = flags.pm || detectPackageManager(cwd);
  const detectedGates = detectGates(stack, packageManager, cwd);
  const suggestedPlugins = detectPlugins(stack);

  // ─── Headless mode — no prompts ────────────────────────────────────────────

  if (headless) {
    const isoMode = flags.isolation || defaults.isolation?.plan || 'none';
    const autoCommit = flags.autocommit !== undefined ? flags.autocommit !== 'false' : (defaults.autoCommit !== undefined ? defaults.autoCommit : true);
    const tdd = flags.tdd !== undefined ? flags.tdd === 'true' : (defaults.tdd?.default || false);
    const browser = flags.browser !== 'false';
    const context = flags.context === 'true' ? true : (flags.context === 'false' ? false : detectContextMode());
    const design = flags.design !== 'false';
    const pluginList = flags.plugins
      ? flags.plugins.split(',').map(s => s.trim())
      : suggestedPlugins;

    // Auto-install context-mode in headless mode if not detected but enabled
    if (context && !detectContextMode()) {
      try { installContextMode(); } catch { /* graceful degradation */ }
    }

    const config = buildConfig({
      stack, packageManager, gates: detectedGates,
      isolation: isoMode, autoCommit, tdd, browser, context, design,
      plugins: pluginList, defaults,
    });

    writeFiles(mintDir, configPath, config);

    console.log(`\n  \x1b[32m✓\x1b[0m mint configured (v${VERSION})`);
    console.log(`    Stack: ${stack} · PM: ${packageManager} · Isolation: ${isoMode}`);
    console.log(`    Plugins: ${pluginList.length ? pluginList.join(', ') : 'none'}\n`);
    return;
  }

  // ─── Fallback: no Claude, no --yes — use headless auto-detect anyway ──────
  // Never show interactive prompts. If we got here, Claude wasn't available and
  // --yes wasn't passed. Just do headless auto-detect.
  {
    const isoMode = defaults.isolation?.plan || 'none';
    const autoCommit = defaults.autoCommit !== undefined ? defaults.autoCommit : true;
    const tdd = defaults.tdd?.default || false;
    const browser = true;
    const context = detectContextMode();
    const design = true;
    const pluginList = suggestedPlugins;

    const config = buildConfig({
      stack, packageManager, gates: detectedGates,
      isolation: isoMode, autoCommit, tdd, browser, context, design,
      plugins: pluginList, defaults,
    });

    writeFiles(mintDir, configPath, config);
    registerProject(cwd);

    console.log(`\n  \x1b[32m✓\x1b[0m mint configured (v${VERSION})`);
    console.log(`    Stack: ${stack} · PM: ${packageManager}`);
    console.log(`    No Claude CLI found — used auto-detection. Run mint update when Claude is available.\n`);
    return;
  }
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function buildConfig({ stack, packageManager, gates, isolation, autoCommit, tdd, browser, context, design, plugins, defaults }) {
  const reviewerModels = {
    spec: 'opus', quality: 'sonnet', security: 'sonnet',
    conventions: 'haiku', tests: 'sonnet', business: 'opus', performance: 'sonnet',
  };

  // Preserve existing reviewer settings or use smart defaults
  const existingReviewers = defaults.reviewers || {};
  const defaultEnabled = ['spec', 'quality', 'conventions', 'business'];
  const reviewers = {};
  for (const [key, model] of Object.entries(reviewerModels)) {
    const existing = existingReviewers[key];
    if (existing && typeof existing === 'object') {
      reviewers[key] = existing;
    } else if (existing === true || defaultEnabled.includes(key)) {
      reviewers[key] = { enabled: true, model };
    } else {
      reviewers[key] = false;
    }
  }

  return {
    stack,
    packageManager,
    gates: { lint: gates.lint || false, types: gates.types || false, tests: gates.tests || false, coverage: false },
    reviewers,
    conventions: defaults.conventions || { docs: ['docs/conventions.md', 'CONTRIBUTING.md'] },
    business: defaults.business || { docs: ['docs/architecture.md'] },
    tdd: { default: tdd, desloppify: defaults.tdd?.desloppify !== false, coverageThreshold: defaults.tdd?.coverageThreshold || 80 },
    instincts: defaults.instincts || { enabled: true },
    modelRouting: defaults.modelRouting || { enabled: true, override: {} },
    autoCommit,
    hooks: defaults.hooks || { testOnSave: false },
    isolation: { plan: isolation, ship: isolation, quick: isolation },
    definitionOfDone: defaults.definitionOfDone || {
      gatesPassing: true, specReviewPassed: true, stage2ReviewsPassed: true, screenshotReminder: 'ui-changes',
    },
    browser: {
      enabled: browser,
      ...(defaults.browser?.baseUrl ? { baseUrl: defaults.browser.baseUrl } : {}),
      ...(defaults.browser?.devServer ? { devServer: defaults.browser.devServer } : {}),
    },
    context: {
      enabled: context,
      autoRoute: defaults.context?.autoRoute ?? true,
      sandbox: defaults.context?.sandbox ?? { timeout: 30000 },
      session: defaults.context?.session ?? { enabled: true },
    },
    design: {
      enabled: design,
      stack: defaults.design?.stack ?? 'auto',
      profile: defaults.design?.profile ?? '.mint/design-profile.json',
      notes: defaults.design?.notes ?? '.mint/design-notes.md',
      conventions: defaults.design?.conventions ?? [],
      uiFilePatterns: defaults.design?.uiFilePatterns ?? ['*.tsx', '*.jsx', '*.vue', '*.svelte', '*.css', '*.scss', '*.html'],
      review: {
        accessibility: defaults.design?.review?.accessibility ?? true,
        consistency: defaults.design?.review?.consistency ?? true,
        performance: defaults.design?.review?.performance ?? true,
        rtl: defaults.design?.review?.rtl ?? false,
        i18n: defaults.design?.review?.i18n ?? false,
        brand: defaults.design?.review?.brand ?? false,
      },
    },
    signature: defaults.signature ?? false,
    documenters: defaults.documenters || [],
    plugins: plugins.map(name => `plugins/${name}`),
  };
}

function writeFiles(mintDir, configPath, config) {
  if (!fs.existsSync(mintDir)) {
    fs.mkdirSync(mintDir, { recursive: true });
  }

  fs.writeFileSync(configPath, JSON.stringify(config, null, 2) + '\n');

  // Create markdown files (hard-blocks is still markdown — it's human-authored, not machine-appended)
  const mdFiles = {
    'hard-blocks.md': HARD_BLOCKS_TEMPLATE,
  };

  for (const [name, content] of Object.entries(mdFiles)) {
    const filePath = path.join(mintDir, name);
    if (!fileExists(filePath)) {
      fs.writeFileSync(filePath, content);
    }
  }

  // Create JSONL files (machine-appended logs — concurrent-safe, grep-able)
  const jsonlFiles = ['issues.jsonl', 'wins.jsonl', 'patterns.jsonl', 'instincts.jsonl'];
  for (const name of jsonlFiles) {
    const filePath = path.join(mintDir, name);
    if (!fileExists(filePath)) {
      fs.writeFileSync(filePath, '');
    }
  }

  // Backwards compat: keep old .md files if they exist (migration happens in `mint update`)

  // Add .mint state files to .gitignore
  const gitignorePath = path.join(path.dirname(mintDir), '.gitignore');
  const marker = '# mint local state';
  const mintIgnore = `\n${marker}\n.mint/tasks/\n.mint/research/\n.mint/worktrees/\n.mint/plugins/\n.mint/ssh-cache.json\n.mint/.session-state.json\n.mint/.freeze-list.json\n.mint/.browser-sessions.json\n.mint/.gate-ledger.jsonl\n`;

  let gitignore = '';
  try { gitignore = fs.readFileSync(gitignorePath, 'utf8'); } catch { /* no .gitignore yet */ }

  if (!gitignore.includes(marker)) {
    fs.appendFileSync(gitignorePath, mintIgnore);
  }

  // Generate doc-manifest from template or scan existing docs
  const manifestPath = path.join(mintDir, 'doc-manifest.json');
  if (!fileExists(manifestPath)) {
    const manifest = generateDocManifest(path.dirname(mintDir));
    fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + '\n');
  }

  // Ensure CLAUDE.md has mint section
  ensureClaudeMd(path.dirname(mintDir));
}
