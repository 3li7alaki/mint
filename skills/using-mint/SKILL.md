---
name: using-mint
description: "Pre-modification gate for projects using the mint development framework. Ensures the mint skill is invoked before any file write, edit, or delete — routing tasks to the correct execution mode with quality gates enforced. Use when a .mint/ directory is present in a project, when starting any coding task, or when about to modify files in a mint-managed repository."
---

<MANDATORY-RULE>
For ANY task that writes, edits, or deletes files — invoke `mint` FIRST.
You CANNOT use Write, Edit, or Bash (for file ops) tools until mint is invoked.
</MANDATORY-RULE>

# Using mint

## The Rule

**Invoke mint BEFORE modifying any file.** mint auto-routes to the right mode based on task complexity. You do not decide the workflow — mint does.

## How to Invoke

Use the Skill tool to invoke `mint` with the user's task description:

```
Skill({ skill: "mint", args: "Add dark mode toggle to the settings page" })
```

mint will auto-detect the right mode, announce the routing decision, and execute with quality gates and disciplined delegation.

## When mint Applies

| Task | Mode |
|------|------|
| Feature implementation | plan or ship |
| Bug fix | quick or plan |
| Refactor | plan |
| Config change (≤3 files) | quick |
| Research / investigation | research |
| Check quality gates | verify |
| Pure conversation / reading files | **No — mint not needed** |

**If you're tempted to skip mint** ("this is just a small fix", "I'll edit one file directly") — that's exactly when mint matters most. Small fixes use quick mode. mint enforces gates even on single files.

## What mint Provides

- **Auto-routing** — quick/plan/research/ship/verify based on complexity
- **Quality gates** — lint + types + tests enforced before every commit
- **Multi-stage review** — spec review, then parallel audit (quality, security, conventions, tests, business)
- **Learning loop** — past failures become future prevention via issues.md
- **Context protection** — main context stays clean, heavy work delegated to subagents

## Configuration

mint expects `.mint/config.json` in the project root. If it doesn't exist, mint will prompt to run init.

## Override

The user can always override mint's routing:
- "No, just quick-fix it" → switches to quick mode
- "Actually plan this out" → switches to plan mode
- "Skip mint, just do X" → respect the override, but warn about skipped gates
