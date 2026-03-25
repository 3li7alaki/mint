import * as p from '@clack/prompts';
import fs from 'fs';
import path from 'path';
import { readJsonSafe, fileExists, loadGlobalConfig, saveGlobalConfig, getGlobalConfigPath, mergeConfigs, GLOBAL_KEYS } from '../lib/detect.js';
import { validateConfigValue, listConfigKeys } from '../lib/schema.js';

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

function showGlobalConfig() {
  const config = loadGlobalConfig();
  const c = (v) => `\x1b[36m${v}\x1b[0m`;
  const g = (v) => `\x1b[32m✓\x1b[0m ${v}`;
  const r = (v) => `\x1b[31m✗\x1b[0m ${v || 'off'}`;
  const d = (v) => `\x1b[2m${v}\x1b[0m`;

  console.log(`\n  \x1b[1mmint config\x1b[0m ${d('(global — ~/.mint/config.json)')}\n`);

  if (Object.keys(config).length === 0) {
    console.log(`  ${d('No global config found.')}`);
    console.log(`  ${d('Set defaults with: mint config set --global <key> <value>')}`);
    console.log(`  ${d(`Supported keys: ${GLOBAL_KEYS.join(', ')}`)}`);
    console.log('');
    return;
  }

  if (config.isolation) console.log(`  Isolation:   ${c(config.isolation?.plan || 'none')}`);
  if (config.autoCommit !== undefined) console.log(`  Auto-commit: ${config.autoCommit !== false ? g('on') : r()}`);
  if (config.tdd) console.log(`  TDD:         ${config.tdd?.default ? g('on') : r()}`);
  if (config.modelRouting) console.log(`  Model route: ${config.modelRouting?.enabled !== false ? g('on') : r()}`);

  if (config.reviewers) {
    console.log(`\n  \x1b[1mReviewers\x1b[0m`);
    for (const [name, val] of Object.entries(config.reviewers)) {
      if (val && (val === true || val.enabled)) {
        const model = typeof val === 'object' ? d(` (${val.model})`) : '';
        console.log(`    ${name.padEnd(14)} ${g('')}${model}`);
      } else {
        console.log(`    ${name.padEnd(14)} ${r()}`);
      }
    }
  }

  console.log('');
}

function showConfig() {
  const config = loadConfig();
  const globalConfig = loadGlobalConfig();
  const hasGlobal = Object.keys(globalConfig).length > 0;
  const c = (v) => `\x1b[36m${v}\x1b[0m`;
  const g = (v) => `\x1b[32m✓\x1b[0m ${v}`;
  const r = (v) => `\x1b[31m✗\x1b[0m ${v || 'off'}`;
  const d = (v) => `\x1b[2m${v}\x1b[0m`;

  console.log(`\n  \x1b[1mmint config\x1b[0m${hasGlobal ? ` ${d('(global defaults active)')}` : ''}\n`);
  console.log(`  Stack:       ${c(config.stack || 'none')}`);
  console.log(`  PM:          ${c(config.packageManager || 'none')}`);
  console.log(`  Isolation:   ${c(config.isolation?.plan || 'none')}`);
  console.log(`  Auto-commit: ${config.autoCommit !== false ? g('on') : r()}`);
  console.log(`  TDD:         ${config.tdd?.default ? g('on') : r()}`);
  console.log(`  Browser:     ${config.browser?.enabled ? g('on') : r()}`);
  console.log(`  Context:     ${config.context?.enabled ? g('on') : r()}`);
  console.log(`  Design:      ${config.design?.enabled ? g('on') : r()}`);
  console.log(`  Signature:   ${config.signature ? g('on') : r()}`);

  // Doc-manifest
  const manifestPath = path.join(process.cwd(), '.mint', 'doc-manifest.json');
  if (fileExists(manifestPath)) {
    const manifest = readJsonSafe(manifestPath);
    const docCount = manifest?.docs?.length || 0;
    const sectionCount = manifest?.docs?.reduce((sum, d) => sum + (d.sections?.length || 0), 0) || 0;
    console.log(`  Docs:        ${g(`${docCount} docs, ${sectionCount} sections tracked`)}`);
  } else {
    console.log(`  Docs:        ${d('no manifest')}`);
  }

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

function setConfigValue(key, value, { global: isGlobal = false } = {}) {
  // Validate against schema
  const validation = validateConfigValue(key, value);
  if (!validation.valid) {
    p.log.error(validation.error);
    return;
  }

  const coerced = validation.coerced;
  const config = isGlobal ? loadGlobalConfig() : loadConfig();
  const parts = key.split('.');
  let target = config;
  for (let i = 0; i < parts.length - 1; i++) {
    if (target[parts[i]] === undefined) target[parts[i]] = {};
    target = target[parts[i]];
  }

  target[parts[parts.length - 1]] = coerced;

  if (isGlobal) {
    saveGlobalConfig(config);
    p.log.success(`Set \x1b[36m${key}\x1b[0m = \x1b[32m${String(coerced)}\x1b[0m \x1b[2m(global)\x1b[0m`);
  } else {
    saveConfig(config);
    p.log.success(`Set \x1b[36m${key}\x1b[0m = \x1b[32m${String(coerced)}\x1b[0m`);
  }
}

function showConfigList(scope) {
  const keys = listConfigKeys(scope);
  const d = (v) => `\x1b[2m${v}\x1b[0m`;

  console.log(`\n  \x1b[1mAvailable config keys\x1b[0m${scope ? ` (${scope})` : ''}\n`);

  let currentPrefix = '';
  for (const k of keys) {
    const prefix = k.key.split('.')[0];
    if (prefix !== currentPrefix) {
      if (currentPrefix) console.log('');
      currentPrefix = prefix;
    }
    const type = k.values ? k.values.join('|') : k.type;
    const def = k.default === false ? 'off' : k.default === true ? 'on' : String(k.default ?? '—');
    console.log(`  \x1b[36m${k.key.padEnd(35)}\x1b[0m ${d(type.padEnd(12))} ${d(`[${def}]`.padEnd(10))} ${k.description}`);
  }
  console.log('');
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
  const isGlobal = flags.global || false;

  if (!subcommand || subcommand === 'show') {
    if (isGlobal) {
      showGlobalConfig();
    } else {
      showConfig();
    }
    return;
  }

  if (subcommand === 'set') {
    const key = args[1];
    const value = args[2];
    if (!key || value === undefined) {
      console.log('  Usage: \x1b[36mmint config set [--global] <key> <value>\x1b[0m');
      console.log('  Example: \x1b[2mmint config set isolation.plan none\x1b[0m');
      console.log('  Example: \x1b[2mmint config set --global autoCommit false\x1b[0m');
      return;
    }
    setConfigValue(key, value, { global: isGlobal });
    return;
  }

  if (subcommand === 'list') {
    showConfigList(isGlobal ? 'global' : null);
    return;
  }

  if (subcommand === 'plugins') {
    await managePlugins(args[1], args[2]);
    return;
  }

  p.log.error(`Unknown: ${subcommand}. Try: show, set, list, plugins`);
}
