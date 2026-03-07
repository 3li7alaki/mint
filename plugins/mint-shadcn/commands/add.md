# /shadcn:add Command

Install shadcn components with preview and multi-registry support.

## Usage

```
/shadcn:add <components...> [options]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `components` | Yes | One or more components to add (e.g., `button dialog card`) |

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--dry-run` | Preview what will be added without writing files | enabled |
| `--no-dry-run` | Skip preview, write files directly | - |
| `--registry <name>` | Install from specific registry (e.g., `@magicui`) | auto-detect |
| `--overwrite` | Overwrite existing components | - |
| `--path <path>` | Custom installation path | from components.json |

## Examples

```bash
# Preview adding button (dry-run by default)
/shadcn:add button

# Add multiple components
/shadcn:add button dialog card

# Add from a specific registry
/shadcn:add @magicui/magic-card

# Skip preview and install directly
/shadcn:add button --no-dry-run

# Overwrite existing component
/shadcn:add button --overwrite

# Search across all registries
/shadcn:add "animated button"
```

## Process

1. **Parse component names** — handle registry prefixes
2. **Resolve registries** — check if component exists
3. **Show dry-run preview** — files, dependencies, changes
4. **Confirm with user** — unless `--no-dry-run`
5. **Install components** — run shadcn add

## Output

### Dry-Run Preview (Default)

```
## shadcn Add Preview

### Components
- button (shadcn/ui)
- magic-card (@magicui)

### Files to Create
  src/components/ui/button.tsx
  src/components/ui/magic-card.tsx

### Dependencies to Install
  @radix-ui/react-slot (new)
  framer-motion (new)

### Tailwind Config
  No changes required

### Estimated Size
  +2.3 KB (minified)

─────────────────────────────
Proceed with installation? [Y/n]
```

### Search Mode

When the component name doesn't match exactly:

```
## shadcn Search: "animated button"

Found in registries:

@magicui
  └─ shimmer-button — Button with shimmer effect
  └─ pulsating-button — Button with pulse animation

@animate-ui
  └─ button — Animated button with variants

shadcn/ui
  └─ button — Base button (no animation)

Select component to add (or 'all' for multiple):
> @magicui/shimmer-button
```

### Installation Complete

```
## shadcn Add Complete

✅ button — installed
✅ magic-card — installed

New dependencies added:
  pnpm add @radix-ui/react-slot framer-motion

Files created:
  src/components/ui/button.tsx
  src/components/ui/magic-card.tsx

Next: Import and use
  import { Button } from "@/components/ui/button"
  import { MagicCard } from "@/components/ui/magic-card"
```

## Registry Prefixes

Components can be prefixed with registry names:

| Prefix | Registry | Example |
|--------|----------|---------|
| (none) | shadcn/ui | `button` |
| `@magicui/` | Magic UI | `@magicui/magic-card` |
| `@animate-ui/` | Animate UI | `@animate-ui/accordion` |
| `@aceternity/` | Aceternity | `@aceternity/spotlight` |

Registries must be configured in `components.json` first.

## Smart Detection

When adding components without a registry prefix:

1. **Check shadcn/ui first** — the default registry
2. **Search configured registries** — if not found in default
3. **Present options** — if found in multiple registries
4. **Suggest similar** — if not found anywhere

## Errors

### Component not found

```
Error: Component 'fancy-thing' not found.

Searched registries:
- shadcn/ui ✗
- @magicui ✗
- @animate-ui ✗

Did you mean:
- fancy-button (@magicui)
- thing-card (@aceternity)

Or search: /shadcn:add "fancy"
```

### Registry not configured

```
Error: Registry '@custom' is not configured.

Add it to components.json:
{
  "registries": {
    "@custom": "https://custom-registry.com/r/{name}.json"
  }
}

Then run: /shadcn:add @custom/component
```

### Component exists

```
Component 'button' already exists at:
  src/components/ui/button.tsx

Options:
1. Skip (default)
2. Overwrite: /shadcn:add button --overwrite
3. View diff: /shadcn:upgrade button --diff
```

## Implementation

When this command is invoked:

1. **Parse arguments:**
   - Extract component names
   - Detect registry prefixes
   - Parse flags

2. **For each component:**
   - If prefixed: check specific registry
   - If not: search all configured registries
   - If ambiguous: present options

3. **Generate preview:**
   - Run `npx shadcn add <components> --dry-run`
   - Parse output for files, dependencies
   - Check for existing files

4. **If `--no-dry-run` or user confirms:**
   - Run `npx shadcn add <components>`
   - Report results

5. **Handle errors:**
   - Component not found → suggest alternatives
   - Already exists → offer overwrite
   - Network error → retry or fail gracefully

## Notes

- Dry-run is enabled by default for safety
- Dependencies are installed automatically
- Tailwind config is updated if needed
- Works with monorepo setups (detects workspace root)
- Respects `--path` for custom component locations
