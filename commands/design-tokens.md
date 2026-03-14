# /design:tokens Command

Export and sync design tokens across formats.

## Usage

```
/design:tokens <action> [options]
```

## Actions

### export

Export design system to various formats.

```bash
/design:tokens export [--format <format>] [--output <path>]
```

**Formats:** `css`, `tailwind`, `scss`, `json`, `figma`

**Examples:**
```bash
/design:tokens export                        # CSS to stdout
/design:tokens export --format tailwind      # Tailwind config
/design:tokens export --format css --output src/styles/tokens.css
```

### sync

Sync tokens between design system and code.

```bash
/design:tokens sync [--source <path>] [--target <format>]
```

**Examples:**
```bash
/design:tokens sync --source design-system.json --target css
/design:tokens sync --source tailwind.config.js --target json
/design:tokens sync --source globals.css --target json
```

### validate

Check tokens for consistency across files.

```bash
/design:tokens validate [path]
```

**Examples:**
```bash
/design:tokens validate                      # Check all token files
/design:tokens validate design-system.json   # Check specific file
```

## Implementation

Export reads from design profile (`.mint/design-profile.json`) or explicit `design-system.json`.

Sync compares token files and updates targets. Validate checks naming consistency, value formats, cross-file synchronization, and usage in codebase.

## Notes

- Preserves existing token values when syncing
- Warns before overwriting
- Supports dark mode tokens
- Works with shadcn CSS variable patterns
