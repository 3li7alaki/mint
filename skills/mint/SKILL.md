---
name: mint
description: >
  REQUIRED for ALL coding tasks. Invoke this skill BEFORE writing any code, editing any file,
  or making any changes. Covers: bug fixes, features, refactors, config changes, test updates,
  documentation edits — anything that modifies files. Auto-routes to quick/plan/ship/research/verify
  mode. Enforces quality gates, multi-stage review, atomic commits. If you're about to use Write,
  Edit, or modify files — invoke mint first. No exceptions.
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

1. **SSH** — user says "ssh to", "connect to staging/prod", "run X on staging/prod", "check logs on",
   "run migrations on", "tinker on", or any remote server command
   → Run in main context using ssh config from `.mint/config.json`. See "SSH Execution" below.

1b. **Browse** — user says "browse to", "open", "go to", "scrape", "check the site",
   "fill out the form", "screenshot", "what does X look like", "debug in browser"
   → Delegate to browser-runner agent (or browser-debugger for debug tasks). See "Browser Execution" below.

1c. **Design** — user says "design review", "design profile", "design teach", "design steer",
   "design tokens", "design notes", or invokes `/design:*` commands
   → Execute the corresponding design command. See "Design Intelligence" below.

2. **Verify** — user says "verify", "check gates", "audit", "run checks"
   → Delegate to `mint-verifier` subagent

2b. **Build Fix** — user says "build is broken", "fix build errors", "type errors", gate output shows
   build/type failures, or a planner reports gate failures that look like build issues
   → Delegate to `mint-build-error-resolver` subagent

3. **Research** — user says "research", "how to", "what's the best", "compare", "should I use"
   → Delegate to `mint-researcher` subagent

3b. **Refactor/Cleanup** — user says "clean up", "dead code", "unused imports", "remove unused",
   "refactor cleanup"
   → Delegate to `mint-refactor-cleaner` subagent

4. **Quick** — task touches ≤3 files AND scope is obvious (rename, typo, config tweak, bug fix)
   → Run in main context. No subagent. Gates still enforced.

5. **Ship** — user describes multiple features, says "ship", "build all", or lists a batch of work
   → Interview user in main context → delegate execution to `mint-shipper` subagent

6. **Plan** — everything else (single feature, >3 files, unclear scope, architectural work)
   → Delegate to `mint-planner` subagent

### Write session state

After routing (and before announcing), write `.mint/.session-state.json`:

```json
{
  "mintInvoked": true,
  "invokedAt": "<ISO-8601 now>",
  "task": "<user's task description>",
  "mode": "<routed mode>",
  "autoCommitOverride": null,
  "designContextLoaded": false
}
```

If the user included an autocommit flag (`--no-commit` or verbal "no commits"), set
`autoCommitOverride` to `false` (or `true` for explicit commit). This file is read by hooks
and agents — it's the coordination point for the session.

**On task completion**, delete `.mint/.session-state.json` to reset for the next task.

### Announce the route

Always tell the user what you picked and why:

- "This is a quick fix — I'll handle it directly with gates enforced."
- "This needs decomposition — I'll plan it into specs and execute each one."
- "Let me research this first before we build anything."
- "Multiple features — let me interview you on scope, then ship them in phases."

If the user says "no, just quick-fix it" or "actually plan this out" — follow their override.

### Detect autocommit flags

Before routing, scan the user's input for autocommit signals:

- **Explicit flags:** `--no-commit`, `--commit`
- **Verbal:** "no autocommit", "don't commit", "no commits", "skip commits", "stop committing"
- **Positive:** "autocommit on", "start committing", "commit after each"

If detected, set `autoCommitOverride` in `.mint/.session-state.json` immediately. Announce once:
"Autocommit disabled for this session." — then never mention it again. The override persists for
the entire plan/session across all specs.

**Mid-session changes:** If the user says "actually commit from now on" or "stop committing",
update `autoCommitOverride` in session state and announce the change once.

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

### User message awareness

- **NEVER go silent between steps.** Always output a brief status message between dispatching
  agents. This is how Claude Code checks for queued user messages — text output yields control.
- After each subagent returns and before dispatching the next, output status: what completed,
  what's next. This gives the user a chance to redirect, add context, or stop.
- If the user typed something while an agent was running, it will surface after your status
  output. Read it, process it, adjust if needed, then continue.

### Asking the user (one decision per question)

When asking the user a question — whether from the orchestrator or relaying from a subagent —
follow this protocol strictly:

1. **Re-ground context** — state the task, what was just completed, and what decision is needed
   in 1-2 sentences. The user may have context-switched since their last message.

2. **One decision per question.** Never batch unrelated decisions. If you need 3 decisions,
   ask 3 separate questions. Sequential questions > one overloaded question.

3. **Present focused options** with lettered choices and a recommendation:
   ```
   How should we handle the auth token storage?

   A) localStorage — simple, works offline, XSS-vulnerable
   B) httpOnly cookie — secure, requires server changes
   C) In-memory only — most secure, lost on refresh

   Recommendation: B — secure by default, server changes are minimal (1 endpoint).
   ```

4. **Include effort comparison** when relevant: `(human: ~2h / mint: ~15min)` per option.

5. **Never be vague.** "That's interesting, we could look into it" is banned. Take a position.
   "We can defer this" is only acceptable if the cost of doing it now is genuinely high.

### Delegation

- Each subagent gets **one job** with a clear deliverable
- Subagents **cannot spawn other subagents** — only the orchestrator dispatches
- The orchestrator provides subagents with: the task (spec XML or description), project config
  (`.mint/config.json`), and hard blocks (`.mint/hard-blocks.md`)
- Subagents that need to ask questions return the question — orchestrator relays to user

### Repo ownership mode

Check `config.repoMode` (default: `"collaborative"`):

- **`"solo"`** — the user is the sole developer. When an agent discovers issues outside the
  current task's scope (broken import, stale type, obvious bug), fix it proactively. Log what
  was fixed alongside the main task. This prevents "I noticed X is broken but it's out of scope"
  back-and-forth when there's nobody else who might be working on it.

- **`"collaborative"`** — multiple people work in this repo. When an agent finds issues outside
  scope, log them to `.mint/issues.jsonl` but do NOT fix them. Flag to user: "Found N issues
  outside scope — logged to issues." This prevents stepping on other people's work.

### Quality

- **Gates before commit.** Lint + types + tests must pass 100% before any commit.
- **Coverage gate.** If `config.gates.coverage` is configured, coverage must meet the threshold
  before commit. Coverage is checked as part of the test gate — if tests pass but coverage is
  below threshold, the commit is blocked.
- **`tdd` control.** If `config.tdd.default` is `true`, ALL specs get `<tdd>true</tdd>` unless
  the spec explicitly sets `<tdd>false</tdd>`. If `config.tdd.default` is `false` (default),
  specs only use TDD when explicitly opted in via `<tdd>true</tdd>`. This mirrors how `autoCommit`
  works — a global default that individual specs can override. The planner MUST check this config
  before writing specs and set `<tdd>` accordingly.
- **`autoCommit` control.** Autocommit is resolved in priority order:
  1. **Session override** (`autoCommitOverride` in `.mint/.session-state.json`) — if the user said
     "no commits" or used `--no-commit`, this is `false` for all specs in the session. Once set,
     it persists — **never re-ask**.
  2. **Per-spec** (`<autoCommit>` field in spec XML) — `true`/`false` overrides for individual specs.
     `"inherit"` means fall through to the next level.
  3. **Global config** (`config.autoCommit`) — project default. Default: `true`.
  When autocommit is `false`, agents run gates but do NOT commit. Changes stay staged so the user
  can review and commit manually (or batch multiple specs into one commit).
- **Never fix bad output.** If a subagent produces wrong code, diagnose root cause, fix the spec,
  rerun from scratch. Never patch the output.
- **Fail twice → stop.** If the same spec fails gates twice, log to `.mint/issues.jsonl` and escalate
  to the user. Never attempt a third run with the same spec.
- **Never push.** Agents commit only. The user reviews and pushes manually.

### Risk self-regulation (WTF-likelihood)

The orchestrator tracks a **risk score** during task execution. When things go sideways,
the score escalates. When it gets too high, the orchestrator stops and asks the user.

