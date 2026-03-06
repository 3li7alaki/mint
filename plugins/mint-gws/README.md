# mint-gws

Google Workspace integration plugin for [mint](https://github.com/3li7alaki/mint). Multi-step workflows, smart context injection, and setup assistance for the official Google Workspace CLI.

## Install

mint-gws ships with mint. After installing mint (`npx skills add 3li7alaki/mint`), enable it in `.mint/config.json`:

```json
{
  "plugins": ["plugins/mint-gws"],
  "gws": {
    "services": ["drive", "gmail", "calendar", "sheets"],
    "toolMode": "compact"
  }
}
```

## Requirements

**Required:** The Google Workspace CLI (`gws`) must be installed and authenticated:

```bash
# Install
npm install -g @googleworkspace/cli

# One-time setup (creates GCP project, enables APIs, authenticates)
gws auth setup

# Or if you already have a GCP project
gws auth login
```

**MCP Setup:** Add `gws` as an MCP server in your Claude Code settings (`.claude/settings.json`):

```json
{
  "mcpServers": {
    "gws": {
      "command": "gws",
      "args": ["mcp", "-s", "drive,gmail,calendar,sheets", "--tool-mode", "compact"]
    }
  }
}
```

> **Tip:** Use `--tool-mode compact` to reduce tool count. Each service adds 10-80 tools — compact mode exposes one meta-tool per service plus `gws_discover`.

## Setup Wizard

If you haven't set up `gws` yet, use the setup command:

```
/gws setup
```

This walks you through:
1. Installing the CLI (npm or cargo)
2. Running `gws auth setup` for GCP project + OAuth
3. Configuring MCP server in Claude Code settings
4. Testing the connection

## What It Does

### Pre-Plan: Workspace Context

When your feature description references workspace resources (e.g., "sync data from the Q1 Budget spreadsheet" or "send weekly digest emails"), the plugin:

- Identifies referenced resources (spreadsheets, docs, folders, email threads)
- Fetches metadata and structure (sheet names, doc headings, folder contents)
- Provides schema context to the planner (column headers, data types)

This means your specs automatically include accurate workspace context.

### Multi-Step Workflows

The real power — chain workspace operations intelligently:

```
/gws workflow "Create a report from Q1 sales data"
```

The workflow agent:
1. Finds the sales spreadsheet
2. Fetches data and analyzes structure
3. Creates a new doc or spreadsheet for the report
4. Formats and populates it
5. Optionally shares it or sends via email

### Quick Commands

```bash
# List recent files
/gws drive list

# Search for documents
/gws drive search "Q1 Budget"

# Get spreadsheet schema
/gws sheets schema "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"

# Send quick email
/gws gmail send --to team@example.com --subject "Update" --body "..."

# Create calendar event
/gws calendar create "Team Sync" --when "tomorrow 3pm" --duration 30m
```

## Configuration

| Key | Default | Description |
|-----|---------|-------------|
| `gws.services` | `["drive", "gmail", "calendar", "sheets"]` | Services to expose via MCP |
| `gws.toolMode` | `"compact"` | `compact` (fewer tools) or `full` (all tools) |
| `gws.defaultFolder` | `null` | Default Drive folder ID for file operations |
| `gws.emailSignature` | `null` | Signature to append to outgoing emails |

## Agents

| Agent | Hook | Role |
|-------|------|------|
| `setup-wizard.md` | command | Guides through CLI install, auth, and MCP config |
| `workspace-context.md` | pre-plan | Fetches workspace resource context for planner |
| `workflow.md` | command | Executes multi-step workspace operations |

## Commands

| Command | Description |
|---------|-------------|
| `/gws setup` | Interactive setup wizard |
| `/gws workflow "<task>"` | Execute a multi-step workflow |
| `/gws drive <action>` | Drive operations (list, search, download, upload) |
| `/gws sheets <action>` | Sheets operations (list, schema, read, write) |
| `/gws gmail <action>` | Gmail operations (list, search, send, reply) |
| `/gws calendar <action>` | Calendar operations (list, create, update) |

## Example Workflows

### 1. Data Pipeline

```
/gws workflow "Export contacts from CRM sheet to a draft email list"
```

Steps executed:
1. Find "CRM" spreadsheet
2. Read contacts with email column
3. Create draft email with BCC list
4. Report completion

### 2. Meeting Prep

```
/gws workflow "Find all docs related to Project Alpha and create a meeting agenda"
```

Steps executed:
1. Search Drive for "Project Alpha"
2. List recent docs and their key sections
3. Create new agenda doc
4. Link referenced documents
5. Optionally create calendar event

### 3. Report Generation

```
/gws workflow "Create weekly metrics report from Analytics sheet"
```

Steps executed:
1. Find Analytics spreadsheet
2. Read last 7 days of data
3. Calculate summaries (via sheets formulas or local compute)
4. Create formatted report doc
5. Share with stakeholders

## Security Notes

- OAuth tokens are encrypted at rest (AES-256-GCM) in OS keyring
- Service account credentials supported for CI/CD workflows
- Use `gws.services` to limit scope — don't expose Admin API unless needed
- The plugin never stores credentials in `.mint/` — all auth is handled by `gws`

## Troubleshooting

### "gws command not found"

Install the CLI:
```bash
npm install -g @googleworkspace/cli
```

### "Not authenticated"

Run auth setup:
```bash
gws auth setup    # First time
gws auth login    # Subsequent
```

### "MCP tools not available"

Ensure MCP server is configured in `.claude/settings.json` and restart Claude Code.

### "Too many tools"

Use compact mode:
```json
"args": ["mcp", "-s", "drive,gmail", "--tool-mode", "compact"]
```

Or reduce services to only what you need.
