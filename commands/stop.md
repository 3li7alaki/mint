---
description: "Interrupt running agents — halts at next checkpoint, asks what's wrong, steers correction"
---

# /stop Command

Interrupt running mint agents and course-correct.

## Usage

```
/stop
```

No arguments needed — the command will ask you what's wrong.

## What It Does

1. **Creates stop signal** — `.mint/stop` file
2. **Asks what's wrong** — "What's the issue? What should the agent do differently?"
3. **Captures your feedback** — saved to `.mint/stop` for the agent
4. **Reports status** — "Stop signal sent. Agent will halt and receive your feedback."

## Flow

```
User: /stop

Claude: Stop signal sent.

What's wrong with the current approach?
(Your feedback will be passed to the agent when it halts)

> [User types: "It's using the old auth pattern, should use the new JWT flow from auth.ts"]

Claude: Got it. Feedback saved.

When the agent halts, it will see:
- What it completed
- What remains
- Your feedback: "It's using the old auth pattern, should use the new JWT flow from auth.ts"

You'll then choose: resume with this feedback / restart fresh / abandon
```

## Implementation

When this command is invoked:

1. **Create stop file:**
   ```bash
   touch .mint/stop
   ```

2. **Ask user for feedback:**
   ```
   What's wrong with the current approach?
   What should the agent do differently?
   ```

3. **Save feedback to stop file:**
   ```bash
   echo "<user's feedback>" > .mint/stop
   ```

4. **Report:**
   ```
   Stop signal sent with feedback.

   The agent will halt at its next checkpoint and see your correction.
   You'll then choose how to proceed.
   ```

## When Agent Halts

The orchestrator reads the stop file, then asks:

```
Agent interrupted.

Completed:
  ✅ 001-create-model
  🔄 002-add-endpoint (partial)

Your feedback: "It's using the old auth pattern, should use the new JWT flow from auth.ts"

How do you want to proceed?
1. Resume with feedback — agent continues with your correction in mind
2. Restart spec — discard current progress, rerun with adjusted spec
3. Restart task — discard all progress, replan from scratch
4. Abandon — stop entirely, keep what's committed
```

## Resume with Feedback

If user picks "Resume with feedback":
- Orchestrator re-dispatches the agent
- Agent receives: remaining work + user's feedback as additional context
- Agent adjusts approach based on feedback

This is the fastest correction path — no need to rewrite specs manually.

## Notes

- Agents don't stop instantly — they finish their current atomic operation first
- Feedback is preserved even if you close the chat (it's in the file)
- To cancel before agent sees it: `rm .mint/stop`
