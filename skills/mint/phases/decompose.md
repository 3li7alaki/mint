# Phase: Decompose

Break a feature into atomic XML specs.

---

## Pre-Dispatch

1. Read `reference/learning-loop.md` for learning context
2. Read `.mint/issues.jsonl` — include relevant past failures
3. Read `.mint/wins.jsonl` — include successful patterns
4. Read `.mint/instincts.jsonl` — include high-confidence instincts (confidence ≥ 3)

## Dispatch

**Dispatch tier: background** (`run_in_background: true`) — decomposition explores the
codebase and may take 20-60s.

Dispatch `mint-decomposer` subagent.
Build prompt from `templates/agent-context.md` → "Decomposer" section:
- Feature description, config, hard blocks, learning context (issues, wins, instincts)

Planner reads existing code, breaks work into atomic specs saved to `.mint/tasks/<slug>/`.

## Verify Specs Exist (hard gate)

After planner returns, VERIFY spec files were created:

1. Check `.mint/tasks/<slug>/` exists and contains `.xml` files
2. If **no XML files** → planner skipped creation. This is a failure:
   - Log to `.mint/issues.jsonl`: "spec-skip: planner returned without creating specs"
   - Re-dispatch with explicit instruction: "Create XML spec files ONLY — do not implement"
   - If second attempt fails → escalate to user
3. If specs found → verify required fields: `<id>`, `<title>`, `<goal>`, `<scope>`, `<steps>`, `<acceptance>`, `<commit>`
4. Only proceed to execution after this gate passes

## Output

Report: list of specs with titles, dependencies, and wave grouping.
