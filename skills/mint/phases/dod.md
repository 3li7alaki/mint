# Phase: Definition of Done

Final verification gate. Orchestrator does this — no agent dispatch.

---

## Verification Checklist

Read `execution.json` and verify every criterion:

| Criterion | Check | On failure |
|-----------|-------|------------|
| Gates passing | `gates.*` all show `pass` | Go back to implement phase |
| Spec review passed | `reviews.spec === "passed"` | Go back to review-stage1 |
| Stage 2 passed | No reviewer has unresolved BLOCKINGs | Go back to fix-blockings |
| Doc check passed | Docs phase completed | Go back to docs phase |
| Screenshot reminder | If `"ui-changes"` and diff touched UI files | Warn user (non-blocking) |

If ANY criterion fails → go back to the failing step. Do NOT proceed.

## Set Execution State

- Update `execution.json`: status → `passed`, record `completedAt`
- Verify by reading the file back

## On Final Spec Only

If this is the LAST spec and all passed:

1. **Log win** — append to `.mint/wins.jsonl`: date, task slug, pattern, why
2. **Session cleanup** — delete `.mint/sessions/<session-id>.json`, verify gone

## Output

"Spec NNN: DoD ✅ — all criteria met."

If failed: set status to `rewriting` (retriable) or `failed` (escalate).
