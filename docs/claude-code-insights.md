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

**1. ~~Auto-Background Threshold~~ → Tiered Foreground/Background Dispatch (DONE)**

Claude Code agents run in foreground for 2 seconds, then auto-background if still running.
Since Claude Code's Agent tool doesn't support auto-switching at runtime, mint uses a
**static tiered dispatch model** instead — each phase file declares its dispatch tier upfront.

```
# Old behavior
Agent spawned → always background → user waits for notification

# Implemented behavior
Fast agents (spec reviewer S1, documenter, verifier) → foreground → immediate result
Slow agents (planner, decomposer, S2 reviewers, shipper, etc.) → background → user free
```

Tier table in `reference/orchestrator-laws.md`. Each phase file annotates its dispatch tier.

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

**Implemented: Static/dynamic prompt split**

Agent `.md` files are already the static system prompt (cached by API). Dynamic context is
passed via the `prompt` parameter. To formalize this and prevent instruction duplication:

```
agents/planner.md              → static system prompt (cached by API)
templates/agent-context.md     → structured templates for ALL agents' dynamic prompts

# At dispatch time:
Agent tool:
  subagent_type: "mint:mint-planner"       ← loads static agents/planner.md
  prompt: "<spec>...</spec><config>..."    ← dynamic from templates/agent-context.md
```

What was added:
- `templates/agent-context.md` — structured XML templates for every agent's dynamic prompt
- Phase files reference the template: "Build prompt from templates/agent-context.md → section"
- SKILL.md documents the static/dynamic split with the "never duplicate" rule
- Architecture docs describe the caching model and savings math

Estimated savings: 4-spec wave × 2000 token static prompt = 8000 tokens. With caching,
pay full price once + cache read 3× (~500). Net saving: ~5500 tokens per wave.

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

**Implemented: `mint dream` — learning consolidation**

Built as agent + CLI + skill mode. Complementary to Claude Code's autoDream — handles
project-specific learning data, not generic memory.

**What was built:**

- `agents/dream-consolidator.md` — full agent (lock, triage, decay, promote, archive, report)
- `cli/commands/dream.js` — CLI subcommands (status, decay, instincts)
- `skills/mint/modes/dream.md` — mode file with auto-trigger detection
- `templates/agent-context.md` — context template for dream consolidator
- `tests/dream.test.js` — tests for issue triage, instinct lifecycle, win archival, metrics, locking

**What it does:**

1. **Issue triage** — resolve, deduplicate, escalate recurring (3+) to hard-blocks
2. **Instinct decay** — reduce confidence for stale (30d), remove at 0
3. **Pattern promotion** — flag candidates (confidence ≥7, occurrences ≥10), human reviews
4. **Win archival** — keep 50 active, archive rest
5. **Health report** — `.mint/dream-report.md` with pipeline health, trends, suggestions

**Trigger conditions:**
- Manual: `mint dream` CLI or tell Claude `"dream"` / `"consolidate learning"`
- Auto-suggest: 7+ days since last dream AND 10+ new entries → suggested on resume/plan
- Lock: `.mint/.dream-lock` prevents concurrent runs (1h stale timeout)

**Why complementary to autoDream:**
Claude's dream → generic conversation memory consolidation (when released).
Mint's dream → project-specific JSONL learning data. Different data, different location,
different purpose. When autoDream ships, it can trigger `mint dream` as part of its cycle.

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
| 1 | ~~Dream consolidation (`mint dream`)~~ | ~~Medium~~ | **DONE** — Agent + CLI + mode + tests. Complementary to Claude autoDream. |
| 2 | ~~Auto-background threshold for agents~~ | ~~Small~~ | **DONE** — Tiered dispatch: fast agents foreground, slow agents background |
| 3 | ~~Prompt caching (static/dynamic split)~~ | ~~Small~~ | **DONE** — `templates/agent-context.md` + phase files reference templates + architecture docs |
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
