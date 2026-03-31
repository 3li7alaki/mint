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

## User Message Awareness

**All pipeline agents run in background** (`run_in_background: true`). This lets the user
send messages, corrections, or stop signals while agents work. The orchestrator is notified
when each agent completes — do NOT poll or sleep.

**Output status BEFORE every dispatch and AFTER every completion.** Use the standard format
below. After a background agent completes and before dispatching the next, check if the user
sent messages and respond first.

When a user message arrives mid-pipeline:
- **Correction** → adjust remaining specs
- **Addition** → incorporate or queue as follow-up
- **Stop** → pause, await direction
- **Unrelated** → acknowledge, continue

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

- Gates before commit — lint + types + tests must pass 100%
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
