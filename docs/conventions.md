# mint Conventions

## File Format

| Content type | Format | Why |
|---|---|---|
| Agent prompts | Markdown (.md) | Static system prompt per agent. Max 150 lines. Cached by API. |
| Context templates | Markdown (.md) | Dynamic prompt schemas in `templates/agent-context.md`. Per-dispatch inputs only. |
| Skill router | Markdown (.md) | Thin router (~125 lines). Loads modes/phases on demand. |
| Mode/phase files | Markdown (.md) | One per mode/pipeline step. Loaded individually. |
| Task specs | XML (.xml) | Structured, parseable, validated |
| Execution state | JSON (.json) | Per-spec tracking, machine-readable, resumable |
| Pipeline state | JSON (.json) | Per-spec pipeline step tracker (plan mode) |
| Config | JSON (.json) | Zero ambiguity, schema-validated. Has `mintVersion` field. |
| Learning logs | JSONL (.jsonl) | Append-only, concurrent-safe, grep-able |
| Instincts | JSONL (.jsonl) | Scored format: confidence, occurrences, sources, decay |
| Execution metrics | JSONL (.jsonl) | Per-spec performance data for evidence-based evolution |
| Session state | JSON (.json) | Single-writer, read by hooks. Timestamp-prefixed IDs. |
| Freeze/guard list | JSON (.json) | Read-heavy by hooks on every Edit/Write |
| Gate ledger | JSONL (.jsonl) | Concurrent append by parallel agents |
| Documentation | Markdown (.md) | Standard, renderable everywhere |
| Hard blocks | Markdown (.md) | Human-authored constraints |
| Standards | Markdown (.md) | Reference docs for writing CLAUDE.md and agent prompts |
| Migrations | JavaScript (.js) | Versioned migration steps in `cli/migrations/` |

### JSONL Format

Learning logs (issues, wins, patterns, instincts) and the gate ledger use JSONL:

```jsonl
{"date":"2025-03-25","task":"auth-003","severity":"BLOCKING","issue":"Scope leak","rootCause":"scope-leak","resolution":"Split into two specs"}
{"date":"2025-03-25","task":"auth","pattern":"Split API + UI specs","why":"Kept context focused"}
```

- **Append:** `appendJsonl(file, entry)` from `cli/lib/jsonl.js`
- **Read:** `readJsonl(path)` — skips blank lines and invalid JSON
- **Query:** `queryJsonl(path, predicate)` for filtered reads
- **Instincts:** `upsertInstinct(path, { category, observation, source, examples })` — deduplicates by category+observation, increments confidence
- **Decay:** `decayInstincts(path, maxAgeDays)` — reduces confidence on stale instincts, removes at 0
- **Top-N:** `topInstincts(path, maxCount, minConfidence)` — for planner injection
- **Metrics:** `logMetric(path, { task, spec, instinctsApplied, reviewResult, gateResults, attempts })`
- **Analyze:** `analyzeInstinctEffectiveness(path)` — correlates instinct usage with pass rates
- **Atomic:** Lines under 4KB are atomic on POSIX — safe for concurrent append

## Agent Prompts

See `standards/agent-prompts.md` for the full reference on writing effective agent prompts.

### Structure

Every agent file follows this order:

1. **Role** — one-line role statement
2. **Inputs** — what the orchestrator provides
3. **Process** — numbered steps (max 10)
4. **Output** — exact return format
5. **Rules** — constraints (max 10, most important first)

### Naming

- Agent files: `kebab-case.md` (e.g., `spec-reviewer.md`, `quality-reviewer.md`)
- Agent dispatch names: `mint-kebab-case` (e.g., `mint-spec-reviewer`)
- Plugin agent names: `plugin-name:agent-name` (e.g., `mint-nuxt:nuxt-reviewer`)

### Conditional Sections

Agents may include conditional sections gated by config flags. The standard pattern:

```markdown
## Context Mode

When `config.context.enabled` is `true` and context-mode MCP tools are available, prefer
sandboxed execution to keep raw output out of context:

- [specific tool mappings for this agent]
- If context-mode tools are unavailable, fall back to standard tools transparently.
```

Conditional tools are listed in the frontmatter with `(conditional)` suffix.

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

