# Eval: <feature-name>

## Capability Evals

Define what the implementation must be able to do. Each eval is a testable assertion.

| ID | Description | Grader | Target |
|----|-------------|--------|--------|
| cap-1 | <what it should do> | code / model / human | pass@3 >= 90% |
| cap-2 | <what it should do> | code / model / human | pass@3 >= 90% |

### Grader Types

- **code** — deterministic: run a command, check exit code or output
- **model** — LLM-as-judge: evaluate output quality with a rubric
- **human** — manual review required (flag in report)

## Regression Evals

Ensure changes don't break existing functionality.

| ID | Description | Baseline | Target |
|----|-------------|----------|--------|
| reg-1 | <existing feature still works> | <baseline SHA or test name> | pass^3 = 100% |
| reg-2 | <existing feature still works> | <baseline SHA or test name> | pass^3 = 100% |

## Metrics

- **pass@k** — at least one success in k attempts (practical reliability)
- **pass^k** — all k attempts succeed (stability for critical paths)

Recommended thresholds:
- Capability evals: pass@3 >= 90%
- Regression evals: pass^3 = 100% for release-critical paths

## Run History

| Date | Capability | Regression | Notes |
|------|-----------|------------|-------|
| YYYY-MM-DD | N/N passed (pass@1: X%) | N/N passed | Initial run |

## Status

[ ] DEFINING — evals being written
[ ] BASELINE — initial run captured
[ ] PASSING — all evals green
[ ] SHIP IT — ready for review
