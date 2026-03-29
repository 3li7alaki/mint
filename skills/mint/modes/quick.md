# Quick Mode

For tasks touching ≤3 files with clear scope. No subagents, no spec files. Gates enforced.

---

## Pipeline

```
Step 1: Inline spec       → orchestrator
──── output status ────
Step 2: Implement         → orchestrator (main context)
──── output status ────
Step 3: Run gates         → orchestrator (bash)
──── output status ────
Step 4: Commit or stage   → orchestrator
──── output status ────
Step 5: Doc-manifest      → documenter agent (if needed)
──── output status ────
Step 6: Session cleanup   → orchestrator
```

## Step 1: Inline Spec

Write an inline spec (not saved to disk):
- Goal, files to modify, steps, acceptance criteria
- Output: "Quick mode: implementing [task]. Files: [list]."

## Step 2: Implement

Implement in main context. You write the code directly.
- Output: "Implementation complete. Running gates..."

## Step 3: Gates

Run gate commands from `config.gates` (lint, types, tests).
- Resolve autocommit: session override → config default
- If red → one retry with fixed approach, then escalate
- Output: "Gates: lint ✅ types ✅ tests ✅" (or failure details)

When context-mode is enabled, use `ctx_execute` for gate runs.

## Step 4: Commit

- If gates green AND autocommit true → commit with descriptive message
- If autocommit false → leave staged, inform user
- Output: "Committed: [hash]" or "Changes staged for manual review."

## Step 5: Doc-Manifest Check

MANDATORY — quick mode does NOT skip docs.

1. Get changed files from the commit or staged diff
2. Read `.mint/doc-manifest.json` and check for matches
3. If matches: dispatch `mint-documenter` with doc path, section IDs, change summary
4. If no matches: "No tracked docs affected."

## Step 6: Session Cleanup

1. Delete `.mint/sessions/<session-id>.json`
2. Verify the file no longer exists
3. Output summary

## Auto-Escalation

If during implementation you realize the task needs >3 files or has architectural decisions:
"This is bigger than expected — switching to plan mode." Then read `modes/plan.md`.
