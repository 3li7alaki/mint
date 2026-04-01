# Dream Mode

Consolidate learning data — prune, promote, archive, report.

Complementary to Claude Code's `autoDream` (if/when it ships). Claude's dream handles
generic conversation memory. Mint's dream handles project-specific learning: issues,
instincts, wins, metrics. No overlap — they enhance each other.

---

## Trigger Conditions

Dream runs when ANY is true:
- User explicitly invokes (`"dream"`, `"consolidate learning"`, `"clean up learning"`)
- `mint dream` CLI command
- Auto-trigger: 7+ days since last dream AND 10+ new JSONL entries

Auto-trigger check (orchestrator does this on `mint resume-work` or first plan mode invocation):
1. Read `.mint/dream-report.md` → parse date from header
2. If >7 days old (or missing) → count total JSONL entries across all files
3. If 10+ entries since last dream → suggest: "Learning data is stale. Run dream? (Y/n)"
4. Don't force — suggest only

## Process

1. Check `.mint/.dream-lock` — if locked and <1 hour old, abort
2. **Dispatch tier: background** (`run_in_background: true`) — consolidation takes 30-60s
3. Dispatch `mint-dream-consolidator` with all learning files
4. On completion, read and display the summary
5. Delete session state

## Output

```
[mint] dream · consolidate — dispatching dream consolidator...
[mint] dream · consolidate — complete

Dream consolidation complete.
  Issues: 12 → 8 active, 3 resolved, 1 archived
  Instincts: 15 → 12 active, 2 decayed, 1 removed
  Promotion candidates: 2 (review in .mint/dream-report.md)
  Report: .mint/dream-report.md
```

## Post-Dream

If promotion candidates exist, offer:
"2 instincts are ready for promotion. Review now? (Y/n)"

If yes → show each candidate with its data, let user decide:
- Promote to `docs/conventions.md`
- Promote to `.mint/hard-blocks.md`
- Dismiss (remove candidate flag)
- Skip (leave for next dream)
