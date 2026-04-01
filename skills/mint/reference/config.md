# Configuration — Reference

Load when handling config changes, CLI commands, or plugin setup.

---

## Two-Layer Config

1. **Project:** `.mint/config.json` — always takes precedence
2. **Global:** `~/.mint/config.json` — fallback for user preferences
3. **Defaults:** built-in if neither layer has the key

Global supports: reviewers, autoCommit, tdd, isolation, modelRouting, instincts, hooks, definitionOfDone.
Project-specific (not inherited): stack, packageManager, gates, browser, context, design, plugins.

## CLI Commands

| Command | Purpose |
|---------|---------|
| `mint init` | Claude reads project, creates config |
| `mint init --yes` | Headless auto-detect |
| `mint config` | Display config |
| `mint config set <key> <value>` | Edit with dot notation |
| `mint config set --global <key> <value>` | Set global default |
| `mint doctor` | Health check |
| `mint doctor --fix` | Health check + fix |
| `mint update` | Update mint + migrate projects |
| `mint status` | Quick health check |
| `mint dream` | Learning consolidation — status overview |
| `mint dream status` | Dream status and entry counts |
| `mint dream decay` | Run instinct decay |
| `mint dream instincts` | List all instincts with scores |
| `mint clean` | Remove stale worktrees + sessions |

## Multi-Model Dispatch

Reviewers specify models in config:
- `true` = enabled, session default model
- `{ "enabled": true, "model": "sonnet" }` = specific model
- Valid: `"opus"`, `"sonnet"`, `"haiku"`

## Plugin Loading

Read `config.plugins` array. Each plugin has `manifest.json` with name, type, agents.
Agents namespaced as `plugin-name:agent-name`.

Hook points: `pre-plan`, `post-plan`, `pre-review`, `post-commit`, `on-init`.

## Doc-Manifest

`.mint/doc-manifest.json` tracks which docs depend on which code.
Staleness strategies: `glob-count`, `content-hash`, `git-diff`.
