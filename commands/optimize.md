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

1. **Run `mint doctor`** via Bash — capture output
2. **Check mint version** — read package.json version, compare with what's installed at ~/.mint/package.json. If mismatch, suggest `mint update`
3. **Check Claude Code hooks** — verify the pre-edit mint-check hook is installed and working:
   - Look for hooks in `.claude-plugin/plugin.json` or project settings
   - If missing, the auto-invocation warning won't fire
4. **Check CLAUDE.md** — does it have the mint section? Is it the latest version?
5. **Check .gitignore** — are all mint temp files ignored?

### Phase 2: Configuration Optimization

Audit the config for maximum capability:

1. **Gates** — are lint, types, and tests all configured? If any are `false`, ask why and suggest alternatives:
   - No lint? — check for ESLint, Biome, Ruff configs the user might have missed
   - No types? — check for tsconfig.json
   - No tests? — check for test runner configs

2. **Reviewers** — are all valuable reviewers enabled?
   - `spec` — MUST be enabled (core quality gate)
   - `quality` — should be enabled
   - `security` — should be enabled
   - `conventions` — should be enabled (auto-discovers patterns)
   - `tests` — should be enabled
   - `business` — enabled if business docs exist
   - `performance` — suggest enabling for frontend projects
   - Each should have a model assignment (not just `true`)

3. **Core features:**
   - `browser.enabled` — suggest for web projects
   - `context.enabled` — suggest for large codebases (>100 files)
   - `design.enabled` — suggest for projects with UI files
   - `design.uiFilePatterns` — present? Covers the project's actual UI file types?

4. **Doc-manifest** — does it exist? Does it have tracked sections?
   - If empty sections — offer to run /doc-setup
   - If missing entirely — offer to create via `mint init` or manual

5. **Definition of Done** — all keys present?
   - `gatesPassing`, `specReviewPassed`, `stage2ReviewsPassed`, `docCheckPassed`

6. **Learning files** — do issues.md, wins.md, instincts.md, patterns.md exist?

7. **Convention docs** — are convention doc paths pointing to real files?
   - Check each path in `conventions.docs` and `business.docs`

### Phase 3: Project-Specific Recommendations

Based on what you found:

1. **Frontend projects** (has .tsx/.jsx/.vue/.svelte files):
   - Enable design intelligence if not already
   - Enable browser support for visual testing
   - Enable performance reviewer
   - Suggest design profile build: `/design:profile build`

2. **Backend projects** (API, server, database):
   - Ensure security reviewer is enabled
   - Suggest business reviewer with API docs
   - Check if test coverage gate is configured

3. **Monorepo/workspace** (sibling git repos found):
   - Suggest workspace config
   - Show how to configure repo dependencies

4. **Existing project with docs** (README, CONTRIBUTING, docs/ exist):
   - Run /doc-setup to build comprehensive manifest
   - Ensure documenter will track the right things

### Phase 4: Apply Fixes

For each issue found, offer to fix it:

```
mint optimize report
---

Infrastructure:
  [pass] mint v0.6.4 (latest)
  [pass] Claude hooks installed
  [warn] CLAUDE.md mint section outdated (v0 -> v1)
  [warn] .gitignore missing .mint/.session-state.json

Configuration:
  [pass] Gates: lint [pass] types [pass] tests [pass]
  [warn] Reviewer 'conventions' has no model -- assigning haiku
  [warn] Reviewer 'performance' disabled -- recommended for frontend
  [warn] context.enabled is false -- recommended for this codebase size
  [pass] design.enabled with accessibility, consistency, performance checks
  [warn] Doc-manifest: 4 docs but 0 tracked sections -- /doc-setup recommended

Learning:
  [pass] issues.md present (3 entries)
  [pass] wins.md present (1 entry)
  [warn] instincts.md missing
  [warn] patterns.md missing

Recommendations:
  1. Enable context mode for sandboxed execution (large codebase)
  2. Run /doc-setup to populate doc-manifest sections
  3. Enable performance reviewer for frontend code
  4. Run /design:profile build to learn your design DNA

Fix 6 issues automatically? (yes / pick individually / skip)
```

Then apply fixes using the same patterns as `mint doctor --fix` — write config, create files, etc.

### Phase 5: Verify

After applying fixes:
1. Run `mint doctor` again to verify
2. Show the before/after comparison
3. Suggest next steps ("Now describe a feature and mint will handle it with full capability")

## Rules

- **Don't force features.** Explain what each feature does and let the user decide.
- **Explain WHY.** Don't just say "enable X" — say "enable X because your project has 200+ files and context mode will save ~97% tokens on test output."
- **Be conversational.** This isn't a script — adapt to what you find.
- **Fix safely.** Only modify .mint/ files and .gitignore — never modify source code.
- **Check existing state first.** Don't suggest enabling something that's already enabled.
- **Run doctor at the end.** Prove the fixes worked.
