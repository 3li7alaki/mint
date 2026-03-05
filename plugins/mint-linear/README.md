# mint-linear

Linear integration plugin for [mint](https://github.com/3li7alaki/mint).

## Install

mint-linear ships with mint. After installing mint (`npx skills add 3li7alaki/mint`), enable it in `.mint/config.json`:

```json
{
  "plugins": ["plugins/mint-linear"],
  "linear": {
    "team": "your-team-name",
    "project": "your-project-name",
    "labels": {
      "inProgress": "In Progress",
      "inReview": "In Review",
      "done": "Done"
    }
  }
}
```

## Requirements

**Required:** The `linear` Claude Code plugin must be enabled in your project's `.claude/settings.json`:

```json
{
  "enabledPlugins": {
    "linear@claude-plugins-official": true
  }
}
```

This provides the Linear MCP tools (`get_issue`, `list_issues`, `list_comments`, etc.) that the agents use to fetch and update ticket data.

The plugin gracefully degrades if tools are unavailable — it logs a warning and lets the pipeline continue.

## What It Does

### Pre-Plan: Ticket Context

When you reference a ticket in your feature description (e.g., "implement SAL-20" or "the checkout flow from SAL-20"), the plugin:

- Fetches the full ticket description and acceptance criteria
- Pulls related tickets and their statuses
- Grabs discussion comments for decision context
- Provides all of this as structured context to the planner

This means your specs automatically include ticket requirements as acceptance criteria.

### Project Updates (Manual)

When you ask for a project update (e.g., "post a Linear update", "update the project status"), the plugin posts a status update to your Linear project with health assessment (on track / at risk / off track) and a summary of work done. Use at the start or end of significant work, or whenever a milestone is reached.

### Post-Commit: Status Sync

After each spec is committed, the plugin:

- Updates the ticket status to "In Progress" or "In Review"
- Adds a comment summarizing what was committed (spec title, files, gate results)

The plugin **never** marks tickets as "Done" — that's always a human decision.

## Configuration

| Key | Default | Description |
|-----|---------|-------------|
| `linear.team` | null | Linear team name or ID |
| `linear.project` | null | Linear project name (optional, for milestone context) |
| `linear.labels.inProgress` | "In Progress" | Status name for work in progress |
| `linear.labels.inReview` | "In Review" | Status name for review |
| `linear.labels.done` | "Done" | Status name for done (read-only, never set by plugin) |

## Agents

| Agent | Hook | Role |
|-------|------|------|
| `ticket-context.md` | pre-plan | Fetches ticket details as planner context |
| `project-update.md` | manual | Posts project status updates on request |
| `status-sync.md` | post-commit | Updates ticket status and adds commit comments |
