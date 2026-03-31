# mint Architecture

## Philosophy

**Slop is an engineering problem, not an LLM problem.** If an agent produces bad code, the environment failed — not the model. Fix the spec, the constraints, the review pipeline. Never patch the output.

### Core Beliefs

1. **Agents are disposable, specs are permanent.** A subagent runs once with fresh context. The spec that drives it is the lasting artifact. If the output is wrong, the spec was wrong.

2. **Separation of concerns is absolute.** The orchestrator delegates. Agents execute. Reviewers audit. The documenter documents. No agent crosses its boundary. No context leaks between roles.

3. **Quality is a gate, not a goal.** Lint, types, and tests don't make code good — they prevent code from being bad. Quality comes from precise specs, focused agents, and honest review.

4. **Mistakes are data.** Every failure is logged with a root cause. The issue log feeds the planner. Past mistakes become future prevention. The system learns.

5. **Humans review, agents commit.** Agents never push. The human is the final gate. Always.

## System Design

```
User
  │
  ▼
Router (SKILL.md — ~125 lines)
  │
  ├─ classify task → load mode file
  │   ├─ modes/quick.md → main context (no subagent)
  │   ├─ modes/plan.md → state machine pipeline
  │   ├─ modes/ship.md → interview + batch execute
  │   ├─ modes/research.md → investigate + report
  │   ├─ modes/verify.md → audit + report
  │   ├─ modes/ssh.md → remote commands
  │   ├─ modes/browse.md → browser automation
  │   └─ modes/design.md → design intelligence
  │
  ├─ (plan mode) load phase file per pipeline step
  │   ├─ phases/decompose.md → phases/implement.md → phases/desloppify.md
  │   └─ phases/review-stage1.md → phases/review-stage2.md → phases/docs.md → phases/dod.md
  │
  ▼
Subagent Pool (fresh context per dispatch)
  │
  ├─ Decomposer ─────────── breaks features into XML specs (decompose only)
  ├─ Planner ────────────── implements a single spec, runs gates, commits
  ├─ Pipeline Checker ───── verifies pipeline steps were executed
  ├─ Researcher ─────────── investigates, writes reports
  ├─ Shipper ────────────── batch executes ship plans
  ├─ Verifier ───────────── runs all gates and audits
  ├─ Spec Reviewer ──────── stage 1 gate (spec compliance)
  ├─ Quality Reviewer ───── stage 2 (code quality)
  ├─ Security Auditor ───── stage 2 (vulnerabilities)
  ├─ Conventions Enforcer ─ stage 2 (project conventions)
  ├─ Test Auditor ────────── stage 2 (test quality)
  ├─ Business Reviewer ──── stage 2 (requirements alignment)
  ├─ Performance Reviewer ─ stage 2 (performance, opt-in)
  ├─ Documenter ─────────── updates project documentation (manifest-guided)
  ├─ De-sloppifier ──────── post-implementation cleanup
  ├─ Build Error Resolver ─ minimal build/type error fixes
  ├─ Refactor Cleaner ───── dead code detection and removal
  └─ Plugin Agents ──────── stack/PM/design/memory extensions
```

### Skill Decomposition

The orchestrator is split into focused, loadable pieces — not one monolithic file:

```
skills/mint/
  SKILL.md            ← Router only (~125 lines). Routes, manages state, dispatches.
  modes/              ← One file per execution mode. Loaded on route.
  phases/             ← One file per pipeline step. Loaded per-step.
  reference/          ← Detailed docs. Loaded on demand only.
```

**Why:** LLM instruction compliance degrades linearly with prompt length. A 1900-line skill
gets ~40% compliance. The ~125-line router gets ~85%. Phase files load one at a time — the
agent never holds the full pipeline in memory.

### Pipeline State Machine

Plan mode uses a state file (`pipeline-state.json`) to track which step the orchestrator is on.
The router reads state → loads the phase file → dispatches → updates state → loops. No reliance
on LLM memory for sequencing.

## Configuration Layers

mint uses a two-layer config system:

1. **Global config** (`~/.mint/config.json`) — user preferences that apply across all projects (reviewer models, autoCommit, TDD, isolation, modelRouting). Set via `mint config set --global`.
2. **Project config** (`.mint/config.json`) — project-specific settings (stack, gates, browser, plugins). Created by `mint init`.

