# Automate Mode

Detect repeated workflows and generate project-specific skills from them.

---

## Process

1. **Mine patterns** — dispatch `mint-workflow-miner` (background) to scan session
   traces and learning data for repeated command sequences
2. **Review candidates** — display detected workflow candidates from
   `.mint/workflow-candidates.jsonl`, let user confirm which to automate
3. **Generate skills** — for each confirmed candidate, dispatch `mint-skill-generator`
   (background) to produce a skill in `.mint/skills/<name>/`
4. **Promote** — offer to copy generated skills to `.claude/skills/` for sharing
5. **Record** — append workflow metadata to `.mint/workflow-traces.jsonl`
6. Delete session state on completion

## Trust Gradient

Generated skills start at low trust and graduate:

| Level | Behavior | Promotion |
|-------|----------|-----------|
| `suggest` | Show the skill, ask before running | Default for new skills |
| `confirm` | Run automatically, ask to commit results | After 3 successful uses |
| `auto` | Run and commit without asking | Manual promotion only |

Trust level is stored in the skill's `manifest.json`.

## Output

```
[mint] automate · mine — dispatching workflow miner...
[mint] automate · mine — 3 candidates detected

  1. test-fix-commit (seen 5x) — run tests, fix failures, commit
  2. lint-format (seen 3x) — lint, auto-fix, format
  3. doc-update (seen 4x) — update docs after code change

Generate skills for which? (1,2,3 or all)

[mint] automate · generate — skill "test-fix-commit" created in .mint/skills/test-fix-commit/
  Promote to .claude/skills/? (Y/n)
```
