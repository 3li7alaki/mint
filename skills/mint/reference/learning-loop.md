# Learning Loop — Reference

Load before dispatching planner for decomposition.

---

## What to Read

1. `.mint/issues.jsonl` — past failures as pitfalls
2. `.mint/wins.jsonl` — successful patterns for decomposition
3. `.mint/patterns.jsonl` — promoted patterns
4. `.mint/instincts.jsonl` — auto-learned conventions (confidence ≥ 3 only)

Pass all four as learning context to the planner.

## How Planner Uses It

- Past issues → `<pitfalls>` in new specs
- Winning patterns → inform `<steps>` structure
- High-confidence instincts → code conventions to follow

## Verification

After planner returns specs, spot-check that at least one spec has `<pitfalls>` populated
(if issues had relevant entries). If planner ignored learning context, log WARNING.

## Logging Wins

After full task success (all specs passed + reviewed):
```jsonl
{"date":"2025-03-25","task":"auth-reset","pattern":"Split API + UI into separate specs","why":"Kept agent context focused"}
```

## Logging Issues

After failures:
```jsonl
{"date":"2025-03-25","task":"auth-003","severity":"BLOCKING","issue":"Scope leak","rootCause":"scope-leak","resolution":"Split into two specs"}
```

## JSONL Rules

All learning files are JSONL — one JSON object per line.
Append-only, concurrent-safe. Never read-parse-modify-write.
To add: `fs.appendFileSync(file, JSON.stringify(entry) + '\n')`.

## Promotion Lifecycle

1. **Log** — specific entry in issues/wins
2. **Recur** — same pattern 2-3 times across tasks
3. **Dream** — `mint dream` consolidates: prunes stale, promotes high-confidence, archives old
4. **Promote** — codify into hard-blocks, spec template, or agent prompt (human decides)
5. **Prune** — remove original entries (learning is now structural)

Dream consolidation is the automated step between recurring patterns and promotion.
It flags candidates but never auto-promotes — the human reviews and decides.

## Dream Integration

The `mint-dream-consolidator` agent handles batch consolidation:
- Issue triage (resolve, merge duplicates, escalate recurring)
- Instinct decay (reduce confidence for stale, remove at 0)
- Pattern promotion candidates (confidence ≥ 7, occurrences ≥ 10)
- Win archival (keep 50 active, archive rest)
- Health report generation (`.mint/dream-report.md`)

See `modes/dream.md` for trigger conditions and `agents/dream-consolidator.md` for details.