Resolution: project config > global config > hardcoded defaults. The orchestrator merges these before dispatching agents — agents always receive a single resolved config.

When `mint init` runs, global config seeds the defaults. `mint update` offers to create global config from existing project preferences.

## Workspace

When `workspace.repos` is configured, the orchestrator loads repo metadata (name, path, stack, role, dependsOn) and feeds scoped context to agents. This gives agents awareness of cross-repo dependencies without loading entire codebases into context.

Workspace is opt-in. Without it, mint operates on the current repo only. See `SKILL.md` for the full workspace config schema and dispatch behavior.

## Agent Isolation

Each subagent:
- Gets **fresh context** — no memory from previous agents
- Receives **exactly what it needs** — spec XML, config, workspace context, or diff (not everything)
- Returns **a concise summary** — never raw tool output
- Writes **artifacts to disk** — `.mint/tasks/`, `.mint/research/`, commits
- **Cannot spawn other subagents** — only the orchestrator dispatches

This prevents context pollution. An agent that builds up too much context makes worse decisions. Fresh agents make better decisions.

When workspace is configured, agents receive scoped workspace context relevant to their task — not the full workspace. The orchestrator decides what each agent needs to see.

## Wave-Based Parallel Execution

Specs don't execute sequentially. The orchestrator builds a dependency graph from `<depends-on>`
fields and groups independent specs into waves:

```
Specs: 001, 002, 003, 004, 005
Dependencies: 003→001, 004→001, 005→003+004

Wave 1: [001, 002]       ← independent, run in parallel
Wave 2: [003, 004]       ← both depend on 001 (done), parallel
Wave 3: [005]            ← depends on 003+004 (done)
```

**Two parallel modes:**
- **In-session** (`isolation: "none"`) — parallel Agent calls within one Claude Code session.
  Scope enforcement prevents file conflicts between parallel specs.
- **Multi-session** (`isolation: "worktree"`) — `cli/lib/parallel.js` spawns separate `claude -p`
  processes, each in its own git worktree at `.mint/worktrees/<slug>`. Fully isolated.
  `cli/lib/worktree.js` handles creation, env propagation, merge, and cleanup.

**Safety:** Before dispatching parallel specs, the orchestrator verifies `<can-modify>` paths
don't overlap. Overlapping scopes → forced sequential execution.

**Concurrency:** Configurable via `config.parallel.concurrency` (default: 3) and
`config.parallel.maxBudgetPerSpec` (default: $5 USD).

## File Freezing

Runtime file protection via `/freeze`, `/guard`, `/unfreeze` commands. The pre-edit hook
reads `.mint/.freeze-list.json` and **hard-blocks** writes to matching paths.

- `/freeze src/auth/` — blocks all writes under that directory
- `/guard package.json "no new deps"` — blocks + shows reason to agents
- Supports exact files, directories, and glob patterns (`src/**/*.test.ts`)
- Session-scoped — clears on task completion
- Agents see the block reason and must adjust their approach

## Scope Enforcement

Every spec declares `<can-modify>` paths. The pre-edit hook reads the active spec from
session state and blocks writes outside scope. This is preventive (blocks before the write)
not reactive (catches after the fact).

## Risk Self-Regulation (WTF-Likelihood)

The orchestrator tracks a cumulative risk score during execution. Gate failures (+10%),
spec rewrites (+15%), out-of-scope file modifications (+20%) all escalate the score.
At 25% the orchestrator warns; at 50% it stops. Hard cap: 30 fix attempts across all specs.

This catches the pattern where nothing is working well but individual failures don't
trigger the 2-attempt limit.

## Review Pipeline

Two stages, by design. **Review intensity scales by diff size** — small diffs (<30 lines) get
spec review only. Large diffs (300+ lines) get full review with model escalation.

**Stage 1 (sequential gate):** Spec reviewer. Must pass before anything else. Checks: does the implementation match what was asked? No extra code, no missing requirements, scope respected.

**Stage 2 (parallel audit):** Up to 6 reviewers run simultaneously. Each checks one dimension. Each is independently enabled/disabled in config. Each returns a severity-tagged report.

Why two stages? Because there's no point auditing code quality on an implementation that doesn't match its spec. Fix spec compliance first, then check everything else.