**Risk events:**
- Gate failure: +10%
- Spec retry (rewrite): +15%
- Fix touches >3 files: +5%
- File modified outside `<can-modify>`: +20%
- After 10 total fixes across all specs: +2% per additional fix
- Revert detected (undoing previous work): +15%

**Thresholds:**
- **> 25%** → warn user: "Risk is elevated. N issues so far. Continue?"
- **> 50%** → stop: "Risk too high. Recommend reviewing progress before continuing."
- **Hard cap:** 30 total fix attempts across all specs → stop regardless of score

The risk score resets to 0 at the start of each task. It's tracked in session state,
not persisted between tasks.

This prevents runaway agents from making things worse when they're clearly struggling.
The earlier fail-twice-stop rule catches individual spec failures; WTF-likelihood catches
the cumulative pattern of "nothing is working well."

### Completion check — Definition of Done (hard gate)

Before marking a spec as `passed` in `execution.json`, the orchestrator MUST verify each
criterion below. This is not advisory — it is a blocking gate. If any criterion fails,
the spec cannot be marked as passed.

| Criterion | How to verify | On failure |
|-----------|---------------|------------|
| `gatesPassing` | Read `execution.json` → `gates.*` all show `pass` | Block — planner must fix and rerun gates |
| `specReviewPassed` | Read `execution.json` → `reviews.spec === "passed"` | Block — planner fixes, spec-reviewer re-reviews |
| `stage2ReviewsPassed` | Read `execution.json` → no reviewer key has unresolved BLOCKING issues | Block — planner fixes BLOCKINGs, failed auditors re-run |
| `docCheckPassed` | Verify doc-manifest was checked (step d.2 in completion) and documenter dispatched for any matches | Block — dispatch documenter for missed sections |
| `screenshotReminder` | If `"ui-changes"`: check if diff touched `.vue/.tsx/.jsx/.svelte/.html/.css` files. If `"always"`: always remind. If `false`: skip. | Warn user (non-blocking) |

The orchestrator reads `execution.json` to verify — it does not trust the planner's verbal
report. If `execution.json` is missing or incomplete, that is itself a failure.

The finish step includes DoD status per spec in the summary.

### Pipeline enforcement

The orchestrator is the enforcement layer. Agents follow instructions; the orchestrator
**verifies** they followed them. Every pipeline stage has a verification step — if the
orchestrator skips verification, the feature is effectively disabled.

| Stage | What orchestrator verifies | On failure |
|-------|---------------------------|------------|
| Decomposition | `.mint/tasks/<slug>/` contains `.xml` files with required fields | Re-dispatch planner with explicit decompose-only instruction |
| Execution tracking | `execution.json` exists and was updated after implementation | Create/update it from agent's return value |
| Autocommit resolution | Resolved value matches what actually happened (commit exists or changes staged) | Log mismatch, fix state |
| Gates | `execution.json.gates.*` all show pass | Planner fixes and reruns |
| De-sloppify trigger | Evaluates `config.tdd` + spec `<tdd>` + spec `<tests>` | Dispatches de-sloppifier if conditions met |
| Spec review | `reviews.spec` in execution.json is `"passed"` | Planner fixes, spec-reviewer re-reviews (max 2 rounds) |
| Stage 2 audit | All enabled reviewers dispatched, all returned results, no unresolved BLOCKINGs | Fix BLOCKINGs, re-run failed auditors (max 3 rounds) |
| Doc-manifest | Tracked files checked against diff, documenter dispatched for matches | Dispatch documenter for missed sections |
| Architectural change | Diff checked against critical file patterns, documenter dispatched | Dispatch documenter for matching trigger docs |
| Win logging | `.mint/wins.jsonl` updated on final spec success | Append win entry |
| DoD | All criteria verified via execution.json reads | Block completion until all criteria pass |
| Session cleanup | `.mint/.session-state.json` deleted on task completion | Delete file |
| Stop signal | Agent `interrupted` status triggers stop file consumption + user prompt | Consume stop file, update execution.json, prompt user |
| Learning loop | Issues/wins/patterns/instincts read and passed to planner context | Read files before dispatch |
| Retry protocol | `attempts[]` length checked before re-dispatch, max 2 | Escalate to user on third attempt |

The orchestrator reads state files (execution.json, session-state, doc-manifest) to verify —
it does not trust agent verbal reports alone.

---

## Execution Flow — Plan Mode

This is the primary workflow for non-trivial tasks.

### 1. Setup

- Check `.mint/config.json` exists (if not, prompt user to run `mint init`)
- **Isolation:** Check `config.isolation.plan` (default: `"worktree"`):
  - `"worktree"` — create worktree at `.mint/worktrees/<task-slug>`, work there
  - `"branch"` — create a feature branch, work in the main checkout
  - `"none"` — work directly on current branch, no isolation
- Read `.mint/issues.jsonl` for relevant past pitfalls
- Check for resumable specs (see "Resuming Interrupted Work" below)

### 1b. Challenge (optional, before decomposition)

If `config.challenge` is `true` or `"auto"`, the orchestrator challenges the task before
planning. This catches scope problems, missed alternatives, and unnecessary work early.

**When it triggers (auto mode):**
- Task description contains "system", "architecture", "redesign", "migrate", "real-time",
  "rewrite", or other large-scope keywords
- Task will likely touch >5 files
- Task involves new infrastructure or external dependencies

**What the challenge covers:**
1. **Scope** — is this too big? Can it be smaller? Is the user asking for more than needed?
2. **Alternatives** — is there a simpler way? Does something already exist in the codebase?
   Check with grep/glob before suggesting new code.
3. **Dependencies** — what else does this need? New packages? External services?
4. **Risk** — what could go wrong? What's the blast radius?
5. **Existing patterns** — does the codebase already have something similar?

**Format:** Present the challenge as a focused question (following the one-decision-per-question
protocol). The user can:
- Address the concerns → orchestrator incorporates into planning
- Say "just build it" → skip challenge, proceed to decomposition
- Redirect → change approach before any code is written

**Config:** `config.challenge: true | false | "auto"` (default: `"auto"`)

### 2. Decompose

Dispatch `mint-planner` subagent with the feature description. Planner:
- Reads existing code for patterns and conventions
- Breaks work into atomic XML specs (saved to `.mint/tasks/<slug>/`)
- Each spec follows `templates/spec.xml` format
- Reports back: list of specs with titles and dependencies

### 2b. Verify specs exist (mandatory gate)

After the planner returns from decomposition, **verify that spec files were actually created**
before proceeding to execution. This is a hard gate — not optional.

1. Check `.mint/tasks/<slug>/` exists and contains `.xml` files
2. If **no XML files found** → the planner skipped spec creation. This is a failure:
   - Log to `.mint/issues.jsonl`: "spec-skip: planner returned without creating spec files for task <slug>"
   - Re-dispatch planner with explicit instruction: "You MUST create XML spec files in
     `.mint/tasks/<slug>/`. Do not implement anything — only decompose and save specs."
   - If second attempt also produces no specs → escalate to user
3. If specs found → read each one and verify it has required fields (`<id>`, `<title>`, `<goal>`,
   `<scope>`, `<steps>`, `<acceptance>`, `<commit>`)
4. Only proceed to step 3 (execution) after this gate passes

This gate exists because agents can silently skip spec creation and jump straight to
implementation, bypassing the entire review pipeline.

### 3. Build dependency graph and execute in waves

Instead of executing specs sequentially, the orchestrator builds a dependency graph from
`<depends-on>` fields and groups independent specs into parallel waves.

**Step 3a: Build the graph**

1. Read all spec XML files in `.mint/tasks/<slug>/`
2. For each spec, parse `<id>` and `<depends-on>` (comma-separated IDs, or "none")
3. Build a DAG (directed acyclic graph): edges from dependency → dependent
4. Group specs into waves — a wave contains specs whose dependencies are ALL satisfied
   (completed in a previous wave or have no dependencies)

**Example:**
```
Specs: 001, 002, 003, 004, 005
Dependencies: 003 depends on 001, 004 depends on 001, 005 depends on 003 and 004

Wave 1: [001, 002]       ← no dependencies, run in parallel
Wave 2: [003, 004]       ← both depend only on 001 (done in wave 1), run in parallel
Wave 3: [005]            ← depends on 003 and 004 (done in wave 2)
```

