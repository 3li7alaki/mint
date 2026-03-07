# /shadcn:migrate Command

Run migrations to update components for breaking changes.

## Usage

```
/shadcn:migrate [migration] [options]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `migration` | No | Specific migration to run. If omitted, lists available. |

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--list` | List all available migrations | - |
| `--dry-run` | Preview changes without applying | - |
| `-y, --yes` | Skip confirmation prompt | - |

## Available Migrations

| Migration | Description |
|-----------|-------------|
| `icons` | Migrate icon library (e.g., lucide-react to @phosphor-icons) |
| `radix` | Update Radix UI package structure (monorepo → individual packages) |
| `rtl` | Add RTL support to components |

## Examples

```bash
# List available migrations
/shadcn:migrate --list

# Preview icon migration
/shadcn:migrate icons --dry-run

# Run radix package migration
/shadcn:migrate radix

# Run all migrations (with confirmation)
/shadcn:migrate

# Run RTL migration without prompts
/shadcn:migrate rtl --yes
```

## Output

### List Mode

```
## Available Migrations

| Migration | Status | Description |
|-----------|--------|-------------|
| icons     | ready  | Migrate to new icon library |
| radix     | done   | Already migrated |
| rtl       | ready  | Add RTL support |

Run: /shadcn:migrate <name>
Preview: /shadcn:migrate <name> --dry-run
```

### Dry Run

```
## Migration Preview: icons

### Files to Modify (12)

src/components/ui/button.tsx
  - import { Loader2 } from "lucide-react"
  + import { Spinner } from "@phosphor-icons/react"

src/components/ui/dialog.tsx
  - import { X } from "lucide-react"
  + import { X } from "@phosphor-icons/react"

...

### Dependencies
  Remove: lucide-react
  Add:    @phosphor-icons/react

─────────────────────────────
Apply this migration? [Y/n]
```

### Completed Migration

```
## Migration Complete: icons

✅ Updated 12 files
✅ Removed lucide-react
✅ Added @phosphor-icons/react

Changes committed: chore(shadcn): migrate icons to phosphor

Run /shadcn:info to verify changes.
```

## Migration Details

### icons

Updates import statements across all components when switching icon libraries:
- Detects current icon library from `components.json`
- Maps icon names between libraries
- Updates all import statements
- Swaps dependencies

### radix

Handles Radix UI package restructuring:
- Old: `@radix-ui/react-*` individual packages
- New: `radix-ui` monorepo package
- Updates all imports and package.json

### rtl

Adds RTL (Right-to-Left) support:
- Adds `dir` prop support to relevant components
- Updates CSS for logical properties
- Configures `components.json` for RTL

## Implementation

When this command is invoked:

1. **If `--list` or no migration specified:**
   - Run `npx shadcn migrate --list`
   - Show available migrations with status

2. **If migration specified:**
   - Run `npx shadcn migrate <name> --dry-run` first
   - Show preview of changes
   - If `--yes` or user confirms, apply
   - Run `npx shadcn migrate <name>`

3. **Report results:**
   - Files modified
   - Dependencies changed
   - Any manual steps required

## Notes

- Migrations are idempotent — safe to run multiple times
- Always preview with `--dry-run` first
- Some migrations may require manual review
- Commit changes after each migration
