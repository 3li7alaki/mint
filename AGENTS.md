# mint — the discipline contract

mint is the engine that guarantees a unit of work is actually done properly — the same way
no matter which agent runs it, how many run, or whether a human is watching. It is not a
harness, scheduler, plugin, or loop runner. **You own the shape of the work; mint owns the
completion floor.**

You are the driver. mint is the floor you can't fall through. Run `mint <command> --help` for
exact commands and flags — this file is the contract, not the manual.

## The unit of work

One unit = one spec: `{ goal, scope, acceptance }`.

- **goal** — what outcome the unit must produce.
- **scope** — what may and may not be modified (`can-modify`).
- **acceptance** — testable criteria for done, in EARS form
  (`WHEN <trigger> THE <system> SHALL <observable response>`).

A commit, PR, push, or deploy is the *output* of a unit, not the unit — each inherits the
unit's `mint done` verdict. Create and edit specs through mint's CLI, not by hand-writing the
file; check `mint spec --help` for the current interface. State observable behavior in
acceptance, not the gates — deterministic gates run through mint, so acceptance should
describe what the change does, not restate gate checks.

## You own the shape — call the primitives the diff needs

You're already smart and already in a loop. mint doesn't run a pipeline; it gives you verbs
to call à la carte. The exact names live in `mint --help`; the judgment is yours:

| When the diff… | reach for |
|---|---|
| changed files | a scope check — confirm you stayed in your `can-modify` lane |
| needs a quality / perf / convention pass | a review on that dimension |
| touches security, trust boundaries, accessibility, or data-loss handling | the matching safety check, plus adversarial probing where it fits |
| might be gamed or fragile | adversarial probing — try to break it |
| **is finished** | `mint done` — **always; this is the gate** |

A typo runs straight to `done`. An auth change runs review → adversarial → `done`. You pick
the shape; mint enforces the floor at `done`, no matter what shape you chose.

## Capture the work that spawns work — `mint hit`