**Step 3b: Report the execution plan**

Before executing, show the user the wave plan:
```
Execution plan (3 waves):
  Wave 1: [001] auth-types, [002] api-schema  (parallel)
  Wave 2: [003] auth-handler, [004] api-routes  (parallel)
  Wave 3: [005] integration-tests  (sequential — depends on 003, 004)
```

**Step 3c: Execute wave by wave**

For each wave:
- If the wave has **1 spec** → execute sequentially (same as before)
- If the wave has **2+ specs** → dispatch all in parallel using parallel Agent calls
- Wait for ALL specs in the wave to complete before starting the next wave
- If any spec in a wave fails → retry that spec (per retry protocol). Other passed specs
  in the wave are not affected. Failed spec retries in the same wave position.
- After the wave completes, run any shared gates if not isolated (see gate ledger, future)

**MANDATORY: Status output between every step.** The orchestrator MUST output a brief status
message between every major step — between specs, between waves, between review stages, between
pipeline phases. Examples:

- "Wave 1/3 complete (001, 002 passed). Starting wave 2..."
- "Spec 003 implemented. Running gates..."
- "Stage 1 review passed. Dispatching stage 2 reviewers..."
- "All specs complete. Running final verification..."

**Why this matters:** Claude Code checks for queued user messages when the assistant produces
text output. If the orchestrator goes silent while dispatching agents, the user's typed messages
sit unread in the queue. By always outputting status, the orchestrator naturally yields to check
for user input. If the user typed something (a correction, an addition, a "wait stop"), it gets
surfaced at the next status checkpoint.

After outputting status and before dispatching the next step, if the user has responded:
- **Correction** ("not that approach") → adjust remaining specs or re-plan
- **Addition** ("also add X") → note it, incorporate in remaining specs or queue as follow-up
- **Stop** ("wait" / "hold on") → pause execution, await direction
- **Unrelated** → acknowledge briefly, continue

**For each spec in a wave (parallel or sequential):**

**Execution tracking (mandatory):** Before starting a spec, the orchestrator MUST:
1. Create `.mint/tasks/<slug>/<spec-id>/execution.json` (see `templates/execution.json`)
2. Verify the file was created (read it back)
3. Update it at every stage transition listed below
4. After each agent returns, read `execution.json` to verify the agent updated it

If `execution.json` is missing at any point, the orchestrator creates it. The orchestrator
owns this file — agents update it, but the orchestrator is the final authority. If an agent
returns without updating it, the orchestrator updates it based on the agent's return value.

**Parallel execution modes:**

| Isolation config | How parallel specs run | Merge strategy |
|-----------------|----------------------|---------------|
| `"worktree"` | **Separate Claude Code sessions** — each spec gets its own git worktree + its own `claude -p` process via `cli/lib/parallel.js`. Fully independent. No scope conflicts possible. | Merge worktree branches back after wave completes. |
| `"none"` or `"branch"` | **Parallel Agent calls** — specs run as parallel subagents within the same session. Scope enforcement via hook prevents file conflicts. | Direct — all changes are in the same working directory. |

**Worktree mode (recommended for parallel):**

The orchestrator uses `cli/lib/parallel.js` to spawn isolated sessions:
1. `createWorktree(projectRoot, slug)` — creates `.mint/worktrees/spec-NNN/` with its own branch
2. Copies `.env` files and `.mint/` config into the worktree
3. Spawns `claude -p` pointed at the worktree with the spec prompt
4. Collects JSON results from each session
5. Merges successful worktree branches back to the base branch
6. Cleans up worktrees (or preserves on failure for debugging)

Concurrency is limited (default: 3 parallel sessions) to avoid API rate limits and cost explosion.
Configurable via `config.parallel.concurrency` (default: 3) and `config.parallel.maxBudgetPerSpec`
(default: 5.0 USD).

**Non-worktree mode safety:** When specs share the working directory:
- Each spec gets its own `execution.json` (no conflicts — different directories)
- Scope enforcement prevents specs from stepping on each other's files
- Specs in the same wave MUST NOT have overlapping `<can-modify>` paths
- The orchestrator verifies this before dispatching: if two specs in the same wave share
  any `<can-modify>` paths, execute them sequentially instead

**Cleanup:** `mint clean` removes all stale worktrees from `.mint/worktrees/`.

**a) Implementation**
- Set `execution.json` status to `running`, record `startedAt` and new attempt entry
- **Set active spec** — update `.mint/.session-state.json`: set `activeSpec` to the spec's
  file path (e.g., `.mint/tasks/auth/001-handler.xml`). The pre-edit hook reads this to enforce
  `<can-modify>` scope. Clear it (`null`) after the planner returns.
- **Resolve autocommit** (orchestrator responsibility — do this BEFORE dispatching):
  1. Read `.mint/.session-state.json` → if `autoCommitOverride` is not `null`, use it
  2. Read spec `<autoCommit>` → if `true`/`false` (not `"inherit"`), use it
  3. Fall back to `config.autoCommit` (default: `true`)
  4. Pass the resolved value to the planner — planner does NOT re-resolve this
- **Model routing:** Read the spec's `<estimate>` field and select the execution model:
  - `trivial` → dispatch with `model: "haiku"`
  - `small` or `medium` → dispatch with `model: "sonnet"`
  - `large` → dispatch with `model: "opus"`
  - If `config.modelRouting` is `false`, use session default for all specs
  - If `config.modelRouting.override` has a mapping for this estimate, use that model
- Dispatch `mint-planner` subagent with: spec XML (full text), resolved autocommit value,
  and resolved TDD value (from `config.tdd.default` or spec `<tdd>`)
- Planner implements and runs gates
- **Orchestrator verifies after planner returns:**
  - Read `execution.json` → confirm `gates` field is populated
  - If planner reported gates green + autocommit true: verify commit exists (`git log -1`)
  - If planner reported gates green + autocommit false: verify changes are staged (`git diff --cached`)
  - If planner reported failure: verify `.mint/issues.jsonl` was updated
  - Update `execution.json`: gate results in `gates`, commit hash in `commit` (or `null`)

**a2) De-sloppify (orchestrator evaluates trigger)**

The orchestrator — not the agent — evaluates whether de-sloppify runs. Check these conditions:

1. Read `config.tdd.desloppify` — if explicitly `false`, skip entirely
2. Read spec `<tdd>` field — if `true` (explicitly or inherited from `config.tdd.default`), trigger
3. If `config.tdd.desloppify` is `true` AND the spec has `<tests>` entries, trigger
4. If neither condition met, skip

When triggered:
- Dispatch `mint-de-sloppifier` subagent with: git diff + spec XML + gate commands
- De-sloppifier cleans up AI-generated slop (framework tests, over-defensive code, console.log)
- Runs tests after cleanup to verify nothing broke
- **Orchestrator verifies:** gates still pass after de-sloppify (re-run if needed)

**b) Stage 1 — Spec Review (mandatory sequential gate)**

This is NOT optional. The orchestrator MUST dispatch the spec reviewer after implementation.

1. Dispatch `mint-spec-reviewer` subagent with: spec XML + git diff
2. **Orchestrator reads the reviewer's report** — look for the verdict line (`PASS` or `FAIL`)
3. If `FAIL` with BLOCKING issues:
   - Re-dispatch planner to fix the specific issues cited
   - Re-dispatch spec-reviewer to re-review
   - Max 2 review rounds — if still failing, escalate to user
4. Update `execution.json`: `reviews.spec` = `"passed"` or `"failed"`
5. **Gate check:** If `reviews.spec` is `"failed"`, do NOT proceed to stage 2. Block here.

**c) Stage 2 — Audit (mandatory parallel dispatch)**

The orchestrator MUST dispatch enabled reviewers, scaled by diff size.

**Step 1: Measure diff size**
Run `git diff --stat HEAD~1` (or against the pre-spec state) and count total lines changed.

**Step 2: Scale review intensity**

| Diff size | Review level | What runs |
|-----------|-------------|-----------|
| **< 30 lines** | Light | spec review only (stage 1). Skip stage 2 entirely. |
| **30-100 lines** | Standard | spec + quality + conventions |
| **100-300 lines** | Full | spec + all enabled reviewers |
| **300+ lines** | Deep | spec + all enabled reviewers + model escalation (use opus for security + quality) |

