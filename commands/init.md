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

### 4. Detect doc paths

Scan for common convention/business doc locations and auto-populate config:

**Convention docs** — check for and add any found:
- `docs/conventions/` or `docs/standards/`
- `docs/adr/`
- `CLAUDE.md`, `AGENTS.md`, `CONTRIBUTING.md`
- `.editorconfig`

**Business docs** — check for and add any found:
- `docs/requirements/` or `docs/specs/`
- `docs/business/`
- `MENTAL-MAP.md`
- Any `local-docs/` directory

If convention or business docs are found, enable the corresponding reviewer automatically.

### 5. Create `.mint/` directory

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
    "business": false,
    "performance": false
  },
  "conventions": {
    "docs": []
  },
  "business": {
    "docs": []
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

### 6. Discover plugins

Check for existing plugins:
- Look for `.mint/plugins/` directory with subdirectories containing `manifest.json`
- For each valid plugin found, add its path to `config.plugins`

If the detected stack has a known mint plugin, suggest it:
- Node/TypeScript → "A mint-nuxt or mint-react plugin may be available"
- Python → "A mint-django or mint-fastapi plugin may be available"

Don't auto-install — just inform the user. Plugin installation is opt-in:
```
To install a plugin:
  git clone <plugin-repo> .mint/plugins/<plugin-name>
  Then add ".mint/plugins/<plugin-name>" to config.plugins
```

If plugins are found, run any `on-init` hooks defined in their manifests.

### 7. Detect workspace

Discover sibling repos to build workspace context.

**Detection flow:**

a. Get the **parent directory** of the current project (one level up).

b. List immediate child directories in the parent that contain a `.git/` directory. **Do not scan recursively** — only direct siblings.

c. For each sibling repo, detect its stack using the same logic as step 1 (check for `package.json`, `pyproject.toml`, `Cargo.toml`, `go.mod`).

d. If sibling repos are found, present them to the user:
```
Workspace detected:
  sales-portal/  (nuxt)     ← you are here
  geins-sdk/     (typescript)
  ralph-ui/      (nuxt)

Add these as workspace repos? (y/n)
```

e. If the user confirms, populate `workspace.repos` in `.mint/config.json`:

```json
{
  "workspace": {
    "repos": [
      { "name": "sales-portal", "path": "../sales-portal", "stack": "nuxt", "role": "primary", "dependsOn": [] },
      { "name": "geins-sdk", "path": "../geins-sdk", "stack": "typescript", "role": "dependency", "dependsOn": [] },
      { "name": "ralph-ui", "path": "../ralph-ui", "stack": "nuxt", "role": "reference", "dependsOn": [] }
    ]
  }
}
```

- The **current project** gets `role: "primary"`
- All **siblings** default to `role: "dependency"` (user can change to `"reference"` later)
- `dependsOn` is **empty by default** — relationships require human confirmation, don't auto-populate

f. If **no sibling git repos** are found, skip silently — no prompt, no config entry.

**Important:**
- Never modify sibling repos — only read their project markers
- Some siblings may be unrelated — always ask the user before writing config
- Don't assume dependency relationships between repos

### 8. Add to .gitignore

Add working state directories to `.gitignore` if not already present:

```
.mint/tasks/
.mint/research/
.mint/worktrees/
.mint/plugins/
```

**Committed** (shared, version-controlled): `config.json`, `hard-blocks.md`, `issues.md`
**Ignored** (local, per-developer): `tasks/`, `research/`, `worktrees/`, `plugins/`

### 9. Report

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
Business reviewer: disabled (enable + add docs paths in config.json)
Performance reviewer: disabled (enable in config.json)

Convention docs: <paths if detected, or "none — add paths to conventions.docs">
Business docs: <paths if detected, or "none — add paths to business.docs">
Plugins: <list if found, or "none — see docs for plugin installation">
Workspace: <list repos if configured, or "none detected">

⚠️  Missing gate commands: <list any missing>
    Add these scripts to your project before mint can enforce gates.

Ready to go. Just describe what you want to build.
```
