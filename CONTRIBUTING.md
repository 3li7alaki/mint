# Contributing to mint

## Getting Started

1. Fork the repo
2. Clone your fork
3. `bun install`
4. Create a feature branch: `git checkout -b feat/your-feature`
5. Make your changes
6. Run tests: `bun test`
7. Commit with the conventions below
8. Push and open a PR

## Project Structure

```
mint/
├── SKILL.md                 # Orchestrator brain — auto-routing, execution flows
├── agents/                  # Agent prompts (one per agent)
│   ├── planner.md
│   ├── researcher.md
│   ├── shipper.md
│   ├── verifier.md
│   ├── documenter.md
│   ├── spec-reviewer.md
│   ├── quality-reviewer.md
│   ├── security-auditor.md
│   ├── conventions-enforcer.md
│   ├── test-auditor.md
│   ├── performance-reviewer.md
│   ├── business-reviewer.md
│   ├── browser-runner.md      # Core — PinchTab browser automation
│   ├── browser-reviewer.md    # Core — UI verification
│   ├── browser-context.md     # Core — pre-plan page state
│   ├── browser-debugger.md    # Core — live app debugging
│   ├── browser-setup.md       # Core — PinchTab install/config
│   ├── design-context.md      # Core — pre-plan design intelligence
│   ├── design-reviewer.md     # Core — design quality auditor
│   ├── design-profile.md      # Core — project visual DNA builder
│   └── design-setup.md        # Core — Impeccable install/config
├── commands/                # User-invocable commands
│   ├── init.md
│   ├── verify.md
│   ├── help.md
│   ├── browse.md
│   ├── screenshot.md
│   ├── scrape.md
│   ├── design.md              # /design — search, system, palette, typography
│   ├── design-profile.md      # /design:profile — build/view/update
│   ├── design-notes.md        # /design:notes — manage rules/prefs
│   ├── design-review.md       # /design:review — standalone design review
│   ├── design-tokens.md       # /design:tokens — export/sync/validate
│   ├── design-teach.md        # /design:teach — project design setup
│   ├── design-steer.md        # /design:steer — 16 steering directions
│   ├── doc-setup.md           # /doc-setup — build doc-manifest for existing projects
│   └── optimize.md            # /mint:optimize — full setup audit and optimization
├── cli/                     # mint CLI (bun + @clack/prompts)
│   ├── mint.js              # Entry point
│   ├── commands/            # init, config, doctor, update
│   └── lib/                 # detect.js — stack/PM/gates detection
├── tests/                   # Test suite — bun test
├── standards/               # Core feature reference knowledge
│   └── design/              # Design standards and reference docs
│       ├── reference/       # Vendored Impeccable docs (typography, color, etc.)
│       ├── rtl.md           # RTL logical properties
│       ├── i18n.md          # Internationalization standards
│       ├── anti-patterns.md # AI slop detection + design anti-patterns
│       └── design-direction.md # Aesthetic guidelines
├── references/              # PinchTab API docs, token strategy
├── templates/               # Templates and schemas
│   ├── spec.xml
│   ├── doc-manifest.json
│   └── plugin-manifest.json
├── hooks/                   # Claude Code hooks
│   ├── hooks.json
│   └── scripts/
├── plugins/                 # Community plugins
├── docs/                    # Documentation
│   ├── architecture.md
│   ├── conventions.md
│   └── plugin-guide.md
└── .mint/                   # Project config
    ├── config.json          # Committed — shared project settings
    ├── hard-blocks.md       # Committed — rules everyone follows
    ├── doc-manifest.json    — doc tracking manifest (committed)
    ├── issues.md            # Committed — failure log
    ├── wins.md              # Committed — success patterns
    ├── instincts.md         # Committed — auto-learned conventions
    ├── tasks/               # Gitignored — in-progress specs
    └── research/            # Gitignored — local research reports

# Global config (user-level defaults)
~/.mint/
└── config.json              # User preferences — reviewer models, autoCommit, TDD, isolation
                             # Seeds new projects via `mint init`, overridden by project config
```

## Commit Conventions

Format: `type(scope): description`

**Types:**
- `feat` — new agent, command, template, or capability
- `fix` — bug fix in an agent prompt or orchestrator logic
- `docs` — documentation only
- `refactor` — restructuring without changing behavior
- `chore` — maintenance (gitignore, CI, etc.)

**Scope:** use the file or component name (e.g., `planner`, `cli`, `config`).

**Examples:**
```
feat(planner): add dependency resolution to decompose step
fix(security-auditor): clarify input validation checks
docs(readme): add plugin installation guide
feat(cli): add mint config plugins interactive mode
```

## Writing Agents

Agents are markdown files in `agents/`. Each agent is a prompt that Claude receives as a fresh subagent.

**Conventions:**
- One agent, one responsibility
- Start with a clear role statement: "You are the mint X agent."
- Define inputs (what the agent receives) and outputs (what it returns)
- Include severity levels for reviewers: BLOCKING, WARNING, INFO
- Never reference other agents — agents don't know about each other
- Never assume tool availability — list required tools explicitly
- Keep prompts focused — a long prompt means the agent is doing too much

## Writing Commands

Commands are markdown files in `commands/`. They're invoked by the user or orchestrator.

**Conventions:**
- Use imperative mood for the command name
- Document what the command does, what it expects, and what it produces
- Commands delegate to agents — they don't contain implementation logic
- The /doc-setup command helps existing projects build their doc-manifest.

## Writing Plugins

Plugins live in their own directories with a `manifest.json`. Browser, context, and design are core features, not plugins.

**Conventions:**
- Follow `templates/plugin-manifest.json` schema exactly
- Plugin agents follow the same conventions as core agents
- Include a README.md in the plugin directory
- Plugin names: `mint-{name}` (e.g., `mint-nuxt`, `mint-linear`)
- Don't duplicate core agent checks — extend, don't repeat
- Bundled plugins live in `plugins/` and ship with mint
- Community plugins are standalone repos cloned into `.mint/plugins/`

## Tests

Run the full suite before submitting a PR:

```bash
bun test
```

Tests cover CLI commands (init, config, doctor), stack detection, and integration flows.

## PRs

- One logical change per PR
- Reference the spec ID if applicable (e.g., "Implements mint-004")
- Keep PRs small — if it touches more than 5 files, consider splitting
- All tests must pass

## What Not to Do

- Don't break backwards compatibility with existing SKILL.md format
- Don't add features without updating documentation
- Don't add features without tests
