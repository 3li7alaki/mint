# Contributing to mint

Read [AGENTS.md](AGENTS.md) first. Changes must serve mint's single purpose: recording why an
atomic unit is allowed to be called done.

The Go module is in `cli/`. Keep core driver-agnostic: no ticket system, terminal, agent process,
worktree lifecycle, or model registry belongs here. Declare gates in the mint unit for the
change, preserve the seven floor clauses, and add focused tests for new behavior.

Before completion run:

```bash
cd cli
GOCACHE=/tmp/mint-gocache go test ./...
GOCACHE=/tmp/mint-gocache go vet ./...
cd ..
git diff --check
```

Use a genuinely independent context for required reviews and acceptance. Safety-sensitive work
requires a different model vendor. Do not add repo-local `.mint` state or mint entries to
`.gitignore`; tests use `MINT_STATE_HOME`.
