# How to Write Agent Prompts

Reference standard for writing effective subagent prompts. Used by mint's own agents as the
template, and available to projects that need custom agents.

---

## Core Principle

An agent prompt is a job description, not a manual. Tell the agent what to do, what it
receives, and what to return. Don't explain how LLMs work or how Claude Code operates.

---

## Budget

| Agent type | Max lines | Max rules |
|------------|-----------|-----------|
| Implementer | 200 | 10 |
| Reviewer | 150 | 8 |
| Researcher | 150 | 6 |
| Documenter | 100 | 5 |
| Utility | 80 | 5 |

Compliance drops sharply after ~200 lines. If your prompt is longer, you're encoding two
jobs in one agent — split it.

---

## Structure

Every agent prompt follows this structure. No exceptions.

```markdown
# <Agent Name>

<one-line role statement — what this agent does>

## Inputs

<what the agent receives when dispatched — be explicit>

## Process

1. <step>
2. <step>
...max 10 steps

## Output

<what the agent returns — format, structure, expected content>

## Rules

- <rule 1 — most important>
- <rule 2>
...max 5-10 rules
```

### Why This Order

**Primacy effect** — the first things in the prompt get highest compliance. Put the role
and inputs first so the agent knows what it IS and what it HAS before processing instructions.

**Recency effect** — the last things get second-highest compliance. Put rules last. The
middle (process steps) gets the weakest attention, but numbered steps with clear sequencing
survive better than prose.

**The middle gets lost.** Never put critical rules in the middle of a long process section.
If a rule is critical, put it in Rules (end) or the role statement (beginning).

---

## Writing Rules That Get Followed

### Be Specific

```
BAD:  "Be careful with security"
GOOD: "Use parameterized queries for all database operations"

BAD:  "Write good tests"
GOOD: "Each test must have at least one assertion on the actual business logic output"

BAD:  "Follow conventions"
GOOD: "Use the existing ApiError class from src/errors.ts for all error responses"
```

Specific rules are verifiable. Vague rules are ignorable.

### Fewer > More

5 rules at 90% compliance each = 4.5 rules followed.
15 rules at 60% compliance each = 9 rules followed, but 6 rules violated.

The 5-rule version has fewer violations despite less coverage. Focus on rules that
prevent the most damage.

### Front-Load Critical Rules

```markdown
## Rules

- NEVER modify files outside <can-modify> scope    ← critical, position 1
- Run gates before committing                      ← critical, position 2
- Use existing patterns from the codebase          ← important, position 3
- Prefer composition over inheritance              ← preference, position 4
```

Position 1-2 get ~90% compliance. Position 5+ drops to ~70%.

### Make Rules Testable

Every rule should be checkable by a reviewer or hook:

```
UNTESTABLE: "Write clean code"
TESTABLE:   "No function longer than 50 lines"

UNTESTABLE: "Handle errors properly"
TESTABLE:   "All async functions must have try/catch with specific error types"
```

If you can't verify a rule was followed, it's not a rule — it's a wish.

---

## One Job Per Agent

An agent that implements AND reviews will do neither well. The implementer skips review
steps because it's eager to write code. The reviewer is compromised because it wrote the code.

| Pattern | Compliance |
|---------|-----------|
| Agent does 1 thing | ~85% |
| Agent does 2 things | ~60% per thing |
| Agent does 3+ things | ~40% per thing |

If your agent prompt has two Process sections or two Output formats, split it into two agents.

---

## Tool Listing

Always list required tools explicitly. Never assume availability.

```markdown
## Tools Required

- Read — read source files
- Edit — modify source files
- Bash — run gates (lint, types, tests)
- Grep — search codebase for patterns
```

Agents that assume they have tools they don't will hallucinate tool calls or silently skip steps.

---

## Context Injection

Agents receive context from the orchestrator at dispatch time. Design prompts to use
injected context, not to gather their own.

```markdown
## Inputs

- Spec XML (full text) — the task specification
- Config (.mint/config.json) — project configuration
- Hard blocks (.mint/hard-blocks.md) — immutable constraints
- Git diff — changes to review (reviewers only)
- Learning context — relevant issues/wins/instincts (planner only)
```

The orchestrator decides what context each agent needs. The agent doesn't go looking for it.

**Exception:** Agents CAN and SHOULD read source files referenced in their inputs. "Read the
files listed in `<can-modify>`" is fine. "Grep the entire codebase to understand the project"
is not — that's the orchestrator's job.

---

## Reviewer-Specific Rules

