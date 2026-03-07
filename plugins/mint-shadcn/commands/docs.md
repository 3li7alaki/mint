# /shadcn:docs Command

Get documentation, code examples, and API reference for any component.

## Usage

```
/shadcn:docs <component> [options]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `component` | Yes | Component name (e.g., "button", "dialog", "data-table") |

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--base <lib>` | Base library (radix, base) | from components.json |
| `--json` | Output as JSON | - |

## Examples

```bash
# Get button documentation
/shadcn:docs button

# Get dialog docs with Base UI primitives
/shadcn:docs dialog --base base

# Output as JSON for parsing
/shadcn:docs accordion --json
```

## Output

```
## Button

A button component with multiple variants and sizes.

### Installation

npx shadcn add button

### Usage

import { Button } from "@/components/ui/button"

<Button variant="outline">Click me</Button>

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| variant | "default" | "destructive" | "outline" | "secondary" | "ghost" | "link" | "default" | Visual style |
| size | "default" | "sm" | "lg" | "icon" | "default" | Size variant |
| asChild | boolean | false | Render as child component |

### Variants

- **default**: Primary action button
- **destructive**: Dangerous/delete actions
- **outline**: Secondary with border
- **secondary**: Less prominent action
- **ghost**: Minimal, no background
- **link**: Styled as a link

### Examples

#### With Icon
<Button>
  <Mail className="mr-2 h-4 w-4" /> Login with Email
</Button>

#### Loading State
<Button disabled>
  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
  Please wait
</Button>

### Source
https://ui.shadcn.com/docs/components/button
```

## Implementation

When this command is invoked:

1. **Run shadcn docs:**
   ```bash
   npx shadcn docs <component> --base <base>
   ```

2. **Parse and format output:**
   - Installation command
   - Import statement
   - Props table
   - Code examples
   - Source link

3. **If `--json` flag:**
   - Output raw JSON for programmatic use

## AI Agent Integration

When an AI agent needs component info:

1. Agent calls `/shadcn:docs <component>`
2. Gets complete API reference
3. Uses correct props and patterns
4. Avoids hallucinating non-existent APIs

This is especially useful when:
- Building new UIs with unfamiliar components
- Checking available variants/sizes
- Finding correct import paths
- Getting example code patterns

## Notes

- Fetches from official shadcn documentation
- Includes both Radix and Base UI examples when applicable
- Props table matches the exact TypeScript types
- Examples are copy-paste ready
