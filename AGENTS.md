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
   (`mint session set-reviews` / spec `<reviews>`) has an attached passing verdict. A
   self-made claim is an input to verification, never the verdict.
2. **Maker ≠ checker, graded by risk.** The verdict's independence is verified by its
   provenance. Normal work needs a **fresh independent context** — a separate session with
   none of the maker's state. The safety carve-out needs a **genuinely different engine**
   than the maker.
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
mint verifies that judgment *happened and was independent*, not its content. Depth comes from
you; integrity comes from mint.

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
