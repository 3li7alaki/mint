# Agent Control — Reference

Stop, pause, and freeze signals for controlling running agents.

---

## Stop Signal

**File:** `.mint/stop`

**Create:** `touch .mint/stop` or `echo "reason" > .mint/stop`

**Agent behavior:** Check at major checkpoints (between specs, review stages, phases).
Stop at next checkpoint, save progress to execution.json, return `"interrupted"`.

**Orchestrator behavior (mandatory):**
1. Read `.mint/stop` contents for reason
2. Delete stop file (single-use)
3. Update execution.json: status → `interrupted`
4. Report: "Agent interrupted. Completed: X. Remaining: Y. Reason: Z"
5. Ask: resume / restart / abandon
6. Do NOT dispatch further agents until user responds

**Recovery options:**
- Resume with feedback — correction in context, continue
- Restart spec — discard, rerun with new approach
- Restart task — replan from scratch
- Abandon — keep committed work, stop

---

## Pause Signal

**File:** `.mint/pause`

**Create:** `touch .mint/pause` or `echo "let me check" > .mint/pause`

**Agent behavior:** Freeze in place, poll every 5s for removal.

**Resume:** `rm .mint/pause` — if file had content, agent reads it as a correction.

| Signal | Meaning | Recovery |
|--------|---------|----------|
| `.mint/stop` | Abort | Resume/restart/abandon |
| `.mint/pause` | Wait | Remove file to continue |

---

## Freeze / Guard

**Commands:**
- `/freeze <path>` — block agent from modifying path
- `/freeze <glob>` — pattern: `/freeze src/**/*.test.ts`
- `/guard <path> <reason>` — freeze + explain why
- `/unfreeze <path>` — remove
- `/unfreeze --all` — remove all

**Storage:** `.mint/.freeze-list.json` (gitignored)

**Enforcement:** Pre-edit hook blocks Edit/Write to frozen paths.
Agent sees: `[mint] FROZEN: path is frozen.` or `[mint] GUARDED: path — reason`.
