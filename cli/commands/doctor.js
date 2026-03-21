import * as p from '@clack/prompts';
import path from 'path';
import fs from 'fs';
import { execSync } from 'child_process';
import { readJsonSafe, fileExists, detectStack, detectTool, detectContextMode, getGlobalConfigPath, loadGlobalConfig, GLOBAL_KEYS } from '../lib/detect.js';

export async function run(flags = {}) {
  const cwd = process.cwd();
  const mintDir = path.join(cwd, '.mint');
  const configPath = path.join(mintDir, 'config.json');
  const autoFix = flags.fix || false;

  p.intro(`\x1b[32m mint doctor${autoFix ? ' --fix' : ''} \x1b[0m`);

  let passed = 0, warnings = 0, failed = 0, fixed = 0;

  function ok(msg) { p.log.success(msg); passed++; }
  function warn(msg) { p.log.warn(msg); warnings++; }
  function fail(msg) { p.log.error(msg); failed++; }

  function fixable(msg, fixFn) {
    if (autoFix) {
      try {
        fixFn();
        p.log.success(`${msg} — fixed`);
        fixed++;
      } catch (e) {
        p.log.error(`${msg} — fix failed: ${e.message}`);
        failed++;
      }
    } else {
      warn(`${msg} — run with --fix to repair`);
    }
  }

  // Config
  if (fileExists(configPath) && readJsonSafe(configPath)) ok('.mint/config.json valid');
  else fail('.mint/config.json missing or invalid');

  // Global config
  const globalConfigPath = getGlobalConfigPath();
  const globalConfig = loadGlobalConfig();
  if (fileExists(globalConfigPath) && Object.keys(globalConfig).length > 0) {
    const keys = Object.keys(globalConfig).filter(k => GLOBAL_KEYS.includes(k));
    ok(`Global config: ${keys.length} preference${keys.length !== 1 ? 's' : ''} set (${keys.join(', ')})`);
  } else {
    p.log.info('Global config: not set — use \x1b[36mmint config set --global\x1b[0m to set user defaults');
  }

  // Hard blocks
  if (fileExists(path.join(mintDir, 'hard-blocks.md'))) ok('.mint/hard-blocks.md present');
  else warn('.mint/hard-blocks.md missing');

  const config = readJsonSafe(configPath) || {};

  // Stack
  const detected = detectStack(cwd);
  if (config.stack && config.stack !== 'none' && detected !== 'none' && config.stack !== detected) {
    warn(`Stack: config=${config.stack}, detected=${detected}`);
  } else {
    ok(`Stack: ${config.stack || 'none'}`);
  }

  // PM
  ok(`Package manager: ${config.packageManager || 'none'}`);

  // Gates
  if (config.gates) {
    function runGate(label, cmd) {
      if (!cmd || cmd === false) return;
      if (typeof cmd === 'string') {
        try {
          execSync(cmd, { cwd, stdio: 'ignore', timeout: 30000 });
          ok(`Gate: ${label} — ${cmd}`);
        } catch {
          fail(`Gate: ${label} — ${cmd}`);
        }
      } else if (typeof cmd === 'object' && cmd.command) {
        // { command: "...", threshold: 80 } format
        runGate(label, cmd.command);
      } else if (typeof cmd === 'object') {
        // Nested sub-gates: { "api": "cmd1", "web": "cmd2" }
        for (const [sub, subCmd] of Object.entries(cmd)) {
          runGate(`${label}.${sub}`, subCmd);
        }
      }
    }
    for (const [name, cmd] of Object.entries(config.gates)) {
      runGate(name, cmd);
    }
  }

  // Browser (core feature)
  if (config.browser?.enabled) {
    ok('Browser: enabled');
    if (detectTool('pinchtab')) ok('PinchTab installed');
    else warn('PinchTab not installed — run: curl -fsSL https://pinchtab.com/install.sh | sh');
  }

  // Context Mode (core feature)
  if (config.context?.enabled) {
    ok('Context Mode: enabled');
    if (detectContextMode()) ok('context-mode installed');
    else warn('context-mode not installed — install via: claude mcp add context-mode -- npx -y context-mode');
  } else {
    warn('Context Mode: disabled — enable in .mint/config.json for sandboxed execution in large codebases');
  }

  // Design Intelligence (core feature)
  if (config.design?.enabled) {
    ok('Design: enabled');
    const reviewChecks = config.design.review || {};
    const enabledChecks = Object.entries(reviewChecks).filter(([, v]) => v).map(([k]) => k);
    ok(`Design checks: ${enabledChecks.join(', ') || 'none'}`);
    const profilePath = config.design.profile
      ? path.join(cwd, config.design.profile)
      : path.join(mintDir, 'design-profile.json');
    if (fileExists(profilePath)) ok(`Design profile: ${config.design.profile || '.mint/design-profile.json'}`);
    else warn('No design profile — run /design:profile build or it will auto-build on first UI task');

    // Check for Impeccable (optional)
    const hasImpeccable = fileExists(path.join(cwd, '.claude', 'skills', 'frontend-design', 'SKILL.md'))
      || fileExists(path.join(process.env.HOME || '', '.claude', 'skills', 'frontend-design', 'SKILL.md'));
    if (hasImpeccable) ok('Impeccable skill installed (steering commands available)');
  }

  // Doc-manifest
  const manifestPath = path.join(mintDir, 'doc-manifest.json');
  if (fileExists(manifestPath)) {
    const manifest = readJsonSafe(manifestPath);
    if (manifest && manifest.$schema === 'doc-manifest-v1') {
      const docCount = manifest.docs?.length || 0;
      const sectionCount = manifest.docs?.reduce((sum, d) => sum + (d.sections?.length || 0), 0) || 0;
      ok(`Doc-manifest: ${docCount} docs, ${sectionCount} tracked sections`);

      // Check that tracked doc files actually exist
      for (const doc of manifest.docs || []) {
        if (!fileExists(path.join(cwd, doc.path))) {
          warn(`Doc-manifest: ${doc.path} listed but file not found`);
        }
      }
    } else {
      warn('Doc-manifest: invalid schema — expected doc-manifest-v1');
    }
  } else {
    warn('Doc-manifest: not found — run mint init to generate, or create .mint/doc-manifest.json');
  }

  // Learning files
  const issuesPath = path.join(mintDir, 'issues.md');
  if (!fileExists(issuesPath)) {
    fixable('.mint/issues.md missing', () => {
      fs.writeFileSync(issuesPath, '# Mint Issues & Learnings\n\n_Centralized log. All agent blockers, root causes, and learnings go here._\n\n| Date | Task | Severity | Issue | Root Cause | Resolution | Spec Fix |\n|------|------|----------|-------|------------|------------|----------|\n');
    });
  } else {
    ok('.mint/issues.md present');
  }

  const winsPath = path.join(mintDir, 'wins.md');
  if (!fileExists(winsPath)) {
    fixable('.mint/wins.md missing', () => {
      fs.writeFileSync(winsPath, '# Wins\n\n_Successful patterns. The planner reads this before writing specs._\n\n| Date | Task | Pattern | Why It Worked |\n|------|------|---------|---------------|\n');
    });
  } else {
    ok('.mint/wins.md present');
  }

  const instinctsPath = path.join(mintDir, 'instincts.md');
  if (!fileExists(instinctsPath)) {
    fixable('.mint/instincts.md missing', () => {
      fs.writeFileSync(instinctsPath, '# Instincts\n\nAuto-extracted patterns from code observations. The planner reads this to match\nexisting project conventions. Confidence grows as patterns repeat across files.\n\n**High-confidence (>= 3):** Follow by default when writing new code.\n**Low-confidence (1-2):** Observed but not yet established — use judgment.\n\n| Category | Pattern | Files Seen | Confidence | Last Updated |\n|----------|---------|-----------|------------|--------------|\n');
    });
  } else {
    ok('.mint/instincts.md present');
  }

  const patternsPath = path.join(mintDir, 'patterns.md');
  if (!fileExists(patternsPath)) {
    fixable('.mint/patterns.md missing', () => {
      fs.writeFileSync(patternsPath, '# Patterns\n\n_Promoted from issues/wins. Higher confidence than individual log entries._\n\n| Pattern | Source | Confidence | Action |\n|---------|--------|------------|--------|\n');
    });
  } else {
    ok('.mint/patterns.md present');
  }

  // Config completeness
  const configChecks = [
    { path: 'definitionOfDone', default: { gatesPassing: true, specReviewPassed: true, stage2ReviewsPassed: true, docCheckPassed: true, screenshotReminder: 'ui-changes' } },
    { path: 'definitionOfDone.docCheckPassed', default: true },
    { path: 'instincts', default: { enabled: true } },
    { path: 'modelRouting', default: { enabled: true, override: {} } },
    { path: 'tdd', default: { default: false, desloppify: true, coverageThreshold: 80 } },
    { path: 'isolation', default: { plan: 'none', ship: 'none', quick: 'none' } },
  ];

  for (const check of configChecks) {
    const parts = check.path.split('.');
    let val = config;
    for (const part of parts) {
      val = val?.[part];
    }
    if (val === undefined) {
      fixable(`Config missing: ${check.path}`, () => {
        let target = config;
        const pathParts = check.path.split('.');
        for (let i = 0; i < pathParts.length - 1; i++) {
          if (target[pathParts[i]] === undefined) target[pathParts[i]] = {};
          target = target[pathParts[i]];
        }
        target[pathParts[pathParts.length - 1]] = check.default;
        fs.writeFileSync(configPath, JSON.stringify(config, null, 2) + '\n');
      });
    }
  }

  // Reviewer optimization
  const importantReviewers = ['spec', 'quality', 'security', 'conventions'];
  for (const name of importantReviewers) {
    const rev = config.reviewers?.[name];
    if (!rev || rev === false || (typeof rev === 'object' && !rev.enabled)) {
      warn(`Reviewer disabled: ${name} — this is a key quality gate. Enable in config if possible.`);
    }
  }

  for (const [name, val] of Object.entries(config.reviewers || {})) {
    if (val === true) {
      fixable(`Reviewer ${name}: enabled but no model assigned`, () => {
        const defaults = { spec: 'opus', quality: 'sonnet', security: 'sonnet', conventions: 'haiku', tests: 'sonnet', business: 'opus', performance: 'sonnet' };
        config.reviewers[name] = { enabled: true, model: defaults[name] || 'sonnet' };
        fs.writeFileSync(configPath, JSON.stringify(config, null, 2) + '\n');
      });
    }
  }

  // CLAUDE.md mint section
  const claudeMdPath = path.join(cwd, 'CLAUDE.md');
  if (fileExists(claudeMdPath)) {
    let claudeMd = '';
    try { claudeMd = fs.readFileSync(claudeMdPath, 'utf8'); } catch { /* */ }
    if (claudeMd.includes('<!-- mint:start')) {
      const versionMatch = claudeMd.match(/<!-- mint:start v(\d+) -->/);
      const version = versionMatch ? versionMatch[1] : '0';
      ok(`CLAUDE.md: mint section present (v${version})`);
    } else {
      warn('CLAUDE.md: missing mint section — run mint init to add it');
    }
  } else {
    warn('CLAUDE.md: file not found — run mint init to create it with mint section');
  }

  // Doc-manifest section quality
  if (fileExists(manifestPath)) {
    const manifestData = readJsonSafe(manifestPath);
    if (manifestData?.docs) {
      const emptySections = manifestData.docs.filter(d => !d.sections || d.sections.length === 0);
      if (emptySections.length > 0) {
        warn(`Doc-manifest: ${emptySections.length} docs with no tracked sections — run /doc-setup to populate`);
      }
    }
  }

  // .gitignore completeness
  const gitignorePath = path.join(cwd, '.gitignore');
  const requiredIgnores = ['.mint/tasks/', '.mint/research/', '.mint/worktrees/', '.mint/.session-state.json'];
  if (fileExists(gitignorePath)) {
    const gi = fs.readFileSync(gitignorePath, 'utf8');
    for (const ignore of requiredIgnores) {
      if (!gi.includes(ignore)) {
        fixable(`.gitignore missing: ${ignore}`, () => {
          fs.appendFileSync(gitignorePath, `${ignore}\n`);
        });
      }
    }
  }

  // Plugin hooks
  const home = process.env.HOME || process.env.USERPROFILE || '';
  const pluginJsonPaths = [
    path.join(cwd, '.claude-plugin', 'plugin.json'),
    path.join(home, '.mint', '.claude-plugin', 'plugin.json'),
    path.join(home, '.claude', 'plugins', 'marketplaces', 'mint', '.claude-plugin', 'plugin.json'),
  ];
  for (const pjPath of pluginJsonPaths) {
    if (fileExists(pjPath)) {
      const pj = readJsonSafe(pjPath);
      if (pj && !pj.hooks) {
        fail(`plugin.json missing "hooks" field at ${pjPath} — hooks won't load. Add: "hooks": "./hooks/hooks.json"`);
      } else if (pj && pj.hooks) {
        ok('Plugin hooks declared in plugin.json');
      }
      break;
    }
  }

  // Plugins
  const marketplaceDir = path.join(home, '.claude', 'plugins', 'marketplaces', 'mint');

  for (const plugin of config.plugins || []) {
    const name = path.basename(plugin);
    const paths = [
      path.join(cwd, plugin, 'manifest.json'),
      path.join(marketplaceDir, plugin, 'manifest.json'),
    ];

    let found = false;
    for (const mp of paths) {
      if (fileExists(mp) && readJsonSafe(mp)) { found = true; break; }
    }

    if (found) {
      ok(`Plugin: ${name}`);
    } else {
      fail(`Plugin: ${name} — manifest not found`);
    }
  }

  // Tools
  // Runtime tools
  if (detectTool('bun')) {
    try {
      const bunVersion = execSync('bun --version', { encoding: 'utf8', timeout: 5000 }).trim();
      ok(`Bun: ${bunVersion} (required for mint CLI)`);
    } catch {
      ok('Bun: installed (required for mint CLI)');
    }
  } else {
    fail('Bun not installed — mint CLI requires Bun. Install: curl -fsSL https://bun.sh/install | bash');
  }

  if (detectTool('claude')) ok('Claude CLI installed');
  else warn('Claude CLI not found');

  if (fileExists(path.join(cwd, '.git'))) ok('Git initialized');
  else warn('Not a git repository');

  ok(`Node.js ${process.version}`);

  // Summary
  const parts = [];
  if (passed) parts.push(`\x1b[32m${passed} passed\x1b[0m`);
  if (warnings) parts.push(`\x1b[33m${warnings} warnings\x1b[0m`);
  if (failed) parts.push(`\x1b[31m${failed} failed\x1b[0m`);
  if (fixed) parts.push(`\x1b[34m${fixed} fixed\x1b[0m`);

  p.outro(parts.join(', '));
}
