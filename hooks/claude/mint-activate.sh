#!/usr/bin/env bash
# mint discipline injector (opt-in Claude Code hook).
# Fires on SessionStart + UserPromptSubmit. Injects the mint floor into attention
# so the agent actually uses it — instead of reading the contract as optional and
# drifting. This is the ponytail pattern: always-on, no blocking, no auto-run.
#
# It does NOT run `mint done` (that needs a spec, which can't be fabricated). It
# keeps mint present so the agent reaches for it on real units of work.
#
# Wire it via `mint hook claude` (see mint --help) or point your hooks config here.
# No-op if mint isn't on PATH, so a shared config never breaks a machine without mint.

command -v mint >/dev/null 2>&1 || exit 0

cat <<'RULE'
MINT DISCIPLINE ACTIVE — the floor, not a suggestion.

For any real unit of work (a feature, fix, or change worth proving done):
- `mint spec new "<title>"` scaffolds it, then fill goal/scope/acceptance. NEVER
  hand-write the spec XML — use the CLI so the format is always valid.
- Do the work inside scope.
- `mint done <slug> <spec-id>` re-checks the ACTUAL diff against the floor — gates
  pass, scope respected, nothing gamed, an independent acceptance verdict attached.
- Never claim "done" on a real unit without it. "Looks done" is not done.
- A failed `done` writes a spec-keyed note — re-mint the spec sharper, don't patch the work.

Trivial work (typo, one-liner, docs) skips the ceremony. Judgment, not ritual.
`mint --help` is the command surface. Reach for the floor when the work matters.
RULE
