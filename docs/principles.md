# mint principles

## One purpose

Drivers organize and execute work. mint records why an atomic unit is allowed to be called done.

Every surviving feature must define an atomic claim, collect or validate its evidence, preserve
the deterministic floor, or issue/validate a receipt for an exact source snapshot.

## Ownership

Drivers own workflow shape: projects, issues, task decomposition, terminals, agent selection,
processes, browsers, Git worktrees, branches, commits, PRs, retries, schedules, and deployment.

mint owns completion semantics: goal, scope, acceptance, declared gates/reviews, attempts,
attributable provenance, safety, anti-gaming, bounded termination, snapshots, receipts, and
freshness.

## Invariants

1. The same floor applies to every driver, agent, CI job, and human.
2. A unit is one atomic claim, not a ticket or project.
3. Evidence precedes status; `done` derives a result rather than setting a task label.
4. A receipt certifies one exact source digest and becomes stale when relevant source changes.
5. mint never owns an implementation/retry loop or executor lifecycle.
6. Provenance is generic and attributable. Typed claims are explicit but unauthenticated;
   external observations carry only the trust of their attester.
7. Unit policy is explicit. mint runs declared gates and never guesses commands.
8. State is global, private, repository-keyed, and worktree-isolated. No repo-local state exists.
9. JSON schemas are versioned; human output is never the integration protocol.
10. SlayZone and every future driver integrate through the same generic protocol.

## The floor

The seven fixed decisions are: completion evidence, maker/checker independence, safety carve-out,
anti-gaming, scope, bounded termination, and consequential-action gating. Safety-sensitive work
requires cross-vendor checking. A failed floor is evidence to sharpen the unit or rerun missing
checks, never permission to weaken verifiers.
