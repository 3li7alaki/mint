```
 ███╗   ███╗██╗███╗   ██╗████████╗
 ████╗ ████║██║████╗  ██║╚══██╔══╝
 ██╔████╔██║██║██╔██╗ ██║   ██║
 ██║╚██╔╝██║██║██║╚██╗██║   ██║
 ██║ ╚═╝ ██║██║██║ ╚████║   ██║
 ╚═╝     ╚═╝╚═╝╚═╝  ╚═══╝   ╚═╝
```

### Disciplined agentic development for Claude Code

> v0.6.0 — Fresh context per task. Clean orchestration. Zero slop.

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
mint update        # update to latest (core + dependencies)
```

### Project Setup

Run `mint init` in your project:

```
.mint/
├── config.json       — gates, reviewers, browser, design, plugins
├── hard-blocks.md    — what agents can never do
├── issues.md         — failure log — what went wrong and why
├── wins.md           — success log — what worked and why
├── instincts.md      — auto-learned conventions
└── patterns.md       — graduated recurring patterns
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
| "Design review", "Design profile" | **Design** — design intelligence commands |

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
    Quality + Security + Conventions + Tests + Business + Performance + Design
        │
  Atomic commit per spec
        │
  You review the final result
```

## Ecosystem & Integrations

mint integrates with best-in-class external tools. Each is optional and toggleable — mint works without any of them, but they make it significantly more capable.

| Tool | What it does for mint | Install |
|------|----------------------|---------|
| [PinchTab](https://github.com/pinchtab/pinchtab) | Browser automation — navigate, scrape, debug, screenshot via lightweight Go binary + Chrome. Agents talk to it via HTTP API, get compact accessibility tree (~800 tokens vs 10k+ raw DOM). | `curl -fsSL https://pinchtab.com/install.sh \| sh` |
| [context-mode](https://github.com/mksglu/context-mode) | Sandboxed execution + FTS5 search + session continuity. Keeps verbose tool output out of context window. ~97% token savings on test output, ~99% on URL fetching. | `claude mcp add context-mode -- npx -y context-mode` |
| [Impeccable](https://impeccable.style) | Design steering commands (`/polish`, `/audit`, `/critique`, `/bolder`, etc.) with curated anti-patterns and design vocabulary. By Paul Bakaus, Apache 2.0. | `npx skills add pbakaus/impeccable` |

`mint init` offers to install each one. `mint update` keeps them current. `mint doctor` checks their health.

## Core Features

### Browser Support

Built-in browser automation powered by [PinchTab](https://github.com/pinchtab/pinchtab). Not a plugin — a core feature. Agents can navigate pages, fill forms, scrape data, take screenshots, and debug live apps.

**How it works:** PinchTab runs a lightweight Go binary that controls Chrome via an HTTP API. mint agents talk to it with curl — no Puppeteer, no Selenium, no heavy dependencies. The accessibility tree gives agents a compact page representation (~800 tokens vs 10k+ for raw DOM).

```bash
# Install PinchTab (mint init offers to do this)
curl -fsSL https://pinchtab.com/install.sh | sh

# WSL2: also install Linux-side Chromium
sudo apt install -y chromium-browser

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

### Context Mode

Optional integration with [context-mode](https://github.com/mksglu/context-mode) for sandboxed execution, session continuity, and FTS5 full-text search. Not a plugin -- a core feature. Keeps raw tool output out of the context window so agents stay focused.

**What it does:**
- **Sandboxed execution** -- test runners, build tools, and lint output stay out of context via `ctx_execute`. Only errors/failures return.
- **FTS5 search** -- index files, URLs, and command output into a full-text search database. Query with `ctx_search` instead of loading raw content.
- **Session continuity** -- file operations, task state, and decisions survive context compaction automatically.
- **Intent-driven filtering** -- add an `intent` parameter to large outputs and only relevant sections return.

**Setup:**
```bash
# Install via MCP server (mint init offers to do this)
claude mcp add context-mode -- npx -y context-mode
```

**How agents use it:** When `context.enabled` is `true`, every agent activates its Context Mode section automatically. Test runs use `ctx_execute` with `intent: "errors"`, file analysis uses `ctx_execute_file`, URL fetching uses `ctx_fetch_and_index` + `ctx_search`. Standard tools are the fallback if context-mode is unavailable.

**Token savings:** ~97% on test output, ~99% on URL fetching, ~90% on file analysis.

Enable/disable in config:
```json
{
  "context": {
    "enabled": true,
    "autoRoute": true,
    "sandbox": { "timeout": 30000 },
    "session": { "enabled": true }
  }
}
```

### Design Intelligence

Automatic UI/UX awareness powered by vendored [Impeccable](https://impeccable.style) reference knowledge (Apache 2.0) merged with project-specific design learning. Not a plugin — a core feature. When enabled, every UI task automatically gets design context injected into planning and design quality checked during review.

**What it does:**
- **Pre-plan hook** — loads your project's design profile, design notes, and relevant reference knowledge (typography, color theory, spatial design, motion, interaction patterns, responsive design, UX writing). Injects structured design context into the planner.
- **Pre-review hook** — stage 2 auditor that checks for AI slop (purple gradients, glassmorphism, generic card grids), RTL violations, i18n compliance, WCAG 2.1 AA accessibility, design system consistency, and performance.
- **Profile learning** — analyzes existing UI code to extract colors, typography, spacing, and component patterns into `.mint/design-profile.json`. Learns your project's visual DNA.
- **Design notes** — persistent rules ("never use red for success") and preferences that override all other design guidance.
- **AI slop test** — "If you showed this interface to someone and said 'AI made this,' would they believe you immediately? If yes, that's the problem."

**Commands:**
- `/design search|system|palette|typography|inspiration` — design intelligence queries
- `/design:profile build|view|update` — manage project design profile
- `/design:notes add|list|remove` — manage design rules and preferences
- `/design:review [target] [--fix]` — standalone design review
- `/design:tokens export|sync|validate` — design token management
- `/design:teach` — one-time project design context setup
- `/design:steer <direction>` — 16 steering commands (polish, critique, audit, bolder, quieter, distill, colorize, animate, delight, clarify, harden, adapt, normalize, extract, optimize, onboard)

**Optional:** Install [Impeccable](https://impeccable.style) (`npx skills add pbakaus/impeccable`) for editor-level steering commands. mint's design features work without it — reference knowledge is vendored.

Enable/disable in config:
```json
{
  "design": {
    "enabled": true,
    "review": {
      "accessibility": true,
      "consistency": true,
      "performance": true,
      "rtl": false,
      "i18n": false
    }
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
   - **Design** — AI slop, RTL, i18n, accessibility, design consistency (if `design.enabled`)

Issues are categorized: BLOCKING (must fix), WARNING (should fix), INFO (logged). Each reviewer can use a different Claude model.

## Learning

mint learns your project's conventions automatically. Three mechanisms:

- **Instincts** (`.mint/instincts.md`) — a PostToolUse hook observes every file edit and extracts patterns: import style, naming conventions, test framework, component patterns. Confidence grows as the same pattern appears across different files. The planner reads this before writing specs so new code matches existing conventions without scanning every file. High-confidence patterns (>= 3) are treated as project conventions.

- **Issues** (`.mint/issues.md`) — every blocker, root cause, and resolution is logged. The planner reads this to avoid repeating mistakes.

- **Wins** (`.mint/wins.md`) — successful patterns and why they worked. The planner reads this to replicate what works.

Patterns graduate automatically: instincts → patterns → permanent conventions. All three are committed to git — they're shared team knowledge, not throwaway state.

## Plugins

Plugins extend mint with stack-specific or integration capabilities.

| Plugin | What it does |
|--------|-------------|
| `mint-nuxt` | Nuxt file structure, auto-imports, server patterns |
| `mint-e2e` | E2E testing with Playwright |
| `mint-linear` | Linear ticket context and status sync |
| `mint-figma` | Design tokens and specs from Figma |
| `mint-ssh` | SSH connections and remote commands |
| `mint-gws` | Google Workspace — Sheets, Gmail, Calendar |

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
| `context.enabled` | `false` | Context Mode via context-mode |
| `design.enabled` | `true` | Design intelligence — profiling, anti-patterns, RTL/i18n |
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
