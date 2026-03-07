# /shadcn:info Command

Display complete project configuration and installed components.

## Usage

```
/shadcn:info [options]
```

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--json` | Output as JSON (for AI agents) | - |

## Examples

```bash
# Show project info
/shadcn:info

# Get JSON for programmatic use
/shadcn:info --json
```

## Output

```
## shadcn Project Info

### Configuration
Framework:        react (vite)
Tailwind:         v4.1.18
Base Library:     radix
Icon Library:     lucide
Style:            new-york
CSS Variables:    enabled
RSC:              false

### Paths
Components:       src/components
UI Components:    src/components/ui
Utils:            src/lib/utils
Hooks:            src/hooks
Lib:              src/lib

### Registries
shadcn/ui         https://ui.shadcn.com/registry
@magicui          https://magicui.design/r/{name}.json
@animate-ui       https://animate-ui.com/r/{name}.json

### Installed Components (23)
accordion, alert, alert-dialog, avatar, badge, button,
calendar, card, checkbox, collapsible, command, dialog,
dropdown-menu, input, label, popover, progress, radio-group,
scroll-area, select, separator, skeleton, switch

### CLI Version
shadcn:           4.2.1 (latest)

### Documentation
Components:       https://ui.shadcn.com/docs/components
Theming:          https://ui.shadcn.com/docs/theming
CLI Reference:    https://ui.shadcn.com/docs/cli
```

## JSON Output

When `--json` is provided, outputs structured data:

```json
{
  "framework": "react",
  "bundler": "vite",
  "tailwindVersion": "4.1.18",
  "baseLibrary": "radix",
  "iconLibrary": "lucide",
  "style": "new-york",
  "cssVariables": true,
  "rsc": false,
  "paths": {
    "components": "src/components",
    "ui": "src/components/ui",
    "utils": "src/lib/utils",
    "hooks": "src/hooks",
    "lib": "src/lib"
  },
  "registries": {
    "shadcn": "https://ui.shadcn.com/registry",
    "@magicui": "https://magicui.design/r/{name}.json"
  },
  "installedComponents": ["accordion", "alert", ...],
  "cliVersion": "4.2.1"
}
```

## Implementation

When this command is invoked:

1. **Run shadcn info:**
   ```bash
   npx shadcn info --json
   ```

2. **Parse output** and format for display

3. **Check CLI version:**
   ```bash
   npx shadcn --version
   npm view shadcn version
   ```

4. **Compare versions** and note if update available

## AI Agent Use

This command is critical for AI agents to understand project context:

- **shadcn/skills** calls this automatically via `shadcn info --json`
- Agents learn your aliases, paths, and installed components
- Prevents hallucinating wrong import paths
- Enables intelligent component suggestions

## Notes

- Reads from `components.json` and `package.json`
- Shows resolved paths (not just aliases)
- Lists all installed UI components
- Indicates if CLI update is available
