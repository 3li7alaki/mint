import * as p from '@clack/prompts';
import path from 'path';
import fs from 'fs';
import { execSync } from 'child_process';
import { detectTool, detectContextMode, readJsonSafe, fileExists, ensureClaudeMd, getGlobalConfigPath, loadGlobalConfig, saveGlobalConfig, GLOBAL_KEYS, registerProject, getRegisteredProjects } from '../lib/detect.js';
import { ensureGitignore } from '../lib/gitignore.js';
import { spinner } from '../lib/spinner.js';

// New core config keys added per version.
const NEW_CONFIG_KEYS = [
  {
    key: 'browser',
    label: 'Browser support (PinchTab — debug, scrape, test live apps)',
    default: { enabled: true },
    defaultOff: { enabled: false },
  },
  {
    key: 'context',
    label: 'Context Mode (sandboxed execution, session continuity, FTS5 search)',
    default: { enabled: true, autoRoute: true, sandbox: { timeout: 30000 }, session: { enabled: true } },
    defaultOff: { enabled: false },
  },
  {
    key: 'design',
    label: 'Design & UI/UX (design profiling, typography/color/motion expertise, anti-pattern detection, RTL, i18n, accessibility review)',
    default: { enabled: true, stack: 'auto', profile: '.mint/design-profile.json', notes: '.mint/design-notes.md', conventions: [], review: { accessibility: true, consistency: true, performance: true, rtl: false, i18n: false, brand: false } },
    defaultOff: { enabled: false },
  },
  {
    key: 'signature',
    label: '"Minted with mint" signature in README (for open source projects)',
    default: true,
    defaultOff: false,
  },
];

// ─── Dependency updaters ──────────────────────────────────────────────────────

function getVersion(cmd) {
  try { return execSync(cmd, { encoding: 'utf8', stdio: 'pipe', timeout: 10000 }).trim(); }
  catch { return null; }
}

function updatePinchTab(s) {
  const before = getVersion('pinchtab --version 2>/dev/null');
  if (!before && !detectTool('pinchtab')) {
    p.log.warn('PinchTab not installed — skipping. Install: curl -fsSL https://pinchtab.com/install.sh | bash');
    return;
  }

  s.start(`Updating PinchTab${before ? ` (current: ${before})` : ''}...`);
  try {
    execSync('curl -fsSL https://pinchtab.com/install.sh | bash', { stdio: 'pipe', timeout: 120000 });
    const after = getVersion('pinchtab --version 2>/dev/null');
    if (before && after && before !== after) {
      s.stop(`PinchTab updated: ${before} → ${after}`);
    } else if (after) {
      s.stop(`PinchTab up to date (${after})`);
    } else {
      s.stop('PinchTab updated');
    }
  } catch {
    s.stop('PinchTab update failed — try manually: curl -fsSL https://pinchtab.com/install.sh | bash');
  }
}

function updateContextMode(s) {
  const before = getVersion('npx -y context-mode --version 2>/dev/null');
  if (!before && !detectContextMode()) {
    p.log.warn('context-mode not installed — skipping. Install: claude mcp add context-mode -- npx -y context-mode');
    return;
  }

  s.start(`Updating context-mode${before ? ` (current: ${before})` : ''}...`);
  try {
    // Clear npx cache to force latest
    execSync('npx -y context-mode@latest --version', { stdio: 'pipe', timeout: 120000 });
    const after = getVersion('npx -y context-mode --version 2>/dev/null');
    if (before && after && before !== after) {
      s.stop(`context-mode updated: ${before} → ${after}`);
    } else if (after) {
      s.stop(`context-mode up to date (${after})`);
    } else {
      s.stop('context-mode updated');
    }
  } catch {
    s.stop('context-mode update failed — try: npx -y context-mode@latest --version');
  }
}

function updateImpeccable(s) {
  const home = process.env.HOME || process.env.USERPROFILE || '';
  const skillPath = path.join(home, '.claude', 'skills', 'frontend-design', 'SKILL.md');
  const projectSkillPath = path.join(process.cwd(), '.claude', 'skills', 'frontend-design', 'SKILL.md');

  if (!fs.existsSync(skillPath) && !fs.existsSync(projectSkillPath)) {
    p.log.warn('Impeccable not installed — skipping. Install: npx skills add pbakaus/impeccable');
    return;
  }

  s.start('Updating Impeccable...');
  try {
    execSync('npx skills add pbakaus/impeccable', { stdio: 'pipe', timeout: 60000 });
    s.stop('Impeccable updated');
  } catch {
    s.stop('Impeccable update failed — try: npx skills add pbakaus/impeccable');
  }
}

const DEPS = {
  pinchtab: updatePinchTab,
  'context-mode': updateContextMode,
  impeccable: updateImpeccable,
};

// ─── Main ─────────────────────────────────────────────────────────────────────

