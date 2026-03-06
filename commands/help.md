---
description: "Show the mint guide — how it works, what it does, golden rules"
---

# mint help

Print this guide when the user asks for help.

## Output exactly this:

---

# mint — disciplined agentic development

> Fresh context per task. Clean orchestration. Zero slop.

---

## How it works

You describe what you want. mint auto-detects the right approach:

| What you say | What mint does |
|---|---|
| Small fix, typo, config tweak | **Quick mode** — fixes directly, gates enforced |
| Feature, component, API route | **Plan mode** — decomposes into specs, executes each atomically |
| Multiple features, batch of work | **Ship mode** — interviews you, plans phases, executes all |
| "How should I...", "Compare..." | **Research mode** — investigates, saves report |
| "Check quality", "Audit" | **Verify mode** — runs all gates, mock audit, hard block scan |

No commands to remember. Just describe what you want.

---

## The pipeline

```
You describe a feature
        |
  mint decomposes into XML specs
        |
  Fresh subagent executes each spec
  (reads existing code, matches patterns)
        |
  Gates run: lint → types → tests
        |
  Stage 1: Spec reviewer (gate)
        |
  Stage 2 (parallel):
    Quality + Security + Conventions + Tests + Performance
        |
  Atomic commit per spec
        |
  Documentation auto-updated
        |
  You review the final result
```

---

## Golden rules

1. **Never fix bad output.** Reset and fix the spec — not the code.
2. **One agent, one task, one prompt.** Focused agents are correct agents.
3. **Gates before everything.** Lint + types + tests pass 100% before any commit.
4. **Never mock what you can use for real.** Mocks hide failures.
5. **Precise specs, zero inference.** Agents don't guess.
6. **Escalate, don't improvise.** If stuck, stop and ask — never silently work around.

---

## Project files

After `mint init`, your project has:

```
.mint/
├── config.json       — gate settings, reviewers, stack config
├── hard-blocks.md    — what agents can never do
├── issues.md         — agent learnings and blockers log
└── tasks/            — XML spec files from plan mode
```

---

## Commands

| Command | What it does |
|---|---|
| `/mint:help` | Show this guide |
| `/mint:init` | Set up mint in a project |
| `/mint:verify` | Run all quality gates |
| `/mint:status` | Check running tasks and progress |
| `/mint:stop [reason]` | Interrupt running agents |

---

## When something goes wrong

| Symptom | Fix |
|---|---|
| Agent went off-scope | Tighten `<can-modify>` in spec |
| Wrong assumptions | Add context/snippets to spec |
| Gates pass but feature is wrong | Rewrite `<acceptance>` criteria |
| Excessive mocks | Add `<no-mocks>` to spec |
| Same failure on retry | Spec is the problem — rewrite it |
| Agent going wrong direction | `/mint:stop` then resume with changes |

**Rule:** if the same spec fails twice, stop and rewrite it.
