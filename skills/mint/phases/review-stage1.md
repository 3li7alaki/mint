# Phase: Spec Review (Stage 1)

Mandatory review of implementation against spec intent. NOT optional.

---

## Dispatch

`mint-spec-reviewer` subagent.
**Dispatch tier: foreground** (omit `run_in_background`) — fast (<10s), result needed
immediately to decide stage 2 vs fix-blockings.

Build prompt from `templates/agent-context.md` → "Spec Reviewer (Stage 1)" section:
- Spec XML + git diff. Nothing else — don't repeat reviewer instructions in the prompt.

## Read Result

1. Look for verdict line: `PASS` or `FAIL`
2. Update `execution.json`: `reviews.spec` = `"passed"` or `"failed"`
3. If `FAIL` → do NOT proceed to stage 2. Set pipeline-state to `fix-blockings`.

## Output

"Spec review: PASSED." or "Spec review: FAILED — N blocking issues. Fixing..."