Mid-task, work spawns work: fixing X you notice Y is broken, Z should be refactored, the user
asks for W "later". That agenda dies in scrollback or your head. **When you notice a discrete
follow-up that won't be done in the current unit, capture it yourself** —
`mint hit add "<text>" --priority now|next|later` — and **say so visibly** in one line
(`📌 noted: …`) so the user can correct or drop it. Never capture silently (silent capture
breeds distrust and junk); never wait to be told (that's the pain this removes). It's your
judgment, not a regex: capture a real intention, not every passing thought. Open hits resurface
on their own at `mint status` and ride into the `mint handoff` seed, so they survive `/clear`.
`mint hit done <id>` / `drop <id>` when one is handled or abandoned. A hit with a substantial
write-up takes `--body <file>`; one about specific files takes `--file <path>`.

Likewise, when you work something out that a fresh context would need — a diagnosis, what you
tried, why you ruled an approach out — record it with `mint note add <topic> "<reasoning>"`
(append-mostly, topic-keyed; big analysis via `--body <file>`). A hit is *what to do*; a note
is *what you've figured out*. Both ride into the `mint handoff` seed, so the next session
inherits your agenda and your reasoning instead of re-deriving them.

## The floor — what `mint done` enforces

`mint done` is the choke point. It re-checks the **actual produced diff** — never your
"I'm done" — and clears a unit only when all seven clauses hold:

1. **Completion is proven, not declared.** Deterministic gates pass, an independent
   acceptance verdict is attached, and — on a code diff — every review lens you declared
   (`mint session set-reviews` / spec `<reviews>`) has a registry-validated, independently
   attributable passing verdict. A bare pass string is not evidence. A self-made claim is
   an input to verification, never the verdict.
2. **Maker ≠ checker, graded by risk.** The verdict's independence is verified by its
   provenance. Normal work needs a **fresh independent context** — a separate session with
   none of the maker's state. The safety carve-out needs a checker backed by a **different
   model vendor** than the maker; changing only the runner/chassis is not independence.
   Maker provenance comes from the maker's recorded execution state, never fields supplied
   by the checker. Multi-model chassis must report registry-validated vendor and model
   provenance, and local-only work fails closed unless mint can prove locality from trusted
   execution or fixed-registry evidence rather than a verdict claim.
3. **Safety carve-out is never minimized or skipped.** Security, trust-boundary validation,
   **accessibility**, and **data-loss** handling are floor. A diff that touches any of the
   four cannot reach done without the matching independent check.
4. **Nothing gamed.** A green is not trusted if the diff weakened, deleted, or skipped
   verifiers, assertions, coverage, or scoring. A tampered green is a FAIL.
5. **Scope respected.** Every changed file is inside the unit's declared `can-modify`.
6. **Bounded and terminating.** The unit ends in one hard terminal state:
   `done-verified | budget-exhausted | stuck-escalated | external-stop`.
7. **Every consequential action is gated.** Commit, merge, push, and deploy require the same
   full floor as any other done claim. No privileged path.

mint checks the deterministic clauses itself and **never calls a model**. The semantic
judgment — "does this diff satisfy acceptance" — is yours to supply as the attached verdict;
mint verifies that the provenance of that judgment is *well-formed, registry-valid, and
distinct from the maker*, not its content. Depth comes from you; integrity comes from mint.

**What "independent" means here, precisely — and its limit.** Provenance is declared, not
authenticated. mint reads the maker's identity from write-once execution state (so a checker
cannot forge the maker), validates every engine/vendor/locality claim against a registry
compiled into the binary, and fails closed on missing, malformed, or self-matching provenance.
What mint does **not** do is prove that the named engine actually *ran* — it trusts that the
`--maker-engine` at init and the `byEngine`/`--by-engine` on a verdict name the context that
truly produced the work. A single actor that declares one engine at init and a different one on
the verdict can therefore still manufacture an apparent maker≠checker split without a second
engine ever executing. mint raises the floor from "type the word `passed`" to "attach
registry-valid, fail-closed, maker-distinct provenance"; it does not yet make that provenance
unforgeable. Closing that gap requires authenticating execution (a signed attestation, or mint
itself invoking the checker) — an open, deliberately-unshipped decision, because the latter
collides with mint's rule that it is not a harness. Until then: the floor stops gaming and
accident, not a determined actor forging identity against their own gate.

**When `done` FAILS, re-mint the spec — don't patch the work.** A failed `done` appends a note
keyed to the spec (`done-fail-<slug>-<id>`) naming the clause that failed and why; retries
accumulate under that topic. Read the failure as evidence the *spec* was underspecified, not
that the code needs a patch. Work is disposable; the spec is the asset. mint won't block a
patch — but the cheaper, correct move is almost always to sharpen the spec and re-fire.

## If the floor blocks you, there's always a way forward

The floor catches gaming — it never traps honest work. If `done` blocks you, you are never
dead-ended into patching code or inventing a workaround. There is always an in-band path:

- **scope was genuinely too narrow** → widen it through mint's spec-change interface, with
  the reason recorded, before the edit is part of a done claim. If you already touched the
  file, revert it or record an independently-checked scope change before proceeding — the
  recovery path is in-band, but it can't be a silent post-hoc widening to cover a violation.
- **needs a verdict or safety review** → attach it (a fresh context; a different engine for
  the safety carve-out).
- **a declared review lens hasn't run** → run it (`mint review --<lens>`) and record the
  result (`mint exec record-review`); the in-band path is to actually run the review you
  declared, never to drop the declaration to get past the gate.
- **an honest refactor got flagged as gaming** → acknowledge with a reason; the acknowledgment
  is independence-checked, so you can't wave your own violation through.
- **genuinely stuck** → escalate; the unit terminates honestly as `stuck-escalated` and says why.

The rule is firm about one thing — *was this actually, independently done?* — and gets out of
your way everywhere else.

## What mint will not do

It won't run your loop, hold a schedule, or be a harness — that's the driver's job. It won't
grade your work with a model — it checks integrity, you supply judgment. It won't privilege
any caller — every agent, raw loop, or human driver loads this contract and clears the
identical floor.

## Operating rule

For each unit:

