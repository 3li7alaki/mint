```
 ███╗   ███╗██╗███╗   ██╗████████╗
 ████╗ ████║██║████╗  ██║╚══██╔══╝
 ██╔████╔██║██║██╔██╗ ██║   ██║
 ██║╚██╔╝██║██║██║╚██╗██║   ██║
 ██║ ╚═╝ ██║██║██║ ╚████║   ██║
 ╚═╝     ╚═╝╚═╝╚═╝  ╚═══╝   ╚═╝
```

### Disciplined agentic development for Claude Code

> v0.4.0 — Fresh context per task. Clean orchestration. Zero slop.

**Core philosophy:** Slop is an engineering problem, not an LLM problem. If an agent produces bad code, fix the environment — never patch the output.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/3li7alaki/mint/main/install.sh | bash
```

This installs the `mint` CLI globally and (if Claude Code is installed) the Claude plugin for auto-routing.

### CLI

```bash
mint init          # interactive setup — detects your stack, asks 5 questions
mint init --yes    # headless — auto-detect everything, zero prompts
mint config        # view current config
mint config set    # edit config (dot notation)
mint doctor        # health check
mint update        # update to latest
```

### Project Setup

Run `mint init` in your project:

```
.mint/
├── config.json       — gates, reviewers, browser, plugins
├── hard-blocks.md    — what agents can never do
├── issues.md         — failure log — what went wrong and why
└── wins.md           — success log — what worked and why
```

## How It Works

You describe what you want. mint auto-detects the right approach:

| What you say | What mint does |
|---|---|
| Small fix, typo, config tweak | **Quick** — fixes directly, gates enforced |
| Feature, component, API route | **Plan** — decomposes into XML specs, executes atomically |
| Multiple features, batch of work | **Ship** — interviews you, plans phases, executes all |
| "Browse to", "scrape", "debug in browser" | **Browse** — PinchTab-powered browser automation |
| "How should I...", "Compare..." | **Research** — investigates, saves structured report |
| "Check quality", "Audit" | **Verify** — runs all gates and audits |

No commands to memorize. Just describe what you want to build.

## The Pipeline

```
You describe a feature
        │
  mint decomposes into XML specs
        │
  Fresh subagent executes each spec
  (reads existing code, matches patterns)
        │
  Gates run: lint → types → tests
        │
  Stage 1: Spec reviewer (gate)
        │
  Stage 2 (parallel):
    Quality + Security + Conventions + Tests + Business + Performance
        │
  Atomic commit per spec
        │
  You review the final result
```

## Browser Support

Built-in browser automation powered by [PinchTab](https://github.com/pinchtab/pinchtab). Not a plugin — a core feature. Agents can navigate pages, fill forms, scrape data, take screenshots, and debug live apps.

**How it works:** PinchTab runs a lightweight Go binary that controls Chrome via an HTTP API. mint agents talk to it with curl — no Puppeteer, no Selenium, no heavy dependencies. The accessibility tree gives agents a compact page representation (~800 tokens vs 10k+ for raw DOM).

```bash
# Install PinchTab (mint init offers to do this)
curl -fsSL https://pinchtab.com/install.sh | sh

