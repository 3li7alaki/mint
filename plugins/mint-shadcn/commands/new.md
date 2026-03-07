# /shadcn:new Command

Scaffold a new project with shadcn/ui preconfigured.

## Usage

```
/shadcn:new <name> [options]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | Yes | Project name (creates directory) |

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--template <name>` | Project template | next |
| `--base <lib>` | Component library (radix, base) | radix |
| `--preset <code>` | Apply preset (colors, fonts, etc.) | - |
| `--monorepo` | Scaffold as monorepo | - |
| `--rtl` | Enable RTL support | - |
| `-y, --yes` | Skip prompts, use defaults | - |

## Templates

| Template | Description |
|----------|-------------|
| `next` | Next.js with App Router |
| `vite` | Vite + React |
| `start` | TanStack Start |
| `react-router` | React Router v7 |
| `laravel` | Laravel + Inertia |
| `astro` | Astro with React islands |

## Examples

```bash
# Create Next.js project with defaults
/shadcn:new my-app

# Create Vite project with Base UI
/shadcn:new my-app --template vite --base base

# Apply a preset (from shadcn/create)
/shadcn:new my-app --preset "zinc-orange-bold"

# Create TanStack Start project for RTL
/shadcn:new my-app --template start --rtl

# Scaffold monorepo
/shadcn:new my-workspace --template next --monorepo

# Non-interactive
/shadcn:new my-app --template vite --yes
```

## Presets

Presets bundle your entire design system in one code:
- Colors (primary, secondary, accent)
- Fonts (heading, body)
- Border radius
- Icon library

**Create presets at:** https://ui.shadcn.com/themes

Example preset codes:
- `zinc-orange-bold` — Zinc base, orange accent, rounded corners
- `slate-blue-sharp` — Slate base, blue accent, sharp corners
- `neutral-green-soft` — Neutral base, green accent, soft radius

## Output

```
## Creating Project: my-app

Template:   next
Base:       radix
Preset:     zinc-orange-bold (applied)
RTL:        disabled
Monorepo:   no

─────────────────────────────

Creating project structure...
Installing dependencies...
Configuring shadcn/ui...
Applying preset...

✅ Project created: my-app/

Next steps:
  cd my-app
  pnpm dev

─────────────────────────────

Installed components:
  button, card, input, label (from preset)

MCP server:
  Run /shadcn:init to configure AI tooling
```

## Implementation

When this command is invoked:

1. **Validate project name:**
   - Check directory doesn't exist
   - Validate naming conventions

2. **Run shadcn init with template:**
   ```bash
   npx shadcn init \
     --name <name> \
     --template <template> \
     --base <base> \
     --preset <preset> \
     [--monorepo] \
     [--rtl] \
     [--yes]
   ```

3. **Post-creation:**
   - Change to project directory
   - Suggest running `/shadcn:init` for MCP setup
   - Show next steps

## Notes

- Templates include dark mode support by default
- Presets can be shared between projects
- Monorepo mode creates `apps/` and `packages/` structure
- RTL adds bidirectional text support
- All templates use Tailwind CSS v4
