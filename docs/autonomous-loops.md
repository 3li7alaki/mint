# Autonomous Loop Patterns

Patterns for running mint non-interactively — CI pipelines, scripted workflows, continuous development.

## Sequential Pipeline

The simplest pattern. Chain `claude -p` calls for focused steps:

```bash
#!/bin/bash
set -e

# Step 1: Implement
claude -p "Read the spec in docs/feature-spec.md. Implement with TDD. Use mint skill."

# Step 2: De-sloppify
claude -p "Review changes in working tree. Remove test/code slop. Run tests after cleanup."

# Step 3: Verify
claude -p "Run build + lint + types + tests. Fix failures. Don't add features."

# Step 4: Commit
claude -p "Create conventional commit for staged changes."
```

### Key principles:
- Each step gets fresh context (no bleed between steps)
- Order matters — each builds on filesystem state from previous
- `set -e` stops pipeline on failure
- Negative instructions degrade quality — use separate cleanup steps instead

### Model routing:
```bash
claude -p --model opus "Analyze architecture and write plan..."   # Deep reasoning
claude -p "Implement according to plan..."                         # Standard execution
claude -p --model opus "Review changes for security/edge cases..." # Thorough review
```

## Continuous PR Loop

For multi-iteration projects with CI gates:

```
1. Create branch
2. Run claude -p with task prompt
3. Commit changes
4. Push + create PR
5. Wait for CI checks
6. CI failure? → auto-fix pass
7. Merge PR
8. Return to main → repeat
```

### Cross-iteration context

Use a shared notes file to bridge context between iterations:

```markdown
<!-- SHARED_TASK_NOTES.md -->
## Progress
- [x] Added auth module (iteration 1)
- [x] Fixed token refresh (iteration 2)
- [ ] Rate limiting tests needed

## Next Steps
- Focus on rate limiting module
- Reuse mock setup in tests/helpers.ts
```

Claude reads this at iteration start and updates at end.

### Exit conditions

Always set limits: `--max-runs N`, `--max-cost $X`, `--max-duration 2h`.

## De-Sloppify Pattern

Add a dedicated cleanup step after every implementation step:

```bash
# Let agent be thorough
claude -p "Implement feature with full TDD. Be thorough."

# Clean up in fresh context
claude -p "Review changes. Remove: language/framework tests, over-defensive checks,
console.log, commented code. Keep business logic tests. Run tests after cleanup."
```

**Why separate steps?** Negative instructions ("don't write sloppy tests") make agents hesitant
about ALL testing. Two focused agents > one constrained agent.

## Integration with mint

### Ship mode as autonomous pipeline
mint's ship mode already supports batch execution with gates between tasks.
For CI use, configure `autoCommit: true` and use the ship plan as a pipeline spec.

### Worktree isolation
Each task runs in an isolated worktree by default (plan and ship modes).
This enables parallel execution without conflicts.

### Quality gates as CI steps
Use `mint verify` as a CI step to run all gates + mock audit + hard block scan.

## Anti-Patterns

1. **No exit conditions** — always set max-runs, max-cost, or max-duration
2. **No context bridge** — use SHARED_TASK_NOTES.md between iterations
3. **Retrying same failure** — capture error context and feed to next attempt
4. **Negative instructions** — use cleanup passes instead
5. **Single context for everything** — separate planning, implementation, and review
