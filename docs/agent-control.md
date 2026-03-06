# Agent Control

How to monitor and control mint agents during execution.

## The Problem

When you dispatch an agent (planner, shipper, researcher), it runs autonomously. If it starts going in the wrong direction, you need a way to intervene.

## Stop Signal

**Fastest way:** Use the `/mint:stop` command directly in chat:

```
/mint:stop                          # Simple stop
/mint:stop wrong approach           # Stop with reason
```

**Alternative:** Create the file manually:

```bash
touch .mint/stop
echo "wrong approach - using deprecated API" > .mint/stop
```

### What Happens

1. Agent finishes its current atomic operation
2. Checks for stop file at the next checkpoint
3. Saves progress to `execution.json`
4. Returns to orchestrator with status `interrupted`
5. Orchestrator reads and deletes the stop file
6. You're asked how to proceed: resume / restart / abandon

### Checkpoints

| Agent | When it checks |
|-------|---------------|
| Planner (decompose) | After codebase scan, before writing specs |
| Planner (execute) | Before each file modification, after gates |
| Shipper | Between phases, between tasks |
| Researcher | Between codebase scan and web search |
| Stage 2 reviewers | Before starting (can't interrupt mid-review) |

### Limitations

- **Not instant** — agents finish their current operation first
- **Parallel reviewers** — once dispatched, they run to completion
- **No partial commits** — if stopped mid-spec, changes stay uncommitted

## Background Execution

For long tasks, agents can run in background:

```
Agent dispatched in background.
Task ID: abc123
Monitor: tail -f .mint/tasks/<slug>/output.log
Stop: touch .mint/stop
```

Check progress anytime:
```bash
tail -50 .mint/tasks/<slug>/output.log
```

## Recovery Options

After interruption:

| Option | What happens |
|--------|-------------|
| **Resume** | Continue from last checkpoint |
| **Restart** | Discard progress, start fresh (maybe with modified approach) |
| **Abandon** | Mark specs as `interrupted`, clean up |

## Monitoring Progress

### Check execution state

```bash
cat .mint/tasks/<slug>/<spec-id>/execution.json
```

Shows: status, attempts, gate results, review verdicts.

### Check issues

```bash
cat .mint/issues.md
```

Shows: past failures, root causes, what to avoid.

### Check wins

```bash
cat .mint/wins.md
```

Shows: successful patterns, what works in this codebase.

## Preventing Runaway Agents

### Tight specs

The best control is a precise spec:
- Explicit `<can-modify>` scope
- Concrete `<steps>` with file paths and line numbers
- Clear `<acceptance>` criteria
- `<pitfalls>` from past issues

### Fail-safe rules

Agents already have built-in limits:
- **Fail twice → stop** — same spec fails twice, agent escalates
- **Scope is law** — can't modify files outside declared scope
- **Never push** — only commit; you review and push

### Model selection

Use lighter models for lower-stakes work:
```json
{
  "reviewers": {
    "conventions": { "enabled": true, "model": "haiku" },
    "security": { "enabled": true, "model": "opus" }
  }
}
```

Haiku is fast and cheap. Opus is thorough. Match model to task risk.