This prevents review fatigue on tiny changes and ensures thorough review on large ones.
The user can override: `config.reviewScaling: false` disables scaling (always full review).

**Step 3: Build dispatch list (for standard/full/deep)**

1. Read `config.reviewers` — for each key:
   - `true` or `{ "enabled": true }` → dispatch (if diff size qualifies)
   - `false` or `{ "enabled": false }` → skip
   - Not present → skip
2. Build dispatch list from enabled reviewers:
   - `spec: true` → already ran in stage 1, skip here
   - `quality: true` → dispatch `mint-quality-reviewer`
   - `security: true` → dispatch `mint-security-auditor`
   - `conventions: true` → dispatch `mint-conventions-enforcer`
   - `tests: true` → dispatch `mint-test-auditor`
   - `performance: true` → dispatch `mint-performance-reviewer`
   - `business: true` → dispatch `mint-business-reviewer`
   - If `config.design.enabled` → dispatch `mint-design-reviewer`
3. For **deep** diffs (300+ lines): override `model` to `"opus"` for security and quality
   reviewers regardless of their configured model
4. Dispatch ALL in the list simultaneously (parallel Agent calls)
4. **Orchestrator collects ALL results** — wait for every dispatched reviewer to return
5. Parse each report for severity counts:
   - Count total BLOCKING, WARNING, INFO across all reports
   - Record each reviewer's verdict in `execution.json` → `reviews.<key>` = `"passed"` or `"failed"`
6. If any BLOCKING issues exist:
   - Re-dispatch planner to fix BLOCKING + WARNING issues
   - Re-run ONLY the reviewers that returned FAIL (not all of them)
   - Track round count — max 3 rounds total, then escalate to user
7. **Gate check:** All dispatched reviewers must show `"passed"` before proceeding

**d) Completion — MANDATORY CHECKLIST**

Every spec that passes all stages MUST complete ALL of these steps. The orchestrator
executes each step and verifies it completed. Do not skip any.

**d.1) Definition of Done verification (runs BEFORE marking passed)**
- Run the DoD gate (see "Completion check" in Orchestrator Rules above)
- Read `execution.json` and verify every criterion (gates, spec review, stage 2 reviews)
- If any criterion fails → address the failing criterion first. Do NOT proceed to d.2.
- This runs first because marking `passed` before verifying DoD creates a wrong state.

**d.2) Set execution state**
- Update `execution.json`: status → `passed`, record `completedAt`
- Verify by reading the file back
- Clear `activeSpec` in `.mint/.session-state.json` (set to `null`)

**d.3) Doc-manifest check**
- Read `.mint/doc-manifest.json` (if it exists)
- For each doc entry: check if any files matching its `sections[].tracks` globs were
  modified in this spec's diff (use `git diff --name-only` against the glob patterns)
- If matches found: dispatch `mint-documenter` subagent with: the doc path, its description,
  the matching section IDs, and a summary of what changed
- **Verify:** documenter returned successfully and reported which sections were updated
- This is NOT optional — skipping doc updates when tracked files changed is a pipeline violation
- If no doc-manifest exists, skip this step (not a failure)

**d.4) Architectural change detection**
- Check if the diff (via `git diff --name-only`) touches ANY of these patterns:
  - `.mint/config.json`
  - `skills/mint/SKILL.md` or `SKILL.md`
  - `agents/*.md`
  - `package.json` or lockfiles (`bun.lockb`, `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`)
  - `CLAUDE.md`
  - `templates/*`
  - `cli/commands/*.js`
- If YES: read doc-manifest for docs with `trigger: "on-architectural-change"`
- Dispatch `mint-documenter` for each matching doc
- **Verify:** documenter returned for each dispatched doc

**d.5) Log win**
- If this is the LAST spec in the task AND all specs passed:
  - Append to `.mint/wins.jsonl`: Date, task slug, what pattern worked, why
  - **Verify:** read `.mint/wins.jsonl` to confirm the entry was added
- If not the last spec, skip (not a failure)

**d.6) Session state cleanup (on final spec only)**
- If this is the LAST spec and all passed: delete `.mint/.session-state.json`
- **Verify:** file no longer exists

If spec failed and will be rewritten: set status to `rewriting`.
If spec failed twice: set status to `failed`, log to `.mint/issues.jsonl`.

### 3b. Spec Retry Protocol (orchestrator-driven)

The orchestrator — not the planner — drives retry logic. When a spec fails gates or review:

**Step 1: Read and classify failure**
1. Read the failure report from the subagent
2. Read `execution.json` → check `attempts[]` count. If already 2 attempts → STOP, escalate
3. Cross-reference `.mint/issues.jsonl` for similar past failures (same files, similar patterns)
4. Diagnose root cause category:
   - `bad-spec` → spec was ambiguous, agent had to guess
   - `missing-context` → not enough info about existing code patterns or dependencies
   - `scope-leak` → agent needed files outside declared scope
   - `environment` → missing dependency, broken config, tooling issue
   - `hard-block` → violates a constraint in `.mint/hard-blocks.md`
   - `unknown-pattern` → codebase has a pattern the spec didn't account for

**Step 2: Rewrite the spec (orchestrator does this, not the planner)**
Based on root cause:
- `bad-spec` → narrow scope, add explicit constraints, clarify ambiguous steps
- `missing-context` → add file paths, type definitions, function signatures to `<context>`
- `scope-leak` → tighten `<can-modify>`, expand `<cannot-modify>`, or split into two specs
- `environment` → add environment pre-conditions or notes
- `hard-block` → redesign approach to avoid the constraint
- `unknown-pattern` → add the pattern to `<context>` and `<pitfalls>`

**Step 3: Update tracking and re-dispatch**
1. Update `execution.json`: status → `rewriting`, log the adjustment in `attempts[]`
2. Save the rewritten spec XML to disk (overwrite the original `.xml` file)
3. Dispatch fresh `mint-planner` with the rewritten spec + previous attempt's failure details
4. If rewrite also fails → set status to `failed`, log to `.mint/issues.jsonl`, escalate to user

**Attempt tracking is enforced:** The orchestrator reads `execution.json` `attempts[]` length
before every dispatch. Original attempt + one rewrite = two total. A third attempt is NEVER
dispatched — the orchestrator escalates to the user instead.

### 4. Finish

After all specs complete, the orchestrator runs a final verification pass:

1. **Read all execution.json files** — confirm every spec has status `passed`
2. **Verify DoD** — for each spec, confirm all DoD criteria were met (see Completion check)
3. **Verify doc-manifest** — confirm all stale sections were addressed (documenter dispatched)
4. **Verify win logged** — if all specs passed, confirm `.mint/wins.jsonl` has the new entry
5. **Verify session cleanup** — confirm `.mint/.session-state.json` was deleted
6. **Present summary:**
   - Tasks, commits, gate results, doc updates, issues
   - Doc-manifest status: which docs were updated, which sections were refreshed
   - DoD checklist per spec: all criteria met / which failed
7. **Offer choices:** merge locally / push + PR / keep branch / discard
8. If user picks PR: push and create PR

---

## Execution Flow — Quick Mode

For tasks touching ≤3 files with clear scope.

1. Write an inline spec (not saved to disk):
   - Goal, files to modify, steps, acceptance criteria
2. Implement in main context
3. Run gates
4. Resolve autocommit: session override → `config.autoCommit`
5. If green AND autocommit resolved to `true` → commit
6. If green AND autocommit resolved to `false` → skip commit, inform user changes are ready
6. If red → one retry with fixed approach, then escalate

**Auto-escalation:** If during implementation you realize the task needs >3 files or has
architectural decisions, announce: "This is bigger than expected — switching to plan mode."

No worktree. No reviewers. No spec files. Just gates.

When context-mode is enabled, gate runs use `ctx_execute` to keep output sandboxed even in
quick mode.

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

## Execution Flow — SSH Mode

For running commands on remote servers. Requires `mint-ssh` plugin and `ssh` config in `.mint/config.json`.

### When to route here

User intent involves a remote server:
- "ssh to staging", "connect to prod"
- "run migrations on staging", "run artisan X on prod"
- "check logs on staging", "tinker on prod"
- "restart queue on prod", "clear cache on staging"

### Config lookup

Read `.mint/config.json` for SSH settings:

