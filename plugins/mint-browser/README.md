# mint-browser

Browser automation plugin for [mint](https://github.com/3li7alaki/mint). Navigate, interact, extract, and verify web pages through PinchTab's HTTP API.

## What It Does

- **Browser automation** in the mint pipeline via `/browse`, `/screenshot`, `/scrape` commands
- **UI review** as a Stage 2 reviewer — after implementation, navigates to the live app and verifies the rendered result matches acceptance criteria
- **Pre-plan context** — captures current page state before planning UI work, giving the planner accurate context about what already exists
- **Live debugging** — inspects browser state, console errors, and DOM to diagnose issues in running apps

All operations use PinchTab's HTTP API via curl. No runtime dependencies, no MCP tools, no build step.

## Prerequisites

- **PinchTab** binary (v0.7.0+) — browser automation via HTTP API
  - Install: `curl -fsSL https://pinchtab.com/install.sh | sh`
  - Or: `npm install -g pinchtab`
  - Or: `docker pull pinchtab/pinchtab`
- **Chrome/Chromium** — PinchTab manages its own Chrome instance

## Quick Setup

1. Enable the plugin in `.mint/config.json`:

```json
{
  "plugins": ["plugins/mint-browser"],
  "browser": {
    "enabled": true,
    "baseUrl": "http://localhost:9867",
    "devServer": "http://localhost:3000"
  }
}
```

2. Start PinchTab:

```bash
pinchtab &
```

3. Use browser commands:

```bash
/browse https://myapp.com/login "fill in the form"
/screenshot localhost:3000/dashboard
/scrape https://docs.example.com/api "list all endpoints"
```

Or let it work automatically — the browser reviewer activates on UI file changes, and the context agent enriches planning for UI tasks.

## Configuration

All config lives under the `browser` key in `.mint/config.json`:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | boolean | `true` | Enable/disable the entire plugin |
| `baseUrl` | string | `http://localhost:9867` | PinchTab API base URL |
| `token` | string | `null` | Bearer token for PinchTab auth |
| `headless` | boolean | `true` | Run Chrome in headless mode |
| `devServer` | string | `http://localhost:3000` | Local dev server URL |
| `uiFilePatterns` | string[] | `["*.vue", "*.tsx", "*.jsx", "*.svelte", "*.html", "*.css", "*.scss"]` | File patterns that trigger browser review |
| `reviewMode` | string | `"auto"` | When to run browser review: `"auto"` (on UI files), `"always"`, `"off"` |
| `timeout` | number | `30` | Navigation timeout in seconds |
| `blockImages` | boolean | `false` | Block image loading (saves bandwidth) |

### Full Example

```json
{
  "plugins": ["plugins/mint-browser"],
  "browser": {
    "enabled": true,
    "baseUrl": "http://localhost:9867",
    "token": null,
    "headless": true,
    "devServer": "http://localhost:5173",
    "uiFilePatterns": ["*.vue", "*.tsx", "*.jsx", "*.svelte", "*.html", "*.css", "*.scss"],
    "reviewMode": "auto",
    "timeout": 30,
    "blockImages": false
  }
}
```

## Commands

| Command | Description |
|---------|-------------|
| `/browse <url> [task]` | Navigate to URL and perform task (form filling, clicking, verification) |
| `/screenshot [url]` | Capture page screenshot (defaults to devServer) |
| `/scrape <url> [what]` | Extract structured data from a page |

## Pipeline Integration

### Pre-Plan Hook: Browser Context

When a task involves UI work, the context agent navigates to the relevant page and captures:
- Interactive elements (forms, buttons, links)
- Page text content
- Navigation structure

This gives the planner accurate context about the current state before decomposing specs.

**Activation:** Opt-in — only runs when the task mentions a URL or is clearly UI-focused.

### Pre-Review Hook: Browser Reviewer

After UI implementation, the reviewer:
1. Navigates to the affected page on the dev server
2. Snapshots the rendered result
3. Verifies acceptance criteria are met
4. Checks for console errors (if evaluate is enabled)
5. Returns BLOCKING/WARNING/INFO findings

**Activation:** Based on `reviewMode`:
- `"auto"` — only when the diff touches files matching `uiFilePatterns`
- `"always"` — on every spec
- `"off"` — disabled

### On-Init Hook: Browser Setup

During `mint init`, the setup agent:
1. Checks if PinchTab is installed
2. Offers installation if missing
3. Configures secure defaults
4. Writes browser config to `.mint/config.json`
5. Verifies with a health check

### Graceful Degradation

If PinchTab is not running or the dev server is unavailable, all agents degrade gracefully:
- Reviewers return WARNING and skip (never block the pipeline)
- Context agent skips and planning proceeds without browser context
- Commands show clear instructions for starting PinchTab

## Agents

| Agent | Role | Hook |
|-------|------|------|
| `browser-runner.md` | Core automation — navigate, snapshot, interact, extract | Commands |
| `browser-reviewer.md` | Stage 2 UI verification | pre-review |
| `browser-context.md` | Pre-plan page state capture | pre-plan |
| `browser-debugger.md` | Live app debugging and diagnosis | On demand |
| `browser-setup.md` | Installation and configuration | on-init |

## Token Cost Guide

PinchTab operations have varying token costs. Agents use the cheapest method that accomplishes the task.

| Method | Typical Tokens | When Used |
|--------|---------------|-----------|
| `/text` | ~800 | Reading page content, verifying text |
| `/snapshot?filter=interactive&format=compact` | ~3,600 | Finding elements to interact with |
| `/snapshot?diff=true` | ~200-1,000 | Seeing what changed after an action |
| `/screenshot` | ~2,000 (vision) | Visual verification |
| Full `/snapshot` | ~10,500 | Full page understanding (rare) |

See `references/token-strategy.md` for the complete optimization guide.

## Security

- PinchTab binds to `127.0.0.1` by default — local access only
- Set `browser.token` for authenticated PinchTab instances
- JavaScript evaluation is never used by default — only the debugger uses it when explicitly requested
- Always use a dedicated Chrome profile for automation, not your daily browser profile
- See PinchTab's security model: `references/` directory in this plugin

## References

Bundled PinchTab documentation:

| File | Content |
|------|---------|
| `references/api.md` | Full HTTP API reference (navigate, snapshot, action, text, screenshot, etc.) |
| `references/env.md` | Environment variables for PinchTab configuration |
| `references/profiles.md` | Profile management for multi-account automation |
| `references/token-strategy.md` | Token optimization guide for agents |
