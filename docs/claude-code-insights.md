# Claude Code Insights for Mint

Findings from analyzing Claude Code's source (~1,931 TypeScript files) that reveal how Anthropic builds their internal orchestration, agent system, and quality infrastructure. Actionable opportunities for mint to get ahead.

---

## 1. Agent Architecture Comparison

### Claude Code's Agent System

Claude Code has 5 built-in agent types, each spawned via `AgentTool`:

| Agent Type | Purpose | Isolation |
|---|---|---|
| `GENERAL_PURPOSE_AGENT` | Catch-all for complex tasks | Optional worktree |
| `EXPLORE_AGENT` | Read-only codebase exploration | None (read-only tools) |
| `VERIFICATION_AGENT_TYPE` | Adversarial verification of implementations | Worktree |
| `PLAN_AGENT` | Plan mode execution | None |
| `FORK_AGENT` | Background fork with isolated context | Worktree or remote |

Each agent gets:
- `buildEffectiveSystemPrompt()` — custom system prompt per type
- Agent-specific MCP servers (connected at spawn, cleaned at exit)
- Progress tracking via `createProgressTracker()` with live UI updates
- Auto-backgrounding after 2s threshold (`PROGRESS_THRESHOLD_MS`)

### What Mint Already Has

Mint has 27 agents (decomposer, planner, 6 reviewers, documenter, researcher, etc.) — more specialized than Claude Code. But Claude Code's execution model has some patterns mint should adopt.

### Gaps to Fill

**1. Auto-Background Threshold**

Claude Code agents run in foreground for 2 seconds, then auto-background if still running. The user can keep working while the agent finishes.

Mint runs agents via `run_in_background: true` explicitly. Adding an auto-background threshold would let quick agents (lint check, small edit) stay foreground while long agents (full implementation, research) background automatically.

```
# Current mint behavior
Agent spawned → always background → user polls status

# Proposed behavior
Agent spawned → foreground 3s → still running? → auto-background → notify on complete
```

**2. Adversarial Verification Agent**

Claude Code has a `VERIFICATION_AGENT_TYPE` that independently verifies implementations — not just reviewing code, but trying to break it. This runs in a separate worktree so it can't modify the implementation.

Mint has spec-reviewer (stage 1) and parallel auditors (stage 2), but they're review-focused, not adversarial. Adding a "red team" agent that:
- Reads the spec and implementation
- Tries to construct inputs that violate acceptance criteria
- Runs edge-case tests against the implementation
- Reports failures as BLOCKING findings

This slots in as an optional stage 2 auditor: `mint-adversarial-tester`.

**3. Fork Agent (Speculative Execution)**

Claude Code's `FORK_AGENT` creates a background fork that inherits the parent's prompt cache. This enables speculative work — start implementing while the user is still reviewing the plan.

Mint could use this for parallel spec attempts: if a spec fails on first try, fork a second attempt with different approach while the first is being debugged.

---

## 2. Swarm System — Visual Multi-Agent

### How Claude Code Does It

The swarm system (`src/utils/swarm/`) spawns parallel agents as tmux panes:

- **Backends**: Tmux (primary), iTerm2, InProcess
- **Team lead**: Central coordinator managing teammates
- **Session isolation**: Separate tmux socket per PID
- **Permission sync**: `leaderPermissionBridge` — teammates ask leader for permission
- **Color coding**: `TEAMMATE_COLOR_ENV_VAR` for visual identification
- **Reconnection**: Auto-recovery for dropped connections

### What Mint Can Learn

Mint's wave-based parallelism uses worktrees (isolated git branches). This works but is invisible — users can't see agents working.

**Proposal: `--visual` mode for wave execution**

```bash
# Current: invisible worktree execution
mint ship "build auth module"

# Proposed: tmux pane visualization
mint ship "build auth module" --visual

# Opens tmux layout:
# ┌──────────────────┬──────────────────┐
# │ [spec-001] auth  │ [spec-002] types │  ← Wave 1 (parallel)
# │ Running planner  │ Running planner  │
# │ > editing auth.ts│ > editing types/ │
# ├──────────────────┴──────────────────┤
# │ [orchestrator] Wave 1: 2/2 specs    │  ← Status bar
# │ Next: Wave 2 (spec-003, spec-004)   │
# └─────────────────────────────────────┘
```

