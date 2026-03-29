# Phase: Implement

Dispatch planner to implement a single spec. Gates + commit only — no reviews.

---

## Pre-Dispatch (orchestrator does this)

1. Set `execution.json` status to `running`, record `startedAt`, add attempt entry
2. **Set active spec** — update session state: `activeSpec` = spec file path.
   Pre-edit hook reads this to enforce `<can-modify>` scope.
3. **Resolve autocommit:**
   - Session override (`autoCommitOverride`) → if set, use it
   - Spec `<autoCommit>` → if true/false (not "inherit"), use it
   - `config.autoCommit` → default (true)
4. **Model routing:** Read spec `<estimate>`:
   - `trivial` → `model: "haiku"`
   - `small` / `medium` → `model: "sonnet"`
   - `large` → `model: "opus"`

## Dispatch

`mint-planner` subagent with: spec XML, resolved autocommit, resolved TDD value.

The planner implements, runs gates, commits (or stages). **That's all it does.**

## Post-Dispatch Verification

1. Read `execution.json` → confirm `gates` field populated
2. If autocommit true: verify commit exists (`git log -1`)
3. If autocommit false: verify changes staged (`git diff --cached`)
4. If failure: verify `.mint/issues.jsonl` updated
5. Update `execution.json`: gate results, commit hash
6. Clear `activeSpec` in session state (set to `null`)

## Output

"Spec NNN implemented. Gates: lint ✅ types ✅ tests ✅. Moving to review..."

## On Failure

If gates fail, update pipeline-state to trigger retry protocol per `modes/plan.md`.
