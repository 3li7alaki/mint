/**
 * Shell completion generator for mint CLI.
 *
 * Usage:
 *   mint completions bash > ~/.local/share/bash-completion/completions/mint
 *   mint completions zsh > ~/.zfunc/_mint
 *   mint completions install   # auto-detect and install
 */
import fs from 'fs';
import path from 'path';
import { listConfigKeys } from './lib/schema.js';

const COMMANDS = [
  'init', 'config', 'doctor', 'update', 'clean', 'status', 'plugin', 'help',
];

const CONFIG_SUBCOMMANDS = ['show', 'set', 'list', 'plugins'];
const PLUGIN_SUBCOMMANDS = ['list', 'add', 'remove', 'info'];

const CONFIG_KEYS = listConfigKeys().map(k => k.key);

export function generateBashCompletion() {
  return `# mint bash completion
_mint() {
  local cur prev commands
  COMPREPLY=()
  cur="\${COMP_WORDS[COMP_CWORD]}"
  prev="\${COMP_WORDS[COMP_CWORD-1]}"
  commands="${COMMANDS.join(' ')}"

  case "\${COMP_WORDS[1]}" in
    config)
      case "\${COMP_WORDS[2]}" in
        set)
          COMPREPLY=( $(compgen -W "${CONFIG_KEYS.join(' ')}" -- "$cur") )
          return 0
          ;;
        *)
          COMPREPLY=( $(compgen -W "${CONFIG_SUBCOMMANDS.join(' ')}" -- "$cur") )
          return 0
          ;;
      esac
      ;;
    plugin)
      COMPREPLY=( $(compgen -W "${PLUGIN_SUBCOMMANDS.join(' ')}" -- "$cur") )
      return 0
      ;;
    init)
      COMPREPLY=( $(compgen -W "--yes --isolation --tdd --browser --context --design --plugins" -- "$cur") )
      return 0
      ;;
    doctor)
      COMPREPLY=( $(compgen -W "--fix --quick" -- "$cur") )
      return 0
      ;;
    *)
      COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
      return 0
      ;;
  esac
}
complete -F _mint mint
`;
}

export function generateZshCompletion() {
  return `#compdef mint
# mint zsh completion

_mint() {
  local -a commands
  commands=(
    'init:Set up mint in the current project'
    'config:View or edit configuration'
    'doctor:Run health checks'
    'update:Update mint to latest version'
    'clean:Remove stale worktrees'
    'status:Quick health check'
    'plugin:Manage plugins'
    'help:Show help'
  )

  _arguments -C \\
    '1: :->command' \\
    '*::arg:->args'

  case "$state" in
    command)
      _describe 'command' commands
      ;;
    args)
      case "\${words[1]}" in
        config)
          local -a subcommands
          subcommands=(${CONFIG_SUBCOMMANDS.map(s => `'${s}'`).join(' ')})
          _describe 'subcommand' subcommands
          ;;
        plugin)
          local -a subcommands
          subcommands=(${PLUGIN_SUBCOMMANDS.map(s => `'${s}'`).join(' ')})
          _describe 'subcommand' subcommands
          ;;
        doctor)
          _arguments '--fix[Auto-repair issues]' '--quick[Skip gate execution]'
          ;;
        init)
          _arguments '--yes[No prompts]' '--isolation[Isolation mode]' '--tdd[Enable TDD]' '--browser[Browser support]'
          ;;
      esac
      ;;
  esac
}

_mint
`;
}

export function installCompletions() {
  const shell = path.basename(process.env.SHELL || 'bash');

  if (shell === 'zsh') {
    const dir = path.join(process.env.HOME || '', '.zfunc');
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(path.join(dir, '_mint'), generateZshCompletion());
    console.log(`  Installed zsh completions to ${dir}/_mint`);
    console.log(`  Add to ~/.zshrc if not already: fpath=(~/.zfunc $fpath) && autoload -Uz compinit && compinit`);
  } else {
    const dir = path.join(process.env.HOME || '', '.local', 'share', 'bash-completion', 'completions');
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(path.join(dir, 'mint'), generateBashCompletion());
    console.log(`  Installed bash completions to ${dir}/mint`);
    console.log(`  Restart your shell or run: source ${dir}/mint`);
  }
}
