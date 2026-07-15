# mint — atomic completion ledger contract

> Drivers organize and execute work. mint records why an atomic unit is allowed to be called done.

mint is driver-, agent-, and user-agnostic. A human shell, CI job, SlayZone card, or any other
driver uses the same commands and clears the same floor. mint never launches agents, manages
tickets, owns terminals, or creates and removes Git worktrees.

## One unit, one claim

A unit contains one goal, an explicit `can-modify` scope, observable EARS acceptance criteria,
and its declared gates and review lenses. Create it through the CLI:

```bash
mint spec new "harden token validation" --slug harden-token \
  --goal "Reject empty tokens" --scope "src/auth/**,tests/auth/**" \
  --acceptance "WHEN an empty token is supplied, THE system SHALL reject it" \
  --gate "tests: go test ./..." --reviews "security,adversarial"
```

The JSON response contains the unit ID and absolute spec path. Specs are operational state,
not repository files.

## State and worktrees

mint writes no `.mint` directory and never edits `.gitignore`. State lives at:

```text
$MINT_STATE_HOME/repos/<repo-id>/worktrees/<worktree-id>/{units,attempts,receipts,notes}
```

Without `MINT_STATE_HOME`, the base is `$XDG_STATE_HOME/mint`, or
`~/.local/state/mint`. Repository identity comes from the canonical common Git directory.
Worktree identity comes from the canonical worktree root and worktree Git directory. Linked
worktrees therefore aggregate under one repository while keeping all mutable evidence isolated.
There is no repo-local compatibility reader or migration path.

## Attempts and provenance

An attempt is one bounded effort against a unit. Initialize it with generic provenance:

```bash
mint exec init harden-token 001 --attempt maker-1 \
  --executor codex --vendor openai --model gpt-5 \
  --locality remote --execution-ref driver-run-42
```

`executor`, `vendor`, `model`, `locality`, and `executionRef` are opaque attributable claims.
A driver may add `--observed-by` and `--attestation`; mint stores them but owns no executor
registry, binary adapter, default executor, or process lifecycle. Typed provenance is declared
evidence, not authenticated identity. `observedBy` and `attestation` improve attribution only to
the strength of the supplying driver's trust boundary.

Attempt evidence is immutable after a terminal outcome. All read-modify-write evidence updates
are serialized by an attempt lock. If a unit has multiple attempts, commands require
`--attempt`; mint never guesses which effort should receive evidence.

## Verification and reviews

Run only the gates declared in the unit:

```bash
mint verify harden-token 001 --attempt maker-1
```

mint never guesses stack commands. Record independently produced review results:

```bash
mint exec record-review harden-token 001 security passed --attempt maker-1 \
  --executor opencode --vendor zai --model glm-5.2 --locality remote \
  --execution-ref reviewer-run-9
```

Normal work may be checked by the same executor/vendor only from a distinct execution reference.
A different executor under the same vendor does not establish independence. Safety reviews and
safety-tier acceptance require a different vendor from the maker.

## Acceptance verdict and done

An independent checker supplies a versioned JSON verdict:

```json
{
  "schemaVersion": 1,
  "accepted": true,
  "executor": "opencode",
  "vendor": "zai",
  "model": "glm-5.2",
  "locality": "remote",
  "executionRef": "acceptance-run-10",
  "adversarialReviewed": true,
  "adversarialReason": "exercised malformed and boundary inputs",
  "safetyReviewed": true,
  "safetyReason": "checked authentication bypass and leakage paths"
}
```

The adversarial and safety fields are required only when the floor classifies the diff into
those tiers. Then run:

```bash
mint done harden-token 001 --attempt maker-1 --verdict /tmp/verdict.json --json
```

`done` runs the declared gates, evaluates all seven clauses, captures the source before and
after evaluation, rejects a race, terminates the attempt, and writes an immutable receipt for
that exact source digest. `mint receipt verify <path>` reports whether it is still current.

The seven clauses remain fixed: verifiable completion; maker/checker independence; safety
carve-out; anti-gaming; scope; bounded terminal state; and floor-gated consequential action.
Only `done-verified`, `budget-exhausted`, `stuck-escalated`, and `external-stop` are hard terminal
outcomes.

## Operating rule

1. Define or load the atomic unit.
2. Initialize an explicit attempt with honest provenance.
3. Work only inside scope.
4. Run declared gates and required independent reviews.
5. Attach an independent acceptance verdict.
6. Run `mint done` and treat only its receipt as proof.
7. Use `mint status --json` to inspect units, attempts, missing evidence, receipts, and freshness.

If work spawns a separate future task, give it to the external driver. `mint note` is only for
unit/floor reasoning; mint has no backlog, handoff, session, project, or retry-loop ownership.
