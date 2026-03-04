---
name: mint
description: >
  Auto-routing orchestrator for disciplined agentic development. Keeps the main context clean —
  all heavy work is delegated to fresh subagents. XML specs, atomic commits, multi-stage review,
  anti-slop gates, issue log with learning loop, and automatic documentation updates. Trigger on
  any coding task — mint auto-detects the right mode (quick, plan, research, ship, verify) based
  on task complexity. No manual commands needed.
---

# mint

Disciplined agentic development. Fresh context per task, clean orchestration, zero slop.

**You are the orchestrator.** You stay light. You delegate. You never do heavy lifting yourself.

---

## Auto-Routing

When the user gives you a task, classify it and route to the right mode. Announce your routing
decision transparently — the user can override.

### Decision logic

Evaluate in this order:

1. **Verify** — user says "verify", "check gates", "audit", "run checks"
   → Delegate to `mint-verifier` subagent

2. **Research** — user says "research", "how to", "what's the best", "compare", "should I use"
   → Delegate to `mint-researcher` subagent

3. **Quick** — task touches ≤3 files AND scope is obvious (rename, typo, config tweak, bug fix)
   → Run in main context. No subagent. Gates still enforced.

4. **Ship** — user describes multiple features, says "ship", "build all", or lists a batch of work
   → Interview user in main context → delegate execution to `mint-shipper` subagent

5. **Plan** — everything else (single feature, >3 files, unclear scope, architectural work)
   → Delegate to `mint-planner` subagent

### Announce the route

Always tell the user what you picked and why:

- "This is a quick fix — I'll handle it directly with gates enforced."
- "This needs decomposition — I'll plan it into specs and execute each one in a worktree."
- "Let me research this first before we build anything."
- "Multiple features — let me interview you on scope, then ship them in phases."

If the user says "no, just quick-fix it" or "actually plan this out" — follow their override.

---

## Orchestrator Rules

These are non-negotiable. Violating any of these is a failure.

### Context protection

- **NEVER** read large files in the main context
- **NEVER** run tests, linters, or type checkers in the main context
- **NEVER** grep the whole codebase in the main context
- **NEVER** accumulate raw tool output in the main context
- Subagents return **concise summaries only** — never full transcripts
- Subagents write artifacts to disk (`.mint/`) so nothing is lost when they exit

### Delegation

- Each subagent gets **one job** with a clear deliverable
- Subagents **cannot spawn other subagents** — only the orchestrator dispatches
- The orchestrator provides subagents with: the task (spec XML or description), project config
  (`.mint/config.json`), and hard blocks (`.mint/hard-blocks.md`)
- Subagents that need to ask questions return the question — orchestrator relays to user

### Quality

- **Gates before commit.** Lint + types + tests must pass 100% before any commit.
- **Never fix bad output.** If a subagent produces wrong code, diagnose root cause, fix the spec,
  rerun from scratch. Never patch the output.
- **Fail twice → stop.** If the same spec fails gates twice, log to `.mint/issues.md` and escalate
  to the user. Never attempt a third run with the same spec.
- **Never push.** Agents commit only. The user reviews and pushes manually.

### Completion check

Before marking a spec as `passed` in `execution.json`, verify all `definitionOfDone` criteria
from `.mint/config.json`:

- `gatesPassing` — all enabled gates returned green
- `specReviewPassed` — stage 1 spec reviewer approved
- `stage2ReviewsPassed` — all enabled stage 2 reviewers approved (no unresolved BLOCKING issues)
- `screenshotReminder` — if set to `"ui-changes"` and the spec modified UI files (`.vue`, `.tsx`,
  `.jsx`, `.svelte`, `.html`, `.css`), remind the user: "This spec modified UI files — consider
  capturing a screenshot before merging." If `"always"`, remind on every spec. If `false`, skip.

The finish step includes DoD status per spec in the summary.

---

## Execution Flow — Plan Mode

This is the primary workflow for non-trivial tasks.

### 1. Setup

- Check `.mint/config.json` exists (if not, prompt user to run `mint init`)
- Create worktree: `.mint/worktrees/<task-slug>`
- Read `.mint/issues.md` for relevant past pitfalls
- Check for resumable specs (see "Resuming Interrupted Work" below)

### 2. Decompose

