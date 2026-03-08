# Agent Harness Design

Guidance for optimizing how mint agents plan, execute, recover from errors, and converge
on completion. Based on agent harness construction principles.

## Core Model

Agent output quality is constrained by four dimensions:

1. **Action space quality** — are the right tools available with clear schemas?
2. **Observation quality** — does tool output give the agent what it needs to decide next?
3. **Recovery quality** — can the agent diagnose and recover from errors?
4. **Context budget quality** — is the agent working with focused, relevant context?

## Action Space Design

- Use stable, explicit tool names
- Keep input schemas narrow and well-typed
- Return deterministic output shapes
- Avoid catch-all tools — one tool, one job

### Granularity Rules

| Risk level | Tool granularity | Examples |
|-----------|-----------------|---------|
| High (deploy, migrate, permissions) | Micro-tools | `deploy-to-staging`, `run-migration` |
| Medium (edit, read, search) | Standard tools | `Read`, `Edit`, `Grep` |
| Low (format, lint) | Can be hooks | PostToolUse auto-format |

## Observation Design

Every tool response should enable the next decision:

```
status: success | warning | error
summary: one-line result
next_actions: what the agent should do next
artifacts: file paths, IDs, or references created
```

## Error Recovery

For every error path, provide:
- **Root cause hint** — why did this fail?
- **Safe retry instruction** — how to try again safely
- **Explicit stop condition** — when to give up

mint's spec retry protocol implements this: diagnose root cause category → rewrite spec → retry once → escalate.

## Context Budgeting

1. **Keep system prompts minimal** — the orchestrator stays light
2. **Load skills on demand** — don't frontload all knowledge
3. **Reference files, don't inline** — pass file paths, not file contents
4. **Compact at phase boundaries** — between specs, not mid-implementation
5. **Fresh context per agent** — no memory from previous agents

### Context budget rules of thumb

- System prompt: <2000 tokens
- Spec XML: <1000 tokens per spec
- Context paste: <500 tokens of relevant code snippets
- Reserve 80%+ of context window for agent's own reasoning and tool output

## Model Routing

Different tasks benefit from different models:

| Task type | Recommended model | Why |
|----------|-------------------|-----|
| Architecture, planning | Opus | Deep reasoning, long-range coherence |
| Implementation | Sonnet | Fast, capable, cost-effective |
| Quick fixes, formatting | Haiku | Speed, low cost |
| Security review | Opus | Thoroughness critical |
| Convention checking | Haiku | Pattern matching, low complexity |

mint supports per-reviewer model configuration in `config.reviewers`.

## Benchmarking

Track these metrics to improve agent performance:

- **Completion rate** — % of specs that pass on first attempt
- **Retries per task** — lower is better (target: <0.3 retries/spec)
- **pass@1** — first attempt success rate
- **Cost per task** — use cost-tracker hook data
- **Time per spec** — wall clock from dispatch to commit

## Anti-Patterns

- Too many tools with overlapping semantics (agents get confused)
- Opaque tool output with no recovery hints
- Error output without suggested next steps
- Context overloading with irrelevant references
- Using Opus for everything (expensive, often unnecessary)
- Retrying without diagnosing (same input → same output)
