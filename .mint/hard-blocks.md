# Hard Blocks — What Agents Can NEVER Do

## Universal
- NEVER `git push` — human reviews and pushes manually
- NEVER modify files outside declared task scope
- NEVER fix bad output directly — reset and fix the spec
- NEVER continue after 2 failures on the same spec

## Context Protection
- NEVER read large files in the main orchestrator context
- Subagents return summaries only

## Project-Specific
- NEVER break backwards compatibility with v1 skill format
- NEVER add runtime dependencies without making them optional — core must work with markdown files alone. Structured storage (SQLite, vector DB) is opt-in, not required.
- NEVER merge plugin system code without a working reference plugin

## Release & CLI Integrity
- NEVER create a PR for a new feature or core feature without bumping version — use `./scripts/bump.sh minor` for features, `./scripts/bump.sh patch` for fixes. NEVER edit version strings manually across files.
- NEVER add a core feature (top-level config key like browser, context, design) without updating ALL CLI touchpoints: init.js (prompt + buildConfig + headless), doctor.js (health check), config.js (status display), update.js (NEW_CONFIG_KEYS + dep updater)
- NEVER add a core feature without updating SKILL.md (routing + execution flow + startup detection), docs/conventions.md (config schema), docs/architecture.md (feature section), and CLAUDE.md (key files table)
- NEVER ship CLI changes without running `bun test` — all 87+ tests must pass
