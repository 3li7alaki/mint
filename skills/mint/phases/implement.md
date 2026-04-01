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

`mint-planner` subagent. **Dispatch tier: background** (`run_in_background: true`).

Build prompt from `templates/agent-context.md` → "Planner (implement)" section.
Pass ONLY dynamic context — the planner's `.md` file is its system prompt (cached).

The planner implements, runs gates, commits (or stages). **That's all it does.**

## Post-Dispatch Verification

1. Read `execution.json` → confirm `gates` field populated
2. Verify gate tier is recorded — `gates.tier` should be `skip`, `quick`, or `full`
3. If tier is `skip`, gates can be empty — that's valid for docs-only changes
4. If autocommit true: verify commit exists (`git log -1`)
5. If autocommit false: verify changes staged (`git diff --cached`)
6. If failure: verify `.mint/issues.jsonl` updated
7. Update `execution.json`: gate results, commit hash, tier
8. Clear `activeSpec` in session state (set to `null`)

## Output

"Spec NNN implemented. Gates: lint ✅ types ✅ tests ✅. Moving to review..."

## On Failure

If gates fail, update pipeline-state to trigger retry protocol per `modes/plan.md`.
