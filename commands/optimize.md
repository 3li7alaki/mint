---
description: >
  Full audit and optimization of a project's mint setup. Detects misconfiguration, missing features,
  stale config, disabled capabilities, and guides the user through fixes. For fresh installs,
  existing projects, or projects that feel like mint isn't working at full capacity.
---

# mint optimize

Comprehensive audit and optimization of this project's mint configuration. Run this when:
- You just installed mint on an existing project
- mint feels like it's not working right
- You updated mint and want to use new features
- You want to make sure you're getting the most out of mint

**This runs in main context** — conversational, not a subagent.

## Process

### Phase 1: Infrastructure Audit

Check the foundational setup:

1. **Run `mint doctor --fix`** via Bash — capture output. This auto-repairs missing files and simple config issues.
2. **Check mint version** — read package.json version, compare with what's installed at ~/.mint/package.json. If mismatch, suggest `mint update`.
3. **Check Claude Code hooks** — verify the pre-edit mint-check hook is installed and working:
   - Look for hooks in `.claude-plugin/plugin.json` or project settings
   - If missing, the auto-invocation warning won't fire — mint could be silently bypassed
4. **Check CLAUDE.md** — does it have the mint section? Is it the latest version?
5. **Check .gitignore** — are all mint temp files ignored?

### Phase 2: Documentation Intelligence

This is critical — most projects have docs but mint doesn't know about them.

1. **Scan project for all documentation files:**
   - Root: README.md, CONTRIBUTING.md, CHANGELOG.md, CLAUDE.md, AGENTS.md, CODE_OF_CONDUCT.md
   - `docs/` directory (recursive — architecture, conventions, guides, ADRs)
   - `local-docs/` or `wiki/` if they exist
   - Any `.md` files referenced in the project but not tracked
   - API docs, OpenAPI/Swagger specs, Postman collections

2. **Check doc-manifest quality:**
   - Does `.mint/doc-manifest.json` exist?
   - How many docs are tracked? How many have sections?
   - Are the sections meaningful (have tracks and staleness strategies)?
   - Do tracked files actually exist?
   - Are there docs in the project that AREN'T in the manifest?

3. **Convention docs audit:**
   - Check `config.conventions.docs` — do all paths resolve?
   - Scan for convention-like files the config doesn't know about:
     - `.editorconfig`, `.prettierrc`, `biome.json`, `.eslintrc.*`
     - `docs/adr/`, `docs/standards/`, `docs/conventions/`
     - `CONTRIBUTING.md` (often has conventions)
   - Suggest adding found convention docs to the config

4. **Business docs audit:**
   - Check `config.business.docs` — do all paths resolve?
   - Scan for business-like docs:
     - `docs/requirements/`, `docs/specs/`, `docs/business/`
     - `PRD.md`, `BRD.md`, `MENTAL-MAP.md`, `PROJECT.md`
     - Any `local-docs/` with business context
   - If business docs found but reviewer disabled → suggest enabling

5. **Offer /doc-setup** — if manifest has empty sections or is missing, offer to run the doc-setup command to build comprehensive section tracking.

### Phase 3: Workspace Detection

Check if the project is part of a multi-repo workspace:

1. **Scan parent directory** for sibling git repos
2. **For each sibling**, detect:
   - Stack (nuxt, react, python, go, etc.)
   - Role (is it an SDK? A frontend? A backend? Documentation?)
   - Dependencies (does package.json reference the sibling?)
3. **If siblings found and no workspace config exists**, present them:
   ```
   Workspace detected:
     my-app/     (nuxt)        ← you are here
     my-sdk/     (typescript)  — likely dependency (referenced in package.json)
     my-docs/    (none)        — likely reference

   Configure workspace awareness? This helps:
   - Planner knows about cross-repo impact
   - Reviewer checks if changes break dependent repos
   - Researcher can search across repos for patterns
   ```
4. **If workspace already configured**, validate:
   - Do all repo paths still exist?
   - Are stacks still accurate?
   - Are dependency relationships correct?

### Phase 4: Agent & Reviewer Optimization

This is where you tailor mint to this specific project:

1. **Gate detection** — are lint, types, and tests all configured?
   - No lint? → scan for ESLint, Biome, Ruff, golangci-lint configs
   - No types? → check for tsconfig.json, mypy, cargo check
   - No tests? → check for vitest, jest, pytest, go test configs
   - Coverage gate? → suggest if test runner supports coverage

2. **Reviewer selection by stack:**

   | Stack | Must-have reviewers | Recommended | Model suggestions |
   |-------|-------------------|-------------|-------------------|
   | Frontend (React/Vue/Svelte) | spec, quality, security, conventions, tests | performance, design | security:sonnet, quality:sonnet, design:haiku |
   | Backend (Node/Python/Go) | spec, quality, security, conventions, tests | business, performance | security:opus, business:opus |
   | Full-stack | all of the above | all | security:opus, design:sonnet |
   | Library/SDK | spec, quality, security, conventions, tests | performance | quality:opus (API surface matters) |
   | CLI/Tool | spec, quality, conventions, tests | — | conventions:sonnet |

3. **Model routing optimization:**
   - Is `modelRouting.enabled`? It should be for cost efficiency.
   - Are there custom overrides that make sense for this project?
   - Large projects benefit from: trivial→haiku, small/medium→sonnet, large→opus
   - Small projects can use sonnet for everything (simpler, still fast)

