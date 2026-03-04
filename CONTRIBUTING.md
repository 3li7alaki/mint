# Contributing to mint

## Getting Started

1. Fork the repo
2. Clone your fork
3. Create a feature branch: `git checkout -b feat/your-feature`
4. Make your changes
5. Commit with the conventions below
6. Push and open a PR

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
│   └── business-reviewer.md
├── commands/                # User-invocable commands
│   ├── init.md
│   ├── verify.md
│   └── help.md
├── templates/               # Templates and schemas
│   ├── spec.xml
│   └── plugin-manifest.json
├── plugins/                 # Reference and community plugins
├── docs/                    # Documentation
│   ├── architecture.md
│   └── conventions.md
└── .mint/                   # Project config (partially committed)
    ├── config.json          # Committed — shared project settings
    ├── hard-blocks.md       # Committed — rules everyone follows
    ├── issues.md            # Committed — institutional knowledge
    ├── tasks/               # Gitignored — in-progress specs
    ├── research/            # Gitignored — local research reports
    └── plugins/             # Gitignored — installed plugins
```

## Commit Conventions

Format: `type(scope): description`

**Types:**
- `feat` — new agent, command, template, or capability
- `fix` — bug fix in an agent prompt or orchestrator logic
- `docs` — documentation only
- `refactor` — restructuring without changing behavior
- `chore` — maintenance (gitignore, CI, etc.)

**Scope:** use the file or component name. For spec-driven work, use `mint-NNN`.

**Examples:**
```
feat(planner): add dependency resolution to decompose step
fix(security-auditor): clarify input validation checks
docs(readme): add plugin installation guide
refactor(skill): simplify auto-routing decision logic
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

## Writing Plugins

Plugins live in their own directories with a `manifest.json`.

**Conventions:**
- Follow `templates/plugin-manifest.json` schema exactly
- Plugin agents follow the same conventions as core agents
- Include a README.md in the plugin directory
- Plugin names: `mint-{name}` (e.g., `mint-nuxt`, `mint-linear`)
- Don't duplicate core agent checks — extend, don't repeat

## Writing XML Specs

Specs follow `templates/spec.xml`. Key rules:
- `<scope>` is strict — agents cannot modify files outside it
- `<no-mocks>` explains what to use instead of mocks
- `<anti-patterns>` lists what NOT to do
- `<pitfalls>` includes lessons from `.mint/issues.md`
- `<commit>` defines the exact commit message

## PRs

- One logical change per PR
- Reference the spec ID if applicable (e.g., "Implements mint-004")
- Keep PRs small — if it touches more than 5 files, consider splitting
- All markdown must be well-formed (no broken links, valid JSON in code blocks)

## What Not to Do

- Don't add runtime dependencies — mint is markdown files only
- Don't add npm/package.json — mint has no build step
- Don't break backwards compatibility with existing SKILL.md format
- Don't add features without updating documentation