Why parallel? Because reviewers don't depend on each other. Quality review doesn't need security results. Conventions don't need test results. Running them in parallel saves time without losing accuracy.

## Learning Loop

All learning logs use **JSONL** (JSON Lines) — one JSON object per line. Append-only,
concurrent-safe (two agents appending simultaneously just add two lines), `grep`-able.
No read-parse-modify-write cycle.

**Instincts** (`.mint/instincts.jsonl`) — auto-extracted by hooks observing every Edit/Write.
Tracks import styles, naming conventions, test patterns, framework usage. Each instinct has
confidence scoring, deduplication (via `upsertInstinct()`), and decay. Confidence increases
with repetition, decreases when not reinforced for 30 days. High-confidence (≥ 3) are treated
as project conventions. Top 20 by confidence are injected into the decomposer. Sources track
where the instinct came from (observer hook, reviewer feedback, etc.).

**Execution metrics** (`.mint/metrics.jsonl`) — per-spec performance data: which instincts were
applied, review outcomes, gate results, attempt counts. Enables evidence-based evolution —
`analyzeInstinctEffectiveness()` correlates instinct usage with review pass rates.

**Issue log** (`.mint/issues.jsonl`) — failures with root cause categories. Become `<pitfalls>`.

**Wins log** (`.mint/wins.jsonl`) — successes and what patterns worked. Inform decomposition.

**Patterns** (`.mint/patterns.jsonl`) — graduated from issues/wins when a pattern repeats 3+ times.

The decomposer reads all four before creating specs. Past mistakes become prevention.
Past wins become guidance. JSONL utilities: `cli/lib/jsonl.js`.

## Documentation Intelligence

The doc-manifest (`.mint/doc-manifest.json`) tracks which documentation sections depend on which code artifacts. This closes the feedback loop between code changes and documentation:

1. **Manifest** — each doc section declares `tracks` (glob patterns) and a `staleness` strategy
2. **Completion protocol** — after every spec, the orchestrator checks if tracked files changed
3. **Documenter dispatch** — stale sections trigger the documenter with precise context
4. **Architectural detection** — changes to config, agents, CLI, or templates trigger broader doc updates
5. **Verifier integration** — `mint verify` reports stale docs as warnings

Three staleness strategies:
- `glob-count` — file count changed (best for directory listings, agent inventories)
- `content-hash` — file contents changed (best for config schemas, API references)
- `git-diff` — tracked files modified since last doc update (best for narrative descriptions)

The manifest is committed to git — it's shared team knowledge about documentation dependencies.

## Execution Tracking

Every spec gets a per-spec `execution.json` that records status transitions, attempt history, gate results, review verdicts, and commit hashes. This enables:
- **Resumability** — interrupted sessions can pick up where they left off
- **Visibility** — clear record of what happened at each pipeline stage
- **Retry intelligence** — the spec retry protocol uses attempt history to write targeted rewrites

## Spec Retry Protocol

When a spec fails, the orchestrator diagnoses the root cause category, cross-references the issue log for similar past failures, and rewrites the spec with targeted adjustments. One rewrite attempt, then escalate. This is how "never fix bad output — fix the spec" works in practice.

## Plugin System

Plugins are directories with a `manifest.json` + agents + commands. They hook into the pipeline at defined points (pre-plan, post-plan, pre-review, post-commit, on-init).

Plugin agents are namespaced (`plugin-name:agent-name`) and dispatched the same way as core agents. They follow the same isolation rules.

Four plugin types map to four extension dimensions:
- **Stack** — framework-specific conventions and review
- **PM** — project management tool sync
- **Design** — design tool integration
- **Memory** — knowledge persistence and retrieval

## Design Intelligence

Core feature that makes UI/UX awareness automatic. When `config.design.enabled` is `true`:

1. **Pre-plan hook** — `design-context` agent loads the project's design profile (`.mint/design-profile.json`), design notes (`.mint/design-notes.md`), and relevant reference knowledge from `standards/design/reference/` (typography, color, spatial, motion, interaction, responsive, ux-writing). Injects structured design context into the planner spec.

2. **Pre-review hook** — `design-reviewer` agent runs as a stage 2 parallel auditor. Checks for AI slop (always), RTL violations, i18n compliance, accessibility (WCAG 2.1 AA), design consistency, performance, and brand compliance.

