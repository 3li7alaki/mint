# /design:review Command

Review UI implementation against design conventions.

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

- `accessibility` — WCAG 2.1 AA compliance
- `consistency` — Design system adherence
- `performance` — Bundle and motion
- `rtl` — Right-to-left support
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

# Auto-fix what's possible
/design:review staged --fix

# JSON output for CI
/design:review staged --json
```

## Output

```
## Design Review: src/components/

Reviewing 12 files...

### Accessibility (8 issues)

BLOCKING
  ❌ Card.tsx:45 — Image missing alt text
  ❌ Form.tsx:23 — Input missing associated label

WARNING
  ⚠️ Button.tsx:12 — Touch target 32x32px (minimum 44x44)
  ⚠️ Modal.tsx:67 — Missing focus trap

### Consistency (3 issues)

WARNING
  ⚠️ Hero.tsx:34 — Color #3b82f6 not in design system
     Fix: Use var(--primary) or brand-blue
  ⚠️ Card.tsx:12 — Font size 15px not in scale
     Fix: Use text-sm (14px) or text-base (16px)

INFO
  ℹ️ Sidebar.tsx:89 — Could use existing NavItem component

### RTL (2 issues)

BLOCKING
  ❌ Layout.tsx:23 — Using ml-4, should be ms-4
  ❌ Header.tsx:56 — Using pr-6, should be pe-6

### Performance (1 issue)

WARNING
  ⚠️ AnimatedCard.tsx:34 — No prefers-reduced-motion check

─────────────────────────────

Summary: 4 blocking, 5 warnings, 1 info
Files:   12 reviewed, 8 with issues

Fix blocking issues to pass design review.

Auto-fixable: 2 issues
Run /design:review staged --fix to apply.
```

## Auto-Fix

The `--fix` flag can automatically fix:
- RTL property swaps (`ml-` → `ms-`, etc.)
- Simple color variable replacements
- Spacing scale corrections

It will NOT auto-fix:
- Missing alt text (needs human input)
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
  "issues": [
    {
      "file": "Card.tsx",
      "line": 45,
      "severity": "blocking",
      "check": "accessibility",
      "message": "Image missing alt text",
      "fix": null
    }
  ]
}
```

Exit codes:
- `0` — Passed (no blocking issues)
- `1` — Failed (has blocking issues)

## Implementation

Invokes the `design-reviewer.md` agent with target files.

## Notes

- Only reviews UI files (.tsx, .jsx, .vue, .svelte, .css)
- Loads conventions from config
- Integrates with git for staged file detection
- Respects enabled/disabled checks in config
