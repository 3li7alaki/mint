# Phase: Adversarial Testing (conditional)

Red team probe — actively tries to break the implementation.

---

## Trigger Evaluation (orchestrator decides)

1. If `config.reviewers.adversarial` is not enabled → **skip**
2. If diff < 30 lines → **skip** (too small for meaningful adversarial testing)
3. If spec has `<adversarial>false</adversarial>` → **skip**
4. Otherwise → **trigger**

## Pre-Dispatch

1. Identify the test framework from config (`gates.tests` command)
2. Get the list of modified files from `execution.json`
3. Collect acceptance criteria from spec XML

## Dispatch

**Dispatch tier: background** (`run_in_background: true`) — adversarial testing writes
and runs tests, takes 30-60s.

**MUST use `isolation: "worktree"`** — adversarial tests are throwaway and must never
pollute the real codebase. The orchestrator creates a worktree for this agent.

`mint-adversarial-tester` subagent with:
- Spec XML (acceptance criteria are attack targets)
- Git diff (what was implemented)
- Test framework/runner from config
- Modified file paths

Build prompt from `templates/agent-context.md` → "Adversarial Tester" section.

## Post-Dispatch Verification

1. Read the report — count BLOCKING findings
2. Record in `execution.json` → `reviews.adversarial`
3. If BLOCKING findings → set pipeline-state to `fix-blockings`
4. The worktree is automatically cleaned up (adversarial tests don't merge back)

## Output

"Adversarial testing: PASS (15 probes, all defended)." or
"Adversarial testing: FAIL — 2 vulnerabilities found. Fixing..."
