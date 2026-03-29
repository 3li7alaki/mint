# How to Write CLAUDE.md Files

Reference standard for writing effective CLAUDE.md files. Used by `mint init` to generate
project files and by `mint optimize` to audit existing ones.

---

## The Golden Test

Before writing any line, ask:

> **Would removing this line cause Claude to make a real mistake?**

If not, delete it. Every line competes for attention. Fewer rules = higher compliance.

---

## Budget

| Scope | Max lines | Purpose |
|-------|-----------|---------|
| Project CLAUDE.md | 100 | Team rules, shared via git |
| Global ~/.claude/CLAUDE.md | 30 | Personal prefs across all projects |
| Subdirectory CLAUDE.md | 50 | Domain-specific rules |

Compliance degrades linearly with instruction volume. 20 well-chosen rules outperform
200 comprehensive ones. A 1000-token CLAUDE.md gets ~85% compliance. A 6000-token one
gets ~40%.

---

## What Belongs

- **Commands** — build, test, lint, dev. Things Claude can't guess from reading code.
- **Non-default code style** — only what differs from language conventions. Not "use camelCase
  in JavaScript" (Claude already knows). Yes "use snake_case in JavaScript" (unexpected).
- **Git conventions** — commit format, branch naming, PR process.
- **Architecture decisions** — structural choices Claude can't infer by reading the code.
  "API handlers live in src/api/handlers/" is useful. "We use React" is not (package.json says so).
- **Doc pointers** — "See docs/architecture.md for system design." Don't embed docs, point to them.
- **Gotchas** — non-obvious behaviors that will trip Claude up without warning.

## What Does NOT Belong

- **File listings** — Claude can read the filesystem. Don't list every file and its purpose.
- **Standard conventions** — Claude already knows language idioms, framework patterns, common tools.
- **Linter rules** — belong in ESLint/Biome/Prettier config. Never send an LLM to do a linter's job.
- **API documentation** — link to it, don't embed it. `See @docs/api.md` not 200 lines of endpoints.
- **Tutorials or explanations** — CLAUDE.md is instructions, not a textbook. Use skills for workflows.
- **README content** — if Claude can read your README.md, don't duplicate it.
- **Self-evident advice** — "write clean code", "be careful with security", "add tests."
- **Code snippets** — they go stale fast. Point to source: `See src/auth/handler.ts:42`.

---

## Template

```markdown
# <project-name>

<one-line description>

## Commands

- Build: `<command>`
- Test: `<command>`
- Lint: `<command>`
- Dev: `<command>`

## Code Style

- <only non-obvious conventions>

## Architecture

- <key structural decisions>
- See `docs/architecture.md` for full design

## Workflow

- Commits: `type(scope): description`
- Branches: `feat/<name>` or `fix/<name>` off main

## Gotchas

- <things that will trip Claude up>
```

Each section: 3-10 lines. Total: 40-80 lines.

---

## Hierarchy

Claude Code loads CLAUDE.md files in layers. Use the right layer for the right rules.

| Layer | Location | Loaded | Use for |
|-------|----------|--------|---------|
| Global | `~/.claude/CLAUDE.md` | Every session | Personal prefs (no AI attribution, response style) |
| Project | `./CLAUDE.md` | Every session | Team standards, commands, architecture |
| Rules | `.claude/rules/*.md` | On demand by path | Domain rules scoped to file patterns |
| Skills | `.claude/skills/` | On demand by match | Task-specific workflows |
| Hooks | `.claude/hooks/` | Every tool call | Hard enforcement (100% compliance) |

**Rules of thumb:**
- Instructions = guidance (~80% compliance). For things that SHOULD happen.
- Hooks = laws (100% compliance). For things that MUST happen.
- If Claude keeps ignoring a rule despite emphasis, make it a hook.

### Path-Scoped Rules

```markdown
---
paths:
  - "src/api/**/*.ts"
---

# API Rules
- All endpoints require auth middleware
- Use standard error response format from src/api/errors.ts
```

Only loaded when Claude touches matching files. Reduces noise for everything else.

---

## Emphasis

Claude Code respects emphasis markers. Use sparingly — if everything is IMPORTANT, nothing is.

- `IMPORTANT:` or `MUST` — for rules that cause real breakage if ignored
- `NEVER` — for hard prohibitions
- Regular text — for preferences and conventions

Reserve emphasis for max 3-5 rules. The rest should be regular weight.

---

## Anti-Patterns

### The Kitchen Sink
Stuffing every possible instruction into one file. Claude ignores half of it because
important rules get lost in noise.

**Fix:** Prune ruthlessly. Move domain knowledge to rules/ or skills.

### The File Inventory
```markdown
| File | Purpose |
| src/auth/handler.ts | Handles auth |
| src/auth/middleware.ts | Auth middleware |
... (40 more rows)
```

Claude can read the filesystem. This wastes 40 lines of instruction budget on zero value.

**Fix:** Delete. If Claude needs to find files, it will use Glob/Grep.

### The Linter
```markdown
- Use 2-space indentation
- Always use semicolons
- Prefer single quotes
```

This is ESLint's job, not Claude's. And if your linter config says one thing but CLAUDE.md
says another, you've created a conflict.

**Fix:** Put formatting rules in your linter config. Only mention code style in CLAUDE.md if
it's unusual or can't be expressed in a linter rule.

### The Duplicator
Copy-pasting content from README, docs, or other files. Goes stale immediately.

**Fix:** Reference instead of embed. `See @README.md for project overview.`

### The Emphasizer
```markdown
IMPORTANT: Always use TypeScript.
CRITICAL: Never use any.
MUST: Add type annotations to all functions.
NEVER: Skip type checking.
```

When everything is emphasized, nothing is. Claude stops treating emphasis as a signal.

**Fix:** Reserve emphasis for 3-5 truly critical rules. Everything else is regular weight.

---

## Maintenance

CLAUDE.md is code. Maintain it like code.

- **Audit when things go wrong** — if Claude keeps making a specific mistake, check if the
  relevant rule exists. If it does and Claude ignores it, the file is probably too long.
- **Prune regularly** — remove rules that Claude follows naturally (they're wasting budget).
- **Test changes** — after editing CLAUDE.md, observe whether Claude's behavior actually shifts.
- **Keep it current** — stale instructions (wrong file paths, old commands) actively mislead.
