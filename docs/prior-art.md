# Prior art

What other people built and measured for the problems mint solves, researched 2026-10-17.
Dates are publication or last commit. Re-check anything marked "new" before depending on it:
steal the idea, not the dependency.

## What the ecosystem now covers natively

- **Triggering a check.** Claude Code Stop, SubagentStop and TaskCompleted hooks block with exit 2.
  On those three events a block only sends the agent back to work; `onFailure: "block"` has no
  effect on them (https://code.claude.com/docs/en/hooks). Codex has Stop and SubagentStop too, and a
  Stop block becomes a continuation prompt (https://learn.chatgpt.com/docs/hooks).
- **`/goal`** on both harnesses: a small model judges completion from the transcript only, runs no
  command (https://code.claude.com/docs/en/goal).
- **Specs.** Spec Kit `/speckit-converge`, OpenSpec `/opsx:verify`, Kiro property tests, Tessl
  `[@test]` links. Advisory, no record.
- **Authorship provenance.** git-ai v3 (`refs/notes/ai`, Claude and Codex), Copilot
  `Agent-Logs-Url` trailer and signed commits. Agent Trace and in-toto AI predicates (#601 to #604)
  are proposals, not shipped.
- **Same-vendor review.** Claude Code Review, Codex `/review`, Copilot review.

Nothing found binds an acceptance verdict to a code snapshot and checks freshness later. That part
of mint is still uncovered.

## Measured findings

- **Agents cheat often when tests and spec disagree.** ImpossibleBench (arXiv 2510.20270,
  2025-10): GPT-5 cheated on 54 to 76% of impossible tasks. Read-only tests stopped edits but not
  special-casing. A "flag it as impossible" exit cut GPT-5 from 54% to 9%.
- **Held-out tests catch hacks on clear problems.** EvilGenie (arXiv 2511.21654, 2026-05): with
  about 30% of tests held out, no hack passed every test on unambiguous problems. Claude Code
  special-cased 20.7% of problems.
- **Never train or loop the worker against the detector.** OpenAI CoT monitoring (arXiv 2503.11926):
  penalising what the monitor sees made the model hide the hack, recall fell to near 0.
- **Reset test files before grading.** NIST CAISI (2025-11-28) recommends it after finding
  disabled-assertion overfits on SWE-bench Verified.
- **Reviewer capability beats vendor diversity.** arXiv 2610.01471 (2026-10): same plus
  cross-model review found 56.7% of planted errors vs 42.7% for two same-model reviews, but was not
  better than two reviews by the strongest model. arXiv 2607.21656 (2026-07): Claude reviewing Codex
  raised pass rate 71.6% to 89.7%; Codex reviewing Claude lowered it 91.4% to 82.8% by overwriting
  correct code.

## Code worth copying

- **Baseline tree with untracked files.** kylesnowschwartz/claude-bumper-lanes,
  `bumper-lanes-plugin/tools/bumper-lanes/internal/git/git.go:28-95` (`CaptureTree`): temp
  `GIT_INDEX_FILE`, `read-tree HEAD`, `add -u`, add untracked paths, `write-tree`. Scope check
  becomes `git diff-tree` against that SHA. Fixes the predating-untracked-files bug.
- **Replay into a clean grader.** vercel-labs/agent-eval PR #213 (2026-10-08), `verifier.ts`
  (`snapshotWorkspace`, `pathsChangedSince`, `findTampering`): apply the agent's diff minus
  protected paths to a fresh sandbox, then grade. Tampering becomes inert instead of detected.
- **Verdict from checker stdout, fail closed.** openai/codex-plugin-cc,
  `plugins/codex/scripts/stop-review-gate-hook.mjs`: first line `ALLOW:` or `BLOCK:`, timeout or bad
  output blocks. hidevinliu/red-green-mode `tools/rgm_stop_hook.sh`: pass only on rc 0 AND a
  printed `RGM_GATE=PASS`.
- **Loop policy.** wasaybuilds/proof-of-done (new): block once, re-block only on new findings, cap
  3, then hand to the human. Bare `stop_hook_active` exits bounce only once (claudekit
  `cli/hooks/base.ts:76`). Ralph loops without an iteration cap ran 1966 times (dev.to @sean8).
- **AST test counting over regex.** nizos/tdd-guard `src/hooks/testCounter.ts`, `diffTestCount` in
  `src/hooks/processHookData.ts`.

## Guarding the guard

- **Hooks are flaky as hard gates.** SubagentStop open bugs: anthropics/claude-code #82249 (not
  firing for background subagents), #78463, #88995. Never fires for `codex exec` from Bash. Use
  hooks to nudge; the merge step is the gate.
- **Agents route around in-session rules.** #40117: Opus 4.6 used `--no-verify` and stash tricks
  past husky hooks. `permissions.deny` is evaded by equivalent command forms (#39987).
  hahahahahahahahah6/agent-guard `bashwrite.py` parses Bash write targets.
- **Pin and manage the hook.** proof-of-done pins a SHA-256 of its own launcher after a live agent
  patched it. Claude `allowManagedHooksOnly`, Codex `allow_managed_hooks_only`. agent-guard
  `verify.py` reports a configured guard that ran zero times.
- **Unforgeable means a separate trust domain.** SLSA Build L3 (https://slsa.dev/spec/v1.1/levels):
  signing material out of reach of build steps. in-toto layout signed before work, functionaries
  sign their own step. Google Binary Authorization for Borg re-verifies at least daily.
- **Signed agent receipts exist but nobody uses them.** zhengqiuyang/agent-attest (new):
  `verification.source: executed|claimed`, ed25519, `verify --recheck`. MurphySig: zero outside
  adoption after 14 weeks. No evidence found that reviewers read attached evidence.

## Known blind spots of diff-based test checks

Skylos (https://skylos.dev/blog/testwashing-coding-agents-delete-tests, 2026-10-14) catches
deleted, skipped, emptied tests and always-true `__eq__`, and admits it misses an edited expected
value, an assertion loosened to truthiness, an `if amount == 3` special case, and a removed
`@login_required`. Held-out tests and an LLM reviewer pointed at test diffs cover these; regexes do
not.
