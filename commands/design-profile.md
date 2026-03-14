# /design:profile Command

Build, view, and update the project's design profile.

## Usage

```
/design:profile <action> [options]
```

## Actions

### build

Analyze existing UI code and build/update the design profile.

```bash
/design:profile build [--path <dir>] [--force]
```

| Option | Description |
|--------|-------------|
| `--path <dir>` | Directory to analyze (default: src/) |
| `--force` | Rebuild from scratch instead of updating |

**Examples:**
```bash
/design:profile build                    # Analyze src/
/design:profile build --path app/       # Analyze specific dir
/design:profile build --force           # Full rebuild
```

### view

Display the current design profile.

```bash
/design:profile view [--section <name>]
```

**Sections:** `identity`, `colors`, `typography`, `spacing`, `borders`, `shadows`, `components`, `layout`, `constraints`

**Examples:**
```bash
/design:profile view                  # Full profile
/design:profile view --section colors # Just colors
```

### update

Manually update a specific profile value.

```bash
/design:profile update <key> <value>
```

**Examples:**
```bash
/design:profile update identity.style "bold-vibrant"
/design:profile update constraints.rtl true
/design:profile update colors.primary "#6366f1"
```

### diff

Show what changed since the profile was last built.

```bash
/design:profile diff
```

## Implementation

Invokes the `design-profile` agent. Profile is saved to `.mint/design-profile.json`.

## Notes

- Profile builds incrementally — new patterns supplement existing ones
- Use `--force` only for complete rebuild
- Confidence levels (high/medium/low) indicate certainty of inferred patterns
- Explicit design-system.json values override inferred values
