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

### Delegation

- Each subagent gets **one job** with a clear deliverable
- Subagents **cannot spawn other subagents** — only the orchestrator dispatches
- The orchestrator provides subagents with: the task (spec XML or description), project config
  (`.mint/config.json`), and hard blocks (`.mint/hard-blocks.md`)
- Subagents that need to ask questions return the question — orchestrator relays to user

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
- **Fail twice → stop.** If the same spec fails gates twice, log to `.mint/issues.md` and escalate
  to the user. Never attempt a third run with the same spec.
- **Never push.** Agents commit only. The user reviews and pushes manually.

### Completion check

Before marking a spec as `passed` in `execution.json`, verify all `definitionOfDone` criteria
from `.mint/config.json`:

- `gatesPassing` — all enabled gates returned green
- `specReviewPassed` — stage 1 spec reviewer approved
- `stage2ReviewsPassed` — all enabled stage 2 reviewers approved (no unresolved BLOCKING issues)
- `docCheckPassed` — doc-manifest check completed and documenter dispatched for affected docs
- `screenshotReminder` — if set to `"ui-changes"` and the spec modified UI files (`.vue`, `.tsx`,
  `.jsx`, `.svelte`, `.html`, `.css`), remind the user: "This spec modified UI files — consider
  capturing a screenshot before merging." If `"always"`, remind on every spec. If `false`, skip.

The finish step includes DoD status per spec in the summary.

---

## Execution Flow — Plan Mode

This is the primary workflow for non-trivial tasks.

### 1. Setup

- Check `.mint/config.json` exists (if not, prompt user to run `mint init`)
- **Isolation:** Check `config.isolation.plan` (default: `"worktree"`):
  - `"worktree"` — create worktree at `.mint/worktrees/<task-slug>`, work there
  - `"branch"` — create a feature branch, work in the main checkout
  - `"none"` — work directly on current branch, no isolation
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
- **Model routing:** Read the spec's `<estimate>` field and select the execution model:
  - `trivial` → dispatch with `model: "haiku"`
  - `small` or `medium` → dispatch with `model: "sonnet"`
  - `large` → dispatch with `model: "opus"`
  - If `config.modelRouting` is `false`, use session default for all specs
  - If `config.modelRouting.override` has a mapping for this estimate, use that model
- Dispatch `mint-planner` subagent with the spec XML (full text, not file path)
- Planner implements and runs gates
- Resolve autocommit: session override → spec `<autoCommit>` → `config.autoCommit`
- If gates green AND autocommit resolved to `true`: commit
- If gates green AND autocommit resolved to `false`: skip commit, changes stay staged
- Update `execution.json`: gate results in `gates`, commit hash in `commit` (or `null` if no commit)
- Returns: commit hash + summary, or failure report

**a2) De-sloppify (optional)**
- Runs when ANY of these are true:
  - The spec has `<tdd>true</tdd>` (explicitly or inherited from `config.tdd.default`)
  - `config.tdd.desloppify` is `true` AND the spec has tests
- If triggered:
  - Dispatch `mint-de-sloppifier` subagent with: git diff + spec XML + gate commands
  - De-sloppifier cleans up AI-generated slop (framework tests, over-defensive code, console.log)
  - Runs tests after cleanup to verify nothing broke
  - Returns cleanup report
- Skipped entirely when `config.tdd.desloppify` is `false`

**b) Stage 1 — Spec Review (sequential gate)**
- Dispatch `mint-spec-reviewer` subagent with: spec XML + git diff
- Must pass before stage 2
- If gaps found → planner fixes → spec-reviewer re-reviews
- Update `execution.json`: `reviews.spec` = `"passed"` or `"failed"`

**c) Stage 2 — Audit (parallel)**
- Dispatch ALL enabled reviewers simultaneously (see "Multi-model dispatch" below):
  - `mint-quality-reviewer` — code quality, patterns, DRY
  - `mint-security-auditor` — injection, XSS, auth, secrets
  - `mint-conventions-enforcer` — naming, structure, imports (reads convention docs)
  - `mint-test-auditor` — test quality, mock audit
  - `mint-performance-reviewer` — re-renders, N+1, bundle
  - `mint-business-reviewer` — business logic, requirements alignment (reads business docs)
  - `mint-design-reviewer` — design quality, RTL, i18n, accessibility, anti-patterns (if `design.enabled`)
- Each returns: PASS or issues with severity (BLOCKING/WARNING/INFO)
- Update `execution.json`: each reviewer key in `reviews` = `"passed"` or `"failed"`
- Planner fixes BLOCKING + WARNING issues
- Only failed auditors re-run (not all of them)
- 3 review rounds max, then escalate

**d) Completion — MANDATORY CHECKLIST**

Every spec that passes all stages MUST complete ALL of these steps. Do not skip any.