`estimate`, `depends-on`, `pre-conditions`, `context`, `references`, `pitfalls`, `anti-patterns`, `no-mocks`, `tests`, `workspace-impact`, `tdd`, `test-first`.

### Scope Rules

- `<can-modify>` — exhaustive list of files the agent may touch
- `<cannot-modify>` — explicit exclusions (use globs: `src/api/*`)
- Agents that modify files outside scope are in violation — the spec reviewer catches this

### Commit Messages

All commits: `type(scope): description`

Scope is the component or area changed (e.g., `planner`, `auth`, `api`). Do NOT use mint
task IDs (`mint-NNN`) or include spec file paths in commit messages.

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

### PRs

- **Always squash merge** — PRs merge as a single squash commit into main.
- **Delete branch after merge** — clean up remote and local branches.
- **PR before merge** — all work goes through a PR, even solo work. No direct merges.
- **Version bump** — before creating a PR, run `./scripts/bump.sh minor` (features) or `./scripts/bump.sh patch` (fixes). Never edit version strings manually — the script updates all 3 files (package.json, plugin.json, marketplace.json).

### Post-Merge

After a PR is merged:
1. `git checkout main && git pull && git fetch --prune`
2. Delete the local feature branch
3. Verify main is clean

## Config Schema

mint uses a two-layer config system:

- **Global config** (`~/.mint/config.json`) — user-level defaults, apply across all projects
- **Project config** (`.mint/config.json`) — project-specific settings, override global

Resolution order: project config > global config > hardcoded defaults.

### Global-eligible keys