1. Make or load a spec — goal, scope, acceptance.
2. Shape the work with the primitives this diff needs.
3. Produce the change, inside scope.
4. Obtain the required independent acceptance verdict.
5. Run `mint done`.
6. Treat only a floor-clear terminal verdict as done.

The shape flexes. The floor does not.

## Worked example — clearing the floor on a real diff

Abstract steps become concrete here. This is a safety-tier change (touches a trust
boundary), made by one engine and independently verified by a *different vendor* — the
clause-2/clause-3 bar. Exact flags live in `mint <cmd> --help`; what's fixed here is the
*shape* and the verdict contract.

```bash
# 1. Define the unit. Bake gates/reviews from the session (or pass --scope/--acceptance).
mint spec new "harden token check" --slug harden-auth \
  --scope "src/auth/**" --acceptance "WHEN a token is empty, THE system SHALL reject it"

# 2. Record WHO is making the diff. Maker identity is written once, here, from the
#    maker's own context — never supplied later on the verdict a checker writes.
mint exec init harden-auth 001 --maker-engine codex        # codex/OpenAI is the maker

# 3. Do the work inside scope. Then get an INDEPENDENT verdict from a DIFFERENT context —
#    and, for the safety carve-out, a different model VENDOR than the maker. (Here: a
#    Claude/Anthropic reviewer verifies OpenAI's diff.) The reviewer adversarially checks
#    the diff and hands back its judgment; you attach it, you do not author its verdict.

# 4. Record each declared review lens with the reviewer's registry-valid provenance.
mint exec record-review harden-auth 001 security passed \
  --by-engine claude --by-vendor anthropic --by-model claude --by-locality remote \
  --by-session <the-reviewer-session>

# 5. Run the declared gates.
mint verify harden-auth 001

# 6. Route the done-decision through the floor with the acceptance verdict attached.
mint done harden-auth 001 --verdict verdict.json --terminal done-verified
```

The **acceptance verdict** (`--verdict <path>`, default `.mint/verdicts/<slug>-<id>.json`)
is the independent judgment mint checks for *provenance*, never content. Its contract:

| field | when | meaning |
|---|---|---|
| `accepted` | always | `true` only if the diff meets acceptance. `false`/absent fails clause 1. |
| `byEngine` | always | the engine that produced the verdict. Must be present and registry-known (`engine.IsKnown`); resolved against the compiled registry. |
| `bySession` | always | the session that produced the verdict. Must be present; *not* registry-backed. It only matters in one clause-2 path: when checker and maker are the **same vendor and same engine**, a `bySession` that differs from the maker's (and is strict-ASCII) is what establishes independence. It does nothing when the checker is a different vendor (that already passes) — and a same-vendor *different-engine* verdict fails regardless of `bySession`. The precise decision tree is `clauseMakerChecker`; the safe reading is: a genuinely different vendor is the clean path, and re-using the maker's own engine needs a real fresh session. |
| `byVendor`, `byModel`, `byLocality` | optional | resolved by engine *type*: for a **fixed** engine (e.g. `codex`, `claude`) they may be omitted, and if supplied must match the registry-pinned values; for a **configurable** chassis (e.g. `opencode`) they must be present and structurally valid (visible-ASCII vendor/model, locality `local`\|`remote`) but are *not* checked against a registry of allowed values. |
| `adversarialReviewed` + `adversarialReason` | logic / trust / safety diff | attests someone tried to *break* the behavior; the reason must be substantive. |
| `safetyReviewed` + `safetyReason` | safety carve-out (security / trust-boundary / accessibility / data-loss) | the matching independent safety review. |
| `tamperingReviewed` + `tamperingReason` | only if a green looks gamed | acknowledges a flagged verifier/assertion change with a reason. |

A *substantive* reason is at least 8 characters and contains a letter — a bare `"ok"` or
`"true"` does not clear it. Provenance names the context that produced the work; mint
registry-validates the *engine* (`byEngine`, and the vendor/model/locality of a fixed engine),
checks maker-distinctness by comparing that resolved provenance against the maker's, and — as
the **What "independent" means here** paragraph states — trusts that the named engine truly ran
(identity is *declared*, not authenticated).
