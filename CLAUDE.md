# mint

Disciplined agentic development framework for Claude Code.

<!-- mint:start v2 -->
## MANDATORY: Use mint for ALL Code Changes

**For ANY task that modifies files in this repo, invoke the `mint` skill FIRST.**

This is not optional. Before writing, editing, or deleting any code:
1. Invoke `mint` with the task description
2. mint auto-routes to the right mode (quick/plan/ship/research/verify)
3. Follow mint's execution flow with gates and reviews

The only exceptions:
- Pure conversation / answering questions
- Reading files to understand context (no modifications)

If you catch yourself thinking "this is just a small fix" or "I'll just edit one file" — STOP. Invoke mint. Small fixes use quick mode. mint decides the workflow, not you.

**NEVER use Claude Code's built-in plan mode (EnterPlanMode/ExitPlanMode).** mint has its own planning flow — Claude Code plan mode is redundant and conflicts with mint's orchestration. Always stay in normal mode and let mint handle planning via its plan/ship modes.
<!-- mint:end -->

## What This Is

A Claude Code skill (`SKILL.md`) + agent prompts (`agents/`) + CLI (`cli/`) + config (`.mint/`). The orchestrator auto-routes tasks to the right mode (quick/plan/ship/research/verify) and delegates to fresh subagents.

## Working Here

- **No superpowers.** This repo disables the superpowers plugin. Mint is the orchestration framework — don't layer another one on top.
- **No AI attribution.** Never add `Co-Authored-By` or mention AI tools in commits.
- **CLI deps only.** The CLI uses bun + @clack/prompts. Core orchestration is still markdown + JSON + XML.
- **Tests:** `bun test` — run before committing.
- **Commits:** `type(scope): description` — see `docs/conventions.md:66-72` for types.
- **Branches:** `feat/<name>` or `fix/<name>` off main. Squash merge via PR. Delete after merge.
- **Never push from agents.** Commit only. Human reviews and pushes.
- **Version bumps:** `./scripts/bump.sh [major|minor|patch]` — updates version in plugin.json, marketplace.json, package.json, and README.md. Does not commit.

## Key Files

| File | Purpose |
|------|---------|
| `SKILL.md` | Orchestrator brain — routing, execution flows, plugin loading |
| `agents/*.md` | One prompt per agent — planner, reviewers, browser, researcher, etc. |
| `cli/` | mint CLI — init, config, doctor, update |
| `tests/` | Test suite — `bun test` |
| `references/` | PinchTab API docs, token strategy, context-mode API, context-mode strategy |
| `agents/context-setup.md` | Context Mode setup agent — detection, installation, configuration |
| `standards/design/` | Design reference knowledge — typography, color, spatial, motion, interaction, responsive, ux-writing, RTL, i18n, anti-patterns |
| `agents/design-*.md` | Design agents — context (pre-plan), reviewer (pre-review), profile builder, setup |
| `~/.mint/config.json` | Global user defaults — reviewer models, autoCommit, TDD, isolation prefs |
| `.mint/config.json` | Project config — gates, reviewers, browser, context, design, plugins |
| `.mint/hard-blocks.md` | Immutable constraints agents can never violate |
| `.mint/issues.md` | Failure log — planner reads before writing specs |
| `.mint/instincts.md` | Auto-learned conventions — committed, shared knowledge |
| `.mint/.session-state.json` | Session state — mint invocation, autocommit override, task info (gitignored) |
| `.mint/doc-manifest.json` | Doc tracking manifest — maps doc sections to code dependencies |
| `commands/doc-setup.md` | Command to build doc-manifest for existing projects |
| `templates/doc-manifest.json` | Doc-manifest template for new projects |
| `templates/spec.xml` | XML spec schema — every task gets one |
| `hooks/scripts/pre-edit-mint-check.cjs` | PreToolUse hook — warns if mint not invoked before file edits |
| `scripts/bump.sh` | Version bump script — updates all version locations |
| `docs/conventions.md` | File formats, naming, config schema, git strategy |
| `docs/architecture.md` | System design, philosophy, isolation rules |

## Agent Conventions

- One agent, one job. Agents don't know about each other.
- Structure: role statement → inputs → process → outputs → rules.
- List required tools explicitly. Never assume availability.
- Reviewers use three severities: BLOCKING, WARNING, INFO.

## Design Plans

Design docs go in `docs/plans/` and are gitignored. They don't get committed.
