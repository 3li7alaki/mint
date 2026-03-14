import * as p from '@clack/prompts';
import fs from 'fs';
import path from 'path';
import { readJsonSafe, fileExists } from '../lib/detect.js';

function getConfigPath() {
  return path.join(process.cwd(), '.mint', 'config.json');
}

function loadConfig() {
  const config = readJsonSafe(getConfigPath());
  if (!config) {
    p.log.error('No .mint/config.json found. Run \x1b[36mmint init\x1b[0m first.');
    process.exit(1);
  }
  return config;
}

function saveConfig(config) {
  fs.writeFileSync(getConfigPath(), JSON.stringify(config, null, 2) + '\n');
}

function showConfig() {
  const config = loadConfig();
  const c = (v) => `\x1b[36m${v}\x1b[0m`;
  const g = (v) => `\x1b[32m✓\x1b[0m ${v}`;
  const r = (v) => `\x1b[31m✗\x1b[0m ${v || 'off'}`;
  const d = (v) => `\x1b[2m${v}\x1b[0m`;

  console.log(`\n  \x1b[1mmint config\x1b[0m\n`);
  console.log(`  Stack:       ${c(config.stack || 'none')}`);
  console.log(`  PM:          ${c(config.packageManager || 'none')}`);
  console.log(`  Isolation:   ${c(config.isolation?.plan || 'none')}`);
  console.log(`  Auto-commit: ${config.autoCommit !== false ? g('on') : r()}`);
  console.log(`  TDD:         ${config.tdd?.default ? g('on') : r()}`);
  console.log(`  Browser:     ${config.browser?.enabled ? g('on') : r()}`);
  console.log(`  Context:     ${config.context?.enabled ? g('on') : r()}`);
  console.log(`  Design:      ${config.design?.enabled ? g('on') : r()}`);

  console.log(`\n  \x1b[1mGates\x1b[0m`);
  for (const key of ['lint', 'types', 'tests']) {
    const val = config.gates?.[key];
    console.log(`    ${key.padEnd(8)} ${val ? g(typeof val === 'object' ? val.command : val) : r()}`);
  }

  console.log(`\n  \x1b[1mReviewers\x1b[0m`);
  if (config.reviewers) {
    for (const [name, val] of Object.entries(config.reviewers)) {
      if (val && (val === true || val.enabled)) {
        const model = typeof val === 'object' ? d(` (${val.model})`) : '';
        console.log(`    ${name.padEnd(14)} ${g('')}${model}`);
      } else {
        console.log(`    ${name.padEnd(14)} ${r()}`);
      }
    }
  }

  console.log(`\n  \x1b[1mPlugins\x1b[0m`);
  if (config.plugins?.length > 0) {
    for (const pl of config.plugins) {
      console.log(`    ${c(path.basename(pl))}`);
    }
  } else {
    console.log(`    ${d('none')}`);
  }
  console.log('');
}

function setConfigValue(key, value) {
  const config = loadConfig();
  const parts = key.split('.');
  let target = config;
  for (let i = 0; i < parts.length - 1; i++) {
    if (target[parts[i]] === undefined) target[parts[i]] = {};
    target = target[parts[i]];
  }

  if (value === 'true') value = true;
  else if (value === 'false') value = false;
  else if (!isNaN(value) && value !== '') value = Number(value);

  target[parts[parts.length - 1]] = value;
  saveConfig(config);
  p.log.success(`Set \x1b[36m${key}\x1b[0m = \x1b[32m${String(value)}\x1b[0m`);
}

async function managePlugins(action, pluginName) {
  const config = loadConfig();
  const currentPlugins = config.plugins || [];
  const currentNames = currentPlugins.map(pl => path.basename(pl));

  if (action === 'add' && pluginName) {
    if (currentNames.includes(pluginName)) {
      p.log.warn(`${pluginName} is already enabled.`);
      return;
    }
    config.plugins = [...currentPlugins, `plugins/${pluginName}`];
    saveConfig(config);
    p.log.success(`Added \x1b[32m${pluginName}\x1b[0m`);
    return;
  }

  if (action === 'remove' && pluginName) {
    config.plugins = currentPlugins.filter(pl => path.basename(pl) !== pluginName);
    saveConfig(config);
    p.log.success(`Removed ${pluginName}`);
    return;
  }

  // Interactive mode — browser is a core feature, not listed here
  const allPlugins = [
    { value: 'mint-e2e', label: 'E2E Testing', hint: 'Playwright' },
    { value: 'mint-linear', label: 'Linear', hint: 'ticket sync' },
    { value: 'mint-figma', label: 'Figma', hint: 'design specs' },
    { value: 'mint-nuxt', label: 'Nuxt', hint: 'conventions' },
    { value: 'mint-shadcn', label: 'shadcn/ui', hint: 'components' },
    { value: 'mint-ssh', label: 'SSH', hint: 'remote access' },
    { value: 'mint-gws', label: 'Google Workspace', hint: 'Sheets, Gmail' },
  ];

  const selected = await p.multiselect({
    message: 'Toggle plugins',
    options: allPlugins,
    initialValues: currentNames,
    required: false,
  });

  if (p.isCancel(selected)) return;

  config.plugins = selected.map(name => `plugins/${name}`);
  saveConfig(config);
  p.log.success(`Plugins: ${selected.length ? selected.join(', ') : 'none'}`);
}

export async function run(args, flags = {}) {
  const subcommand = args[0];

  if (!subcommand || subcommand === 'show') {
    showConfig();
    return;
  }

  if (subcommand === 'set') {
    const key = args[1];
    const value = args[2];
    if (!key || value === undefined) {
      console.log('  Usage: \x1b[36mmint config set <key> <value>\x1b[0m');
      console.log('  Example: \x1b[2mmint config set isolation.plan none\x1b[0m');
      return;
    }
    setConfigValue(key, value);
    return;
  }

  if (subcommand === 'plugins') {
    await managePlugins(args[1], args[2]);
    return;
  }

  p.log.error(`Unknown: ${subcommand}. Try: show, set, plugins`);
}