4. **Core feature recommendations:**
   - `browser.enabled` — YES for any project with a web UI, dev server, or that generates HTML
   - `context.enabled` — YES if codebase has >50 files or test suite takes >5s. Saves ~97% tokens on test output.
   - `design.enabled` — YES if project has ANY UI files (check for .tsx, .jsx, .vue, .svelte, .css, .html)
   - `design.uiFilePatterns` — must match the project's actual UI file extensions
   - `design.review` sub-checks:
     - `accessibility` — YES for any user-facing UI
     - `rtl` — YES if project serves RTL locales (Arabic, Hebrew, Persian)
     - `i18n` — YES if project has i18n/l10n setup (i18next, vue-i18n, etc.)
     - `brand` — YES if project has a brand guide or design system docs

5. **Isolation mode recommendation:**
   - Solo developer? → `none` is fine (simplest)
   - Team? → `branch` (safe, creates feature branches)
   - Large features? → `worktree` (full isolation, advanced)
   - Currently `none` but frequent conflicts? → suggest `branch`

6. **TDD recommendation:**
   - Is coverage gate configured? If not, suggest for mature codebases
   - Is `tdd.default` false? For test-heavy projects, suggest enabling
   - Is `desloppify` on? Should be — it catches AI-generated code smells

### Phase 5: Hard Blocks Customization

Check `.mint/hard-blocks.md` for project-specific rules:

1. **Read current hard blocks** — are there any project-specific entries beyond the defaults?
2. **Suggest common additions based on stack:**
   - Frontend: "NEVER use inline styles when Tailwind/CSS modules are available"
   - API: "NEVER expose internal error details in API responses"
   - Database: "NEVER run migrations without backup confirmation"
   - Auth: "NEVER store secrets in code or config files"
3. **Ask the user** — "Are there any absolute rules for this project that agents should NEVER violate?"

### Phase 6: Learning System Health

Check the feedback loop:

1. **issues.md** — exists? Has entries? Are entries recent or ancient?
2. **wins.md** — exists? Has entries? Is the team logging successes?
3. **instincts.md** — exists? How many patterns? Are confidence levels growing?
4. **patterns.md** — exists? Have any patterns been promoted from issues/wins?
5. **If learning files are empty**, explain their value:
   - "issues.md feeds the planner — past failures become future prevention"
   - "wins.md guides decomposition — past successes inform spec structure"
   - "instincts.md learns your code style — new code matches existing conventions automatically"

### Phase 7: Apply Fixes

For each issue found, present a clear report:

```
mint optimize report
━━━━━━━━━━━━━━━━━━━━━

Infrastructure:          3/3 passing
Documentation:           2/5 — 3 issues
Workspace:               not configured (2 siblings found)
Agents & Reviewers:      5/7 optimal
Core Features:           2/3 enabled
Learning System:         3/4 files present
Hard Blocks:             defaults only (no project-specific rules)

Issues (by priority):

  HIGH
  1. Doc-manifest has 4 docs but 0 tracked sections
     → Run /doc-setup to map doc sections to code artifacts

  2. Convention docs include "docs/conventions.md" but also found:
     — .editorconfig (not tracked)
     — CONTRIBUTING.md (not in conventions.docs)
     → Add to conventions.docs in config

  MEDIUM
  3. context.enabled is false — project has 247 files
     → Enable for ~97% token savings on test/lint output

  4. Reviewer 'performance' disabled — project has 18 .tsx files
     → Enable for re-render, N+1, and bundle checks

  5. Workspace: found 2 sibling repos (my-sdk, my-api)
     → Configure workspace for cross-repo awareness

  LOW
  6. instincts.md has 1 pattern at confidence 5
     → Working correctly, will grow as more files are edited

  7. No project-specific hard blocks
     → Consider adding rules for your domain

Fix issues 1-5 automatically? (yes / pick individually / skip)
```

Then apply fixes:
- Config changes → edit `.mint/config.json`
- Missing files → create from templates
- Doc-manifest → run /doc-setup logic
- Workspace → build workspace config from detected repos
- Convention docs → add paths to config

### Phase 8: Verify

After applying fixes:
1. Run `mint doctor` to verify everything is green
2. Show before/after comparison
3. Suggest next steps:
   - "Run `/design:profile build` to learn your project's visual DNA"
   - "Run `/doc-setup` for deep documentation tracking"
   - "Describe a feature and mint will handle it with full capability"

## Rules

- **Don't force features.** Explain what each feature does and let the user decide.
- **Explain WHY.** Don't just say "enable X" — say "enable X because your project has 247 files and context mode will save ~97% tokens on test output."
- **Be conversational.** This isn't a script — adapt to what you find. Ask questions when the right answer depends on context only the user knows.
- **Fix safely.** Only modify `.mint/` files, `.gitignore`, and CLAUDE.md — never modify source code.
- **Check existing state first.** Don't suggest enabling something that's already enabled.
- **Prioritize by impact.** HIGH = missing quality gates or broken tracking. MEDIUM = disabled features that would help. LOW = nice-to-haves.
- **Run doctor at the end.** Prove the fixes worked.
- **Be specific about models.** Don't just say "assign a model" — recommend the right model based on the reviewer's job (opus for deep reasoning like security/business, sonnet for pattern matching like quality/tests, haiku for mechanical checks like conventions).
