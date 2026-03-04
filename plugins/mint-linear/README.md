# mint-linear

Linear integration plugin for [mint](https://github.com/3li7alaki/mint).

## Install

```bash
git clone https://github.com/3li7alaki/mint-linear .mint/plugins/mint-linear
```

Add to `.mint/config.json`:

```json
{
  "plugins": [".mint/plugins/mint-linear"],
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

Requires Linear MCP tools to be available in your Claude Code environment. The plugin gracefully degrades if tools are unavailable — it logs a warning and lets the pipeline continue.

## What It Does

### Pre-Plan: Ticket Context

When you reference a ticket in your feature description (e.g., "implement SAL-20" or "the checkout flow from SAL-20"), the plugin:

- Fetches the full ticket description and acceptance criteria
- Pulls related tickets and their statuses
- Grabs discussion comments for decision context
- Provides all of this as structured context to the planner

This means your specs automatically include ticket requirements as acceptance criteria.

### Post-Plan: Project Updates

After planning is complete (specs created), the plugin posts a project status update to Linear summarizing what's being worked on, how many specs, and estimated scope. Also posts updates at ship completion and milestone points with health assessment (on track / at risk / off track).

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
| `project-update.md` | post-plan | Posts project status updates at key milestones |
| `status-sync.md` | post-commit | Updates ticket status and adds commit comments |
