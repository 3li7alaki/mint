```
 ███╗   ███╗██╗███╗   ██╗████████╗
 ████╗ ████║██║████╗  ██║╚══██╔══╝
 ██╔████╔██║██║██╔██╗ ██║   ██║
 ██║╚██╔╝██║██║██║╚██╗██║   ██║
 ██║ ╚═╝ ██║██║██║ ╚████║   ██║
 ╚═╝     ╚═╝╚═╝╚═╝  ╚═══╝   ╚═╝
```

![Version](https://img.shields.io/github/v/release/3li7alaki/mint?label=version)
![Stars](https://img.shields.io/github/stars/3li7alaki/mint?style=flat)
![License](https://img.shields.io/github/license/3li7alaki/mint)
![CI](https://github.com/3li7alaki/mint/actions/workflows/ci.yml/badge.svg)

### Disciplined agentic development for Claude Code

> Self-evolving skill architecture. Pipeline enforcement. Scored instincts. Zero slop.

**Core philosophy:** Slop is an engineering problem, not an LLM problem. If an agent produces bad code, fix the environment — never patch the output.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/3li7alaki/mint/main/install.sh | bash
```

This installs the `mint` CLI globally and (if Claude Code is installed) the Claude plugin for auto-routing.

### CLI

```bash
mint init                       # Claude reads your project, configures mint perfectly
mint init --yes                 # headless — auto-detect, no prompts (CI/scripts)
mint config                     # view current config
mint config --global            # view global user defaults
mint config set key value       # edit project config (dot notation, validated)
mint config set --global k v    # set a global default
mint config list                # show all available config keys with types and defaults
mint doctor                     # health check — static checks + tiered output
mint doctor --fix               # health check + Claude applies context-aware fixes
mint update                     # update mint + Claude migrates ALL registered projects
mint clean                      # remove stale worktrees from parallel execution
```

### Global Defaults

Set user preferences that apply to all projects:

```bash
mint config set --global autoCommit false
mint config set --global reviewers.security.model opus
mint config set --global isolation.plan worktree
```

Global config lives at `~/.mint/config.json`. Project config always overrides global. Global config also tracks a **project registry** — every `mint init` registers the project, and `mint update` migrates all registered projects at once.

### Project Setup

Run `mint init` in your project (seeds from global defaults if set):

```
.mint/
├── config.json             — gates, reviewers, browser, design, plugins + mintVersion
├── hard-blocks.md          — what agents can never do
├── issues.jsonl            — failure log with root cause categories
├── wins.jsonl              — success patterns
├── patterns.jsonl          — graduated recurring patterns
├── instincts.jsonl         — scored conventions (confidence, dedup, decay)
├── metrics.jsonl           — per-spec execution metrics for evidence-based evolution
├── sessions/               — per-session state (gitignored, timestamp-prefixed IDs)
├── tasks/<session-id>/<slug>/ — per-session spec XMLs + execution.json + pipeline-state.json
├── .freeze-list.json       — frozen/guarded paths (gitignored)
├── .browser-sessions.json  — browser cookies (gitignored)
└── .gate-ledger.jsonl      — gate run dedup (gitignored)
```

## How It Works

You describe what you want. mint auto-detects the right approach:

| What you say | What mint does |
|---|---|
| Small fix, typo, config tweak | **Quick** — fixes directly, gates enforced |
| Feature, component, API route | **Plan** — decomposes into XML specs, executes atomically |
| Multiple features, batch of work | **Ship** — interviews you, plans phases, executes all |
| "Browse to", "scrape", "debug in browser" | **Browse** — PinchTab-powered browser automation |
| "How should I...", "Compare..." | **Research** — investigates, saves structured report |
| "Check quality", "Audit" | **Verify** — runs all gates and audits |
| "Dream", "Consolidate learning" | **Dream** — prunes instincts, triages issues, promotes patterns, health report |
| "Automate", "Detect workflows" | **Automate** — mine patterns, generate skills, graduate trust |
| "Design review", "Design profile" | **Design** — design intelligence commands |
| "Set up doc tracking" | **Doc Setup** — scans docs, maps sections to code, builds manifest |
| "Optimize my setup", "Am I using mint fully?" | **Optimize** — full audit of config, docs, workspace, agents, features |
| `/freeze src/auth/` | **Freeze** — blocks all file modifications under that path |
| `/guard package.json "no new deps"` | **Guard** — freeze + reason shown to agents |
| `/unfreeze --all` | **Unfreeze** — remove all freezes |

No commands to memorize. Just describe what you want to build.

## The Pipeline

```
You describe a feature
        │
  Router (SKILL.md, ~125 lines) → loads mode file
        │
  Challenge (optional) — is this the right thing to build?
        │
  Decomposer agent → XML specs with dependency graph
        │
  Build wave plan from <depends-on>
        │
  Per spec — state-machine pipeline (each step loads its own phase file):
    ┌─ implement  → Planner agent (gates + commit)
    ├─ desloppify → De-sloppifier (conditional)
    ├─ review-s1  → Spec reviewer (mandatory gate)
    ├─ review-s2  → Parallel auditors (scaled by diff size)
    ├─ fix        → Fix BLOCKINGs (if any)
    ├─ docs       → Documenter (if manifest matches)
    └─ dod        → Definition of Done verification
        │
  Pipeline-complete hook blocks stop if steps were skipped
        │
  You review the final result
```

**Parallel execution:** Independent specs run simultaneously — either as parallel Agent calls
within one session, or as separate `claude -p` processes in isolated git worktrees. Concurrency
is configurable (default: 3). Scope enforcement prevents parallel specs from modifying the same files.

**Prompt caching:** Agent prompts are split into static (`.md` file = system prompt, cached
by API) and dynamic (per-dispatch context from `templates/agent-context.md`). In a 4-spec wave,
the planner's system prompt is cached after the first dispatch — remaining specs pay ~75% less.

**Tiered dispatch:** Fast agents (spec reviewer, documenter) run foreground for immediate
results. Slow agents (planner, decomposer, reviewers) run background so you stay free to
send corrections or stop signals.

## Ecosystem & Integrations

mint integrates with best-in-class external tools. Each is optional and toggleable — mint works without any of them, but they make it significantly more capable.

| Tool | What it does for mint | Install |
|------|----------------------|---------|
| [PinchTab](https://github.com/pinchtab/pinchtab) | Browser automation — navigate, scrape, debug, screenshot via lightweight Go binary + Chrome. Agents talk to it via HTTP API, get compact accessibility tree (~800 tokens vs 10k+ raw DOM). | `curl -fsSL https://pinchtab.com/install.sh \| sh` |
| [context-mode](https://github.com/mksglu/context-mode) | Sandboxed execution + FTS5 search + session continuity. Keeps verbose tool output out of context window. ~97% token savings on test output, ~99% on URL fetching. | `claude mcp add context-mode -- npx -y context-mode` |
| [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp) | Code knowledge graph — 66 languages, cross-file call traces, blast radius analysis, architecture overview, Cypher queries. Agents see through the graph for smarter specs, reviews, and impact analysis. | `curl -fsSL .../install.sh \| bash` |
| [Impeccable](https://impeccable.style) | Design steering commands (`/polish`, `/audit`, `/critique`, `/bolder`, etc.) with curated anti-patterns and design vocabulary. By Paul Bakaus, Apache 2.0. | `npx skills add pbakaus/impeccable` |

`mint init` offers to install each one. `mint update` keeps them current. `mint doctor` checks their health.

### Code Knowledge Graph

When enabled (`graph.enabled: true`), agents see through a code knowledge graph that indexes
your entire codebase — 66 languages, cross-file call chains, module boundaries, coupling hotspots.

**What it changes:**
- **Decomposer** traces blast radius before splitting work — specs get accurate `<can-modify>` scopes and split at natural module seams instead of arbitrary file boundaries
- **Planner** knows who calls the function you're changing — won't break a public interface with 17 callers without updating them all
- **Security auditor** traces data flow from user input through function calls to output — finds injection paths that static review misses
- **Adversarial tester** targets high-fanout functions first — the ones where a bug affects the most code
- **Cross-boundary detection** — a backend route change traces through to the API spec, the generated client, and the frontend component that calls it

```bash
mint stats    # shows graph health: nodes, edges, hotspot functions with caller counts

# Example output:
#   Code Graph (codebase-memory-mcp)
#     Nodes:     2320
#     Edges:     2409
#     Breakdown: 151 functions, 25 classes, 183 files
#     Hotspots:
#        17 callers  readJsonl (cli/lib/jsonl.js)
#         8 callers  appendJsonl (cli/lib/jsonl.js)
#         7 callers  fileExists (cli/lib/detect.js)
```

Graceful degradation — if the graph isn't installed, agents fall back to grep/read. Nothing breaks, you just get less precision.

## Core Features

### Parallel Execution

Specs don't run one-by-one. mint builds a dependency graph from `<depends-on>` fields, groups independent specs into waves, and dispatches them in parallel.

**Two modes:**
- **In-session:** Parallel Agent calls within one Claude Code session (no isolation needed for non-overlapping scopes)
- **Multi-session:** Separate `claude -p` processes, each in its own git worktree. Fully isolated. Configurable concurrency (default: 3).

```json
{
  "isolation": { "plan": "worktree" },
  "parallel": { "concurrency": 3, "maxBudgetPerSpec": 5.0 }
}
```

After a wave completes, worktree branches merge back. Scope enforcement via the pre-edit hook prevents parallel specs from touching the same files. Cleanup: `mint clean`.

### File Freezing

Protect files and directories from modification. The pre-edit hook **hard-blocks** writes — agents can't bypass it.

```
/freeze src/auth/                          # Block all edits under src/auth/
/guard package.json "no new deps"          # Block + tell agents why
/freeze src/**/*.test.ts                   # Glob patterns work
/unfreeze src/auth/                        # Remove specific freeze
/unfreeze --all                            # Remove all
```

Agents see the block reason and must adjust their approach — find an alternative, skip the file, or ask you. Session-scoped (clears when task completes).

### Scope Enforcement

Every spec declares `<can-modify>` and `<cannot-modify>`. The pre-edit hook reads the active spec and **blocks** writes outside scope — not advisory, not cooperative. Hard block.

If an agent legitimately needs a file outside scope, it stops and reports the blocker. The orchestrator can expand scope with your approval.

### Browser Support

Built-in browser automation powered by [PinchTab](https://github.com/pinchtab/pinchtab). Not a plugin — a core feature. Agents can navigate pages, fill forms, scrape data, take screenshots, and debug live apps.

**How it works:** PinchTab runs a lightweight Go binary that controls Chrome via an HTTP API. mint agents talk to it with curl — no Puppeteer, no Selenium, no heavy dependencies. The accessibility tree gives agents a compact page representation (~800 tokens vs 10k+ for raw DOM).

```bash
# Install PinchTab (mint init offers to do this)
curl -fsSL https://pinchtab.com/install.sh | sh

# WSL2: also install Linux-side Chromium
sudo apt install -y chromium-browser

# Start it
pinchtab &
```

**What agents can do:**
- Navigate to URLs and interact with elements (click, type, select)
- Extract structured data from pages
- Debug live apps — check console errors, DOM state, localStorage
- Verify UI changes after implementation
- Capture screenshots for review
- **Persistent sessions** — login once, cookies survive between tasks

**Commands:**
- `/browse <url> [task]` — navigate and interact
- `/screenshot [url]` — capture page state
- `/scrape <url> [what]` — extract structured data
- `/browser login <url>` — log in manually, mint saves the session
- `/browser sessions` — list saved sessions

**Smart, not blind:** Agents poll for page readiness instead of `sleep 3`. Error recovery diagnoses failures (stale refs, timeout, PinchTab down) and retries intelligently. Auto-starts PinchTab if not running.

**Token-efficient:** Agents use the cheapest PinchTab endpoint per task — `/text` for content (~800 tokens), filtered snapshots for interactions (~3600), diffs for changes. Full snapshots only when needed.

Enable/disable in config:

```json
{
  "browser": {
    "enabled": true,
    "devServer": "http://localhost:3000"
  }
}
```

### Context Mode

Optional integration with [context-mode](https://github.com/mksglu/context-mode) for sandboxed execution, session continuity, and FTS5 full-text search. Not a plugin -- a core feature. Keeps raw tool output out of the context window so agents stay focused.

**What it does:**
- **Sandboxed execution** -- test runners, build tools, and lint output stay out of context via `ctx_execute`. Only errors/failures return.
- **FTS5 search** -- index files, URLs, and command output into a full-text search database. Query with `ctx_search` instead of loading raw content.
- **Session continuity** -- file operations, task state, and decisions survive context compaction automatically.
- **Intent-driven filtering** -- add an `intent` parameter to large outputs and only relevant sections return.

**Setup:**
```bash
# Install via MCP server (mint init offers to do this)
claude mcp add context-mode -- npx -y context-mode
```

**How agents use it:** When `context.enabled` is `true`, every agent activates its Context Mode section automatically. Test runs use `ctx_execute` with `intent: "errors"`, file analysis uses `ctx_execute_file`, URL fetching uses `ctx_fetch_and_index` + `ctx_search`. Standard tools are the fallback if context-mode is unavailable.

**Token savings:** ~97% on test output, ~99% on URL fetching, ~90% on file analysis.

Enable/disable in config:
```json
{
  "context": {
    "enabled": true,
    "autoRoute": true,
    "sandbox": { "timeout": 30000 },
    "session": { "enabled": true }
  }
}
```

### Design Intelligence

Automatic UI/UX awareness powered by vendored [Impeccable](https://impeccable.style) reference knowledge (Apache 2.0) merged with project-specific design learning. Not a plugin — a core feature. When enabled, every UI task automatically gets design context injected into planning and design quality checked during review.

**What it does:**
- **Pre-plan hook** — loads your project's design profile, design notes, and relevant reference knowledge (typography, color theory, spatial design, motion, interaction patterns, responsive design, UX writing). Injects structured design context into the planner.
- **Pre-review hook** — stage 2 auditor that checks for AI slop (purple gradients, glassmorphism, generic card grids), RTL violations, i18n compliance, WCAG 2.1 AA accessibility, design system consistency, and performance.
- **Profile learning** — analyzes existing UI code to extract colors, typography, spacing, and component patterns into `.mint/design-profile.json`. Learns your project's visual DNA.
- **Design notes** — persistent rules ("never use red for success") and preferences that override all other design guidance.
- **AI slop test** — "If you showed this interface to someone and said 'AI made this,' would they believe you immediately? If yes, that's the problem."
- **File-pattern detection** — design context activates when task description mentions UI keywords OR when files matching `design.uiFilePatterns` (`.tsx`, `.jsx`, `.vue`, `.svelte`, `.css`, `.scss`, `.html`) are in scope. No more silent misses on implicit UI work.

**Commands:**
- `/design search|system|palette|typography|inspiration` — design intelligence queries
- `/design:profile build|view|update` — manage project design profile
- `/design:notes add|list|remove` — manage design rules and preferences
- `/design:review [target] [--fix]` — standalone design review
- `/design:tokens export|sync|validate` — design token management
- `/design:teach` — one-time project design context setup
- `/design:steer <direction>` — 16 steering commands (polish, critique, audit, bolder, quieter, distill, colorize, animate, delight, clarify, harden, adapt, normalize, extract, optimize, onboard)

**Optional:** Install [Impeccable](https://impeccable.style) (`npx skills add pbakaus/impeccable`) for editor-level steering commands. mint's design features work without it — reference knowledge is vendored.

Enable/disable in config:
```json
{
  "design": {
    "enabled": true,
    "review": {
      "accessibility": true,
      "consistency": true,
      "performance": true,
      "rtl": false,
      "i18n": false
    }
  }
}
```

### Documentation Intelligence

Automatic documentation tracking powered by the **doc-manifest** system. When enabled, mint tracks which documentation sections depend on which code files — so docs never silently go stale.

**What it does:**
- **Doc-manifest** (`.mint/doc-manifest.json`) — maps each documentation section to the code artifacts it describes via glob patterns
- **Staleness detection** — three strategies: `glob-count` (file count changed), `content-hash` (file contents changed), `git-diff` (files modified since last doc update)
- **Completion protocol** — after every spec, the orchestrator checks the manifest and dispatches the documenter for any stale sections
- **Architectural change detection** — changes to config, agents, CLI, or templates automatically trigger documentation updates
- **Verifier integration** — `mint verify` reports doc staleness as warnings in the gate report

**Setup:**
```bash
# During mint init (automatic)
mint init

# For existing projects
# Use /doc-setup command in Claude Code to analyze and map your docs
```

**How it works:** Each doc section declares which files it tracks. When those files change, the documenter knows exactly what to update and where. No more "I forgot to update the README."

Enable/disable in config:
```json
{
  "definitionOfDone": {
    "docCheckPassed": true
  }
}
```

### Adaptive Automation

mint watches how you work and turns repeated workflows into reusable skills -- automatically.

**How it works:**
1. **Trace** -- session activity is recorded in `.mint/workflow-traces.jsonl`
2. **Mine** -- `mint-workflow-miner` detects repeated command patterns across sessions
3. **Generate** -- confirmed patterns become self-contained skills in `.mint/skills/`
4. **Graduate** -- skills start at `suggest` (ask first), promote to `confirm` (auto-run, ask to commit), then `auto` (fully autonomous) after proven reliability

**Example:** You keep running test-fix-commit cycles. mint detects the pattern, generates a skill, and offers to run it next time. After 3 successful runs, it stops asking.

```
[mint] automate · 3 workflow candidates detected
  1. test-fix-commit (seen 5x)
  2. lint-format (seen 3x)
  3. doc-update (seen 4x)
```

Skills in `.mint/skills/` are personal (gitignored). Promote to `.claude/skills/` to share with the team.

Say `"automate"` or `"detect workflows"` to trigger. Dream mode also mines workflows during consolidation.

## TDD Support

Test-first development built into the pipeline. Toggle via config or per-spec:

1. **RED** — write tests first, verify they fail
2. **GREEN** — implement minimal code to pass
3. **REFACTOR** — clean up while tests stay green
4. **COVERAGE** — verify coverage meets threshold
5. **DE-SLOPPIFY** — optional cleanup pass in fresh context

Edge cases (null, empty, boundary, error paths) are auto-injected. Coverage gating blocks commits below threshold.

```json
{
  "tdd": { "default": true, "coverageThreshold": 80 }
}
```

## Autocommit Control

Autocommit is situational — not just a global toggle. Three levels of control:

1. **Session override** — say "no commits" or use `--no-commit` once, and it persists for the entire plan. Never re-asks.
2. **Per-spec** — `<autoCommit>true|false|inherit</autoCommit>` in the spec XML template.
3. **Global config** — `config.autoCommit` (default: `true`).

Session override wins over per-spec, which wins over global. When autocommit is off, agents run gates but leave changes staged for manual review.

## Auto-Invocation Enforcement

A `PreToolUse` hook on `Edit|Write` checks whether mint was invoked before file modifications. If not, a visible warning fires — so you never accidentally bypass mint's quality pipeline. Session state is tracked per-session in `.mint/sessions/<session-id>.json` (gitignored), so concurrent sessions never stomp on each other.

`mint init` and `mint update` automatically inject a version-tagged mint section into your project's `CLAUDE.md`. `mint doctor` warns if it's missing.

## Review Pipeline

Every spec goes through multi-stage review, **scaled by diff size:**

| Diff size | Review level |
|-----------|-------------|
| < 30 lines | Spec review only (light) |
| 30-100 lines | Spec + quality + conventions (standard) |
| 100-300 lines | Spec + all enabled reviewers (full) |
| 300+ lines | Full + model escalation (opus for security/quality) |

**Stage 1** (sequential gate): Spec reviewer — does the implementation match the spec?

**Stage 2** (parallel audit): Enabled reviewers run simultaneously:
- **Quality** — patterns, types, readability, over-engineering
- **Security** — injection, XSS, auth, secrets
- **Conventions** — naming, file structure, imports
- **Tests** — mock audit, assertion quality, edge cases
- **Business** — requirements alignment, domain logic
- **Performance** — re-renders, N+1, bundle impact (opt-in)
- **Adversarial** — red team probing: writes edge-case tests designed to break the
  implementation, runs in isolated worktree. A passing test = the attack succeeded (opt-in)
- **Design** — AI slop, RTL, i18n, accessibility (if `design.enabled`)

Issues are categorized: BLOCKING (must fix), WARNING (should fix), INFO (logged). Each reviewer can use a different Claude model. The adversarial tester is special — it writes and executes code in a worktree, unlike read-only reviewers. Disable scaling with `config.reviewScaling: false`.

## Learning

mint learns your project's conventions automatically. All learning logs use **JSONL** (one JSON object per line) — append-only, concurrent-safe, `grep`-able. No read-parse-modify-write cycle.

- **Instincts** (`.mint/instincts.jsonl`) — a PostToolUse hook observes every edit and extracts patterns. Each instinct has **confidence scoring** (increments on repeat observations), **deduplication** (same pattern never logged twice), and **decay** (unused instincts lose confidence over 30 days). Top 20 by confidence are injected into the decomposer. Sources track where the instinct came from (observer hook, reviewer feedback).

- **Execution metrics** (`.mint/metrics.jsonl`) — per-spec performance data: which instincts were applied, review outcomes, gate results. Enables evidence-based evolution — instincts that correlate with review passes get boosted.

- **Issues** (`.mint/issues.jsonl`) — every failure with root cause category. Become `<pitfalls>` in new specs.

- **Wins** (`.mint/wins.jsonl`) — successful patterns and why they worked.

- **Patterns** (`.mint/patterns.jsonl`) — graduated from issues/wins when a pattern repeats 3+ times.

### Dream Consolidation

Learning data grows stale. `mint dream` consolidates it — issue triage, instinct decay,
pattern promotion candidates, win archival, health report. Runs automatically when learning
data is stale (7+ days since last dream, 10+ new entries) — suggested on `resume-work` or
first plan invocation. Also available via CLI:

```bash
mint dream              # status overview
mint dream decay        # run instinct decay
mint dream instincts    # list all with scores
```

Full consolidation (issue triage, promotion, report) runs via Claude: just say `"dream"`.

**Complementary to Claude Code's autoDream** — Claude's dream handles generic memory. Mint's
dream handles project-specific learning data. No overlap, they enhance each other.

The system is self-evolving: observe → score → correlate → boost/decay → dream → promote. All committed to git — shared team knowledge.

### Pipeline Analytics

`mint stats` shows how the pipeline is performing:

```bash
mint stats
# Gate pass rate, first-try success (with ↑/↓ trend), avg attempts,
# review pass rate, top issues by root cause, reviewer value (which
# reviewers catch the most BLOCKINGs), instinct health with promotion
# candidates, win patterns, and git activity breakdown.
```

## Plugins

Plugins extend mint with stack-specific or integration capabilities.

| Plugin | What it does |
|--------|-------------|
| `mint-nuxt` | Nuxt file structure, auto-imports, server patterns |
| `mint-e2e` | E2E testing with Playwright |
| `mint-linear` | Linear ticket context and status sync |
| `mint-figma` | Design tokens and specs from Figma |
| `mint-ssh` | SSH connections and remote commands |
| `mint-gws` | Google Workspace — Sheets, Gmail, Calendar |

> **Note:** Plugins are community-extensible and may not cover every edge case. PRs and issues welcome — if something doesn't work right, fix it and contribute back.

Enable plugins:

```json
{
  "plugins": ["plugins/mint-nuxt", "plugins/mint-linear"]
}
```

## Configuration

Two-layer config: global (`~/.mint/config.json`) for user defaults, project (`.mint/config.json`) for project settings. Project always overrides global.

Key config in `.mint/config.json`:

| Key | Default | Description |
|-----|---------|-------------|
| `stack` | auto-detected | Framework (nuxt, react, vue, etc.) |
| `packageManager` | auto-detected | npm, pnpm, yarn, bun |
| `gates` | `{}` | lint/types/tests/coverage commands |
| `gates.tiered` | `true` | Gate tier classification — skip/quick/full based on changed files |
| `gates.tiers` | defaults | Custom glob patterns for skip/quick/full classification |
| `autoCommit` | `true` | Commit after passing gates (overridable per-session and per-spec) |
| `tdd.default` | `false` | TDD-first by default |
| `browser.enabled` | `true` | Browser automation via PinchTab |
| `context.enabled` | `false` | Context Mode via context-mode |
| `graph.enabled` | `false` | Code knowledge graph — blast radius, call traces, architecture |
| `graph.autoIndex` | `true` | Auto-reindex on plan mode start |
| `definitionOfDone.docCheckPassed` | `true` | Check doc-manifest after each spec |
| `design.enabled` | `true` | Design intelligence — profiling, anti-patterns, RTL/i18n |
| `design.uiFilePatterns` | `["*.tsx","*.jsx",...]` | File patterns that auto-trigger design context |
| `reviewers` | smart defaults | Which reviewers run and their models |
| `reviewScaling` | `true` | Scale review intensity by diff size |
| `isolation` | `none` | Work isolation: none, branch, or worktree |
| `parallel.concurrency` | `3` | Max parallel Claude Code sessions |
| `parallel.maxBudgetPerSpec` | `5.0` | Max USD per spec in parallel mode |
| `repoMode` | `collaborative` | `solo` (fix incidental issues) or `collaborative` (flag only) |
| `challenge` | `auto` | Challenge assumptions before planning (auto = large tasks only) |
| `plugins` | `[]` | Plugin paths |

## Golden Rules

1. **Never fix bad output.** Reset and fix the spec — not the code.
2. **One agent, one task, one prompt.** Focused agents are correct agents.
3. **Gates before everything.** Lint + types + tests pass 100% before any commit.
4. **Never mock what you can use for real.** Mocks hide failures.
5. **Precise specs, zero inference.** Agents don't guess.
6. **Escalate, don't improvise.** If stuck, stop and ask — never silently work around.

## Contributing

Plugins aren't perfect — they're starting points. If you hit an edge case or something doesn't work:

1. Open an issue describing what went wrong
2. PRs welcome — especially for plugins and stack-specific conventions
3. Follow the existing patterns: one agent per file, role → inputs → process → outputs → rules

See [Plugin Guide](docs/plugin-guide.md) for creating custom plugins.

## Documentation

| Doc | What it covers |
|-----|---------------|
| [Plugin Guide](docs/plugin-guide.md) | Creating custom plugins |
| [Conventions](docs/conventions.md) | File formats, naming, config schema |
| [Architecture](docs/architecture.md) | System design and philosophy |
| [Autonomous Loops](docs/autonomous-loops.md) | CI/CD and scripted workflows |
| [Doc Setup](commands/doc-setup.md) | Building doc-manifest for existing projects |
| [Optimize](commands/optimize.md) | Full setup audit — config, docs, workspace, agents, features |

## License

MIT

---

<!-- mint:signature -->
<p align="center">
  <sub>Minted with <a href="https://github.com/3li7alaki/mint">mint</a> — disciplined agentic development.</sub>
</p>