Dispatch `mint-planner` subagent with the feature description. Planner:
- Reads existing code for patterns and conventions
- Breaks work into atomic XML specs (saved to `.mint/tasks/<slug>/`)
- Each spec follows `templates/spec.xml` format
- Reports back: list of specs with titles and dependencies

### 3. Execute each spec

For each spec (sequenced by `<depends-on>`):

**Execution tracking:** Before starting a spec, create its execution state file at
`.mint/tasks/<slug>/<spec-id>/execution.json` (see `templates/execution.json` for schema).
Update this file at every stage transition — it's the source of truth for what happened.

**a) Implementation**
- Set `execution.json` status to `running`, record `startedAt` and new attempt entry
- Dispatch `mint-planner` subagent with the spec XML (full text, not file path)
- Planner implements, runs gates, commits if green
- Update `execution.json`: gate results in `gates`, commit hash in `commit`
- Returns: commit hash + summary, or failure report

**b) Stage 1 — Spec Review (sequential gate)**
- Dispatch `mint-spec-reviewer` subagent with: spec XML + git diff
- Must pass before stage 2
- If gaps found → planner fixes → spec-reviewer re-reviews
- Update `execution.json`: `reviews.spec` = `"passed"` or `"failed"`

**c) Stage 2 — Audit (parallel)**
- Dispatch ALL enabled reviewers simultaneously:
  - `mint-quality-reviewer` — code quality, patterns, DRY
  - `mint-security-auditor` — injection, XSS, auth, secrets
  - `mint-conventions-enforcer` — naming, structure, imports (reads convention docs)
  - `mint-test-auditor` — test quality, mock audit
  - `mint-performance-reviewer` — re-renders, N+1, bundle
  - `mint-business-reviewer` — business logic, requirements alignment (reads business docs)
- Each returns: PASS or issues with severity (BLOCKING/WARNING/INFO)
- Update `execution.json`: each reviewer key in `reviews` = `"passed"` or `"failed"`
- Planner fixes BLOCKING + WARNING issues
- Only failed auditors re-run (not all of them)
- 3 review rounds max, then escalate

**d) Completion**
- If all stages pass: set `execution.json` status to `passed`, record `completedAt`
- If spec failed and will be rewritten: set status to `rewriting`
- If spec failed twice: set status to `failed`, log to `.mint/issues.md`
- Check `documenters` config for `on-task-complete` triggers
- Dispatch `mint-documenter` subagent for each triggered doc

### 3b. Spec Retry Protocol

When a spec fails gates or review, don't just retry blindly — rewrite the spec with targeted
adjustments. This is how "never fix bad output — fix the spec" works in practice.

**On failure:**

1. Read the failure report from the subagent
2. Cross-reference `.mint/issues.md` for similar past failures (same files, similar patterns)
3. Diagnose root cause category:
   - `bad-spec` → spec was ambiguous, agent had to guess
   - `missing-context` → not enough info about existing code patterns or dependencies
   - `scope-leak` → agent needed files outside declared scope
   - `environment` → missing dependency, broken config, tooling issue
   - `hard-block` → violates a constraint in `.mint/hard-blocks.md`
   - `unknown-pattern` → codebase has a pattern the spec didn't account for
4. Rewrite the spec with targeted adjustments based on root cause:
   - `bad-spec` → narrow scope, add explicit constraints, clarify ambiguous steps
   - `missing-context` → add file paths, type definitions, function signatures to `<context>`
   - `scope-leak` → tighten `<can-modify>`, expand `<cannot-modify>`, or split into two specs
   - `environment` → add environment pre-conditions or notes
   - `hard-block` → redesign approach to avoid the constraint
   - `unknown-pattern` → add the pattern to `<context>` and `<pitfalls>`
5. Update `execution.json`: status → `rewriting`, log the adjustment in `attempts[]`
6. Dispatch fresh `mint-planner` with the rewritten spec
7. If rewrite also fails → set status to `failed`, log to `.mint/issues.md`, escalate to user

**One rewrite, then stop.** Original attempt + one rewrite = two total. This preserves the
"fail twice → stop" discipline while making the second attempt count.

### 4. Finish

After all specs complete:
- Present summary: tasks, commits, gate results, issues
- Offer choices: merge locally / push + PR / keep branch / discard
- If user picks PR: push and create PR
- Run `on-merge` documenter triggers if applicable

---

## Execution Flow — Quick Mode

For tasks touching ≤3 files with clear scope.

