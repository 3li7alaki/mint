# /design Command

Search design intelligence and generate design systems.

## Usage

```
/design <action> [args] [options]
```

## Actions

### search

Search design knowledge base for guidance.

```bash
/design search "<query>" [--domain <domain>]
```

**Domains:**
- `product` — Product type recommendations
- `style` — UI styles (glassmorphism, minimalism, etc.)
- `typography` — Font pairings and type scales
- `color` — Color palettes by industry or mood
- `landing` — Page structure and CTAs
- `chart` — Chart types and libraries
- `ux` — Best practices and anti-patterns
- `motion` — Animation patterns and timing
- `interaction` — Form, focus, and loading patterns

**Examples:**
```bash
/design search "dashboard"
/design search "glassmorphism" --domain style
/design search "saas landing" --domain landing
/design search "data visualization" --domain chart
```

### system

Generate or update design system.

```bash
/design system [--product <type>] [--output <path>]
```

**Examples:**
```bash
/design system                              # Auto-detect from profile
/design system --product "saas dashboard"   # Specify type
/design system --output design-system.json  # Custom output
```

### palette

Generate color palette.

```bash
/design palette [--industry <type>] [--mood <mood>]
```

**Examples:**
```bash
/design palette                          # Based on project profile
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
/design typography                    # Based on project profile
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
```

## Implementation

Search and generation use the vendored reference docs from `standards/design/reference/` combined with the project's design profile (`.mint/design-profile.json`).

If Impeccable skill is installed (`.claude/skills/frontend-design/`), its knowledge supplements the vendored references.

## Notes

- Results are project-context aware when a design profile exists
- Integrates with existing brand guide if configured
- Outputs can be saved to design-system.json
