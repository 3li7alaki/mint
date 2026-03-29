# Phase: Fix BLOCKINGs

Re-dispatch planner to fix blocking issues from reviews.

---

## Trigger

Only runs if spec review (stage 1) or audit (stage 2) found BLOCKING issues.

## Process

1. Collect all BLOCKING + WARNING issues from review reports
2. Re-dispatch `mint-planner` with: spec XML + specific issues to fix
3. Re-run gates after fixes
4. Re-run ONLY the reviewers that returned FAIL (not all)
5. Track round count — max 3 rounds total

## Loop

If BLOCKINGs persist after fix → return to review-stage1 or review-stage2 as appropriate.
Pipeline-state tracks which reviewers failed.

## Escalation

After 3 rounds with unresolved BLOCKINGs → stop and escalate to user:
"BLOCKING issues persist after 3 rounds. Escalating."

## Output

"Fixed N blocking issues. Re-running failed reviewers..."
Or: "BLOCKING issues persist after 3 rounds. Escalating to user."