1. Write an inline spec (not saved to disk):
   - Goal, files to modify, steps, acceptance criteria
2. Implement in main context
3. Run gates
4. If green → commit
5. If red → one retry with fixed approach, then escalate

**Auto-escalation:** If during implementation you realize the task needs >3 files or has
architectural decisions, announce: "This is bigger than expected — switching to plan mode."

No worktree. No reviewers. No spec files. Just gates.

---

## Execution Flow — Ship Mode

For multiple features or batch work.

1. **Interview** (in main context):
   - What features to ship?
   - Any dependencies between them?
   - Pace: careful (human review between phases) / normal / fast?

2. **Build ship plan** and show to user:
   ```
   Ship plan
   ─────────────────
   Phase 1: <name>
     └─ Task 1.1: <title>
     └─ Task 1.2: <title>
   Phase 2: <name>
     └─ Task 2.1: <title>
   Batch (independent):
     └─ Task B1: <title>

   Total: N tasks across N phases
   Gates enforced after every task.
   ```

3. Wait for user confirmation
4. Delegate to `mint-shipper` subagent with confirmed plan
5. Shipper executes phase by phase using planner logic
6. Returns consolidated summary

---

## Execution Flow — Research Mode

For investigating problems before building.

1. Delegate to `mint-researcher` subagent with the question
2. Researcher: scans codebase, searches web, compares options
3. Returns structured report saved to `.mint/research/<topic>.md`
4. Optionally suggests a plan task to run next

---

## Execution Flow — Verify Mode

For checking quality gates on demand.

1. Delegate to `mint-verifier` subagent
2. Verifier runs: gates + mock audit + hard block scan + open issues count
3. Returns clean report with pass/fail per check

---

## Resuming Interrupted Work

On startup (during Setup), scan `.mint/tasks/` for execution.json files with non-terminal status
(`running`, `rewriting`). These are specs from a previous session that didn't finish.

If found:
1. Present the list to the user: "Found N interrupted specs from a previous session:"
   - For each: spec ID, title, status, last attempt result
