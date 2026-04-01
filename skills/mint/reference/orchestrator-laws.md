# Orchestrator Laws — Full Reference

These laws apply in ALL modes that involve code changes (quick, plan, ship, design).
Lightweight modes (research, verify, browse, ssh) are exempt from most of these —
they have their own simpler rules in their mode files.

Load this file when your routed mode involves code modification or agent dispatch.

---

## Context Protection

- NEVER read large files in main context — delegate to subagents
- NEVER run tests/linters in main context (exceptions: quick mode gates, verify mode layer 1)
- NEVER accumulate raw tool output — subagents return summaries only

---

## Agent Dispatch — Tiered Foreground / Background

Not all agents need background dispatch. Fast agents run foreground for tighter feedback
loops. Slow agents run background so the user stays free.

### Dispatch Tiers

| Tier | `run_in_background` | When to use | Agents |
|------|---------------------|-------------|--------|
| **Foreground** | `false` (or omit) | Agent finishes in <15s, result needed immediately for next pipeline step | spec-reviewer (S1), documenter, verifier (layer 2) |
| **Background** | `true` | Agent takes 30s+, user should be free to talk/correct/stop | planner, decomposer, de-sloppifier, fix-blockings, shipper, researcher |
| **Parallel background** | `true` (multiple) | Multiple agents dispatched simultaneously | stage 2 reviewers (quality, security, conventions, etc.) |

### Foreground Rules
- Output status BEFORE dispatch: `[mint] plan · 001 · review-s1 — dispatching spec reviewer...`
- Result arrives immediately — proceed to next step without delay
- User is blocked during foreground dispatch — keep these **fast agents only**
- If a foreground agent takes unexpectedly long, that's acceptable — don't retry

### Background Rules
- Output status BEFORE dispatch: `[mint] plan · 001 · implement — dispatching planner...`
- User gets their prompt back — can send messages, corrections, stop signals
- You'll be notified automatically when the agent completes — do NOT poll or sleep
- After completion notification, check for user messages before continuing

### User Message Awareness (background agents)

When a user message arrives while a background agent is running:
- **Correction** → adjust remaining specs when agent completes
- **Addition** → incorporate or queue as follow-up
- **Stop** → pause pipeline, await direction
- **Unrelated** → acknowledge, continue pipeline when agent completes

---

## Status Format

**Always use this format — never freeform status text:**

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

---

## Quality Gates

### Gate Tier Classification

Not all changes need full gate runs. Before running gates, classify changed files:

| Tier | Trigger | Gates | Status output |
|------|---------|-------|---------------|
| `skip` | Only docs, assets, `.mint/` files | None | "Gates (skip): docs only" |
| `quick` | New test files, CSS/styles | Types only | "Gates (quick): types ✅" |
| `full` | Source code, configs, modified tests | All (lint + types + tests) | "Gates (full): lint ✅ types ✅ tests ✅" |

**Highest tier wins:** If ANY changed file matches `full`, run full gates — even if other
files are docs. Classification patterns are in `config.gates.tiers` (defaults in
`cli/lib/gate-tiers.js`). Unmatched files always default to `full`.

**Spec overrides:** If a spec has explicit `<gates>` overrides, those take precedence over
tier classification. The spec author knows best.

**Exceptions:** De-sloppifier and verifier ALWAYS run full gates regardless of tier —
they need to verify complete integrity after code modification.

### Gate Rules

- Gates (at the classified tier) must pass before commit
- Never fix bad output — diagnose, fix spec, rerun fresh
- Fail twice → stop — log to `.mint/issues.jsonl`, escalate to user
- Never push — agents commit only, user reviews and pushes

---

## Autocommit Resolution

Priority order — first match wins:

1. **Session override** (`autoCommitOverride`) — if set, use it
2. **Per-spec** `<autoCommit>` field — if present in spec XML, use it
3. **Config default** (`config.autoCommit`) — project-level (default: `true`)

---

## Repo Mode

- `"solo"` — fix out-of-scope issues proactively
- `"collaborative"` — log out-of-scope issues, don't fix

---

## Delegation Rules

- One subagent, one job, one clear deliverable
- Subagents cannot spawn other subagents
- Subagents that need user input return the question — orchestrator relays
