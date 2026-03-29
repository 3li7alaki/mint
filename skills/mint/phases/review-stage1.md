# Phase: Spec Review (Stage 1)

Mandatory review of implementation against spec intent. NOT optional.

---

## Dispatch

`mint-spec-reviewer` subagent with:
- Spec XML (the intended task)
- Git diff (what was actually implemented)

## Read Result

1. Look for verdict line: `PASS` or `FAIL`
2. Update `execution.json`: `reviews.spec` = `"passed"` or `"failed"`
3. If `FAIL` → do NOT proceed to stage 2. Set pipeline-state to `fix-blockings`.

## Output

"Spec review: PASSED." or "Spec review: FAILED — N blocking issues. Fixing..."