3. **Profile learning** — `design-profile` agent analyzes existing UI code to extract colors, typography, spacing, component patterns into a project-specific design profile. Builds incrementally.

4. **Design notes** — persistent rules and preferences (`.mint/design-notes.md`). Hard rules become BLOCKING constraints; preferences become WARNINGs.

Reference knowledge is vendored from [Impeccable](https://impeccable.style) (Apache 2.0). Impeccable itself is an optional install that adds steering commands (`/polish`, `/audit`, `/critique`, etc.) to the editor.

Design intelligence requires no user invocation — it activates automatically on UI tasks and catches design problems before they ship.

## Context Mode

Optional infrastructure layer powered by [context-mode](https://github.com/mksglu/context-mode). When `config.context.enabled` is `true`, agents prefer sandboxed execution tools (`ctx_execute`, `ctx_execute_file`, `ctx_batch_execute`) over raw Bash/Read for data-heavy operations. This keeps verbose tool output out of the context window structurally rather than relying on agent discipline.

Three capabilities:
1. **Sandboxed execution** -- commands run in isolated subprocesses. Only filtered output enters context. Supports 11 languages.
2. **FTS5 knowledge base** -- files, URLs, and command output are chunked and indexed into SQLite FTS5 tables. Agents query with `ctx_search` instead of loading raw content.
3. **Session continuity** -- hooks (PreCompact, SessionStart, PostToolUse) track file operations, task state, errors, and decisions in a per-project SQLite database. After context compaction, agents search `source: "session-events"` to recover working state.

context-mode is an external dependency (ELv2 license). mint wraps it, never forks or embeds. Graceful degradation -- all agents fall back to standard tools if context-mode is unavailable.

## Hooks System

Real-time Claude Code hooks provide instant feedback and enforcement:

- **PreToolUse (Edit|Write)** — freeze/guard enforcement, spec scope enforcement, mint invocation check
- **PreToolUse (Bash)** — git push blocker, bash interpolation in commit messages blocker
- **PostToolUse (Edit)** — auto-format, typecheck, console.log warning
- **PostToolUse (Edit|Write)** — quality gate (lint check), instinct observation
- **Stop** — pipeline completion check (blocks stop if reviews not run), cost tracking

Pre-edit hooks return `decision: "block"` to prevent operations — not warnings. git push,
frozen file writes, and out-of-scope writes are hard-blocked. Agents see the block reason
and must adjust.

Hooks are lightweight Node.js scripts in `hooks/scripts/`. They fire deterministically on every
tool use — no probability, no skipping.

## TDD Support

When `<tdd>true</tdd>` is set in a spec:

1. **RED** — planner writes tests first, verifies they fail
2. **GREEN** — implements minimal code to make tests pass
3. **REFACTOR** — cleans up while keeping tests green
4. **COVERAGE** — verifies coverage meets threshold
5. **DE-SLOPPIFY** — optional cleanup pass in fresh context

The edge case checklist (null, empty, boundary, error paths, race conditions, special chars) is
auto-injected into TDD specs.

## Eval-Driven Development

Evals are "unit tests for agent quality" — stored in `.mint/evals/`:

- **Capability evals** — can the agent do something it couldn't before?
- **Regression evals** — did changes break existing functionality?
- **pass@k** — success within k attempts (practical reliability)
- **pass^k** — all k attempts succeed (stability gate)

## Model Routing

The orchestrator auto-selects which Claude model executes each spec based on complexity:

| Estimate | Model | Rationale |
|----------|-------|-----------|
| `trivial` | Haiku | Config tweaks, renames — speed and cost |
| `small`/`medium` | Sonnet | Standard implementation — fast and capable |
| `large` | Opus | Architecture, novel patterns — deep reasoning |

Model routing is configured via `config.modelRouting`. Per-reviewer model config (already supported) is separate — this adds per-spec routing for the planner/executor. The planner assigns estimates during decomposition; the orchestrator maps estimates to models during dispatch.

## Golden Rules

1. Never fix bad output — fix the spec
2. One agent, one task, one prompt
3. Gates before everything
4. Never mock what you can use for real
5. Precise specs, zero inference
6. Escalate, don't improvise
