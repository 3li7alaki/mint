---
description: "Check execution status — running tasks, spec progress, gate results, issues"
---

# /status Command

Check mint execution status — running tasks, progress, issues.

## Usage

```
/status
```

## What It Shows

```
mint status
──────────────────────────────

Active task: feat-user-auth
  Spec: 002-add-login-endpoint
  Status: running (attempt 1)
  Started: 2 minutes ago

Progress:
  ✅ 001-create-user-model — committed abc123
  🔄 002-add-login-endpoint — in progress
  ⏳ 003-add-auth-middleware — pending
  ⏳ 004-add-tests — pending

Gates: lint ✅ types ✅ tests ⏳
Stop signal: none

Recent issues: 0
Open issues: 2 — see .mint/issues.md
```

## Implementation

When this command is invoked:

1. **Find active task:**
   - Scan `.mint/tasks/` for directories
   - Find most recent with non-terminal execution.json status

2. **Read execution state:**
   - Parse `execution.json` for each spec
   - Determine: completed, running, pending

3. **Check stop signal:**
   - If `.mint/stop` exists: "Stop signal: ACTIVE — agents will halt"
   - Otherwise: "Stop signal: none"

4. **Count issues:**
   - Parse `.mint/issues.md` for recent entries (last 24h)
   - Count total open issues

5. **Format and display**

## When No Task Is Running

```
mint status
──────────────────────────────

No active task.

Last completed: feat-user-auth (3 hours ago)
  4 specs, all passed
  Commits: abc123, def456, ghi789, jkl012

Open issues: 2 — see .mint/issues.md
```

## Notes

- This is read-only — doesn't affect execution
- Shows the orchestrator's view, not individual agent internals
- Use `tail -f .mint/tasks/<slug>/output.log` for real-time agent output
