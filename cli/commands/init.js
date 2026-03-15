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
  const headless = flags.yes || !process.stdin.isTTY;

  // Load existing config for defaults
  let existing = null;
  if (fileExists(configPath)) {
    existing = readJsonSafe(configPath);
  }
  const defaults = existing || {};

  // ─── Auto-detect silently ──────────────────────────────────────────────────

  const stack = flags.stack || detectStack(cwd);
  const packageManager = flags.pm || detectPackageManager(cwd);
  const detectedGates = detectGates(stack, packageManager, cwd);
  const suggestedPlugins = detectPlugins(stack);

  // ─── Headless mode — no prompts ────────────────────────────────────────────

  if (headless) {
    const isoMode = flags.isolation || 'none';
    const autoCommit = flags.autocommit !== 'false';
    const tdd = flags.tdd === 'true';
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

  // ─── Interactive mode — clack prompts ──────────────────────────────────────

  p.intro('\x1b[32m mint \x1b[0m — project setup');

  // Show what we detected
  if (stack !== 'none' || packageManager !== 'none') {
    p.log.info(
      `Detected: ${stack !== 'none' ? `\x1b[36m${stack}\x1b[0m` : ''}` +
      `${stack !== 'none' && packageManager !== 'none' ? ' + ' : ''}` +
      `${packageManager !== 'none' ? `\x1b[36m${packageManager}\x1b[0m` : ''}`
    );
  }

  if (existing) {
    const proceed = await p.confirm({
      message: 'Config exists. Reconfigure?',
      initialValue: true,
    });
    if (p.isCancel(proceed) || !proceed) {
      p.outro('Keeping existing config.');
      return;
    }
  }

  // ─── Only the questions that matter ────────────────────────────────────────

  const answers = await p.group({
    isolation: () => p.select({
      message: 'Work isolation mode',
      options: [
        { value: 'none', label: 'None', hint: 'work on current branch — recommended' },
        { value: 'branch', label: 'Branch', hint: 'create feature branches per task' },
        { value: 'worktree', label: 'Worktree', hint: 'full git worktree — advanced' },
      ],
      initialValue: defaults.isolation?.plan || 'none',
    }),

    autoCommit: () => p.confirm({
      message: 'Auto-commit after passing gates?',
      initialValue: defaults.autoCommit !== undefined ? defaults.autoCommit : true,
    }),

    tdd: () => p.confirm({
      message: 'TDD mode by default?',
      initialValue: defaults.tdd?.default || false,
    }),

    browser: () => p.confirm({
      message: 'Browser support? (PinchTab — debug, scrape, test live apps)',
      initialValue: defaults.browser?.enabled ?? true,
    }),

    context: () => p.confirm({
      message: 'Context Mode? (sandboxed execution, session continuity, FTS5 search)',
      initialValue: defaults.context?.enabled ?? false,
    }),

    design: () => p.confirm({
      message: 'Design & UI/UX? (design profiling, typography/color/motion expertise, anti-pattern detection, RTL, i18n, accessibility review)',
      initialValue: defaults.design?.enabled ?? true,
    }),

    plugins: () => p.multiselect({
      message: 'Plugins',
      options: [
        { value: 'mint-e2e', label: 'E2E Testing', hint: 'Playwright patterns and runner' },
        { value: 'mint-linear', label: 'Linear', hint: 'ticket context + status sync' },
        { value: 'mint-figma', label: 'Figma', hint: 'design tokens and specs' },
        { value: 'mint-nuxt', label: 'Nuxt', hint: 'Nuxt.js conventions' },
        { value: 'mint-shadcn', label: 'shadcn/ui', hint: 'component management' },
        { value: 'mint-ssh', label: 'SSH', hint: 'remote server access' },
        { value: 'mint-gws', label: 'Google Workspace', hint: 'Sheets, Gmail, Calendar' },
      ],
      initialValues: [
        ...(defaults.plugins || []).map(p => path.basename(p)),
        ...suggestedPlugins,
      ],
      required: false,
    }),
  }, {
    onCancel: () => {
      p.cancel('Setup cancelled.');
      process.exit(0);
    },
  });

  // ─── PinchTab install offer ────────────────────────────────────────────────

  if (answers.browser && !detectTool('pinchtab')) {
    const install = await p.confirm({
      message: 'PinchTab not found. Install now?',
      initialValue: true,
    });
    if (!p.isCancel(install) && install) {
      const s = p.spinner();
      s.start('Installing PinchTab...');
      try {
        execSync('curl -fsSL https://pinchtab.com/install.sh | sh', { stdio: 'pipe' });
        s.stop('PinchTab installed');
      } catch {
        s.stop('PinchTab install failed — install manually later');
      }
    }
  }

  // ─── Impeccable install offer ────────────────────────────────────────────

  if (answers.design && !detectTool('npx skills list 2>/dev/null | grep -q impeccable')) {
    const hasImpeccable = fileExists(path.join(cwd, '.claude', 'skills', 'frontend-design', 'SKILL.md'))
      || fileExists(path.join(process.env.HOME, '.claude', 'skills', 'frontend-design', 'SKILL.md'));
    if (!hasImpeccable) {
      const install = await p.confirm({
        message: 'Install Impeccable? (design steering commands — optional, design works without it)',
        initialValue: false,
      });
      if (!p.isCancel(install) && install) {
        const s = p.spinner();
        s.start('Installing Impeccable...');
        try {
          execSync('npx skills add pbakaus/impeccable', { stdio: 'pipe', timeout: 60000 });
          s.stop('Impeccable installed');
        } catch {
          s.stop('Impeccable install failed — install manually: npx skills add pbakaus/impeccable');
        }
      }
    }
  }

  // ─── Context Mode install offer ──────────────────────────────────────────

  if (answers.context && !detectContextMode()) {
    const install = await p.confirm({
      message: 'context-mode not found. Install now?',
      initialValue: true,
    });
    if (!p.isCancel(install) && install) {
      const s = p.spinner();
      s.start('Installing context-mode...');
      try {
        installContextMode();
        s.stop('context-mode installed');
      } catch {
        s.stop('context-mode install failed — install manually: claude mcp add context-mode -- npx -y context-mode');
      }
    }
  }

  // ─── Build and write config ────────────────────────────────────────────────

  const config = buildConfig({
    stack, packageManager, gates: detectedGates,
    isolation: answers.isolation,
    autoCommit: answers.autoCommit,
    tdd: answers.tdd,
    browser: answers.browser,
    context: answers.context,
    design: answers.design,
    plugins: answers.plugins,
    defaults,
  });

  writeFiles(mintDir, configPath, config);

  p.outro(`\x1b[32mmint configured!\x1b[0m Run \x1b[36mmint doctor\x1b[0m to verify.`);
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
    documenters: defaults.documenters || [],
    plugins: plugins.map(name => `plugins/${name}`),
  };
}

function writeFiles(mintDir, configPath, config) {
  if (!fs.existsSync(mintDir)) {
    fs.mkdirSync(mintDir, { recursive: true });
  }

  fs.writeFileSync(configPath, JSON.stringify(config, null, 2) + '\n');

  const files = {
    'hard-blocks.md': HARD_BLOCKS_TEMPLATE,
    'issues.md': '# Mint Issues & Learnings\n\n_Centralized log. All agent blockers, root causes, and learnings go here._\n\n| Date | Task | Severity | Issue | Root Cause | Resolution | Spec Fix |\n|------|------|----------|-------|------------|------------|----------|\n',
    'wins.md': '# Wins\n\n_Successful patterns. The planner reads this before writing specs._\n\n| Date | Task | Pattern | Why It Worked |\n|------|------|---------|---------------|\n',
  };

  for (const [name, content] of Object.entries(files)) {
    const filePath = path.join(mintDir, name);
    if (!fileExists(filePath)) {
      fs.writeFileSync(filePath, content);
    }
  }

  // Add .mint state files to .gitignore
  const gitignorePath = path.join(path.dirname(mintDir), '.gitignore');
  const marker = '# mint local state';
  const mintIgnore = `\n${marker}\n.mint/tasks/\n.mint/research/\n.mint/worktrees/\n.mint/plugins/\n.mint/ssh-cache.json\n.mint/.session-state.json\n`;

  let gitignore = '';
  try { gitignore = fs.readFileSync(gitignorePath, 'utf8'); } catch { /* no .gitignore yet */ }

  if (!gitignore.includes(marker)) {
    fs.appendFileSync(gitignorePath, mintIgnore);
  }

  // Ensure CLAUDE.md has mint section
  ensureClaudeMd(path.dirname(mintDir));
}
