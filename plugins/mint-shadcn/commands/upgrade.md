# /shadcn:upgrade Command

Check for and apply shadcn component updates with smart diff merging.

## Usage

```
/shadcn:upgrade [component] [options]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `component` | No | Specific component to upgrade (e.g., `button`). If omitted, checks all. |

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--diff` | Show diff without applying changes | - |
| `--all` | Upgrade all components with changes | - |
| `--force` | Apply registry version, discard local changes | - |
| `--cli` | Update shadcn CLI to latest | - |

## Examples

```bash
# Check all components for updates
/shadcn:upgrade

# Preview button changes without applying
/shadcn:upgrade button --diff

# Upgrade all outdated components
/shadcn:upgrade --all

# Just update the CLI
/shadcn:upgrade --cli

# Force registry version (discard local changes)
/shadcn:upgrade dialog --force
```

## Process

1. **Check CLI version** — compare installed vs latest
2. **Scan components** — run `shadcn diff` to find changes
3. **Present options** — show what's changed
4. **Apply selected updates** — merge or replace

## Output

### Diff Mode (`--diff`)

```
## shadcn Diff Report

### button.tsx
Local modifications detected.

  - Added custom `loading` prop
  - Changed default variant to "outline"

Registry changes:
  + New `asChild` prop support
  + Updated Radix dependency

Recommendation: Manual merge required

### dialog.tsx
No local modifications.

Registry changes:
  + Accessibility improvements
  + New animation options

Recommendation: Safe to auto-upgrade
```

### Upgrade Mode

```
## shadcn Upgrade

Upgrading 3 components...

✅ dialog.tsx — upgraded (no conflicts)
✅ card.tsx — upgraded (no conflicts)
⚠️ button.tsx — skipped (local modifications)

To upgrade button.tsx:
  /shadcn:upgrade button --diff   # Review changes
  /shadcn:upgrade button --force  # Discard local changes
```

## Smart Merge

When upgrading components with local modifications:

1. **Detect modification type:**
   - Added props → usually safe to merge
   - Changed implementation → needs review
   - Removed features → may conflict

2. **Suggest action:**
   - `auto-merge` — registry changes don't conflict with local
   - `manual-merge` — show both versions, let user decide
   - `skip` — too complex, recommend manual review

3. **Preserve customizations:**
   - Track local additions in a comment block
   - Apply registry updates around local code
   - Flag for human review after merge

## CLI Update

When `--cli` is provided:

```bash
# Check current version
npx shadcn@latest --version

# If outdated, update
npm install -D shadcn@latest
# or pnpm/yarn/bun equivalent
```

Report new features in the latest CLI version.

## Errors

### No components installed

```
Error: No shadcn components found in this project.

Install components first:
  npx shadcn add button

Then run /shadcn:upgrade to check for updates.
```

### Component not found

```
Error: Component 'foo' not found.

Installed components:
- button
- dialog
- card

Usage: /shadcn:upgrade <component>
```

## Implementation

When this command is invoked:

1. **Detect package manager** from lockfiles

2. **If `--cli` flag:**
   - Check current CLI version: `npx shadcn --version`
   - Check latest: `npm view shadcn version`
   - If different, update and report

3. **If component specified:**
   - Run `npx shadcn diff <component>`
   - Parse output for changes
   - Apply based on flags (`--diff`, `--force`)

4. **If no component (check all):**
   - Run `npx shadcn diff`
   - Categorize components: up-to-date, auto-upgradable, needs-review
   - If `--all`: apply auto-upgradable, list needs-review
   - If no flag: show summary, prompt for action

5. **Report results:**
   - What was upgraded
   - What was skipped and why
   - Next steps for manual merges

## Notes

- Uses `shadcn diff` under the hood (requires shadcn CLI v4+)
- Local modifications are detected by comparing to registry hash
- `--force` is destructive — always show confirmation
- Registry updates may include breaking changes — check release notes