Reviewers use a three-severity system. Every finding must have exactly one severity.

| Severity | Meaning | Action |
|----------|---------|--------|
| BLOCKING | Must fix before merge | Planner re-dispatched to fix |
| WARNING | Should fix, not a blocker | Logged, may be addressed |
| INFO | Suggestion or observation | Logged only |

Reviewer prompts must specify:
- What counts as BLOCKING vs WARNING vs INFO for their domain
- A verdict line: `PASS` (no BLOCKINGs) or `FAIL` (has BLOCKINGs)
- Max findings to report (prevent report bloat — cap at 10)

---

## Anti-Patterns

### The Textbook
```markdown
# Security Auditor

Security is important because vulnerabilities can lead to data breaches,
financial losses, and reputational damage. The OWASP Top 10 categorizes
the most critical web application security risks...
```

The agent already knows what security is. Don't teach — instruct.

**Fix:** Skip the explanation. Start with "You are a security auditor. Check the diff for..."

### The Kitchen Sink
An agent prompt that tries to handle every possible scenario with conditional logic:
"If the project uses React, do X. If Vue, do Y. If the diff is small, skip Z..."

**Fix:** Let the orchestrator handle routing. The agent receives exactly what it needs.

### The Copy-Paste
Duplicating CLAUDE.md content, project conventions, or other agent prompts into this agent.

**Fix:** The orchestrator injects shared context. The agent prompt only contains agent-specific
instructions.

### The Micro-Manager
```markdown
1. Open the file
2. Find the function
3. Read line 42
4. Check if the variable name starts with...
```

Over-specifying low-level steps makes the agent brittle. If file structure changes, every
step breaks.

**Fix:** Describe the goal, not the clicks. "Check that all exported functions have type
annotations" not "Open each .ts file, find export statements, check if..."

### The Silent Agent
An agent that does work but returns no structured output. The orchestrator can't verify
what happened.

**Fix:** Always define Output format. Reviewers return findings with severities. Implementers
return gate results and commit hashes. Researchers return structured reports.

---

## Templates

### Implementer Template

```markdown
# <Name> Implementer

Implement the given spec: read existing code, make changes, run gates, commit.

## Inputs

- Spec XML — task specification with scope, steps, acceptance criteria
- Resolved autocommit — true (commit) or false (stage only)
- Resolved TDD — true (write tests first) or false

## Process

1. Read files listed in <can-modify> to understand existing patterns
2. If TDD: write failing tests first, then implement to pass them
3. Implement changes following spec <steps>
4. Run gates: lint, types, tests
5. If gates fail: fix and rerun (max 2 attempts)
6. If autocommit true: commit with message from <commit>
7. If autocommit false: stage changes, do not commit
8. Update execution.json with gate results and commit hash

## Output

Return concise summary: what was implemented, gate results (pass/fail per gate),
commit hash (or "staged"), any issues encountered.

## Rules

- NEVER modify files outside <can-modify> scope
- NEVER skip gates — all must pass before commit/stage
- Use existing patterns from the codebase (read before writing)
- One commit per spec, message from <commit> field
- If gates fail twice, stop and report — do not attempt a third time
```

### Reviewer Template

```markdown
# <Name> Reviewer

Review the git diff for <domain> issues. Read-only — never modify code.

## Inputs

- Git diff — the changes to review
- Spec XML — what was intended (check diff against intent)

## Process

1. Read the full diff
2. For each file changed, check for <domain-specific checks>
3. Cross-reference with spec acceptance criteria
4. Classify each finding as BLOCKING, WARNING, or INFO
5. Write verdict: PASS (no BLOCKINGs) or FAIL (has BLOCKINGs)

## Output

```
VERDICT: PASS|FAIL

BLOCKING:
- [file:line] description

WARNING:
- [file:line] description

INFO:
- [file:line] description
```

## Rules

- Read-only — never modify files
- Max 10 findings — prioritize by severity
- Every finding must cite file:line
- BLOCKING means "this will break in production or violates a hard constraint"
- When in doubt between WARNING and BLOCKING, use WARNING
```

---

## Maintenance

Agent prompts evolve. Maintain them like code.

- **Track effectiveness** — if a reviewer never catches real issues, its prompt needs work
- **Incorporate instincts** — recurring patterns from .mint/instincts.jsonl should inform
  agent prompts (future: auto-injected via prompt patches)
- **Test against real runs** — after editing a prompt, run it on a known diff and check if
  the output improves
- **Prune ruthlessly** — remove rules the agent follows naturally. Every line costs compliance.
