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

**Formats:**
- `css` — CSS custom properties
- `tailwind` — Tailwind config theme
- `scss` — SCSS variables
- `json` — JSON tokens
- `figma` — Figma variables format

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
# Sync design-system.json to CSS variables
/design:tokens sync --source design-system.json --target css

# Sync from Tailwind config to design system
/design:tokens sync --source tailwind.config.js --target json

# Sync shadcn CSS variables to design system
/design:tokens sync --source globals.css --target json
```

### validate

Check tokens for consistency.

```bash
/design:tokens validate [path]
```

**Examples:**
```bash
/design:tokens validate                      # Check all token files
/design:tokens validate design-system.json   # Check specific file
```

## Output

### CSS Export

```css
:root {
  /* Colors */
  --color-primary: #3b82f6;
  --color-secondary: #64748b;
  --color-accent: #f59e0b;
  --color-background: #ffffff;
  --color-foreground: #0f172a;

  /* Typography */
  --font-heading: 'Inter', sans-serif;
  --font-body: 'Inter', sans-serif;
  --font-size-xs: 0.75rem;
  --font-size-sm: 0.875rem;
  --font-size-base: 1rem;
  --font-size-lg: 1.125rem;
  --font-size-xl: 1.25rem;

  /* Spacing */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-3: 0.75rem;
  --space-4: 1rem;
  --space-6: 1.5rem;
  --space-8: 2rem;

  /* Radius */
  --radius-sm: 0.25rem;
  --radius-md: 0.5rem;
  --radius-lg: 0.75rem;
  --radius-full: 9999px;
}

[data-theme="dark"] {
  --color-background: #0f172a;
  --color-foreground: #f8fafc;
}
```

### Tailwind Export

```javascript
// tailwind.config.js theme extension
{
  colors: {
    primary: 'var(--color-primary)',
    secondary: 'var(--color-secondary)',
    accent: 'var(--color-accent)',
  },
  fontFamily: {
    heading: ['Inter', 'sans-serif'],
    body: ['Inter', 'sans-serif'],
  },
  spacing: {
    '1': '0.25rem',
    '2': '0.5rem',
    // ...
  },
  borderRadius: {
    sm: '0.25rem',
    md: '0.5rem',
    lg: '0.75rem',
    full: '9999px',
  },
}
```

### Validation Output

```
## Token Validation

### design-system.json
✅ Colors: 5 tokens, all valid
⚠️ Typography: font-size-md missing from scale
✅ Spacing: 8 tokens, consistent scale
❌ Radius: radius-xl defined but not used anywhere

### globals.css
✅ Synced with design-system.json
⚠️ 2 extra variables not in design system:
   --header-height
   --sidebar-width

### tailwind.config.js
❌ Out of sync: colors.primary differs from design system
   Tailwind: #2563eb
   Design:   #3b82f6

─────────────────────────────

3 warnings, 2 errors
Run /design:tokens sync to fix sync issues.
```

## Implementation

Export reads from `uiux.designSystem` or `design-system.json`.

Sync compares token files and updates targets.

Validate checks:
- Token naming consistency
- Value format correctness
- Cross-file synchronization
- Usage in codebase

## Notes

- Preserves existing token values when syncing
- Warns before overwriting
- Supports dark mode tokens
- Works with shadcn CSS variable patterns