These keys can be set globally (they're user preferences, not project-specific):
`reviewers`, `autoCommit`, `tdd`, `isolation`, `modelRouting`, `instincts`, `hooks`, `definitionOfDone`.

Set via: `mint config set --global <key> <value>`

Project-specific keys (`stack`, `packageManager`, `gates`, `browser`, `context`, `design`, `plugins`, `workspace`) must be set per-project — they are not inherited from global config.

### Project config keys

`.mint/config.json` keys:

| Key | Type | Purpose |
|---|---|---|
| `stack` | string | Detected framework |
| `packageManager` | string | npm, pnpm, yarn, bun |
| `gates` | object | lint/types/tests commands |
| `gates.coverage` | object | Coverage gate: `{ "command": "...", "threshold": 80 }` |
| `reviewers` | object | Which reviewers are enabled — values can be `true`/`false` or `{ enabled, model }` |
| `conventions.docs` | array | Paths to convention docs the enforcer reads |
| `business.docs` | array | Paths to business docs the reviewer reads |
| `isolation` | object | Isolation per mode: `"worktree"`, `"branch"`, or `"none"` |
| `definitionOfDone` | object | Completion criteria checklist (gates, reviews, screenshot reminder) |
| `documenters` | array | Auto-doc configurations |
| `plugins` | array | Plugin directory paths |
| `autoCommit` | boolean | Global default for autocommit. Overridden by session state `autoCommitOverride` and per-spec `<autoCommit>`. Default: `true` |
| `tdd.default` | boolean | If `true`, all specs get TDD by default. Individual specs can override. Default: `false` |
| `tdd.desloppify` | boolean | Run de-sloppify pass after TDD implementation. Default: `true` |
| `tdd.coverageThreshold` | number | Default coverage threshold for TDD specs. Default: `80` |
| `instincts.enabled` | boolean | Enable instinct-based learning from hooks. Default: `true` |
| `modelRouting.enabled` | boolean | Auto-select model per spec complexity. Default: `true` |
| `modelRouting.override` | object | Map estimate → model (e.g., `{ "small": "opus" }`) |
| `hooks.testOnSave` | boolean/object | Auto-run tests on edit. `true`, `false`, or `{ enabled, timeout }`. Default: `false` |
| `workspace.repos` | array | Workspace repo registry (see below) |
| `context.enabled` | boolean | Enable/disable Context Mode (sandboxed execution via context-mode). Default: `false` |
| `context.autoRoute` | boolean | Auto-route data-heavy operations to sandbox. Default: `true` |
| `context.sandbox.timeout` | number | Sandbox execution timeout in ms. Default: `30000` |
| `context.session.enabled` | boolean | Enable session continuity via context-mode hooks. Default: `true` |
| `browser.enabled` | boolean | Enable/disable browser plugin. Default: `true` |
| `browser.baseUrl` | string | PinchTab API base URL. Default: `http://localhost:9867` |
| `browser.token` | string | Bearer token for PinchTab auth. Default: `null` |
| `browser.headless` | boolean | Run Chrome headless. Default: `true` |
| `browser.devServer` | string | Local dev server URL. Default: `http://localhost:3000` |
| `browser.uiFilePatterns` | string[] | File patterns that trigger browser review. Default: `["*.vue", "*.tsx", "*.jsx", "*.svelte", "*.html", "*.css", "*.scss"]` |
| `browser.reviewMode` | string | When to run browser review: `"auto"`, `"always"`, `"off"`. Default: `"auto"` |
| `browser.timeout` | number | Navigation timeout in seconds. Default: `30` |
| `browser.blockImages` | boolean | Block image loading globally. Default: `false` |
| `design.enabled` | boolean | Enable/disable design intelligence. Default: `true` |
| `design.stack` | string | Frontend stack for design context. `"auto"` or explicit. Default: `"auto"` |
| `design.profile` | string | Path to design profile JSON. Default: `".mint/design-profile.json"` |
| `design.notes` | string | Path to design notes markdown. Default: `".mint/design-notes.md"` |
| `design.conventions` | string[] | Paths to design convention docs (brand guides, design systems). Default: `[]` |
| `design.uiFilePatterns` | string[] | File patterns that trigger design context (fallback when keywords aren't in task description). Default: `["*.tsx", "*.jsx", "*.vue", "*.svelte", "*.css", "*.scss", "*.html"]` |
| `design.review.accessibility` | boolean | WCAG 2.1 AA checks. Default: `true` |
| `design.review.consistency` | boolean | Design system adherence checks. Default: `true` |
| `design.review.performance` | boolean | Animation and bundle performance checks. Default: `true` |
| `design.review.rtl` | boolean | RTL logical property enforcement. Default: `false` |
| `design.review.i18n` | boolean | Internationalization string checks. Default: `false` |
| `design.review.brand` | boolean | Brand guide compliance checks. Default: `false` |
| `definitionOfDone.docCheckPassed` | boolean | Whether doc-manifest check is required after each spec. Default: `true` |
| `signature` | boolean | Add "Minted with mint" signature to README.md when asked. Default: `false` |
| `mintVersion` | string | Current mint version for migration tracking. Set by `mint init` or `mint update`. |
| `lastUpdated` | string | ISO-8601 timestamp of last migration run. |
| `appliedMigrations` | string[] | List of applied migration IDs (e.g., `["0.7.6-to-0.8.0"]`). |

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
- New doc section tracked → update .mint/doc-manifest.json
- Doc-manifest system changed → update docs/conventions.md, docs/architecture.md

### Adding a Core Feature (like browser, context, design)

Core features are toggleable capabilities with their own config, agents, and CLI integration. When adding one, hit every touchpoint:

| # | File | What to do |
|---|------|------------|
| 1 | `agents/<feature>-*.md` | Create core agents (context, reviewer, setup, etc.) |
| 2 | `commands/<feature>*.md` | Create user-facing commands |
| 3 | `standards/<feature>/` | Add reference docs, standards if applicable |
| 4 | `skills/mint/SKILL.md` | Add routing decision. Create `modes/<feature>.md` for execution flow. Add stage 2 reviewer dispatch in `phases/review-stage2.md` if applicable. |
| 5 | `cli/commands/init.js` | Add confirm prompt, install hook (if external dep), add to `buildConfig()`, add to headless mode |
| 6 | `cli/commands/doctor.js` | Add health checks (enabled status, deps installed, assets present) |
| 7 | `cli/commands/config.js` | Add status display line, remove from plugins list if migrated |
| 8 | `cli/commands/update.js` | Add to `NEW_CONFIG_KEYS` (for upgrade prompts), add dep updater to `DEPS`, add update dispatch |
| 9 | `docs/conventions.md` | Add all config keys to schema table |
| 10 | `docs/architecture.md` | Add feature section explaining how it works |
| 11 | `CLAUDE.md` | Update key files table |
| 12 | `./scripts/bump.sh minor` | Bump version in all 3 files (never manually) |

**Config pattern**: top-level key with `enabled` boolean + feature-specific settings. Not under `plugins`.

**Agent pattern**: agents live in `agents/` prefixed with feature name (`design-context.md`, `browser-runner.md`). Not in a plugin directory.

**Hooks**: core features use hardcoded hook logic in SKILL.md (pre-plan, pre-review, startup detection), not the generic plugin hook system.

### Where Things Live

| Doc | Purpose | Updated by |
|---|---|---|
| `README.md` | Public overview, install, pipeline | Manual or documenter |
| `SKILL.md` | Orchestrator prompt — the brain | Manual only (careful) |
| `CONTRIBUTING.md` | How to contribute | Manual |
| `docs/conventions.md` | This file — internal conventions | Manual or conventions-enforcer discovery |
| `docs/architecture.md` | System design and agent roles | Manual on architectural changes |
| `.mint/doc-manifest.json` | Doc tracking manifest — section→code mappings | `mint init` / `/doc-setup` |

### Doc-Manifest

The doc-manifest (`.mint/doc-manifest.json`) maps documentation sections to code dependencies:

- **Schema:** `doc-manifest-v1` — see `templates/doc-manifest.json`
- **Created by:** `mint init` (basic) or `/doc-setup` command (comprehensive)
- **Read by:** Orchestrator (completion protocol), verifier (staleness check), documenter (update guidance)
- **Committed:** Yes — shared team knowledge

Each section entry has:
| Field | Purpose |
|-------|---------|
| `id` | Unique kebab-case identifier |
| `heading` | Markdown heading to locate the section |
| `tracks` | Glob patterns of code files this section depends on |
| `staleness` | Detection strategy: `glob-count`, `content-hash`, `git-diff` |
| `description` | What this section must contain (guides the documenter) |

### Workflow Traces (`.mint/workflow-traces.jsonl`)

```jsonl
{"date":"2026-04-01","session":"a1b2c3","commands":["bun test","edit src/auth.ts","bun test"],"duration":45,"outcome":"pass","fingerprint":"test-edit-test-abc123"}
```

| Field | Type | Purpose |
|-------|------|---------|
| `date` | string | ISO-8601 date |
| `session` | string | Session ID that produced the trace |
| `commands` | string[] | Ordered sequence of commands/actions |
| `duration` | number | Seconds from first to last command |
| `outcome` | string | `"pass"`, `"fail"`, or `"abandoned"` |
| `fingerprint` | string | Normalized hash for clustering |

### Workflow Candidates (`.mint/workflow-candidates.jsonl`)

```jsonl
{"id":"test-fix-commit","pattern":["test","fix","test","commit"],"occurrences":5,"firstSeen":"2026-03-15","lastSeen":"2026-04-01","status":"pending"}
```

| Field | Type | Purpose |
|-------|------|---------|
| `id` | string | Kebab-case candidate identifier |
| `pattern` | string[] | Normalized command sequence |
| `occurrences` | number | Times this pattern was observed |
| `firstSeen` | string | ISO-8601 date of first observation |
| `lastSeen` | string | ISO-8601 date of most recent observation |
| `status` | string | `"pending"`, `"accepted"`, `"dismissed"` |

### Generated Skills (`.mint/skills/<name>/`)

Each generated skill is a directory:

```
.mint/skills/<name>/
  manifest.json     ← metadata, trust level, usage stats
  SKILL.md          ← the skill prompt (Claude-readable)
  test.md           ← validation criteria
```

**Naming:** `<name>` is kebab-case, derived from the candidate ID (e.g., `test-fix-commit`).

**manifest.json schema:**

| Field | Type | Purpose |
|-------|------|---------|
| `name` | string | Skill name (matches directory name) |
| `description` | string | One-line description |
| `trust` | string | `"suggest"`, `"confirm"`, or `"auto"` |
| `sourceCandidate` | string | Candidate ID this skill was generated from |
| `usageCount` | number | Times successfully executed |
| `createdAt` | string | ISO-8601 creation timestamp |
| `lastUsed` | string | ISO-8601 last execution timestamp |

Generated skills are gitignored. Promote to `.claude/skills/` to share.