Implementation:
- Use tmux backend from Claude Code's swarm as reference
- Each pane runs `claude -p "<spec prompt>"` in its worktree
- Orchestrator pane shows wave progress and gate results
- Color-code panes by spec status (green=passing, yellow=running, red=failing)

---

## 3. System Prompt Caching Architecture

### How Claude Code Does It

The system prompt is split at a `SYSTEM_PROMPT_DYNAMIC_BOUNDARY`:

```
[Static prefix — cached cross-org, scope: 'global']
  Identity, coding guidelines, tool descriptions, safety rules
  
--- DYNAMIC BOUNDARY ---

[Dynamic suffix — session-specific, recomputes per turn]
  Memory, MCP instructions, language preference, output style
```

Sections use a registry pattern:
- `systemPromptSection()` — standard cached sections (persist until /clear or /compact)
- `DANGEROUS_uncachedSystemPromptSection()` — volatile sections that bust cache on change

### Application to Mint

Mint agent prompts (27 agents in `/agents/`) are loaded as full markdown files. Each agent gets the same static instructions every time, plus dynamic context (spec XML, learning JSONL, config).

**Proposal: Split agent prompts into static + dynamic**

```
agents/
  planner.md          → static instructions (cached by API)
  planner.context.md  → template for dynamic context injection

# At dispatch time:
prompt = cache(read('agents/planner.md')) + render('agents/planner.context.md', {
  spec: currentSpec,
  issues: recentIssues,
  instincts: topInstincts,
  config: projectConfig,
})
```

Benefits:
- Static part (agent identity, rules, patterns) gets Anthropic's prompt caching discount
- Dynamic part (spec, learning context) changes per dispatch but is small
- Multiple specs in the same wave share the cached static prefix

**Estimated savings**: If 4 specs run in a wave, the static agent prompt (~2000 tokens) is cached 4x instead of sent 4x. With Anthropic's cache pricing, this reduces input token cost by ~75% for the static portion.

---

## 4. Dream Consolidation (autoDream)

### How Claude Code Does It

`autoDream` is a background memory consolidation system:

- **Trigger**: 24 hours elapsed + 5 sessions completed + no concurrent consolidation
- **Process**: Spawns a forked subagent that reads all session memories, prunes duplicates, merges contradictions, updates indexes
- **Safety**: Read-only bash, process lock to prevent concurrent runs, rollback on crash
- **Output**: Cleaned, indexed, deduplicated memory files

### Application to Mint: `mint dream`

Mint accumulates learning data in JSONL files (issues, wins, instincts, patterns, metrics). Over time these grow stale — old issues get fixed, instincts drift, patterns become conventions.

**Proposal: `mint dream` — learning consolidation**

Runs between milestones or on schedule:

```
mint dream
```

What it does:

1. **Issue triage**: Scan `issues.jsonl` — mark resolved issues (fix is in git history), merge duplicates, escalate recurring issues to `hard-blocks.md`
2. **Instinct pruning**: Remove instincts with confidence < threshold that haven't been reinforced in 30 days
3. **Pattern promotion**: Instincts with confidence > 0.9 get promoted to `conventions.md` (documented patterns)
4. **Win archival**: Move old wins to `wins-archive.jsonl`, keep recent 50 active
5. **Metrics summary**: Generate `.mint/health-report.md` with trends (gate pass rate, avg fix cycles, spec accuracy)
6. **Config suggestions**: If metrics show patterns (e.g., security reviewer catches things 80% of the time), suggest enabling it by default

**Trigger conditions** (same pattern as autoDream):
- Time: 7 days since last dream (or manual invoke)
- Volume: 10+ new JSONL entries since last dream
- Lock: `.mint/.dream-lock` file prevents concurrent runs
- Auto-trigger: After `mint complete-milestone` or on `mint resume-work` if stale

