---
description: "Run all quality gates and report results — lint, types, tests, mock audit, hard blocks"
---

# mint verify

Run all quality gates and report status. Does not modify source files.

> **Context rule:** Delegates all work to the `mint-verifier` subagent.
> The main context only receives the gate report.

## Steps

1. Check `.mint/config.json` exists — if not, prompt user to run `mint init`
2. Dispatch `mint-verifier` subagent with the config
3. Present the report to the user

That's it. The verifier agent does all the heavy lifting.
