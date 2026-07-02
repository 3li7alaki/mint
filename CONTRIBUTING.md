# Contributing to mint

mint is a small, deterministic engine. The bar for any change is the same bar mint enforces
on the work it verifies: does it serve the engine, and does it hold the line? Read
[`docs/principles.md`](docs/principles.md) first — it is the checkable definition of what mint
is and what it refuses to be. A change that violates it does not ship, regardless of whether
tests pass.

## Getting started

```bash
git clone https://github.com/3li7alaki/mint
cd mint/cli
go build ./...
go test ./...       # the suite must be green before and after your change
```

mint is a single static **Go** binary with **zero runtime dependencies**. Keep it that way —
adding a dependency for what a few lines can do is exactly the bloat the engine refuses
(principles §2, §4). The two-halves rule: the CLI binary can be substantial (you don't read
it), but the prose surface an agent reads — `AGENTS.md` — stays minimal, because bloat there
degrades instruction-following.

## Project structure

```
mint/
├── AGENTS.md                  # the contract a driver loads (CLAUDE.md is a one-line @AGENTS.md bridge)
├── cli/                       # the Go module (module root — run go from here)
│   ├── cmd/mint/main.go       # entry point + command dispatch
│   └── internal/
│       ├── floor/             # the floor: the 7 clauses, anti-gaming, scope, termination
│       ├── acceptance/        # EARS acceptance parsing
│       ├── verify/            # runs driver-declared gates
│       ├── notelist/          # the note wire (failed done → spec-keyed note)
│       ├── execstate/         # per-spec verdict/gate/review state
│       ├── session/           # session state (declared gates/reviews, cached)
│       ├── specschema/        # spec (XML) parsing + resolution
│       ├── scopematch/        # can-modify scope matching
│       └── command/           # one package per command (donecmd, speccmd, verifycmd,
│                              #   reviewcmd, execcmd, notecmd, hitcmd, handoffcmd, ...)
├── docs/                      # principles.md, mint-gum-ledger.md
└── .mint/                     # per-session work-state (gitignored) — lazy-created, never committed
```

There is no pipeline, modes/phases, plugins, hooks, learning loop, config subsystem, `mint init`,
or engine-adapter/orchestration layer. If you find a reference to any of those, it's stale prose
to fix — they were deleted (principles §2).

## The floor is the engine — handle it with care

`cli/internal/floor/` (`floor.go`, `input.go`) and `donecmd` are the load-bearing core. Two rules:

- **It never calls a model.** The floor is deterministic and zero-token. If a change to the
  floor needs an LLM, it belongs in the driver's attached verdict, not in mint.
- **No clause may dead-end honest work.** Any clause that can block ships with an in-band path
  forward — an independence-gated override or an escalate-to-caller exit. A clause that can trap
  a correct agent is not done.

Changes to floor logic need a test that pins the new behavior against a real diff, not a mock.

## Tests

```bash
cd cli && go test ./...
```

- Tests run against real files, real git, real lenses — **no mocking** of the things under test.
- Non-trivial logic ships with a table-driven test that fails if the logic breaks. Trivial
  one-liners do not need one (YAGNI applies to tests too).
- Green tests on dead code prove nothing. If you delete a feature, delete its tests too — a test
  referencing X is not evidence X should exist.

## Writing a review lens

Lenses are a curated set embedded in `cli/internal/command/reviewcmd/reviewcmd.go` (a
`map[string]lens`), selected by the driver via `mint review --<lens>`. Adding a lens is a small
code change there: a name, a one-line description, and the prompt body. A lens is pure prose —
it emits a prompt and runs no model; the teeth are in the floor (clause 1 refuses `done` until
every declared review has a passing verdict attached). A lens must never contain pipeline
framing, MCP coupling, or config-schema reads.

## Commit conventions

Format: `type(scope): description`.

**Types:** `feat` (new capability), `fix` (bug fix), `docs`, `refactor` (no behavior change),
`test`, `chore` (maintenance).

**Scope:** the file or component (e.g. `floor`, `donecmd`, `review`, `cli`).

```
feat(donecmd): append a spec-keyed note when done fails
fix(floor): account assertions per-file to close the net-zero swap
docs(principles): record the orchestration cut
```

**No AI attribution** — never add `Co-Authored-By` or mention AI tooling in a commit, PR title,
or PR body. Never include a version number in a commit message.

## PRs

- One logical change per PR. If it touches more than ~5 files, consider splitting.
- Branch `feat/<name>` or `fix/<name>` off `main`; squash-merge.
- All tests pass, and the change traces to a principles clause (or removes something that
  violated one).
- Agents commit; a human reviews and pushes.

## What not to do

- Don't add a feature that matches a §2 refusal (a pipeline, orchestration/loop-running, a
  learning loop, stack detection, per-project setup, a plugin system, a model call in the floor).
- Don't add a runtime dependency for what a few lines do.
- Don't bloat the prose surface — keep `AGENTS.md` minimal, deferring specifics to
  `mint <command> --help`.
- Don't let the floor trap honest work, and don't weaken a clause to make a test pass.
