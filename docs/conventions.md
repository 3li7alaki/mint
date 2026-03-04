# mint Conventions

## File Format

| Content type | Format | Why |
|---|---|---|
| Agent prompts | Markdown (.md) | What Claude expects as input |
| Task specs | XML (.xml) | Structured, parseable, validated |
| Execution state | JSON (.json) | Per-spec tracking, machine-readable, resumable |
| Config | JSON (.json) | Zero ambiguity, machine-readable |
| Issue log | Markdown table (.md) | Human-readable, git-diffable |
| Documentation | Markdown (.md) | Standard, renderable everywhere |

## Agent Prompts

### Structure

Every agent file follows this order:

1. **Role statement** — "You are the mint X agent."
2. **What you receive** — inputs the orchestrator provides
3. **What you do** — step-by-step process
4. **What you return** — exact output format
5. **Rules** — constraints and non-negotiables

### Naming

- Agent files: `kebab-case.md` (e.g., `spec-reviewer.md`, `quality-reviewer.md`)
- Agent dispatch names: `mint-kebab-case` (e.g., `mint-spec-reviewer`)
- Plugin agent names: `plugin-name:agent-name` (e.g., `mint-nuxt:nuxt-reviewer`)

### Tool Lists

Every agent explicitly lists the tools it needs. Never assume tool availability.

```markdown
**Tools you need:** Read, Write, Edit, Glob, Grep, Bash
```

### Severity Levels (Reviewers)

All reviewer agents use three severity levels:

| Level | Meaning | Action |
|---|---|---|
| BLOCKING | Must fix before commit | Planner fixes, reviewer re-reviews |
| WARNING | Should fix, won't block | Planner fixes in same pass |
| INFO | Noted, logged only | Added to `.mint/issues.md` for learning |

## XML Specs

### Required Fields

Every spec must have: `id`, `title`, `goal`, `scope` (with `can-modify` and `cannot-modify`), `steps`, `acceptance`, `commit`, `gates`.

### Optional Fields

`estimate`, `depends-on`, `pre-conditions`, `context`, `references`, `pitfalls`, `anti-patterns`, `no-mocks`, `tests`, `workspace-impact`.

### Scope Rules

- `<can-modify>` — exhaustive list of files the agent may touch
- `<cannot-modify>` — explicit exclusions (use globs: `src/api/*`)
- Agents that modify files outside scope are in violation — the spec reviewer catches this

### Commit Messages

Spec-driven commits: `type(mint-NNN): description`
Non-spec commits: `type(scope): description`

## Git Strategy

### Branching

- `main` — stable, always clean. Never commit directly.
- `feat/<name>` — feature branches off main. One feature per branch.
- `fix/<name>` — bug fix branches off main.

### Commits

- **Atomic commits** — one spec = one commit. Don't bundle multiple specs.
- **Commit message format** — `type(scope): description` (see above)
- **Never push from agents** — agents commit only. Human reviews and pushes.
- **No AI attribution** — never add "Co-Authored-By" or mention AI tools in commits.

### Merging

- **Always squash merge** — PRs merge as a single squash commit into main.
- **Delete branch after merge** — clean up remote and local branches.
- **PR before merge** — all work goes through a PR, even solo work. No direct merges.

### Post-Merge

After a PR is merged:
1. `git checkout main && git pull && git fetch --prune`
2. Delete the local feature branch
3. Verify main is clean

## Config Schema

`.mint/config.json` keys:

| Key | Type | Purpose |
|---|---|---|
| `stack` | string | Detected framework |
| `packageManager` | string | npm, pnpm, yarn, bun |
| `gates` | object | lint/types/tests commands |
| `reviewers` | object | Which reviewers are enabled |
| `conventions.docs` | array | Paths to convention docs the enforcer reads |
| `business.docs` | array | Paths to business docs the reviewer reads |
| `isolation` | object | Worktree/branch strategy per mode |
| `definitionOfDone` | object | Completion criteria checklist (gates, reviews, screenshot reminder) |
| `documenters` | array | Auto-doc configurations |
| `plugins` | array | Plugin directory paths |
| `workspace.repos` | array | Workspace repo registry (see below) |

### Workspace Registry (`workspace.repos`)

Each entry in `workspace.repos` describes a repository in the workspace:

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Human-readable identifier (e.g., `"my-app"`) |
| `path` | string | yes | Relative or absolute path to the repo root |
| `stack` | string | yes | Detected or declared framework (`nuxt`, `react`, `typescript`, `python`, etc.) |
| `role` | string | yes | One of `"primary"`, `"dependency"`, `"reference"` |
| `dependsOn` | string[] | no | Array of other repo `name` values this repo depends on |

## Documentation

### When to Update

- New agent → update agents table in README
- New command → update CONTRIBUTING.md project structure
- New config key → update docs/conventions.md config schema table
- Pipeline change → update SKILL.md and README pipeline diagram
- New plugin hook → update SKILL.md plugin loading section

### Where Things Live

| Doc | Purpose | Updated by |
|---|---|---|
| `README.md` | Public overview, install, pipeline | Manual or documenter |
| `SKILL.md` | Orchestrator prompt — the brain | Manual only (careful) |
| `CONTRIBUTING.md` | How to contribute | Manual |
| `docs/conventions.md` | This file — internal conventions | Manual or conventions-enforcer discovery |
| `docs/architecture.md` | System design and agent roles | Manual on architectural changes |
