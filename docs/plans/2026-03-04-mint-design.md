# mint — Design Document

> Local reference only. Do not commit.

## What

A pluggable, auto-routing Claude Code skill for disciplined agentic development. Main context stays clean as orchestrator, all work delegated to fresh subagents with XML specs, atomic commits, and multi-stage review.

## File Structure

```
mint/
├── SKILL.md                    # Skill manifest + auto-routing orchestrator
├── LICENSE
├── README.md
├── agents/
│   ├── planner.md              # Decomposes → XML specs → executes → commits
│   ├── researcher.md           # Investigates problem, saves structured report
│   ├── shipper.md              # Multi-feature bulk execution
│   ├── verifier.md             # Gates + anti-mock + hard block scan
│   ├── spec-reviewer.md        # Does implementation match the spec?
│   ├── quality-reviewer.md     # Is the code well-written?
│   ├── security-auditor.md     # Injection, XSS, auth, secrets, OWASP top 10
│   ├── conventions-enforcer.md # Naming, file structure, import patterns
│   ├── test-auditor.md         # Test quality, mock audit, coverage gaps
│   ├── performance-reviewer.md # Re-renders, N+1, large imports, bundle impact
│   └── documenter.md           # Updates/creates docs based on triggers
├── commands/
│   ├── init.md                 # Set up .mint/ in a project
│   ├── verify.md               # Manual gate check
│   └── help.md                 # Show guide
└── templates/
    └── spec.xml                # XML spec template
```

## What mint init creates in user's project

```
.mint/
├── config.json
├── hard-blocks.md
├── issues.md
└── tasks/
```

## Auto-Routing (SKILL.md)

| Signal | Route | Agent |
|--------|-------|-------|
| "research", "how to", "compare" | Research | researcher |
| ≤3 files, clear scope | Quick | main context |
| Single feature, >3 files | Plan | planner |
| Multiple features, batch | Ship | shipper |
| "verify", "check", "audit" | Verify | verifier |

Transparent — announces route, user can override.

## XML Spec Format

```xml
<task>
  <id>001</id>
  <title>Short title</title>
  <goal>One sentence outcome</goal>
  <estimate>small|medium|large</estimate>

  <depends-on>none</depends-on>

  <pre-conditions>
    - What must be true before starting
  </pre-conditions>

  <scope>
    <can-modify>exact/file/paths</can-modify>
    <cannot-modify>everything else</cannot-modify>
  </scope>

  <context>
    Relevant existing code, signatures, patterns, line numbers.
  </context>

  <references>
    - docs, ADRs, wiki pages
  </references>

  <steps>
    1. Concrete steps, no inference
  </steps>

  <tests>
    - Test file and test cases spelled out
  </tests>

  <acceptance>
    - Testable conditions
    - lint ✅ types ✅ tests ✅
  </acceptance>

  <pitfalls>
    - Known gotchas from past issues
  </pitfalls>

  <anti-patterns>
    - What NOT to do
  </anti-patterns>

  <no-mocks>What to use real instead of mocking</no-mocks>

  <commit>feat(mint-001): description</commit>

  <gates>lint, types, tests</gates>
</task>
```

## Review Pipeline

```
STAGE 1 — GATE (sequential)
  Spec Reviewer → must pass before stage 2

STAGE 2 — AUDIT (parallel)
  Quality Reviewer
  Security Auditor
  Conventions Enforcer
  Test Auditor
  Performance Reviewer

Severity: BLOCKING (must fix) | WARNING (should fix) | INFO (logged)
Re-run only failed auditors after fixes.
3 rounds max then escalate.
```

## Issue Log (.mint/issues.md)

| Date | Task | Severity | Issue | Root Cause | Resolution | Spec Fix |
|------|------|----------|-------|------------|------------|----------|

Root cause categories: bad-spec, missing-context, scope-leak, environment, hard-block, unknown-pattern

Planner scans issues.md before creating new specs → past issues become <pitfalls>.

## Hard Blocks (.mint/hard-blocks.md)

Universal defaults:
- NEVER git push
- NEVER modify outside scope
- NEVER delete tests to pass gates
- NEVER use any/@ts-ignore/eslint-disable
- NEVER commit with failing gates
- NEVER fix output directly — fix the spec
- NEVER continue after 2 failures on same spec
- NEVER mock internal modules

Context protection:
- NEVER read large files in orchestrator
- NEVER run tests/linters in orchestrator
- Subagents return summaries only

User adds project-specific blocks after init.

## Quick Mode

≤3 files, clear scope → runs in main context, no subagent, no spec files, no reviewers. Gates still enforced. Auto-escalates to plan mode if scope grows.

## Documenters

```json
{
  "documenters": [
    {
      "path": "MENTAL-MAP.md",
      "trigger": "on-task-complete",
      "description": "Project overview",
      "mode": "update"
    },
    {
      "path": "weekly-reports/",
      "trigger": "on-session-end",
      "description": "Daily work log",
      "mode": "template",
      "template": "# {weekday} {date}\n\n## Done\n- \n\n## Next\n- "
    }
  ]
}
```

Modes: update (edit existing) | template (create new from template)
Triggers: on-task-complete | on-session-end | on-architectural-change | on-merge | manual

## Config (.mint/config.json)

```json
{
  "stack": "node",
  "packageManager": "pnpm",
  "gates": {
    "lint": "pnpm lint",
    "types": "pnpm typecheck",
    "tests": "pnpm test"
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

## Isolation

- Plan/ship → git worktree in .mint/worktrees/
- Quick → current branch
- Finish: merge locally / push + PR / keep / discard

## Phase 2 (post-MVP)

- Plugin system (stack plugins, PM plugins, design plugins, memory adapters)
- Workspace registry (multi-repo awareness)
- npm distribution option
