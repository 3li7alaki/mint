# Plugin Authoring Guide

How to build high-quality mint plugins.

## What a Plugin Is

A plugin is a directory with a `manifest.json` and optional agents/commands that extend mint's pipeline. Plugins are static files — no runtime, no build step, no package manager.

## Directory Structure

```
mint-yourname/
├── manifest.json           # Required — plugin identity and hooks
├── agents/                 # Agent prompts dispatched by the orchestrator
│   └── your-reviewer.md
├── commands/               # User-invocable commands (optional)
│   └── your-command.md
└── README.md               # Install and usage docs
```

## Manifest Schema

```json
{
  "name": "mint-yourname",
  "version": "0.1.0",
  "description": "What this plugin adds to mint",
  "type": "stack | pm | design | memory",
  "agents": [
    "agents/your-reviewer.md"
  ],
  "commands": [],
  "config": {
    "yourKey": "default-value"
  },
  "hooks": {
    "pre-plan": null,
    "post-plan": null,
    "pre-review": "agents/your-reviewer.md",
    "post-commit": null,
    "on-init": null
  }
}
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Plugin name. Convention: `mint-{name}` |
| `version` | Yes | Semver version string |
| `description` | Yes | One-line description |
| `type` | Yes | Plugin category (see types below) |
| `agents` | Yes | Array of agent file paths relative to plugin dir |
| `commands` | No | Array of command file paths relative to plugin dir |
| `config` | No | Config keys this plugin adds (with defaults) |
| `hooks` | No | Which pipeline stages to hook into |

### Plugin Types

| Type | Purpose | Example |
|------|---------|---------|
| `stack` | Framework-specific conventions and review | mint-nuxt, mint-react, mint-django |
| `pm` | Project management tool integration | mint-linear, mint-jira |
| `design` | Design tool integration | mint-figma |
| `memory` | Knowledge persistence and retrieval | mint-embeddings |

### Hook Points

| Hook | When | Receives | Good For |
|------|------|----------|----------|
| `pre-plan` | Before planner decomposes | Feature description + config | Fetching external context (ticket details, design specs) |
| `post-plan` | After specs are created | List of spec files | Syncing specs to external tools |
| `pre-review` | Stage 2 parallel audit | Git diff + config | Framework/domain-specific review |
| `post-commit` | After each atomic commit | Commit hash + spec + config | Updating external status (ticket progress) |
| `on-init` | During `mint init` | Project config | Setting up plugin-specific config |

A hook value is an agent file path (relative to plugin dir) or `null` (not used).

## Writing Plugin Agents

Plugin agents follow the same conventions as core agents. Every agent needs:

### 1. Role Statement

Start with who the agent is:

```markdown
# mint-yourname: Your Agent Name

You are the **mint-yourname agent** — [one line about what you do].
```

### 2. What You Receive

Be explicit about inputs:

```markdown
## What You Receive

- Git diff of the implemented changes
- Project config (`.mint/config.json`)
- [Any hook-specific context]
```

### 3. What You Do

Step-by-step process. Be specific — agents don't infer:

```markdown
## What You Do

1. Read the diff
2. Check for [specific patterns]
3. Classify findings by severity
```

### 4. What You Return

Exact output format:

```markdown
## What You Return

[Format template with BLOCKING/WARNING/INFO sections]
```

### 5. Tool List

Explicitly declare tools needed:

```markdown
**Tools you need:** Read, Glob, Grep
```

### 6. Rules

Non-negotiable constraints:

```markdown
## Rules

