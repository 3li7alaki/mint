---
name: using-mint
description: Use when starting any conversation - establishes mint as the orchestrator for all coding tasks, requiring mint invocation before any implementation work
---

<EXTREMELY-IMPORTANT>
For ALL coding tasks — features, bug fixes, refactors, anything that touches code — you MUST invoke mint first.

This is not optional. This is not negotiable. You cannot skip this.
</EXTREMELY-IMPORTANT>

# Using mint

## The Rule

**Invoke mint BEFORE writing any code.** mint auto-routes to the right mode based on task complexity. You do not decide the workflow — mint does.

## How to Invoke

Use the `Skill` tool to invoke `mint` with the user's task description. mint will:
- Auto-detect the right mode (quick, plan, research, ship, verify)
- Announce the routing decision
- Execute with quality gates, reviews, and disciplined delegation

## When mint Applies

| Task | Use mint? |
|------|-----------|
| Feature implementation | YES — plan or ship mode |
| Bug fix | YES — quick or plan mode |
| Refactor | YES — plan mode |
| Config change (≤3 files) | YES — quick mode |
| Research / investigation | YES — research mode |
| Check quality gates | YES — verify mode |
| Pure conversation / questions | No |
| Reading files for context | No |

## Red Flags

These thoughts mean STOP — you're about to skip mint:

| Thought | Reality |
|---------|---------|
| "This is just a small fix" | Small fixes use quick mode. Invoke mint. |
| "I'll just edit this one file" | mint enforces gates even on single files. |
| "Let me code first, review later" | mint reviews during execution, not after. |
| "This doesn't need planning" | mint decides that, not you. |
| "I know what to do" | Knowing what ≠ disciplined execution. |

## What mint Provides

- **Auto-routing** — quick/plan/research/ship/verify based on complexity
- **Quality gates** — lint + types + tests enforced before every commit
- **Multi-stage review** — spec review → parallel audit (quality, security, conventions, tests, business)
- **Learning loop** — past failures become future prevention via issues.md
- **Plugin system** — stack, PM, design, and memory integrations
- **Context protection** — main context stays clean, all heavy work delegated to subagents

## Configuration

mint expects `.mint/config.json` in the project root. If it doesn't exist, mint will prompt to run init.

## Override

The user can always override mint's routing:
- "No, just quick-fix it" → switches to quick mode
- "Actually plan this out" → switches to plan mode
- "Skip mint, just do X" → respect the override, but warn about skipped gates
