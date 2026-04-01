# Research Mode

For investigating problems before building.

---

## Process

1. Delegate to `mint-researcher` subagent with the question
   **Dispatch tier: background** (`run_in_background: true`) — research is slow
2. Researcher: scans codebase, searches web, compares options
3. Returns structured report saved to `.mint/research/<topic>.md`
4. Optionally suggests a plan task to run next
5. Delete session state on completion
