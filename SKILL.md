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

---

## Execution Flow — Plan Mode

This is the primary workflow for non-trivial tasks.

### 1. Setup

- Check `.mint/config.json` exists (if not, prompt user to run `mint init`)
- Create worktree: `.mint/worktrees/<task-slug>`
- Read `.mint/issues.md` for relevant past pitfalls

### 2. Decompose

Dispatch `mint-planner` subagent with the feature description. Planner:
- Reads existing code for patterns and conventions
- Breaks work into atomic XML specs (saved to `.mint/tasks/<slug>/`)
- Each spec follows `templates/spec.xml` format
- Reports back: list of specs with titles and dependencies

### 3. Execute each spec

For each spec (sequenced by `<depends-on>`):

**a) Implementation**
- Dispatch `mint-planner` subagent with the spec XML (full text, not file path)
- Planner implements, runs gates, commits if green
- Returns: commit hash + summary, or failure report

**b) Stage 1 — Spec Review (sequential gate)**
- Dispatch `mint-spec-reviewer` subagent with: spec XML + git diff
- Must pass before stage 2
- If gaps found → planner fixes → spec-reviewer re-reviews

**c) Stage 2 — Audit (parallel)**
- Dispatch ALL enabled reviewers simultaneously:
  - `mint-quality-reviewer` — code quality, patterns, DRY
  - `mint-security-auditor` — injection, XSS, auth, secrets
  - `mint-conventions-enforcer` — naming, structure, imports
  - `mint-test-auditor` — test quality, mock audit
  - `mint-performance-reviewer` — re-renders, N+1, bundle
- Each returns: PASS or issues with severity (BLOCKING/WARNING/INFO)
- Planner fixes BLOCKING + WARNING issues
- Only failed auditors re-run (not all of them)
- 3 review rounds max, then escalate

**d) Documentation**
- Check `documenters` config for `on-task-complete` triggers
- Dispatch `mint-documenter` subagent for each triggered doc

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

## Issue Log Learning Loop

Before creating any new specs, the planner MUST:

1. Read `.mint/issues.md`
2. Find entries relevant to the current task (same files, similar patterns)
3. Add relevant past issues as `<pitfalls>` in the new specs

This is how mint gets smarter over time. Past mistakes become future prevention.

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

## What Agents Receive

Every subagent gets exactly what it needs — no more, no less:

| Agent | Receives |
|-------|----------|
| Planner | Feature description OR spec XML + config + hard blocks |
| Researcher | Question + config |
| Spec reviewer | Spec XML + git diff |
| Stage 2 reviewers | Git diff + relevant convention docs (if any) |
| Documenter | File path + file description + change summary |
| Shipper | Confirmed ship plan + config + hard blocks |
| Verifier | Config only |
