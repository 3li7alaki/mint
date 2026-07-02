# mint — the principles (what it is, what it refuses, how it stays slim)

*The firm definition. Not vision, not history — the rules every file is held against. If a
file, command, or dependency violates these, it is a deletion candidate by definition,
regardless of whether it "still works" or tests pass. This is the thing that makes "why did
this survive?" impossible to ask twice.*

---

## 0. The three invariances (the definition of "engine")

mint is an *engine* because its guarantee does not change with context. If the guarantee bends
for any of these, it is a plugin, not an engine.

- **Caller-invariant** — Claude Code, Codex, an OpenCode/GLM worker, a raw `while` loop, a
  human's agent: every driver loads and calls mint identically and gets the *same floor*. No
  caller is privileged; none can negotiate a weaker bar. mint does not know or care who fired it.
- **Scale-invariant** — 1 agent or 40: the per-unit floor is identical. Parallelism is where
  discipline usually erodes (the race to merge, the skipped review under load). mint refuses
  that erosion. Scale changes throughput, never the standard.
- **Driver-invariant** — human-in-the-loop or fully autonomous overnight: the same floor. The
  human leaving the loop does not lower the bar — it *raises* how load-bearing mint is, because
  no human is there to catch what slips. The more autonomous the driver, the more mint matters.

**The one-line test every feature must pass:** *Does this hold regardless of who runs it, how
many run it, and whether a human is watching?* If no — it is shape, not floor, and it does not
belong in the core.

---

## 1. What mint IS (the only things it does)

mint is a small **Go** CLI + one contract file (`AGENTS.md`) + a fixed set of review-lens
prompts. It does exactly this and nothing else:

1. **Defines a unit of work** — one spec `{goal, scope, acceptance}`. (`spec`)
2. **Enforces the floor at `done`** — the 7 deterministic clauses, against the real diff.
   (`done`, `internal/floor`, `internal/verify`, `internal/termination`, `internal/scopematch`,
   `internal/acceptance`)
3. **Runs declared gates** — the commands the *driver* declared (session/spec), reports
   pass/fail. (`verify`)
4. **Records the failure as evidence** — a failed `done` writes a spec-keyed note naming the
   gap, so failure re-mints the spec instead of patching the work. (`done` → `internal/notelist`)
5. **Provides review lenses the driver selects** — fixed prompts, picked by flag. (`review`)
6. **Owns its state in commands, not chat** — session/exec/verdicts, plus the driver's agenda
   (`hit`) and reasoning (`note`) that ride the `handoff` seed across `/clear`.
   (`session`, `exec`, `hit`, `note`, `handoff`, `status`, `clean`)

That is the whole job. The shipped command surface is exactly: `spec`, `done`, `verify`,
`review`, `exec`, `note`, `hit`, `handoff`, `session`, `status`, `clean`. A file that serves
none of §1 is not mint's.

### The note wire (§1.4, the soul made mechanical)

When work fails acceptance, you do **not** debug the work — the failure is evidence the *spec*
was underspecified. `mint done` embodies this: on FAIL it appends one note under a spec-keyed
topic (`done-fail-<slug>-<spec-id>`) naming the failed clauses and why. Retries accumulate under
the same topic, so an underspecified spec's inadequacy becomes visible and compounding. mint
never *blocks* a patch — it makes re-minting the spec the obviously cheaper path. Embody, not
enforce.

---

## 2. What mint REFUSES to be (anything matching = rot, delete it)

- ❌ **Owns a pipeline or fixed sequence.** No modes, phases, stages, ordered step machine. The
  driver sequences primitives.
- ❌ **Runs the loop / orchestrates.** No `run`, `run-spec`, headless runtime, parallel runner,
  engine adapters. Orchestration is the driver's/harness's job — mint is called *from* a loop,
  it does not *be* the loop. (killed in the slim: `run`, `run-spec`, `continue`, `runtime/`,
  `parallel/`.)
- ❌ **Runs a learning loop.** No instincts, wins, metrics, workflow mining, skill generation,
  trust gradients.
- ❌ **Detects the stack or guesses commands.** The driver knows the repo and declares
  gates/reviews. mint runs what's declared and NEVER invents a command from a hardcoded table.
- ❌ **Has per-project setup.** Install once + `@AGENTS.md` in global config = works everywhere.
  No `mint init`, no config subsystem. `.mint/` is lazy-created per session.
- ❌ **Privileges a caller.** No Claude-special path. A Claude-Code hook can never hold floor
  logic — it's caller-privileged by definition. Floor lives only in the CLI every caller invokes.
- ❌ **Does domain work.** Design, browser, code-graph, context-mode are gum packs, not core.
  See `mint-gum-ledger.md`.
- ❌ **Is a plugin / has a marketplace / has a health-check.** Running mint IS the health check.
- ❌ **Calls a model itself.** The floor is deterministic, zero-token. Judgment is the driver's
  attached verdict; mint checks its provenance, never its content.

---

## 3. Gates and reviews are DYNAMIC and driver-declared (never hardcoded)

The single most important anti-staleness rule, because hardcoded gate/review tables are what rot
when tooling changes.

- **The driver declares what runs.** Gates set per session (cached in session state, inherited
  by specs, spec `<gates>` override). mint runs what's declared and guesses nothing — no
  declaration on a code diff → `done`/`verify` reports "no gates declared" and escalates. A
  guessed-wrong command is worse than none.
- **Review lenses are a fixed, curated set, selected by flag.** `mint review --list` prints the
  menu; `mint review --<lens>` loads one; `mint review --focus "…"` handles anything off-menu.
  The lens prompts are embedded in the CLI (`internal/command/reviewcmd`), not read from a
  directory — so a lens ships and versions with the binary. Adding a lens is a small code change,
  not a loose `.md` file.
- The teeth are in the floor, not the `review` command: `review` runs no model (it emits the
  prompt); clause 1 REFUSES `done` until every review the driver declared has an attached PASSING
  verdict (`mint exec record-review`). Mandatory-without-prompting, on the same rail as gates.

---

## 4. The slim/timeless discipline (how it stays this way)

- **The contract (`AGENTS.md`) lists no menus, no flags, no versions.** It states the contract
  and defers all specifics to `mint <command> --help` / `mint review --list`. A menu in the
  contract rots; a pointer to a runtime query does not.
- **Two halves, opposite rules.** The BINARY (`cli/` Go code) can be substantial — it's a
  binary, you don't read it; bloat there is a maintainability cost. The PROSE SURFACE an agent
  reads (`AGENTS.md`) must be minimal — bloat there degrades instruction-following.
- **Green tests on dead code prove nothing.** Dead code doesn't fail; it sits. The only thing
  that finds it is caller-reachability swept against §1/§2 — run that sweep, don't trust green.

---

## 5. How to use these principles (the maintenance rule)

- **Every new file/command/dependency** names which §1 job it serves. Serves none → it doesn't
  ship. Matches a §2 refusal → it doesn't ship.
- **Before adding anything**, climb the ladder: does a §1 primitive already cover it? Is it the
  driver's job (§3 — declared, not built-in)? Is it a gum pack (§2)? Only then build it, minimal.
- **Periodically sweep survivors** against §1/§2 by caller-reachability (grep for exports with
  zero callers; grep for refs to anything on the §2 kill-list).
- **These principles are firm.** They change only by an explicit, recorded decision that updates
  this file — never by a feature quietly accreting against them.
