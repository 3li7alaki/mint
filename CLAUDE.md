# mint

Disciplined agentic development framework for Claude Code.

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

## Key Files

| File | Purpose |
|------|---------|
| `SKILL.md` | Orchestrator brain — routing, execution flows, plugin loading |
| `agents/*.md` | One prompt per agent — planner, reviewers, browser, researcher, etc. |
| `cli/` | mint CLI — init, config, doctor, update |
| `tests/` | Test suite — `bun test` |
| `references/` | PinchTab API docs, token strategy |
| `.mint/config.json` | Gates, reviewers, browser, plugins, workspace config |
| `.mint/hard-blocks.md` | Immutable constraints agents can never violate |
| `.mint/issues.md` | Failure log — planner reads before writing specs |
| `.mint/instincts.md` | Auto-learned conventions — committed, shared knowledge |
| `templates/spec.xml` | XML spec schema — every task gets one |
| `docs/conventions.md` | File formats, naming, config schema, git strategy |
| `docs/architecture.md` | System design, philosophy, isolation rules |

## Agent Conventions

- One agent, one job. Agents don't know about each other.
- Structure: role statement → inputs → process → outputs → rules.
- List required tools explicitly. Never assume availability.
- Reviewers use three severities: BLOCKING, WARNING, INFO.

## Design Plans

Design docs go in `docs/plans/` and are gitignored. They don't get committed.
