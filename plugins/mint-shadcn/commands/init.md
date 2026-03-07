# /shadcn:init Command

Set up shadcn AI tooling for an existing project.

## Usage

```
/shadcn:init [options]
```

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--mcp` | Configure MCP server only | - |
| `--skills` | Install shadcn/skills only | - |
| `--registries` | Add registries only | - |
| `--client <name>` | Target specific client (claude, cursor, vscode) | all |
| `--preset <code>` | Apply a preset (colors, fonts, radius) | - |
| `--base <lib>` | Switch base library (radix, base) | - |
| `--rtl` | Enable RTL support | - |
| `--no-mcp` | Skip MCP configuration | - |
| `--no-skills` | Skip skills installation | - |
| `--reinstall` | Re-install all components with current config | - |

If no options provided, runs full setup.

## Examples

```bash
# Full setup
/shadcn:init

# Just add MCP server for Claude Code
/shadcn:init --mcp --client claude

# Add registries and skills, skip MCP
/shadcn:init --registries --skills --no-mcp

# Quick setup for Cursor only
/shadcn:init --client cursor

# Apply a preset (changes colors, fonts, radius)
/shadcn:init --preset "zinc-orange-bold"

# Switch to Base UI primitives
/shadcn:init --base base --reinstall

# Enable RTL support
/shadcn:init --rtl
```

## Presets

Presets bundle design system config in a single code. Create them at https://ui.shadcn.com/themes

When applying a preset:
1. Updates CSS variables in your globals.css
2. Reconfigures `components.json`
3. Optionally reinstalls components with new defaults

Example presets:
- `zinc-orange-bold` — Zinc base, orange accent
- `slate-blue-sharp` — Slate with sharp corners
- `neutral-green-soft` — Green accent, soft radius

## Process

1. **Validate project** — check for `components.json`
2. **Detect variant** — React, Vue, or Svelte based on schema URL
3. **Parse options** — determine what to configure
4. **Invoke shadcn-setup agent** with the parsed options

## Errors

### No components.json

```
Error: No components.json found.

This command requires an existing shadcn project.

To initialize shadcn from scratch:
  npx shadcn@latest init

Then run /shadcn:init to add AI tooling.
```

### Invalid client

```
Error: Unknown client 'foo'.

Supported clients:
- claude (Claude Code)
- cursor (Cursor IDE)
- vscode (VS Code)

Usage: /shadcn:init --client claude
```

## What It Sets Up

### MCP Server

Enables natural language component installation:
- "Add a button and dialog"
- "Show me all available components"
- "Search for a date picker"

### shadcn/skills

Injects project context into AI agents:
- Component aliases and paths
- Installed components list
- Framework-specific patterns
- Registry awareness

### Registries

Third-party component sources:
- **@magicui** — animated components, backgrounds, effects
- **@animate-ui** — motion-first components
- **@aceternity** — creative UI patterns

## Implementation

When this command is invoked:

1. **Check for components.json:**
   - If missing: show "No components.json" error
   - If present: read and parse it

2. **Detect package manager:**
   - Check for lockfiles: `pnpm-lock.yaml`, `package-lock.json`, `yarn.lock`, `bun.lockb`
   - Default to `npm` if none found

3. **Parse command options:**
   - Build options object for the setup agent
   - Validate client names if `--client` provided

4. **Invoke shadcn-setup agent:**
   - Pass: project root, components.json content, options, package manager
   - Agent handles all file modifications

5. **Report results:**
   - Show what was configured
   - Provide next steps (restart IDE, test commands)

## Notes

- Safe to run multiple times — merges into existing configs
- Won't overwrite custom MCP servers
- Respects `.gitignore` for cache files
