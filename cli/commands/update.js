import * as p from '@clack/prompts';
import path from 'path';
import fs from 'fs';
import { execSync } from 'child_process';
import { detectTool, detectContextMode, readJsonSafe } from '../lib/detect.js';

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
    label: 'Design intelligence (design profiling, anti-pattern detection, RTL/i18n enforcement)',
    default: { enabled: true, stack: 'auto', profile: '.mint/design-profile.json', notes: '.mint/design-notes.md', conventions: [], review: { accessibility: true, consistency: true, performance: true, rtl: false, i18n: false, brand: false } },
    defaultOff: { enabled: false },
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
    p.log.warn('PinchTab not installed — skipping. Install: curl -fsSL https://pinchtab.com/install.sh | sh');
    return;
  }

  s.start(`Updating PinchTab${before ? ` (current: ${before})` : ''}...`);
  try {
    execSync('curl -fsSL https://pinchtab.com/install.sh | sh', { stdio: 'pipe', timeout: 120000 });
    const after = getVersion('pinchtab --version 2>/dev/null');
    if (before && after && before !== after) {
      s.stop(`PinchTab updated: ${before} → ${after}`);
    } else if (after) {
      s.stop(`PinchTab up to date (${after})`);
    } else {
      s.stop('PinchTab updated');
    }
  } catch {
    s.stop('PinchTab update failed — try manually: curl -fsSL https://pinchtab.com/install.sh | sh');
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

  p.intro(`\x1b[32m mint update${depsOnly ? ' deps' : ''} \x1b[0m`);

  const s = p.spinner();

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
        execSync(`git -C "${mintHome}" fetch origin main -q`, { stdio: 'pipe' });
        execSync(`git -C "${mintHome}" reset --hard origin/main -q`, { stdio: 'pipe' });
        s.stop('Updated ~/.mint');
      } catch {
        s.stop('Failed to update ~/.mint');
      }
    }

    // Install/update dependencies
    s.start('Installing dependencies...');
    try {
      if (detectTool('bun')) {
        execSync(`cd "${mintHome}" && bun install`, { stdio: 'pipe' });
      } else {
        execSync(`cd "${mintHome}" && npm install --omit=dev`, { stdio: 'pipe' });
      }
      s.stop('Dependencies up to date');
    } catch {
      s.stop('Dependency install failed — run manually in ~/.mint');
    }

    // Update Claude plugin if available
    if (detectTool('claude')) {
      s.start('Updating Claude plugin...');
      try {
        if (fs.existsSync(path.join(marketplaceDir, '.git'))) {
          execSync(`git -C "${marketplaceDir}" fetch origin main -q`, { stdio: 'pipe' });
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

    // Offer new config keys if project has a config
    const projectConfig = path.join(process.cwd(), '.mint', 'config.json');
    const config = readJsonSafe(projectConfig);
    if (config) {
      const missing = NEW_CONFIG_KEYS.filter(k => config[k.key] === undefined);
      if (missing.length > 0) {
        p.log.info(`${missing.length} new feature${missing.length > 1 ? 's' : ''} available:`);

        for (const feat of missing) {
          const enable = await p.confirm({
            message: `Enable ${feat.label}?`,
            initialValue: true,
          });

          if (p.isCancel(enable)) break;
          config[feat.key] = enable ? feat.default : feat.defaultOff;
        }

        fs.writeFileSync(projectConfig, JSON.stringify(config, null, 2) + '\n');
        p.log.success('Config updated');
      }
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

  p.outro(`mint v${version}${depsOnly ? '' : ' — restart Claude Code to activate'}`);
}
