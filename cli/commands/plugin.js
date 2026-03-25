import * as p from '@clack/prompts';
import fs from 'fs';
import path from 'path';
import { readJsonSafe, fileExists } from '../lib/detect.js';

// Known plugins — registry of available plugins
const PLUGIN_REGISTRY = [
  { name: 'mint-e2e', description: 'E2E testing with Playwright', type: 'stack', hint: 'page objects, flaky detection, semantic locators' },
  { name: 'mint-nuxt', description: 'Nuxt.js conventions and review', type: 'stack', hint: 'auto-imports, server routes, composables' },
  { name: 'mint-react', description: 'React conventions and review', type: 'stack', hint: 'hooks, component patterns, state management' },
  { name: 'mint-laravel', description: 'Laravel conventions and review', type: 'stack', hint: 'eloquent, migrations, artisan' },
  { name: 'mint-linear', description: 'Linear issue tracking sync', type: 'pm', hint: 'ticket context, status updates' },
  { name: 'mint-figma', description: 'Figma design token sync', type: 'design', hint: 'tokens, specs, components' },
  { name: 'mint-shadcn', description: 'shadcn/ui component management', type: 'stack', hint: 'component registry, variants' },
  { name: 'mint-ssh', description: 'SSH remote server access', type: 'stack', hint: 'staging, production, docker' },
  { name: 'mint-gws', description: 'Google Workspace integration', type: 'pm', hint: 'Sheets, Gmail, Calendar' },
];

function getConfig() {
  const configPath = path.join(process.cwd(), '.mint', 'config.json');
  return readJsonSafe(configPath) || {};
}

function saveConfig(config) {
  const configPath = path.join(process.cwd(), '.mint', 'config.json');
  fs.writeFileSync(configPath, JSON.stringify(config, null, 2) + '\n');
}

export async function run(args = [], flags = {}) {
  const subcommand = args[0];

  if (!subcommand || subcommand === 'list') {
    showList();
    return;
  }

  if (subcommand === 'add') {
    const name = args[1];
    if (!name) {
      p.log.error('Usage: mint plugin add <name>');
      return;
    }
    addPlugin(name);
    return;
  }

  if (subcommand === 'remove') {
    const name = args[1];
    if (!name) {
      p.log.error('Usage: mint plugin remove <name>');
      return;
    }
    removePlugin(name);
    return;
  }

  if (subcommand === 'info') {
    const name = args[1];
    if (!name) {
      p.log.error('Usage: mint plugin info <name>');
      return;
    }
    showInfo(name);
    return;
  }

  p.log.error(`Unknown: ${subcommand}. Try: list, add, remove, info`);
}

function showList() {
  const config = getConfig();
  const installed = (config.plugins || []).map(p => path.basename(p));

  console.log('\n  \x1b[1mAvailable plugins\x1b[0m\n');

  for (const plugin of PLUGIN_REGISTRY) {
    const status = installed.includes(plugin.name)
      ? '\x1b[32minstalled\x1b[0m'
      : '\x1b[2mnot installed\x1b[0m';
    const hint = `\x1b[2m${plugin.hint}\x1b[0m`;
    console.log(`  \x1b[36m${plugin.name.padEnd(16)}\x1b[0m ${plugin.description.padEnd(42)} ${status}`);
    console.log(`  ${''.padEnd(16)} ${hint}`);
  }

  console.log(`\n  \x1b[2mInstall: mint plugin add <name>  |  Remove: mint plugin remove <name>\x1b[0m\n`);
}

function addPlugin(name) {
  const plugin = PLUGIN_REGISTRY.find(p => p.name === name);
  if (!plugin) {
    p.log.error(`Unknown plugin: ${name}`);
    p.log.info(`Available: ${PLUGIN_REGISTRY.map(p => p.name).join(', ')}`);
    return;
  }

  const config = getConfig();
  const current = config.plugins || [];
  const currentNames = current.map(p => path.basename(p));

  if (currentNames.includes(name)) {
    p.log.warn(`${name} is already installed.`);
    return;
  }

  config.plugins = [...current, `plugins/${name}`];
  saveConfig(config);
  p.log.success(`Added \x1b[32m${name}\x1b[0m — ${plugin.description}`);
}

function removePlugin(name) {
  const config = getConfig();
  const current = config.plugins || [];
  const filtered = current.filter(p => path.basename(p) !== name);

  if (filtered.length === current.length) {
    p.log.warn(`${name} is not installed.`);
    return;
  }

  config.plugins = filtered;
  saveConfig(config);
  p.log.success(`Removed ${name}`);
}

function showInfo(name) {
  const plugin = PLUGIN_REGISTRY.find(p => p.name === name);
  if (!plugin) {
    p.log.error(`Unknown plugin: ${name}`);
    return;
  }

  const config = getConfig();
  const installed = (config.plugins || []).map(p => path.basename(p)).includes(name);

  console.log(`\n  \x1b[1m${name}\x1b[0m — ${plugin.description}`);
  console.log(`  ─────────────────`);
  console.log(`  Type:      ${plugin.type}`);
  console.log(`  Features:  ${plugin.hint}`);
  console.log(`  Status:    ${installed ? '\x1b[32minstalled\x1b[0m' : '\x1b[2mnot installed\x1b[0m'}`);

  // Try to read manifest for more details
  const home = process.env.HOME || process.env.USERPROFILE || '';
  const manifestPaths = [
    path.join(process.cwd(), 'plugins', name, 'manifest.json'),
    path.join(home, '.mint', 'plugins', name, 'manifest.json'),
    path.join(home, '.claude', 'plugins', 'marketplaces', 'mint', 'plugins', name, 'manifest.json'),
  ];

  for (const mp of manifestPaths) {
    const manifest = readJsonSafe(mp);
    if (manifest) {
      if (manifest.agents) {
        const agents = Object.keys(manifest.agents).map(a => `${name}:${a}`);
        console.log(`  Agents:    ${agents.join(', ')}`);
      }
      if (manifest.hooks) {
        console.log(`  Hooks:     ${Object.keys(manifest.hooks).join(', ')}`);
      }
      break;
    }
  }

  console.log('');
}
