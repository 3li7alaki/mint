# mint-ui-ux

UI/UX design intelligence plugin for [mint](https://github.com/3li7alaki/mint).

Enhances [ui-ux-pro-max](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill) with project learning, design review, and mint integration.

## Features

- **Project profile learning** — Analyzes existing UI code to build a design profile
- **Design notes** — Remembers preferences, rules, and decisions over time
- **Design review** — Stage 2 auditor for accessibility, consistency, RTL, performance
- **Convention injection** — Feeds learned patterns into planning
- **Token management** — Export/sync tokens across CSS, Tailwind, SCSS, JSON
- **shadcn integration** — Cross-references with mint-shadcn for component awareness

## How It Learns

### Design Profile (`.mint/design-profile.json`)

When you start working on a project, mint-ui-ux analyzes existing UI code:

```json
{
  "identity": {
    "style": "minimal-professional",
    "mood": ["clean", "trustworthy"]
  },
  "colors": {
    "primary": "#3b82f6",
    "palette": "cool-neutral"
  },
  "typography": {
    "headings": "Inter",
    "scale": "1.250"
  },
  "components": {
    "buttons": "rounded-lg, no full-round",
    "cards": "subtle shadow, no borders"
  },
  "learnedFrom": ["src/components/ui/"]
}
```

The profile is built incrementally and tracks where patterns came from.

### Design Notes (`.mint/design-notes.md`)

As you work, your preferences become persistent rules:

```markdown
## Hard Rules
- Never use red for success states
- Buttons are always rounded-lg

## Preferences
- Prefer subtle shadows over borders

## Decisions Made
- 2024-03-07: Chose Inter over Geist for Arabic support
```

When you say "we never use that color" or "buttons should always be rounded" — it's remembered.

## Requirements

**ui-ux-pro-max skill** — The design intelligence engine:

```bash
# Install via CLI (recommended)
npm install -g uipro-cli
uipro init --ai claude

# Or via Claude marketplace
/plugin marketplace add nextlevelbuilder/ui-ux-pro-max
```

**Python 3** — For the search engine:

```bash
python3 --version  # Must be 3.x
```

## Install

After installing mint (`npx skills add 3li7alaki/mint`), enable in `.mint/config.json`:

```json
{
  "plugins": ["plugins/mint-ui-ux"]
}
```

Or run setup:

```bash
/design:setup
```

## Commands

### `/design:setup`

Interactive setup wizard.

```bash
/design:setup                    # Full interactive
/design:setup --yes              # Accept defaults
/design:setup --generate         # Generate design system
```

### `/design:profile`

Build and manage the project's design profile.

```bash
/design:profile build              # Analyze code, build profile
/design:profile view               # Show current profile
/design:profile view --section colors  # Specific section
/design:profile update colors.primary "#ff2a2a"
```

### `/design:notes`

Manage design notes and rules.

```bash
/design:notes add "never use red for success" --type rule
/design:notes add "prefer shadows over borders" --type preference
/design:notes list
/design:notes remove --number 3
```

### `/design search`

Search design intelligence databases.

```bash
/design search "dashboard"                    # All domains
/design search "glassmorphism" --domain style # Specific domain
/design search "saas" --stack react           # Stack-specific
```

Domains: `product`, `style`, `typography`, `color`, `landing`, `chart`, `ux`

### `/design system`

Generate or update design system.

```bash
/design system                              # Auto-detect
/design system --product "saas dashboard"   # Specify type
```

### `/design palette`

Generate color palettes.

```bash
/design palette                      # Based on project
/design palette --industry fintech   # Industry-specific
```

### `/design typography`

Get font pairing recommendations.

```bash
/design typography                # Based on project
/design typography --style modern # Style-specific
```

### `/design inspiration`

Get style inspiration.

```bash
/design inspiration "hero section"
/design inspiration "pricing table"
```

### `/design:review`

Review UI implementation.

```bash
/design:review staged              # Review staged changes
/design:review src/components/     # Review directory
/design:review --check accessibility
/design:review staged --fix        # Auto-fix simple issues
```

Checks: `accessibility`, `consistency`, `performance`, `rtl`, `brand`

### `/design:tokens`

Manage design tokens.

```bash
/design:tokens export --format css           # Export to CSS
/design:tokens export --format tailwind      # Tailwind config
/design:tokens sync                          # Sync across files
/design:tokens validate                      # Check consistency
```

Formats: `css`, `tailwind`, `scss`, `json`, `figma`

## Configuration

In `.mint/config.json`:

```json
{
  "uiux": {
    "conventions": [
      "BRAND_GUIDE.md",
      "docs/design-system.md"
    ],
    "profile": ".mint/design-profile.json",
    "notes": ".mint/design-notes.md",
    "stack": "react",
    "review": {
      "accessibility": true,
      "consistency": true,
      "performance": true,
      "rtl": true,
      "brand": true
    }
  }
}
```

## How It Works

### Pre-Plan Hook

When you describe a UI task, the `design-context` agent:

1. Loads your design profile
2. Reads your design notes (rules become constraints)
3. Checks brand guide, design system, tokens
4. Queries ui-ux-pro-max for relevant styles
5. Checks shadcn for available components
6. Injects all context into the planner

Your specs automatically know your project's design language.

### Pre-Review Hook

After implementation, the `design-reviewer` agent:

1. Checks accessibility (WCAG 2.1 AA)
2. Validates against design profile
3. Enforces design notes (rules block, preferences warn)
4. Checks RTL support
5. Flags performance issues
6. Reports brand compliance

### Profile Learning

The profile is built incrementally:
- First UI task: analyze existing code
- Ongoing: learn from new patterns
- Manual: `/design:profile build` to refresh

Notes accumulate over time:
- Natural language: "we never use that" → rule
- Explicit: `/design:notes add` command
- Review: suggestions become preferences

## Integration with mint-shadcn

When both plugins are enabled:

- Design profile includes shadcn style
- Available components considered during planning
- Review checks shadcn usage patterns
- Token export includes shadcn format

## Agents

| Agent | Hook | Role |
|-------|------|------|
| `setup-wizard.md` | on-init | Install and configure |
| `design-context.md` | pre-plan | Inject design conventions |
| `design-reviewer.md` | pre-review | Stage 2 design audit |
| `profile-builder.md` | on-demand | Analyze code, build profile |

## Files

| File | Purpose |
|------|---------|
| `.mint/design-profile.json` | Learned design patterns |
| `.mint/design-notes.md` | User rules and preferences |
| `design-system.json` | Explicit design tokens |
| `BRAND_GUIDE.md` | Brand guidelines |

## Sources

- [ui-ux-pro-max](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill) — Design intelligence engine
- [elite-frontend-ux](https://gist.github.com/majidmanzarpour/8b95e5e0e78d7eeacd3ee54606c7acc6) — Additional UX patterns
- [shadcn/ui](https://ui.shadcn.com) — Component library integration
