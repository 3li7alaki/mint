# /design Command

Search design intelligence and generate design systems.

## Usage

```
/design <action> [args] [options]
```

## Actions

### search

Search ui-ux-pro-max databases.

```bash
/design search "<query>" [--domain <domain>] [--stack <stack>]
```

**Domains:**
- `product` — Product type recommendations
- `style` — UI styles (glassmorphism, minimalism, etc.)
- `typography` — Font pairings
- `color` — Color palettes by industry
- `landing` — Page structure and CTAs
- `chart` — Chart types and libraries
- `ux` — Best practices and anti-patterns

**Stacks:**
`react`, `nextjs`, `vue`, `nuxt`, `svelte`, `astro`, `react-native`, `flutter`, `shadcn`

**Examples:**
```bash
/design search "dashboard"
/design search "glassmorphism" --domain style
/design search "saas landing" --domain landing
/design search "data visualization" --stack react
```

### system

Generate or update design system.

```bash
/design system [--product <type>] [--output <path>]
```

**Examples:**
```bash
/design system                              # Auto-detect
/design system --product "saas dashboard"   # Specify type
/design system --output design-system.json  # Custom output
```

**Output:**
```json
{
  "colors": {
    "primary": "#3b82f6",
    "secondary": "#64748b",
    "accent": "#f59e0b"
  },
  "typography": {
    "headings": "Inter",
    "body": "Inter",
    "scale": [12, 14, 16, 18, 20, 24, 30, 36, 48]
  },
  "spacing": [4, 8, 12, 16, 24, 32, 48, 64],
  "radius": [4, 8, 12, 16],
  "shadows": ["sm", "md", "lg"],
  "style": "minimal-clean",
  "antiPatterns": [
    "No purple-blue gradients",
    "Avoid placeholder-as-label"
  ]
}
```

### palette

Generate color palette.

```bash
/design palette [--industry <type>] [--mood <mood>]
```

**Examples:**
```bash
/design palette                          # Based on project
/design palette --industry fintech       # Industry-specific
/design palette --mood "bold energetic"  # Mood-based
```

### typography

Get font pairing recommendations.

```bash
/design typography [--style <style>]
```

**Examples:**
```bash
/design typography                    # Based on project
/design typography --style modern     # Modern pairings
/design typography --style editorial  # Editorial pairings
```

### inspiration

Get style inspiration for a component or page.

```bash
/design inspiration "<element>"
```

**Examples:**
```bash
/design inspiration "hero section"
/design inspiration "pricing table"
/design inspiration "dashboard sidebar"
/design inspiration "mobile navigation"
```

## Implementation

All search commands use:
```bash
python3 .claude/skills/ui-ux-pro-max/scripts/search.py "<query>" --domain <domain>
```

Design system generation uses:
```bash
python3 .claude/skills/ui-ux-pro-max/scripts/design_system.py --product "<type>" --stack "<stack>"
```

## Notes

- Requires ui-ux-pro-max skill installed
- Results are project-context aware
- Integrates with existing brand guide if configured
- Outputs can be saved to design-system.json