```json
{
  "ssh": {
    "key": "~/.ssh/my-key",
    "environments": {
      "staging": {
        "host": "1.2.3.4",
        "doppler": { "config": "staging", "var": "SERVER_IP" },
        "user": "root",
        "docker": { "container": "my-app-web" }
      }
    }
  }
}
```

### Execution flow

1. **Resolve host** — check `.mint/ssh-cache.json` first, then Doppler if configured, then static `host`
2. **Build SSH command**:
   - Base: `ssh -i {key} {user}@{host}`
   - If `docker` configured: append `docker exec -i {container} {command}`
   - If no docker: run command directly on host
3. **Execute command** — run via Bash, return output to user
4. **On connection failure with cached host** — invalidate cache, re-fetch from Doppler, retry once

### Cache management

- Cache file: `.mint/ssh-cache.json` (gitignored)
- Structure: `{ "env": { "host": "1.2.3.4", "fetched_at": "ISO-8601" } }`
- Only cache Doppler-fetched hosts (static hosts don't need caching)
- Invalidate on connection failure, re-fetch, retry once

### Example commands

| User says | SSH command executed |
|-----------|---------------------|
| "run migrations on staging" | `ssh -i ~/.ssh/key root@host "docker exec -i container php artisan migrate"` |
| "check queue status on prod" | `ssh -i ~/.ssh/key root@host "docker exec -i container php artisan queue:status"` |
| "restart nginx on staging" | `ssh -i ~/.ssh/key root@host "systemctl restart nginx"` |
| "tail logs on prod" | `ssh -i ~/.ssh/key root@host "docker exec -i container tail -100 storage/logs/laravel.log"` |

### Notes

- Container names may change between deploys — use `docker ps --filter 'name={container}' --format '{{.Names}}' | head -1` to find current name
- For interactive commands (tinker, shell), inform user this requires a terminal
- Always expand `~` in key path to actual home directory

---

## Execution Flow — Browse Mode

For browser automation tasks. Requires `browser.enabled: true` in `.mint/config.json`. Powered by [PinchTab](https://github.com/pinchtab/pinchtab).

### When to route here

User intent involves a browser or web page:
- "browse to", "open", "go to" + URL
- "scrape", "extract from" + URL
- "screenshot", "what does X look like"
- "fill out the form", "click the button", "check the site"
- "debug in browser", "check console errors"

### Config lookup

Read `.mint/config.json` for browser settings:

```json
{
  "browser": {
    "enabled": true,
    "baseUrl": "http://localhost:9867",
    "token": null,
    "headless": true,
    "devServer": "http://localhost:3000",
    "timeout": 30,
    "blockImages": false
  }
}
```

### Execution flow

1. **Check plugin is enabled** — `browser.enabled` must be `true`
2. **Pre-flight** — verify PinchTab is running at `browser.baseUrl` via `/health`
3. **Load session cookies** — if `browser.persistSessions` is `true`:
   a. Read `.mint/.browser-sessions.json`
   b. Find the active session's cookies
   c. Pass cookies to the browser agent for loading via `POST /cookies`
4. **Route to agent:**
   - Debug tasks → `browser-debugger` agent
   - Everything else → `browser-runner` agent
5. **Agent executes** — navigate, snapshot, act, verify loop via PinchTab HTTP API (curl)
6. **Save session cookies** — if `browser.persistSessions` and agent returned cookies:
   a. Read `.mint/.browser-sessions.json` (create if doesn't exist)
   b. Update the active session's cookies with the agent's exported cookies
   c. Write file back
7. **Return result** — page state, extracted data, task confirmation, or debug report

### Session management

Browser sessions persist login state between tasks. Stored in `.mint/.browser-sessions.json`
(gitignored):

```json
{
  "activeSession": "default",
  "sessions": {
    "default": {
      "cookies": [],
      "savedAt": "ISO-8601",
      "url": "http://localhost:3000"
    }
  }
}
```

### Graceful degradation

If PinchTab is not running:
- Auto-start it (`pinchtab &`) if `browser.autoStart` is true
- Poll for health (max 10s) — don't blind `sleep 3`
- If still not running: return WARNING with start instructions
- Never block the user's workflow
- Suggest installing PinchTab if the binary is not found

### Commands

Users can invoke browser operations directly:
- `/browse <url> [task]` — navigate and interact
- `/screenshot [url]` — capture page screenshot
- `/scrape <url> [what]` — extract structured data
- `/browser login <url>` — navigate, user logs in manually, mint saves session
- `/browser sessions` — list saved sessions
- `/browser switch <name>` — switch active session
- `/browser clear` — wipe all saved sessions

---

## Design Intelligence

Design intelligence is a core feature that makes UI/UX awareness automatic. When enabled, every
UI task gets design context injected into planning and design quality checked during review —
without the user asking.

### Startup Detection

On startup (after plugin loading, before routing), check `config.design.enabled`:

1. If `false` or not present: skip design intelligence entirely. No design context or review.
2. If `true`: design intelligence is active. The following hooks engage automatically:
   - **Pre-plan**: `design-context` agent loads project design profile, design notes, relevant
     reference knowledge (typography, color, spatial, motion, interaction, responsive, ux-writing),
     anti-patterns, and shadcn integration — injects all as structured XML into the planner context.
   - **Pre-review**: `design-reviewer` agent runs as a stage 2 parallel auditor, checking the
     diff for AI slop, RTL violations, i18n compliance, accessibility, design consistency,
     performance, and brand compliance.

### Config

```json
{
  "design": {
    "enabled": true,
    "stack": "auto",
    "profile": ".mint/design-profile.json",
    "notes": ".mint/design-notes.md",
    "conventions": [],
    "review": {
      "accessibility": true,
      "consistency": true,
      "performance": true,
      "rtl": false,
      "i18n": false,
      "brand": false
    }
  }
}
```

### UI Task Detection

Design context activates when EITHER condition is true:

1. **Keyword detection** — the task description contains UI keywords: component, page, layout,
   styling, theming, animation, form, dashboard, landing page, card, mobile, responsive, empty
   state, loading state, error state, modal, sidebar, navigation, header, footer, button, input
2. **File-pattern detection** — the task's scope includes files matching `config.design.uiFilePatterns`
   (default: `["*.tsx", "*.jsx", "*.vue", "*.svelte", "*.css", "*.scss", "*.html"]`)

This means: if a user says "fix the admin page" and it touches `.tsx` files, design context
activates even without explicit UI keywords. File patterns are the safety net for implicit UI work.

When routing to plan/ship mode, the orchestrator passes file context (from spec `<can-modify>` or
the user's description) to the design-context agent alongside the task description.

### How Design Context Flows

1. User starts a task (may or may not mention UI explicitly)
2. Orchestrator checks: UI keywords in description OR files matching `design.uiFilePatterns` in scope
3. If either matches → **pre-plan hook** fires → `design-context` agent runs:
   - Loads `.mint/design-profile.json` (project's learned visual DNA)
   - Loads `.mint/design-notes.md` (user's hard rules and preferences)
   - Selects relevant reference docs from `standards/design/reference/` based on task type
   - Loads `standards/design/anti-patterns.md` (AI slop detection)
   - Loads `standards/design/design-direction.md` (aesthetic guidelines)
   - Returns structured `<design-context>` XML that's injected into the planner
4. Planner creates spec with design context baked in
5. Implementation runs with design-aware spec
6. **Pre-review hook** fires → `design-reviewer` agent runs alongside other stage 2 auditors:
   - AI slop test (always — is this distinguishable from generic AI output?)
   - RTL check (if enabled — logical properties, directional icons)
   - i18n check (if enabled — hardcoded strings, inline conditionals)
   - Accessibility (WCAG 2.1 AA — alt text, contrast, focus, semantic HTML)
   - Design consistency (design tokens, spacing scale, component reuse)
   - Performance (animation, reduced motion, bundle)
   - Brand compliance (if brand guide configured)
7. Returns BLOCKING/WARNING/INFO report

### Reference Knowledge

Vendored in `standards/design/reference/` (from Impeccable, Apache 2.0):
- `typography.md` — type scales, font pairing, fluid sizing, OpenType
- `color-and-contrast.md` — OKLCH, palettes, dark mode, WCAG contrast
- `spatial-design.md` — grids, spacing systems, visual hierarchy, container queries
- `motion-design.md` — timing, easing, reduced motion, perception
- `interaction-design.md` — forms, focus, loading, modals, keyboard navigation
- `responsive-design.md` — mobile-first, fluid design, input detection
- `ux-writing.md` — labels, errors, empty states, voice/tone

Plus mint's own:
- `standards/design/rtl.md` — logical CSS properties reference
- `standards/design/i18n.md` — translation standards
- `standards/design/anti-patterns.md` — AI slop detection, design anti-patterns
- `standards/design/design-direction.md` — aesthetic direction and DO/DON'T guidelines

### Commands

- `/design search|system|palette|typography|inspiration` — design intelligence queries
- `/design:profile build|view|update|diff` — manage project design profile
- `/design:notes add|list|remove|clear` — manage design rules and preferences
- `/design:review [target] [--check type] [--fix]` — standalone design review
- `/design:tokens export|sync|validate` — design token management
- `/design:teach` — one-time project design context setup
- `/design:steer <direction>` — steering commands (polish, critique, audit, bolder, quieter, distill, colorize, animate, delight, clarify, harden, adapt, normalize, extract, optimize, onboard)

### Installation

During `mint init`, if design is enabled:
1. Optionally installs Impeccable skill (`npx skills add pbakaus/impeccable`) for editor-level steering commands
2. Auto-detects design assets (components.json, tailwind.config, brand guides)
3. Builds initial design profile if UI code exists
4. Configures review checks based on detected project features (i18n, RTL, brand)

---

## Execution Flow — Verify Mode

For checking quality gates on demand. Uses a two-layer approach to avoid wasting tokens when
everything is green.

### Layer 1 — Bash pre-check (zero tokens)

Run gate commands directly as bash in the main context (this is the one exception to "never run
gates in main context" — these are quick pass/fail checks, not heavy analysis):

1. Run each enabled gate command from `config.gates` (lint, types, tests)
2. If ALL pass → report "All gates green. No issues found." — done, no subagent needed
3. If ANY fail → proceed to layer 2

When context-mode is enabled, Layer 1 gate runs can use `ctx_execute` for cleaner output,
but this is optional -- the main benefit is in Layer 2 where the verifier agent uses it for
deep analysis.

### Layer 2 — Deep analysis (subagent)

Only dispatched when layer 1 detects a problem:

1. Delegate to `mint-verifier` subagent with the failing gate output
2. Verifier runs: deeper analysis of failures + mock audit + hard block scan + open issues count
3. Returns detailed report with root cause analysis and suggested fixes

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

The orchestrator — not the planner — is responsible for providing learning context.
Before dispatching the planner for decomposition, the orchestrator MUST:

1. Read `.mint/issues.jsonl` — include relevant past failures in the planner's context
2. Read `.mint/wins.jsonl` — include relevant successful patterns in the planner's context
3. Read `.mint/patterns.jsonl` — include promoted patterns in the planner's context
4. If `config.instincts.enabled` is `true` (default): read `.mint/instincts.jsonl` and include
   high-confidence instincts (confidence >= 3) in the planner's context

The orchestrator passes this learning context to the planner as part of the dispatch.
The planner then uses it to:
- Add relevant past issues as `<pitfalls>` in new specs
- Use winning patterns to inform `<steps>` structure and decomposition strategy
- Match high-confidence instincts when writing new code

**Orchestrator verification:** After the planner returns specs, spot-check that at least one
spec has `<pitfalls>` populated (if issues.md had relevant entries). If the planner ignored
the learning context entirely, log a WARNING — but don't block execution.

This is how mint gets smarter over time. Past mistakes become future prevention. Past wins
become future guidance.

### Logging wins

After a full task completes successfully (all specs passed, reviews done), the orchestrator
appends a win to `.mint/wins.jsonl`:

```jsonl
{"date":"2025-03-25","task":"auth-reset","pattern":"Split API + UI into separate specs","why":"Kept agent context focused, prevented scope leak"}
```

Similarly, failures are logged to `.mint/issues.jsonl`:

```jsonl
{"date":"2025-03-25","task":"auth-003","severity":"BLOCKING","issue":"Scope leak into shared utils","rootCause":"scope-leak","resolution":"Split into two specs","specFix":"Tighten can-modify"}
```

**JSONL format for all logs.** All learning files (issues, wins, patterns, instincts) use JSONL —
one JSON object per line. Append-only, concurrent-safe, `grep`-able. Never read-parse-modify-write.
To add an entry: `fs.appendFileSync(file, JSON.stringify(entry) + '\n')`.

Agents can read with: `readJsonl(path)` or filter with: `queryJsonl(path, predicate)`.

### Doc-manifest as knowledge graph

The doc-manifest is also a learning artifact. When the conventions-enforcer discovers undocumented patterns, the orchestrator can:
1. Add a new section to the relevant doc in the manifest
2. Dispatch the documenter to write the section
3. The pattern is now tracked — future changes to tracked files automatically trigger doc updates

This closes the loop: code → convention discovery → manifest entry → doc section → future enforcement.

### Log lifecycle

Issues and wins are specific, searchable entries — not general principles. They stay specific so
the planner can match them against concrete files and patterns.

But the logs shouldn't grow forever. When a pattern has been observed enough to become a permanent
rule, it graduates:

1. **Log** — specific entry recorded in `issues.md` or `wins.md`
2. **Recur** — same pattern appears 2-3 times across different tasks
3. **Promote** — codify the pattern into `SKILL.md`, `hard-blocks.md`, a spec template default,
   or an agent prompt rule
4. **Prune** — remove the original entries since the learning is now structural

The orchestrator should flag promotion candidates when it notices repeated patterns during the
learning loop read. Present them to the user: "This pattern has appeared N times — promote to
a permanent rule?"

### Eval-Driven Development

For tracking agent quality over time, use eval templates (see `templates/eval.md`):

- **Capability evals** — define what the implementation must be able to do before coding
- **Regression evals** — ensure changes don't break existing functionality
- **pass@k metrics** — track reliability (pass@3 >= 90% for capabilities, pass^3 = 100% for regressions)
- Store evals in `.mint/evals/<feature>.md`

Evals are optional but recommended for critical features where agent reliability matters.

---

## Doc-Manifest System

mint uses a **doc-manifest** (`.mint/doc-manifest.json`) to track which documentation sections depend on which code artifacts. This replaces the old trigger-only documenter config with structural staleness detection.

### How it works

1. Each doc in the manifest declares **sections** with **tracks** (glob patterns of code files)
2. When code changes, the verifier can check: "did any tracked files change since the doc was last updated?"
3. The documenter reads the manifest to know exactly what to update and where

### Staleness detection strategies

| Strategy | How it detects staleness | Best for |
|----------|------------------------|----------|
| `glob-count` | File count in tracked globs changed (new file added, file deleted) | Directory listings, agent inventories, file trees |
| `content-hash` | File contents changed (hash mismatch) | Config schemas, API references |
| `git-diff` | Tracked files modified since last doc commit | Narrative docs, architecture descriptions |

### Manifest location

- **Project manifest:** `.mint/doc-manifest.json` (committed, shared)
- **Template:** `templates/doc-manifest.json` (for new projects)

The manifest is created during `mint init` and can be customized. The documenter reads it before every update.

---

## mint CLI

The `mint` CLI manages project setup and configuration. You can run these commands via Bash:

| Command | What it does |
|---------|-------------|
| `mint init` | Interactive setup wizard — detects stack, asks 5 questions |
| `mint init --yes` | Headless setup — auto-detects everything, zero prompts. Use this for automated setup. |
| `mint config` | Display current configuration |
| `mint config --global` | Display global user defaults (`~/.mint/config.json`) |
| `mint config set <key> <value>` | Edit config with dot notation (e.g., `mint config set browser.enabled true`) |
| `mint config set --global <key> <value>` | Set a global user default (e.g., `mint config set --global autoCommit false`) |
| `mint config plugins` | Interactive plugin management |
| `mint doctor` | Health check — validates config, gates, tools, plugins |
| `mint doctor --fix` | Health check + auto-repair missing files, incomplete config, .gitignore gaps |
| `mint update` | Update mint to latest version |

**Headless flags for `mint init`:**
- `--yes` / `-y` — skip all prompts, use auto-detected defaults
- `--isolation <mode>` — none, branch, or worktree
- `--tdd true` — enable TDD by default
- `--browser false` — disable browser support
- `--plugins mint-nuxt,mint-e2e` — comma-separated plugin list

When setting up a project automatically, prefer `mint init --yes` over manually creating config files.

---

## Configuration

mint uses a two-layer config system:

1. **Global config** (`~/.mint/config.json`) — user-level defaults that apply across all projects
2. **Project config** (`.mint/config.json`) — project-specific settings, created by `mint init`

### Config resolution order

When the orchestrator reads config, values are resolved in this order (first wins):

1. **Project config** (`.mint/config.json`) — always takes precedence
2. **Global config** (`~/.mint/config.json`) — fallback for user preferences
3. **Hardcoded defaults** — built-in defaults if neither layer has the key

Global config supports these user-preference keys: `reviewers`, `autoCommit`, `tdd`, `isolation`,
`modelRouting`, `instincts`, `hooks`, `definitionOfDone`. Project-specific keys like `stack`,
`packageManager`, `gates`, `browser`, `context`, `design`, and `plugins` are not inherited from
global config — they must be set per-project.

### Managing global config

```bash
mint config --global              # Show global config
mint config set --global key val  # Set a global default
```

Examples:
```bash
mint config set --global autoCommit false
mint config set --global reviewers.security.model opus
mint config set --global isolation.plan worktree
mint config set --global tdd.default true
```

When `mint init` runs, it seeds project config from global defaults. Interactive mode uses
global values as initial values for prompts.

If no project config exists when a task comes in, offer to set it up:
"No mint config found. Want me to set up this project?" — then run `mint init --yes` via Bash.

### Multi-model dispatch

Reviewers can optionally specify which Claude model to use. In `config.reviewers`, each entry
can be a boolean (`true`/`false`) or an object with `enabled` and `model`:

```json
{
  "reviewers": {
    "spec": true,
    "quality": { "enabled": true, "model": "sonnet" },
    "security": { "enabled": true, "model": "opus" },
    "conventions": true
  }
}
```

- `true` = enabled, uses the session's default model
- `{ "enabled": true }` = same as `true`
- `{ "enabled": true, "model": "sonnet" }` = enabled, dispatched with `model: "sonnet"`
- `{ "enabled": false }` = disabled (same as `false`)

Valid model values: `"opus"`, `"sonnet"`, `"haiku"`. When dispatching a reviewer subagent, pass
the `model` parameter to the Agent tool if configured. Different models catch different things —
heavier models for security/quality, lighter models for conventions/formatting.

### Doc-manifest

The doc-manifest (`.mint/doc-manifest.json`) replaces the old `documenters` config array with a richer system. The `documenters` array in config is still supported for backwards compatibility — if both exist, the manifest takes precedence.

See `templates/doc-manifest.json` for the schema.

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
| `design` | Design tool integration (e.g., Figma). Note: core design intelligence is built-in — plugins extend with external tool connections |
| `memory` | Knowledge persistence (e.g., embeddings, vector search) |

---

## Context Mode

Context mode is an optional integration with [context-mode](https://github.com/mksglu/context-mode),
an MCP server that keeps raw tool output out of the context window via sandboxed execution and
provides FTS5 full-text search over indexed content.

### Startup Detection

On startup (after plugin loading, before routing), check `config.context.enabled`:

1. If `false` or not present: skip context mode entirely. All agents use standard tools.
2. If `true`: verify context-mode MCP tools respond by calling `ctx_doctor` or a simple
   `ctx_execute(language: "shell", code: "echo ok")` test.
   - If tools respond: set internal flag. All agents activate their Context Mode sections
     and prefer sandboxed execution for data-heavy operations.
   - If tools do not respond: log WARNING ("Context mode enabled in config but context-mode
     MCP tools are unavailable. Agents will fall back to standard tools."). Set internal flag
     to disabled. Agents fall back to normal tools transparently.

### Agent Dispatch Context

When `config.context.enabled` is `true` and context-mode is verified available, all agents also
receive a reference to `references/context-mode-api.md` and `references/context-mode-strategy.md`.
Agents don't receive the full reference content in their dispatch -- they have routing guidance in
their prompt sections. The config flag tells them to activate their Context Mode behavior.

### Context Protection Enhancement

When context-mode is enabled, the existing context protection rules are enforced automatically
via sandboxed execution. Agents use `ctx_execute` instead of raw Bash for data-heavy operations,
`ctx_execute_file` instead of Read for large files, and `ctx_fetch_and_index` instead of WebFetch
for URLs. This makes context protection structural rather than relying on agent discipline alone.

### Session Continuity

context-mode's session hooks (PreCompact, SessionStart) automatically track file operations,
task state, errors, and decisions. After context compaction, agents can use
`ctx_search(queries: [...], source: "session-events")` to recover working state. No
mint-specific session code is needed -- context-mode handles this natively.

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

## Session State

mint tracks session-level state in `.mint/.session-state.json` (gitignored). This file is the
source of truth for cross-agent and cross-hook coordination within a session.

### Schema

```json
{
  "mintInvoked": true,
  "invokedAt": "ISO-8601",
  "task": "short task description",
  "mode": "quick|plan|ship|research|verify",
  "autoCommitOverride": null,
  "designContextLoaded": false,
  "activeSpec": null
}
```

### Fields

| Field | Type | Purpose |
|-------|------|---------|
| `mintInvoked` | boolean | Whether mint has been invoked this session — hooks check this |
| `invokedAt` | string | ISO timestamp of invocation |
| `task` | string | Current task description |
| `mode` | string | Routed execution mode |
| `autoCommitOverride` | boolean\|null | Session-level override: `true` = force commit, `false` = skip commit, `null` = use config default |
| `designContextLoaded` | boolean | Whether design context was loaded for this task |
| `activeSpec` | string\|null | Path to the currently executing spec XML (relative to project root). Set before dispatching planner, cleared after. The pre-edit hook reads this to enforce `<can-modify>` scope. |

### Lifecycle (orchestrator-enforced)

1. **On mint invocation:** Write session state with `mintInvoked: true`, task info, and mode.
   **Verify:** Read the file back to confirm it was written correctly.
2. **On user autocommit override:** Set `autoCommitOverride` to `true` or `false` — this persists
   for the entire plan/session. Once the user says "no autocommit", ALL specs in the plan respect
   it without asking again.
3. **On task completion (success or final failure):** Delete the session state file (clean slate
   for next task). **Verify:** Confirm the file no longer exists. If deletion fails, warn user.
4. **On task abandonment or escalation:** Also delete session state — stale state from a failed
   task must not leak into the next task.
5. **Hooks read this file** to check invocation status and autocommit preference

### Writing session state

On mint invocation, the orchestrator writes `.mint/.session-state.json`:

```javascript
// Pseudo — orchestrator writes this before routing
{
  "mintInvoked": true,
  "invokedAt": new Date().toISOString(),
  "task": "<user's task description>",
  "mode": "<routed mode>",
  "autoCommitOverride": null,  // or false if user said --no-commit
  "designContextLoaded": false
}
```

### Autocommit override

The user can override autocommit for the current session in three ways:

1. **Inline flag:** "implement X --no-commit" or "implement X --commit"
2. **Verbal override:** "don't autocommit for this plan" or "no commits please"
3. **Mid-session:** "stop committing" / "start committing again"

When an override is detected:
- Set `autoCommitOverride` in session state
- Announce: "Autocommit disabled for this session. Changes will stay staged."
- **Never ask again** — the override persists until the task completes or the user changes it

All agents and the orchestrator read `autoCommitOverride` from session state. If it's not `null`,
it takes precedence over `config.autoCommit`. The check order is:

1. Session state `autoCommitOverride` (if not `null`) → use it
2. Spec-level `<autoCommit>` field (if present) → use it
3. `config.autoCommit` → use it (default: `true`)

---

## What Agents Receive

Every subagent gets exactly what it needs — no more, no less:

| Agent | Receives |
|-------|----------|
| Planner | Feature description OR spec XML + config + hard blocks + issues.md + wins.md + retry history (if rewrite) + full workspace map (if configured) |
| Researcher | Question + config + full workspace map (if configured) |
| Spec reviewer | Spec XML + git diff + current repo and dependsOn repos from workspace (if configured) |
| Stage 2 reviewers | Git diff + relevant docs (conventions, business, as configured) + current repo context (if configured) |
| Documenter | File path + file description + change summary + matching manifest sections + current repo context (if configured) |
| Shipper | Confirmed ship plan + config + hard blocks + full workspace map (if configured) |
| Verifier | Config only |
| De-sloppifier | Git diff + spec XML + gate commands |
| Build Error Resolver | Build/type error output + config + in-scope files |
| Refactor Cleaner | Config + detection tool output + files to analyze |

---

## Agent Control — Pause Signal

Pause is different from stop. Stop means "abort." Pause means "wait — I want to look at something."

### How It Works

**Pause file location:** `.mint/pause`

**User action:**
```bash
touch .mint/pause                              # Pause with no message
echo "let me check something" > .mint/pause    # Pause with reason
```

**Agent behavior:** Agents check for `.mint/pause` at the same checkpoints as `.mint/stop`.
When detected:

1. Agent finishes its current atomic operation (file write, single gate run)
2. Agent **waits** — does NOT stop, does NOT proceed
3. Reports: "Paused. Reason: <contents>. Waiting for resume..."
4. Polls every 5 seconds for the pause file to disappear

**Resume options:**

```bash
rm .mint/pause                                           # Resume — agent continues
echo "change approach to X" > .mint/pause && rm .mint/pause  # Redirect then resume
mv .mint/pause .mint/stop                                # Convert pause to stop
```

When the pause file is removed:
- If `.mint/pause` had content before removal, agent reads it as a **correction** and
  adjusts approach (same as `<correction>` block in retry)
- If empty, agent continues exactly where it left off

**Orchestrator behavior:** When an agent reports "paused":
1. Relay the pause to the user: "Agent paused. <reason>"
2. Wait for the user to resume (don't dispatch further agents)
3. On resume: if user provided feedback, pass it to the agent as a correction

### Why pause exists alongside stop

| Signal | Meaning | Agent behavior | Recovery |
|--------|---------|---------------|----------|
| `.mint/stop` | Abort this approach | Stop, save progress, return | Resume/restart/abandon |
| `.mint/pause` | Wait, I need to check | Freeze in place, poll | Continue/redirect/convert to stop |

Stop requires explicit recovery choices. Pause is lightweight — remove the file and the
agent picks up where it left off.

---

## Agent Control — Stop Signal

Agents can be interrupted mid-execution using a stop file. This gives users control over runaway
or misdirected agents without killing the entire session.

### How It Works

**Stop file location:** `.mint/stop`

**User action:** Create the file to signal agents to stop:
```bash
touch .mint/stop
# Or with a reason:
echo "wrong approach, need to rethink" > .mint/stop
```

**Agent behavior:** All agents (planner, shipper, reviewers) check for the stop file at major
checkpoints (between specs, between review stages, between phases). When detected:

1. Agent stops immediately at the next checkpoint
2. Saves current progress to `execution.json`
3. Returns to orchestrator with status: `"interrupted"`
4. Reports what was completed and what remains

**Orchestrator behavior (mandatory):** When ANY agent returns with status `interrupted`
or mentions being stopped, the orchestrator MUST:
1. Check if `.mint/stop` exists — read its contents for the reason
2. Delete the stop file (it's single-use — consumed on read)
3. Update `execution.json` for the current spec: status → `interrupted`
4. Report to user: "Agent interrupted. Completed: X. Remaining: Y. Reason: Z"
5. Ask user how to proceed: resume / restart with changes / abandon
6. Do NOT dispatch any further agents until the user responds

### Checkpoints

Agents check for stop signal at these points:

| Agent | Checkpoints |
|-------|-------------|
| Planner (decompose) | After analyzing codebase, before writing specs |
| Planner (execute) | Before each file modification, after gates |
| Shipper | Between phases, between tasks within a phase |
| Stage 2 reviewers | Before starting (parallel dispatch checks once) |
| Researcher | Between search/fetch operations |

### Limitations

- **Not instant** — agents finish their current atomic operation before checking
- **Parallel reviewers** — if already dispatched, they run to completion (but orchestrator
  won't act on their results if stop was signaled)
- **No partial commits** — if stopped mid-spec, changes are uncommitted (staged or unstaged)

### Background Execution

For long-running tasks, dispatch agents in background mode:

```
Agent dispatched in background. Task ID: abc123
Monitor: tail -f .mint/tasks/<slug>/output.log
Stop: touch .mint/stop
```

The orchestrator can periodically check agent output and relay progress to user.

### Recovery

After interruption, present the user with options:

```
Agent interrupted.

Completed:
  ✅ [list completed specs/steps]
  🔄 [current spec] (partial)

Remaining:
  ⏳ [list pending specs/steps]

Your feedback: "<contents of .mint/stop>"

How do you want to proceed?
1. Resume with feedback — agent continues with your correction in context
2. Restart spec — discard current spec progress, rerun with adjusted approach
3. Restart task — discard all progress, replan from scratch
4. Abandon — stop entirely, keep what's committed
```

**Resume with feedback:**
- Re-dispatch agent with: remaining work + user's feedback as `<correction>` context
- Agent adjusts approach based on feedback without full replan
- Fastest path to course-correct

**Restart spec:**
- Discard uncommitted changes for current spec
- Optionally rewrite spec based on feedback
- Re-execute from scratch

**Restart task:**
- Discard all uncommitted work
- Return to decomposition with feedback informing new specs

**Abandon:**
- Mark incomplete specs as `interrupted`
- Keep any committed work
- Clean up worktree

The stop file is single-use — once consumed, agents run normally until a new stop is created.

---

## Agent Control — Freeze / Guard / Unfreeze

Protect files and directories from modification. The pre-edit hook **blocks** (not warns)
writes to frozen/guarded paths. Agents see the block reason and must adjust their approach.

### Commands

- `/freeze <path>` — freeze a file or directory. Agent cannot Edit/Write to it.
- `/freeze <glob>` — freeze by pattern: `/freeze src/**/*.test.ts`
- `/guard <path> <reason>` — freeze + explain why. Agent sees the reason in the block message.
- `/unfreeze <path>` — remove freeze/guard from a specific path.
- `/unfreeze --all` — remove all freezes/guards.

### How it works

Freezes are stored in `.mint/.freeze-list.json` (gitignored, session-scoped):

```json
{
  "entries": [
    { "path": "src/auth/", "type": "freeze", "reason": null, "frozenAt": "ISO-8601" },
    { "path": "package.json", "type": "guard", "reason": "no new deps without discussion", "frozenAt": "ISO-8601" }
  ]
}
```

The pre-edit hook reads this file on every Edit/Write and blocks if the target path matches.
Matching rules:
- Exact file match: `/freeze src/auth/middleware.ts` blocks that file
- Directory match: `/freeze src/auth/` blocks all files under that directory
- Glob match: `/freeze src/**/*.test.ts` blocks all test files under src

### Orchestrator behavior

When the user says `/freeze`, `/guard`, or `/unfreeze`:

**`/freeze <path>`:**
1. Resolve path relative to project root
2. Read `.mint/.freeze-list.json` (create if doesn't exist)
3. Add entry: `{ "path": "<resolved>", "type": "freeze", "reason": null, "frozenAt": "<now>" }`
4. Write file back
5. Announce: "Frozen: `<path>`. Agents cannot modify files matching this path."

**`/guard <path> <reason>`:**
1. Same as freeze but with `"type": "guard"` and `"reason": "<reason>"`
2. Announce: "Guarded: `<path>` — <reason>. Agents will see this reason if they try to modify it."

**`/unfreeze <path>`:**
1. Read `.mint/.freeze-list.json`
2. Remove entries matching the path
3. Write file back
4. Announce: "Unfrozen: `<path>`."

**`/unfreeze --all`:**
1. Delete `.mint/.freeze-list.json`
2. Announce: "All freezes/guards removed."

### Agent experience

When an agent tries to Edit/Write a frozen file, the hook returns:

```
[mint] FROZEN: src/auth/middleware.ts is frozen. Use /unfreeze to remove.
```

Or for guarded files:

```
[mint] GUARDED: package.json — no new deps without discussion
```

The agent sees this as a tool call rejection and must adjust its approach — find an
alternative path, skip that file, or ask the orchestrator to relay the blocker to the user.
