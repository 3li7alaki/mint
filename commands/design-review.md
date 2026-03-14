# /design:review Command

Review UI implementation against design conventions, anti-patterns, and standards.

## Usage

```
/design:review [target] [options]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `target` | No | File, directory, or "staged" for git staged files |

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--check <type>` | Run specific check only | all |
| `--fix` | Auto-fix simple issues | - |
| `--json` | Output as JSON | - |

## Check Types

- `slop` — AI slop detection (always runs)
- `accessibility` — WCAG 2.1 AA compliance
- `consistency` — Design system adherence
- `performance` — Bundle and motion
- `rtl` — Right-to-left support
- `i18n` — Internationalization compliance
- `brand` — Brand guide compliance
- `all` — Run all enabled checks

## Examples

```bash
# Review staged changes
/design:review staged

# Review specific file
/design:review src/components/Button.tsx

# Review directory
/design:review src/pages/

# Run specific check
/design:review --check accessibility
/design:review --check slop

# Auto-fix what's possible
/design:review staged --fix

# JSON output for CI
/design:review staged --json
```

## Auto-Fix

The `--fix` flag can automatically fix:
- RTL property swaps (`ml-` → `ms-`, `pl-` → `ps-`, etc.)
- Simple color variable replacements
- Spacing scale corrections

It will NOT auto-fix:
- Missing alt text (needs human input)
- AI slop patterns (needs design rethinking)
- Accessibility structure issues
- Complex refactoring

## CI Integration

Use `--json` for CI pipelines:

```json
{
  "passed": false,
  "blocking": 4,
  "warnings": 5,
  "info": 1,
  "issues": [...]
}
```

Exit codes: `0` = passed, `1` = blocking issues found.

## Implementation

Invokes the `design-reviewer` agent with target files. Loads project design profile, notes, and standards automatically.

## Notes

- AI slop detection always runs regardless of other check settings
- Only reviews UI files (.tsx, .jsx, .vue, .svelte, .css, .scss, .html)
- Respects enabled/disabled checks from design config
- Design notes hard rules are always BLOCKING
