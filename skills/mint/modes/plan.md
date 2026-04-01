# Plan Mode

Primary workflow for non-trivial tasks. State-machine driven — the orchestrator reads
pipeline state after each step and loads the next phase file.

---

## 1. Setup

- Check `.mint/config.json` exists (if not, prompt user to run `mint init`)
- **Isolation:** Check `config.isolation.plan` (default: `"worktree"`):
  - `"worktree"` — create worktree at `.mint/worktrees/<task-slug>`
  - `"branch"` — create feature branch in main checkout
  - `"none"` — work on current branch
- Read `reference/learning-loop.md` — load issues/wins/instincts for planner context
- Check for resumable specs (scan `.mint/tasks/` for non-terminal execution.json)
- **Dream auto-check:** If `.mint/dream-report.md` is >7 days old (or missing) AND total
  JSONL entries across issues/wins/instincts/metrics > 10 since last dream → run dream
  consolidation automatically in background before decomposing. Don't ask — just do it.
  Output: `[mint] plan · setup · dream — auto-consolidating stale learning data...`

## 2. Challenge (optional)

If `config.challenge` is `true` or `"auto"`, challenge the task before planning:
- Scope — too big? Can it be smaller?
- Alternatives — simpler way? Something already exists?
- Risk — what could go wrong? Blast radius?

User can: address concerns, say "just build it", or redirect.

## 3. Decompose

Read `phases/decompose.md` and follow its instructions.
Dispatch `mint-decomposer` with feature description + learning context.
**Verify specs exist** after decomposer returns — hard gate.

## 4. Build Waves

1. Read all spec XML files in `.mint/tasks/<slug>/`
2. Parse `<id>` and `<depends-on>` for each spec
3. Group into parallel waves (specs whose deps are all satisfied)
4. Show the wave plan to user before executing

```
Execution plan (3 waves):
  Wave 1: [001] auth-types, [002] api-schema  (parallel)
  Wave 2: [003] auth-handler, [004] api-routes  (parallel)
  Wave 3: [005] integration-tests
```

## 5. Execute Wave by Wave

For each wave, for each spec — run the **per-spec pipeline**.

### Pipeline State Machine

Before starting a spec, create `.mint/tasks/<slug>/<spec-id>/pipeline-state.json`:

```json
{
  "currentStep": "implement",
  "completedSteps": [],
  "skippedSteps": [],
  "specId": "001",
  "attempts": 0
}
```

**The pipeline loop:**

```
1. Read pipeline-state.json → get currentStep
2. Read phases/<currentStep>.md → get instructions for this step
3. Dispatch agent per phase instructions
4. Verify result (read execution.json)
5. Update pipeline-state.json → advance currentStep
6. OUTPUT STATUS TEXT (mandatory — user is locked out without this)
7. If more steps remain → go to 1
```

### Pipeline Steps (in order)

| Step | Phase file | Agent | Condition |
|------|-----------|-------|-----------|
| implement | `phases/implement.md` | mint-planner | Always |
| desloppify | `phases/desloppify.md` | mint-de-sloppifier | If TDD or config.tdd.desloppify |
| review-stage1 | `phases/review-stage1.md` | mint-spec-reviewer | Always |
| review-stage2 | `phases/review-stage2.md` | reviewer agents | If diff ≥ 30 lines |
| fix-blockings | `phases/fix-blockings.md` | mint-planner | If BLOCKINGs found |
| docs | `phases/docs.md` | mint-documenter | If doc-manifest matches |
| dod | `phases/dod.md` | orchestrator | Always (final gate) |

### Execution Tracking

Before starting each spec, create `execution.json`:
```json
{
  "status": "running",
  "startedAt": "<ISO-8601>",
  "gates": {},
  "reviews": {},
  "commit": null,
  "attempts": []
}
```

The orchestrator owns this file. After each agent returns, read it to verify updates.
If the agent didn't update it, the orchestrator does.

### Parallel Execution

| Isolation | Method | Merge |
|-----------|--------|-------|
| `"worktree"` | Separate `claude -p` sessions via `cli/lib/parallel.js` | Merge branches after wave |
| `"none"` / `"branch"` | Parallel Agent calls in same session | Direct (same working dir) |

Non-worktree safety: specs in the same wave MUST NOT have overlapping `<can-modify>` paths.
If they do, execute sequentially instead.

### Retry Protocol

If a spec fails gates or review:
1. Read failure report, check `attempts[]` count
2. If 2 attempts already → STOP, escalate to user
3. Classify root cause (bad-spec, missing-context, scope-leak, etc.)
4. Rewrite spec based on root cause
5. Re-dispatch with rewritten spec + failure details

### Status Format

Always use: `[mint] plan · <spec> · <step> — <result>`

### Tiered Dispatch

Each phase file specifies its dispatch tier (foreground or background).
See `reference/orchestrator-laws.md` for the full tier table and rules.

**Quick reference:**
- **Foreground** (fast agents — spec reviewer S1, documenter, verifier): result arrives
  immediately, proceed to next step without delay. User is briefly blocked.
- **Background** (slow agents — planner, decomposer, de-sloppifier, fix-blockings, S2
  reviewers): user gets prompt back, can send corrections or stop signals.

After any background agent completes, check for user messages before continuing:
- **Correction** → adjust remaining specs
- **Addition** → incorporate or queue as follow-up
- **Stop** → pause, await direction
- **Unrelated** → acknowledge, continue

## 6. Finish

After all specs pass:
1. Read all execution.json files — confirm every spec `passed`
2. Log win to `.mint/wins.jsonl`
3. Delete session state
4. Present summary: tasks, commits, gate results, doc updates
5. Offer choices: merge locally / push + PR / keep branch / discard
