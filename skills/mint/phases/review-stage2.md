# Phase: Audit (Stage 2)

Parallel dispatch of enabled reviewers, scaled by diff size.

---

## 1. Measure Diff

Run `git diff --stat HEAD~1` and count total lines changed.

## 2. Scale Review Intensity

| Diff size | Level | What runs |
|-----------|-------|-----------|
| < 30 lines | Light | Skip stage 2 entirely (stage 1 is enough) |
| 30-100 lines | Standard | quality + conventions |
| 100-300 lines | Full | All enabled reviewers |
| 300+ lines | Deep | All enabled + opus for security & quality |

Override: `config.reviewScaling: false` = always full review.

## 3. Build Dispatch List

Read `config.reviewers` — for each enabled key:
- `quality` → `mint-quality-reviewer`
- `security` → `mint-security-auditor`
- `conventions` → `mint-conventions-enforcer`
- `tests` → `mint-test-auditor`
- `performance` → `mint-performance-reviewer`
- `business` → `mint-business-reviewer`
- If `config.design.enabled` → `mint-design-reviewer`

For deep diffs: override model to `"opus"` for security and quality.

## 4. Dispatch and Collect

Output status FIRST: "Dispatching stage 2 reviewers: quality, security, conventions..."
Then dispatch ALL simultaneously (parallel Agent calls).

Wait for ALL to return. Parse severity counts:
- Count BLOCKING, WARNING, INFO across all reports
- Record each verdict in `execution.json` → `reviews.<key>`

## Output

"Stage 2: quality ✅ security ✅ conventions ⚠️ (2 warnings). No blockers."
Or list blockers if any, then set pipeline-state to `fix-blockings`.