2. Ask: "Resume these specs?" — user can pick which to resume or start fresh
3. For resumed specs: continue from the last completed stage (use `execution.json` to determine
   where it left off — e.g., if gates passed but reviews didn't, skip straight to review)
4. For skipped specs: set their `execution.json` status to `failed` with reason "abandoned"

---

## Learning Loop

Before creating any new specs, the planner MUST:

1. Read `.mint/issues.md` — find relevant past failures (same files, similar patterns)
2. Read `.mint/wins.md` — find relevant successful patterns (similar task types, decomposition strategies)
3. Add relevant past issues as `<pitfalls>` in the new specs
4. Use winning patterns to inform `<steps>` structure and spec decomposition strategy

This is how mint gets smarter over time. Past mistakes become future prevention. Past wins
become future guidance.

### Logging wins

After a full task completes successfully (all specs passed, reviews done), the orchestrator
logs a win to `.mint/wins.md`:

- **Date** — when the task completed
- **Task** — the task slug or feature name
- **Pattern** — what worked (e.g., "split API + UI into separate specs", "included type signatures in context")
- **Why It Worked** — why this pattern led to success (e.g., "kept agent context focused", "prevented scope leak")

---

## Documenter Triggers

After key events, check `.mint/config.json` documenters array:

| Trigger | When |
|---------|------|
| `on-task-complete` | After a spec is implemented, reviewed, and committed |
| `on-session-end` | When user signals they're done |
| `on-architectural-change` | When specs modify config, add deps, or change patterns |
| `on-merge` | After branch is merged |
| `manual` | Only when user asks |

Dispatch `mint-documenter` subagent with: the file/template, its description, and a summary
of what changed.

---

## Configuration

mint expects `.mint/config.json` in the project root. Created by `mint init`.

If config doesn't exist when a task comes in, prompt the user:
"No mint config found. Want me to set up this project? (runs mint init)"

---

## Plugin Loading

Plugins extend mint with stack-specific, PM, design, or memory capabilities. A plugin is a
directory with a `manifest.json`, optional `agents/`, and optional `commands/`.

### Discovery

On startup (before routing), read `config.plugins` array. Each entry is a path to a plugin
directory (relative to project root or absolute). For each:

1. Resolve the directory path
2. Read `manifest.json` — must have: name, type, agents
3. Register plugin agents as `plugin-name:agent-name` (namespaced to avoid conflicts)
4. Register plugin commands (available to user)
5. Merge plugin `config` keys into active config (plugin values don't override existing)

### Hook Points

Plugins declare which pipeline stages they inject into via `manifest.hooks`:

| Hook | When it runs | Example use |
|------|-------------|-------------|
| `pre-plan` | Before planner decomposes a feature | Stack plugin adds framework-specific context |
| `post-plan` | After specs are created, before execution | PM plugin syncs specs to issue tracker |
| `pre-review` | Added to stage 2 parallel reviewers | Stack plugin runs framework-specific checks |
| `post-commit` | After each atomic commit | Memory plugin saves embeddings |
| `on-init` | During `mint init` | Stack plugin sets up framework config |

Plugin agents dispatched the same way as core agents — fresh subagent, same isolation rules.
Plugin agents receive the same context as their hook stage (e.g., pre-review gets git diff).

### Plugin Types

| Type | Purpose |
|------|---------|
| `stack` | Framework-specific conventions, reviewers, setup (e.g., Nuxt, React) |
| `pm` | Project management integration (e.g., Linear, Jira) |
| `design` | Design tool integration (e.g., Figma) |
| `memory` | Knowledge persistence (e.g., embeddings, vector search) |

---

## Workspace Context

Workspace awareness is opt-in. If `workspace.repos` is not defined in config, everything works
exactly as before — single-repo mode. When configured, the orchestrator gains cross-repo context
without performing cross-repo git operations.

### Startup

On startup (after plugin loading, before routing), if `config.workspace.repos` exists:

1. Read the repos array once — do not re-read on every task
2. For each repo entry, note: `name`, `path`, `stack`, `role`, `dependsOn`
3. Identify the **current repo** by matching the working directory to a repo path
4. Build a lightweight workspace map (repo names, roles, and dependency edges — not full analysis)

The workspace map is a summary, not a deep scan. It tells agents what exists and how repos relate.

### What Each Agent Type Sees

Not every agent needs the full picture. Context is scoped by role:

| Agent type | Workspace context |
|------------|-------------------|
| Planner | Full workspace map — knows all repos, their stacks, roles, and dependency relationships |
| Researcher | Full workspace map — can search across repos for patterns and usage |
| Spec reviewer | Current repo + its `dependsOn` repos — checks that scope doesn't leak across boundaries |
| Stage 2 reviewers | Current repo context only — they review diffs, not architecture |
| Documenter | Current repo context only |
| Shipper | Full workspace map — needs to sequence work that may span dependencies |

### Cross-Repo Awareness Behaviors

The orchestrator provides context but never performs cross-repo git operations (no cross-repo
commits, checkouts, or merges). Agents use workspace context for awareness only:

**Planning:**
- When decomposing a feature, if work touches a dependency repo, the planner notes it in the
  spec's `<workspace-impact>` field (e.g., "requires SDK changes in my-sdk")
- Specs that affect multiple repos get explicit call-outs so the user can coordinate

**Reviewing:**
- Spec reviewer checks whether changes in a dependency repo could break dependents
- If a spec modifies a shared interface (e.g., an SDK method), the reviewer flags downstream repos
  that consume it

**Researching:**
- Researcher can scan dependent repos for patterns, usage examples, and conventions
- Cross-repo search helps find how an API is consumed before changing it

### Workspace Impact in Spec Execution

If a spec includes `<workspace-impact>`:

1. The orchestrator includes the affected repos in the execution summary
2. The finish step reports: "This change affects: repo-a, repo-b — coordinate before merging"
3. No automated cross-repo actions — the user decides how to handle multi-repo changes

---

## What Agents Receive

Every subagent gets exactly what it needs — no more, no less:

| Agent | Receives |
|-------|----------|
| Planner | Feature description OR spec XML + config + hard blocks + issues.md + wins.md + retry history (if rewrite) + full workspace map (if configured) |
| Researcher | Question + config + full workspace map (if configured) |
| Spec reviewer | Spec XML + git diff + current repo and dependsOn repos from workspace (if configured) |
| Stage 2 reviewers | Git diff + relevant docs (conventions, business, as configured) + current repo context (if configured) |
| Documenter | File path + file description + change summary + current repo context (if configured) |
| Shipper | Confirmed ship plan + config + hard blocks + full workspace map (if configured) |
| Verifier | Config only |
