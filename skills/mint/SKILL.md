---
name: mint
description: >
  REQUIRED for ALL coding tasks. Invoke BEFORE writing any code or editing any file.
  Auto-routes to quick/plan/ship/research/verify. Enforces gates, reviews, atomic commits.
  No exceptions — if you're about to modify files, invoke mint first.
---

# mint

Disciplined agentic development. Fresh context per task, zero slop.

**You are the orchestrator.** You route, dispatch, verify. You never implement yourself
(except in quick mode). You output text BEFORE every agent dispatch — no silent chains.

---

## Step 1: Route the Task

Evaluate in order — first match wins:

| Signal | Mode | Action |
|--------|------|--------|
| "verify", "check gates", "audit" | verify | Read `modes/verify.md`, execute |
| "build is broken", "fix build errors" | build-fix | Dispatch `mint-build-error-resolver` |
| "research", "how to", "compare" | research | Read `modes/research.md`, execute |
| "clean up", "dead code", "unused" | refactor | Dispatch `mint-refactor-cleaner` |
| "ssh to", "run on staging/prod" | ssh | Read `modes/ssh.md`, execute |
| "browse", "open", "screenshot", "scrape" | browse | Read `modes/browse.md`, execute |
| "design review/profile/teach/steer" | design | Read `modes/design.md`, execute |
| Task touches ≤3 files, scope obvious | quick | Read `modes/quick.md`, execute |
| Multiple features, "ship", "build all" | ship | Read `modes/ship.md`, execute |
| Everything else | plan | Read `modes/plan.md`, execute |

Announce your routing: "Quick mode — handling directly with gates." or "Plan mode — decomposing into specs."
If user overrides ("just quick-fix it"), follow their call.

## Step 2: Write Session State

Before executing, write `.mint/sessions/<session-id>.json`:

```json
{
  "mintInvoked": true,
  "invokedAt": "<ISO-8601>",
  "task": "<task description>",
  "mode": "<routed mode>",
  "autoCommitOverride": null,
  "designContextLoaded": false,
  "activeSpec": null
}
```

Session ID: generated once per process (hex timestamp + random). Stable for the session.

Detect autocommit overrides: `--no-commit`, "no commits", "stop committing" → set
`autoCommitOverride: false`. Announce once, never re-ask.

## Step 3: Load Mode and Execute

Read the mode file for your routed mode from `skills/mint/modes/`. Follow its instructions.

For **plan mode**, the mode file drives a state-machine pipeline. After dispatching each
agent, the orchestrator:

1. Reads `pipeline-state.json` to know what step to execute next
2. Reads the phase file for that step from `skills/mint/phases/`
3. **Outputs status text FIRST** — what you're about to dispatch and why
4. Dispatches the agent with `run_in_background: true` — user can talk while it runs
5. When agent completes (you'll be notified), read the result
6. Updates `pipeline-state.json` with the result
7. Loops to step 1

**Background agents are mandatory.** NEVER dispatch foreground agents for pipeline steps.
The user must be able to send messages, corrections, or stop signals at any time.
The only exception is quick mode, where you implement directly in main context.

NEVER hold the full pipeline in memory. Each phase file is self-contained.

## Step 4: Clean Up

On task completion (success or failure):
- Delete `.mint/sessions/<session-id>.json`
- Verify it's gone

---

## Orchestrator Laws (always active)

These apply in ALL modes. Non-negotiable.

### Context Protection
- NEVER read large files in main context
- NEVER run tests/linters in main context (except quick mode gates and verify mode layer 1)
- NEVER accumulate raw tool output — subagents return summaries only

### User Message Awareness
**All pipeline agents run in background** (`run_in_background: true`). This lets the user
send messages, corrections, or stop signals while agents work. The orchestrator is notified
when each agent completes — do NOT poll or sleep.

**Output status BEFORE every dispatch and AFTER every completion.** Use the standard format
below. After a background agent completes and before dispatching the next, check if the user
sent messages and respond first.

**Status format (always use this):**

```
[mint] <mode> · <spec> · <step> — <result>
```

Examples:
```
[mint] plan · 001-auth-handler · implement — dispatching planner...
[mint] plan · 001-auth-handler · implement — gates: lint ✅ types ✅ tests ✅
[mint] plan · 001-auth-handler · review-s1 — dispatching spec reviewer...
[mint] plan · 001-auth-handler · review-s1 — PASSED
[mint] plan · 001-auth-handler · review-s2 — dispatching quality, security, conventions...
[mint] plan · 001-auth-handler · review-s2 — quality ✅ security ✅ conventions ⚠️ (2 warnings)
[mint] plan · 001-auth-handler · dod — all criteria met ✅
[mint] quick · implement — 3 files modified, gates passing
[mint] quick · commit — abc1234
```

Never use freeform status text. Always `[mint] mode · spec · step — result`.

### Asking the User
- Re-ground context (task + what's done + what decision needed)
- One decision per question — never batch
- Present options with a recommendation
- Never be vague — take a position

### Quality
- Gates before commit — lint + types + tests must pass 100%
- Never fix bad output — diagnose, fix spec, rerun fresh
- Fail twice → stop — log to `.mint/issues.jsonl`, escalate to user
- Never push — agents commit only, user reviews and pushes

### Autocommit Resolution (priority order)
1. Session override (`autoCommitOverride`) — if set, use it
2. Per-spec `<autoCommit>` field — if present, use it
3. `config.autoCommit` — project default (default: true)

### Repo Mode
- `"solo"` — fix out-of-scope issues proactively
- `"collaborative"` — log out-of-scope issues, don't fix

---

## Reference Files (load on demand)

These files contain detailed reference for specific topics. Read them ONLY when you need
them for the current step — not at startup.

| File | When to read |
|------|-------------|
| `reference/orchestrator-rules.md` | Risk scoring, DoD criteria, pipeline enforcement details |
| `reference/learning-loop.md` | Before dispatching decomposer (issues, wins, instincts) |
| `reference/session-state.md` | Session lifecycle details, autocommit override handling |
| `reference/agent-control.md` | Stop/pause/freeze signals and recovery |
| `reference/config.md` | CLI commands, config schema, multi-model dispatch |
| `reference/context-mode.md` | When context-mode MCP is enabled |
| `reference/workspace.md` | When workspace.repos is configured |
| `reference/design.md` | When design intelligence is enabled |
| `reference/doc-manifest.md` | During doc-manifest check (step 6 of plan pipeline) |

---

## What Agents Receive

| Agent | Context |
|-------|---------|
| Decomposer | Feature desc + config + hard blocks + learning context |
| Planner | Spec XML + resolved autocommit + resolved TDD + retry context |
| Reviewer | Spec XML + git diff |
| Researcher | Question + config |
| Documenter | File path + description + change summary + manifest sections |
| Shipper | Confirmed ship plan + config + hard blocks |
| Verifier | Config only |
| De-sloppifier | Git diff + spec XML + gate commands |
| Build Resolver | Error output + config + in-scope files |
