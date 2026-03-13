import * as p from '@clack/prompts';
import path from 'path';
import { execSync } from 'child_process';
import { readJsonSafe, fileExists, detectStack, detectTool, detectContextMode } from '../lib/detect.js';

export async function run() {
  const cwd = process.cwd();
  const mintDir = path.join(cwd, '.mint');
  const configPath = path.join(mintDir, 'config.json');

  p.intro('\x1b[32m mint doctor \x1b[0m');

  let passed = 0, warnings = 0, failed = 0;

  function ok(msg) { p.log.success(msg); passed++; }
  function warn(msg) { p.log.warn(msg); warnings++; }
  function fail(msg) { p.log.error(msg); failed++; }

  // Config
  if (fileExists(configPath) && readJsonSafe(configPath)) ok('.mint/config.json valid');
  else fail('.mint/config.json missing or invalid');

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
    for (const [name, cmd] of Object.entries(config.gates)) {
      if (!cmd || cmd === false) continue;
      const command = typeof cmd === 'object' ? cmd.command : cmd;
      try {
        execSync(command, { cwd, stdio: 'ignore', timeout: 30000 });
        ok(`Gate: ${name} — ${command}`);
      } catch {
        fail(`Gate: ${name} — ${command}`);
      }
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
  }

  // Plugins
  const home = process.env.HOME || process.env.USERPROFILE || '';
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

  p.outro(parts.join(', '));
}
