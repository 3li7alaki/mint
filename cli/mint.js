#!/usr/bin/env bun
import { log } from '@clack/prompts';
import { readFileSync } from 'fs';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const VERSION = JSON.parse(readFileSync(join(__dirname, '..', 'package.json'), 'utf8')).version;

const LOGO = `
\x1b[32m             __         __
   ____ ___ /\\_\\ ___   /\\ \\__
  /\\  _\` \\/\\ \\/  _\`\\ \\ \\ ,_\\
  \\ \\ \\/\\ \\\\ \\ \\ \\ \\/\\ \\\\ \\ \\/
   \\ \\ \\ \\ \\\\ \\ \\ \\ \\ \\ \\\\ \\ \\_
    \\ \\_\\ \\_\\\\ \\_\\ \\_\\ \\_\\\\ \\__\\
     \\/_/\\/_/ \\/_/\\/_/\\/_/ \\/__/\x1b[0m
`;

function showHelp() {
  console.log(LOGO);
  console.log(`  \x1b[2mv${VERSION} — disciplined agentic development\x1b[0m\n`);
  console.log('  \x1b[1mCommands:\x1b[0m\n');
  console.log('    \x1b[36mmint init\x1b[0m                Set up mint in the current project');
  console.log('    \x1b[36mmint init --yes\x1b[0m          Auto-detect everything, no prompts');
  console.log('    \x1b[36mmint config\x1b[0m              Show current configuration');
  console.log('    \x1b[36mmint config --global\x1b[0m     Show global user defaults');
  console.log('    \x1b[36mmint config set\x1b[0m k v      Set a config value (dot notation)');
  console.log('    \x1b[36mmint config set --global\x1b[0m  Set a global default');
  console.log('    \x1b[36mmint config plugins\x1b[0m      Manage plugins');
  console.log('    \x1b[36mmint doctor\x1b[0m              Health check — static checks + tiered output');
  console.log('    \x1b[36mmint doctor --fix\x1b[0m        Health check + Claude applies context-aware fixes');
  console.log('    \x1b[36mmint update\x1b[0m              Update mint + Claude migrates config intelligently');
  console.log('    \x1b[36mmint update --deps\x1b[0m       Update core deps (PinchTab, context-mode)');
  console.log('    \x1b[36mmint update <dep>\x1b[0m        Update one dep (pinchtab, context-mode)');
  console.log('    \x1b[36mmint clean\x1b[0m               Remove stale worktrees from parallel execution');
  console.log('    \x1b[36mmint status\x1b[0m              Quick health check (instant, no gate runs)');
  console.log('    \x1b[36mmint plugin list\x1b[0m         Browse available plugins');
  console.log('    \x1b[36mmint plugin add <name>\x1b[0m   Install a plugin');
  console.log('    \x1b[36mmint plugin info <name>\x1b[0m  Plugin details');
  console.log('');
  console.log('  \x1b[1mExamples:\x1b[0m\n');
  console.log('    \x1b[2m$ mint init\x1b[0m');
  console.log('    \x1b[2m$ mint init --yes --plugins mint-nuxt,mint-e2e\x1b[0m');
  console.log('    \x1b[2m$ mint config set isolation.plan none\x1b[0m');
  console.log('    \x1b[2m$ mint config set --global autoCommit false\x1b[0m');
  console.log('    \x1b[2m$ mint doctor\x1b[0m');
  console.log('');
}

function parseFlags(args) {
  const flags = {};
  const positional = [];
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--yes' || args[i] === '-y') {
      flags.yes = true;
    } else if (args[i] === '--global') {
      flags.global = true;
    } else if (args[i].startsWith('--') && i + 1 < args.length && !args[i + 1].startsWith('--')) {
      flags[args[i].slice(2)] = args[i + 1];
      i++;
    } else if (args[i].startsWith('--')) {
      flags[args[i].slice(2)] = true;
    } else {
      positional.push(args[i]);
    }
  }
  return { flags, positional };
}

const args = process.argv.slice(2);
const { flags, positional } = parseFlags(args);
const command = positional[0] || (flags.version ? '--version' : (flags.help ? '--help' : undefined));

try {
  switch (command) {
    case 'init': {
      const { run } = await import('./commands/init.js');
      await run(flags);
      break;
    }
    case 'config': {
      const { run } = await import('./commands/config.js');
      await run(positional.slice(1), flags);
      break;
    }
    case 'doctor': {
      const { run } = await import('./commands/doctor.js');
      await run(flags);
      break;
    }
    case 'update': {
      const { run } = await import('./commands/update.js');
      await run(positional.slice(1), flags);
      break;
    }
    case 'clean': {
      const { run } = await import('./commands/clean.js');
      await run(positional.slice(1), flags);
      break;
    }
    case 'status': {
      const { run } = await import('./commands/status.js');
      await run(positional.slice(1), flags);
      break;
    }
    case 'plugin': {
      const { run } = await import('./commands/plugin.js');
      await run(positional.slice(1), flags);
      break;
    }
    case 'completions': {
      const { generateBashCompletion, generateZshCompletion, installCompletions } = await import('./completions.js');
      const sub = positional[1];
      if (sub === 'bash') console.log(generateBashCompletion());
      else if (sub === 'zsh') console.log(generateZshCompletion());
      else if (sub === 'install') installCompletions();
      else console.log('  Usage: mint completions bash|zsh|install');
      break;
    }
    case 'help':
    case '--help':
    case '-h':
    case undefined:
      showHelp();
      break;
    case '--version':
    case '-v':
      console.log(`mint v${VERSION}`);
      break;
    default:
      log.error(`Unknown command: ${command}`);
      console.log(`  Run \x1b[36mmint help\x1b[0m for available commands.\n`);
      process.exit(1);
  }
} catch (err) {
  if (err?.message?.includes('cancel') || err?.message === 'readline was closed') {
    console.log('\n');
    process.exit(0);
  }
  console.error(`\n\x1b[31m${err.message}\x1b[0m`);
  if (process.env.DEBUG) console.error(err.stack);
  process.exit(1);
}
