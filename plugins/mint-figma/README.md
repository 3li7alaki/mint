# mint-figma

Figma integration plugin for [mint](https://github.com/3li7alaki/mint).

## Install

mint-figma ships with mint. After installing mint (`npx skills add 3li7alaki/mint`), enable it in `.mint/config.json`:

```json
{
  "plugins": ["plugins/mint-figma"],
  "figma": {
    "fileId": "your-figma-file-id",
    "apiToken": "your-figma-api-token",
    "exports": "local-docs/figma-exports",
    "pages": ["PDP", "PLP", "Homepage"]
  }
}
```

## Requirements

Requires a Figma API token (`figma.apiToken` in config). The plugin uses the Figma REST API directly to fetch file structure, component specs, design variables, styles, and rendered screenshots.

Figma MCP tools are used when available. Local exports directory (`figma.exports`) provides supplemental curated data like processed token docs.

## What It Does

### Pre-Plan: Design Context

When you describe a feature to build, the plugin:

- Checks local exports for relevant design data (tokens, screenshots, component specs)
- Fetches from Figma API if local data doesn't cover the feature
- Extracts actionable design requirements: layout, colors, spacing, typography, states
- Maps values to design token names
- Saves context to `.mint/research/` for the review phase

The planner receives design requirements alongside ticket context, producing specs that reference exact token values and component specs.

### Pre-Review: Design Alignment

During the stage 2 parallel audit, the plugin:

- Checks implementation against design specs (colors, spacing, typography, layout)
- Flags hardcoded values that should use design tokens
- Identifies missing states (hover, disabled, loading, error, empty)
- Verifies responsive breakpoint handling
- Reports design alignment per category

## Configuration

| Key | Default | Description |
|-----|---------|-------------|
| `figma.fileId` | null | Figma file ID (from the URL) |
| `figma.apiToken` | null | Figma API token for fetching data |
| `figma.exports` | `local-docs/figma-exports` | Local directory for exported design data |
| `figma.pages` | `[]` | Figma page names to focus on |

## Local Exports

For best results, maintain a local exports directory with:

```
local-docs/figma-exports/
├── DESIGN-TOKENS.md      # Colors, spacing, typography tokens
├── variables.json         # Figma variables export
├── styles.json            # Figma styles export
├── pdp-desktop.png        # Page screenshots
├── pdp-mobile.png
└── component-specs.json   # Component dimension/spacing data
```

The plugin checks local exports first before making API calls.

## Agents

| Agent | Hook | Role |
|-------|------|------|
| `design-context.md` | pre-plan | Fetches design specs as planner context |
| `design-reviewer.md` | pre-review | Checks implementation against design specs |