export async function run(positional = [], flags = {}) {
  const target = positional[0];
  const depsOnly = flags.deps || !!target;
  // Never interactive — all decisions via Claude or sensible defaults. No p.confirm() ever.
  const headless = true;

  p.intro(`\x1b[32m mint update${depsOnly ? ' deps' : ''} \x1b[0m`);

  const s = spinner();

  // ─── Single dep update ────────────────────────────────────────────────────
  if (target) {
    const updater = DEPS[target.toLowerCase()];
    if (!updater) {
      p.log.error(`Unknown dependency: ${target}`);
      p.log.info(`Available: ${Object.keys(DEPS).join(', ')}`);
      p.outro('');
      return;
    }
    updater(s);
    p.outro('Done');
    return;
  }

  // ─── Update mint itself (unless --deps) ───────────────────────────────────
  if (!depsOnly) {
    const home = process.env.HOME || process.env.USERPROFILE || '';
    const mintHome = path.join(home, '.mint');
    const marketplaceDir = path.join(home, '.claude', 'plugins', 'marketplaces', 'mint');
    const cacheDir = path.join(home, '.claude', 'plugins', 'cache', 'mint');

    // Update ~/.mint repo
    if (fs.existsSync(path.join(mintHome, '.git'))) {
      s.start('Fetching latest...');
      try {
        // Ensure LF line endings (WSL with Windows git may default to CRLF)
        execSync(`git -C "${mintHome}" config core.autocrlf input`, { stdio: 'pipe' });
        execSync(`git -C "${mintHome}" fetch origin main -q`, { stdio: 'pipe' });
        // Preserve global config before cleaning (it lives in the repo dir as an untracked file)
        const globalCfgPath = path.join(mintHome, 'config.json');
        let savedGlobalConfig = null;
        if (fs.existsSync(globalCfgPath)) {
          try { savedGlobalConfig = fs.readFileSync(globalCfgPath, 'utf8'); } catch { /* */ }
        }
        execSync(`git -C "${mintHome}" clean -fd -q`, { stdio: 'pipe' });
        execSync(`git -C "${mintHome}" reset --hard origin/main -q`, { stdio: 'pipe' });
        // Restore global config after clean
        if (savedGlobalConfig) {
          try { fs.writeFileSync(globalCfgPath, savedGlobalConfig); } catch { /* */ }
        }
        s.stop('Updated ~/.mint');
      } catch {
        s.stop('Failed to update ~/.mint');
      }
    }

    // Install/update dependencies (whichever install path exists)
    const installDir = fs.existsSync(path.join(mintHome, 'package.json')) ? mintHome
      : fs.existsSync(path.join(marketplaceDir, 'package.json')) ? marketplaceDir
      : null;

    if (installDir) {
      s.start('Installing dependencies...');
      try {
        if (detectTool('bun')) {
          execSync(`cd "${installDir}" && bun install`, { stdio: 'pipe' });
        } else {
          execSync(`cd "${installDir}" && npm install --omit=dev`, { stdio: 'pipe' });
        }
        s.stop('Dependencies up to date');
      } catch {
        s.stop(`Dependency install failed — run manually in ${installDir}`);
      }
    }

    // Update Claude plugin if available
    if (detectTool('claude')) {
      s.start('Updating Claude plugin...');
      try {
        if (fs.existsSync(path.join(marketplaceDir, '.git'))) {
          execSync(`git -C "${marketplaceDir}" config core.autocrlf input`, { stdio: 'pipe' });
          execSync(`git -C "${marketplaceDir}" fetch origin main -q`, { stdio: 'pipe' });
          execSync(`git -C "${marketplaceDir}" clean -fd -q`, { stdio: 'pipe' });
          execSync(`git -C "${marketplaceDir}" reset --hard origin/main -q`, { stdio: 'pipe' });
        }
        execSync(`rm -rf "${cacheDir}"`, { stdio: 'pipe' });
        execSync('claude plugin install "mint@mint"', { stdio: 'pipe' });
        s.stop('Claude plugin updated');
      } catch {
        s.stop('Plugin update failed — try: claude plugin install "mint@mint"');
      }
    } else {
      p.log.info('Claude CLI not found — skipping plugin update');
    }

    // Create global config on first ever run — auto-seed from current project, no prompts
    const globalConfigPath = getGlobalConfigPath();
    if (!fs.existsSync(globalConfigPath)) {
      const projectConfigForGlobal = readJsonSafe(path.join(process.cwd(), '.mint', 'config.json'));
      if (projectConfigForGlobal) {
        const globalDefaults = {};
        for (const key of GLOBAL_KEYS) {
          if (projectConfigForGlobal[key] !== undefined) {
            globalDefaults[key] = projectConfigForGlobal[key];
          }
        }
        saveGlobalConfig(globalDefaults);
        p.log.success(`Global config seeded from current project`);
      }
    }

    // Offer new config keys if project has a config
    const projectConfig = path.join(process.cwd(), '.mint', 'config.json');
    const config = readJsonSafe(projectConfig);
    if (config) {
      const missing = NEW_CONFIG_KEYS.filter(k => config[k.key] === undefined);
      if (missing.length > 0) {
        // Safe defaults — Claude smart session will optimize further
        for (const feat of missing) {
          config[feat.key] = feat.defaultOff;
        }
        fs.writeFileSync(projectConfig, JSON.stringify(config, null, 2) + '\n');
        p.log.success(`${missing.length} new config key${missing.length > 1 ? 's' : ''} added`);
      }

      // Migrate sub-keys for existing features
      let configChanged = false;

      // design.uiFilePatterns (added in v0.6.3)
      if (config.design && config.design.enabled && !config.design.uiFilePatterns) {
        config.design.uiFilePatterns = ['*.tsx', '*.jsx', '*.vue', '*.svelte', '*.css', '*.scss', '*.html'];
        p.log.success('Added design.uiFilePatterns (file-pattern design auto-detection)');
        configChanged = true;
      }

      // doc-manifest.json (added in v0.6.4)
      const manifestPath = path.join(process.cwd(), '.mint', 'doc-manifest.json');
      if (!fileExists(manifestPath)) {
        const { generateDocManifest } = await import('../lib/detect.js');
        if (typeof generateDocManifest === 'function') {
          const manifest = generateDocManifest(process.cwd());
          fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + '\n');
          p.log.success('Created .mint/doc-manifest.json (doc tracking)');
          configChanged = true;
        }
      }

      // Migrate old documenters array to doc-manifest
      if (config.documenters && config.documenters.length > 0 && fileExists(manifestPath)) {
        const manifest = readJsonSafe(manifestPath);
        if (manifest && (!manifest.docs || manifest.docs.length === 0)) {
          manifest.docs = config.documenters.map(d => ({
            path: d.path,
            description: d.description,
            trigger: d.trigger || 'on-architectural-change',
            sections: [],
          }));
          fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + '\n');
          p.log.success('Migrated documenters config to doc-manifest.json');
          configChanged = true;
        }
      }

      // Add definitionOfDone.docCheckPassed if missing
      if (config.definitionOfDone && config.definitionOfDone.docCheckPassed === undefined) {
        config.definitionOfDone.docCheckPassed = true;
        p.log.success('Added definitionOfDone.docCheckPassed');
        configChanged = true;
      }

      if (configChanged) {
        fs.writeFileSync(projectConfig, JSON.stringify(config, null, 2) + '\n');
      }

      // Ensure CLAUDE.md has mint section
      const claudeResult = ensureClaudeMd(process.cwd());
      if (claudeResult === 'created') p.log.success('Created CLAUDE.md with mint section');
      else if (claudeResult === 'updated') p.log.success('Updated mint section in CLAUDE.md');

      // Ensure .gitignore has all mint entries
      const { added } = ensureGitignore(process.cwd());
      if (added.length > 0) {
        p.log.success(`Added ${added.length} entries to .gitignore`);
      }
      // Clean up old .session-state.json file if it exists
      const oldSessionState = path.join(process.cwd(), '.mint', '.session-state.json');
      try { fs.unlinkSync(oldSessionState); } catch { /* already gone */ }
    }
  }

  // ─── Update core deps (--deps flag or after mint update) ──────────────────
  if (depsOnly || !flags['no-deps']) {
    const projectConfig = path.join(process.cwd(), '.mint', 'config.json');
    const config = readJsonSafe(projectConfig);

    if (config?.browser?.enabled || depsOnly) {
      updatePinchTab(s);
    }

    if (config?.context?.enabled || depsOnly) {
      updateContextMode(s);
    }

    if (config?.design?.enabled || depsOnly) {
      updateImpeccable(s);
    }
  }

  // Read version
  let version = 'unknown';
  try {
    const home = process.env.HOME || process.env.USERPROFILE || '';
    const pkgPath = path.join(home, '.mint', 'package.json');
    if (fs.existsSync(pkgPath)) {
      version = JSON.parse(fs.readFileSync(pkgPath, 'utf8')).version;
    }
  } catch { /* ignore */ }

  // ─── Register current project ───────────────────────────────────────────
  const currentProjectConfig = path.join(process.cwd(), '.mint', 'config.json');
  if (readJsonSafe(currentProjectConfig)) {
    registerProject(process.cwd());
  }

  // ─── Smart update — all registered projects ───────────────────────────────
  // Claude reads each project and applies context-aware config changes.
  if (detectTool('claude') && !depsOnly) {
    const projects = getRegisteredProjects();
    if (projects.length > 0) {
      p.log.info(`${projects.length} registered project${projects.length > 1 ? 's' : ''} found`);

      for (const project of projects) {
        const projName = path.basename(project.path);
        const s = spinner();
        s.start(`Smart update: ${projName}...`);
        try {
          const { smartUpdate } = await import('../lib/smart-session.js');
          const analysis = await smartUpdate(project.path, '0.6.x', version);
          if (analysis.success && analysis.result) {
            s.stop(`${projName}: updated`);
            // Show brief summary — first 3 lines of result
            const lines = analysis.result.split('\n').filter(l => l.trim()).slice(0, 3);
            for (const line of lines) {
              console.log(`    \x1b[2m${line}\x1b[0m`);
            }
          } else {
            s.stop(`${projName}: already optimal`);
          }
        } catch (err) {
          s.stop(`${projName}: smart update failed — ${err.message}`);
        }
      }
    }
  }

  p.outro(`mint v${version}${depsOnly ? '' : ' — restart Claude Code to activate'}`);
}
