# Mint Issues & Learnings

_Centralized log. All agent blockers, root causes, and learnings go here._

| Date | Task | Severity | Issue | Root Cause | Resolution | Spec Fix |
|------|------|----------|-------|------------|------------|----------|
| 2026-03-04 | ws-002/003 | WARNING | Specs ws-002 and ws-003 executed sequentially despite only sharing ws-001 as dependency — should have been parallel | missing-context | Orchestrator must check depends-on graph and dispatch independent specs concurrently using parallel Task calls | Add pitfall: "Check dependency graph — specs with same parent but no mutual dependency run in parallel" |
