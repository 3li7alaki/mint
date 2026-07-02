<div align="center">

# mint

**The completion floor any agent can stand on.**

mint is a small, deterministic engine that guarantees a unit of work is *actually done
properly* — the same way no matter which agent runs it, how many run, or whether a human is
watching. You own the shape of the work. mint owns the floor you can't fall through.

[![CI](https://github.com/3li7alaki/mint/actions/workflows/ci.yml/badge.svg)](https://github.com/3li7alaki/mint/actions/workflows/ci.yml)
[![Stars](https://img.shields.io/github/stars/3li7alaki/mint?style=flat)](https://github.com/3li7alaki/mint/stargazers)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
![single Go binary](https://img.shields.io/badge/single-Go%20binary-00ADD8?logo=go&logoColor=white)
![zero deps](https://img.shields.io/badge/runtime%20deps-zero-success)

</div>

---

## The problem

Coding agents self-certify. They say "done," and the loop believes them. As autonomy rises —
more agents, longer runs, fewer humans in the inner loop — that self-certification is exactly
what breaks: a green test suite the agent quietly weakened, a change that drifted out of
scope, a "fix" that never met the requirement, a push to prod no one independently checked.

Every harness is a **driver**: it runs loops, spawns sub-agents, schedules work. What no
harness has is a **per-unit verification contract it cannot talk its way around.** That is the
scarce thing as autonomy scales, and it's the one thing mint does.

mint is not a harness with fewer features. **mint is the floor a harness stands on.**

## What mint is

An **engine**: a load-bearing guarantee, invariant to its caller.

- **Caller-invariant** — Claude Code, Codex, Cursor, a raw `while` loop, a human's agent all
  load mint identically and get the *same floor*. No caller is privileged; none can negotiate
  a weaker bar.
- **Scale-invariant** — one agent or forty, the per-unit floor is identical. A 40-wide wave is
  held to the same bar as a solo run. Scale changes throughput, never the standard.
- **Driver-invariant** — human-in-the-loop or fully autonomous overnight, same floor. The
  human leaving the loop doesn't lower the bar; it *raises* how load-bearing the engine is.

The seatbelt is not the car and not the driver. It's the safety contract that rides with
whoever drives. mint is that, for software work.

## How it works

You drive. mint doesn't own a pipeline, modes, or stages — **you** sequence the work. mint
gives you a small set of discipline primitives to call à la carte, and one choke point that
enforces the floor:

```
                  you shape the work ───────────────┐
                                                     ▼
   spec ──► implement ──► review (the lenses you pick) ──► mint done
   {goal,                                                      │
    scope,                          ┌─────────────────────────┘
    acceptance}                     ▼
                          the 7-clause floor, re-checked
                          against the ACTUAL diff — not
                          against the agent's "I'm done"
```

A typo runs straight to `done`. An auth change runs `review --security → review --adversarial
→ done`. You pick the shape; **`mint done` enforces the same floor no matter what shape you
chose.**

### The unit

One unit = one spec: `{ goal, scope, acceptance }`. Scope says what may and may not change
(`can-modify`). Acceptance is testable, in EARS form (`WHEN <trigger> THE <system> SHALL
<response>`). A commit, PR, or push is the *output* of a unit, not the unit — each inherits
the unit's `done` verdict.

### The floor — what `mint done` enforces

`mint done` re-checks the real produced diff and clears a unit only when all seven hold:

| # | Clause | Guarantee |
|---|--------|-----------|
| 1 | **Verifiable completion** | done is *proven* against the diff, never the agent's claim. Gates pass, an independent acceptance verdict is attached, and every declared review lens has a passing verdict. |
| 2 | **Maker ≠ checker** | the verdict's independence is verified by *provenance*. Normal work needs a fresh independent context; the safety carve-out needs a genuinely different engine than the maker. |
| 3 | **Safety carve-out** | security, trust-boundary, accessibility, and data-loss handling are floor — never minimized, never skipped. |
| 4 | **Anti-gaming** | a green is not trusted if the diff weakened, deleted, or skipped verifiers, assertions, coverage, or scoring. A tampered green is a FAIL. |
| 5 | **Scope respected** | every changed file is inside the unit's declared `can-modify`. |
| 6 | **Bounded + terminating** | the unit ends in one hard terminal state: `done-verified \| budget-exhausted \| stuck-escalated \| external-stop`. |
| 7 | **Every action gated** | commit, merge, push, deploy require the same full floor. No privileged path. |

**mint never calls a model.** It checks the deterministic clauses itself (zero-token,
portable) and *requires the driver to attach the semantic judgment* — "does this diff satisfy
acceptance" — then verifies that judgment **happened and was independent**, not its content.
Depth comes from the driver; integrity comes from mint.

### The floor never traps honest work

The single worst failure of a discipline engine is blocking an agent doing the *right* thing.
So every clause that can block ships with an in-band way forward — an independence-gated
acknowledge-with-reason override, or an escalate-to-caller exit. The floor is **strict against
gaming** (you can't wave your own violation through) and **never strict against honesty** (a
genuine refactor, an obsolete test removed, a legitimately-widened scope always has a clean,
recorded path). There is never a state where a correct agent must patch code to proceed.

### Failure re-mints the spec, not the code

When `mint done` FAILS, it doesn't just say no — it appends a **note** keyed to the spec
(`done-fail-<slug>-<id>`) naming which clause failed and why. Retries accumulate under that one
topic. The philosophy: a failure is evidence the *spec* was underspecified, not that the code
needs patching. Work is disposable; the spec is the asset. mint never blocks a patch — it makes
the spec's inadequacy visible and compounding, so re-minting it sharper is the cheaper path.

### Gates and reviews are yours to declare

mint guesses nothing. The **driver** declares what runs — `mint session set-gates 'go test ./...,
golangci-lint run'`, `mint session set-reviews security,quality` — cached per session, inherited by
specs, overridable per spec. No hardcoded stack tables (the thing that rots). No declaration
on a code diff → mint reports "no gates declared" and escalates, rather than inventing a
command. Review lenses are a curated set selected by flag:

```bash
mint review --list             # the menu
mint review --security         # load one lens
mint review --focus "race on the retry path"   # an ad-hoc lens
```

The lens prompts ship embedded in the binary, so they version with the tool. `review` runs no
model — it emits the prompt; the teeth are in the floor (clause 1 refuses `done` until every
declared review has a passing verdict attached).

## Install

mint is a single static Go binary with zero runtime dependencies. One line installs it to
your PATH — no root, no toolchain:

```bash
curl -fsSL https://raw.githubusercontent.com/3li7alaki/mint/main/install.sh | sh
```

**Updating is the same command** — re-run it and it replaces the binary in place with the
latest build (mint ships as a rolling `latest` release; there are no version numbers). Remove
it with `… | sh -s -- --uninstall`.

Prefer to build from source? `git clone`, then `cd cli && go build -o mint ./cmd/mint`.

Then point your agent at the contract once — add `@AGENTS.md` (or a copy of
[`AGENTS.md`](AGENTS.md)) to your global agent config. Any agent that reads it can wield mint;
there is no per-project setup.

## Using it

[`AGENTS.md`](AGENTS.md) is the contract every driver loads — the discipline, not the manual.
For exact commands and flags, the CLI is the source of truth:

```bash
mint session new                 # start a unit-of-work session
mint spec new                    # define a spec: goal, scope, acceptance
mint review --<lens>             # load a review lens the diff needs
mint verify                      # run the declared gates
mint done <slug> <id>            # route the done-decision through the floor
```

mint is a **floor, not a driver** — it does not run your loop, dispatch to engines, or hold a
schedule. You (or your harness) sequence the work and call `mint done` at the terminal edge.

Run `mint <command> --help` for any of them. The contract lists no menus and no version-pinned
flags on purpose — a menu in prose rots; a pointer to `--help` does not.

## What mint refuses to be

The engine is defined by its refusals as much as its features. mint does **not**: own a
pipeline or fixed sequence; run a learning loop; detect your stack or guess commands; have
per-project setup; privilege any caller; do domain work (design/browser/code-graph are
optional packs, not core); ship a plugin/marketplace; run a health-check subsystem; or call a
model itself. See [`docs/principles.md`](docs/principles.md) for the full, checkable list — if
a file violates it, it's a deletion candidate by definition.

## Documentation

- [`AGENTS.md`](AGENTS.md) — the contract a driver loads.
- [`docs/principles.md`](docs/principles.md) — the checkable definition: the three invariances,
  what mint is, what it refuses, how it stays slim.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — how to work on mint.

## License

MIT.
