# mint

> Drivers organize and execute work. mint records why an atomic unit is allowed
> to be called done.

[![License: MIT](https://img.shields.io/badge/license-MIT-black)](LICENSE)
[![Site](https://img.shields.io/badge/site-mint.area--51.cloud-1f6f5c)](https://mint.area-51.cloud)
[![Ko-fi](https://img.shields.io/badge/Ko--fi-support-ff5e5b?logo=kofi&logoColor=white)](https://ko-fi.com/3li7alaki)

An agent will report success. That report is a sentence, not evidence, and
nothing in the loop was ever asked to tell the two apart. mint is a small CLI
that holds the completion floor: declared gates, independent review, and an
immutable receipt bound to the exact source it verified.

It owns only that. Your driver keeps projects, tickets, terminals, agents,
worktrees, retries, Git and deployment.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/3li7alaki/mint/main/install.sh | sh
```

One static binary. It creates no `.mint` directory in your repo and never edits
your `.gitignore`. State lives under `$XDG_STATE_HOME/mint`, keyed by repository
and isolated per worktree. Set `MINT_STATE_HOME` to override it for tests.

## What a run looks like

Real output, with state paths abbreviated.

```console
$ mint spec new "expose service health" --slug health-check \
    --goal "Expose service health" --scope "src/**" \
    --acceptance "WHEN the health endpoint is called, THE service SHALL return its status" \
    --gate "tests: bun test" --reviews quality
{
  "schemaVersion": 1,
  "specPath": ".../units/health-check/001/spec.xml",
  "slug": "health-check",
  "id": "001"
}

$ mint exec init health-check 001 --attempt maker-1 \
    --executor codex --vendor openai --model gpt-5 \
    --locality remote --execution-ref driver-run-42

$ mint verify health-check 001 --attempt maker-1
  tier: full - ok tests

$ mint done health-check 001 --attempt maker-1 --verdict verdict.json
  FAIL - floor not clean (failed clauses: 1, 3)
    fail clause 1: verifiable-completion
    ok clause 2: maker-not-checker
    fail clause 3: safety-carve-out
    ok clause 4: anti-gaming
    ok clause 5: scope-respected
    ok clause 6: bounded-terminating
    ok clause 7: floor-gated-action
```

It classified the diff into the safety tier itself and refuses to issue a
receipt until an independent adversarial and safety check are attested in the
verdict. Attach them and re-run:

```console
$ mint done health-check 001 --attempt maker-1 --verdict verdict.json
  PASS - floor clean, done
    ok clause 1: verifiable-completion
    ...
  receipt: .../receipts/health-check/001/20260726T120221Z-2169e1977a45.json

$ mint receipt verify .../20260726T120221Z-2169e1977a45.json
  current sha256:3118a072308ac4a7f0e5b9ef13394b505c7aebdb0a93fcc7a1aa9fc4d2337248
```

Change one file inside the unit's declared scope, then ask again:

```console
$ mint receipt verify .../20260726T120221Z-2169e1977a45.json
  stale sha256:3118a072308ac4a7f0e5b9ef13394b505c7aebdb0a93fcc7a1aa9fc4d2337248 - source snapshot changed after receipt issuance
```

That is the whole product. A claim that outlives the code it described is not a
claim any more, and now you can ask.

## The floor

Seven fixed clauses, the same for a typo and for an auth rewrite: verifiable
completion, maker and checker independence, the safety carve-out, anti-gaming,
scope, bounded terminal state, and floor-gated consequential action.

Independence is structural. Normal work may be checked by the same vendor only
from a distinct execution reference; a different executor under the same vendor
does not establish it. Safety reviews and safety-tier acceptance require a
different vendor from the maker.

Provenance is a declared record. `executor`, `vendor`, `model`, `locality` and
`executionRef` are opaque attributable claims. mint stores them and owns no
executor registry, no binary adapters and no process lifecycle. It does not
pretend to authenticate an identity it cannot see.

## Commands

| command | does |
|---|---|
| `mint spec` | define, edit, validate and inspect an atomic unit |
| `mint exec` | initialize an attempt and record gate, review and terminal evidence |
| `mint verify` | run the unit's declared deterministic gates |
| `mint review` | print a review lens prompt. It never launches a model |
| `mint done` | evaluate the floor and issue an immutable receipt |
| `mint status` | report units, attempts, missing evidence and freshness |
| `mint receipt` | show or verify a receipt |
| `mint note` | retain unit and floor reasoning |
| `mint clean` | report or remove orphanable locks and incomplete receipt claims |

Every command takes `--json`. Human output is never the integration protocol.

## What it is not

- Not a work organizer. Tickets, worktrees, branches and retries stay with your driver.
- Not a decision about what is worth building. That is [blueprint](https://github.com/3li7alaki/blueprint).
- Not an agent runner, and not an authority on who ran what.

## More

- [AGENTS.md](AGENTS.md) is the exact operating contract for agents working through mint.
- [docs/principles.md](docs/principles.md) covers the product boundaries.
- [mint.area-51.cloud](https://mint.area-51.cloud) is the overview, with the rest of the lab at [area-51.cloud](https://area-51.cloud).

## Support

mint is free and MIT licensed, and there is no pricing page. If it saved you an
afternoon, that is worth about the price of a coffee.

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/3li7alaki)

Need something like this built for your own stack? Commissions are open at
[ko-fi.com/3li7alaki](https://ko-fi.com/3li7alaki).

## License

MIT. See [LICENSE](LICENSE).
