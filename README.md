# mint

Disciplined agentic development for Claude Code. Fresh context per task, clean orchestration, zero slop.

**Core philosophy:** Slop is an engineering problem, not an LLM problem. If an agent produces bad code, fix the environment — never patch the output.

## Install

```bash
npx skills add 3li7alaki/mint
```

Or manually: clone this repo into `.claude/commands/` in your project.

## How It Works

You describe what you want. mint auto-detects the right approach:

| What you say | What mint does |
|---|---|
| Small fix, typo, config tweak | **Quick** — fixes directly, gates enforced |
| Feature, component, API route | **Plan** — decomposes into XML specs, executes atomically |
| Multiple features, batch of work | **Ship** — interviews you, plans phases, executes all |
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
    Quality + Security + Conventions + Tests + Performance
        │
  Atomic commit per spec
        │
  Documentation auto-updated
        │
  You review the final result
```

## Setup

Run `mint init` in your project (or just start working — mint will prompt you to init).

This creates:

```
.mint/
├── config.json       — gate settings, reviewers, stack config
├── hard-blocks.md    — what agents can never do
├── issues.md         — agent learnings and blockers log
└── tasks/            — XML spec files from plan mode
```

## XML Specs

The core artifact. Each task gets a structured spec:

```xml
<task>
  <id>001</id>
  <title>Add password validation</title>
  <goal>Login form validates password before submission</goal>
  <estimate>small</estimate>
  <scope>
    <can-modify>src/components/LoginForm.vue, src/utils/validation.ts</can-modify>
    <cannot-modify>src/api/*, src/store/*</cannot-modify>
  </scope>
  <steps>...</steps>
  <acceptance>...</acceptance>
  <pitfalls>...</pitfalls>
  <anti-patterns>...</anti-patterns>
  <no-mocks>Use real zod validation</no-mocks>
  <commit>feat(mint-001): add password validation</commit>
  <gates>lint, types, tests</gates>
</task>
```

See `templates/spec.xml` for the full format.

## Review Pipeline

Every spec goes through:

1. **Spec Review** (gate) — does the implementation match the spec?
2. **Parallel Audit** — 5 reviewers run simultaneously:
   - **Quality** — patterns, types, readability, over-engineering
   - **Security** — injection, XSS, auth, secrets
   - **Conventions** — naming, file structure, imports (reads your convention docs)
   - **Tests** — mock audit, assertion quality, edge cases
   - **Business** — requirements alignment, domain logic correctness (reads your business docs)
   - **Performance** — re-renders, N+1, bundle impact (opt-in)

Issues are categorized: BLOCKING (must fix), WARNING (should fix), INFO (logged).

## Issue Log

`.mint/issues.md` tracks every blocker and gotcha. The planner reads past issues before creating new specs — turning mistakes into prevention.

## Documenters

Auto-update project docs when code changes:

```json
{
  "documenters": [
    {
      "path": "CHANGELOG.md",
      "trigger": "on-task-complete",
      "mode": "update"
    },
    {
      "path": "weekly-reports/",
      "trigger": "on-session-end",
      "mode": "template",
      "template": "# {weekday} {date}\n\n## Done\n- "
    }
  ]
}
```

## Golden Rules

1. **Never fix bad output.** Reset and fix the spec — not the code.
2. **One agent, one task, one prompt.** Focused agents are correct agents.
3. **Gates before everything.** Lint + types + tests pass 100% before any commit.
4. **Never mock what you can use for real.** Mocks hide failures.
5. **Precise specs, zero inference.** Agents don't guess.
6. **Escalate, don't improvise.** If stuck, stop and ask — never silently work around.

## Agents

| Agent | Role |
|---|---|
| `planner` | Decomposes features into specs, executes, commits |
| `researcher` | Investigates problems, saves structured reports |
| `shipper` | Bulk executes multi-feature ship plans |
| `verifier` | Runs all gates and audits on demand |
| `spec-reviewer` | Stage 1 gate — spec compliance |
| `quality-reviewer` | Stage 2 — code quality |
| `security-auditor` | Stage 2 — security vulnerabilities |
| `conventions-enforcer` | Stage 2 — project conventions (configurable doc paths) |
| `test-auditor` | Stage 2 — test quality and mock discipline |
| `business-reviewer` | Stage 2 — business logic and requirements alignment |
| `performance-reviewer` | Stage 2 — performance (opt-in) |
| `documenter` | Auto-updates project documentation |

## Plugins

Plugins extend mint with stack-specific, PM, design, or memory capabilities. A plugin is a directory with a `manifest.json` and optional agents/commands.

**Install a plugin:**

```bash
# Clone into .mint/plugins/
git clone https://github.com/user/mint-nuxt .mint/plugins/mint-nuxt
```

Then add it to `.mint/config.json`:

```json
{
  "plugins": [".mint/plugins/mint-nuxt"]
}
```

**Create a plugin:**

```
my-plugin/
├── manifest.json          # Required — name, type, agents, hooks
├── agents/                # Plugin agent prompts
│   └── my-reviewer.md
├── commands/              # Plugin commands (optional)
└── README.md
```

**Plugin types:** `stack` (framework conventions), `pm` (project management), `design` (design tools), `memory` (knowledge persistence).

**Hook points:** `pre-plan`, `post-plan`, `pre-review`, `post-commit`, `on-init`.

See [`plugins/mint-nuxt/`](plugins/mint-nuxt/) for a working reference plugin and `templates/plugin-manifest.json` for the manifest schema.

## Built With mint

Once the core was ready, mint became the first project developed using mint itself — specs, reviews, the full pipeline. Like Git, it manages its own development.

## License

MIT