- Only check [your domain] — don't duplicate core reviewer checks
- Focus on the diff — don't audit the entire codebase
- If unsure, classify as INFO
```

## Severity Levels

All reviewer agents use three levels:

| Level | Meaning | What Happens |
|-------|---------|-------------|
| **BLOCKING** | Must fix before commit | Planner fixes, reviewer re-reviews |
| **WARNING** | Should fix | Planner fixes in same pass |
| **INFO** | Noted for learning | Logged to `.mint/issues.md` |

## Writing Good Plugins

### Do

- **One plugin, one concern.** A Nuxt plugin checks Nuxt conventions. It doesn't also check CSS or testing patterns.
- **Complement, don't duplicate.** Core agents handle general quality, security, tests, conventions. Your plugin adds domain-specific knowledge they don't have.
- **Be specific.** "Check file structure" is vague. "Pages go in `pages/`, composables go in `composables/`" is actionable.
- **Provide context for findings.** Don't just say "wrong" — explain what's right and why.
- **Include a README.** Install instructions, what it checks, config options.
- **Use the config object.** If your plugin needs settings (e.g., framework version), put defaults in manifest `config` and read from project config at runtime.
- **Test against a real project.** Install the plugin in a project that uses the framework. Does it catch real issues? Does it produce false positives?

### Don't

- **Don't add runtime dependencies.** Plugins are markdown files. No npm, no pip, no cargo.
- **Don't modify core files.** Plugins extend the pipeline — they don't change SKILL.md or core agents.
- **Don't make agents too broad.** An agent that checks 15 different categories is doing too much. Split into multiple agents if needed.
- **Don't hardcode project paths.** Use config keys and let the project set values.
- **Don't assume tool availability.** List every tool your agent needs explicitly.
- **Don't add unnecessary config.** If a value never changes, don't make it configurable.

## Plugin Types — Detailed

### Stack Plugins

Framework-specific conventions and review.

**Hook:** Usually `pre-review` (adds a reviewer to stage 2).

**Agent pattern:** Read the diff, check framework conventions, return severity-tagged findings.

**Examples:**
- File structure (pages, components, composables in right dirs)
- Auto-import patterns (don't manually import what's auto-imported)
- Framework-specific anti-patterns (wrong data fetching method, client/server boundary violations)
- Config conventions (runtime vs build-time settings)

**Reference:** `plugins/mint-nuxt/`

### PM Plugins

Project management tool integration.

**Hooks:** `pre-plan` (fetch ticket context), `post-commit` (update ticket status), `on-init` (detect PM config).

**Agent pattern:** Use MCP tools or APIs to read/write external state. Bring ticket requirements into the spec pipeline. Push progress updates back.

**What they provide:**
- Ticket description and acceptance criteria as spec context
- Automatic status updates when specs complete
- Link commits to tickets

### Design Plugins

Design tool integration.

**Hooks:** `pre-plan` (fetch design specs), `pre-review` (check against design).

**Agent pattern:** Fetch design data (tokens, components, specs) and make it available as review context. Check implementations against design requirements.

**What they provide:**
- Design tokens and component specs as planner context
- Visual fidelity checks in review
- Design-to-code alignment reports

### Memory Plugins

Knowledge persistence and retrieval.

**Hooks:** `post-commit` (save knowledge), `pre-plan` (retrieve relevant context).

**Agent pattern:** After commits, extract and store knowledge (embeddings, summaries). Before planning, retrieve relevant past decisions and patterns.

**What they provide:**
- Architectural decision recall
- Pattern recognition across sessions
- Relevant past context for new specs

## Publishing

Plugins are git repos. To publish:

1. Create a repo named `mint-yourname`
2. Include `manifest.json`, agents, README
3. Users install with: `git clone <repo> .mint/plugins/mint-yourname`

No registry, no package manager. Just git.

## Checklist

Before publishing a plugin:

- [ ] `manifest.json` is valid JSON with all required fields
- [ ] Plugin `type` is correct (stack/pm/design/memory)
- [ ] All agent files listed in `agents` array exist
- [ ] All command files listed in `commands` array exist
- [ ] Hook values point to existing agent files
- [ ] Agents have: role statement, inputs, process, output format, tool list, rules
- [ ] Agents use BLOCKING/WARNING/INFO severity levels
- [ ] Agents don't duplicate core reviewer checks
- [ ] README has install instructions and describes what the plugin checks
- [ ] Tested against a real project using the framework/tool
