# /shadcn:search Command

Search for components across all configured registries.

## Usage

```
/shadcn:search <query> [options]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `query` | Yes | Search term (e.g., "button", "animated card", "date picker") |

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--registry <name>` | Search specific registry only | all |
| `--limit <n>` | Max results per registry | 10 |
| `--json` | Output as JSON | - |

## Examples

```bash
# Search all registries
/shadcn:search button

# Search for animated components
/shadcn:search "animated"

# Search specific registry
/shadcn:search card --registry @magicui

# Get more results
/shadcn:search input --limit 20
```

## Output

```
## shadcn Search: "button"

### shadcn/ui (3 results)
  button          Base button component with variants
  toggle          Toggle button with pressed state
  toggle-group    Group of toggle buttons

### @magicui (4 results)
  shimmer-button  Button with shimmer animation effect
  pulsating-button Button with pulse animation
  animated-subscribe Subscribe button with animation
  magic-card      Card with magical hover effects

### @animate-ui (2 results)
  button          Motion-enhanced button
  icon-button     Animated icon button

─────────────────────────────
Total: 9 results across 3 registries

To add: /shadcn:add <component>
To preview: /shadcn:add <component> --dry-run
```

## Implementation

When this command is invoked:

1. **Parse query and options**

2. **Run shadcn search:**
   ```bash
   npx shadcn search --query "<query>" --limit <n>
   ```

3. **If `--registry` specified:**
   - Filter results to that registry only

4. **Format output:**
   - Group by registry
   - Show component name and description
   - Include total count

5. **If `--json` flag:**
   - Output raw JSON from CLI

## Notes

- Searches component names, descriptions, and tags
- Fuzzy matching supported ("btn" matches "button")
- Registry must be configured in `components.json` to appear in results
- Use `/shadcn:add` to install found components
