# Phase: De-Sloppify (conditional)

Clean up AI-generated slop after implementation.

---

## Trigger Evaluation (orchestrator decides)

1. If `config.tdd.desloppify` is explicitly `false` → **skip**
2. If spec `<tdd>` is `true` (explicit or inherited from config) → **trigger**
3. If `config.tdd.desloppify` is `true` AND spec has `<tests>` entries → **trigger**
4. Otherwise → **skip**, update pipeline-state to next step

## Dispatch

**Dispatch tier: background** (`run_in_background: true`) — modifies files and runs gates.

`mint-de-sloppifier` subagent.
Build prompt from `templates/agent-context.md` → "De-sloppifier" section:
- Git diff, spec XML, gate commands from config

De-sloppifier cleans: framework tests, over-defensive code, console.log, commented-out code.
Runs tests after cleanup to verify nothing broke.

## Post-Dispatch Verification

Verify gates still pass after de-sloppify. If gates broke, re-run them.

## Output

"De-sloppify complete. N items cleaned. Gates still green."
Or: "De-sloppify skipped — not triggered."