**Agent prompt sketch**:
```markdown
You are the mint dream consolidator. Your job is to review accumulated
learning data and produce a cleaner, more useful knowledge base.

Rules:
- Never delete information — archive or merge
- Promote high-confidence instincts to documented conventions
- Escalate recurring issues (3+ occurrences) to hard-blocks
- Keep the active working set small (50 wins, 100 instincts, 200 issues max)
- Generate a health report summarizing trends
```

---

## 5. Permission & Classifier System

### How Claude Code Does It

Four-layer permission system:

```
1. validateInput()     — schema validation (fast, sync)
2. checkPermissions()  — tool-specific security checks
3. Hook execution      — pre_tool_use / post_tool_use scripts
4. Classifier          — async ML-based auto-approval running in parallel with UI prompt
```

The classifier is the interesting part: while the user sees "Allow this?" dialog, a background classifier analyzes the command and may auto-approve it before the user responds.

Pattern matching supports globs: `Bash(git *)`, `FileEdit(*.ts)`, `Bash(npm test *)`.

### Application to Mint

Mint's hooks (pre-edit, post-edit) are layer 3. Mint doesn't have layers 1-2 or 4.

**Proposal: Confidence-based gate skipping**

Not all edits need full gate runs. If the edit is:
- Formatting-only (prettier/eslint autofix)
- Comment/docstring changes
- Test file additions (not modifications)
- Import reordering

...then gates can run in "quick" mode (types only, skip full test suite).

Implementation:
```jsonc
// .mint/config.json
{
  "gates": {
    "quickPatterns": [
      "*.test.ts:new",       // new test files → quick gates
      "*.md",                // docs → skip gates
      "**/*.css",            // styles → lint only
    ],
    "fullPatterns": [
      "src/core/**",         // core code → always full gates
      "*.config.*",          // config files → always full gates
    ]
  }
}
```

---

## 6. Tool Search / Deferred Loading

### How Claude Code Does It

With 60+ tools, Claude Code doesn't load all tool schemas into the system prompt. Instead:

- Tool names are listed in a system reminder (lightweight)
- `ToolSearch` tool fetches full schemas on demand
- LLM sees tool names, decides which to use, fetches schema, then calls

This keeps the system prompt small while supporting massive tool catalogs.

### Application to Mint

Mint loads agent prompts (27 files) and phase instructions (7 files) on demand — already good. But the orchestrator SKILL.md router (155 lines) references all modes.

**Implemented: Two-tier mode loading**

SKILL.md split into two tiers:

```
Tier 1 (always loaded — SKILL.md, ~125 lines):
  - Routing table, session state, universal rules (3 lines)
  - Reference file index
  - Agent context table

Tier 2 (loaded on demand — reference/orchestrator-laws.md, 90 lines):
  - Context protection, status format, background dispatch
  - Quality gates, autocommit resolution, repo mode
  - Only loaded for code-modifying modes (quick, plan, ship, design)
  - Skipped for lightweight modes (research, verify, browse, ssh)
```

Savings: 182 → 126 lines in always-loaded context (~2KB / ~700 tokens).
Lightweight mode invocations skip ~90 lines of irrelevant orchestrator rules.

---

## 7. Owl Post Integration (Cross-Project)

### Mint Agents as Owl Post Participants

Claude Code's swarm has a `leaderPermissionBridge` for agent communication. Mint could use Owl Post channels for the same purpose, but visible to the whole team.

**Proposal: `mint ship --notify #channel`**

```bash
mint ship "build auth module" --notify "#backend-team"
```

Behavior:
- On spec start: `owl send #backend-team "[mint] Starting spec-001: auth types"`
- On gate pass: `owl send #backend-team "[mint] spec-001: gates passed, committing"`  
- On review finding: `owl send #backend-team "[mint] spec-001: BLOCKING — security reviewer found SQL injection in auth.ts:42"`
- On completion: `owl send #backend-team "[mint] auth module complete. 4 specs, 3 waves, 12 min. PR ready."`

