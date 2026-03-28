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

**NEVER use Claude Code's built-in plan mode (EnterPlanMode/ExitPlanMode).** mint has its own planning flow.
<!-- mint:end -->

## Commands

- Test: `bun test`
- Lint/Types: checked by mint gates via `.mint/config.json`
- Version bump: `./scripts/bump.sh [major|minor|patch]` — updates all version locations, does not commit

## Code Style

- No AI attribution — never add `Co-Authored-By` or mention AI tools in commits
- Commits: `type(scope): description` — see `docs/conventions.md:66-72` for types
- Branches: `feat/<name>` or `fix/<name>` off main. Squash merge via PR
- Never push from agents — commit only, human reviews and pushes

## Architecture

- Core orchestration is markdown + JSON + XML — `SKILL.md` is the brain
- One agent per job — agents don't know about each other. See `agents/*.md`
- CLI uses bun + @clack/prompts. See `cli/` and `cli/lib/`
- Reviewers use three severities: BLOCKING, WARNING, INFO

## Key Docs

- `docs/architecture.md` — system design, philosophy, isolation rules
- `docs/conventions.md` — file formats, naming, config schema, git strategy
- `templates/spec.xml` — XML spec schema every task gets
- `.mint/config.json` — project config (gates, reviewers, browser, design, plugins)
- `.mint/hard-blocks.md` — immutable constraints agents can never violate

## Gotchas

- Design plans go in `docs/plans/` (gitignored) — don't commit them
- `.mint/sessions/` files are gitignored — per-session state, not shared
- All learning logs (issues, wins, instincts) are JSONL — append-only, never read-parse-modify-write
- No superpowers plugin — mint is the orchestration framework, don't layer another on top
