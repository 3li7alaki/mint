# mint

> Drivers organize and execute work. mint records why an atomic unit is allowed to be called done.

mint is a small CLI for atomic completion claims. A driver owns projects, tickets, terminals,
agents, worktrees, retries, Git/PR UI, and deployment. mint owns only the unit contract, declared
evidence policy, attributable provenance, the seven-clause completion floor, exact source
snapshots, immutable receipts, and receipt freshness.

SlayZone is a useful first integration target, never a dependency or privileged caller.

## Model

```text
optional issue (Linear/GitHub/Jira)
└── external driver task/workspace
    └── isolated Git worktree
        └── mint unit {goal, scope, acceptance, policy}
            ├── attempt + evidence
            └── snapshot-bound completion receipt
```

State is global and worktree-isolated:

```text
$XDG_STATE_HOME/mint/repos/<repo-id>/worktrees/<worktree-id>/
├── units/
├── attempts/
├── receipts/
└── notes/
```

Set `MINT_STATE_HOME` to override the base for tests or automation. mint creates no repository
`.mint` directory and never changes `.gitignore`.

## Quick start

```bash
mint spec new "add health check" --slug health-check \
  --goal "Expose service health" --scope "cmd/**,internal/health/**" \
  --acceptance "WHEN the health endpoint is called, THE service SHALL return its status" \
  --gate "tests: go test ./..." --reviews quality

mint exec init health-check 001 --attempt maker-1 \
  --executor codex --vendor openai --model gpt-5 \
  --locality remote --execution-ref run-123

mint verify health-check 001 --attempt maker-1

mint exec record-review health-check 001 quality passed --attempt maker-1 \
  --executor opencode --vendor zai --model glm-5.2 \
  --locality remote --execution-ref review-456

mint done health-check 001 --attempt maker-1 --verdict /tmp/acceptance.json
mint status --json
mint receipt verify /absolute/path/from-done.json
```

The acceptance verdict is schema version 1 and contains `accepted`, `executor`, `vendor`,
`model`, `locality`, and `executionRef`. Logic and safety changes additionally require
substantive adversarial/safety attestations. Generic provenance is a declared record; mint does
not authenticate executors or run checker processes.

## Commands

- `mint spec` — define, edit, validate, and inspect an atomic unit.
- `mint exec` — initialize an attempt and record gate/review/terminal evidence.
- `mint verify` — run the unit's declared deterministic gates.
- `mint review` — print a review lens prompt; it never launches a model.
- `mint done` — evaluate the floor and issue an immutable receipt.
- `mint status` — report worktree units, attempts, missing evidence, and freshness.
- `mint receipt` — show or verify a receipt.
- `mint note` — retain unit/floor reasoning.
- `mint clean` — report or remove orphanable locks and incomplete receipt claims in the current worktree state.

See [AGENTS.md](AGENTS.md) for the exact operating contract and
[docs/principles.md](docs/principles.md) for product boundaries.
