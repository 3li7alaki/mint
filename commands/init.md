---
description: "Set up this project for mint — detects stack, creates .mint/ directory, configures gates"
---

# mint init

Set up this project for disciplined agentic development.

**This runs in the main context** — no subagent needed (one-time setup).

## Steps

### 1. Detect stack

Check for project markers in this order:
- `package.json` → Node/TypeScript
- `pyproject.toml` or `requirements.txt` → Python
- `Cargo.toml` → Rust
- `go.mod` → Go

### 2. Detect package manager

For Node projects:
- `pnpm-lock.yaml` → pnpm
- `yarn.lock` → yarn
- `bun.lockb` → bun
- `package-lock.json` or none → npm

### 3. Detect gate commands

For Node, check `package.json` scripts for:
- `lint` or `lint:fix` → lint gate
- `typecheck` or `type-check` → types gate
- `test` → tests gate

For Python, check for:
- `ruff` in dependencies → `ruff check .`
- `mypy` in dependencies → `mypy .`
- `pytest` in dependencies → `pytest`

If a gate command is missing, warn the user and ask what to use.

### 4. Create `.mint/` directory

Create `.mint/config.json`:
```json
{
  "stack": "<detected>",
  "packageManager": "<detected>",
  "gates": {
    "lint": "<detected command>",
    "types": "<detected command>",
    "tests": "<detected command>"
  },
  "reviewers": {
    "spec": true,
    "quality": true,
    "security": true,
    "conventions": true,
    "tests": true,
    "performance": false
  },
  "isolation": {
    "plan": "worktree",
    "ship": "worktree",
    "quick": "branch"
  },
  "documenters": [],
  "plugins": []
}
```

Create `.mint/hard-blocks.md`:
```markdown
# Hard Blocks — What Agents Can NEVER Do

Violations trigger immediate stop and escalation to user.

## Universal
- NEVER `git push` — human reviews and pushes manually
- NEVER modify files outside declared task scope
- NEVER delete or skip tests to make gates pass
- NEVER use `any` type or `@ts-ignore` to silence type errors
- NEVER disable lint rules to make gates pass
- NEVER commit with failing gates
- NEVER fix bad output directly — reset and fix the spec
- NEVER continue after 2 failures on the same spec
- NEVER mock internal modules — only mock external services

## Context Protection
- NEVER read large files in the main orchestrator context
- NEVER run tests or linters in the main orchestrator context
- Subagents return summaries only

## Project-Specific
<!-- Add your own rules below -->
```

Create `.mint/issues.md`:
```markdown
# Mint Issues & Learnings

_Centralized log. All agent blockers, root causes, and learnings go here._

| Date | Task | Severity | Issue | Root Cause | Resolution | Spec Fix |
|------|------|----------|-------|------------|------------|----------|
```

Create `.mint/tasks/` directory with `.gitkeep`.

### 5. Add to .gitignore

Check if `.mint/` is in `.gitignore`. If not, add it.

The `.mint/` directory is project-local working state — not committed to git by default.
Users can choose to commit config and hard-blocks if they want team-wide settings.

### 6. Report

Show the user:
```
mint initialized

Stack:     <detected>
Package:   <detected>
Gates:
  lint:    <command>
  types:   <command>
  tests:   <command>

Created:
  .mint/config.json
  .mint/hard-blocks.md
  .mint/issues.md
  .mint/tasks/

Reviewers enabled: spec, quality, security, conventions, tests
Performance reviewer: disabled (enable in config.json)

⚠️  Missing gate commands: <list any missing>
    Add these scripts to your project before mint can enforce gates.

Ready to go. Just describe what you want to build.
```