1. **Set execution state** — `execution.json` status → `passed`, record `completedAt`
2. **Doc-manifest check** — read `.mint/doc-manifest.json` (if it exists):
   - For each doc entry: check if any files matching its `sections[].tracks` globs were modified in this spec's diff
   - If matches found: dispatch `mint-documenter` subagent with: the doc path, its description, the matching section IDs, and a summary of what changed
   - This is NOT optional — skipping doc updates is a pipeline violation
3. **Architectural change detection** — check if the diff touches ANY of these:
   - `.mint/config.json` (config schema changed)
   - `skills/mint/SKILL.md` (orchestrator logic changed)
   - `agents/*.md` (agent added/removed/modified)
   - `package.json` or lockfiles (dependencies changed)
   - `CLAUDE.md` (project instructions changed)
   - `templates/*` (templates changed)
   - `cli/commands/*.js` (CLI changed)
   - If YES: dispatch `mint-documenter` for ALL docs with `trigger: "on-architectural-change"` in the manifest
4. **Log win** — if this is the LAST spec in the task and all specs passed, log to `.mint/wins.md`:
   - Date, task slug, what pattern worked, why it worked
5. **Definition of Done** — verify all `config.definitionOfDone` criteria are met:
   - `gatesPassing` — all enabled gates green
   - `specReviewPassed` — stage 1 approved
   - `stage2ReviewsPassed` — no unresolved BLOCKING issues
   - `docCheckPassed` — doc-manifest check completed (step 2 above)
   - `screenshotReminder` — remind if UI files changed

If spec failed and will be rewritten: set status to `rewriting`.
If spec failed twice: set status to `failed`, log to `.mint/issues.md`.

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
- Present summary: tasks, commits, gate results, doc updates, issues
- Show doc-manifest status: which docs were updated, which sections were refreshed
- Offer choices: merge locally / push + PR / keep branch / discard
- If user picks PR: push and create PR

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
3. **Route to agent:**
   - Debug tasks → `browser-debugger` agent
   - Everything else → `browser-runner` agent
4. **Agent executes** — navigate, snapshot, act, verify loop via PinchTab HTTP API (curl)
5. **Return result** — page state, extracted data, task confirmation, or debug report

### Graceful degradation

If PinchTab is not running:
- Return WARNING with start instructions: `pinchtab &`
- Never block the user's workflow
- Suggest installing PinchTab if the binary is not found

### Commands

Users can invoke browser operations directly:
- `/browse <url> [task]` — navigate and interact
- `/screenshot [url]` — capture page screenshot
- `/scrape <url> [what]` — extract structured data

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

Before creating any new specs, the planner MUST:

1. Read `.mint/issues.md` — find relevant past failures (same files, similar patterns)
2. Read `.mint/wins.md` — find relevant successful patterns (similar task types, decomposition strategies)
3. Read `.mint/patterns.md` — find promoted patterns (recurring successes and anti-patterns with
   higher confidence than individual log entries)
3b. Read `.mint/instincts.md` (if it exists) — find auto-extracted project conventions (import
   style, naming, test patterns). These are observed by hooks during normal development and grow
   in confidence as patterns repeat. Controlled by `config.instincts.enabled` (default: `true`).
4. Add relevant past issues as `<pitfalls>` in the new specs
5. Use winning patterns to inform `<steps>` structure and spec decomposition strategy

This is how mint gets smarter over time. Past mistakes become future prevention. Past wins
become future guidance.

### Logging wins

After a full task completes successfully (all specs passed, reviews done), the orchestrator
logs a win to `.mint/wins.md`:

- **Date** — when the task completed
- **Task** — the task slug or feature name
- **Pattern** — what worked (e.g., "split API + UI into separate specs", "included type signatures in context")
- **Why It Worked** — why this pattern led to success (e.g., "kept agent context focused", "prevented scope leak")

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
| `mint config set <key> <value>` | Edit config with dot notation (e.g., `mint config set browser.enabled true`) |
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

mint expects `.mint/config.json` in the project root. Created by `mint init`.

If config doesn't exist when a task comes in, offer to set it up:
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
  "designContextLoaded": false
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

### Lifecycle

1. **On mint invocation:** Write session state with `mintInvoked: true`, task info, and mode
2. **On user autocommit override:** Set `autoCommitOverride` to `true` or `false` — this persists
   for the entire plan/session. Once the user says "no autocommit", ALL specs in the plan respect
   it without asking again.
3. **On task completion:** Delete the session state file (clean slate for next task)
4. **Hooks read this file** to check invocation status and autocommit preference

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

**Orchestrator behavior:** When an agent returns `interrupted`:
1. Read `.mint/stop` for the reason (if provided)
2. Delete the stop file (consumed)
3. Report to user: "Agent interrupted. Completed: X. Remaining: Y. Reason: Z"
4. Ask user how to proceed: resume / restart with changes / abandon

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
