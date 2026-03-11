import * as p from '@clack/prompts';
import path from 'path';
import fs from 'fs';
import { execSync } from 'child_process';
import { detectTool, readJsonSafe } from '../lib/detect.js';

// New core config keys added per version. When we add a new core feature,
// add it here so `mint update` can offer it to existing users.
const NEW_CONFIG_KEYS = [
  {
    key: 'browser',
    label: 'Browser support (PinchTab — debug, scrape, test live apps)',
    default: { enabled: true },
    defaultOff: { enabled: false },
  },
  // Future example:
  // {
  //   key: 'someNewFeature',
  //   label: 'Description of the new feature',
  //   default: { enabled: true, option: 'value' },
  //   defaultOff: { enabled: false },
  // },
];

export async function run() {
  p.intro('\x1b[32m mint update \x1b[0m');

  const home = process.env.HOME || process.env.USERPROFILE || '';
  const mintHome = path.join(home, '.mint');
  const marketplaceDir = path.join(home, '.claude', 'plugins', 'marketplaces', 'mint');
  const cacheDir = path.join(home, '.claude', 'plugins', 'cache', 'mint');

  const s = p.spinner();

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

  // Read version
  let version = 'unknown';
  try {
    const pkgPath = path.join(mintHome, 'package.json');
    if (fs.existsSync(pkgPath)) {
      version = JSON.parse(fs.readFileSync(pkgPath, 'utf8')).version;
    }
  } catch { /* ignore */ }

  p.outro(`mint v${version} — restart Claude Code to activate`);
}
