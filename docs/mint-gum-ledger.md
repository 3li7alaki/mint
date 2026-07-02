# mint gum — domain pack ledger

`mint gum` is the planned OPTIONAL extension mechanism for swappable DOMAIN capability
packs. Core mint stays domain-agnostic + timeless; a pack brings its own agents, config
keys, and detectors when installed. This file tracks capabilities cut from core that are
gum candidates — so the design intent isn't lost when core is slimmed.

NOT YET BUILT. This is a ledger of intent, not a spec.

## Candidates (cut from core, parked for gum)

- **design** — design intelligence: profiling, design review (a11y/RTL/i18n/brand/consistency),
  anti-pattern detection. Was `config.design.*` + design agents + design commands. The most
  obvious first pack.
- **browser** — PinchTab browser automation: runner, review mode, screenshot/scrape. Was
  `config.browser.*` + browser agents + browse/scrape/screenshot commands.
- **context-mode** — sandboxed execution + FTS5 search (github context-mode). Was
  `config.context.*` + `detectContextMode`. A power-user/perf pack.
- **code-graph** — code knowledge graph (codebase-memory-mcp): blast radius, call traces,
  dead-code, architecture awareness. Was `config.graph.*` + `detectCodeGraph`. Pairs well
  with review/decompose; a power-user pack.
- **context-autofill** — auto-populate a spec's `<context>` section (decided 2026-06-30; the
  spec gains an optional `<context>`/`<references>` block baked at creation like gates/reviews)
  so a dispatched worker doesn't read the whole repo cold. v1 = the spec CHANNEL (hand/driver-
  filled, built in core, NOT a pack). v2 = AUTO-FILL the section here as a pack: Aider-style
  tree-sitter+PageRank repo map, or repomix-style XML compression (mint already speaks XML).
  Pairs with code-graph (the graph slice IS the context). Build the channel before the filler.

## Open design question (the real work)

How does a gum pack attach cleanly? Old-mint wired these as deep `config.*` integrations
threaded through core. A pack must instead bring its OWN config schema, agents, and detectors
via a pack manifest, and attach at a defined seam — WITHOUT core knowing the pack exists.
context-mode + code-graph specifically need a cleaner attach point than the old config.graph/
config.context threading. Design the gum seam before resurrecting any of these.
