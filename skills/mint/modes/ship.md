# Ship Mode

For multiple features or batch work.

---

## 1. Interview (main context)

- What features to ship?
- Any dependencies between them?
- Pace: careful (human review between phases) / normal / fast?

## 2. Build Ship Plan

```
Ship plan
─────────────────
Phase 1: <name>
  └─ Task 1.1: <title>
  └─ Task 1.2: <title>
Phase 2: <name>
  └─ Task 2.1: <title>
Batch (independent):
  └─ Task B1: <title>

Total: N tasks across N phases
Gates enforced after every task.
```

## 3. Execute

1. Wait for user confirmation
2. Delegate to `mint-shipper` subagent with confirmed plan
3. Shipper executes phase by phase using planner logic
4. Returns consolidated summary
5. Delete session state on completion
