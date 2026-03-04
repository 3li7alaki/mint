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

## Issue Log

`.mint/issues.md` is a markdown table with columns: Date, Task, Severity, Issue, Root Cause, Resolution, Spec Fix.

Root cause categories: `bad-spec`, `missing-context`, `scope-leak`, `environment`, `hard-block`, `unknown-pattern`.

The planner reads this before creating new specs. If a past issue is relevant (same files, similar patterns), it becomes a `<pitfalls>` entry in the new spec. This is the learning loop.

## Plugin System

Plugins are directories with a `manifest.json` + agents + commands. They hook into the pipeline at defined points (pre-plan, post-plan, pre-review, post-commit, on-init).

Plugin agents are namespaced (`plugin-name:agent-name`) and dispatched the same way as core agents. They follow the same isolation rules.

Four plugin types map to four extension dimensions:
- **Stack** — framework-specific conventions and review
- **PM** — project management tool sync
- **Design** — design tool integration
- **Memory** — knowledge persistence and retrieval

## Golden Rules

1. Never fix bad output — fix the spec
2. One agent, one task, one prompt
3. Gates before everything
4. Never mock what you can use for real
5. Precise specs, zero inference
6. Escalate, don't improvise
