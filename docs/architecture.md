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
Orchestrator (SKILL.md)
  │
  ├─ classify task
  │   ├─ quick → main context (no subagent)
  │   ├─ plan → decompose + execute
  │   ├─ ship → interview + batch execute
  │   ├─ research → investigate + report
  │   └─ verify → audit + report
  │
  ▼
Subagent Pool (fresh context per dispatch)
  │
  ├─ Planner ────────────── decomposes, implements, commits
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
  ├─ Documenter ─────────── updates project documentation
  ├─ De-sloppifier ──────── post-implementation cleanup
  ├─ Build Error Resolver ─ minimal build/type error fixes
  ├─ Refactor Cleaner ───── dead code detection and removal
  └─ Plugin Agents ──────── stack/PM/design/memory extensions
```

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

## Review Pipeline

Two stages, by design:

**Stage 1 (sequential gate):** Spec reviewer. Must pass before anything else. Checks: does the implementation match what was asked? No extra code, no missing requirements, scope respected.

**Stage 2 (parallel audit):** Up to 6 reviewers run simultaneously. Each checks one dimension. Each is independently enabled/disabled in config. Each returns a severity-tagged report.

Why two stages? Because there's no point auditing code quality on an implementation that doesn't match its spec. Fix spec compliance first, then check everything else.

Why parallel? Because reviewers don't depend on each other. Quality review doesn't need security results. Conventions don't need test results. Running them in parallel saves time without losing accuracy.

## Learning Loop

Two complementary logs feed the planner:

**Instincts** (`.mint/instincts.md`) — auto-extracted by hooks observing every Edit/Write. Tracks import styles, naming conventions, test patterns, and framework usage. Confidence increases when the same pattern appears across multiple files. High-confidence instincts (>= 3) are treated as project conventions by the planner. Controlled by `config.instincts.enabled`.

**Issue log** (`.mint/issues.md`) — tracks failures. Columns: Date, Task, Severity, Issue, Root Cause, Resolution, Spec Fix. Root cause categories: `bad-spec`, `missing-context`, `scope-leak`, `environment`, `hard-block`, `unknown-pattern`. Relevant issues become `<pitfalls>` in new specs.

**Wins log** (`.mint/wins.md`) — tracks successes. Columns: Date, Task, Pattern, Why It Worked. Logged by the orchestrator after full task completion. Wins inform spec decomposition strategy.

The planner reads both before creating new specs. Past mistakes become prevention. Past wins become guidance.

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

## Hooks System

Real-time Claude Code hooks provide instant feedback during development:

- **PostToolUse (Edit)** — auto-format, typecheck, console.log warning
- **PostToolUse (Edit|Write)** — quality gate (lint check)
- **PreToolUse (Bash)** — git push safety reminder
- **Stop** — cost tracking per session

Hooks are lightweight Node.js scripts in `hooks/scripts/`. They fire deterministically on every
tool use — no probability, no skipping. Hook scripts are the one exception to mint's "no runtime"
rule: they're standalone scripts with no dependencies beyond Node.js.

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
