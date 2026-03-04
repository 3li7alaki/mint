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
- NEVER add runtime dependencies — mint is markdown files only
- NEVER merge plugin system code without a working reference plugin
