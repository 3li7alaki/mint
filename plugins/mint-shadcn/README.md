# mint-shadcn

shadcn/ui integration plugin for [mint](https://github.com/3li7alaki/mint).

Full shadcn CLI v4 support: MCP server, skills, presets, registries, migrations, and component conventions.

## Features

- **Auto-detection** — mint prompts to enable when `components.json` is found
- **MCP server setup** — Configure for Claude Code, Cursor, VS Code
- **Skills installation** — Inject component context into AI agents
- **Presets** — Apply design system configs (colors, fonts, radius)
- **Registry management** — Multi-registry component installation
- **Migrations** — Update components for breaking changes
- **Convention reviewer** — Stage 2 auditor for shadcn patterns

## Install

mint-shadcn ships with mint. Enable it in `.mint/config.json`:

```json
{
  "plugins": ["plugins/mint-shadcn"]
}
```

Or let mint auto-detect — when it finds `components.json`:

```
shadcn detected. Enable mint-shadcn plugin for component conventions and AI tooling setup?
```

## Commands

### `/shadcn:new`

Scaffold a new project with shadcn preconfigured.

```bash
/shadcn:new my-app                           # Next.js default
/shadcn:new my-app --template vite           # Vite + React
/shadcn:new my-app --template start          # TanStack Start
/shadcn:new my-app --preset "zinc-orange"    # With preset
/shadcn:new my-app --base base               # Use Base UI
/shadcn:new my-app --monorepo                # Monorepo setup
```

Templates: `next`, `vite`, `start`, `react-router`, `laravel`, `astro`

### `/shadcn:init`

Set up AI tooling for an existing project.

```bash
/shadcn:init                                 # Full setup
/shadcn:init --mcp --client claude           # MCP for Claude only
/shadcn:init --preset "slate-blue-sharp"     # Apply preset
/shadcn:init --base base --reinstall         # Switch to Base UI
/shadcn:init --rtl                           # Enable RTL
```

### `/shadcn:add`

Install components with preview and multi-registry support.

```bash
/shadcn:add button                           # Preview (dry-run default)
/shadcn:add button dialog card               # Multiple components
/shadcn:add @magicui/magic-card              # From registry
/shadcn:add button --no-dry-run              # Install directly
/shadcn:add "animated card"                  # Search and add
```

### `/shadcn:search`

Search across all configured registries.

```bash
/shadcn:search button                        # Search all
/shadcn:search "date picker"                 # Fuzzy search
/shadcn:search card --registry @magicui      # Specific registry
```

### `/shadcn:docs`

Get component documentation and API reference.

```bash
/shadcn:docs button                          # Full docs
/shadcn:docs dialog --base base              # Base UI variant
/shadcn:docs accordion --json                # JSON output
```

### `/shadcn:info`

Display project configuration and installed components.

```bash
/shadcn:info                                 # Full info
/shadcn:info --json                          # JSON for AI agents
```

### `/shadcn:upgrade`

Check for and apply component updates.

```bash
/shadcn:upgrade                              # Check all
/shadcn:upgrade button --diff                # Preview changes
/shadcn:upgrade --all                        # Upgrade all
/shadcn:upgrade --cli                        # Update CLI
```

### `/shadcn:migrate`

Run migrations for breaking changes.

```bash
/shadcn:migrate --list                       # List available
/shadcn:migrate icons --dry-run              # Preview
/shadcn:migrate radix                        # Run migration
/shadcn:migrate rtl                          # Add RTL support
```

Migrations: `icons`, `radix`, `rtl`

## What It Sets Up

### MCP Server

Natural language component management:
- "Add a button and dialog"
- "Search for a date picker"
- "Show me magic-card from @magicui"

### shadcn/skills

AI agents gain context about:
- Your aliases and paths
- Installed components
- Framework patterns
- CLI commands and flags

### Presets

Design system in one code:
- Colors (primary, secondary, accent)
- Fonts (heading, body)
- Border radius
- Icon library

Create at: https://ui.shadcn.com/themes

### Registries

Third-party component sources:

| Registry | URL | Description |
|----------|-----|-------------|
| @magicui | magicui.design | Animated backgrounds, effects |
| @animate-ui | animate-ui.com | Motion-first components |
| @aceternity | ui.aceternity.com | Creative UI patterns |

## Supported Variants

Auto-detected from `components.json`:

| Schema | Variant | Framework |
|--------|---------|-----------|
| `ui.shadcn.com` | shadcn/ui | React |
| `shadcn-vue.com` | shadcn-vue | Vue |
| `shadcn-svelte.com` | shadcn-svelte | Svelte |

## Base Libraries

| Base | Description |
|------|-------------|
| `radix` | Radix UI primitives (default) |
| `base` | Base UI primitives (headless) |

Switch with: `/shadcn:init --base base --reinstall`

## Reviewer

Stage 2 auditor checks:

- **Import patterns** — using configured aliases
- **Component usage** — not reinventing existing components
- **Styling** — CSS variables, dark mode, `cn()` utility
- **Framework patterns** — React `forwardRef`, Vue `defineProps`
- **Registry opportunities** — suggesting available components

Severity: BLOCKING, WARNING, INFO

## Configuration

In `.mint/config.json`:

```json
{
  "shadcn": {
    "variant": "auto",
    "base": "radix",
    "registries": ["@magicui", "@animate-ui", "@aceternity"],
    "mcp": {
      "enabled": true,
      "clients": ["claude", "cursor", "vscode"]
    },
    "skills": true,
    "presets": []
  }
}
```

## Requirements

- shadcn CLI v4+
- `components.json` in project root
- Node.js 18+

## Sources

- [shadcn/cli v4 Changelog](https://ui.shadcn.com/docs/changelog/2026-03-cli-v4)
- [shadcn/skills Documentation](https://ui.shadcn.com/docs/skills)
- [CLI Reference](https://ui.shadcn.com/docs/cli)
- [MCP Server Setup](https://ui.shadcn.com/docs/mcp)
