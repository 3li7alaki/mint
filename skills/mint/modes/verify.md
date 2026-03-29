# Verify Mode

Check quality gates on demand. Two-layer approach — zero tokens when everything is green.

---

## Layer 1: Bash Pre-Check (zero tokens)

Run gate commands directly as bash (exception to "never run gates in main context"):

1. Run each enabled gate from `config.gates` (lint, types, tests)
2. If ALL pass → "All gates green. No issues found." — done, no subagent
3. If ANY fail → proceed to Layer 2

When context-mode is enabled, can use `ctx_execute` for cleaner output.

## Layer 2: Deep Analysis (subagent)

Only dispatched when Layer 1 finds problems:

1. Delegate to `mint-verifier` subagent with failing gate output
2. Verifier runs deeper analysis + mock audit + hard block scan + open issues count
3. Returns detailed report with root cause analysis and suggested fixes
4. Delete session state on completion
