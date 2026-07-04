# mint — opt-in Claude Code hook

mint core is a floor binary, agent-blind. This directory is an **optional** convenience
for Claude Code users who want mint's discipline *always in attention* — the same way
ponytail/caveman stay on — instead of reading the contract as optional and drifting past it.

`mint-activate.sh` injects the floor into context on SessionStart + UserPromptSubmit. It
**does not run `mint done`** (the floor needs a spec, which can't be fabricated) and it
**never blocks** — it keeps mint present so the agent reaches for it on real units of work.
It no-ops if `mint` isn't on PATH, so a shared config is safe on any machine.

## Wire it

Add to `~/.claude/settings.json` (adjust the path to where mint is cloned/installed):

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "startup|resume|clear|compact",
        "hooks": [{ "type": "command", "command": "bash /path/to/mint/hooks/claude/mint-activate.sh" }] }
    ],
    "UserPromptSubmit": [
      { "hooks": [{ "type": "command", "command": "bash /path/to/mint/hooks/claude/mint-activate.sh" }] }
    ]
  }
}
```

That's it. mint's discipline is now always-on, no per-turn reminding.