# Start it
pinchtab &
```

**What agents can do:**
- Navigate to URLs and interact with elements (click, type, select)
- Extract structured data from pages
- Debug live apps — check console errors, DOM state, localStorage
- Verify UI changes after implementation
- Capture screenshots for review

**Commands:**
- `/browse <url> [task]` — navigate and interact
- `/screenshot [url]` — capture page state
- `/scrape <url> [what]` — extract structured data

**Token-efficient:** Agents use the cheapest PinchTab endpoint per task — `/text` for content (~800 tokens), filtered snapshots for interactions (~3600), diffs for changes. Full snapshots only when needed.

Enable/disable in config:

```json
{
  "browser": {
    "enabled": true,
    "devServer": "http://localhost:3000"
  }
}
```

## TDD Support

Test-first development built into the pipeline. Toggle via config or per-spec:

1. **RED** — write tests first, verify they fail
2. **GREEN** — implement minimal code to pass
3. **REFACTOR** — clean up while tests stay green
4. **COVERAGE** — verify coverage meets threshold
5. **DE-SLOPPIFY** — optional cleanup pass in fresh context

Edge cases (null, empty, boundary, error paths) are auto-injected. Coverage gating blocks commits below threshold.

```json
{
  "tdd": { "default": true, "coverageThreshold": 80 }
}
```

## Review Pipeline

Every spec goes through multi-stage review:

1. **Spec Review** (gate) — does the implementation match the spec?
2. **Parallel Audit** — reviewers run simultaneously:
   - **Quality** — patterns, types, readability, over-engineering
   - **Security** — injection, XSS, auth, secrets
   - **Conventions** — naming, file structure, imports
   - **Tests** — mock audit, assertion quality, edge cases
   - **Business** — requirements alignment, domain logic
   - **Performance** — re-renders, N+1, bundle impact (opt-in)

Issues are categorized: BLOCKING (must fix), WARNING (should fix), INFO (logged). Each reviewer can use a different Claude model.

## Learning

mint learns your project's conventions automatically. Three mechanisms:

- **Instincts** (`.mint/instincts.md`) — a PostToolUse hook observes every file edit and extracts patterns: import style, naming conventions, test framework, component patterns. Confidence grows as the same pattern appears across different files. The planner reads this before writing specs so new code matches existing conventions without scanning every file. High-confidence patterns (>= 3) are treated as project conventions.

- **Issues** (`.mint/issues.md`) — every blocker, root cause, and resolution is logged. The planner reads this to avoid repeating mistakes.

- **Wins** (`.mint/wins.md`) — successful patterns and why they worked. The planner reads this to replicate what works.

All three are committed to git — they're shared team knowledge, not throwaway state.

## Plugins

Plugins extend mint with stack-specific or integration capabilities.

| Plugin | What it does |
|--------|-------------|
| `mint-nuxt` | Nuxt file structure, auto-imports, server patterns |
| `mint-e2e` | E2E testing with Playwright |
| `mint-linear` | Linear ticket context and status sync |
| `mint-figma` | Design tokens and specs from Figma |
| `mint-shadcn` | shadcn/ui component conventions |
| `mint-ssh` | SSH connections and remote commands |
| `mint-gws` | Google Workspace — Sheets, Gmail, Calendar |
| `mint-ui-ux` | RTL, i18n, accessibility standards |

> **Note:** Plugins are community-extensible and may not cover every edge case. PRs and issues welcome — if something doesn't work right, fix it and contribute back.

Enable plugins:

```json
{
  "plugins": ["plugins/mint-nuxt", "plugins/mint-linear"]
}
```

## Configuration

Key config in `.mint/config.json`:

| Key | Default | Description |
|-----|---------|-------------|
| `stack` | auto-detected | Framework (nuxt, react, vue, etc.) |
| `packageManager` | auto-detected | npm, pnpm, yarn, bun |
| `gates` | `{}` | lint/types/tests/coverage commands |
| `autoCommit` | `true` | Commit after passing gates |
| `tdd.default` | `false` | TDD-first by default |
| `browser.enabled` | `true` | Browser automation via PinchTab |
| `reviewers` | smart defaults | Which reviewers run and their models |
| `isolation` | `none` | Work isolation: none, branch, or worktree |
| `plugins` | `[]` | Plugin paths |

## Golden Rules

1. **Never fix bad output.** Reset and fix the spec — not the code.
2. **One agent, one task, one prompt.** Focused agents are correct agents.
3. **Gates before everything.** Lint + types + tests pass 100% before any commit.
4. **Never mock what you can use for real.** Mocks hide failures.
5. **Precise specs, zero inference.** Agents don't guess.
6. **Escalate, don't improvise.** If stuck, stop and ask — never silently work around.

## Contributing

Plugins aren't perfect — they're starting points. If you hit an edge case or something doesn't work:

1. Open an issue describing what went wrong
2. PRs welcome — especially for plugins and stack-specific conventions
3. Follow the existing patterns: one agent per file, role → inputs → process → outputs → rules

See [Plugin Guide](docs/plugin-guide.md) for creating custom plugins.

## Documentation

| Doc | What it covers |
|-----|---------------|
| [Plugin Guide](docs/plugin-guide.md) | Creating custom plugins |
| [Conventions](docs/conventions.md) | File formats, naming, config schema |
| [Architecture](docs/architecture.md) | System design and philosophy |
| [Autonomous Loops](docs/autonomous-loops.md) | CI/CD and scripted workflows |

## License

MIT
