# /gws Command

Google Workspace operations — setup, workflows, and quick actions.

## Usage

```
/gws <subcommand> [args]
```

## Subcommands

### setup

Interactive setup wizard for CLI, auth, and MCP configuration.

```
/gws setup
```

**Process:** Invokes `setup-wizard.md` agent.

---

### workflow

Execute a multi-step workflow with natural language.

```
/gws workflow "<task description>"
```

**Examples:**
```bash
/gws workflow "Create a summary doc from the Q1 Sales spreadsheet"
/gws workflow "Find all Project Alpha docs and create a meeting agenda"
/gws workflow "Export contacts from CRM sheet to a draft email"
```

**Process:** Invokes `workflow.md` agent with the task description.

---

### drive

Drive operations.

```
/gws drive list [folder]              # List files (default: root)
/gws drive search "<query>"           # Search files
/gws drive info <file-id>             # Get file metadata
/gws drive share <file-id> <email>    # Share file
```

**Process:** Executes gws CLI directly for simple operations.

---

### sheets

Spreadsheet operations.

```
/gws sheets list                      # List spreadsheets
/gws sheets schema <spreadsheet-id>   # Show sheet structure
/gws sheets read <id> <range>         # Read cell range
/gws sheets write <id> <range> <data> # Write to cells
```

**Examples:**
```bash
/gws sheets schema 1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms
/gws sheets read 1Bxi... "Sales Data!A1:G10"
```

**Process:** Executes gws CLI or MCP tools as appropriate.

---

### gmail

Gmail operations.

```
/gws gmail list [--unread]            # List recent emails
/gws gmail search "<query>"           # Search emails
/gws gmail send --to <email> --subject "<subj>" --body "<body>"
/gws gmail draft --to <email> --subject "<subj>" --body "<body>"
```

**Examples:**
```bash
/gws gmail list --unread
/gws gmail search "from:boss@company.com"
/gws gmail draft --to team@example.com --subject "Weekly Update" --body "..."
```

**Note:** `send` asks for confirmation before sending. Use `draft` to create without sending.

---

### calendar

Calendar operations.

```
/gws calendar list [--days N]         # List upcoming events (default: 7 days)
/gws calendar create "<title>" --when "<time>" [--duration <mins>]
/gws calendar info <event-id>         # Get event details
```

**Examples:**
```bash
/gws calendar list --days 14
/gws calendar create "Team Sync" --when "tomorrow 3pm" --duration 30
```

---

### docs

Document operations.

```
/gws docs list                        # List recent documents
/gws docs create "<title>"            # Create new document
/gws docs outline <doc-id>            # Show document structure
```

---

## Arguments

| Argument | Description |
|----------|-------------|
| `<file-id>` | Google Drive file ID (from URL or previous command) |
| `<spreadsheet-id>` | Spreadsheet ID (same as file-id for sheets) |
| `<range>` | A1 notation range, e.g., "Sheet1!A1:C10" |
| `<query>` | Search query string |
| `<email>` | Email address |

## Errors

### gws not installed

```
Error: Google Workspace CLI not found.

Run setup:
  /gws setup

Or install manually:
  npm install -g @googleworkspace/cli
```

### Not authenticated

```
Error: Not authenticated with Google.

Run:
  gws auth login

Or for first-time setup:
  /gws setup
```

### MCP not configured

```
Error: gws MCP server not configured.

Add to .claude/settings.json:
{
  "mcpServers": {
    "gws": {
      "command": "gws",
      "args": ["mcp", "-s", "drive,gmail,calendar,sheets", "--tool-mode", "compact"]
    }
  }
}

Then restart Claude Code.
```

### Unknown subcommand

```
Error: Unknown subcommand 'foo'.

Available subcommands:
  setup     — Interactive setup wizard
  workflow  — Multi-step workflows
  drive     — Drive operations
  sheets    — Spreadsheet operations
  gmail     — Email operations
  calendar  — Calendar operations
  docs      — Document operations

Usage: /gws <subcommand> [args]
```

## Implementation

When this command is invoked:

1. **Parse subcommand:**
   - Extract first argument as subcommand
   - Remaining arguments passed to handler

2. **Check prerequisites:**
   - For `setup`: no prerequisites
   - For others: verify `gws` installed and authenticated

3. **Route to handler:**
   - `setup` → invoke `setup-wizard.md` agent
   - `workflow` → invoke `workflow.md` agent with task
   - Others → execute gws CLI or MCP tools directly

4. **Report results:**
   - Show output from gws/MCP
   - Format for readability
   - Include relevant links (Drive URLs, etc.)

## Notes

- The `workflow` subcommand is the most powerful — use it for anything multi-step
- Simple operations (list, search, read) run directly via CLI/MCP
- Write operations ask for confirmation when appropriate
- Email `send` always asks for confirmation — use `draft` to skip
- All operations respect configured `gws.services` scope
