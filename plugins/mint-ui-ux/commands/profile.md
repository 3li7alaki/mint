# /design:profile Command

Build, view, and update the project's design profile.

## Usage

```
/design:profile <action> [options]
```

## Actions

### build

Analyze existing code to build/update design profile.

```bash
/design:profile build [--path <dir>] [--force]
```

**Options:**
- `--path <dir>` — Analyze specific directory (default: src/)
- `--force` — Rebuild from scratch, ignore existing profile

**Examples:**
```bash
/design:profile build                      # Analyze and build
/design:profile build --path src/components  # Specific directory
/design:profile build --force              # Full rebuild
```

### view

Display current design profile.

```bash
/design:profile view [--section <name>]
```

**Sections:**
- `colors` — Color palette and usage
- `typography` — Fonts and scale
- `spacing` — Spacing system
- `components` — Component patterns
- `all` — Everything (default)

**Examples:**
```bash
/design:profile view                # Full profile
/design:profile view --section colors  # Just colors
```

### update

Manually update profile values.

```bash
/design:profile update <key> <value>
```

**Examples:**
```bash
/design:profile update identity.style "glassmorphic"
/design:profile update colors.primary "#ff2a2a"
/design:profile update constraints.rtl true
```

### diff

Show what's changed since last profile build.

```bash
/design:profile diff
```

Shows new patterns, changed values, removed items.

## Output

### Build Output

```
## Design Profile Built

Analyzed 47 files in src/

### Summary
| Category   | Items Found | Confidence |
|------------|-------------|------------|
| Colors     | 12 tokens   | High       |
| Typography | 3 fonts     | High       |
| Spacing    | 8 values    | Medium     |
| Components | 15 patterns | Medium     |

### Key Patterns
- **Style**: minimal-professional
- **Primary**: #3b82f6 (45 uses)
- **Radius**: rounded-lg dominant
- **Shadows**: subtle (shadow-sm)

Profile saved: .mint/design-profile.json

### Suggested Notes
Based on analysis, consider adding these rules to design-notes.md:

1. Buttons use rounded-md consistently
2. Cards always have p-6 padding
3. Primary blue used for interactive elements only
```

### View Output

```
## Design Profile: my-project

**Generated:** 2024-03-07
**Files Analyzed:** 47
**Confidence:** High

### Identity
- **Product**: SaaS dashboard
- **Industry**: fintech
- **Style**: minimal-professional
- **Mood**: clean, trustworthy, modern

### Colors
| Token     | Value   | Usage                    |
|-----------|---------|--------------------------|
| primary   | #3b82f6 | buttons, links, accents  |
| secondary | #64748b | text, borders            |
| success   | #22c55e | success states           |
| warning   | #eab308 | warning states           |
| error     | #ef4444 | error states             |

### Typography
- **Headings**: Inter, semibold
- **Body**: Inter, regular
- **Scale**: 1.250 (major third)

### Spacing
- **Base**: 4px
- **Common**: p-4, p-6, gap-4, gap-6
- **Cards**: p-6
- **Sections**: gap-8

### Components
- **Buttons**: rounded-md, shadow-sm
- **Cards**: rounded-xl, p-6, shadow-sm
- **Inputs**: rounded-md, bordered

### Constraints
- RTL: No
- Dark mode: Yes
- WCAG: AA

### Learned From
- src/components/ui/
- src/app/dashboard/
- tailwind.config.ts
```

## Implementation

Invokes the `profile-builder.md` agent.

Profile stored at `.mint/design-profile.json`.

## Notes

- Safe to run multiple times — merges with existing
- Tracks confidence per section
- Records what files informed each pattern
- Works with or without explicit design system
