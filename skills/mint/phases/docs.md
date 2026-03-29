# Phase: Documentation

Check doc-manifest for affected docs and dispatch documenter.

---

## 1. Doc-Manifest Check

- Read `.mint/doc-manifest.json` (if exists)
- Get changed files: `git diff --name-only HEAD~1`
- For each doc entry: check if changed files match `sections[].tracks` globs
- Build dispatch list: (doc path, matching section IDs, description)

## 2. Architectural Change Detection

Check if changed files match critical patterns:
- `.mint/config.json`, `SKILL.md`, `agents/*.md`, `package.json`, lockfiles,
  `CLAUDE.md`, `templates/*`, `cli/commands/*.js`

If yes: read doc-manifest for docs with `trigger: "on-architectural-change"`.
Add to dispatch list.

## 3. Dispatch Documenter(s)

- If dispatch list non-empty: dispatch `mint-documenter` for each doc
  - Pass: doc path, description, matching section IDs, change summary
  - Verify: documenter reported which sections were updated
- If no matches: "No tracked docs affected." (not a failure)

## Output

"Docs: updated architecture.md (2 sections), conventions.md (1 section)."
Or: "Docs: no tracked files affected, skipping."
