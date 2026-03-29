# Orchestrator Rules — Detailed Reference

Load this when you need detailed enforcement rules, risk scoring, or DoD criteria.
The router's "Orchestrator Laws" section has the essentials — this file has the full details.

---

## Risk Self-Regulation (WTF-likelihood)

Track a risk score during execution. When too high, stop and ask the user.

**Risk events:**
- Gate failure: +10%
- Spec retry (rewrite): +15%
- Fix touches >3 files: +5%
- File modified outside `<can-modify>`: +20%
- After 10 total fixes: +2% per additional fix
- Revert detected: +15%

**Thresholds:**
- **> 25%** → warn: "Risk is elevated. N issues so far. Continue?"
- **> 50%** → stop: "Risk too high. Recommend reviewing progress."
- **Hard cap:** 30 total fix attempts → stop regardless

Risk resets to 0 at task start. Tracked in session state.

---

## Definition of Done (hard gate)

Before marking a spec as `passed`, verify ALL criteria:

| Criterion | Check | On failure |
|-----------|-------|------------|
| Gates passing | `gates.*` all `pass` | Block — planner fixes + reruns |
| Spec review passed | `reviews.spec === "passed"` | Block — fix + re-review |
| Stage 2 passed | No unresolved BLOCKINGs | Block — fix BLOCKINGs + re-run |
| Doc check passed | Docs phase completed | Block — dispatch documenter |
| Screenshot reminder | If `"ui-changes"` + UI files touched | Warn user (non-blocking) |

Read `execution.json` to verify — do not trust agent verbal reports.

---

## Pipeline Enforcement Table

The orchestrator verifies each stage. If it skips verification, the feature is disabled.

| Stage | Verify | On failure |
|-------|--------|------------|
| Decomposition | `.xml` files exist with required fields | Re-dispatch planner (decompose only) |
| Execution tracking | `execution.json` exists and updated | Create/update from return value |
| Autocommit | Resolved value matches reality | Log mismatch, fix state |
| Gates | All `gates.*` pass | Planner fixes and reruns |
| De-sloppify | Triggered when conditions met | Dispatch if conditions met |
| Spec review | `reviews.spec` passed | Fix + re-review (max 2 rounds) |
| Stage 2 audit | All enabled reviewers dispatched | Fix BLOCKINGs + re-run (max 3) |
| Doc-manifest | Tracked files checked | Dispatch documenter for missed |
| Win logging | Entry added on final success | Append win entry |
| Session cleanup | Session file deleted | Delete file |
| Stop signal | `interrupted` status handled | Consume stop file, prompt user |
| Learning loop | Issues/wins/instincts read | Read before dispatch |
| Retry | `attempts[]` checked, max 2 | Escalate on third attempt |

---

## TDD Control

- `config.tdd.default: true` → ALL specs get `<tdd>true</tdd>` unless spec overrides
- `config.tdd.default: false` (default) → TDD only when spec explicitly opts in
- Planner MUST check this config before writing specs

## Coverage Gate

If `config.gates.coverage` is configured, coverage must meet threshold before commit.
Checked as part of test gate — tests pass but coverage below threshold = blocked.

## Delegation Rules

- One subagent, one job, one clear deliverable
- Subagents cannot spawn other subagents
- Subagents that need user input return the question — orchestrator relays