Implementation:
- Add `owl_send` MCP tool calls in orchestrator phases
- Config: `.mint/config.json` → `notifications.channel`, `notifications.server`
- Respect quiet hours: don't notify for INFO-level findings

---

## 8. Metrics & Observability (from Claude Code Analytics)

### How Claude Code Does It

Extensive analytics system:
- `firstPartyEventLogger` — structured event logging
- `datadog` integration — APM and metrics
- `growthbook` — feature flags and A/B testing
- `diagnosticTracking` — performance diagnostics
- Session-level tracking with `x-claude-code-session-id`

### Application to Mint

Mint has `metrics.jsonl` but it's append-only with no visualization.

**Proposal: `mint stats` enhancement**

```bash
mint stats

# Output:
Pipeline Health (last 30 days)
  Specs executed:     47
  Gate pass rate:     89% (↑ from 82% last month)
  First-try success:  71%
  Avg fix cycles:     1.3
  
Reviewer Value
  spec-reviewer:      12 BLOCKING caught
  security-auditor:   4 BLOCKING caught  
  quality-reviewer:   8 WARNING (2 promoted to BLOCKING)
  conventions:        3 WARNING
  
Top Issues (recurring)
  1. Missing null checks in API handlers (5 occurrences)
  2. Import order violations (4 occurrences)  
  3. Test mocking instead of real DB (3 occurrences)
  
Instinct Health
  Active instincts:   34
  High confidence:    12 (candidates for conventions promotion)
  Stale (30d+):       8 (candidates for pruning)
```

---

## 9. Implementation Priority

| Priority | Feature | Effort | Impact for Mint |
|----------|---------|--------|----------------|
| 1 | Dream consolidation (`mint dream`) | Medium | Keeps learning data useful, prevents JSONL bloat |
| 2 | Auto-background threshold for agents | Small | Better UX — quick agents stay foreground |
| 3 | Prompt caching (static/dynamic split) | Small | Cost reduction, faster agent startup |
| 4 | Adversarial verification agent | Medium | Catches bugs that reviewers miss |
| 5 | Visual wave execution (`--visual`) | Large | Team visibility, debugging aid |
| 6 | Confidence-based gate skipping | Medium | Faster pipeline for safe edits |
| 7 | Owl Post notifications | Small | Team awareness of mint activity |
| 8 | Enhanced `mint stats` | Medium | Data-driven pipeline improvement |
| 9 | ~~Two-tier mode loading~~ | ~~Small~~ | **DONE** — SKILL.md 182→126 lines, orchestrator laws deferred to reference file |

---

## Reference Files (Claude Code Source)

Key files studied for these insights:

**Agent System:**
- `src/tools/AgentTool/AgentTool.tsx` — Agent spawning, types, isolation modes
- `src/tools/AgentTool/runAgent.ts` — Agent execution engine
- `src/tools/AgentTool/built-in/` — Built-in agent type definitions
- `src/tasks/LocalAgentTask/LocalAgentTask.tsx` — Lifecycle, auto-backgrounding

**Swarm:**
- `src/utils/swarm/backends/TmuxBackend.ts` — Tmux pane management
- `src/utils/swarm/permissionSync.ts` — Cross-agent permission bridging
- `src/utils/swarm/spawnUtils.ts` — Agent spawning utilities

**System Prompt:**
- `src/constants/prompts.ts` — Prompt construction, caching boundaries
- `src/constants/systemPromptSections.ts` — Section registry pattern

**Quality & Permissions:**
- `src/hooks/toolPermission/PermissionContext.ts` — 4-layer permission system
- `src/hooks/toolPermission/handlers/interactiveHandler.ts` — Classifier auto-approval

**Learning & Memory:**
- `src/services/autoDream/` — Background consolidation trigger logic
- `src/services/SessionMemory/` — Session memory management
- `src/services/extractMemories/` — Memory extraction from conversations

**Analytics:**
- `src/services/analytics/` — Event logging, Datadog, GrowthBook
- `src/services/diagnosticTracking.ts` — Performance tracking
